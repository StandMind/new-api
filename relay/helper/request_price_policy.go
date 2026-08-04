package helper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/sjson"
)

func NormalizeRequestPricePolicy(model string, request dto.Request) (map[string]float64, error) {
	policy, ok := billing_setting.GetRequestPricePolicy(model)
	if !ok {
		return nil, nil
	}
	if policy.Dimension != billing_setting.RequestPriceDimensionImageResolution {
		return nil, fmt.Errorf("unsupported request price dimension %q", policy.Dimension)
	}

	var (
		value string
		err   error
	)
	switch typedRequest := request.(type) {
	case *dto.GeminiChatRequest:
		value, err = normalizeGeminiImageResolution(typedRequest, policy)
	case *dto.GeneralOpenAIRequest:
		value, err = normalizeOpenAIImageResolution(typedRequest, policy)
	default:
		return nil, fmt.Errorf("model %s does not support resolution pricing on this endpoint", model)
	}
	if err != nil {
		return nil, err
	}

	tier, ok := policy.FindTier(value)
	if !ok {
		return nil, fmt.Errorf("image resolution %q is not supported for model %s", value, model)
	}
	return map[string]float64{
		billing_setting.ImageResolutionBillingRatioPrefix + tier.Value: tier.Multiplier,
	}, nil
}

// SyncRequestPricePolicyBody applies only the normalized resolution field to
// the cached JSON body. This preserves unknown pass-through fields while
// ensuring retries and pass-through channels send the same tier that was used
// for billing.
func SyncRequestPricePolicyBody(c *gin.Context, request dto.Request) error {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return err
	}
	body, err := storage.Bytes()
	if err != nil {
		return err
	}

	var value string
	switch typedRequest := request.(type) {
	case *dto.GeminiChatRequest:
		imageConfig, err := decodeOptionalObject(typedRequest.GenerationConfig.ImageConfig, "generationConfig.imageConfig")
		if err != nil {
			return err
		}
		value, _, err = decodeResolutionField(imageConfig, "imageSize", "generationConfig.imageConfig")
		if err != nil {
			return err
		}
		body, err = sjson.SetBytes(body, "generationConfig.imageConfig.imageSize", value)
		if err != nil {
			return fmt.Errorf("normalize generationConfig.imageConfig.imageSize: %w", err)
		}
		body, err = sjson.DeleteBytes(body, "generationConfig.imageConfig.image_size")
		if err != nil {
			return fmt.Errorf("remove generationConfig.imageConfig.image_size: %w", err)
		}
	case *dto.GeneralOpenAIRequest:
		extraBody, err := decodeOptionalObject(typedRequest.ExtraBody, "extra_body")
		if err != nil {
			return err
		}
		googleBody, err := decodeNestedObject(extraBody, "google", "extra_body.google")
		if err != nil {
			return err
		}
		imageConfig, err := decodeNestedObject(googleBody, "image_config", "extra_body.google.image_config")
		if err != nil {
			return err
		}
		value, _, err = decodeResolutionField(imageConfig, "image_size", "extra_body.google.image_config")
		if err != nil {
			return err
		}
		body, err = sjson.SetBytes(body, "extra_body.google.image_config.image_size", value)
		if err != nil {
			return fmt.Errorf("normalize extra_body.google.image_config.image_size: %w", err)
		}
	default:
		return fmt.Errorf("unsupported request type %T for request price body normalization", request)
	}

	return common.ReplaceBodyStorage(c, body)
}

func normalizeGeminiImageResolution(request *dto.GeminiChatRequest, policy billing_setting.RequestPricePolicy) (string, error) {
	imageConfig, err := decodeOptionalObject(request.GenerationConfig.ImageConfig, "generationConfig.imageConfig")
	if err != nil {
		return "", err
	}
	value, err := resolveResolutionValue(imageConfig, "imageSize", "image_size", policy, "generationConfig.imageConfig")
	if err != nil {
		return "", err
	}
	delete(imageConfig, "image_size")
	imageConfig["imageSize"], err = marshalString(value)
	if err != nil {
		return "", err
	}
	request.GenerationConfig.ImageConfig, err = common.Marshal(imageConfig)
	if err != nil {
		return "", fmt.Errorf("encode generationConfig.imageConfig: %w", err)
	}
	return value, nil
}

