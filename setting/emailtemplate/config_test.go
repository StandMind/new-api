package emailtemplatesetting

import "testing"

func TestNormalizeLocale(t *testing.T) {
	tests := map[string]string{
		"en":    "en",
		"en-US": "en",
		"zh-CN": "zh",
		"zh_TW": "zh",
		"es-MX": "es",
		"fr":    "fr",
		"ru":    "ru",
		"ja":    "ja",
		"vi":    "vi",
		"de":    "",
		"":      "",
	}

	for input, expected := range tests {
		if got := NormalizeLocale(input); got != expected {
			t.Fatalf("NormalizeLocale(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestValidateSettingsJSON(t *testing.T) {
	valid := `{
		"default_locale": "en",
		"templates": {
			"verification": {
				"en": {
					"subject": "{{.SystemName}} code {{.Code}}",
					"html": "<p>{{.Code}}</p>"
				}
			}
		}
	}`
	if err := ValidateSettingsJSON(valid); err != nil {
		t.Fatalf("ValidateSettingsJSON(valid) error = %v", err)
	}

	invalidLocale := `{
		"default_locale": "en",
		"templates": {
			"verification": {
				"de": {
					"subject": "Subject",
					"html": "<p>Body</p>"
				}
			}
		}
	}`
	if err := ValidateSettingsJSON(invalidLocale); err == nil {
		t.Fatal("ValidateSettingsJSON(invalidLocale) expected error")
	}
}
