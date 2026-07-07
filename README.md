# metrics-collector

Lightweight metrics ingestion + Prometheus exposition + OTel traces. PG stores time-series for dashboard queries; Grafana included in compose.

## Quick start

```bash
make docker-up
make ingest

curl -s localhost:8089/metrics | grep metrics_ingested
curl -s "localhost:8089/api/v1/metrics/cpu_usage" | jq
```

**Grafana:** http://localhost:3000 (admin/admin)  
**Prometheus:** http://localhost:9094

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/ingest` | batch ingest custom metrics |
| GET | `/api/v1/metrics` | list metric names |
| GET | `/api/v1/metrics/{name}?from=&to=&limit=` | time series query |
| GET | `/metrics` | Prometheus scrape endpoint |

## Ingest example

```json
{
  "metrics": [
    {
      "name": "cpu_usage",
      "value": 72.1,
      "labels": {"service": "api", "host": "node-1"}
    }
  ]
}
```

## Observability stack

```
Apps ──POST /ingest──► metrics-collector ──► PostgreSQL (history)
                              │
                         /metrics ◄── Prometheus ◄── Grafana
                              │
                         OTel traces (stdout)
```

Prometheus metrics: `http_requests_total`, `metrics_ingested_total`, `metrics_ingest_duration_seconds`, `custom_metric_value`.

## Decisions

- **PG for history** — not a full TSDB; enough for portfolio dashboard API without VictoriaMetrics overhead.
- **Stdout OTel** — zero-deps tracing demo; swap exporter for Jaeger in prod.
- **Dual path** — ingest updates both PG and Prometheus gauges for live + historical views.

## Stack

Go 1.25 · chi · PostgreSQL · Prometheus · OpenTelemetry · Grafana · [go-toolkit](https://github.com/ezhigval/go-toolkit)

Port **8089** · MIT
