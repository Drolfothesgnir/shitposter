package api

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/tmpstore"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/Drolfothesgnir/shitposter/wauthn"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Configure the validator to use json tags for field names in errors
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}

	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

var testConfig = util.Config{
	TokenSymmetricKey:        util.RandomString(32),
	AccessTokenDuration:      time.Minute,
	RefreshTokenDuration:     time.Minute,
	PublicOrigin:             "http://localhost:8080",
	AllowedOrigins:           []string{"*"},
	AuthenticationSessionTTL: time.Minute,
	RegistrationSessionTTL:   time.Minute,
}

func newTestService(
	t *testing.T,
	store db.Store,
	tokenMaker token.Maker,
	rs tmpstore.Store,
	wa wauthn.WebAuthnConfig,
) *Service {

	service, err := NewService(testConfig, store, tokenMaker, rs, wa)
	require.NoError(t, err)
	return service
}

func setAuthorizationHeader(t *testing.T, tokenMaker token.Maker, authorizationType string, userId int64, duration time.Duration, request *http.Request) {
	accessToken, payload, err := tokenMaker.CreateToken(userId, duration)
	require.NoError(t, err)
	require.NotEmpty(t, payload)
	authorizationToken := fmt.Sprintf("%s %s", authorizationType, accessToken)
	request.Header.Set(authorizationheaderKey, authorizationToken)
}
