package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const openLuxPriceSyncStateID = 1

var ErrOpenLuxPriceSyncRevisionConflict = errors.New("openlux price sync binding revision conflict")

type OpenLuxPriceSyncBinding struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	SourceGroup string `json:"source_group" gorm:"type:varchar(128);not null;uniqueIndex:idx_openlux_source_channel,priority:1"`
	ChannelID   int    `json:"channel_id" gorm:"not null;uniqueIndex;uniqueIndex:idx_openlux_source_channel,priority:2"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (OpenLuxPriceSyncBinding) TableName() string {
	return "openlux_price_sync_bindings"
}

type OpenLuxPriceSyncState struct {
	ID              int   `json:"id" gorm:"primaryKey;autoIncrement:false"`
	BindingRevision int64 `json:"binding_revision" gorm:"not null"`
	UpdatedAt       int64 `json:"updated_at" gorm:"autoUpdateTime"`
}

func (OpenLuxPriceSyncState) TableName() string {
	return "openlux_price_sync_states"
}

type OpenLuxSyncSnapshot struct {
	BindingRevision int64
	Bindings        []OpenLuxPriceSyncBinding
	Channels        []Channel
	Abilities       []Ability
	Routes          []GroupModelRoute
	Options         map[string]string
}

type OpenLuxGroupReferences struct {
	UserLevelGrants            []string
	Tokens                     []int
	GroupRoutes                []string
	BoundChannelRoutes         []string
	NonBoundChannels           []int
	UnexpectedAbilityRows      int
	UnexpectedBoundAbilityRows int
}

func (references OpenLuxGroupReferences) HasBusinessReferences() bool {
	return len(references.UserLevelGrants) > 0 ||
		len(references.Tokens) > 0 ||
		len(references.GroupRoutes) > 0 ||
		len(references.BoundChannelRoutes) > 0 ||
		references.UnexpectedAbilityRows > 0
}

type OpenLuxAbilityRemoval struct {
	ChannelID int
	Group     string
	Model     string
}

type OpenLuxSyncMutation struct {
	Options                   map[string]string
	ChannelModels             map[int]string
	AbilityRemovals           []OpenLuxAbilityRemoval
	DeleteChannelIDs          []int
	DeleteBindingSourceGroups []string
	DeleteRouteGroupCodes     []string
}

type openLuxAbilityTuple struct {
	ChannelID int
	Group     string
	Model     string
}

func GetOpenLuxPriceSyncRevision(db *gorm.DB, forUpdate bool) (int64, error) {
	query := db
	if forUpdate {
		query = lockForUpdate(query)
	}
	var state OpenLuxPriceSyncState
	err := query.First(&state, "id = ?", openLuxPriceSyncStateID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return state.BindingRevision, err
}

func ensureOpenLuxPriceSyncState(tx *gorm.DB) (*OpenLuxPriceSyncState, error) {
	state := OpenLuxPriceSyncState{ID: openLuxPriceSyncStateID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&state).Error; err != nil {
		return nil, err
	}
	if err := lockForUpdate(tx).First(&state, "id = ?", openLuxPriceSyncStateID).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

func LoadOpenLuxSyncSnapshot(db *gorm.DB, forUpdate bool, optionKeys []string) (*OpenLuxSyncSnapshot, error) {
	query := db
	if forUpdate {
		query = lockForUpdate(query)
	}
	revision, err := GetOpenLuxPriceSyncRevision(db, forUpdate)
	if err != nil {
		return nil, err
	}

	snapshot := &OpenLuxSyncSnapshot{
		BindingRevision: revision,
		Options:         make(map[string]string, len(optionKeys)),
	}
	if err := query.Order("source_group asc, channel_id asc").Find(&snapshot.Bindings).Error; err != nil {
		return nil, err
	}

	query = db
	if forUpdate {
		query = lockForUpdate(query)
	}
	if err := query.Order("id asc").Find(&snapshot.Channels).Error; err != nil {
		return nil, err
	}
	query = db
	if forUpdate {
		query = lockForUpdate(query)
	}
	if err := query.Order("channel_id asc").Find(&snapshot.Abilities).Error; err != nil {
		return nil, err
	}
	query = db
	if forUpdate {
		query = lockForUpdate(query)
	}
	if err := query.Order(commonGroupCol + " asc, model asc").Find(&snapshot.Routes).Error; err != nil {
		return nil, err
	}

	if len(optionKeys) > 0 {
		var options []Option
		query = db
		if forUpdate {
			query = lockForUpdate(query)
		}
		if err := query.Where(commonKeyCol+" IN ?", optionKeys).Find(&options).Error; err != nil {
			return nil, err
		}
		for _, option := range options {
			snapshot.Options[option.Key] = option.Value
		}
	}
	return snapshot, nil
}

func ReplaceOpenLuxPriceSyncBindings(tx *gorm.DB, expectedRevision int64, bindings []OpenLuxPriceSyncBinding) (int64, error) {
	state, err := ensureOpenLuxPriceSyncState(tx)
	if err != nil {
		return 0, err
	}
	if state.BindingRevision != expectedRevision {
		return state.BindingRevision, ErrOpenLuxPriceSyncRevisionConflict
	}
	if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&OpenLuxPriceSyncBinding{}).Error; err != nil {
		return 0, err
	}
	if len(bindings) > 0 {
		if err := tx.Create(&bindings).Error; err != nil {
			return 0, err
		}
	}
	state.BindingRevision++
	if err := tx.Model(state).Select("binding_revision", "updated_at").Updates(state).Error; err != nil {
		return 0, err
	}
	return state.BindingRevision, nil
}

func InspectOpenLuxGroupReferences(db *gorm.DB, group string, boundChannelIDs []int, forUpdate bool) (OpenLuxGroupReferences, error) {
	references := OpenLuxGroupReferences{}
	query := db
	if forUpdate {
		query = lockForUpdate(query)
	}
	if err := query.Model(&UserLevelRouteGroup{}).Where("route_group_code = ?", group).Pluck("user_level_code", &references.UserLevelGrants).Error; err != nil {
		return references, err
	}

	var tokens []Token
	query = db
	if forUpdate {
		query = lockForUpdate(query)
	}
	if err := query.Select("id", "group", "group_chain").Find(&tokens).Error; err != nil {
		return references, err
	}
	for _, token := range tokens {
		if token.Group == group || stringSliceContains(token.GroupChain, group) {
			references.Tokens = append(references.Tokens, token.Id)
		}
	}

	bound := make(map[int]struct{}, len(boundChannelIDs))
	for _, id := range boundChannelIDs {
		bound[id] = struct{}{}
	}
	var channels []Channel
	query = db
	if forUpdate {
		query = lockForUpdate(query)
	}
	if err := query.Select("id", "group", "models").Find(&channels).Error; err != nil {
		return references, err
	}
	channelGroups := make(map[int][]string, len(channels))
	channelModels := make(map[int][]string, len(channels))
	for _, channel := range channels {
		channelGroups[channel.Id] = splitCommaValues(channel.Group)
		channelModels[channel.Id] = splitCommaValues(channel.Models)
		if _, ok := bound[channel.Id]; ok {
			continue
		}
		if stringSliceContains(splitCommaValues(channel.Group), group) {
			references.NonBoundChannels = append(references.NonBoundChannels, channel.Id)
		}
	}

	var routes []GroupModelRoute
	query = db
	if forUpdate {
		query = lockForUpdate(query)
	}
	if err := query.Find(&routes).Error; err != nil {
		return references, err
	}
	for _, route := range routes {
		key := route.Group + "/" + route.Model
		if route.Group == group {
			references.GroupRoutes = append(references.GroupRoutes, key)
		}
		if routeUsesAnyChannel(route, bound) {
			references.BoundChannelRoutes = append(references.BoundChannelRoutes, key)
		}
	}

	var abilities []Ability
	query = db
	if forUpdate {
		query = lockForUpdate(query)
	}
	if err := query.Find(&abilities).Error; err != nil {
		return references, err
	}
	abilityTuples := make(map[openLuxAbilityTuple]struct{}, len(abilities))
	for _, ability := range abilities {
		abilityTuples[openLuxAbilityTuple{ChannelID: ability.ChannelId, Group: ability.Group, Model: ability.Model}] = struct{}{}
		groups, channelExists := channelGroups[ability.ChannelId]
		_, isBound := bound[ability.ChannelId]
		if !isBound && ability.Group != group {
			continue
		}
		if !channelExists || !stringSliceContains(groups, ability.Group) ||
			!stringSliceContains(channelModels[ability.ChannelId], ability.Model) ||
			(isBound && ability.Group != group) {
			references.UnexpectedAbilityRows++
			if isBound {
				references.UnexpectedBoundAbilityRows++
			}
		}
	}
	for channelID := range bound {
		for _, modelName := range channelModels[channelID] {
			tuple := openLuxAbilityTuple{ChannelID: channelID, Group: group, Model: modelName}
			if _, exists := abilityTuples[tuple]; !exists {
				references.UnexpectedAbilityRows++
				references.UnexpectedBoundAbilityRows++
			}
		}
	}

	sort.Strings(references.UserLevelGrants)
	sort.Ints(references.Tokens)
	sort.Ints(references.NonBoundChannels)
	sort.Strings(references.GroupRoutes)
	sort.Strings(references.BoundChannelRoutes)
	return references, nil
}

func ApplyOpenLuxSyncMutation(tx *gorm.DB, mutation OpenLuxSyncMutation) (int64, error) {
	if len(mutation.DeleteRouteGroupCodes) > 0 {
		if err := LockAccessPolicyState(tx); err != nil {
			return 0, err
		}
	}
	for key, value := range mutation.Options {
		option := Option{Key: key}
		if err := tx.FirstOrCreate(&option, Option{Key: key}).Error; err != nil {
			return 0, err
		}
		option.Value = value
		if err := tx.Save(&option).Error; err != nil {
			return 0, err
		}
	}
	for channelID, models := range mutation.ChannelModels {
		result := tx.Model(&Channel{}).Where("id = ?", channelID).Update("models", models)
		if result.Error != nil {
			return 0, result.Error
		}
		if result.RowsAffected != 1 {
			return 0, fmt.Errorf("OpenLux sync expected channel %d to exist", channelID)
		}
	}
	for _, removal := range mutation.AbilityRemovals {
		result := tx.Where("channel_id = ? AND "+commonGroupCol+" = ? AND model = ?", removal.ChannelID, removal.Group, removal.Model).
			Delete(&Ability{})
		if result.Error != nil {
			return 0, result.Error
		}
		if result.RowsAffected != 1 {
			return 0, fmt.Errorf("OpenLux sync expected ability %d/%s/%s to exist", removal.ChannelID, removal.Group, removal.Model)
		}
	}
	if len(mutation.DeleteChannelIDs) > 0 {
		if err := tx.Where("channel_id IN ?", mutation.DeleteChannelIDs).Delete(&Ability{}).Error; err != nil {
			return 0, err
		}
		result := tx.Where("id IN ?", mutation.DeleteChannelIDs).Delete(&Channel{})
		if result.Error != nil {
			return 0, result.Error
		}
		if result.RowsAffected != int64(len(mutation.DeleteChannelIDs)) {
			return 0, fmt.Errorf("OpenLux sync expected %d channels to exist, deleted %d", len(mutation.DeleteChannelIDs), result.RowsAffected)
		}
	}
	if len(mutation.DeleteRouteGroupCodes) > 0 {
		result := tx.Where("code IN ?", mutation.DeleteRouteGroupCodes).Delete(&RouteGroup{})
		if result.Error != nil {
			return 0, result.Error
		}
		if result.RowsAffected != int64(len(mutation.DeleteRouteGroupCodes)) {
			return 0, fmt.Errorf("OpenLux sync expected %d route groups to exist, deleted %d", len(mutation.DeleteRouteGroupCodes), result.RowsAffected)
		}
		if err := SyncLegacyAccessPolicyOptions(tx); err != nil {
			return 0, err
		}
	}

	revision, err := GetOpenLuxPriceSyncRevision(tx, false)
	if err != nil {
		return 0, err
	}
	if len(mutation.DeleteBindingSourceGroups) == 0 {
		return revision, nil
	}
	state, err := ensureOpenLuxPriceSyncState(tx)
	if err != nil {
		return 0, err
	}
	if err := tx.Where("source_group IN ?", mutation.DeleteBindingSourceGroups).Delete(&OpenLuxPriceSyncBinding{}).Error; err != nil {
		return 0, err
	}
	state.BindingRevision++
	if err := tx.Model(state).Select("binding_revision", "updated_at").Updates(state).Error; err != nil {
		return 0, err
	}
	return state.BindingRevision, nil
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func splitCommaValues(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func routeUsesAnyChannel(route GroupModelRoute, channelIDs map[int]struct{}) bool {
	for _, tier := range route.Tiers {
		for _, channel := range tier.Channels {
			if _, ok := channelIDs[channel.ChannelID]; ok {
				return true
			}
		}
	}
	return false
}
