package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvalidRequestsNeverReachDatabase(t *testing.T) {
	handler := NewHandler(nil)
	tests := []struct {
		name string
		method string
		target string
		body string
		handle http.HandlerFunc
	}{
		{"invalid file depth", http.MethodGet, "/api/files?depth=0", "", handler.GetFiles},
		{"invalid file depth string", http.MethodGet, "/api/files?depth=xyz", "", handler.GetFiles},
		{"missing log path", http.MethodGet, "/api/logs", "", handler.GetLogs},
		{"invalid log time", http.MethodGet, "/api/logs?file=app.log&before=not-time", "", handler.GetLogs},
		{"invalid search method", http.MethodGet, "/api/logs/search", "", handler.SearchLogs},
		{"empty search", http.MethodPost, "/api/logs/search", "{\"query\":\"\"}", handler.SearchLogs},
		{"malformed JSON", http.MethodPost, "/api/logs/search", "{", handler.SearchLogs},
		{"invalid network time", http.MethodGet, "/api/network/metrics?start=not-time", "", handler.GetNetworkMetrics},
		{"reversed network range", http.MethodGet, "/api/network/metrics?start=2026-10-09T12:00:00Z&end=2026-10-09T11:00:00Z", "", handler.GetNetworkMetrics},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			tc.handle(rec, req)
			if rec.Code < 400 || rec.Code >= 500 {
				t.Fatalf("expected client error, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
