package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const (
	StandardUserLevelCode  = "standard"
	LegacyDefaultGroupCode = "default"
)

type AccessPolicyRatio float64

func (AccessPolicyRatio) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "real"
	case "postgres":
		return "numeric(20,15)"
	default:
		return "decimal(20,15)"
	}
}

// UserLevel contains account policy only. It is deliberately independent from
// channel routing groups, whose codes may happen to have the same text.
type UserLevel struct {
	Code                string                `json:"code" gorm:"type:varchar(64);primaryKey"`
	Name                string                `json:"name" gorm:"type:varchar(128);not null"`
	Description         string                `json:"description" gorm:"type:varchar(255);default:''"`
	IsDefault           bool                  `json:"is_default" gorm:"not null;index"`
	Enabled             bool                  `json:"enabled" gorm:"not null"`
	TopupRatio          AccessPolicyRatio     `json:"topup_ratio" gorm:"not null"`
	RequestLimit        int                   `json:"request_limit" gorm:"not null;default:0"`
	SuccessRequestLimit int                   `json:"success_request_limit" gorm:"not null;default:0"`
	CreatedAt           int64                 `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           int64                 `json:"updated_at" gorm:"autoUpdateTime"`
	RouteGroups         []UserLevelRouteGroup `json:"route_groups,omitempty" gorm:"foreignKey:UserLevelCode;references:Code"`
}

func (UserLevel) TableName() string { return "user_levels" }

type RouteGroup struct {
	Code        string            `json:"code" gorm:"type:varchar(64);primaryKey"`
	Name        string            `json:"name" gorm:"type:varchar(128);not null"`
	Description string            `json:"description" gorm:"type:varchar(255);default:''"`
	BaseRatio   AccessPolicyRatio `json:"base_ratio" gorm:"not null"`
	Enabled     bool              `json:"enabled" gorm:"not null;index"`
	CreatedAt   int64             `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64             `json:"updated_at" gorm:"autoUpdateTime"`
}

func (RouteGroup) TableName() string { return "route_groups" }

type UserLevelRouteGroup struct {
	UserLevelCode  string             `json:"user_level_code" gorm:"type:varchar(64);primaryKey;index"`
	RouteGroupCode string             `json:"route_group_code" gorm:"type:varchar(64);primaryKey;index"`
	PriceRatio     *AccessPolicyRatio `json:"price_ratio"`
	CreatedAt      int64              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      int64              `json:"updated_at" gorm:"autoUpdateTime"`
}

func (UserLevelRouteGroup) TableName() string { return "user_level_route_groups" }

type AccessPolicyState struct {
	ID                           int    `json:"id" gorm:"primaryKey"`
	InitialRouteGroupCodes       string `json:"initial_route_group_codes" gorm:"type:text;not null"`
	InitialRouteGroupFingerprint string `json:"initial_route_group_fingerprint" gorm:"type:varchar(64);not null"`
	ExpandedAt                   int64  `json:"expanded_at" gorm:"not null"`
	InitialUserLevelCleanupAt    int64  `json:"initial_user_level_cleanup_at" gorm:"not null;default:0"`
	ContractedAt                 int64  `json:"contracted_at" gorm:"not null;default:0"`
}

func (AccessPolicyState) TableName() string { return "access_policy_states" }

// AccessPolicyMigrationGuard is armed by the production dry-run tool before
// the expand migration. Regular installations do not need a guard, but when a
// pending row exists the migration must match both the reviewed route-group
// code set and the reviewed legacy database state.
type AccessPolicyMigrationGuard struct {
	ID                      int    `json:"id" gorm:"primaryKey"`
	ExpectedRouteGroupCodes string `json:"expected_route_group_codes" gorm:"type:text;not null"`
	ExpectedFingerprint     string `json:"expected_fingerprint" gorm:"type:varchar(64);not null"`
	ExpectedLegacyState     string `json:"-" gorm:"type:text;not null"`
	ArmedAt                 int64  `json:"armed_at" gorm:"not null"`
	AppliedAt               int64  `json:"applied_at" gorm:"not null;default:0"`
}

func (AccessPolicyMigrationGuard) TableName() string { return "access_policy_migration_guards" }

