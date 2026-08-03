package controller

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/smart_routing_setting"

	"github.com/gin-gonic/gin"
)

func buildMaskedTokenResponse(token *model.Token) *model.Token {
	if token == nil {
		return nil
	}
	maskedToken := *token
	maskedToken.Key = token.GetMaskedKey()
	maskedToken.GroupName = model.GetRouteGroupDisplayName(maskedToken.Group)
	maskedToken.GroupChainNames = make([]string, 0, len(maskedToken.GroupChain))
	for _, group := range maskedToken.GroupChain {
		maskedToken.GroupChainNames = append(maskedToken.GroupChainNames, model.GetRouteGroupDisplayName(group))
	}
	return &maskedToken
}

func buildMaskedTokenResponses(tokens []*model.Token) []*model.Token {
	maskedTokens := make([]*model.Token, 0, len(tokens))
	for _, token := range tokens {
		maskedTokens = append(maskedTokens, buildMaskedTokenResponse(token))
	}
	return maskedTokens
}

type tokenRequest struct {
	Id                 int                `json:"id"`
	Status             int                `json:"status"`
	Name               string             `json:"name"`
	ExpiredTime        int64              `json:"expired_time"`
	RemainQuota        int                `json:"remain_quota"`
	UnlimitedQuota     bool               `json:"unlimited_quota"`
	ModelLimitsEnabled bool               `json:"model_limits_enabled"`
	ModelLimits        string             `json:"model_limits"`
	AllowIps           *string            `json:"allow_ips"`
	Group              string             `json:"group"`
	GroupChain         *model.StringArray `json:"group_chain"`
	RoutingPriority    *string            `json:"routing_priority"`
}

func normalizeTokenGroupChain(c *gin.Context, request *tokenRequest) error {
	if request.GroupChain == nil || len(*request.GroupChain) == 0 {
		return errors.New("group chain must contain at least one group")
	}
	rawChain := append(model.StringArray(nil), (*request.GroupChain)...)

	usableGroups := service.GetUserUsableGroups(common.GetContextKeyString(c, constant.ContextKeyUserGroup))
	seen := make(map[string]struct{}, len(rawChain))
	normalized := make(model.StringArray, 0, len(rawChain))
	for _, rawGroup := range rawChain {
		group := strings.TrimSpace(rawGroup)
		if group == "" {
			return errors.New("group chain cannot contain an empty group")
		}
		if group == "auto" {
			return errors.New("group chain cannot contain auto")
		}
		if _, exists := seen[group]; exists {
			return fmt.Errorf("group %s is duplicated in group chain", group)
		}
		if _, allowed := usableGroups[group]; !allowed {
			return fmt.Errorf("no access to group %s", group)
		}
		if routeGroup, exists := model.GetRouteGroupFromSnapshot(group); !exists || !routeGroup.Enabled {
			return fmt.Errorf("group %s is deprecated", group)
		}
		seen[group] = struct{}{}
		normalized = append(normalized, group)
	}
	request.GroupChain = &normalized
	request.Group = normalized[0]
	return nil
}

func defaultCompatibilityGroupChain(userGroup string) model.StringArray {
	usableGroups := service.GetUserUsableGroups(userGroup)
	groups := make([]string, 0, len(usableGroups))
	for group := range usableGroups {
		if routeGroup, exists := model.GetRouteGroupFromSnapshot(group); !exists || !routeGroup.Enabled {
			continue
		}
		if len(model.GetEnabledModelsForGroups([]string{group})) == 0 {
			continue
		}
		groups = append(groups, group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		left, _ := model.ResolveAccessPolicyRatio(userGroup, groups[i], "")
		right, _ := model.ResolveAccessPolicyRatio(userGroup, groups[j], "")
		if left != right {
			return left < right
		}
		return groups[i] < groups[j]
	})
	return model.StringArray(groups)
}

