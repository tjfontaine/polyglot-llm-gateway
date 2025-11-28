package domain

import (
	"net/http"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name:     "error with type and message",
			err:      &APIError{Type: ErrorTypeInvalidRequest, Message: "bad request"},
			expected: "invalid_request: bad request",
		},
		{
			name:     "error with type, code, and message",
			err:      &APIError{Type: ErrorTypeRateLimit, Code: ErrorCodeRateLimitExceeded, Message: "rate limited"},
			expected: "rate_limit (rate_limit_exceeded): rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAPIError_HTTPStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected int
	}{
		{
			name:     "invalid request",
			err:      &APIError{Type: ErrorTypeInvalidRequest},
			expected: http.StatusBadRequest,
		},
		{
			name:     "authentication error",
			err:      &APIError{Type: ErrorTypeAuthentication},
			expected: http.StatusUnauthorized,
		},
		{
			name:     "permission error",
			err:      &APIError{Type: ErrorTypePermission},
			expected: http.StatusForbidden,
		},
		{
			name:     "not found error",
			err:      &APIError{Type: ErrorTypeNotFound},
			expected: http.StatusNotFound,
		},
		{
			name:     "rate limit error",
			err:      &APIError{Type: ErrorTypeRateLimit},
			expected: http.StatusTooManyRequests,
		},
		{
			name:     "overloaded error",
			err:      &APIError{Type: ErrorTypeOverloaded},
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "server error",
			err:      &APIError{Type: ErrorTypeServer},
			expected: http.StatusInternalServerError,
		},
		{
			name:     "context length error",
			err:      &APIError{Type: ErrorTypeContextLength},
			expected: http.StatusBadRequest,
		},
		{
			name:     "max tokens error",
			err:      &APIError{Type: ErrorTypeMaxTokens},
			expected: http.StatusBadRequest,
		},
		{
			name:     "unknown error type",
			err:      &APIError{Type: ErrorType("unknown")},
			expected: http.StatusInternalServerError,
		},
		{
			name:     "explicit status code",
			err:      &APIError{Type: ErrorTypeInvalidRequest, StatusCode: http.StatusConflict},
			expected: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.HTTPStatusCode(); got != tt.expected {
				t.Errorf("HTTPStatusCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(ErrorTypeInvalidRequest, "test message")
	if err.Type != ErrorTypeInvalidRequest {
		t.Errorf("Type = %v, want %v", err.Type, ErrorTypeInvalidRequest)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %q, want %q", err.Message, "test message")
	}
}

func TestAPIError_WithCode(t *testing.T) {
	err := NewAPIError(ErrorTypeRateLimit, "rate limited").WithCode(ErrorCodeRateLimitExceeded)
	if err.Code != ErrorCodeRateLimitExceeded {
		t.Errorf("Code = %v, want %v", err.Code, ErrorCodeRateLimitExceeded)
	}
}

func TestAPIError_WithParam(t *testing.T) {
	err := NewAPIError(ErrorTypeInvalidRequest, "invalid value").WithParam("temperature")
	if err.Param != "temperature" {
		t.Errorf("Param = %q, want %q", err.Param, "temperature")
	}
}

func TestAPIError_WithStatusCode(t *testing.T) {
	err := NewAPIError(ErrorTypeInvalidRequest, "conflict").WithStatusCode(http.StatusConflict)
	if err.HTTPStatusCode() != http.StatusConflict {
		t.Errorf("HTTPStatusCode() = %d, want %d", err.HTTPStatusCode(), http.StatusConflict)
	}
}

func TestAPIError_WithSourceAPI(t *testing.T) {
	err := NewAPIError(ErrorTypeServer, "error").WithSourceAPI(APITypeOpenAI)
	if err.SourceAPI != APITypeOpenAI {
		t.Errorf("SourceAPI = %v, want %v", err.SourceAPI, APITypeOpenAI)
	}
}

func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name         string
		constructor  func(string) *APIError
		message      string
		expectedType ErrorType
		expectedCode ErrorCode
	}{
		{
			name:         "ErrInvalidRequest",
			constructor:  ErrInvalidRequest,
			message:      "bad request",
			expectedType: ErrorTypeInvalidRequest,
			expectedCode: "",
		},
		{
			name:         "ErrAuthentication",
			constructor:  ErrAuthentication,
			message:      "invalid key",
			expectedType: ErrorTypeAuthentication,
			expectedCode: "",
		},
		{
			name:         "ErrPermission",
			constructor:  ErrPermission,
			message:      "access denied",
			expectedType: ErrorTypePermission,
			expectedCode: "",
		},
		{
			name:         "ErrNotFound",
			constructor:  ErrNotFound,
			message:      "model not found",
			expectedType: ErrorTypeNotFound,
			expectedCode: "",
		},
		{
			name:         "ErrRateLimit",
			constructor:  ErrRateLimit,
			message:      "rate limited",
			expectedType: ErrorTypeRateLimit,
			expectedCode: ErrorCodeRateLimitExceeded,
		},
		{
			name:         "ErrOverloaded",
			constructor:  ErrOverloaded,
			message:      "server overloaded",
			expectedType: ErrorTypeOverloaded,
			expectedCode: "",
		},
		{
			name:         "ErrServer",
			constructor:  ErrServer,
			message:      "internal error",
			expectedType: ErrorTypeServer,
			expectedCode: "",
		},
		{
			name:         "ErrContextLength",
			constructor:  ErrContextLength,
			message:      "context too long",
			expectedType: ErrorTypeContextLength,
			expectedCode: ErrorCodeContextLengthExceeded,
		},
		{
			name:         "ErrMaxTokens",
			constructor:  ErrMaxTokens,
			message:      "max tokens exceeded",
			expectedType: ErrorTypeMaxTokens,
			expectedCode: ErrorCodeMaxTokensExceeded,
		},
		{
			name:         "ErrOutputTruncated",
			constructor:  ErrOutputTruncated,
			message:      "output was truncated",
			expectedType: ErrorTypeMaxTokens,
			expectedCode: ErrorCodeOutputTruncated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor(tt.message)
			if err.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", err.Type, tt.expectedType)
			}
			if err.Code != tt.expectedCode {
				t.Errorf("Code = %v, want %v", err.Code, tt.expectedCode)
			}
			if err.Message != tt.message {
				t.Errorf("Message = %q, want %q", err.Message, tt.message)
			}
		})
	}
}

func TestAPIError_Chaining(t *testing.T) {
	err := NewAPIError(ErrorTypeInvalidRequest, "test").
		WithCode(ErrorCodeContextLengthExceeded).
		WithParam("messages").
		WithStatusCode(http.StatusBadRequest).
		WithSourceAPI(APITypeAnthropic)

	if err.Type != ErrorTypeInvalidRequest {
		t.Errorf("Type = %v, want %v", err.Type, ErrorTypeInvalidRequest)
	}
	if err.Code != ErrorCodeContextLengthExceeded {
		t.Errorf("Code = %v, want %v", err.Code, ErrorCodeContextLengthExceeded)
	}
	if err.Param != "messages" {
		t.Errorf("Param = %q, want %q", err.Param, "messages")
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, http.StatusBadRequest)
	}
	if err.SourceAPI != APITypeAnthropic {
		t.Errorf("SourceAPI = %v, want %v", err.SourceAPI, APITypeAnthropic)
	}
}
