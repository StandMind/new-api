package model

import (
	"strconv"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func preserveInvitationGlobalsForTest(t *testing.T) {
	t.Helper()
	previousInviterQuota := common.QuotaForInviter
	previousInviteeQuota := common.QuotaForInvitee
	common.OptionMapRWMutex.Lock()
	previousValues := make(map[string]string, len(invitationOptionKeys))
	previousExists := make(map[string]bool, len(invitationOptionKeys))
	for _, key := range invitationOptionKeys {
		previousValues[key], previousExists[key] = common.OptionMap[key]
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.QuotaForInviter = previousInviterQuota
		common.QuotaForInvitee = previousInviteeQuota
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		for _, key := range invitationOptionKeys {
			if previousExists[key] {
				common.OptionMap[key] = previousValues[key]
			} else {
				delete(common.OptionMap, key)
			}
		}
	})
}

func setInvitationSettingForTest(t *testing.T, setting InvitationSetting) {
	t.Helper()
	values := map[string]string{
		InvitationModeOptionKey:         setting.Mode,
		"QuotaForInviter":               strconv.Itoa(setting.FixedInviterQuota),
		"QuotaForInvitee":               strconv.Itoa(setting.FixedInviteeQuota),
		InvitationRebateBpsOptionKey:    strconv.Itoa(setting.RebateBps),
		InvitationRebateTopupsOptionKey: strconv.Itoa(setting.RebateTopupCount),
	}
	for key, value := range values {
		option := Option{Key: key}
		require.NoError(t, DB.Where(commonKeyCol+" = ?", key).Assign(Option{Value: value}).FirstOrCreate(&option).Error)
	}
}

func createInvitationUserForTest(t *testing.T, id int, username string, inviterId int) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:        id,
		Username:  username,
		AffCode:   "aff-" + strconv.Itoa(id),
		Status:    common.UserStatusEnabled,
		InviterId: inviterId,
	}).Error)
}

func createInvitationTopUpForTest(t *testing.T, userId int, tradeNo string, provider string, status string) TopUp {
	t.Helper()
	topUp := TopUp{
		UserId:          userId,
		Amount:          1,
		Money:           1,
		TradeNo:         tradeNo,
		PaymentMethod:   provider,
		PaymentProvider: provider,
		CreateTime:      common.GetTimestamp(),
		Status:          status,
	}
	if status == common.TopUpStatusSuccess {
		topUp.CompleteTime = common.GetTimestamp()
	}
	require.NoError(t, DB.Create(&topUp).Error)
	return topUp
}

func loadInvitationUserForTest(t *testing.T, id int) User {
	t.Helper()
	var user User
	require.NoError(t, DB.Where("id = ?", id).First(&user).Error)
	return user
}

func TestInvitationSettingValidationAndPersistence(t *testing.T) {
	truncateTables(t)
	preserveInvitationGlobalsForTest(t)

	setting := InvitationSetting{
		Mode:              InvitationModeRebate,
		FixedInviterQuota: 800,
		FixedInviteeQuota: 400,
		RebateBps:         500,
		RebateTopupCount:  3,
	}
	require.NoError(t, SaveInvitationSetting(setting))

	actual, err := GetInvitationSetting()
	require.NoError(t, err)
	assert.Equal(t, setting, actual)

	setting.Mode = InvitationModeFixed
	require.NoError(t, SaveInvitationSetting(setting))
	actual, err = GetInvitationSetting()
	require.NoError(t, err)
	assert.Equal(t, InvitationModeFixed, actual.Mode)
	assert.Equal(t, 500, actual.RebateBps)
	assert.Equal(t, 3, actual.RebateTopupCount)

	setting.Mode = InvitationModeRebate
	setting.RebateTopupCount = 0
	require.Error(t, SaveInvitationSetting(setting))
}

