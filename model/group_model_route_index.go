package model

import (
	"errors"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

type groupModelRouteIndexKey struct {
	group string
	model string
}

type groupModelRouteIndex struct {
	db                *gorm.DB
	routes            map[groupModelRouteIndexKey]*GroupModelRoute
	candidates        map[groupModelRouteIndexKey][]GroupModelRouteCandidate
	channels          map[int]*Channel
	groupsByModel     map[string][]string
	modelsByGroup     map[string][]string
	chatModelsByGroup map[string][]string
}

const openAIChatCompletionsPath = "/v1/chat/completions"

var (
	groupModelRouteIndexValue atomic.Pointer[groupModelRouteIndex]
	groupModelRouteIndexLock  sync.Mutex
	routingDataHookLock       sync.RWMutex
	routingDataHook           func()
)

func RegisterRoutingDataChangeHook(hook func()) {
	routingDataHookLock.Lock()
	routingDataHook = hook
	routingDataHookLock.Unlock()
}

func notifyRoutingDataChanged() {
	routingDataHookLock.RLock()
	hook := routingDataHook
	routingDataHookLock.RUnlock()
	if hook != nil {
		hook()
	}
}

func InitGroupModelRouteIndex() error {
	groupModelRouteIndexLock.Lock()
	defer groupModelRouteIndexLock.Unlock()

	if DB == nil {
		return errors.New("database is not initialized")
	}

	var routes []GroupModelRoute
	if DB.Migrator().HasTable(&GroupModelRoute{}) {
		if err := DB.Find(&routes).Error; err != nil {
			return err
		}
	}
	var abilities []Ability
	if DB.Migrator().HasTable(&Ability{}) {
		if err := DB.Where("enabled = ?", true).Find(&abilities).Error; err != nil {
			return err
		}
	}

	channelIDs := make([]int, 0, len(abilities))
	seenChannelIDs := make(map[int]struct{}, len(abilities))
	for _, ability := range abilities {
		if _, ok := seenChannelIDs[ability.ChannelId]; ok {
			continue
		}
		seenChannelIDs[ability.ChannelId] = struct{}{}
		channelIDs = append(channelIDs, ability.ChannelId)
	}

	channels := make(map[int]*Channel, len(channelIDs))
	if len(channelIDs) > 0 && DB.Migrator().HasTable(&Channel{}) {
		var rows []*Channel
		if err := DB.Where("id IN ?", channelIDs).Find(&rows).Error; err != nil {
			return err
		}
		for _, channel := range rows {
			if channel.Status != common.ChannelStatusEnabled {
				continue
			}
			if common.MemoryCacheEnabled {
				if cached, cacheErr := CacheGetChannel(channel.Id); cacheErr == nil && cached != nil {
					channels[channel.Id] = cached
					continue
				}
			}
			channels[channel.Id] = channel
		}
	}

	index := &groupModelRouteIndex{
		db:                DB,
		routes:            make(map[groupModelRouteIndexKey]*GroupModelRoute, len(routes)),
		candidates:        make(map[groupModelRouteIndexKey][]GroupModelRouteCandidate),
		channels:          channels,
		groupsByModel:     make(map[string][]string),
		modelsByGroup:     make(map[string][]string),
		chatModelsByGroup: make(map[string][]string),
	}
	for routeIndex := range routes {
		route := routes[routeIndex]
		route.Tiers = cloneGroupModelRouteTiers(route.Tiers)
		sort.SliceStable(route.Tiers, func(i, j int) bool {
			return route.Tiers[i].Priority > route.Tiers[j].Priority
		})
		index.routes[groupModelRouteIndexKey{group: route.Group, model: route.Model}] = &route
	}

	groupSetsByModel := make(map[string]map[string]struct{})
	modelSetsByGroup := make(map[string]map[string]struct{})
	for _, ability := range abilities {
		channel, ok := channels[ability.ChannelId]
		if !ok {
			continue
		}
		priority := int64(0)
		if ability.Priority != nil {
			priority = *ability.Priority
		}
		key := groupModelRouteIndexKey{group: ability.Group, model: ability.Model}
		index.candidates[key] = append(index.candidates[key], GroupModelRouteCandidate{
			ChannelID:   ability.ChannelId,
			ChannelName: channel.Name,
			Priority:    priority,
			Weight:      ability.Weight,
		})
		if groupSetsByModel[ability.Model] == nil {
			groupSetsByModel[ability.Model] = make(map[string]struct{})
		}
		groupSetsByModel[ability.Model][ability.Group] = struct{}{}
		if modelSetsByGroup[ability.Group] == nil {
			modelSetsByGroup[ability.Group] = make(map[string]struct{})
		}
		modelSetsByGroup[ability.Group][ability.Model] = struct{}{}
	}
	for key := range index.candidates {
		candidates := index.candidates[key]
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].Priority != candidates[j].Priority {
				return candidates[i].Priority > candidates[j].Priority
			}
			if candidates[i].Weight != candidates[j].Weight {
				return candidates[i].Weight > candidates[j].Weight
			}
			return candidates[i].ChannelID < candidates[j].ChannelID
		})
		index.candidates[key] = candidates
	}
	for modelName, groups := range groupSetsByModel {
		index.groupsByModel[modelName] = sortedStringSet(groups)
	}
	for group, models := range modelSetsByGroup {
		index.modelsByGroup[group] = sortedStringSet(models)
	}
	for group, models := range index.modelsByGroup {
		routableModels := make([]string, 0, len(models))
		chatModels := make([]string, 0, len(models))
		for _, modelName := range models {
			plan := cachedGroupModelRoutePlan(index, group, modelName, "")
			if cachedGroupModelRoutePlanHasCandidate(plan) {
				routableModels = append(routableModels, modelName)
			}
			chatPlan := cachedGroupModelRoutePlan(index, group, modelName, openAIChatCompletionsPath)
			if cachedGroupModelRoutePlanHasCandidate(chatPlan) {
				chatModels = append(chatModels, modelName)
			}
		}
		index.modelsByGroup[group] = routableModels
		index.chatModelsByGroup[group] = chatModels
	}

	groupModelRouteIndexValue.Store(index)
	notifyRoutingDataChanged()
	return nil
}

