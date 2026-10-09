package subject

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
)

type fakeLister struct{ err error }

func (f fakeLister) List(context.Context) ([]Subject, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []Subject{{ID: 1, Slug: "history", Name: "History", BookCount: 357}}, nil
}

func serve(t *testing.T, l lister) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	if err := logger.Init(); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	NewHandler(l).Register(r.Group("/api/v1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subjects", nil))
	return w
}

func TestList(t *testing.T) {
	w := serve(t, fakeLister{})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	want := `{"items":[{"slug":"history","name":"History","bookCount":357}]}`
	if w.Body.String() != want {
		t.Errorf("body = %s, want %s", w.Body, want)
	}
}

func TestListError(t *testing.T) {
	w := serve(t, fakeLister{err: errors.New("mssql: boom")})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if strings.Contains(w.Body.String(), "mssql") {
		t.Errorf("body leaks the error: %s", w.Body)
	}
}
