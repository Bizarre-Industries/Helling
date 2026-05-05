package firewall

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestValidateNFTArgsAllowsHellingRuleShape(t *testing.T) {
	t.Parallel()
	args := []string{
		"add", "rule", "inet", "helling", "input",
		"ip", "saddr", "203.0.113.0/24",
		"tcp", "dport", "443",
		"comment", "helling:018f3a40-0000-7000-8000-000000000000",
		"accept",
	}
	if err := ValidateNFTArgs(args); err != nil {
		t.Fatalf("ValidateNFTArgs: %v", err)
	}
}

func TestHelperDeleteRequiresHellingOwnedComment(t *testing.T) {
	t.Parallel()
	deleteArgs := []string{"delete", "rule", "inet", "helling", "input", "handle", "42"}
	helper := Helper{
		RunCommand: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if slices.Equal(args, []string{"-a", "list", "chain", "inet", "helling", "input"}) {
				return []byte(`table inet helling {
 chain input {
  tcp dport 443 comment "helling:018f3a40-0000-7000-8000-000000000000" accept # handle 42
 }
}`), nil
			}
			if slices.Equal(args, deleteArgs) {
				return nil, nil
			}
			return nil, errors.New("unexpected command")
		},
	}
	if err := helper.Handle(context.Background(), deleteArgs); err != nil {
		t.Fatalf("Handle delete: %v", err)
	}

	helper.RunCommand = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if slices.Equal(args, []string{"-a", "list", "chain", "inet", "helling", "input"}) {
			return []byte(`ip saddr 203.0.113.0/24 accept # handle 42`), nil
		}
		return nil, errors.New("delete should not execute")
	}
	if err := helper.Handle(context.Background(), deleteArgs); err == nil {
		t.Fatal("Handle delete without Helling comment unexpectedly succeeded")
	}
}

func TestValidateNFTArgsRejectsArbitraryNFT(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"flush", "ruleset"},
		{"add", "rule", "inet", "filter", "input", "counter", "accept"},
		{"add", "rule", "inet", "helling", "input", "counter", "accept"},
		{"delete", "table", "inet", "helling"},
		{"add", "rule", "inet", "helling", "input", "comment", "not-helling", "accept"},
		{"add", "rule", "inet", "helling", "input", "tcp", "dport", "99999", "comment", "helling:018f3a40-0000-7000-8000-000000000000", "accept"},
	} {
		if err := ValidateNFTArgs(args); err == nil {
			t.Fatalf("ValidateNFTArgs(%v) unexpectedly succeeded", args)
		}
	}
}