func cloneGroupModelRouteTiers(tiers GroupModelRouteTiers) GroupModelRouteTiers {
	cloned := make(GroupModelRouteTiers, len(tiers))
	for index, tier := range tiers {
		cloned[index] = GroupModelRouteTier{
			Priority: tier.Priority,
			Channels: append([]GroupModelRouteChannel(nil), tier.Channels...),
		}
	}
	return cloned
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func loadGroupModelRouteIndex() (*groupModelRouteIndex, error) {
	if index := groupModelRouteIndexValue.Load(); index != nil && index.db == DB {
		return index, nil
	}
	if err := InitGroupModelRouteIndex(); err != nil {
		return nil, err
	}
	return groupModelRouteIndexValue.Load(), nil
}

func cachedRouteCandidates(index *groupModelRouteIndex, key groupModelRouteIndexKey, requestPath, requestModel string) []GroupModelRouteCandidate {
	candidates := index.candidates[key]
	if requestPath == "" || len(candidates) == 0 {
		return candidates
	}
	filtered := make([]GroupModelRouteCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		channel := index.channels[candidate.ChannelID]
		if channel == nil {
			continue
		}
		if channel.Type != constant.ChannelTypeAdvancedCustom {
			filtered = append(filtered, candidate)
			continue
		}
		config := channel.GetOtherSettings().AdvancedCustom
		if config != nil && config.SupportsPathForModel(requestPath, requestModel) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func cachedGroupModelRoutePlan(index *groupModelRouteIndex, group, modelName, requestPath string) GroupModelRoutePlan {
	normalizedModel := ratio_setting.FormatMatchingModelName(modelName)
	exactKey := groupModelRouteIndexKey{group: group, model: modelName}
	routeModel := modelName
	route := index.routes[exactKey]
	candidates := cachedRouteCandidates(index, exactKey, requestPath, modelName)
	if len(candidates) == 0 && normalizedModel != modelName {
		candidates = cachedRouteCandidates(index, groupModelRouteIndexKey{group: group, model: normalizedModel}, requestPath, modelName)
	}
	if route == nil && normalizedModel != modelName {
		route = index.routes[groupModelRouteIndexKey{group: group, model: normalizedModel}]
		if route != nil {
			routeModel = normalizedModel
		}
	}
	return GroupModelRoutePlan{
		Group:      group,
		RouteModel: routeModel,
		Route:      route,
		Explicit:   route != nil,
		Candidates: candidates,
	}
}

func cachedGroupModelRoutePlanHasCandidate(plan GroupModelRoutePlan) bool {
	if len(plan.Candidates) == 0 {
		return false
	}
	if !plan.Explicit {
		return true
	}

	candidateIDs := make(map[int]struct{}, len(plan.Candidates))
	for _, candidate := range plan.Candidates {
		candidateIDs[candidate.ChannelID] = struct{}{}
	}
	for _, tier := range plan.Route.Tiers {
		for _, channel := range tier.Channels {
			if _, ok := candidateIDs[channel.ChannelID]; ok {
				return true
			}
		}
	}
	return false
}

func GetRouteIndexGroupsByModel() map[string][]string {
	index := groupModelRouteIndexValue.Load()
	if index == nil {
		return map[string][]string{}
	}
	result := make(map[string][]string, len(index.groupsByModel))
	for modelName, groups := range index.groupsByModel {
		result[modelName] = append([]string(nil), groups...)
	}
	return result
}

func FilterRouteGroupsForRequest(groups []string, modelName, requestPath string) []string {
	index := groupModelRouteIndexValue.Load()
	if index == nil {
		return nil
	}
	result := make([]string, 0, len(groups))
	for _, group := range groups {
		plan := cachedGroupModelRoutePlan(index, group, modelName, requestPath)
		if cachedGroupModelRoutePlanHasCandidate(plan) {
			result = append(result, group)
		}
	}
	return result
}

func GetEnabledModelsForGroups(groups []string) []string {
	return GetEnabledModelsForGroupsForRequest(groups, "")
}

func GetEnabledModelsForGroupsForRequest(groups []string, requestPath string) []string {
	index := groupModelRouteIndexValue.Load()
	if index == nil {
		return nil
	}
	modelsByGroup := index.modelsByGroup
	precomputed := requestPath == ""
	if requestPath == openAIChatCompletionsPath {
		modelsByGroup = index.chatModelsByGroup
		precomputed = true
	}
	seen := make(map[string]struct{})
	models := make([]string, 0)
	for _, group := range groups {
		for _, modelName := range modelsByGroup[group] {
			if _, ok := seen[modelName]; ok {
				continue
			}
			if !precomputed {
				plan := cachedGroupModelRoutePlan(index, group, modelName, requestPath)
				if !cachedGroupModelRoutePlanHasCandidate(plan) {
					continue
				}
			}
			seen[modelName] = struct{}{}
			models = append(models, modelName)
		}
	}
	sort.Strings(models)
	return models
}

func GetGroupModelRouteChannel(channelID int) (*Channel, bool) {
	index := groupModelRouteIndexValue.Load()
	if index == nil {
		return nil, false
	}
	channel, ok := index.channels[channelID]
	return channel, ok
}
