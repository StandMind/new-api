package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChannelOwnerNameUsesAdaptorChannelName(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		expected    string
	}{
		{
			name:        "openai",
			channelType: constant.ChannelTypeOpenAI,
			expected:    "openai",
		},
		{
			name:        "codex",
			channelType: constant.ChannelTypeCodex,
			expected:    "codex",
		},
		{
			name:        "openrouter",
			channelType: constant.ChannelTypeOpenRouter,
			expected:    "openrouter",
		},
		{
			name:        "azure fallback",
			channelType: constant.ChannelTypeAzure,
			expected:    "azure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, channelOwnerName(tt.channelType))
		})
	}
}

func TestBuildOpenAIModelOverridesOwnedBy(t *testing.T) {
	modelItem := buildOpenAIModel("gpt-5.4", map[string]string{"gpt-5.4": "openai"})
	require.Equal(t, "gpt-5.4", modelItem.Id)
	require.Equal(t, "openai", modelItem.OwnedBy)
}

func TestBuildOpenAIModelFallsBackToCustomForUnknownModels(t *testing.T) {
	modelItem := buildOpenAIModel("custom-test-model", nil)
	require.Equal(t, "custom-test-model", modelItem.Id)
	require.Equal(t, "custom", modelItem.OwnedBy)
}

func TestGetModelListGroupsUsesAllGrantedRouteGroupsWhenTokenGroupIsEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.UserLevel{}, &model.RouteGroup{}, &model.UserLevelRouteGroup{}))
	require.NoError(t, db.Create(&model.UserLevel{Code: "standard", Name: "Standard", IsDefault: true, Enabled: true, TopupRatio: 1}).Error)
	require.NoError(t, db.Create(&[]model.RouteGroup{
		{Code: "route-a", Name: "Route A", BaseRatio: 1, Enabled: true},
		{Code: "route-b", Name: "Route B", BaseRatio: 1, Enabled: true},
	}).Error)
	require.NoError(t, db.Create(&[]model.UserLevelRouteGroup{
		{UserLevelCode: "standard", RouteGroupCode: "route-a"},
		{UserLevelCode: "standard", RouteGroupCode: "route-b"},
	}).Error)
	require.NoError(t, model.RebuildAccessPolicySnapshot())
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "standard")

	groups, err := getModelListGroups(ctx)
	require.NoError(t, err)

	require.Equal(t, "standard", groups.userGroup)
	require.Empty(t, groups.tokenGroup)
	require.Equal(t, []string{"route-a", "route-b"}, groups.ownerGroups)
}

func TestGetModelListGroupsUsesExplicitTokenGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenGroup, "vip")

	groups, err := getModelListGroups(ctx)
	require.NoError(t, err)

	require.Equal(t, "default", groups.userGroup)
	require.Equal(t, "vip", groups.tokenGroup)
	require.Equal(t, []string{"vip"}, groups.ownerGroups)
}
