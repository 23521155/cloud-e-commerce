package router

import (
	"time"

	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupRouter builds the Gin engine for the catalog service: global
// middleware and the health check route.
func SetupRouter(db pinger, routes ...Registrar) *gin.Engine {

	r := gin.New()

	// Only reachable through the gateway, so client IPs from
	// X-Forwarded-For are not trusted here.
	if err := r.SetTrustedProxies(nil); err != nil {
		logger.Log.Fatal(
			"cannot configure trusted proxies",
			zap.Error(err),
		)
	}

	r.Use(
		gin.Recovery(),
		requestLogger(),
	)

	registerHealthRoutes(r, db)

	// Served at the root: the gateway strips its /api/catalog prefix.
	api := r.Group("")
	for _, route := range routes {
		route.Register(api)
	}

	return r
}

// Registrar is implemented by every handler that mounts routes on the API group.
type Registrar interface {
	Register(api *gin.RouterGroup)
}

// requestLogger logs one structured entry per request. The request ID
// is the one assigned by the gateway, so logs correlate across services.
func requestLogger() gin.HandlerFunc {

	return func(c *gin.Context) {

		start := time.Now()

		c.Next()

		fields := []zap.Field{
			zap.String("request_id", c.GetHeader("X-Request-ID")),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		}

		status := c.Writer.Status()

		switch {
		case status >= 500:
			logger.Log.Error("HTTP Request", fields...)
		case status >= 400:
			logger.Log.Warn("HTTP Request", fields...)
		default:
			logger.Log.Info("HTTP Request", fields...)
		}
	}
}
