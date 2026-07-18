package router

import (
	"github.com/QuantumNous/new-api/controller"

	"github.com/gin-gonic/gin"
)

func SetHealthRouter(router *gin.Engine) {
	router.GET("/readyz", controller.GetReadiness)
}