func normalizeTokenRouting(c *gin.Context, request *tokenRequest, existing *model.Token) (constant.RoutingPriority, error) {
	priority := smart_routing_setting.GetDefaultPriority()
	if existing != nil {
		priority = constant.RoutingPriority(existing.RoutingPriority)
	}
	if request.RoutingPriority != nil {
		priority = constant.RoutingPriority(*request.RoutingPriority)
	}
	if !constant.IsValidRoutingPriority(priority, true) {
		return constant.RoutingPriorityManual, fmt.Errorf("invalid routing_priority %q", priority)
	}
	normalized := string(priority)
	request.RoutingPriority = &normalized

	if priority == constant.RoutingPriorityManual {
		if err := normalizeTokenGroupChain(c, request); err != nil {
			return priority, err
		}
		return priority, nil
	}
	if existing != nil && request.GroupChain != nil && len(*request.GroupChain) > 0 {
		if err := normalizeTokenGroupChain(c, request); err != nil {
			return priority, err
		}
		return priority, nil
	}
	if existing != nil && len(existing.GroupChain) > 0 {
		chain := append(model.StringArray(nil), existing.GroupChain...)
		request.GroupChain = &chain
		request.Group = existing.Group
		return priority, nil
	}

	chain := defaultCompatibilityGroupChain(common.GetContextKeyString(c, constant.ContextKeyUserGroup))
	if len(chain) == 0 && request.GroupChain != nil && len(*request.GroupChain) > 0 {
		if err := normalizeTokenGroupChain(c, request); err != nil {
			return priority, err
		}
		return priority, nil
	}
	if len(chain) == 0 {
		return priority, errors.New("no accessible group is available for the compatibility group chain")
	}
	request.GroupChain = &chain
	request.Group = chain[0]
	return priority, nil
}

func GetTokenRoutingConfig(c *gin.Context) {
	common.ApiSuccess(c, gin.H{
		"default_priority": smart_routing_setting.GetDefaultPriority(),
		"priorities":       constant.SmartRoutingPriorities,
	})
}

func GetAllTokens(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	tokens, err := model.GetAllUserTokens(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, _ := model.CountUserTokens(userId)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(buildMaskedTokenResponses(tokens))
	common.ApiSuccess(c, pageInfo)
}

func SearchTokens(c *gin.Context) {
	userId := c.GetInt("id")
	keyword := c.Query("keyword")
	token := c.Query("token")

	pageInfo := common.GetPageQuery(c)

	tokens, total, err := model.SearchUserTokens(userId, keyword, token, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(buildMaskedTokenResponses(tokens))
	common.ApiSuccess(c, pageInfo)
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildMaskedTokenResponse(token))
}

func GetTokenKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"key": token.GetFullKey(),
	})
}

func GetTokenStatus(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"object":          "credit_summary",
		"total_granted":   token.RemainQuota,
		"total_used":      0, // not supported currently
		"total_available": token.RemainQuota,
		"expires_at":      expiredAt * 1000,
	})
}

func GetTokenUsage(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "No Authorization header",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Invalid Bearer token",
		})
		return
	}
	tokenKey := parts[1]

	token, err := model.GetTokenByKey(strings.TrimPrefix(tokenKey, "sk-"), false)
	if err != nil {
		common.SysError("failed to get token by key: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgTokenGetInfoFailed)
		return
	}

	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    true,
		"message": "ok",
		"data": gin.H{
			"object":               "token_usage",
			"name":                 token.Name,
			"total_granted":        token.RemainQuota + token.UsedQuota,
			"total_used":           token.UsedQuota,
			"total_available":      token.RemainQuota,
			"unlimited_quota":      token.UnlimitedQuota,
			"model_limits":         token.GetModelLimitsMap(),
			"model_limits_enabled": token.ModelLimitsEnabled,
			"expires_at":           expiredAt,
		},
	})
}

