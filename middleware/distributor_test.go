package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDistributorRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
	})

	model.DB = db
	model.LOG_DB = db
	common.MemoryCacheEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(
		&model.Channel{},
		&model.Ability{},
		&model.GroupModelRoute{},
	))
	return db
}

func TestDistributeAllowsRequestsThatDoNotSelectAChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/videos/:task_id", Distribute(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/videos/task-1", nil)
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestDistributeSkipsCandidateWithoutAnEnabledKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupDistributorRouteTestDB(t)

	highPriority := int64(100)
	lowPriority := int64(50)
	channels := []model.Channel{
		{
			Id:     8201,
			Type:   constant.ChannelTypeOpenAI,
			Key:    "disabled-key",
			Status: common.ChannelStatusEnabled,
			Name:   "no-enabled-key",
			ChannelInfo: model.ChannelInfo{
				IsMultiKey: true,
				MultiKeyStatusList: map[int]int{
					0: common.ChannelStatusManuallyDisabled,
				},
			},
		},
		{
			Id:     8202,
			Type:   constant.ChannelTypeOpenAI,
			Key:    "enabled-key",
			Status: common.ChannelStatusEnabled,
			Name:   "usable-channel",
		},
	}
	require.NoError(t, db.Create(&channels).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{
			Group:     "default",
			Model:     "route-model",
			ChannelId: 8201,
			Enabled:   true,
			Priority:  &highPriority,
		},
		{
			Group:     "default",
			Model:     "route-model",
			ChannelId: 8202,
			Enabled:   true,
			Priority:  &lowPriority,
		},
	}).Error)

	router := gin.New()
	router.POST(
		"/v1/chat/completions",
		func(c *gin.Context) {
			common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
			common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
			c.Next()
		},
		Distribute(),
		func(c *gin.Context) {
			assert.Equal(t, 8202, common.GetContextKeyInt(c, constant.ContextKeyChannelId))
			c.Status(http.StatusNoContent)
		},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(`{"model":"route-model"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestDistributePlaygroundUsesCanonicalChatPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupDistributorRouteTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.UserLevel{},
		&model.RouteGroup{},
		&model.UserLevelRouteGroup{},
	))
	require.NoError(t, db.Create(&model.UserLevel{
		Code: "standard", Name: "Standard", IsDefault: true, Enabled: true, TopupRatio: 1,
	}).Error)
	require.NoError(t, db.Create(&model.RouteGroup{
		Code: "playground-group", Name: "Playground", BaseRatio: 1, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&model.UserLevelRouteGroup{
		UserLevelCode: "standard", RouteGroupCode: "playground-group",
	}).Error)
	require.NoError(t, model.RebuildAccessPolicySnapshot())

	priority := int64(0)
	channel := model.Channel{
		Id:       8301,
		Type:     constant.ChannelTypeAdvancedCustom,
		Key:      "advanced-custom-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "playground-advanced-custom",
		Group:    "playground-group",
		Models:   "playground-chat-model",
		Priority: &priority,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{Routes: []dto.AdvancedCustomRoute{
			{IncomingPath: "/v1/chat/completions", UpstreamPath: "/v1/chat/completions"},
		}},
	})
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     "playground-group",
		Model:     "playground-chat-model",
		ChannelId: channel.Id,
		Enabled:   true,
		Priority:  &priority,
	}).Error)
	require.NoError(t, model.InitGroupModelRouteIndex())

	router := gin.New()
	router.POST(
		"/pg/chat/completions",
		func(c *gin.Context) {
			common.SetContextKey(c, constant.ContextKeyUsingGroup, "playground-group")
			common.SetContextKey(c, constant.ContextKeyUserLevel, "standard")
			c.Next()
		},
		Distribute(),
		func(c *gin.Context) {
			assert.Equal(t, channel.Id, common.GetContextKeyInt(c, constant.ContextKeyChannelId))
			c.Status(http.StatusNoContent)
		},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/pg/chat/completions",
		strings.NewReader(`{"model":"playground-chat-model","route_group":"playground-group"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func setupPlaygroundSmartRoutingTest(t *testing.T) *gin.Engine {
	t.Helper()
	require.NoError(t, i18n.Init())

	db := setupDistributorRouteTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.UserLevel{},
		&model.RouteGroup{},
		&model.UserLevelRouteGroup{},
		&model.PerfMetric{},
		&model.Token{},
	))
	require.NoError(t, model.MigrateLegacyTokenGroupChains())
	require.NoError(t, db.Create(&model.UserLevel{
		Code: model.StandardUserLevelCode, Name: "Standard", IsDefault: true,
		Enabled: true, TopupRatio: 1,
	}).Error)
	require.NoError(t, db.Create(&[]model.RouteGroup{
		{Code: "playground-cheap", Name: "Cheap", BaseRatio: 0.5, Enabled: true},
		{Code: "playground-expensive", Name: "Expensive", BaseRatio: 1.5, Enabled: true},
	}).Error)
	require.NoError(t, db.Create(&[]model.UserLevelRouteGroup{
		{UserLevelCode: model.StandardUserLevelCode, RouteGroupCode: "playground-cheap"},
		{UserLevelCode: model.StandardUserLevelCode, RouteGroupCode: "playground-expensive"},
	}).Error)

	priority := int64(0)
	weight := uint(100)
	require.NoError(t, db.Create(&[]model.Channel{
		{
			Id: 8401, Type: constant.ChannelTypeOpenAI, Key: "cheap-key",
			Status: common.ChannelStatusEnabled, Name: "playground-cheap",
			Group: "playground-cheap", Models: "playground-smart-model",
			Priority: &priority, Weight: &weight,
		},
		{
			Id: 8402, Type: constant.ChannelTypeOpenAI, Key: "expensive-key",
			Status: common.ChannelStatusEnabled, Name: "playground-expensive",
			Group: "playground-expensive", Models: "playground-smart-model",
			Priority: &priority, Weight: &weight,
		},
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{
			Group: "playground-cheap", Model: "playground-smart-model",
			ChannelId: 8401, Enabled: true, Priority: &priority, Weight: weight,
		},
		{
			Group: "playground-expensive", Model: "playground-smart-model",
			ChannelId: 8402, Enabled: true, Priority: &priority, Weight: weight,
		},
	}).Error)
	require.NoError(t, model.RebuildAccessPolicySnapshot())
	require.NoError(t, model.InitGroupModelRouteIndex())
	require.NoError(t, service.RebuildSmartRoutingSnapshot())

	router := gin.New()
	for _, path := range []string{
		"/pg/chat/completions",
		"/v1/chat/completions",
	} {
		router.POST(
			path,
			func(c *gin.Context) {
				common.SetContextKey(c, constant.ContextKeyUsingGroup, "playground-expensive")
				common.SetContextKey(c, constant.ContextKeyUserLevel, model.StandardUserLevelCode)
				common.SetContextKey(c, constant.ContextKeyUserGroup, model.StandardUserLevelCode)
				common.SetContextKey(c, constant.ContextKeyTokenGroupChain, []string{"playground-expensive"})
				c.Next()
			},
			Distribute(),
			func(c *gin.Context) {
				plan := service.GetRouteAttemptPlan(c)
				if plan == nil {
					c.Status(http.StatusInternalServerError)
					return
				}
				c.Header("X-Test-Routing", string(plan.RoutingPriority()))
				c.Header("X-Test-Basis", plan.RankingBasis())
				c.Header("X-Test-Groups", strings.Join(plan.ConfiguredGroups(), ","))
				c.Status(http.StatusNoContent)
			},
		)
	}
	return router
}

func postDistributorRequest(router *gin.Engine, path, body string, languages ...string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if len(languages) > 0 && languages[0] != "" {
		request.Header.Set("Accept-Language", languages[0])
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestDistributePlaygroundSmartRoutingModesUseSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupPlaygroundSmartRoutingTest(t)

	tests := []struct {
		mode  string
		basis string
	}{
		{mode: "auto", basis: "price_fallback"},
		{mode: "price", basis: "price"},
		{mode: "speed", basis: "price_fallback"},
		{mode: "success_rate", basis: "price_fallback"},
	}
	for _, test := range tests {
		t.Run(test.mode, func(t *testing.T) {
			recorder := postDistributorRequest(
				router,
				"/pg/chat/completions",
				fmt.Sprintf(`{"model":"playground-smart-model","routing_priority":%q}`, test.mode),
			)

			assert.Equal(t, http.StatusNoContent, recorder.Code)
			assert.Equal(t, test.mode, recorder.Header().Get("X-Test-Routing"))
			assert.Equal(t, test.basis, recorder.Header().Get("X-Test-Basis"))
			assert.Equal(
				t,
				"playground-cheap,playground-expensive",
				recorder.Header().Get("X-Test-Groups"),
			)
		})
	}
}

func TestDistributePlaygroundRoutingValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupPlaygroundSmartRoutingTest(t)

	tests := []struct {
		name       string
		body       string
		statusCode int
		message    string
		language   string
	}{
		{
			name:       "invalid mode",
			body:       `{"model":"playground-smart-model","routing_priority":"random"}`,
			statusCode: http.StatusBadRequest,
			message:    "Invalid Playground routing_priority",
		},
		{
			name:       "conflicting fields in Chinese",
			body:       `{"model":"playground-smart-model","route_group":"playground-cheap","routing_priority":"price"}`,
			statusCode: http.StatusBadRequest,
			message:    "不能同时使用",
			language:   "zh-CN",
		},
		{
			name:       "unauthorized manual group",
			body:       `{"model":"playground-smart-model","route_group":"not-granted"}`,
			statusCode: http.StatusForbidden,
		},
		{
			name:       "no smart candidate",
			body:       `{"model":"missing-model","routing_priority":"price"}`,
			statusCode: http.StatusServiceUnavailable,
			message:    "Smart routing has no available group that supports model missing-model",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := postDistributorRequest(router, "/pg/chat/completions", test.body, test.language)
			assert.Equal(t, test.statusCode, recorder.Code)
			if test.message != "" {
				assert.Contains(t, recorder.Body.String(), test.message)
			}
		})
	}
}

func TestDistributeV1IgnoresPlaygroundRoutingOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupPlaygroundSmartRoutingTest(t)

	recorder := postDistributorRequest(
		router,
		"/v1/chat/completions",
		`{"model":"playground-smart-model","routing_priority":"price"}`,
	)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Empty(t, recorder.Header().Get("X-Test-Routing"))
	assert.Equal(t, "playground-expensive", recorder.Header().Get("X-Test-Groups"))
}
