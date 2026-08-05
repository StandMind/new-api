package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutingSuccessLogKeepsSummaryOnly(t *testing.T) {
	truncate(t)

	context, _ := gin.CreateTestContext(nil)
	plan := &RouteAttemptPlan{
		configuredGroups: []string{"vip", "default"},
		attempts: []RouteAttempt{
			{Group: "vip", Model: "route-log-model", Priority: 100, ChannelID: 8101, Explicit: true},
			{Group: "default", Model: "route-log-model", Priority: 50, ChannelID: 8102, Explicit: false},
		},
		consumed:     make(map[int]struct{}),
		currentIndex: -1,
		exhaustive:   true,
	}
	SetRouteAttemptPlan(context, plan)
	for _, attempt := range plan.Attempts() {
		common.SetContextKey(context, constant.ContextKeyRouteAttempt, attempt)
		RecordRouteUpstreamAttempt(context)
	}

	relayInfo := &relaycommon.RelayInfo{
		UsingGroup: "default",
		PriceData: types.PriceData{
			GroupRatioInfo: types.GroupRatioInfo{Source: "group_model_ratio:exact"},
		},
	}
	other := map[string]interface{}{}
	AppendRoutingLogInfo(context, relayInfo, other)

	assert.Equal(t, []string{"vip", "default"}, other["group_chain"])
	assert.Equal(t, []string{"vip", "default"}, other["attempted_groups"])
	assert.Equal(t, RouteModeManual, other["routing_mode"])
	assert.Equal(t, "default", other["final_group"])
	assert.Equal(t, "group_model_ratio:exact", other["group_ratio_source"])
	assert.NotContains(t, other, "admin_info")

	require.NoError(t, model.DB.Create(&model.Log{
		UserId:    8201,
		CreatedAt: 1,
		Type:      model.LogTypeConsume,
		ModelName: "route-log-model",
		Group:     "default",
		Other:     common.MapToJsonStr(other),
	}).Error)

	adminLogs, adminTotal, err := model.GetAllLogs(
		model.LogTypeConsume, 0, 0, "route-log-model", "", "", 0, 10, 0, "", "", "",
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), adminTotal)
	require.Len(t, adminLogs, 1)
	adminOther, err := common.StrToMap(adminLogs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, adminOther, "admin_info")

	userLogs, userTotal, err := model.GetUserLogs(
		8201, model.LogTypeConsume, 0, 0, "route-log-model", "", 0, 10, "", "", "",
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), userTotal)
	require.Len(t, userLogs, 1)
	userOther, err := common.StrToMap(userLogs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, userOther, "admin_info")
	assert.Equal(t, RouteModeManual, userOther["routing_mode"])
	assert.Equal(t, "default", userOther["final_group"])
	assert.NotContains(t, userLogs[0].Other, "channel_id")
	assert.NotContains(t, userLogs[0].Other, "priority")
}

func TestRoutingSnapshotIncludesSmartAndFixedModes(t *testing.T) {
	smartContext, _ := gin.CreateTestContext(nil)
	smartPlan := NewRouteAttemptPlan([]string{"fast", "cheap"}, true)
	smartPlan.SetSmartRouting(constant.RoutingPrioritySpeed, "price_fallback")
	SetRouteAttemptPlan(smartContext, smartPlan)

	smartSnapshot := BuildRoutingDiagnosticSnapshot(smartContext, "fast")
	require.NotNil(t, smartSnapshot)
	assert.Equal(t, string(constant.RoutingPrioritySpeed), smartSnapshot.Mode)
	assert.Equal(t, "price_fallback", smartSnapshot.Basis)
	assert.Equal(t, []string{"fast", "cheap"}, smartSnapshot.Groups)

	fixedContext, _ := gin.CreateTestContext(nil)
	common.SetContextKey(fixedContext, constant.ContextKeyTokenSpecificChannelId, "42")
	common.SetContextKey(fixedContext, constant.ContextKeyChannelId, 42)
	common.SetContextKey(fixedContext, constant.ContextKeyChannelName, "fixed-channel")
	fixedSnapshot := BuildRoutingDiagnosticSnapshot(fixedContext, "fixed")
	require.NotNil(t, fixedSnapshot)
	assert.Equal(t, RouteModeFixedChannel, fixedSnapshot.Mode)
	assert.Equal(t, 42, fixedSnapshot.FinalChannelID)
	assert.Equal(t, "fixed-channel", fixedSnapshot.FinalChannelName)
}

