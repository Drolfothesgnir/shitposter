package util

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	MigrationURL         string        `mapstructure:"MIGRATION_URL"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS"`
	GRPCServerAddress    string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	EmailSenderName      string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress   string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword  string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}

// ExtractHostPort parses the HTTP server address and returns the host and port components.
// If no port is specified in the URL, port will be an empty string.
func (config *Config) ExtractHostPort() (host string, port string, err error) {
	urlStr, err := url.Parse(config.HTTPServerAddress)
	if err != nil {
		err = fmt.Errorf("error parsing http server url: %w", err)
		return
	}

	host, port, err = net.SplitHostPort(urlStr.Host)
	if err != nil {
		// If there's no port, SplitHostPort returns an error,
		// in which case the host itself is the hostname.
		host = urlStr.Host
		err = nil
	}

	return
}
