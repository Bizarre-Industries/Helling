package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newAuthMfaCmd returns `helling auth mfa`. Covers the TOTP enrolment and
// disable surface of /api/v1/auth/totp/* per docs/spec/auth.md §4.
func newAuthMfaCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "mfa",
		Short: "Enroll, verify, and disable TOTP MFA",
	}
	c.AddCommand(newAuthMfaSetupCmd(), newAuthMfaVerifyCmd(), newAuthMfaDisableCmd())
	return c
}

func newAuthMfaSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Begin TOTP enrolment (prints otpauth URI + recovery codes)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "POST", "/api/v1/auth/totp/setup", struct{}{})
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
}

func newAuthMfaVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify <code>",
		Short: "Confirm the TOTP enrolment with a 6-digit code from the authenticator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"totp_code": args[0]}
			raw, err := cli.Do(ctx, "POST", "/api/v1/auth/totp/verify", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
}

func newAuthMfaDisableCmd() *cobra.Command {
	var password string
	c := &cobra.Command{
		Use:   "disable",
		Short: "Disable TOTP MFA (requires the current password)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"password": password}
			raw, err := cli.Do(ctx, "POST", "/api/v1/auth/totp/disable", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&password, "password", "", "Current password (required)")
	_ = c.MarkFlagRequired("password")
	return c
}
