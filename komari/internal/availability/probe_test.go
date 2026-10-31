package availability

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/komari-monitor/komari/database/models"
)

func TestProbe(t *testing.T) {
	tests := []struct {
		name, kind, body, keyword, bot string
		code, expected                 int
		up                             bool
	}{
		{"website", "http", "hello", "", "", 200, 0, true},
		{"no content", "http", "", "", "", 204, 0, true},
		{"server error", "http", "error", "", "", 503, 0, false},
		{"expected status", "http", "missing", "", "", 404, 404, true},
		{"keyword", "http", "service healthy", "healthy", "", 200, 0, true},
		{"missing keyword", "http", "login", "healthy", "", 200, 0, false},
		{"astrbot", "astrbot", `{"status":"ok","data":{"bots":[]}}`, "", "", 200, 0, true},
		{"astrbot bot list", "astrbot", `{"status":"ok","data":{"bots":[{"id":"qq"}]}}`, "", "", 200, 0, true},
		{"null bot list", "astrbot", `{"status":"ok","data":{"bots":null}}`, "", "", 200, 0, false},
		{"login page", "astrbot", "<html>login</html>", "", "", 200, 0, false},
		{"business error", "astrbot", `{"status":"error","data":{"start_time":1234}}`, "", "", 200, 0, false},
		{"empty data", "astrbot", `{"status":"ok","data":{}}`, "", "", 200, 0, false},
		{"unauthorized", "astrbot", `{"status":"error"}`, "", "", 401, 0, false},
		{"running bot", "astrbot", `{"status":"ok","data":{"platforms":[{"id":"qq","status":"running"}]}}`, "", "qq", 200, 0, true},
		{"stopped bot", "astrbot", `{"status":"ok","data":{"platforms":[{"id":"qq","status":"stopped"}]}}`, "", "qq", 200, 0, false},
		{"missing bot", "astrbot", `{"status":"ok","data":{"platforms":[]}}`, "", "qq", 200, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.kind == "astrbot" {
					if r.Header.Get("X-API-Key") != "secret" {
						t.Error("missing API key")
					}
					path := "/proxy/api/v1/bots"
					if tt.bot != "" {
						path = "/proxy/api/v1/bots/stats"
					}
					if r.URL.Path != path {
						t.Errorf("path = %s", r.URL.Path)
					}
				}
				w.WriteHeader(tt.code)
				fmt.Fprint(w, tt.body)
			}))
			defer s.Close()
			m := models.ServiceMonitor{ID: 7, Name: "test", Type: tt.kind, URL: s.URL + "/proxy", Timeout: 1, Interval: 60, Token: "secret", Keyword: tt.keyword, BotID: tt.bot, ExpectedStatus: tt.expected}
			if err := Validate(&m); err != nil {
				t.Fatal(err)
			}
			r := Probe(context.Background(), m)
			if r.Up != tt.up || r.StatusCode != tt.code || r.MonitorID != 7 || r.Latency < 0 {
				t.Fatalf("result = %+v", r)
			}
			if !r.Up && r.Error == "" {
				t.Error("missing failure reason")
			}
		})
	}
}

func TestRedirectCredentialsAndWebsiteRedirect(t *testing.T) {
	var received bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Header.Get("X-API-Key") != "" {
			t.Error("leaked secret")
		}
		fmt.Fprint(w, "ok")
	}))
	defer target.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer s.Close()
	m := models.ServiceMonitor{Type: "astrbot", URL: s.URL, APIPath: "/api/v1/stats", Token: "secret", Timeout: 1}
	if r := Probe(context.Background(), m); r.Up || received {
		t.Fatalf("redirect escaped origin: %+v", r)
	}
	if r := Probe(context.Background(), m); !strings.Contains(r.Error, "use the final HTTPS URL") {
		t.Fatalf("unclear redirect error: %+v", r)
	}
	m.Type, m.Token = "http", ""
	if r := Probe(context.Background(), m); !r.Up || !received {
		t.Fatalf("website redirect failed: %+v", r)
	}
}

func TestProbeTimeoutTLSAndBodyLimit(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer s.Close()
	start := time.Now()
	r := Probe(context.Background(), models.ServiceMonitor{URL: s.URL, Type: "http", Timeout: 1})
	if r.Up || r.Error == "" || time.Since(start) > 3*time.Second {
		t.Fatalf("timeout: %+v", r)
	}
	tls := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer tls.Close()
	if r := Probe(context.Background(), models.ServiceMonitor{URL: tls.URL, Type: "http", Timeout: 1}); r.Up || !strings.Contains(r.Error, "not trusted") {
		t.Fatalf("untrusted TLS diagnostics: %+v", r)
	}
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, strings.Repeat("x", maxBody+1)) }))
	defer big.Close()
	if r := Probe(context.Background(), models.ServiceMonitor{URL: big.URL, Type: "http", Keyword: "x", Timeout: 1}); r.Up || !strings.Contains(r.Error, "2 MiB") {
		t.Fatalf("body limit: %+v", r)
	}
}

func TestValidate(t *testing.T) {
	for _, change := range []func(*models.ServiceMonitor){
		func(m *models.ServiceMonitor) { m.URL = "file:///etc/passwd" },
		func(m *models.ServiceMonitor) { m.URL = "http://user:password@example.com" },
		func(m *models.ServiceMonitor) { m.Interval = 0 },
		func(m *models.ServiceMonitor) { m.Timeout = m.Interval },
		func(m *models.ServiceMonitor) { m.ExpectedStatus = 600 },
		func(m *models.ServiceMonitor) { m.Type = "astrbot"; m.Token = "" },
		func(m *models.ServiceMonitor) { m.Type = "astrbot"; m.Token = "secret"; m.APIPath = "/api/delete" },
		func(m *models.ServiceMonitor) {
			m.Type = "astrbot"
			m.Token = "secret"
			m.APIPath = "/api/stat/get"
			m.BotID = "qq"
		},
	} {
		m := models.ServiceMonitor{Name: "test", Type: "http", URL: "https://example.com", Interval: 60, Timeout: 10}
		change(&m)
		if Validate(&m) == nil {
			t.Fatalf("accepted invalid monitor: %+v", m)
		}
	}
}
