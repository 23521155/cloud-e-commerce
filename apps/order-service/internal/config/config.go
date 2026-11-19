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

	SePay SePay

	// StorageConnectionString is the Azure Storage account whose
	// user-email-verified queue the user service writes (Azurite locally).
	// Empty: guest orders are not linked to accounts.
	StorageConnectionString string
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

// SePay is the bank account VietQR orders are paid into, and the SePay webhook
// that reports the transfers. VietQR orders are paid by bank transfer while
// SEPAY_API_KEY is set; without it they go through VNPay like a card.
type SePay struct {
	// APIKey is the key set on the SePay webhook. Never log it.
	APIKey string

	// Bank is the bank's short name in VietQR, e.g. BIDV.
	Bank string

	// AccountNo is the account the customer pays into. For BIDV, the virtual
	// account (VA) SePay watches.
	AccountNo string

	// AccountName is the owner's name as the bank spells it.
	AccountName string

	// CodePrefix is how the order codes start, without the dash. It must be
	// the prefix of the codes the service draws (MA-123456 is "MA").
	CodePrefix string
}

// Enabled reports whether bank transfers are configured.
func (s SePay) Enabled() bool {
	return s.APIKey != ""
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

	sepay := SePay{
		APIKey:      os.Getenv("SEPAY_API_KEY"),
		Bank:        os.Getenv("SEPAY_BANK"),
		AccountNo:   os.Getenv("SEPAY_ACCOUNT_NO"),
		AccountName: os.Getenv("SEPAY_ACCOUNT_NAME"),
		CodePrefix:  getEnv("SEPAY_CODE_PREFIX", "MA"),
	}

	// All four or none: a half-copied account would put a QR on the page that
	// pays nobody, or a webhook nobody can check.
	if set := countSet(sepay.APIKey, sepay.Bank, sepay.AccountNo, sepay.AccountName); set != 0 && set != 4 {
		return nil, fmt.Errorf("SEPAY_API_KEY, SEPAY_BANK, SEPAY_ACCOUNT_NO and SEPAY_ACCOUNT_NAME must be set together")
	}

	cfg := &Config{
		AppName: getEnv("APP_NAME", "order-service"),

		Port: getEnv("PORT", "8084"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		CatalogGRPCAddr: getEnv("CATALOG_GRPC_ADDR", "localhost:9083"),

		CatalogGRPCTLS: tls,

		InternalToken: os.Getenv("INTERNAL_TOKEN"),

		StorageConnectionString: os.Getenv("STORAGE_CONNECTION_STRING"),

		PaymentHold: time.Duration(holdMinutes) * time.Minute,

		VNPay: vnpay,

		SePay: sepay,
	}

	return cfg, nil
}

// countSet returns how many of values are not empty.
func countSet(values ...string) int {
	n := 0
	for _, v := range values {
		if v != "" {
			n++
		}
	}
	return n
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
