package billing_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestPricePoliciesCoverOnlyGeminiResolutionModels(t *testing.T) {
	want := map[string][]RequestPriceTier{
		"gemini-3-pro-image": {
			{Value: "1K", Multiplier: 1},
			{Value: "2K", Multiplier: 1},
			{Value: "4K", Multiplier: 1.79},
		},
		"gemini-3-pro-image-preview": {
			{Value: "1K", Multiplier: 1},
			{Value: "2K", Multiplier: 1},
			{Value: "4K", Multiplier: 1.79},
		},
		"gemini-3.1-flash-image": {
			{Value: "512", Multiplier: 0.66696},
			{Value: "1K", Multiplier: 1},
			{Value: "2K", Multiplier: 1},
			{Value: "4K", Multiplier: 1.78571},
		},
		"gemini-3.1-flash-image-preview": {
			{Value: "512", Multiplier: 0.66696},
			{Value: "1K", Multiplier: 1},
			{Value: "2K", Multiplier: 1},
			{Value: "4K", Multiplier: 1.78571},
		},
	}

	require.Len(t, requestPricePolicies, len(want))
	for model, wantTiers := range want {
		policy, ok := GetRequestPricePolicy(model)
		require.True(t, ok, model)
		require.Equal(t, RequestPriceDimensionImageResolution, policy.Dimension)
		require.Equal(t, "1K", policy.DefaultValue)
		require.Equal(t, wantTiers, policy.Tiers)
	}
	_, ok := GetRequestPricePolicy("gemini-2.5-flash-image")
	require.False(t, ok)
}

func TestGetRequestPricePolicyReturnsIndependentTierCopy(t *testing.T) {
	policy, ok := GetRequestPricePolicy("gemini-3-pro-image")
	require.True(t, ok)
	policy.Tiers[0].Multiplier = 99

	unchanged, ok := GetRequestPricePolicy("gemini-3-pro-image")
	require.True(t, ok)
	require.Equal(t, 1.0, unchanged.Tiers[0].Multiplier)
}
