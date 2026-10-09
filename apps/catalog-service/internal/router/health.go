package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerHealthRoutes(r *gin.Engine) {

	r.GET("/health", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})

	})

}
