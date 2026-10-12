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
}

func (f *fakeCreator) Create(ctx context.Context, in CreateInput) (Order, bool, error) {
	f.got = in
	return f.order, f.replayed, f.err
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

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(svc, zap.NewNop()).Register(r.Group(""))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(idempotencyHeader, "3f9c1a52-7b0e-4d4a-9c1e-8a2b5d6f7e10")

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
