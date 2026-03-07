package api

import "github.com/gin-gonic/gin"

type Kind string

const (
	KindAuth      Kind = "auth"
	KindPayload   Kind = "payload"
	KindOperation Kind = "operation"
)

type APIError interface {
	StatusCode() int
}

func abortWithError(ctx *gin.Context, err APIError) {
	ctx.AbortWithStatusJSON(err.StatusCode(), err)
}
