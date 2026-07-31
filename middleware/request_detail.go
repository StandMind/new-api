package middleware

import (
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// RequestDetailCapture must run after authentication and before routing. This
// keeps unauthenticated payloads out while still capturing distributor errors.
func RequestDetailCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		service.CaptureRequestDetailFromContext(c)
	}
}