func TestRoutingErrorLogKeepsAttemptDetailsAdminOnly(t *testing.T) {
	truncate(t)

	context, _ := gin.CreateTestContext(nil)
	plan := &RouteAttemptPlan{
		configuredGroups: []string{"fast", "fallback"},
		attempts: []RouteAttempt{{
			Group: "fast", Model: "error-route-model", Priority: 100,
			ChannelID: 8301, ChannelName: "failed-channel",
		}},
		consumed:     make(map[int]struct{}),
		currentIndex: -1,
		exhaustive:   true,
	}
	plan.SetSmartRouting(constant.RoutingPriorityPrice, "configured_price")
	SetRouteAttemptPlan(context, plan)
	attempt, ok := plan.Next()
	require.True(t, ok)
	common.SetContextKey(context, constant.ContextKeyRouteAttempt, attempt)
	common.SetContextKey(context, constant.ContextKeyUsingGroup, attempt.Group)
	RecordRouteUpstreamAttempt(context)
	RecordCurrentRouteAttemptResult(context, RouteAttemptResult{
		Phase:           RouteAttemptPhaseUpstream,
		Outcome:         RouteAttemptOutcomeFailed,
		StatusCode:      429,
		ErrorCode:       "insufficient_user_quota",
		ErrorMessage:    "status_code=429, insufficient quota",
		RetryDecision:   RouteRetryDecisionStop,
		RetryStopReason: "retry_limit_reached",
	})

	intermediateOther := map[string]interface{}{}
	AppendRoutingErrorLogInfo(context, intermediateOther, false)
	assert.Equal(t, string(constant.RoutingPriorityPrice), intermediateOther["routing_mode"])
	assert.NotContains(t, intermediateOther, "admin_info")

	other := map[string]interface{}{}
	AppendRoutingErrorLogInfo(context, other, true)
	require.NoError(t, model.DB.Create(&model.Log{
		UserId: 8302, CreatedAt: 1, Type: model.LogTypeError,
		ModelName: "error-route-model", Other: common.MapToJsonStr(other),
	}).Error)

	adminLogs, _, err := model.GetAllLogs(
		model.LogTypeError, 0, 0, "error-route-model", "", "", 0, 10, 0, "", "", "",
	)
	require.NoError(t, err)
	require.Len(t, adminLogs, 1)
	adminOther, err := common.StrToMap(adminLogs[0].Other)
	require.NoError(t, err)
	adminRouting := adminOther["admin_info"].(map[string]interface{})["routing"].(map[string]interface{})
	attempts := adminRouting["attempts"].([]interface{})
	require.Len(t, attempts, 1)
	assert.Equal(t, float64(429), attempts[0].(map[string]interface{})["status_code"])
	assert.Equal(t, "retry_limit_reached", adminRouting["final_stop_reason"])

	userLogs, _, err := model.GetUserLogs(
		8302, model.LogTypeError, 0, 0, "error-route-model", "", 0, 10, "", "", "",
	)
	require.NoError(t, err)
	require.Len(t, userLogs, 1)
	userOther, err := common.StrToMap(userLogs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, userOther, "admin_info")
	assert.Equal(t, string(constant.RoutingPriorityPrice), userOther["routing_mode"])
	assert.Equal(t, []interface{}{"fast", "fallback"}, userOther["group_chain"])
}
