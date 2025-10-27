package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/Drolfothesgnir/shitposter/api"
	db "github.com/Drolfothesgnir/shitposter/db/sqlc"
	"github.com/Drolfothesgnir/shitposter/tmpstore"
	"github.com/Drolfothesgnir/shitposter/token"
	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/Drolfothesgnir/shitposter/wauthn"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

// TODO: how to remove stale sessions and other garbage data from the db?
func main() {
	// reading .env config file
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot read config file")
	}

	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

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

	// catching interrupt signals for graceful shutdown
	// stop() or a signal catch makes context Done
	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	// Postgres connection
	conn, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to the database")
	}

	store := db.NewStore(conn)

	// running db migrations every time the server starts
	// it's idempotent, so the schema establishes only once if no new versions added
	runDBMigration(config.MigrationURL, config.DBSource)

	// waitgroup which manages goroutines for starting and stopping HTTP server
	waitGroup, ctx := errgroup.WithContext(ctx)

	RunGinServer(ctx, waitGroup, config, store)

	err = waitGroup.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("error from wait group")
	}
}

func runDBMigration(migrationURL string, dbSource string) {
	mig, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create new migrate instance")
	}

	if err = mig.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("failed to run migrate up")
	}

	log.Info().Msg("db migrated successfully")
}

func RunGinServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
) {
	rs := tmpstore.NewStore(&config)

	wa, err := wauthn.NewWebAuthnConfig(config)
	if err != nil {
		log.Error().Err(err).Msg("failed to create Webauthn config")
		return
	}

	tokenMaker, err := token.NewJWTMaker(config.TokenSymmetricKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to create JWT token maker")
		return
	}

	service, err := api.NewService(config, store, tokenMaker, rs, wa)

	if err != nil {
		log.Error().Err(err).Msg("cannot create HTTP service")
		return
	}

	waitGroup.Go(func() error {
		log.Info().Msgf("start HTTP server at %s", config.HTTPServerAddress)

		err := service.Start()

		if err != nil {
			//http.ErrServerClosed is returned once the server begins shutting down
			// which is normal
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			log.Error().Err(err).Msg("cannot start HTTP server")
		}

		return err
	})

	waitGroup.Go(func() error {
		<-ctx.Done()

		log.Info().Msg("HTTP server: graceful shutdown")

		// give the server 5 secs to finish all his processes
		toCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := service.Shutdown(toCtx)

		if err != nil {
			log.Error().Err(err).Msg("cannot shutdown HTTP server gracefully")
		}

		// closing the db connection pool
		store.Shutdown()

		log.Info().Msg("gateway server is stopped")

		return err
	})
}
