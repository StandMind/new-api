package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupModelRouteChannel struct {
	ChannelID int   `json:"channel_id"`
	Weight    int64 `json:"weight"`
}

type GroupModelRouteTier struct {
	Priority int64                    `json:"priority"`
	Channels []GroupModelRouteChannel `json:"channels"`
}

type GroupModelRouteTiers []GroupModelRouteTier

func (tiers GroupModelRouteTiers) Value() (driver.Value, error) {
	data, err := common.Marshal(tiers)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (tiers *GroupModelRouteTiers) Scan(value any) error {
	if value == nil {
		*tiers = nil
		return nil
	}

	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("unsupported GroupModelRouteTiers database value %T", value)
	}
	if len(data) == 0 {
		*tiers = nil
		return nil
	}
	return common.Unmarshal(data, tiers)
}

type GroupModelRoute struct {
	Group     string               `json:"group" gorm:"type:varchar(64);primaryKey;autoIncrement:false"`
	Model     string               `json:"model" gorm:"type:varchar(255);primaryKey;autoIncrement:false"`
	Tiers     GroupModelRouteTiers `json:"tiers" gorm:"type:text;not null"`
	UpdatedAt int64                `json:"updated_at" gorm:"autoUpdateTime:milli"`
}

type GroupModelRouteCandidate struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Priority    int64  `json:"priority"`
	Weight      uint   `json:"weight"`
}

type GroupModelRoutePlan struct {
	Group      string
	RouteModel string
	Route      *GroupModelRoute
	Explicit   bool
	Candidates []GroupModelRouteCandidate
}

type GroupModelRouteListChannel struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Weight      int64  `json:"weight"`
	Eligible    bool   `json:"eligible"`
}

type GroupModelRouteListTier struct {
	Priority int64                        `json:"priority"`
	Channels []GroupModelRouteListChannel `json:"channels"`
}

type GroupModelRouteListItem struct {
	Group     string                    `json:"group"`
	Model     string                    `json:"model"`
	Tiers     []GroupModelRouteListTier `json:"tiers"`
	UpdatedAt int64                     `json:"updated_at"`
}

func NormalizeGroupModelRoute(route *GroupModelRoute) error {
	if route == nil {
		return errors.New("route is required")
	}
	route.Group = strings.TrimSpace(route.Group)
	route.Model = strings.TrimSpace(route.Model)
	if route.Group == "" {
		return errors.New("group is required")
	}
	if route.Model == "" {
		return errors.New("model is required")
	}
	if len(route.Tiers) == 0 {
		return errors.New("at least one priority tier is required")
	}

	prioritySet := make(map[int64]struct{}, len(route.Tiers))
	channelSet := make(map[int]struct{})
	for tierIndex := range route.Tiers {
		tier := &route.Tiers[tierIndex]
		if _, exists := prioritySet[tier.Priority]; exists {
			return fmt.Errorf("priority %d is duplicated", tier.Priority)
		}
		prioritySet[tier.Priority] = struct{}{}
		if len(tier.Channels) == 0 {
			return fmt.Errorf("priority %d has no channels", tier.Priority)
		}
		for _, channel := range tier.Channels {
			if channel.ChannelID <= 0 {
				return errors.New("channel_id must be greater than 0")
			}
			if channel.Weight < 0 || channel.Weight > math.MaxInt32 {
				return fmt.Errorf("channel %d weight must be between 0 and %d", channel.ChannelID, math.MaxInt32)
			}
			if _, exists := channelSet[channel.ChannelID]; exists {
				return fmt.Errorf("channel %d is duplicated", channel.ChannelID)
			}
			channelSet[channel.ChannelID] = struct{}{}
		}
	}

	sort.SliceStable(route.Tiers, func(i, j int) bool {
		return route.Tiers[i].Priority > route.Tiers[j].Priority
	})
	return nil
}

func GetGroupModelRoute(group, model string) (*GroupModelRoute, bool, error) {
	var route GroupModelRoute
	err := DB.Where(&GroupModelRoute{Group: group, Model: model}).First(&route).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &route, true, nil
}

