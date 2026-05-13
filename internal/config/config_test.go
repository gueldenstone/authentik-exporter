package config

import (
	"testing"
	"time"
)

func TestParseWindows(t *testing.T) {
	tests := []struct {
		in      string
		want    []Window
		wantErr bool
	}{
		{"5m,1h,24h", []Window{{"5m", 5 * time.Minute}, {"1h", time.Hour}, {"24h", 24 * time.Hour}}, false},
		{"1h , 30m", []Window{{"1h", time.Hour}, {"30m", 30 * time.Minute}}, false},
		{"1h,1h,2h", []Window{{"1h", time.Hour}, {"2h", 2 * time.Hour}}, false},
		{"7d,2w", []Window{{"7d", 7 * 24 * time.Hour}, {"2w", 14 * 24 * time.Hour}}, false},
		{"5m,1h,24h,7d", []Window{{"5m", 5 * time.Minute}, {"1h", time.Hour}, {"24h", 24 * time.Hour}, {"7d", 7 * 24 * time.Hour}}, false},
		{"bogus", nil, true},
		{"0s", nil, true},
		{"", nil, true},
	}
	for _, tt := range tests {
		got, err := parseWindows(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseWindows(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("parseWindows(%q) len=%d want %d", tt.in, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseWindows(%q)[%d]=%v want %v", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMaxWindow(t *testing.T) {
	c := &Config{Windows: []Window{
		{"5m", 5 * time.Minute},
		{"7d", 7 * 24 * time.Hour},
		{"1h", time.Hour},
	}}
	if got := c.MaxWindow(); got != 7*24*time.Hour {
		t.Errorf("MaxWindow=%v want 168h", got)
	}
}
