package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fizzisme/order-service/internal/catalogclient"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// fakeCreator returns what the test set and records the input it got.
type fakeCreator struct {
	order    Order
	replayed bool
	err      error

	got CreateInput

	orders    []Order
	gotUserID string

	gotCode, gotEmail string
}

func (f *fakeCreator) Create(ctx context.Context, in CreateInput) (Order, bool, error) {
	f.got = in
	return f.order, f.replayed, f.err
}

func (f *fakeCreator) ListMine(ctx context.Context, userID string) ([]Order, error) {
	f.gotUserID = userID
	return f.orders, f.err
}

func (f *fakeCreator) Lookup(ctx context.Context, code, email string) (Order, error) {
	f.gotCode, f.gotEmail = code, email
	return f.order, f.err
}

const validBody = `{
	"items": [{"slug": "nha-gia-kim"}, {"slug": "dac-nhan-tam"}],
	"paymentMethod": "cod",
	"email": "guest@example.com",
	"phone": "0901234567",
	"fullName": "Nguyễn Văn A",
	"address": "12 Lê Lợi",
	"city": "Hồ Chí Minh",
	"postalCode": null,
	"country": "Vietnam"
}`

// post sends body to POST / with an idempotency key and returns the response.
func post(t *testing.T, svc creator, body string) *httptest.ResponseRecorder {
	t.Helper()
	return postAs(t, svc, body, "")
}

// postAs is post for a signed-in customer; userID "" is a guest.
func postAs(t *testing.T, svc creator, body, userID string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(svc, zap.NewNop()).Register(r.Group(""))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(idempotencyHeader, "3f9c1a52-7b0e-4d4a-9c1e-8a2b5d6f7e10")
	if userID != "" {
		req.Header.Set(userIDHeader, userID)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func TestCreateHandlerCreated(t *testing.T) {

	svc := &fakeCreator{order: Order{Code: "MA-048291", Status: StatusConfirmed, SubtotalVND: 200000}}

	w := post(t, svc, validBody)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", w.Code, w.Body)
	}

	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["code"] != "MA-048291" || got["subtotalVnd"] != float64(200000) {
		t.Errorf("body = %v, want the order", got)
	}
	if _, ok := got["id"]; ok {
		t.Error("the internal id is in the response")
	}

	in := svc.got
	if in.Key != "3f9c1a52-7b0e-4d4a-9c1e-8a2b5d6f7e10" {
		t.Errorf("Key = %q, want the header value", in.Key)
	}
	if len(in.Slugs) != 2 || in.Slugs[0] != "nha-gia-kim" || in.FullName != "Nguyễn Văn A" || in.PostalCode != nil {
		t.Errorf("input = %+v, want the body fields", in)
	}
	if in.UserID != nil {
		t.Errorf("UserID = %q, want nil for a guest", *in.UserID)
	}
}

func TestCreateHandlerSignedIn(t *testing.T) {

	svc := &fakeCreator{order: Order{Code: "MA-048291"}}

	w := postAs(t, svc, validBody, "01a14046-945e-79f4-9de0-dcc903fd7cea")

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if svc.got.UserID == nil || *svc.got.UserID != "01a14046-945e-79f4-9de0-dcc903fd7cea" {
		t.Errorf("UserID = %v, want the X-User-ID value", svc.got.UserID)
	}
}

// getMine sends GET / as userID ("" for a guest) and returns the response.
func getMine(t *testing.T, svc creator, userID string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(svc, zap.NewNop()).Register(r.Group(""))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if userID != "" {
		req.Header.Set(userIDHeader, userID)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func TestListMineHandler(t *testing.T) {

	svc := &fakeCreator{orders: []Order{{Code: "MA-000002"}, {Code: "MA-000001"}}}

	w := getMine(t, svc, "u1")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body)
	}
	if svc.gotUserID != "u1" {
		t.Errorf("listed for %q, want u1", svc.gotUserID)
	}

	var body struct {
		Items []struct {
			Code string `json:"code"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 || body.Items[0].Code != "MA-000002" {
		t.Errorf("body = %s", w.Body)
	}
}

func TestListMineHandlerNeedsSignIn(t *testing.T) {

	svc := &fakeCreator{}
	w := getMine(t, svc, "")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if svc.gotUserID != "" {
		t.Error("the service was asked for a guest's orders")
	}
}

func TestListMineHandlerHidesInternalErrors(t *testing.T) {

	w := getMine(t, &fakeCreator{err: errors.New("mssql: login failed for user sa")}, "u1")

	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "mssql") {
		t.Errorf("status %d, body %s", w.Code, w.Body)
	}
}

// lookup sends body to POST /lookup and returns the response.
func lookup(t *testing.T, svc creator, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(svc, zap.NewNop()).Register(r.Group(""))

	req := httptest.NewRequest(http.MethodPost, "/lookup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func TestLookupHandler(t *testing.T) {

	svc := &fakeCreator{order: Order{Code: "MA-048291"}}

	w := lookup(t, svc, `{"code": "MA-048291", "email": "guest@example.com"}`)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":"MA-048291"`) {
		t.Fatalf("status %d, body %s", w.Code, w.Body)
	}
	if svc.gotCode != "MA-048291" || svc.gotEmail != "guest@example.com" {
		t.Errorf("looked up %q / %q", svc.gotCode, svc.gotEmail)
	}
}

func TestLookupHandlerNeedsCodeAndEmail(t *testing.T) {

	for _, body := range []string{"", "not json", `{"code": "MA-048291"}`, `{"email": "guest@example.com"}`, `{"code": " ", "email": " "}`} {
		svc := &fakeCreator{}
		if w := lookup(t, svc, body); w.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, w.Code)
		}
		if svc.gotCode != "" {
			t.Errorf("body %q reached the service", body)
		}
	}
}

