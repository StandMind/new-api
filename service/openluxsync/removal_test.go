package openluxsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openLuxTestDB(t *testing.T, extraModels ...any) *gorm.DB {
	t.Helper()
	initializeOpenLuxTestColumns(t)
	databasePath := filepath.Join(t.TempDir(), "openlux-sync.db")
	db, err := gorm.Open(sqlite.Open(databasePath+"?_busy_timeout=30000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	models := []any{
		&model.Channel{},
		&model.Ability{},
		&model.GroupModelRoute{},
		&model.User{},
		&model.Token{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.Option{},
		&model.OpenLuxPriceSyncBinding{},
		&model.OpenLuxPriceSyncState{},
		&model.UserLevel{},
		&model.RouteGroup{},
		&model.UserLevelRouteGroup{},
		&model.AccessPolicyState{},
	}
	models = append(models, extraModels...)
	require.NoError(t, db.AutoMigrate(models...))
	require.NoError(t, db.Create(&model.AccessPolicyState{
		ID: 1, InitialRouteGroupCodes: "[]", InitialRouteGroupFingerprint: strings.Repeat("0", 64),
	}).Error)
	return db
}

func initializeOpenLuxTestColumns(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		model.DB = originalDB
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = filepath.Join(t.TempDir(), "initialize-columns.db")
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))
	require.NoError(t, model.InitDB())
	if model.DB != nil {
		sqlDB, err := model.DB.DB()
		if err == nil {
			require.NoError(t, sqlDB.Close())
		}
	}
}

func createReferenceChannel(t *testing.T, db *gorm.DB, name, group, models, baseURL string) model.Channel {
	t.Helper()
	channel := previewChannel(0, name, group, models, baseURL)
	require.NoError(t, db.Create(&channel).Error)
	return channel
}

func TestSourceGroupRemovalDeletesWholeGroupOnlyWithoutBusinessReferences(t *testing.T) {
	const (
		localGroup  = "openlux-only-local"
		sourceGroup = "deleted-source"
		modelName   = "openlux-sync-delete-group-model"
	)
	db := openLuxTestDB(t)
	require.NoError(t, db.Create(&model.RouteGroup{Code: localGroup, Name: localGroup, BaseRatio: 1, Enabled: true}).Error)
	bound := createReferenceChannel(t, db, "OpenLux/deleted-source", localGroup, modelName, "https://api.openlux.ai")
	require.NoError(t, db.Create(&model.Ability{
		ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true,
	}).Error)
	binding := resolvedBinding{
		SourceGroup: sourceGroup,
		LocalGroup:  localGroup,
		ChannelIDs:  []int{bound.Id},
		Healthy:     true,
	}

	change, err := buildSourceGroupRemoval(db, binding, &model.OpenLuxSyncSnapshot{}, false)
	require.NoError(t, err)
	assert.True(t, change.Actionable)
	assert.Equal(t, "delete_local_group", change.RemoveMode)
	assert.Len(t, change.mutations, 1)

	require.NoError(t, db.Create(&model.UserLevel{Code: "referencing-level", Name: "Referencing", Enabled: true, TopupRatio: 1}).Error)
	require.NoError(t, db.Create(&model.UserLevelRouteGroup{UserLevelCode: "referencing-level", RouteGroupCode: localGroup}).Error)

	user := model.User{Username: "openlux-ref-user", Password: "password", Group: localGroup}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Key: "openlux-ref-token", Name: "ref", Group: "auto", GroupChain: model.StringArray{"fallback", localGroup}}
	require.NoError(t, db.Create(&token).Error)
	plan := model.SubscriptionPlan{
		Title: "OpenLux ref plan", Currency: "USD", DurationUnit: "month", DurationValue: 1,
		UpgradeGroup: localGroup,
	}
	require.NoError(t, db.Create(&plan).Error)
	subscription := model.UserSubscription{
		UserId: user.Id, PlanId: plan.Id, Status: "expired", PrevUserGroup: localGroup,
	}
	require.NoError(t, db.Create(&subscription).Error)
	route := model.GroupModelRoute{
		Group: localGroup,
		Model: modelName,
		Tiers: model.GroupModelRouteTiers{{
			Priority: 0,
			Channels: []model.GroupModelRouteChannel{{ChannelID: bound.Id, Weight: 1}},
		}},
	}
	require.NoError(t, db.Create(&route).Error)
	require.NoError(t, db.Create(&model.Ability{
		ChannelId: bound.Id, Group: localGroup, Model: "orphan-model", Enabled: true,
	}).Error)

	blocked, err := buildSourceGroupRemoval(db, binding, &model.OpenLuxSyncSnapshot{}, false)
	require.NoError(t, err)
	assert.False(t, blocked.Actionable)
	reasons := strings.Join(blocked.BlockedReasons, " ")
	assert.Contains(t, reasons, "用户等级仍授权")
	assert.Contains(t, reasons, "Token 或分组链")
	assert.NotContains(t, reasons, "订阅计划")
	assert.NotContains(t, reasons, "订阅记录")
	assert.Contains(t, reasons, "显式模型路由")
	assert.Contains(t, reasons, "显式路由引用待删渠道")
	assert.Contains(t, reasons, localGroup+"/"+modelName)
	assert.Contains(t, reasons, "Ability")
}