func normalizeOpenAIImageResolution(request *dto.GeneralOpenAIRequest, policy billing_setting.RequestPricePolicy) (string, error) {
	extraBody, err := decodeOptionalObject(request.ExtraBody, "extra_body")
	if err != nil {
		return "", err
	}
	googleBody, err := decodeNestedObject(extraBody, "google", "extra_body.google")
	if err != nil {
		return "", err
	}
	if _, exists := googleBody["imageConfig"]; exists {
		return "", fmt.Errorf("extra_body.google.imageConfig is not supported, use extra_body.google.image_config instead")
	}
	imageConfig, err := decodeNestedObject(googleBody, "image_config", "extra_body.google.image_config")
	if err != nil {
		return "", err
	}
	if _, exists := imageConfig["imageSize"]; exists {
		if _, conflict := imageConfig["image_size"]; conflict {
			return "", fmt.Errorf("conflicting fields extra_body.google.image_config.imageSize and image_size")
		}
		return "", fmt.Errorf("extra_body.google.image_config.imageSize is not supported, use extra_body.google.image_config.image_size instead")
	}

	value, err := resolveResolutionValue(imageConfig, "image_size", "", policy, "extra_body.google.image_config")
	if err != nil {
		return "", err
	}
	imageConfig["image_size"], err = marshalString(value)
	if err != nil {
		return "", err
	}
	googleBody["image_config"], err = common.Marshal(imageConfig)
	if err != nil {
		return "", fmt.Errorf("encode extra_body.google.image_config: %w", err)
	}
	extraBody["google"], err = common.Marshal(googleBody)
	if err != nil {
		return "", fmt.Errorf("encode extra_body.google: %w", err)
	}
	request.ExtraBody, err = common.Marshal(extraBody)
	if err != nil {
		return "", fmt.Errorf("encode extra_body: %w", err)
	}
	return value, nil
}

func decodeOptionalObject(raw json.RawMessage, path string) (map[string]json.RawMessage, error) {
	if len(raw) == 0 {
		return make(map[string]json.RawMessage), nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return make(map[string]json.RawMessage), nil
	}
	if trimmed == "null" {
		return nil, fmt.Errorf("%s must be an object", path)
	}
	var object map[string]json.RawMessage
	if err := common.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("%s must be an object", path)
	}
	return object, nil
}

func decodeNestedObject(parent map[string]json.RawMessage, key, path string) (map[string]json.RawMessage, error) {
	raw, exists := parent[key]
	if !exists {
		return make(map[string]json.RawMessage), nil
	}
	return decodeOptionalObject(raw, path)
}

func resolveResolutionValue(fields map[string]json.RawMessage, canonicalKey, aliasKey string, policy billing_setting.RequestPricePolicy, path string) (string, error) {
	canonicalValue, canonicalExists, err := decodeResolutionField(fields, canonicalKey, path)
	if err != nil {
		return "", err
	}
	aliasValue, aliasExists, err := decodeResolutionField(fields, aliasKey, path)
	if err != nil {
		return "", err
	}
	if canonicalExists && aliasExists && canonicalValue != aliasValue {
		return "", fmt.Errorf("conflicting fields %s.%s and %s", path, canonicalKey, aliasKey)
	}
	value := policy.DefaultValue
	if canonicalExists {
		value = canonicalValue
	} else if aliasExists {
		value = aliasValue
	}
	if _, ok := policy.FindTier(value); !ok {
		return "", fmt.Errorf("%s.%s must be one of %s", path, canonicalKey, allowedResolutionValues(policy))
	}
	return value, nil
}

func decodeResolutionField(fields map[string]json.RawMessage, key, path string) (string, bool, error) {
	if key == "" {
		return "", false, nil
	}
	raw, exists := fields[key]
	if !exists {
		return "", false, nil
	}
	var value string
	if err := common.Unmarshal(raw, &value); err != nil {
		return "", false, fmt.Errorf("%s.%s must be a string", path, key)
	}
	return strings.ToUpper(strings.TrimSpace(value)), true, nil
}

func allowedResolutionValues(policy billing_setting.RequestPricePolicy) string {
	values := make([]string, 0, len(policy.Tiers))
	for _, tier := range policy.Tiers {
		values = append(values, tier.Value)
	}
	return strings.Join(values, ", ")
}

func marshalString(value string) (json.RawMessage, error) {
	encoded, err := common.Marshal(value)
	return json.RawMessage(encoded), err
}
