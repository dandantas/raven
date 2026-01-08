package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery middleware recovers from panics and logs them
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				correlationID := GetCorrelationID(r.Context())

				// Log the panic
				slog.Error("Panic recovered",
					"error", err,
					"stack_trace", string(debug.Stack()),
					"method", r.Method,
					"path", r.URL.Path,
					"correlation_id", correlationID,
				)

				// Return 500 Internal Server Error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
