package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fizzisme/api-gateway/internal/logger"
	"github.com/fizzisme/api-gateway/internal/metrics"
	"github.com/fizzisme/api-gateway/internal/resilience"
	"go.uber.org/zap"
)

// TestMain registers the Prometheus collectors once: the breaker and retry
// transports record into them.
func TestMain(m *testing.M) {
	metrics.Init()
	os.Exit(m.Run())
}

// setupLogger gives the proxy a logger that writes nowhere. logger.Init
// also writes logs/gateway.log, which only exists in the container.
func setupLogger(t *testing.T) {
	t.Helper()

	if logger.Log == nil {
		logger.Log = zap.NewNop()
	}
}

// newTestProxy is New for one upstream with its own breaker, no retries,
// no streaming and no upload handling.
func newTestProxy(t *testing.T, target, prefix string, timeout time.Duration) *ReverseProxy {
	t.Helper()

	rp, err := New(target, prefix, timeout,
		resilience.NewBreaker(t.Name()),
		&resilience.RetryConfig{MaxAttempts: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond},
		t.Name(), false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return rp
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
	rp := newTestProxy(t, auth.URL, "/auth", 5*time.Second)

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

	rp := newTestProxy(t, auth.URL, "/api/v1/auth", 5*time.Second)

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

	// The backend echoes what reached it.
	auth := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Got-Authorization", r.Header.Get("Authorization"))
			w.Header().Set("Got-User-ID", r.Header.Get("X-User-ID"))
			_, hasEmail := r.Header["X-User-Email"]
			if hasEmail {
				w.Header().Set("Got-User-Email", "present")
			}
		}),
	)

	defer auth.Close()

	rp := newTestProxy(t, auth.URL, "/auth", 5*time.Second)

	// As JWTAuth leaves it: the token, the verified identity, and an
	// identity header it emptied.
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	req.Header.Set("Authorization", "Bearer abc")
	req.Header.Set("X-User-ID", "u1")
	req.Header.Set("X-User-Email", "")

	rr := httptest.NewRecorder()
	rp.ServeHTTP(rr, req)

	// The backend trusts the identity the gateway verified, never the raw token.
	if got := rr.Header().Get("Got-Authorization"); got != "" {
		t.Errorf("Authorization reached the backend: %q", got)
	}
	if got := rr.Header().Get("Got-User-ID"); got != "u1" {
		t.Errorf("X-User-ID = %q, want u1", got)
	}
	if rr.Header().Get("Got-User-Email") != "" {
		t.Error("an empty identity header reached the backend; it should be absent")
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

	rp := newTestProxy(t, auth.URL, "/auth", 5*time.Second)

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
	rp := newTestProxy(t, auth.URL, "/auth", 1*time.Second)

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