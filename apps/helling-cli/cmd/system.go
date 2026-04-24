package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewSystemCmd returns the `helling system` parent. Covers the
// /api/v1/system/* surface per docs/design/cli.md.
func NewSystemCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "system",
		Short: "Inspect host info, hardware, config, and diagnostics",
	}
	c.AddCommand(newSystemInfoCmd(), newSystemHardwareCmd(), newSystemDiagnosticsCmd(),
		newSystemConfigGetCmd(), newSystemConfigPutCmd(), newSystemUpgradeCmd(),
		newSystemHealthCmd())
	return c
}

func systemGet(cmd *cobra.Command, path string) error {
	cli, ctx, cancel, err := userClient(cmd.Context())
	if err != nil {
		return err
	}
	defer cancel()
	raw, err := cli.Do(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
	return werr
}

func newSystemInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show hellingd version, uptime, and runtime info",
		RunE:  func(cmd *cobra.Command, _ []string) error { return systemGet(cmd, "/api/v1/system/info") },
	}
}

func newSystemHardwareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hardware",
		Short: "Show CPU, memory, and NIC inventory",
		RunE:  func(cmd *cobra.Command, _ []string) error { return systemGet(cmd, "/api/v1/system/hardware") },
	}
}

func newSystemDiagnosticsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diagnostics",
		Short: "Run the built-in health probes (db, signer, sysinfo)",
		RunE:  func(cmd *cobra.Command, _ []string) error { return systemGet(cmd, "/api/v1/system/diagnostics") },
	}
}

func newSystemConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config-get <key>",
		Short: "Get a runtime configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return systemGet(cmd, "/api/v1/system/config/"+args[0])
		},
	}
}

func newSystemConfigPutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config-set <key> <value>",
		Short: "Set a runtime configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"value": args[1]}
			raw, err := cli.Do(ctx, "PUT", "/api/v1/system/config/"+args[0], body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
}

func newSystemHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Query hellingd health endpoint (liveness + subsystem checks)",
		RunE:  func(cmd *cobra.Command, _ []string) error { return systemGet(cmd, "/api/v1/health") },
	}
}

func newSystemUpgradeCmd() *cobra.Command {
	var rollback bool
	c := &cobra.Command{
		Use:   "upgrade",
		Short: "Trigger an upgrade check or rollback",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"rollback": rollback}
			raw, err := cli.Do(ctx, "POST", "/api/v1/system/upgrade", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().BoolVar(&rollback, "rollback", false, "Revert to the previous version instead of upgrading")
	return c
}
