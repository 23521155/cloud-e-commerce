package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fizzisme/catalog-service/internal/book"
	"github.com/fizzisme/catalog-service/internal/config"
	"github.com/fizzisme/catalog-service/internal/database"
	"github.com/fizzisme/catalog-service/internal/grpcserver"
	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/fizzisme/catalog-service/internal/router"
	"github.com/fizzisme/catalog-service/internal/subject"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// main is the entry point of the catalog service. It wires up config
// and logging, starts the HTTP server and the gRPC server, and handles
// graceful shutdown on SIGINT/SIGTERM.
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

	bookRepo := book.NewRepository(db)
	bookHandler := book.NewHandler(book.NewService(bookRepo))
	subjectHandler := subject.NewHandler(subject.NewRepository(db))

	r := router.SetupRouter(db, bookHandler, subjectHandler)

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

	// Other services reserve and sell books over gRPC. Without the shared
	// secret the port would be open to anyone who can reach it, so the
	// server does not start.
	var grpcSrv *grpc.Server

	if cfg.InternalToken == "" {
		logger.Log.Warn("INTERNAL_TOKEN is not set: the gRPC server is disabled")
	} else {

		lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
		if err != nil {
			logger.Log.Fatal(
				"cannot listen for gRPC",
				zap.String("port", cfg.GRPCPort),
				zap.Error(err),
			)
		}

		grpcSrv = grpcserver.New(cfg.InternalToken, book.NewReservationService(bookRepo))

		go func() {

			logger.Log.Info(
				"gRPC server started",
				zap.String("port", cfg.GRPCPort),
			)

			// Serve returns nil after GracefulStop or Stop.
			if err := grpcSrv.Serve(lis); err != nil {
				logger.Log.Fatal(
					"cannot start gRPC server",
					zap.Error(err),
				)
			}
		}()
	}

	quit := make(chan os.Signal, 1)

	// os.Interrupt: Ctrl+C; SIGTERM: sent by Docker / App Service on stop.
	signal.Notify(
		quit,
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer signal.Stop(quit)

	<-quit

	logger.Log.Info("Shutting down server...")

	// Allow in-flight requests up to 5 seconds to complete.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	if grpcSrv != nil {

		stopped := make(chan struct{})

		// GracefulStop waits for running calls, so a reservation that has
		// started is finished. A client that never ends its call must not
		// hold the shutdown, hence the deadline.
		go func() {
			grpcSrv.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
		case <-ctx.Done():
			grpcSrv.Stop()
		}
	}

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal(
			"server forced to shutdown",
			zap.Error(err),
		)
	}

	logger.Log.Info("Server stopped gracefully")
}
