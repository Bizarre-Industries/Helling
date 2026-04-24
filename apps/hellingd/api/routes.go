// Package api defines Huma operations for Helling-owned routes.
package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const (
	authUsernameAdmin     = "admin"
	authPasswordAdmin     = "correct-horse-battery-staple"
	authUsernameMFA       = "mfa"
	authUsernameRateLimit = "ratelimit"
)

// HealthData is the minimal payload used to prove the Huma pipeline wiring.
type HealthData struct {
	Status string `json:"status" doc:"Service health state." enum:"ok"`
}

// HealthMeta keeps the envelope shape aligned with docs/spec/api.md.
type HealthMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// HealthEnvelope follows the Helling success envelope contract.
type HealthEnvelope struct {
	Data HealthData `json:"data"`
	Meta HealthMeta `json:"meta"`
}

// HealthOutput is the response shape for GET /api/v1/health.
type HealthOutput struct {
	Body HealthEnvelope
}

// AuthLoginRequest is the login request payload for auth login.
type AuthLoginRequest struct {
	Username string `json:"username" minLength:"1" maxLength:"64" doc:"Username for PAM authentication."`
	Password string `json:"password" minLength:"1" maxLength:"256" doc:"Password for PAM authentication."`
	TOTPCode string `json:"totp_code,omitempty" minLength:"6" maxLength:"8" doc:"Optional TOTP code for MFA completion."`
}

// AuthLoginInput wraps the auth login request body.
type AuthLoginInput struct {
	Body AuthLoginRequest `doc:"PAM credentials with optional inline TOTP code."`
}

// AuthLoginData is the result payload for auth login.
type AuthLoginData struct {
	AccessToken string `json:"access_token,omitempty" doc:"JWT access token when login succeeds without MFA challenge."`
	TokenType   string `json:"token_type,omitempty" doc:"Token scheme for access token responses." enum:"Bearer"`
	ExpiresIn   int    `json:"expires_in,omitempty" doc:"Access token TTL in seconds." minimum:"1"`
	MFARequired bool   `json:"mfa_required,omitempty" doc:"Indicates whether MFA completion is required before token issuance."`
	MFAToken    string `json:"mfa_token,omitempty" doc:"Opaque token used to complete MFA challenge."`
}

// AuthLoginMeta contains request metadata for auth responses.
type AuthLoginMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthLoginEnvelope follows the Helling success envelope shape for auth login.
type AuthLoginEnvelope struct {
	Data AuthLoginData `json:"data"`
	Meta AuthLoginMeta `json:"meta"`
}

// AuthLoginOutput supports 200 and 202 responses for auth login.
type AuthLoginOutput struct {
	Status int `status:"200"`
	Body   AuthLoginEnvelope
}

// UserListInput contains pagination controls for listing users.
type UserListInput struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of users to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// UserRecord is a lightweight user summary payload.
type UserRecord struct {
	ID       string `json:"id" doc:"User identifier."`
	Username string `json:"username" doc:"Username."`
	Role     string `json:"role" doc:"Assigned role." enum:"admin,user,auditor"`
	Status   string `json:"status" doc:"Account status." enum:"active,disabled"`
}

// UserPageMeta is pagination metadata for list endpoints.
type UserPageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// UserListMeta captures request and paging metadata for user list responses.
type UserListMeta struct {
	RequestID string       `json:"request_id" doc:"Request correlation ID."`
	Page      UserPageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// UserListEnvelope follows the Helling list envelope shape.
type UserListEnvelope struct {
	Data []UserRecord `json:"data"`
	Meta UserListMeta `json:"meta"`
}

// UserListOutput is the response shape for GET /api/v1/users.
type UserListOutput struct {
	Body UserListEnvelope
}

var stubUsers = []UserRecord{
	{ID: "user_admin", Username: "admin", Role: "admin", Status: "active"},
	{ID: "user_alice", Username: "alice", Role: "user", Status: "active"},
}

const userIDUnknown = "user_missing"

const (
	scheduleIDExisting = "sched_01JZSCHEDULE00000000001"
	scheduleIDUnknown  = "sched_missing"
)

// ScheduleRecord is a schedule summary.
type ScheduleRecord struct {
	ID       string `json:"id" doc:"Schedule identifier."`
	Name     string `json:"name" doc:"Schedule name."`
	Type     string `json:"type" doc:"Schedule kind." enum:"backup,snapshot"`
	Target   string `json:"target" doc:"Target Incus/Podman resource identifier."`
	CronExpr string `json:"cron_expr" doc:"Systemd OnCalendar expression (ADR-017)."`
	Enabled  bool   `json:"enabled" doc:"Whether the underlying systemd timer is enabled."`
	NextRun  string `json:"next_run,omitempty" doc:"Next scheduled run timestamp (ISO-8601)." format:"date-time"`
}

// SchedulePageMeta is pagination metadata for schedule lists.
type SchedulePageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// ScheduleListInput has pagination controls.
type ScheduleListInput struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of schedules to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// ScheduleListMeta contains request and paging metadata.
type ScheduleListMeta struct {
	RequestID string           `json:"request_id" doc:"Request correlation ID."`
	Page      SchedulePageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// ScheduleListEnvelope follows the list envelope shape.
type ScheduleListEnvelope struct {
	Data []ScheduleRecord `json:"data"`
	Meta ScheduleListMeta `json:"meta"`
}

// ScheduleListOutput is the response shape for GET /api/v1/schedules.
type ScheduleListOutput struct {
	Body ScheduleListEnvelope
}

// ScheduleCreateRequest creates a new schedule.
type ScheduleCreateRequest struct {
	Name     string `json:"name" minLength:"1" maxLength:"128" doc:"Schedule display name."`
	Type     string `json:"type" doc:"Schedule kind." enum:"backup,snapshot"`
	Target   string `json:"target" minLength:"1" maxLength:"256" doc:"Target Incus/Podman resource identifier."`
	CronExpr string `json:"cron_expr" minLength:"1" maxLength:"256" doc:"Systemd OnCalendar expression."`
}

// ScheduleCreateInput wraps the create body.
type ScheduleCreateInput struct {
	Body ScheduleCreateRequest `doc:"Schedule creation payload."`
}

// ScheduleCreateData returns the new schedule summary.
type ScheduleCreateData = ScheduleRecord

// ScheduleCreateMeta contains request metadata.
type ScheduleCreateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// ScheduleCreateEnvelope follows the success envelope shape.
type ScheduleCreateEnvelope struct {
	Data ScheduleCreateData `json:"data"`
	Meta ScheduleCreateMeta `json:"meta"`
}

// ScheduleCreateOutput is the response shape for POST /api/v1/schedules.
type ScheduleCreateOutput struct {
	Body ScheduleCreateEnvelope
}

// ScheduleGetInput binds the path id.
type ScheduleGetInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Schedule identifier."`
}

// ScheduleGetMeta contains request metadata.
type ScheduleGetMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// ScheduleGetEnvelope follows the success envelope shape.
type ScheduleGetEnvelope struct {
	Data ScheduleRecord  `json:"data"`
	Meta ScheduleGetMeta `json:"meta"`
}

// ScheduleGetOutput is the response shape for GET /api/v1/schedules/{id}.
type ScheduleGetOutput struct {
	Body ScheduleGetEnvelope
}

// ScheduleUpdateRequest applies a partial update.
type ScheduleUpdateRequest struct {
	Name     string `json:"name,omitempty" maxLength:"128" doc:"New name."`
	CronExpr string `json:"cron_expr,omitempty" maxLength:"256" doc:"New systemd OnCalendar expression."`
	Enabled  *bool  `json:"enabled,omitempty" doc:"Enable or disable the timer without deleting it."`
}

// ScheduleUpdateInput combines path id with update body.
type ScheduleUpdateInput struct {
	ID   string                `path:"id" minLength:"1" maxLength:"64" doc:"Schedule identifier."`
	Body ScheduleUpdateRequest `doc:"Partial update payload."`
}

// ScheduleUpdateMeta contains request metadata.
type ScheduleUpdateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// ScheduleUpdateEnvelope follows the success envelope shape.
type ScheduleUpdateEnvelope struct {
	Data ScheduleRecord     `json:"data"`
	Meta ScheduleUpdateMeta `json:"meta"`
}

// ScheduleUpdateOutput is the response shape for PATCH /api/v1/schedules/{id}.
type ScheduleUpdateOutput struct {
	Body ScheduleUpdateEnvelope
}

// ScheduleDeleteInput binds the path id.
type ScheduleDeleteInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Schedule identifier."`
}

// ScheduleDeleteData is empty on success.
type ScheduleDeleteData struct{}

// ScheduleDeleteMeta contains request metadata.
type ScheduleDeleteMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// ScheduleDeleteEnvelope follows the success envelope shape.
type ScheduleDeleteEnvelope struct {
	Data ScheduleDeleteData `json:"data"`
	Meta ScheduleDeleteMeta `json:"meta"`
}

// ScheduleDeleteOutput is the response shape for DELETE /api/v1/schedules/{id}.
type ScheduleDeleteOutput struct {
	Body ScheduleDeleteEnvelope
}

// ScheduleRunNowInput binds the path id.
type ScheduleRunNowInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Schedule identifier."`
}

// ScheduleRunNowData returns the run status.
type ScheduleRunNowData struct {
	ID      string `json:"id" doc:"Schedule identifier."`
	Status  string `json:"status" doc:"Run status." enum:"triggered"`
	StartAt string `json:"start_at" doc:"Run start timestamp (ISO-8601)." format:"date-time"`
}

// ScheduleRunNowMeta contains request metadata.
type ScheduleRunNowMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// ScheduleRunNowEnvelope follows the success envelope shape.
type ScheduleRunNowEnvelope struct {
	Data ScheduleRunNowData `json:"data"`
	Meta ScheduleRunNowMeta `json:"meta"`
}

// ScheduleRunNowOutput is the response shape for POST /api/v1/schedules/{id}/run.
type ScheduleRunNowOutput struct {
	Body ScheduleRunNowEnvelope
}

var stubSchedules = []ScheduleRecord{
	{ID: scheduleIDExisting, Name: "nightly-backup", Type: "backup", Target: "vm-web", CronExpr: "*-*-* 02:00:00", Enabled: true, NextRun: "2026-04-24T02:00:00Z"},
	{ID: "sched_01JZSCHEDULE00000000002", Name: "hourly-snapshot", Type: "snapshot", Target: "vm-db", CronExpr: "*-*-* *:00:00", Enabled: true, NextRun: "2026-04-23T20:00:00Z"},
}

const (
	webhookIDExisting = "whk_01JZWEBHOOK0000000000001"
	webhookIDUnknown  = "whk_missing"
)

// WebhookRecord is a webhook summary.
type WebhookRecord struct {
	ID       string   `json:"id" doc:"Webhook identifier."`
	Name     string   `json:"name" doc:"Webhook name."`
	URL      string   `json:"url" doc:"Destination URL."`
	Events   []string `json:"events" doc:"Event types this webhook subscribes to."`
	Enabled  bool     `json:"enabled" doc:"Whether deliveries are currently active."`
	LastSent string   `json:"last_sent,omitempty" doc:"ISO-8601 timestamp of last delivery attempt." format:"date-time"`
}

// WebhookPageMeta is pagination metadata for webhook lists.
type WebhookPageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// WebhookListInput has pagination controls.
type WebhookListInput struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of webhooks to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// WebhookListMeta contains request and paging metadata.
type WebhookListMeta struct {
	RequestID string          `json:"request_id" doc:"Request correlation ID."`
	Page      WebhookPageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// WebhookListEnvelope follows the list envelope shape.
type WebhookListEnvelope struct {
	Data []WebhookRecord `json:"data"`
	Meta WebhookListMeta `json:"meta"`
}

// WebhookListOutput is the response shape for GET /api/v1/webhooks.
type WebhookListOutput struct {
	Body WebhookListEnvelope
}

// WebhookCreateRequest creates a new webhook.
type WebhookCreateRequest struct {
	Name   string   `json:"name" minLength:"1" maxLength:"128" doc:"Webhook name."`
	URL    string   `json:"url" minLength:"1" maxLength:"2048" doc:"Destination URL. HTTPS recommended; HTTP allowed only for loopback targets."`
	Secret string   `json:"secret" minLength:"16" maxLength:"256" doc:"HMAC signing secret. Stored encrypted."`
	Events []string `json:"events" minItems:"1" doc:"Event types to subscribe to."`
}

// WebhookCreateInput wraps the create body.
type WebhookCreateInput struct {
	Body WebhookCreateRequest `doc:"Webhook creation payload."`
}

// WebhookCreateMeta contains request metadata.
type WebhookCreateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// WebhookCreateEnvelope follows the success envelope shape.
type WebhookCreateEnvelope struct {
	Data WebhookRecord     `json:"data"`
	Meta WebhookCreateMeta `json:"meta"`
}

// WebhookCreateOutput is the response shape for POST /api/v1/webhooks.
type WebhookCreateOutput struct {
	Body WebhookCreateEnvelope
}

// WebhookGetInput binds the path id.
type WebhookGetInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Webhook identifier."`
}

