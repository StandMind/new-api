package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type programmableRouteUpstream struct {
	mu    sync.Mutex
	mode  string
	calls []string
}

func (upstream *programmableRouteUpstream) reset(mode string) {
	upstream.mu.Lock()
	defer upstream.mu.Unlock()
	upstream.mode = mode
	upstream.calls = nil
}

func (upstream *programmableRouteUpstream) recordedCalls() []string {
	upstream.mu.Lock()
	defer upstream.mu.Unlock()
	return append([]string(nil), upstream.calls...)
}

func (upstream *programmableRouteUpstream) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	upstream.mu.Lock()
	upstream.calls = append(upstream.calls, request.URL.Path)
	mode := upstream.mode
	upstream.mu.Unlock()

	writer.Header().Set("Content-Type", "application/json")
	switch {
	case mode == "partial-stream" && strings.HasPrefix(request.URL.Path, "/channel-1/"):
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("data: {\"id\":\"chatcmpl-route-stream\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"route-integration-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"partial\"},\"finish_reason\":null}]}\n\n"))
		_, _ = writer.Write([]byte("data: {invalid-json}\n\n"))
	case mode == "retry-to-success" && strings.HasPrefix(request.URL.Path, "/channel-1/"):
		writer.WriteHeader(http.StatusTooManyRequests)
		_, _ = writer.Write([]byte(`{"error":{"message":"channel 1 busy"}}`))
	case mode == "retry-to-success" && strings.HasPrefix(request.URL.Path, "/channel-2/"):
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte(`{"error":{"message":"channel 2 unavailable"}}`))
	case mode == "bad-request" && strings.HasPrefix(request.URL.Path, "/channel-1/"):
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":{"message":"invalid request"}}`))
	default:
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{
            "id":"chatcmpl-route-test",
            "object":"chat.completion",
            "created":1,
            "model":"route-integration-model",
            "choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
            "usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
        }`))
	}
}

func TestRelayGroupRouteRetryBoundariesWithProgrammableUpstream(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.Token{},
		&model.GroupModelRoute{},
		&model.Log{},
		&model.UserSubscription{},
		&model.SubscriptionPreConsumeRecord{},
	))

	originalRetryTimes := common.RetryTimes
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalBatchUpdateEnabled := common.BatchUpdateEnabled
	originalLogConsumeEnabled := common.LogConsumeEnabled
	originalCountToken := constant.CountToken
	originalErrorLogEnabled := constant.ErrorLogEnabled
	originalFreeModelPreConsume := operation_setting.GetQuotaSetting().EnableFreeModelPreConsume
	originalRetryStatuses := operation_setting.AutomaticRetryStatusCodesToString()
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalGroupGroupRatio := ratio_setting.GroupGroupRatio2JSONString()
	originalGroupModelRatio := ratio_setting.GroupModelRatio2JSONString()
	t.Cleanup(func() {
		common.RetryTimes = originalRetryTimes
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.BatchUpdateEnabled = originalBatchUpdateEnabled
		common.LogConsumeEnabled = originalLogConsumeEnabled
		constant.CountToken = originalCountToken
		constant.ErrorLogEnabled = originalErrorLogEnabled
		operation_setting.GetQuotaSetting().EnableFreeModelPreConsume = originalFreeModelPreConsume
		require.NoError(t, operation_setting.AutomaticRetryStatusCodesFromString(originalRetryStatuses))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(originalModelRatio))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(originalGroupGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(originalGroupModelRatio))
	})

	common.RetryTimes = 0
	common.MemoryCacheEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true
	constant.CountToken = false
	constant.ErrorLogEnabled = false
	service.InitHttpClient()
	operation_setting.GetQuotaSetting().EnableFreeModelPreConsume = false
	require.NoError(t, operation_setting.AutomaticRetryStatusCodesFromString("429,500-503"))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"route-integration-model":1}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"group-a":0,"group-b":0}`))
	require.NoError(t, ratio_setting.UpdateGroupGroupRatioByJSONString(`{}`))
	require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(`{}`))

	require.NoError(t, db.Create(&model.User{
		Id:       99001,
		Username: "route-integration-user",
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    1_000_000,
	}).Error)

	upstream := &programmableRouteUpstream{}
	upstreamServer := httptest.NewServer(upstream)
	t.Cleanup(upstreamServer.Close)

	priority := int64(0)
	weight := uint(100)
	autoBan := 0
	channels := []model.Channel{
		{
			Id: 99101, Type: constant.ChannelTypeOpenAI, Key: "channel-1-key",
			Status: common.ChannelStatusEnabled, Name: "route-channel-1",
			BaseURL: common.GetPointer(upstreamServer.URL + "/channel-1"),
			Group:   "group-a", Models: "route-integration-model",
			Priority: &priority, Weight: &weight, AutoBan: &autoBan,
		},
		{
			Id: 99102, Type: constant.ChannelTypeOpenAI, Key: "channel-2-key",
			Status: common.ChannelStatusEnabled, Name: "route-channel-2",
			BaseURL: common.GetPointer(upstreamServer.URL + "/channel-2"),
			Group:   "group-a", Models: "route-integration-model",
			Priority: &priority, Weight: &weight, AutoBan: &autoBan,
		},
		{
			Id: 99103, Type: constant.ChannelTypeOpenAI, Key: "channel-3-key",
			Status: common.ChannelStatusEnabled, Name: "route-channel-3",
			BaseURL: common.GetPointer(upstreamServer.URL + "/channel-3"),
			Group:   "group-b", Models: "route-integration-model",
			Priority: &priority, Weight: &weight, AutoBan: &autoBan,
		},
	}
	require.NoError(t, db.Create(&channels).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "group-a", Model: "route-integration-model", ChannelId: 99101, Enabled: true, Priority: &priority, Weight: weight},
		{Group: "group-a", Model: "route-integration-model", ChannelId: 99102, Enabled: true, Priority: &priority, Weight: weight},
		{Group: "group-b", Model: "route-integration-model", ChannelId: 99103, Enabled: true, Priority: &priority, Weight: weight},
	}).Error)
	require.NoError(t, db.Create(&[]model.GroupModelRoute{
		{
			Group: "group-a", Model: "route-integration-model",
			Tiers: model.GroupModelRouteTiers{
				{Priority: 300, Channels: []model.GroupModelRouteChannel{{ChannelID: 99101, Weight: 100}}},
				{Priority: 200, Channels: []model.GroupModelRouteChannel{{ChannelID: 99102, Weight: 100}}},
			},
		},
		{
			Group: "group-b", Model: "route-integration-model",
			Tiers: model.GroupModelRouteTiers{
				{Priority: 100, Channels: []model.GroupModelRouteChannel{{ChannelID: 99103, Weight: 100}}},
			},
		},
	}).Error)

	engine := gin.New()
	engine.Use(middleware.BodyStorageCleanup())
	engine.Use(func(c *gin.Context) {
		groups := strings.Split(c.GetHeader("X-Test-Group-Chain"), ",")
		common.SetContextKey(c, constant.ContextKeyUserId, 99001)
		common.SetContextKey(c, constant.ContextKeyUserName, "route-integration-user")
		common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
		common.SetContextKey(c, constant.ContextKeyUserQuota, 1_000_000)
		common.SetContextKey(c, constant.ContextKeyTokenId, 0)
		common.SetContextKey(c, constant.ContextKeyTokenKey, "route-integration-token")
		common.SetContextKey(c, constant.ContextKeyTokenUnlimited, true)
		common.SetContextKey(c, constant.ContextKeyTokenGroup, groups[0])
		common.SetContextKey(c, constant.ContextKeyTokenGroupChain, groups)
		common.SetContextKey(c, constant.ContextKeyUsingGroup, groups[0])
		common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
		c.Set("token_name", "route-integration-token")
		c.Next()
	})
	engine.POST(
		"/v1/chat/completions",
		middleware.Distribute(),
		func(c *gin.Context) { Relay(c, types.RelayFormatOpenAI) },
	)

	performRequest := func(groupChain string, stream bool) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		requestBody := `{"model":"route-integration-model","messages":[{"role":"user","content":"hello"}]}`
		if stream {
			requestBody = `{"model":"route-integration-model","stream":true,"messages":[{"role":"user","content":"hello"}]}`
		}
		request := httptest.NewRequest(
			http.MethodPost,
			"/v1/chat/completions",
			strings.NewReader(requestBody),
		)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Test-Group-Chain", groupChain)
		engine.ServeHTTP(recorder, request)
		return recorder
	}

	t.Run("multi-group ignores numeric retry budget and reaches fallback group", func(t *testing.T) {
		upstream.reset("retry-to-success")
		recorder := performRequest("group-a,group-b", false)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		assert.Equal(t, []string{
			"/channel-1/v1/chat/completions",
			"/channel-2/v1/chat/completions",
			"/channel-3/v1/chat/completions",
		}, upstream.recordedCalls())
		assert.Contains(t, recorder.Body.String(), `"content":"ok"`)
	})

	t.Run("single group remains bounded by RetryTimes", func(t *testing.T) {
		upstream.reset("retry-to-success")
		recorder := performRequest("group-a", false)

		require.Equal(t, http.StatusTooManyRequests, recorder.Code, recorder.Body.String())
		assert.Equal(t, []string{"/channel-1/v1/chat/completions"}, upstream.recordedCalls())
	})

	t.Run("non-retry client error stops the multi-group chain", func(t *testing.T) {
		upstream.reset("bad-request")
		recorder := performRequest("group-a,group-b", false)

		require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
		assert.Equal(t, []string{"/channel-1/v1/chat/completions"}, upstream.recordedCalls())
	})

	t.Run("stream output prevents a second upstream call", func(t *testing.T) {
		upstream.reset("partial-stream")
		recorder := performRequest("group-a,group-b", true)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		assert.Equal(t, []string{"/channel-1/v1/chat/completions"}, upstream.recordedCalls())
		assert.Contains(t, recorder.Body.String(), "partial")
	})
}
