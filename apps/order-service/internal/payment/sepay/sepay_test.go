package sepay

import (
	"encoding/json"
	"errors"
	"net/url"
	"testing"
)

const testKey = "TESTKEY"

func validConfig() Config {
	return Config{
		APIKey:      testKey,
		Bank:        "BIDV",
		AccountNo:   "96247SHOPVA",
		AccountName: "NGUYEN VAN A",
		CodePrefix:  "MA",
	}
}

func newTestClient(t *testing.T) *Client {
	t.Helper()

	c, err := New(validConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNewRequiresEveryField(t *testing.T) {
	cases := map[string]func(*Config){
		"APIKey":      func(c *Config) { c.APIKey = "" },
		"Bank":        func(c *Config) { c.Bank = "" },
		"AccountNo":   func(c *Config) { c.AccountNo = "" },
		"AccountName": func(c *Config) { c.AccountName = "" },
		"CodePrefix":  func(c *Config) { c.CodePrefix = "" },
	}

	for name, empty := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := validConfig()
			empty(&cfg)

			if _, err := New(cfg); err == nil {
				t.Fatalf("New without %s: want an error", name)
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	c := newTestClient(t)

	cases := []struct {
		name   string
		header string
		ok     bool
	}{
		{"right key", "Apikey TESTKEY", true},
		{"scheme in lower case", "apikey TESTKEY", true},
		{"wrong key", "Apikey WRONG", false},
		{"empty header", "", false},
		{"no scheme", "TESTKEY", false},
		{"other scheme", "Bearer TESTKEY", false},
		{"key with a suffix", "Apikey TESTKEY2", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.Authenticate(tc.header)

			if tc.ok && err != nil {
				t.Fatalf("Authenticate(%q): %v", tc.header, err)
			}
			if !tc.ok && !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("Authenticate(%q) = %v, want ErrUnauthorized", tc.header, err)
			}
		})
	}
}

func TestQRURL(t *testing.T) {
	c := newTestClient(t)

	got, err := c.QRURL("MA-123456", 125000)
	if err != nil {
		t.Fatalf("QRURL: %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("QRURL gave an invalid URL %q: %v", got, err)
	}

	if u.Scheme != "https" || u.Host != "img.vietqr.io" {
		t.Errorf("host: got %s://%s", u.Scheme, u.Host)
	}
	if u.Path != "/image/BIDV-96247SHOPVA-compact2.png" {
		t.Errorf("path: got %s", u.Path)
	}

	q := u.Query()
	want := map[string]string{
		"amount":      "125000",
		"addInfo":     "MA123456", // no dash
		"accountName": "NGUYEN VAN A",
	}
	for name, v := range want {
		if q.Get(name) != v {
			t.Errorf("%s: got %q, want %q", name, q.Get(name), v)
		}
	}
}

func TestQRURLRefusesBadInput(t *testing.T) {
	c := newTestClient(t)

	if _, err := c.QRURL("", 125000); err == nil {
		t.Error("empty order code: want an error")
	}
	if _, err := c.QRURL("MA-123456", 0); err == nil {
		t.Error("amount 0: want an error")
	}
	if _, err := c.QRURL("MA-123456", -5); err == nil {
		t.Error("negative amount: want an error")
	}
}

func TestOrderCode(t *testing.T) {
	c := newTestClient(t)

	cases := []struct {
		name    string
		code    string // Webhook.Code
		content string // Webhook.Content
		want    string
	}{
		// The real shape of a BIDV line: sender account, sender phone, text.
		{"after the bank's prefix", "", "111122223333 0900000000 MA123456", "MA-123456"},
		{"with the dash", "", "MA-123456", "MA-123456"},
		{"lower case", "", "thanh toan ma123456", "MA-123456"},
		{"words around it", "", "thanh toan MA123456 cam on", "MA-123456"},
		{"in SePay's own code field", "MA123456", "111122223333 0900000000", "MA-123456"},
		{"code field first", "MA111111", "MA222222", "MA-111111"},
		{"seven digits", "", "MA1234567", ""},
		{"five digits", "", "MA12345", ""},
		{"no code", "", "111122223333 0900000000 ung ho", ""},
		{"empty", "", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := c.OrderCode(Webhook{Code: tc.code, Content: tc.content})

			if ok != (tc.want != "") {
				t.Fatalf("found = %v, want %v (got %q)", ok, tc.want != "", got)
			}
			if got != tc.want {
				t.Fatalf("OrderCode = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAccepts(t *testing.T) {
	c := newTestClient(t)

	cases := []struct {
		name string
		w    Webhook
		want bool
	}{
		{"money in on our VA", Webhook{TransferType: "in", SubAccount: "96247SHOPVA"}, true},
		{"money out", Webhook{TransferType: "out", SubAccount: "96247SHOPVA"}, false},
		{"another VA", Webhook{TransferType: "in", SubAccount: "96247OTHER"}, false},
		{"no VA", Webhook{TransferType: "in"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.Accepts(tc.w); got != tc.want {
				t.Fatalf("Accepts = %v, want %v", got, tc.want)
			}
		})
	}
}

// The example of SePay's documentation, with the line our bank sends.
func TestWebhookDecodes(t *testing.T) {
	body := `{
		"id": 92704,
		"gateway": "BIDV",
		"transactionDate": "2026-10-18 23:32:58",
		"accountNumber": "1234567890",
		"subAccount": "96247SHOPVA",
		"code": null,
		"content": "111122223333 0900000000 MA123456",
		"transferType": "in",
		"description": "BankAPINotify 111122223333 0900000000 MA123456",
		"transferAmount": 125000,
		"referenceCode": "MBVCB.3278907687",
		"accumulated": 0
	}`

	var w Webhook
	if err := json.Unmarshal([]byte(body), &w); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if w.ID != 92704 || w.AmountVND != 125000 || w.SubAccount != "96247SHOPVA" ||
		w.TransferType != "in" || w.ReferenceCode != "MBVCB.3278907687" {
		t.Fatalf("decoded wrong: %+v", w)
	}

	c := newTestClient(t)
	if !c.Accepts(w) {
		t.Error("Accepts: want true")
	}
	if code, ok := c.OrderCode(w); !ok || code != "MA-123456" {
		t.Errorf("OrderCode = %q, %v", code, ok)
	}
}

func TestInstructions(t *testing.T) {
	c := newTestClient(t)

	got, err := c.Instructions("MA-123456", 125000)
	if err != nil {
		t.Fatalf("Instructions: %v", err)
	}

	image, err := c.QRURL("MA-123456", 125000)
	if err != nil {
		t.Fatal(err)
	}

	want := Instructions{
		ImageURL:    image,
		Bank:        "BIDV",
		AccountNo:   "96247SHOPVA",
		AccountName: "NGUYEN VAN A",
		AmountVND:   125000,
		Content:     "MA123456",
	}
	if got != want {
		t.Fatalf("Instructions:\n got %+v\nwant %+v", got, want)
	}

	if _, err := c.Instructions("", 125000); err == nil {
		t.Error("empty order code: want an error")
	}
}
