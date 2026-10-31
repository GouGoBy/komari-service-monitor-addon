package availability

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/komari-monitor/komari/cmd/flags"
	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/internal/scheduler"
)

func TestServicePersistenceVisibilityAndHistory(t *testing.T) {
	flags.DatabaseType = flags.DatabaseTypeSQLite
	flags.DatabaseFile = "file:availability_test?mode=memory&cache=shared"
	db := dbcore.GetDBInstance()
	t.Cleanup(scheduler.StopAll)
	secret := "private-token"
	m := models.ServiceMonitor{Name: "Bot", Type: "astrbot", URL: "http://127.0.0.1:6185", Interval: 60, Timeout: 10}
	if err := Save(&m, &secret); err != nil {
		t.Fatal(err)
	}
	if m.ID == 0 {
		t.Fatal("missing ID")
	}
	m.Name = "Renamed"
	if err := Save(&m, nil); err != nil {
		t.Fatal(err)
	}
	var saved models.ServiceMonitor
	if err := db.First(&saved, m.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Token != secret {
		t.Fatal("edit erased token")
	}
	views, err := List(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(views)
	if strings.Contains(string(raw), secret) || len(views) != 1 || !views[0].HasToken || views[0].Status != "paused" {
		t.Fatalf("admin views: %s", raw)
	}
	views, err = List(false)
	if err != nil || len(views) != 0 {
		t.Fatalf("private monitor exposed: %+v, %v", views, err)
	}
	if _, err = History(m.ID, false); err == nil {
		t.Fatal("private history exposed")
	}
	// Enable directly to inspect stored results without issuing network probes.
	if err = db.Model(&saved).UpdateColumns(map[string]any{"enabled": true, "public": true}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	checks := []models.ServiceCheck{
		{MonitorID: m.ID, Time: now.Add(-time.Minute), Up: false, Error: "private diagnostic"},
		{MonitorID: m.ID, Time: now, Up: true, StatusCode: 200},
		{MonitorID: m.ID, Time: now.Add(-25 * time.Hour), Up: false},
	}
	if err = db.Create(&checks).Error; err != nil {
		t.Fatal(err)
	}
	views, err = List(false)
	if err != nil || len(views) != 1 {
		t.Fatalf("public list: %+v %v", views, err)
	}
	if views[0].Status != "up" || views[0].Availability == nil || *views[0].Availability != 50 || views[0].Config != nil || views[0].HasToken {
		t.Fatalf("invalid summary: %+v", views[0])
	}
	history, err := History(m.ID, false)
	if err != nil || len(history) != 2 {
		t.Fatalf("history: %+v %v", history, err)
	}
	for _, r := range history {
		if r.Error != "" {
			t.Fatal("public diagnostic exposed")
		}
	}
	if err = db.Model(&models.ServiceCheck{}).Where("monitor_id = ?", m.ID).Update("time", now.Add(-10*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	views, err = List(false)
	if err != nil || views[0].Status != "unknown" {
		t.Fatalf("stale status: %+v %v", views, err)
	}
	if err = Delete(m.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.ServiceCheck{}).Where("monitor_id = ?", m.ID).Count(&count)
	if count != 0 {
		t.Fatal("delete left orphan history")
	}
}
