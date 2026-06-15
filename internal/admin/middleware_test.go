package admin

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newAdminTestApp(secret string) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(RequireAdminToken(secret))
	app.Get("/probe", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func doAdminReq(t *testing.T, app *fiber.App, method, path, headerVal string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, path, nil)
	if headerVal != "" {
		req.Header.Set("X-Admin-Token", headerVal)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test err = %v", err)
	}
	return resp
}

func readAdminBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestRequireAdminToken_SecretEmpty_AllowsAll(t *testing.T) {
	app := newAdminTestApp("")
	if r := doAdminReq(t, app, "GET", "/probe", ""); r.StatusCode != 200 {
		t.Fatalf("no header, empty secret: status = %d, want 200", r.StatusCode)
	}
	if r := doAdminReq(t, app, "GET", "/probe", "anything"); r.StatusCode != 200 {
		t.Fatalf("any header, empty secret: status = %d, want 200", r.StatusCode)
	}
}

func TestRequireAdminToken_SecretSet_HeaderAbsent_401(t *testing.T) {
	app := newAdminTestApp("secret123")
	resp := doAdminReq(t, app, "GET", "/probe", "")
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestRequireAdminToken_SecretSet_HeaderMatch_Allow(t *testing.T) {
	app := newAdminTestApp("secret123")
	resp := doAdminReq(t, app, "GET", "/probe", "secret123")
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body := readAdminBody(t, resp)
	if body != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

func TestRequireAdminToken_SecretSet_HeaderMismatch_401(t *testing.T) {
	app := newAdminTestApp("secret123")
	for _, bad := range []string{"wrong", "Secret123", "secret12", "secret1234"} {
		if r := doAdminReq(t, app, "GET", "/probe", bad); r.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("header %q: status = %d, want 401", bad, r.StatusCode)
		}
	}
}

func TestRequireAdminToken_SecretSet_HeaderDifferentLength_401(t *testing.T) {
	app := newAdminTestApp("secret")
	for _, bad := range []string{"s", "secre", "secretsecret"} {
		if r := doAdminReq(t, app, "GET", "/probe", bad); r.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("header len %d: status = %d, want 401", len(bad), r.StatusCode)
		}
	}
}

func TestRequireAdminToken_EmptyHeader_WithSecretSet_401(t *testing.T) {
	app := newAdminTestApp("nonempty")
	if r := doAdminReq(t, app, "GET", "/probe", ""); r.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("empty header, set secret: status = %d, want 401", r.StatusCode)
	}
}

func TestRequireAdminToken_ResponseBodyIsJSON(t *testing.T) {
	app := newAdminTestApp("secret")
	resp := doAdminReq(t, app, "GET", "/probe", "")
	body := readAdminBody(t, resp)
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("body = %q, want JSON with error field", body)
	}
}