func TestInvitationRegistrationStoresRelationshipWithoutFixedReward(t *testing.T) {
	for _, mode := range []string{InvitationModeDisabled, InvitationModeRebate} {
		t.Run(mode, func(t *testing.T) {
			truncateTables(t)
			setInvitationSettingForTest(t, InvitationSetting{
				Mode:              mode,
				FixedInviterQuota: 600,
				FixedInviteeQuota: 300,
				RebateBps:         500,
				RebateTopupCount:  3,
			})

			createInvitationUserForTest(t, 1051, "inviter", 0)
			createInvitationUserForTest(t, 1052, "invitee", 1051)
			invitee := loadInvitationUserForTest(t, 1052)
			reward, err := applyInvitationRegistrationTx(DB, &invitee, 1051)
			require.NoError(t, err)

			assert.Zero(t, reward.InviterQuota)
			assert.Zero(t, reward.InviteeQuota)
			inviter := loadInvitationUserForTest(t, 1051)
			invitee = loadInvitationUserForTest(t, 1052)
			assert.Equal(t, 1, inviter.AffCount)
			assert.Zero(t, inviter.AffQuota)
			assert.Zero(t, inviter.AffHistoryQuota)
			assert.Zero(t, invitee.Quota)
			assert.Equal(t, 1051, invitee.InviterId)
		})
	}
}

func TestInvitationRegistrationUsesCurrentFixedMode(t *testing.T) {
	truncateTables(t)
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:              InvitationModeFixed,
		FixedInviterQuota: 600,
		FixedInviteeQuota: 300,
		RebateBps:         500,
		RebateTopupCount:  3,
	})
	paymentSetting := operation_setting.GetPaymentSetting()
	previousConfirmed := paymentSetting.ComplianceConfirmed
	previousVersion := paymentSetting.ComplianceTermsVersion
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() {
		paymentSetting.ComplianceConfirmed = previousConfirmed
		paymentSetting.ComplianceTermsVersion = previousVersion
	})

	createInvitationUserForTest(t, 1001, "inviter", 0)
	createInvitationUserForTest(t, 1002, "invitee", 1001)
	invitee := loadInvitationUserForTest(t, 1002)
	reward, err := applyInvitationRegistrationTx(DB, &invitee, 1001)
	require.NoError(t, err)

	assert.Equal(t, 600, reward.InviterQuota)
	assert.Equal(t, 300, reward.InviteeQuota)
	inviter := loadInvitationUserForTest(t, 1001)
	invitee = loadInvitationUserForTest(t, 1002)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, 600, inviter.AffQuota)
	assert.Equal(t, 600, inviter.AffHistoryQuota)
	assert.Equal(t, 300, invitee.Quota)
}

func TestInvitationRebateUsesLifetimeWalletTopupOrdinal(t *testing.T) {
	truncateTables(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	createInvitationUserForTest(t, 1101, "reward-owner", 0)
	createInvitationUserForTest(t, 1102, "paying-user", 1101)
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:              InvitationModeFixed,
		FixedInviterQuota: 0,
		FixedInviteeQuota: 0,
		RebateBps:         500,
		RebateTopupCount:  3,
	})

	for index := 1; index <= 2; index++ {
		topUp := createInvitationTopUpForTest(t, 1102, "fixed-"+strconv.Itoa(index), PaymentProviderEpay, common.TopUpStatusPending)
		result, err := completeWalletTopUp(topUpCompletionParams{TradeNo: topUp.TradeNo, ExpectedProvider: PaymentProviderEpay})
		require.NoError(t, err)
		assert.Zero(t, result.RewardQuota)
	}

	setInvitationSettingForTest(t, InvitationSetting{
		Mode:              InvitationModeRebate,
		FixedInviterQuota: 0,
		FixedInviteeQuota: 0,
		RebateBps:         500,
		RebateTopupCount:  3,
	})
	third := createInvitationTopUpForTest(t, 1102, "rebate-third", PaymentProviderEpay, common.TopUpStatusPending)
	result, err := completeWalletTopUp(topUpCompletionParams{TradeNo: third.TradeNo, ExpectedProvider: PaymentProviderEpay})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TopupOrdinal)
	assert.Equal(t, 50, result.RewardQuota)

	fourth := createInvitationTopUpForTest(t, 1102, "rebate-fourth", PaymentProviderEpay, common.TopUpStatusPending)
	result, err = completeWalletTopUp(topUpCompletionParams{TradeNo: fourth.TradeNo, ExpectedProvider: PaymentProviderEpay})
	require.NoError(t, err)
	assert.Zero(t, result.RewardQuota)

	inviter := loadInvitationUserForTest(t, 1101)
	invitee := loadInvitationUserForTest(t, 1102)
	assert.Equal(t, 50, inviter.Quota)
	assert.Equal(t, 50, inviter.AffHistoryQuota)
	assert.Equal(t, 4000, invitee.Quota)

	var rewards []InvitationTopupReward
	require.NoError(t, DB.Order("id").Find(&rewards).Error)
	require.Len(t, rewards, 1)
	assert.Equal(t, third.Id, rewards[0].TopUpId)
	assert.Equal(t, 3, rewards[0].TopupOrdinal)
	assert.Equal(t, 1000, rewards[0].CreditedQuota)

	result, err = completeWalletTopUp(topUpCompletionParams{TradeNo: third.TradeNo, ExpectedProvider: PaymentProviderEpay})
	require.NoError(t, err)
	assert.False(t, result.Applied)
	var rewardCount int64
	require.NoError(t, DB.Model(&InvitationTopupReward{}).Count(&rewardCount).Error)
	assert.Equal(t, int64(1), rewardCount)
}

