package client

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlexibleTimeUnmarshal(t *testing.T) {
	cases := []struct {
		in   string
		want time.Time
	}{
		{`"2026-05-07T12:00:00"`, time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)},
		{`"2026-05-07T12:00:00Z"`, time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)},
		{`"2026-05-07T12:00:00+02:00"`, time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)},
		{`"2026-05-07T12:00:00.123456Z"`, time.Date(2026, 5, 7, 12, 0, 0, 123456000, time.UTC)},
	}
	for _, tc := range cases {
		var ft flexibleTime
		if err := json.Unmarshal([]byte(tc.in), &ft); err != nil {
			t.Errorf("Unmarshal(%s) err=%v", tc.in, err)
			continue
		}
		if !ft.Time.Equal(tc.want) {
			t.Errorf("Unmarshal(%s)=%s want %s", tc.in, ft.Time, tc.want)
		}
	}
}

func TestFlexibleTimeBadFormat(t *testing.T) {
	var ft flexibleTime
	if err := json.Unmarshal([]byte(`"not-a-date"`), &ft); err == nil {
		t.Errorf("expected error for bad format")
	}
}
