package collector

import "github.com/prometheus/client_golang/prometheus"

const namespace = "authentik"

// State label values used by the Signups and Users metrics.
const (
	StateVerified   = "verified"
	StateUnverified = "unverified"
)

type Metrics struct {
	EventCount         *prometheus.GaugeVec
	Signups            *prometheus.GaugeVec
	Users              *prometheus.GaugeVec
	GroupMembers       *prometheus.GaugeVec
	Groups             prometheus.Gauge
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
		Signups: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "signups",
			Help:      "Users created in the trailing window, partitioned by verification state. state=verified means is_active=true (email confirmed); state=unverified means is_active=false. Sum over state for the total.",
		}, []string{"window", "state"}),
		Users: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "users",
			Help:      "Total users in authentik, partitioned by verification state. state=verified means is_active=true; state=unverified means is_active=false. Sum over state for the grand total.",
		}, []string{"state"}),
		GroupMembers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "group_members",
			Help:      "Number of users belonging to each group. Note: a user can be a member of multiple groups, so summing across groups does not equal the total user count.",
		}, []string{"group"}),
		Groups: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "groups",
			Help:      "Total number of groups in authentik.",
		}),
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
		m.Signups,
		m.Users,
		m.GroupMembers,
		m.Groups,
		m.ExporterUp,
		m.ScrapeDuration,
		m.ScrapeErrorsTotal,
		m.LastSuccessSeconds,
	)
	return m
}
