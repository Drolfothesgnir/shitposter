package api

import (
	"fmt"
	"net/http"
	"strconv"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
)

func (service *Service) getUser(ctx *gin.Context) {
	param := ctx.Param("id")
	userID, err := strconv.ParseInt(param, 10, 64)
	// need to check if user id is a positive integer
	if err != nil || userID <= 0 {
		// using %q to format param in double quotes and excape special characters
		msg := fmt.Sprintf("invalid user id: %q", param)
		ctx.JSON(http.StatusBadRequest, newPayloadError(msg, nil))
		return
	}

	user, err := service.store.GetUser(ctx, userID)
	if err != nil {
		resErr := newResourceError(err)
		if resErr.opErr != nil && resErr.opErr.Kind == db.KindDeleted {
			resErr = notFoundResourceError(fmt.Sprintf("user with id [%d] not found", userID))
		}
		ctx.JSON(resErr.StatusCode(), resErr)
		return
	}

	ctx.JSON(http.StatusOK, createPublicUserResponse(user))
}
