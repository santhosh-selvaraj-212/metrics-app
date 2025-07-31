package endpoints

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Status    bool        `json:"status"`
	Value     interface{} `json:"value,omitempty"`
	Error     string      `json:"error,omitempty"`
	ErrorCode int         `json:"error_code"`
}

func (res APIResponse) WriteErrorResponse(w http.ResponseWriter, err error) {
	res.Status = false
	res.Error = err.Error()
	res.ErrorCode = GetErrorCode(err)

	errJson, _ := json.Marshal(res)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(errJson)
}
func (res APIResponse) WriteErrorResponseWithStatusCode(w http.ResponseWriter, err error, StatusCode int) {
	res.Status = false
	res.Error = err.Error()
	if StatusCode == http.StatusUnauthorized {
		res.ErrorCode = API_UNAUTHORIZED
	} else {
		res.ErrorCode = GetErrorCode(err)
	}

	errJson, _ := json.Marshal(res)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(StatusCode)
	w.Write(errJson)
}

func (res APIResponse) WriteResultResponse(w http.ResponseWriter, result interface{}) {
	res.Status = true
	res.Value = result
	res.ErrorCode = GetErrorCode(nil)

	errJson, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(errJson)
}
