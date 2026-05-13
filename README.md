# authentik-exporter

A Prometheus exporter that turns authentik's Events and Users APIs into event-derived
business metrics. It **complements** the built-in `:9300` metrics endpoint (which exposes
process-level counters) by surfacing data that lives in the authentik database — most
notably the **verified vs. unverified signup** breakdown.

## What it exposes

| Metric | Type | Labels | Source |
|---|---|---|---|
| `authentik_event_count` | gauge | `action`, `window` | `/events/events/volume/` summed over 6 h buckets falling inside `window` |
| `authentik_signups_verified` | gauge | `window` | `/core/users/?is_active=true&date_joined__gt=…&page_size=1` |
| `authentik_signups_unverified` | gauge | `window` | `/core/users/?is_active=false&date_joined__gt=…&page_size=1` |
| `authentik_signups_total` | gauge | `window` | sum of the two above |
| `authentik_exporter_up` | gauge | — | 1 if the last poll cycle was clean |
| `authentik_exporter_scrape_duration_seconds` | gauge | `target` | seconds spent in the last poll |
| `authentik_exporter_scrape_errors_total` | counter | `target` | poll failure count |
| `authentik_exporter_last_success_timestamp_seconds` | gauge | `target` | Unix time of last clean poll |

`target` is `events` or `signups`.

### Caveat: 6 h bucket granularity

authentik's `volume` endpoint aggregates events into **six-hour buckets**. For windows
shorter than 6 h (e.g. `5m`, `1h`) the exporter reports the count from the most recent
bucket up to "now" — accurate for trend lines and rate-of-change, but **not** a precise
sliding window. Sub-hour precision needs the event-tailing extension (see *Future work*).

## Quick start

```sh
export AUTHENTIK_URL=https://authentik.example.com
export AUTHENTIK_TOKEN=ak-xxxxxxxxxxxxxxxxxxxxxxxxxxxx   # Admin > Tokens & App passwords, intent=API
./authentik-exporter
curl localhost:9119/metrics | grep authentik_
```

### Docker

```sh
docker run --rm -p 9119:9119 \
  -e AUTHENTIK_URL=https://authentik.example.com \
  -e AUTHENTIK_TOKEN=ak-xxx \
  authentik-exporter:dev
```

### Prometheus scrape config

```yaml
scrape_configs:
  - job_name: authentik-events
    static_configs:
      - targets: ['authentik-exporter:9119']
```

## Configuration

All settings are environment variables. Required fields are marked.

| Variable | Default | Purpose |
|---|---|---|
| `AUTHENTIK_URL` | **required** | Base URL, e.g. `https://authentik.example.com` |
| `AUTHENTIK_TOKEN` | **required** | API token (intent = API) |
| `AUTHENTIK_INSECURE_SKIP_VERIFY` | `false` | Skip TLS verification (dev only) |
| `EXPORTER_LISTEN_ADDR` | `:9119` | Bind address |
| `EXPORTER_METRICS_PATH` | `/metrics` | Metrics endpoint path |
| `EXPORTER_POLL_INTERVAL` | `60s` | Background poll cadence |
| `EXPORTER_WINDOWS` | `5m,1h,24h,7d` | Comma-separated durations; supports `d` and `w` |
| `EXPORTER_ACTIONS` | `login,login_failed,logout,password_set,suspicious_request,authorize_application` | Event actions to track |
| `EXPORTER_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |

Available event action names match authentik's `EventActions` enum: `login`, `login_failed`,
`logout`, `user_write`, `suspicious_request`, `password_set`, `secret_view`, `secret_rotate`,
`invitation_used`, `authorize_application`, `source_linked`, `impersonation_started`,
`impersonation_ended`, `flow_execution`, `policy_execution`, `policy_exception`,
`property_mapping_exception`, `system_task_execution`, `system_task_exception`,
`system_exception`, `configuration_error`, `configuration_warning`, `model_created`,
`model_updated`, `model_deleted`, `email_sent`, `update_available`.

## Example alerts

```yaml
groups:
  - name: authentik-signups
    rules:
      - alert: AuthentikStuckUnverifiedSignups
        expr: authentik_signups_unverified{window="24h"} > 5
        for: 1h
        annotations:
          summary: "Unverified signups are accumulating in the last 24h"
          description: "{{ $value }} users created in the last 24h haven't verified their email."

      - alert: AuthentikLoginFailureSpike
        expr: |
          rate(authentik_event_count{action="login_failed",window="1h"}[15m]) > 10
        for: 10m
        annotations:
          summary: "Login failure rate spike on authentik"

      - alert: AuthentikExporterDown
        expr: authentik_exporter_up == 0
        for: 5m
        annotations:
          summary: "authentik-exporter polling is failing"
```

## Build

```sh
make build      # bin/authentik-exporter
make test       # go test ./...
make docker     # builds and loads authentik-exporter:dev
```

Requires Go 1.25+. The Docker build is fully static (`CGO_ENABLED=0`) and runs on
`gcr.io/distroless/static-debian12:nonroot` (~12 MB).

## How it works

```
┌─ background poller (every EXPORTER_POLL_INTERVAL) ─┐
│                                                    │
│   for each action in EXPORTER_ACTIONS:             │
│       GET /events/events/volume/?action=…&         │
│           history_days=ceil(max(windows)/24h)      │
│       for each window: sum buckets newer than      │
│           now-window → set GaugeVec(action,window) │
│                                                    │
│   for each window in EXPORTER_WINDOWS:             │
│       GET /core/users/?is_active=true&             │
│           date_joined__gt=now-window&page_size=1   │
│       GET /core/users/?is_active=false& …          │
│       → set signup gauges                          │
└─────────────────────────────────────────────────────┘

┌─ HTTP server (promhttp) ─┐
│ /metrics → registry      │ ← scraped by Prometheus
└──────────────────────────┘
```

Signup tracking assumes the **standard default-enrollment flow**: users are created with
`is_active=false`, and `is_active` flips to `true` when they complete email verification.
If your deployment uses a different model (verification flag in `attributes`, always-active
users, etc.), the verified/unverified split won't match your reality — adjust the source
or open an issue.

## Future work

- **Event tailing**: maintain a per-action cursor on `created` timestamps, fetch new events
  via `/events/events/?ordering=-created&created__gt=…`, dedupe by `event_uuid`, and feed a
  `CounterVec`. Gives true monotonic counters with full `rate()`/`increase()` semantics and
  sub-hour resolution. Trade-off: more API load (one paginated call per action per poll)
  and an in-process dedupe set.
- **Brand label**: split metrics by `brand_name` for multi-brand deployments.
- **TLS / basic auth on `/metrics`**: wire `prometheus/exporter-toolkit/web` if exposing
  publicly.
