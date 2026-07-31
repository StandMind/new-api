package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/official_price_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type groupModelRouteControllerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Explicit   bool                             `json:"explicit"`
		Route      model.GroupModelRoute            `json:"route"`
		Candidates []model.GroupModelRouteCandidate `json:"candidates"`
	} `json:"data"`
}

type groupModelRouteListControllerResponse struct {
	Success bool                            `json:"success"`
	Data    []model.GroupModelRouteListItem `json:"data"`
}

type userGroupsControllerResponse struct {
	Success bool                              `json:"success"`
	Data    map[string]map[string]interface{} `json:"data"`
}

type pricingControllerResponse struct {
	Success                  bool                          `json:"success"`
	Data                     []model.Pricing               `json:"data"`
	GroupModelRatio          map[string]map[string]float64 `json:"group_model_ratio"`
	EffectiveGroupModelRatio map[string]map[string]float64 `json:"effective_group_model_ratio"`
}

func withGroupRoutingSettings(t *testing.T) {
	t.Helper()

	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalGroupGroupRatio := ratio_setting.GroupGroupRatio2JSONString()
	originalGroupModelRatio := ratio_setting.GroupModelRatio2JSONString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(originalGroupGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(originalGroupModelRatio))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		model.InvalidatePricingCache()
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(
		`{"default":1,"vip":2}`,
	))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(
		`{"default":{"vip":1.4}}`,
	))
	require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(
		`{"vip":{"zz-priced-route-model":0.75,"zz-priced-*":0.8}}`,
	))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(
		`{"default":"Default","vip":"VIP"}`,
	))
	model.InvalidatePricingCache()
}

func TestGetUserGroupsPublishesUsableGroups(t *testing.T) {
	withGroupRoutingSettings(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       9101,
		Username: "group-chain-metadata-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self/groups", nil)
	ctx.Set("id", 9101)

	GetUserGroups(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response userGroupsControllerResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Contains(t, response.Data, "vip")
	assert.Equal(t, 1.4, response.Data["vip"]["ratio"])
}

func TestGetPricingReturnsRawAndEffectiveGroupModelRatios(t *testing.T) {
	withGroupRoutingSettings(t)
	savedOfficialPrices := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		if key == official_price_setting.OptionKey {
			savedOfficialPrices[key] = value
		}
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedOfficialPrices))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		official_price_setting.OptionKey: `{"zz-priced-route-model":{"unit":"usd_per_million_input_tokens","source_model":"official-route-model","verified_at":"2026-07-30","tiers":[{"price":1.25}]}}`,
	}))
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       9102,
		Username: "group-model-pricing-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Id:     9102,
		Name:   "pricing-route-channel",
		Key:    "pricing-route-key",
		Status: common.ChannelStatusEnabled,
		Group:  "vip",
		Models: "zz-priced-route-model",
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     "vip",
		Model:     "zz-priced-route-model",
		ChannelId: 9102,
		Enabled:   true,
	}).Error)
	model.InvalidatePricingCache()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	ctx.Set("id", 9102)

	GetPricing(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response pricingControllerResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, 0.75, response.GroupModelRatio["vip"]["zz-priced-route-model"])
	assert.Equal(t, 0.75, response.EffectiveGroupModelRatio["zz-priced-route-model"]["vip"])

	var pricing *model.Pricing
	for index := range response.Data {
		if response.Data[index].ModelName == "zz-priced-route-model" {
			pricing = &response.Data[index]
			break
		}
	}
	require.NotNil(t, pricing)
	assert.Equal(t, 0.75, pricing.EffectiveGroupRatio["vip"])
	require.NotNil(t, pricing.OfficialPrice)
	assert.Equal(t, official_price_setting.UnitUSDPerMillionInputTokens, pricing.OfficialPrice.Unit)
	assert.Equal(t, "official-route-model", pricing.OfficialPrice.SourceModel)
	require.Len(t, pricing.OfficialPrice.Tiers, 1)
	assert.Equal(t, 1.25, pricing.OfficialPrice.Tiers[0].Price)
}

