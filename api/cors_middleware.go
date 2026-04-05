package api

import (
	"net/http"
	"slices"
	"strings"
)

func (s Service) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Always tell caches that this response varies based on the Origin
		w.Header().Add("Vary", "Origin")

		origin := r.Header.Get("Origin")

		// 2. Only attach CORS headers if the origin is actually in our allowed list
		if origin != "" && slices.Contains(s.config.AllowedOrigins, origin) {
			// FIX: Set on the ResponseWriter (w), not the Request (r)
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// It's cleaner to only set these if the origin is allowed,
			// since the browser rejects the preflight anyway if Origin is missing/wrong.
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

			allowedHeaders := []string{
				"Content-Type",
				"Authorization",
				WebauthnTransportHeader, // Assuming this is defined elsewhere in your package
			}
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ","))
		}

		// 3. Intercept preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			// Respond with 204 No Content and stop the chain
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 4. Pass to the actual handler
		next.ServeHTTP(w, r)
	})
}