func TestLookupHandlerErrors(t *testing.T) {

	if w := lookup(t, &fakeCreator{err: ErrNotFound}, `{"code": "MA-048291", "email": "a@b.co"}`); w.Code != http.StatusNotFound {
		t.Errorf("not found: status = %d, want 404", w.Code)
	}

	w := lookup(t, &fakeCreator{err: errors.New("mssql: login failed for user sa")}, `{"code": "MA-048291", "email": "a@b.co"}`)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "mssql") {
		t.Errorf("status %d, body %s", w.Code, w.Body)
	}
}

func TestCreateHandlerReplay(t *testing.T) {

	w := post(t, &fakeCreator{order: Order{Code: "MA-048291"}, replayed: true}, validBody)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestCreateHandlerBadBody(t *testing.T) {

	svc := &fakeCreator{}

	for _, body := range []string{"", "not json", `{"items": "nha-gia-kim"}`} {
		if w := post(t, svc, body); w.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, w.Code)
		}
	}
}

func TestCreateHandlerBodyTooLarge(t *testing.T) {

	body := `{"address": "` + strings.Repeat("a", maxBodyBytes) + `"}`

	if w := post(t, &fakeCreator{}, body); w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreateHandlerErrors(t *testing.T) {

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"invalid input", fmt.Errorf("%w: email is not a valid address", ErrInvalidInput), http.StatusBadRequest},
		{"key reused", ErrKeyReused, http.StatusUnprocessableEntity},
		{"books unavailable", &catalogclient.UnavailableError{}, http.StatusConflict},
		{"catalogue down", fmt.Errorf("%w: deadline", catalogclient.ErrCatalogUnavailable), http.StatusServiceUnavailable},
		{"no free code", ErrNoFreeCode, http.StatusServiceUnavailable},
		{"anything else", errors.New("connection lost"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := post(t, &fakeCreator{err: tt.err}, validBody); w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestCreateHandlerNamesTheUnavailableBooks(t *testing.T) {

	err := &catalogclient.UnavailableError{Books: []catalogclient.Unavailable{
		{ID: 7, Slug: "nha-gia-kim", Status: catalogclient.StatusSold},
	}}

	w := post(t, &fakeCreator{err: err}, validBody)

	var got struct {
		Books []map[string]any `json:"books"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	if len(got.Books) != 1 || got.Books[0]["slug"] != "nha-gia-kim" || got.Books[0]["status"] != "SOLD" {
		t.Errorf("books = %v, want nha-gia-kim SOLD", got.Books)
	}
	if _, ok := got.Books[0]["id"]; ok {
		t.Error("the catalogue id is in the response")
	}
}

func TestCreateHandlerHidesInternalErrors(t *testing.T) {

	w := post(t, &fakeCreator{err: errors.New("mssql: login failed for user sa")}, validBody)

	if strings.Contains(w.Body.String(), "mssql") {
		t.Errorf("body %s leaks the internal error", w.Body)
	}
}
