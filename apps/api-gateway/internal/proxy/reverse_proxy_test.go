package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"github.com/fizzisme/api-gateway/internal/logger"
	"github.com/fizzisme/api-gateway/internal/resilience"
	"go.uber.org/zap"
)

func setupLogger(t *testing.T) {
	t.Helper()

	if logger.Log == nil {
		logger.Log = zap.NewNop()
	}
}

func TestForwardRequest(t *testing.T) {

	setupLogger(t)

	// Fake Auth Service
	auth := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.WriteHeader(http.StatusOK)

			_, _ = w.Write([]byte("hello auth"))

		}),
	)

	defer auth.Close()

	// Create Proxy
	rp, err := New(
		auth.URL,
		"/auth",
		5*time.Second,
		resilience.NewBreaker("auth"),
		&resilience.RetryConfig{MaxAttempts: 1},
		"auth",
		false,
		0,
		0,
	)

	if err != nil {
		t.Fatal(err)
	}

	// Fake Client Request
	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/login",
		nil,
	)

	rr := httptest.NewRecorder()

	rp.ServeHTTP(
		rr,
		req,
	)

	if rr.Code != http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rr.Code,
		)

	}

	body, _ := io.ReadAll(rr.Body)

	if string(body) != "hello auth" {

		t.Fatalf(
			"unexpected body: %s",
			string(body),
		)

	}
}

func TestRewritePath(t *testing.T) {

	setupLogger(t)

	auth := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			_, _ = w.Write(
				[]byte(r.URL.Path),
			)

		}),
	)

	defer auth.Close()

	rp, _ := New(
		auth.URL,
		"/api/v1/auth",
		5*time.Second,
		resilience.NewBreaker("auth"),
		&resilience.RetryConfig{MaxAttempts: 1},
		"auth",
		false,
		0,
		0,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/login",
		nil,
	)

	rr := httptest.NewRecorder()

	rp.ServeHTTP(
		rr,
		req,
	)

	body, _ := io.ReadAll(rr.Body)

	if string(body) != "/login" {

		t.Fatalf(
			"expected /login, got %s",
			string(body),
		)

	}
}

func TestForwardHeaders(t *testing.T) {

	setupLogger(t)

	auth := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			_, _ = w.Write(
				[]byte(
					r.Header.Get("Authorization"),
				),
			)

		}),
	)

	defer auth.Close()

	rp, _ := New(
		auth.URL,
		"/auth",
		5*time.Second,
		resilience.NewBreaker("auth"),
		&resilience.RetryConfig{MaxAttempts: 1},
		"auth",
		false,
		0,
		0,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/login",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer abc",
	)

	rr := httptest.NewRecorder()

	rp.ServeHTTP(
		rr,
		req,
	)

	body, _ := io.ReadAll(rr.Body)

	if string(body) != "" {

		t.Fatalf(
			"Authorization header should not be forwarded",
		)

	}
}

// A header that is not a JWT is the backend's own business: SePay's webhook
// authenticates with "Authorization: Apikey <key>" and the order service checks it.
func TestForwardNonBearerAuthorization(t *testing.T) {

	setupLogger(t)

	backend := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(r.Header.Get("Authorization")))
		}),
	)

	defer backend.Close()

	rp, _ := New(
		backend.URL,
		"/orders",
		5*time.Second,
		resilience.NewBreaker("orders"),
		&resilience.RetryConfig{MaxAttempts: 1},
		"orders",
		false,
		0,
		0,
	)

	req := httptest.NewRequest(http.MethodPost, "/orders/payments/sepay/webhook", nil)
	req.Header.Set("Authorization", "Apikey s3cret")

	rr := httptest.NewRecorder()
	rp.ServeHTTP(rr, req)

	body, _ := io.ReadAll(rr.Body)

	if string(body) != "Apikey s3cret" {
		t.Fatalf("expected the Apikey header to be forwarded, got %q", string(body))
	}
}

func TestForwardQuery(t *testing.T) {

	setupLogger(t)

	auth := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			_, _ = w.Write(
				[]byte(
					r.URL.RawQuery,
				),
			)

		}),
	)

	defer auth.Close()

	rp, _ := New(
		auth.URL,
		"/auth",
		5*time.Second,
		resilience.NewBreaker("auth"),
		&resilience.RetryConfig{MaxAttempts: 1},
		"auth",
		false,
		0,
		0,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/login?id=10&name=phi",
		nil,
	)

	rr := httptest.NewRecorder()

	rp.ServeHTTP(
		rr,
		req,
	)

	body, _ := io.ReadAll(rr.Body)

	if string(body) != "id=10&name=phi" {

		t.Fatalf(
			"query not forwarded",
		)

	}
}

func TestTimeout(t *testing.T) {

	setupLogger(t)

	// Fake Auth Service
	auth := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			time.Sleep(2 * time.Second)

			w.WriteHeader(http.StatusOK)

			_, _ = w.Write([]byte("OK"))

		}),
	)

	defer auth.Close()

	// Proxy timeout sau 1 giây
	rp, err := New(
		auth.URL,
		"/auth",
		1*time.Second,
		resilience.NewBreaker("auth"),
		&resilience.RetryConfig{MaxAttempts: 1},
		"auth",
		false,
		0,
		0,
	)

	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/auth/login",
		nil,
	)

	rr := httptest.NewRecorder()

	start := time.Now()

	rp.ServeHTTP(
		rr,
		req,
	)

	elapsed := time.Since(start)

	if rr.Code != http.StatusGatewayTimeout {

		t.Fatalf(
			"expected %d got %d",
			http.StatusGatewayTimeout,
			rr.Code,
		)

	}

	if elapsed >= 2*time.Second {

		t.Fatalf(
			"request should timeout before upstream finishes, took %v",
			elapsed,
		)

	}
}