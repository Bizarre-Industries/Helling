package api

import (
	"context"
	"net/http"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
)

// CertIssuer issues a fresh per-user mTLS certificate at user-creation time
// per ADR-024. Implementations age-encrypt the PEM artifacts and persist
// them via authrepo.InsertUserCertificate. A nil CertIssuer disables PKI
// issuance (dev environments without HELLING_CA_DIR set).
type CertIssuer interface {
	IssueForUser(ctx context.Context, userID, username string) error
}

// Deps carries optional runtime dependencies for Helling API handlers.
// A zero Deps keeps the stubbed handlers, which is what the unit tests for
// the Huma spike rely on. Production wiring passes the real services.
type Deps struct {
	// Auth is the auth service powering setup/login/logout/refresh.
	// When nil, stub handlers are registered instead.
	Auth *auth.Service
	// IncusProxy handles /api/incus/* when non-nil (ADR-014).
	IncusProxy http.Handler
	// PodmanProxy handles /api/podman/* when non-nil.
	PodmanProxy http.Handler
	// CertIssuer issues per-user mTLS certificates on userCreate when non-nil.
	CertIssuer CertIssuer
}

// HasAuth reports whether a real auth service is wired in.
func (d Deps) HasAuth() bool { return d.Auth != nil }

// HasIncusProxy reports whether the Incus reverse-proxy is wired.
func (d Deps) HasIncusProxy() bool { return d.IncusProxy != nil }

// HasPodmanProxy reports whether the Podman reverse-proxy is wired.
func (d Deps) HasPodmanProxy() bool { return d.PodmanProxy != nil }

// HasCertIssuer reports whether per-user PKI issuance is wired.
func (d Deps) HasCertIssuer() bool { return d.CertIssuer != nil }
