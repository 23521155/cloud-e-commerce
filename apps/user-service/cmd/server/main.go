package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fizzisme/user-service/internal/config"
	"github.com/fizzisme/user-service/internal/database"
	"github.com/fizzisme/user-service/internal/google"
	"github.com/fizzisme/user-service/internal/logger"
	"github.com/fizzisme/user-service/internal/outbox"
	"github.com/fizzisme/user-service/internal/router"
	"github.com/fizzisme/user-service/internal/secretbox"
	"github.com/fizzisme/user-service/internal/token"
	"github.com/fizzisme/user-service/internal/user"
	"go.uber.org/zap"
)

// main is the entry point of the user service. It wires up config and
// logging, starts the HTTP server, and handles graceful shutdown on
// SIGINT/SIGTERM.
func main() {

	if err := logger.Init(); err != nil {
		// logger.Log is not available yet, fall back to the standard logger.
		log.Fatal(err)
	}

	// Flush any buffered log entries before the process exits.
	defer logger.Log.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatal(
			"cannot load config",
			zap.Error(err),
		)
	}

	ctx := context.Background()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Log.Fatal(
			"cannot open database",
			zap.Error(err),
		)
	}
	defer db.Close()

	logger.Log.Info(
		"Application starting",
		zap.String("app", cfg.AppName),
		zap.String("port", cfg.Port),
	)

	// Without the gateway's secret no one could sign in, so do not start.
	tokens, err := token.NewIssuer(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL)
	if err != nil {
		logger.Log.Fatal(
			"cannot sign access tokens",
			zap.Error(err),
		)
	}

	options := user.Options{
		RefreshTTL: cfg.RefreshTokenTTL,
		WebURL:     cfg.WebURL,
	}

	// TOTP secrets are kept encrypted; without the key two-step sign-in
	// cannot be set up, and accounts that have it cannot finish signing in.
	if cfg.MFAEncryptionKey == "" {
		logger.Log.Warn("MFA_ENCRYPTION_KEY is not set: two-step sign-in is off")
	} else {
		box, err := secretbox.New(cfg.MFAEncryptionKey)
		if err != nil {
			logger.Log.Fatal(
				"invalid MFA_ENCRYPTION_KEY",
				zap.Error(err),
			)
		}
		options.Box = box
	}

	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		logger.Log.Warn("GOOGLE_CLIENT_ID or GOOGLE_CLIENT_SECRET is not set: Google sign-in is off")
	} else {
		options.Google = google.New(google.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
		})
	}

	userHandler := user.NewHandler(user.NewService(user.NewRepository(db), tokens, options), logger.Log)

	// Emails and events (outbox.QueueEmail, outbox.QueueUserEmailVerified)
	// wait in the outbox table until a relay puts them on their queues.
	// Stopped with the server.
	relayCtx, stopRelay := context.WithCancel(context.Background())
	defer stopRelay()

	if cfg.StorageConnectionString == "" {
		logger.Log.Warn("STORAGE_CONNECTION_STRING is not set: outbox messages are kept but not sent")
	} else {
		sender, err := outbox.NewQueueSender(cfg.StorageConnectionString)
		if err != nil {
			logger.Log.Fatal(
				"invalid STORAGE_CONNECTION_STRING",
				zap.Error(err),
			)
		}
		go outbox.NewRelay(outbox.NewStore(db), sender, logger.Log).Run(relayCtx)
	}

	r := router.SetupRouter(db, userHandler)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,

		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {

		logger.Log.Info(
			"Server started",
			zap.String("port", cfg.Port),
		)

		// http.ErrServerClosed is the expected result of a graceful shutdown.
		if err := srv.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {

			logger.Log.Fatal(
				"cannot start server",
				zap.Error(err),
			)
		}
	}()

	quit := make(chan os.Signal, 1)

	// os.Interrupt: Ctrl+C; SIGTERM: sent by Docker / Container Apps on stop.
	signal.Notify(
		quit,
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer signal.Stop(quit)

	<-quit

	logger.Log.Info("Shutting down server...")

	stopRelay()

	// Allow in-flight requests up to 5 seconds to complete.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal(
			"server forced to shutdown",
			zap.Error(err),
		)
	}

	logger.Log.Info("Server stopped gracefully")
}
