package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

// NewEventsCmd returns `helling events`. v0.1-alpha exposes eventsSse as a
// polling snapshot (docs/spec/events.md); `tail` fetches the latest batch.
// Streaming SSE lands in v0.1-beta.
func NewEventsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "events",
		Short: "Inspect recent internal events",
	}
	c.AddCommand(newEventsTailCmd())
	return c
}

func newEventsTailCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "tail",
		Short: "Show the most recent events (polling snapshot)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			q := url.Values{}
			if limit > 0 {
				q.Set("limit", strconv.Itoa(limit))
			}
			path := "/api/v1/events"
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
	c.Flags().IntVar(&limit, "limit", 0, "Max events to return (default 50, max 500)")
	return c
}
