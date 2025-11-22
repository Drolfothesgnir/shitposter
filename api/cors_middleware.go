package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// handling CORS
func (s *Service) corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// since I want my service as a REST API I will allow connection from every origin
		ctx.Header("Access-Control-Allow-Origin", "*")

		// for every method
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		// X-Webauthn-Challenge and X-Webauthn-Transports — my headers for passkey auth
		allowedHeaders := []string{
			"Content-Type",
			"Authorization",
			WebauthnChallengeHeader,
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
