// Package admin provides the dynamic-route API for nght: at runtime, an
// SRE can register, list, and unregister HTTP routes via POST/GET/DELETE
// on /admin/routes without restarting the binary or rebuilding the image.
//
// Lookups use a sync.RWMutex because the SRE use case is read-heavy
// (one POST register, N GET dispatches on the hot path).
package admin

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// RouteConfig describes a dynamically-registered route.
type RouteConfig struct {
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int    `json:"latency_ms"`
}

var (
	ErrEmptyPath         = errors.New("admin: path is empty")
	ErrInvalidStatusCode = errors.New("admin: status_code must be in [100, 599]")
	ErrNegativeLatency   = errors.New("admin: latency_ms must be >= 0")
	ErrReservedPath      = errors.New("admin: path is reserved by a hardcoded endpoint")
)

// RouteTable is a concurrent-safe in-memory map of path -> RouteConfig.
//
// Reserved paths are the set of hardcoded endpoints declared in
// fiber_server/routes.go; the route setup calls MarkReserved for each of
// them at startup so the dynamic API cannot shadow nght's built-in
// behavior. This is the single source of truth (see plan-eng-review C1).
type RouteTable struct {
	mu       sync.RWMutex
	routes   map[string]RouteConfig
	reserved map[string]bool
}

func NewRouteTable() *RouteTable {
	return &RouteTable{
		routes:   make(map[string]RouteConfig),
		reserved: make(map[string]bool),
	}
}

// MarkReserved marks a path as protected from dynamic registration.
//
// Convention: a path WITHOUT a trailing slash is an exact-match reservation
// (e.g., "/echo" blocks "/echo" only). A path WITH a trailing slash is a
// prefix reservation (e.g., "/health/" blocks "/health/true", "/health/false",
// "/health/random/30"). This lets fiber routes.go call one MarkReserved per
// hardcoded endpoint.
//
// Must be called at startup before any concurrent Register. Behavior under
// concurrent MarkReserved + Register is undefined.
func (t *RouteTable) MarkReserved(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reserved[path] = true
}

func (t *RouteTable) IsReserved(path string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isReservedNoLock(path)
}

// isReservedNoLock assumes t.mu is held by the caller. Used by Register
// to avoid re-acquiring the lock (Go's sync.RWMutex deadlocks if a writer
// calls RLock from the same goroutine).
func (t *RouteTable) isReservedNoLock(path string) bool {
	if t.reserved[path] {
		return true
	}
	for p := path; ; {
		idx := strings.LastIndex(p, "/")
		if idx < 0 {
			break
		}
		p = p[:idx]
		if t.reserved[p] || t.reserved[p+"/"] {
			return true
		}
		if p == "" {
			break
		}
	}
	return false
}

func (t *RouteTable) Register(cfg RouteConfig) error {
	if cfg.Path == "" {
		return ErrEmptyPath
	}
	if cfg.StatusCode < 100 || cfg.StatusCode > 599 {
		return ErrInvalidStatusCode
	}
	if cfg.LatencyMs < 0 {
		return ErrNegativeLatency
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isReservedNoLock(cfg.Path) {
		return ErrReservedPath
	}
	t.routes[cfg.Path] = cfg
	return nil
}

// Unregister is idempotent — safe to call repeatedly on the same path.
func (t *RouteTable) Unregister(path string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.routes[path]; !ok {
		return false
	}
	delete(t.routes, path)
	return true
}

func (t *RouteTable) Get(path string) (RouteConfig, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	cfg, ok := t.routes[path]
	return cfg, ok
}

// List returns a snapshot. Order is unspecified (map iteration).
func (t *RouteTable) List() []RouteConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]RouteConfig, 0, len(t.routes))
	for _, cfg := range t.routes {
		out = append(out, cfg)
	}
	return out
}

func (t *RouteTable) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.routes)
}

func (t *RouteTable) String() string {
	return fmt.Sprintf("RouteTable{routes=%d, reserved=%d}", t.Count(), len(t.reserved))
}
