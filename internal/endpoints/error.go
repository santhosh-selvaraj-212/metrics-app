package endpoints

import (
	"errors"
)

const (
	API_SUCCESS      = iota + 303000 // 303000
	API_FAILURE                      // 303001 - Generic API failure
	API_UNAUTHORIZED                 // 303002 - Authentication/Authorization failure
)

const (
	METRICS_NOT_AVAILABLE = iota + 101 // 101 - No metrics found for the given criteria
	INVALID_REQUEST_BODY               // 102 - Error parsing request body
	INVALID_PARAMETERS                 // 103 - Invalid URL parameters (e.g., non-integer limit/offset)
	INVALID_TIME_RANGE                 // 104 - Start time is after end time
	REQUEST_CANCELLED                  // 105 - Request was cancelled by client or server timeout
)

var (
	ErrNoMetricsAvailable = errors.New("no metrics available for the specified criteria")
	ErrInvalidRequestBody = errors.New("invalid request body format or missing fields")
	ErrInvalidParameters  = errors.New("invalid limit or offset parameter; must be integers")
	ErrInvalidTimeRange   = errors.New("start timestamp cannot be after end timestamp")
	ErrRequestCancelled   = errors.New("request cancelled by client or server timeout")
)

func GetErrorCode(err error) int {
	if err == nil {
		return API_SUCCESS
	}

	switch {
	case errors.Is(err, ErrNoMetricsAvailable):
		return METRICS_NOT_AVAILABLE
	case errors.Is(err, ErrInvalidRequestBody):
		return INVALID_REQUEST_BODY
	case errors.Is(err, ErrInvalidParameters):
		return INVALID_PARAMETERS
	case errors.Is(err, ErrInvalidTimeRange):
		return INVALID_TIME_RANGE
	case errors.Is(err, ErrRequestCancelled):
		return REQUEST_CANCELLED
	default:
		return API_FAILURE // Default for any unhandled error
	}
}
