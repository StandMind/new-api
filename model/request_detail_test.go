package model

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func withRequestDetailTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RequestDetail{}))

	previousLogDB := LOG_DB
	previousLogType := common.LogDatabaseType()
	LOG_DB = db
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		LOG_DB = previousLogDB
		common.SetLogDatabaseType(previousLogType)
	})
	return db
}

func TestRequestDetailListOmitsPayloadAndDetailReturnsIt(t *testing.T) {
	withRequestDetailTestDB(t)
	require.NoError(t, CreateRequestDetails([]*RequestDetail{{
		RequestID:    "req-list",
		CreatedAt:    100,
		Username:     "alice",
		ModelName:    "gpt-test",
		Group:        "vip",
		Outcome:      RequestDetailOutcomeFailed,
		Payload:      []byte("compressed"),
		StorageBytes: 1024,
	}}))

	details, total, err := GetRequestDetails(RequestDetailFilter{Limit: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, details, 1)
	assert.Equal(t, "vip", details[0].Group)
	assert.True(t, details[0].HasPayload)
	assert.Empty(t, details[0].Payload)

	detail, err := GetRequestDetailByRequestID("req-list")
	require.NoError(t, err)
	assert.Equal(t, []byte("compressed"), detail.Payload)
}

func TestRequestDetailCleanupUsesTimeAndCapacityAndPreservesFailuresFirst(t *testing.T) {
	withRequestDetailTestDB(t)
	require.NoError(t, CreateRequestDetails([]*RequestDetail{
		{RequestID: "expired", CreatedAt: 10, Outcome: RequestDetailOutcomeFailed, StorageBytes: 100},
		{RequestID: "old-failure", CreatedAt: 100, Outcome: RequestDetailOutcomeFailed, StorageBytes: 100},
		{RequestID: "old-success", CreatedAt: 110, Outcome: RequestDetailOutcomeSuccess, StorageBytes: 100},
		{RequestID: "new-success", CreatedAt: 120, Outcome: RequestDetailOutcomeSuccess, StorageBytes: 100},
	}))

	result, err := CleanupRequestDetails(context.Background(), 50, 150, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.DeletedCount)
	assert.Equal(t, int64(300), result.FreedBytes)

	remaining, total, err := GetRequestDetails(RequestDetailFilter{Limit: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, remaining, 1)
	assert.Equal(t, "old-failure", remaining[0].RequestID)
}
