package model

import "testing"

func TestResolveLocalizedTextLocale(t *testing.T) {
	got := ResolveLocalizedTextLocale("de-DE,de;q=0.9", "es-MX;q=0.8", "zh-CN;q=0.7")
	if got != "es" {
		t.Fatalf("ResolveLocalizedTextLocale = %q, want es", got)
	}
}

func TestLocalizedTextLocalizeFallback(t *testing.T) {
	value := LocalizedText{
		"en": "English description",
		"zh": "Chinese description",
	}

	if got := value.Localize("zh-CN", "fallback"); got != "Chinese description" {
		t.Fatalf("Localize zh-CN = %q, want Chinese description", got)
	}
	if got := value.Localize("fr", "fallback"); got != "fallback" {
		t.Fatalf("Localize fr = %q, want fallback", got)
	}
	if got := value.Localize("fr", ""); got != "English description" {
		t.Fatalf("Localize fr without fallback = %q, want English description", got)
	}
}

func TestLocalizedTextJSONRoundTrip(t *testing.T) {
	var value LocalizedText
	if err := value.UnmarshalJSON([]byte(`{"en":" English ","zh-CN":"Chinese","de":"Deutsch","ja":""}`)); err != nil {
		t.Fatalf("UnmarshalJSON error = %v", err)
	}
	if value["en"] != "English" || value["zh"] != "Chinese" {
		t.Fatalf("unexpected normalized value: %#v", value)
	}
	if _, ok := value["de"]; ok {
		t.Fatalf("unsupported locale was retained: %#v", value)
	}

	if bytes, err := value.MarshalJSON(); err != nil {
		t.Fatalf("MarshalJSON error = %v", err)
	} else if string(bytes) != `{"en":"English","zh":"Chinese"}` && string(bytes) != `{"zh":"Chinese","en":"English"}` {
		t.Fatalf("MarshalJSON = %s", string(bytes))
	}

	if dbValue, err := value.Value(); err != nil {
		t.Fatalf("Value error = %v", err)
	} else if dbValue == nil {
		t.Fatal("Value returned nil for non-empty localized text")
	}
}
