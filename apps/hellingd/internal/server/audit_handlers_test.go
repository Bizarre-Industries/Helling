package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuditJournalArgsFiltersAuditMarker(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?actor=alice&outcome=denied", http.NoBody)
	args := auditJournalArgs(req, 25)
	for _, want := range []string{"SYSLOG_IDENTIFIER=hellingd", "HELLING_AUDIT=1", "HELLING_ACTOR=alice", "HELLING_OUTCOME=denied"} {
		if !containsString(args, want) {
			t.Fatalf("auditJournalArgs missing %q in %#v", want, args)
		}
	}
}

func TestParseAuditJSONLinesSkipsNonAuditLogs(t *testing.T) {
	t.Parallel()
	body := strings.Join([]string{
		`{"SYSLOG_IDENTIFIER":"hellingd","MESSAGE":"ordinary log line"}`,
		`{"SYSLOG_IDENTIFIER":"hellingd","HELLING_ACTION":"user.create","MESSAGE":"missing outcome"}`,
		`{"SYSLOG_IDENTIFIER":"hellingd","HELLING_AUDIT":"1","HELLING_ACTION":"schedule.create","HELLING_OUTCOME":"success","MESSAGE":"schedule created","__REALTIME_TIMESTAMP":"1710000000000000"}`,
	}, "\n")
	rows, err := parseAuditJSONLines([]byte(body))
	if err != nil {
		t.Fatalf("parseAuditJSONLines: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("parseAuditJSONLines returned %d rows, want 1: %#v", len(rows), rows)
	}
	if rows[0].Action != "schedule.create" || rows[0].Outcome != outcomeSuccess {
		t.Fatalf("parsed audit row = %#v", rows[0])
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
