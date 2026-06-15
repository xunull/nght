package gin_server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func resetHealth() {
	mu.Lock()
	health = true
	mu.Unlock()
}

func newTestApp() *gin.Engine {
	gin.SetMode(gin.TestMode)
	ginServer = gin.New()
	AddRoute()
	return ginServer
}

func doGET(t *testing.T, app *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	return w
}

func TestLivez_AlwaysOK(t *testing.T) {
	app := newTestApp()
	w := doGET(t, app, "/livez")
	if w.Code != 200 {
		t.Fatalf("/livez = %d, want 200", w.Code)
	}
}

func TestLivez_AfterHealthFlip(t *testing.T) {
	defer resetHealth()
	app := newTestApp()

	if w := doGET(t, app, "/livez"); w.Code != 200 {
		t.Fatalf("initial /livez = %d", w.Code)
	}

	doGET(t, app, "/health/false")

	if w := doGET(t, app, "/livez"); w.Code != 200 {
		t.Fatalf("/livez after /health/false = %d, want 200", w.Code)
	}

	if w := doGET(t, app, "/health"); w.Code != 502 {
		t.Fatalf("/health after false = %d, want 502", w.Code)
	}
}
