package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/ezhigval/metrics-collector/internal/model"
	promreg "github.com/ezhigval/metrics-collector/internal/prom"
	"github.com/ezhigval/metrics-collector/internal/repository"
	"github.com/ezhigval/metrics-collector/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestIngestAndQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pg, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("metrics"),
		postgres.WithUsername("metrics"),
		postgres.WithPassword("metrics"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `
		CREATE TABLE metric_points (
			id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL, labels JSONB NOT NULL DEFAULT '{}',
			value DOUBLE PRECISION NOT NULL, recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	repo := repository.NewMetricRepository(pool)
	svc := service.NewMetricService(repo, promreg.NewRegistry())

	now := time.Now().UTC()
	n, err := svc.Ingest(ctx, model.IngestRequest{Metrics: []model.IngestPoint{
		{Name: "cpu_usage", Value: 55.0, At: &now},
	}})
	require.NoError(t, err)
	require.Equal(t, 1, n)

	series, err := svc.Query(ctx, "cpu_usage", now.Add(-time.Minute), now.Add(time.Minute), 10)
	require.NoError(t, err)
	require.Len(t, series.Points, 1)
	require.Equal(t, 55.0, series.Points[0].Value)
}
