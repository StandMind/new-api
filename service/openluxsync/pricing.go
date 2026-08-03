package openluxsync

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/shopspring/decimal"
)

var (
	markupDecimal        = decimal.RequireFromString(SaleMultiplier)
	baseInputUSDDecimal  = decimal.NewFromInt(2)
	cacheCreation1hScale = decimal.RequireFromString("1.6")
)

type localPricingOptions struct {
	modelRatio           numberMap
	modelPrice           numberMap
	completionRatio      numberMap
	cacheRatio           numberMap
	createCacheRatio     numberMap
	imageRatio           numberMap
	audioRatio           numberMap
	audioCompletionRatio numberMap
	routeGroupRatio      map[string]decimal.Decimal
	groupModelRatio      nestedNumberMap
	billingMode          stringMap
	billingExpr          stringMap
}

type modelBillingResult struct {
	change  *Change
	blocked []string
}

func parseLocalPricingOptions(snapshot *model.OpenLuxSyncSnapshot) (*localPricingOptions, error) {
	result := &localPricingOptions{}
	var err error
	if result.modelRatio, err = parseNumberMap(snapshot.Options[optionModelRatio], optionModelRatio); err != nil {
		return nil, err
	}
	if result.modelPrice, err = parseNumberMap(snapshot.Options[optionModelPrice], optionModelPrice); err != nil {
		return nil, err
	}
	if result.completionRatio, err = parseNumberMap(snapshot.Options[optionCompletionRatio], optionCompletionRatio); err != nil {
		return nil, err
	}
	if result.cacheRatio, err = parseNumberMap(snapshot.Options[optionCacheRatio], optionCacheRatio); err != nil {
		return nil, err
	}
	if result.createCacheRatio, err = parseNumberMap(snapshot.Options[optionCreateCacheRatio], optionCreateCacheRatio); err != nil {
		return nil, err
	}
	if result.imageRatio, err = parseNumberMap(snapshot.Options[optionImageRatio], optionImageRatio); err != nil {
		return nil, err
	}
	if result.audioRatio, err = parseNumberMap(snapshot.Options[optionAudioRatio], optionAudioRatio); err != nil {
		return nil, err
	}
	if result.audioCompletionRatio, err = parseNumberMap(snapshot.Options[optionAudioCompletionRatio], optionAudioCompletionRatio); err != nil {
		return nil, err
	}
	result.routeGroupRatio = make(map[string]decimal.Decimal, len(snapshot.RouteGroups))
	for _, group := range snapshot.RouteGroups {
		result.routeGroupRatio[group.Code] = decimal.NewFromFloat(float64(group.BaseRatio))
	}
	if result.groupModelRatio, err = parseNestedNumberMap(snapshot.Options[optionGroupModelRatio], optionGroupModelRatio); err != nil {
		return nil, err
	}
	if result.billingMode, err = parseStringMap(snapshot.Options[optionBillingMode], optionBillingMode); err != nil {
		return nil, err
	}
	if result.billingExpr, err = parseStringMap(snapshot.Options[optionBillingExpr], optionBillingExpr); err != nil {
		return nil, err
	}
	return result, nil
}

func (options *localPricingOptions) localBase(modelName string) (int, decimal.Decimal, bool) {
	matchingName := ratio_setting.FormatMatchingModelName(modelName)
	if value, ok := options.modelPrice[matchingName]; ok {
		parsed, valid := decimalFromRaw(value)
		return 1, parsed, valid
	}
	if value, ok := options.modelRatio[matchingName]; ok {
		parsed, valid := decimalFromRaw(value)
		return 0, parsed, valid
	}
	return -1, decimal.Zero, false
}

