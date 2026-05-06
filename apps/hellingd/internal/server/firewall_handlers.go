package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	hostfirewall "github.com/Bizarre-Industries/helling/apps/hellingd/internal/firewall"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

const (
	firewallHelperPath   = hostfirewall.DefaultHelperPath
	firewallProtocolTCP  = "tcp"
	firewallProtocolUDP  = "udp"
	firewallProtocolICMP = "icmp"
	firewallProtocolAny  = "any"
)

type createFirewallRuleRequest struct {
	Direction       string  `json:"direction"`
	Action          string  `json:"action"`
	Protocol        string  `json:"protocol"`
	SourceCIDR      *string `json:"source_cidr"`
	DestinationCIDR *string `json:"destination_cidr"`
	DestinationPort *int    `json:"destination_port"`
	Enabled         *bool   `json:"enabled"`
	Comment         *string `json:"comment"`
}

type firewallRuleResponse struct {
	ID              string    `json:"id"`
	Direction       string    `json:"direction"`
	Action          string    `json:"action"`
	Protocol        string    `json:"protocol"`
	SourceCIDR      *string   `json:"source_cidr,omitempty"`
	DestinationCIDR *string   `json:"destination_cidr,omitempty"`
	DestinationPort *int      `json:"destination_port,omitempty"`
	Enabled         bool      `json:"enabled"`
	Comment         *string   `json:"comment,omitempty"`
	NFTComment      string    `json:"nft_comment"`
	NFTHandle       *int64    `json:"nft_handle,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (s *Server) handleListFirewallRules(w http.ResponseWriter, r *http.Request) {
	rows, err := s.cfg.Store.ListFirewallRules(r.Context())
	if err != nil {
		s.cfg.Logger.Error("list firewall rules", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	out := make([]firewallRuleResponse, 0, len(rows))
	for i := range rows {
		out = append(out, firewallRuleToResponse(&rows[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *Server) handleCreateFirewallRule(w http.ResponseWriter, r *http.Request) {
	var req createFirewallRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if err := validateFirewallRule(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row, err := s.cfg.Store.CreateFirewallRule(r.Context(), user.ID, &store.FirewallRuleInput{
		Direction:       req.Direction,
		Action:          req.Action,
		Protocol:        req.Protocol,
		SourceCIDR:      req.SourceCIDR,
		DestinationCIDR: req.DestinationCIDR,
		DestinationPort: req.DestinationPort,
		Enabled:         enabled,
		Comment:         req.Comment,
	})
	if err != nil {
		s.cfg.Logger.Error("create firewall rule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if row.Enabled {
		if err := applyFirewallRule(r.Context(), &row); err != nil {
			_ = s.cfg.Store.DeleteFirewallRule(context.WithoutCancel(r.Context()), row.ID)
			s.cfg.Logger.Error("firewall rule apply failed", slog.String("id", row.ID), slog.Any("err", err))
			writeError(w, http.StatusBadGateway, "firewall_apply_failed", "could not apply firewall rule")
			return
		}
		if handle, err := lookupFirewallHandle(r.Context(), &row); err == nil {
			_ = s.cfg.Store.UpdateFirewallRuleHandle(r.Context(), row.ID, handle)
			row.NFTHandle = &handle
		}
	}
	_, _ = s.emitEvent(r.Context(), "firewall.rule.created", row.ID, nil)
	s.audit(r, "firewall.rule.create", outcomeSuccess, "firewall_rule", row.ID, "firewall rule created")
	writeJSON(w, http.StatusCreated, map[string]any{"data": firewallRuleToResponse(&row)})
}

func (s *Server) handleDeleteFirewallRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "rule id required")
		return
	}
	row, err := s.cfg.Store.GetFirewallRule(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "firewall rule not found")
			return
		}
		s.cfg.Logger.Error("get firewall rule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if row.Enabled {
		if err := removeFirewallRule(r.Context(), &row); err != nil {
			s.cfg.Logger.Error("remove firewall rule", slog.Any("err", err), slog.String("id", row.ID))
			writeError(w, http.StatusInternalServerError, "internal", "could not remove firewall rule")
			return
		}
	}
	if err := s.cfg.Store.DeleteFirewallRule(r.Context(), id); err != nil {
		s.cfg.Logger.Error("delete firewall rule", slog.Any("err", err))
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	_, _ = s.emitEvent(r.Context(), "firewall.rule.deleted", id, nil)
	s.audit(r, "firewall.rule.delete", outcomeSuccess, "firewall_rule", id, "firewall rule deleted")
	w.WriteHeader(http.StatusNoContent)
}

func validateFirewallRule(req *createFirewallRuleRequest) error {
	if err := validateFirewallEnums(req); err != nil {
		return err
	}
	if err := validateOptionalCIDR(req.SourceCIDR, "source_cidr"); err != nil {
		return err
	}
	if err := validateOptionalCIDR(req.DestinationCIDR, "destination_cidr"); err != nil {
		return err
	}
	if req.DestinationPort != nil && (*req.DestinationPort < 1 || *req.DestinationPort > 65535) {
		return errors.New("destination_port must be 1..65535")
	}
	if req.DestinationPort != nil && req.Protocol != firewallProtocolTCP && req.Protocol != firewallProtocolUDP {
		return errors.New("destination_port requires tcp or udp protocol")
	}
	return nil
}

func validateFirewallEnums(req *createFirewallRuleRequest) error {
	if req.Direction != "input" && req.Direction != "output" && req.Direction != "forward" {
		return errors.New("direction must be input, output, or forward")
	}
	if req.Action != "accept" && req.Action != "drop" && req.Action != "reject" {
		return errors.New("action must be accept, drop, or reject")
	}
	if req.Protocol != firewallProtocolTCP && req.Protocol != firewallProtocolUDP && req.Protocol != firewallProtocolICMP && req.Protocol != firewallProtocolAny {
		return errors.New("protocol must be tcp, udp, icmp, or any")
	}
	return nil
}

func validateOptionalCIDR(value *string, field string) error {
	if value == nil {
		return nil
	}
	if _, _, err := net.ParseCIDR(*value); err != nil {
		return errors.New(field + " must be CIDR notation")
	}
	return nil
}

func nftArgsForRule(row *store.FirewallRule) []string {
	args := []string{"add", "rule", "inet", "helling", row.Direction}
	if row.SourceCIDR != nil {
		args = append(args, "ip", "saddr", *row.SourceCIDR)
	}
	if row.DestinationCIDR != nil {
		args = append(args, "ip", "daddr", *row.DestinationCIDR)
	}
	if row.Protocol != firewallProtocolAny {
		args = append(args, row.Protocol)
		if row.DestinationPort != nil && (row.Protocol == firewallProtocolTCP || row.Protocol == firewallProtocolUDP) {
			args = append(args, "dport", strconv.Itoa(*row.DestinationPort))
		}
	}
	args = append(args, "comment", row.NFTComment, row.Action)
	return args
}

func applyFirewallRule(ctx context.Context, row *store.FirewallRule) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := ensureHellingNFTBase(ctx); err != nil {
		return err
	}
	return runNFT(ctx, nftArgsForRule(row)...)
}

func removeFirewallRule(ctx context.Context, row *store.FirewallRule) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var handle int64
	if row.NFTHandle != nil {
		handle = *row.NFTHandle
	} else {
		var err error
		handle, err = lookupFirewallHandle(ctx, row)
		if err != nil {
			return err
		}
	}
	return runNFT(ctx, "delete", "rule", "inet", "helling", row.Direction, "handle", strconv.FormatInt(handle, 10))
}

func lookupFirewallHandle(ctx context.Context, row *store.FirewallRule) (int64, error) {
	out, err := runFirewallHelperOutput(ctx, "-a", "list", "chain", "inet", "helling", row.Direction)
	if err != nil {
		return 0, fmt.Errorf("listing nft chain: %w", err)
	}
	return parseNFTHandleByComment(strings.NewReader(string(out)), row.NFTComment)
}

func ensureHellingNFTBase(ctx context.Context) error {
	if err := runNFT(ctx, "list", "table", "inet", "helling"); err != nil {
		if err := runNFT(ctx, "add", "table", "inet", "helling"); err != nil {
			return fmt.Errorf("creating nft table: %w", err)
		}
	}
	for _, chain := range []struct {
		name string
		hook string
	}{
		{name: "input", hook: "input"},
		{name: "output", hook: "output"},
		{name: "forward", hook: "forward"},
	} {
		if err := runNFT(ctx, "list", "chain", "inet", "helling", chain.name); err == nil {
			continue
		}
		if err := runNFT(ctx, "add", "chain", "inet", "helling", chain.name, "{", "type", "filter", "hook", chain.hook, "priority", "0", ";", "policy", "accept", ";", "}"); err != nil {
			return fmt.Errorf("creating nft chain %s: %w", chain.name, err)
		}
	}
	return nil
}

func runNFT(ctx context.Context, args ...string) error {
	return runFirewallHelper(ctx, args...)
}

func runFirewallHelper(ctx context.Context, args ...string) error {
	// #nosec G204 -- helper path is fixed and helper revalidates every nft argv token before executing nft.
	return exec.CommandContext(ctx, firewallHelperPath, args...).Run()
}

func runFirewallHelperOutput(ctx context.Context, args ...string) ([]byte, error) {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "-a" {
			continue
		}
		filtered = append(filtered, arg)
	}
	if err := hostfirewall.ValidateNFTArgs(filtered); err != nil {
		return nil, err
	}
	// #nosec G204 -- helper path is fixed and helper revalidates every nft argv token before executing nft.
	return exec.CommandContext(ctx, firewallHelperPath, args...).Output()
}

func parseNFTHandleByComment(r io.Reader, comment string) (int64, error) {
	scanner := bufio.NewScanner(r)
	quotedComment := `comment "` + comment + `"`
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, quotedComment) {
			continue
		}
		idx := strings.LastIndex(line, "# handle ")
		if idx < 0 {
			return 0, fmt.Errorf("nft rule %q has no handle", comment)
		}
		fields := strings.Fields(line[idx+len("# handle "):])
		if len(fields) == 0 {
			return 0, fmt.Errorf("nft rule %q has empty handle", comment)
		}
		handle, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parsing nft handle for %q: %w", comment, err)
		}
		return handle, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("reading nft ruleset: %w", err)
	}
	return 0, fmt.Errorf("nft rule with comment %q not found", comment)
}

func firewallRuleToResponse(row *store.FirewallRule) firewallRuleResponse {
	return firewallRuleResponse{
		ID:              row.ID,
		Direction:       row.Direction,
		Action:          row.Action,
		Protocol:        row.Protocol,
		SourceCIDR:      row.SourceCIDR,
		DestinationCIDR: row.DestinationCIDR,
		DestinationPort: row.DestinationPort,
		Enabled:         row.Enabled,
		Comment:         row.Comment,
		NFTComment:      row.NFTComment,
		NFTHandle:       row.NFTHandle,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
