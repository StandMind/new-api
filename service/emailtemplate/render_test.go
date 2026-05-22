package emailtemplate

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	emailtemplatesetting "github.com/QuantumNous/new-api/setting/emailtemplate"
)

func withEmailTemplates(t *testing.T, raw string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	originalMap := common.OptionMap
	copied := make(map[string]string, len(originalMap)+1)
	for key, value := range originalMap {
		copied[key] = value
	}
	copied[emailtemplatesetting.OptionKey] = raw
	common.OptionMap = copied
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalMap
		common.OptionMapRWMutex.Unlock()
	})
}

func TestBuildVerificationEmailUsesConfiguredLocale(t *testing.T) {
	withEmailTemplates(t, `{
		"default_locale": "en",
		"templates": {
			"verification": {
				"fr": {
					"subject": "Code {{.Code}} for {{.SystemName}}",
					"html": "<p>{{.SystemName}} {{.Code}}</p>"
				}
			}
		}
	}`)

	message, err := BuildVerificationEmail("fr", emailtemplatesetting.RenderData{
		SystemName:   "Gateway",
		Code:         "123456",
		ValidMinutes: 10,
	})
	if err != nil {
		t.Fatalf("BuildVerificationEmail error = %v", err)
	}
	if message.Subject != "Code 123456 for Gateway" {
		t.Fatalf("subject = %q", message.Subject)
	}
	if !strings.Contains(message.HTML, "Gateway 123456") {
		t.Fatalf("html = %q", message.HTML)
	}
}

func TestBuildPasswordResetEmailFallsBackToDefaultTemplates(t *testing.T) {
	withEmailTemplates(t, `{"default_locale":"en","templates":{}}`)

	message, err := BuildPasswordResetEmail("vi", emailtemplatesetting.RenderData{
		SystemName:   "Gateway",
		ResetLink:    "https://example.com/reset",
		ValidMinutes: 10,
	})
	if err != nil {
		t.Fatalf("BuildPasswordResetEmail error = %v", err)
	}
	if !strings.Contains(message.Subject, "Gateway") {
		t.Fatalf("subject = %q", message.Subject)
	}
	if !strings.Contains(message.HTML, "https://example.com/reset") {
		t.Fatalf("html = %q", message.HTML)
	}
}

func TestBuildVerificationEmailPrefersDefaultTemplateForRequestedLocale(t *testing.T) {
	withEmailTemplates(t, `{
		"default_locale": "en",
		"templates": {
			"verification": {
				"zh": {
					"subject": "自定义 {{.Code}}",
					"html": "<p>自定义 {{.Code}}</p>"
				}
			}
		}
	}`)

	message, err := BuildVerificationEmail("fr", emailtemplatesetting.RenderData{
		SystemName:   "Gateway",
		Code:         "123456",
		ValidMinutes: 10,
	})
	if err != nil {
		t.Fatalf("BuildVerificationEmail error = %v", err)
	}
	if strings.Contains(message.Subject, "自定义") {
		t.Fatalf("subject used fallback locale before requested default template: %q", message.Subject)
	}
	if !strings.Contains(message.Subject, "Gateway") {
		t.Fatalf("subject = %q", message.Subject)
	}
}
