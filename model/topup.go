package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TopUp struct {
	Id              int     `json:"id"`
	UserId          int     `json:"user_id" gorm:"index;index:idx_topups_affiliate_eligible,priority:1"`
	Amount          int64   `json:"amount"`
	Money           float64 `json:"money"`
	TradeNo         string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string  `json:"payment_provider" gorm:"type:varchar(50);default:'';index:idx_topups_affiliate_eligible,priority:3"`
	CreateTime      int64   `json:"create_time"`
	CompleteTime    int64   `json:"complete_time"`
	Status          string  `json:"status" gorm:"type:varchar(32);index:idx_topups_affiliate_eligible,priority:2"`
}

const (
	PaymentMethodStripe       = "stripe"
	PaymentMethodCreem        = "creem"
	PaymentMethodWaffo        = "waffo"
	PaymentMethodWaffoPancake = "waffo_pancake"
	PaymentMethodBalance      = "balance"
)

const (
	PaymentProviderEpay         = "epay"
	PaymentProviderStripe       = "stripe"
	PaymentProviderCreem        = "creem"
	PaymentProviderWaffo        = "waffo"
	PaymentProviderWaffoPancake = "waffo_pancake"
	PaymentProviderBalance      = "balance"
)

var (
	ErrPaymentMethodMismatch = errors.New("payment method mismatch")
	ErrTopUpNotFound         = errors.New("topup not found")
	ErrTopUpStatusInvalid    = errors.New("topup status invalid")
)

var eligibleInvitationTopupProviders = []string{
	PaymentProviderEpay,
	PaymentProviderStripe,
	PaymentProviderCreem,
	PaymentProviderWaffo,
	PaymentProviderWaffoPancake,
}

type topUpCompletionParams struct {
	TradeNo             string
	ExpectedProvider    string
	ActualPaymentMethod string
	StripeCustomerId    string
	CustomerEmail       string
}

type topUpCompletionResult struct {
	TopUp          TopUp
	CreditedQuota  int
	InviterId      int
	RewardQuota    int
	TopupOrdinal   int
	CustomerUpdate bool
	Applied        bool
}

func isEligibleInvitationTopupProvider(provider string) bool {
	for _, eligible := range eligibleInvitationTopupProviders {
		if provider == eligible {
			return true
		}
	}
	return false
}

func calculateTopUpCreditedQuota(topUp *TopUp) (int, error) {
	if topUp == nil {
		return 0, errors.New("invalid topup")
	}
	if topUp.PaymentProvider == PaymentProviderCreem {
		return common.QuotaFromFloatStrict(float64(topUp.Amount))
	}

	amount := decimal.NewFromInt(topUp.Amount)
	if topUp.PaymentProvider == PaymentProviderStripe {
		amount = decimal.NewFromFloat(topUp.Money)
	}
	quota := amount.Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Truncate(0)
	return common.QuotaFromFloatStrict(quota.InexactFloat64())
}

func calculateInvitationReward(creditedQuota int, rebateBps int) (int, error) {
	reward := decimal.NewFromInt(int64(creditedQuota)).
		Mul(decimal.NewFromInt(int64(rebateBps))).
		Div(decimal.NewFromInt(10000)).
		Truncate(0)
	return common.QuotaFromFloatStrict(reward.InexactFloat64())
}

