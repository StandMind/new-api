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

func TestRoutingLogKeepsChannelDetailsAdminOnly(t *testing.T) {
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
	assert.Equal(t, "default", other["final_group"])
	assert.Equal(t, "group_model_ratio:exact", other["group_ratio_source"])
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	routing, ok := adminInfo["routing"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, plan.Attempts(), routing["planned"])
	assert.Equal(t, plan.Attempts(), routing["attempted"])

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
	require.Contains(t, adminOther, "admin_info")

	userLogs, userTotal, err := model.GetUserLogs(
		8201, model.LogTypeConsume, 0, 0, "route-log-model", "", 0, 10, "", "", "",
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), userTotal)
	require.Len(t, userLogs, 1)
	userOther, err := common.StrToMap(userLogs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, userOther, "admin_info")
	assert.Equal(t, "default", userOther["final_group"])
	assert.NotContains(t, userLogs[0].Other, "channel_id")
	assert.NotContains(t, userLogs[0].Other, "priority")
}
