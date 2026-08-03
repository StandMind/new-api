package openluxsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type resolvedBinding struct {
	SourceGroup       string
	LocalGroup        string
	ChannelIDs        []int
	MixedChannelCount int
	Healthy           bool
	Issues            []string
}

type associationState struct {
	Models             map[string]struct{}
	ConsistencyByModel map[string][]string
	ChannelModelsByID  map[int][]string
	AbilityByTuple     map[string]struct{}
}

func Preview(ctx context.Context) (*PreviewResponse, error) {
	source, err := fetchSource(ctx)
	if err != nil {
		return nil, err
	}
	snapshot, err := loadSnapshot(model.DB, false)
	if err != nil {
		return nil, err
	}
	return buildPreview(model.DB, source, snapshot, false)
}

func buildPreview(db *gorm.DB, source *sourceSnapshot, snapshot *model.OpenLuxSyncSnapshot, forUpdate bool) (*PreviewResponse, error) {
	options, err := parseLocalPricingOptions(snapshot)
	if err != nil {
		return nil, unprocessable(err.Error())
	}
	bindings := resolveBindings(snapshot)
	boundSources := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		boundSources[binding.SourceGroup] = struct{}{}
	}
	fingerprint, err := localFingerprint(snapshot)
	if err != nil {
		return nil, unprocessable("无法生成本地状态指纹: " + err.Error())
	}

	channelByID := make(map[int]model.Channel, len(snapshot.Channels))
	localModels := make(map[string]struct{})
	for _, channel := range snapshot.Channels {
		channelByID[channel.Id] = channel
		for _, modelName := range splitCSV(channel.Models) {
			localModels[modelName] = struct{}{}
		}
	}
	groupsByModel := make(map[string]map[string]struct{})
	for _, ability := range snapshot.Abilities {
		if groupsByModel[ability.Model] == nil {
			groupsByModel[ability.Model] = make(map[string]struct{})
		}
		groupsByModel[ability.Model][ability.Group] = struct{}{}
	}

	response := &PreviewResponse{
		SourceHash:       source.Hash,
		LocalFingerprint: fingerprint,
		BindingRevision:  snapshot.BindingRevision,
		FetchedAt:        source.FetchedAt,
		Changes:          make([]Change, 0),
		Notices:          make([]Notice, 0),
	}
	for name := range source.Groups {
		if name == "default" {
			continue
		}
		if _, bound := boundSources[name]; !bound {
			response.Notices = append(response.Notices, Notice{Kind: "new_source_group", SourceGroup: name, Message: "OpenLux 来源组尚未绑定，仅提示，不会自动创建令牌或渠道"})
		}
	}
	for group := range source.ReferencedGroups {
		if _, registered := source.Groups[group]; !registered {
			response.Notices = append(response.Notices, Notice{Kind: "unregistered_source_group", SourceGroup: group, Message: "模型仍引用该组，但 OpenLux 顶层分组表未注册；已忽略且不会删除"})
		}
	}
	for _, item := range source.OrderedModels {
		if _, exists := localModels[item.Name]; !exists {
			response.Notices = append(response.Notices, Notice{Kind: "new_model", Model: item.Name, Message: "OpenLux 提供了本地尚未启用的模型，仅提示"})
		}
	}

	associationBySource := make(map[string]associationState, len(bindings))
	eligibleModels := make(map[string]struct{})
	for _, binding := range bindings {
		state := buildAssociationState(binding, snapshot, channelByID)
		associationBySource[binding.SourceGroup] = state
		if !binding.Healthy {
			response.Notices = append(response.Notices, Notice{
				Kind: "binding_error", SourceGroup: binding.SourceGroup, LocalGroup: binding.LocalGroup,
				Message: strings.Join(binding.Issues, "；"),
			})
		}
		for modelName := range state.Models {
			item, exists := source.Models[modelName]
			if exists && containsString(item.EnableGroups, binding.SourceGroup) {
				eligibleModels[modelName] = struct{}{}
			}
		}
	}

	billingByModel := make(map[string]modelBillingResult, len(eligibleModels))
	eligibleNames := make([]string, 0, len(eligibleModels))
	for modelName := range eligibleModels {
		eligibleNames = append(eligibleNames, modelName)
	}
	sort.Strings(eligibleNames)
	for _, modelName := range eligibleNames {
		affected := make([]string, 0, len(groupsByModel[modelName]))
		for group := range groupsByModel[modelName] {
			affected = append(affected, group)
		}
		sort.Strings(affected)
		result := buildModelBillingChange(source.Models[modelName], options, affected)
		billingByModel[modelName] = result
		if result.change != nil {
			response.Changes = append(response.Changes, *result.change)
		}
	}

	for _, binding := range bindings {
		state := associationBySource[binding.SourceGroup]
		sourceGroup, sourceGroupExists := source.Groups[binding.SourceGroup]
		if !sourceGroupExists {
			if _, stillReferenced := source.ReferencedGroups[binding.SourceGroup]; stillReferenced {
				response.Notices = append(response.Notices, Notice{
					Kind: "source_group_registry_missing", SourceGroup: binding.SourceGroup, LocalGroup: binding.LocalGroup,
					Message: "绑定组从顶层分组表消失但仍被模型引用，禁止按删除处理",
				})
				continue
			}
			change, buildErr := buildSourceGroupRemoval(db, binding, snapshot, forUpdate)
			if buildErr != nil {
				return nil, buildErr
			}
			response.Changes = append(response.Changes, change)
			continue
		}

		modelNames := make([]string, 0, len(state.Models))
		for modelName := range state.Models {
			modelNames = append(modelNames, modelName)
		}
		sort.Strings(modelNames)
		for _, modelName := range modelNames {
			item, modelExists := source.Models[modelName]
			if !modelExists {
				response.Notices = append(response.Notices, Notice{
					Kind: "source_model_missing", Model: modelName, SourceGroup: binding.SourceGroup, LocalGroup: binding.LocalGroup,
					Message: "模型已从 OpenLux 定价列表整体消失，不会自动删除本地模型或关联",
				})
				continue
			}
			if !containsString(item.EnableGroups, binding.SourceGroup) {
				change := buildGroupModelRemoval(binding, modelName, state, snapshot, channelByID, options)
				response.Changes = append(response.Changes, change)
				continue
			}

			quotaType, localBase, baseExists := options.localBase(modelName)
			blocked := append([]string(nil), binding.Issues...)
			blocked = append(blocked, state.ConsistencyByModel[modelName]...)
			if !baseExists || !localBase.IsPositive() {
				blocked = append(blocked, "本地模型缺少正数基础价格")
			} else if quotaType != item.QuotaType {
				blocked = append(blocked, "OpenLux 与本地的 Token/固定价计费类型不一致")
			}
			target, targetErr := targetGroupModelRatio(item, sourceGroup.Ratio, localBase)
			if targetErr != nil {
				blocked = append(blocked, targetErr.Error())
			}
			billing := billingByModel[modelName]
			if len(billing.blocked) > 0 {
				blocked = append(blocked, billing.blocked...)
			}
			current := options.effectiveGroupModelRatio(binding.LocalGroup, modelName).Round(15)
			if targetErr != nil || current.Equal(target) {
				continue
			}
			change := Change{
				Kind: "group_model_price_update", Model: modelName, SourceGroup: binding.SourceGroup,
				LocalGroup: binding.LocalGroup, CurrentValue: current.String(), TargetValue: target.String(),
				Actionable: len(blocked) == 0, BlockedReasons: uniqueStrings(blocked),
				MixedChannelCount: binding.MixedChannelCount,
				mutations:         []optionMutation{{Key: optionGroupModelRatio, Action: "set_nested", Group: binding.LocalGroup, Model: modelName, Value: target.String()}},
			}
			if billing.change != nil && billing.change.Actionable {
				change.Requires = []string{billing.change.ID}
			}
			if quotaType == 0 {
				change.PriceUnit = "USD / 1M input tokens"
				change.CurrentPrice = localBase.Mul(baseInputUSDDecimal).Mul(current).Round(12).String()
				change.TargetPrice = localBase.Mul(baseInputUSDDecimal).Mul(target).Round(12).String()
			} else {
				change.PriceUnit = "USD / request"
				change.CurrentPrice = localBase.Mul(current).Round(12).String()
				change.TargetPrice = localBase.Mul(target).Round(12).String()
			}
			change.PercentChange = percentChange(current, target)
			change.ID = makeChangeID(change)
			response.Changes = append(response.Changes, change)
		}

		for _, item := range source.OrderedModels {
			if !containsString(item.EnableGroups, binding.SourceGroup) {
				continue
			}
			if _, exists := state.Models[item.Name]; exists {
				continue
			}
			if _, locallyKnown := localModels[item.Name]; locallyKnown {
				response.Notices = append(response.Notices, Notice{
					Kind: "new_model_group_association", Model: item.Name, SourceGroup: binding.SourceGroup,
					LocalGroup: binding.LocalGroup, Message: "OpenLux 新增了模型与来源组关联；需要新令牌能力，仅提示",
				})
			}
		}
	}

	sortPreview(response)
	for _, change := range response.Changes {
		if change.Actionable {
			response.Summary.Actionable++
		} else {
			response.Summary.Blocked++
		}
		if change.Kind == "group_model_price_update" {
			response.Summary.Price++
		}
		if change.Destructive {
			response.Summary.Removal++
		}
	}
	response.Summary.Notices = len(response.Notices)
	return response, nil
}

