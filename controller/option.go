package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/console_setting"
	emailtemplatesetting "github.com/QuantumNous/new-api/setting/emailtemplate"
	"github.com/QuantumNous/new-api/setting/official_price_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/request_detail_setting"
	"github.com/QuantumNous/new-api/setting/smart_routing_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

var completionRatioMetaOptionKeys = []string{
	"ModelPrice",
	"ModelRatio",
	"CompletionRatio",
	"CacheRatio",
	"CreateCacheRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
}

func isPaymentComplianceOptionKey(key string) bool {
	return strings.HasPrefix(key, "payment_setting.compliance_")
}

func isPositiveOptionValue(value string) bool {
	intValue, err := strconv.Atoi(strings.TrimSpace(value))
	if err == nil {
		return intValue > 0
	}
	floatValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && floatValue > 0
}

func isVisiblePublicKeyOption(key string) bool {
	switch key {
	case "WaffoPancakeWebhookPublicKey", "WaffoPancakeWebhookTestKey":
		return true
	default:
		return false
	}
}

func isVisibleRootPaymentSecretOption(key string) bool {
	switch key {
	case "CreemApiKey",
		"CreemWebhookSecret",
		"CreemTestApiKey",
		"CreemTestWebhookSecret":
		return true
	default:
		return false
	}
}

func collectModelNamesFromOptionValue(raw string, modelNames map[string]struct{}) {
	if strings.TrimSpace(raw) == "" {
		return
	}

	var parsed map[string]any
	if err := common.UnmarshalJsonStr(raw, &parsed); err != nil {
		return
	}

	for modelName := range parsed {
		modelNames[modelName] = struct{}{}
	}
}

