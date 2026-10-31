package models

import "time"

// ServiceMonitor is a server-side HTTP or AstrBot availability check.
type ServiceMonitor struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Name           string    `json:"name" gorm:"size:100"`
	Type           string    `json:"type" gorm:"size:16"`
	URL            string    `json:"url" gorm:"type:text"`
	Interval       int       `json:"interval"`
	Timeout        int       `json:"timeout"`
	ExpectedStatus int       `json:"expected_status"`
	Keyword        string    `json:"keyword" gorm:"type:text"`
	Token          string    `json:"-" gorm:"type:text"`
	AuthType       string    `json:"auth_type" gorm:"size:16"`
	APIPath        string    `json:"api_path" gorm:"size:100"`
	BotID          string    `json:"bot_id" gorm:"size:255"`
	Enabled        bool      `json:"enabled"`
	Public         bool      `json:"public"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ServiceCheck struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	MonitorID  uint      `json:"monitor_id" gorm:"index:idx_service_check_time,priority:1"`
	Time       time.Time `json:"time" gorm:"index:idx_service_check_time,priority:2;index"`
	Up         bool      `json:"up"`
	Latency    float64   `json:"latency"`
	StatusCode int       `json:"status_code"`
	Error      string    `json:"error,omitempty" gorm:"type:text"`
}
