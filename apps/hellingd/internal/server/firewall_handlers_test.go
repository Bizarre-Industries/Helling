package server

import (
	"strings"
	"testing"
)

func TestParseNFTHandleByComment(t *testing.T) {
	t.Parallel()
	handle, err := parseNFTHandleByComment(strings.NewReader(`
table inet helling {
  chain input {
    type filter hook input priority filter; policy accept;
    ip saddr 203.0.113.0/24 tcp dport 443 comment "helling:018f3a40-0000-7000-8000-000000000000" accept # handle 42
  }
}
`), "helling:018f3a40-0000-7000-8000-000000000000")
	if err != nil {
		t.Fatalf("parseNFTHandleByComment: %v", err)
	}
	if handle != 42 {
		t.Fatalf("handle: got %d want 42", handle)
	}
}

func TestParseNFTHandleByCommentRequiresExactComment(t *testing.T) {
	t.Parallel()
	_, err := parseNFTHandleByComment(strings.NewReader(`
ip saddr 203.0.113.0/24 comment "helling:018f3a40-0000-7000-8000-000000000000-extra" accept # handle 42
`), "helling:018f3a40-0000-7000-8000-000000000000")
	if err == nil {
		t.Fatal("parseNFTHandleByComment matched a partial comment")
	}
}

func TestValidateFirewallRuleRejectsPortWithoutTCPOrUDP(t *testing.T) {
	t.Parallel()
	port := 22
	for _, protocol := range []string{"any", "icmp"} {
		err := validateFirewallRule(&createFirewallRuleRequest{
			Direction:       "input",
			Action:          "drop",
			Protocol:        protocol,
			DestinationPort: &port,
		})
		if err == nil {
			t.Fatalf("validateFirewallRule accepted destination_port with protocol %q", protocol)
		}
	}
}