// WebhookGetMeta contains request metadata.
type WebhookGetMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// WebhookGetEnvelope follows the success envelope shape.
type WebhookGetEnvelope struct {
	Data WebhookRecord  `json:"data"`
	Meta WebhookGetMeta `json:"meta"`
}

// WebhookGetOutput is the response shape for GET /api/v1/webhooks/{id}.
type WebhookGetOutput struct {
	Body WebhookGetEnvelope
}

// WebhookUpdateRequest applies a partial update.
type WebhookUpdateRequest struct {
	Name    string   `json:"name,omitempty" maxLength:"128" doc:"New name."`
	URL     string   `json:"url,omitempty" maxLength:"2048" doc:"New destination URL."`
	Events  []string `json:"events,omitempty" doc:"New event subscription list; replaces the existing set."`
	Enabled *bool    `json:"enabled,omitempty" doc:"Enable or disable without deleting."`
}

// WebhookUpdateInput combines path id with update body.
type WebhookUpdateInput struct {
	ID   string               `path:"id" minLength:"1" maxLength:"64" doc:"Webhook identifier."`
	Body WebhookUpdateRequest `doc:"Partial update payload."`
}

// WebhookUpdateMeta contains request metadata.
type WebhookUpdateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// WebhookUpdateEnvelope follows the success envelope shape.
type WebhookUpdateEnvelope struct {
	Data WebhookRecord     `json:"data"`
	Meta WebhookUpdateMeta `json:"meta"`
}

// WebhookUpdateOutput is the response shape for PATCH /api/v1/webhooks/{id}.
type WebhookUpdateOutput struct {
	Body WebhookUpdateEnvelope
}

// WebhookDeleteInput binds the path id.
type WebhookDeleteInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Webhook identifier."`
}

// WebhookDeleteData is empty on success.
type WebhookDeleteData struct{}

// WebhookDeleteMeta contains request metadata.
type WebhookDeleteMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// WebhookDeleteEnvelope follows the success envelope shape.
type WebhookDeleteEnvelope struct {
	Data WebhookDeleteData `json:"data"`
	Meta WebhookDeleteMeta `json:"meta"`
}

// WebhookDeleteOutput is the response shape for DELETE /api/v1/webhooks/{id}.
type WebhookDeleteOutput struct {
	Body WebhookDeleteEnvelope
}

// WebhookTestInput binds the path id.
type WebhookTestInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Webhook identifier."`
}

// WebhookTestData returns the synthetic delivery status.
type WebhookTestData struct {
	ID         string `json:"id" doc:"Webhook identifier."`
	Status     string `json:"status" doc:"Synthetic delivery status." enum:"delivered,failed"`
	StatusCode int    `json:"status_code,omitempty" doc:"HTTP status code returned by the destination." minimum:"100" maximum:"599"`
	Latency    int    `json:"latency_ms,omitempty" doc:"Observed latency in milliseconds." minimum:"0"`
}

// WebhookTestMeta contains request metadata.
type WebhookTestMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// WebhookTestEnvelope follows the success envelope shape.
type WebhookTestEnvelope struct {
	Data WebhookTestData `json:"data"`
	Meta WebhookTestMeta `json:"meta"`
}

// WebhookTestOutput is the response shape for POST /api/v1/webhooks/{id}/test.
type WebhookTestOutput struct {
	Body WebhookTestEnvelope
}

var stubWebhooks = []WebhookRecord{
	{ID: webhookIDExisting, Name: "slack-incidents", URL: "https://hooks.slack.com/stub", Events: []string{"instance.created", "instance.deleted"}, Enabled: true, LastSent: "2026-04-22T10:00:00Z"},
	{ID: "whk_01JZWEBHOOK0000000000002", Name: "discord-alerts", URL: "https://discord.com/api/webhooks/stub", Events: []string{"schedule.failed"}, Enabled: false},
}

const (
	kubernetesNameExisting = "prod-cluster"
	kubernetesNameUnknown  = "missing-cluster"
)

// KubernetesRecord is a k3s cluster summary.
type KubernetesRecord struct {
	Name         string `json:"name" doc:"Cluster name (also the URL-path identifier)."`
	Version      string `json:"version" doc:"k3s version."`
	Status       string `json:"status" doc:"Cluster status." enum:"provisioning,ready,upgrading,error"`
	Workers      int    `json:"workers" doc:"Declared worker count." minimum:"0"`
	ReadyWorkers int    `json:"ready_workers" doc:"Workers currently ready." minimum:"0"`
	CreatedAt    string `json:"created_at" doc:"ISO-8601 creation timestamp." format:"date-time"`
}

// KubernetesListInput has pagination controls.
type KubernetesListInput struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of clusters to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// KubernetesPageMeta is pagination metadata for cluster lists.
type KubernetesPageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// KubernetesListMeta contains request and paging metadata.
type KubernetesListMeta struct {
	RequestID string             `json:"request_id" doc:"Request correlation ID."`
	Page      KubernetesPageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// KubernetesListEnvelope follows the list envelope shape.
type KubernetesListEnvelope struct {
	Data []KubernetesRecord `json:"data"`
	Meta KubernetesListMeta `json:"meta"`
}

// KubernetesListOutput is the response shape for GET /api/v1/kubernetes.
type KubernetesListOutput struct {
	Body KubernetesListEnvelope
}

// KubernetesCreateRequest creates a k3s cluster.
type KubernetesCreateRequest struct {
	Name    string `json:"name" minLength:"1" maxLength:"64" pattern:"^[a-z0-9-]+$" doc:"Cluster name. Lowercase alphanumerics and hyphens."`
	Version string `json:"version" minLength:"1" maxLength:"32" doc:"Requested k3s version (e.g. v1.30.5+k3s1)."`
	Workers int    `json:"workers" minimum:"0" maximum:"64" doc:"Worker count to provision."`
}

// KubernetesCreateInput wraps the create body.
type KubernetesCreateInput struct {
	Body KubernetesCreateRequest `doc:"Cluster creation payload."`
}

// KubernetesCreateMeta contains request metadata.
type KubernetesCreateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// KubernetesCreateEnvelope follows the success envelope shape.
type KubernetesCreateEnvelope struct {
	Data KubernetesRecord     `json:"data"`
	Meta KubernetesCreateMeta `json:"meta"`
}

// KubernetesCreateOutput is the response shape for POST /api/v1/kubernetes.
type KubernetesCreateOutput struct {
	Body KubernetesCreateEnvelope
}

// KubernetesGetInput binds the path name.
type KubernetesGetInput struct {
	Name string `path:"name" minLength:"1" maxLength:"64" doc:"Cluster name."`
}

// KubernetesGetMeta contains request metadata.
type KubernetesGetMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// KubernetesGetEnvelope follows the success envelope shape.
type KubernetesGetEnvelope struct {
	Data KubernetesRecord  `json:"data"`
	Meta KubernetesGetMeta `json:"meta"`
}

// KubernetesGetOutput is the response shape for GET /api/v1/kubernetes/{name}.
type KubernetesGetOutput struct {
	Body KubernetesGetEnvelope
}

// KubernetesDeleteInput binds the path name.
type KubernetesDeleteInput struct {
	Name string `path:"name" minLength:"1" maxLength:"64" doc:"Cluster name."`
}

// KubernetesDeleteData is empty on success.
type KubernetesDeleteData struct{}

// KubernetesDeleteMeta contains request metadata.
type KubernetesDeleteMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// KubernetesDeleteEnvelope follows the success envelope shape.
type KubernetesDeleteEnvelope struct {
	Data KubernetesDeleteData `json:"data"`
	Meta KubernetesDeleteMeta `json:"meta"`
}

// KubernetesDeleteOutput is the response shape for DELETE /api/v1/kubernetes/{name}.
type KubernetesDeleteOutput struct {
	Body KubernetesDeleteEnvelope
}

// KubernetesScaleRequest specifies the desired worker count.
type KubernetesScaleRequest struct {
	Workers int `json:"workers" minimum:"0" maximum:"64" doc:"Desired worker count."`
}

// KubernetesScaleInput combines path name with scale body.
type KubernetesScaleInput struct {
	Name string                 `path:"name" minLength:"1" maxLength:"64" doc:"Cluster name."`
	Body KubernetesScaleRequest `doc:"Scale target payload."`
}

// KubernetesScaleMeta contains request metadata.
type KubernetesScaleMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// KubernetesScaleEnvelope follows the success envelope shape.
type KubernetesScaleEnvelope struct {
	Data KubernetesRecord    `json:"data"`
	Meta KubernetesScaleMeta `json:"meta"`
}

// KubernetesScaleOutput is the response shape for PATCH /api/v1/kubernetes/{name}/scale.
type KubernetesScaleOutput struct {
	Body KubernetesScaleEnvelope
}

// KubernetesUpgradeRequest specifies a target k3s version.
type KubernetesUpgradeRequest struct {
	Version string `json:"version" minLength:"1" maxLength:"32" doc:"Target k3s version."`
}

// KubernetesUpgradeInput combines path name with upgrade body.
type KubernetesUpgradeInput struct {
	Name string                   `path:"name" minLength:"1" maxLength:"64" doc:"Cluster name."`
	Body KubernetesUpgradeRequest `doc:"Upgrade target payload."`
}

// KubernetesUpgradeMeta contains request metadata.
type KubernetesUpgradeMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// KubernetesUpgradeEnvelope follows the success envelope shape.
type KubernetesUpgradeEnvelope struct {
	Data KubernetesRecord      `json:"data"`
	Meta KubernetesUpgradeMeta `json:"meta"`
}

// KubernetesUpgradeOutput is the response shape for POST /api/v1/kubernetes/{name}/upgrade.
type KubernetesUpgradeOutput struct {
	Body KubernetesUpgradeEnvelope
}

// KubernetesKubeconfigInput binds the path name.
type KubernetesKubeconfigInput struct {
	Name string `path:"name" minLength:"1" maxLength:"64" doc:"Cluster name."`
}

// KubernetesKubeconfigData returns the kubeconfig inline.
type KubernetesKubeconfigData struct {
	Kubeconfig string `json:"kubeconfig" doc:"kubeconfig YAML contents."`
}

// KubernetesKubeconfigMeta contains request metadata.
type KubernetesKubeconfigMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// KubernetesKubeconfigEnvelope follows the success envelope shape.
type KubernetesKubeconfigEnvelope struct {
	Data KubernetesKubeconfigData `json:"data"`
	Meta KubernetesKubeconfigMeta `json:"meta"`
}

// KubernetesKubeconfigOutput is the response shape for GET /api/v1/kubernetes/{name}/kubeconfig.
type KubernetesKubeconfigOutput struct {
	Body KubernetesKubeconfigEnvelope
}

var stubKubernetesClusters = []KubernetesRecord{
	{Name: kubernetesNameExisting, Version: "v1.30.5+k3s1", Status: "ready", Workers: 3, ReadyWorkers: 3, CreatedAt: "2026-03-01T00:00:00Z"},
	{Name: "staging-cluster", Version: "v1.30.5+k3s1", Status: "ready", Workers: 1, ReadyWorkers: 1, CreatedAt: "2026-03-15T00:00:00Z"},
}

const kubernetesStubKubeconfig = `apiVersion: v1
kind: Config
clusters:
  - cluster:
      server: https://127.0.0.1:6443
    name: helling-stub
contexts:
  - context:
      cluster: helling-stub
      user: helling-stub
    name: helling-stub
current-context: helling-stub
users:
  - name: helling-stub
    user:
      token: stub-kubeconfig-token
`

// UserCreateRequest creates a new PAM-backed user account.
type UserCreateRequest struct {
	Username string `json:"username" minLength:"1" maxLength:"64" doc:"Unix-safe username. PAM constraints apply."`
	Role     string `json:"role" doc:"Fixed role assignment." enum:"admin,user,auditor"`
	Password string `json:"password" minLength:"8" maxLength:"256" doc:"Initial password, fed to passwd(1). Optional when delegating to external provisioning."`
}

// UserCreateInput wraps the create body.
type UserCreateInput struct {
	Body UserCreateRequest `doc:"User creation payload."`
}

// UserCreateData returns the created user summary.
type UserCreateData struct {
	ID       string `json:"id" doc:"New user identifier."`
	Username string `json:"username" doc:"Username."`
	Role     string `json:"role" doc:"Assigned role." enum:"admin,user,auditor"`
	Status   string `json:"status" doc:"Account status." enum:"active,disabled"`
}

// UserCreateMeta contains request metadata.
type UserCreateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserCreateEnvelope follows the success envelope shape.
type UserCreateEnvelope struct {
	Data UserCreateData `json:"data"`
	Meta UserCreateMeta `json:"meta"`
}

// UserCreateOutput is the response shape for POST /api/v1/users.
type UserCreateOutput struct {
	Body UserCreateEnvelope
}

// UserGetInput binds the path id.
type UserGetInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"User identifier."`
}

// UserGetData returns detailed user fields including MFA status.
type UserGetData struct {
	ID          string `json:"id" doc:"User identifier."`
	Username    string `json:"username" doc:"Username."`
	Role        string `json:"role" doc:"Assigned role." enum:"admin,user,auditor"`
	Status      string `json:"status" doc:"Account status." enum:"active,disabled"`
	TotpEnabled bool   `json:"totp_enabled" doc:"Whether TOTP is currently enrolled."`
	LastLogin   string `json:"last_login,omitempty" doc:"ISO-8601 timestamp of last successful login." format:"date-time"`
}

// UserGetMeta contains request metadata.
type UserGetMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserGetEnvelope follows the success envelope shape.
type UserGetEnvelope struct {
	Data UserGetData `json:"data"`
	Meta UserGetMeta `json:"meta"`
}

// UserGetOutput is the response shape for GET /api/v1/users/{id}.
type UserGetOutput struct {
	Body UserGetEnvelope
}

// UserUpdateRequest applies a partial update to a user.
type UserUpdateRequest struct {
	Role   string `json:"role,omitempty" doc:"New role assignment." enum:"admin,user,auditor"`
	Status string `json:"status,omitempty" doc:"New account status." enum:"active,disabled"`
}

// UserUpdateInput combines path id with update body.
type UserUpdateInput struct {
	ID   string            `path:"id" minLength:"1" maxLength:"64" doc:"User identifier."`
	Body UserUpdateRequest `doc:"Partial update payload."`
}

// UserUpdateData returns the updated user summary.
type UserUpdateData struct {
	ID       string `json:"id" doc:"User identifier."`
	Username string `json:"username" doc:"Username."`
	Role     string `json:"role" doc:"Assigned role." enum:"admin,user,auditor"`
	Status   string `json:"status" doc:"Account status." enum:"active,disabled"`
}

// UserUpdateMeta contains request metadata.
type UserUpdateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserUpdateEnvelope follows the success envelope shape.
type UserUpdateEnvelope struct {
	Data UserUpdateData `json:"data"`
	Meta UserUpdateMeta `json:"meta"`
}

// UserUpdateOutput is the response shape for PATCH /api/v1/users/{id}.
type UserUpdateOutput struct {
	Body UserUpdateEnvelope
}

// UserDeleteInput binds the path id.
type UserDeleteInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"User identifier."`
}

