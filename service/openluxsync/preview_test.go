package openluxsync

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func optionJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

func previewOptions(t *testing.T, localGroup, modelName string, currentGroupModelRatio float64) map[string]string {
	t.Helper()
	return map[string]string{
		optionModelRatio:           optionJSON(t, map[string]float64{modelName: 1}),
		optionModelPrice:           `{}`,
		optionCompletionRatio:      optionJSON(t, map[string]float64{modelName: 1}),
		optionCacheRatio:           optionJSON(t, map[string]float64{modelName: 1}),
		optionCreateCacheRatio:     optionJSON(t, map[string]float64{modelName: 1.25}),
		optionImageRatio:           optionJSON(t, map[string]float64{modelName: 1}),
		optionAudioRatio:           optionJSON(t, map[string]float64{modelName: 1}),
		optionAudioCompletionRatio: optionJSON(t, map[string]float64{modelName: 1}),
		optionGroupModelRatio: optionJSON(t, map[string]map[string]float64{
			localGroup: {modelName: currentGroupModelRatio},
		}),
		optionBillingMode: `{}`,
		optionBillingExpr: `{}`,
	}
}

func previewChannel(id int, name, localGroup, models, baseURL string) model.Channel {
	return model.Channel{
		Id: id, Name: name, Group: localGroup, Models: models, BaseURL: &baseURL,
		Key: "not-a-real-key",
	}
}

func previewSource(models []sourceModel, groups map[string]decimal.Decimal) *sourceSnapshot {
	source := &sourceSnapshot{
		Models:           make(map[string]sourceModel, len(models)),
		OrderedModels:    append([]sourceModel(nil), models...),
		Groups:           make(map[string]sourceGroup, len(groups)),
		ReferencedGroups: make(map[string]struct{}),
		Hash:             strings.Repeat("a", 64),
		FetchedAt:        123,
	}
	for _, item := range models {
		source.Models[item.Name] = item
		for _, group := range item.EnableGroups {
			source.ReferencedGroups[group] = struct{}{}
		}
	}
	for name, ratio := range groups {
		source.Groups[name] = sourceGroup{Name: name, Description: name, Ratio: ratio}
	}
	return source
}

func previewSnapshot(t *testing.T, sourceGroup, localGroup, modelName string, currentRatio float64, channels []model.Channel, abilities []model.Ability) *model.OpenLuxSyncSnapshot {
	t.Helper()
	return &model.OpenLuxSyncSnapshot{
		BindingRevision: 4,
		Bindings: []model.OpenLuxPriceSyncBinding{{
			ID: 1, SourceGroup: sourceGroup, ChannelID: channels[0].Id,
		}},
		RouteGroups: []model.RouteGroup{{Code: localGroup, Name: localGroup, BaseRatio: 1, Enabled: true}},
		Channels:    channels,
		Abilities:   abilities,
		Options:     previewOptions(t, localGroup, modelName, currentRatio),
	}
}

func findPreviewChange(t *testing.T, preview *PreviewResponse, kind, modelName string) Change {
	t.Helper()
	for _, change := range preview.Changes {
		if change.Kind == kind && change.Model == modelName {
			return change
		}
	}
	require.FailNow(t, "preview change not found", "kind=%s model=%s", kind, modelName)
	return Change{}
}

func hasNotice(preview *PreviewResponse, kind, modelName string) bool {
	for _, notice := range preview.Notices {
		if notice.Kind == kind && (modelName == "" || notice.Model == modelName) {
			return true
		}
	}
	return false
}

