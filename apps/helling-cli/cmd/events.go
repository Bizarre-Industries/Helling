package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// NewEventsCmd returns `helling events`.
func NewEventsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "events",
		Short: "Inspect recent internal events",
	}
	c.AddCommand(newEventsTailCmd(), newEventsListCmd())
	return c
}

func newEventsTailCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tail",
		Short: "Stream events over SSE",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			path := "/api/v1/events"
			resp, err := cli.Stream(ctx, path, "text/event-stream")
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()
			return copySSEData(cmd.OutOrStdout(), resp.Body)
		},
	}
	return c
}

func newEventsListCmd() *cobra.Command {
	var limit int
	c := &cobra.Command{
		Use:   "list [count]",
		Short: "List recent events as a JSON snapshot",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			if len(args) == 1 && !cmd.Flags().Changed("limit") {
				parsed, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("parse count %q: %w", args[0], err)
				}
				limit = parsed
			}
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

func copySSEData(out io.Writer, in io.Reader) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		if data, ok := strings.CutPrefix(line, "data: "); ok {
			if _, err := fmt.Fprintln(out, data); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") || strings.HasPrefix(line, "id: ") || strings.HasPrefix(line, ":") || line == "" {
			continue
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return scanner.Err()
}