// UserDeleteData is empty on success.
type UserDeleteData struct{}

// UserDeleteMeta contains request metadata.
type UserDeleteMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserDeleteEnvelope follows the success envelope shape.
type UserDeleteEnvelope struct {
	Data UserDeleteData `json:"data"`
	Meta UserDeleteMeta `json:"meta"`
}

// UserDeleteOutput is the response shape for DELETE /api/v1/users/{id}.
type UserDeleteOutput struct {
	Body UserDeleteEnvelope
}

// UserSetScopeRequest applies an Incus trust-scope change.
type UserSetScopeRequest struct {
	Scope string `json:"scope" doc:"Incus trust scope. Controls the set of Incus projects/resources the user's cert is allowed to touch." enum:"default,restricted,admin"`
}

// UserSetScopeInput combines path id with scope body.
type UserSetScopeInput struct {
	ID   string              `path:"id" minLength:"1" maxLength:"64" doc:"User identifier."`
	Body UserSetScopeRequest `doc:"Trust-scope assignment payload."`
}

// UserSetScopeData returns the updated scope.
type UserSetScopeData struct {
	ID    string `json:"id" doc:"User identifier."`
	Scope string `json:"scope" doc:"Applied trust scope." enum:"default,restricted,admin"`
}

// UserSetScopeMeta contains request metadata.
type UserSetScopeMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// UserSetScopeEnvelope follows the success envelope shape.
type UserSetScopeEnvelope struct {
	Data UserSetScopeData `json:"data"`
	Meta UserSetScopeMeta `json:"meta"`
}

// UserSetScopeOutput is the response shape for PUT /api/v1/users/{id}/scope.
type UserSetScopeOutput struct {
	Body UserSetScopeEnvelope
}

// AuthLogoutData is the payload returned on successful logout.
// Empty object preserves the envelope contract (data + meta) without leaking session material.
type AuthLogoutData struct{}

// AuthLogoutMeta contains request metadata for logout responses.
type AuthLogoutMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthLogoutEnvelope follows the Helling success envelope shape for logout.
type AuthLogoutEnvelope struct {
	Data AuthLogoutData `json:"data"`
	Meta AuthLogoutMeta `json:"meta"`
}

// AuthLogoutOutput is the response shape for POST /api/v1/auth/logout.
type AuthLogoutOutput struct {
	Body AuthLogoutEnvelope
}

// AuthRefreshRequest is the refresh request payload.
// v0.1-alpha accepts the refresh token in the body; v0.1-beta moves to the httpOnly
// cookie model documented in docs/spec/auth.md §2.2.
type AuthRefreshRequest struct {
	RefreshToken string `json:"refresh_token" minLength:"1" maxLength:"4096" doc:"Opaque refresh token issued by a prior login."`
}

// AuthRefreshInput wraps the refresh request body.
type AuthRefreshInput struct {
	Body AuthRefreshRequest `doc:"Refresh token exchange payload."`
}

// AuthRefreshData is the result payload for refresh.
type AuthRefreshData struct {
	AccessToken string `json:"access_token" doc:"New JWT access token."`
	TokenType   string `json:"token_type" doc:"Token scheme for access token responses." enum:"Bearer"`
	ExpiresIn   int    `json:"expires_in" doc:"Access token TTL in seconds." minimum:"1"`
}

// AuthRefreshMeta contains request metadata for refresh responses.
type AuthRefreshMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthRefreshEnvelope follows the Helling success envelope shape for refresh.
type AuthRefreshEnvelope struct {
	Data AuthRefreshData `json:"data"`
	Meta AuthRefreshMeta `json:"meta"`
}

// AuthRefreshOutput is the response shape for POST /api/v1/auth/refresh.
type AuthRefreshOutput struct {
	Body AuthRefreshEnvelope
}

const (
	cursorPage2 = "cursor_page_2"

	authRefreshTokenStub  = "stub_refresh_token_01JZREFRESHABCDEFGHJK"
	authRefreshTokenInval = "invalid"
	authMfaTokenStub      = "mfa_01JZABC0123456789ABCDEFGJK" //nolint:gosec // G101: stub fixture, not a real credential.
	authTotpCodeValid     = "123456"
	authTokenIDExisting   = "tok_01JZTOKEN000000000000001" //nolint:gosec // G101: stub path parameter, not a real credential.
	authTokenIDUnknown    = "tok_missing"                  //nolint:gosec // G101: stub path parameter, not a real credential.
)

// AuthSetupRequest is the initial-admin-setup payload.
type AuthSetupRequest struct {
	Username string `json:"username" minLength:"1" maxLength:"64" doc:"Username for the initial admin account."`
	Password string `json:"password" minLength:"8" maxLength:"256" doc:"Password for the initial admin account."`
}

// AuthSetupInput wraps the setup request body.
type AuthSetupInput struct {
	Body AuthSetupRequest `doc:"Initial admin account credentials."`
}

// AuthSetupData mirrors AuthLoginData on successful first-time setup.
type AuthSetupData struct {
	AccessToken string `json:"access_token" doc:"JWT access token issued for the new admin account."`
	TokenType   string `json:"token_type" doc:"Token scheme." enum:"Bearer"`
	ExpiresIn   int    `json:"expires_in" doc:"Access token TTL in seconds." minimum:"1"`
}

// AuthSetupMeta contains request metadata for setup responses.
type AuthSetupMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthSetupEnvelope follows the Helling success envelope shape.
type AuthSetupEnvelope struct {
	Data AuthSetupData `json:"data"`
	Meta AuthSetupMeta `json:"meta"`
}

// AuthSetupOutput is the response shape for POST /api/v1/auth/setup.
type AuthSetupOutput struct {
	Body AuthSetupEnvelope
}

// AuthMfaCompleteRequest completes an MFA challenge with a TOTP code or a
// recovery code (recovery codes are 16-char base32 strings per
// docs/spec/auth.md §2.3).
type AuthMfaCompleteRequest struct {
	MfaToken string `json:"mfa_token" minLength:"1" maxLength:"256" doc:"MFA token returned by the login endpoint."`
	TotpCode string `json:"totp_code" minLength:"6" maxLength:"32" doc:"Six-digit TOTP code or 16-char recovery code."`
}

// AuthMfaCompleteInput wraps the MFA completion body.
type AuthMfaCompleteInput struct {
	Body AuthMfaCompleteRequest `doc:"MFA completion payload."`
}

// AuthMfaCompleteData returns full token pair after successful MFA.
type AuthMfaCompleteData struct {
	AccessToken string `json:"access_token" doc:"JWT access token."`
	TokenType   string `json:"token_type" doc:"Token scheme." enum:"Bearer"`
	ExpiresIn   int    `json:"expires_in" doc:"Access token TTL in seconds." minimum:"1"`
}

// AuthMfaCompleteMeta contains request metadata.
type AuthMfaCompleteMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthMfaCompleteEnvelope follows the Helling success envelope shape.
type AuthMfaCompleteEnvelope struct {
	Data AuthMfaCompleteData `json:"data"`
	Meta AuthMfaCompleteMeta `json:"meta"`
}

// AuthMfaCompleteOutput is the response shape for POST /api/v1/auth/mfa/complete.
type AuthMfaCompleteOutput struct {
	Body AuthMfaCompleteEnvelope
}

// AuthTotpSetupData returns enrollment artifacts for a new TOTP factor.
type AuthTotpSetupData struct {
	ProvisioningURI string   `json:"provisioning_uri" doc:"otpauth:// URI that authenticator apps can scan."`
	Secret          string   `json:"secret" doc:"Raw base32 TOTP secret for manual entry."`
	RecoveryCodes   []string `json:"recovery_codes" doc:"One-time recovery codes for out-of-band access."`
}

// AuthTotpSetupMeta contains request metadata for TOTP setup responses.
type AuthTotpSetupMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTotpSetupEnvelope follows the Helling success envelope shape.
type AuthTotpSetupEnvelope struct {
	Data AuthTotpSetupData `json:"data"`
	Meta AuthTotpSetupMeta `json:"meta"`
}

// AuthTotpSetupOutput is the response shape for POST /api/v1/auth/totp/setup.
type AuthTotpSetupOutput struct {
	Body AuthTotpSetupEnvelope
}

// AuthTotpVerifyRequest confirms a TOTP enrollment with a code.
type AuthTotpVerifyRequest struct {
	TotpCode string `json:"totp_code" minLength:"6" maxLength:"8" doc:"TOTP code from the user's authenticator app."`
}

// AuthTotpVerifyInput wraps the verify request body.
type AuthTotpVerifyInput struct {
	Body AuthTotpVerifyRequest `doc:"TOTP verification payload."`
}

// AuthTotpVerifyData is empty on success; envelope preserved.
type AuthTotpVerifyData struct{}

// AuthTotpVerifyMeta contains request metadata.
type AuthTotpVerifyMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTotpVerifyEnvelope follows the Helling success envelope shape.
type AuthTotpVerifyEnvelope struct {
	Data AuthTotpVerifyData `json:"data"`
	Meta AuthTotpVerifyMeta `json:"meta"`
}

// AuthTotpVerifyOutput is the response shape for POST /api/v1/auth/totp/verify.
type AuthTotpVerifyOutput struct {
	Body AuthTotpVerifyEnvelope
}

// AuthTotpDisableData is empty on success.
type AuthTotpDisableData struct{}

// AuthTotpDisableMeta contains request metadata.
type AuthTotpDisableMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTotpDisableEnvelope follows the Helling success envelope shape.
type AuthTotpDisableEnvelope struct {
	Data AuthTotpDisableData `json:"data"`
	Meta AuthTotpDisableMeta `json:"meta"`
}

// AuthTotpDisableOutput is the response shape for POST /api/v1/auth/totp/disable.
type AuthTotpDisableOutput struct {
	Body AuthTotpDisableEnvelope
}

// AuthTokenRecord is an API token summary. Scope enum matches
// docs/spec/auth.md §5 (`read`, `write`, `admin`).
type AuthTokenRecord struct {
	ID        string `json:"id" doc:"Token identifier."`
	Name      string `json:"name" doc:"User-supplied token name."`
	Scope     string `json:"scope" doc:"Token scope." enum:"read,write,admin"`
	CreatedAt string `json:"created_at" doc:"ISO-8601 timestamp when the token was created." format:"date-time"`
	LastUsed  string `json:"last_used,omitempty" doc:"ISO-8601 timestamp of last successful use, if any." format:"date-time"`
}

