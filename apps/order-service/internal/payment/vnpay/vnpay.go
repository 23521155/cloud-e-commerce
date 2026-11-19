// Package vnpay builds VNPay payment URLs and checks the answers VNPay sends
// back (the IPN call and the customer's return redirect), following the
// VNPay payment API v2.1.0.
//
// Both directions are signed the same way: the vnp_* parameters, sorted by
// name and URL-encoded, joined with "&", then HMAC-SHA512 with the merchant's
// hash secret, written as hex.
package vnpay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrInvalidSignature means the parameters were not signed with our hash
// secret: they did not come from VNPay, or were changed on the way.
var ErrInvalidSignature = errors.New("vnpay: invalid signature")

// Fixed values of a payment request.
const (
	version   = "2.1.0"
	command   = "pay"
	currency  = "VND"
	locale    = "vn"
	orderType = "other"
)

// Names of the signature parameters. They are never part of the signed data.
const (
	paramSecureHash     = "vnp_SecureHash"
	paramSecureHashType = "vnp_SecureHashType"
)

// timeLayout is how VNPay writes a time (yyyyMMddHHmmss), always in Vietnam
// time.
const timeLayout = "20060102150405"

// vietnam is GMT+7. A fixed zone, so the container needs no tzdata.
var vietnam = time.FixedZone("ICT", 7*60*60)

// Codes that mean the payment went through.
const (
	responseSuccess    = "00" // vnp_ResponseCode
	transactionSuccess = "00" // vnp_TransactionStatus
)

// Config is the merchant account given by VNPay at registration.
type Config struct {
	// TmnCode is the merchant (terminal) code, vnp_TmnCode.
	TmnCode string

	// HashSecret signs every request and checks every answer. Never log it.
	HashSecret string

	// PayURL is the payment page, e.g.
	// https://sandbox.vnpayment.vn/paymentv2/vpcpay.html in the sandbox.
	PayURL string

	// ReturnURL is where VNPay sends the customer after paying (a web page).
	ReturnURL string
}

// Client signs and checks VNPay parameters.
type Client struct {
	cfg Config

	// now is time.Now; tests replace it.
	now func() time.Time
}

// New checks cfg and returns a Client.
func New(cfg Config) (*Client, error) {

	if cfg.TmnCode == "" || cfg.HashSecret == "" || cfg.PayURL == "" || cfg.ReturnURL == "" {
		return nil, errors.New("vnpay: TmnCode, HashSecret, PayURL and ReturnURL are required")
	}

	return &Client{cfg: cfg, now: time.Now}, nil
}

// PayRequest is one payment to start.
type PayRequest struct {
	// TxnRef is our reference, unique per payment: the order code.
	TxnRef string

	// AmountVND is the amount in dong. VNPay wants it times 100.
	AmountVND int64

	// OrderInfo is shown to the customer. Plain ASCII, no accents.
	OrderInfo string

	// IPAddr is the customer's IP address.
	IPAddr string

	// ExpiresAt is when the payment page stops accepting the payment; the
	// same as the order's hold.
	ExpiresAt time.Time
}

// Result is what VNPay says about one payment, read from a signed answer.
type Result struct {
	TxnRef            string // vnp_TxnRef, our order code
	AmountVND         int64  // vnp_Amount / 100
	ResponseCode      string // vnp_ResponseCode
	TransactionStatus string // vnp_TransactionStatus
	TransactionNo     string // vnp_TransactionNo, VNPay's id of the payment
}

// Succeeded reports whether the customer paid.
func (r Result) Succeeded() bool {
	return r.ResponseCode == responseSuccess && r.TransactionStatus == transactionSuccess
}

// BuildPayURL returns the URL of VNPay's payment page for r, signed.
func (c *Client) BuildPayURL(r PayRequest) (string, error) {
	if r.TxnRef == "" {
		return "", errors.New("vnpay: TxnRef is required")
	}
	if r.AmountVND <= 0 {
		return "", errors.New("vnpay: AmountVND must be positive")
	}
	if r.ExpiresAt.IsZero() {
		return "", errors.New("vnpay: ExpiresAt is required")
	}

	params := url.Values{}
	params.Set("vnp_Version", version)
	params.Set("vnp_Command", command)
	params.Set("vnp_TmnCode", c.cfg.TmnCode)
	params.Set("vnp_Amount", strconv.FormatInt(r.AmountVND*100, 10))
	params.Set("vnp_CurrCode", currency)
	params.Set("vnp_TxnRef", r.TxnRef)
	params.Set("vnp_OrderInfo", r.OrderInfo)
	params.Set("vnp_OrderType", orderType)
	params.Set("vnp_Locale", locale)
	params.Set("vnp_ReturnUrl", c.cfg.ReturnURL)
	params.Set("vnp_IpAddr", r.IPAddr)
	params.Set("vnp_CreateDate", c.now().In(vietnam).Format(timeLayout))
	params.Set("vnp_ExpireDate", r.ExpiresAt.In(vietnam).Format(timeLayout))

	data := canonical(params)
	hash := sign(c.cfg.HashSecret, data)

	return c.cfg.PayURL + "?" + data + "&vnp_SecureHash=" + hash, nil
}

// Verify checks the signature of the parameters VNPay sent (the query string
// of the IPN call or of the return redirect) and reads the payment out of
// them. A missing or wrong signature is ErrInvalidSignature.
func (c *Client) Verify(query url.Values) (Result, error) {
	got := query.Get(paramSecureHash)

	if got == "" {
		return Result{}, ErrInvalidSignature
	}

	signed := make(url.Values, len(query))

	for name, values := range query {
		if name == paramSecureHash || name == paramSecureHashType {
			continue
		}
		signed[name] = values
	}
	expected := sign(c.cfg.HashSecret, canonical(signed))

	if !hmac.Equal([]byte(strings.ToLower(got)), []byte(expected)) {
		return Result{}, ErrInvalidSignature
	}

	amount, err := strconv.ParseInt(query.Get("vnp_Amount"), 10, 64)
	if err != nil || amount < 0 || amount%100 != 0 {
		return Result{}, fmt.Errorf("vnpay: invalid vnp_Amount %q", query.Get("vnp_Amount"))
	}

	return Result{
		TxnRef:            query.Get("vnp_TxnRef"),
		AmountVND:         amount / 100,
		ResponseCode:      query.Get("vnp_ResponseCode"),
		TransactionStatus: query.Get("vnp_TransactionStatus"),
		TransactionNo:     query.Get("vnp_TransactionNo"),
	}, nil
}

// canonical is the data that gets signed: every parameter, sorted by name,
// as name=value with both URL-encoded (a space is "+"), joined with "&".

func canonical(params url.Values) string {
	return params.Encode()
}

// sign is HMAC-SHA512 of data with secret, as lower-case hex.

func sign(secret, data string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