const postgresLegacyAccessPolicyStateSQL = `
SELECT jsonb_build_object(
    'options', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.key COLLATE "C")
        FROM (
            SELECT key, value FROM options
            WHERE key IN (
                'GroupRatio', 'UserUsableGroups', 'TopupGroupRatio',
                'ModelRequestRateLimitGroup', 'GroupGroupRatio',
                'group_ratio_setting.group_special_usable_group'
            ) ORDER BY key COLLATE "C"
        ) AS item
    ), '[]'::jsonb),
    'channels', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (SELECT id, "group" AS group_name FROM channels ORDER BY id) AS item
    ), '[]'::jsonb),
    'abilities', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.channel_id, item.group_name COLLATE "C", item.model COLLATE "C", item.enabled, item.priority, item.weight)
        FROM (
            SELECT channel_id, "group" AS group_name, model, enabled, priority, weight
            FROM abilities ORDER BY channel_id, "group" COLLATE "C", model COLLATE "C", enabled, priority, weight
        ) AS item
    ), '[]'::jsonb),
    'tokens', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, "group" AS group_name, group_chain
            FROM tokens WHERE deleted_at IS NULL ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'routes', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.group_name COLLATE "C", item.model COLLATE "C")
        FROM (
            SELECT "group" AS group_name, model, tiers, updated_at
            FROM group_model_routes ORDER BY "group" COLLATE "C", model COLLATE "C"
        ) AS item
    ), '[]'::jsonb),
    'users', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, "group" AS group_name FROM users
            WHERE deleted_at IS NULL ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'subscription_plans', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, upgrade_group, downgrade_group
            FROM subscription_plans ORDER BY id
        ) AS item
    ), '[]'::jsonb),
    'user_subscriptions', COALESCE((
        SELECT jsonb_agg(to_jsonb(item) ORDER BY item.id)
        FROM (
            SELECT id, upgrade_group, downgrade_group, prev_user_group
            FROM user_subscriptions ORDER BY id
        ) AS item
    ), '[]'::jsonb)
)::text`

func LegacyGroupForUserLevel(code string) string {
	if strings.TrimSpace(code) == StandardUserLevelCode {
		return LegacyDefaultGroupCode
	}
	return strings.TrimSpace(code)
}

func UserLevelForLegacyGroup(code string) string {
	code = strings.TrimSpace(code)
	if code == "" || code == LegacyDefaultGroupCode {
		return StandardUserLevelCode
	}
	return code
}

func ListUserLevels() ([]UserLevel, error) {
	var levels []UserLevel
	err := DB.Preload("RouteGroups").Order("is_default DESC").Order("code ASC").Find(&levels).Error
	return levels, err
}

func ListRouteGroups() ([]RouteGroup, error) {
	var groups []RouteGroup
	err := DB.Order("code ASC").Find(&groups).Error
	return groups, err
}

func RefreshAccessPolicyCaches() error {
	InitOptionMap()
	if err := RebuildAccessPolicySnapshot(); err != nil {
		return err
	}
	NotifyAccessPolicyChanged()
	return nil
}

func LockAccessPolicyState(tx *gorm.DB) error {
	if tx == nil {
		return errors.New("transaction is required")
	}
	var state AccessPolicyState
	return lockForUpdate(tx).Where("id = ?", 1).First(&state).Error
}

type AccessPolicyReference struct {
	Kind  string `json:"kind"`
	Count int64  `json:"count"`
}

func UserLevelReferences(code string) ([]AccessPolicyReference, error) {
	return InspectUserLevelReferences(DB, code)
}

func InspectUserLevelReferences(db *gorm.DB, code string) ([]AccessPolicyReference, error) {
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	checks := []struct {
		kind  string
		model interface{}
		query string
		args  []interface{}
	}{
		{"users", &User{}, "user_level = ?", []interface{}{code}},
		{"subscription_plans", &SubscriptionPlan{}, "upgrade_user_level = ? OR downgrade_user_level = ?", []interface{}{code, code}},
		{"user_subscriptions", &UserSubscription{}, "upgrade_user_level = ? OR downgrade_user_level = ? OR previous_user_level = ?", []interface{}{code, code, code}},
	}
	result := make([]AccessPolicyReference, 0, len(checks))
	for _, check := range checks {
		if !db.Migrator().HasTable(check.model) {
			continue
		}
		var count int64
		if err := db.Model(check.model).Where(check.query, check.args...).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			result = append(result, AccessPolicyReference{Kind: check.kind, Count: count})
		}
	}
	return result, nil
}

func RouteGroupReferences(code string) ([]AccessPolicyReference, error) {
	return InspectRouteGroupReferences(DB, code)
}

