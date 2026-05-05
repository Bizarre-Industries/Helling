package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

type createWebhookRequest struct {
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Secret  string   `json:"secret"`
	Events  []string `json:"events"`
	Enabled *bool    `json:"enabled"`
}

type updateWebhookRequest struct {
	Name    *string   `json:"name"`
	URL     *string   `json:"url"`
	Secret  *string   `json:"secret"`
	Events  *[]string `json:"events"`
	Enabled *bool     `json:"enabled"`
}

type webhookResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type webhookDeliveryResponse struct {
	ID         string    `json:"id"`
	WebhookID  string    `json:"webhook_id"`
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	Status     string    `json:"status"`
	Attempt    int       `json:"attempt"`
	HTTPStatus *int      `json:"http_status,omitempty"`
	Error      *string   `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	rows, err := s.cfg.Store.ListWebhooks(r.Context())
	if err != nil {
		s.cfg.Logger.Error("list webhooks", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := make([]webhookResponse, 0, len(rows))
	for i := range rows {
		out = append(out, webhookToResponse(&rows[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)
	if req.Name == "" || req.URL == "" || req.Secret == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name, url, and secret are required")
		return
	}
	if err := validateWebhookSecret(req.Secret); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "events are required")
		return
	}
	if _, err := store.ValidateWebhookEvents(req.Events); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := validateWebhookURL(r.Context(), req.URL); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	row, err := s.cfg.Store.CreateWebhook(r.Context(), user.ID, req.Name, req.URL, req.Secret, req.Events)
	if err != nil {
		s.cfg.Logger.Error("create webhook", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if req.Enabled != nil && !*req.Enabled {
		if err := s.cfg.Store.UpdateWebhook(r.Context(), row.ID, row.Name, row.URL, req.Secret, row.Events, false); err != nil {
			s.cfg.Logger.Error("create webhook: disable", slog.Any("err", err))
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
			return
		}
		row.Enabled = false
	}
	_, _ = s.emitEvent(r.Context(), "webhook.created", row.ID, map[string]any{"name": row.Name})
	s.audit(r, "webhook.create", outcomeSuccess, "webhook", row.ID, "webhook created")
	writeJSON(w, http.StatusCreated, map[string]any{"data": webhookToResponse(&row)})
}

func (s *Server) handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	row, err := s.cfg.Store.GetWebhook(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "webhook not found")
			return
		}
		s.cfg.Logger.Error("get webhook", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": webhookToResponse(&row)})
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	row, err := s.cfg.Store.GetWebhook(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "webhook not found")
			return
		}
		s.cfg.Logger.Error("update webhook: get", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	var req updateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	secret, err := s.cfg.Store.DecryptSecret(row.SecretEncrypted)
	if err != nil {
		s.cfg.Logger.Error("update webhook: decrypt", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	secret, err = applyWebhookUpdate(r.Context(), &row, req, secret)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := s.cfg.Store.UpdateWebhook(r.Context(), row.ID, row.Name, row.URL, secret, row.Events, row.Enabled); err != nil {
		s.cfg.Logger.Error("update webhook", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	row, _ = s.cfg.Store.GetWebhook(r.Context(), row.ID)
	_, _ = s.emitEvent(r.Context(), "webhook.updated", row.ID, map[string]any{"name": row.Name})
	s.audit(r, "webhook.update", outcomeSuccess, "webhook", row.ID, "webhook updated")
	writeJSON(w, http.StatusOK, map[string]any{"data": webhookToResponse(&row)})
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.cfg.Store.DeleteWebhook(r.Context(), id); err != nil {
		s.cfg.Logger.Error("delete webhook", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	_, _ = s.emitEvent(r.Context(), "webhook.deleted", id, nil)
	s.audit(r, "webhook.delete", outcomeSuccess, "webhook", id, "webhook deleted")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	row, err := s.cfg.Store.GetWebhook(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "webhook not found")
			return
		}
		s.cfg.Logger.Error("test webhook: get", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	ev, err := s.emitEvent(r.Context(), "webhook.test", row.ID, map[string]any{"webhook_id": row.ID})
	if err != nil {
		s.cfg.Logger.Error("test webhook: event", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	secret, err := s.cfg.Store.DecryptSecret(row.SecretEncrypted)
	if err != nil {
		s.cfg.Logger.Error("test webhook: decrypt", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	body, err := json.Marshal(eventToResponse(&ev))
	if err != nil {
		s.cfg.Logger.Error("test webhook: marshal", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	statusCode, responseBody, deliveryErr := deliverWebhookOnce(r.Context(), row.URL, secret, body)
	status := outcomeSuccess
	var errMsg *string
	if deliveryErr != nil {
		status = outcomeFailed
		msg := deliveryErr.Error()
		errMsg = &msg
	}
	delivery, err := s.cfg.Store.CreateWebhookDelivery(r.Context(), row.ID, ev.ID, ev.Type, status, statusCode, responseBody, errMsg, 1)
	if err != nil {
		s.cfg.Logger.Error("test webhook: delivery", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	s.audit(r, "webhook.test", status, "webhook", row.ID, "webhook test delivery")
	writeJSON(w, http.StatusAccepted, map[string]any{"data": webhookDeliveryToResponse(&delivery)})
}

func applyWebhookUpdate(ctx context.Context, row *store.Webhook, req updateWebhookRequest, secret string) (string, error) {
	if req.Name != nil {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.URL != nil {
		row.URL = strings.TrimSpace(*req.URL)
		if err := validateWebhookURL(ctx, row.URL); err != nil {
			return "", err
		}
	}
	if req.Events != nil {
		events, err := store.ValidateWebhookEvents(*req.Events)
		if err != nil {
			return "", err
		}
		row.Events = events
	}
	if req.Secret != nil {
		secret = *req.Secret
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if row.Name == "" || row.URL == "" || secret == "" || len(row.Events) == 0 {
		return "", errors.New("name, url, secret, and events cannot be empty")
	}
	if err := validateWebhookSecret(secret); err != nil {
		return "", err
	}
	return secret, nil
}

func validateWebhookSecret(secret string) error {
	const (
		minWebhookSecretLength = 16
		maxWebhookSecretLength = 512
	)
	if len(secret) < minWebhookSecretLength {
		return fmt.Errorf("secret must be at least %d characters", minWebhookSecretLength)
	}
	if len(secret) > maxWebhookSecretLength {
		return fmt.Errorf("secret must be at most %d characters", maxWebhookSecretLength)
	}
	return nil
}

func webhookToResponse(row *store.Webhook) webhookResponse {
	return webhookResponse{
		ID:        row.ID,
		Name:      row.Name,
		URL:       row.URL,
		Events:    row.Events,
		Enabled:   row.Enabled,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func webhookDeliveryToResponse(row *store.WebhookDelivery) webhookDeliveryResponse {
	return webhookDeliveryResponse{
		ID:         row.ID,
		WebhookID:  row.WebhookID,
		EventID:    row.EventID,
		EventType:  row.EventType,
		Status:     row.Status,
		Attempt:    row.Attempt,
		HTTPStatus: row.StatusCode,
		Error:      row.Error,
		CreatedAt:  row.CreatedAt,
	}
}

func webhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

var defaultWebhookHTTPClient = &http.Client{
	Timeout:   10 * time.Second,
	Transport: safeWebhookTransport(),
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("webhook redirect limit exceeded")
		}
		return validateWebhookURL(req.Context(), req.URL.String())
	},
}

func deliverWebhookOnce(ctx context.Context, destURL, secret string, body []byte) (status *int, responseBody *string, err error) {
	return deliverWebhookWithClient(ctx, defaultWebhookHTTPClient, destURL, secret, body)
}

func deliverWebhookWithClient(ctx context.Context, client *http.Client, destURL, secret string, body []byte) (status *int, responseBody *string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := validateWebhookURL(ctx, destURL); err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destURL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("creating webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Helling-Signature", webhookSignature(secret, body))

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("delivering webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	statusCode := resp.StatusCode
	const maxWebhookResponseBody = 4096
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxWebhookResponseBody+1))
	if len(bodyBytes) > maxWebhookResponseBody {
		return &statusCode, nil, errors.New("webhook response body too large")
	}
	response := string(bodyBytes)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &statusCode, &response, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return &statusCode, &response, nil
}

func safeWebhookTransport() *http.Transport {
	return &http.Transport{
		DialContext:           safeWebhookDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
}

func safeWebhookDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("parsing webhook dial address: %w", err)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolving webhook dial host: %w", err)
	}
	for _, addr := range ips {
		if blockedWebhookIP(addr.IP) {
			return nil, errors.New("webhook url resolves to a blocked address range")
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("webhook host resolved no addresses")
	}
	var d net.Dialer
	return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
}

func validateWebhookURL(ctx context.Context, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return errors.New("webhook url must be absolute https")
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if blockedWebhookIP(ip) {
			return errors.New("webhook url resolves to a blocked address range")
		}
		return nil
	}
	resolver := net.DefaultResolver
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolving webhook host: %w", err)
	}
	for _, addr := range addrs {
		if blockedWebhookIP(addr.IP) {
			return errors.New("webhook url resolves to a blocked address range")
		}
	}
	return nil
}

func blockedWebhookIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	if !ip.IsGlobalUnicast() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	for _, blockedRange := range blockedWebhookIPRanges {
		if blockedRange.Contains(ip) {
			return true
		}
	}
	return false
}

var blockedWebhookIPRanges = mustParseIPRanges([]string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.2/32",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
	"::/128",
	"::1/128",
	"::ffff:0:0/96",
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"100:0:0:1::/64",
	"2001::/23",
	"2001:db8::/32",
	"2002::/16",
	"3fff::/20",
	"5f00::/16",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
})

func mustParseIPRanges(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("invalid webhook blocked range %q: %v", cidr, err))
		}
		out = append(out, network)
	}
	return out
}
