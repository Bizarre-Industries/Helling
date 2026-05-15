package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewWebhookCmd returns the `helling webhook` parent. Covers the
// /api/v1/webhooks CRUD + test surface per docs/design/cli.md.
func NewWebhookCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "webhook",
		Short: "Manage outbound webhooks",
	}
	c.AddCommand(newWebhookListCmd(), newWebhookCreateCmd(), newWebhookGetCmd(),
		newWebhookUpdateCmd(), newWebhookDeleteCmd(), newWebhookTestCmd())
	return c
}

func newWebhookListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/webhooks", nil)
			if err != nil {
				return err
			}
			var env struct {
				Data []struct {
					ID      string   `json:"id"`
					Name    string   `json:"name"`
					URL     string   `json:"url"`
					Events  []string `json:"events"`
					Enabled bool     `json:"enabled"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &env); err != nil {
				_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return werr
			}
			if outputFormat(cmd) == outputJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(env.Data)
			}
			var b strings.Builder
			fmt.Fprintf(&b, "%-24s %-20s %-8s %s\n", "ID", "NAME", "ENABLED", "URL")
			for _, w := range env.Data {
				fmt.Fprintf(&b, "%-24s %-20s %-8t %s\n", w.ID, w.Name, w.Enabled, w.URL)
			}
			_, werr := fmt.Fprint(cmd.OutOrStdout(), b.String())
			return werr
		},
	}
}

func newWebhookCreateCmd() *cobra.Command {
	var url, secret string
	var secretStdin bool
	var events []string
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Register a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			if secretStdin {
				secret, err = readSecretFromStdin(cmd.InOrStdin())
				if err != nil {
					return err
				}
			}
			body := map[string]any{
				"name":   args[0],
				"url":    url,
				"secret": secret,
				"events": events,
			}
			raw, err := cli.Do(ctx, "POST", "/api/v1/webhooks", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&url, "url", "", "Destination URL (required)")
	c.Flags().StringVar(&secret, "secret", "", "HMAC secret (deprecated: exposes secret in process args)")
	c.Flags().BoolVar(&secretStdin, "secret-stdin", false, "Read HMAC secret from stdin")
	_ = c.Flags().MarkDeprecated("secret", "use --secret-stdin to avoid leaking secrets in process args")
	c.Flags().StringSliceVar(&events, "events", nil, "Event types (comma-separated)")
	_ = c.MarkFlagRequired("url")
	return c
}

func newWebhookGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Fetch a webhook by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/webhooks/"+args[0], nil)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
}

func newWebhookUpdateCmd() *cobra.Command {
	var name, url string
	var enabled bool
	var events []string
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a webhook (partial)",
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
			if url != "" {
				body["url"] = url
			}
			if len(events) > 0 {
				body["events"] = events
			}
			if cmd.Flags().Changed("enabled") {
				body["enabled"] = enabled
			}
			raw, err := cli.Do(ctx, "PATCH", "/api/v1/webhooks/"+args[0], body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&name, "name", "", "New name")
	c.Flags().StringVar(&url, "url", "", "New URL")
	c.Flags().StringSliceVar(&events, "events", nil, "New event list")
	c.Flags().BoolVar(&enabled, "enabled", true, "Enabled flag")
	return c
}

func newWebhookDeleteCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmDestructiveAction(cmd, yes, "delete webhook", args[0]); err != nil {
				return err
			}
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			if _, err := cli.Do(ctx, "DELETE", "/api/v1/webhooks/"+args[0], nil); err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), "deleted "+args[0])
			return werr
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "Skip interactive confirmation")
	return c
}

func newWebhookTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <id>",
		Short: "Send a ping to a webhook and record the delivery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "POST", "/api/v1/webhooks/"+args[0]+"/test", struct{}{})
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
}
