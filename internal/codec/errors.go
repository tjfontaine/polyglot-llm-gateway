// Package codec provides error conversion utilities for mapping between
// API-specific errors and canonical domain errors.
package codec

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/domain"
)

// ErrorResponse represents a generic error response that can be serialized
// to different API formats.
type ErrorResponse struct {
	StatusCode int
	Body       []byte
}

// ErrorFormatter formats domain errors for a specific API type.
type ErrorFormatter interface {
	// FormatError converts a domain error to an API-specific error response.
	FormatError(err error) *ErrorResponse
}

// ErrorParser parses API-specific errors into domain errors.
type ErrorParser interface {
	// ParseError attempts to parse an error into a domain error.
	// Returns the original error if it cannot be parsed.
	ParseError(err error) error
}

// ToCanonicalError converts any error to a domain.APIError.
// If the error is already a domain.APIError, it returns it directly.
// Otherwise, it wraps the error in a generic server error.
func ToCanonicalError(err error) *domain.APIError {
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return domain.ErrServer(err.Error())
}

// detectErrorTypeFromMessage attempts to detect the error type from the message content.
func detectErrorTypeFromMessage(message string) (domain.ErrorType, domain.ErrorCode) {
	msgLower := strings.ToLower(message)

	switch {
	case strings.Contains(msgLower, "context length") ||
		strings.Contains(msgLower, "context window") ||
		strings.Contains(msgLower, "too many tokens"):
		return domain.ErrorTypeContextLength, domain.ErrorCodeContextLengthExceeded

	case strings.Contains(msgLower, "truncated") ||
		strings.Contains(msgLower, "could not finish") ||
		strings.Contains(msgLower, "output limit"):
		return domain.ErrorTypeMaxTokens, domain.ErrorCodeOutputTruncated

	case strings.Contains(msgLower, "max_tokens") ||
		strings.Contains(msgLower, "maximum tokens"):
		return domain.ErrorTypeMaxTokens, domain.ErrorCodeMaxTokensExceeded

	case strings.Contains(msgLower, "rate limit"):
		return domain.ErrorTypeRateLimit, domain.ErrorCodeRateLimitExceeded

	case strings.Contains(msgLower, "api key") ||
		strings.Contains(msgLower, "authentication") ||
		strings.Contains(msgLower, "unauthorized"):
		return domain.ErrorTypeAuthentication, domain.ErrorCodeInvalidAPIKey

	case strings.Contains(msgLower, "model not found") ||
		strings.Contains(msgLower, "does not exist"):
		return domain.ErrorTypeNotFound, domain.ErrorCodeModelNotFound
	}

	return "", ""
}

// OpenAIErrorFormatter formats errors for OpenAI API responses.
type OpenAIErrorFormatter struct{}

// FormatError formats a domain error as an OpenAI API error response.
func (f *OpenAIErrorFormatter) FormatError(err error) *ErrorResponse {
	apiErr := ToCanonicalError(err)

	// Map domain error type to OpenAI error type
	errType := mapDomainToOpenAIErrorType(apiErr.Type)
	code := mapDomainToOpenAIErrorCode(apiErr.Code)

	// Build error object
	errObj := map[string]interface{}{
		"message": apiErr.Message,
		"type":    errType,
	}
	if code != "" {
		errObj["code"] = code
	}
	if apiErr.Param != "" {
		errObj["param"] = apiErr.Param
	}

	body, _ := json.Marshal(map[string]interface{}{
		"error": errObj,
	})

	return &ErrorResponse{
		StatusCode: apiErr.HTTPStatusCode(),
		Body:       body,
	}
}

func mapDomainToOpenAIErrorType(t domain.ErrorType) string {
	switch t {
	case domain.ErrorTypeInvalidRequest, domain.ErrorTypeContextLength, domain.ErrorTypeMaxTokens:
		return "invalid_request_error"
	case domain.ErrorTypeAuthentication:
		return "authentication_error"
	case domain.ErrorTypePermission:
		return "permission_denied"
	case domain.ErrorTypeNotFound:
		return "not_found"
	case domain.ErrorTypeRateLimit:
		return "rate_limit_error"
	case domain.ErrorTypeOverloaded:
		return "service_unavailable"
	case domain.ErrorTypeServer:
		return "server_error"
	default:
		return "server_error"
	}
}

func mapDomainToOpenAIErrorCode(c domain.ErrorCode) string {
	switch c {
	case domain.ErrorCodeContextLengthExceeded:
		return "context_length_exceeded"
	case domain.ErrorCodeRateLimitExceeded:
		return "rate_limit_exceeded"
	case domain.ErrorCodeInvalidAPIKey:
		return "invalid_api_key"
	case domain.ErrorCodeModelNotFound:
		return "model_not_found"
	case domain.ErrorCodeMaxTokensExceeded, domain.ErrorCodeOutputTruncated:
		return "max_tokens_exceeded"
	default:
		return string(c)
	}
}

// AnthropicErrorFormatter formats errors for Anthropic API responses.
type AnthropicErrorFormatter struct{}

// FormatError formats a domain error as an Anthropic API error response.
func (f *AnthropicErrorFormatter) FormatError(err error) *ErrorResponse {
	apiErr := ToCanonicalError(err)

	// Map domain error type to Anthropic error type
	errType := mapDomainToAnthropicErrorType(apiErr.Type)

	body, _ := json.Marshal(map[string]interface{}{
		"type": "error",
		"error": map[string]string{
			"type":    errType,
			"message": apiErr.Message,
		},
	})

	return &ErrorResponse{
		StatusCode: apiErr.HTTPStatusCode(),
		Body:       body,
	}
}

func mapDomainToAnthropicErrorType(t domain.ErrorType) string {
	switch t {
	case domain.ErrorTypeInvalidRequest, domain.ErrorTypeContextLength, domain.ErrorTypeMaxTokens:
		return "invalid_request_error"
	case domain.ErrorTypeAuthentication:
		return "authentication_error"
	case domain.ErrorTypePermission:
		return "permission_error"
	case domain.ErrorTypeNotFound:
		return "not_found_error"
	case domain.ErrorTypeRateLimit:
		return "rate_limit_error"
	case domain.ErrorTypeOverloaded:
		return "overloaded_error"
	case domain.ErrorTypeServer:
		return "api_error"
	default:
		return "api_error"
	}
}

// WriteError writes an error response using the appropriate formatter for the API type.
func WriteError(w http.ResponseWriter, err error, apiType domain.APIType) {
	var formatter ErrorFormatter
	switch apiType {
	case domain.APITypeOpenAI, domain.APITypeResponses:
		formatter = &OpenAIErrorFormatter{}
	case domain.APITypeAnthropic:
		formatter = &AnthropicErrorFormatter{}
	default:
		formatter = &OpenAIErrorFormatter{}
	}

	resp := formatter.FormatError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(resp.Body)
}
