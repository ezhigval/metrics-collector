package model

import "time"

type IngestPoint struct {
	Name   string            `json:"name"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels,omitempty"`
	At     *time.Time        `json:"at,omitempty"`
}

type IngestRequest struct {
	Metrics []IngestPoint `json:"metrics"`
}

type DataPoint struct {
	Value float64   `json:"value"`
	At    time.Time `json:"at"`
}

type SeriesResponse struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
	Points []DataPoint       `json:"points"`
}
