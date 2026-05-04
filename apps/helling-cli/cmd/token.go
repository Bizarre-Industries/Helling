package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// newAuthTokenCmd returns `helling auth token`. Covers the API token surface
// of /api/v1/auth/tokens per docs/spec/auth.md §3 (automation bearer tokens).
func newAuthTokenCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "token",
		Short: "Manage API tokens for automation",
	}
	c.AddCommand(newAuthTokenListCmd(), newAuthTokenCreateCmd(), newAuthTokenRevokeCmd())
	return c
}

func newAuthTokenListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List API tokens",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/auth/tokens", nil)
			if err != nil {
				return err
			}
			var env struct {
				Data []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Scope  string `json:"scope"`
					Prefix string `json:"prefix"`
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
			fmt.Fprintf(&b, "%-24s %-20s %-8s %s\n", "ID", "NAME", "SCOPE", "PREFIX")
			for _, t := range env.Data {
				fmt.Fprintf(&b, "%-24s %-20s %-8s %s\n", t.ID, t.Name, t.Scope, t.Prefix)
			}
			_, werr := fmt.Fprint(cmd.OutOrStdout(), b.String())
			return werr
		},
	}
}

func newAuthTokenCreateCmd() *cobra.Command {
	var scope string
	var expiresIn int
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Issue a new API token (plaintext shown once)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"name": args[0], "scope": scope}
			if expiresIn > 0 {
				body["expires_in_seconds"] = expiresIn
			}
			raw, err := cli.Do(ctx, "POST", "/api/v1/auth/tokens", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&scope, "scope", "read", "Token scope: read | write | admin")
	c.Flags().IntVar(&expiresIn, "expires-in", 0, "Expiry in seconds (default: no expiry)")
	return c
}

func newAuthTokenRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <id>",
		Short: "Revoke an API token by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			if _, err := cli.Do(ctx, "DELETE", "/api/v1/auth/tokens/"+args[0], nil); err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), "revoked "+args[0])
			return werr
		},
	}
}