func completeWalletTopUp(params topUpCompletionParams) (topUpCompletionResult, error) {
	if params.TradeNo == "" {
		return topUpCompletionResult{}, errors.New("未提供支付单号")
	}

	var result topUpCompletionResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		topUp := TopUp{}
		if err := lockForUpdate(tx).Where(commonKeyForTopUpTradeNo()+" = ?", params.TradeNo).First(&topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if params.ExpectedProvider != "" && topUp.PaymentProvider != params.ExpectedProvider {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			result.TopUp = topUp
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		creditedQuota, err := calculateTopUpCreditedQuota(&topUp)
		if err != nil || creditedQuota <= 0 {
			return errors.New("无效的充值额度")
		}

		invitee := User{}
		if err := lockForUpdate(tx).Where("id = ?", topUp.UserId).First(&invitee).Error; err != nil {
			return err
		}

		result = topUpCompletionResult{
			TopUp:         topUp,
			CreditedQuota: creditedQuota,
			InviterId:     invitee.InviterId,
			Applied:       true,
		}

		inviteeUpdates := map[string]interface{}{
			"quota": gorm.Expr("quota + ?", creditedQuota),
		}
		if int64(invitee.Quota)+int64(creditedQuota) > int64(common.MaxQuota) {
			return errors.New("充值后余额超过存储上限")
		}
		if params.StripeCustomerId != "" {
			inviteeUpdates["stripe_customer"] = params.StripeCustomerId
		}
		if params.CustomerEmail != "" && invitee.Email == "" {
			inviteeUpdates["email"] = params.CustomerEmail
			result.CustomerUpdate = true
		}

		if invitee.InviterId > 0 && isEligibleInvitationTopupProvider(topUp.PaymentProvider) {
			setting, err := loadInvitationSetting(tx)
			if err != nil {
				return err
			}
			if setting.Mode == InvitationModeRebate {
				inviter := User{}
				inviterErr := lockForUpdate(tx).Where("id = ?", invitee.InviterId).First(&inviter).Error
				if inviterErr != nil && !errors.Is(inviterErr, gorm.ErrRecordNotFound) {
					return inviterErr
				}
				if inviterErr == nil {
					var priorTopupIds []int
					query := lockForUpdate(tx.Model(&TopUp{})).
						Select("id").
						Where("user_id = ? AND status = ? AND payment_provider IN ?", invitee.Id, common.TopUpStatusSuccess, eligibleInvitationTopupProviders).
						Order("id ASC").
						Limit(setting.RebateTopupCount)
					if err := query.Find(&priorTopupIds).Error; err != nil {
						return err
					}
					if len(priorTopupIds) < setting.RebateTopupCount {
						rewardQuota, err := calculateInvitationReward(creditedQuota, setting.RebateBps)
						if err != nil {
							return err
						}
						ordinal := len(priorTopupIds) + 1
						reward := InvitationTopupReward{
							TopUpId:       topUp.Id,
							InviterId:     inviter.Id,
							InviteeId:     invitee.Id,
							TopupOrdinal:  ordinal,
							CreditedQuota: creditedQuota,
							RebateBps:     setting.RebateBps,
							RewardQuota:   rewardQuota,
							CreatedAt:     common.GetTimestamp(),
						}
						if err := tx.Create(&reward).Error; err != nil {
							return err
						}
						if rewardQuota > 0 {
							if int64(inviter.Quota)+int64(rewardQuota) > int64(common.MaxQuota) ||
								int64(inviter.AffHistoryQuota)+int64(rewardQuota) > int64(common.MaxQuota) {
								return errors.New("邀请返利后余额超过存储上限")
							}
							if err := tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]interface{}{
								"quota":       gorm.Expr("quota + ?", rewardQuota),
								"aff_history": gorm.Expr("aff_history + ?", rewardQuota),
							}).Error; err != nil {
								return err
							}
						}
						result.RewardQuota = rewardQuota
						result.TopupOrdinal = ordinal
					}
				}
			}
		}

		if err := tx.Model(&User{}).Where("id = ?", invitee.Id).Updates(inviteeUpdates).Error; err != nil {
			return err
		}
		if params.ActualPaymentMethod != "" {
			topUp.PaymentMethod = params.ActualPaymentMethod
		}
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(&topUp).Error; err != nil {
			return err
		}
		result.TopUp = topUp
		return nil
	})
	if err != nil {
		return topUpCompletionResult{}, err
	}

	if result.Applied {
		gopool.Go(func() {
			if err := cacheIncrUserQuota(result.TopUp.UserId, int64(result.CreditedQuota)); err != nil {
				common.SysLog("failed to update topup user quota cache: " + err.Error())
			}
			if result.RewardQuota > 0 {
				if err := cacheIncrUserQuota(result.InviterId, int64(result.RewardQuota)); err != nil {
					common.SysLog("failed to update invitation reward quota cache: " + err.Error())
				}
			}
			if result.CustomerUpdate {
				if err := updateUserEmailCache(result.TopUp.UserId, params.CustomerEmail); err != nil {
					common.SysLog("failed to update topup user email cache: " + err.Error())
				}
			}
		})
	}
	return result, nil
}

func commonKeyForTopUpTradeNo() string {
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		return `"trade_no"`
	}
	return "`trade_no`"
}

func recordInvitationRewardLog(result topUpCompletionResult) {
	if !result.Applied || result.RewardQuota <= 0 {
		return
	}
	RecordLog(result.InviterId, LogTypeSystem, fmt.Sprintf(
		"邀请用户第 %d 次充值返利 %s",
		result.TopupOrdinal,
		logger.LogQuota(result.RewardQuota),
	))
}

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func UpdatePendingTopUpStatus(tradeNo string, expectedPaymentProvider string, targetStatus string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		topUp.Status = targetStatus
		return tx.Save(topUp).Error
	})
}

