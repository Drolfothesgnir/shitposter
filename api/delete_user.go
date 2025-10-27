package api

import (
	"net/http"

	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/gin-gonic/gin"
)

func (service *Service) deleteUser(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	err := service.store.SoftDeleteUserTx(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}
