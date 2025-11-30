// Package domain provides canonical error types for the gateway.
package domain

import (
	"fmt"
	"net/http"
)

// ErrorType represents the category of an API error.
type ErrorType string

const (
	// ErrorTypeInvalidRequest indicates a malformed or invalid request.
	ErrorTypeInvalidRequest ErrorType = "invalid_request"

	// ErrorTypeAuthentication indicates an authentication failure.
	ErrorTypeAuthentication ErrorType = "authentication"

	// ErrorTypePermission indicates a permission/authorization failure.
	ErrorTypePermission ErrorType = "permission"

	// ErrorTypeNotFound indicates a resource was not found.
	ErrorTypeNotFound ErrorType = "not_found"

	// ErrorTypeRateLimit indicates rate limiting was triggered.
	ErrorTypeRateLimit ErrorType = "rate_limit"

	// ErrorTypeOverloaded indicates the service is overloaded.
	ErrorTypeOverloaded ErrorType = "overloaded"

	// ErrorTypeServer indicates an internal server error.
	ErrorTypeServer ErrorType = "server"

	// ErrorTypeContextLength indicates the context length was exceeded.
	ErrorTypeContextLength ErrorType = "context_length"

	// ErrorTypeMaxTokens indicates a max_tokens limit issue.
	ErrorTypeMaxTokens ErrorType = "max_tokens"
)

// ErrorCode provides additional specificity beyond the error type.
type ErrorCode string

const (
	// Common error codes
	ErrorCodeContextLengthExceeded ErrorCode = "context_length_exceeded"
	ErrorCodeRateLimitExceeded     ErrorCode = "rate_limit_exceeded"
	ErrorCodeInvalidAPIKey         ErrorCode = "invalid_api_key"
	ErrorCodeModelNotFound         ErrorCode = "model_not_found"
	ErrorCodeMaxTokensExceeded     ErrorCode = "max_tokens_exceeded"
	ErrorCodeOutputTruncated       ErrorCode = "output_truncated"
)

// APIError represents a canonical API error that can be returned by providers
// and translated to the appropriate format by frontdoors.
type APIError struct {
	// Type is the category of error
	Type ErrorType `json:"type"`

	// Code is an optional specific error code
	Code ErrorCode `json:"code,omitempty"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// Param is the parameter that caused the error (if applicable)
	Param string `json:"param,omitempty"`

	// StatusCode is the suggested HTTP status code
	StatusCode int `json:"-"`

	// SourceAPI indicates which API the error originated from (for debugging)
	SourceAPI APIType `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (%s): %s", e.Type, e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// HTTPStatusCode returns the appropriate HTTP status code for this error.
func (e *APIError) HTTPStatusCode() int {
	if e.StatusCode != 0 {
		return e.StatusCode
	}

	// Map error types to default status codes
	switch e.Type {
	case ErrorTypeInvalidRequest, ErrorTypeContextLength, ErrorTypeMaxTokens:
		return http.StatusBadRequest
	case ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case ErrorTypePermission:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeOverloaded:
		return http.StatusServiceUnavailable
	case ErrorTypeServer:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewAPIError creates a new API error.
func NewAPIError(errType ErrorType, message string) *APIError {
	return &APIError{
		Type:    errType,
		Message: message,
	}
}

// WithCode adds an error code to the error.
func (e *APIError) WithCode(code ErrorCode) *APIError {
	e.Code = code
	return e
}

// WithParam adds a parameter name to the error.
func (e *APIError) WithParam(param string) *APIError {
	e.Param = param
	return e
}

// WithStatusCode sets a specific HTTP status code.
func (e *APIError) WithStatusCode(code int) *APIError {
	e.StatusCode = code
	return e
}

// WithSourceAPI sets the source API type.
func (e *APIError) WithSourceAPI(api APIType) *APIError {
	e.SourceAPI = api
	return e
}

// Convenience constructors for common errors

// ErrInvalidRequest creates an invalid request error.
func ErrInvalidRequest(message string) *APIError {
	return NewAPIError(ErrorTypeInvalidRequest, message)
}

// ErrAuthentication creates an authentication error.
func ErrAuthentication(message string) *APIError {
	return NewAPIError(ErrorTypeAuthentication, message)
}

// ErrPermission creates a permission error.
func ErrPermission(message string) *APIError {
	return NewAPIError(ErrorTypePermission, message)
}

// ErrNotFound creates a not found error.
func ErrNotFound(message string) *APIError {
	return NewAPIError(ErrorTypeNotFound, message)
}

// ErrRateLimit creates a rate limit error.
func ErrRateLimit(message string) *APIError {
	return NewAPIError(ErrorTypeRateLimit, message).
		WithCode(ErrorCodeRateLimitExceeded)
}

// ErrOverloaded creates an overloaded error.
func ErrOverloaded(message string) *APIError {
	return NewAPIError(ErrorTypeOverloaded, message)
}

// ErrServer creates a server error.
func ErrServer(message string) *APIError {
	return NewAPIError(ErrorTypeServer, message)
}

// ErrContextLength creates a context length exceeded error.
func ErrContextLength(message string) *APIError {
	return NewAPIError(ErrorTypeContextLength, message).
		WithCode(ErrorCodeContextLengthExceeded)
}

// ErrMaxTokens creates a max tokens error.
func ErrMaxTokens(message string) *APIError {
	return NewAPIError(ErrorTypeMaxTokens, message).
		WithCode(ErrorCodeMaxTokensExceeded)
}

// ErrOutputTruncated creates an output truncated error (max_tokens reached during generation).
func ErrOutputTruncated(message string) *APIError {
	return NewAPIError(ErrorTypeMaxTokens, message).
		WithCode(ErrorCodeOutputTruncated)
}