func InspectRouteGroupReferences(db *gorm.DB, code string) ([]AccessPolicyReference, error) {
	if db == nil {
		return nil, errors.New("database is not initialized")
	}
	result := make([]AccessPolicyReference, 0)
	appendCount := func(kind string, count int64) {
		if count > 0 {
			result = append(result, AccessPolicyReference{Kind: kind, Count: count})
		}
	}
	exactChecks := []struct {
		kind  string
		model interface{}
	}{
		{"abilities", &Ability{}},
		{"explicit_routes", &GroupModelRoute{}},
		{"level_grants", &UserLevelRouteGroup{}},
	}
	for _, check := range exactChecks {
		var count int64
		column := commonGroupCol
		if check.kind == "level_grants" {
			column = "route_group_code"
		}
		if err := db.Model(check.model).Where(column+" = ?", code).Count(&count).Error; err != nil {
			return nil, err
		}
		appendCount(check.kind, count)
	}

	var channels []struct {
		ID    int
		Group string
	}
	if err := db.Model(&Channel{}).Select("id", commonGroupCol).Scan(&channels).Error; err != nil {
		return nil, err
	}
	var channelCount int64
	for _, channel := range channels {
		for _, group := range splitRouteGroupCodes(channel.Group) {
			if group == code {
				channelCount++
				break
			}
		}
	}
	appendCount("channels", channelCount)

	var tokens []struct {
		ID         int
		Group      string
		GroupChain StringArray
	}
	if err := db.Model(&Token{}).Select("id", commonGroupCol, "group_chain").Scan(&tokens).Error; err != nil {
		return nil, err
	}
	var tokenCount int64
	for _, token := range tokens {
		found := token.Group == code
		if !found {
			for _, group := range token.GroupChain {
				if group == code {
					found = true
					break
				}
			}
		}
		if found {
			tokenCount++
		}
	}
	appendCount("tokens", tokenCount)

	// OpenLux bindings resolve their local group through their bound channels;
	// channels already block deletion. This separate count makes the reason clear.
	if db.Migrator().HasTable(&OpenLuxPriceSyncBinding{}) {
		var bindingCount int64
		if err := db.Table("openlux_price_sync_bindings AS b").
			Joins("JOIN channels AS c ON c.id = b.channel_id").
			Where("c."+commonGroupCol+" = ?", code).Count(&bindingCount).Error; err == nil {
			appendCount("openlux_bindings", bindingCount)
		}
	}
	return result, nil
}

func AccessPolicyFingerprintCodes() ([]string, string, error) {
	var codes []string
	if err := DB.Model(&RouteGroup{}).Where("code <> ?", LegacyDefaultGroupCode).Order("code ASC").Pluck("code", &codes).Error; err != nil {
		return nil, "", err
	}
	encoded, err := common.Marshal(codes)
	if err != nil {
		return nil, "", err
	}
	return codes, fmt.Sprintf("%x", sha256.Sum256(encoded)), nil
}

func accessPolicyCodesFingerprint(codes []string) (string, error) {
	encoded, err := common.Marshal(codes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(encoded)), nil
}

func normalizedJSONFingerprint(raw string) (string, error) {
	var value interface{}
	if err := common.Unmarshal([]byte(raw), &value); err != nil {
		return "", err
	}
	encoded, err := common.Marshal(value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(encoded)), nil
}

func lockPendingAccessPolicyMigrationGuard(tx *gorm.DB) (*AccessPolicyMigrationGuard, error) {
	if !tx.Migrator().HasTable(&AccessPolicyMigrationGuard{}) {
		return nil, nil
	}
	var guard AccessPolicyMigrationGuard
	err := lockForUpdate(tx).Where("id = ? AND applied_at = 0", 1).First(&guard).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		if err := tx.Exec(`LOCK TABLE options, channels, abilities, tokens, users,
subscription_plans, user_subscriptions, group_model_routes IN SHARE ROW EXCLUSIVE MODE`).Error; err != nil {
			return nil, fmt.Errorf("failed to lock access-policy migration inputs: %w", err)
		}
	}
	return &guard, nil
}