// AuthTokenPageMeta is pagination metadata for token lists.
type AuthTokenPageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// AuthTokenListInput has pagination controls.
type AuthTokenListInput struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of tokens to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// AuthTokenListMeta contains request and paging metadata.
type AuthTokenListMeta struct {
	RequestID string            `json:"request_id" doc:"Request correlation ID."`
	Page      AuthTokenPageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// AuthTokenListEnvelope follows the Helling list envelope shape.
type AuthTokenListEnvelope struct {
	Data []AuthTokenRecord `json:"data"`
	Meta AuthTokenListMeta `json:"meta"`
}

// AuthTokenListOutput is the response shape for GET /api/v1/auth/tokens.
type AuthTokenListOutput struct {
	Body AuthTokenListEnvelope
}

// AuthTokenCreateRequest creates a new API token. Scope enum matches
// docs/spec/auth.md §5.
type AuthTokenCreateRequest struct {
	Name  string `json:"name" minLength:"1" maxLength:"128" doc:"User-visible token name."`
	Scope string `json:"scope" doc:"Token scope." enum:"read,write,admin"`
}

// AuthTokenCreateInput wraps the create request body.
type AuthTokenCreateInput struct {
	Body AuthTokenCreateRequest `doc:"Token creation payload."`
}

// AuthTokenCreateData returns the newly-created token plaintext exactly once.
type AuthTokenCreateData struct {
	ID    string `json:"id" doc:"New token identifier."`
	Name  string `json:"name" doc:"Token name."`
	Scope string `json:"scope" doc:"Token scope." enum:"read,write,admin"`
	Token string `json:"token" doc:"Plaintext API token. Surfaced only once; store it securely."`
}

// AuthTokenCreateMeta contains request metadata.
type AuthTokenCreateMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTokenCreateEnvelope follows the Helling success envelope shape.
type AuthTokenCreateEnvelope struct {
	Data AuthTokenCreateData `json:"data"`
	Meta AuthTokenCreateMeta `json:"meta"`
}

// AuthTokenCreateOutput is the response shape for POST /api/v1/auth/tokens.
type AuthTokenCreateOutput struct {
	Body AuthTokenCreateEnvelope
}

// AuthTokenRevokeInput binds the path id.
type AuthTokenRevokeInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Token identifier to revoke."`
}

// AuthTokenRevokeData is empty on success.
type AuthTokenRevokeData struct{}

// AuthTokenRevokeMeta contains request metadata.
type AuthTokenRevokeMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuthTokenRevokeEnvelope follows the Helling success envelope shape.
type AuthTokenRevokeEnvelope struct {
	Data AuthTokenRevokeData `json:"data"`
	Meta AuthTokenRevokeMeta `json:"meta"`
}

// AuthTokenRevokeOutput is the response shape for DELETE /api/v1/auth/tokens/{id}.
type AuthTokenRevokeOutput struct {
	Body AuthTokenRevokeEnvelope
}

var stubAuthTokens = []AuthTokenRecord{
	{ID: authTokenIDExisting, Name: "ci-bot", Scope: "write", CreatedAt: "2026-04-01T00:00:00Z", LastUsed: "2026-04-20T12:00:00Z"},
	{ID: "tok_01JZTOKEN000000000000002", Name: "auditor-readonly", Scope: "read", CreatedAt: "2026-04-10T00:00:00Z"},
}

// RegisterOperations wires the current Huma spike operations with stubbed
// handlers, preserving the Huma-spike test coverage.
func RegisterOperations(api huma.API) {
	RegisterOperationsWith(api, Deps{})
}

// RegisterOperationsWith wires the Huma operations, optionally substituting
// real handlers when deps carries concrete services. Stubs remain the default
// so contract tests keep passing when deps is zero.
func RegisterOperationsWith(api huma.API, deps Deps) {
	if deps.HasAuth() {
		registerAuthSetupReal(api, deps.Auth)
		registerAuthLoginReal(api, deps.Auth)
		registerAuthLogoutReal(api, deps.Auth)
		registerAuthRefreshReal(api, deps.Auth)
		registerAuthMfaCompleteReal(api, deps.Auth)
		registerAuthTotpSetupReal(api, deps.Auth)
		registerAuthTotpVerifyReal(api, deps.Auth)
		registerAuthTotpDisableReal(api, deps.Auth)
		registerAuthTokenListReal(api, deps.Auth)
		registerAuthTokenCreateReal(api, deps.Auth)
		registerAuthTokenRevokeReal(api, deps.Auth)
		registerUserListReal(api, deps.Auth)
		registerUserCreateReal(api, deps.Auth)
		registerUserGetReal(api, deps.Auth)
		registerUserDeleteReal(api, deps.Auth)
		registerUserSetScopeReal(api, deps.Auth)
		registerSystemInfoReal(api, deps.Auth)
		registerSystemHardwareReal(api, deps.Auth)
		registerSystemConfigGetReal(api, deps.Auth)
		registerSystemConfigPutReal(api, deps.Auth)
		registerSystemUpgradeReal(api, deps.Auth)
		registerSystemDiagnosticsReal(api, deps.Auth)
		registerWebhookListReal(api, deps.Auth)
		registerWebhookCreateReal(api, deps.Auth)
		registerWebhookGetReal(api, deps.Auth)
		registerWebhookUpdateReal(api, deps.Auth)
		registerWebhookDeleteReal(api, deps.Auth)
		registerWebhookTestReal(api, deps.Auth)
	} else {
		registerAuthSetup(api)
		registerAuthLogin(api)
		registerAuthLogout(api)
		registerAuthRefresh(api)
		registerAuthMfaComplete(api)
		registerAuthTotpSetup(api)
		registerAuthTotpVerify(api)
		registerAuthTotpDisable(api)
		registerAuthTokenList(api)
		registerAuthTokenCreate(api)
		registerAuthTokenRevoke(api)
		registerUserList(api)
		registerUserCreate(api)
		registerUserGet(api)
		registerUserDelete(api)
		registerUserSetScope(api)
		registerSystemInfo(api)
		registerSystemHardware(api)
		registerSystemConfigGet(api)
		registerSystemConfigPut(api)
		registerSystemUpgrade(api)
		registerSystemDiagnostics(api)
		registerWebhookList(api)
		registerWebhookCreate(api)
		registerWebhookGet(api)
		registerWebhookUpdate(api)
		registerWebhookDelete(api)
		registerWebhookTest(api)
	}
	// userUpdate is stub-only for v0.1; the real handler lands with PAM
	// integration in v0.1-beta. Registering it unconditionally keeps the
	// OpenAPI contract intact without double-registering the real user list
	// handlers.
	registerUserUpdate(api)
	registerScheduleList(api)
	registerScheduleCreate(api)
	registerScheduleGet(api)
	registerScheduleUpdate(api)
	registerScheduleDelete(api)
	registerScheduleRunNow(api)
	registerKubernetesList(api)
	registerKubernetesCreate(api)
	registerKubernetesGet(api)
	registerKubernetesDelete(api)
	registerKubernetesScale(api)
	registerKubernetesUpgrade(api)
	registerKubernetesKubeconfig(api)
	registerFirewallHostList(api)
	registerFirewallHostCreate(api)
	registerFirewallHostDelete(api)
	registerAuditQuery(api)
	registerAuditExport(api)
	registerEventsSse(api)
	registerHealth(api)
}

func registerHealth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "healthGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		Summary:     "Health check",
		Description: "Returns service health for unauthenticated readiness checks.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		_ = ctx
		_ = input

		return &HealthOutput{
			Body: HealthEnvelope{
				Data: HealthData{Status: "ok"},
				Meta: HealthMeta{RequestID: "req_huma_spike"},
			},
		}, nil
	})
}

func registerAuthLogin(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authLogin",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "PAM authenticate and issue JWT pair",
		Description: "Authenticates a PAM user and returns tokens or an MFA challenge.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "PAM credentials with optional inline TOTP code.",
			Required:    true,
		},
		Errors: []int{http.StatusUnauthorized, http.StatusTooManyRequests},
		Responses: map[string]*huma.Response{
			"202": {
				Description: "MFA challenge required before token issuance.",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: &huma.Schema{Ref: "#/components/schemas/AuthLoginEnvelope"},
					},
				},
			},
		},
	}, func(ctx context.Context, input *AuthLoginInput) (*AuthLoginOutput, error) {
		_ = ctx

		if input.Body.Username == authUsernameRateLimit {
			return nil, huma.Error429TooManyRequests("AUTH_RATE_LIMITED")
		}

		if input.Body.Username == authUsernameMFA && input.Body.TOTPCode == "" {
			return &AuthLoginOutput{
				Status: http.StatusAccepted,
				Body: AuthLoginEnvelope{
					Data: AuthLoginData{
						MFARequired: true,
						MFAToken:    "mfa_01JZABC0123456789ABCDEFGJK",
					},
					Meta: AuthLoginMeta{RequestID: "req_auth_login_mfa"},
				},
			}, nil
		}

		if input.Body.Username != authUsernameAdmin || input.Body.Password != authPasswordAdmin {
			return nil, huma.Error401Unauthorized("AUTH_INVALID_CREDENTIALS")
		}

		return &AuthLoginOutput{
			Status: http.StatusOK,
			Body: AuthLoginEnvelope{
				Data: AuthLoginData{
					AccessToken: "eyJhbGciOiJFZERTQSJ9.stub",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Meta: AuthLoginMeta{RequestID: "req_auth_login_ok"},
			},
		}, nil
	})
}

func registerAuthLogout(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authLogout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Revoke the current session",
		Description: "Invalidates the caller's refresh token server-side. Stub implementation returns empty success envelope until the token store lands. Bearer-auth requirement will be declared once the bearerAuth scheme ships with the JWT middleware.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *struct{}) (*AuthLogoutOutput, error) {
		_ = ctx
		_ = input

		return &AuthLogoutOutput{
			Body: AuthLogoutEnvelope{
				Data: AuthLogoutData{},
				Meta: AuthLogoutMeta{RequestID: "req_auth_logout"},
			},
		}, nil
	})
}

func registerAuthRefresh(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authRefresh",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/refresh",
		Summary:     "Exchange a refresh token for a new access token",
		Description: "Issues a new short-lived access token when the supplied refresh token is valid and within the inactivity window.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "Refresh token exchange payload.",
			Required:    true,
		},
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *AuthRefreshInput) (*AuthRefreshOutput, error) {
		_ = ctx

		if input.Body.RefreshToken == authRefreshTokenInval {
			return nil, huma.Error401Unauthorized("AUTH_REFRESH_INVALID")
		}

		if input.Body.RefreshToken != authRefreshTokenStub {
			return nil, huma.Error401Unauthorized("AUTH_REFRESH_INVALID")
		}

		return &AuthRefreshOutput{
			Body: AuthRefreshEnvelope{
				Data: AuthRefreshData{
					AccessToken: "eyJhbGciOiJFZERTQSJ9.refresh.stub",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Meta: AuthRefreshMeta{RequestID: "req_auth_refresh"},
			},
		}, nil
	})
}

//nolint:dupl // deliberate parallel to registerAuthTokenList; cursor pagination shape is the repo idiom.
func registerUserList(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userList",
		Method:      http.MethodGet,
		Path:        "/api/v1/users",
		Summary:     "List users",
		Description: "Lists users using cursor pagination metadata in the response envelope.",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *UserListInput) (*UserListOutput, error) {
		_ = ctx

		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}

		start := 0
		if input.Cursor == cursorPage2 {
			start = 1
		}
		if start > len(stubUsers) {
			start = len(stubUsers)
		}

		end := start + limit
		if end > len(stubUsers) {
			end = len(stubUsers)
		}

		hasNext := end < len(stubUsers)
		nextCursor := ""
		if hasNext {
			nextCursor = cursorPage2
		}

		users := append([]UserRecord(nil), stubUsers[start:end]...)
		return &UserListOutput{
			Body: UserListEnvelope{
				Data: users,
				Meta: UserListMeta{
					RequestID: "req_user_list",
					Page: UserPageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerAuthSetup(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authSetup",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/setup",
		Summary:     "Create the initial admin account",
		Description: "Bootstraps the first administrator on a fresh install. Idempotently refuses if any admin already exists; stub accepts any payload.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "Initial admin credentials.",
			Required:    true,
		},
		Errors: []int{http.StatusConflict},
	}, func(ctx context.Context, input *AuthSetupInput) (*AuthSetupOutput, error) {
		_ = ctx
		_ = input
		return &AuthSetupOutput{
			Body: AuthSetupEnvelope{
				Data: AuthSetupData{
					AccessToken: "eyJhbGciOiJFZERTQSJ9.setup.stub",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Meta: AuthSetupMeta{RequestID: "req_auth_setup"},
			},
		}, nil
	})
}

func registerAuthMfaComplete(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authMfaComplete",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/mfa/complete",
		Summary:     "Complete an MFA challenge with a TOTP code",
		Description: "Exchanges an mfa_token plus a valid TOTP code for a full JWT access token.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "MFA completion payload.",
			Required:    true,
		},
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *AuthMfaCompleteInput) (*AuthMfaCompleteOutput, error) {
		_ = ctx
		if input.Body.MfaToken != authMfaTokenStub {
			return nil, huma.Error401Unauthorized("AUTH_MFA_INVALID")
		}
		if input.Body.TotpCode != authTotpCodeValid {
			return nil, huma.Error401Unauthorized("AUTH_MFA_CODE_INVALID")
		}
		return &AuthMfaCompleteOutput{
			Body: AuthMfaCompleteEnvelope{
				Data: AuthMfaCompleteData{
					AccessToken: "eyJhbGciOiJFZERTQSJ9.mfa.stub",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Meta: AuthMfaCompleteMeta{RequestID: "req_auth_mfa_complete"},
			},
		}, nil
	})
}

