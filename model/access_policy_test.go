package model

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAccessPolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	originalDB := DB
	originalLogDB := LOG_DB
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	DB = db
	LOG_DB = db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	initCol()
	require.NoError(t, db.AutoMigrate(
		&Option{}, &User{}, &Channel{}, &Ability{}, &Token{}, &GroupModelRoute{},
		&SubscriptionPlan{}, &UserSubscription{},
		&UserLevel{}, &RouteGroup{}, &UserLevelRouteGroup{}, &AccessPolicyState{},
		&AccessPolicyMigrationGuard{},
	))
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
		DB = originalDB
		LOG_DB = originalLogDB
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		initCol()
	})
	return db
}

func writeAccessPolicyTestOption(t *testing.T, db *gorm.DB, key string, value interface{}) {
	t.Helper()
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, db.Create(&Option{Key: key, Value: string(encoded)}).Error)
}

func addLegacyAccessPolicyColumns(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		"ALTER TABLE users ADD COLUMN `group` varchar(64) DEFAULT 'default'",
		"ALTER TABLE subscription_plans ADD COLUMN upgrade_group varchar(64) DEFAULT ''",
		"ALTER TABLE subscription_plans ADD COLUMN downgrade_group varchar(64) DEFAULT ''",
		"ALTER TABLE user_subscriptions ADD COLUMN upgrade_group varchar(64) DEFAULT ''",
		"ALTER TABLE user_subscriptions ADD COLUMN downgrade_group varchar(64) DEFAULT ''",
		"ALTER TABLE user_subscriptions ADD COLUMN prev_user_group varchar(64) DEFAULT ''",
	}
	for _, statement := range statements {
		require.NoError(t, db.Exec(statement).Error)
	}
}

