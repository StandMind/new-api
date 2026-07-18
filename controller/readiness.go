package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

func GetReadiness(c *gin.Context) {
	ready := common.IsProcessReady()
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{
		"success": ready,
		"ready":   ready,
	})
}
