package official_price_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateModelPricesJSON(t *testing.T) {
	valid := `{
		"tiered-model": {
			"unit": "usd_per_million_input_tokens",
			"source_model": "official-tiered-model",
			"source_url": "https://example.com/pricing",
			"verified_at": "2026-07-30",
			"tiers": [
				{"up_to_input_tokens": 200000, "price": 1.25},
				{"price": 2.5}
			]
		},
		"image-model": {
			"unit": "usd_per_request",
			"tiers": [{"price": 0.067}]
		}
	}`
	require.NoError(t, ValidateModelPricesJSON(valid))

	tests := map[string]string{
		"unsupported unit":       `{"model":{"unit":"credits","tiers":[{"price":1}]}}`,
		"non-positive price":     `{"model":{"unit":"usd_per_request","tiers":[{"price":0}]}}`,
		"missing final tier":     `{"model":{"unit":"usd_per_million_input_tokens","tiers":[{"up_to_input_tokens":200000,"price":1}]}}`,
		"unordered limits":       `{"model":{"unit":"usd_per_million_input_tokens","tiers":[{"up_to_input_tokens":200000,"price":1},{"up_to_input_tokens":100000,"price":2},{"price":3}]}}`,
		"multiple request tiers": `{"model":{"unit":"usd_per_request","tiers":[{"up_to_input_tokens":1,"price":1},{"price":2}]}}`,
		"invalid source URL":     `{"model":{"unit":"usd_per_request","source_url":"ftp://example.com","tiers":[{"price":1}]}}`,
		"invalid date":           `{"model":{"unit":"usd_per_request","verified_at":"2026-02-30","tiers":[{"price":1}]}}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, ValidateModelPricesJSON(raw))
		})
	}
}

func TestGetModelPriceReturnsDetachedTierSlice(t *testing.T) {
	original := officialPriceSetting.ModelPrices
	t.Cleanup(func() {
		officialPriceSetting.ModelPrices = original
	})
	officialPriceSetting.ModelPrices = map[string]ModelPrice{
		"model": {
			Unit:  UnitUSDPerRequest,
			Tiers: []PriceTier{{Price: 1}},
		},
	}

	price, ok := GetModelPrice("model")
	require.True(t, ok)
	price.Tiers[0].Price = 9

	actual, ok := GetModelPrice("model")
	require.True(t, ok)
	assert.Equal(t, 1.0, actual.Tiers[0].Price)
}
