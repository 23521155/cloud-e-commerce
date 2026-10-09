package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
)

type fakePinger struct{ err error }

func (f fakePinger) PingContext(context.Context) error { return f.err }

func TestHealth(t *testing.T) {

	gin.SetMode(gin.TestMode)

	if err := logger.Init(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		pinger   fakePinger
		wantCode int
		wantBody string
	}{
		{"database up", fakePinger{}, http.StatusOK, `{"status":"ok"}`},
		{"database down", fakePinger{err: errors.New("down")}, http.StatusServiceUnavailable, `{"status":"unavailable"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r := SetupRouter(tt.pinger)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)

			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}

			if w.Body.String() != tt.wantBody {
				t.Fatalf("body = %s, want %s", w.Body.String(), tt.wantBody)
			}
		})
	}
}
