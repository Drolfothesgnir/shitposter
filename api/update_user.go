package api

import (
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/gin-gonic/gin"
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
			newPayloadError("invalid request parameters", err))
		return
	}

	if !req.isValid() {
		ctx.JSON(http.StatusBadRequest, newPayloadError("request body is empty", nil))
		return
	}

	arg := db.UpdateUserParams{
		ID:            authPayload.UserID,
		Username:      req.Username,
		Email:         req.Email,
		ProfileImgURL: req.ProfileImgURL,
	}

	user, err := service.store.UpdateUser(ctx, arg)

	if err != nil {
		opErr := newResourceError(err)
		ctx.JSON(opErr.StatusCode(), opErr)
		return
	}

	ctx.JSON(http.StatusOK, user)
}
