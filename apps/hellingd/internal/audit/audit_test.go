package audit

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateJournalFieldLimitsBytes(t *testing.T) {
	t.Parallel()
	got := truncateJournalField(strings.Repeat("a", 513), 512)
	if len(got) != 512 {
		t.Fatalf("truncateJournalField length = %d, want 512", len(got))
	}
}

func TestTruncateJournalFieldPreservesUTF8(t *testing.T) {
	t.Parallel()
	got := truncateJournalField(strings.Repeat("a", 511)+"é", 512)
	if len(got) > 512 {
		t.Fatalf("truncateJournalField length = %d, want <= 512", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("truncateJournalField returned invalid UTF-8: %q", got)
	}
}
