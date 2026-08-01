package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelRequestRateLimitCountsMultiGroupRequestAgainstFirstGroupOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalEnabled := setting.ModelRequestRateLimitEnabled
	originalDuration := setting.ModelRequestRateLimitDurationMinutes
	originalCount := setting.ModelRequestRateLimitCount
	originalSuccessCount := setting.ModelRequestRateLimitSuccessCount
	originalGroups := setting.ModelRequestRateLimitGroup2JSONString()
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		setting.ModelRequestRateLimitEnabled = originalEnabled
		setting.ModelRequestRateLimitDurationMinutes = originalDuration
		setting.ModelRequestRateLimitCount = originalCount
		setting.ModelRequestRateLimitSuccessCount = originalSuccessCount
		require.NoError(t, setting.UpdateModelRequestRateLimitGroupByJSONString(originalGroups))
		common.RedisEnabled = originalRedisEnabled
	})

	setting.ModelRequestRateLimitEnabled = true
	setting.ModelRequestRateLimitDurationMinutes = 1
	setting.ModelRequestRateLimitCount = 0
	setting.ModelRequestRateLimitSuccessCount = 1000
	require.NoError(t, setting.UpdateModelRequestRateLimitGroupByJSONString(
		`{"default":[1,1000],"backup":[100,1000]}`,
	))
	common.RedisEnabled = false

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("id", 991001)
		common.SetContextKey(c, constant.ContextKeyTokenGroup, "default")
		common.SetContextKey(c, constant.ContextKeyTokenGroupChain, []string{"default", "backup"})
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "backup")
		c.Next()
	})
	engine.Use(ModelRequestRateLimit())
	engine.GET("/v1/chat/completions", func(c *gin.Context) {
		for _, group := range []string{"default", "backup", "default"} {
			common.SetContextKey(c, constant.ContextKeyUsingGroup, group)
		}
		c.Status(http.StatusOK)
	})

	firstRecorder := httptest.NewRecorder()
	engine.ServeHTTP(firstRecorder, httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil))
	assert.Equal(t, http.StatusOK, firstRecorder.Code)

	secondRecorder := httptest.NewRecorder()
	engine.ServeHTTP(secondRecorder, httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil))
	assert.Equal(t, http.StatusTooManyRequests, secondRecorder.Code)
}
