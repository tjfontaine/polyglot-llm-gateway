package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// logFieldsKey identifies request-scoped logging fields.
type logFieldsKey struct{}

// LoggingMiddleware logs HTTP requests with structured logging.
// It captures request details at the start and completion of each request,
// including method, path, status code, duration, and any custom fields added via AddLogField.
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Attach mutable log fields map to context for handlers to enrich
			fields := make(map[string]string)
			ctxWithFields := context.WithValue(r.Context(), logFieldsKey{}, fields)

			// Wrap response writer to capture status code
			wrapped := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Get request ID from context
			requestID, _ := r.Context().Value(RequestIDKey).(string)

			// Log request start
			logger.Info("request started",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)

			next.ServeHTTP(wrapped, r.WithContext(ctxWithFields))

			// Log request completion
			duration := time.Since(start)
			attrs := []slog.Attr{
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
			}

			if len(fields) > 0 {
				for k, v := range fields {
					attrs = append(attrs, slog.String(k, v))
				}
			}

			logger.LogAttrs(ctxWithFields, slog.LevelInfo, "request completed", attrs...)
		})
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *loggingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush forwards Flush to the underlying ResponseWriter if it supports http.Flusher,
// preserving streaming support (e.g., for SSE).
func (rw *loggingResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// AddLogField attaches a key/value to the request-scoped log fields map so LoggingMiddleware can emit it.
// It is safe to call multiple times. No-op if middleware isn't present.
func AddLogField(ctx context.Context, key, value string) {
	if value == "" {
		return
	}
	if fields, ok := ctx.Value(logFieldsKey{}).(map[string]string); ok {
		fields[key] = value
	}
}

// AddError attaches an error message to the request-scoped log fields map so it
// appears in the structured request log emitted by LoggingMiddleware. No-op if
// middleware isn't present or err is nil.
func AddError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	AddLogField(ctx, "error", err.Error())
}
