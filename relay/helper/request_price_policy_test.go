package helper

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeRequestPricePolicySupportsEveryTierOnBothProtocols(t *testing.T) {
	models := []string{
		"gemini-3-pro-image",
		"gemini-3-pro-image-preview",
		"gemini-3.1-flash-image",
		"gemini-3.1-flash-image-preview",
	}
	for _, model := range models {
		policy, ok := billing_setting.GetRequestPricePolicy(model)
		require.True(t, ok)
		for _, tier := range policy.Tiers {
			for _, protocol := range []string{"gemini", "openai"} {
				t.Run(model+"/"+protocol+"/"+tier.Value, func(t *testing.T) {
					var request dto.Request
					if protocol == "gemini" {
						request = &dto.GeminiChatRequest{
							GenerationConfig: dto.GeminiChatGenerationConfig{
								ImageConfig: mustJSON(t, map[string]any{
									"aspectRatio": "1:1",
									"imageSize":   " " + strings.ToLower(tier.Value) + " ",
								}),
							},
						}
					} else {
						request = &dto.GeneralOpenAIRequest{
							Model: model,
							ExtraBody: mustJSON(t, map[string]any{
								"google": map[string]any{
									"image_config": map[string]any{
										"aspect_ratio": "1:1",
										"image_size":   " " + strings.ToLower(tier.Value) + " ",
									},
								},
							}),
						}
					}

					ratios, err := NormalizeRequestPricePolicy(model, request)
					require.NoError(t, err)
					require.Equal(t, map[string]float64{
						billing_setting.ImageResolutionBillingRatioPrefix + tier.Value: tier.Multiplier,
					}, ratios)
					if native, ok := request.(*dto.GeminiChatRequest); ok {
						require.Equal(t, tier.Value, gjson.GetBytes(native.GenerationConfig.ImageConfig, "imageSize").String())
						require.False(t, gjson.GetBytes(native.GenerationConfig.ImageConfig, "image_size").Exists())
					} else {
						compatible := request.(*dto.GeneralOpenAIRequest)
						require.Equal(t, tier.Value, gjson.GetBytes(compatible.ExtraBody, "google.image_config.image_size").String())
					}
				})
			}
		}
	}
}

func TestNormalizeRequestPricePolicyDefaultsToOneK(t *testing.T) {
	native := &dto.GeminiChatRequest{}
	nativeRatios, err := NormalizeRequestPricePolicy("gemini-3-pro-image", native)
	require.NoError(t, err)
	require.Equal(t, 1.0, nativeRatios["image_resolution:1K"])
	require.Equal(t, "1K", gjson.GetBytes(native.GenerationConfig.ImageConfig, "imageSize").String())

	compatible := &dto.GeneralOpenAIRequest{Model: "gemini-3.1-flash-image"}
	compatibleRatios, err := NormalizeRequestPricePolicy("gemini-3.1-flash-image", compatible)
	require.NoError(t, err)
	require.Equal(t, 1.0, compatibleRatios["image_resolution:1K"])
	require.Equal(t, "1K", gjson.GetBytes(compatible.ExtraBody, "google.image_config.image_size").String())
}

