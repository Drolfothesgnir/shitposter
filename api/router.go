package api

import (
	"net/http"
)

// Establishes HTTP router.
func (service *Service) setupRouter(server *http.Server) {
	// TODO: create private user path

	// privatePostGroup.DELETE("/posts/:post_id")
	// privatePostGroup.POST("/posts/:post_id/vote", notImplemented)

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
	// one for root comments
	router.HandleFunc("POST /posts/{post_id}/comments", service.authMiddleware(http.HandlerFunc(service.createComment)))
	// and one for replies
	router.HandleFunc("POST /posts/{post_id}/comments/{comment_id}", service.authMiddleware(http.HandlerFunc(service.createComment)))
	router.HandleFunc("GET /posts/{post_id}/comments", service.getComments)
	router.HandleFunc("PATCH /posts/{post_id}/comments/{comment_id}", service.authMiddleware(http.HandlerFunc(service.updateComment)))
	router.HandleFunc("DELETE /posts/{post_id}/comments/{comment_id}", service.authMiddleware(http.HandlerFunc(service.deleteComment)))

	// users CRUD
	router.HandleFunc("GET /users/{id}", service.getUser)
	router.HandleFunc("PATCH /users", service.authMiddleware(http.HandlerFunc(service.updateUser)))
	router.HandleFunc("DELETE /users", service.authMiddleware(http.HandlerFunc(service.deleteUser)))

	server.Handler = service.corsMiddleware(router)
	service.router = router
}
