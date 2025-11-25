package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Establishes HTTP router.
func (service *Service) setupRouter(server *http.Server) {
	router := gin.Default()

	router.Use(service.corsMiddleware())

	router.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})

	// passkey auth
	router.POST("/users/signup/start", service.signupStart)
	router.POST("/users/signup/finish", service.signupFinish)
	router.POST("/users/signin/start", service.signinStart)
	router.POST("/users/signin/finish", service.signinFinish)

	// renew access token
	router.POST("/users/renew_access", service.renewAccessToken)

	// get user's public info
	router.GET("/users/:id", service.getUser)

	// public routes where post id is checked
	publicPostGroup := router.Group("/posts").Use(service.postIDMiddleware())
	publicPostGroup.GET("/:post_id/comments", service.getComments)

	// protected routes
	authGroup := router.Group("/").Use(authMiddleware(service.tokenMaker))
	authGroup.DELETE("/users", service.deleteUser)
	authGroup.PATCH("/users", service.updateUser)

	// private routes where post id is checked
	privatePostGroup := authGroup.Use(service.postIDMiddleware())
	privatePostGroup.POST("/posts/:post_id/comments", service.createComment)
	privatePostGroup.POST("/posts/:post_id/comments/:comment_id", service.createComment)
	privatePostGroup.DELETE("/posts/:post_id")
	privatePostGroup.POST("/posts/:post_id/vote", notImplemented)

	privatePostCommentGroup := privatePostGroup.Use(service.commentIDMiddleware())
	privatePostCommentGroup.PATCH("/posts/:post_id/comments/:comment_id", service.updateComment)
	privatePostCommentGroup.DELETE("/posts/:post_id/comments/:comment_id", service.deleteComment)
	privatePostCommentGroup.POST("/posts/:post_id/comments/:comment_id/vote", notImplemented)

	server.Handler = router
	service.router = router
}

func notImplemented(ctx *gin.Context) {
	ctx.Status(http.StatusNotImplemented)
}
