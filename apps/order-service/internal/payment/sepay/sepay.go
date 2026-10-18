// Package sepay builds the VietQR image a customer pays an order with, and
// checks the webhook SePay calls when money arrives in the shop's bank
// account.
//
// The shop's BIDV account is personal, so money goes to a virtual account
// (VA) that SePay watches. SePay calls us with the bank's statement line:
// the amount, the VA it came in on, and the text the customer typed. The bank
// puts the sender's account and phone number in front of that text, so the
// order code is found inside it, not compared to all of it.
package sepay

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ErrUnauthorized means the Authorization header is not "Apikey <our key>":
// the call did not come from SePay.
var ErrUnauthorized = errors.New("sepay: unauthorized")

// Config is what the SePay dashboard and the bank give us.
type Config struct {
	// APIKey is the key set on the SePay webhook. Calls must carry it as
	// "Authorization: Apikey <key>". Never log it.
	APIKey string

	// Bank is the bank's short name in VietQR, e.g. BIDV.
	Bank string

	// AccountNo is the virtual account the customer pays into, e.g.
	// 96247SHOPVA. For BIDV it is the VA, not the main account.
	AccountNo string

	// AccountName is the owner's name as the bank spells it, shown by the
	// banking app before the customer confirms.
	AccountName string

	// CodePrefix is how our order codes start, without the dash: "MA" for
	// MA-123456. Banking apps often drop the dash from the transfer text.
	CodePrefix string
}

// Client builds QR images and checks what SePay sends.
type Client struct {
	cfg Config

	// codeRe finds an order code in a transfer text. Built by New.
	codeRe *regexp.Regexp
}

// Webhook is the body SePay POSTs when money moves on the account. Only the
// fields we read are listed.
type Webhook struct {
	ID            int64  `json:"id"`            // SePay's id of the transaction: the key to ignore repeats
	Gateway       string `json:"gateway"`       // bank name
	AccountNumber string `json:"accountNumber"` // the bank account
	SubAccount    string `json:"subAccount"`    // the VA the money came in on
	Code          string `json:"code"`          // payment code SePay recognised, often empty
	Content       string `json:"content"`       // what the customer typed, with the bank's prefix
	TransferType  string `json:"transferType"`  // "in" or "out"
	AmountVND     int64  `json:"transferAmount"`
	ReferenceCode string `json:"referenceCode"` // the bank's reference
}

// New checks cfg and returns a Client.
func New(cfg Config) (*Client, error) {

	required := []struct{ name, value string }{
		{"APIKey", cfg.APIKey},
		{"Bank", cfg.Bank},
		{"AccountNo", cfg.AccountNo},
		{"AccountName", cfg.AccountName},
		{"CodePrefix", cfg.CodePrefix},
	}
	for _, f := range required {
		if f.value == "" {
			return nil, fmt.Errorf("sepay: %s is required", f.name)
		}
	}

	// The prefix, an optional dash, then exactly 6 digits. Go has no
	// lookahead, so a 7th digit is refused by asking for a non-digit or the
	// end of the text after the 6th.
	codeRe, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(cfg.CodePrefix) + `-?(\d{6})(?:\D|$)`)
	if err != nil {
		return nil, fmt.Errorf("sepay: invalid CodePrefix: %w", err)
	}

	return &Client{cfg: cfg, codeRe: codeRe}, nil
}

// Authenticate checks the Authorization header of a webhook call. The scheme
// name is case-insensitive; the key is compared in constant time
// (crypto/subtle). Anything else is ErrUnauthorized.
func (c *Client) Authenticate(authorization string) error {

	scheme, key, ok := strings.Cut(authorization, " ")
	if !ok || !strings.EqualFold(scheme, "Apikey") {
		return ErrUnauthorized
	}

	// A plain == would stop at the first different byte, and the time it
	// takes would tell an attacker how much of the key is right.
	if subtle.ConstantTimeCompare([]byte(key), []byte(c.cfg.APIKey)) != 1 {
		return ErrUnauthorized
	}

	return nil
}

// Instructions is what a customer needs to pay one order by bank transfer:
// the QR image to scan, and the same details to type in by hand.
type Instructions struct {
	ImageURL    string `json:"imageUrl"`
	Bank        string `json:"bank"`
	AccountNo   string `json:"accountNo"`
	AccountName string `json:"accountName"`
	AmountVND   int64  `json:"amountVnd"`

	// Content is the transfer text, the order code without its dash.
	Content string `json:"content"`
}

// Instructions returns how to pay amountVND for the order with orderCode.
func (c *Client) Instructions(orderCode string, amountVND int64) (Instructions, error) {

	image, err := c.QRURL(orderCode, amountVND)
	if err != nil {
		return Instructions{}, err
	}

	return Instructions{
		ImageURL:    image,
		Bank:        c.cfg.Bank,
		AccountNo:   c.cfg.AccountNo,
		AccountName: c.cfg.AccountName,
		AmountVND:   amountVND,
		Content:     strings.ReplaceAll(orderCode, "-", ""),
	}, nil
}

// QRURL returns the address of a VietQR image that pays amountVND to the VA
// with orderCode as the transfer text. The text is the code without its dash
// (MA-123456 becomes MA123456), as that is what survives a banking app.
//
// The image is served by img.vietqr.io:
//
//	https://img.vietqr.io/image/<bank>-<account>-compact2.png?amount=..&addInfo=..&accountName=..
func (c *Client) QRURL(orderCode string, amountVND int64) (string, error) {

	if orderCode == "" {
		return "", errors.New("sepay: orderCode is required")
	}
	if amountVND <= 0 {
		return "", errors.New("sepay: amountVND must be positive")
	}

	// url.Values escapes the owner's name, which has spaces.
	q := url.Values{}
	q.Set("amount", strconv.FormatInt(amountVND, 10))
	q.Set("addInfo", strings.ReplaceAll(orderCode, "-", ""))
	q.Set("accountName", c.cfg.AccountName)

	return "https://img.vietqr.io/image/" + c.cfg.Bank + "-" + c.cfg.AccountNo +
		"-compact2.png?" + q.Encode(), nil
}

// OrderCode finds the order code in a webhook, in the canonical form
// "MA-123456", or reports false when there is none. SePay's own Code field is
// tried first, then Content.
func (c *Client) OrderCode(w Webhook) (string, bool) {

	for _, text := range []string{w.Code, w.Content} {
		if m := c.codeRe.FindStringSubmatch(text); m != nil {
			// m[1] is the 6 digits.
			return strings.ToUpper(c.cfg.CodePrefix) + "-" + m[1], true
		}
	}

	return "", false
}

// Accepts reports whether w is money coming in on our VA. Money going out, or
// coming in on another account, is none of our business.
func (c *Client) Accepts(w Webhook) bool {
	return w.TransferType == "in" && w.SubAccount == c.cfg.AccountNo
}
