// Package firewall validates the narrow nft(8) argv surface used by Helling's
// privileged host-firewall helper.
package firewall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	// DefaultHelperPath is the installed path for the narrow firewall helper.
	DefaultHelperPath = "/usr/lib/helling/helling-firewall"
	// DefaultNFTPath is the nft(8) binary path executed by the helper.
	DefaultNFTPath = "/usr/sbin/nft"

	nftFamily = "inet"
	nftTable  = "helling"
)

var nftCommentPattern = regexp.MustCompile(`^helling:[0-9a-f-]{36}$`)

// Helper validates helper arguments and executes nft(8).
type Helper struct {
	NFTPath    string
	Stdout     io.Writer
	Stderr     io.Writer
	RunCommand func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// Handle validates and executes one helper invocation.
func (h Helper) Handle(ctx context.Context, args []string) error {
	if err := ValidateNFTArgs(args); err != nil {
		return err
	}
	if args[0] == "delete" {
		if err := h.validateDeleteOwnership(ctx, args); err != nil {
			return err
		}
	}
	out, err := h.run(ctx, h.nftPath(), args...)
	if len(out) > 0 {
		w := h.Stdout
		if w == nil {
			w = os.Stdout
		}
		if _, werr := w.Write(out); werr != nil {
			return fmt.Errorf("writing nft output: %w", werr)
		}
	}
	if err != nil {
		return fmt.Errorf("running nft: %w", err)
	}
	return nil
}

func (h Helper) validateDeleteOwnership(ctx context.Context, args []string) error {
	chain := args[4]
	handle := args[6]
	out, err := h.run(ctx, h.nftPath(), "-a", "list", "chain", nftFamily, nftTable, chain)
	if err != nil {
		return fmt.Errorf("listing nft chain before delete: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "handle "+handle) {
			continue
		}
		if strings.Contains(line, `comment "helling:`) {
			return nil
		}
		return errors.New("nft delete target is not Helling-owned")
	}
	return errors.New("nft delete target handle not found")
}

// ValidateNFTArgs accepts only Helling-owned nft table, chain, and rule forms.
func ValidateNFTArgs(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: helling-firewall <validated nft args>")
	}
	if args[0] == "-a" {
		if len(args) < 2 || args[1] != "list" {
			return errors.New("nft -a is only allowed for list commands")
		}
		return validateListArgs(args[1:])
	}
	switch args[0] {
	case "list":
		return validateListArgs(args)
	case "add":
		return validateAddArgs(args)
	case "delete":
		return validateDeleteArgs(args)
	default:
		return fmt.Errorf("unsupported nft action %q", args[0])
	}
}

func validateListArgs(args []string) error {
	if len(args) == 4 && args[1] == "table" && args[2] == nftFamily && args[3] == nftTable {
		return nil
	}
	if len(args) == 5 && args[1] == "chain" && args[2] == nftFamily && args[3] == nftTable && validChain(args[4]) {
		return nil
	}
	return errors.New("unsupported nft list arguments")
}

func validateAddArgs(args []string) error {
	if len(args) >= 2 {
		switch args[1] {
		case "table":
			if len(args) == 4 && args[2] == nftFamily && args[3] == nftTable {
				return nil
			}
		case "chain":
			return validateAddChainArgs(args)
		case "rule":
			return validateAddRuleArgs(args)
		}
	}
	return errors.New("unsupported nft add arguments")
}

func validateAddChainArgs(args []string) error {
	if len(args) != 17 {
		return errors.New("unsupported nft add chain arguments")
	}
	chain := args[4]
	if args[2] != nftFamily || args[3] != nftTable || !validChain(chain) {
		return errors.New("invalid nft chain target")
	}
	want := []string{"{", "type", "filter", "hook", chain, "priority", "0", ";", "policy", "accept", ";", "}"}
	for i, token := range want {
		if args[i+5] != token {
			return errors.New("unsupported nft chain definition")
		}
	}
	return nil
}

func validateAddRuleArgs(args []string) error {
	if len(args) < 8 || args[2] != nftFamily || args[3] != nftTable || !validChain(args[4]) {
		return errors.New("invalid nft rule target")
	}
	for i := 5; i < len(args); {
		next, done, err := validateRuleToken(args, i)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		i = next
	}
	return errors.New("nft rule missing comment/action")
}

func validateRuleToken(args []string, i int) (next int, done bool, err error) {
	switch args[i] {
	case "ip":
		return validateCIDRToken(args, i)
	case "tcp", "udp":
		return validateTransportToken(args, i)
	case "icmp":
		return i + 1, false, nil
	case "comment":
		if i+2 >= len(args) || !nftCommentPattern.MatchString(args[i+1]) || !validAction(args[i+2]) {
			return 0, false, errors.New("invalid nft comment/action")
		}
		if i+3 != len(args) {
			return 0, false, errors.New("trailing nft rule arguments")
		}
		return len(args), true, nil
	default:
		return 0, false, fmt.Errorf("unsupported nft rule token %q", args[i])
	}
}

func validateCIDRToken(args []string, i int) (next int, done bool, err error) {
	if i+2 >= len(args) || (args[i+1] != "saddr" && args[i+1] != "daddr") {
		return 0, false, errors.New("invalid nft cidr selector")
	}
	if _, _, err := net.ParseCIDR(args[i+2]); err != nil {
		return 0, false, fmt.Errorf("invalid nft cidr: %w", err)
	}
	return i + 3, false, nil
}

func validateTransportToken(args []string, i int) (next int, done bool, err error) {
	if i+2 < len(args) && args[i+1] == "dport" {
		if err := validatePort(args[i+2]); err != nil {
			return 0, false, err
		}
		return i + 3, false, nil
	}
	return i + 1, false, nil
}

func validateDeleteArgs(args []string) error {
	if len(args) != 7 || args[1] != "rule" || args[2] != nftFamily || args[3] != nftTable || !validChain(args[4]) || args[5] != "handle" {
		return errors.New("unsupported nft delete arguments")
	}
	handle, err := strconv.ParseInt(args[6], 10, 64)
	if err != nil || handle <= 0 {
		return errors.New("invalid nft handle")
	}
	return nil
}

func validatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return errors.New("invalid nft destination port")
	}
	return nil
}

func validChain(chain string) bool {
	return chain == "input" || chain == "output" || chain == "forward"
}

func validAction(action string) bool {
	return action == "accept" || action == "drop" || action == "reject"
}

func (h Helper) nftPath() string {
	if h.NFTPath != "" {
		return h.NFTPath
	}
	return DefaultNFTPath
}

func (h Helper) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if h.RunCommand != nil {
		return h.RunCommand(ctx, name, args...)
	}
	// #nosec G204 -- helper validates the fixed nft binary and every argv token before execution.
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C"}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil && h.Stderr != nil && stderr.Len() > 0 {
		_, _ = h.Stderr.Write(stderr.Bytes())
	}
	return out, err
}
