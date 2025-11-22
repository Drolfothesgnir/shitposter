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
	router.POST(UsersSignupStartURL, service.signupStart)
	router.POST(UsersSignupFinishURL, service.signupFinish)
	router.POST(UsersSigninStartURL, service.signinStart)
	router.POST(UsersSigninFinishURL, service.signinFinish)

	// renew access token
	router.POST(UsersRenewAccessURL, service.renewAccessToken)

	router.GET(UsersGetUser+"/:id", service.getUser)

	// public routes where post id is checked
	publicPostGroup := router.Group("/posts").Use(service.postIDMiddleware())
	publicPostGroup.GET("/:post_id/comments", service.getComments)

	// protected routes
	authGroup := router.Group("/").Use(authMiddleware(service.tokenMaker))
	authGroup.DELETE(UsersDeleteUser, service.deleteUser)
	authGroup.PATCH(UsersUpdateUser, service.updateUser)

	// private routes where post id is checked
	privatePostGroup := authGroup.Use(service.postIDMiddleware())
	privatePostGroup.POST(CommentsCreateRoot, service.createComment)
	privatePostGroup.POST(CommentsCreateReply, service.createComment)
	privatePostGroup.DELETE("/posts/:post_id")
	privatePostGroup.POST("/posts/:post_id/vote", notImplemented)

	privatePostCommentGroup := privatePostGroup.Use(service.commentIDMiddleware())
	privatePostCommentGroup.PATCH("/posts/:post_id/comments/:comment_id", notImplemented)
	privatePostCommentGroup.DELETE("/posts/:post_id/comments/:comment_id", notImplemented)
	privatePostCommentGroup.POST("/posts/:post_id/comments/:comment_id/vote", notImplemented)

	server.Handler = router
	service.router = router
}
