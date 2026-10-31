package availability

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/komari-monitor/komari/database/models"
)

const maxBody = 2 * 1024 * 1024

var errRedirectOrigin = errors.New("AstrBot redirected to another origin; use the final HTTPS URL directly")
var errRedirectLimit = errors.New("too many redirects; check the reverse proxy")

func Validate(m *models.ServiceMonitor) error {
	m.Name = strings.TrimSpace(m.Name)
	u, err := url.Parse(m.URL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("url must be an absolute HTTP(S) URL without credentials or fragment")
	}
	if m.Name == "" || len(m.Name) > 100 || len(m.URL) > 4096 || len(m.Keyword) > 4096 || len(m.Token) > 8192 || strings.ContainsAny(m.Token, "\r\n") {
		return fmt.Errorf("invalid name, URL, keyword or token")
	}
	if m.Interval < 10 || m.Interval > 86400 || m.Timeout < 1 || m.Timeout > 60 || m.Timeout >= m.Interval {
		return fmt.Errorf("interval must be 10..86400 seconds; timeout must be 1..60 and less than interval")
	}
	if m.Type != "http" && m.Type != "astrbot" {
		return fmt.Errorf("type must be http or astrbot")
	}
	if m.ExpectedStatus != 0 && (m.ExpectedStatus < 100 || m.ExpectedStatus > 599) {
		return fmt.Errorf("invalid expected_status")
	}
	if m.Type == "astrbot" {
		if u.RawQuery != "" {
			return fmt.Errorf("AstrBot URL must not include a query")
		}
		if m.Token == "" {
			return fmt.Errorf("AstrBot token is required")
		}
		if m.AuthType == "" {
			m.AuthType = "apikey"
		}
		if m.AuthType != "apikey" && m.AuthType != "bearer" {
			return fmt.Errorf("auth_type must be apikey or bearer")
		}
		if m.APIPath == "" {
			m.APIPath = "/api/v1/bots"
		}
		if m.APIPath != "/api/v1/stats" && m.APIPath != "/api/v1/bots" && m.APIPath != "/api/stat/get" {
			return fmt.Errorf("unsupported AstrBot API path")
		}
		if m.BotID != "" && m.APIPath == "/api/stat/get" {
			return fmt.Errorf("bot_id requires the v1 API")
		}
	}
	return nil
}