func TestNormalizeRequestPricePolicyRejectsInvalidResolutionBeforeBilling(t *testing.T) {
	tests := []struct {
		name    string
		request dto.Request
		wantErr string
	}{
		{
			name: "native unknown value",
			request: &dto.GeminiChatRequest{GenerationConfig: dto.GeminiChatGenerationConfig{
				ImageConfig: []byte(`{"imageSize":"8K"}`),
			}},
			wantErr: "must be one of 1K, 2K, 4K",
		},
		{
			name: "native non-string value",
			request: &dto.GeminiChatRequest{GenerationConfig: dto.GeminiChatGenerationConfig{
				ImageConfig: []byte(`{"imageSize":4096}`),
			}},
			wantErr: "must be a string",
		},
		{
			name: "native conflicting aliases",
			request: &dto.GeminiChatRequest{GenerationConfig: dto.GeminiChatGenerationConfig{
				ImageConfig: []byte(`{"imageSize":"1K","image_size":"4K"}`),
			}},
			wantErr: "conflicting fields",
		},
		{
			name:    "openai extra body is not object",
			request: &dto.GeneralOpenAIRequest{ExtraBody: []byte(`[]`)},
			wantErr: "extra_body must be an object",
		},
		{
			name:    "openai google is not object",
			request: &dto.GeneralOpenAIRequest{ExtraBody: []byte(`{"google":true}`)},
			wantErr: "extra_body.google must be an object",
		},
		{
			name:    "openai image config is not object",
			request: &dto.GeneralOpenAIRequest{ExtraBody: []byte(`{"google":{"image_config":"4K"}}`)},
			wantErr: "extra_body.google.image_config must be an object",
		},
		{
			name:    "openai camel case is rejected",
			request: &dto.GeneralOpenAIRequest{ExtraBody: []byte(`{"google":{"image_config":{"imageSize":"4K"}}}`)},
			wantErr: "use extra_body.google.image_config.image_size instead",
		},
		{
			name:    "openai conflicting fields",
			request: &dto.GeneralOpenAIRequest{ExtraBody: []byte(`{"google":{"image_config":{"imageSize":"1K","image_size":"4K"}}}`)},
			wantErr: "conflicting fields",
		},
		{
			name:    "unsupported request format",
			request: &dto.BaseRequest{},
			wantErr: "does not support resolution pricing on this endpoint",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizeRequestPricePolicy("gemini-3-pro-image", test.request)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestNormalizeRequestPricePolicyLeavesOtherModelsUntouched(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{ExtraBody: []byte(`{"custom":true}`)}
	rawBefore := string(request.ExtraBody)

	ratios, err := NormalizeRequestPricePolicy("gemini-2.5-flash-image", request)

	require.NoError(t, err)
	require.Nil(t, ratios)
	require.Equal(t, rawBefore, string(request.ExtraBody))
}

func TestSyncRequestPricePolicyBodyPreservesUnknownFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		model      string
		body       string
		request    dto.Request
		valuePath  string
		absentPath string
		wantValue  string
	}{
		{
			name:  "gemini native alias",
			model: "gemini-3-pro-image",
			body:  `{"contents":[],"custom":{"keep":true},"generationConfig":{"imageConfig":{"aspectRatio":"1:1","image_size":" 4k "}}}`,
			request: &dto.GeminiChatRequest{GenerationConfig: dto.GeminiChatGenerationConfig{
				ImageConfig: []byte(`{"aspectRatio":"1:1","image_size":" 4k "}`),
			}},
			valuePath:  "generationConfig.imageConfig.imageSize",
			absentPath: "generationConfig.imageConfig.image_size",
			wantValue:  "4K",
		},
		{
			name:  "openai default",
			model: "gemini-3.1-flash-image",
			body:  `{"model":"gemini-3.1-flash-image","messages":[],"custom":{"keep":true}}`,
			request: &dto.GeneralOpenAIRequest{
				Model: "gemini-3.1-flash-image",
			},
			valuePath: "extra_body.google.image_config.image_size",
			wantValue: "1K",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(test.body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			_, err := common.GetBodyStorage(ctx)
			require.NoError(t, err)
			t.Cleanup(func() { common.CleanupBodyStorage(ctx) })

			_, err = NormalizeRequestPricePolicy(test.model, test.request)
			require.NoError(t, err)
			require.NoError(t, SyncRequestPricePolicyBody(ctx, test.request))

			storage, err := common.GetBodyStorage(ctx)
			require.NoError(t, err)
			normalized, err := storage.Bytes()
			require.NoError(t, err)
			require.Equal(t, test.wantValue, gjson.GetBytes(normalized, test.valuePath).String(), string(normalized))
			if test.absentPath != "" {
				require.False(t, gjson.GetBytes(normalized, test.absentPath).Exists(), string(normalized))
			}
			require.True(t, gjson.GetBytes(normalized, "custom.keep").Bool(), string(normalized))
			require.Equal(t, int64(len(normalized)), ctx.Request.ContentLength)
		})
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	return encoded
}