func resolveBindings(snapshot *model.OpenLuxSyncSnapshot) []resolvedBinding {
	channelByID := make(map[int]model.Channel, len(snapshot.Channels))
	for _, channel := range snapshot.Channels {
		channelByID[channel.Id] = channel
	}
	idsBySource := make(map[string][]int)
	for _, binding := range snapshot.Bindings {
		idsBySource[binding.SourceGroup] = append(idsBySource[binding.SourceGroup], binding.ChannelID)
	}
	sources := make([]string, 0, len(idsBySource))
	for sourceGroup := range idsBySource {
		sources = append(sources, sourceGroup)
	}
	sort.Strings(sources)
	result := make([]resolvedBinding, 0, len(sources))
	for _, sourceGroup := range sources {
		view := bindingView(sourceGroup, idsBySource[sourceGroup], channelByID, snapshot.Channels, false)
		result = append(result, resolvedBinding{
			SourceGroup: sourceGroup, LocalGroup: view.LocalGroup, ChannelIDs: append([]int(nil), idsBySource[sourceGroup]...),
			MixedChannelCount: view.MixedChannelCount, Healthy: view.Healthy, Issues: append([]string(nil), view.Issues...),
		})
	}
	return result
}

func buildAssociationState(binding resolvedBinding, snapshot *model.OpenLuxSyncSnapshot, channelByID map[int]model.Channel) associationState {
	state := associationState{
		Models: make(map[string]struct{}), ConsistencyByModel: make(map[string][]string),
		ChannelModelsByID: make(map[int][]string), AbilityByTuple: make(map[string]struct{}),
	}
	bound := make(map[int]struct{}, len(binding.ChannelIDs))
	for _, channelID := range binding.ChannelIDs {
		bound[channelID] = struct{}{}
		channel, exists := channelByID[channelID]
		if !exists {
			continue
		}
		models := splitCSV(channel.Models)
		state.ChannelModelsByID[channelID] = models
		for _, modelName := range models {
			state.Models[modelName] = struct{}{}
		}
	}
	for _, ability := range snapshot.Abilities {
		if _, ok := bound[ability.ChannelId]; !ok {
			continue
		}
		state.Models[ability.Model] = struct{}{}
		key := associationKey(ability.ChannelId, ability.Group, ability.Model)
		state.AbilityByTuple[key] = struct{}{}
		if ability.Group != binding.LocalGroup {
			state.ConsistencyByModel[ability.Model] = append(state.ConsistencyByModel[ability.Model], fmt.Sprintf("渠道 #%d 存在不匹配分组的 Ability", ability.ChannelId))
			continue
		}
		if !containsString(state.ChannelModelsByID[ability.ChannelId], ability.Model) {
			state.ConsistencyByModel[ability.Model] = append(state.ConsistencyByModel[ability.Model], fmt.Sprintf("渠道 #%d 的 Ability 与模型列表不一致", ability.ChannelId))
		}
	}
	for channelID, models := range state.ChannelModelsByID {
		for _, modelName := range models {
			if _, ok := state.AbilityByTuple[associationKey(channelID, binding.LocalGroup, modelName)]; !ok {
				state.ConsistencyByModel[modelName] = append(state.ConsistencyByModel[modelName], fmt.Sprintf("渠道 #%d 缺少对应 Ability", channelID))
			}
		}
	}
	for modelName, issues := range state.ConsistencyByModel {
		state.ConsistencyByModel[modelName] = uniqueStrings(issues)
	}
	return state
}

