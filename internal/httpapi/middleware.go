package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
)

type requestIDKey struct{}

func requestIDMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var bytes [16]byte
			if _, err := rand.Read(bytes[:]); err != nil {
				logger.ErrorContext(r.Context(), "generate request ID", "error", err)
				writeError(w, r, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
				return
			}
			requestID := hex.EncodeToString(bytes[:])
			ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
			r.Header.Set("X-Request-ID", requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}
