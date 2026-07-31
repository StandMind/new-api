package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
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
	CaptureRequestDetailFromContext(context)

	select {
	case queued := <-writer.failureQueue:
		assert.True(t, queued.failed)
		assert.Equal(t, "req-distributor-failure", queued.detail.RequestID)
		assert.Equal(t, "gpt-test", queued.detail.ModelName)
		assert.Equal(t, "vip", queued.detail.Group)
		assert.Equal(t, 503, queued.detail.StatusCode)
		assert.Equal(t, "model_not_found", queued.detail.ErrorCode)
	default:
		t.Fatal("expected distributor failure to be queued")
	}
}
