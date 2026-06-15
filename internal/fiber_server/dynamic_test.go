package fiber_server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xunull/nght/internal/admin"
)

func TestDynamicRouteDispatch_Hit(t *testing.T) {
	if err := admin.Register(admin.RouteConfig{Path: "/api/timeout", StatusCode: 504, LatencyMs: 0}); err != nil {
		t.Fatalf("register: %v", err)
	}
	t.Cleanup(func() { admin.Unregister("/api/timeout") })

	app := newTestApp()
	req := httptest.NewRequest("GET", "/api/timeout", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 504 {
		t.Fatalf("status = %d, want 504", resp.StatusCode)
	}
}

func TestDynamicRouteDispatch_Latency(t *testing.T) {
	if err := admin.Register(admin.RouteConfig{Path: "/api/slow", StatusCode: 200, LatencyMs: 200}); err != nil {
		t.Fatalf("register: %v", err)
	}
	t.Cleanup(func() { admin.Unregister("/api/slow") })

	app := newTestApp()
	req := httptest.NewRequest("GET", "/api/slow", nil)
	start := time.Now()
	resp, _ := app.Test(req, -1)
	elapsed := time.Since(start)
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if elapsed < 150*time.Millisecond {
		t.Fatalf("elapsed = %v, want >= 150ms (latency not applied)", elapsed)
	}
}

func TestDynamicRouteDispatch_MissFallsToWildcard(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest("GET", "/totally-dynamic-miss", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (wildcard fallback)", resp.StatusCode)
	}
}

func TestDynamicRouteDispatch_DoesNotShadowHardcoded(t *testing.T) {
	err := admin.Register(admin.RouteConfig{Path: "/echo", StatusCode: 200})
	if err == nil {
		t.Fatal("register /echo: should be rejected (reserved)")
	}
}

func TestDynamicRouteDispatch_DoesNotShadowHealthPrefix(t *testing.T) {
	err := admin.Register(admin.RouteConfig{Path: "/health/anything", StatusCode: 200})
	if err == nil {
		t.Fatal("register /health/anything: should be rejected (prefix reserved)")
	}
}

func TestWildcard_AfterT5_NoLongerConflictsWithDynamic(t *testing.T) {
	// Original TestWildcardFallback covers the basic case; this is a
	// regression test that the dispatch middleware doesn't break the
	// hardcoded wildcard for unknown paths.
	app := newTestApp()
	req := httptest.NewRequest("GET", "/some/never/seen/path", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
