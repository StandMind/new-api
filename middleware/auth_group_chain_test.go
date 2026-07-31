package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTokenAuthRequiresValidExplicitGroupChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:%s_init?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))
	require.NoError(t, model.InitDB())
	if initSQLDB, initErr := model.DB.DB(); initErr == nil {
		require.NoError(t, initSQLDB.Close())
	}

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.RedisEnabled = originalRedisEnabled
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	})

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}))
	require.NoError(t, db.Create(&model.User{
		Id:       8811,
		Username: "auto-token-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		Quota:    1000,
	}).Error)
	tokens := []model.Token{
		{
			UserId:         8811,
			Key:            "deprecatedautotoken",
			Name:           "deprecated-auto-token",
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    -1,
			UnlimitedQuota: true,
			Group:          "auto",
		},
		{
			UserId:         8811,
			Key:            "emptygroupchaintoken",
			Name:           "empty-group-chain-token",
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    -1,
			UnlimitedQuota: true,
			Group:          "default",
		},
		{
			UserId:         8811,
			Key:            "mismatchedgroupchain",
			Name:           "mismatched-group-chain-token",
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    -1,
			UnlimitedQuota: true,
			Group:          "default",
			GroupChain:     model.StringArray{"vip", "default"},
		},
		{
			UserId:         8811,
			Key:            "explicitgroupchain",
			Name:           "explicit-group-chain-token",
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    -1,
			UnlimitedQuota: true,
			Group:          "default",
			GroupChain:     model.StringArray{"default"},
		},
	}
	require.NoError(t, db.Create(&tokens).Error)

	router := gin.New()
	router.GET("/v1/models", TokenAuth(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	tests := []struct {
		name       string
		key        string
		statusCode int
		message    string
	}{
		{name: "auto group", key: "deprecatedautotoken", statusCode: http.StatusForbidden, message: "auto"},
		{name: "empty chain", key: "emptygroupchaintoken", statusCode: http.StatusForbidden, message: "未配置分组链"},
		{name: "mismatched first group", key: "mismatchedgroupchain", statusCode: http.StatusForbidden, message: "首组与分组链不一致"},
		{name: "valid explicit chain", key: "explicitgroupchain", statusCode: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			request.Header.Set("Authorization", "Bearer sk-"+test.key)
			router.ServeHTTP(recorder, request)

			assert.Equal(t, test.statusCode, recorder.Code)
			if test.message != "" {
				assert.Contains(t, recorder.Body.String(), test.message)
			}
		})
	}
}
