package fiber_server

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	app.Use(func(c *fiber.Ctx) error {
		SetCommonHeader(c)
		return c.Next()
	})
	SetupRoutes(app)
	return app
}

func doGET(t *testing.T, app *fiber.App, path string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", path, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test(%s) err = %v", path, err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestEchoTextResp(t *testing.T) {
	app := newTestApp()
	resp := doGET(t, app, "/echo/hello")
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "hello") {
		t.Fatalf("body = %q", body)
	}
}

func TestEchoUrlResp(t *testing.T) {
	app := newTestApp()
	resp := doGET(t, app, "/echo_url")
	body := readBody(t, resp)
	if !strings.Contains(body, "/echo_url") {
		t.Fatalf("body = %q", body)
	}
}

func TestStatusResp(t *testing.T) {
	app := newTestApp()
	for _, st := range []int{200, 404, 502, 418} {
		resp := doGET(t, app, "/status/"+strconv.Itoa(st))
		if resp.StatusCode != st {
			t.Fatalf("/status/%d returned %d", st, resp.StatusCode)
		}
	}
}

// CRITICAL regression: response_time sleep unit must be seconds, not nanoseconds.
// Pre-fix the handler returned in nanoseconds; this test must fail if regressed.
func TestResponseTime_UnitFix(t *testing.T) {
	app := newTestApp()
	req, _ := http.NewRequest("GET", "/response_time/1", nil)
	start := time.Now()
	resp, err := app.Test(req, 3000)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("app.Test err = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if elapsed < 800*time.Millisecond {
		t.Fatalf("response_time/1 returned in %v; sleep unit bug regressed", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("response_time/1 took too long: %v", elapsed)
	}
}

func TestRandomStatusResp_AllInSet(t *testing.T) {
	app := newTestApp()
	allowed := map[int]bool{200: true, 502: true, 503: true}
	for i := 0; i < 30; i++ {
		resp := doGET(t, app, "/random/200502503")
		if !allowed[resp.StatusCode] {
			t.Fatalf("/random/200502503 returned %d (not in {200,502,503})", resp.StatusCode)
		}
	}
}

func TestRandomStatusResp_BadInput(t *testing.T) {
	app := newTestApp()
	// "abc" causes SplitStatus to return error; handler returns the error,
	// which fiber's default error handler renders as 500.
	resp := doGET(t, app, "/random/abc")
	if resp.StatusCode == 200 {
		t.Fatalf("bad input should not return 200, got %d", resp.StatusCode)
	}
}

func TestStatusResp_JSONToggle(t *testing.T) {
	prev := responseJsonFlag
	SetResponseJson(true)
	defer SetResponseJson(prev)

	app := newTestApp()
	resp := doGET(t, app, "/status/200")
	body := readBody(t, resp)
	if !strings.Contains(body, `"hostname"`) {
		t.Fatalf("expected JSON containing hostname, got %q", body)
	}
}

func TestStatusResp_TextWhenJSONOff(t *testing.T) {
	prev := responseJsonFlag
	SetResponseJson(false)
	defer SetResponseJson(prev)

	app := newTestApp()
	resp := doGET(t, app, "/status/200")
	body := readBody(t, resp)
	if strings.Contains(body, `"hostname"`) {
		t.Fatalf("expected plain text, got JSON-like: %q", body)
	}
	if !strings.Contains(body, "status:") {
		t.Fatalf("plain-text body missing 'status:' prefix: %q", body)
	}
}

// Common header middleware must set NGHT-Hostname on every response.
func TestCommonHeader_HostnameSet(t *testing.T) {
	app := newTestApp()
	resp := doGET(t, app, "/echo/x")
	if got := resp.Header.Get("NGHT-Hostname"); got == "" {
		t.Fatalf("NGHT-Hostname header missing")
	}
}

// Wildcard route returns 200 with url + hostname for unrecognized paths.
func TestWildcardFallback(t *testing.T) {
	app := newTestApp()
	resp := doGET(t, app, "/totally-not-a-route")
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "/totally-not-a-route") {
		t.Fatalf("wildcard did not echo url: %q", body)
	}
}

// CRITICAL regression: log.Infof("%s", body) must not panic or crash on
// format-string-like bodies. Pre-fix the handler used body as the format string.
func TestLogReqData_NoFormatStringCrash(t *testing.T) {
	app := newTestApp()
	req, _ := http.NewRequest("POST", "/log_req_data", strings.NewReader("%s %d %x %!s(MISSING)"))
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test err = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
