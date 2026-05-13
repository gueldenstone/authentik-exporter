package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// EventVolumeFilter narrows a volume query. Action is required; other fields optional.
type EventVolumeFilter struct {
	Action           string
	ContextModelName string // e.g. "user" to count User-model creations
	HistoryDays      int    // 1..90; if zero, server default (7) is used
}

// EventBucket is one row from the /events/events/volume/ endpoint.
type EventBucket struct {
	Action string
	Time   time.Time
	Count  int
}

// flexibleTime accepts both RFC3339 (with offset/Z) and the naive
// "YYYY-MM-DDTHH:MM:SS" form authentik returns from the volume endpoint.
// Naive timestamps are interpreted as UTC, which matches authentik's
// default server-side TZ assumption.
type flexibleTime struct{ time.Time }

func (t *flexibleTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed.UTC()
			return nil
		}
	}
	return fmt.Errorf("unrecognized time format: %q", s)
}

type volumeRow struct {
	Action string       `json:"action"`
	Time   flexibleTime `json:"time"`
	Count  int          `json:"count"`
}

// EventVolume fetches aggregated event counts. Authentik returns one entry per
// 6-hour bucket per action within the requested history window.
//
// We bypass the generated client's JSON decoder here because the server returns
// naive timestamps (e.g. "2026-05-07T12:00:00") which the generated code
// rejects under RFC3339.
func (c *Client) EventVolume(ctx context.Context, f EventVolumeFilter) ([]EventBucket, error) {
	if f.Action == "" {
		return nil, fmt.Errorf("action is required")
	}

	q := url.Values{}
	q.Set("action", f.Action)
	if f.ContextModelName != "" {
		q.Set("context_model_name", f.ContextModelName)
	}
	if f.HistoryDays > 0 {
		q.Set("history_days", strconv.Itoa(f.HistoryDays))
	}

	endpoint := c.baseURL + "/events/events/volume/?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("events volume action=%s: %w", f.Action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("events volume action=%s: HTTP %d: %s", f.Action, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rows []volumeRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode volume response: %w", err)
	}

	out := make([]EventBucket, len(rows))
	for i, r := range rows {
		out[i] = EventBucket{Action: r.Action, Time: r.Time.Time, Count: r.Count}
	}
	return out, nil
}
