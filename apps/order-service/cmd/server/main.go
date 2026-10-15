package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fizzisme/order-service/internal/accountevents"
	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/fizzisme/order-service/internal/config"
	"github.com/fizzisme/order-service/internal/database"
	"github.com/fizzisme/order-service/internal/logger"
	"github.com/fizzisme/order-service/internal/order"
	"github.com/fizzisme/order-service/internal/payment/vnpay"
	"github.com/fizzisme/order-service/internal/router"
	"go.uber.org/zap"
)

// main is the entry point of the order service. It wires up config
// and logging, starts the HTTP server, and handles graceful shutdown
// on SIGINT/SIGTERM.
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

	// The connection is lazy, so the service starts even when the catalogue
	// is not up yet; a missing token, though, stops it here.
	catalog, err := catalogclient.New(catalogclient.Config{
		Addr:   cfg.CatalogGRPCAddr,
		Token:  cfg.InternalToken,
		UseTLS: cfg.CatalogGRPCTLS,
	})
	if err != nil {
		logger.Log.Fatal(
			"cannot create catalogue client",
			zap.Error(err),
		)
	}
	defer catalog.Close()

	// Without a VNPay account the service still runs: online payment methods
	// are refused and the payment routes are not mounted.
	var payments order.Payments
	var gateway *vnpay.Client
	if cfg.VNPay.Enabled() {
		gateway, err = vnpay.New(vnpay.Config{
			TmnCode:    cfg.VNPay.TmnCode,
			HashSecret: cfg.VNPay.HashSecret,
			PayURL:     cfg.VNPay.PayURL,
			ReturnURL:  cfg.VNPay.ReturnURL,
		})
		if err != nil {
			logger.Log.Fatal(
				"cannot create VNPay client",
				zap.Error(err),
			)
		}
		payments = gateway
	} else {
		logger.Log.Warn("VNPay is not configured, online payments are off")
	}

	service := order.NewService(order.NewRepository(db), catalog, payments, cfg.PaymentHold, logger.Log)

	// Cancels the online orders nobody paid in time and gives their books
	// back. It runs even without VNPay: orders placed earlier may still wait.
	expiryCtx, stopExpiry := context.WithCancel(context.Background())
	defer stopExpiry()
	go service.RunExpiry(expiryCtx, time.Minute)

	// Gives readers the orders they placed as guests once the user service
	// says they confirmed the address (internal/accountevents).
	eventsCtx, stopEvents := context.WithCancel(context.Background())
	defer stopEvents()

	if cfg.StorageConnectionString == "" {
		logger.Log.Warn("STORAGE_CONNECTION_STRING is not set: guest orders are not linked to accounts")
	} else {
		queue, err := accountevents.NewStorageQueue(cfg.StorageConnectionString, accountevents.QueueEmailVerified)
		if err != nil {
			logger.Log.Fatal(
				"invalid STORAGE_CONNECTION_STRING",
				zap.Error(err),
			)
		}
		go accountevents.NewConsumer(queue, service, logger.Log).Run(eventsCtx)
	}

	routes := []router.Registrar{order.NewHandler(service, logger.Log)}
	if gateway != nil {
		routes = append(routes, order.NewPaymentHandler(service, gateway, cfg.VNPay.ConfirmOnReturn, logger.Log))
	}

	r := router.SetupRouter(db, routes...)

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

	// os.Interrupt: Ctrl+C; SIGTERM: sent by Docker / App Service on stop.
	signal.Notify(
		quit,
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer signal.Stop(quit)

	<-quit

	logger.Log.Info("Shutting down server...")

	stopExpiry()
	stopEvents()

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
