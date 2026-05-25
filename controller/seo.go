package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/service/seo"
	"github.com/gin-gonic/gin"
)

func GetSitemapXML(c *gin.Context) {
	body, err := seo.GenerateSitemapXML(c.Request)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate sitemap")
		return
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", body)
}

func GetRobotsTxt(c *gin.Context) {
	baseURL := seo.ResolveBaseURL(c.Request)

	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(seo.BuildRobotsTxt(baseURL)))
}
