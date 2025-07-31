package domain

import "context"

type Metric struct {
	Timestamp   int64   `json:"timestamp"`
	CPULoad     float64 `json:"cpu_load"`
	Concurrency int     `json:"concurrency"`
}

type MetricStore interface {
	Init() error
	StoreMetric(ctx context.Context, metric Metric) error
	GetMetrics(ctx context.Context, startTime, endTime int64, limit, offset int) ([]Metric, error)
	Close() error
}
