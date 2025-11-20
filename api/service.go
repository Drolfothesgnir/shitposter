package api

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/tmpstore"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/Drolfothesgnir/shitposter/wauthn"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// TODO: rename consts properly
const (
	WebauthnChallengeHeader = "X-Webauthn-Challenge"
	WebauthnTransportHeader = "X-Webauthn-Transports"
	// api routes
	UsersSignupStartURL  = "/users/signup/start"
	UsersSignupFinishURL = "/users/signup/finish"
	UsersSigninStartURL  = "/users/signin/start"
	UsersSigninFinishURL = "/users/signin/finish"
	UsersRenewAccessURL  = "/users/renew_access"
	UsersDeleteUser      = "/users"
	UsersGetUser         = "/users"
	UsersUpdateUser      = "/users"
	CommentsCreateRoot   = "/posts/:post_id/comments"
	CommentsCreateReply  = "/posts/:post_id/comments/:comment_id"
)

var (
	// api errors
	ErrSessionBlocked              = errors.New("session is blocked")
	ErrSessionUserMismatch         = errors.New("incorrect session user")
	ErrSessionRefreshTokenMismatch = errors.New("refresh token mismatch")
	ErrSessionExpired              = errors.New("session is expired")
	ErrInvalidPostID               = errors.New("invalid post id")
	ErrInvalidParams               = errors.New("invalid params")
	ErrInvalidCommentID            = errors.New("invalid comment id")
	ErrInvalidParentCommentId      = errors.New("invalid parent comment id")
)

type Service struct {
	config         util.Config
	store          db.Store
	tokenMaker     token.Maker
	server         *http.Server
	router         *gin.Engine
	webauthnConfig wauthn.WebAuthnConfig
	redisStore     tmpstore.Store
}

// Returns new service instance with provided config and store.
func NewService(
	config util.Config,
	store db.Store,
	tokenMaker token.Maker,
	rs tmpstore.Store,
	wa wauthn.WebAuthnConfig,
) (*Service, error) {

	service := &Service{
		config:         config,
		store:          store,
		tokenMaker:     tokenMaker,
		redisStore:     rs,
		webauthnConfig: wa,
	}

	server := &http.Server{
		Addr: config.HTTPServerAddress.String(),
	}

	// caps how long a client can take to send just the headers (blocks slowloris).
	server.ReadHeaderTimeout = 5 * time.Second
	// caps time to read the full request (incl. body).
	server.ReadTimeout = 10 * time.Second
	// caps time you’ll spend writing the response (no “forever hanging” clients)
	server.WriteTimeout = 15 * time.Second
	// how long to keep idle keep-alive connections open.
	server.IdleTimeout = 60 * time.Second

	service.SetupRouter(server)

	// custom validator to check if comment order in requests is valid.
	// used in 'binding' tag in request.
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("comment_order", isValidCommentOrder)
	}

	service.server = server

	return service, nil
}

// Establishes HTTP router.
func (service *Service) SetupRouter(server *http.Server) {
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

	server.Handler = router
	service.router = router
}

// handling CORS
//
// TODO: if I want my server as a REST API platform
// then I need to be able to handle requests from different clients
// and not only from predefined domains.
func (service *Service) corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")

		if slices.Contains(service.config.AllowedOrigins, origin) {
			ctx.Header("Access-Control-Allow-Origin", origin)
		}

		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// X-Webauthn-Challenge and X-Webauthn-Transports are critical for passkey auth
		allowedHeaders := []string{
			"Content-Type",
			WebauthnChallengeHeader,
			WebauthnTransportHeader,
		}

		ctx.Header("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ","))

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}

// Start runs the HTTP server
func (service *Service) Start() error {
	return service.server.ListenAndServe()
}

func (service *Service) Shutdown(ctx context.Context) error {
	return service.server.Shutdown(ctx)
}
