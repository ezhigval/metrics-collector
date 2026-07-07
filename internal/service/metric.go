package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ezhigval/metrics-collector/internal/model"
	promreg "github.com/ezhigval/metrics-collector/internal/prom"
	"github.com/ezhigval/metrics-collector/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrEmptyBatch  = errors.New("metrics batch is empty")
	ErrInvalidName = errors.New("metric name required")
)

type MetricService struct {
	repo   *repository.MetricRepository
	prom   *promreg.Registry
	tracer trace.Tracer
}

func NewMetricService(repo *repository.MetricRepository, prom *promreg.Registry) *MetricService {
	return &MetricService{
		repo:   repo,
		prom:   prom,
		tracer: otel.Tracer("metrics-collector/service"),
	}
}

func (s *MetricService) Ingest(ctx context.Context, req model.IngestRequest) (int, error) {
	ctx, span := s.tracer.Start(ctx, "MetricService.Ingest")
	defer span.End()

	if len(req.Metrics) == 0 {
		return 0, ErrEmptyBatch
	}

	valid := make([]model.IngestPoint, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		m.Name = strings.TrimSpace(m.Name)
		if m.Name == "" {
			return 0, ErrInvalidName
		}
		valid = append(valid, m)
	}

	span.SetAttributes(attribute.Int("batch_size", len(valid)))

	timer := time.Now()
	if err := s.repo.InsertBatch(ctx, valid); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	s.prom.IngestDuration.Observe(time.Since(timer).Seconds())

	for _, m := range valid {
		s.prom.IngestTotal.Inc()
		service := m.Labels["service"]
		if service == "" {
			service = "unknown"
		}
		s.prom.RecordCustom(m.Name, service, m.Value)
	}

	return len(valid), nil
}

func (s *MetricService) Query(ctx context.Context, name string, from, to time.Time, limit int) (*model.SeriesResponse, error) {
	ctx, span := s.tracer.Start(ctx, "MetricService.Query")
	defer span.End()

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}
	span.SetAttributes(attribute.String("metric.name", name))

	points, err := s.repo.QuerySeries(ctx, name, from, to, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if points == nil {
		points = []model.DataPoint{}
	}
	return &model.SeriesResponse{Name: name, Points: points}, nil
}

func (s *MetricService) ListNames(ctx context.Context) ([]string, error) {
	names, err := s.repo.ListNames(ctx)
	if err != nil {
		return nil, err
	}
	if names == nil {
		return []string{}, nil
	}
	return names, nil
}
