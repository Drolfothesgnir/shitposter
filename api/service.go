package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
)

type Service struct {
	config util.Config
	store  db.Store
	// tokenMaker token.Maker
	server         *http.Server
	webauthnConfig *webauthn.WebAuthn
}

// Returns new service instance with provided config and store.
func NewService(config util.Config, store db.Store) (*Service, error) {
	service := &Service{
		config: config,
		store:  store,
	}

	server := &http.Server{
		Addr: config.HTTPServerAddress,
	}

	// Relay Party id must be the same as domain of the server and most NOT be changed
	// otherwise all stored creds will be lost
	host, _, err := config.ExtractHostPort()

	if err != nil {
		return nil, fmt.Errorf("failed to parse server http address: %w", err)
	}

	waConfig := &webauthn.Config{
		RPDisplayName: "Shitposter",
		RPID:          host,
		RPOrigins:     []string{config.HTTPServerAddress},
	}

	service.webauthnConfig, err = webauthn.New(waConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Webauthn config: %w", err)
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

	service.server = server

	return service, nil
}

// Establishes HTTP router.
func (service *Service) SetupRouter(server *http.Server) {
	router := gin.Default()

	// TODO: add some routes
	router.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})

	server.Handler = router
}

// Start runs the HTTP server
func (service *Service) Start() error {
	return service.server.ListenAndServe()
}

func (service *Service) Shutdown(ctx context.Context) error {
	return service.server.Shutdown(ctx)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
