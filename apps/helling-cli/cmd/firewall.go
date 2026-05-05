package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewFirewallCmd returns the host firewall command tree.
func NewFirewallCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "firewall",
		Short: "Manage Helling host firewall rules",
	}
	c.AddCommand(newFirewallListCmd(), newFirewallAddCmd(), newFirewallDeleteCmd())
	return c
}

func newFirewallListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List host firewall rules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			raw, err := cli.Do(ctx, "GET", "/api/v1/firewall/host", nil)
			if err != nil {
				return err
			}
			var env struct {
				Data []struct {
					ID         string `json:"id"`
					Direction  string `json:"direction"`
					Action     string `json:"action"`
					Protocol   string `json:"protocol"`
					SourceCIDR string `json:"source_cidr"`
					DestPort   int    `json:"destination_port"`
					NFTComment string `json:"nft_comment"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &env); err != nil || outputFormat(cmd) == outputJSON {
				_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
				return werr
			}
			var b strings.Builder
			fmt.Fprintf(&b, "%-36s %-8s %-8s %-8s %-18s %-6s %s\n", "ID", "DIR", "ACTION", "PROTO", "SOURCE", "DPORT", "NFT")
			for _, r := range env.Data {
				fmt.Fprintf(&b, "%-36s %-8s %-8s %-8s %-18s %-6d %s\n", r.ID, r.Direction, r.Action, r.Protocol, r.SourceCIDR, r.DestPort, r.NFTComment)
			}
			_, werr := fmt.Fprint(cmd.OutOrStdout(), b.String())
			return werr
		},
	}
}

func newFirewallAddCmd() *cobra.Command {
	var direction, action, protocol, sourceCIDR, destCIDR, comment string
	var destPort int
	c := &cobra.Command{
		Use:   "add",
		Short: "Add a host firewall rule",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			body := map[string]any{"direction": direction, "action": action, "protocol": protocol}
			if sourceCIDR != "" {
				body["source_cidr"] = sourceCIDR
			}
			if destCIDR != "" {
				body["destination_cidr"] = destCIDR
			}
			if destPort > 0 {
				body["destination_port"] = destPort
			}
			if comment != "" {
				body["comment"] = comment
			}
			raw, err := cli.Do(ctx, "POST", "/api/v1/firewall/host", body)
			if err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return werr
		},
	}
	c.Flags().StringVar(&direction, "direction", "input", "Direction: input | output | forward")
	c.Flags().StringVar(&action, "action", "accept", "Action: accept | drop | reject")
	c.Flags().StringVar(&protocol, "protocol", "tcp", "Protocol: tcp | udp | icmp | any")
	c.Flags().StringVar(&sourceCIDR, "source-cidr", "", "Optional source CIDR")
	c.Flags().StringVar(&destCIDR, "destination-cidr", "", "Optional destination CIDR")
	c.Flags().IntVar(&destPort, "destination-port", 0, "Optional destination port")
	c.Flags().StringVar(&comment, "comment", "", "Operator comment")
	return c
}

func newFirewallDeleteCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a host firewall rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := confirmDestructiveAction(cmd, yes, "delete host firewall rule", args[0]); err != nil {
				return err
			}
			cli, ctx, cancel, err := userClient(cmd.Context())
			if err != nil {
				return err
			}
			defer cancel()
			if _, err := cli.Do(ctx, "DELETE", "/api/v1/firewall/host/"+args[0], nil); err != nil {
				return err
			}
			_, werr := fmt.Fprintln(cmd.OutOrStdout(), "deleted "+args[0])
			return werr
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "Skip interactive confirmation")
	return c
}
