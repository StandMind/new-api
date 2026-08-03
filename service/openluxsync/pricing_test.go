package openluxsync

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decimalPointer(value string) *decimal.Decimal {
	parsed := decimal.RequireFromString(value)
	return &parsed
}

func rawNumber(value string) json.RawMessage {
	return json.RawMessage(value)
}

func emptyLocalPricingOptions() *localPricingOptions {
	return &localPricingOptions{
		modelRatio:           make(numberMap),
		modelPrice:           make(numberMap),
		completionRatio:      make(numberMap),
		cacheRatio:           make(numberMap),
		createCacheRatio:     make(numberMap),
		imageRatio:           make(numberMap),
		audioRatio:           make(numberMap),
		audioCompletionRatio: make(numberMap),
		routeGroupRatio:      make(map[string]decimal.Decimal),
		groupModelRatio:      make(nestedNumberMap),
		billingMode:          make(stringMap),
		billingExpr:          make(stringMap),
	}
}

func TestTargetGroupModelRatioUsesOpenLuxMarkupWithoutChangingLocalBase(t *testing.T) {
	tests := []struct {
		name        string
		model       sourceModel
		sourceRatio string
		localBase   string
		want        string
	}{
		{
			name:        "token model",
			model:       sourceModel{QuotaType: 0, ModelRatio: decimal.RequireFromString("2.5")},
			sourceRatio: "0.4",
			localBase:   "0.625",
			want:        "2.08",
		},
		{
			name:        "fixed price model",
			model:       sourceModel{QuotaType: 1, ModelPrice: decimal.RequireFromString("0.02")},
			sourceRatio: "0.5",
			localBase:   "0.01",
			want:        "1.3",
		},
		{
			name:        "fifteen decimal places",
			model:       sourceModel{QuotaType: 0, ModelRatio: decimal.NewFromInt(1)},
			sourceRatio: "1",
			localBase:   "3",
			want:        "0.433333333333333",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := targetGroupModelRatio(
				test.model,
				decimal.RequireFromString(test.sourceRatio),
				decimal.RequireFromString(test.localBase),
			)
			require.NoError(t, err)
			assert.Equal(t, test.want, got.String())
			assert.Equal(t, test.want, got.Round(15).String())
		})
	}
}

func TestEffectiveGroupModelRatioUsesRuntimeWildcardPrecedence(t *testing.T) {
	options := emptyLocalPricingOptions()
	options.routeGroupRatio["local"] = decimal.RequireFromString("0.2")
	options.groupModelRatio["local"] = numberMap{
		"*":      rawNumber("0.4"),
		"gpt-*":  rawNumber("0.6"),
		"gpt-4o": rawNumber("0.8"),
	}

	assert.Equal(t, "0.8", options.effectiveGroupModelRatio("local", "gpt-4o").String())
	assert.Equal(t, "0.6", options.effectiveGroupModelRatio("local", "gpt-4.1").String())
	assert.Equal(t, "0.4", options.effectiveGroupModelRatio("local", "claude").String())
	assert.Equal(t, "1", options.effectiveGroupModelRatio("missing", "gpt-4o").String())
}

func TestModelBillingChangeIncludesSupportedOpenLuxFields(t *testing.T) {
	const modelName = "openlux-sync-all-fields"
	options := emptyLocalPricingOptions()
	options.modelRatio[modelName] = rawNumber("1")
	item := sourceModel{
		Name:                 modelName,
		QuotaType:            0,
		ModelRatio:           decimal.NewFromInt(1),
		CompletionRatio:      decimalPointer("3"),
		CacheRatio:           decimalPointer("0.2"),
		CacheCreation5mRatio: decimalPointer("0.5"),
		CacheCreation1hRatio: decimalPointer("0.8"),
		ImageRatio:           decimalPointer("2"),
		AudioRatio:           decimalPointer("3"),
		AudioCompletionRatio: decimalPointer("4"),
	}

	result := buildModelBillingChange(item, options, []string{"other-group"})
	require.NotNil(t, result.change)
	assert.True(t, result.change.Actionable)
	assert.Empty(t, result.blocked)
	assert.Equal(t, []string{"other-group"}, result.change.AffectedGroups)

	details := make(map[string]string, len(result.change.Details))
	for _, detail := range result.change.Details {
		details[detail.Field] = detail.Target
	}
	assert.Equal(t, map[string]string{
		"audio_completion_ratio":  "4",
		"audio_ratio":             "3",
		"cache_creation_5m_ratio": "0.5",
		"cache_ratio":             "0.2",
		"completion_ratio":        "3",
		"image_ratio":             "2",
	}, details)
}

