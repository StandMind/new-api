package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPrepareMigrationConnectionLimitsAndRestoresPool(t *testing.T) {
	t.Setenv("SQL_MAX_OPEN_CONNS", "7")
	t.Setenv("SQL_MAX_IDLE_CONNS", "3")

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	configureConnectionPool(sqlDB)
	require.Equal(t, 7, sqlDB.Stats().MaxOpenConnections)

	restore, err := prepareMigrationConnection(
		db,
		sqlDB,
		common.DatabaseTypeSQLite,
	)
	require.NoError(t, err)
	require.Equal(t, 1, sqlDB.Stats().MaxOpenConnections)

	require.NoError(t, restore())
	require.Equal(t, 7, sqlDB.Stats().MaxOpenConnections)
}
