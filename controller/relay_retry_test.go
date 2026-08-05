package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestShouldRetryTaskRelayRequiresExplicitSafeRejection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())

	tests := []struct {
		name      string
		taskError *dto.TaskError
		retries   int
		want      bool
	}{
		{
			name: "explicit 429 rejection",
			taskError: &dto.TaskError{
				StatusCode: http.StatusTooManyRequests,
				RetrySafe:  true,
				Error:      errors.New("rejected"),
			},
			retries: 1,
			want:    true,
		},
		{
			name: "explicit retryable 5xx rejection",
			taskError: &dto.TaskError{
				StatusCode: http.StatusInternalServerError,
				RetrySafe:  true,
				Error:      errors.New("rejected"),
			},
			retries: 1,
			want:    true,
		},
		{
			name: "ambiguous network error",
			taskError: &dto.TaskError{
				StatusCode: http.StatusInternalServerError,
				Error:      errors.New("timeout"),
			},
			retries: 1,
			want:    false,
		},
		{
			name: "accepted response parse error",
			taskError: &dto.TaskError{
				StatusCode: http.StatusInternalServerError,
				Error:      errors.New("missing task id"),
			},
			retries: 1,
			want:    false,
		},
		{
			name: "local error",
			taskError: &dto.TaskError{
				StatusCode: http.StatusInternalServerError,
				LocalError: true,
				RetrySafe:  true,
				Error:      errors.New("local"),
			},
			retries: 1,
			want:    false,
		},
		{
			name: "retry budget exhausted",
			taskError: &dto.TaskError{
				StatusCode: http.StatusTooManyRequests,
				RetrySafe:  true,
				Error:      errors.New("rejected"),
			},
			retries: 0,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, shouldRetryTaskRelay(context, test.taskError, test.retries))
		})
	}
}

func TestShouldRetryKeepsChannelFailuresInsideTheRouteChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	channelError := types.NewError(
		errors.New("invalid channel model mapping"),
		types.ErrorCodeChannelModelMappedError,
		types.ErrOptionWithSkipRetry(),
	)

	assert.True(t, shouldRetry(context, channelError, 1))
	assert.Equal(t, "channel_error", getRelayRetryDecision(context, channelError, 1).Reason)

	context.Set("specific_channel_id", "1")
	assert.False(t, shouldRetry(context, channelError, 1))
	assert.Equal(t, "specific_channel", getRelayRetryDecision(context, channelError, 1).Reason)
}

func TestShouldRetryTaskRelayStopsForSpecifiedChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set("specific_channel_id", "locked-task-channel")
	taskError := &dto.TaskError{
		StatusCode: http.StatusTooManyRequests,
		RetrySafe:  true,
		Error:      errors.New("rejected"),
	}

	assert.False(t, shouldRetryTaskRelay(context, taskError, 1))
}

func TestRelayRetryStopsWhenClientRequestIsCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil).
		WithContext(requestContext)

	channelError := types.NewError(
		errors.New("upstream request failed"),
		types.ErrorCodeChannelModelMappedError,
	)
	taskError := &dto.TaskError{
		StatusCode: http.StatusTooManyRequests,
		RetrySafe:  true,
		Error:      errors.New("upstream request failed"),
	}

	assert.False(t, shouldRetry(ginContext, channelError, 1))
	assert.False(t, shouldRetryTaskRelay(ginContext, taskError, 1))
	assert.Equal(t, "client_context_canceled", getRelayRetryDecision(ginContext, channelError, 1).Reason)
	assert.Equal(t, "client_context_canceled", getTaskRelayRetryDecision(ginContext, taskError, 1).Reason)
}
