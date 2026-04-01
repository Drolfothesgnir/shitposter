package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/gin-gonic/gin"
)

type contextKey string

const (
	authorizationheaderKey                = "Authorization"
	authorizationTypeBearer               = "bearer"
	ctxAuthorizationPayloadKey contextKey = "authorization_payload"
)

// authMiddleware checks if the request sender has valid auth token and
// possibly aborts with [AuthError].
// TODO: check sessions
func (s *Service) authMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get(authorizationheaderKey)
		if authorizationHeader == "" {
			abortWithError(w, newAuthError(AuthHeaderNotProvided, "authorization header is not provided"))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			abortWithError(w, newAuthError(AuthInvalidHeaderFormat, "invalid authorization header format"))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			abortWithError(w, newAuthError(AuthTypeUnsupported, fmt.Sprintf("unsupported authorization type: %s", authorizationType)))
			return
		}

		accessToken := fields[1]

		payload, err := s.tokenMaker.VerifyToken(accessToken)
		if err != nil {
			abortWithError(w, newAuthError(AuthAccessTokenErr, err.Error()))
			return
		}

		ctx := context.WithValue(r.Context(), ctxAuthorizationPayloadKey, payload)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	}
}

// Helper to get auth data after the middleware check.
func extractAuthPayloadFromCtx(ctx *gin.Context) *token.Payload {
	return ctx.MustGet(string(ctxAuthorizationPayloadKey)).(*token.Payload)
}

// getAuthPayload helps to extract [token.Payload] from the request context.
func getAuthPayload(ctx context.Context) (*token.Payload, error) {
	payload, ok := ctx.Value(ctxAuthorizationPayloadKey).(*token.Payload)
	if !ok {
		// This should theoretically never happen if the middleware is wired correctly,
		// but it prevents nil pointer panics if you forget to attach the middleware to a route!
		return nil, fmt.Errorf("authorization payload not found in context")
	}
	return payload, nil
}
