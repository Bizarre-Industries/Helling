package server

import (
	"net/http"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "uses remote addr ip when present",
			remoteAddr: "203.0.113.10:443",
			want:       "203.0.113.10",
		},
		{
			name:       "uses first forwarded ip for unix peer",
			remoteAddr: "@",
			headers: map[string]string{
				"X-Forwarded-For": "198.51.100.20, 10.0.0.2",
			},
			want: "198.51.100.20",
		},
		{
			name:       "falls back to x real ip for unix peer",
			remoteAddr: "@",
			headers: map[string]string{
				"X-Real-IP": "2001:db8::42",
			},
			want: "2001:db8::42",
		},
		{
			name:       "returns unknown for empty remote addr",
			remoteAddr: "",
			want:       "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: make(http.Header), RemoteAddr: tt.remoteAddr}
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			got := clientIP(req)
			if got != tt.want {
				t.Fatalf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