func buildGroupModelRemoval(binding resolvedBinding, modelName string, state associationState, snapshot *model.OpenLuxSyncSnapshot, channelByID map[int]model.Channel, options *localPricingOptions) Change {
	blocked := append([]string(nil), binding.Issues...)
	blocked = append(blocked, state.ConsistencyByModel[modelName]...)
	bound := make(map[int]struct{}, len(binding.ChannelIDs))
	for _, id := range binding.ChannelIDs {
		bound[id] = struct{}{}
	}
	routeKeys := make([]string, 0)
	for _, route := range snapshot.Routes {
		if route.Model == modelName && routeUsesChannels(route, bound) {
			routeKeys = append(routeKeys, route.Group+"/"+route.Model)
		}
	}
	if routeKeys = uniqueStrings(routeKeys); len(routeKeys) > 0 {
		blocked = append(blocked, "显式模型路由仍引用待移除的 OpenLux 渠道: "+strings.Join(routeKeys, ", "))
	}
	remainingProvider := false
	for _, channel := range snapshot.Channels {
		if _, isBound := bound[channel.Id]; isBound || !containsString(splitCSV(channel.Group), binding.LocalGroup) || !containsString(splitCSV(channel.Models), modelName) {
			continue
		}
		for _, ability := range snapshot.Abilities {
			if ability.ChannelId == channel.Id && ability.Group == binding.LocalGroup && ability.Model == modelName {
				remainingProvider = true
				break
			}
		}
	}
	change := Change{
		Kind: "group_model_remove", Model: modelName, SourceGroup: binding.SourceGroup, LocalGroup: binding.LocalGroup,
		CurrentValue: "associated", TargetValue: "removed_from_openlux", Actionable: len(blocked) == 0,
		Destructive: true, BlockedReasons: uniqueStrings(blocked), MixedChannelCount: binding.MixedChannelCount,
	}
	for _, channelID := range binding.ChannelIDs {
		if containsString(state.ChannelModelsByID[channelID], modelName) {
			change.abilityRemovals = append(change.abilityRemovals, abilityRemoval{ChannelID: channelID, Group: binding.LocalGroup, Model: modelName})
		}
	}
	if remainingProvider {
		change.RemoveMode = "openlux_only_price_retained"
		change.Details = append(change.Details, ChangeDetail{Field: "GroupModelRatio", Current: options.effectiveGroupModelRatio(binding.LocalGroup, modelName).String(), Target: "retained_unmanaged"})
	} else {
		change.RemoveMode = "association_and_price"
		change.mutations = append(change.mutations, optionMutation{Key: optionGroupModelRatio, Action: "delete_nested", Group: binding.LocalGroup, Model: modelName})
	}
	change.ID = makeChangeID(change)
	return change
}

