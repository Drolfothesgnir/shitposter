package api

import (
	"fmt"
	"net/http"
	"strconv"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
)

func (service *Service) getUser(w http.ResponseWriter, r *http.Request) {
	param := r.PathValue("id")
	userID, err := strconv.ParseInt(param, 10, 64)
	// need to check if user id is a positive integer
	if err != nil || userID <= 0 {
		// using %q to format param in double quotes and excape special characters
		msg := fmt.Sprintf("invalid user id: %q", param)
		vErr := puke(
			ReqInvalidArguments,
			http.StatusBadRequest,
			msg,
			err,
		)
		abortWithError(w, vErr)
		return
	}

	ctx := r.Context()

	user, err := service.store.GetUser(ctx, userID)
	if err != nil {
		resErr := newResourceError(err)
		if resErr.opErr != nil && resErr.opErr.Kind == db.KindDeleted {
			resErr = notFoundResourceError(fmt.Sprintf("user with id [%d] not found", userID))
		}
		abortWithError(w, resErr)
		return
	}

	respondWithJSON(w, http.StatusOK, createPublicUserResponse(user))
}
