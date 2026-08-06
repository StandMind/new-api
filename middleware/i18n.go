package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
)

// I18n middleware detects and sets the language preference for the request
func I18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := detectLanguage(c)
		c.Set(string(constant.ContextKeyLanguage), lang)
		c.Next()
	}
}

// detectLanguage parses only the request header. Authentication runs after this
// middleware and GetLangFromContext applies the persisted user preference.
func detectLanguage(c *gin.Context) string {
	return i18n.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
}

// GetLanguage returns the current language from gin context
func GetLanguage(c *gin.Context) string {
	if lang := c.GetString(string(constant.ContextKeyLanguage)); lang != "" {
		return lang
	}
	return i18n.DefaultLang
}