func buildSourceGroupRemoval(db *gorm.DB, binding resolvedBinding, snapshot *model.OpenLuxSyncSnapshot, forUpdate bool) (Change, error) {
	change := Change{
		Kind: "source_group_remove", SourceGroup: binding.SourceGroup, LocalGroup: binding.LocalGroup,
		CurrentValue: "bound", TargetValue: "removed", Destructive: true,
		MixedChannelCount: binding.MixedChannelCount, deleteChannelIDs: append([]int(nil), binding.ChannelIDs...),
		deleteBindings: []string{binding.SourceGroup},
	}
	blocked := append([]string(nil), binding.Issues...)
	if binding.SourceGroup == "default" || binding.LocalGroup == "default" || binding.LocalGroup == "" {
		blocked = append(blocked, "default 或无法解析的分组禁止自动删除")
	}
	references, err := model.InspectOpenLuxGroupReferences(db, binding.LocalGroup, binding.ChannelIDs, forUpdate)
	if err != nil {
		return Change{}, err
	}
	if len(references.NonBoundChannels) > 0 {
		change.RemoveMode = "detach_openlux_only"
		if len(references.BoundChannelRoutes) > 0 {
			blocked = append(blocked, routeReferenceReason("显式路由引用待删 OpenLux 渠道", references.BoundChannelRoutes))
		}
		if references.UnexpectedBoundAbilityRows > 0 {
			blocked = append(blocked, fmt.Sprintf("待删 OpenLux 渠道存在 %d 条异常 Ability", references.UnexpectedBoundAbilityRows))
		}
	} else {
		change.RemoveMode = "delete_local_group"
		blocked = append(blocked, referenceBlockers(references)...)
		change.mutations = append(change.mutations, groupDeletionMutations(binding.LocalGroup)...)
	}
	change.BlockedReasons = uniqueStrings(blocked)
	change.Actionable = len(change.BlockedReasons) == 0
	change.ID = makeChangeID(change)
	return change, nil
}

