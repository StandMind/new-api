package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTaskModel2UserDtoOmitsChannelAndSanitizesInternalFailure(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyLanguageResolved, true)
	common.SetContextKey(ctx, constant.ContextKeyLanguage, i18n.LangEn)
	task := &model.Task{
		ID:         1,
		TaskID:     "task_public_1",
		Platform:   constant.TaskPlatform("video"),
		Group:      "primary",
		ChannelId:  99403,
		Status:     model.TaskStatusFailure,
		FailReason: "Failed to get channel info, channel ID: 99403",
	}

	userTask := TaskModel2UserDto(ctx, task)
	encoded, err := common.Marshal(userTask)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "channel_id")
	require.NotContains(t, string(encoded), "99403")
	require.NotContains(t, string(encoded), "channel")
	require.Empty(t, userTask.ResultURL)
	require.Equal(t, i18n.Translate(i18n.LangEn, i18n.MsgRelayTaskUnavailable), userTask.FailReason)
	require.Equal(t, "Failed to get channel info, channel ID: 99403", task.FailReason)

	adminTask := TaskModel2Dto(task)
	require.Equal(t, 99403, adminTask.ChannelId)
	require.Equal(t, task.FailReason, adminTask.FailReason)
}

func TestTaskModel2UserDtoPreservesUpstreamFailure(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)
	task := &model.Task{FailReason: "provider queue is full", ChannelId: 99403}

	userTask := TaskModel2UserDto(ctx, task)

	require.Equal(t, "provider queue is full", userTask.FailReason)
}
