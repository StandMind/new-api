package model

import (
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGroupModelRoutePersistsAcrossDatabaseReopen(t *testing.T) {
	originalDB := DB
	originalLogDB := LOG_DB
	originalMainType := common.MainDatabaseType()
	originalLogType := common.LogDatabaseType()
	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		common.SetDatabaseTypes(originalMainType, originalLogType)
		initCol()
	})

	databasePath := filepath.Join(t.TempDir(), "group-model-route.db")
	openDatabase := func() *gorm.DB {
		db, err := gorm.Open(sqlite.Open(databasePath+"?_busy_timeout=30000"), &gorm.Config{})
		require.NoError(t, err)
		return db
	}

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	initCol()
	DB = openDatabase()
	require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &GroupModelRoute{}))

	priority := int64(10)
	weight := uint(100)
	channel := Channel{
		Id:       7201,
		Name:     "persistent-route-channel",
		Key:      "sk-persistent-route",
		Status:   common.ChannelStatusEnabled,
		Priority: &priority,
		Weight:   &weight,
	}
	require.NoError(t, DB.Create(&channel).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "persistent-group",
		Model:     "persistent-model",
		ChannelId: channel.Id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
	require.NoError(t, SaveGroupModelRoute(&GroupModelRoute{
		Group: "persistent-group",
		Model: "persistent-model",
		Tiers: GroupModelRouteTiers{{
			Priority: 100,
			Channels: []GroupModelRouteChannel{{
				ChannelID: channel.Id,
				Weight:    75,
			}},
		}},
	}))

	firstSQLDB, err := DB.DB()
	require.NoError(t, err)
	require.NoError(t, firstSQLDB.Close())

	DB = openDatabase()
	reopenedSQLDB, err := DB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = reopenedSQLDB.Close()
	})

	route, explicit, err := GetGroupModelRoute("persistent-group", "persistent-model")
	require.NoError(t, err)
	require.True(t, explicit)
	require.NotNil(t, route)
	assert.Equal(t, GroupModelRouteTiers{{
		Priority: 100,
		Channels: []GroupModelRouteChannel{{
			ChannelID: channel.Id,
			Weight:    75,
		}},
	}}, route.Tiers)

	plans, err := LoadGroupModelRoutePlans(
		[]string{"persistent-group"},
		"persistent-model",
		"/v1/chat/completions",
	)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.True(t, plans[0].Explicit)
	assert.Equal(t, "persistent-model", plans[0].RouteModel)
	require.Len(t, plans[0].Candidates, 1)
	assert.Equal(t, channel.Id, plans[0].Candidates[0].ChannelID)
}