func ListGroupModelRoutes() ([]GroupModelRouteListItem, error) {
	var routes []GroupModelRoute
	if err := DB.
		Order("updated_at DESC").
		Order(commonGroupCol + " ASC").
		Order("model ASC").
		Find(&routes).Error; err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return []GroupModelRouteListItem{}, nil
	}

	type routeKey struct {
		Group string
		Model string
	}
	type abilityKey struct {
		RouteKey  routeKey
		ChannelID int
	}

	groups := make([]string, 0, len(routes))
	models := make([]string, 0, len(routes))
	channelIDs := make([]int, 0)
	seenGroups := make(map[string]struct{}, len(routes))
	seenModels := make(map[string]struct{}, len(routes))
	seenChannelIDs := make(map[int]struct{})
	for _, route := range routes {
		if _, ok := seenGroups[route.Group]; !ok {
			seenGroups[route.Group] = struct{}{}
			groups = append(groups, route.Group)
		}
		if _, ok := seenModels[route.Model]; !ok {
			seenModels[route.Model] = struct{}{}
			models = append(models, route.Model)
		}
		for _, tier := range route.Tiers {
			for _, channel := range tier.Channels {
				if _, ok := seenChannelIDs[channel.ChannelID]; ok {
					continue
				}
				seenChannelIDs[channel.ChannelID] = struct{}{}
				channelIDs = append(channelIDs, channel.ChannelID)
			}
		}
	}

	var channels []Channel
	if len(channelIDs) > 0 {
		if err := DB.Select("id", "name", "status").
			Where("id IN ?", channelIDs).
			Find(&channels).Error; err != nil {
			return nil, err
		}
	}
	channelNames := make(map[int]string, len(channels))
	enabledChannels := make(map[int]struct{}, len(channels))
	for _, channel := range channels {
		channelNames[channel.Id] = channel.Name
		if channel.Status == common.ChannelStatusEnabled {
			enabledChannels[channel.Id] = struct{}{}
		}
	}

	var abilities []Ability
	if len(channelIDs) > 0 {
		if err := DB.Select(commonGroupCol, "model", "channel_id").
			Where(commonGroupCol+" IN ? AND model IN ? AND channel_id IN ? AND enabled = ?", groups, models, channelIDs, true).
			Find(&abilities).Error; err != nil {
			return nil, err
		}
	}
	enabledAbilities := make(map[abilityKey]struct{}, len(abilities))
	for _, ability := range abilities {
		enabledAbilities[abilityKey{
			RouteKey:  routeKey{Group: ability.Group, Model: ability.Model},
			ChannelID: ability.ChannelId,
		}] = struct{}{}
	}

	items := make([]GroupModelRouteListItem, 0, len(routes))
	for _, route := range routes {
		key := routeKey{Group: route.Group, Model: route.Model}
		item := GroupModelRouteListItem{
			Group:     route.Group,
			Model:     route.Model,
			Tiers:     make([]GroupModelRouteListTier, 0, len(route.Tiers)),
			UpdatedAt: route.UpdatedAt,
		}
		for _, tier := range route.Tiers {
			listTier := GroupModelRouteListTier{
				Priority: tier.Priority,
				Channels: make([]GroupModelRouteListChannel, 0, len(tier.Channels)),
			}
			for _, channel := range tier.Channels {
				_, channelEnabled := enabledChannels[channel.ChannelID]
				_, abilityEnabled := enabledAbilities[abilityKey{
					RouteKey:  key,
					ChannelID: channel.ChannelID,
				}]
				listTier.Channels = append(listTier.Channels, GroupModelRouteListChannel{
					ChannelID:   channel.ChannelID,
					ChannelName: channelNames[channel.ChannelID],
					Weight:      channel.Weight,
					Eligible:    channelEnabled && abilityEnabled,
				})
			}
			item.Tiers = append(item.Tiers, listTier)
		}
		items = append(items, item)
	}
	return items, nil
}

func GetGroupModelRouteCandidates(group, model string) ([]GroupModelRouteCandidate, error) {
	return getGroupModelRouteCandidates(group, model, "")
}

