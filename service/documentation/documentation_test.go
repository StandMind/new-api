package docservice

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	documentationsetting "github.com/QuantumNous/new-api/setting/documentation"
)

func withDocumentationOption(t *testing.T, raw string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	previousMap := common.OptionMap
	common.OptionMap = map[string]string{
		documentationsetting.OptionKey: raw,
	}
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousMap
		common.OptionMapRWMutex.Unlock()
	})
}

func TestGetPageUsesLocalizedContentAndNavTitle(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "en"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "zh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "en", "guide.md"), []byte("# Guide\n\nEnglish"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "zh", "guide.md"), []byte("# 指南\n\n中文"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := `{
		"enabled": true,
		"content_dir": ` + quoteJSON(t, dir) + `,
		"default_locale": "en",
		"default_slug": "guide",
		"debug_examples": {
			"guide": {
				"method": "POST",
				"path": "/v1/chat/completions",
				"auth": "bearer",
				"body": {"zh": {"model": "gpt-test"}}
			}
		},
		"nav": [
			{"slug": "guide", "title": {"en": "Guide", "zh": "指南"}}
		]
	}`
	withDocumentationOption(t, raw)

	page, err := GetPage("guide", "zh-CN,zh;q=0.9")
	if err != nil {
		t.Fatalf("GetPage returned error: %v", err)
	}
	if page.Locale != "zh" {
		t.Fatalf("Locale = %q, want zh", page.Locale)
	}
	if page.Title != "指南" {
		t.Fatalf("Title = %q, want 指南", page.Title)
	}
	if page.Content != "# 指南\n\n中文" {
		t.Fatalf("Content = %q", page.Content)
	}
	if page.Debug == nil || page.Debug.Method != "POST" {
		t.Fatalf("Debug = %#v, want POST debug example", page.Debug)
	}
}

func TestGetPageFallsBackToDefaultLocale(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "en"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "en", "index.md"), []byte("# Welcome"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := `{
		"enabled": true,
		"content_dir": ` + quoteJSON(t, dir) + `,
		"default_locale": "en",
		"default_slug": "index",
		"nav": []
	}`
	withDocumentationOption(t, raw)

	page, err := GetPage("", "fr")
	if err != nil {
		t.Fatalf("GetPage returned error: %v", err)
	}
	if page.Locale != "en" {
		t.Fatalf("Locale = %q, want en", page.Locale)
	}
	if page.Title != "Welcome" {
		t.Fatalf("Title = %q, want Welcome", page.Title)
	}
}

func TestGetPageUsesConfiguredContentFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "en", "getting-started"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "en", "getting-started", "intro.md"), []byte("# Intro"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := `{
		"enabled": true,
		"content_dir": ` + quoteJSON(t, dir) + `,
		"default_locale": "en",
		"default_slug": "intro",
		"nav": [
			{"slug": "intro", "file": "getting-started/intro.md", "title": {"en": "Introduction"}}
		]
	}`
	withDocumentationOption(t, raw)

	page, err := GetPage("intro", "en")
	if err != nil {
		t.Fatalf("GetPage returned error: %v", err)
	}
	if page.Title != "Introduction" {
		t.Fatalf("Title = %q, want Introduction", page.Title)
	}
	if page.Content != "# Intro" {
		t.Fatalf("Content = %q, want # Intro", page.Content)
	}
}

func TestGetPageReturnsDisabled(t *testing.T) {
	withDocumentationOption(t, `{"enabled": false, "content_dir": "data/docs", "default_locale": "en", "nav": []}`)

	_, err := GetPage("index", "en")
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("GetPage error = %v, want ErrDisabled", err)
	}
}

func quoteJSON(t *testing.T, value string) string {
	t.Helper()
	bytes, err := common.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}
