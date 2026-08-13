package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func preserveCreemAmountSettingsForTest(t *testing.T) {
	t.Helper()
	originalQuotaPerUnit := common.QuotaPerUnit
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalAmountOptions := append([]int(nil), operation_setting.GetPaymentSetting().AmountOptions...)
	originalTestMode := setting.CreemTestMode
	originalProducts := setting.CreemProducts
	originalTestProducts := setting.CreemTestProducts
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
		operation_setting.GetPaymentSetting().AmountOptions = originalAmountOptions
		setting.CreemTestMode = originalTestMode
		setting.CreemProducts = originalProducts
		setting.CreemTestProducts = originalTestProducts
	})
}

func TestCreemQuotaForTopUpAmountUsesDisplayMode(t *testing.T) {
	preserveCreemAmountSettingsForTest(t)
	common.QuotaPerUnit = 500000

	tests := []struct {
		name        string
		displayType string
		expected    int64
	}{
		{name: "USD", displayType: operation_setting.QuotaDisplayTypeUSD, expected: 1500000},
		{name: "CNY", displayType: operation_setting.QuotaDisplayTypeCNY, expected: 1500000},
		{name: "custom", displayType: operation_setting.QuotaDisplayTypeCustom, expected: 1500000},
		{name: "tokens", displayType: operation_setting.QuotaDisplayTypeTokens, expected: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation_setting.GetGeneralSetting().QuotaDisplayType = test.displayType
			quota, err := creemQuotaForTopUpAmount(3)
			require.NoError(t, err)
			assert.Equal(t, test.expected, quota)
		})
	}
}

func TestResolveCreemProductSupportsAmountAndLegacyProductID(t *testing.T) {
	preserveCreemAmountSettingsForTest(t)
	common.QuotaPerUnit = 100
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountOptions = []int{10, 20}
	products := []CreemProduct{
		{ProductId: "prod_10", Name: "Ten", Price: 9, Currency: "USD", Quota: 1000},
		{ProductId: "prod_20", Name: "Twenty", Price: 18, Currency: "USD", Quota: 2000},
	}

	amount := int64(10)
	selected, err := resolveCreemProduct(&CreemPayRequest{Amount: &amount}, products)
	require.NoError(t, err)
	assert.Equal(t, "prod_10", selected.ProductId)

	selected, err = resolveCreemProduct(&CreemPayRequest{ProductId: "prod_20"}, products)
	require.NoError(t, err)
	assert.Equal(t, "prod_20", selected.ProductId)

	selected, err = resolveCreemProduct(&CreemPayRequest{Amount: &amount, ProductId: "prod_10"}, products)
	require.NoError(t, err)
	assert.Equal(t, "prod_10", selected.ProductId)
}

func TestResolveCreemProductRejectsInvalidAmountMappings(t *testing.T) {
	preserveCreemAmountSettingsForTest(t)
	common.QuotaPerUnit = 100
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountOptions = []int{10, 20}

	tests := []struct {
		name     string
		request  CreemPayRequest
		products []CreemProduct
		expected error
	}{
		{
			name:     "amount is not configured",
			request:  CreemPayRequest{Amount: int64Pointer(15)},
			products: []CreemProduct{{ProductId: "prod_15", Quota: 1500}},
			expected: errCreemAmountNotConfigured,
		},
		{
			name:     "amount has no product",
			request:  CreemPayRequest{Amount: int64Pointer(10)},
			products: []CreemProduct{{ProductId: "prod_20", Quota: 2000}},
			expected: errCreemProductNotFound,
		},
		{
			name:    "amount has duplicate products",
			request: CreemPayRequest{Amount: int64Pointer(10)},
			products: []CreemProduct{
				{ProductId: "prod_10_a", Quota: 1000},
				{ProductId: "prod_10_b", Quota: 1000},
			},
			expected: errCreemProductAmbiguous,
		},
		{
			name:     "amount conflicts with product id",
			request:  CreemPayRequest{Amount: int64Pointer(10), ProductId: "prod_20"},
			products: []CreemProduct{{ProductId: "prod_10", Quota: 1000}, {ProductId: "prod_20", Quota: 2000}},
			expected: errCreemRequestConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selected, err := resolveCreemProduct(&test.request, test.products)
			assert.Nil(t, selected)
			assert.ErrorIs(t, err, test.expected)
		})
	}
}

func TestCreemProductsForTopUpInfoAddsDerivedAmounts(t *testing.T) {
	preserveCreemAmountSettingsForTest(t)
	common.QuotaPerUnit = 100
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	operation_setting.GetPaymentSetting().AmountOptions = []int{0, 10, 10, 20, -1}
	setting.CreemTestMode = false
	setting.CreemProducts = `[
		{"productId":"prod_10","name":"Ten","price":9,"currency":"USD","quota":1000,"topupAmount":999},
		{"productId":"prod_20","name":"Twenty","price":18,"currency":"USD","quota":2000},
		{"productId":"prod_other","name":"Other","price":30,"currency":"USD","quota":3000}
	]`

	encoded, err := creemProductsForTopUpInfo()
	require.NoError(t, err)
	var products []CreemProduct
	require.NoError(t, common.Unmarshal([]byte(encoded), &products))
	require.Len(t, products, 3)
	assert.Equal(t, int64(10), products[0].TopUpAmount)
	assert.Equal(t, int64(20), products[1].TopUpAmount)
	assert.Zero(t, products[2].TopUpAmount)
	assert.Equal(t, []int{10, 20}, normalizeTopUpAmountOptions(operation_setting.GetPaymentSetting().AmountOptions))
}

func int64Pointer(value int64) *int64 {
	return &value
}
