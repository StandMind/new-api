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

package emailtemplate

import (
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"

	"github.com/QuantumNous/new-api/common"
	emailtemplatesetting "github.com/QuantumNous/new-api/setting/emailtemplate"
)

type Message struct {
	Subject string
	HTML    string
}

func BuildVerificationEmail(lang string, data emailtemplatesetting.RenderData) (Message, error) {
	return build(emailtemplatesetting.TemplateVerification, lang, data)
}

func BuildPasswordResetEmail(lang string, data emailtemplatesetting.RenderData) (Message, error) {
	return build(emailtemplatesetting.TemplatePasswordReset, lang, data)
}

func ResolveRequestLocale(queryLang string, acceptLanguage string, contextLang string) string {
	return emailtemplatesetting.ResolveLocale(queryLang, acceptLanguage, contextLang)
}

func build(templateName string, lang string, data emailtemplatesetting.RenderData) (Message, error) {
	settings := loadSettings()
	if data.SupportEmail == "" {
		data.SupportEmail = settings.SupportEmail
	}
	if data.SupportEmail == "" {
		data.SupportEmail = common.SMTPFrom
	}

	tpl, ok := findTemplate(settings, templateName, lang)
	if !ok {
		return Message{}, fmt.Errorf("email template %q is not configured", templateName)
	}

	subject, err := renderSubject(templateName, tpl.Subject, data)
	if err != nil {
		return Message{}, err
	}
	html, err := renderHTML(templateName, tpl.HTML, data)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Subject: subject,
		HTML:    html,
	}, nil
}

func loadSettings() emailtemplatesetting.Settings {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[emailtemplatesetting.OptionKey]
	common.OptionMapRWMutex.RUnlock()

	settings, err := emailtemplatesetting.ParseSettings(raw)
	if err != nil {
		common.SysError(fmt.Sprintf("failed to parse %s: %v", emailtemplatesetting.OptionKey, err))
		return emailtemplatesetting.DefaultSettings()
	}
	return settings
}

func findTemplate(settings emailtemplatesetting.Settings, templateName string, lang string) (emailtemplatesetting.Template, bool) {
	defaults := emailtemplatesetting.DefaultSettings()
	localeChain := buildLocaleChain(lang, settings.DefaultLocale)

	for _, locale := range localeChain {
		for _, source := range []emailtemplatesetting.Settings{settings, defaults} {
			tpl, ok := source.Templates[templateName][locale]
			if ok && strings.TrimSpace(tpl.Subject) != "" && strings.TrimSpace(tpl.HTML) != "" {
				return tpl, true
			}
		}
	}

	return emailtemplatesetting.Template{}, false
}

func buildLocaleChain(values ...string) []string {
	seen := make(map[string]struct{})
	locales := make([]string, 0, 4)

	for _, value := range values {
		locale := emailtemplatesetting.ResolveLocale(value)
		if _, ok := seen[locale]; !ok {
			seen[locale] = struct{}{}
			locales = append(locales, locale)
		}
	}
	for _, locale := range []string{emailtemplatesetting.DefaultLocale, "zh"} {
		if _, ok := seen[locale]; !ok {
			seen[locale] = struct{}{}
			locales = append(locales, locale)
		}
	}
	return locales
}

func renderSubject(name string, subject string, data emailtemplatesetting.RenderData) (string, error) {
	tpl, err := texttemplate.New(name + "_subject").Parse(subject)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	if err = tpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

func renderHTML(name string, body string, data emailtemplatesetting.RenderData) (string, error) {
	tpl, err := htmltemplate.New(name + "_html").Parse(body)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	if err = tpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}
