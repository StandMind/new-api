package service

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeightedRouteChannelsOrdersWithoutReplacement(t *testing.T) {
	channels := []model.GroupModelRouteChannel{
		{ChannelID: 1, Weight: 100},
		{ChannelID: 2, Weight: 50},
		{ChannelID: 3, Weight: 0},
	}
	randomValues := []int64{120, 0, 0}
	randomIndex := 0

	ordered := weightedRouteChannels(channels, func(limit int64) int64 {
		require.Positive(t, limit)
		value := randomValues[randomIndex]
		randomIndex++
		require.Less(t, value, limit)
		return value
	})

	assert.Equal(t, []model.GroupModelRouteChannel{
		{ChannelID: 2, Weight: 50},
		{ChannelID: 1, Weight: 100},
		{ChannelID: 3, Weight: 0},
	}, ordered)
	assert.Equal(t, []model.GroupModelRouteChannel{
		{ChannelID: 1, Weight: 100},
		{ChannelID: 2, Weight: 50},
		{ChannelID: 3, Weight: 0},
	}, channels)
}

func TestWeightedRouteChannelsBoundsInheritedWeights(t *testing.T) {
	assert.Equal(t, int64(math.MaxInt32), boundedRouteWeight(^uint(0)))

	channels := []model.GroupModelRouteChannel{
		{ChannelID: 1, Weight: math.MaxInt64},
		{ChannelID: 2, Weight: 1},
	}
	call := 0
	ordered := weightedRouteChannels(channels, func(limit int64) int64 {
		call++
		if call == 1 {
			require.Equal(t, int64(math.MaxInt32)+1, limit)
			return math.MaxInt32
		}
		require.Equal(t, int64(math.MaxInt32), limit)
		return 0
	})
	require.Len(t, ordered, 2)
	assert.Equal(t, 2, ordered[0].ChannelID)
	assert.Equal(t, 1, ordered[1].ChannelID)
}

func TestRouteAttemptPlanPreferredChannelDoesNotMutatePlan(t *testing.T) {
	attempts := []RouteAttempt{
		{Group: "default", ChannelID: 1},
		{Group: "default", ChannelID: 2},
		{Group: "default", ChannelID: 3},
	}
	plan := &RouteAttemptPlan{
		configuredGroups: []string{"default"},
		attempts:         append([]RouteAttempt(nil), attempts...),
		consumed:         make(map[int]struct{}),
		currentIndex:     -1,
	}

	preferred, ok := plan.TakePreferred(3)
	require.True(t, ok)
	assert.Equal(t, 3, preferred.ChannelID)
	assert.Equal(t, attempts, plan.Attempts())

	first, ok := plan.NextAfterFailure()
	require.True(t, ok)
	assert.Equal(t, 1, first.ChannelID)
	second, ok := plan.NextAfterFailure()
	require.True(t, ok)
	assert.Equal(t, 2, second.ChannelID)
	_, ok = plan.NextAfterFailure()
	assert.False(t, ok)
}

func TestRouteAttemptPlanContinuesAcrossGroupsAfterFailure(t *testing.T) {
	plan := &RouteAttemptPlan{
		configuredGroups: []string{"a", "b"},
		attempts: []RouteAttempt{
			{Group: "a", ChannelID: 1},
			{Group: "a", ChannelID: 2},
			{Group: "b", ChannelID: 3},
		},
		consumed:     make(map[int]struct{}),
		currentIndex: -1,
		exhaustive:   true,
	}

	first, ok := plan.Next()
	require.True(t, ok)
	assert.Equal(t, 1, first.ChannelID)
	second, ok := plan.NextAfterFailure()
	require.True(t, ok)
	assert.Equal(t, 2, second.ChannelID)

	third, ok := plan.NextAfterFailure()
	require.True(t, ok)
	assert.Equal(t, 3, third.ChannelID)
	_, ok = plan.NextAfterFailure()
	assert.False(t, ok)
}

func TestExhaustiveRouteAttemptPlanTerminatesAfterEveryCandidateOnce(t *testing.T) {
	attempts := []RouteAttempt{
		{Group: "a", ChannelID: 1},
		{Group: "a", ChannelID: 2},
		{Group: "a", ChannelID: 3},
		{Group: "b", ChannelID: 4},
		{Group: "b", ChannelID: 5},
		{Group: "b", ChannelID: 6},
	}
	plan := &RouteAttemptPlan{
		configuredGroups: []string{"a", "b"},
		attempts:         attempts,
		consumed:         make(map[int]struct{}),
		currentIndex:     -1,
		exhaustive:       true,
	}

	seen := make(map[int]struct{})
	for {
		attempt, ok := plan.Next()
		if !ok {
			break
		}
		_, duplicate := seen[attempt.ChannelID]
		assert.False(t, duplicate)
		seen[attempt.ChannelID] = struct{}{}
	}

	assert.True(t, plan.IsExhaustive())
	assert.Len(t, seen, len(attempts))
}