func GetGroupModelRouteCandidatesForRequest(group, model, requestPath string) ([]GroupModelRouteCandidate, error) {
	candidates, err := getGroupModelRouteCandidates(group, model, requestPath)
	if err != nil || len(candidates) > 0 {
		return candidates, err
	}
	normalizedModel := ratio_setting.FormatMatchingModelName(model)
	if normalizedModel == model {
		return candidates, nil
	}
	return getGroupModelRouteCandidates(group, normalizedModel, requestPath)
}

func LoadGroupModelRoutePlans(groups []string, modelName, requestPath string) ([]GroupModelRoutePlan, error) {
	uniqueGroups := make([]string, 0, len(groups))
	seenGroups := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if _, seen := seenGroups[group]; seen {
			continue
		}
		seenGroups[group] = struct{}{}
		uniqueGroups = append(uniqueGroups, group)
	}
	if len(uniqueGroups) == 0 {
		return []GroupModelRoutePlan{}, nil
	}

	index, err := loadGroupModelRouteIndex()
	if err != nil {
		return nil, err
	}
	normalizedModel := ratio_setting.FormatMatchingModelName(modelName)

	plans := make([]GroupModelRoutePlan, 0, len(uniqueGroups))
	for _, group := range uniqueGroups {
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
		plans = append(plans, GroupModelRoutePlan{
			Group:      group,
			RouteModel: routeModel,
			Route:      route,
			Explicit:   route != nil,
			Candidates: candidates,
		})
	}
	return plans, nil
}

func getGroupModelRouteCandidates(group, model, requestPath string) ([]GroupModelRouteCandidate, error) {
	var abilities []Ability
	if err := DB.Where(&Ability{Group: group, Model: model, Enabled: true}).
		Order("priority DESC").
		Order("weight DESC").
		Find(&abilities).Error; err != nil {
		return nil, err
	}
	if len(abilities) == 0 {
		return []GroupModelRouteCandidate{}, nil
	}
	abilities = filterAbilitiesByRequestPathAndModel(abilities, requestPath, model)

	channelIDs := make([]int, 0, len(abilities))
	for _, ability := range abilities {
		channelIDs = append(channelIDs, ability.ChannelId)
	}
	var channels []Channel
	if err := DB.Select("id", "name", "status").Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
		return nil, err
	}
	channelNames := make(map[int]string, len(channels))
	for _, channel := range channels {
		if channel.Status == common.ChannelStatusEnabled {
			channelNames[channel.Id] = channel.Name
		}
	}

	candidates := make([]GroupModelRouteCandidate, 0, len(abilities))
	for _, ability := range abilities {
		channelName, enabled := channelNames[ability.ChannelId]
		if !enabled {
			continue
		}
		priority := int64(0)
		if ability.Priority != nil {
			priority = *ability.Priority
		}
		candidates = append(candidates, GroupModelRouteCandidate{
			ChannelID:   ability.ChannelId,
			ChannelName: channelName,
			Priority:    priority,
			Weight:      ability.Weight,
		})
	}
	return candidates, nil
}

func SaveGroupModelRoute(route *GroupModelRoute) error {
	if err := NormalizeGroupModelRoute(route); err != nil {
		return err
	}

	candidates, err := GetGroupModelRouteCandidates(route.Group, route.Model)
	if err != nil {
		return err
	}
	available := make(map[int]struct{}, len(candidates))
	for _, candidate := range candidates {
		available[candidate.ChannelID] = struct{}{}
	}
	for _, tier := range route.Tiers {
		for _, channel := range tier.Channels {
			if _, ok := available[channel.ChannelID]; !ok {
				return fmt.Errorf("channel %d has no enabled ability for group %s and model %s", channel.ChannelID, route.Group, route.Model)
			}
		}
	}

	if err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "group"}, {Name: "model"}},
		DoUpdates: clause.AssignmentColumns([]string{"tiers", "updated_at"}),
	}).Create(route).Error; err != nil {
		return err
	}
	return InitGroupModelRouteIndex()
}

func DeleteGroupModelRoute(group, model string) (bool, error) {
	result := DB.Where(&GroupModelRoute{Group: group, Model: model}).Delete(&GroupModelRoute{})
	if result.Error != nil || result.RowsAffected == 0 {
		return result.RowsAffected > 0, result.Error
	}
	if err := InitGroupModelRouteIndex(); err != nil {
		return true, err
	}
	return true, nil
}
