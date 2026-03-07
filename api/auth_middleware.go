package api

import (
	"fmt"
	"strings"

	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/gin-gonic/gin"
)

const (
	authorizationheaderKey     = "authorization"
	authorizationTypeBearer    = "bearer"
	ctxAuthorizationPayloadKey = "authorization_payload"
)

// authMiddleware checks if the request sender has valid auth token and
// possibly aborts with [AuthError].
// TODO: check sessions
func (s *Service) authMiddleware(ctx *gin.Context) {
	authorizationHeader := ctx.GetHeader(authorizationheaderKey)
	if len(authorizationHeader) == 0 {
		abortWithError(ctx, newAuthError("authorization header is not provided"))
		return
	}

	fields := strings.Fields(authorizationHeader)
	if len(fields) < 2 {
		abortWithError(ctx, newAuthError("invalid authorization header format"))
		return
	}

	authoriztionType := strings.ToLower(fields[0])
	if authoriztionType != authorizationTypeBearer {
		abortWithError(ctx, newAuthError(fmt.Sprintf("unsupported authorization type: %s", authoriztionType)))
		return
	}

	accessToken := fields[1]

	payload, err := s.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		abortWithError(ctx, newAuthError(err.Error()))
		return
	}

	ctx.Set(ctxAuthorizationPayloadKey, payload)
	ctx.Next()
}

// Helper to get auth data after the middleware check.
func extractAuthPayloadFromCtx(ctx *gin.Context) *token.Payload {
	return ctx.MustGet(ctxAuthorizationPayloadKey).(*token.Payload)
}
