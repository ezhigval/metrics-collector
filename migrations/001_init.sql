-- +goose Up
CREATE TABLE IF NOT EXISTS metric_points (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    labels      JSONB NOT NULL DEFAULT '{}',
    value       DOUBLE PRECISION NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_metric_points_name_time ON metric_points(name, recorded_at DESC);
CREATE INDEX idx_metric_points_labels ON metric_points USING gin(labels);

-- +goose Down
DROP TABLE IF EXISTS metric_points;
