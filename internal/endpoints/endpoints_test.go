package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"metrics-app/internal/domain"
	"metrics-app/internal/util"
)

type MockMetricStore struct {
	Metrics []domain.Metric
	Err     error
}

func (m *MockMetricStore) Init() error {
	return m.Err
}

func (m *MockMetricStore) StoreMetric(ctx context.Context, metric domain.Metric) error {
	if m.Err != nil {
		return m.Err
	}
	m.Metrics = append(m.Metrics, metric)
	return nil
}

func (m *MockMetricStore) GetMetrics(ctx context.Context, startTime, endTime int64, limit, offset int) ([]domain.Metric, error) {
	if m.Err != nil {
		return nil, m.Err
	}

	var filtered []domain.Metric
	for _, metric := range m.Metrics {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if metric.Timestamp >= startTime && metric.Timestamp <= endTime {
			filtered = append(filtered, metric)
		}
	}

	if offset >= len(filtered) {
		return []domain.Metric{}, nil
	}
	if offset > 0 {
		filtered = filtered[offset:]
	}

	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

func (m *MockMetricStore) Close() error {
	return m.Err
}

func TestGetMetricsHandler(t *testing.T) {
	mockStore := &MockMetricStore{
		Metrics: make([]domain.Metric, 0),
	}
	mockStore.Init()

	now := time.Now().Unix()

	for i := 0; i < 10; i++ {
		mockStore.StoreMetric(context.Background(), domain.Metric{Timestamp: now - int64(9-i)*10, CPULoad: float64(i * 10), Concurrency: i * 100})
	}

	metricsHandler := &Metrics{}
	metricsHandler.Init(mockStore, &util.MetricsLogger{})

	requestBody := MetricsRequest{
		Start: now - 100,
		End:   now + 10,
	}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("GET", "/metrics/100/0", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	assert.NoError(t, err)

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "100",
		"offset": "0",
	})

	rr := httptest.NewRecorder()

	metricsHandler.GetMetricsHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Expected status OK")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "Expected Content-Type: application/json")

	var apiResponse APIResponse
	body, err := ioutil.ReadAll(rr.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(body, &apiResponse)
	assert.NoError(t, err)

	assert.True(t, apiResponse.Status, "Expected API status to be true for success")
	assert.Equal(t, API_SUCCESS, apiResponse.ErrorCode, "Expected API_SUCCESS error code")
	assert.Empty(t, apiResponse.Error, "Expected no error message on success")

	var returnedMetrics []domain.Metric

	valueBytes, _ := json.Marshal(apiResponse.Value)
	json.Unmarshal(valueBytes, &returnedMetrics)

	assert.Len(t, returnedMetrics, 10, "Expected 10 metrics (all available) in the response")

	// case 2: Invalid JSON body (using GET)
	req, _ = http.NewRequest("GET", "/metrics/100/0", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "100",
		"offset": "0",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected Bad Request for invalid JSON body")

	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.False(t, apiResponse.Status, "Expected API status to be false for error")
	assert.Equal(t, INVALID_REQUEST_BODY, apiResponse.ErrorCode, "Expected INVALID_REQUEST_BODY error code")
	assert.Contains(t, apiResponse.Error, ErrInvalidRequestBody.Error(), "Expected specific error message for invalid JSON")

	// case 3: start > end timestamp in JSON (using GET)
	invalidRequestBody := MetricsRequest{
		Start: now + 100,
		End:   now,
	}
	invalidJsonBody, _ := json.Marshal(invalidRequestBody)
	req, _ = http.NewRequest("GET", "/metrics/100/0", bytes.NewBuffer(invalidJsonBody))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "100",
		"offset": "0",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected Bad Request for start > end")

	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.False(t, apiResponse.Status, "Expected API status to be false for error")
	assert.Equal(t, INVALID_TIME_RANGE, apiResponse.ErrorCode, "Expected INVALID_TIME_RANGE error code")
	assert.Contains(t, apiResponse.Error, ErrInvalidTimeRange.Error(), "Expected specific error message for start > end")

	// case 4: Context cancellation during GetMetrics (using GET)
	mockStoreWithCancel := &MockMetricStore{
		Metrics: []domain.Metric{
			{Timestamp: now - 10, CPULoad: 1.0, Concurrency: 1},
		},
		Err: context.Canceled,
	}
	mockStoreWithCancel.Init()

	reqBodyCancel := MetricsRequest{
		Start: now - 20,
		End:   now + 20,
	}
	jsonBodyCancel, _ := json.Marshal(reqBodyCancel)
	req, _ = http.NewRequest("GET", "/metrics/10/0", bytes.NewBuffer(jsonBodyCancel))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "10",
		"offset": "0",
	})

	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)

	rr = httptest.NewRecorder()

	metricsHandlerWithCancel := &Metrics{}
	metricsHandlerWithCancel.Init(mockStoreWithCancel, &util.MetricsLogger{})
	metricsHandlerWithCancel.GetMetricsHandler(rr, req)

	assert.Equal(t, http.StatusRequestTimeout, rr.Code, "Expected Request Timeout for cancelled context")
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.False(t, apiResponse.Status, "Expected API status to be false for error")
	assert.Equal(t, REQUEST_CANCELLED, apiResponse.ErrorCode, "Expected REQUEST_CANCELLED error code")
	assert.Contains(t, apiResponse.Error, ErrRequestCancelled.Error(), "Expected specific error message for cancellation")

	// case 5: POST request (should be rejected by handler's method check, as it now expects GET)
	req, _ = http.NewRequest("POST", "/metrics/10/0", nil) // Changed to POST to test method rejection

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "10",
		"offset": "0",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code, "Expected Method Not Allowed for GET request")

	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.False(t, apiResponse.Status, "Expected API status to be false for error")
	assert.Equal(t, API_FAILURE, apiResponse.ErrorCode, "Expected API_FAILURE error code for generic method not allowed")
	assert.Contains(t, apiResponse.Error, "method Not Allowed. Only GET requests are supported", "Expected specific error message for wrong method")

	// case 6: Limit 5, Offset 0
	paginationBody1 := MetricsRequest{
		Start: now - 100,
		End:   now + 10,
	}
	jsonBody1, _ := json.Marshal(paginationBody1)
	req, _ = http.NewRequest("GET", "/metrics/5/0", bytes.NewBuffer(jsonBody1))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "5",
		"offset": "0",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.True(t, apiResponse.Status)
	var pagedMetrics1 []domain.Metric
	valueBytes1, _ := json.Marshal(apiResponse.Value)
	json.Unmarshal(valueBytes1, &pagedMetrics1)
	assert.Len(t, pagedMetrics1, 5, "Expected 5 metrics for limit 5, offset 0")
	assert.Equal(t, mockStore.Metrics[0].Timestamp, pagedMetrics1[0].Timestamp, "First metric should be at offset 0")

	// case 7: Limit 5, Offset 5
	paginationBody2 := MetricsRequest{
		Start: now - 100,
		End:   now + 10,
	}
	jsonBody2, _ := json.Marshal(paginationBody2)
	req, _ = http.NewRequest("GET", "/metrics/5/5", bytes.NewBuffer(jsonBody2))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "5",
		"offset": "5",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.True(t, apiResponse.Status)
	var pagedMetrics2 []domain.Metric
	valueBytes2, _ := json.Marshal(apiResponse.Value)
	json.Unmarshal(valueBytes2, &pagedMetrics2)
	assert.Len(t, pagedMetrics2, 5, "Expected 5 metrics for limit 5, offset 5")
	assert.Equal(t, mockStore.Metrics[5].Timestamp, pagedMetrics2[0].Timestamp, "First metric should be at offset 5")

	// case 8: Limit 5, Offset 8 (fewer than limit available)
	paginationBody3 := MetricsRequest{
		Start: now - 100,
		End:   now + 10,
	}
	jsonBody3, _ := json.Marshal(paginationBody3)
	req, _ = http.NewRequest("GET", "/metrics/5/8", bytes.NewBuffer(jsonBody3))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "5",
		"offset": "8",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.True(t, apiResponse.Status)

	var pagedMetrics3 []domain.Metric
	valueBytes3, _ := json.Marshal(apiResponse.Value)
	json.Unmarshal(valueBytes3, &pagedMetrics3)
	assert.Len(t, pagedMetrics3, 2, "Expected 2 metrics for limit 5, offset 8")
	assert.Equal(t, mockStore.Metrics[8].Timestamp, pagedMetrics3[0].Timestamp, "First metric should be at offset 8")

	// case 9: Offset beyond total available data (should return no metrics, but status OK)
	paginationBody4 := MetricsRequest{
		Start: now - 100,
		End:   now + 10,
	}
	jsonBody4, _ := json.Marshal(paginationBody4)
	req, _ = http.NewRequest("GET", "/metrics/5/100", bytes.NewBuffer(jsonBody4))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "5",
		"offset": "100",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.False(t, apiResponse.Status)
	assert.Equal(t, METRICS_NOT_AVAILABLE, apiResponse.ErrorCode)
	assert.Contains(t, apiResponse.Error, ErrNoMetricsAvailable.Error(), "Expected specific error message for no metrics")

	// case 10: Limit 0 (should default to 100)
	paginationBody5 := MetricsRequest{
		Start: now - 100,
		End:   now + 10,
	}
	jsonBody5, _ := json.Marshal(paginationBody5)
	req, _ = http.NewRequest("GET", "/metrics/0/0", bytes.NewBuffer(jsonBody5))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "0",
		"offset": "0",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.True(t, apiResponse.Status)
	var pagedMetrics5 []domain.Metric
	valueBytes5, _ := json.Marshal(apiResponse.Value)
	json.Unmarshal(valueBytes5, &pagedMetrics5)
	assert.Len(t, pagedMetrics5, 10, "Expected 10 metrics for limit 0 (default 100)")

	// case 11: Negative offset (should default to 0)
	paginationBody6 := MetricsRequest{
		Start: now - 100,
		End:   now + 10,
	}
	jsonBody6, _ := json.Marshal(paginationBody6)
	req, _ = http.NewRequest("GET", "/metrics/5/-5", bytes.NewBuffer(jsonBody6))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "5",
		"offset": "-5",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.True(t, apiResponse.Status)
	var pagedMetrics6 []domain.Metric
	valueBytes6, _ := json.Marshal(apiResponse.Value)
	json.Unmarshal(valueBytes6, &pagedMetrics6)
	assert.Len(t, pagedMetrics6, 5, "Expected 5 metrics for negative offset (default 0)")
	assert.Equal(t, mockStore.Metrics[0].Timestamp, pagedMetrics6[0].Timestamp, "First metric should be at offset 0")

	// case 12: Invalid limit parameter (non-integer)
	req, _ = http.NewRequest("GET", "/metrics/abc/0", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "abc",
		"offset": "0",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected Bad Request for invalid limit parameter")
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.False(t, apiResponse.Status)
	assert.Equal(t, INVALID_PARAMETERS, apiResponse.ErrorCode)
	assert.Contains(t, apiResponse.Error, ErrInvalidParameters.Error(), "Expected error for invalid limit")

	// case 13: Invalid offset parameter (non-integer)
	req, _ = http.NewRequest("GET", "/metrics/10/xyz", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "10",
		"offset": "xyz",
	})
	rr = httptest.NewRecorder()
	metricsHandler.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected Bad Request for invalid offset parameter")
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.False(t, apiResponse.Status)
	assert.Equal(t, INVALID_PARAMETERS, apiResponse.ErrorCode)
	assert.Contains(t, apiResponse.Error, ErrInvalidParameters.Error(), "Expected error for invalid offset")

	// case 14: No metrics found for a valid query (should return 404 with specific error code)
	mockStoreNoMetrics := &MockMetricStore{
		Metrics: make([]domain.Metric, 0),
	}
	mockStoreNoMetrics.Init()
	metricsHandlerNoMetrics := &Metrics{}
	metricsHandlerNoMetrics.Init(mockStoreNoMetrics, &util.MetricsLogger{})

	reqBodyNoMetrics := MetricsRequest{
		Start: now - 100,
		End:   now + 10,
	}
	jsonBodyNoMetrics, _ := json.Marshal(reqBodyNoMetrics)
	req, _ = http.NewRequest("GET", "/metrics/10/0", bytes.NewBuffer(jsonBodyNoMetrics))
	req.Header.Set("Content-Type", "application/json")

	req = mux.SetURLVars(req, map[string]string{
		"limit":  "10",
		"offset": "0",
	})
	rr = httptest.NewRecorder()
	metricsHandlerNoMetrics.GetMetricsHandler(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code, "Expected Not Found for no metrics available")
	json.Unmarshal(rr.Body.Bytes(), &apiResponse)
	assert.False(t, apiResponse.Status)
	assert.Equal(t, METRICS_NOT_AVAILABLE, apiResponse.ErrorCode)
	assert.Contains(t, apiResponse.Error, ErrNoMetricsAvailable.Error(), "Expected specific error message for no metrics")
}