func TestAccessPolicyAutoMigrateIsIdempotentOnSQLite(t *testing.T) {
	db := setupAccessPolicyTestDB(t)
	override := AccessPolicyRatio(0.8)
	require.NoError(t, db.Create(&UserLevel{
		Code: "standard", Name: "Standard", IsDefault: true, Enabled: true, TopupRatio: 1.25,
	}).Error)
	require.NoError(t, db.Create(&RouteGroup{
		Code: "route-a", Name: "Route A", BaseRatio: 1.5, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&UserLevelRouteGroup{
		UserLevelCode: "standard", RouteGroupCode: "route-a", PriceRatio: &override,
	}).Error)

	for range 2 {
		require.NoError(t, db.AutoMigrate(&UserLevel{}, &RouteGroup{}, &UserLevelRouteGroup{}))
	}

	var level UserLevel
	require.NoError(t, db.First(&level, "code = ?", "standard").Error)
	assert.InDelta(t, 1.25, float64(level.TopupRatio), 0.0000001)
	var route RouteGroup
	require.NoError(t, db.First(&route, "code = ?", "route-a").Error)
	assert.InDelta(t, 1.5, float64(route.BaseRatio), 0.0000001)
	var grant UserLevelRouteGroup
	require.NoError(t, db.First(&grant, "user_level_code = ? AND route_group_code = ?", "standard", "route-a").Error)
	require.NotNil(t, grant.PriceRatio)
	assert.InDelta(t, 0.8, float64(*grant.PriceRatio), 0.0000001)
}

func TestMigrateLegacyAccessPolicyFreezesStandardRouteGrants(t *testing.T) {
	db := setupAccessPolicyTestDB(t)
	addLegacyAccessPolicyColumns(t, db)
	writeAccessPolicyTestOption(t, db, "GroupRatio", map[string]float64{
		"default": 1.3,
		"free":    0,
		"paid":    2,
	})
	writeAccessPolicyTestOption(t, db, "UserUsableGroups", map[string]string{"free": "Free route"})
	writeAccessPolicyTestOption(t, db, "TopupGroupRatio", map[string]float64{"default": 1.2, "vip": 0.9, "paid": 1})
	writeAccessPolicyTestOption(t, db, "ModelRequestRateLimitGroup", map[string][2]int{"default": {20, 10}})
	writeAccessPolicyTestOption(t, db, "GroupGroupRatio", map[string]map[string]float64{
		"default": {"free": 0.75, "price-only": 0.8},
		"vip":     {"paid": 1.5},
	})
	writeAccessPolicyTestOption(t, db, "group_ratio_setting.group_special_usable_group", map[string]map[string]string{
		"vip": {"+:paid": "Paid route", "+:special-only": "Special route", "-:stale-only": "Stale route"},
	})

	require.NoError(t, db.Create(&[]User{
		{Id: 1, Username: "standard-user", AffCode: "std1", Status: common.UserStatusEnabled},
		{Id: 2, Username: "vip-user", AffCode: "vip2", Status: common.UserStatusEnabled},
	}).Error)
	require.NoError(t, db.Exec("UPDATE users SET `group` = 'vip' WHERE id = 2").Error)
	require.NoError(t, db.Create(&Channel{Id: 10, Name: "channel-only", Group: "channel-only", Status: common.ChannelStatusManuallyDisabled}).Error)
	require.NoError(t, db.Create(&Ability{Group: "ability-only", Model: "m", ChannelId: 10, Enabled: false}).Error)
	require.NoError(t, db.Create(&Token{Id: 20, UserId: 1, Name: "chain", Key: "chain-key", Group: "free", GroupChain: StringArray{"free", "token-only"}}).Error)
	require.NoError(t, db.Create(&GroupModelRoute{Group: "route-only", Model: "m", Tiers: GroupModelRouteTiers{}}).Error)
	plan := SubscriptionPlan{Title: "Legacy level", Currency: "USD", DurationUnit: "month", DurationValue: 1}
	require.NoError(t, db.Create(&plan).Error)
	require.NoError(t, db.Exec("UPDATE subscription_plans SET upgrade_group = 'plan-only' WHERE id = ?", plan.Id).Error)

	require.NoError(t, MigrateLegacyAccessPolicy())

	var state AccessPolicyState
	require.NoError(t, db.First(&state, "id = ?", 1).Error)
	var initialCodes []string
	require.NoError(t, common.Unmarshal([]byte(state.InitialRouteGroupCodes), &initialCodes))
	expectedCodes := []string{"ability-only", "channel-only", "free", "paid", "price-only", "route-only", "special-only", "token-only"}
	require.Equal(t, expectedCodes, initialCodes)
	expectedCodesJSON := mustAccessPolicyJSON(t, expectedCodes)
	require.Equal(t, state.InitialRouteGroupFingerprint, fmt.Sprintf("%x", sha256.Sum256(expectedCodesJSON)))

	var grants []UserLevelRouteGroup
	require.NoError(t, db.Where("user_level_code = ?", StandardUserLevelCode).Order("route_group_code ASC").Find(&grants).Error)
	grantCodes := make([]string, 0, len(grants))
	for _, grant := range grants {
		grantCodes = append(grantCodes, grant.RouteGroupCode)
	}
	require.Equal(t, expectedCodes, grantCodes)

	var free RouteGroup
	require.NoError(t, db.First(&free, "code = ?", "free").Error)
	require.Zero(t, free.BaseRatio)
	var standard UserLevel
	require.NoError(t, db.First(&standard, "code = ?", StandardUserLevelCode).Error)
	require.Equal(t, "普通用户", standard.Name)
	require.InDelta(t, 1.2, float64(standard.TopupRatio), 0.0000001)
	require.Equal(t, 20, standard.RequestLimit)
	require.Equal(t, 10, standard.SuccessRequestLimit)
	var standardFree UserLevelRouteGroup
	require.NoError(t, db.First(&standardFree, "user_level_code = ? AND route_group_code = ?", StandardUserLevelCode, "free").Error)
	require.NotNil(t, standardFree.PriceRatio)
	require.InDelta(t, 0.75, float64(*standardFree.PriceRatio), 0.0000001)

	var users []User
	require.NoError(t, db.Order("id ASC").Find(&users).Error)
	require.Equal(t, StandardUserLevelCode, users[0].UserLevel)
	require.Equal(t, "vip", users[1].UserLevel)
	var planLevel UserLevel
	require.NoError(t, db.First(&planLevel, "code = ?", "plan-only").Error)
	var routeOnlyLevelCount int64
	require.NoError(t, db.Model(&UserLevel{}).Where("code = ?", "paid").Count(&routeOnlyLevelCount).Error)
	require.Zero(t, routeOnlyLevelCount)

	require.NoError(t, db.Create(&RouteGroup{Code: "later", Name: "Later", BaseRatio: 3, Enabled: true}).Error)
	require.NoError(t, MigrateLegacyAccessPolicy())
	var laterGrantCount int64
	require.NoError(t, db.Model(&UserLevelRouteGroup{}).
		Where("user_level_code = ? AND route_group_code = ?", StandardUserLevelCode, "later").
		Count(&laterGrantCount).Error)
	require.Zero(t, laterGrantCount)
	require.NoError(t, db.First(&state, "id = ?", 1).Error)
	require.Equal(t, string(expectedCodesJSON), state.InitialRouteGroupCodes)
}

func TestMigrateLegacyAccessPolicyBlocksOrphanNonNeutralLevelConfig(t *testing.T) {
	db := setupAccessPolicyTestDB(t)
	writeAccessPolicyTestOption(t, db, "GroupRatio", map[string]float64{"default": 1, "route-a": 1})
	writeAccessPolicyTestOption(t, db, "TopupGroupRatio", map[string]float64{"default": 1, "route-a": 1.1})

	err := MigrateLegacyAccessPolicy()
	require.ErrorContains(t, err, "ambiguous legacy user policy TopupGroupRatio.route-a")
	var stateCount int64
	require.NoError(t, db.Model(&AccessPolicyState{}).Count(&stateCount).Error)
	require.Zero(t, stateCount)
}

func TestMigrateLegacyAccessPolicyRerunKeepsNewSchemaAuthoritative(t *testing.T) {
	db := setupAccessPolicyTestDB(t)
	writeAccessPolicyTestOption(t, db, "GroupRatio", map[string]float64{"default": 1, "route-a": 1.3})
	require.NoError(t, MigrateLegacyAccessPolicy())

	var state AccessPolicyState
	require.NoError(t, db.First(&state, "id = ?", 1).Error)
	require.NoError(t, db.Create(&UserLevel{
		Code: "custom", Name: "Custom", Enabled: true, TopupRatio: 1,
	}).Error)
	require.NoError(t, db.Create(&UserLevelRouteGroup{
		UserLevelCode: "custom", RouteGroupCode: "route-a",
	}).Error)

	require.NoError(t, MigrateLegacyAccessPolicy())
	var levelCount int64
	require.NoError(t, db.Model(&UserLevel{}).Where("code = ?", "custom").Count(&levelCount).Error)
	require.EqualValues(t, 1, levelCount)
	var customGrantCount int64
	require.NoError(t, db.Model(&UserLevelRouteGroup{}).
		Where("user_level_code = ? AND route_group_code = ?", "custom", "route-a").
		Count(&customGrantCount).Error)
	require.EqualValues(t, 1, customGrantCount)
}

func mustAccessPolicyJSON(t *testing.T, value interface{}) []byte {
	t.Helper()
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	return encoded
}

func TestMigrateLegacyAccessPolicyBlocksEnabledDefaultAbility(t *testing.T) {
	db := setupAccessPolicyTestDB(t)
	writeAccessPolicyTestOption(t, db, "GroupRatio", map[string]float64{"default": 1, "route-a": 1})
	require.NoError(t, db.Create(&Ability{Group: "default", Model: "m", ChannelId: 1, Enabled: true}).Error)

	err := MigrateLegacyAccessPolicy()
	require.ErrorContains(t, err, "default still has 1 enabled abilities")
	var stateCount int64
	require.NoError(t, db.Model(&AccessPolicyState{}).Count(&stateCount).Error)
	require.Zero(t, stateCount)
	var levelCount int64
	require.NoError(t, db.Model(&UserLevel{}).Count(&levelCount).Error)
	require.Zero(t, levelCount)
}

func TestMigrateLegacyAccessPolicyHonorsPendingGuard(t *testing.T) {
	db := setupAccessPolicyTestDB(t)
	writeAccessPolicyTestOption(t, db, "GroupRatio", map[string]float64{
		"default": 1,
		"route-a": 1.25,
	})
	expectedCodes := []string{"route-a"}
	expectedJSON := mustAccessPolicyJSON(t, expectedCodes)
	fingerprint, err := accessPolicyCodesFingerprint(expectedCodes)
	require.NoError(t, err)
	require.NoError(t, db.Create(&AccessPolicyMigrationGuard{
		ID: 1, ExpectedRouteGroupCodes: string(expectedJSON),
		ExpectedFingerprint: fingerprint, ArmedAt: time.Now().Unix(),
	}).Error)

	require.NoError(t, MigrateLegacyAccessPolicy())
	var guard AccessPolicyMigrationGuard
	require.NoError(t, db.First(&guard, "id = ?", 1).Error)
	require.NotZero(t, guard.AppliedAt)
}

func TestMigrateLegacyAccessPolicyRejectsStaleGuard(t *testing.T) {
	db := setupAccessPolicyTestDB(t)
	writeAccessPolicyTestOption(t, db, "GroupRatio", map[string]float64{
		"default": 1,
		"route-a": 1.25,
	})
	expectedCodes := []string{"route-a"}
	fingerprint, err := accessPolicyCodesFingerprint(expectedCodes)
	require.NoError(t, err)
	require.NoError(t, db.Create(&AccessPolicyMigrationGuard{
		ID: 1, ExpectedRouteGroupCodes: string(mustAccessPolicyJSON(t, expectedCodes)),
		ExpectedFingerprint: fingerprint, ArmedAt: time.Now().Unix(),
	}).Error)

	writeAccessPolicyTestOption(t, db, "UserUsableGroups", map[string]string{"route-b": "Route B"})
	err = MigrateLegacyAccessPolicy()
	require.ErrorContains(t, err, "route-group code set changed")

	var levelCount int64
	require.NoError(t, db.Model(&UserLevel{}).Count(&levelCount).Error)
	require.Zero(t, levelCount)
	var guard AccessPolicyMigrationGuard
	require.NoError(t, db.First(&guard, "id = ?", 1).Error)
	require.Zero(t, guard.AppliedAt)
}

func TestPostgresGuardUsesCollationIndependentTextOrdering(t *testing.T) {
	require.Contains(t, postgresLegacyAccessPolicyStateSQL,
		`ORDER BY item.key COLLATE "C"`)
	require.Contains(t, postgresLegacyAccessPolicyStateSQL,
		`ORDER BY item.channel_id, item.group_name COLLATE "C", item.model COLLATE "C", item.enabled, item.priority, item.weight`)
	require.Contains(t, postgresLegacyAccessPolicyStateSQL,
		`ORDER BY channel_id, "group" COLLATE "C", model COLLATE "C", enabled, priority, weight`)
	require.Contains(t, postgresLegacyAccessPolicyStateSQL,
		`ORDER BY item.group_name COLLATE "C", item.model COLLATE "C"`)
}

func TestValidateLegacyAccessPolicyOptionsRejectsAmbiguousValues(t *testing.T) {
	tests := []struct {
		name   string
		groups map[string]float64
		topup  map[string]float64
		limits map[string][2]int
		prices map[string]map[string]float64
		rules  map[string]map[string]string
	}{
		{name: "negative route ratio", groups: map[string]float64{"route": -1}},
		{name: "zero topup ratio", topup: map[string]float64{"default": 0}},
		{name: "negative request limit", limits: map[string][2]int{"default": {-1, 0}}},
		{name: "negative level price", prices: map[string]map[string]float64{"default": {"route": -1}}},
		{name: "empty special route", rules: map[string]map[string]string{"default": {"+:": "bad"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateLegacyAccessPolicyOptions(test.groups, nil, test.topup, test.limits, test.prices, test.rules)
			require.Error(t, err)
		})
	}
}

func TestResolveAccessPolicyRatioPrecedence(t *testing.T) {
	db := setupAccessPolicyTestDB(t)
	require.NoError(t, db.Create(&[]UserLevel{
		{Code: "standard", Name: "Standard", IsDefault: true, Enabled: true, TopupRatio: 1},
		{Code: "basic", Name: "Basic", Enabled: true, TopupRatio: 1},
	}).Error)
	require.NoError(t, db.Create(&RouteGroup{Code: "route", Name: "Route", BaseRatio: 2, Enabled: true}).Error)
	override := AccessPolicyRatio(1.5)
	require.NoError(t, db.Create(&[]UserLevelRouteGroup{
		{UserLevelCode: "standard", RouteGroupCode: "route", PriceRatio: &override},
		{UserLevelCode: "basic", RouteGroupCode: "route"},
	}).Error)
	require.NoError(t, RebuildAccessPolicySnapshot())

	originalRatios := ratio_setting.GetGroupModelRatioCopy()
	t.Cleanup(func() {
		encoded, err := common.Marshal(originalRatios)
		require.NoError(t, err)
		require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(string(encoded)))
	})
	require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(`{"route":{"special":0.8}}`))

	ratio, source := ResolveAccessPolicyRatio("standard", "route", "special")
	assert.InDelta(t, 0.8, ratio, 0.0000001)
	assert.Equal(t, "group_model_ratio.exact", source)
	ratio, source = ResolveAccessPolicyRatio("standard", "route", "other")
	assert.InDelta(t, 1.5, ratio, 0.0000001)
	assert.Equal(t, "user_level_route_group.price_ratio", source)
	ratio, source = ResolveAccessPolicyRatio("basic", "route", "other")
	assert.InDelta(t, 2, ratio, 0.0000001)
	assert.Equal(t, "route_group.base_ratio", source)

	accessible := GetUserLevelRouteGroups("standard", false)
	require.Len(t, accessible, 1)
	require.Equal(t, "Route", accessible[0].Name)
	require.NoError(t, db.Model(&RouteGroup{}).Where("code = ?", "route").Update("enabled", false).Error)
	require.NoError(t, RebuildAccessPolicySnapshot())
	require.False(t, UserLevelCanAccessRouteGroup("standard", "route"))
	require.Empty(t, GetUserLevelRouteGroups("standard", false))
}
