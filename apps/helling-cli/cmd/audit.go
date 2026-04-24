package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

// NewAuditCmd returns the `helling audit` parent. Covers /api/v1/audit
// query + export per docs/design/cli.md.
func NewAuditCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "audit",
		Short: "Query and export audit events",
	}
	c.AddCommand(newAuditQueryCmd(), newAuditExportCmd())
	return c
}

func newAuditQueryCmd() *cobra.Command {
	var actor, action, cursor string
	var limit int
	c := &cobra.Command{
		Use:   "query",
		Short: "Query audit events (optional actor/action filters)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			q := url.Values{}
			if actor != "" {
				q.Set("actor", actor)
			}
			if action != "" {
				q.Set("action", action)
			}
			if cursor != "" {
				q.Set("cursor", cursor)
			}
			if limit > 0 {
				q.Set("limit", strconv.Itoa(limit))
			}
			path := "/api/v1/audit"
			if len(q) > 0 {
				path += "?" + q.Encode()
			}
			raw, err := cli.Do(ctx, "GET", path, nil)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&actor, "actor", "", "Filter by actor (username)")
	c.Flags().StringVar(&action, "action", "", "Filter by action (e.g. auth.login)")
	c.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	c.Flags().IntVar(&limit, "limit", 0, "Max rows (default 100)")
	return c
}

func newAuditExportCmd() *cobra.Command {
	var format string
	c := &cobra.Command{
		Use:   "export",
		Short: "Bulk export audit events as JSON or CSV",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/audit/export?format="+url.QueryEscape(format), nil)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&format, "format", "json", "Output format: json | csv")
	return c
}
