package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGenerateTextOtherInfoIncludesResolutionBillingSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	now := time.Now()
	priceData := types.PriceData{}
	priceData.AddOtherRatio("image_resolution:4K", 1.79)
	info := &relaycommon.RelayInfo{
		StartTime:         now,
		FirstResponseTime: now,
		PriceData:         priceData,
		ChannelMeta:       &relaycommon.ChannelMeta{},
	}

	other := GenerateTextOtherInfo(ctx, info, 0, 1, 0, 0, 0, 0.33, 0)

	require.Equal(t, "4K", other["image_resolution"])
	require.Equal(t, 1.79, other["image_resolution_multiplier"])
	require.Equal(t, map[string]float64{"image_resolution:4K": 1.79}, other["billing_ratios"])
}
