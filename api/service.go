package api

import (
	"context"
	"errors"
	"net/http"
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

const (
	WebauthnChallengeHeader = "X-Webauthn-Challenge"
	WebauthnTransportHeader = "X-Webauthn-Transports"
)

// keys for values parsed from the uri and set into the request context
const (
	ctxPostIDKey    = "post_id"
	ctxCommentIDKey = "comment_id"
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
	ErrCommentDeleted              = errors.New("comment is deleted")
	ErrCannotUpdate                = errors.New("could not update the entity")
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

	service.setupRouter(server)

	// custom validator to check if comment order in requests is valid.
	// used in 'binding' tag in request.
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("comment_order", isValidCommentOrder)
	}

	service.server = server

	return service, nil
}

// Start runs the HTTP server
func (service *Service) Start() error {
	return service.server.ListenAndServe()
}

func (service *Service) Shutdown(ctx context.Context) error {
	return service.server.Shutdown(ctx)
}
