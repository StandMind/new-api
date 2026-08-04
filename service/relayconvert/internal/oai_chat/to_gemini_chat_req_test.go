package oaichat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIChatRequestToGeminiImageConfigKeepsThinkingAdapter(t *testing.T) {
	geminiSettings := model_setting.GetGeminiSettings()
	originalEnabled := geminiSettings.ThinkingAdapterEnabled
	geminiSettings.ThinkingAdapterEnabled = true
	t.Cleanup(func() {
		geminiSettings.ThinkingAdapterEnabled = originalEnabled
	})

	extraBody, err := common.Marshal(map[string]any{
		"google": map[string]any{
			"image_config": map[string]any{
				"aspect_ratio": "1:1",
				"image_size":   "4K",
			},
		},
	})
	require.NoError(t, err)
	request := dto.GeneralOpenAIRequest{
		Model: "gemini-3-pro-image-preview",
		Messages: []dto.Message{
			{Role: "user", Content: "test"},
		},
		ExtraBody: extraBody,
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gemini-3-pro-image-preview-thinking",
		},
	}

	converted, err := OpenAIChatRequestToGeminiGenerateContent(nil, request, info)

	require.NoError(t, err)
	require.NotNil(t, converted.GenerationConfig.ThinkingConfig)
	require.True(t, converted.GenerationConfig.ThinkingConfig.IncludeThoughts)
	require.Equal(t, "1:1", gjson.GetBytes(converted.GenerationConfig.ImageConfig, "aspectRatio").String())
	require.Equal(t, "4K", gjson.GetBytes(converted.GenerationConfig.ImageConfig, "imageSize").String())
}
