package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/client"
	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

// NewUserCmd returns the `helling user` parent. Covers the user management
// surface of /api/v1/users per docs/design/cli.md.
func NewUserCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "user",
		Short: "Manage Helling-managed user accounts",
	}
	c.AddCommand(newUserListCmd(), newUserCreateCmd(), newUserGetCmd(),
		newUserDeleteCmd(), newUserSetScopeCmd())
	return c
}

func userClient(ctx context.Context) (*client.Client, context.Context, context.CancelFunc, error) {
	prof, err := config.Load("")
	if err != nil {
		return nil, nil, nil, err
	}
	if prof.API == "" {
		return nil, nil, nil, errors.New("no API endpoint configured (run 'helling auth login')")
	}
	cli, err := client.New(&prof, "")
	if err != nil {
		return nil, nil, nil, err
	}
	c, cancel := context.WithTimeout(ctx, 30*time.Second)
	return cli, c, cancel, nil
}

//nolint:dupl // list rendering parallels token.go list; kept separate for schema clarity.
func newUserListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/users", nil)
			if err != nil {
				return err
			}
			var env struct {
				Data []struct {
					ID       string `json:"id"`
					Username string `json:"username"`
					Role     string `json:"role"`
					Status   string `json:"status"`
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
			fmt.Fprintf(&b, "%-24s %-20s %-10s %-10s\n", "ID", "USERNAME", "ROLE", "STATUS")
			for _, u := range env.Data {
				fmt.Fprintf(&b, "%-24s %-20s %-10s %-10s\n", u.ID, u.Username, u.Role, u.Status)
			}
			_, werr := fmt.Fprint(cmd.OutOrStdout(), b.String())
			return werr
		},
	}
}

func newUserCreateCmd() *cobra.Command {
	var role, password string
	c := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a Helling-managed user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"username": args[0], "role": role}
			if password != "" {
				body["password"] = password
			}
			raw, err := cli.Do(ctx, "POST", "/api/v1/users", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&role, "role", "user", "Role: admin|user|auditor")
	c.Flags().StringVar(&password, "password", "", "Optional password (argon2id hashed)")
	return c
}

func newUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Fetch a user by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/users/"+args[0], nil)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
}

func newUserDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			if _, err := cli.Do(ctx, "DELETE", "/api/v1/users/"+args[0], nil); err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), "deleted "+args[0])
			return werr
		},
	}
}

func newUserSetScopeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-scope <id> <scope>",
		Short: "Set the Incus trust scope hint for a user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"scope": args[1]}
			raw, err := cli.Do(ctx, "PUT", "/api/v1/users/"+args[0]+"/scope", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
}

// outputJSON is the enum value for JSON output; hoisted into a constant so
// subcommand format checks stay consistent and satisfy goconst lint.
const outputJSON = "json"

func outputFormat(cmd *cobra.Command) string {
	out, _ := cmd.Flags().GetString("output")
	if out == "" {
		out, _ = cmd.Root().PersistentFlags().GetString("output")
	}
	return out
}
