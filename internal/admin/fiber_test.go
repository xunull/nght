package admin

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newAdminFiberApp() (*fiber.App, *RouteTable) {
	tbl := NewRouteTable()
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	RegisterFiberRoutesWithTable(app, "", tbl)
	return app, tbl
}

func doFiberReq(t *testing.T, app *fiber.App, method, path, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test err = %v", err)
	}
	return resp
}

func TestRegister_Happy(t *testing.T) {
	app, tbl := newAdminFiberApp()
	resp := doFiberReq(t, app, "POST", "/admin/routes",
		`{"path":"/api/timeout","status_code":504,"latency_ms":3000}`)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if tbl.Count() != 1 {
		t.Fatalf("table count = %d, want 1", tbl.Count())
	}
	cfg, ok := tbl.Get("/api/timeout")
	if !ok || cfg.StatusCode != 504 || cfg.LatencyMs != 3000 {
		t.Fatalf("Get = %+v, ok=%v, want 504/3000", cfg, ok)
	}
}

func TestRegister_InvalidJSON_400(t *testing.T) {
	app, _ := newAdminFiberApp()
	resp := doFiberReq(t, app, "POST", "/admin/routes", `{not json`)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRegister_ReservedPath_400(t *testing.T) {
	app, tbl := newAdminFiberApp()
	tbl.MarkReserved("/echo")
	resp := doFiberReq(t, app, "POST", "/admin/routes",
		`{"path":"/echo","status_code":200}`)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRegister_InvalidStatus_400(t *testing.T) {
	app, _ := newAdminFiberApp()
	resp := doFiberReq(t, app, "POST", "/admin/routes",
		`{"path":"/x","status_code":99}`)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestList_Empty(t *testing.T) {
	app, _ := newAdminFiberApp()
	resp := doFiberReq(t, app, "GET", "/admin/routes", "")
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "[]" {
		t.Fatalf("empty list body = %q, want []", string(body))
	}
}

func TestList_Multiple(t *testing.T) {
	app, tbl := newAdminFiberApp()
	_ = tbl.Register(RouteConfig{Path: "/a", StatusCode: 200})
	_ = tbl.Register(RouteConfig{Path: "/b", StatusCode: 503})
	resp := doFiberReq(t, app, "GET", "/admin/routes", "")
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"/a"`) || !strings.Contains(string(body), `"/b"`) {
		t.Fatalf("body missing entries: %s", string(body))
	}
}

func TestUnregister_Existing_204(t *testing.T) {
	app, tbl := newAdminFiberApp()
	_ = tbl.Register(RouteConfig{Path: "/api/x", StatusCode: 200})
	resp := doFiberReq(t, app, "DELETE", "/admin/routes/api/x", "")
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if tbl.Count() != 0 {
		t.Fatalf("count = %d, want 0", tbl.Count())
	}
}

func TestUnregister_NestedPath_204(t *testing.T) {
	app, tbl := newAdminFiberApp()
	_ = tbl.Register(RouteConfig{Path: "/api/v1/timeout", StatusCode: 504})
	resp := doFiberReq(t, app, "DELETE", "/admin/routes/api/v1/timeout", "")
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if tbl.Count() != 0 {
		t.Fatalf("count = %d, want 0", tbl.Count())
	}
}

func TestUnregister_NonExisting_204_Idempotent(t *testing.T) {
	app, _ := newAdminFiberApp()
	resp := doFiberReq(t, app, "DELETE", "/admin/routes/never-existed", "")
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("idempotent: status = %d, want 204", resp.StatusCode)
	}
}

func TestUnregister_EmptyPath_400(t *testing.T) {
	app, _ := newAdminFiberApp()
	resp := doFiberReq(t, app, "DELETE", "/admin/routes/", "")
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRegisterThenListThenUnregister_E2E(t *testing.T) {
	app, _ := newAdminFiberApp()
	r1 := doFiberReq(t, app, "POST", "/admin/routes",
		`{"path":"/api/timeout","status_code":504,"latency_ms":3000}`)
	if r1.StatusCode != 201 {
		t.Fatalf("register: status = %d", r1.StatusCode)
	}
	r2 := doFiberReq(t, app, "GET", "/admin/routes", "")
	if r2.StatusCode != 200 {
		t.Fatalf("list: status = %d", r2.StatusCode)
	}
	body, _ := io.ReadAll(r2.Body)
	if !strings.Contains(string(body), `"/api/timeout"`) {
		t.Fatalf("list missing /api/timeout: %s", body)
	}
	r3 := doFiberReq(t, app, "DELETE", "/admin/routes/api/timeout", "")
	if r3.StatusCode != 204 {
		t.Fatalf("unregister: status = %d", r3.StatusCode)
	}
	r4 := doFiberReq(t, app, "GET", "/admin/routes", "")
	body4, _ := io.ReadAll(r4.Body)
	if string(body4) != "[]" {
		t.Fatalf("after unregister, list = %s, want []", body4)
	}
}

func TestRegister_AuthEnforced_WhenSecretSet(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	tbl := NewRouteTable()
	RegisterFiberRoutesWithTable(app, "secret123", tbl)

	r1 := doFiberReq(t, app, "POST", "/admin/routes",
		`{"path":"/api/timeout","status_code":504}`)
	if r1.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("no auth: status = %d, want 401", r1.StatusCode)
	}
	if tbl.Count() != 0 {
		t.Fatalf("table changed on 401")
	}

	req, _ := http.NewRequest("POST", "/admin/routes",
		strings.NewReader(`{"path":"/api/timeout","status_code":504}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "secret123")
	r2, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test err = %v", err)
	}
	if r2.StatusCode != fiber.StatusCreated {
		t.Fatalf("with auth: status = %d, want 201", r2.StatusCode)
	}
}