func verifyAccessPolicyMigrationGuard(tx *gorm.DB, guard *AccessPolicyMigrationGuard, routeCodes []string) error {
	if guard == nil {
		return nil
	}
	standardRouteCodes := make([]string, 0, len(routeCodes))
	for _, code := range routeCodes {
		if code != LegacyDefaultGroupCode {
			standardRouteCodes = append(standardRouteCodes, code)
		}
	}
	var expectedCodes []string
	if err := common.Unmarshal([]byte(guard.ExpectedRouteGroupCodes), &expectedCodes); err != nil {
		return fmt.Errorf("invalid access-policy migration guard code list: %w", err)
	}
	if strings.Join(expectedCodes, "\x00") != strings.Join(standardRouteCodes, "\x00") {
		return fmt.Errorf("access-policy migration guard mismatch: route-group code set changed")
	}
	fingerprint, err := accessPolicyCodesFingerprint(standardRouteCodes)
	if err != nil {
		return err
	}
	if fingerprint != guard.ExpectedFingerprint {
		return fmt.Errorf("access-policy migration guard mismatch: route-group fingerprint changed")
	}
	if !common.UsingMainDatabase(common.DatabaseTypePostgreSQL) || strings.TrimSpace(guard.ExpectedLegacyState) == "" {
		return nil
	}
	var currentState string
	if err := tx.Raw(postgresLegacyAccessPolicyStateSQL).Scan(&currentState).Error; err != nil {
		return fmt.Errorf("failed to read guarded access-policy state: %w", err)
	}
	expectedStateFingerprint, err := normalizedJSONFingerprint(guard.ExpectedLegacyState)
	if err != nil {
		return fmt.Errorf("invalid guarded access-policy state: %w", err)
	}
	currentStateFingerprint, err := normalizedJSONFingerprint(currentState)
	if err != nil {
		return fmt.Errorf("invalid current access-policy state: %w", err)
	}
	if expectedStateFingerprint != currentStateFingerprint {
		return fmt.Errorf("access-policy migration guard mismatch: legacy routing state changed")
	}
	return nil
}

func GetDefaultUserLevelCode(tx *gorm.DB) (string, error) {
	if tx == nil {
		tx = DB
	}
	if !tx.Migrator().HasTable(&UserLevel{}) {
		return StandardUserLevelCode, nil
	}
	var level UserLevel
	err := tx.Where("is_default = ? AND enabled = ?", true, true).Order("code ASC").First(&level).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return StandardUserLevelCode, nil
	}
	return level.Code, err
}

func optionValue(tx *gorm.DB, key string) (string, error) {
	var option Option
	err := tx.Where(commonKeyCol+" = ?", key).First(&option).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return option.Value, err
}

func decodeOptionMap[T any](tx *gorm.DB, key string) (T, error) {
	var result T
	value, err := optionValue(tx, key)
	if err != nil || strings.TrimSpace(value) == "" {
		return result, err
	}
	if err := common.Unmarshal([]byte(value), &result); err != nil {
		return result, fmt.Errorf("invalid legacy option %s: %w", key, err)
	}
	return result, nil
}

func validateLegacyAccessPolicyOptions(
	groupRatios map[string]float64,
	usableGroups map[string]string,
	topupRatios map[string]float64,
	limitGroups map[string][2]int,
	priceOverrides map[string]map[string]float64,
	specialUsable map[string]map[string]string,
) error {
	validateCode := func(option, code string) error {
		if strings.TrimSpace(code) == "" || code == "auto" {
			return fmt.Errorf("invalid legacy option %s: empty or reserved code %q", option, code)
		}
		return nil
	}
	for code, ratio := range groupRatios {
		if err := validateCode("GroupRatio", code); err != nil {
			return err
		}
		if ratio < 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			return fmt.Errorf("invalid legacy option GroupRatio: %s has a negative or non-finite ratio", code)
		}
	}
	for code := range usableGroups {
		if err := validateCode("UserUsableGroups", code); err != nil {
			return err
		}
	}
	for code, ratio := range topupRatios {
		if err := validateCode("TopupGroupRatio", code); err != nil {
			return err
		}
		if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			return fmt.Errorf("invalid legacy option TopupGroupRatio: %s must have a positive finite ratio", code)
		}
	}
	for code, limits := range limitGroups {
		if err := validateCode("ModelRequestRateLimitGroup", code); err != nil {
			return err
		}
		if limits[0] < 0 || limits[1] < 0 {
			return fmt.Errorf("invalid legacy option ModelRequestRateLimitGroup: %s contains a negative limit", code)
		}
	}
	for levelCode, overrides := range priceOverrides {
		if err := validateCode("GroupGroupRatio", levelCode); err != nil {
			return err
		}
		for routeCode, ratio := range overrides {
			if err := validateCode("GroupGroupRatio", routeCode); err != nil {
				return err
			}
			if ratio < 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
				return fmt.Errorf("invalid legacy option GroupGroupRatio: %s/%s has a negative or non-finite ratio", levelCode, routeCode)
			}
		}
	}
	for levelCode, rules := range specialUsable {
		if err := validateCode("group_ratio_setting.group_special_usable_group", levelCode); err != nil {
			return err
		}
		for rawRouteCode := range rules {
			routeCode := strings.TrimPrefix(strings.TrimPrefix(rawRouteCode, "+:"), "-:")
			if err := validateCode("group_ratio_setting.group_special_usable_group", routeCode); err != nil {
				return err
			}
		}
	}
	return nil
}

