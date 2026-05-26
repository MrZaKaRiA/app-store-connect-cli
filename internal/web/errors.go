package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// IsDuplicateAppNameError reports whether an internal API error means app name is taken.
func IsDuplicateAppNameError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr == nil || len(apiErr.rawResponseBody()) == 0 {
		return false
	}

	var payload struct {
		Errors []struct {
			Code   string `json:"code"`
			Detail string `json:"detail"`
			Title  string `json:"title"`
		} `json:"errors"`
	}
	if json.Unmarshal(apiErr.rawResponseBody(), &payload) != nil {
		body := strings.ToLower(string(apiErr.rawResponseBody()))
		return strings.Contains(body, "app name") && strings.Contains(body, "already")
	}

	for _, e := range payload.Errors {
		code := strings.ToUpper(strings.TrimSpace(e.Code))
		detail := strings.ToLower(strings.TrimSpace(e.Detail))
		title := strings.ToLower(strings.TrimSpace(e.Title))

		if strings.Contains(code, "DUPLICATE") && (strings.Contains(detail, "app name") || strings.Contains(title, "app name")) {
			return true
		}
		if strings.Contains(detail, "app name you entered is already being used") {
			return true
		}
	}
	return false
}

// IsAlreadyExistsConflict reports whether an internal API error is a 409 caused
// by trying to create a resource that already exists.
func IsAlreadyExistsConflict(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr == nil || apiErr.Status != http.StatusConflict {
		return false
	}

	var payload struct {
		Errors []struct {
			Code   string `json:"code"`
			Detail string `json:"detail"`
			Title  string `json:"title"`
		} `json:"errors"`
	}
	if json.Unmarshal(apiErr.rawResponseBody(), &payload) != nil {
		body := strings.ToLower(string(apiErr.rawResponseBody()))
		return strings.Contains(body, "already") && (strings.Contains(body, "exist") || strings.Contains(body, "attached") || strings.Contains(body, "submitted"))
	}

	for _, e := range payload.Errors {
		code := strings.ToUpper(strings.TrimSpace(e.Code))
		detail := strings.ToLower(strings.TrimSpace(e.Detail))
		title := strings.ToLower(strings.TrimSpace(e.Title))
		text := detail + " " + title
		if strings.Contains(code, "ALREADY") || (strings.Contains(text, "already") && (strings.Contains(text, "exist") || strings.Contains(text, "attached") || strings.Contains(text, "submitted"))) {
			return true
		}
	}
	return false
}
