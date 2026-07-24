package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGPTImageModelsExposeGenerationAndEditEndpoints(t *testing.T) {
	t.Parallel()

	expected := []constant.EndpointType{
		constant.EndpointTypeImageGeneration,
		constant.EndpointTypeImageEdit,
		constant.EndpointTypeOpenAI,
	}

	for _, modelName := range []string{"gpt-image-1", "gpt-image-1.5", "gpt-image-2"} {
		t.Run(modelName, func(t *testing.T) {
			t.Parallel()
			actual := GetEndpointTypesByChannelType(constant.ChannelTypeOpenAI, modelName)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestImageEditEndpointDefaults(t *testing.T) {
	t.Parallel()

	info, ok := GetDefaultEndpointInfo(constant.EndpointTypeImageEdit)
	require.True(t, ok)
	assert.Equal(t, EndpointInfo{Path: "/v1/images/edits", Method: "POST"}, info)
}