func TestModelBillingChangePreservesExplicitZeroRatios(t *testing.T) {
	const modelName = "openlux-sync-zero-fields"
	options := emptyLocalPricingOptions()
	options.modelRatio[modelName] = rawNumber("1")
	item := sourceModel{
		Name:                 modelName,
		QuotaType:            0,
		ModelRatio:           decimal.NewFromInt(1),
		CacheRatio:           decimalPointer("0"),
		CacheCreation5mRatio: decimalPointer("0"),
		ImageRatio:           decimalPointer("0"),
		AudioRatio:           decimalPointer("0"),
		AudioCompletionRatio: decimalPointer("0"),
	}

	result := buildModelBillingChange(item, options, nil)
	require.NotNil(t, result.change)
	assert.True(t, result.change.Actionable)
	targets := make(map[string]string, len(result.change.Details))
	for _, detail := range result.change.Details {
		targets[detail.Field] = detail.Target
	}
	assert.Equal(t, "0", targets["cache_ratio"])
	assert.Equal(t, "0", targets["cache_creation_5m_ratio"])
	assert.Equal(t, "0", targets["image_ratio"])
	assert.Equal(t, "0", targets["audio_ratio"])
	assert.Equal(t, "0", targets["audio_completion_ratio"])
}

func TestTierExpressionPreservesOpenLuxTiersAndRejectsThinkingRates(t *testing.T) {
	item := sourceModel{
		Name:                 "openlux-sync-tiered",
		QuotaType:            0,
		ModelRatio:           decimal.RequireFromString("2.5"),
		CompletionRatio:      decimalPointer("6"),
		CacheRatio:           decimalPointer("0.1"),
		CacheCreation5mRatio: decimalPointer("1.25"),
		CacheCreation1hRatio: decimalPointer("2"),
		StepRatios: []stepRatio{
			{
				StepSize:            decimal.NewFromInt(200000),
				CompletionStepSize:  -1,
				PromptStepRatio:     decimal.NewFromInt(1),
				CompletionStepRatio: decimal.NewFromInt(1),
				CacheStepRatio:      decimal.NewFromInt(1),
			},
			{
				StepSize:            decimal.NewFromInt(1000000),
				CompletionStepSize:  -1,
				PromptStepRatio:     decimal.NewFromInt(2),
				CompletionStepRatio: decimal.RequireFromString("1.5"),
				CacheStepRatio:      decimal.NewFromInt(2),
			},
		},
	}

	expression, err := buildTierExpression(item, decimal.RequireFromString("2.5"))
	require.NoError(t, err)
	standard, trace, err := billingexpr.RunExpr(expression, billingexpr.TokenParams{
		P: 1, C: 1, Len: 100000, CR: 1, CC: 1, CC1h: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, "tier_1", trace.MatchedTier)
	assert.InDelta(t, 51.75, standard, 0.000001)

	longContext, trace, err := billingexpr.RunExpr(expression, billingexpr.TokenParams{
		P: 1, C: 1, Len: 300000, CR: 1, CC: 1, CC1h: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, "tier_2", trace.MatchedTier)
	assert.InDelta(t, 88.5, longContext, 0.000001)

	item.StepRatios[0].PromptThinkingStepRatio = decimal.NewFromInt(1)
	_, err = buildTierExpression(item, decimal.RequireFromString("2.5"))
	assert.ErrorContains(t, err, "thinking")
}

func TestFixedPriceTierIsBlockedInsteadOfApproximated(t *testing.T) {
	const modelName = "openlux-sync-fixed-tier"
	options := emptyLocalPricingOptions()
	options.modelPrice[modelName] = rawNumber("0.01")
	item := sourceModel{
		Name:       modelName,
		QuotaType:  1,
		ModelPrice: decimal.RequireFromString("0.01"),
		StepRatios: []stepRatio{{
			StepSize:            decimal.NewFromInt(1000),
			CompletionStepSize:  -1,
			PromptStepRatio:     decimal.NewFromInt(1),
			CompletionStepRatio: decimal.NewFromInt(1),
		}},
	}

	result := buildModelBillingChange(item, options, nil)
	require.NotNil(t, result.change)
	assert.False(t, result.change.Actionable)
	assert.Contains(t, result.blocked, "固定价格模型的阶梯计费无法无损表达")
}