func registerAuthTotpSetup(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpSetup",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/setup",
		Summary:     "Begin TOTP enrollment for the current user",
		Description: "Issues a new TOTP secret, provisioning URI, and a set of single-use recovery codes. The factor must be confirmed with /auth/totp/verify before it is active.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *struct{}) (*AuthTotpSetupOutput, error) {
		_ = ctx
		_ = input
		return &AuthTotpSetupOutput{
			Body: AuthTotpSetupEnvelope{
				Data: AuthTotpSetupData{
					ProvisioningURI: "otpauth://totp/Helling:admin?secret=JBSWY3DPEHPK3PXP&issuer=Helling",
					Secret:          "JBSWY3DPEHPK3PXP",
					RecoveryCodes: []string{
						"11112222", "33334444", "55556666", "77778888", "99990000",
						"aaaabbbb", "ccccdddd", "eeeeffff", "gggghhhh", "iiiijjjj",
					},
				},
				Meta: AuthTotpSetupMeta{RequestID: "req_auth_totp_setup"},
			},
		}, nil
	})
}

func registerAuthTotpVerify(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpVerify",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/verify",
		Summary:     "Confirm a pending TOTP enrollment",
		Description: "Activates the pending TOTP factor when the supplied code is valid.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "TOTP verification payload.",
			Required:    true,
		},
		Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *AuthTotpVerifyInput) (*AuthTotpVerifyOutput, error) {
		_ = ctx
		if input.Body.TotpCode != authTotpCodeValid {
			return nil, huma.Error401Unauthorized("AUTH_TOTP_CODE_INVALID")
		}
		return &AuthTotpVerifyOutput{
			Body: AuthTotpVerifyEnvelope{
				Data: AuthTotpVerifyData{},
				Meta: AuthTotpVerifyMeta{RequestID: "req_auth_totp_verify"},
			},
		}, nil
	})
}

func registerAuthTotpDisable(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTotpDisable",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/totp/disable",
		Summary:     "Disable TOTP for the current user",
		Description: "Removes the active TOTP factor. Admin-initiated removals are out of scope in v0.1.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *struct{}) (*AuthTotpDisableOutput, error) {
		_ = ctx
		_ = input
		return &AuthTotpDisableOutput{
			Body: AuthTotpDisableEnvelope{
				Data: AuthTotpDisableData{},
				Meta: AuthTotpDisableMeta{RequestID: "req_auth_totp_disable"},
			},
		}, nil
	})
}

//nolint:dupl // deliberate parallel to registerUserList; cursor pagination shape is the repo idiom.
func registerAuthTokenList(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenList",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/tokens",
		Summary:     "List API tokens",
		Description: "Lists API tokens belonging to the current user using cursor pagination.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *AuthTokenListInput) (*AuthTokenListOutput, error) {
		_ = ctx
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		start := 0
		if input.Cursor == cursorPage2 {
			start = 1
		}
		if start > len(stubAuthTokens) {
			start = len(stubAuthTokens)
		}
		end := start + limit
		if end > len(stubAuthTokens) {
			end = len(stubAuthTokens)
		}
		hasNext := end < len(stubAuthTokens)
		nextCursor := ""
		if hasNext {
			nextCursor = cursorPage2
		}
		tokens := append([]AuthTokenRecord(nil), stubAuthTokens[start:end]...)
		return &AuthTokenListOutput{
			Body: AuthTokenListEnvelope{
				Data: tokens,
				Meta: AuthTokenListMeta{
					RequestID: "req_auth_token_list",
					Page: AuthTokenPageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerAuthTokenCreate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/tokens",
		Summary:     "Create a new API token",
		Description: "Creates and returns a new API token. Plaintext token is surfaced exactly once.",
		Tags:        []string{"Auth"},
		RequestBody: &huma.RequestBody{
			Description: "Token creation payload.",
			Required:    true,
		},
	}, func(ctx context.Context, input *AuthTokenCreateInput) (*AuthTokenCreateOutput, error) {
		_ = ctx
		return &AuthTokenCreateOutput{
			Body: AuthTokenCreateEnvelope{
				Data: AuthTokenCreateData{
					ID:    "tok_01JZTOKEN000000000000003",
					Name:  input.Body.Name,
					Scope: input.Body.Scope,
					Token: "htk_live_stubtokenvalue0123456789abcdef",
				},
				Meta: AuthTokenCreateMeta{RequestID: "req_auth_token_create"},
			},
		}, nil
	})
}

//nolint:dupl // deliberate parallel to userList and authTokenList; cursor pagination shape is the repo idiom.
func registerScheduleList(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "scheduleList",
		Method:      http.MethodGet,
		Path:        "/api/v1/schedules",
		Summary:     "List schedules",
		Description: "Lists backup and snapshot schedules using cursor pagination.",
		Tags:        []string{"Schedules"},
	}, func(ctx context.Context, input *ScheduleListInput) (*ScheduleListOutput, error) {
		_ = ctx
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		start := 0
		if input.Cursor == cursorPage2 {
			start = 1
		}
		if start > len(stubSchedules) {
			start = len(stubSchedules)
		}
		end := start + limit
		if end > len(stubSchedules) {
			end = len(stubSchedules)
		}
		hasNext := end < len(stubSchedules)
		nextCursor := ""
		if hasNext {
			nextCursor = cursorPage2
		}
		items := append([]ScheduleRecord(nil), stubSchedules[start:end]...)
		return &ScheduleListOutput{
			Body: ScheduleListEnvelope{
				Data: items,
				Meta: ScheduleListMeta{
					RequestID: "req_schedule_list",
					Page: SchedulePageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerScheduleCreate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "scheduleCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/schedules",
		Summary:     "Create a schedule",
		Description: "Creates a new backup or snapshot schedule wired to a systemd timer.",
		Tags:        []string{"Schedules"},
		RequestBody: &huma.RequestBody{
			Description: "Schedule creation payload.",
			Required:    true,
		},
	}, func(ctx context.Context, input *ScheduleCreateInput) (*ScheduleCreateOutput, error) {
		_ = ctx
		return &ScheduleCreateOutput{
			Body: ScheduleCreateEnvelope{
				Data: ScheduleRecord{
					ID:       "sched_01JZSCHEDULE00000000003",
					Name:     input.Body.Name,
					Type:     input.Body.Type,
					Target:   input.Body.Target,
					CronExpr: input.Body.CronExpr,
					Enabled:  true,
				},
				Meta: ScheduleCreateMeta{RequestID: "req_schedule_create"},
			},
		}, nil
	})
}

func registerScheduleGet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "scheduleGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/schedules/{id}",
		Summary:     "Get a schedule",
		Description: "Returns a schedule by identifier.",
		Tags:        []string{"Schedules"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *ScheduleGetInput) (*ScheduleGetOutput, error) {
		_ = ctx
		if input.ID == scheduleIDUnknown {
			return nil, huma.Error404NotFound("SCHEDULE_NOT_FOUND")
		}
		return &ScheduleGetOutput{
			Body: ScheduleGetEnvelope{
				Data: ScheduleRecord{
					ID: input.ID, Name: "nightly-backup", Type: "backup", Target: "vm-web",
					CronExpr: "*-*-* 02:00:00", Enabled: true, NextRun: "2026-04-24T02:00:00Z",
				},
				Meta: ScheduleGetMeta{RequestID: "req_schedule_get"},
			},
		}, nil
	})
}

func registerScheduleUpdate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "scheduleUpdate",
		Method:      http.MethodPatch,
		Path:        "/api/v1/schedules/{id}",
		Summary:     "Update a schedule",
		Description: "Applies a partial update. Name, cron expression, and enabled state are the only mutable fields in v0.1.",
		Tags:        []string{"Schedules"},
		RequestBody: &huma.RequestBody{
			Description: "Partial update payload.",
			Required:    true,
		},
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *ScheduleUpdateInput) (*ScheduleUpdateOutput, error) {
		_ = ctx
		if input.ID == scheduleIDUnknown {
			return nil, huma.Error404NotFound("SCHEDULE_NOT_FOUND")
		}
		name := input.Body.Name
		if name == "" {
			name = "nightly-backup"
		}
		cron := input.Body.CronExpr
		if cron == "" {
			cron = "*-*-* 02:00:00"
		}
		enabled := true
		if input.Body.Enabled != nil {
			enabled = *input.Body.Enabled
		}
		return &ScheduleUpdateOutput{
			Body: ScheduleUpdateEnvelope{
				Data: ScheduleRecord{
					ID: input.ID, Name: name, Type: "backup", Target: "vm-web",
					CronExpr: cron, Enabled: enabled, NextRun: "2026-04-24T02:00:00Z",
				},
				Meta: ScheduleUpdateMeta{RequestID: "req_schedule_update"},
			},
		}, nil
	})
}

func registerScheduleDelete(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "scheduleDelete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/schedules/{id}",
		Summary:     "Delete a schedule",
		Description: "Removes the schedule and its systemd timer.",
		Tags:        []string{"Schedules"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *ScheduleDeleteInput) (*ScheduleDeleteOutput, error) {
		_ = ctx
		if input.ID == scheduleIDUnknown {
			return nil, huma.Error404NotFound("SCHEDULE_NOT_FOUND")
		}
		return &ScheduleDeleteOutput{
			Body: ScheduleDeleteEnvelope{
				Data: ScheduleDeleteData{},
				Meta: ScheduleDeleteMeta{RequestID: "req_schedule_delete"},
			},
		}, nil
	})
}

func registerScheduleRunNow(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "scheduleRunNow",
		Method:      http.MethodPost,
		Path:        "/api/v1/schedules/{id}/run",
		Summary:     "Trigger a schedule immediately",
		Description: "Fires the schedule's underlying systemd timer now regardless of its next scheduled run.",
		Tags:        []string{"Schedules"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *ScheduleRunNowInput) (*ScheduleRunNowOutput, error) {
		_ = ctx
		if input.ID == scheduleIDUnknown {
			return nil, huma.Error404NotFound("SCHEDULE_NOT_FOUND")
		}
		return &ScheduleRunNowOutput{
			Body: ScheduleRunNowEnvelope{
				Data: ScheduleRunNowData{
					ID:      input.ID,
					Status:  "triggered",
					StartAt: "2026-04-23T19:05:00Z",
				},
				Meta: ScheduleRunNowMeta{RequestID: "req_schedule_run_now"},
			},
		}, nil
	})
}

//nolint:dupl // deliberate parallel to scheduleList; cursor pagination shape is the repo idiom.
func registerWebhookList(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookList",
		Method:      http.MethodGet,
		Path:        "/api/v1/webhooks",
		Summary:     "List webhooks",
		Description: "Lists webhook subscriptions with cursor pagination.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, input *WebhookListInput) (*WebhookListOutput, error) {
		_ = ctx
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		start := 0
		if input.Cursor == cursorPage2 {
			start = 1
		}
		if start > len(stubWebhooks) {
			start = len(stubWebhooks)
		}
		end := start + limit
		if end > len(stubWebhooks) {
			end = len(stubWebhooks)
		}
		hasNext := end < len(stubWebhooks)
		nextCursor := ""
		if hasNext {
			nextCursor = cursorPage2
		}
		items := append([]WebhookRecord(nil), stubWebhooks[start:end]...)
		return &WebhookListOutput{
			Body: WebhookListEnvelope{
				Data: items,
				Meta: WebhookListMeta{
					RequestID: "req_webhook_list",
					Page: WebhookPageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerWebhookCreate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/webhooks",
		Summary:     "Create a webhook",
		Description: "Subscribes a destination URL to one or more event types with HMAC signing.",
		Tags:        []string{"Webhooks"},
		RequestBody: &huma.RequestBody{
			Description: "Webhook creation payload.",
			Required:    true,
		},
	}, func(ctx context.Context, input *WebhookCreateInput) (*WebhookCreateOutput, error) {
		_ = ctx
		return &WebhookCreateOutput{
			Body: WebhookCreateEnvelope{
				Data: WebhookRecord{
					ID:      "whk_01JZWEBHOOK0000000000003",
					Name:    input.Body.Name,
					URL:     input.Body.URL,
					Events:  input.Body.Events,
					Enabled: true,
				},
				Meta: WebhookCreateMeta{RequestID: "req_webhook_create"},
			},
		}, nil
	})
}

func registerWebhookGet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/webhooks/{id}",
		Summary:     "Get a webhook",
		Description: "Returns a webhook by identifier.",
		Tags:        []string{"Webhooks"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *WebhookGetInput) (*WebhookGetOutput, error) {
		_ = ctx
		if input.ID == webhookIDUnknown {
			return nil, huma.Error404NotFound("WEBHOOK_NOT_FOUND")
		}
		return &WebhookGetOutput{
			Body: WebhookGetEnvelope{
				Data: WebhookRecord{
					ID: input.ID, Name: "slack-incidents", URL: "https://hooks.slack.com/stub",
					Events: []string{"instance.created"}, Enabled: true, LastSent: "2026-04-22T10:00:00Z",
				},
				Meta: WebhookGetMeta{RequestID: "req_webhook_get"},
			},
		}, nil
	})
}