func TestGroupModelRouteControllerSaveGetAndRestore(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.GroupModelRoute{}, &model.Log{}))
	require.NoError(t, db.Create(&model.User{
		Id:       9103,
		Username: "route-admin",
		Password: "password",
		Group:    "default",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
	}).Error)

	priorityHigh := int64(100)
	priorityLow := int64(50)
	require.NoError(t, db.Create(&[]model.Channel{
		{
			Id:     9111,
			Name:   "route-high",
			Key:    "route-high-key",
			Status: common.ChannelStatusEnabled,
			Group:  "vip",
			Models: "zz-route-api-model",
		},
		{
			Id:     9112,
			Name:   "route-low",
			Key:    "route-low-key",
			Status: common.ChannelStatusEnabled,
			Group:  "vip",
			Models: "zz-route-api-model",
		},
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{
			Group:     "vip",
			Model:     "zz-route-api-model",
			ChannelId: 9111,
			Enabled:   true,
			Priority:  &priorityHigh,
			Weight:    10,
		},
		{
			Group:     "vip",
			Model:     "zz-route-api-model",
			ChannelId: 9112,
			Enabled:   true,
			Priority:  &priorityLow,
			Weight:    20,
		},
	}).Error)

	route := model.GroupModelRoute{
		Group: "vip",
		Model: "zz-route-api-model",
		Tiers: model.GroupModelRouteTiers{
			{
				Priority: 200,
				Channels: []model.GroupModelRouteChannel{
					{ChannelID: 9112, Weight: 7},
				},
			},
			{
				Priority: 100,
				Channels: []model.GroupModelRouteChannel{
					{ChannelID: 9111, Weight: 3},
				},
			},
		},
	}
	body, err := common.Marshal(route)
	require.NoError(t, err)
	putRecorder := httptest.NewRecorder()
	putContext, _ := gin.CreateTestContext(putRecorder)
	putContext.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/group-model-routes",
		strings.NewReader(string(body)),
	)
	putContext.Request.Header.Set("Content-Type", "application/json")
	putContext.Set("id", 9103)
	putContext.Set("username", "route-admin")
	putContext.Set("role", common.RoleRootUser)

	PutGroupModelRoute(putContext)

	require.Equal(t, http.StatusOK, putRecorder.Code)
	var putResponse struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(putRecorder.Body.Bytes(), &putResponse))
	require.True(t, putResponse.Success)

	listRecorder := httptest.NewRecorder()
	listContext, _ := gin.CreateTestContext(listRecorder)
	listContext.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/group-model-routes/list",
		nil,
	)

	ListGroupModelRoutes(listContext)

	require.Equal(t, http.StatusOK, listRecorder.Code)
	var listResponse groupModelRouteListControllerResponse
	require.NoError(t, common.Unmarshal(listRecorder.Body.Bytes(), &listResponse))
	require.True(t, listResponse.Success)
	require.Len(t, listResponse.Data, 1)
	assert.Equal(t, "vip", listResponse.Data[0].Group)
	assert.Equal(t, "zz-route-api-model", listResponse.Data[0].Model)
	require.Len(t, listResponse.Data[0].Tiers, 2)
	require.Len(t, listResponse.Data[0].Tiers[0].Channels, 1)
	assert.Equal(t, "route-low", listResponse.Data[0].Tiers[0].Channels[0].ChannelName)
	assert.True(t, listResponse.Data[0].Tiers[0].Channels[0].Eligible)

	require.NoError(t, db.Model(&model.Ability{}).
		Where("channel_id = ?", 9112).
		Update("enabled", false).Error)

	unavailableListRecorder := httptest.NewRecorder()
	unavailableListContext, _ := gin.CreateTestContext(unavailableListRecorder)
	unavailableListContext.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/group-model-routes/list",
		nil,
	)
	ListGroupModelRoutes(unavailableListContext)

	var unavailableListResponse groupModelRouteListControllerResponse
	require.NoError(t, common.Unmarshal(unavailableListRecorder.Body.Bytes(), &unavailableListResponse))
	require.True(t, unavailableListResponse.Success)
	require.Len(t, unavailableListResponse.Data, 1)
	require.Len(t, unavailableListResponse.Data[0].Tiers[0].Channels, 1)
	assert.Equal(t, "route-low", unavailableListResponse.Data[0].Tiers[0].Channels[0].ChannelName)
	assert.False(t, unavailableListResponse.Data[0].Tiers[0].Channels[0].Eligible)

	require.NoError(t, db.Model(&model.Ability{}).
		Where("channel_id = ?", 9112).
		Update("enabled", true).Error)

	require.NoError(t, db.Delete(&model.Channel{}, 9112).Error)

	deletedChannelListRecorder := httptest.NewRecorder()
	deletedChannelListContext, _ := gin.CreateTestContext(deletedChannelListRecorder)
	deletedChannelListContext.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/group-model-routes/list",
		nil,
	)
	ListGroupModelRoutes(deletedChannelListContext)

	var deletedChannelListResponse groupModelRouteListControllerResponse
	require.NoError(t, common.Unmarshal(deletedChannelListRecorder.Body.Bytes(), &deletedChannelListResponse))
	require.True(t, deletedChannelListResponse.Success)
	require.Len(t, deletedChannelListResponse.Data, 1)
	require.Len(t, deletedChannelListResponse.Data[0].Tiers[0].Channels, 1)
	assert.Empty(t, deletedChannelListResponse.Data[0].Tiers[0].Channels[0].ChannelName)
	assert.False(t, deletedChannelListResponse.Data[0].Tiers[0].Channels[0].Eligible)

	require.NoError(t, db.Create(&model.Channel{
		Id:     9112,
		Name:   "route-low",
		Key:    "route-low-key",
		Status: common.ChannelStatusEnabled,
		Group:  "vip",
		Models: "zz-route-api-model",
	}).Error)

	getRecorder := httptest.NewRecorder()
	getContext, _ := gin.CreateTestContext(getRecorder)
	getContext.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/group-model-routes?group=vip&model=zz-route-api-model",
		nil,
	)

	GetGroupModelRoute(getContext)

	var getResponse groupModelRouteControllerResponse
	require.NoError(t, common.Unmarshal(getRecorder.Body.Bytes(), &getResponse))
	require.True(t, getResponse.Success)
	assert.True(t, getResponse.Data.Explicit)
	require.Len(t, getResponse.Data.Route.Tiers, 2)
	assert.Equal(t, 9112, getResponse.Data.Route.Tiers[0].Channels[0].ChannelID)
	assert.Len(t, getResponse.Data.Candidates, 2)

	deleteRecorder := httptest.NewRecorder()
	deleteContext, _ := gin.CreateTestContext(deleteRecorder)
	deleteContext.Request = httptest.NewRequest(
		http.MethodDelete,
		"/api/group-model-routes?group=vip&model=zz-route-api-model",
		nil,
	)
	deleteContext.Set("id", 9103)
	deleteContext.Set("username", "route-admin")
	deleteContext.Set("role", common.RoleRootUser)

	DeleteGroupModelRoute(deleteContext)

	var deletedRoute model.GroupModelRoute
	result := db.Where(&model.GroupModelRoute{
		Group: "vip",
		Model: "zz-route-api-model",
	}).First(&deletedRoute)
	require.Error(t, result.Error)

	emptyListRecorder := httptest.NewRecorder()
	emptyListContext, _ := gin.CreateTestContext(emptyListRecorder)
	emptyListContext.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/group-model-routes/list",
		nil,
	)

	ListGroupModelRoutes(emptyListContext)

	var emptyListResponse groupModelRouteListControllerResponse
	require.NoError(t, common.Unmarshal(emptyListRecorder.Body.Bytes(), &emptyListResponse))
	require.True(t, emptyListResponse.Success)
	assert.Empty(t, emptyListResponse.Data)

	inheritedRecorder := httptest.NewRecorder()
	inheritedContext, _ := gin.CreateTestContext(inheritedRecorder)
	inheritedContext.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/group-model-routes?group=vip&model=zz-route-api-model",
		nil,
	)
	GetGroupModelRoute(inheritedContext)

	var inheritedResponse groupModelRouteControllerResponse
	require.NoError(t, common.Unmarshal(inheritedRecorder.Body.Bytes(), &inheritedResponse))
	require.True(t, inheritedResponse.Success)
	assert.False(t, inheritedResponse.Data.Explicit)
	require.Len(t, inheritedResponse.Data.Route.Tiers, 2)
	assert.Equal(t, int64(100), inheritedResponse.Data.Route.Tiers[0].Priority)

	var auditCount int64
	require.NoError(t, db.Model(&model.Log{}).
		Where("type = ?", model.LogTypeManage).
		Where("content IN ?", []string{
			"Updated group model route vip/zz-route-api-model",
			"Deleted group model route vip/zz-route-api-model",
		}).
		Count(&auditCount).Error)
	assert.Equal(t, int64(2), auditCount)
}
