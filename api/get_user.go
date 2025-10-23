package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (service *Service) getUser(ctx *gin.Context) {
	param := ctx.Param("id")
	userID, err := strconv.ParseInt(param, 10, 64)
	// need to check if user id is a positive integer
	if err != nil || userID <= 0 {
		// using %q to format param in double quotes and excape special characters
		err := fmt.Errorf("invalid user id: %q", param)
		ctx.JSON(http.StatusBadRequest, NewErrorResponse(err))
		return
	}

	user, err := service.store.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err := fmt.Errorf("user with id [%d] not found", userID)
			ctx.JSON(http.StatusNotFound, NewErrorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	if user.IsDeleted {
		err := fmt.Errorf("user with id [%d] not found", userID)
		ctx.JSON(http.StatusNotFound, NewErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, createPublicUserResponse(user))
}