func (options *localPricingOptions) effectiveGroupModelRatio(localGroup, modelName string) decimal.Decimal {
	if modelRatios, ok := options.groupModelRatio[localGroup]; ok {
		if value, exists := modelRatios[modelName]; exists {
			if parsed, valid := decimalFromRaw(value); valid {
				return parsed
			}
		}
		longestPrefix := ""
		matched := decimal.Zero
		found := false
		for pattern, value := range modelRatios {
			if pattern != "*" && !strings.HasSuffix(pattern, "*") {
				continue
			}
			prefix := strings.TrimSuffix(pattern, "*")
			if !strings.HasPrefix(modelName, prefix) || (found && len(prefix) <= len(longestPrefix)) {
				continue
			}
			parsed, valid := decimalFromRaw(value)
			if !valid {
				continue
			}
			longestPrefix = prefix
			matched = parsed
			found = true
		}
		if found {
			return matched
		}
	}
	if ratio, ok := options.routeGroupRatio[localGroup]; ok {
		return ratio
	}
	return decimal.NewFromInt(1)
}

func buildModelBillingChange(item sourceModel, options *localPricingOptions, affectedGroups []string) modelBillingResult {
	quotaType, localBase, baseExists := options.localBase(item.Name)
	blocked := make([]string, 0)
	if !baseExists || !localBase.IsPositive() {
		blocked = append(blocked, "本地模型缺少正数基础价格")
	} else if quotaType != item.QuotaType {
		blocked = append(blocked, "OpenLux 与本地的 Token/固定价计费类型不一致")
	}
	upstreamBase := item.ModelRatio
	if item.QuotaType == 1 {
		upstreamBase = item.ModelPrice
	}
	if !upstreamBase.IsPositive() {
		blocked = append(blocked, "OpenLux 基础价格不是正数")
	}

	completion := decimalValue(item.CompletionRatio, decimal.NewFromInt(1))
	cache := decimalValue(item.CacheRatio, decimal.NewFromInt(1))
	createCache, createCache1h, createCacheConfigured, createCache1hConfigured, cacheErr := cacheCreationRatios(item)
	if cacheErr != nil {
		blocked = append(blocked, cacheErr.Error())
	}
	if !createCacheConfigured {
		createCache = decimal.RequireFromString("1.25")
	}
	if createCache1hConfigured && !createCache1h.Equal(createCache.Mul(cacheCreation1hScale)) && len(item.StepRatios) == 0 {
		blocked = append(blocked, "OpenLux 的 1 小时缓存写入倍率无法由当前普通计费模式无损表达")
	}
	image := decimalValue(item.ImageRatio, decimal.NewFromInt(1))
	audio := decimalValue(item.AudioRatio, decimal.NewFromInt(1))
	audioCompletion := decimalValue(item.AudioCompletionRatio, decimal.NewFromInt(1))

	details := make([]ChangeDetail, 0, 8)
	mutations := make([]optionMutation, 0, 8)
	appendNumberChange := func(optionKey, field string, values numberMap, fallback, target decimal.Decimal) {
		current := numberMapValue(values, item.Name, fallback)
		if current.Equal(target) {
			return
		}
		details = append(details, ChangeDetail{Field: field, Current: current.String(), Target: target.String()})
		mutations = append(mutations, optionMutation{Key: optionKey, Action: "set", Model: item.Name, Value: target.String()})
	}

	completionInfo := ratio_setting.GetCompletionRatioInfo(item.Name)
	if completionInfo.Locked && !decimal.NewFromFloat(completionInfo.Ratio).Equal(completion) {
		blocked = append(blocked, "本地输出倍率由代码锁定，无法覆盖为 OpenLux 值")
	} else {
		appendNumberChange(optionCompletionRatio, "completion_ratio", options.completionRatio, decimal.NewFromFloat(completionInfo.Ratio), completion)
	}
	appendNumberChange(optionCacheRatio, "cache_ratio", options.cacheRatio, decimal.NewFromInt(1), cache)
	appendNumberChange(optionCreateCacheRatio, "cache_creation_5m_ratio", options.createCacheRatio, decimal.RequireFromString("1.25"), createCache)
	appendNumberChange(optionImageRatio, "image_ratio", options.imageRatio, decimal.NewFromInt(1), image)
	appendNumberChange(optionAudioRatio, "audio_ratio", options.audioRatio, decimal.NewFromInt(1), audio)
	appendNumberChange(optionAudioCompletionRatio, "audio_completion_ratio", options.audioCompletionRatio, decimal.NewFromInt(1), audioCompletion)

	targetMode := billing_setting.BillingModeRatio
	targetExpression := ""
	if len(item.StepRatios) > 0 {
		if item.QuotaType != 0 {
			blocked = append(blocked, "固定价格模型的阶梯计费无法无损表达")
		} else if localBase.IsPositive() {
			expression, err := buildTierExpression(item, localBase)
			if err != nil {
				blocked = append(blocked, err.Error())
			} else {
				targetMode = billing_setting.BillingModeTieredExpr
				targetExpression = expression
			}
		}
	}
	currentMode := options.billingMode[item.Name]
	if currentMode == "" {
		currentMode = billing_setting.BillingModeRatio
	}
	if currentMode != targetMode {
		details = append(details, ChangeDetail{Field: "billing_mode", Current: currentMode, Target: targetMode})
		if targetMode == billing_setting.BillingModeRatio {
			mutations = append(mutations,
				optionMutation{Key: optionBillingMode, Action: "delete", Model: item.Name},
				optionMutation{Key: optionBillingExpr, Action: "delete", Model: item.Name},
			)
		} else {
			mutations = append(mutations, optionMutation{Key: optionBillingMode, Action: "set_string", Model: item.Name, Value: targetMode})
		}
	}
	if targetMode == billing_setting.BillingModeTieredExpr && options.billingExpr[item.Name] != targetExpression {
		details = append(details, ChangeDetail{Field: "billing_expr", Current: options.billingExpr[item.Name], Target: targetExpression})
		mutations = append(mutations, optionMutation{Key: optionBillingExpr, Action: "set_string", Model: item.Name, Value: targetExpression})
	}

	if len(details) == 0 && len(blocked) == 0 {
		return modelBillingResult{}
	}
	sort.Slice(details, func(i, j int) bool { return details[i].Field < details[j].Field })
	change := &Change{
		Kind:           "model_billing_update",
		Model:          item.Name,
		Actionable:     len(blocked) == 0 && len(details) > 0,
		BlockedReasons: blocked,
		AffectedGroups: affectedGroups,
		Details:        details,
		mutations:      mutations,
	}
	change.ID = makeChangeID(*change)
	return modelBillingResult{change: change, blocked: blocked}
}

