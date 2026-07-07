package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ezhigval/metrics-collector/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricRepository struct {
	pool *pgxpool.Pool
}

func NewMetricRepository(pool *pgxpool.Pool) *MetricRepository {
	return &MetricRepository{pool: pool}
}

func (r *MetricRepository) InsertBatch(ctx context.Context, points []model.IngestPoint) error {
	if len(points) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, p := range points {
		if err := insertOne(ctx, tx, p); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func insertOne(ctx context.Context, tx pgx.Tx, p model.IngestPoint) error {
	labels := p.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	lb, err := json.Marshal(labels)
	if err != nil {
		return err
	}
	at := time.Now().UTC()
	if p.At != nil {
		at = p.At.UTC()
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO metric_points (name, labels, value, recorded_at)
		VALUES ($1, $2, $3, $4)
	`, p.Name, lb, p.Value, at)
	if err != nil {
		return fmt.Errorf("insert metric: %w", err)
	}
	return nil
}

func (r *MetricRepository) QuerySeries(ctx context.Context, name string, from, to time.Time, limit int) ([]model.DataPoint, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		from = to.Add(-1 * time.Hour)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT value, recorded_at
		FROM metric_points
		WHERE name = $1 AND recorded_at >= $2 AND recorded_at <= $3
		ORDER BY recorded_at ASC
		LIMIT $4
	`, name, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query series: %w", err)
	}
	defer rows.Close()

	var points []model.DataPoint
	for rows.Next() {
		var dp model.DataPoint
		if err := rows.Scan(&dp.Value, &dp.At); err != nil {
			return nil, err
		}
		points = append(points, dp)
	}
	return points, rows.Err()
}

func (r *MetricRepository) ListNames(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT name FROM metric_points ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
