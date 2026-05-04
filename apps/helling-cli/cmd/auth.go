// Package cmd hosts the Cobra subcommands consumed from main.go.
//
// Each subcommand is a small wrapper that reads the persisted profile,
// performs one HTTP round-trip, prints a human-friendly summary, and
// persists any rotated credentials back to disk.
package cmd

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/client"
	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

// NewAuthCmd returns the `helling auth` parent command.
func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage the CLI session (login, logout, whoami)",
	}
	cmd.AddCommand(newAuthSetupCmd())
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthWhoamiCmd())
	cmd.AddCommand(newAuthTokenCmd())
	cmd.AddCommand(newAuthMfaCmd())
	return cmd
}

func newAuthSetupCmd() *cobra.Command {
	var username, passwordFile, setupTokenFile string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Create the first admin account with the installer setup token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAuthSetup(cmd, &username, passwordFile, setupTokenFile)
		},
	}
	cmd.Flags().StringVar(&username, "username", "admin", "Initial admin username")
	cmd.Flags().StringVar(&passwordFile, "password-file", "", "Read initial admin password from file (prompted when omitted)")
	cmd.Flags().StringVar(&setupTokenFile, "setup-token-file", "", "Read setup token from file, usually /etc/helling/setup-token (prompted when omitted)")
	return cmd
}

func runAuthSetup(cmd *cobra.Command, username *string, passwordFile, setupTokenFile string) error {
	api, _ := cmd.Flags().GetString("api")
	if api == "" {
		prof, _ := config.Load("")
		api = prof.API
	}
	if api == "" {
		return errors.New("--api is required for first-admin setup (e.g. http+unix:///run/helling/api.sock)")
	}
	if err := promptIfEmpty(cmd, username, "Username: "); err != nil {
		return err
	}
	password, passwordPrompted, err := readSetupSecret(cmd, passwordFile, "password-file", "Password: ")
	if err != nil {
		return err
	}
	if passwordPrompted {
		confirmation, err := promptSecret(cmd, "password-file", "Confirm password: ")
		if err != nil {
			return err
		}
		if password != confirmation {
			return errors.New("passwords do not match")
		}
	}
	setupToken, _, err := readSetupSecret(cmd, setupTokenFile, "setup-token-file", "Setup token: ")
	if err != nil {
		return err
	}
	inputs, err := validateAuthSetupInputs(*username, password, setupToken)
	if err != nil {
		return err
	}

	prof, err := config.Load("")
	if err != nil {
		return err
	}
	prof.API = api
	cli, err := client.New(&prof, api)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	if _, err := cli.Do(ctx, "POST", "/api/v1/auth/setup", map[string]string{
		"username":    inputs.username,
		"password":    password,
		"setup_token": inputs.setupToken,
	}); err != nil {
		return err
	}
	if err := config.Save(&prof, ""); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created first admin %s at %s. Run 'helling auth login' next.\n", inputs.username, api)
	return err
}

type authSetupInputs struct {
	username   string
	setupToken string
}

func validateAuthSetupInputs(username, password, setupToken string) (authSetupInputs, error) {
	username = strings.TrimSpace(username)
	setupToken = strings.TrimSpace(setupToken)
	if username == "" {
		return authSetupInputs{}, errors.New("username is required")
	}
	if len(username) > 64 {
		return authSetupInputs{}, errors.New("username must be at most 64 characters")
	}
	if len(password) < 8 || len(password) > 256 {
		return authSetupInputs{}, errors.New("password must be 8 to 256 characters")
	}
	if len(setupToken) < 32 || len(setupToken) > 128 {
		return authSetupInputs{}, errors.New("setup token must be 32 to 128 characters")
	}
	return authSetupInputs{username: username, setupToken: setupToken}, nil
}

func newAuthLoginCmd() *cobra.Command {
	var username, password string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate to hellingd and persist the session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAuthLogin(cmd, &username, &password)
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Username (prompted when omitted)")
	cmd.Flags().StringVar(&password, "password", "", "Password (prompted when omitted; NOT recommended on shared hosts)")
	return cmd
}

func runAuthLogin(cmd *cobra.Command, username, password *string) error {
	api, _ := cmd.Flags().GetString("api")
	if api == "" {
		prof, _ := config.Load("")
		api = prof.API
	}
	if api == "" {
		return errors.New("--api is required on first login (e.g. http://127.0.0.1:8080)")
	}
	if err := promptIfEmpty(cmd, username, "Username: "); err != nil {
		return err
	}
	if err := promptSecretIfEmpty(cmd, password, "password", "Password: "); err != nil {
		return err
	}
	usernameValue := strings.TrimSpace(*username)
	if usernameValue == "" {
		return errors.New("username is required")
	}
	if *password == "" {
		return errors.New("password is required")
	}

	prof, err := config.Load("")
	if err != nil {
		return err
	}
	prof.API = api
	cli, err := client.New(&prof, api)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	access, err := performLogin(ctx, cmd, cli, usernameValue, *password)
	if err != nil {
		return err
	}

	prof.AccessToken = access
	prof.RefreshCookie = cli.RefreshCookie()
	if err := config.Save(&prof, ""); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s at %s.\n", usernameValue, api)
	return err
}

