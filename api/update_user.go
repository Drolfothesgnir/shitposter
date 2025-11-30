package api

import (
	"errors"
	"fmt"
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// CAUTION: bindings order matters in gin validator v10!
// To truly omit empty fields "omitempty" must be first
type UpdateUserRequest struct {
	Username      *string `json:"username" binding:"omitempty,min=3,max=50,alphanum"`
	Email         *string `json:"email" binding:"omitempty,email"`
	ProfileImgURL *string `json:"profile_img_url" binding:"omitempty,url"`
}

func (req *UpdateUserRequest) isValid() bool {
	return req.Username != nil || req.Email != nil || req.ProfileImgURL != nil
}

// TODO: handle profile image update as file
func (service *Service) updateUser(ctx *gin.Context) {
	authPayload := extractAuthPayloadFromCtx(ctx)

	var req UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(
			http.StatusBadRequest,
			NewErrorResponse(ErrInvalidParams, ExtractErrorFields(err)...))
		return
	}

	if !req.isValid() {
		ctx.JSON(http.StatusBadRequest, NewErrorResponse(errors.New("request body is empty")))
		return
	}

	arg := db.UpdateUserParams{
		ID:            authPayload.UserID,
		Username:      util.StringToPgxText(req.Username),
		Email:         util.StringToPgxText(req.Email),
		ProfileImgUrl: util.StringToPgxText(req.ProfileImgURL),
	}

	user, err := service.store.UpdateUser(ctx, arg)

	if err != nil {
		// 404 when no row (nonexistent or soft-deleted)
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, NewErrorResponse(fmt.Errorf("user with id [%d] not found", authPayload.UserID)))
			return
		}
		// 409 on unique conflicts
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			var field string
			switch pgErr.ConstraintName {
			case "uniq_users_username_active":
				field = "username"
			case "uniq_users_email_active":
				field = "email"
			}

			ctx.JSON(http.StatusConflict, NewErrorResponse(
				fmt.Errorf("%s already in use", field),
				ErrorField{FieldName: field, ErrorMessage: "already in use"},
			))
			return
		}
		ctx.JSON(http.StatusInternalServerError, NewErrorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, createPrivateUserResponse(user))
}
