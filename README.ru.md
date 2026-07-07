# metrics-collector

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
[![CI](https://github.com/ezhigval/metrics-collector/actions/workflows/ci.yml/badge.svg)](https://github.com/ezhigval/metrics-collector/actions/workflows/ci.yml)
![License](https://img.shields.io/badge/license-MIT-blue)
![Tier](https://img.shields.io/badge/tier-middle-5319e7)

**[English](README.md)** · Русский

Лёгкий приём метрик + экспозиция Prometheus + OTel-трейсы. PG хранит time-series для дашбордов; Grafana в compose.

## Быстрый старт

```bash
make docker-up
make ingest

curl -s localhost:8089/metrics | grep metrics_ingested
curl -s "localhost:8089/api/v1/metrics/cpu_usage" | jq
```

**Grafana:** http://localhost:3000 (admin/admin)  
**Prometheus:** http://localhost:9094

## API

| Метод | Путь | Описание |
|--------|------|----------|
| POST | `/api/v1/ingest` | пакетная загрузка метрик |
| GET | `/api/v1/metrics` | список имён метрик |
| GET | `/api/v1/metrics/{name}?from=&to=&limit=` | запрос time series |
| GET | `/metrics` | endpoint для scrape Prometheus |

## Пример ingest

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

## Observability-стек

```
Apps ──POST /ingest──► metrics-collector ──► PostgreSQL (history)
                              │
                         /metrics ◄── Prometheus ◄── Grafana
                              │
                         OTel traces (stdout)
```

Метрики Prometheus: `http_requests_total`, `metrics_ingested_total`, `metrics_ingest_duration_seconds`, `custom_metric_value`.

## Решения

- **PG для истории** — не полноценный TSDB; хватает для API дашборда в портфолио без накладных VictoriaMetrics.
- **Stdout OTel** — демо трейсинга без зависимостей; в проде — экспортёр в Jaeger.
- **Два пути** — ingest пишет и в PG, и в gauges Prometheus: live + история.

## Стек

Go 1.25 · chi · PostgreSQL · Prometheus · OpenTelemetry · Grafana · [go-toolkit](https://github.com/ezhigval/go-toolkit)

Порт **8089** · MIT
