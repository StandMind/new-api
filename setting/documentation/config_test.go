package documentationsetting

import "testing"

func TestParseSettingsNormalizesValues(t *testing.T) {
	raw := `{
		"enabled": true,
		"content_dir": "docs",
		"default_locale": "zh-CN",
		"default_slug": "/guide/intro",
		"pages": [
			{"path": "terms", "slug": "/legal/terms", "title": {"en": "Terms"}}
		],
		"debug_examples": {
			"/guide/intro": {
				"method": "post",
				"path": "/v1/chat/completions",
				"auth": "bearer",
				"body": {"en": {"model": "test"}}
			}
		},
		"nav": [
			{
				"slug": "/guide/intro",
				"file": "getting-started/intro.md",
				"title": {"en": "Intro", "zh": "介绍"},
				"children": [{"slug": "guide/advanced"}]
			}
		]
	}`

	settings, err := ParseSettings(raw)
	if err != nil {
		t.Fatalf("ParseSettings returned error: %v", err)
	}

	if settings.DefaultLocale != "zh" {
		t.Fatalf("DefaultLocale = %q, want zh", settings.DefaultLocale)
	}
	if settings.DefaultSlug != "guide/intro" {
		t.Fatalf("DefaultSlug = %q, want guide/intro", settings.DefaultSlug)
	}
	if settings.Nav[0].Slug != "guide/intro" {
		t.Fatalf("nav slug = %q, want guide/intro", settings.Nav[0].Slug)
	}
	if settings.Nav[0].File != "getting-started/intro.md" {
		t.Fatalf("nav file = %q, want getting-started/intro.md", settings.Nav[0].File)
	}
	if settings.Nav[0].Children[0].Slug != "guide/advanced" {
		t.Fatalf("child nav slug = %q, want guide/advanced", settings.Nav[0].Children[0].Slug)
	}
	if settings.Pages[0].Path != "/terms" || settings.Pages[0].Slug != "legal/terms" {
		t.Fatalf("page = %#v, want normalized path and slug", settings.Pages[0])
	}
	if settings.DebugExamples["guide/intro"].Method != "POST" {
		t.Fatalf("debug method = %q, want POST", settings.DebugExamples["guide/intro"].Method)
	}
}

func TestValidateSettingsRejectsUnsafeSlug(t *testing.T) {
	raw := `{
		"enabled": true,
		"content_dir": "docs",
		"default_locale": "en",
		"default_slug": "../secret",
		"nav": []
	}`

	if err := ValidateSettingsJSON(raw); err == nil {
		t.Fatal("ValidateSettingsJSON should reject path traversal slugs")
	}
}

func TestResolveLocaleAcceptLanguage(t *testing.T) {
	got := ResolveLocale("de-DE,de;q=0.9", "fr-FR;q=0.8", "zh-CN;q=0.7")
	if got != "fr" {
		t.Fatalf("ResolveLocale = %q, want fr", got)
	}
}
