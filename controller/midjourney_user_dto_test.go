package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserMidjourneyTaskDtoOmitsChannelAndSanitizesFailure(t *testing.T) {
	previousForward := setting.MjForwardUrlEnabled
	setting.MjForwardUrlEnabled = false
	t.Cleanup(func() { setting.MjForwardUrlEnabled = previousForward })

	ctx, _ := gin.CreateTestContext(nil)
	common.SetContextKey(ctx, constant.ContextKeyLanguageResolved, true)
	common.SetContextKey(ctx, constant.ContextKeyLanguage, i18n.LangZhCN)
	task := &model.Midjourney{
		Id:         1,
		MjId:       "mj_public_1",
		ChannelId:  99403,
		Status:     "FAILURE",
		FailReason: "获取渠道信息失败，请联系管理员，渠道ID：99403",
	}

	items := userMidjourneyTasksToDto(ctx, []*model.Midjourney{task})
	require.Len(t, items, 1)
	encoded, err := common.Marshal(items[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "channel_id")
	require.NotContains(t, string(encoded), "99403")
	require.NotContains(t, string(encoded), "渠道")
	require.Equal(t, i18n.Translate(i18n.LangZhCN, i18n.MsgRelayTaskUnavailable), items[0].FailReason)
	require.Equal(t, 99403, task.ChannelId)
	require.Contains(t, task.FailReason, "99403")
}
