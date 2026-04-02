package api

import (
	"net/http"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

type UpdateUserRequest struct {
	Username      *string `json:"username"`
	Email         *string `json:"email"`
	ProfileImgURL *string `json:"profile_img_url"`
}

func (r UpdateUserRequest) Validate() *Vomit {
	issues := make([]Issue, 0, 4)

	if r.Username != nil {
		validate(&issues, *r.Username, "username", strMin(3), strMax(50), strAlphanum)
	}

	if r.Email != nil {
		validate(&issues, *r.Email, "email", strEmail)
	}

	if r.ProfileImgURL != nil {
		validate(&issues, *r.ProfileImgURL, "profile_img_url", strURL)
	}

	return barf(issues)
}

// TODO: handle profile image update as file
func (service *Service) updateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authPayload := getAuthPayload(ctx)

	var req UpdateUserRequest
	if vErr := ingestJSONBody(w, r, &req); vErr != nil {
		abortWithError(w, vErr)
		return
	}

	// validating the body
	if vErr := req.Validate(); vErr != nil {
		abortWithError(w, vErr)
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
		abortWithError(w, opErr)
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}
