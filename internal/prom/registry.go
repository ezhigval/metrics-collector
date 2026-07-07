package prom

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	registry *prometheus.Registry

	HTTPRequests   *prometheus.CounterVec
	IngestTotal    prometheus.Counter
	IngestDuration prometheus.Histogram
	CustomGauges   *prometheus.GaugeVec
}

func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()
	r := &Registry{registry: reg}

	r.HTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	r.IngestTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "metrics_ingested_total",
		Help: "Total custom metric points ingested",
	})

	r.IngestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "metrics_ingest_duration_seconds",
		Help:    "Ingest handler duration",
		Buckets: prometheus.DefBuckets,
	})

	r.CustomGauges = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "custom_metric_value",
		Help: "Latest value of ingested custom metrics",
	}, []string{"name", "service"})

	reg.MustRegister(
		r.HTTPRequests,
		r.IngestTotal,
		r.IngestDuration,
		r.CustomGauges,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
	return r
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

func (r *Registry) RecordCustom(name, service string, value float64) {
	r.CustomGauges.WithLabelValues(name, service).Set(value)
}
