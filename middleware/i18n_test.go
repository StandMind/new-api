package middleware

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestI18nSuccessRequestDoesNotLoadUserLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var loaderCalls atomic.Int32
	i18n.SetUserLangLoader(func(int) string {
		loaderCalls.Add(1)
		return i18n.LangFr
	})
	t.Cleanup(func() { i18n.SetUserLangLoader(nil) })

	router := gin.New()
	router.Use(I18n())
	router.GET("/success", func(c *gin.Context) {
		c.Set("id", 42)
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/success", nil)
	request.Header.Set("Accept-Language", "ja")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Zero(t, loaderCalls.Load())
}

func TestI18nMiddlewareLanguageIsReusedByErrorRendering(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(I18n())
	router.GET("/error", func(c *gin.Context) {
		// Changing the header after middleware execution must not cause a second
		// parse or change this request's resolved language.
		c.Request.Header.Set("Accept-Language", "vi")
		c.String(http.StatusBadRequest, i18n.T(c, i18n.MsgInvalidParams))
	})

	request := httptest.NewRequest(http.MethodGet, "/error", nil)
	request.Header.Set("Accept-Language", "fr")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, i18n.Translate(i18n.LangFr, i18n.MsgInvalidParams), recorder.Body.String())
}

func TestAuthenticatedLanguageSourcesDoNotUseFallbackLoader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var loaderCalls atomic.Int32
	i18n.SetUserLangLoader(func(int) string {
		loaderCalls.Add(1)
		return i18n.LangRu
	})
	t.Cleanup(func() { i18n.SetUserLangLoader(nil) })

	t.Run("token user cache setting", func(t *testing.T) {
		router := gin.New()
		router.Use(I18n())
		router.GET("/token", func(c *gin.Context) {
			userCache := model.UserBase{Setting: `{"language":"vi"}`}
			userCache.WriteContext(c)
			c.Set("id", 42)
			c.String(http.StatusBadRequest, i18n.T(c, i18n.MsgInvalidParams))
		})

		request := httptest.NewRequest(http.MethodGet, "/token", nil)
		request.Header.Set("Accept-Language", "fr")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		assert.Equal(t, i18n.Translate(i18n.LangVi, i18n.MsgInvalidParams), recorder.Body.String())
		assert.Zero(t, loaderCalls.Load())
	})

	t.Run("valid session language", func(t *testing.T) {
		router := gin.New()
		router.Use(I18n())
		router.Use(sessions.Sessions("session", cookie.NewStore([]byte("i18n-session-test-secret"))))
		router.GET("/login", func(c *gin.Context) {
			session := sessions.Default(c)
			session.Set("id", 42)
			session.Set("language", i18n.LangJa)
			assert.NoError(t, session.Save())
			c.Status(http.StatusNoContent)
		})
		router.GET("/session", TryUserAuth(), func(c *gin.Context) {
			c.String(http.StatusBadRequest, i18n.T(c, i18n.MsgInvalidParams))
		})

		loginRecorder := httptest.NewRecorder()
		router.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/login", nil))
		assert.Equal(t, http.StatusNoContent, loginRecorder.Code)

		request := httptest.NewRequest(http.MethodGet, "/session", nil)
		request.Header.Set("Accept-Language", "fr")
		for _, sessionCookie := range loginRecorder.Result().Cookies() {
			request.AddCookie(sessionCookie)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		assert.Equal(t, i18n.Translate(i18n.LangJa, i18n.MsgInvalidParams), recorder.Body.String())
		assert.Zero(t, loaderCalls.Load())
	})
}

func legacyNormalizeLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(value, "zh-tw"):
		return i18n.LangZhTW
	case strings.HasPrefix(value, "zh"):
		return i18n.LangZhCN
	case strings.HasPrefix(value, "en"):
		return i18n.LangEn
	default:
		return i18n.LangEn
	}
}

func legacyDetectLanguage(c *gin.Context) string {
	header := c.GetHeader("Accept-Language")
	if header == "" {
		return i18n.LangEn
	}
	parts := strings.Split(header, ",")
	value := strings.TrimSpace(parts[0])
	if index := strings.IndexByte(value, ';'); index > 0 {
		value = value[:index]
	}
	lang := legacyNormalizeLanguage(value)
	normalized := legacyNormalizeLanguage(lang)
	for _, supported := range []string{i18n.LangZhCN, i18n.LangZhTW, i18n.LangEn} {
		if normalized == supported {
			return lang
		}
	}
	return i18n.LangEn
}

func languageBenchmarkRouter(legacy bool) *gin.Engine {
	router := gin.New()
	if legacy {
		router.Use(func(c *gin.Context) {
			c.Set(string(constant.ContextKeyLanguage), legacyDetectLanguage(c))
			c.Next()
		})
	} else {
		router.Use(I18n())
	}
	router.GET("/success", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return router
}

func benchmarkSuccessRequests(b *testing.B, legacy bool) {
	b.Helper()
	gin.SetMode(gin.TestMode)
	router := languageBenchmarkRouter(legacy)
	request := httptest.NewRequest(http.MethodGet, "/success", nil)
	request.Header.Set("Accept-Language", "en")
	latencies := make([]time.Duration, 0, b.N/64+1)

	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		recorder := httptest.NewRecorder()
		startedAt := time.Now()
		router.ServeHTTP(recorder, request)
		if index%64 == 0 {
			latencies = append(latencies, time.Since(startedAt))
		}
	}
	b.StopTimer()

	sort.Slice(latencies, func(left, right int) bool { return latencies[left] < latencies[right] })
	if len(latencies) > 0 {
		p95Index := (len(latencies)*95 + 99) / 100
		if p95Index > 0 {
			p95Index--
		}
		b.ReportMetric(float64(latencies[p95Index].Nanoseconds()), "p95-ns")
	}
}

func BenchmarkI18nSuccessRequest(b *testing.B) {
	b.Run("before", func(b *testing.B) { benchmarkSuccessRequests(b, true) })
	b.Run("after", func(b *testing.B) { benchmarkSuccessRequests(b, false) })
}
