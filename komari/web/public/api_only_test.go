package public

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIOnlyDoesNotExtractOrServeFrontend(t *testing.T) {
	if !APIOnly {
		t.Skip("requires -tags api_only")
	}
	t.Chdir(t.TempDir())
	if err := PrepareDefaultDist(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("cache/dist"); !os.IsNotExist(err) {
		t.Fatal("API server extracted a frontend")
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Static(r.Group("/"), func(h ...gin.HandlerFunc) { r.NoRoute(h...) })
	for _, path := range []string{"/", "/admin/service-monitor", "/assets/index.js"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Fatalf("%s served frontend: %d %s", path, w.Code, w.Body.String())
		}
	}
}
