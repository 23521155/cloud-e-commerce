package resilience
import (
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/fizzisme/api-gateway/internal/metrics"
	"github.com/sony/gobreaker/v2"
)

// TestMain registers the Prometheus collectors once: a breaker records its
// state changes into them.
func TestMain(m *testing.M) {
	metrics.Init()
	os.Exit(m.Run())
}

func TestBreaker_OpenAfterFailures(t *testing.T){
	breaker := NewBreaker("auth")

	for i := 0; i < 5; i++ {

	_, _ = breaker.Execute(func() (*http.Response, error) {

		return nil, errors.New("boom")

	})

}
_, err := breaker.Execute(func() (*http.Response, error) {

	return &http.Response{
		StatusCode: http.StatusOK,
	}, nil

})
if !errors.Is(err, gobreaker.ErrOpenState) {

	t.Fatalf(
		"expected ErrOpenState, got %v",
		err,
	)

}
}