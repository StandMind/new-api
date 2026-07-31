package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

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
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()

	db, err := gorm.Open(sqlite.Open("file:distributor_route?mode=memory&cache=shared"), &gorm.Config{})
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
