package api

import (
	"os"
	"testing"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/tmpstore"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/Drolfothesgnir/shitposter/wauthn"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func newTestService(t *testing.T, store db.Store, rs tmpstore.Store, wa wauthn.WebAuthnConfig) *Service {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
		PublicOrigin:        "http://localhost:8080",
		AllowedOrigins:      []string{"*"},
	}

	service, err := NewService(config, store, rs, wa)
	require.NoError(t, err)
	return service
}
