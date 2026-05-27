package model

import "testing"

func TestLocalizePricingDataLocalizesModelTags(t *testing.T) {
	pricing := []Pricing{
		{
			ModelName: "gemini-test",
			Tags:      "chat,reasoning",
			TagsI18n: LocalizedText{
				"zh": "对话,思考",
				"es": "chat,razonamiento",
			},
		},
	}

	localized, _ := LocalizePricingData(pricing, nil, "zh-CN")
	if got := localized[0].Tags; got != "对话,思考" {
		t.Fatalf("localized zh tags = %q, want 对话,思考", got)
	}

	localized, _ = LocalizePricingData(pricing, nil, "fr")
	if got := localized[0].Tags; got != "chat,reasoning" {
		t.Fatalf("fallback tags = %q, want chat,reasoning", got)
	}
}
