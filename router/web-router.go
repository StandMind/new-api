package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/webbranding"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// ThemeAssets holds the embedded frontend assets for both themes.
type ThemeAssets struct {
	DefaultBuildFS   embed.FS
	DefaultIndexPage []byte
	ClassicBuildFS   embed.FS
	ClassicIndexPage []byte
}

func SetWebRouter(router *gin.Engine, assets ThemeAssets) {
	defaultFS := common.EmbedFolder(assets.DefaultBuildFS, "web/default/dist")
	classicFS := common.EmbedFolder(assets.ClassicBuildFS, "web/classic/dist")
	themeFS := common.NewThemeAwareFS(defaultFS, classicFS)

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())

	router.GET("/", func(c *gin.Context) {
		serveIndexPage(c, assets)
	})
	router.HEAD("/", func(c *gin.Context) {
		serveIndexPage(c, assets)
	})
	router.GET("/index.html", func(c *gin.Context) {
		serveIndexPage(c, assets)
	})
	router.HEAD("/index.html", func(c *gin.Context) {
		serveIndexPage(c, assets)
	})
	router.Use(static.Serve("/", themeFS))
	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		serveIndexPage(c, assets)
	})
}

func serveIndexPage(c *gin.Context, assets ThemeAssets) {
	serveIndexPageWithMeta(c, assets, webbranding.PageMeta{}, http.StatusOK)
}

func serveIndexPageWithMeta(c *gin.Context, assets ThemeAssets, meta webbranding.PageMeta, status int) {
	c.Set(middleware.RouteTagKey, "web")
	c.Header("Cache-Control", "no-cache")
	if common.GetTheme() == "classic" {
		c.Data(status, "text/html; charset=utf-8", webbranding.ApplyIndexPageBrandingWithMeta(assets.ClassicIndexPage, meta))
		return
	}
	c.Data(status, "text/html; charset=utf-8", webbranding.ApplyIndexPageBrandingWithMeta(assets.DefaultIndexPage, meta))
}
