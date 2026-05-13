package collector

import "github.com/prometheus/client_golang/prometheus"

const namespace = "authentik"

type Metrics struct {
	EventCount         *prometheus.GaugeVec
	SignupsVerified    *prometheus.GaugeVec
	SignupsUnverified  *prometheus.GaugeVec
	SignupsTotal       *prometheus.GaugeVec
	ExporterUp         prometheus.Gauge
	ScrapeDuration     *prometheus.GaugeVec
	ScrapeErrorsTotal  *prometheus.CounterVec
	LastSuccessSeconds *prometheus.GaugeVec
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		EventCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "event_count",
			Help:      "Number of events with the given action observed in the trailing window. Sourced from the /events/events/volume/ endpoint which buckets at 6h granularity; windows shorter than 6h represent the most recent 6h bucket.",
		}, []string{"action", "window"}),
		SignupsVerified: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "signups_verified",
			Help:      "Users created in the trailing window with is_active=true (email verified).",
		}, []string{"window"}),
		SignupsUnverified: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "signups_unverified",
			Help:      "Users created in the trailing window with is_active=false (email not yet verified).",
		}, []string{"window"}),
		SignupsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "signups_total",
			Help:      "Sum of verified and unverified signups in the trailing window.",
		}, []string{"window"}),
		ExporterUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "up",
			Help:      "1 if the last poll cycle completed without errors, 0 otherwise.",
		}),
		ScrapeDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "Duration of the last poll for the given target.",
		}, []string{"target"}),
		ScrapeErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrape_errors_total",
			Help:      "Total number of poll errors per target.",
		}, []string{"target"}),
		LastSuccessSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "last_success_timestamp_seconds",
			Help:      "Unix timestamp of the last successful poll for the given target.",
		}, []string{"target"}),
	}
	reg.MustRegister(
		m.EventCount,
		m.SignupsVerified,
		m.SignupsUnverified,
		m.SignupsTotal,
		m.ExporterUp,
		m.ScrapeDuration,
		m.ScrapeErrorsTotal,
		m.LastSuccessSeconds,
	)
	return m
}