func registerWebhookUpdate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookUpdate",
		Method:      http.MethodPatch,
		Path:        "/api/v1/webhooks/{id}",
		Summary:     "Update a webhook",
		Description: "Applies a partial update. Name, URL, event list, and enabled state are mutable.",
		Tags:        []string{"Webhooks"},
		RequestBody: &huma.RequestBody{
			Description: "Partial update payload.",
			Required:    true,
		},
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *WebhookUpdateInput) (*WebhookUpdateOutput, error) {
		_ = ctx
		if input.ID == webhookIDUnknown {
			return nil, huma.Error404NotFound("WEBHOOK_NOT_FOUND")
		}
		name := input.Body.Name
		if name == "" {
			name = "slack-incidents"
		}
		url := input.Body.URL
		if url == "" {
			url = "https://hooks.slack.com/stub"
		}
		events := input.Body.Events
		if len(events) == 0 {
			events = []string{"instance.created"}
		}
		enabled := true
		if input.Body.Enabled != nil {
			enabled = *input.Body.Enabled
		}
		return &WebhookUpdateOutput{
			Body: WebhookUpdateEnvelope{
				Data: WebhookRecord{
					ID: input.ID, Name: name, URL: url, Events: events, Enabled: enabled,
				},
				Meta: WebhookUpdateMeta{RequestID: "req_webhook_update"},
			},
		}, nil
	})
}

func registerWebhookDelete(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookDelete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/webhooks/{id}",
		Summary:     "Delete a webhook",
		Description: "Removes the webhook subscription.",
		Tags:        []string{"Webhooks"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *WebhookDeleteInput) (*WebhookDeleteOutput, error) {
		_ = ctx
		if input.ID == webhookIDUnknown {
			return nil, huma.Error404NotFound("WEBHOOK_NOT_FOUND")
		}
		return &WebhookDeleteOutput{
			Body: WebhookDeleteEnvelope{
				Data: WebhookDeleteData{},
				Meta: WebhookDeleteMeta{RequestID: "req_webhook_delete"},
			},
		}, nil
	})
}

func registerWebhookTest(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "webhookTest",
		Method:      http.MethodPost,
		Path:        "/api/v1/webhooks/{id}/test",
		Summary:     "Send a synthetic delivery to a webhook",
		Description: "Fires a test event to the webhook destination and returns the observed status + latency.",
		Tags:        []string{"Webhooks"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *WebhookTestInput) (*WebhookTestOutput, error) {
		_ = ctx
		if input.ID == webhookIDUnknown {
			return nil, huma.Error404NotFound("WEBHOOK_NOT_FOUND")
		}
		return &WebhookTestOutput{
			Body: WebhookTestEnvelope{
				Data: WebhookTestData{
					ID:         input.ID,
					Status:     "delivered",
					StatusCode: 200,
					Latency:    42,
				},
				Meta: WebhookTestMeta{RequestID: "req_webhook_test"},
			},
		}, nil
	})
}

//nolint:dupl // deliberate parallel to other list registrations.
func registerKubernetesList(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "kubernetesList",
		Method:      http.MethodGet,
		Path:        "/api/v1/kubernetes",
		Summary:     "List k3s clusters",
		Description: "Lists managed k3s clusters with cursor pagination.",
		Tags:        []string{"Kubernetes"},
	}, func(ctx context.Context, input *KubernetesListInput) (*KubernetesListOutput, error) {
		_ = ctx
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		start := 0
		if input.Cursor == cursorPage2 {
			start = 1
		}
		if start > len(stubKubernetesClusters) {
			start = len(stubKubernetesClusters)
		}
		end := start + limit
		if end > len(stubKubernetesClusters) {
			end = len(stubKubernetesClusters)
		}
		hasNext := end < len(stubKubernetesClusters)
		nextCursor := ""
		if hasNext {
			nextCursor = cursorPage2
		}
		items := append([]KubernetesRecord(nil), stubKubernetesClusters[start:end]...)
		return &KubernetesListOutput{
			Body: KubernetesListEnvelope{
				Data: items,
				Meta: KubernetesListMeta{
					RequestID: "req_kubernetes_list",
					Page: KubernetesPageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerKubernetesCreate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "kubernetesCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/kubernetes",
		Summary:     "Create a k3s cluster",
		Description: "Provisions a new k3s cluster via cloud-init. Returns the newly-created cluster in `provisioning` status.",
		Tags:        []string{"Kubernetes"},
		RequestBody: &huma.RequestBody{
			Description: "Cluster creation payload.",
			Required:    true,
		},
		Errors: []int{http.StatusConflict},
	}, func(ctx context.Context, input *KubernetesCreateInput) (*KubernetesCreateOutput, error) {
		_ = ctx
		if input.Body.Name == kubernetesNameExisting {
			return nil, huma.Error409Conflict("KUBERNETES_ALREADY_EXISTS")
		}
		return &KubernetesCreateOutput{
			Body: KubernetesCreateEnvelope{
				Data: KubernetesRecord{
					Name:    input.Body.Name,
					Version: input.Body.Version,
					Status:  "provisioning",
					Workers: input.Body.Workers,
				},
				Meta: KubernetesCreateMeta{RequestID: "req_kubernetes_create"},
			},
		}, nil
	})
}

func registerKubernetesGet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "kubernetesGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/kubernetes/{name}",
		Summary:     "Get a k3s cluster",
		Description: "Returns a single cluster by name.",
		Tags:        []string{"Kubernetes"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *KubernetesGetInput) (*KubernetesGetOutput, error) {
		_ = ctx
		if input.Name == kubernetesNameUnknown {
			return nil, huma.Error404NotFound("KUBERNETES_NOT_FOUND")
		}
		return &KubernetesGetOutput{
			Body: KubernetesGetEnvelope{
				Data: KubernetesRecord{
					Name: input.Name, Version: "v1.30.5+k3s1", Status: "ready",
					Workers: 3, ReadyWorkers: 3, CreatedAt: "2026-03-01T00:00:00Z",
				},
				Meta: KubernetesGetMeta{RequestID: "req_kubernetes_get"},
			},
		}, nil
	})
}

func registerKubernetesDelete(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "kubernetesDelete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/kubernetes/{name}",
		Summary:     "Delete a k3s cluster",
		Description: "Tears down the cluster and its underlying Incus resources.",
		Tags:        []string{"Kubernetes"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *KubernetesDeleteInput) (*KubernetesDeleteOutput, error) {
		_ = ctx
		if input.Name == kubernetesNameUnknown {
			return nil, huma.Error404NotFound("KUBERNETES_NOT_FOUND")
		}
		return &KubernetesDeleteOutput{
			Body: KubernetesDeleteEnvelope{
				Data: KubernetesDeleteData{},
				Meta: KubernetesDeleteMeta{RequestID: "req_kubernetes_delete"},
			},
		}, nil
	})
}

func registerKubernetesScale(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "kubernetesScale",
		Method:      http.MethodPatch,
		Path:        "/api/v1/kubernetes/{name}/scale",
		Summary:     "Scale a cluster's worker pool",
		Description: "Adjusts the declared worker count for a cluster.",
		Tags:        []string{"Kubernetes"},
		RequestBody: &huma.RequestBody{
			Description: "Scale target payload.",
			Required:    true,
		},
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *KubernetesScaleInput) (*KubernetesScaleOutput, error) {
		_ = ctx
		if input.Name == kubernetesNameUnknown {
			return nil, huma.Error404NotFound("KUBERNETES_NOT_FOUND")
		}
		return &KubernetesScaleOutput{
			Body: KubernetesScaleEnvelope{
				Data: KubernetesRecord{
					Name: input.Name, Version: "v1.30.5+k3s1", Status: "ready",
					Workers: input.Body.Workers, ReadyWorkers: input.Body.Workers,
					CreatedAt: "2026-03-01T00:00:00Z",
				},
				Meta: KubernetesScaleMeta{RequestID: "req_kubernetes_scale"},
			},
		}, nil
	})
}

func registerKubernetesUpgrade(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "kubernetesUpgrade",
		Method:      http.MethodPost,
		Path:        "/api/v1/kubernetes/{name}/upgrade",
		Summary:     "Roll-upgrade a cluster",
		Description: "Starts a rolling upgrade to a new k3s version.",
		Tags:        []string{"Kubernetes"},
		RequestBody: &huma.RequestBody{
			Description: "Upgrade target payload.",
			Required:    true,
		},
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *KubernetesUpgradeInput) (*KubernetesUpgradeOutput, error) {
		_ = ctx
		if input.Name == kubernetesNameUnknown {
			return nil, huma.Error404NotFound("KUBERNETES_NOT_FOUND")
		}
		return &KubernetesUpgradeOutput{
			Body: KubernetesUpgradeEnvelope{
				Data: KubernetesRecord{
					Name: input.Name, Version: input.Body.Version, Status: "upgrading",
					Workers: 3, ReadyWorkers: 3, CreatedAt: "2026-03-01T00:00:00Z",
				},
				Meta: KubernetesUpgradeMeta{RequestID: "req_kubernetes_upgrade"},
			},
		}, nil
	})
}

func registerKubernetesKubeconfig(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "kubernetesKubeconfig",
		Method:      http.MethodGet,
		Path:        "/api/v1/kubernetes/{name}/kubeconfig",
		Summary:     "Download cluster kubeconfig",
		Description: "Returns the kubeconfig YAML for the named cluster, inlined as a JSON string.",
		Tags:        []string{"Kubernetes"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *KubernetesKubeconfigInput) (*KubernetesKubeconfigOutput, error) {
		_ = ctx
		if input.Name == kubernetesNameUnknown {
			return nil, huma.Error404NotFound("KUBERNETES_NOT_FOUND")
		}
		return &KubernetesKubeconfigOutput{
			Body: KubernetesKubeconfigEnvelope{
				Data: KubernetesKubeconfigData{Kubeconfig: kubernetesStubKubeconfig},
				Meta: KubernetesKubeconfigMeta{RequestID: "req_kubernetes_kubeconfig"},
			},
		}, nil
	})
}

func registerUserCreate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/users",
		Summary:     "Create a user",
		Description: "Creates a new PAM-backed user and returns the summary. Stub accepts any payload and echoes the username back.",
		Tags:        []string{"Users"},
		RequestBody: &huma.RequestBody{
			Description: "User creation payload.",
			Required:    true,
		},
		Errors: []int{http.StatusConflict},
	}, func(ctx context.Context, input *UserCreateInput) (*UserCreateOutput, error) {
		_ = ctx
		if input.Body.Username == "admin" {
			return nil, huma.Error409Conflict("USER_ALREADY_EXISTS")
		}
		return &UserCreateOutput{
			Body: UserCreateEnvelope{
				Data: UserCreateData{
					ID:       "user_" + input.Body.Username,
					Username: input.Body.Username,
					Role:     input.Body.Role,
					Status:   "active",
				},
				Meta: UserCreateMeta{RequestID: "req_user_create"},
			},
		}, nil
	})
}

func registerUserGet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/users/{id}",
		Summary:     "Get a user",
		Description: "Returns detailed user fields including TOTP enrollment state and last-login timestamp.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *UserGetInput) (*UserGetOutput, error) {
		_ = ctx
		if input.ID == userIDUnknown {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		return &UserGetOutput{
			Body: UserGetEnvelope{
				Data: UserGetData{
					ID:          input.ID,
					Username:    "admin",
					Role:        "admin",
					Status:      "active",
					TotpEnabled: true,
					LastLogin:   "2026-04-23T19:00:00Z",
				},
				Meta: UserGetMeta{RequestID: "req_user_get"},
			},
		}, nil
	})
}

func registerUserUpdate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userUpdate",
		Method:      http.MethodPatch,
		Path:        "/api/v1/users/{id}",
		Summary:     "Update a user",
		Description: "Applies a partial update. Only role and status are mutable in v0.1.",
		Tags:        []string{"Users"},
		RequestBody: &huma.RequestBody{
			Description: "Partial update payload.",
			Required:    true,
		},
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *UserUpdateInput) (*UserUpdateOutput, error) {
		_ = ctx
		if input.ID == userIDUnknown {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		role := input.Body.Role
		if role == "" {
			role = "user"
		}
		status := input.Body.Status
		if status == "" {
			status = "active"
		}
		return &UserUpdateOutput{
			Body: UserUpdateEnvelope{
				Data: UserUpdateData{
					ID:       input.ID,
					Username: "admin",
					Role:     role,
					Status:   status,
				},
				Meta: UserUpdateMeta{RequestID: "req_user_update"},
			},
		}, nil
	})
}

func registerUserDelete(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userDelete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/users/{id}",
		Summary:     "Delete a user",
		Description: "Removes a PAM-backed user. Deleting an unknown user returns 404.",
		Tags:        []string{"Users"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *UserDeleteInput) (*UserDeleteOutput, error) {
		_ = ctx
		if input.ID == userIDUnknown {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		return &UserDeleteOutput{
			Body: UserDeleteEnvelope{
				Data: UserDeleteData{},
				Meta: UserDeleteMeta{RequestID: "req_user_delete"},
			},
		}, nil
	})
}

func registerUserSetScope(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "userSetScope",
		Method:      http.MethodPut,
		Path:        "/api/v1/users/{id}/scope",
		Summary:     "Assign a trust scope to a user",
		Description: "Applies an Incus trust-scope change. The scope controls which Incus projects the user's cert can touch.",
		Tags:        []string{"Users"},
		RequestBody: &huma.RequestBody{
			Description: "Trust-scope assignment payload.",
			Required:    true,
		},
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, input *UserSetScopeInput) (*UserSetScopeOutput, error) {
		_ = ctx
		if input.ID == userIDUnknown {
			return nil, huma.Error404NotFound("USER_NOT_FOUND")
		}
		return &UserSetScopeOutput{
			Body: UserSetScopeEnvelope{
				Data: UserSetScopeData{
					ID:    input.ID,
					Scope: input.Body.Scope,
				},
				Meta: UserSetScopeMeta{RequestID: "req_user_set_scope"},
			},
		}, nil
	})
}

