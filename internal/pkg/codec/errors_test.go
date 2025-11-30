package codec

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tjfontaine/polyglot-llm-gateway/internal/core/domain"
)

func TestToCanonicalError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType domain.ErrorType
		expectedMsg  string
	}{
		{
			name:         "domain APIError passes through",
			err:          domain.ErrInvalidRequest("bad request"),
			expectedType: domain.ErrorTypeInvalidRequest,
			expectedMsg:  "bad request",
		},
		{
			name:         "regular error becomes server error",
			err:          errors.New("something went wrong"),
			expectedType: domain.ErrorTypeServer,
			expectedMsg:  "something went wrong",
		},
		{
			name:         "rate limit error passes through",
			err:          domain.ErrRateLimit("too many requests"),
			expectedType: domain.ErrorTypeRateLimit,
			expectedMsg:  "too many requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCanonicalError(tt.err)
			if result.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", result.Type, tt.expectedType)
			}
			if result.Message != tt.expectedMsg {
				t.Errorf("Message = %v, want %v", result.Message, tt.expectedMsg)
			}
		})
	}
}

func TestOpenAIErrorFormatter_FormatError(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedStatusCode int
		expectedType       string
		expectedCode       string
	}{
		{
			name:               "invalid request error",
			err:                domain.ErrInvalidRequest("bad request"),
			expectedStatusCode: http.StatusBadRequest,
			expectedType:       "invalid_request_error",
			expectedCode:       "",
		},
		{
			name:               "authentication error",
			err:                domain.ErrAuthentication("invalid key"),
			expectedStatusCode: http.StatusUnauthorized,
			expectedType:       "authentication_error",
			expectedCode:       "",
		},
		{
			name:               "permission error",
			err:                domain.ErrPermission("access denied"),
			expectedStatusCode: http.StatusForbidden,
			expectedType:       "permission_denied",
			expectedCode:       "",
		},
		{
			name:               "not found error",
			err:                domain.ErrNotFound("model not found"),
			expectedStatusCode: http.StatusNotFound,
			expectedType:       "not_found",
			expectedCode:       "",
		},
		{
			name:               "rate limit error",
			err:                domain.ErrRateLimit("rate limited"),
			expectedStatusCode: http.StatusTooManyRequests,
			expectedType:       "rate_limit_error",
			expectedCode:       "rate_limit_exceeded",
		},
		{
			name:               "overloaded error",
			err:                domain.ErrOverloaded("server overloaded"),
			expectedStatusCode: http.StatusServiceUnavailable,
			expectedType:       "service_unavailable",
			expectedCode:       "",
		},
		{
			name:               "server error",
			err:                domain.ErrServer("internal error"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedType:       "server_error",
			expectedCode:       "",
		},
		{
			name:               "context length error",
			err:                domain.ErrContextLength("context too long"),
			expectedStatusCode: http.StatusBadRequest,
			expectedType:       "invalid_request_error",
			expectedCode:       "context_length_exceeded",
		},
		{
			name:               "max tokens error",
			err:                domain.ErrMaxTokens("max tokens exceeded"),
			expectedStatusCode: http.StatusBadRequest,
			expectedType:       "invalid_request_error",
			expectedCode:       "max_tokens_exceeded",
		},
		{
			name:               "output truncated error",
			err:                domain.ErrOutputTruncated("output was truncated"),
			expectedStatusCode: http.StatusBadRequest,
			expectedType:       "invalid_request_error",
			expectedCode:       "max_tokens_exceeded",
		},
		{
			name:               "regular go error becomes server error",
			err:                errors.New("something went wrong"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedType:       "server_error",
			expectedCode:       "",
		},
	}

	formatter := &OpenAIErrorFormatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := formatter.FormatError(tt.err)

			if resp.StatusCode != tt.expectedStatusCode {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tt.expectedStatusCode)
			}

			// Parse response body
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			errObj, ok := result["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected 'error' object in response")
			}

			if errType, _ := errObj["type"].(string); errType != tt.expectedType {
				t.Errorf("error.type = %q, want %q", errType, tt.expectedType)
			}

			if tt.expectedCode != "" {
				if code, _ := errObj["code"].(string); code != tt.expectedCode {
					t.Errorf("error.code = %q, want %q", code, tt.expectedCode)
				}
			}
		})
	}
}

