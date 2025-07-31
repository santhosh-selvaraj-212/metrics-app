package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"metrics-app/internal/domain"
	"metrics-app/internal/util"

	"github.com/gorilla/mux"
)

type MetricsRequest struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type Metrics struct {
	Response APIResponse
	logger   *util.MetricsLogger
	store    domain.MetricStore
}

func (m *Metrics) Init(store domain.MetricStore, webSlogger *util.MetricsLogger) {
	m.store = store
	m.logger = webSlogger
}

func (m *Metrics) GetMetricsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		m.logger.LogEvent(util.LOG_LEVEL_ERROR, "Method Not Allowed. Only GET requests are supported", http.StatusMethodNotAllowed)
		m.Response.WriteErrorResponseWithStatusCode(w, errors.New("method Not Allowed. Only GET requests are supported"), http.StatusMethodNotAllowed)
		return
	}

	routeParamValue := mux.Vars(r)

	limitStr := routeParamValue["limit"]
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		m.logger.LogEvent(util.LOG_LEVEL_ERROR, "While getting limit from URL. Err - ", err)
		m.Response.WriteErrorResponseWithStatusCode(w, ErrInvalidParameters, http.StatusBadRequest)
		return
	}

	offsetStr := routeParamValue["offset"]
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		m.logger.LogEvent(util.LOG_LEVEL_ERROR, "While getting offset from URL. Err - ", err)
		m.Response.WriteErrorResponseWithStatusCode(w, ErrInvalidParameters, http.StatusBadRequest)
		return
	}

	var reqBody MetricsRequest

	err = json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		m.logger.LogEvent(util.LOG_LEVEL_ERROR, "Occured while unmarshalling JSON Body. Err -", err)
		m.Response.WriteErrorResponseWithStatusCode(w, ErrInvalidRequestBody, http.StatusBadRequest)
		return
	}

	startTime := reqBody.Start
	endTime := reqBody.End

	if startTime == 0 {
		startTime = time.Now().Add(-24 * time.Hour).Unix()
	}
	if endTime == 0 {
		endTime = time.Now().Unix()
	}

	if startTime > endTime {
		m.logger.LogEvent(util.LOG_LEVEL_ERROR, "Given startTime is greater than endTime. startTime - ", startTime, " endTime - ", endTime)
		m.Response.WriteErrorResponseWithStatusCode(w, ErrInvalidTimeRange, http.StatusBadRequest)
		return
	}

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	fetchedMetrics, err := m.store.GetMetrics(r.Context(), startTime, endTime, limit, offset)
	if err != nil {
		if err == context.Canceled {
			m.logger.LogEvent(util.LOG_LEVEL_WARN, "Context cancelled")
			m.Response.WriteErrorResponseWithStatusCode(w, ErrRequestCancelled, http.StatusRequestTimeout)
			return
		}
		m.logger.LogEvent(util.LOG_LEVEL_ERROR, "Occured while GetMetrics(). Err - ", err)
		m.Response.WriteErrorResponse(w, err)
		return
	}

	if len(fetchedMetrics) == 0 {
		m.logger.LogEvent(util.LOG_LEVEL_WARN, "Insufficient Metrics Data")
		m.Response.WriteErrorResponseWithStatusCode(w, ErrNoMetricsAvailable, http.StatusNotFound)
		return
	}

	m.Response.WriteResultResponse(w, fetchedMetrics)
}
