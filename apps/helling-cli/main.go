// Package main is the entry point for the helling CLI.
//
// Scope for v0.1.0-alpha: offline-only subcommands (version, completion).
// Network-backed subcommands (auth, user, system, ...) land once the generated
// oapi-codegen client is wired and hellingd exposes more than the 3-endpoint
// Huma spike. See docs/spec/cli.md and docs/roadmap/plan.md.
package main

import (
	"io"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/cmd"
)

// Build-time injected via -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var exitFunc = os.Exit

func newRootCmd(out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "helling",
		Short:         "Helling control-plane CLI",
		Long:          "helling manages Helling-owned resources. See docs/spec/cli.md for the full command tree.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetOut(out)
	root.SetErr(errOut)

	// Global flags per docs/spec/cli.md §Global Flags. Values are wired to
	// persistent config once the HTTP client lands; unused until then.
	root.PersistentFlags().String("api", "", "hellingd API endpoint (default: from config)")
	root.PersistentFlags().String("token", "", "API token (default: from config)")
	root.PersistentFlags().String("output", "table", "Output format: table, json, yaml")
	root.PersistentFlags().Bool("quiet", false, "Minimal output")

	root.AddCommand(newVersionCmd())
	root.AddCommand(cmd.NewAuthCmd())
	root.AddCommand(cmd.NewComputeCmd())
	root.AddCommand(cmd.NewUserCmd())
	root.AddCommand(cmd.NewWebhookCmd())
	root.AddCommand(cmd.NewSystemCmd())
	root.AddCommand(cmd.NewAuditCmd())
	root.AddCommand(cmd.NewEventsCmd())

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, git commit, build date, and Go runtime",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitCommit := commit
			if gitCommit == "none" {
				if info, ok := debug.ReadBuildInfo(); ok {
					for _, s := range info.Settings {
						if s.Key == "vcs.revision" && s.Value != "" {
							gitCommit = s.Value
							break
						}
					}
				}
			}
			_, err := cmd.OutOrStdout().Write([]byte(
				"helling " + version +
					"\n  commit: " + gitCommit +
					"\n  built:  " + date +
					"\n  go:     " + runtime.Version() +
					"\n",
			))
			return err
		},
	}
}

func run(args []string, out, errOut io.Writer) int {
	root := newRootCmd(out, errOut)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		_, _ = errOut.Write([]byte("helling: " + err.Error() + "\n"))
		return 1
	}
	return 0
}

func main() {
	exitFunc(run(os.Args[1:], os.Stdout, os.Stderr))
}
