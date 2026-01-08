package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// CorrelationIDKey is the context key for correlation ID
type contextKey string

const CorrelationIDKey contextKey = "correlation_id"

// CorrelationID middleware generates or extracts correlation ID from requests
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if correlation ID exists in header
		correlationID := r.Header.Get("X-Correlation-ID")

		// Generate new one if not present
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Add to response header
		w.Header().Set("X-Correlation-ID", correlationID)

		// Add to context
		ctx := context.WithValue(r.Context(), CorrelationIDKey, correlationID)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCorrelationID extracts correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}
