package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriceOrderedGroupsUsesEffectiveModelPrice(t *testing.T) {
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalGroupModelRatio := ratio_setting.GroupModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(originalGroupModelRatio))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"cheap":1,"middle":0.5,"expensive":1}`))
	require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(`{"cheap":{"smart-model":0.4},"expensive":{"smart-model":1.5}}`))

	assert.Equal(t,
		[]string{"cheap", "middle", "expensive"},
		priceOrderedGroups("default", "smart-model", []string{"expensive", "middle", "cheap"}),
	)
}

func TestPerformanceRoutingFallsBackAndKeepsSparseGroupsInPriceOrder(t *testing.T) {
	priceOrder := []string{"cheap", "slow", "sparse", "fast"}
	stats := map[string]perfmetrics.RoutingStat{
		"slow": {RequestCount: 10, SuccessCount: 3, SuccessLatencyMs: 900},
		"fast": {RequestCount: 10, SuccessCount: 9, SuccessLatencyMs: 900},
	}

	ordered, basis := performanceOrderedGroups(
		constant.RoutingPrioritySpeed,
		"default",
		"smart-model",
		priceOrder,
		map[string]perfmetrics.RoutingStat{
			"slow": {RequestCount: 10, SuccessCount: 3, SuccessLatencyMs: 900},
			"fast": {RequestCount: 10, SuccessCount: 3, SuccessLatencyMs: 300},
		},
	)
	assert.Equal(t, "speed", basis)
	assert.Equal(t, []string{"cheap", "fast", "sparse", "slow"}, ordered)

	ordered, basis = performanceOrderedGroups(
		constant.RoutingPrioritySuccessRate,
		"default",
		"smart-model",
		priceOrder,
		stats,
	)
	assert.Equal(t, "success_rate", basis)
	assert.Equal(t, []string{"cheap", "fast", "sparse", "slow"}, ordered)

	ordered, basis = performanceOrderedGroups(
		constant.RoutingPrioritySuccessRate,
		"default",
		"smart-model",
		priceOrder,
		map[string]perfmetrics.RoutingStat{
			"fast": {RequestCount: 3, SuccessCount: 3, SuccessLatencyMs: 300},
		},
	)
	assert.Equal(t, "price_fallback", basis)
	assert.Equal(t, priceOrder, ordered)
}

func TestAutoRoutingCombinesSuccessPriceAndSpeed(t *testing.T) {
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"reliable":1,"cheap":0.5,"slow":1.5}`))

	ordered, basis := performanceOrderedGroups(
		constant.RoutingPriorityAuto,
		"default",
		"smart-model",
		[]string{"cheap", "reliable", "slow"},
		map[string]perfmetrics.RoutingStat{
			"cheap":    {RequestCount: 10, SuccessCount: 5, SuccessLatencyMs: 1000},
			"reliable": {RequestCount: 10, SuccessCount: 10, SuccessLatencyMs: 1000},
			"slow":     {RequestCount: 10, SuccessCount: 8, SuccessLatencyMs: 8000},
		},
	)

	assert.Equal(t, "auto", basis)
	assert.Equal(t, "reliable", ordered[0])
}

func TestSmartRoutingSnapshotRebuildAndFailureFallback(t *testing.T) {
	truncate(t)
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalGroupModelRatio := ratio_setting.GroupModelRatio2JSONString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(originalGroupModelRatio))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		require.NoError(t, model.DB.AutoMigrate(&model.PerfMetric{}))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"cheap":0.5,"expensive":1.5}`))
	require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(`{}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","cheap":"Cheap","expensive":"Expensive"}`))

	priority := int64(0)
	weight := uint(100)
	require.NoError(t, model.DB.Create(&[]model.Channel{
		{Id: 9101, Name: "snapshot-cheap", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Group: "cheap", Models: "snapshot-model", Priority: &priority, Weight: &weight},
		{Id: 9102, Name: "snapshot-expensive", Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Group: "expensive", Models: "snapshot-model", Priority: &priority, Weight: &weight},
	}).Error)
	require.NoError(t, model.DB.Create(&[]model.Ability{
		{Group: "cheap", Model: "snapshot-model", ChannelId: 9101, Enabled: true, Priority: &priority, Weight: weight},
		{Group: "expensive", Model: "snapshot-model", ChannelId: 9102, Enabled: true, Priority: &priority, Weight: weight},
	}).Error)
	require.NoError(t, model.InitGroupModelRouteIndex())
	require.NoError(t, RebuildSmartRoutingSnapshot())

	groups, basis := GetSmartRoutingGroups("default", "snapshot-model", "/v1/chat/completions", constant.RoutingPriorityPrice)
	assert.Equal(t, "price", basis)
	assert.Equal(t, []string{"cheap", "expensive"}, groups)

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"cheap":2,"expensive":0.25}`))
	require.NoError(t, RebuildSmartRoutingSnapshot())
	groups, _ = GetSmartRoutingGroups("default", "snapshot-model", "/v1/chat/completions", constant.RoutingPriorityPrice)
	assert.Equal(t, []string{"expensive", "cheap"}, groups)

	require.NoError(t, model.DB.Migrator().DropTable(&model.PerfMetric{}))
	assert.Error(t, RebuildSmartRoutingSnapshot())
	groups, basis = GetSmartRoutingGroups("default", "snapshot-model", "/v1/chat/completions", constant.RoutingPriorityPrice)
	assert.Equal(t, "price", basis)
	assert.Equal(t, []string{"expensive", "cheap"}, groups)
}
