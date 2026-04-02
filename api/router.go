package api

import (
	"net/http"
)

// Establishes HTTP router.
func (service *Service) setupRouter(server *http.Server) {
	// TODO: create private user path

	// authGroup.DELETE("/users", service.deleteUser)
	// authGroup.PATCH("/users", service.updateUser)

	// // private routes where post id is checked
	// privatePostGroup := authGroup.Use(service.postIDMiddleware())
	// privatePostGroup.POST("/posts/:post_id/comments", service.createComment)
	// privatePostGroup.POST("/posts/:post_id/comments/:comment_id", service.createComment)
	// privatePostGroup.DELETE("/posts/:post_id")
	// privatePostGroup.POST("/posts/:post_id/vote", notImplemented)

	// privatePostCommentGroup := privatePostGroup.Use(service.commentIDMiddleware())
	// privatePostCommentGroup.DELETE("/posts/:post_id/comments/:comment_id", service.deleteComment)
	// privatePostCommentGroup.POST("/posts/:post_id/comments/:comment_id/vote", notImplemented)

	router := http.NewServeMux()

	// passkey auth
	router.HandleFunc("POST /users/signup/start", service.signupStart)
	router.HandleFunc("POST /users/signup/finish", service.signupFinish)
	router.HandleFunc("POST /users/signin/start", service.signinStart)
	router.HandleFunc("POST /users/signin/finish", service.signinFinish)

	// renew access token
	router.HandleFunc("POST /users/renew_access", service.renewAccessToken)

	// comments CRUD
	router.HandleFunc("PATCH /posts/:post_id/comments/:comment_id", service.authMiddleware(http.HandlerFunc(service.updateComment)))
	router.HandleFunc("GET /{post_id}/comments", service.getComments)

	// get user's public info
	router.HandleFunc("GET /users/{id}", service.getUser)

	server.Handler = service.corsMiddleware(router)
	service.router = router
}
