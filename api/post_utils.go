package api

import (
	"fmt"
	"net/http"
	"strconv"
)

// extractPostID parses post ID from the URL path and returns it.
// If the ID is invalid, then -1 and a [Vomit] will be returned.
func extractPostID(r *http.Request) (int64, *Vomit) {
	// getting mandatory post id form the request, abort with 400 on error
	postIDRaw := r.PathValue("post_id")

	postID, err := strconv.ParseInt(postIDRaw, 10, 64)
	if err != nil {
		msg := fmt.Sprintf("Invalid post id: %s", postIDRaw)

		vErr := puke(
			ReqInvalidArguments,
			http.StatusBadRequest,
			msg,
			err,
		)
		return -1, vErr
	}

	return postID, nil
}
