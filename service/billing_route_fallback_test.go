package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFallbackBillingContext() *gin.Context {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	return context
}

func newFallbackBillingInfo(userID int) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		UserId:          userID,
		OriginModelName: "fallback-model",
		IsPlayground:    true,
		ForcePreConsume: true,
		UserSetting: dto.UserSetting{
			BillingPreference: "wallet_only",
		},
	}
}

func TestFreeToPaidFallbackCreatesBillingSession(t *testing.T) {
	truncate(t)
	seedUser(t, 8101, 1_000)
	info := newFallbackBillingInfo(8101)

	apiErr := PreConsumeBilling(newFallbackBillingContext(), 300, info)
	require.Nil(t, apiErr)
	require.NotNil(t, info.Billing)
	assert.Equal(t, 300, info.Billing.GetPreConsumedQuota())
	assert.Equal(t, 700, getUserQuota(t, 8101))
}

func TestFallbackReserveAndCheaperSettlementUseSuccessfulGroupQuota(t *testing.T) {
	truncate(t)
	seedUser(t, 8102, 1_000)
	context := newFallbackBillingContext()
	info := newFallbackBillingInfo(8102)

	require.Nil(t, PreConsumeBilling(context, 100, info))
	require.NoError(t, info.Billing.Reserve(300))
	assert.Equal(t, 300, info.Billing.GetPreConsumedQuota())
	assert.Equal(t, 700, getUserQuota(t, 8102))

	require.NoError(t, SettleBilling(context, info, 50))
	assert.Equal(t, 950, getUserQuota(t, 8102))
}

func TestFallbackReserveFailureDoesNotIncreaseReservation(t *testing.T) {
	truncate(t)
	seedUser(t, 8103, 250)
	info := newFallbackBillingInfo(8103)

	require.Nil(t, PreConsumeBilling(newFallbackBillingContext(), 100, info))
	reserveErr := info.Billing.Reserve(300)
	require.Error(t, reserveErr)
	assert.Equal(t, 100, info.Billing.GetPreConsumedQuota())
	assert.Equal(t, 150, getUserQuota(t, 8103))

	var user model.User
	require.NoError(t, model.DB.First(&user, 8103).Error)
	assert.Equal(t, 150, user.Quota)
}

func TestTrustedFirstGroupStillReservesPaidFallback(t *testing.T) {
	truncate(t)
	trustQuota := common.GetTrustQuota()
	userQuota := trustQuota + 100
	seedUser(t, 8104, userQuota)

	context := newFallbackBillingContext()
	context.Set("token_quota", userQuota)
	info := newFallbackBillingInfo(8104)
	info.ForcePreConsume = false

	require.Nil(t, PreConsumeBilling(context, 100, info))
	require.NotNil(t, info.Billing)
	assert.Equal(t, 0, info.Billing.GetPreConsumedQuota())
	assert.Equal(t, userQuota, getUserQuota(t, 8104))

	reserveErr := info.Billing.Reserve(userQuota + 1)
	require.Error(t, reserveErr)
	assert.Equal(t, 0, info.Billing.GetPreConsumedQuota())
	assert.Equal(t, userQuota, getUserQuota(t, 8104))
}
