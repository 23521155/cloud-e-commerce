package config

import (
	"strings"
	"testing"
	"time"
)

// clearEnv gives each test a clean slate, whatever the machine has set.
func clearEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"APP_NAME", "PORT", "DATABASE_URL",
		"CATALOG_GRPC_ADDR", "CATALOG_GRPC_TLS", "INTERNAL_TOKEN", "PAYMENT_HOLD_MINUTES",
		"VNPAY_TMN_CODE", "VNPAY_HASH_SECRET", "VNPAY_PAY_URL", "VNPAY_RETURN_URL",
		"VNPAY_CONFIRM_ON_RETURN",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Port != "8084" || cfg.AppName != "order-service" {
		t.Errorf("port/app = %q/%q", cfg.Port, cfg.AppName)
	}
	if cfg.CatalogGRPCAddr != "localhost:9083" {
		t.Errorf("CatalogGRPCAddr = %q", cfg.CatalogGRPCAddr)
	}
	if cfg.CatalogGRPCTLS {
		t.Error("TLS must be off by default: local containers speak plain text")
	}
	if cfg.InternalToken != "" {
		t.Errorf("InternalToken = %q: there must be no default secret", cfg.InternalToken)
	}
	if cfg.PaymentHold != 15*time.Minute {
		t.Errorf("PaymentHold = %v, want 15m", cfg.PaymentHold)
	}
	if cfg.VNPay.Enabled() {
		t.Error("VNPay must be off without an account")
	}
	if cfg.VNPay.PayURL != "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html" {
		t.Errorf("VNPay.PayURL = %q, want the sandbox", cfg.VNPay.PayURL)
	}
	if cfg.VNPay.ConfirmOnReturn {
		t.Error("ConfirmOnReturn must be off by default: only the IPN confirms a payment")
	}
}

func TestLoadReadsTheVNPaySettings(t *testing.T) {
	clearEnv(t)

	t.Setenv("VNPAY_TMN_CODE", "TESTTMN1")
	t.Setenv("VNPAY_HASH_SECRET", "s3cret")
	t.Setenv("VNPAY_RETURN_URL", "https://shop.example/checkout/vnpay-return")
	t.Setenv("VNPAY_CONFIRM_ON_RETURN", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	v := cfg.VNPay
	if !v.Enabled() || v.TmnCode != "TESTTMN1" || v.HashSecret != "s3cret" ||
		v.ReturnURL != "https://shop.example/checkout/vnpay-return" || !v.ConfirmOnReturn {
		t.Errorf("VNPay = %+v", v)
	}
}

func TestLoadRejectsHalfAVNPayAccount(t *testing.T) {

	for _, key := range []string{"VNPAY_TMN_CODE", "VNPAY_HASH_SECRET"} {
		t.Run(key, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(key, "only-this")

			if _, err := Load(); err == nil {
				t.Fatalf("Load accepted %s without the other VNPay key", key)
			}
		})
	}
}

func TestLoadReadsTheCatalogSettings(t *testing.T) {
	clearEnv(t)

	t.Setenv("CATALOG_GRPC_ADDR", "catalog-service:9083")
	t.Setenv("CATALOG_GRPC_TLS", "true")
	t.Setenv("INTERNAL_TOKEN", "s3cret")
	t.Setenv("PAYMENT_HOLD_MINUTES", "30")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.CatalogGRPCAddr != "catalog-service:9083" || !cfg.CatalogGRPCTLS ||
		cfg.InternalToken != "s3cret" || cfg.PaymentHold != 30*time.Minute {
		t.Errorf("config = %+v", cfg)
	}
}

func TestLoadRejectsValuesItCannotUse(t *testing.T) {

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"tls typo", "CATALOG_GRPC_TLS", "ture"},
		{"hold is not a number", "PAYMENT_HOLD_MINUTES", "fifteen"},
		{"hold is zero", "PAYMENT_HOLD_MINUTES", "0"},
		{"hold is negative", "PAYMENT_HOLD_MINUTES", "-5"},
		{"hold is longer than the catalogue allows", "PAYMENT_HOLD_MINUTES", "61"},
		{"hold has a unit", "PAYMENT_HOLD_MINUTES", "15m"},
		{"confirm on return typo", "VNPAY_CONFIRM_ON_RETURN", "yes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(tt.key, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load accepted %s=%q", tt.key, tt.value)
			}
			// The message must name the variable, or nobody can fix it.
			if !strings.Contains(err.Error(), tt.key) {
				t.Errorf("error %q does not name %s", err, tt.key)
			}
		})
	}
}

func TestPaymentHoldBounds(t *testing.T) {

	for value, want := range map[string]time.Duration{"1": time.Minute, "60": time.Hour} {
		t.Run(value, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("PAYMENT_HOLD_MINUTES", value)

			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.PaymentHold != want {
				t.Fatalf("PaymentHold = %v, want %v", cfg.PaymentHold, want)
			}
		})
	}
}
