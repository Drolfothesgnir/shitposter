package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Drolfothesgnir/shitposter/token"
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
			authErr := newAuthError(
				AuthHeaderNotProvided,
				http.StatusUnauthorized,
				"authorization header is not provided",
				nil,
			)
			abortWithError(w, authErr)
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			authErr := newAuthError(
				AuthInvalidHeaderFormat,
				http.StatusUnauthorized,
				"invalid authorization header format",
				nil)
			abortWithError(w, authErr)
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			authErr := newAuthError(
				AuthTypeUnsupported,
				http.StatusUnauthorized,
				fmt.Sprintf("unsupported authorization type: %s", authorizationType),
				nil,
			)
			abortWithError(w, authErr)
			return
		}

		accessToken := fields[1]

		payload, err := s.tokenMaker.VerifyToken(accessToken)
		if err != nil {
			// We pass the raw err in so the server logs it,
			// but the user just sees "invalid or expired token"
			authErr := newAuthError(
				AuthAccessTokenErr,
				http.StatusUnauthorized,
				"invalid or expired token",
				err,
			)
			abortWithError(w, authErr)
			return
		}

		ctx := context.WithValue(r.Context(), ctxAuthorizationPayloadKey, payload)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	}
}

// getAuthPayload helps to extract [token.Payload] from the request context.
func getAuthPayload(ctx context.Context) *token.Payload {
	payload, ok := ctx.Value(ctxAuthorizationPayloadKey).(*token.Payload)
	if !ok {
		panic("AAAAAAAA: [getAuthPayload]: authorization payload not found in context. Helper must be called ONLY in one chain with the auth middleware, after it.")
		// This should theoretically never happen if the middleware is wired correctly,
		// but it prevents nil pointer panics if you forget to attach the middleware to a route!
	}
	return payload
}