func buildCompletionRatioMetaValue(optionValues map[string]string) string {
	modelNames := make(map[string]struct{})
	for _, key := range completionRatioMetaOptionKeys {
		collectModelNamesFromOptionValue(optionValues[key], modelNames)
	}

	meta := make(map[string]ratio_setting.CompletionRatioInfo, len(modelNames))
	for modelName := range modelNames {
		meta[modelName] = ratio_setting.GetCompletionRatioInfo(modelName)
	}

	jsonBytes, err := common.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

func GetOptions(c *gin.Context) {
	var options []*model.Option
	optionValues := make(map[string]string)
	common.OptionMapRWMutex.Lock()
	for k, v := range common.OptionMap {
		value := common.Interface2String(v)
		isSensitiveKey := strings.HasSuffix(k, "Token") ||
			strings.HasSuffix(k, "Secret") ||
			strings.HasSuffix(k, "Key") ||
			strings.HasSuffix(k, "secret") ||
			strings.HasSuffix(k, "api_key")
		if isSensitiveKey &&
			!isVisiblePublicKeyOption(k) &&
			!isVisibleRootPaymentSecretOption(k) {
			continue
		}
		options = append(options, &model.Option{
			Key:   k,
			Value: value,
		})
		for _, optionKey := range completionRatioMetaOptionKeys {
			if optionKey == k {
				optionValues[k] = value
				break
			}
		}
	}
	common.OptionMapRWMutex.Unlock()
	options = append(options, &model.Option{
		Key:   "CompletionRatioMeta",
		Value: buildCompletionRatioMetaValue(optionValues),
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
}

type OptionUpdateRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func UpdateOption(c *gin.Context) {
	var option OptionUpdateRequest
	err := common.DecodeJson(c.Request.Body, &option)
	if err != nil {
		common.ApiErrorI18nStatus(c, http.StatusBadRequest, i18n.MsgInvalidParams)
		return
	}
	switch option.Value.(type) {
	case bool:
		option.Value = common.Interface2String(option.Value.(bool))
	case float64:
		option.Value = common.Interface2String(option.Value.(float64))
	case int:
		option.Value = common.Interface2String(option.Value.(int))
	default:
		option.Value = fmt.Sprintf("%v", option.Value)
	}
	switch option.Key {
	case "QuotaForInviter", "QuotaForInvitee":
		if isPositiveOptionValue(option.Value.(string)) && !operation_setting.IsPaymentComplianceConfirmed() {
			common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
			return
		}
	default:
		if option.Key == "AutoGroups" || option.Key == "DefaultUseAutoGroup" || option.Key == "routing_setting.user_group_chain_enabled" {
			common.ApiErrorI18n(c, i18n.MsgSettingDeprecated)
			return
		}
		if isPaymentComplianceOptionKey(option.Key) {
			common.ApiErrorI18n(c, i18n.MsgSettingComplianceReadOnly)
			return
		}
		switch option.Key {
		case "GroupRatio", "GroupGroupRatio", "UserUsableGroups", "TopupGroupRatio", "ModelRequestRateLimitGroup", "group_ratio_setting.group_special_usable_group":
			common.ApiErrorI18n(c, i18n.MsgSettingAccessPolicyMigrated)
			return
		}
	}
	switch option.Key {
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GitHubClientId == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingGitHubOAuthConfigRequired)
			return
		}
	case "discord.enabled":
		if option.Value == "true" && system_setting.GetDiscordSettings().ClientId == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingDiscordOAuthConfigRequired)
			return
		}
	case "oidc.enabled":
		if option.Value == "true" && system_setting.GetOIDCSettings().ClientId == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingOIDCConfigRequired)
			return
		}
	case "LinuxDOOAuthEnabled":
		if option.Value == "true" && common.LinuxDOClientId == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingLinuxDOConfigRequired)
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(common.EmailDomainWhitelist) == 0 {
			common.ApiErrorI18n(c, i18n.MsgSettingEmailDomainRequired)
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && common.WeChatServerAddress == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingWeChatConfigRequired)
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && common.TurnstileSiteKey == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingTurnstileConfigRequired)

			return
		}
	case "TelegramOAuthEnabled":
		if option.Value == "true" && common.TelegramBotToken == "" {
			common.ApiErrorI18n(c, i18n.MsgSettingTelegramConfigRequired)
			return
		}
	case "theme.frontend":
		if option.Value != "default" && option.Value != "classic" {
			common.ApiErrorI18n(c, i18n.MsgSettingThemeInvalid)
			return
		}
	case smart_routing_setting.ConfigName + ".default_priority":
		value := option.Value.(string)
		if !smart_routing_setting.ValidateDefaultPriority(value) {
			common.ApiErrorI18n(c, i18n.MsgSettingRoutingPriorityInvalid)
			return
		}
	case "GroupRatio":
		err = ratio_setting.CheckGroupRatio(option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "GroupModelRatio":
		err = ratio_setting.CheckGroupModelRatio(option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case official_price_setting.OptionKey:
		err = official_price_setting.ValidateModelPricesJSON(option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case request_detail_setting.ConfigName + ".mode",
		request_detail_setting.ConfigName + ".retention_days",
		request_detail_setting.ConfigName + ".max_storage_mb":
		err = request_detail_setting.ValidateOption(option.Key, option.Value.(string))
		if err != nil {
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "ImageRatio":
		err = ratio_setting.UpdateImageRatioByJSONString(option.Value.(string))
		if err != nil {
			common.SysError(fmt.Sprintf("invalid ImageRatio setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(option.Value.(string))
		if err != nil {
			common.SysError(fmt.Sprintf("invalid AudioRatio setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(option.Value.(string))
		if err != nil {
			common.SysError(fmt.Sprintf("invalid AudioCompletionRatio setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(option.Value.(string))
		if err != nil {
			common.SysError(fmt.Sprintf("invalid CreateCacheRatio setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "ModelRequestRateLimitGroup":
		err = setting.CheckModelRequestRateLimitGroup(option.Value.(string))
		if err != nil {
			common.SysError(fmt.Sprintf("invalid ModelRequestRateLimitGroup setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "AutomaticDisableStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			common.SysError(fmt.Sprintf("invalid AutomaticDisableStatusCodes setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "AutomaticRetryStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			common.SysError(fmt.Sprintf("invalid AutomaticRetryStatusCodes setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "console_setting.api_info":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "ApiInfo")
		if err != nil {
			common.SysError(fmt.Sprintf("invalid console ApiInfo setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "console_setting.announcements":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "Announcements")
		if err != nil {
			common.SysError(fmt.Sprintf("invalid console Announcements setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "console_setting.faq":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "FAQ")
		if err != nil {
			common.SysError(fmt.Sprintf("invalid console FAQ setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case "console_setting.uptime_kuma_groups":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "UptimeKumaGroups")
		if err != nil {
			common.SysError(fmt.Sprintf("invalid console UptimeKumaGroups setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	case emailtemplatesetting.OptionKey:
		err = emailtemplatesetting.ValidateSettingsJSON(option.Value.(string))
		if err != nil {
			common.SysError(fmt.Sprintf("invalid email template setting: %v", err))
			common.ApiErrorI18n(c, i18n.MsgSettingValueInvalid)
			return
		}
	}
	err = model.UpdateOption(option.Key, option.Value.(string))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 出于安全考虑只记录被修改的配置项名称，不记录配置值（可能含密钥等敏感信息）。
	recordManageAudit(c, "option.update", map[string]interface{}{
		"key": option.Key,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func GetInvitationSetting(c *gin.Context) {
	setting, err := model.GetInvitationSetting()
	if err != nil {
		common.SysLog("failed to load invitation setting: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, setting)
}

func UpdateInvitationSetting(c *gin.Context) {
	var setting model.InvitationSetting
	if err := c.ShouldBindJSON(&setting); err != nil {
		common.ApiErrorI18nStatus(c, http.StatusBadRequest, i18n.MsgInvalidParams)
		return
	}
	rewardEnabled := setting.Mode == model.InvitationModeRebate ||
		(setting.Mode == model.InvitationModeFixed && (setting.FixedInviterQuota > 0 || setting.FixedInviteeQuota > 0))
	if rewardEnabled && !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
		return
	}
	if err := model.SaveInvitationSetting(setting); err != nil {
		common.SysLog("failed to save invitation setting: " + err.Error())
		common.ApiErrorI18nStatus(c, http.StatusBadRequest, i18n.MsgInvalidParams)
		return
	}
	saved, err := model.GetInvitationSetting()
	if err != nil {
		common.SysLog("failed to reload invitation setting: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	common.ApiSuccess(c, saved)
}
