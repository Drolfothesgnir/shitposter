package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/gin-gonic/gin"
)

const (
	authorizationheaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationheaderKey)
		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, NewErrorResponse(err))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, NewErrorResponse(err))
			return
		}

		authoriztionType := strings.ToLower(fields[0])
		if authoriztionType != authorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type: %s", authoriztionType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, NewErrorResponse(err))
			return
		}

		accessToken := fields[1]

		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, NewErrorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}
