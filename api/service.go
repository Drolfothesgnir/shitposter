package api

import (
	"context"
	"net/http"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/tmpstore"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/Drolfothesgnir/shitposter/wauthn"
)

const (
	WebauthnTransportHeader = "X-Webauthn-Transports"
	webauthnSessionCookie   = "webauthn_session"
)

type Service struct {
	config         util.Config
	store          db.Store
	tokenMaker     token.Maker
	server         *http.Server
	router         http.Handler
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

	service.server = server

	return service, nil
}

// Start runs the HTTP server
func (service *Service) Start() error {
	return service.server.ListenAndServe()
}

func (s *Service) setWebauthnSessionCookie(w http.ResponseWriter, sessionID string, maxAge int) {
	secure := s.config.Environment != "development"
	cookie := &http.Cookie{
		Name:     webauthnSessionCookie,
		Value:    sessionID,
		MaxAge:   maxAge,
		Path:     "/",
		Domain:   "",
		Secure:   secure,
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
}

func getWebauthnSessionCookieValue(r *http.Request) string {
	c, err := r.Cookie(webauthnSessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

func clearWebauthnSessionCookie(w http.ResponseWriter, secure bool) {
	cookie := &http.Cookie{
		Name:     webauthnSessionCookie,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   "",
		Secure:   secure,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

// TODO: shutdown db store and other sub-services
func (service *Service) Shutdown(ctx context.Context) error {
	return service.server.Shutdown(ctx)
}
