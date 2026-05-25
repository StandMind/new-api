package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func SetSEORouter(router *gin.Engine) {
	seoRouter := router.Group("/")
	seoRouter.Use(middleware.RouteTag("web"))
	{
		seoRouter.GET("/robots.txt", controller.GetRobotsTxt)
		seoRouter.HEAD("/robots.txt", controller.GetRobotsTxt)
		seoRouter.GET("/sitemap.xml", controller.GetSitemapXML)
		seoRouter.HEAD("/sitemap.xml", controller.GetSitemapXML)
	}
}
