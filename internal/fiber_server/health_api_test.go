package fiber_server

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func resetHealth() {
	healthMu.Lock()
	healthStatus = true
	healthMu.Unlock()
}

func TestHealthResp_Toggle(t *testing.T) {
	defer resetHealth()
	app := newTestApp()

	// initial UP
	resp := doGET(t, app, "/health")
	if resp.StatusCode != 200 {
		t.Fatalf("initial: %d", resp.StatusCode)
	}

	// flip to false
	doGET(t, app, "/health/false")
	resp = doGET(t, app, "/health")
	if resp.StatusCode != fiber.StatusBadGateway {
		t.Fatalf("after false: %d", resp.StatusCode)
	}

	// flip back to true
	doGET(t, app, "/health/true")
	resp = doGET(t, app, "/health")
	if resp.StatusCode != 200 {
		t.Fatalf("after true: %d", resp.StatusCode)
	}
}

func TestHealthzAlias(t *testing.T) {
	defer resetHealth()
	app := newTestApp()
	resp := doGET(t, app, "/healthz")
	if resp.StatusCode != 200 {
		t.Fatalf("/healthz = %d", resp.StatusCode)
	}
}

// CRITICAL regression: HealthRandomResp must follow the percentage parameter.
// Pre-fix it ignored the param and always returned UP.
func TestHealthRandomResp_Distribution(t *testing.T) {
	app := newTestApp()
	const (
		samples = 200
		pct     = 30
	)

	ups := 0
	for i := 0; i < samples; i++ {
		req, _ := http.NewRequest("GET", "/health/random/30", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("sample %d: %v", i, err)
		}
		if resp.StatusCode == 200 {
			ups++
		}
	}

	// expected ~60 UPs for 30%, allow ±25 (regression safety)
	target := samples * pct / 100
	if ups < target-25 || ups > target+25 {
		t.Fatalf("HealthRandomResp distribution off: %d/%d UPs, expected ~%d ±25", ups, samples, target)
	}

	// also check that it's not just always-UP (the pre-fix bug)
	if ups == samples {
		t.Fatalf("HealthRandomResp returned UP for all %d samples — regression to old behavior", samples)
	}
}

func TestSetHealthEndpoints_ReturnOK(t *testing.T) {
	defer resetHealth()
	app := newTestApp()

	resp := doGET(t, app, "/health/true")
	if resp.StatusCode != 200 {
		t.Fatalf("/health/true = %d", resp.StatusCode)
	}

	resp = doGET(t, app, "/health/false")
	if resp.StatusCode != 200 {
		t.Fatalf("/health/false = %d", resp.StatusCode)
	}
}
