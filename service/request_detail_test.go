package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/request_detail_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldCaptureRequestDetailModes(t *testing.T) {
	tests := []struct {
		name   string
		mode   string
		failed bool
		want   bool
	}{
		{name: "all success", mode: request_detail_setting.ModeAll, want: true},
		{name: "all failure", mode: request_detail_setting.ModeAll, failed: true, want: true},
		{name: "failed success", mode: request_detail_setting.ModeFailed, want: false},
		{name: "failed failure", mode: request_detail_setting.ModeFailed, failed: true, want: true},
		{name: "none success", mode: request_detail_setting.ModeNone, want: false},
		{name: "none failure", mode: request_detail_setting.ModeNone, failed: true, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, shouldCaptureRequestDetail(test.mode, test.failed))
		})
	}
}

func TestParseRequestDetailBodyRedactsCredentials(t *testing.T) {
	body, err := parseRequestDetailBody("application/json", []byte(`{
			"api_key":"secret-key",
			"token":"secret-token",
			"max_tokens":128,
		"messages":[{"role":"user","content":"hello"}],
		"nested":{"password":"secret-password"}
	}`))
	require.NoError(t, err)
	parsed, ok := body.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "[REDACTED]", parsed["api_key"])
	assert.Equal(t, "[REDACTED]", parsed["token"])
	assert.Equal(t, float64(128), parsed["max_tokens"])
	nested, ok := parsed["nested"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "[REDACTED]", nested["password"])
	assert.NotEmpty(t, parsed["messages"])
}

func TestBoundedRequestDetailStringKeepsValidUTF8(t *testing.T) {
	assert.Equal(t, "中文", boundedRequestDetailString("中文内容", 7))
}

func TestRequestDetailRoutingSnapshotOnlyForFinalFailure(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	plan := NewRouteAttemptPlan([]string{"primary", "fallback"}, true)
	SetRouteAttemptPlan(context, plan)

	assert.Nil(t, requestDetailRoutingSnapshot(context, "primary", false))
	routing := requestDetailRoutingSnapshot(context, "fallback", true)
	require.NotNil(t, routing)
	assert.Equal(t, RouteModeManual, routing.Mode)
	assert.Equal(t, []string{"primary", "fallback"}, routing.Groups)
	assert.Equal(t, "fallback", routing.FinalGroup)
}

func TestCaptureRequestDetailFromContextQueuesDistributorFailure(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(
		"POST",
		"/v1/chat/completions",
		strings.NewReader(`{"model":"gpt-test"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set(common.RequestIdKey, "req-distributor-failure")
	context.Set("id", 7)
	context.Set("username", "alice")
	common.SetContextKey(context, constant.ContextKeyOriginalModel, "gpt-test")
	common.SetContextKey(context, constant.ContextKeyUsingGroup, "vip")

	writer := &requestDetailWriter{
		failureQueue: make(chan *queuedRequestDetail, 1),
		successQueue: make(chan *queuedRequestDetail, 1),
	}
	previousWriter := activeRequestDetailWriter.Swap(writer)
	t.Cleanup(func() {
		activeRequestDetailWriter.Store(previousWriter)
	})

	SetRequestDetailFailure(context, &RequestDetailFailure{
		StatusCode: 503,
		ErrorType:  "new_api_error",
		ErrorCode:  "model_not_found",
		Message:    "no available channel",
	})
	SetRequestDetailResponse(context, &RequestDetailResponseSnapshot{
		StatusCode:  503,
		ContentType: "application/json",
		BodySize:    36,
		Body:        []byte(`{"error":{"message":"unavailable"}}`),
	})
	CaptureRequestDetailFromContext(context)

	select {
	case queued := <-writer.failureQueue:
		assert.True(t, queued.failed)
		assert.Equal(t, "req-distributor-failure", queued.detail.RequestID)
		assert.Equal(t, "gpt-test", queued.detail.ModelName)
		assert.Equal(t, "vip", queued.detail.Group)
		assert.Equal(t, 503, queued.detail.StatusCode)
		assert.Equal(t, "model_not_found", queued.detail.ErrorCode)
		require.NotNil(t, queued.response)
		assert.Equal(t, 503, queued.response.StatusCode)
		assert.JSONEq(t, `{"error":{"message":"unavailable"}}`, string(queued.response.Body))
	default:
		t.Fatal("expected distributor failure to be queued")
	}
}

func TestPrepareRequestDetailStoresCompleteSanitizedResponse(t *testing.T) {
	queued := &queuedRequestDetail{
		detail: &model.RequestDetail{
			RequestID: "req-response",
			Outcome:   model.RequestDetailOutcomeFailed,
		},
		response: &RequestDetailResponseSnapshot{
			StatusCode:  429,
			ContentType: "application/json; charset=utf-8",
			BodySize:    183,
			Body: []byte(`{
				"id":"response-id",
				"choices":[{"index":0,"message":{"role":"assistant","content":"full output"}}],
				"usage":{"input_tokens":12,"output_tokens":34},
				"api_key":"must-not-be-stored"
			}`),
		},
	}

	detail, err := prepareRequestDetail(queued)
	require.NoError(t, err)
	payload, err := DecodeRequestDetailPayload(detail.Payload)
	require.NoError(t, err)
	require.NotNil(t, payload.Response)
	assert.Equal(t, 429, payload.Response.StatusCode)
	assert.Equal(t, int64(183), payload.Response.BodySize)

	body, ok := payload.Response.Body.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "response-id", body["id"])
	assert.Equal(t, "[REDACTED]", body["api_key"])
	assert.NotEmpty(t, body["choices"])
	assert.NotEmpty(t, body["usage"])
}