func Recharge(referenceId string, customerId string, callerIp string) (err error) {
	result, err := completeWalletTopUp(topUpCompletionParams{
		TradeNo:          referenceId,
		ExpectedProvider: PaymentProviderStripe,
		StripeCustomerId: customerId,
	})
	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if result.Applied {
		RecordTopupLog(result.TopUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%d", logger.FormatQuota(result.CreditedQuota), result.TopUp.Amount), callerIp, result.TopUp.PaymentMethod, PaymentMethodStripe)
		recordInvitationRewardLog(result)
	}
	return nil
}

// topUpQueryWindowSeconds 限制充值记录查询的时间窗口（秒）。
const topUpQueryWindowSeconds int64 = 30 * 24 * 60 * 60

// topUpQueryCutoff 返回允许查询的最早 create_time（秒级 Unix 时间戳）。
func topUpQueryCutoff() int64 {
	return common.GetTimestamp() - topUpQueryWindowSeconds
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cutoff := topUpQueryCutoff()

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, cutoff).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Where("user_id = ? AND create_time >= ?", userId, cutoff).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用，不限制时间窗口）
func GetAllTopUps(pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err = tx.Model(&TopUp{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// searchTopUpCountHardLimit 搜索充值记录时 COUNT 的安全上限，
// 防止对超大表执行无界 COUNT 触发 DoS。
const searchTopUpCountHardLimit = 10000

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// SearchAllTopUps 按订单号搜索全平台充值记录（管理员使用，不限制时间窗口）
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{})
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值
func ManualCompleteTopUp(tradeNo string, callerIp string) error {
	result, err := completeWalletTopUp(topUpCompletionParams{TradeNo: tradeNo})
	if err != nil {
		return err
	}
	if result.Applied {
		RecordTopupLog(result.TopUp.UserId, fmt.Sprintf("管理员补单成功，充值金额: %v，支付金额：%f", logger.FormatQuota(result.CreditedQuota), result.TopUp.Money), callerIp, result.TopUp.PaymentMethod, "admin")
		recordInvitationRewardLog(result)
	}
	return nil
}

func RechargeEpay(tradeNo string, actualPaymentMethod string, callerIp string) error {
	result, err := completeWalletTopUp(topUpCompletionParams{
		TradeNo:             tradeNo,
		ExpectedProvider:    PaymentProviderEpay,
		ActualPaymentMethod: actualPaymentMethod,
	})
	if err != nil {
		return err
	}
	if result.Applied {
		RecordTopupLog(result.TopUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(result.CreditedQuota), result.TopUp.Money), callerIp, result.TopUp.PaymentMethod, PaymentProviderEpay)
		recordInvitationRewardLog(result)
	}
	return nil
}

func RechargeCreem(referenceId string, customerEmail string, customerName string, callerIp string) (err error) {
	_ = customerName
	result, err := completeWalletTopUp(topUpCompletionParams{
		TradeNo:          referenceId,
		ExpectedProvider: PaymentProviderCreem,
		CustomerEmail:    customerEmail,
	})
	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if result.Applied {
		RecordTopupLog(result.TopUp.UserId, fmt.Sprintf("使用Creem充值成功，支付金额：%.2f", result.TopUp.Money), callerIp, result.TopUp.PaymentMethod, PaymentMethodCreem)
		recordInvitationRewardLog(result)
	}
	return nil
}

func RechargeWaffo(tradeNo string, callerIp string) (err error) {
	result, err := completeWalletTopUp(topUpCompletionParams{
		TradeNo:          tradeNo,
		ExpectedProvider: PaymentProviderWaffo,
	})
	if err != nil {
		common.SysError("waffo topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if result.Applied {
		RecordTopupLog(result.TopUp.UserId, fmt.Sprintf("Waffo充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(result.CreditedQuota), result.TopUp.Money), callerIp, result.TopUp.PaymentMethod, PaymentMethodWaffo)
		recordInvitationRewardLog(result)
	}
	return nil
}

func RechargeWaffoPancake(tradeNo string) (err error) {
	result, err := completeWalletTopUp(topUpCompletionParams{
		TradeNo:          tradeNo,
		ExpectedProvider: PaymentProviderWaffoPancake,
	})
	if err != nil {
		common.SysError("waffo pancake topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	if result.Applied {
		RecordLog(result.TopUp.UserId, LogTypeTopup, fmt.Sprintf("Waffo Pancake充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(result.CreditedQuota), result.TopUp.Money))
		recordInvitationRewardLog(result)
	}
	return nil
}
