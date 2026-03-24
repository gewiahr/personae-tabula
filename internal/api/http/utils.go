package apiHttp

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func (h *APIHandler) sendJSON(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode response: %v", err)
		// Не можем отправить ошибку, так как статус уже отправлен
	}
}

func (h *APIHandler) newApiResponse(payload any, code int) *APIResponse {
	return &APIResponse{
		Payload: payload,
		Error:   nil,
		Code:    code,
	}
}

func (h *APIHandler) newErrorResponse(err string, message string, code int) *APIResponse {
	return &APIResponse{
		Payload: nil,
		Error: &APIError{
			Error:   err,
			Message: message,
		},
		Code: code,
	}
}

func (h *APIHandler) newError(err string, message string) *APIError {
	return &APIError{
		Error:   err,
		Message: message,
	}
}

func (h *APIHandler) getQueryInt(r *http.Request, key string, defaultValue int) int {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	if value < 0 {
		return defaultValue
	}

	return value
}