func TestPreviewUsesRenamedLocalGroupAndLeavesMixedChannelConfigurationUntouched(t *testing.T) {
	const (
		sourceGroup = "OpenLux-Source"
		localGroup  = "renamed-local-group"
		modelName   = "openlux-sync-mixed-model"
	)
	bound := previewChannel(1, "OpenLux/OpenLux-Source", localGroup, modelName, "https://api.openlux.ai")
	selfHosted := previewChannel(2, "self-hosted-pool", localGroup, modelName, "https://pool.internal.example")
	snapshot := previewSnapshot(t, sourceGroup, localGroup, modelName, 0.8, []model.Channel{bound, selfHosted}, []model.Ability{
		{ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true},
		{ChannelId: selfHosted.Id, Group: localGroup, Model: modelName, Enabled: true},
	})
	source := previewSource([]sourceModel{{
		Name: modelName, QuotaType: 0, ModelRatio: decimal.NewFromInt(2), EnableGroups: []string{sourceGroup},
	}}, map[string]decimal.Decimal{sourceGroup: decimal.RequireFromString("0.5")})

	bindings := resolveBindings(snapshot)
	require.Len(t, bindings, 1)
	assert.Equal(t, localGroup, bindings[0].LocalGroup)
	assert.Equal(t, 1, bindings[0].MixedChannelCount)

	preview, err := buildPreview(nil, source, snapshot, false)
	require.NoError(t, err)
	change := findPreviewChange(t, preview, "group_model_price_update", modelName)
	assert.True(t, change.Actionable)
	assert.Equal(t, "1.3", change.TargetValue)
	assert.Equal(t, 1, change.MixedChannelCount)

	mutation, _, err := aggregateChanges(snapshot, []Change{change})
	require.NoError(t, err)
	assert.Empty(t, mutation.ChannelModels)
	assert.Empty(t, mutation.AbilityRemovals)
	assert.Empty(t, mutation.DeleteChannelIDs)
	assert.Equal(t, modelName, selfHosted.Models)

	var ratios map[string]map[string]json.RawMessage
	require.NoError(t, common.UnmarshalJsonStr(mutation.Options[optionGroupModelRatio], &ratios))
	assert.JSONEq(t, `1.3`, string(ratios[localGroup][modelName]))
}

func TestMixedGroupModelRemovalRetainsSelfHostedPriceAndHonorsExplicitRoutes(t *testing.T) {
	const (
		sourceGroup = "OpenLux-Source"
		localGroup  = "mixed-local"
		modelName   = "openlux-sync-removed-model"
	)
	bound := previewChannel(11, "OpenLux/OpenLux-Source", localGroup, modelName, "https://api.openlux.ai")
	selfHosted := previewChannel(12, "self-hosted-pool", localGroup, modelName, "https://pool.internal.example")
	snapshot := previewSnapshot(t, sourceGroup, localGroup, modelName, 0.7, []model.Channel{bound, selfHosted}, []model.Ability{
		{ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true},
		{ChannelId: selfHosted.Id, Group: localGroup, Model: modelName, Enabled: true},
	})
	source := previewSource([]sourceModel{{
		Name: modelName, QuotaType: 0, ModelRatio: decimal.NewFromInt(1),
		CompletionRatio: decimalPointer("3"), EnableGroups: []string{"another-source"},
	}}, map[string]decimal.Decimal{sourceGroup: decimal.NewFromInt(1)})

	preview, err := buildPreview(nil, source, snapshot, false)
	require.NoError(t, err)
	change := findPreviewChange(t, preview, "group_model_remove", modelName)
	assert.True(t, change.Actionable)
	assert.Equal(t, "openlux_only_price_retained", change.RemoveMode)
	assert.Empty(t, change.mutations)
	for _, previewChange := range preview.Changes {
		assert.NotEqual(t, "model_billing_update", previewChange.Kind)
	}

	mutation, _, err := aggregateChanges(snapshot, []Change{change})
	require.NoError(t, err)
	assert.Equal(t, "", mutation.ChannelModels[bound.Id])
	assert.NotContains(t, mutation.ChannelModels, selfHosted.Id)
	assert.Empty(t, mutation.Options)
	require.Len(t, mutation.AbilityRemovals, 1)
	assert.Equal(t, bound.Id, mutation.AbilityRemovals[0].ChannelID)

	const routedGroup = "another-local-group"
	snapshot.Routes = []model.GroupModelRoute{{
		Group: routedGroup,
		Model: modelName,
		Tiers: model.GroupModelRouteTiers{{
			Priority: 0,
			Channels: []model.GroupModelRouteChannel{{ChannelID: bound.Id, Weight: 1}},
		}},
	}}
	blockedPreview, err := buildPreview(nil, source, snapshot, false)
	require.NoError(t, err)
	blocked := findPreviewChange(t, blockedPreview, "group_model_remove", modelName)
	assert.False(t, blocked.Actionable)
	reasons := strings.Join(blocked.BlockedReasons, " ")
	assert.Contains(t, reasons, "显式模型路由")
	assert.Contains(t, reasons, routedGroup+"/"+modelName)
}

