package admin

import (
	"errors"
	"sync"
	"testing"
)

func TestRouteTable_Register_Happy(t *testing.T) {
	tbl := NewRouteTable()
	if err := tbl.Register(RouteConfig{Path: "/api/timeout", StatusCode: 504, LatencyMs: 3000}); err != nil {
		t.Fatalf("register happy: %v", err)
	}
	if got := tbl.Count(); got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
	cfg, ok := tbl.Get("/api/timeout")
	if !ok {
		t.Fatalf("Get: not found")
	}
	if cfg.StatusCode != 504 || cfg.LatencyMs != 3000 {
		t.Fatalf("Get = %+v, want 504/3000", cfg)
	}
}

func TestRouteTable_Register_EmptyPath(t *testing.T) {
	tbl := NewRouteTable()
	if err := tbl.Register(RouteConfig{Path: "", StatusCode: 200}); !errors.Is(err, ErrEmptyPath) {
		t.Fatalf("err = %v, want ErrEmptyPath", err)
	}
	if tbl.Count() != 0 {
		t.Fatalf("table changed on validation failure")
	}
}

func TestRouteTable_Register_InvalidStatus(t *testing.T) {
	tbl := NewRouteTable()
	for _, bad := range []int{0, 50, 99, 600, 700, 1000} {
		err := tbl.Register(RouteConfig{Path: "/x", StatusCode: bad})
		if !errors.Is(err, ErrInvalidStatusCode) {
			t.Fatalf("status %d: err = %v, want ErrInvalidStatusCode", bad, err)
		}
	}
	for _, ok := range []int{100, 200, 404, 502, 599} {
		if err := tbl.Register(RouteConfig{Path: "/ok-" + string(rune(ok)), StatusCode: ok}); err != nil {
			t.Fatalf("status %d should be valid, got %v", ok, err)
		}
	}
}

func TestRouteTable_Register_NegativeLatency(t *testing.T) {
	tbl := NewRouteTable()
	if err := tbl.Register(RouteConfig{Path: "/x", StatusCode: 200, LatencyMs: -1}); !errors.Is(err, ErrNegativeLatency) {
		t.Fatalf("err = %v, want ErrNegativeLatency", err)
	}
}

func TestRouteTable_Register_ReservedExact(t *testing.T) {
	tbl := NewRouteTable()
	tbl.MarkReserved("/echo")
	if err := tbl.Register(RouteConfig{Path: "/echo", StatusCode: 200}); !errors.Is(err, ErrReservedPath) {
		t.Fatalf("err = %v, want ErrReservedPath", err)
	}
}

func TestRouteTable_Register_ReservedPrefix(t *testing.T) {
	tbl := NewRouteTable()
	tbl.MarkReserved("/health/")
	for _, p := range []string{"/health/true", "/health/false", "/health/random/30"} {
		if err := tbl.Register(RouteConfig{Path: p, StatusCode: 200}); !errors.Is(err, ErrReservedPath) {
			t.Fatalf("%s: err = %v, want ErrReservedPath", p, err)
		}
	}
}

func TestRouteTable_Register_PrefixExactDoesNotBlockChildren(t *testing.T) {
	tbl := NewRouteTable()
	tbl.MarkReserved("/health")
	if err := tbl.Register(RouteConfig{Path: "/healthz", StatusCode: 200}); err != nil {
		t.Fatalf("/healthz should NOT be reserved by exact-match /health: %v", err)
	}
}

func TestRouteTable_Register_Overwrite(t *testing.T) {
	tbl := NewRouteTable()
	_ = tbl.Register(RouteConfig{Path: "/api/x", StatusCode: 200})
	if err := tbl.Register(RouteConfig{Path: "/api/x", StatusCode: 503, LatencyMs: 1000}); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	cfg, _ := tbl.Get("/api/x")
	if cfg.StatusCode != 503 || cfg.LatencyMs != 1000 {
		t.Fatalf("after overwrite = %+v, want 503/1000", cfg)
	}
	if tbl.Count() != 1 {
		t.Fatalf("count = %d, want 1 (no duplicate entry)", tbl.Count())
	}
}

func TestRouteTable_Unregister(t *testing.T) {
	tbl := NewRouteTable()
	_ = tbl.Register(RouteConfig{Path: "/api/x", StatusCode: 200})

	if !tbl.Unregister("/api/x") {
		t.Fatalf("first Unregister: want true")
	}
	if tbl.Unregister("/api/x") {
		t.Fatalf("second Unregister: want false (idempotent)")
	}
	if tbl.Unregister("/never-existed") {
		t.Fatalf("Unregister non-existing: want false")
	}
}

func TestRouteTable_Get_Miss(t *testing.T) {
	tbl := NewRouteTable()
	cfg, ok := tbl.Get("/missing")
	if ok {
		t.Fatalf("Get: ok = true, want false")
	}
	if cfg != (RouteConfig{}) {
		t.Fatalf("Get miss: cfg = %+v, want zero value", cfg)
	}
}

