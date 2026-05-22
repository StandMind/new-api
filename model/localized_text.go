package model

import (
	"database/sql/driver"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var localizedTextLocales = map[string]struct{}{
	"en": {},
	"zh": {},
	"es": {},
	"fr": {},
	"ru": {},
	"ja": {},
	"vi": {},
}

// LocalizedText stores per-locale text in a single TEXT column as JSON.
type LocalizedText map[string]string

func NewLocalizedText(locale string, value string) LocalizedText {
	lt := LocalizedText{}
	return lt.With(locale, value)
}

func (lt LocalizedText) With(locale string, value string) LocalizedText {
	locale = NormalizeLocalizedTextLocale(locale)
	if locale == "" {
		return lt
	}
	if lt == nil {
		lt = LocalizedText{}
	}
	value = strings.TrimSpace(value)
	if value == "" {
		delete(lt, locale)
		return lt
	}
	lt[locale] = value
	return lt
}

func (lt LocalizedText) Localize(locale string, fallback string) string {
	locale = NormalizeLocalizedTextLocale(locale)
	if locale != "" {
		if value := strings.TrimSpace(lt[locale]); value != "" {
			return value
		}
	}
	if fallback = strings.TrimSpace(fallback); fallback != "" {
		return fallback
	}
	for _, candidate := range []string{"en", "zh"} {
		if value := strings.TrimSpace(lt[candidate]); value != "" {
			return value
		}
	}
	for _, value := range lt {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return fallback
}

func (lt LocalizedText) Value() (driver.Value, error) {
	cleaned := cleanLocalizedTextMap(lt)
	if len(cleaned) == 0 {
		return nil, nil
	}
	bytes, err := common.Marshal(map[string]string(cleaned))
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (lt *LocalizedText) Scan(value interface{}) error {
	switch v := value.(type) {
	case nil:
		*lt = nil
		return nil
	case []byte:
		return lt.scanBytes(v)
	case string:
		return lt.scanBytes([]byte(v))
	default:
		bytes, err := common.Marshal(v)
		if err != nil {
			return err
		}
		return lt.scanBytes(bytes)
	}
}

func (lt *LocalizedText) scanBytes(value []byte) error {
	raw := strings.TrimSpace(string(value))
	if raw == "" || raw == "null" {
		*lt = nil
		return nil
	}
	var parsed map[string]string
	if err := common.Unmarshal([]byte(raw), &parsed); err != nil {
		return err
	}
	*lt = cleanLocalizedTextMap(parsed)
	return nil
}

func (lt LocalizedText) MarshalJSON() ([]byte, error) {
	cleaned := cleanLocalizedTextMap(lt)
	if len(cleaned) == 0 {
		return []byte("{}"), nil
	}
	return common.Marshal(map[string]string(cleaned))
}

func (lt *LocalizedText) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*lt = nil
		return nil
	}

	var parsed map[string]string
	if err := common.Unmarshal(data, &parsed); err == nil {
		if cleaned := cleanLocalizedTextMap(parsed); cleaned != nil {
			*lt = cleaned
		} else {
			*lt = LocalizedText{}
		}
		return nil
	}

	var encoded string
	if err := common.Unmarshal(data, &encoded); err != nil {
		return err
	}
	if strings.TrimSpace(encoded) == "" {
		*lt = nil
		return nil
	}
	if err := common.UnmarshalJsonStr(encoded, &parsed); err != nil {
		return err
	}
	if cleaned := cleanLocalizedTextMap(parsed); cleaned != nil {
		*lt = cleaned
	} else {
		*lt = LocalizedText{}
	}
	return nil
}

func SupportedLocalizedTextLocales() []string {
	return []string{"en", "zh", "es", "fr", "ru", "ja", "vi"}
}

func NormalizeLocalizedTextLocale(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "_", "-")))
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, "zh") {
		return "zh"
	}
	if idx := strings.IndexByte(normalized, '-'); idx >= 0 {
		normalized = normalized[:idx]
	}
	if _, ok := localizedTextLocales[normalized]; ok {
		return normalized
	}
	return ""
}

func ResolveLocalizedTextLocale(values ...string) string {
	for _, value := range values {
		for _, candidate := range strings.Split(value, ",") {
			lang := strings.TrimSpace(strings.SplitN(candidate, ";", 2)[0])
			if normalized := NormalizeLocalizedTextLocale(lang); normalized != "" {
				return normalized
			}
		}
	}
	return "en"
}

func cleanLocalizedTextMap(values map[string]string) LocalizedText {
	if len(values) == 0 {
		return nil
	}
	cleaned := LocalizedText{}
	for locale, value := range values {
		locale = NormalizeLocalizedTextLocale(locale)
		value = strings.TrimSpace(value)
		if locale == "" || value == "" {
			continue
		}
		cleaned[locale] = value
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}
