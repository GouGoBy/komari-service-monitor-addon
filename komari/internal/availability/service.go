package availability

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/internal/scheduler"
	logger "github.com/komari-monitor/komari/utils/log"
	"gorm.io/gorm"
)

var mu sync.Mutex
var slots = make(chan struct{}, 16)

func Save(m *models.ServiceMonitor, token *string) error {
	mu.Lock()
	defer mu.Unlock()
	db := dbcore.GetDBInstance()
	m.CreatedAt, m.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	if m.ID != 0 {
		var old models.ServiceMonitor
		if err := db.First(&old, m.ID).Error; err != nil {
			return err
		}
		m.Token, m.CreatedAt = old.Token, old.CreatedAt
	}
	if token != nil {
		m.Token = *token
	}
	if m.Type == "http" {
		m.Token, m.BotID, m.APIPath, m.AuthType = "", "", "", ""
	}
	if err := Validate(m); err != nil {
		return err
	}
	if err := db.Save(m).Error; err != nil {
		return err
	}
	return reload(db)
}

func Delete(id uint) error {
	mu.Lock()
	defer mu.Unlock()
	db := dbcore.GetDBInstance()
	err := db.Transaction(func(tx *gorm.DB) error {
		var m models.ServiceMonitor
		if err := tx.First(&m, id).Error; err != nil {
			return err
		}
		if err := tx.Where("monitor_id = ?", id).Delete(&models.ServiceCheck{}).Error; err != nil {
			return err
		}
		return tx.Delete(&m).Error
	})
	if err != nil {
		return err
	}
	return reload(db)
}

func Reload() error {
	mu.Lock()
	defer mu.Unlock()
	return reload(dbcore.GetDBInstance())
}

func reload(db *gorm.DB) error {
	var monitors []models.ServiceMonitor
	if err := db.Where("enabled = ?", true).Find(&monitors).Error; err != nil {
		return err
	}
	scheduler.RemovePrefix("availability:")
	for _, m := range monitors {
		if err := Validate(&m); err != nil {
			logger.Errorf("availability", "Invalid monitor %d: %v", m.ID, err)
			continue
		}
		var running atomic.Bool
		if err := scheduler.AddContextFunc(fmt.Sprintf("availability:%d", m.ID), scheduler.Every(time.Duration(m.Interval)*time.Second), true, func(ctx context.Context) {
			if !running.CompareAndSwap(false, true) {
				return
			}
			defer running.Store(false)
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			default:
				return
			}
			result := Probe(ctx, m)
			mu.Lock()
			defer mu.Unlock()
			if ctx.Err() != nil {
				return
			}
			if err := db.Create(&result).Error; err != nil {
				logger.Errorf("availability", "Persist check %d: %v", m.ID, err)
			}
		}); err != nil {
			return err
		}
	}
	return scheduler.AddContextFunc("availability:retention", "@every 1h", true, func(ctx context.Context) {
		if err := db.WithContext(ctx).Where("time < ?", time.Now().UTC().Add(-30*24*time.Hour)).Delete(&models.ServiceCheck{}).Error; err != nil && ctx.Err() == nil {
			logger.Errorf("availability", "History cleanup: %v", err)
		}
	})
}

// View explicitly controls which information can leave the server.
type View struct {
	ID           uint                   `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Status       string                 `json:"status"`
	Latest       *models.ServiceCheck   `json:"latest"`
	Availability *float64               `json:"availability_24h"`
	Checks       int64                  `json:"checks_24h"`
	Config       *models.ServiceMonitor `json:"config,omitempty"`
	HasToken     bool                   `json:"has_token,omitempty"`
}

func List(admin bool) ([]View, error) {
	db := dbcore.GetDBInstance()
	query := db.Order("id")
	if !admin {
		query = query.Where("public = ? AND enabled = ?", true, true)
	}
	var monitors []models.ServiceMonitor
	if err := query.Find(&monitors).Error; err != nil {
		return nil, err
	}
	views := make([]View, 0, len(monitors))
	for _, m := range monitors {
		v := View{ID: m.ID, Name: m.Name, Type: m.Type, Status: "unknown"}
		if admin {
			v.Config = &m
			v.HasToken = m.Token != ""
		}
		if !m.Enabled {
			v.Status = "paused"
		}
		var latest models.ServiceCheck
		err := db.Where("monitor_id = ?", m.ID).Order("time DESC, id DESC").First(&latest).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
		if err == nil {
			if !admin {
				latest.Error = ""
			}
			v.Latest = &latest
			if m.Enabled && !latest.Time.Before(m.UpdatedAt) && time.Since(latest.Time) <= time.Duration(m.Interval*2+m.Timeout)*time.Second {
				v.Status = "down"
				if latest.Up {
					v.Status = "up"
				}
			}
		}
		var totals struct {
			Total      int64
			Successful int64
		}
		if err := db.Model(&models.ServiceCheck{}).Select("COUNT(*) AS total, COALESCE(SUM(CASE WHEN up = ? THEN 1 ELSE 0 END), 0) AS successful", true).Where("monitor_id = ? AND time >= ?", m.ID, time.Now().UTC().Add(-24*time.Hour)).Scan(&totals).Error; err != nil {
			return nil, err
		}
		v.Checks = totals.Total
		if totals.Total > 0 {
			value := float64(totals.Successful) * 100 / float64(totals.Total)
			v.Availability = &value
		}
		views = append(views, v)
	}
	return views, nil
}

func History(id uint, admin bool) ([]models.ServiceCheck, error) {
	db := dbcore.GetDBInstance()
	query := db.Where("id = ?", id)
	if !admin {
		query = query.Where("public = ? AND enabled = ?", true, true)
	}
	var m models.ServiceMonitor
	if err := query.First(&m).Error; err != nil {
		return nil, err
	}
	checks := []models.ServiceCheck{}
	if err := db.Where("monitor_id = ? AND time >= ?", id, time.Now().UTC().Add(-24*time.Hour)).Order("time DESC, id DESC").Limit(1000).Find(&checks).Error; err != nil {
		return nil, err
	}
	if !admin {
		for i := range checks {
			checks[i].Error = ""
		}
	}
	return checks, nil
}
