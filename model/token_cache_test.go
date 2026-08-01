package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTokenCacheTest(t *testing.T) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	originalClient := common.RDB
	originalEnabled := common.RedisEnabled
	common.RDB = client
	common.RedisEnabled = true
	t.Cleanup(func() {
		common.RDB = originalClient
		common.RedisEnabled = originalEnabled
		require.NoError(t, client.Close())
	})
}

func TestTokenCacheRoundTripsExplicitGroupChain(t *testing.T) {
	setupTokenCacheTest(t)

	original := Token{
		Id:             91,
		UserId:         42,
		Key:            "group-chain-cache-key",
		Status:         common.TokenStatusEnabled,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
		Group:          "vip",
		GroupChain:     StringArray{"vip", "default"},
	}
	require.NoError(t, cacheSetToken(original))

	cached, err := cacheGetTokenByKey(original.Key)
	require.NoError(t, err)
	assert.Equal(t, original.GroupChain, cached.GroupChain)
	assert.Equal(t, original.Group, cached.Group)
}

func TestTokenCacheRejectsLegacyEntryWithoutGroupChain(t *testing.T) {
	setupTokenCacheTest(t)

	key := "legacy-cache-key"
	legacyEntry := struct {
		Id     int
		UserId int
		Status int
		Group  string
		Key    string
	}{
		Id:     92,
		UserId: 42,
		Status: common.TokenStatusEnabled,
		Group:  "default",
	}
	cacheKey := fmt.Sprintf("token:%s", common.GenerateHMAC(key))
	require.NoError(t, common.RedisHSetObj(cacheKey, &legacyEntry, time.Minute))

	cached, err := cacheGetTokenByKey(key)
	assert.Nil(t, cached)
	assert.ErrorContains(t, err, "missing explicit group chain")
}