func TestPreviewSurfacesAbilityNotPresentInBoundChannelModelList(t *testing.T) {
	const (
		sourceGroup = "OpenLux-Source"
		localGroup  = "ability-anomaly"
		modelName   = "openlux-sync-orphan-ability"
	)
	bound := previewChannel(21, "OpenLux/OpenLux-Source", localGroup, "", "https://api.openlux.ai")
	snapshot := previewSnapshot(t, sourceGroup, localGroup, modelName, 0.5, []model.Channel{bound}, []model.Ability{
		{ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true},
	})
	source := previewSource([]sourceModel{{
		Name: modelName, QuotaType: 0, ModelRatio: decimal.NewFromInt(2), EnableGroups: []string{sourceGroup},
	}}, map[string]decimal.Decimal{sourceGroup: decimal.NewFromInt(1)})

	preview, err := buildPreview(nil, source, snapshot, false)
	require.NoError(t, err)
	change := findPreviewChange(t, preview, "group_model_price_update", modelName)
	assert.False(t, change.Actionable)
	assert.Contains(t, strings.Join(change.BlockedReasons, " "), "Ability 与模型列表不一致")
}

func TestPreviewOnlyWarnsWhenModelOrGroupRegistryDisappears(t *testing.T) {
	const (
		sourceGroup = "OpenLux-Source"
		localGroup  = "local-group"
		modelName   = "openlux-sync-disappeared-model"
	)
	bound := previewChannel(31, "OpenLux/OpenLux-Source", localGroup, modelName, "https://api.openlux.ai")
	snapshot := previewSnapshot(t, sourceGroup, localGroup, modelName, 1, []model.Channel{bound}, []model.Ability{
		{ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true},
	})

	modelMissing := previewSource(nil, map[string]decimal.Decimal{sourceGroup: decimal.NewFromInt(1)})
	preview, err := buildPreview(nil, modelMissing, snapshot, false)
	require.NoError(t, err)
	assert.True(t, hasNotice(preview, "source_model_missing", modelName))
	assert.Empty(t, preview.Changes)

	groupUnregistered := previewSource([]sourceModel{{
		Name: modelName, QuotaType: 0, ModelRatio: decimal.NewFromInt(1), EnableGroups: []string{sourceGroup},
	}}, nil)
	preview, err = buildPreview(nil, groupUnregistered, snapshot, false)
	require.NoError(t, err)
	assert.True(t, hasNotice(preview, "source_group_registry_missing", ""))
	for _, change := range preview.Changes {
		assert.NotEqual(t, "source_group_remove", change.Kind)
	}
}

func TestLocalFingerprintChangesWhenSelfHostedChannelIsAdded(t *testing.T) {
	const (
		sourceGroup = "OpenLux-Source"
		localGroup  = "local-group"
		modelName   = "openlux-sync-fingerprint-model"
	)
	bound := previewChannel(41, "OpenLux/OpenLux-Source", localGroup, modelName, "https://api.openlux.ai")
	snapshot := previewSnapshot(t, sourceGroup, localGroup, modelName, 1, []model.Channel{bound}, []model.Ability{
		{ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true},
	})
	before, err := localFingerprint(snapshot)
	require.NoError(t, err)

	selfHosted := previewChannel(42, "self-hosted-pool", localGroup, modelName, "https://pool.internal.example")
	snapshot.Channels = append(snapshot.Channels, selfHosted)
	snapshot.Abilities = append(snapshot.Abilities, model.Ability{
		ChannelId: selfHosted.Id, Group: localGroup, Model: modelName, Enabled: true,
	})
	after, err := localFingerprint(snapshot)
	require.NoError(t, err)
	assert.NotEqual(t, before, after)
}

func TestLocalFingerprintChangesWhenChannelConfigurationChanges(t *testing.T) {
	const (
		sourceGroup = "OpenLux-Source"
		localGroup  = "local-group"
		modelName   = "openlux-sync-fingerprint-config-model"
	)
	bound := previewChannel(51, "OpenLux/OpenLux-Source", localGroup, modelName, "https://api.openlux.ai")
	snapshot := previewSnapshot(t, sourceGroup, localGroup, modelName, 1, []model.Channel{bound}, []model.Ability{
		{ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true},
	})
	before, err := localFingerprint(snapshot)
	require.NoError(t, err)

	snapshot.Channels[0].OtherSettings = `{"api_version":"v2"}`
	after, err := localFingerprint(snapshot)
	require.NoError(t, err)
	assert.NotEqual(t, before, after)
}
