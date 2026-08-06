package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/types"
)

func newPublicErrorTestContext(language string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.Header.Set("Accept-Language", language)
	return c
}

func TestLocalErrorIsTranslatedWithoutMutatingCause(t *testing.T) {
	require.NoError(t, i18n.Init())
	c := newPublicErrorTestContext("zh-CN")
	cause := errors.New("database connection secret detail")
	err := types.NewErrorWithStatusCode(
		cause,
		types.ErrorCodeInvalidRequest,
		http.StatusBadRequest,
		types.ErrOptionWithPublicMessage(i18n.MsgDistributorTokenModelForbidden, map[string]any{"Model": "gpt-5"}),
	)

	response := OpenAIErrorForResponse(c, err, "req-1")
	assert.Equal(t, "该令牌无权访问模型 gpt-5 (request id: req-1)", response.Message)
	assert.Equal(t, cause.Error(), err.Error())
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
}

func TestUpstreamErrorPreservesProviderMessageAndAddsRequestIDOnCopy(t *testing.T) {
	c := newPublicErrorTestContext("fr")
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "上游提供商原始错误",
		Type:    "provider_error",
		Code:    "provider_code",
	}, http.StatusBadGateway)

	response := OpenAIErrorForResponse(c, err, "req-1")
	assert.Equal(t, "上游提供商原始错误 (request id: req-1)", response.Message)
	assert.Equal(t, "上游提供商原始错误", err.Error())
	assert.Equal(t, "provider_error", response.Type)
	assert.Equal(t, "provider_code", response.Code)
	assert.Equal(t, types.ErrorSourceUpstream, err.GetSource())
}

func TestPublicMessageArgumentsAreCopied(t *testing.T) {
	args := map[string]any{"Model": "before", "Ignored": []string{"large", "mutable"}}
	err := types.NewError(
		errors.New("internal"),
		types.ErrorCodeInvalidRequest,
		types.ErrOptionWithPublicMessage(i18n.MsgDistributorTokenModelForbidden, args),
	)
	args["Model"] = "after"

	stored := err.PublicMessageArgs()
	assert.Equal(t, "before", stored["Model"])
	assert.NotContains(t, stored, "Ignored")
}

func TestLocalTaskErrorIsTranslatedWithoutMutatingOriginal(t *testing.T) {
	c := newPublicErrorTestContext("zh-CN")
	taskErr := TaskErrorWrapperLocal(errors.New("internal pricing detail"), "model_price_error", http.StatusBadRequest)

	response := TaskErrorForResponse(c, taskErr)

	assert.Equal(t, "请求的模型没有有效的价格配置", response.Message)
	assert.Equal(t, "internal pricing detail", taskErr.Message)
	assert.Equal(t, "internal pricing detail", taskErr.Error.Error())
}

func TestUpstreamTaskErrorPreservesProviderMessage(t *testing.T) {
	c := newPublicErrorTestContext("fr")
	taskErr := TaskErrorWrapper(errors.New("供应商原始错误"), "provider_error", http.StatusBadGateway)

	response := TaskErrorForResponse(c, taskErr)

	assert.Equal(t, "供应商原始错误", response.Message)
}

func TestLocalTaskRateLimitLocalizesResponseCopy(t *testing.T) {
	c := newPublicErrorTestContext("fr")
	taskErr := &dto.TaskError{
		Code:       "provider_rate_limit",
		Message:    "provider rate limit",
		StatusCode: http.StatusTooManyRequests,
		LocalError: true,
		Error:      errors.New("provider rate limit"),
	}

	response := TaskErrorForResponse(c, taskErr)

	assert.Equal(t, "La charge en amont du groupe actuel est saturée. Veuillez réessayer plus tard.", response.Message)
	assert.Equal(t, "provider rate limit", taskErr.Message)
}

func TestUpstreamTaskRateLimitPreservesProviderMessage(t *testing.T) {
	c := newPublicErrorTestContext("fr")
	taskErr := &dto.TaskError{
		Code:       "provider_rate_limit",
		Message:    "原始上游限流消息",
		StatusCode: http.StatusTooManyRequests,
		Error:      errors.New("原始上游限流消息"),
	}

	response := TaskErrorForResponse(c, taskErr)

	assert.Equal(t, "原始上游限流消息", response.Message)
	assert.Equal(t, "原始上游限流消息", taskErr.Message)
}

func TestLocalMidjourneyErrorIsTranslatedAndUpstreamMessageIsPreserved(t *testing.T) {
	c := newPublicErrorTestContext("fr")
	local := MidjourneyErrorWrapper(4, "prompt_is_required")
	upstream := &dto.MidjourneyResponse{
		Code:        30,
		Description: "供应商队列已满",
		Result:      "provider detail",
	}

	assert.Equal(t, i18n.Translate(i18n.LangFr, i18n.MsgRelayInvalidRequest), MidjourneyErrorDescriptionForResponse(c, local))
	assert.Equal(t, "供应商队列已满 provider detail", MidjourneyErrorDescriptionForResponse(c, upstream))
	assert.Equal(t, "prompt_is_required", local.Description)
}

func TestTaskErrorFromAPIErrorPreservesSourceAndPublicMessage(t *testing.T) {
	local := types.NewOpenAIError(
		errors.New("internal"),
		types.ErrorCodeModelPriceError,
		http.StatusBadRequest,
		types.ErrOptionWithPublicMessage(i18n.MsgRelayModelPriceError),
	)
	localTask := TaskErrorFromAPIError(local)
	assert.True(t, localTask.LocalError)
	assert.Equal(t, i18n.MsgRelayModelPriceError, localTask.MessageKey)

	upstream := types.WithOpenAIError(types.OpenAIError{Message: "provider", Code: "provider"}, http.StatusBadGateway)
	upstreamTask := TaskErrorFromAPIError(upstream)
	assert.False(t, upstreamTask.LocalError)
	assert.Empty(t, upstreamTask.MessageKey)
}

func BenchmarkOpenAIErrorForResponse(b *testing.B) {
	c := newPublicErrorTestContext("fr-FR,fr;q=0.9,en;q=0.8")
	err := types.NewOpenAIError(
		errors.New("internal model access detail"),
		types.ErrorCodeInvalidRequest,
		http.StatusForbidden,
		types.ErrOptionWithPublicMessage(
			i18n.MsgDistributorTokenModelForbidden,
			map[string]any{"Model": "gpt-5"},
		),
	)

	b.ReportAllocs()
	for b.Loop() {
		response := OpenAIErrorForResponse(c, err, "req-benchmark")
		if response.Message == "" {
			b.Fatal("localized response message is empty")
		}
	}
}
