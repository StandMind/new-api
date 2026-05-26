package router

import (
	"embed"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/bloghtml"
	"github.com/QuantumNous/new-api/service/seo"
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
	router.GET("/blog", func(c *gin.Context) {
		serveBlogIndexPage(c, assets)
	})
	router.HEAD("/blog", func(c *gin.Context) {
		serveBlogIndexPage(c, assets)
	})
	router.GET("/blog/*path", func(c *gin.Context) {
		serveBlogPostPage(c, assets)
	})
	router.HEAD("/blog/*path", func(c *gin.Context) {
		serveBlogPostPage(c, assets)
	})
	router.GET("/bolg", redirectBolgToBlog)
	router.HEAD("/bolg", redirectBolgToBlog)
	router.GET("/bolg/*path", redirectBolgToBlog)
	router.HEAD("/bolg/*path", redirectBolgToBlog)

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

func redirectBolgToBlog(c *gin.Context) {
	target := "/blog"
	if value := strings.TrimPrefix(c.Param("path"), "/"); value != "" {
		target += "/" + value
	}
	if c.Request.URL.RawQuery != "" {
		target += "?" + c.Request.URL.RawQuery
	}
	c.Redirect(http.StatusMovedPermanently, target)
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

func serveBlogIndexPage(c *gin.Context, assets ThemeAssets) {
	if model.DB == nil {
		serveIndexPageWithMeta(c, assets, bloghtml.BuildBlogIndexMeta(blogRenderOptions(c), nil), http.StatusOK)
		return
	}
	options := blogRenderOptions(c)
	posts, _, err := model.ListPublishedBlogPosts(options.Locale, "", "", 0, 12)
	if err != nil {
		posts = nil
	}
	serveIndexPageWithMeta(c, assets, bloghtml.BuildBlogIndexMeta(options, posts), http.StatusOK)
}

func serveBlogPostPage(c *gin.Context, assets ThemeAssets) {
	options := blogRenderOptions(c)
	slug := strings.Trim(strings.TrimSpace(c.Param("path")), "/")
	if slug == "" {
		c.Redirect(http.StatusMovedPermanently, "/blog")
		return
	}
	if strings.Contains(slug, "/") {
		serveIndexPageWithMeta(c, assets, bloghtml.BuildBlogNotFoundMeta(options), http.StatusNotFound)
		return
	}
	if model.DB == nil {
		serveIndexPageWithMeta(c, assets, bloghtml.BuildBlogNotFoundMeta(options), http.StatusNotFound)
		return
	}
	post, err := model.GetPublishedBlogPostBySlug(slug, options.Locale)
	if err != nil || post == nil {
		serveIndexPageWithMeta(c, assets, bloghtml.BuildBlogNotFoundMeta(options), http.StatusNotFound)
		return
	}
	serveIndexPageWithMeta(c, assets, bloghtml.BuildBlogPostMeta(options, *post), http.StatusOK)
}

func blogRenderOptions(c *gin.Context) bloghtml.RenderOptions {
	return bloghtml.RenderOptions{
		BaseURL:  seo.ResolveBaseURL(c.Request),
		Locale:   model.ResolveLocalizedTextLocale(c.Query("lang"), c.GetHeader("Accept-Language")),
		SiteName: strings.TrimSpace(common.SystemName),
	}
}
