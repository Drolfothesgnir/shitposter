package api

import (
	"net/http"

	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TODO: refactor this
func (service *Service) deleteUser(ctx *gin.Context) {
	payload, ok := ctx.Get(authorizationPayloadKey)
	if !ok {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	authPayload := payload.(*token.Payload)

	// find user to check if user can be deleted
	user, err := service.store.GetUser(ctx, authPayload.UserID)
	if err != nil {
		// if there is no user with requested id abort
		if err == pgx.ErrNoRows {
			ctx.Status(http.StatusNoContent)
			return
		}

		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	// check if user is deleted, abort if true
	if user.IsDeleted {
		ctx.Status(http.StatusNoContent)
		return
	}

	err = service.store.SoftDeleteUserTx(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}
