package availability

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/komari-monitor/komari/database/models"
)

func TestAstrBotAuthenticationCompatibility(t *testing.T) {
	for _, tt := range []struct{ name, auth, path, requestPath, body string }{
		{"saved stats config", "apikey", "/api/v1/stats", "/prefix/api/v1/bots", `{"status":"ok","data":{"bots":[]}}`},
		{"bearer API key", "bearer", "/api/v1/bots", "/prefix/api/v1/bots", `{"status":"ok","data":{"bots":[]}}`},
		{"legacy JWT", "bearer", "/api/stat/get", "/prefix/api/stat/get", `{"status":"ok","data":{"start_time":1234}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.requestPath {
					t.Errorf("request path: %s", r.URL.Path)
				}
				if tt.auth == "apikey" && r.Header.Get("X-API-Key") != "test-secret" {
					t.Error("missing API key")
				}
				if tt.auth == "bearer" && r.Header.Get("Authorization") != "Bearer test-secret" {
					t.Error("missing bearer token")
				}
				fmt.Fprint(w, tt.body)
			}))
			defer s.Close()
			r := Probe(context.Background(), models.ServiceMonitor{Type: "astrbot", URL: s.URL + "/prefix/", Timeout: 1, AuthType: tt.auth, APIPath: tt.path, Token: "test-secret"})
			if !r.Up {
				t.Fatalf("probe: %+v", r)
			}
		})
	}
}

func TestAstrBotPermissionFailureHasUsefulDiagnostic(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, `{"status":"error","message":"private upstream details"}`)
	}))
	defer s.Close()
	r := Probe(context.Background(), models.ServiceMonitor{Type: "astrbot", URL: s.URL, Timeout: 1, Token: "secret"})
	if r.Up || !strings.Contains(r.Error, "bot scope") || strings.Contains(r.Error, "private upstream") {
		t.Fatalf("diagnostics: %+v", r)
	}
}
