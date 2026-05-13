package collector

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/lukas/authentik-exporter/internal/client"
	"github.com/lukas/authentik-exporter/internal/config"
)

const (
	targetEvents  = "events"
	targetSignups = "signups"
	targetUsers   = "users"
)

type Poller struct {
	cfg     *config.Config
	cli     *client.Client
	metrics *Metrics
	log     *slog.Logger
}

func NewPoller(cfg *config.Config, cli *client.Client, m *Metrics, log *slog.Logger) *Poller {
	return &Poller{cfg: cfg, cli: cli, metrics: m, log: log}
}

// Run drives the polling loop until ctx is cancelled. It performs an initial
// poll immediately so the /metrics endpoint has data before the first tick.
func (p *Poller) Run(ctx context.Context) {
	p.pollOnce(ctx)
	t := time.NewTicker(p.cfg.PollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.pollOnce(ctx)
		}
	}
}

func (p *Poller) pollOnce(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(3)
	var eventsOK, signupsOK, usersOK bool
	go func() { defer wg.Done(); eventsOK = p.pollEvents(ctx) }()
	go func() { defer wg.Done(); signupsOK = p.pollSignups(ctx) }()
	go func() { defer wg.Done(); usersOK = p.pollUsers(ctx) }()
	wg.Wait()
	if eventsOK && signupsOK && usersOK {
		p.metrics.ExporterUp.Set(1)
	} else {
		p.metrics.ExporterUp.Set(0)
	}
}

// pollEvents fetches the volume of each configured action over the maximum
// configured window, then sums the 6h buckets that fall inside each window.
func (p *Poller) pollEvents(ctx context.Context) bool {
	start := time.Now()
	historyDays := int(math.Ceil(p.cfg.MaxWindow().Hours() / 24))
	if historyDays < 1 {
		historyDays = 1
	}
	allOK := true
	now := time.Now()

	for _, action := range p.cfg.Actions {
		buckets, err := p.cli.EventVolume(ctx, client.EventVolumeFilter{
			Action:      action,
			HistoryDays: historyDays,
		})
		if err != nil {
			p.log.Error("poll events failed", "action", action, "err", err)
			p.metrics.ScrapeErrorsTotal.WithLabelValues(targetEvents).Inc()
			allOK = false
			continue
		}
		for _, w := range p.cfg.Windows {
			cutoff := now.Add(-w.Duration)
			var sum float64
			for _, b := range buckets {
				if !b.Time.Before(cutoff) {
					sum += float64(b.Count)
				}
			}
			p.metrics.EventCount.WithLabelValues(action, w.Label).Set(sum)
		}
	}

	p.metrics.ScrapeDuration.WithLabelValues(targetEvents).Set(time.Since(start).Seconds())
	if allOK {
		p.metrics.LastSuccessSeconds.WithLabelValues(targetEvents).Set(float64(time.Now().Unix()))
	}
	return allOK
}

// pollSignups queries the users API for verified and unverified user counts
// per window. Uses page_size=1 + pagination.count for cheap counting.
func (p *Poller) pollSignups(ctx context.Context) bool {
	start := time.Now()
	allOK := true
	now := time.Now()

	trueVal, falseVal := true, false
	for _, w := range p.cfg.Windows {
		since := now.Add(-w.Duration)

		verified, err := p.cli.UserCount(ctx, client.UserCountFilter{
			IsActive:     &trueVal,
			DateJoinedGt: &since,
		})
		if err != nil {
			p.log.Error("poll signups (verified) failed", "window", w.Label, "err", err)
			p.metrics.ScrapeErrorsTotal.WithLabelValues(targetSignups).Inc()
			allOK = false
			continue
		}
		unverified, err := p.cli.UserCount(ctx, client.UserCountFilter{
			IsActive:     &falseVal,
			DateJoinedGt: &since,
		})
		if err != nil {
			p.log.Error("poll signups (unverified) failed", "window", w.Label, "err", err)
			p.metrics.ScrapeErrorsTotal.WithLabelValues(targetSignups).Inc()
			allOK = false
			continue
		}

		p.metrics.Signups.WithLabelValues(w.Label, StateVerified).Set(float64(verified))
		p.metrics.Signups.WithLabelValues(w.Label, StateUnverified).Set(float64(unverified))
	}

	p.metrics.ScrapeDuration.WithLabelValues(targetSignups).Set(time.Since(start).Seconds())
	if allOK {
		p.metrics.LastSuccessSeconds.WithLabelValues(targetSignups).Set(float64(time.Now().Unix()))
	}
	return allOK
}

// pollUsers queries the total verified and unverified user counts (no time
// filter). Two cheap requests per poll using page_size=1 + pagination.count.
func (p *Poller) pollUsers(ctx context.Context) bool {
	start := time.Now()
	allOK := true
	trueVal, falseVal := true, false

	verified, err := p.cli.UserCount(ctx, client.UserCountFilter{IsActive: &trueVal})
	if err != nil {
		p.log.Error("poll users (verified) failed", "err", err)
		p.metrics.ScrapeErrorsTotal.WithLabelValues(targetUsers).Inc()
		allOK = false
	} else {
		p.metrics.Users.WithLabelValues(StateVerified).Set(float64(verified))
	}

	unverified, err := p.cli.UserCount(ctx, client.UserCountFilter{IsActive: &falseVal})
	if err != nil {
		p.log.Error("poll users (unverified) failed", "err", err)
		p.metrics.ScrapeErrorsTotal.WithLabelValues(targetUsers).Inc()
		allOK = false
	} else {
		p.metrics.Users.WithLabelValues(StateUnverified).Set(float64(unverified))
	}

	p.metrics.ScrapeDuration.WithLabelValues(targetUsers).Set(time.Since(start).Seconds())
	if allOK {
		p.metrics.LastSuccessSeconds.WithLabelValues(targetUsers).Set(float64(time.Now().Unix()))
	}
	return allOK
}
