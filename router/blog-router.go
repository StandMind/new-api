package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func SetBlogRouter(apiRouter *gin.RouterGroup) {
	blogRoute := apiRouter.Group("/blog")
	{
		blogRoute.GET("/posts", controller.GetPublicBlogPosts)
		blogRoute.GET("/posts/:slug", controller.GetPublicBlogPost)
	}

	blogAdminRoute := blogRoute.Group("/admin")
	blogAdminRoute.Use(middleware.AdminAuth())
	{
		blogAdminRoute.GET("/posts", controller.GetAdminBlogPosts)
		blogAdminRoute.GET("/posts/:id", controller.GetAdminBlogPost)
		blogAdminRoute.POST("/posts", controller.CreateAdminBlogPost)
		blogAdminRoute.PUT("/posts/:id", controller.UpdateAdminBlogPost)
		blogAdminRoute.DELETE("/posts/:id", controller.DeleteAdminBlogPost)
	}
}
