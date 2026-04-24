package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/sysinfo"
)

// System real handlers back the /api/v1/system* stubs. info/hardware probe
// the live host via internal/sysinfo; config persists through the
// system_config table added in migration 0004. upgrade + diagnostics stay
// lightweight for v0.1-alpha: upgrade reports no_change; diagnostics runs a
// DB-reachable probe plus a JWT-signer probe.

type systemConfigGetBearerInput struct {
	Authorization string `header:"Authorization"`
	Key           string `path:"key" minLength:"1" maxLength:"128"`
}

type systemConfigPutBearerInput struct {
	Authorization string `header:"Authorization"`
	Key           string `path:"key" minLength:"1" maxLength:"128"`
	Body          SystemConfigPutRequest
}

type systemUpgradeBearerInput struct {
	Authorization string `header:"Authorization"`
	Body          SystemUpgradeRequest
}

const hellingdSemver = "0.1.0-alpha"

func registerSystemInfoReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "systemInfo",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/info",
		Summary:     "Show hellingd system info",
		Description: "Returns live hostname, running hellingd version, uptime-since-boot, arch, and kernel release.",
		Tags:        []string{"System"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *bearerInput) (*SystemInfoOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		info := sysinfo.Collect(hellingdSemver)
		return &SystemInfoOutput{
			Body: SystemInfoEnvelope{
				Data: SystemInfoData{
					Hostname: info.Hostname,
					Version:  info.Version,
					Uptime:   info.Uptime,
					Arch:     info.Arch,
					Kernel:   info.Kernel,
				},
				Meta: SystemMeta{RequestID: "req_system_info"},
			},
		}, nil
	})
}

func registerSystemHardwareReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "systemHardware",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/hardware",
		Summary:     "Show detected hardware",
		Tags:        []string{"System"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *bearerInput) (*SystemHardwareOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		hw := sysinfo.CollectHardware()
		return &SystemHardwareOutput{
			Body: SystemHardwareEnvelope{
				Data: SystemHardwareData{
					CPU: hw.CPU, Cores: hw.Cores, RAMGB: hw.RAMGB, DiskGB: hw.DiskGB, Network: hw.Network,
				},
				Meta: SystemMeta{RequestID: "req_system_hardware"},
			},
		}, nil
	})
}

func registerSystemConfigGetReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "systemConfigGet",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/config/{key}",
		Summary:     "Read a Helling config key",
		Tags:        []string{"System"},
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound},
	}, func(ctx context.Context, input *systemConfigGetBearerInput) (*SystemConfigGetOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		val, err := svc.Repo().GetSystemConfig(ctx, input.Key)
		if errors.Is(err, authrepo.ErrNotFound) {
			return nil, huma.Error404NotFound("SYSTEM_CONFIG_NOT_FOUND")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("SYSTEM_CONFIG_GET_FAILED")
		}
		return &SystemConfigGetOutput{
			Body: SystemConfigEnvelope{
				Data: SystemConfigData{Key: input.Key, Value: val},
				Meta: SystemMeta{RequestID: "req_system_config_get"},
			},
		}, nil
	})
}

func registerSystemConfigPutReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "systemConfigPut",
		Method:      http.MethodPut,
		Path:        "/api/v1/system/config/{key}",
		Summary:     "Upsert a Helling config key",
		Tags:        []string{"System"},
		RequestBody: &huma.RequestBody{Description: "Value payload.", Required: true},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *systemConfigPutBearerInput) (*SystemConfigPutOutput, error) {
		userID, err := resolveCaller(ctx, svc, input.Authorization)
		if err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		if err := svc.Repo().PutSystemConfig(ctx, input.Key, input.Body.Value, userID); err != nil {
			return nil, huma.Error500InternalServerError("SYSTEM_CONFIG_PUT_FAILED")
		}
		return &SystemConfigPutOutput{
			Body: SystemConfigEnvelope{
				Data: SystemConfigData{Key: input.Key, Value: input.Body.Value},
				Meta: SystemMeta{RequestID: "req_system_config_put"},
			},
		}, nil
	})
}

func registerSystemUpgradeReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "systemUpgrade",
		Method:      http.MethodPost,
		Path:        "/api/v1/system/upgrade",
		Summary:     "Upgrade hellingd",
		Description: "v0.1-alpha reports no_change; real upgrade wiring lands with the APT repo (ADR-025) in v0.1-beta.",
		Tags:        []string{"System"},
		RequestBody: &huma.RequestBody{Description: "Upgrade payload.", Required: false},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *systemUpgradeBearerInput) (*SystemUpgradeOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		return &SystemUpgradeOutput{
			Body: SystemUpgradeEnvelope{
				Data: SystemUpgradeData{
					FromVersion: hellingdSemver,
					ToVersion:   hellingdSemver,
					Status:      "no_change",
				},
				Meta: SystemMeta{RequestID: "req_system_upgrade"},
			},
		}, nil
	})
}

func registerSystemDiagnosticsReal(api huma.API, svc *auth.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "systemDiagnostics",
		Method:      http.MethodGet,
		Path:        "/api/v1/system/diagnostics",
		Summary:     "Run hellingd self-diagnostics",
		Tags:        []string{"System"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *bearerInput) (*SystemDiagnosticsOutput, error) {
		if _, err := resolveCaller(ctx, svc, input.Authorization); err != nil {
			return nil, huma.Error401Unauthorized("AUTH_UNAUTHENTICATED")
		}
		checks := make([]SystemDiagnosticsCheck, 0, 3)
		passed, failed := 0, 0

		if _, err := svc.Repo().CountAdmins(ctx); err == nil {
			checks = append(checks, SystemDiagnosticsCheck{Name: "db.reachable", Status: "pass"})
			passed++
		} else {
			checks = append(checks, SystemDiagnosticsCheck{Name: "db.reachable", Status: "fail", Message: err.Error()})
			failed++
		}

		if svc.Signer() != nil && svc.Signer().Public() != nil {
			checks = append(checks, SystemDiagnosticsCheck{Name: "auth.signer", Status: "pass"})
			passed++
		} else {
			checks = append(checks, SystemDiagnosticsCheck{Name: "auth.signer", Status: "fail", Message: "signer unavailable"})
			failed++
		}

		info := sysinfo.Collect(hellingdSemver)
		checks = append(checks, SystemDiagnosticsCheck{
			Name:    "sysinfo.collect",
			Status:  "pass",
			Message: info.Kernel,
		})
		passed++

		return &SystemDiagnosticsOutput{
			Body: SystemDiagnosticsEnvelope{
				Data: SystemDiagnosticsData{Checks: checks, Passed: passed, Failed: failed},
				Meta: SystemMeta{RequestID: "req_system_diagnostics"},
			},
		}, nil
	})
}
