package wauthn

import (
	"fmt"
	"net/http"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// The purpose of this interface is to create abstraction to be able to mock it in the tests.
type WebAuthnConfig interface {
	BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (creation *protocol.CredentialCreation, session *webauthn.SessionData, err error)
	FinishRegistration(user webauthn.User, session webauthn.SessionData, request *http.Request) (credential *webauthn.Credential, err error)
	BeginLogin(user webauthn.User, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	FinishLogin(user webauthn.User, session webauthn.SessionData, request *http.Request) (credential *webauthn.Credential, err error)
}

func NewWebAuthnConfig(config util.Config) (*webauthn.WebAuthn, error) {
	// Relay Party id must be the same as domain of the server and most NOT be changed
	// otherwise all stored creds will be lost
	host, _, err := config.PublicOrigin.ExtractHostPort()

	if err != nil {
		return nil, fmt.Errorf("failed to parse server http address: %w", err)
	}

	waConfig := &webauthn.Config{
		RPDisplayName: config.RPDisplayName,
		RPID:          host,
		RPOrigins:     config.AllowedOrigins,
	}

	return webauthn.New(waConfig)
}
