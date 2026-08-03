package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAccessPolicyControllerTestDB(t *testing.T) {
	t.Helper()
	groupRatioJSON := ratio_setting.GroupRatio2JSONString()
	groupGroupRatioJSON := ratio_setting.GroupGroupRatio2JSONString()
	userUsableGroupsJSON := setting.UserUsableGroups2JSONString()
	topupGroupRatioJSON := common.TopupGroupRatio2JSONString()
	modelRequestRateLimitGroupJSON := setting.ModelRequestRateLimitGroup2JSONString()
	specialUsableGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.ReadAll()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(groupRatioJSON))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(groupGroupRatioJSON))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(userUsableGroupsJSON))
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(topupGroupRatioJSON))
		require.NoError(t, setting.UpdateModelRequestRateLimitGroupByJSONString(modelRequestRateLimitGroupJSON))
		specialUsableGroupSetting := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
		specialUsableGroupSetting.Clear()
		specialUsableGroupSetting.AddAll(specialUsableGroups)
	})
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.Option{}, &model.Token{}, &model.GroupModelRoute{},
		&model.SubscriptionPlan{}, &model.UserSubscription{},
		&model.OpenLuxPriceSyncBinding{}, &model.OpenLuxPriceSyncState{},
		&model.UserLevel{}, &model.RouteGroup{}, &model.UserLevelRouteGroup{},
		&model.AccessPolicyState{}, &model.AccessPolicyMigrationGuard{},
	))
	require.NoError(t, model.RebuildAccessPolicySnapshot())
}

func accessPolicyRequest(t *testing.T, method, target string, body interface{}, handler func(c *gin.Context), params ...gin.Param) (*httptest.ResponseRecorder, tokenAPIResponse) {
	t.Helper()
	ctx, recorder := newAuthenticatedContext(t, method, target, body, 1)
	ctx.Params = append(ctx.Params, params...)
	handler(ctx)
	return recorder, decodeAPIResponse(t, recorder)
}

func TestAccessPolicyCRUDKeepsLevelsAndRoutesIndependent(t *testing.T) {
	setupAccessPolicyControllerTestDB(t)

	_, response := accessPolicyRequest(t, http.MethodPost, "/api/route-groups", map[string]interface{}{
		"code": "route-new", "name": "New route", "description": "test route",
		"base_ratio": 1.5, "enabled": true,
	}, CreateRouteGroup)
	require.True(t, response.Success, response.Message)

	var standardGrantCount int64
	require.NoError(t, model.DB.Model(&model.UserLevelRouteGroup{}).
		Where("user_level_code = ? AND route_group_code = ?", model.StandardUserLevelCode, "route-new").
		Count(&standardGrantCount).Error)
	assert.Zero(t, standardGrantCount, "new route groups must not be granted automatically")

	_, response = accessPolicyRequest(t, http.MethodPost, "/api/user-levels", map[string]interface{}{
		"code": "gold", "name": "Gold", "description": "paid level",
		"is_default": false, "enabled": true, "topup_ratio": 0.9,
		"request_limit": 100, "success_request_limit": 80,
	}, CreateUserLevel)
	require.True(t, response.Success, response.Message)

	priceOverride := 0.75
	_, response = accessPolicyRequest(t, http.MethodPut, "/api/user-levels/gold/route-groups", map[string]interface{}{
		"route_groups": []map[string]interface{}{{"code": "route-new", "price_ratio": priceOverride}},
	}, ReplaceUserLevelRouteGroups, gin.Param{Key: "code", Value: "gold"})
	require.True(t, response.Success, response.Message)
	require.True(t, model.UserLevelCanAccessRouteGroup("gold", "route-new"))
	ratio, source := model.ResolveAccessPolicyRatio("gold", "route-new", "unconfigured-model")
	assert.InDelta(t, priceOverride, ratio, 0.0000001)
	assert.Equal(t, "user_level_route_group.price_ratio", source)

	_, response = accessPolicyRequest(t, http.MethodPost, "/api/user-levels", map[string]interface{}{
		"code": "premium", "name": "Premium", "description": "new default",
		"is_default": true, "enabled": true, "topup_ratio": 1,
		"request_limit": 0, "success_request_limit": 0,
	}, CreateUserLevel)
	require.True(t, response.Success, response.Message)
	var defaults []model.UserLevel
	require.NoError(t, model.DB.Where("is_default = ?", true).Find(&defaults).Error)
	require.Len(t, defaults, 1)
	assert.Equal(t, "premium", defaults[0].Code)

	_, response = accessPolicyRequest(t, http.MethodPut, "/api/user-levels/premium", map[string]interface{}{
		"name": "Premium", "description": "new default", "is_default": false,
		"enabled": true, "topup_ratio": 1, "request_limit": 0, "success_request_limit": 0,
	}, UpdateUserLevel, gin.Param{Key: "code", Value: "premium"})
	assert.False(t, response.Success)
	require.NoError(t, model.DB.Where("is_default = ?", true).Find(&defaults).Error)
	require.Len(t, defaults, 1)
	assert.Equal(t, "premium", defaults[0].Code)
}

func TestAccessPolicyRejectsReservedLevelAndReferencedDeletes(t *testing.T) {
	setupAccessPolicyControllerTestDB(t)

	_, response := accessPolicyRequest(t, http.MethodPost, "/api/user-levels", map[string]interface{}{
		"code": model.LegacyDefaultGroupCode, "name": "Legacy", "is_default": false,
		"enabled": true, "topup_ratio": 1, "request_limit": 0, "success_request_limit": 0,
	}, CreateUserLevel)
	assert.False(t, response.Success)
	assert.Contains(t, response.Message, "保留值")

	require.NoError(t, model.DB.Create(&model.UserLevel{
		Code: "referenced", Name: "Referenced", Enabled: true, TopupRatio: 1,
	}).Error)
	require.NoError(t, model.DB.Create(&model.RouteGroup{
		Code: "route-used", Name: "Used route", BaseRatio: 1, Enabled: true,
	}).Error)
	require.NoError(t, model.DB.Create(&model.UserLevelRouteGroup{
		UserLevelCode: "referenced", RouteGroupCode: "route-used",
	}).Error)
	require.NoError(t, model.DB.Create(&model.User{
		Id: 9901, Username: "access-policy-user", Password: "password",
		UserLevel: "referenced", Status: common.UserStatusEnabled,
		AffCode: "ap9901",
	}).Error)
	require.NoError(t, model.RebuildAccessPolicySnapshot())

	recorder, response := accessPolicyRequest(t, http.MethodDelete, "/api/user-levels/referenced", nil,
		DeleteUserLevel, gin.Param{Key: "code", Value: "referenced"})
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.False(t, response.Success)

	recorder, response = accessPolicyRequest(t, http.MethodDelete, "/api/route-groups/route-used", nil,
		DeleteRouteGroup, gin.Param{Key: "code", Value: "route-used"})
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.False(t, response.Success)
}