func promptIfEmpty(cmd *cobra.Command, target *string, prompt string) error {
	if *target != "" {
		return nil
	}
	v, err := readLine(cmd.InOrStdin(), cmd.OutOrStdout(), prompt)
	if err != nil {
		return err
	}
	*target = v
	return nil
}

func promptSecretIfEmpty(cmd *cobra.Command, target *string, flagName, prompt string) error {
	if *target != "" {
		return nil
	}
	v, err := promptSecret(cmd, flagName, prompt)
	if err != nil {
		return err
	}
	*target = v
	return nil
}

func readSetupSecret(cmd *cobra.Command, filePath, flagName, prompt string) (value string, prompted bool, err error) {
	if filePath != "" {
		raw, err := os.ReadFile(filePath) // #nosec G304 -- local operator-supplied secret file.
		if err != nil {
			return "", false, fmt.Errorf("read --%s %s: %w", flagName, filePath, err)
		}
		return strings.TrimRight(string(raw), "\r\n"), false, nil
	}
	v, err := promptSecret(cmd, flagName, prompt)
	return v, true, err
}

func promptSecret(cmd *cobra.Command, flagName, prompt string) (string, error) {
	inFile, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(inFile.Fd())) {
		return "", fmt.Errorf("--%s is required in non-interactive mode", flagName)
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), prompt); err != nil {
		return "", err
	}
	raw, err := term.ReadPassword(int(inFile.Fd()))
	if _, printErr := fmt.Fprintln(cmd.OutOrStdout()); printErr != nil && err == nil {
		err = printErr
	}
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func performLogin(ctx context.Context, cmd *cobra.Command, cli *client.Client, username, password string) (string, error) {
	body := map[string]string{"username": username, "password": password}
	raw, err := cli.Do(ctx, "POST", "/api/v1/auth/login", body)
	if err != nil {
		return "", err
	}
	var env struct {
		Data struct {
			AccessToken string `json:"access_token"`
			MFARequired bool   `json:"mfa_required"`
			MFAToken    string `json:"mfa_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("login: decode: %w", err)
	}
	if !env.Data.MFARequired {
		return env.Data.AccessToken, nil
	}

	totp, err := readLine(cmd.InOrStdin(), cmd.OutOrStdout(), "TOTP or recovery code: ")
	if err != nil {
		return "", err
	}
	raw, err = cli.Do(ctx, "POST", "/api/v1/auth/mfa/complete", map[string]string{
		"mfa_token": env.Data.MFAToken,
		"totp_code": totp,
	})
	if err != nil {
		return "", err
	}
	var mfaEnv struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &mfaEnv); err != nil {
		return "", fmt.Errorf("mfa: decode: %w", err)
	}
	return mfaEnv.Data.AccessToken, nil
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke the stored session and clear local credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			prof, err := config.Load("")
			if err != nil {
				return err
			}
			if prof.API == "" {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Already logged out.")
				return err
			}
			cli, err := client.New(&prof, "")
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			if _, err := cli.Do(ctx, "POST", "/api/v1/auth/logout", nil); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warn: server logout failed: %v\n", err)
			}
			prof.AccessToken = ""
			prof.RefreshCookie = ""
			if err := config.Save(&prof, ""); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return err
		},
	}
}

func newAuthWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the caller identity by decoding the stored JWT claims",
		RunE: func(cmd *cobra.Command, _ []string) error {
			prof, err := config.Load("")
			if err != nil {
				return err
			}
			if prof.AccessToken == "" && prof.Token == "" {
				return errors.New("not logged in (run 'helling auth login')")
			}
			if prof.Token != "" {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "API token in use.")
				return err
			}
			username, role, expUnix, err := decodeJWTClaims(prof.AccessToken)
			if err != nil {
				return err
			}
			exp := time.Unix(expUnix, 0).UTC().Format(time.RFC3339)
			_, err = fmt.Fprintf(cmd.OutOrStdout(),
				"user:    %s\nrole:    %s\nexpires: %s\napi:     %s\n",
				username, role, exp, prof.API,
			)
			return err
		},
	}
}

// decodeJWTClaims parses the JWT payload without verifying the signature.
// The CLI is not an authoritative verifier; hellingd validates on each call.
func decodeJWTClaims(tok string) (username, role string, exp int64, err error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return "", "", 0, errors.New("invalid JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", 0, err
	}
	var c struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		Exp      int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &c); err != nil {
		return "", "", 0, err
	}
	return c.Username, c.Role, c.Exp, nil
}

func readLine(in io.Reader, out io.Writer, prompt string) (string, error) {
	_, _ = fmt.Fprint(out, prompt)
	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