func TestSourceGroupRemovalInMixedGroupOnlyDetachesOpenLuxChannels(t *testing.T) {
	const (
		localGroup  = "mixed-removal-local"
		sourceGroup = "deleted-source"
		modelName   = "openlux-sync-mixed-delete-model"
	)
	db := openLuxTestDB(t)
	bound := createReferenceChannel(t, db, "OpenLux/deleted-source", localGroup, modelName, "https://api.openlux.ai")
	selfHosted := createReferenceChannel(t, db, "self-hosted-pool", localGroup, modelName, "https://pool.internal.example")
	require.NoError(t, db.Create([]model.Ability{
		{ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true},
		{ChannelId: selfHosted.Id, Group: localGroup, Model: modelName, Enabled: true},
		{ChannelId: selfHosted.Id, Group: localGroup, Model: "self-hosted-orphan", Enabled: true},
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Username: "mixed-ref-user", Password: "password", Group: localGroup,
	}).Error)
	binding := resolvedBinding{
		SourceGroup:       sourceGroup,
		LocalGroup:        localGroup,
		ChannelIDs:        []int{bound.Id},
		Healthy:           true,
		MixedChannelCount: 1,
	}

	change, err := buildSourceGroupRemoval(db, binding, &model.OpenLuxSyncSnapshot{}, false)
	require.NoError(t, err)
	assert.True(t, change.Actionable)
	assert.Equal(t, "detach_openlux_only", change.RemoveMode)
	assert.Empty(t, change.mutations)
	assert.Equal(t, []int{bound.Id}, change.deleteChannelIDs)
	assert.Equal(t, []string{sourceGroup}, change.deleteBindings)

	boundOrphan := model.Ability{
		ChannelId: bound.Id, Group: localGroup, Model: "bound-orphan", Enabled: true,
	}
	require.NoError(t, db.Create(&boundOrphan).Error)
	blocked, err := buildSourceGroupRemoval(db, binding, &model.OpenLuxSyncSnapshot{}, false)
	require.NoError(t, err)
	assert.False(t, blocked.Actionable)
	assert.Contains(t, strings.Join(blocked.BlockedReasons, " "), "异常 Ability")
	require.NoError(t, db.Delete(&boundOrphan).Error)

	route := model.GroupModelRoute{
		Group: localGroup,
		Model: modelName,
		Tiers: model.GroupModelRouteTiers{{
			Priority: 0,
			Channels: []model.GroupModelRouteChannel{{ChannelID: bound.Id, Weight: 1}},
		}},
	}
	require.NoError(t, db.Create(&route).Error)
	blocked, err = buildSourceGroupRemoval(db, binding, &model.OpenLuxSyncSnapshot{}, false)
	require.NoError(t, err)
	assert.False(t, blocked.Actionable)
	reasons := strings.Join(blocked.BlockedReasons, " ")
	assert.Contains(t, reasons, "显式路由引用待删 OpenLux 渠道")
	assert.Contains(t, reasons, localGroup+"/"+modelName)
}
