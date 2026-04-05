package api

import (
	"net/http"
)

func (service *Service) deleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authPayload := getAuthPayload(ctx)

	_, err := service.store.SoftDeleteUserTx(ctx, authPayload.UserID)
	if err != nil {
		opErr := newResourceError(err)
		abortWithError(w, opErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
