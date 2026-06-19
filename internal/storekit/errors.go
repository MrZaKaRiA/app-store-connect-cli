package storekit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError describes an error returned by the Retention Messaging API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RetryAfter time.Time
	Body       string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("StoreKit API HTTP %d", e.StatusCode)}
	if e.Code != "" {
		parts = append(parts, e.Code)
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = strings.TrimSpace(e.Body)
	}
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if message != "" {
		parts = append(parts, message)
	}
	return strings.Join(parts, ": ")
}

func parseAPIError(body []byte, statusCode int, retryAfter string) error {
	var response struct {
		Code    string `json:"errorCode"`
		Message string `json:"errorMessage"`
	}
	_ = json.Unmarshal(body, &response)
	err := &APIError{
		StatusCode: statusCode,
		Code:       strings.TrimSpace(response.Code),
		Message:    strings.TrimSpace(response.Message),
		Body:       sanitizedBody(body),
	}
	if milliseconds, parseErr := strconv.ParseInt(strings.TrimSpace(retryAfter), 10, 64); parseErr == nil && milliseconds > 0 {
		err.RetryAfter = time.UnixMilli(milliseconds)
	}
	return err
}

func sanitizedBody(body []byte) string {
	value := strings.TrimSpace(string(body))
	if len(value) > 4096 {
		return value[:4096]
	}
	return value
}
