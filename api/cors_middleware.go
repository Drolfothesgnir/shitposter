package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// handling CORS
func (s *Service) corsMiddleware() gin.HandlerFunc {
	allowedSet := make(map[string]bool, len(s.config.AllowedOrigins))
	for _, o := range s.config.AllowedOrigins {
		allowedSet[o] = true
	}

	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if allowedSet[origin] {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		allowedHeaders := []string{
			"Content-Type",
			"Authorization",
			WebauthnTransportHeader,
		}
		ctx.Header("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ","))

		// If someone sends preflight (OPTIONS), respond 204 and return
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
