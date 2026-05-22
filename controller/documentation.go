package controller

import (
	"errors"
	"net/http"

	documentation "github.com/QuantumNous/new-api/service/documentation"

	"github.com/gin-gonic/gin"
)

func GetDocumentationConfig(c *gin.Context) {
	config, err := documentation.GetConfig(documentationLanguage(c))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    config,
	})
}

func GetDocumentationPage(c *gin.Context) {
	page, err := documentation.GetPage(c.Param("slug"), documentationLanguage(c))
	if err != nil {
		message := err.Error()
		switch {
		case errors.Is(err, documentation.ErrDisabled):
			message = "Documentation is disabled"
		case errors.Is(err, documentation.ErrNotFound):
			message = "Documentation page not found"
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    page,
	})
}

func documentationLanguage(c *gin.Context) string {
	if lang := c.Query("lang"); lang != "" {
		return lang
	}
	return c.GetHeader("Accept-Language")
}