func AddToken(c *gin.Context) {
	token := tokenRequest{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	priority, err := normalizeTokenRouting(c, &token, nil)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	// 非无限额度时，检查额度值是否超出有效范围
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	}
	// 检查用户令牌数量是否已达上限
	maxTokens := operation_setting.GetMaxUserTokens()
	count, err := model.CountUserTokens(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if int(count) >= maxTokens {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("已达到最大令牌数量限制 (%d)", maxTokens),
		})
		return
	}
	key, err := common.GenerateKey()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTokenGenerateFailed)
		common.SysLog("failed to generate token key: " + err.Error())
		return
	}
	cleanToken := model.Token{
		UserId:             c.GetInt("id"),
		Name:               token.Name,
		Key:                key,
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        token.ExpiredTime,
		RemainQuota:        token.RemainQuota,
		UnlimitedQuota:     token.UnlimitedQuota,
		ModelLimitsEnabled: token.ModelLimitsEnabled,
		ModelLimits:        token.ModelLimits,
		AllowIps:           token.AllowIps,
		Group:              token.Group,
		RoutingPriority:    string(priority),
	}
	cleanToken.GroupChain = append(model.StringArray(nil), (*token.GroupChain)...)
	err = cleanToken.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	err := model.DeleteTokenById(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func UpdateToken(c *gin.Context) {
	userId := c.GetInt("id")
	statusOnly := c.Query("status_only")
	token := tokenRequest{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanToken, err := model.GetTokenByIds(token.Id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	priority := constant.RoutingPriority(cleanToken.RoutingPriority)
	if statusOnly == "" {
		priority, err = normalizeTokenRouting(c, &token, cleanToken)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	}
	if token.Status == common.TokenStatusEnabled {
		if cleanToken.Status == common.TokenStatusExpired && cleanToken.ExpiredTime <= common.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			common.ApiErrorI18n(c, i18n.MsgTokenExpiredCannotEnable)
			return
		}
		if cleanToken.Status == common.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			common.ApiErrorI18n(c, i18n.MsgTokenExhaustedCannotEable)
			return
		}
	}
	if statusOnly != "" {
		cleanToken.Status = token.Status
	} else {
		// If you add more fields, please also update token.Update()
		cleanToken.Name = token.Name
		cleanToken.ExpiredTime = token.ExpiredTime
		cleanToken.RemainQuota = token.RemainQuota
		cleanToken.UnlimitedQuota = token.UnlimitedQuota
		cleanToken.ModelLimitsEnabled = token.ModelLimitsEnabled
		cleanToken.ModelLimits = token.ModelLimits
		cleanToken.AllowIps = token.AllowIps
		cleanToken.Group = token.Group
		cleanToken.GroupChain = append(model.StringArray(nil), (*token.GroupChain)...)
		cleanToken.RoutingPriority = string(priority)
	}
	err = cleanToken.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    buildMaskedTokenResponse(cleanToken),
	})
}

type TokenBatch struct {
	Ids []int `json:"ids"`
}

func DeleteTokenBatch(c *gin.Context) {
	tokenBatch := TokenBatch{}
	if err := c.ShouldBindJSON(&tokenBatch); err != nil || len(tokenBatch.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	userId := c.GetInt("id")
	count, err := model.BatchDeleteTokens(tokenBatch.Ids, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
}

func GetTokenKeysBatch(c *gin.Context) {
	tokenBatch := TokenBatch{}
	if err := c.ShouldBindJSON(&tokenBatch); err != nil || len(tokenBatch.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if len(tokenBatch.Ids) > 100 {
		common.ApiErrorI18n(c, i18n.MsgBatchTooMany, map[string]any{"Max": 100})
		return
	}
	userId := c.GetInt("id")
	tokens, err := model.GetTokenKeysByIds(tokenBatch.Ids, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	keysMap := make(map[int]string)
	for _, t := range tokens {
		keysMap[t.Id] = t.GetFullKey()
	}
	common.ApiSuccess(c, gin.H{"keys": keysMap})
}