func TestInvitationRebateExcludesNonWalletTopups(t *testing.T) {
	truncateTables(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	createInvitationUserForTest(t, 1201, "owner", 0)
	createInvitationUserForTest(t, 1202, "friend", 1201)
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:             InvitationModeRebate,
		RebateBps:        500,
		RebateTopupCount: 1,
	})

	createInvitationTopUpForTest(t, 1202, "subscription-record", "", common.TopUpStatusSuccess)
	wallet := createInvitationTopUpForTest(t, 1202, "wallet-record", PaymentProviderStripe, common.TopUpStatusPending)
	result, err := completeWalletTopUp(topUpCompletionParams{TradeNo: wallet.TradeNo, ExpectedProvider: PaymentProviderStripe})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TopupOrdinal)
	assert.Equal(t, 50, result.RewardQuota)
}

func TestInvitationRebateDoesNotBlockTopupWhenInviterWasDeleted(t *testing.T) {
	truncateTables(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	createInvitationUserForTest(t, 1252, "friend", 1251)
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:             InvitationModeRebate,
		RebateBps:        500,
		RebateTopupCount: 3,
	})

	topUp := createInvitationTopUpForTest(t, 1252, "deleted-inviter", PaymentProviderEpay, common.TopUpStatusPending)
	result, err := completeWalletTopUp(topUpCompletionParams{TradeNo: topUp.TradeNo, ExpectedProvider: PaymentProviderEpay})
	require.NoError(t, err)
	assert.True(t, result.Applied)
	assert.Zero(t, result.RewardQuota)
	assert.Equal(t, 1000, loadInvitationUserForTest(t, 1252).Quota)

	var rewardCount int64
	require.NoError(t, DB.Model(&InvitationTopupReward{}).Count(&rewardCount).Error)
	assert.Zero(t, rewardCount)
}

func TestInvitationRebateSupportsEveryWalletProvider(t *testing.T) {
	truncateTables(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:             InvitationModeRebate,
		RebateBps:        500,
		RebateTopupCount: 1,
	})

	for index, provider := range eligibleInvitationTopupProviders {
		inviterId := 1500 + index*2
		inviteeId := inviterId + 1
		createInvitationUserForTest(t, inviterId, "owner-"+provider, 0)
		createInvitationUserForTest(t, inviteeId, "friend-"+provider, inviterId)
		topUp := createInvitationTopUpForTest(t, inviteeId, "provider-"+provider, provider, common.TopUpStatusPending)
		if provider == PaymentProviderCreem {
			require.NoError(t, DB.Model(&topUp).Update("amount", 1000).Error)
		}

		result, err := completeWalletTopUp(topUpCompletionParams{
			TradeNo:          topUp.TradeNo,
			ExpectedProvider: provider,
		})
		require.NoError(t, err)
		assert.Equal(t, 1000, result.CreditedQuota)
		assert.Equal(t, 50, result.RewardQuota)
		assert.Equal(t, 1, result.TopupOrdinal)
	}
}