func targetGroupModelRatio(item sourceModel, sourceRatio, localBase decimal.Decimal) (decimal.Decimal, error) {
	upstreamBase := item.ModelRatio
	if item.QuotaType == 1 {
		upstreamBase = item.ModelPrice
	}
	if !upstreamBase.IsPositive() || !sourceRatio.IsPositive() || !localBase.IsPositive() {
		return decimal.Zero, fmt.Errorf("基础价格或来源组倍率不是正数")
	}
	return upstreamBase.Mul(sourceRatio).Mul(markupDecimal).Div(localBase).Round(15), nil
}

func buildTierExpression(item sourceModel, localModelRatio decimal.Decimal) (string, error) {
	if len(item.StepRatios) < 1 || len(item.StepRatios) > 4 {
		return "", fmt.Errorf("OpenLux 阶梯数量必须为 1 到 4")
	}
	baseInput := localModelRatio.Mul(baseInputUSDDecimal)
	completion := decimalValue(item.CompletionRatio, decimal.NewFromInt(1))
	cache := decimalValue(item.CacheRatio, decimal.Zero)
	createCache, createCache1h, createCacheConfigured, createCache1hConfigured, err := cacheCreationRatios(item)
	if err != nil {
		return "", err
	}
	if createCacheConfigured && !createCache1hConfigured {
		createCache1h = createCache.Mul(cacheCreation1hScale)
	}
	image := decimalValue(item.ImageRatio, decimal.NewFromInt(1))
	audio := decimalValue(item.AudioRatio, decimal.NewFromInt(1))
	audioCompletion := decimalValue(item.AudioCompletionRatio, decimal.NewFromInt(1))

	leaves := make([]string, 0, len(item.StepRatios))
	previousStep := decimal.Zero
	previousCompletion := int64(-1)
	for index, step := range item.StepRatios {
		if !step.StepSize.IsPositive() || !step.StepSize.Equal(decimal.NewFromInt(step.StepSize.IntPart())) {
			return "", fmt.Errorf("OpenLux 阶梯 %d 的输入边界无效", index+1)
		}
		if step.CompletionStepSize == 0 || step.CompletionStepSize < -1 {
			return "", fmt.Errorf("OpenLux 阶梯 %d 的输出边界无效", index+1)
		}
		if index > 0 {
			increasing := step.StepSize.GreaterThan(previousStep)
			if step.StepSize.Equal(previousStep) {
				previousBoundary := previousCompletion
				if previousBoundary < 0 {
					previousBoundary = math.MaxInt64
				}
				currentBoundary := step.CompletionStepSize
				if currentBoundary < 0 {
					currentBoundary = math.MaxInt64
				}
				increasing = currentBoundary > previousBoundary
			}
			if !increasing {
				return "", fmt.Errorf("OpenLux 阶梯边界未严格递增")
			}
		}
		if !step.PromptStepRatio.IsPositive() || !step.CompletionStepRatio.IsPositive() || step.CacheStepRatio.IsNegative() {
			return "", fmt.Errorf("OpenLux 阶梯 %d 包含无效价格倍率", index+1)
		}
		if !step.PromptThinkingStepRatio.IsZero() || !step.CompletionThinkingStepRatio.IsZero() {
			return "", fmt.Errorf("OpenLux 提供独立 thinking 阶梯倍率，当前计费表达式无法可靠复现")
		}

		promptPrice := baseInput.Mul(step.PromptStepRatio)
		completionPrice := baseInput.Mul(completion).Mul(step.CompletionStepRatio)
		terms := []string{priceTerm("p", promptPrice), priceTerm("c", completionPrice)}
		if cache.IsPositive() {
			terms = append(terms, priceTerm("cr", baseInput.Mul(cache).Mul(step.CacheStepRatio)))
		}
		if createCache.IsPositive() {
			terms = append(terms, priceTerm("cc", baseInput.Mul(createCache).Mul(step.PromptStepRatio)))
		}
		if createCache1h.IsPositive() {
			terms = append(terms, priceTerm("cc1h", baseInput.Mul(createCache1h).Mul(step.PromptStepRatio)))
		}
		if !image.Equal(decimal.NewFromInt(1)) {
			terms = append(terms, priceTerm("img", baseInput.Mul(image).Mul(step.PromptStepRatio)))
		}
		if !audio.Equal(decimal.NewFromInt(1)) {
			terms = append(terms, priceTerm("ai", baseInput.Mul(audio).Mul(step.PromptStepRatio)))
		}
		if !audioCompletion.Equal(decimal.NewFromInt(1)) {
			terms = append(terms, priceTerm("ao", baseInput.Mul(completion).Mul(audioCompletion).Mul(step.CompletionStepRatio)))
		}
		leaves = append(leaves, fmt.Sprintf("tier(\"tier_%d\", %s)", index+1, strings.Join(terms, " + ")))
		previousStep = step.StepSize
		previousCompletion = step.CompletionStepSize
	}

	expression := leaves[len(leaves)-1]
	for index := len(item.StepRatios) - 2; index >= 0; index-- {
		step := item.StepRatios[index]
		conditions := []string{fmt.Sprintf("len <= %d", step.StepSize.IntPart())}
		if step.CompletionStepSize >= 0 {
			conditions = append(conditions, fmt.Sprintf("c <= %d", step.CompletionStepSize))
		}
		expression = fmt.Sprintf("%s ? %s : (%s)", strings.Join(conditions, " && "), leaves[index], expression)
	}
	if err := billing_setting.SmokeTestExpr(expression); err != nil {
		return "", fmt.Errorf("OpenLux 阶梯表达式校验失败: %w", err)
	}
	return expression, nil
}

func priceTerm(variable string, coefficient decimal.Decimal) string {
	return variable + " * " + coefficient.Round(12).String()
}
