package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestClientE2E_HappyPath spins up a local httptest server that always
// returns 200, runs runClientLoad end-to-end (no cobra, no stdio), and
// verifies the report shape and a few invariants.
func TestClientE2E_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	host, port := splitSrvURL(t, srv.URL)
	cfg := clientConfig{
		Host:        host,
		Port:        port,
		Concurrency: 5,
		Duration:    500 * time.Millisecond,
		Total:       0,
		RPS:         0,
		Timeout:     1 * time.Second,
		Path:        "/anything",
	}
	rpt, err := runClientLoad(cfg)
	if err != nil {
		t.Fatalf("runClientLoad: %v", err)
	}
	if rpt.Summary.TotalRequests == 0 {
		t.Fatalf("expected non-zero total_requests, got 0 (server unreachable?)")
	}
	if got := rpt.StatusHistogram["200"]; got == 0 {
		t.Errorf("expected at least one 200 response, got histogram %+v", rpt.StatusHistogram)
	}
	if rpt.Summary.Errors != 0 {
		t.Errorf("expected 0 errors against 200-only server, got %d", rpt.Summary.Errors)
	}
	if rpt.LatencyMs.Samples == 0 {
		t.Errorf("expected non-zero latency samples, got 0")
	}
	if rpt.Config["host"] != host || rpt.Config["port"] != port {
		t.Errorf("config echo wrong: %+v", rpt.Config)
	}
}

// TestClientE2E_Unreachable verifies that a closed port surfaces in the
// network_error bucket (not as a panic or hang).
func TestClientE2E_Unreachable(t *testing.T) {
	cfg := clientConfig{
		Host: "127.0.0.1", Port: 1, // privileged, almost certainly closed
		Concurrency: 2, Total: 5, RPS: 0,
		Timeout: 200 * time.Millisecond, Path: "/",
	}
	rpt, err := runClientLoad(cfg)
	if err != nil {
		t.Fatalf("runClientLoad: %v", err)
	}
	if rpt.Summary.TotalRequests != 5 {
		t.Errorf("expected exactly 5 dispatched, got %d", rpt.Summary.TotalRequests)
	}
	if got := rpt.StatusHistogram["network_error"]; got != 5 {
		t.Errorf("expected 5 network_error, got %d (full: %+v)", got, rpt.StatusHistogram)
	}
	if rpt.Summary.Errors != 5 {
		t.Errorf("expected 5 errors, got %d", rpt.Summary.Errors)
	}
}

// TestClientE2E_JSONMarshals verifies the report's JSON shape matches the
// design doc sample (config / summary / latency_ms / status_histogram
// fields present and exported).
func TestClientE2E_JSONMarshals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	host, port := splitSrvURL(t, srv.URL)
	cfg := clientConfig{Host: host, Port: port, Concurrency: 2, Total: 3, Timeout: time.Second, Path: "/"}
	rpt, err := runClientLoad(cfg)
	if err != nil {
		t.Fatalf("runClientLoad: %v", err)
	}
	b, err := json.MarshalIndent(rpt, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"config"`, `"summary"`, `"latency_ms"`, `"status_histogram"`} {
		if !strings.Contains(string(b), key) {
			t.Errorf("json missing %q: %s", key, string(b))
		}
	}
}

// splitSrvURL extracts (host, port) from an httptest server URL like
// "http://127.0.0.1:54321". t.Fatal on unexpected shape.
func splitSrvURL(t *testing.T, srvURL string) (string, int) {
	t.Helper()
	host := strings.TrimPrefix(srvURL, "http://")
	idx := strings.LastIndex(host, ":")
	if idx < 0 {
		t.Fatalf("unexpected srv.URL %q (no port colon)", srvURL)
	}
	portStr := host[idx+1:]
	port := 0
	for _, r := range portStr {
		if r < '0' || r > '9' {
			t.Fatalf("non-digit in port %q", portStr)
		}
		port = port*10 + int(r-'0')
	}
	return host[:idx], port
}