func splitRouteGroupCodes(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		code := strings.TrimSpace(part)
		if code != "" && code != "auto" {
			result = append(result, code)
		}
	}
	return result
}

func collectLegacyRouteGroupCodes(
	tx *gorm.DB,
	ratios map[string]float64,
	usable map[string]string,
	priceOverrides map[string]map[string]float64,
	specialUsable map[string]map[string]string,
) ([]string, error) {
	codes := make(map[string]struct{}, len(ratios)+len(usable))
	for code := range ratios {
		if strings.TrimSpace(code) != "" && code != "auto" {
			codes[code] = struct{}{}
		}
	}
	for code := range usable {
		if strings.TrimSpace(code) != "" && code != "auto" {
			codes[code] = struct{}{}
		}
	}
	for _, overrides := range priceOverrides {
		for code := range overrides {
			if strings.TrimSpace(code) != "" && code != "auto" {
				codes[code] = struct{}{}
			}
		}
	}
	for _, rules := range specialUsable {
		for rawCode := range rules {
			if strings.HasPrefix(rawCode, "-:") {
				continue
			}
			code := strings.TrimPrefix(rawCode, "+:")
			if strings.TrimSpace(code) != "" && code != "auto" {
				codes[code] = struct{}{}
			}
		}
	}

	var values []string
	queries := []struct {
		model  interface{}
		column string
	}{
		{&Channel{}, "group"},
		{&Ability{}, "group"},
		{&Token{}, "group"},
		{&GroupModelRoute{}, "group"},
	}
	for _, query := range queries {
		values = nil
		if err := tx.Model(query.model).Distinct(query.column).Pluck(query.column, &values).Error; err != nil {
			return nil, err
		}
		for _, value := range values {
			for _, code := range splitRouteGroupCodes(value) {
				codes[code] = struct{}{}
			}
		}
	}

	var tokenChains []string
	if err := tx.Model(&Token{}).Where("group_chain <> ''").Pluck("group_chain", &tokenChains).Error; err != nil {
		return nil, err
	}
	for _, raw := range tokenChains {
		var chain []string
		if err := common.Unmarshal([]byte(raw), &chain); err != nil {
			return nil, fmt.Errorf("invalid token group_chain during access-policy migration: %w", err)
		}
		for _, code := range chain {
			code = strings.TrimSpace(code)
			if code != "" && code != "auto" {
				codes[code] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(codes))
	for code := range codes {
		result = append(result, code)
	}
	sort.Strings(result)
	return result, nil
}

func collectReferencedLegacyUserLevelCodes(tx *gorm.DB) (map[string]struct{}, error) {
	levelCodes := map[string]struct{}{StandardUserLevelCode: {}}
	if tx.Migrator().HasColumn(&User{}, "group") {
		var legacyUserGroups []string
		if err := tx.Model(&User{}).Where("deleted_at IS NULL").Distinct(commonGroupCol).Pluck(commonGroupCol, &legacyUserGroups).Error; err != nil {
			return nil, err
		}
		for _, code := range legacyUserGroups {
			levelCodes[UserLevelForLegacyGroup(code)] = struct{}{}
		}
	}

	legacySubscriptionColumns := []struct {
		model  interface{}
		column string
	}{
		{&SubscriptionPlan{}, "upgrade_group"},
		{&SubscriptionPlan{}, "downgrade_group"},
		{&UserSubscription{}, "upgrade_group"},
		{&UserSubscription{}, "downgrade_group"},
		{&UserSubscription{}, "prev_user_group"},
	}
	for _, source := range legacySubscriptionColumns {
		if !tx.Migrator().HasTable(source.model) || !tx.Migrator().HasColumn(source.model, source.column) {
			continue
		}
		var codes []string
		if err := tx.Model(source.model).Where(source.column+" <> ''").Distinct(source.column).Pluck(source.column, &codes).Error; err != nil {
			return nil, err
		}
		for _, code := range codes {
			levelCodes[UserLevelForLegacyGroup(code)] = struct{}{}
		}
	}
	return levelCodes, nil
}

func validateReferencedLegacyUserLevelConfigs(
	levelCodes map[string]struct{},
	topupRatios map[string]float64,
	limitGroups map[string][2]int,
	priceOverrides map[string]map[string]float64,
	specialUsable map[string]map[string]string,
) error {
	isReferenced := func(legacyCode string) bool {
		_, ok := levelCodes[UserLevelForLegacyGroup(legacyCode)]
		return ok
	}
	for code, ratio := range topupRatios {
		if !isReferenced(code) && ratio != 1 {
			return fmt.Errorf("ambiguous legacy user policy TopupGroupRatio.%s has no user or subscription reference", code)
		}
	}
	for code, limits := range limitGroups {
		if !isReferenced(code) && (limits[0] != 0 || limits[1] != 0) {
			return fmt.Errorf("ambiguous legacy user policy ModelRequestRateLimitGroup.%s has no user or subscription reference", code)
		}
	}
	for code, overrides := range priceOverrides {
		if !isReferenced(code) && len(overrides) != 0 {
			return fmt.Errorf("ambiguous legacy user policy GroupGroupRatio.%s has no user or subscription reference", code)
		}
	}
	for code, rules := range specialUsable {
		if !isReferenced(code) && len(rules) != 0 {
			return fmt.Errorf("ambiguous legacy user policy group_special_usable_group.%s has no user or subscription reference", code)
		}
	}
	return nil
}

func cleanupInitialUnreferencedUserLevels(tx *gorm.DB, state AccessPolicyState) error {
	if state.ExpandedAt <= 0 || state.InitialUserLevelCleanupAt > 0 {
		return nil
	}
	var levels []UserLevel
	if err := tx.Where(
		"code <> ? AND created_at BETWEEN ? AND ? AND updated_at = created_at",
		StandardUserLevelCode, state.ExpandedAt-5, state.ExpandedAt,
	).Find(&levels).Error; err != nil {
		return err
	}
	for _, level := range levels {
		references, err := InspectUserLevelReferences(tx, level.Code)
		if err != nil {
			return err
		}
		if len(references) != 0 {
			continue
		}
		if err := tx.Where("user_level_code = ?", level.Code).Delete(&UserLevelRouteGroup{}).Error; err != nil {
			return err
		}
		if err := tx.Where("code = ? AND updated_at = created_at", level.Code).Delete(&UserLevel{}).Error; err != nil {
			return err
		}
	}
	return tx.Model(&AccessPolicyState{}).
		Where("id = ? AND initial_user_level_cleanup_at = 0", state.ID).
		Update("initial_user_level_cleanup_at", time.Now().Unix()).Error
}

// MigrateLegacyAccessPolicy performs the expand-stage backfill. It is safe to
// run repeatedly: existing new-schema records remain authoritative, while only
// empty compatibility fields and missing records are populated.
func MigrateLegacyAccessPolicy() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var existingState AccessPolicyState
		existingStateErr := lockForUpdate(tx).Where("id = ?", 1).First(&existingState).Error
		if existingStateErr == nil {
			return nil
		}
		if !errors.Is(existingStateErr, gorm.ErrRecordNotFound) {
			return existingStateErr
		}

		migrationGuard, err := lockPendingAccessPolicyMigrationGuard(tx)
		if err != nil {
			return err
		}
		groupRatios, err := decodeOptionMap[map[string]float64](tx, "GroupRatio")
		if err != nil {
			return err
		}
		if groupRatios == nil {
			groupRatios = map[string]float64{LegacyDefaultGroupCode: 1}
		}
		usableGroups, err := decodeOptionMap[map[string]string](tx, "UserUsableGroups")
		if err != nil {
			return err
		}
		topupRatios, err := decodeOptionMap[map[string]float64](tx, "TopupGroupRatio")
		if err != nil {
			return err
		}
		limitGroups, err := decodeOptionMap[map[string][2]int](tx, "ModelRequestRateLimitGroup")
		if err != nil {
			return err
		}
		priceOverrides, err := decodeOptionMap[map[string]map[string]float64](tx, "GroupGroupRatio")
		if err != nil {
			return err
		}
		specialUsable, err := decodeOptionMap[map[string]map[string]string](tx, "group_ratio_setting.group_special_usable_group")
		if err != nil {
			return err
		}
		if err := validateLegacyAccessPolicyOptions(groupRatios, usableGroups, topupRatios, limitGroups, priceOverrides, specialUsable); err != nil {
			return err
		}

		routeCodes, err := collectLegacyRouteGroupCodes(
			tx, groupRatios, usableGroups, priceOverrides, specialUsable,
		)
		if err != nil {
			return err
		}
		if err := verifyAccessPolicyMigrationGuard(tx, migrationGuard, routeCodes); err != nil {
			return err
		}
		for _, code := range routeCodes {
			ratio, hasRatio := groupRatios[code]
			if !hasRatio {
				ratio = 1
			}
			name := strings.TrimSpace(usableGroups[code])
			if name == "" {
				name = code
			}
			enabled := true
			if code == LegacyDefaultGroupCode {
				var count int64
				if err := tx.Model(&Ability{}).Where(commonGroupCol+" = ? AND enabled = ?", code, true).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return fmt.Errorf("access-policy migration blocked: default still has %d enabled abilities", count)
				}
				enabled = false
			}
			group := RouteGroup{Code: code, Name: name, BaseRatio: AccessPolicyRatio(ratio), Enabled: enabled}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&group).Error; err != nil {
				return err
			}
		}

		var migrationState AccessPolicyState
		stateErr := tx.Where("id = ?", 1).First(&migrationState).Error
		if stateErr != nil && !errors.Is(stateErr, gorm.ErrRecordNotFound) {
			return stateErr
		}
		isFirstMigration := errors.Is(stateErr, gorm.ErrRecordNotFound)

		levelCodes, err := collectReferencedLegacyUserLevelCodes(tx)
		if err != nil {
			return err
		}
		if isFirstMigration {
			if err := validateReferencedLegacyUserLevelConfigs(
				levelCodes, topupRatios, limitGroups, priceOverrides, specialUsable,
			); err != nil {
				return err
			}
		}

		orderedLevels := make([]string, 0, len(levelCodes))
		for code := range levelCodes {
			orderedLevels = append(orderedLevels, code)
		}
		sort.Strings(orderedLevels)
		for _, code := range orderedLevels {
			legacyCode := LegacyGroupForUserLevel(code)
			name := code
			if code == StandardUserLevelCode {
				name = "普通用户"
			}
			topup := topupRatios[legacyCode]
			if topup == 0 {
				topup = 1
			}
			limits := limitGroups[legacyCode]
			level := UserLevel{
				Code: code, Name: name, Enabled: true,
				IsDefault: code == StandardUserLevelCode, TopupRatio: AccessPolicyRatio(topup),
				RequestLimit: limits[0], SuccessRequestLimit: limits[1],
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&level).Error; err != nil {
				return err
			}
		}

		if tx.Migrator().HasColumn(&User{}, "group") {
			var users []struct {
				ID    int
				Group string
			}
			if err := tx.Model(&User{}).Select("id", commonGroupCol).Where("user_level = '' OR user_level IS NULL").Scan(&users).Error; err != nil {
				return err
			}
			for _, user := range users {
				if err := tx.Model(&User{}).Where("id = ?", user.ID).Update("user_level", UserLevelForLegacyGroup(user.Group)).Error; err != nil {
					return err
				}
			}
		}

		// Standard receives the exact migration-time set of every existing
		// non-default route group. New groups are never added by later reruns.
		if isFirstMigration {
			if err := tx.Where("user_level_code = ?", StandardUserLevelCode).Delete(&UserLevelRouteGroup{}).Error; err != nil {
				return err
			}
			initialCodes := make([]string, 0, len(routeCodes))
			for _, routeCode := range routeCodes {
				if routeCode == LegacyDefaultGroupCode {
					continue
				}
				initialCodes = append(initialCodes, routeCode)
				ratio, hasRatio := priceOverrides[LegacyDefaultGroupCode][routeCode]
				var override *AccessPolicyRatio
				if hasRatio {
					value := AccessPolicyRatio(ratio)
					override = &value
				}
				grant := UserLevelRouteGroup{UserLevelCode: StandardUserLevelCode, RouteGroupCode: routeCode, PriceRatio: override}
				if err := tx.Create(&grant).Error; err != nil {
					return err
				}
			}
			encodedCodes, err := common.Marshal(initialCodes)
			if err != nil {
				return err
			}
			expandedAt := time.Now().Unix()
			migrationState = AccessPolicyState{
				ID: 1, InitialRouteGroupCodes: string(encodedCodes),
				InitialRouteGroupFingerprint: fmt.Sprintf("%x", sha256.Sum256(encodedCodes)),
				ExpandedAt:                   expandedAt,
				InitialUserLevelCleanupAt:    expandedAt,
			}
			if !legacyAccessPolicyColumnsPresent(tx) {
				migrationState.ContractedAt = expandedAt
			}
			if err := tx.Create(&migrationState).Error; err != nil {
				return err
			}
		}
		for _, levelCode := range orderedLevels {
			if !isFirstMigration {
				break
			}
			if levelCode == StandardUserLevelCode {
				continue
			}
			var count int64
			if err := tx.Model(&UserLevelRouteGroup{}).Where("user_level_code = ?", levelCode).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			legacyCode := LegacyGroupForUserLevel(levelCode)
			allowed := make(map[string]bool, len(usableGroups)+1)
			for code := range usableGroups {
				allowed[code] = true
			}
			allowed[legacyCode] = true
			for key := range specialUsable[legacyCode] {
				switch {
				case strings.HasPrefix(key, "-:"):
					delete(allowed, strings.TrimPrefix(key, "-:"))
				case strings.HasPrefix(key, "+:"):
					allowed[strings.TrimPrefix(key, "+:")] = true
				default:
					allowed[key] = true
				}
			}
			for routeCode := range allowed {
				if routeCode == LegacyDefaultGroupCode {
					continue
				}
				var routeGroupCount int64
				if err := tx.Model(&RouteGroup{}).Where("code = ?", routeCode).Count(&routeGroupCount).Error; err != nil {
					return err
				}
				if routeGroupCount == 0 {
					continue
				}
				ratio, hasRatio := priceOverrides[legacyCode][routeCode]
				var override *AccessPolicyRatio
				if hasRatio {
					value := AccessPolicyRatio(ratio)
					override = &value
				}
				grant := UserLevelRouteGroup{UserLevelCode: levelCode, RouteGroupCode: routeCode, PriceRatio: override}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&grant).Error; err != nil {
					return err
				}
			}
		}

		if err := backfillSubscriptionUserLevels(tx); err != nil {
			return err
		}
		if !isFirstMigration {
			if err := cleanupInitialUnreferencedUserLevels(tx, migrationState); err != nil {
				return err
			}
		}
		if migrationGuard != nil {
			if err := tx.Model(&AccessPolicyMigrationGuard{}).Where("id = ? AND applied_at = 0", migrationGuard.ID).
				Update("applied_at", time.Now().Unix()).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func backfillSubscriptionUserLevels(tx *gorm.DB) error {
	if tx.Migrator().HasTable(&SubscriptionPlan{}) {
		if tx.Migrator().HasColumn(&SubscriptionPlan{}, "upgrade_group") {
			if err := tx.Exec(`UPDATE subscription_plans SET upgrade_user_level = CASE WHEN upgrade_group = 'default' THEN 'standard' ELSE upgrade_group END WHERE (upgrade_user_level = '' OR upgrade_user_level IS NULL) AND upgrade_group <> ''`).Error; err != nil {
				return err
			}
		}
		if tx.Migrator().HasColumn(&SubscriptionPlan{}, "downgrade_group") {
			if err := tx.Exec(`UPDATE subscription_plans SET downgrade_user_level = CASE WHEN downgrade_group = 'default' THEN 'standard' ELSE downgrade_group END WHERE (downgrade_user_level = '' OR downgrade_user_level IS NULL) AND downgrade_group <> ''`).Error; err != nil {
				return err
			}
		}
	}
	if tx.Migrator().HasTable(&UserSubscription{}) {
		updates := []struct {
			column string
			query  string
		}{
			{"upgrade_group", `UPDATE user_subscriptions SET upgrade_user_level = CASE WHEN upgrade_group = 'default' THEN 'standard' ELSE upgrade_group END WHERE (upgrade_user_level = '' OR upgrade_user_level IS NULL) AND upgrade_group <> ''`},
			{"downgrade_group", `UPDATE user_subscriptions SET downgrade_user_level = CASE WHEN downgrade_group = 'default' THEN 'standard' ELSE downgrade_group END WHERE (downgrade_user_level = '' OR downgrade_user_level IS NULL) AND downgrade_group <> ''`},
			{"prev_user_group", `UPDATE user_subscriptions SET previous_user_level = CASE WHEN prev_user_group = 'default' THEN 'standard' ELSE prev_user_group END WHERE (previous_user_level = '' OR previous_user_level IS NULL) AND prev_user_group <> ''`},
		}
		for _, update := range updates {
			if !tx.Migrator().HasColumn(&UserSubscription{}, update.column) {
				continue
			}
			if err := tx.Exec(update.query).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func legacyAccessPolicyColumnsPresent(tx *gorm.DB) bool {
	return tx.Migrator().HasColumn(&User{}, "group") ||
		tx.Migrator().HasColumn(&SubscriptionPlan{}, "upgrade_group") ||
		tx.Migrator().HasColumn(&SubscriptionPlan{}, "downgrade_group") ||
		tx.Migrator().HasColumn(&UserSubscription{}, "upgrade_group") ||
		tx.Migrator().HasColumn(&UserSubscription{}, "downgrade_group") ||
		tx.Migrator().HasColumn(&UserSubscription{}, "prev_user_group")
}
