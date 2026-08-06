package model

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"gorm.io/gorm"
)

const (
	InvitationModeDisabled = "disabled"
	InvitationModeFixed    = "fixed"
	InvitationModeRebate   = "rebate"

	InvitationModeOptionKey           = "invitation_setting.mode"
	InvitationRebateBpsOptionKey      = "invitation_setting.rebate_bps"
	InvitationRebateTopupsOptionKey   = "invitation_setting.rebate_topup_count"
	DefaultInvitationRebateBps        = 500
	DefaultInvitationRebateTopupCount = 3
	MaxInvitationRebateBps            = 10000
	MaxInvitationRebateTopupCount     = 100
)

var invitationOptionKeys = []string{
	InvitationModeOptionKey,
	"QuotaForInviter",
	"QuotaForInvitee",
	InvitationRebateBpsOptionKey,
	InvitationRebateTopupsOptionKey,
}

type InvitationSetting struct {
	Mode              string `json:"mode"`
	FixedInviterQuota int    `json:"fixed_inviter_quota"`
	FixedInviteeQuota int    `json:"fixed_invitee_quota"`
	RebateBps         int    `json:"rebate_bps"`
	RebateTopupCount  int    `json:"rebate_topup_count"`
}

type InvitationTopupReward struct {
	Id            int   `json:"id"`
	TopUpId       int   `json:"-" gorm:"column:top_up_id;uniqueIndex"`
	InviterId     int   `json:"-" gorm:"column:inviter_id;index:idx_invitation_rewards_inviter_created,priority:1"`
	InviteeId     int   `json:"-" gorm:"column:invitee_id;index"`
	TopupOrdinal  int   `json:"topup_ordinal" gorm:"column:topup_ordinal"`
	CreditedQuota int   `json:"credited_quota" gorm:"column:credited_quota"`
	RebateBps     int   `json:"rebate_bps" gorm:"column:rebate_bps"`
	RewardQuota   int   `json:"reward_quota" gorm:"column:reward_quota"`
	CreatedAt     int64 `json:"created_at" gorm:"column:created_at;index:idx_invitation_rewards_inviter_created,priority:2"`
}

type UserInvitationReward struct {
	Id            int    `json:"id"`
	Invitee       string `json:"invitee"`
	TopupOrdinal  int    `json:"topup_ordinal"`
	CreditedQuota int    `json:"credited_quota"`
	RebateBps     int    `json:"rebate_bps"`
	RewardQuota   int    `json:"reward_quota"`
	CreatedAt     int64  `json:"created_at"`
}

type invitationRegistrationReward struct {
	InviterId    int
	InviterQuota int
	InviteeQuota int
}

func applyInvitationRegistrationTx(tx *gorm.DB, user *User, inviterId int) (invitationRegistrationReward, error) {
	if inviterId <= 0 || user == nil {
		return invitationRegistrationReward{}, nil
	}

	inviter := User{}
	if err := lockForUpdate(tx).Where("id = ?", inviterId).First(&inviter).Error; err != nil {
		return invitationRegistrationReward{}, err
	}
	setting, err := loadInvitationSetting(tx)
	if err != nil {
		return invitationRegistrationReward{}, err
	}

	reward := invitationRegistrationReward{InviterId: inviterId}
	if inviter.AffCount >= common.MaxQuota {
		return invitationRegistrationReward{}, errors.New("invitation count exceeds quota storage limit")
	}
	inviterUpdates := map[string]interface{}{
		"aff_count": gorm.Expr("aff_count + 1"),
	}
	if setting.Mode == InvitationModeFixed && operation_setting.IsPaymentComplianceConfirmed() {
		reward.InviterQuota = setting.FixedInviterQuota
		reward.InviteeQuota = setting.FixedInviteeQuota
		if reward.InviterQuota > 0 {
			if int64(inviter.AffQuota)+int64(reward.InviterQuota) > int64(common.MaxQuota) ||
				int64(inviter.AffHistoryQuota)+int64(reward.InviterQuota) > int64(common.MaxQuota) {
				return invitationRegistrationReward{}, errors.New("fixed invitation reward exceeds quota storage limit")
			}
			inviterUpdates["aff_quota"] = gorm.Expr("aff_quota + ?", reward.InviterQuota)
			inviterUpdates["aff_history"] = gorm.Expr("aff_history + ?", reward.InviterQuota)
		}
		if reward.InviteeQuota > 0 {
			if int64(user.Quota)+int64(reward.InviteeQuota) > int64(common.MaxQuota) {
				return invitationRegistrationReward{}, errors.New("invitee reward exceeds quota storage limit")
			}
			if err := tx.Model(&User{}).Where("id = ?", user.Id).Update("quota", gorm.Expr("quota + ?", reward.InviteeQuota)).Error; err != nil {
				return invitationRegistrationReward{}, err
			}
			user.Quota += reward.InviteeQuota
		}
	}
	if err := tx.Model(&User{}).Where("id = ?", inviterId).Updates(inviterUpdates).Error; err != nil {
		return invitationRegistrationReward{}, err
	}
	return reward, nil
}

func defaultInvitationSetting() InvitationSetting {
	mode := InvitationModeDisabled
	if common.QuotaForInviter > 0 || common.QuotaForInvitee > 0 {
		mode = InvitationModeFixed
	}
	return InvitationSetting{
		Mode:              mode,
		FixedInviterQuota: common.QuotaForInviter,
		FixedInviteeQuota: common.QuotaForInvitee,
		RebateBps:         DefaultInvitationRebateBps,
		RebateTopupCount:  DefaultInvitationRebateTopupCount,
	}
}