// Probe restricts authenticated AstrBot redirects to the same origin.
// Internal addresses are allowed intentionally for administrator-created LAN checks.
func Probe(ctx context.Context, m models.ServiceMonitor) (result models.ServiceCheck) {
	result.MonitorID, result.Time = m.ID, time.Now().UTC()
	start := time.Now()
	defer func() { result.Latency = float64(time.Since(start).Microseconds()) / 1000 }()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.Timeout)*time.Second)
	defer cancel()
	target := m.URL
	if m.Type == "astrbot" {
		apiPath := m.APIPath
		// Keep old saved /stats configurations working without requiring the
		// unsupported developer-key "system" scope. No config migration is needed.
		if apiPath == "" || apiPath == "/api/v1/stats" {
			apiPath = "/api/v1/bots"
		}
		target = strings.TrimRight(target, "/") + apiPath
		if m.BotID != "" {
			target = strings.TrimRight(m.URL, "/") + "/api/v1/bots/stats"
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		result.Error = "Invalid request URL"
		return
	}
	req.Header.Set("User-Agent", "Komari-Availability/1.0")
	if m.Type == "astrbot" {
		req.Header.Set("Accept", "application/json")
		if m.AuthType == "bearer" {
			req.Header.Set("Authorization", "Bearer "+m.Token)
		} else {
			req.Header.Set("X-API-Key", m.Token)
		}
	}
	client := &http.Client{CheckRedirect: func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errRedirectLimit
		}
		if next.URL.Scheme != "http" && next.URL.Scheme != "https" {
			return fmt.Errorf("unsupported redirect scheme")
		}
		next.Header.Del("Referer")
		if m.Type == "astrbot" && (next.URL.Scheme != req.URL.Scheme || next.URL.Host != req.URL.Host) {
			return errRedirectOrigin
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		// Do not persist raw transport errors: they can include URLs and secrets.
		var dns *net.DNSError
		var untrusted x509.UnknownAuthorityError
		var hostname x509.HostnameError
		var invalid x509.CertificateInvalidError
		var tlsHeader tls.RecordHeaderError
		if errors.Is(err, errRedirectOrigin) {
			result.Error = errRedirectOrigin.Error()
		} else if errors.Is(err, errRedirectLimit) {
			result.Error = errRedirectLimit.Error()
		} else if ctx.Err() != nil {
			result.Error = "Request timed out or canceled"
		} else if errors.As(err, &dns) {
			result.Error = "DNS lookup failed; check the hostname from the Komari server"
		} else if errors.As(err, &untrusted) {
			result.Error = "TLS certificate is not trusted; configure a valid certificate and full certificate chain"
		} else if errors.As(err, &hostname) {
			result.Error = "TLS certificate hostname mismatch; use the domain covered by the certificate"
		} else if errors.As(err, &invalid) {
			result.Error = "TLS certificate is invalid or expired; check the certificate and server clock"
		} else if errors.As(err, &tlsHeader) {
			result.Error = "TLS handshake failed; check whether this port uses HTTP or HTTPS"
		} else if errors.Is(err, syscall.ECONNREFUSED) {
			result.Error = "Connection refused; check the address, listening port and Docker port mapping"
		} else {
			result.Error = "Connection or TLS handshake failed; check network reachability from the Komari server"
		}
		if resp != nil {
			result.StatusCode = resp.StatusCode
		}
		return
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	validStatus := resp.StatusCode >= 200 && resp.StatusCode < 300
	if m.Type == "http" && m.ExpectedStatus != 0 {
		validStatus = resp.StatusCode == m.ExpectedStatus
	}
	if !validStatus {
		result.Error = fmt.Sprintf("Unexpected HTTP status %d", resp.StatusCode)
		if m.Type == "astrbot" {
			switch resp.StatusCode {
			case 401:
				result.Error = "AstrBot authentication failed (HTTP 401); check API key or JWT"
			case 403:
				result.Error = "AstrBot access denied (HTTP 403); API key requires bot scope; legacy Dashboard API requires a Dashboard JWT"
			case 404:
				result.Error = "AstrBot API not found (HTTP 404); check base URL, reverse proxy prefix and API version"
			}
		}
		return
	}
	if m.Type == "http" && m.Keyword == "" {
		result.Up = true
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		result.Error = "Failed to read response"
		return
	}
	if len(body) > maxBody {
		result.Error = "Response exceeds 2 MiB inspection limit"
		return
	}
	if m.Type == "http" {
		result.Up = strings.Contains(string(body), m.Keyword)
		if !result.Up {
			result.Error = "Required keyword not found"
		}
		return
	}
	var envelope struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if json.Unmarshal(body, &envelope) != nil || envelope.Status != "ok" {
		result.Error = "Invalid AstrBot response or API reported an error"
		return
	}
	if m.BotID != "" {
		var data struct {
			Platforms []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"platforms"`
		}
		if json.Unmarshal(envelope.Data, &data) == nil {
			for _, bot := range data.Platforms {
				if bot.ID == m.BotID {
					result.Up = bot.Status == "running"
					if !result.Up {
						result.Error = "AstrBot platform is not running"
					}
					return
				}
			}
		}
		result.Error = "AstrBot platform not found"
		return
	}
	if m.APIPath != "/api/stat/get" {
		var data struct {
			Bots []json.RawMessage `json:"bots"`
		}
		if json.Unmarshal(envelope.Data, &data) != nil || data.Bots == nil {
			result.Error = "AstrBot bot list response is invalid; check API version and reverse proxy routing"
			return
		}
		result.Up = true
		return
	}
	var data struct {
		StartTime *float64 `json:"start_time"`
	}
	if json.Unmarshal(envelope.Data, &data) != nil || data.StartTime == nil || *data.StartTime <= 0 {
		result.Error = "AstrBot statistics missing start_time"
		return
	}
	result.Up = true
	return
}