func registerAuthTokenRevoke(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "authTokenRevoke",
		Method:      http.MethodDelete,
		Path:        "/api/v1/auth/tokens/{id}",
		Summary:     "Revoke an API token",
		Description: "Invalidates the given token. Revoking an already-unknown token returns 404.",
		Tags:        []string{"Auth"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *AuthTokenRevokeInput) (*AuthTokenRevokeOutput, error) {
		_ = ctx
		if input.ID == authTokenIDUnknown {
			return nil, huma.Error404NotFound("AUTH_TOKEN_NOT_FOUND")
		}
		return &AuthTokenRevokeOutput{
			Body: AuthTokenRevokeEnvelope{
				Data: AuthTokenRevokeData{},
				Meta: AuthTokenRevokeMeta{RequestID: "req_auth_token_revoke"},
			},
		}, nil
	})
}

// ─── System domain stubs ──────────────────────────────────────────────────
const (
	systemConfigKeyKnown   = "auth.session_inactivity_timeout"
	systemConfigKeyMissing = "does.not.exist"
)

// SystemMeta is the standard metadata envelope for the system domain.
type SystemMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// SystemInfoData mirrors `helling system info` output.
type SystemInfoData struct {
	Hostname string `json:"hostname" doc:"System hostname."`
	Version  string `json:"version" doc:"Running hellingd version (semver)."`
	Uptime   string `json:"uptime" doc:"Human-readable uptime (e.g. 14d 3h)."`
	Arch     string `json:"arch" doc:"Host architecture (amd64/arm64)."`
	Kernel   string `json:"kernel" doc:"Kernel release string."`
}

// SystemInfoEnvelope is the success envelope for system info.
type SystemInfoEnvelope struct {
	Data SystemInfoData `json:"data"`
	Meta SystemMeta     `json:"meta"`
}

// SystemInfoOutput is the response shape for GET /api/v1/system/info.
type SystemInfoOutput struct {
	Body SystemInfoEnvelope
}

// SystemHardwareData summarizes detected hardware.
type SystemHardwareData struct {
	CPU     string `json:"cpu" doc:"CPU model string."`
	Cores   int    `json:"cores" doc:"Physical core count." minimum:"1"`
	RAMGB   int    `json:"ram_gb" doc:"Total RAM in GiB." minimum:"1"`
	DiskGB  int    `json:"disk_gb" doc:"Total primary disk capacity in GiB." minimum:"1"`
	Network string `json:"network" doc:"Primary NIC description."`
}

// SystemHardwareEnvelope is the success envelope for system hardware.
type SystemHardwareEnvelope struct {
	Data SystemHardwareData `json:"data"`
	Meta SystemMeta         `json:"meta"`
}

// SystemHardwareOutput is the response shape for GET /api/v1/system/hardware.
type SystemHardwareOutput struct {
	Body SystemHardwareEnvelope
}

// SystemConfigGetInput binds the path key.
type SystemConfigGetInput struct {
	Key string `path:"key" minLength:"1" maxLength:"128" doc:"Config key (dot-separated)."`
}

// SystemConfigData wraps a single config key/value.
type SystemConfigData struct {
	Key   string `json:"key" doc:"Config key."`
	Value string `json:"value" doc:"Config value as a string."`
}

// SystemConfigEnvelope is the success envelope for config reads/writes.
type SystemConfigEnvelope struct {
	Data SystemConfigData `json:"data"`
	Meta SystemMeta       `json:"meta"`
}

// SystemConfigGetOutput is the response shape for GET /api/v1/system/config/{key}.
type SystemConfigGetOutput struct {
	Body SystemConfigEnvelope
}

// SystemConfigPutRequest carries the new value.
type SystemConfigPutRequest struct {
	Value string `json:"value" minLength:"1" maxLength:"1024" doc:"Config value to store."`
}

// SystemConfigPutInput combines path key with value body.
type SystemConfigPutInput struct {
	Key  string                 `path:"key" minLength:"1" maxLength:"128" doc:"Config key."`
	Body SystemConfigPutRequest `doc:"Config update payload."`
}

// SystemConfigPutOutput is the response shape for PUT /api/v1/system/config/{key}.
type SystemConfigPutOutput struct {
	Body SystemConfigEnvelope
}

// SystemUpgradeRequest triggers an upgrade check or rollback.
type SystemUpgradeRequest struct {
	Rollback bool `json:"rollback,omitempty" doc:"If true, revert to the previous version instead of upgrading."`
}

// SystemUpgradeInput wraps the upgrade body.
type SystemUpgradeInput struct {
	Body SystemUpgradeRequest `doc:"Upgrade action payload."`
}

// SystemUpgradeData reports the action result.
type SystemUpgradeData struct {
	FromVersion string `json:"from_version" doc:"Version prior to the action."`
	ToVersion   string `json:"to_version" doc:"Version after the action."`
	Status      string `json:"status" doc:"Upgrade status." enum:"scheduled,rolling_back,no_change"`
}

// SystemUpgradeEnvelope is the success envelope for upgrade responses.
type SystemUpgradeEnvelope struct {
	Data SystemUpgradeData `json:"data"`
	Meta SystemMeta        `json:"meta"`
}

// SystemUpgradeOutput is the response shape for POST /api/v1/system/upgrade.
type SystemUpgradeOutput struct {
	Body SystemUpgradeEnvelope
}

// SystemDiagnosticsCheck is a single diagnostic probe result.
type SystemDiagnosticsCheck struct {
	Name    string `json:"name" doc:"Check name."`
	Status  string `json:"status" doc:"Check status." enum:"pass,warn,fail"`
	Message string `json:"message,omitempty" doc:"Optional detail message."`
}

// SystemDiagnosticsData is the full self-test report.
type SystemDiagnosticsData struct {
	Checks []SystemDiagnosticsCheck `json:"checks" doc:"Ordered list of diagnostic probes."`
	Passed int                      `json:"passed" doc:"Count of passing checks." minimum:"0"`
	Failed int                      `json:"failed" doc:"Count of failing checks." minimum:"0"`
}

// SystemDiagnosticsEnvelope is the success envelope for diagnostics.
type SystemDiagnosticsEnvelope struct {
	Data SystemDiagnosticsData `json:"data"`
	Meta SystemMeta            `json:"meta"`
}

// SystemDiagnosticsOutput is the response shape for GET /api/v1/system/diagnostics.
type SystemDiagnosticsOutput struct {
	Body SystemDiagnosticsEnvelope
}

func registerSystemInfo(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "systemInfo",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/info",
		Summary:     "System info",
		Description: "Returns hostname, hellingd version, uptime, and host kernel/arch.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*SystemInfoOutput, error) {
		_ = ctx
		_ = input
		return &SystemInfoOutput{
			Body: SystemInfoEnvelope{
				Data: SystemInfoData{
					Hostname: "helling-stub", Version: "0.1.0-alpha",
					Uptime: "14d 3h", Arch: "amd64", Kernel: "6.1.0-stub",
				},
				Meta: SystemMeta{RequestID: "req_system_info"},
			},
		}, nil
	})
}

func registerSystemHardware(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "systemHardware",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/hardware",
		Summary:     "Host hardware summary",
		Description: "Returns CPU/RAM/disk/network capabilities detected on the host.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*SystemHardwareOutput, error) {
		_ = ctx
		_ = input
		return &SystemHardwareOutput{
			Body: SystemHardwareEnvelope{
				Data: SystemHardwareData{
					CPU: "Intel Xeon Silver 4214 (stub)", Cores: 12, RAMGB: 64,
					DiskGB: 1024, Network: "2x10GbE",
				},
				Meta: SystemMeta{RequestID: "req_system_hardware"},
			},
		}, nil
	})
}

func registerSystemConfigGet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "systemConfigGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/config/{key}",
		Summary:     "Read a config value",
		Description: "Returns a single config key. 404 if unknown.",
		Tags:        []string{"System"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *SystemConfigGetInput) (*SystemConfigGetOutput, error) {
		_ = ctx
		if input.Key == systemConfigKeyMissing {
			return nil, huma.Error404NotFound("SYSTEM_CONFIG_KEY_NOT_FOUND")
		}
		val := "30m"
		if input.Key != systemConfigKeyKnown {
			val = "stub"
		}
		return &SystemConfigGetOutput{
			Body: SystemConfigEnvelope{
				Data: SystemConfigData{Key: input.Key, Value: val},
				Meta: SystemMeta{RequestID: "req_system_config_get"},
			},
		}, nil
	})
}

func registerSystemConfigPut(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "systemConfigPut",
		Method:      http.MethodPut,
		Path:        "/api/v1/system/config/{key}",
		Summary:     "Write a config value",
		Description: "Upserts a single config key. Validation beyond shape is deferred to the service layer.",
		Tags:        []string{"System"},
		RequestBody: &huma.RequestBody{
			Description: "Config value payload.",
			Required:    true,
		},
	}, func(ctx context.Context, input *SystemConfigPutInput) (*SystemConfigPutOutput, error) {
		_ = ctx
		return &SystemConfigPutOutput{
			Body: SystemConfigEnvelope{
				Data: SystemConfigData{Key: input.Key, Value: input.Body.Value},
				Meta: SystemMeta{RequestID: "req_system_config_put"},
			},
		}, nil
	})
}

func registerSystemUpgrade(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "systemUpgrade",
		Method:      http.MethodPost,
		Path:        "/api/v1/system/upgrade",
		Summary:     "Check or apply a system upgrade",
		Description: "Schedules an upgrade (or rollback if `rollback: true`). Stub reports status without mutating state.",
		Tags:        []string{"System"},
		RequestBody: &huma.RequestBody{
			Description: "Upgrade action payload.",
			Required:    false,
		},
	}, func(ctx context.Context, input *SystemUpgradeInput) (*SystemUpgradeOutput, error) {
		_ = ctx
		status := "scheduled"
		from, to := "0.1.0-alpha", "0.1.0-beta"
		if input.Body.Rollback {
			status = "rolling_back"
			from, to = "0.1.0-beta", "0.1.0-alpha"
		}
		return &SystemUpgradeOutput{
			Body: SystemUpgradeEnvelope{
				Data: SystemUpgradeData{FromVersion: from, ToVersion: to, Status: status},
				Meta: SystemMeta{RequestID: "req_system_upgrade"},
			},
		}, nil
	})
}

func registerSystemDiagnostics(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "systemDiagnostics",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/diagnostics",
		Summary:     "Run self-diagnostics",
		Description: "Runs a suite of self-checks and returns per-check status. Stub always returns a fixed report.",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*SystemDiagnosticsOutput, error) {
		_ = ctx
		_ = input
		checks := []SystemDiagnosticsCheck{
			{Name: "journal-writable", Status: "pass"},
			{Name: "incus-socket", Status: "pass"},
			{Name: "podman-socket", Status: "pass"},
			{Name: "disk-space", Status: "warn", Message: "/var at 78%"},
		}
		return &SystemDiagnosticsOutput{
			Body: SystemDiagnosticsEnvelope{
				Data: SystemDiagnosticsData{Checks: checks, Passed: 3, Failed: 0},
				Meta: SystemMeta{RequestID: "req_system_diagnostics"},
			},
		}, nil
	})
}

// ─── Firewall domain stubs ────────────────────────────────────────────────
const (
	firewallRuleIDExisting = "fwr_01JZFW00000000000000001"
	firewallRuleIDUnknown  = "fwr_missing"
)

// FirewallMeta is the metadata envelope for firewall endpoints.
type FirewallMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// FirewallHostRule is a single host-level nftables rule.
type FirewallHostRule struct {
	ID       string `json:"id" doc:"Rule identifier."`
	Chain    string `json:"chain" doc:"nftables chain name."`
	Priority int    `json:"priority" doc:"Rule priority within the chain." minimum:"0"`
	Action   string `json:"action" doc:"Rule verdict." enum:"accept,drop,reject"`
	Expr     string `json:"expr" doc:"Rule expression (nft syntax)."`
	Enabled  bool   `json:"enabled" doc:"Whether the rule is currently enforced."`
}

