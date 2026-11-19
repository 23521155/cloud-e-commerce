package config

import "testing"

var sepayKeys = []string{
	"SEPAY_API_KEY", "SEPAY_BANK", "SEPAY_ACCOUNT_NO", "SEPAY_ACCOUNT_NAME", "SEPAY_CODE_PREFIX",
}

// clearSePayEnv is clearEnv for the SePay keys too.
func clearSePayEnv(t *testing.T) {
	t.Helper()

	clearEnv(t)
	for _, key := range sepayKeys {
		t.Setenv(key, "")
	}
}

func TestLoadSePayIsOffByDefault(t *testing.T) {
	clearSePayEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.SePay.Enabled() {
		t.Error("SePay must be off without an account")
	}
	if cfg.SePay.CodePrefix != "MA" {
		t.Errorf("CodePrefix = %q, want MA", cfg.SePay.CodePrefix)
	}
}

func TestLoadReadsTheSePaySettings(t *testing.T) {
	clearSePayEnv(t)

	t.Setenv("SEPAY_API_KEY", "s3cret")
	t.Setenv("SEPAY_BANK", "BIDV")
	t.Setenv("SEPAY_ACCOUNT_NO", "96247SHOPVA")
	t.Setenv("SEPAY_ACCOUNT_NAME", "NGUYEN VAN A")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	s := cfg.SePay
	if !s.Enabled() || s.APIKey != "s3cret" || s.Bank != "BIDV" || s.AccountNo != "96247SHOPVA" ||
		s.AccountName != "NGUYEN VAN A" || s.CodePrefix != "MA" {
		t.Errorf("SePay = %+v", s)
	}
}

func TestLoadRejectsHalfASePayAccount(t *testing.T) {

	for _, key := range []string{"SEPAY_API_KEY", "SEPAY_BANK", "SEPAY_ACCOUNT_NO", "SEPAY_ACCOUNT_NAME"} {
		t.Run(key, func(t *testing.T) {
			clearSePayEnv(t)
			t.Setenv(key, "only-this")

			if _, err := Load(); err == nil {
				t.Fatalf("Load accepted %s without the other SePay keys", key)
			}
		})
	}
}
