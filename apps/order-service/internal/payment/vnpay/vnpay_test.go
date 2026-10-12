package vnpay

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

const testSecret = "TESTSECRET"

func newTestClient(t *testing.T) *Client {
	t.Helper()

	c, err := New(Config{
		TmnCode:    "TESTTMN1",
		HashSecret: testSecret,
		PayURL:     "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html",
		ReturnURL:  "http://localhost:3000/checkout/vnpay-return",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// 2026-10-12 03:04:05 UTC is 10:04:05 in Vietnam.
	c.now = func() time.Time { return time.Date(2026, 10, 12, 3, 4, 5, 0, time.UTC) }

	return c
}

// signed returns params with a valid vnp_SecureHash, as VNPay would send them.
func signed(params url.Values) url.Values {
	out := url.Values{}
	for k, v := range params {
		out[k] = v
	}
	out.Set(paramSecureHash, sign(testSecret, canonical(params)))
	return out
}

// paidAnswer is a successful payment of 150,000 VND for order ABC123.
func paidAnswer() url.Values {
	return url.Values{
		"vnp_TmnCode":           {"TESTTMN1"},
		"vnp_TxnRef":            {"ABC123"},
		"vnp_Amount":            {"15000000"},
		"vnp_OrderInfo":         {"Thanh toan don ABC123"},
		"vnp_ResponseCode":      {"00"},
		"vnp_TransactionStatus": {"00"},
		"vnp_TransactionNo":     {"14567890"},
		"vnp_BankCode":          {"NCB"},
	}
}

func TestNewRequiresEveryField(t *testing.T) {
	if _, err := New(Config{TmnCode: "X", HashSecret: "Y", PayURL: "Z"}); err == nil {
		t.Fatal("New without ReturnURL: want an error")
	}
}

func TestCanonicalSortsAndEncodes(t *testing.T) {
	got := canonical(url.Values{
		"vnp_TxnRef":    {"ABC123"},
		"vnp_OrderInfo": {"Thanh toan don ABC123"},
		"vnp_ReturnUrl": {"http://localhost:3000/a?b=c"},
		"vnp_Amount":    {"15000000"},
	})

	want := "vnp_Amount=15000000" +
		"&vnp_OrderInfo=Thanh+toan+don+ABC123" +
		"&vnp_ReturnUrl=http%3A%2F%2Flocalhost%3A3000%2Fa%3Fb%3Dc" +
		"&vnp_TxnRef=ABC123"

	if got != want {
		t.Fatalf("canonical:\n got %s\nwant %s", got, want)
	}
}

func TestSignKnownVector(t *testing.T) {
	// Worked out separately with crypto/hmac.
	const want = "4041c7d720e192c1fd453ecfc7bc945d041a3736358e70cb30938e31ece38c91" +
		"7de7116b1c012ff747b5519052a52396d02212c57600ade306ab8d34c17d8c84"

	got := sign(testSecret, "vnp_Amount=15000000&vnp_OrderInfo=Thanh+toan+don+ABC123&vnp_TxnRef=ABC123")
	if got != want {
		t.Fatalf("sign:\n got %s\nwant %s", got, want)
	}
}

func TestBuildPayURL(t *testing.T) {
	c := newTestClient(t)

	raw, err := c.BuildPayURL(PayRequest{
		TxnRef:    "ABC123",
		AmountVND: 150_000,
		OrderInfo: "Thanh toan don ABC123",
		IPAddr:    "127.0.0.1",
		ExpiresAt: time.Date(2026, 10, 12, 3, 19, 5, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("BuildPayURL: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("not a URL: %v", err)
	}
	if got := u.Scheme + "://" + u.Host + u.Path; got != c.cfg.PayURL {
		t.Errorf("page = %s, want %s", got, c.cfg.PayURL)
	}

	q := u.Query()
	want := map[string]string{
		"vnp_Version":    "2.1.0",
		"vnp_Command":    "pay",
		"vnp_TmnCode":    "TESTTMN1",
		"vnp_Amount":     "15000000", // times 100
		"vnp_CurrCode":   "VND",
		"vnp_TxnRef":     "ABC123",
		"vnp_OrderInfo":  "Thanh toan don ABC123",
		"vnp_OrderType":  "other",
		"vnp_Locale":     "vn",
		"vnp_ReturnUrl":  "http://localhost:3000/checkout/vnpay-return",
		"vnp_IpAddr":     "127.0.0.1",
		"vnp_CreateDate": "20261012100405", // Vietnam time
		"vnp_ExpireDate": "20261012101905",
	}
	for k, v := range want {
		if got := q.Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}

	// The URL must pass our own check, as VNPay's check works the same way.
	hash := q.Get(paramSecureHash)
	q.Del(paramSecureHash)
	if want := sign(testSecret, canonical(q)); hash != want {
		t.Errorf("vnp_SecureHash = %s, want %s", hash, want)
	}
}

func TestVerifyPaid(t *testing.T) {
	c := newTestClient(t)

	res, err := c.Verify(signed(paidAnswer()))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	want := Result{
		TxnRef:            "ABC123",
		AmountVND:         150_000,
		ResponseCode:      "00",
		TransactionStatus: "00",
		TransactionNo:     "14567890",
	}
	if res != want {
		t.Fatalf("Verify = %+v, want %+v", res, want)
	}
	if !res.Succeeded() {
		t.Error("Succeeded = false, want true")
	}
}

func TestVerifyCancelledByCustomer(t *testing.T) {
	c := newTestClient(t)

	answer := paidAnswer()
	answer.Set("vnp_ResponseCode", "24") // customer cancelled
	answer.Set("vnp_TransactionStatus", "02")

	res, err := c.Verify(signed(answer))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if res.Succeeded() {
		t.Error("Succeeded = true, want false")
	}
}

func TestVerifyIgnoresSecureHashType(t *testing.T) {
	c := newTestClient(t)

	q := signed(paidAnswer())
	q.Set(paramSecureHashType, "HmacSHA512")

	if _, err := c.Verify(q); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyAcceptsUpperCaseHash(t *testing.T) {
	c := newTestClient(t)

	q := signed(paidAnswer())
	q.Set(paramSecureHash, strings.ToUpper(q.Get(paramSecureHash)))

	if _, err := c.Verify(q); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyDoesNotChangeTheQuery(t *testing.T) {
	c := newTestClient(t)

	q := signed(paidAnswer())
	if _, err := c.Verify(q); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if q.Get(paramSecureHash) == "" {
		t.Error("Verify removed vnp_SecureHash from the caller's query")
	}
}

func TestVerifyRefuses(t *testing.T) {
	c := newTestClient(t)

	tests := map[string]func() url.Values{
		"missing hash": func() url.Values {
			return paidAnswer()
		},
		"amount changed after signing": func() url.Values {
			q := signed(paidAnswer())
			q.Set("vnp_Amount", "100")
			return q
		},
		"failure turned into success": func() url.Values {
			answer := paidAnswer()
			answer.Set("vnp_ResponseCode", "24")
			q := signed(answer)
			q.Set("vnp_ResponseCode", "00")
			return q
		},
		"signed with another secret": func() url.Values {
			q := paidAnswer()
			q.Set(paramSecureHash, sign("OTHER", canonical(q)))
			return q
		},
	}

	for name, query := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := c.Verify(query()); !errors.Is(err, ErrInvalidSignature) {
				t.Fatalf("Verify err = %v, want ErrInvalidSignature", err)
			}
		})
	}
}

func TestVerifyRefusesBadAmount(t *testing.T) {
	c := newTestClient(t)

	answer := paidAnswer()
	answer.Set("vnp_Amount", "abc")

	if _, err := c.Verify(signed(answer)); err == nil {
		t.Fatal("Verify with a non-numeric amount: want an error")
	}
}
