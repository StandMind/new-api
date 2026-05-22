/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

package documentationsetting

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	OptionKey         = "DocumentationSettings"
	DefaultLocale     = "en"
	DefaultContentDir = "data/docs"
)

var (
	supportedLocales = map[string]struct{}{
		"en": {},
		"zh": {},
		"es": {},
		"fr": {},
		"ru": {},
		"ja": {},
		"vi": {},
	}
	slugPattern = regexp.MustCompile(`^[A-Za-z0-9._~/-]+$`)
)

type LocalizedText map[string]string

type NavItem struct {
	Slug        string        `json:"slug,omitempty"`
	File        string        `json:"file,omitempty"`
	Title       LocalizedText `json:"title,omitempty"`
	Description LocalizedText `json:"description,omitempty"`
	Children    []NavItem     `json:"children,omitempty"`
}

type PageItem struct {
	Path        string        `json:"path"`
	Slug        string        `json:"slug"`
	File        string        `json:"file,omitempty"`
	Title       LocalizedText `json:"title,omitempty"`
	Description LocalizedText `json:"description,omitempty"`
}

type DebugExample struct {
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	PathTemplate string            `json:"path_template,omitempty"`
	Auth         string            `json:"auth,omitempty"`
	Model        string            `json:"model,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         map[string]any    `json:"body,omitempty"`
}

type Settings struct {
	Enabled       bool                    `json:"enabled"`
	ContentDir    string                  `json:"content_dir"`
	DefaultLocale string                  `json:"default_locale"`
	DefaultSlug   string                  `json:"default_slug"`
	Nav           []NavItem               `json:"nav"`
	Pages         []PageItem              `json:"pages,omitempty"`
	DebugExamples map[string]DebugExample `json:"debug_examples,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{
		Enabled:       false,
		ContentDir:    DefaultContentDir,
		DefaultLocale: DefaultLocale,
		DefaultSlug:   "",
		Nav:           []NavItem{},
		Pages:         []PageItem{},
		DebugExamples: map[string]DebugExample{},
	}
}

func DefaultSettingsJSONString() string {
	bytes, err := common.Marshal(DefaultSettings())
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func SupportedLocales() []string {
	return []string{"en", "zh", "es", "fr", "ru", "ja", "vi"}
}

func NormalizeLocale(value string) string {
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
	if _, ok := supportedLocales[normalized]; ok {
		return normalized
	}
	return ""
}

func ResolveLocale(values ...string) string {
	for _, value := range values {
		for _, candidate := range strings.Split(value, ",") {
			lang := strings.TrimSpace(strings.SplitN(candidate, ";", 2)[0])
			if normalized := NormalizeLocale(lang); normalized != "" {
				return normalized
			}
		}
	}
	return DefaultLocale
}

func NormalizeSlug(value string) (string, error) {
	slug := strings.TrimSpace(strings.TrimPrefix(value, "/"))
	if slug == "" {
		return "", fmt.Errorf("slug is empty")
	}
	if strings.Contains(slug, "\\") {
		return "", fmt.Errorf("slug must not contain backslashes")
	}
	if !slugPattern.MatchString(slug) {
		return "", fmt.Errorf("slug %q contains unsupported characters", value)
	}
	cleaned := path.Clean(slug)
	if cleaned == "." || cleaned == "/" || strings.HasPrefix(cleaned, "../") || cleaned == ".." || strings.Contains(cleaned, "/../") {
		return "", fmt.Errorf("slug %q is invalid", value)
	}
	return cleaned, nil
}

func NormalizeContentPath(value string) (string, error) {
	return NormalizeSlug(value)
}

func ParseSettings(raw string) (Settings, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultSettings(), nil
	}

	settings := DefaultSettings()
	if err := common.UnmarshalJsonStr(raw, &settings); err != nil {
		return Settings{}, err
	}

	if strings.TrimSpace(settings.ContentDir) == "" {
		settings.ContentDir = DefaultContentDir
	}
	if normalized := NormalizeLocale(settings.DefaultLocale); normalized != "" {
		settings.DefaultLocale = normalized
	} else {
		settings.DefaultLocale = DefaultLocale
	}
	if strings.TrimSpace(settings.DefaultSlug) != "" {
		defaultSlug, err := NormalizeSlug(settings.DefaultSlug)
		if err != nil {
			return Settings{}, err
		}
		settings.DefaultSlug = defaultSlug
	}
	if settings.Nav == nil {
		settings.Nav = []NavItem{}
	}
	var err error
	settings.Nav, err = normalizeNav(settings.Nav)
	if err != nil {
		return Settings{}, err
	}
	if settings.Pages == nil {
		settings.Pages = []PageItem{}
	}
	settings.Pages, err = normalizePages(settings.Pages)
	if err != nil {
		return Settings{}, err
	}
	if settings.DebugExamples == nil {
		settings.DebugExamples = map[string]DebugExample{}
	}
	settings.DebugExamples, err = normalizeDebugExamples(settings.DebugExamples)
	if err != nil {
		return Settings{}, err
	}

	return settings, nil
}

func normalizeNav(items []NavItem) ([]NavItem, error) {
	normalized := make([]NavItem, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Slug) != "" {
			slug, err := NormalizeSlug(item.Slug)
			if err != nil {
				return nil, err
			}
			item.Slug = slug
		}
		if strings.TrimSpace(item.File) != "" {
			file, err := NormalizeContentPath(item.File)
			if err != nil {
				return nil, err
			}
			item.File = file
		}
		if item.Children == nil {
			item.Children = []NavItem{}
		}
		children, err := normalizeNav(item.Children)
		if err != nil {
			return nil, err
		}
		item.Children = children
		normalized = append(normalized, item)
	}
	return normalized, nil
}

