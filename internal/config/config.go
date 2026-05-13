package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AuthentikURL        string
	AuthentikToken      string
	InsecureSkipVerify  bool
	ListenAddr          string
	MetricsPath         string
	PollInterval        time.Duration
	Windows             []Window
	Actions             []string
	LogLevel            string
}

type Window struct {
	Label    string
	Duration time.Duration
}

func Load() (*Config, error) {
	c := &Config{
		ListenAddr:   getenv("EXPORTER_LISTEN_ADDR", ":9119"),
		MetricsPath:  getenv("EXPORTER_METRICS_PATH", "/metrics"),
		LogLevel:     getenv("EXPORTER_LOG_LEVEL", "info"),
		AuthentikURL: os.Getenv("AUTHENTIK_URL"),
		AuthentikToken: os.Getenv("AUTHENTIK_TOKEN"),
	}

	if c.AuthentikURL == "" {
		return nil, errors.New("AUTHENTIK_URL is required")
	}
	if c.AuthentikToken == "" {
		return nil, errors.New("AUTHENTIK_TOKEN is required")
	}

	if v := os.Getenv("AUTHENTIK_INSECURE_SKIP_VERIFY"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("AUTHENTIK_INSECURE_SKIP_VERIFY: %w", err)
		}
		c.InsecureSkipVerify = b
	}

	pollStr := getenv("EXPORTER_POLL_INTERVAL", "60s")
	pi, err := time.ParseDuration(pollStr)
	if err != nil {
		return nil, fmt.Errorf("EXPORTER_POLL_INTERVAL: %w", err)
	}
	if pi < time.Second {
		return nil, fmt.Errorf("EXPORTER_POLL_INTERVAL too short: %s", pi)
	}
	c.PollInterval = pi

	windowsStr := getenv("EXPORTER_WINDOWS", "5m,1h,24h,7d")
	c.Windows, err = parseWindows(windowsStr)
	if err != nil {
		return nil, fmt.Errorf("EXPORTER_WINDOWS: %w", err)
	}

	actionsStr := getenv("EXPORTER_ACTIONS", "login,login_failed,logout,password_set,suspicious_request,authorize_application")
	c.Actions = splitCSV(actionsStr)
	if len(c.Actions) == 0 {
		return nil, errors.New("EXPORTER_ACTIONS must contain at least one action")
	}

	return c, nil
}

func parseWindows(s string) ([]Window, error) {
	parts := splitCSV(s)
	if len(parts) == 0 {
		return nil, errors.New("at least one window required")
	}
	out := make([]Window, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		d, err := parseDuration(p)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: %w", p, err)
		}
		if d <= 0 {
			return nil, fmt.Errorf("window must be positive: %q", p)
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, Window{Label: p, Duration: d})
	}
	return out, nil
}

// parseDuration accepts everything time.ParseDuration does, plus the "d"
// (day) and "w" (week) suffixes commonly used in metric retention configs.
// Only a single trailing d/w is recognised; combined forms like "1d12h"
// are rejected.
func parseDuration(s string) (time.Duration, error) {
	if len(s) > 1 {
		last := s[len(s)-1]
		if last == 'd' || last == 'w' {
			n, err := strconv.Atoi(s[:len(s)-1])
			if err != nil {
				return 0, err
			}
			unit := 24 * time.Hour
			if last == 'w' {
				unit = 7 * 24 * time.Hour
			}
			return time.Duration(n) * unit, nil
		}
	}
	return time.ParseDuration(s)
}

// MaxWindow returns the longest configured window. Used to determine
// historyDays for the Authentik volume API.
func (c *Config) MaxWindow() time.Duration {
	var max time.Duration
	for _, w := range c.Windows {
		if w.Duration > max {
			max = w.Duration
		}
	}
	return max
}

func splitCSV(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
