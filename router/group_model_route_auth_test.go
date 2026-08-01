package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGroupModelRouteListRejectsCommonUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalRedisEnabled := common.RedisEnabled
	originalGlobalRateLimitEnabled := common.GlobalApiRateLimitEnable
	common.RedisEnabled = false
	common.GlobalApiRateLimitEnable = false
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		common.GlobalApiRateLimitEnable = originalGlobalRateLimitEnabled
	})

	engine := gin.New()
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte("group-route-auth-test"))))
	engine.GET("/test-login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("username", "common-user")
		session.Set("role", common.RoleCommonUser)
		session.Set("id", 42)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(engine)

	loginRecorder := httptest.NewRecorder()
	engine.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/test-login", nil))
	require.Equal(t, http.StatusNoContent, loginRecorder.Code)

	request := httptest.NewRequest(http.MethodGet, "/api/group-model-routes/list", nil)
	request.Header.Set("New-Api-User", "42")
	for _, sessionCookie := range loginRecorder.Result().Cookies() {
		request.AddCookie(sessionCookie)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
}