func TestBuildRouteAttemptPlanUsesIndependentGroupModelRoutes(t *testing.T) {
	truncate(t)

	channelOnePriority := int64(10_000)
	channelTwoPriority := int64(-10_000)
	channelOneWeight := uint(900)
	channelTwoWeight := uint(1)
	channels := []model.Channel{
		{
			Id:       7001,
			Name:     "channel-one",
			Key:      "sk-one",
			Status:   common.ChannelStatusEnabled,
			Priority: &channelOnePriority,
			Weight:   &channelOneWeight,
		},
		{
			Id:       7002,
			Name:     "channel-two",
			Key:      "sk-two",
			Status:   common.ChannelStatusEnabled,
			Priority: &channelTwoPriority,
			Weight:   &channelTwoWeight,
		},
	}
	require.NoError(t, model.DB.Create(&channels).Error)

	vipOnePriority := int64(10)
	vipTwoPriority := int64(20)
	backupOnePriority := int64(40)
	backupTwoPriority := int64(30)
	abilities := []model.Ability{
		{Group: "vip-route-test", Model: "route-model", ChannelId: 7001, Enabled: true, Priority: &vipOnePriority, Weight: 10},
		{Group: "vip-route-test", Model: "route-model", ChannelId: 7002, Enabled: true, Priority: &vipTwoPriority, Weight: 20},
		{Group: "backup-route-test", Model: "route-model", ChannelId: 7001, Enabled: true, Priority: &backupOnePriority, Weight: 30},
		{Group: "backup-route-test", Model: "route-model", ChannelId: 7002, Enabled: true, Priority: &backupTwoPriority, Weight: 40},
	}
	require.NoError(t, model.DB.Create(&abilities).Error)

	require.NoError(t, model.SaveGroupModelRoute(&model.GroupModelRoute{
		Group: "vip-route-test",
		Model: "route-model",
		Tiers: model.GroupModelRouteTiers{
			{Priority: 100, Channels: []model.GroupModelRouteChannel{{ChannelID: 7001, Weight: 1}}},
			{Priority: 50, Channels: []model.GroupModelRouteChannel{{ChannelID: 7002, Weight: 1}}},
		},
	}))
	require.NoError(t, model.SaveGroupModelRoute(&model.GroupModelRoute{
		Group: "backup-route-test",
		Model: "route-model",
		Tiers: model.GroupModelRouteTiers{
			{Priority: 100, Channels: []model.GroupModelRouteChannel{{ChannelID: 7002, Weight: 1}}},
			{Priority: 50, Channels: []model.GroupModelRouteChannel{{ChannelID: 7001, Weight: 1}}},
		},
	}))

	plan, err := BuildRouteAttemptPlan(
		[]string{"vip-route-test", "vip-route-test", "backup-route-test", "vip-route-test"},
		"route-model",
		"/v1/chat/completions",
		true,
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"vip-route-test", "backup-route-test"}, plan.ConfiguredGroups())
	assert.Equal(t, []RouteAttempt{
		{Group: "vip-route-test", Model: "route-model", Priority: 100, ChannelID: 7001, Explicit: true},
		{Group: "vip-route-test", Model: "route-model", Priority: 50, ChannelID: 7002, Explicit: true},
		{Group: "backup-route-test", Model: "route-model", Priority: 100, ChannelID: 7002, Explicit: true},
		{Group: "backup-route-test", Model: "route-model", Priority: 50, ChannelID: 7001, Explicit: true},
	}, plan.Attempts())

	deleted, err := model.DeleteGroupModelRoute("vip-route-test", "route-model")
	require.NoError(t, err)
	require.True(t, deleted)

	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
	})

	common.MemoryCacheEnabled = false
	databasePlan, err := BuildRouteAttemptPlan(
		[]string{"vip-route-test"},
		"route-model",
		"/v1/chat/completions",
		false,
	)
	require.NoError(t, err)
	assert.Equal(t, []RouteAttempt{
		{Group: "vip-route-test", Model: "route-model", Priority: 20, ChannelID: 7002, Explicit: false},
		{Group: "vip-route-test", Model: "route-model", Priority: 10, ChannelID: 7001, Explicit: false},
	}, databasePlan.Attempts())

	common.MemoryCacheEnabled = true
	memoryPlan, err := BuildRouteAttemptPlan(
		[]string{"vip-route-test"},
		"route-model",
		"/v1/chat/completions",
		false,
	)
	require.NoError(t, err)
	assert.Equal(t, databasePlan.Attempts(), memoryPlan.Attempts())
}

func TestNormalizeGroupModelRouteRejectsInvalidTiers(t *testing.T) {
	tests := []model.GroupModelRoute{
		{
			Group: "default",
			Model: "model",
			Tiers: model.GroupModelRouteTiers{
				{Priority: 1, Channels: []model.GroupModelRouteChannel{{ChannelID: 1, Weight: 1}}},
				{Priority: 1, Channels: []model.GroupModelRouteChannel{{ChannelID: 2, Weight: 1}}},
			},
		},
		{
			Group: "default",
			Model: "model",
			Tiers: model.GroupModelRouteTiers{
				{Priority: 1, Channels: []model.GroupModelRouteChannel{
					{ChannelID: 1, Weight: 1},
					{ChannelID: 1, Weight: 2},
				}},
			},
		},
		{
			Group: "default",
			Model: "model",
			Tiers: model.GroupModelRouteTiers{
				{Priority: 1, Channels: []model.GroupModelRouteChannel{{ChannelID: 1, Weight: -1}}},
			},
		},
	}

	for index := range tests {
		assert.Error(t, model.NormalizeGroupModelRoute(&tests[index]))
	}
}