func TestInvitationRebateManualCompletionOnlyCountsEligibleExternalTopups(t *testing.T) {
	truncateTables(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })

	createInvitationUserForTest(t, 1601, "owner", 0)
	createInvitationUserForTest(t, 1602, "friend", 1601)
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:             InvitationModeRebate,
		RebateBps:        500,
		RebateTopupCount: 1,
	})

	ineligible := createInvitationTopUpForTest(t, 1602, "manual-balance", PaymentProviderBalance, common.TopUpStatusPending)
	require.NoError(t, ManualCompleteTopUp(ineligible.TradeNo, "127.0.0.1"))

	eligible := createInvitationTopUpForTest(t, 1602, "manual-external", PaymentProviderEpay, common.TopUpStatusPending)
	require.NoError(t, ManualCompleteTopUp(eligible.TradeNo, "127.0.0.1"))
	require.NoError(t, ManualCompleteTopUp(eligible.TradeNo, "127.0.0.1"))

	inviter := loadInvitationUserForTest(t, 1601)
	assert.Equal(t, 50, inviter.Quota)
	assert.Equal(t, 50, inviter.AffHistoryQuota)

	var rewards []InvitationTopupReward
	require.NoError(t, DB.Find(&rewards).Error)
	require.Len(t, rewards, 1)
	assert.Equal(t, eligible.Id, rewards[0].TopUpId)
	assert.Equal(t, 1, rewards[0].TopupOrdinal)
}

func TestInvitationRebateUsesCurrentRateWithoutChangingHistory(t *testing.T) {
	truncateTables(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })
	createInvitationUserForTest(t, 1701, "owner", 0)
	createInvitationUserForTest(t, 1702, "friend", 1701)
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:             InvitationModeRebate,
		RebateBps:        500,
		RebateTopupCount: 3,
	})

	first := createInvitationTopUpForTest(t, 1702, "rate-first", PaymentProviderEpay, common.TopUpStatusPending)
	firstResult, err := completeWalletTopUp(topUpCompletionParams{TradeNo: first.TradeNo})
	require.NoError(t, err)
	assert.Equal(t, 50, firstResult.RewardQuota)

	setInvitationSettingForTest(t, InvitationSetting{
		Mode:             InvitationModeRebate,
		RebateBps:        1000,
		RebateTopupCount: 2,
	})
	second := createInvitationTopUpForTest(t, 1702, "rate-second", PaymentProviderEpay, common.TopUpStatusPending)
	secondResult, err := completeWalletTopUp(topUpCompletionParams{TradeNo: second.TradeNo})
	require.NoError(t, err)
	assert.Equal(t, 100, secondResult.RewardQuota)

	var rewards []InvitationTopupReward
	require.NoError(t, DB.Order("id ASC").Find(&rewards).Error)
	require.Len(t, rewards, 2)
	assert.Equal(t, 500, rewards[0].RebateBps)
	assert.Equal(t, 50, rewards[0].RewardQuota)
	assert.Equal(t, 1000, rewards[1].RebateBps)
	assert.Equal(t, 100, rewards[1].RewardQuota)
}

