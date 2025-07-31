package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"metrics-app/internal/domain"
)

func TestSQLiteStore_Init(t *testing.T) {

	testDBPath := "./test_metrics_init.db"

	os.Remove(testDBPath)
	defer os.Remove(testDBPath)

	store := NewSQLiteStore(testDBPath)
	err := store.Init()
	assert.NoError(t, err, "Init should not return an error")

	store.Close()
}

func TestSQLiteStore_StoreMetric(t *testing.T) {
	testDBPath := "./test_metrics_store.db"
	os.Remove(testDBPath)
	defer os.Remove(testDBPath)

	sqliteStore := NewSQLiteStore(testDBPath)
	sqliteStore.Init()
	defer sqliteStore.Close()

	metric := domain.Metric{
		Timestamp:   time.Now().Unix(),
		CPULoad:     75.0,
		Concurrency: 250000,
	}

	ctx := context.Background()
	err := sqliteStore.StoreMetric(ctx, metric)
	assert.NoError(t, err, "StoreMetric should not return an error")

	retrievedMetrics, err := sqliteStore.GetMetrics(ctx, metric.Timestamp, metric.Timestamp, 0, 0)
	assert.NoError(t, err)
	assert.Len(t, retrievedMetrics, 1, "Should find the stored metric")
	assert.Equal(t, metric, retrievedMetrics[0], "Retrieved metric should match stored metric")
}

func TestSQLiteStore_GetMetrics(t *testing.T) {
	testDBPath := "./test_metrics_get.db"
	os.Remove(testDBPath)
	defer os.Remove(testDBPath)

	sqliteStore := NewSQLiteStore(testDBPath)
	sqliteStore.Init()
	defer sqliteStore.Close()

	now := time.Now().Unix()

	metricsToStore := []domain.Metric{
		{Timestamp: now - 50, CPULoad: 10.0, Concurrency: 100},
		{Timestamp: now - 40, CPULoad: 20.0, Concurrency: 200},
		{Timestamp: now - 30, CPULoad: 30.0, Concurrency: 300},
		{Timestamp: now - 20, CPULoad: 40.0, Concurrency: 400},
		{Timestamp: now - 10, CPULoad: 50.0, Concurrency: 500},
		{Timestamp: now, CPULoad: 60.0, Concurrency: 600},
	}

	ctx := context.Background()
	for _, m := range metricsToStore {
		err := sqliteStore.StoreMetric(ctx, m)
		assert.NoError(t, err)
	}

	// case 1: Full range (no limit/offset)
	retrievedMetrics, err := sqliteStore.GetMetrics(ctx, now-100, now+100, 0, 0)
	assert.NoError(t, err, "GetMetrics should not return an error for full range")
	assert.Len(t, retrievedMetrics, 6, "Should retrieve all 6 metrics")
	assert.Equal(t, metricsToStore, retrievedMetrics, "Retrieved metrics should match stored metrics for full range")

	// case 2: Partial range (no limit/offset)
	retrievedMetrics, err = sqliteStore.GetMetrics(ctx, now-45, now-5, 0, 0)
	assert.NoError(t, err, "GetMetrics should not return an error for partial range")
	assert.Len(t, retrievedMetrics, 4, "Should retrieve 4 metrics for partial range")
	assert.Equal(t, metricsToStore[1:5], retrievedMetrics, "Retrieved metrics should match expected for partial range")

	// case 3: No metrics in range (no limit/offset)
	retrievedMetrics, err = sqliteStore.GetMetrics(ctx, now+10, now+20, 0, 0)
	assert.NoError(t, err, "GetMetrics should not return an error for empty range")
	assert.Len(t, retrievedMetrics, 0, "Should retrieve 0 metrics for empty range")

	// case 4: Context cancellation during query
	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cancel()
	retrievedMetrics, err = sqliteStore.GetMetrics(ctxWithCancel, now-100, now+100, 0, 0)
	assert.Error(t, err, "GetMetrics should return an error when context is cancelled")
	assert.Contains(t, err.Error(), "context canceled", "Error should indicate context cancellation")
	assert.Len(t, retrievedMetrics, 0, "Should retrieve 0 metrics on cancellation")

	// case 5: Limit 2, Offset 0
	retrievedMetrics, err = sqliteStore.GetMetrics(ctx, now-100, now+100, 2, 0)
	assert.NoError(t, err)
	assert.Len(t, retrievedMetrics, 2)
	assert.Equal(t, metricsToStore[0:2], retrievedMetrics, "Should get first 2 metrics")

	// case 6: Limit 2, Offset 2
	retrievedMetrics, err = sqliteStore.GetMetrics(ctx, now-100, now+100, 2, 2)
	assert.NoError(t, err)
	assert.Len(t, retrievedMetrics, 2)
	assert.Equal(t, metricsToStore[2:4], retrievedMetrics, "Should get metrics from index 2 to 3")

	// case 7: Limit 3, Offset 3 (remaining less than limit)
	retrievedMetrics, err = sqliteStore.GetMetrics(ctx, now-100, now+100, 3, 3)
	assert.NoError(t, err)
	assert.Len(t, retrievedMetrics, 3)
	assert.Equal(t, metricsToStore[3:6], retrievedMetrics, "Should get remaining 3 metrics from index 3")

	// case 8: Offset beyond available data
	retrievedMetrics, err = sqliteStore.GetMetrics(ctx, now-100, now+100, 2, 10)
	assert.NoError(t, err)
	assert.Len(t, retrievedMetrics, 0, "Should get 0 metrics if offset is beyond data")

	// case 9: Limit 0 (should return all filtered)
	retrievedMetrics, err = sqliteStore.GetMetrics(ctx, now-100, now+100, 0, 0)
	assert.NoError(t, err)
	assert.Len(t, retrievedMetrics, 6, "Limit 0 should return all filtered metrics")
	assert.Equal(t, metricsToStore, retrievedMetrics)

	// case 10: Negative offset (should be treated as 0)
	retrievedMetrics, err = sqliteStore.GetMetrics(ctx, now-100, now+100, 2, -5)
	assert.NoError(t, err)
	assert.Len(t, retrievedMetrics, 2, "Negative offset should be treated as 0")
	assert.Equal(t, metricsToStore[0:2], retrievedMetrics)
}
