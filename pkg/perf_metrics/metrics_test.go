package perfmetrics

import (
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupAttemptMetricsRecordOneResultPerEnteredGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	startedAt := time.Unix(100, 0)

	previous, recorded := beginGroupAttempt(context, "routing-model", "group-a", startedAt)
	assert.False(t, recorded)
	assert.Empty(t, previous.Model)

	previous, recorded = beginGroupAttempt(context, "routing-model", "group-a", startedAt.Add(2*time.Second))
	assert.False(t, recorded, "retrying another channel in the same group must not emit a group result")

	previous, recorded = beginGroupAttempt(context, "routing-model", "group-b", startedAt.Add(5*time.Second))
	require.True(t, recorded)
	assert.Equal(t, Sample{
		Model: "routing-model", Group: "group-a", LatencyMs: 5000,
	}, previous)

	tracker, ok := context.Get(routeGroupMetricTrackerKey)
	require.True(t, ok)
	state, ok := tracker.(*routeGroupMetricTracker)
	require.True(t, ok)
	state.mutex.Lock()
	state.startedAt = time.Now().Add(-20 * time.Millisecond)
	state.mutex.Unlock()

	info := &relaycommon.RelayInfo{
		OriginModelName: "wrong-fallback-model",
		UsingGroup:      "wrong-fallback-group",
		StartTime:       time.Now().Add(-time.Second),
	}
	final, recorded := prepareRelaySample(context, info, true, 17)
	require.True(t, recorded)
	assert.Equal(t, "routing-model", final.Model)
	assert.Equal(t, "group-b", final.Group)
	assert.True(t, final.Success)
	assert.Equal(t, int64(17), final.OutputTokens)
	assert.GreaterOrEqual(t, final.LatencyMs, int64(0))

	_, recorded = prepareRelaySample(context, info, true, 17)
	assert.False(t, recorded, "the successful group must be finalized only once")
}
