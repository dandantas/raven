package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// Logging middleware logs HTTP requests and responses
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Get correlation ID from context
		correlationID := GetCorrelationID(r.Context())

		slog.Info("HTTP request received",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"correlation_id", correlationID,
		)

		// Call next handler
		next.ServeHTTP(rw, r)

		// Log response
		duration := time.Since(start)
		slog.Info("HTTP request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status_code", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"bytes_written", rw.written,
			"correlation_id", correlationID,
		)
	})
}
