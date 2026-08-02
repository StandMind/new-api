package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCalculateAudioQuotaFromPriceDataFixedPriceWithoutTokenUsage(t *testing.T) {
	priceData := types.PriceData{
		ModelPrice: 0.3,
		UsePrice:   true,
		GroupRatioInfo: types.GroupRatioInfo{
			GroupRatio: 0.133835,
		},
	}

	quota, clamp := calculateAudioQuotaFromPriceData(
		priceData,
		"gpt-4o-mini-tts",
		TokenDetails{},
		TokenDetails{},
	)

	require.Equal(t, 20075, quota)
	require.Nil(t, clamp)
	require.False(t, missingMeteredUsage(0, priceData.UsePrice))
}

func TestMissingMeteredUsageOnlyAppliesToUsageBasedBilling(t *testing.T) {
	tests := []struct {
		name        string
		totalTokens int
		usePrice    bool
		want        bool
	}{
		{name: "metered without usage", totalTokens: 0, usePrice: false, want: true},
		{name: "fixed price without usage", totalTokens: 0, usePrice: true, want: false},
		{name: "metered with usage", totalTokens: 1, usePrice: false, want: false},
		{name: "fixed price with usage", totalTokens: 1, usePrice: true, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, missingMeteredUsage(test.totalTokens, test.usePrice))
		})
	}
}

func TestPostAudioConsumeQuotaRecordsFixedPriceWithoutTokenUsage(t *testing.T) {
	truncate(t)
	seedUser(t, 8201, 100_000)
	seedChannel(t, 8202)
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("token_name", "fixed-audio-test")
	relayInfo := &relaycommon.RelayInfo{
		UserId:                8201,
		OriginModelName:       "gpt-4o-mini-tts",
		StartTime:             time.Now(),
		UsingGroup:            "Azure-Gpt-2",
		IsPlayground:          true,
		FinalPreConsumedQuota: 20_075,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 8202,
		},
		PriceData: types.PriceData{
			ModelPrice: 0.3,
			UsePrice:   true,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 0.133835,
			},
		},
	}

	PostAudioConsumeQuota(ctx, relayInfo, &dto.Usage{}, "")

	var user model.User
	require.NoError(t, model.DB.First(&user, 8201).Error)
	require.Equal(t, 20_075, user.UsedQuota)
	require.Equal(t, 1, user.RequestCount)

	var channel model.Channel
	require.NoError(t, model.DB.First(&channel, 8202).Error)
	require.Equal(t, int64(20_075), channel.UsedQuota)

	var consumeLog model.Log
	require.NoError(t, model.LOG_DB.Where("model_name = ?", "gpt-4o-mini-tts").First(&consumeLog).Error)
	require.Equal(t, 20_075, consumeLog.Quota)
	require.Equal(t, "Azure-Gpt-2", consumeLog.Group)
}

func TestPostWssConsumeQuotaRecordsFixedPriceWithoutTokenUsage(t *testing.T) {
	truncate(t)
	seedUser(t, 8203, 100_000)
	seedChannel(t, 8204)
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("token_name", "fixed-wss-test")
	relayInfo := &relaycommon.RelayInfo{
		UserId:                8203,
		OriginModelName:       "fixed-realtime-model",
		StartTime:             time.Now(),
		UsingGroup:            "Openai-Gpt-1",
		IsPlayground:          true,
		FinalPreConsumedQuota: 20_075,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 8204,
		},
		PriceData: types.PriceData{
			ModelPrice: 0.3,
			UsePrice:   true,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 0.133835,
			},
		},
	}

	PostWssConsumeQuota(ctx, relayInfo, relayInfo.OriginModelName, &dto.RealtimeUsage{}, "")

	var user model.User
	require.NoError(t, model.DB.First(&user, 8203).Error)
	require.Equal(t, 20_075, user.UsedQuota)
	require.Equal(t, 1, user.RequestCount)

	var channel model.Channel
	require.NoError(t, model.DB.First(&channel, 8204).Error)
	require.Equal(t, int64(20_075), channel.UsedQuota)

	var consumeLog model.Log
	require.NoError(t, model.LOG_DB.Where("model_name = ?", "fixed-realtime-model").First(&consumeLog).Error)
	require.Equal(t, 20_075, consumeLog.Quota)
	require.Equal(t, "Openai-Gpt-1", consumeLog.Group)
}
