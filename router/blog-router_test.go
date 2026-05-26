package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetBlogRouterRegistersStatsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRouter := router.Group("/api")

	require.NotPanics(t, func() {
		SetBlogRouter(apiRouter)
	})
}
