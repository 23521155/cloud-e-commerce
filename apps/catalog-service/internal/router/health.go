package router

import (
	"context"
	"net/http"

	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// pinger is satisfied by *sqlx.DB, and by a fake in tests.
type pinger interface {
	PingContext(ctx context.Context) error
}

func registerHealthRoutes(r *gin.Engine, db pinger) {

	r.GET("/health", func(c *gin.Context) {

		if err := db.PingContext(c.Request.Context()); err != nil {
			logger.Log.Error("health check: database unreachable", zap.Error(err))

			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})

	})

}