func normalizePages(items []PageItem) ([]PageItem, error) {
	normalized := make([]PageItem, 0, len(items))
	for _, item := range items {
		pagePath, err := NormalizePagePath(item.Path)
		if err != nil {
			return nil, err
		}
		slug, err := NormalizeSlug(item.Slug)
		if err != nil {
			return nil, err
		}
		item.Path = pagePath
		item.Slug = slug
		if strings.TrimSpace(item.File) != "" {
			file, err := NormalizeContentPath(item.File)
			if err != nil {
				return nil, err
			}
			item.File = file
		}
		normalized = append(normalized, item)
	}
	return normalized, nil
}

func normalizeDebugExamples(examples map[string]DebugExample) (map[string]DebugExample, error) {
	normalized := make(map[string]DebugExample, len(examples))
	for slug, example := range examples {
		normalizedSlug, err := NormalizeSlug(slug)
		if err != nil {
			return nil, err
		}
		example.Method = strings.ToUpper(strings.TrimSpace(example.Method))
		example.Path = strings.TrimSpace(example.Path)
		example.PathTemplate = strings.TrimSpace(example.PathTemplate)
		example.Auth = strings.TrimSpace(strings.ToLower(example.Auth))
		if example.Headers == nil {
			example.Headers = map[string]string{}
		}
		if example.Body == nil {
			example.Body = map[string]any{}
		}
		normalized[normalizedSlug] = example
	}
	return normalized, nil
}

func NormalizePagePath(value string) (string, error) {
	pagePath := strings.TrimSpace(value)
	if pagePath == "" {
		return "", fmt.Errorf("page path is empty")
	}
	if !strings.HasPrefix(pagePath, "/") {
		pagePath = "/" + pagePath
	}
	if strings.Contains(pagePath, "\\") || strings.Contains(pagePath, "\x00") {
		return "", fmt.Errorf("page path %q is invalid", value)
	}
	cleaned := path.Clean(pagePath)
	if cleaned == "." || cleaned == "/" || strings.HasPrefix(cleaned, "/api/") {
		return "", fmt.Errorf("page path %q is invalid", value)
	}
	return cleaned, nil
}

func ValidateSettingsJSON(raw string) error {
	_, err := NormalizeSettingsJSONString(raw)
	return err
}

func NormalizeSettingsJSONString(raw string) (string, error) {
	settings, err := ParseSettings(raw)
	if err != nil {
		return "", err
	}
	if strings.Contains(settings.ContentDir, "\x00") {
		return "", fmt.Errorf("content_dir contains invalid characters")
	}
	if err := validateNav(settings.Nav); err != nil {
		return "", err
	}
	if err := validatePages(settings.Pages); err != nil {
		return "", err
	}
	if err := validateDebugExamples(settings.DebugExamples); err != nil {
		return "", err
	}
	bytes, err := common.Marshal(settings)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func validateNav(items []NavItem) error {
	for _, item := range items {
		if strings.TrimSpace(item.Slug) != "" {
			if _, err := NormalizeSlug(item.Slug); err != nil {
				return err
			}
		}
		if len(item.Title) == 0 && strings.TrimSpace(item.Slug) == "" {
			return fmt.Errorf("navigation item requires either slug or title")
		}
		for locale := range item.Title {
			if NormalizeLocale(locale) == "" {
				return fmt.Errorf("navigation item title uses unsupported locale %q", locale)
			}
		}
		for locale := range item.Description {
			if NormalizeLocale(locale) == "" {
				return fmt.Errorf("navigation item description uses unsupported locale %q", locale)
			}
		}
		if strings.TrimSpace(item.File) != "" {
			if _, err := NormalizeContentPath(item.File); err != nil {
				return err
			}
		}
		if err := validateNav(item.Children); err != nil {
			return err
		}
	}
	return nil
}

func validatePages(items []PageItem) error {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, ok := seen[item.Path]; ok {
			return fmt.Errorf("duplicate page path %q", item.Path)
		}
		seen[item.Path] = struct{}{}
		for locale := range item.Title {
			if NormalizeLocale(locale) == "" {
				return fmt.Errorf("page title uses unsupported locale %q", locale)
			}
		}
		for locale := range item.Description {
			if NormalizeLocale(locale) == "" {
				return fmt.Errorf("page description uses unsupported locale %q", locale)
			}
		}
		if strings.TrimSpace(item.File) != "" {
			if _, err := NormalizeContentPath(item.File); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDebugExamples(examples map[string]DebugExample) error {
	for slug, example := range examples {
		if strings.TrimSpace(slug) == "" {
			return fmt.Errorf("debug example slug is empty")
		}
		if example.Method == "" {
			return fmt.Errorf("debug example %q requires method", slug)
		}
		switch example.Method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			return fmt.Errorf("debug example %q has unsupported method %q", slug, example.Method)
		}
		if example.Path == "" && example.PathTemplate == "" {
			return fmt.Errorf("debug example %q requires path or path_template", slug)
		}
		if example.Auth != "" && example.Auth != "bearer" && example.Auth != "anthropic" {
			return fmt.Errorf("debug example %q has unsupported auth %q", slug, example.Auth)
		}
		for locale := range example.Body {
			if NormalizeLocale(locale) == "" {
				return fmt.Errorf("debug example %q body uses unsupported locale %q", slug, locale)
			}
		}
	}
	return nil
}

func LocalizeText(text LocalizedText, lang string, fallback string) string {
	if len(text) == 0 {
		return fallback
	}

	for _, locale := range []string{ResolveLocale(lang), DefaultLocale, "zh"} {
		if value := strings.TrimSpace(text[locale]); value != "" {
			return value
		}
	}

	for _, value := range text {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}

	return fallback
}
