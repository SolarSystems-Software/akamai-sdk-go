package akamai

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// GetMessageFromErrorResponse gets the `message` JSON field from the given response body.
func GetMessageFromErrorResponse(body []byte) string {
	type ErrorResponse struct {
		Message string `json:"message"`
	}

	var response ErrorResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return ""
	}
	return response.Message
}

// ApiOperationError represents a generic API request failure due to a bad HTTP status code.
type ApiOperationError struct {
	// StatusCode is the HTTP response status code that caused the error.
	StatusCode int

	// Message is the error message provided by the server.
	Message string
}

func (err ApiOperationError) Error() string {
	var builder strings.Builder
	builder.WriteString("akamai-sdk-go: API operation failed with HTTP ")
	builder.WriteString(strconv.Itoa(err.StatusCode))
	builder.WriteString(" ")
	builder.WriteString(http.StatusText(err.StatusCode))

	if err.Message != "" {
		builder.WriteString("; ")
		builder.WriteString(err.Message)
	}

	return builder.String()
}