func TestAnthropicErrorFormatter_FormatError(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedStatusCode int
		expectedType       string
	}{
		{
			name:               "invalid request error",
			err:                domain.ErrInvalidRequest("bad request"),
			expectedStatusCode: http.StatusBadRequest,
			expectedType:       "invalid_request_error",
		},
		{
			name:               "authentication error",
			err:                domain.ErrAuthentication("invalid key"),
			expectedStatusCode: http.StatusUnauthorized,
			expectedType:       "authentication_error",
		},
		{
			name:               "permission error",
			err:                domain.ErrPermission("access denied"),
			expectedStatusCode: http.StatusForbidden,
			expectedType:       "permission_error",
		},
		{
			name:               "not found error",
			err:                domain.ErrNotFound("model not found"),
			expectedStatusCode: http.StatusNotFound,
			expectedType:       "not_found_error",
		},
		{
			name:               "rate limit error",
			err:                domain.ErrRateLimit("rate limited"),
			expectedStatusCode: http.StatusTooManyRequests,
			expectedType:       "rate_limit_error",
		},
		{
			name:               "overloaded error",
			err:                domain.ErrOverloaded("server overloaded"),
			expectedStatusCode: http.StatusServiceUnavailable,
			expectedType:       "overloaded_error",
		},
		{
			name:               "server error",
			err:                domain.ErrServer("internal error"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedType:       "api_error",
		},
		{
			name:               "context length error",
			err:                domain.ErrContextLength("context too long"),
			expectedStatusCode: http.StatusBadRequest,
			expectedType:       "invalid_request_error",
		},
		{
			name:               "max tokens error",
			err:                domain.ErrMaxTokens("max tokens exceeded"),
			expectedStatusCode: http.StatusBadRequest,
			expectedType:       "invalid_request_error",
		},
		{
			name:               "regular go error becomes api_error",
			err:                errors.New("something went wrong"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedType:       "api_error",
		},
	}

	formatter := &AnthropicErrorFormatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := formatter.FormatError(tt.err)

			if resp.StatusCode != tt.expectedStatusCode {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tt.expectedStatusCode)
			}

			// Parse response body
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			// Anthropic format has {"type": "error", "error": {"type": "...", "message": "..."}}
			if result["type"] != "error" {
				t.Errorf("expected top-level type 'error', got %v", result["type"])
			}

			errObj, ok := result["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected 'error' object in response")
			}

			if errType, _ := errObj["type"].(string); errType != tt.expectedType {
				t.Errorf("error.type = %q, want %q", errType, tt.expectedType)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		apiType            domain.APIType
		expectedStatusCode int
		checkBody          func(t *testing.T, body []byte)
	}{
		{
			name:               "OpenAI format",
			err:                domain.ErrInvalidRequest("bad request"),
			apiType:            domain.APITypeOpenAI,
			expectedStatusCode: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("Failed to parse body: %v", err)
				}
				if _, ok := result["error"]; !ok {
					t.Error("Expected 'error' field in response")
				}
			},
		},
		{
			name:               "Anthropic format",
			err:                domain.ErrInvalidRequest("bad request"),
			apiType:            domain.APITypeAnthropic,
			expectedStatusCode: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("Failed to parse body: %v", err)
				}
				if result["type"] != "error" {
					t.Error("Expected top-level 'type' to be 'error'")
				}
				if _, ok := result["error"]; !ok {
					t.Error("Expected 'error' field in response")
				}
			},
		},
		{
			name:               "Responses API uses OpenAI format",
			err:                domain.ErrRateLimit("rate limited"),
			apiType:            domain.APITypeResponses,
			expectedStatusCode: http.StatusTooManyRequests,
			checkBody: func(t *testing.T, body []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("Failed to parse body: %v", err)
				}
				// OpenAI format: {"error": {...}}
				if _, ok := result["error"]; !ok {
					t.Error("Expected 'error' field in response")
				}
			},
		},
		{
			name:               "default uses OpenAI format",
			err:                domain.ErrServer("internal error"),
			apiType:            domain.APIType("unknown"),
			expectedStatusCode: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body []byte) {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("Failed to parse body: %v", err)
				}
				if _, ok := result["error"]; !ok {
					t.Error("Expected 'error' field in response")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteError(rec, tt.err, tt.apiType)

			if rec.Code != tt.expectedStatusCode {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatusCode)
			}

			if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}

			tt.checkBody(t, rec.Body.Bytes())
		})
	}
}

