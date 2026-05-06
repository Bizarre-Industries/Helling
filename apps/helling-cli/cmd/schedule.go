package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/client"
	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

const defaultScheduleRunnerTokenPath = "/etc/helling/schedule-runner.token" // #nosec G101 -- this is a token file path, not token material.

// NewScheduleCmd returns the schedule command tree.
func NewScheduleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "schedule",
		Short: "Manage systemd-backed Helling schedules",
	}
	c.AddCommand(newScheduleListCmd(), newScheduleCreateCmd(), newScheduleGetCmd(),
		newScheduleUpdateCmd(), newScheduleRunCmd(), newScheduleDeleteCmd())
	return c
}

func newScheduleListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/schedules", nil)
			if err != nil {
				return err
			}
			var env struct {
				Data []struct {
					ID         string `json:"id"`
					Name       string `json:"name"`
					Kind       string `json:"kind"`
					Target     string `json:"target"`
					OnCalendar string `json:"on_calendar"`
					Enabled    bool   `json:"enabled"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &env); err != nil || outputFormat(cmd) == outputJSON {
				_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return werr
			}
			var b strings.Builder
			fmt.Fprintf(&b, "%-36s %-20s %-10s %-20s %-20s %-8s\n", "ID", "NAME", "KIND", "TARGET", "ON_CALENDAR", "ENABLED")
			for _, s := range env.Data {
				fmt.Fprintf(&b, "%-36s %-20s %-10s %-20s %-20s %-8t\n", s.ID, s.Name, s.Kind, s.Target, s.OnCalendar, s.Enabled)
			}
			_, werr := fmt.Fprint(cmd.OutOrStdout(), b.String())
			return werr
		},
	}
}

func newScheduleCreateCmd() *cobra.Command {
	var kind, target, onCalendar string
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"name": args[0], "kind": kind, "target": target, "on_calendar": onCalendar}
			raw, err := cli.Do(ctx, "POST", "/api/v1/schedules", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&kind, "kind", "backup", "Schedule kind: backup | snapshot")
	c.Flags().StringVar(&target, "target", "", "Target instance or selector")
	c.Flags().StringVar(&onCalendar, "on-calendar", "", "systemd OnCalendar expression")
	_ = c.MarkFlagRequired("target")
	_ = c.MarkFlagRequired("on-calendar")
	return c
}

func newScheduleGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Fetch a schedule by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/schedules/"+args[0], nil)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
}

func newScheduleUpdateCmd() *cobra.Command {
	var name, onCalendar string
	var enabled bool
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{}
			if name != "" {
				body["name"] = name
			}
			if onCalendar != "" {
				body["on_calendar"] = onCalendar
			}
			if cmd.Flags().Changed("enabled") {
				body["enabled"] = enabled
			}
			if len(body) == 0 {
				return errors.New("at least one of --name, --on-calendar, or --enabled is required")
			}
			raw, err := cli.Do(ctx, "PATCH", "/api/v1/schedules/"+args[0], body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&name, "name", "", "New schedule name")
	c.Flags().StringVar(&onCalendar, "on-calendar", "", "New systemd OnCalendar expression")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enabled flag")
	return c
}

func newScheduleRunCmd() *cobra.Command {
	var system bool
	c := &cobra.Command{
		Use:   "run <id>",
		Short: "Run a schedule now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if system {
				return runScheduleAsSystem(cmd, args[0])
			}
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "POST", "/api/v1/schedules/"+args[0]+"/run", struct{}{})
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().BoolVar(&system, "system", false, "Use the local schedule-runner token for systemd timers")
	return c
}

func runScheduleAsSystem(cmd *cobra.Command, id string) error {
	tokenPath := os.Getenv("HELLING_SCHEDULE_TOKEN_FILE")
	if tokenPath == "" {
		tokenPath = defaultScheduleRunnerTokenPath
	}
	body, err := os.ReadFile(tokenPath) //nolint:gosec // path is fixed by packaged unit or operator env.
	if err != nil {
		return fmt.Errorf("read schedule runner token: %w", err)
	}
	api := os.Getenv("HELLING_API")
	if api == "" {
		api = "http+unix:///run/helling/api.sock"
	}
	cli, err := client.New(&config.Profile{API: api}, "")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	raw, err := cli.DoWithHeaders(ctx, "POST", "/api/v1/internal/schedules/"+id+"/run", struct{}{}, map[string]string{
		"X-Helling-Schedule-Token": strings.TrimSpace(string(body)),
	})
	if err != nil {
		return err
	}
	_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
	return werr
}

func newScheduleDeleteCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmDestructiveAction(cmd, yes, "delete schedule", args[0]); err != nil {
				return err
			}
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			if _, err := cli.Do(ctx, "DELETE", "/api/v1/schedules/"+args[0], nil); err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), "deleted "+args[0])
			return werr
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "Skip interactive confirmation")
	return c
}
