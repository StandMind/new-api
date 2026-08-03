package model

import (
	"errors"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

type AccessPolicyRouteGroup struct {
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	BaseRatio   AccessPolicyRatio  `json:"base_ratio"`
	Enabled     bool               `json:"enabled"`
	PriceRatio  *AccessPolicyRatio `json:"price_ratio"`
}

type accessPolicySnapshot struct {
	db           *gorm.DB
	levels       map[string]UserLevel
	routeGroups  map[string]RouteGroup
	grants       map[string]map[string]UserLevelRouteGroup
	defaultLevel string
}

var (
	accessPolicyValue atomic.Pointer[accessPolicySnapshot]
	accessPolicyLock  sync.Mutex
)

func InitAccessPolicySnapshot() error {
	return RebuildAccessPolicySnapshot()
}

func RebuildAccessPolicySnapshot() error {
	accessPolicyLock.Lock()
	defer accessPolicyLock.Unlock()
	if DB == nil {
		return errors.New("database is not initialized")
	}
	var levels []UserLevel
	if err := DB.Find(&levels).Error; err != nil {
		return err
	}
	var routeGroups []RouteGroup
	if err := DB.Find(&routeGroups).Error; err != nil {
		return err
	}
	var grants []UserLevelRouteGroup
	if err := DB.Find(&grants).Error; err != nil {
		return err
	}
	snapshot := &accessPolicySnapshot{
		db:          DB,
		levels:      make(map[string]UserLevel, len(levels)),
		routeGroups: make(map[string]RouteGroup, len(routeGroups)),
		grants:      make(map[string]map[string]UserLevelRouteGroup),
	}
	for _, level := range levels {
		level.RouteGroups = nil
		snapshot.levels[level.Code] = level
		if level.IsDefault && level.Enabled && (snapshot.defaultLevel == "" || level.Code < snapshot.defaultLevel) {
			snapshot.defaultLevel = level.Code
		}
	}
	if snapshot.defaultLevel == "" {
		snapshot.defaultLevel = StandardUserLevelCode
	}
	for _, group := range routeGroups {
		snapshot.routeGroups[group.Code] = group
	}
	for _, grant := range grants {
		if snapshot.grants[grant.UserLevelCode] == nil {
			snapshot.grants[grant.UserLevelCode] = make(map[string]UserLevelRouteGroup)
		}
		snapshot.grants[grant.UserLevelCode][grant.RouteGroupCode] = grant
	}
	accessPolicyValue.Store(snapshot)
	return nil
}

func getAccessPolicySnapshot() *accessPolicySnapshot {
	snapshot := accessPolicyValue.Load()
	if snapshot == nil || snapshot.db != DB {
		return nil
	}
	return snapshot
}

func AccessPolicySnapshotReady() bool { return getAccessPolicySnapshot() != nil }

func GetDefaultUserLevelFromSnapshot() string {
	if snapshot := getAccessPolicySnapshot(); snapshot != nil {
		return snapshot.defaultLevel
	}
	return StandardUserLevelCode
}

func GetUserLevelFromSnapshot(code string) (UserLevel, bool) {
	snapshot := getAccessPolicySnapshot()
	if snapshot == nil {
		return UserLevel{}, false
	}
	level, ok := snapshot.levels[code]
	return level, ok
}

func GetUserLevelDisplayName(code string) string {
	if level, ok := GetUserLevelFromSnapshot(code); ok && level.Name != "" {
		return level.Name
	}
	return code
}

func GetRouteGroupDisplayName(code string) string {
	if group, ok := GetRouteGroupFromSnapshot(code); ok && group.Name != "" {
		return group.Name
	}
	return code
}

func GetRouteGroupFromSnapshot(code string) (RouteGroup, bool) {
	snapshot := getAccessPolicySnapshot()
	if snapshot == nil {
		return RouteGroup{}, false
	}
	group, ok := snapshot.routeGroups[code]
	return group, ok
}

func GetEnabledRouteGroupCodes() []string {
	snapshot := getAccessPolicySnapshot()
	if snapshot == nil {
		return nil
	}
	result := make([]string, 0, len(snapshot.routeGroups))
	for code, group := range snapshot.routeGroups {
		if group.Enabled {
			result = append(result, code)
		}
	}
	sort.Strings(result)
	return result
}

func GetKnownUserLevelCodes() []string {
	snapshot := getAccessPolicySnapshot()
	if snapshot == nil {
		return nil
	}
	result := make([]string, 0, len(snapshot.levels))
	for code, level := range snapshot.levels {
		if level.Enabled {
			result = append(result, code)
		}
	}
	sort.Strings(result)
	return result
}

func GetUserLevelRouteGroups(userLevel string, includeDisabled bool) []AccessPolicyRouteGroup {
	snapshot := getAccessPolicySnapshot()
	if snapshot == nil {
		return nil
	}
	level, ok := snapshot.levels[userLevel]
	if !ok || !level.Enabled {
		return nil
	}
	grants := snapshot.grants[userLevel]
	result := make([]AccessPolicyRouteGroup, 0, len(grants))
	for routeCode, grant := range grants {
		group, exists := snapshot.routeGroups[routeCode]
		if !exists || (!includeDisabled && !group.Enabled) {
			continue
		}
		result = append(result, AccessPolicyRouteGroup{
			Code: group.Code, Name: group.Name, Description: group.Description,
			BaseRatio: group.BaseRatio, Enabled: group.Enabled, PriceRatio: grant.PriceRatio,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result
}

func UserLevelCanAccessRouteGroup(userLevel, routeGroup string) bool {
	snapshot := getAccessPolicySnapshot()
	if snapshot == nil {
		return false
	}
	level, levelExists := snapshot.levels[userLevel]
	group, groupExists := snapshot.routeGroups[routeGroup]
	if !levelExists || !level.Enabled || !groupExists || !group.Enabled {
		return false
	}
	_, granted := snapshot.grants[userLevel][routeGroup]
	return granted
}

func ResolveAccessPolicyRatio(userLevel, routeGroup, modelName string) (float64, string) {
	if ratio, source, ok := ratio_setting.ResolveGroupModelRatio(routeGroup, modelName); ok {
		return ratio, source
	}
	snapshot := getAccessPolicySnapshot()
	if snapshot == nil {
		return 1, "access_policy_snapshot.missing"
	}
	if grant, ok := snapshot.grants[userLevel][routeGroup]; ok && grant.PriceRatio != nil {
		return float64(*grant.PriceRatio), "user_level_route_group.price_ratio"
	}
	if group, ok := snapshot.routeGroups[routeGroup]; ok {
		return float64(group.BaseRatio), "route_group.base_ratio"
	}
	return 1, "route_group.missing"
}

func GetUserLevelTopupRatio(userLevel string) float64 {
	if level, ok := GetUserLevelFromSnapshot(userLevel); ok && level.TopupRatio > 0 {
		return float64(level.TopupRatio)
	}
	return 1
}

func GetUserLevelRequestLimits(userLevel string) (int, int, bool) {
	level, ok := GetUserLevelFromSnapshot(userLevel)
	if !ok || !level.Enabled || (level.RequestLimit <= 0 && level.SuccessRequestLimit <= 0) {
		return 0, 0, false
	}
	return level.RequestLimit, level.SuccessRequestLimit, true
}

func NotifyAccessPolicyChanged() {
	notifyRoutingDataChanged()
}