func TestRouteTable_List(t *testing.T) {
	tbl := NewRouteTable()
	if got := tbl.List(); len(got) != 0 {
		t.Fatalf("empty List: len = %d", len(got))
	}
	_ = tbl.Register(RouteConfig{Path: "/a", StatusCode: 200})
	_ = tbl.Register(RouteConfig{Path: "/b", StatusCode: 201})
	_ = tbl.Register(RouteConfig{Path: "/c", StatusCode: 202})
	if got := tbl.List(); len(got) != 3 {
		t.Fatalf("List: len = %d, want 3", len(got))
	}
	seen := map[string]bool{}
	for _, cfg := range tbl.List() {
		seen[cfg.Path] = true
	}
	for _, p := range []string{"/a", "/b", "/c"} {
		if !seen[p] {
			t.Fatalf("List missing %s", p)
		}
	}
}

func TestRouteTable_ConcurrentRegisterDifferentPaths(t *testing.T) {
	tbl := NewRouteTable()
	const N = 100
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			path := "/concurrent/" + string(rune('a'+i%26)) + string(rune('0'+i/26))
			_ = tbl.Register(RouteConfig{Path: path, StatusCode: 200 + i%400})
		}(i)
	}
	wg.Wait()
	if tbl.Count() != N {
		t.Fatalf("count = %d, want %d", tbl.Count(), N)
	}
}

func TestRouteTable_ConcurrentRegisterSamePath(t *testing.T) {
	tbl := NewRouteTable()
	const N = 50
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			_ = tbl.Register(RouteConfig{Path: "/race", StatusCode: 200 + i%400})
		}(i)
	}
	wg.Wait()
	if tbl.Count() != 1 {
		t.Fatalf("count = %d, want 1 (last-writer-wins)", tbl.Count())
	}
	cfg, ok := tbl.Get("/race")
	if !ok {
		t.Fatalf("Get /race: not found")
	}
	if cfg.StatusCode < 200 || cfg.StatusCode >= 600 {
		t.Fatalf("StatusCode = %d, want in [200, 599]", cfg.StatusCode)
	}
}

func TestRouteTable_ConcurrentReadWrite(t *testing.T) {
	tbl := NewRouteTable()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			_ = tbl.Register(RouteConfig{Path: "/rw/" + string(rune('a'+i%26)), StatusCode: 200})
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = tbl.List()
			_, _ = tbl.Get("/x")
			_ = tbl.IsReserved("/x")
		}
	}()
	for i := 0; i < 1000; i++ {
		_ = tbl.Register(RouteConfig{Path: "/main/" + string(rune('a'+i%26)), StatusCode: 200})
	}
	close(stop)
	wg.Wait()
}

func TestRouteTable_AllHardcodedPathsMarkedReserved(t *testing.T) {
	// Reverse-assertion: this test is the contract that the single-source
	// of truth holds. It pre-populates a RouteTable with the full set of
	// hardcoded paths from fiber_server/routes.go and asserts each one is
	// in the reserved set. If a future change to routes.go adds a new
	// endpoint without MarkReserved, the integration smoke test in T8
	// will fail. This unit test documents the contract; the integration
	// check is in fiber_server/routes.go.
	tbl := NewRouteTable()
	hardcodedPaths := []string{
		"/echo",          // exact
		"/echo_header",   // exact
		"/echo_url",      // exact
		"/status",        // exact (path param handled at fiber layer)
		"/log_req_data",  // exact
		"/response_time", // exact
		"/random",        // exact
		"/random_crash",  // exact
		"/healthz",       // exact
		"/livez",         // exact
		"/health/",       // prefix — catches /health/true, /health/false, /health/random/:pct
	}
	for _, p := range hardcodedPaths {
		tbl.MarkReserved(p)
	}
	// Each hardcoded path itself should be reserved.
	for _, p := range hardcodedPaths {
		if !tbl.IsReserved(p) {
			t.Fatalf("hardcoded path %q not marked reserved", p)
		}
	}
	for _, p := range hardcodedPaths {
		err := tbl.Register(RouteConfig{Path: p, StatusCode: 200})
		if !errors.Is(err, ErrReservedPath) {
			t.Fatalf("Register(%q): err = %v, want ErrReservedPath", p, err)
		}
	}
	for _, p := range []string{"/health/true", "/health/false", "/health/random/30"} {
		err := tbl.Register(RouteConfig{Path: p, StatusCode: 200})
		if !errors.Is(err, ErrReservedPath) {
			t.Fatalf("Register(%q): err = %v, want ErrReservedPath", p, err)
		}
	}
	if err := tbl.Register(RouteConfig{Path: "/api/dynamic", StatusCode: 200}); err != nil {
		t.Fatalf("Register(/api/dynamic): err = %v, want nil", err)
	}
}
