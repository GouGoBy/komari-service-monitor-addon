package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/komari-monitor/komari/cmd/flags"
	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/internal/scheduler"
	"github.com/komari-monitor/komari/pkg/rpc"
	"github.com/komari-monitor/komari/web/api"
)

func TestServiceMonitorRoutes(t *testing.T) {
	flags.DatabaseType = flags.DatabaseTypeSQLite
	flags.DatabaseFile = "file:service_routes?mode=memory&cache=shared"
	dbcore.GetDBInstance()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		p := rpc.NewAnonymousPrincipal()
		if c.GetHeader("X-Test-Admin") == "yes" {
			p = rpc.NewAPIKeyPrincipal()
		}
		api.SetPrincipal(c, p)
		c.Set("role", p.PrimaryRole())
		c.Next()
	})
	Register(r)
	t.Cleanup(scheduler.StopAll)
	request := func(method, path, body string, admin bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if admin {
			req.Header.Set("X-Test-Admin", "yes")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	for _, path := range []string{"/api/admin/services"} {
		if w := request("GET", path, "", false); w.Code != http.StatusUnauthorized {
			t.Fatalf("guest access %s: %d", path, w.Code)
		}
	}
	if w := request("POST", "/api/rpc2", `{"jsonrpc":"2.0","id":1,"method":"admin:getServiceMonitors","params":{}}`, false); !strings.Contains(w.Body.String(), "Permission denied") {
		t.Fatalf("RPC auth: %s", w.Body.String())
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "healthy") }))
	defer target.Close()
	body := fmt.Sprintf(`{"name":"Website","type":"http","url":%q,"interval":10,"timeout":1,"keyword":"healthy","enabled":true,"public":true}`, target.URL)
	w := request("POST", "/api/admin/services/save", body, true)
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Data.ID == 0 {
		t.Fatalf("create response: %s", w.Body.String())
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		w = request("GET", "/api/services", "", false)
		if strings.Contains(w.Body.String(), `"status":"up"`) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("probe did not persist: %s", w.Body.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if strings.Contains(w.Body.String(), target.URL) || strings.Contains(w.Body.String(), `"config"`) {
		t.Fatalf("public configuration leak: %s", w.Body.String())
	}
	w = request("GET", fmt.Sprintf("/api/services/history?id=%d", created.Data.ID), "", false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"up":true`) {
		t.Fatalf("history query: %d %s", w.Code, w.Body.String())
	}
	w = request("POST", "/api/admin/services/delete", fmt.Sprintf(`{"id":%d}`, created.Data.ID), true)
	if w.Code != 200 {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	w = request("GET", fmt.Sprintf("/api/services/history?id=%d", created.Data.ID), "", false)
	if w.Code == 200 {
		t.Fatal("deleted history remains accessible")
	}
}