func TestInvitationRebateConcurrentTopupsRespectLimit(t *testing.T) {
	truncateTables(t)
	previousQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1000
	t.Cleanup(func() { common.QuotaPerUnit = previousQuotaPerUnit })
	createInvitationUserForTest(t, 1801, "owner", 0)
	createInvitationUserForTest(t, 1802, "friend", 1801)
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:             InvitationModeRebate,
		RebateBps:        500,
		RebateTopupCount: 1,
	})
	first := createInvitationTopUpForTest(t, 1802, "concurrent-first", PaymentProviderEpay, common.TopUpStatusPending)
	second := createInvitationTopUpForTest(t, 1802, "concurrent-second", PaymentProviderStripe, common.TopUpStatusPending)

	start := make(chan struct{})
	results := make(chan topUpCompletionResult, 2)
	errors := make(chan error, 2)
	var wg sync.WaitGroup
	for _, tradeNo := range []string{first.TradeNo, second.TradeNo} {
		wg.Add(1)
		go func(value string) {
			defer wg.Done()
			<-start
			result, err := completeWalletTopUp(topUpCompletionParams{TradeNo: value})
			results <- result
			errors <- err
		}(tradeNo)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	totalReward := 0
	for result := range results {
		assert.True(t, result.Applied)
		totalReward += result.RewardQuota
	}
	assert.Equal(t, 50, totalReward)
	inviter := loadInvitationUserForTest(t, 1801)
	invitee := loadInvitationUserForTest(t, 1802)
	assert.Equal(t, 50, inviter.Quota)
	assert.Equal(t, 2000, invitee.Quota)
	var rewardCount int64
	require.NoError(t, DB.Model(&InvitationTopupReward{}).Count(&rewardCount).Error)
	assert.Equal(t, int64(1), rewardCount)
}

func TestUserInvitationRewardMasksInviteeName(t *testing.T) {
	truncateTables(t)
	createInvitationUserForTest(t, 1301, "owner", 0)
	createInvitationUserForTest(t, 1302, "Tys756", 1301)
	topUp := createInvitationTopUpForTest(t, 1302, "masked-reward", PaymentProviderCreem, common.TopUpStatusSuccess)
	require.NoError(t, DB.Create(&InvitationTopupReward{
		TopUpId:       topUp.Id,
		InviterId:     1301,
		InviteeId:     1302,
		TopupOrdinal:  1,
		CreditedQuota: 1000,
		RebateBps:     500,
		RewardQuota:   50,
		CreatedAt:     common.GetTimestamp(),
	}).Error)

	rewards, total, err := GetUserInvitationRewards(1301, 0, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rewards, 1)
	assert.Equal(t, "T***6", rewards[0].Invitee)
	assert.NotEqual(t, "Tys756", rewards[0].Invitee)
}

func TestInvitationRewardSchemaEnforcesTopupIdempotency(t *testing.T) {
	truncateTables(t)
	assert.True(t, DB.Migrator().HasIndex(&TopUp{}, "idx_topups_affiliate_eligible"))

	createInvitationUserForTest(t, 1351, "owner", 0)
	createInvitationUserForTest(t, 1352, "friend", 1351)
	topUp := createInvitationTopUpForTest(t, 1352, "unique-reward", PaymentProviderEpay, common.TopUpStatusSuccess)
	reward := InvitationTopupReward{
		TopUpId:       topUp.Id,
		InviterId:     1351,
		InviteeId:     1352,
		TopupOrdinal:  1,
		CreditedQuota: 1000,
		RebateBps:     500,
		RewardQuota:   50,
		CreatedAt:     common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(&reward).Error)
	reward.Id = 0
	require.Error(t, DB.Create(&reward).Error)
}

func TestApplyInvitationRegistrationTxRollsBackWithCaller(t *testing.T) {
	truncateTables(t)
	createInvitationUserForTest(t, 1401, "owner", 0)
	createInvitationUserForTest(t, 1402, "friend", 1401)
	setInvitationSettingForTest(t, InvitationSetting{
		Mode:             InvitationModeDisabled,
		RebateBps:        500,
		RebateTopupCount: 3,
	})

	errExpected := assert.AnError
	err := DB.Transaction(func(tx *gorm.DB) error {
		invitee := User{Id: 1402}
		_, err := applyInvitationRegistrationTx(tx, &invitee, 1401)
		require.NoError(t, err)
		return errExpected
	})
	require.ErrorIs(t, err, errExpected)
	assert.Zero(t, loadInvitationUserForTest(t, 1401).AffCount)
}
