package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (service *Service) deleteUser(ctx *gin.Context) {
	authPayload := extractAuthPayloadFromCtx(ctx)

	err := service.store.SoftDeleteUserTx(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}