func normalizeInvitationSetting(setting InvitationSetting) (InvitationSetting, error) {
	setting.Mode = strings.TrimSpace(setting.Mode)
	switch setting.Mode {
	case InvitationModeDisabled, InvitationModeFixed, InvitationModeRebate:
	default:
		return InvitationSetting{}, errors.New("invalid invitation mode")
	}
	if setting.FixedInviterQuota < 0 || setting.FixedInviteeQuota < 0 ||
		setting.FixedInviterQuota > common.MaxQuota || setting.FixedInviteeQuota > common.MaxQuota {
		return InvitationSetting{}, errors.New("fixed invitation quota is out of range")
	}
	if setting.RebateBps < 0 || setting.RebateBps > MaxInvitationRebateBps {
		return InvitationSetting{}, errors.New("invitation rebate bps is out of range")
	}
	if setting.RebateTopupCount < 0 || setting.RebateTopupCount > MaxInvitationRebateTopupCount {
		return InvitationSetting{}, errors.New("invitation rebate topup count is out of range")
	}
	if setting.Mode == InvitationModeRebate && (setting.RebateBps == 0 || setting.RebateTopupCount == 0) {
		return InvitationSetting{}, errors.New("enabled invitation rebate requires a positive rate and topup count")
	}
	return setting, nil
}

func loadInvitationSetting(tx *gorm.DB) (InvitationSetting, error) {
	setting := defaultInvitationSetting()
	var options []Option
	if err := tx.Where(commonKeyCol+" IN ?", invitationOptionKeys).Find(&options).Error; err != nil {
		return InvitationSetting{}, err
	}

	values := make(map[string]string, len(options))
	for _, option := range options {
		values[option.Key] = option.Value
	}
	if value, ok := values["QuotaForInviter"]; ok {
		setting.FixedInviterQuota, _ = strconv.Atoi(value)
	}
	if value, ok := values["QuotaForInvitee"]; ok {
		setting.FixedInviteeQuota, _ = strconv.Atoi(value)
	}
	if value, ok := values[InvitationRebateBpsOptionKey]; ok {
		setting.RebateBps, _ = strconv.Atoi(value)
	}
	if value, ok := values[InvitationRebateTopupsOptionKey]; ok {
		setting.RebateTopupCount, _ = strconv.Atoi(value)
	}
	if value, ok := values[InvitationModeOptionKey]; ok {
		setting.Mode = value
	} else if setting.FixedInviterQuota > 0 || setting.FixedInviteeQuota > 0 {
		setting.Mode = InvitationModeFixed
	} else {
		setting.Mode = InvitationModeDisabled
	}
	return normalizeInvitationSetting(setting)
}

func GetInvitationSetting() (InvitationSetting, error) {
	return loadInvitationSetting(DB)
}

func SaveInvitationSetting(setting InvitationSetting) error {
	setting, err := normalizeInvitationSetting(setting)
	if err != nil {
		return err
	}
	return UpdateOptionsBulk(map[string]string{
		InvitationModeOptionKey:         setting.Mode,
		"QuotaForInviter":               strconv.Itoa(setting.FixedInviterQuota),
		"QuotaForInvitee":               strconv.Itoa(setting.FixedInviteeQuota),
		InvitationRebateBpsOptionKey:    strconv.Itoa(setting.RebateBps),
		InvitationRebateTopupsOptionKey: strconv.Itoa(setting.RebateTopupCount),
	})
}

func GetUserInvitationRewards(inviterId int, startIdx int, pageSize int) ([]UserInvitationReward, int64, error) {
	if inviterId <= 0 {
		return nil, 0, errors.New("invalid inviter id")
	}

	var total int64
	if err := DB.Model(&InvitationTopupReward{}).Where("inviter_id = ?", inviterId).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	type rewardRow struct {
		Id            int
		InviteeName   string
		TopupOrdinal  int
		CreditedQuota int
		RebateBps     int
		RewardQuota   int
		CreatedAt     int64
	}
	var rows []rewardRow
	err := DB.Table("invitation_topup_rewards AS rewards").
		Select("rewards.id, users.username AS invitee_name, rewards.topup_ordinal, rewards.credited_quota, rewards.rebate_bps, rewards.reward_quota, rewards.created_at").
		Joins("LEFT JOIN users ON users.id = rewards.invitee_id").
		Where("rewards.inviter_id = ?", inviterId).
		Order("rewards.created_at DESC, rewards.id DESC").
		Limit(pageSize).
		Offset(startIdx).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	rewards := make([]UserInvitationReward, 0, len(rows))
	for _, row := range rows {
		rewards = append(rewards, UserInvitationReward{
			Id:            row.Id,
			Invitee:       maskInvitationUsername(row.InviteeName),
			TopupOrdinal:  row.TopupOrdinal,
			CreditedQuota: row.CreditedQuota,
			RebateBps:     row.RebateBps,
			RewardQuota:   row.RewardQuota,
			CreatedAt:     row.CreatedAt,
		})
	}
	return rewards, total, nil
}

func maskInvitationUsername(username string) string {
	runes := []rune(strings.TrimSpace(username))
	switch len(runes) {
	case 0:
		return "***"
	case 1:
		return "*"
	case 2:
		return string(runes[0]) + "*"
	default:
		return string(runes[0]) + "***" + string(runes[len(runes)-1])
	}
}
