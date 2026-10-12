package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Bounds of PAYMENT_HOLD_MINUTES. The catalogue accepts holds of up to an hour.
const (
	minPaymentHoldMinutes = 1
	maxPaymentHoldMinutes = 60
)

// Config holds all runtime configuration for the order service,
// loaded from environment variables.
type Config struct {
	AppName string

	Port string

	// DatabaseURL is a sqlserver:// URL, also used by golang-migrate.
	DatabaseURL string

	// CatalogGRPCAddr is host:port of the catalogue's gRPC server.
	CatalogGRPCAddr string

	// CatalogGRPCTLS turns on TLS towards the catalogue. Azure App Service
	// only accepts gRPC over TLS; local containers speak plain text.
	CatalogGRPCTLS bool

	// InternalToken is the secret sent to the catalogue as x-internal-token.
	// The catalogue and this service must be given the same value.
	InternalToken string

	// PaymentHold is how long the books of an online order stay reserved
	// while the customer pays.
	PaymentHold time.Duration

	VNPay VNPay
}

// VNPay is the merchant account at VNPay. Online payments are off while
// VNPAY_TMN_CODE and VNPAY_HASH_SECRET are both empty, so the service still
// runs on a machine that has no sandbox account.
type VNPay struct {
	TmnCode    string
	HashSecret string
	PayURL     string

	// ReturnURL is the web page VNPay sends the customer back to.
	ReturnURL string

	// ConfirmOnReturn lets the return redirect confirm a payment, not only
	// the IPN call. Only for local runs, where VNPay cannot reach the IPN.
	ConfirmOnReturn bool
}

// Enabled reports whether a VNPay account is configured.
func (v VNPay) Enabled() bool {
	return v.TmnCode != ""
}

// Load reads environment variables (via a .env file) into a Config.
func Load() (*Config, error) {

	// Missing file is not fatal: in containers, config comes from
	// env vars injected by the runtime, not a .env baked into the image.
	if err := godotenv.Load("configs/.env"); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	tls, err := getEnvBool("CATALOG_GRPC_TLS", false)
	if err != nil {
		return nil, err
	}

	holdMinutes, err := getEnvInt("PAYMENT_HOLD_MINUTES", 15, minPaymentHoldMinutes, maxPaymentHoldMinutes)
	if err != nil {
		return nil, err
	}

	confirmOnReturn, err := getEnvBool("VNPAY_CONFIRM_ON_RETURN", false)
	if err != nil {
		return nil, err
	}

	vnpay := VNPay{
		TmnCode:         os.Getenv("VNPAY_TMN_CODE"),
		HashSecret:      os.Getenv("VNPAY_HASH_SECRET"),
		PayURL:          getEnv("VNPAY_PAY_URL", "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html"),
		ReturnURL:       getEnv("VNPAY_RETURN_URL", "http://localhost:3000/checkout/vnpay-return"),
		ConfirmOnReturn: confirmOnReturn,
	}

	// One without the other is a half-copied account, not "VNPay off".
	if (vnpay.TmnCode == "") != (vnpay.HashSecret == "") {
		return nil, fmt.Errorf("VNPAY_TMN_CODE and VNPAY_HASH_SECRET must be set together")
	}

	cfg := &Config{
		AppName: getEnv("APP_NAME", "order-service"),

		Port: getEnv("PORT", "8084"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		CatalogGRPCAddr: getEnv("CATALOG_GRPC_ADDR", "localhost:9083"),

		CatalogGRPCTLS: tls,

		InternalToken: os.Getenv("INTERNAL_TOKEN"),

		PaymentHold: time.Duration(holdMinutes) * time.Minute,

		VNPay: vnpay,
	}

	return cfg, nil
}

// getEnv returns the value of key, or fallback when it is unset or empty.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvBool reads a boolean ("true", "false", "1", "0"...). An unset or
// empty key gives fallback; a value that is not a boolean is an error, so a
// typo cannot silently turn TLS off.
func getEnvBool(key string, fallback bool) (bool, error) {

	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	b, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false, got %q", key, value)
	}

	return b, nil
}

// getEnvInt reads a whole number between min and max. An unset or empty key
// gives fallback.
func getEnvInt(key string, fallback, min, max int) (int, error) {

	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s must be a whole number from %d to %d, got %q", key, min, max, value)
	}

	return n, nil
}
