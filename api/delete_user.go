package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (service *Service) deleteUser(ctx *gin.Context) {
	authPayload := extractAuthPayloadFromCtx(ctx)

	_, err := service.store.SoftDeleteUserTx(ctx, authPayload.UserID)
	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	ctx.Status(http.StatusNoContent)
}