func TestDetectErrorTypeFromMessage(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		expectedType domain.ErrorType
		expectedCode domain.ErrorCode
	}{
		{
			name:         "context length exceeded",
			message:      "This model's maximum context length is 128000 tokens",
			expectedType: domain.ErrorTypeContextLength,
			expectedCode: domain.ErrorCodeContextLengthExceeded,
		},
		{
			name:         "context window exceeded",
			message:      "Request exceeds context window",
			expectedType: domain.ErrorTypeContextLength,
			expectedCode: domain.ErrorCodeContextLengthExceeded,
		},
		{
			name:         "too many tokens",
			message:      "Too many tokens in the request",
			expectedType: domain.ErrorTypeContextLength,
			expectedCode: domain.ErrorCodeContextLengthExceeded,
		},
		{
			name:         "max_tokens truncated",
			message:      "Response was truncated because max_tokens was reached",
			expectedType: domain.ErrorTypeMaxTokens,
			expectedCode: domain.ErrorCodeOutputTruncated,
		},
		{
			name:         "could not finish due to max_tokens",
			message:      "Could not finish the message because max_tokens was reached",
			expectedType: domain.ErrorTypeMaxTokens,
			expectedCode: domain.ErrorCodeOutputTruncated,
		},
		{
			name:         "output limit was reached",
			message:      "The output limit was reached",
			expectedType: domain.ErrorTypeMaxTokens,
			expectedCode: domain.ErrorCodeOutputTruncated,
		},
		{
			name:         "max_tokens exceeded (not truncated)",
			message:      "max_tokens must be less than 4096",
			expectedType: domain.ErrorTypeMaxTokens,
			expectedCode: domain.ErrorCodeMaxTokensExceeded,
		},
		{
			name:         "rate limit error",
			message:      "Rate limit exceeded",
			expectedType: domain.ErrorTypeRateLimit,
			expectedCode: domain.ErrorCodeRateLimitExceeded,
		},
		{
			name:         "api key error",
			message:      "Invalid API key provided",
			expectedType: domain.ErrorTypeAuthentication,
			expectedCode: domain.ErrorCodeInvalidAPIKey,
		},
		{
			name:         "authentication error",
			message:      "Authentication failed",
			expectedType: domain.ErrorTypeAuthentication,
			expectedCode: domain.ErrorCodeInvalidAPIKey,
		},
		{
			name:         "unauthorized error",
			message:      "Unauthorized access",
			expectedType: domain.ErrorTypeAuthentication,
			expectedCode: domain.ErrorCodeInvalidAPIKey,
		},
		{
			name:         "model not found",
			message:      "Model not found: gpt-5-turbo",
			expectedType: domain.ErrorTypeNotFound,
			expectedCode: domain.ErrorCodeModelNotFound,
		},
		{
			name:         "does not exist",
			message:      "The model 'gpt-5' does not exist",
			expectedType: domain.ErrorTypeNotFound,
			expectedCode: domain.ErrorCodeModelNotFound,
		},
		{
			name:         "unrecognized message",
			message:      "Some generic error message",
			expectedType: "",
			expectedCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType, errCode := detectErrorTypeFromMessage(tt.message)
			if errType != tt.expectedType {
				t.Errorf("ErrorType = %q, want %q", errType, tt.expectedType)
			}
			if errCode != tt.expectedCode {
				t.Errorf("ErrorCode = %q, want %q", errCode, tt.expectedCode)
			}
		})
	}
}

