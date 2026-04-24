package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Bizarre-Industries/Helling/apps/helling-cli/internal/client"
	"github.com/Bizarre-Industries/Helling/apps/helling-cli/internal/config"
)

// NewComputeCmd returns the `helling compute` parent. For v0.1-alpha only
// `list` is implemented; per ADR-014 proxy architecture the CLI forwards to
// Incus via hellingd /api/incus/1.0/instances.
func NewComputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compute",
		Short: "List and inspect Incus instances (via hellingd proxy)",
	}
	cmd.AddCommand(newComputeListCmd())
	return cmd
}

func newComputeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Incus instances visible to the caller",
		RunE: func(cmd *cobra.Command, _ []string) error {
			prof, err := config.Load("")
			if err != nil {
				return err
			}
			if prof.API == "" {
				return errors.New("no API endpoint configured (run 'helling auth login')")
			}
			cli, err := client.New(&prof, "")
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/incus/1.0/instances?recursion=1", nil)
			if err != nil {
				return err
			}

			type inst struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Type   string `json:"type"`
			}
			var envelope struct {
				Metadata []inst `json:"metadata"`
			}
			var arr []inst
			if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Metadata != nil {
				arr = envelope.Metadata
			} else if err := json.Unmarshal(raw, &arr); err != nil {
				_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return werr
			}

			if len(arr) == 0 {
				_, werr := fmt.Fprintln(cmd.OutOrStdout(), "No instances found.")
				return werr
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "" {
				output, _ = cmd.Root().PersistentFlags().GetString("output")
			}
			switch output {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(arr)
			default:
				var b strings.Builder
				fmt.Fprintf(&b, "%-30s %-12s %-10s\n", "NAME", "TYPE", "STATUS")
				for _, i := range arr {
					fmt.Fprintf(&b, "%-30s %-12s %-10s\n", i.Name, i.Type, i.Status)
				}
				_, werr := fmt.Fprint(cmd.OutOrStdout(), b.String())
				return werr
			}
		},
	}
}