// FirewallHostPageMeta is pagination metadata for firewall list responses.
type FirewallHostPageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// FirewallHostListInput has pagination controls.
type FirewallHostListInput struct {
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of rules to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// FirewallHostListMeta contains request and paging metadata.
type FirewallHostListMeta struct {
	RequestID string               `json:"request_id" doc:"Request correlation ID."`
	Page      FirewallHostPageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// FirewallHostListEnvelope follows the list envelope shape.
type FirewallHostListEnvelope struct {
	Data []FirewallHostRule   `json:"data"`
	Meta FirewallHostListMeta `json:"meta"`
}

// FirewallHostListOutput is the response shape for GET /api/v1/firewall/host.
type FirewallHostListOutput struct {
	Body FirewallHostListEnvelope
}

// FirewallHostCreateRequest adds a new host-level rule.
type FirewallHostCreateRequest struct {
	Chain    string `json:"chain" minLength:"1" maxLength:"64" doc:"nftables chain name."`
	Priority int    `json:"priority" minimum:"0" maximum:"999" doc:"Rule priority."`
	Action   string `json:"action" doc:"Verdict." enum:"accept,drop,reject"`
	Expr     string `json:"expr" minLength:"1" maxLength:"1024" doc:"Rule expression (nft syntax)."`
}

// FirewallHostCreateInput wraps the create body.
type FirewallHostCreateInput struct {
	Body FirewallHostCreateRequest `doc:"Rule creation payload."`
}

// FirewallHostCreateEnvelope follows the success envelope shape for rule create.
type FirewallHostCreateEnvelope struct {
	Data FirewallHostRule `json:"data"`
	Meta FirewallMeta     `json:"meta"`
}

// FirewallHostCreateOutput is the response shape for POST /api/v1/firewall/host.
type FirewallHostCreateOutput struct {
	Body FirewallHostCreateEnvelope
}

// FirewallHostDeleteInput binds the path id.
type FirewallHostDeleteInput struct {
	ID string `path:"id" minLength:"1" maxLength:"64" doc:"Rule identifier."`
}

// FirewallHostDeleteData is empty on success.
type FirewallHostDeleteData struct{}

// FirewallHostDeleteEnvelope follows the success envelope shape.
type FirewallHostDeleteEnvelope struct {
	Data FirewallHostDeleteData `json:"data"`
	Meta FirewallMeta           `json:"meta"`
}

// FirewallHostDeleteOutput is the response shape for DELETE /api/v1/firewall/host/{id}.
type FirewallHostDeleteOutput struct {
	Body FirewallHostDeleteEnvelope
}

var stubFirewallRules = []FirewallHostRule{
	{ID: firewallRuleIDExisting, Chain: "input", Priority: 100, Action: "accept", Expr: "tcp dport 22 ct state new", Enabled: true},
	{ID: "fwr_01JZFW00000000000000002", Chain: "input", Priority: 110, Action: "drop", Expr: "tcp dport 0-1023 ct state new", Enabled: true},
}

//nolint:dupl // deliberate parallel to other cursor-paginated list registrations.
func registerFirewallHostList(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "firewallHostList",
		Method:      http.MethodGet,
		Path:        "/api/v1/firewall/host",
		Summary:     "List host firewall rules",
		Description: "Lists host-level nftables rules using cursor pagination.",
		Tags:        []string{"Firewall"},
	}, func(ctx context.Context, input *FirewallHostListInput) (*FirewallHostListOutput, error) {
		_ = ctx
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		start := 0
		if input.Cursor == cursorPage2 {
			start = 1
		}
		if start > len(stubFirewallRules) {
			start = len(stubFirewallRules)
		}
		end := start + limit
		if end > len(stubFirewallRules) {
			end = len(stubFirewallRules)
		}
		hasNext := end < len(stubFirewallRules)
		nextCursor := ""
		if hasNext {
			nextCursor = cursorPage2
		}
		items := append([]FirewallHostRule(nil), stubFirewallRules[start:end]...)
		return &FirewallHostListOutput{
			Body: FirewallHostListEnvelope{
				Data: items,
				Meta: FirewallHostListMeta{
					RequestID: "req_firewall_host_list",
					Page: FirewallHostPageMeta{
						HasNext:    hasNext,
						NextCursor: nextCursor,
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerFirewallHostCreate(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "firewallHostCreate",
		Method:      http.MethodPost,
		Path:        "/api/v1/firewall/host",
		Summary:     "Add a host firewall rule",
		Description: "Appends a new host-level nftables rule. Returns the created rule including generated id.",
		Tags:        []string{"Firewall"},
		RequestBody: &huma.RequestBody{
			Description: "Rule creation payload.",
			Required:    true,
		},
	}, func(ctx context.Context, input *FirewallHostCreateInput) (*FirewallHostCreateOutput, error) {
		_ = ctx
		return &FirewallHostCreateOutput{
			Body: FirewallHostCreateEnvelope{
				Data: FirewallHostRule{
					ID: "fwr_01JZFW00000000000000003", Chain: input.Body.Chain,
					Priority: input.Body.Priority, Action: input.Body.Action,
					Expr: input.Body.Expr, Enabled: true,
				},
				Meta: FirewallMeta{RequestID: "req_firewall_host_create"},
			},
		}, nil
	})
}

func registerFirewallHostDelete(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "firewallHostDelete",
		Method:      http.MethodDelete,
		Path:        "/api/v1/firewall/host/{id}",
		Summary:     "Delete a host firewall rule",
		Description: "Removes a host-level nftables rule.",
		Tags:        []string{"Firewall"},
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, input *FirewallHostDeleteInput) (*FirewallHostDeleteOutput, error) {
		_ = ctx
		if input.ID == firewallRuleIDUnknown {
			return nil, huma.Error404NotFound("FIREWALL_RULE_NOT_FOUND")
		}
		return &FirewallHostDeleteOutput{
			Body: FirewallHostDeleteEnvelope{
				Data: FirewallHostDeleteData{},
				Meta: FirewallMeta{RequestID: "req_firewall_host_delete"},
			},
		}, nil
	})
}

// ─── Audit domain stubs ───────────────────────────────────────────────────

// AuditMeta is the metadata envelope for audit endpoints.
type AuditMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// AuditEvent is one audit-log row surfaced from systemd journal.
type AuditEvent struct {
	ID        string `json:"id" doc:"Event identifier."`
	Timestamp string `json:"timestamp" doc:"Event timestamp (ISO-8601)." format:"date-time"`
	Actor     string `json:"actor" doc:"Actor who performed the action."`
	Action    string `json:"action" doc:"Action identifier (dot-separated)."`
	Target    string `json:"target,omitempty" doc:"Target resource identifier, if applicable."`
	Outcome   string `json:"outcome" doc:"Action outcome." enum:"allow,deny,error"`
}

// AuditPageMeta is pagination metadata for audit queries.
type AuditPageMeta struct {
	HasNext    bool   `json:"has_next" doc:"Whether another page is available."`
	NextCursor string `json:"next_cursor,omitempty" doc:"Opaque cursor for the next page when available."`
	Limit      int    `json:"limit" doc:"Applied page size." minimum:"1"`
}

// AuditQueryInput carries filter + pagination controls.
type AuditQueryInput struct {
	Actor  string `query:"actor" maxLength:"128" doc:"Filter by actor username."`
	Action string `query:"action" maxLength:"128" doc:"Filter by action identifier."`
	Limit  int    `query:"limit" default:"100" minimum:"1" maximum:"1000" doc:"Maximum number of events to return."`
	Cursor string `query:"cursor" maxLength:"512" doc:"Opaque pagination cursor from previous response."`
}

// AuditQueryMeta contains request and paging metadata.
type AuditQueryMeta struct {
	RequestID string        `json:"request_id" doc:"Request correlation ID."`
	Page      AuditPageMeta `json:"page" doc:"Cursor pagination metadata."`
}

// AuditQueryEnvelope follows the list envelope shape.
type AuditQueryEnvelope struct {
	Data []AuditEvent   `json:"data"`
	Meta AuditQueryMeta `json:"meta"`
}

// AuditQueryOutput is the response shape for GET /api/v1/audit.
type AuditQueryOutput struct {
	Body AuditQueryEnvelope
}

// AuditExportInput carries the desired export format.
type AuditExportInput struct {
	Format string `query:"format" default:"json" doc:"Export format." enum:"json,csv"`
}

// AuditExportData wraps the inlined export payload.
type AuditExportData struct {
	Format string `json:"format" doc:"Export format." enum:"json,csv"`
	Body   string `json:"body" doc:"Export payload inlined as a string."`
}

// AuditExportEnvelope is the success envelope for exports.
type AuditExportEnvelope struct {
	Data AuditExportData `json:"data"`
	Meta AuditMeta       `json:"meta"`
}

// AuditExportOutput is the response shape for GET /api/v1/audit/export.
type AuditExportOutput struct {
	Body AuditExportEnvelope
}

var stubAuditEvents = []AuditEvent{
	{ID: "aud_01JZAUD00000000000000001", Timestamp: "2026-04-23T10:00:00Z", Actor: "admin", Action: "auth.login", Outcome: "allow"},
	{ID: "aud_01JZAUD00000000000000002", Timestamp: "2026-04-23T10:05:00Z", Actor: "admin", Action: "user.create", Target: "user_bob", Outcome: "allow"},
	{ID: "aud_01JZAUD00000000000000003", Timestamp: "2026-04-23T11:00:00Z", Actor: "alice", Action: "compute.delete", Target: "vm-test", Outcome: "deny"},
}

func registerAuditQuery(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "auditQuery",
		Method:      http.MethodGet,
		Path:        "/api/v1/audit",
		Summary:     "Query audit events",
		Description: "Returns audit events, optionally filtered by actor and action, with cursor pagination. Source of truth is the systemd journal (ADR-019).",
		Tags:        []string{"Audit"},
	}, func(ctx context.Context, input *AuditQueryInput) (*AuditQueryOutput, error) {
		_ = ctx
		filtered := make([]AuditEvent, 0, len(stubAuditEvents))
		for _, e := range stubAuditEvents {
			if input.Actor != "" && e.Actor != input.Actor {
				continue
			}
			if input.Action != "" && e.Action != input.Action {
				continue
			}
			filtered = append(filtered, e)
		}
		limit := input.Limit
		if limit <= 0 {
			limit = 100
		}
		if limit > len(filtered) {
			limit = len(filtered)
		}
		return &AuditQueryOutput{
			Body: AuditQueryEnvelope{
				Data: filtered[:limit],
				Meta: AuditQueryMeta{
					RequestID: "req_audit_query",
					Page: AuditPageMeta{
						HasNext:    false,
						NextCursor: "",
						Limit:      limit,
					},
				},
			},
		}, nil
	})
}

func registerAuditExport(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "auditExport",
		Method:      http.MethodGet,
		Path:        "/api/v1/audit/export",
		Summary:     "Bulk export audit events",
		Description: "Returns the filtered audit event stream inlined as either JSON or CSV.",
		Tags:        []string{"Audit"},
	}, func(ctx context.Context, input *AuditExportInput) (*AuditExportOutput, error) {
		_ = ctx
		body := `[{"id":"aud_01JZAUD00000000000000001","timestamp":"2026-04-23T10:00:00Z","actor":"admin","action":"auth.login","outcome":"allow"}]`
		if input.Format == "csv" {
			body = "id,timestamp,actor,action,target,outcome\naud_01JZAUD00000000000000001,2026-04-23T10:00:00Z,admin,auth.login,,allow\n"
		}
		return &AuditExportOutput{
			Body: AuditExportEnvelope{
				Data: AuditExportData{Format: input.Format, Body: body},
				Meta: AuditMeta{RequestID: "req_audit_export"},
			},
		}, nil
	})
}

// ─── Events domain stub (SSE stand-in) ────────────────────────────────────
// v0.1-alpha exposes the eventsSse operation as a polling snapshot. Real SSE
// streaming lands with the events bus and WebSocket/SSE infra in v0.1-beta
// per docs/spec/events.md.

// EventsMeta is the metadata envelope for the events domain.
type EventsMeta struct {
	RequestID string `json:"request_id" doc:"Request correlation ID."`
}

// EventRecord is one event drop.
type EventRecord struct {
	ID        string `json:"id" doc:"Event identifier."`
	Type      string `json:"type" doc:"Event type (dot-separated)."`
	Timestamp string `json:"timestamp" doc:"Event timestamp (ISO-8601)." format:"date-time"`
	Source    string `json:"source" doc:"Origin subsystem." enum:"hellingd,incus,podman,k8s"`
	Payload   string `json:"payload,omitempty" doc:"Event payload as a JSON string (empty when no body)."`
}

// EventsSseInput caps the snapshot size.
type EventsSseInput struct {
	Limit int `query:"limit" default:"50" minimum:"1" maximum:"500" doc:"Maximum number of recent events to return."`
}

// EventsSseEnvelope is the success envelope for the events snapshot.
type EventsSseEnvelope struct {
	Data []EventRecord `json:"data"`
	Meta EventsMeta    `json:"meta"`
}

// EventsSseOutput is the response shape for GET /api/v1/events.
type EventsSseOutput struct {
	Body EventsSseEnvelope
}

var stubEvents = []EventRecord{
	{ID: "evt_01JZEVT00000000000000001", Type: "instance.created", Timestamp: "2026-04-23T10:00:00Z", Source: "incus"},
	{ID: "evt_01JZEVT00000000000000002", Type: "schedule.triggered", Timestamp: "2026-04-23T10:05:00Z", Source: "hellingd"},
}

func registerEventsSse(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "eventsSse",
		Method:      http.MethodGet,
		Path:        "/api/v1/events",
		Summary:     "Recent events snapshot",
		Description: "Returns a snapshot of recent internal events. Full Server-Sent-Events streaming lands in v0.1-beta.",
		Tags:        []string{"Events"},
	}, func(ctx context.Context, input *EventsSseInput) (*EventsSseOutput, error) {
		_ = ctx
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		if limit > len(stubEvents) {
			limit = len(stubEvents)
		}
		return &EventsSseOutput{
			Body: EventsSseEnvelope{
				Data: stubEvents[:limit],
				Meta: EventsMeta{RequestID: "req_events_sse"},
			},
		}, nil
	})
}