func TestOpenAIErrorFormatter_WithParam(t *testing.T) {
	err := domain.ErrInvalidRequest("invalid value").WithParam("temperature")
	formatter := &OpenAIErrorFormatter{}
	resp := formatter.FormatError(err)

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("Failed to parse body: %v", err)
	}

	errObj := result["error"].(map[string]interface{})
	if param, ok := errObj["param"].(string); !ok || param != "temperature" {
		t.Errorf("Expected param 'temperature', got %v", errObj["param"])
	}
}

func TestMapDomainToOpenAIErrorType(t *testing.T) {
	tests := []struct {
		input    domain.ErrorType
		expected string
	}{
		{domain.ErrorTypeInvalidRequest, "invalid_request_error"},
		{domain.ErrorTypeContextLength, "invalid_request_error"},
		{domain.ErrorTypeMaxTokens, "invalid_request_error"},
		{domain.ErrorTypeAuthentication, "authentication_error"},
		{domain.ErrorTypePermission, "permission_denied"},
		{domain.ErrorTypeNotFound, "not_found"},
		{domain.ErrorTypeRateLimit, "rate_limit_error"},
		{domain.ErrorTypeOverloaded, "service_unavailable"},
		{domain.ErrorTypeServer, "server_error"},
		{domain.ErrorType("unknown"), "server_error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := mapDomainToOpenAIErrorType(tt.input)
			if result != tt.expected {
				t.Errorf("mapDomainToOpenAIErrorType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMapDomainToAnthropicErrorType(t *testing.T) {
	tests := []struct {
		input    domain.ErrorType
		expected string
	}{
		{domain.ErrorTypeInvalidRequest, "invalid_request_error"},
		{domain.ErrorTypeContextLength, "invalid_request_error"},
		{domain.ErrorTypeMaxTokens, "invalid_request_error"},
		{domain.ErrorTypeAuthentication, "authentication_error"},
		{domain.ErrorTypePermission, "permission_error"},
		{domain.ErrorTypeNotFound, "not_found_error"},
		{domain.ErrorTypeRateLimit, "rate_limit_error"},
		{domain.ErrorTypeOverloaded, "overloaded_error"},
		{domain.ErrorTypeServer, "api_error"},
		{domain.ErrorType("unknown"), "api_error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := mapDomainToAnthropicErrorType(tt.input)
			if result != tt.expected {
				t.Errorf("mapDomainToAnthropicErrorType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMapDomainToOpenAIErrorCode(t *testing.T) {
	tests := []struct {
		input    domain.ErrorCode
		expected string
	}{
		{domain.ErrorCodeContextLengthExceeded, "context_length_exceeded"},
		{domain.ErrorCodeRateLimitExceeded, "rate_limit_exceeded"},
		{domain.ErrorCodeInvalidAPIKey, "invalid_api_key"},
		{domain.ErrorCodeModelNotFound, "model_not_found"},
		{domain.ErrorCodeMaxTokensExceeded, "max_tokens_exceeded"},
		{domain.ErrorCodeOutputTruncated, "max_tokens_exceeded"},
		{domain.ErrorCode("custom_code"), "custom_code"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := mapDomainToOpenAIErrorCode(tt.input)
			if result != tt.expected {
				t.Errorf("mapDomainToOpenAIErrorCode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