func referenceBlockers(references model.OpenLuxGroupReferences) []string {
	result := make([]string, 0)
	if len(references.UserLevelGrants) > 0 {
		result = append(result, fmt.Sprintf("%d 个用户等级仍授权该路由分组: %s", len(references.UserLevelGrants), strings.Join(references.UserLevelGrants, ", ")))
	}
	if len(references.Tokens) > 0 {
		result = append(result, fmt.Sprintf("%d 个 Token 或分组链仍引用该分组", len(references.Tokens)))
	}
	if len(references.GroupRoutes) > 0 {
		result = append(result, routeReferenceReason("显式模型路由属于该分组", references.GroupRoutes))
	}
	if len(references.BoundChannelRoutes) > 0 {
		result = append(result, routeReferenceReason("显式路由引用待删渠道", references.BoundChannelRoutes))
	}
	if references.UnexpectedAbilityRows > 0 {
		result = append(result, fmt.Sprintf("发现 %d 条孤立或分组不匹配的 Ability", references.UnexpectedAbilityRows))
	}
	return result
}

func routeReferenceReason(label string, routeKeys []string) string {
	routeKeys = uniqueStrings(routeKeys)
	return fmt.Sprintf("%d 条%s: %s", len(routeKeys), label, strings.Join(routeKeys, ", "))
}

func groupDeletionMutations(group string) []optionMutation {
	return []optionMutation{
		{Key: optionGroupModelRatio, Action: "delete", Model: group},
	}
}

func makeChangeID(change Change) string {
	view := struct {
		Version        string         `json:"version"`
		Kind           string         `json:"kind"`
		Model          string         `json:"model"`
		SourceGroup    string         `json:"source_group"`
		LocalGroup     string         `json:"local_group"`
		CurrentValue   string         `json:"current_value"`
		TargetValue    string         `json:"target_value"`
		RemoveMode     string         `json:"remove_mode"`
		Requires       []string       `json:"requires"`
		BlockedReasons []string       `json:"blocked_reasons"`
		Details        []ChangeDetail `json:"details"`
	}{
		Version: "openlux-price-sync/v1", Kind: change.Kind, Model: change.Model,
		SourceGroup: change.SourceGroup, LocalGroup: change.LocalGroup,
		CurrentValue: change.CurrentValue, TargetValue: change.TargetValue, RemoveMode: change.RemoveMode,
		Requires: change.Requires, BlockedReasons: change.BlockedReasons, Details: change.Details,
	}
	encoded, _ := common.Marshal(view)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func routeUsesChannels(route model.GroupModelRoute, channelIDs map[int]struct{}) bool {
	for _, tier := range route.Tiers {
		for _, channel := range tier.Channels {
			if _, ok := channelIDs[channel.ChannelID]; ok {
				return true
			}
		}
	}
	return false
}

func associationKey(channelID int, group, modelName string) string {
	return fmt.Sprintf("%d\x00%s\x00%s", channelID, group, modelName)
}

func percentChange(current, target decimal.Decimal) string {
	if current.IsZero() {
		return ""
	}
	return target.Div(current).Sub(decimal.NewFromInt(1)).Mul(decimal.NewFromInt(100)).Round(4).String()
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortPreview(response *PreviewResponse) {
	sort.Slice(response.Changes, func(i, j int) bool {
		left, right := response.Changes[i], response.Changes[j]
		if left.SourceGroup != right.SourceGroup {
			return left.SourceGroup < right.SourceGroup
		}
		if left.LocalGroup != right.LocalGroup {
			return left.LocalGroup < right.LocalGroup
		}
		if left.Model != right.Model {
			return left.Model < right.Model
		}
		return left.Kind < right.Kind
	})
	sort.Slice(response.Notices, func(i, j int) bool {
		left, right := response.Notices[i], response.Notices[j]
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.SourceGroup != right.SourceGroup {
			return left.SourceGroup < right.SourceGroup
		}
		return left.Model < right.Model
	})
}
