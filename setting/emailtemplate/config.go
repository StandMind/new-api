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

package emailtemplatesetting

import (
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"

	"github.com/QuantumNous/new-api/common"
)

const (
	OptionKey             = "EmailTemplates"
	TemplateVerification  = "verification"
	TemplatePasswordReset = "password_reset"
	DefaultLocale         = "en"
)

var supportedLocales = map[string]struct{}{
	"en": {},
	"zh": {},
	"fr": {},
	"ru": {},
	"ja": {},
	"vi": {},
}

type Template struct {
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type Settings struct {
	DefaultLocale string                         `json:"default_locale"`
	SupportEmail  string                         `json:"support_email,omitempty"`
	Templates     map[string]map[string]Template `json:"templates"`
}

type RenderData struct {
	SystemName   string
	Email        string
	Code         string
	ResetLink    string
	SupportEmail string
	ValidMinutes int
}

func SupportedLocales() []string {
	return []string{"en", "zh", "fr", "ru", "ja", "vi"}
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

func DefaultSettings() Settings {
	return Settings{
		DefaultLocale: DefaultLocale,
		SupportEmail:  "",
		Templates: map[string]map[string]Template{
			TemplateVerification: {
				"en": {
					Subject: "{{.SystemName}} email verification",
					HTML: verificationHTML(
						"Email verification",
						"Use the code below to verify your email address for {{.SystemName}}.",
						"Verification code",
						"This code expires in {{.ValidMinutes}} minutes.",
						"If you did not request this email, you can safely ignore it.",
					),
				},
				"zh": {
					Subject: "{{.SystemName}} 邮箱验证",
					HTML: verificationHTML(
						"邮箱验证",
						"请使用下方验证码完成 {{.SystemName}} 邮箱验证。",
						"验证码",
						"验证码将在 {{.ValidMinutes}} 分钟后失效。",
						"如果这不是你本人操作，请忽略此邮件。",
					),
				},
				"fr": {
					Subject: "Vérification de l’adresse e-mail {{.SystemName}}",
					HTML: verificationHTML(
						"Vérification de l’adresse e-mail",
						"Utilisez le code ci-dessous pour vérifier votre adresse e-mail pour {{.SystemName}}.",
						"Code de vérification",
						"Ce code expire dans {{.ValidMinutes}} minutes.",
						"Si vous n’êtes pas à l’origine de cette demande, vous pouvez ignorer cet e-mail.",
					),
				},
				"ru": {
					Subject: "Подтверждение электронной почты {{.SystemName}}",
					HTML: verificationHTML(
						"Подтверждение электронной почты",
						"Используйте код ниже, чтобы подтвердить адрес электронной почты для {{.SystemName}}.",
						"Код подтверждения",
						"Код действует {{.ValidMinutes}} минут.",
						"Если вы не запрашивали это письмо, просто проигнорируйте его.",
					),
				},
				"ja": {
					Subject: "{{.SystemName}} メール認証",
					HTML: verificationHTML(
						"メール認証",
						"以下のコードを使用して {{.SystemName}} のメールアドレスを認証してください。",
						"認証コード",
						"このコードは {{.ValidMinutes}} 分後に期限切れになります。",
						"このメールに心当たりがない場合は無視してください。",
					),
				},
				"vi": {
					Subject: "Xác minh email {{.SystemName}}",
					HTML: verificationHTML(
						"Xác minh email",
						"Dùng mã bên dưới để xác minh địa chỉ email của bạn cho {{.SystemName}}.",
						"Mã xác minh",
						"Mã này hết hạn sau {{.ValidMinutes}} phút.",
						"Nếu bạn không yêu cầu email này, bạn có thể bỏ qua.",
					),
				},
			},
			TemplatePasswordReset: {
				"en": {
					Subject: "{{.SystemName}} password reset",
					HTML: resetHTML(
						"Password reset",
						"We received a request to reset the password for your {{.SystemName}} account.",
						"Reset password",
						"This link expires in {{.ValidMinutes}} minutes.",
						"If the button does not work, copy and paste this link into your browser:",
						"If you did not request a password reset, you can safely ignore this email.",
					),
				},
				"zh": {
					Subject: "{{.SystemName}} 密码重置",
					HTML: resetHTML(
						"密码重置",
						"我们收到了重置你的 {{.SystemName}} 账户密码的请求。",
						"重置密码",
						"该链接将在 {{.ValidMinutes}} 分钟后失效。",
						"如果按钮无法点击，请复制下方链接到浏览器打开：",
						"如果这不是你本人操作，请忽略此邮件。",
					),
				},
				"fr": {
					Subject: "Réinitialisation du mot de passe {{.SystemName}}",
					HTML: resetHTML(
						"Réinitialisation du mot de passe",
						"Nous avons reçu une demande de réinitialisation du mot de passe de votre compte {{.SystemName}}.",
						"Réinitialiser le mot de passe",
						"Ce lien expire dans {{.ValidMinutes}} minutes.",
						"Si le bouton ne fonctionne pas, copiez ce lien dans votre navigateur :",
						"Si vous n’êtes pas à l’origine de cette demande, vous pouvez ignorer cet e-mail.",
					),
				},
				"ru": {
					Subject: "Сброс пароля {{.SystemName}}",
					HTML: resetHTML(
						"Сброс пароля",
						"Мы получили запрос на сброс пароля для вашей учетной записи {{.SystemName}}.",
						"Сбросить пароль",
						"Ссылка действует {{.ValidMinutes}} минут.",
						"Если кнопка не работает, скопируйте эту ссылку в браузер:",
						"Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.",
					),
				},
				"ja": {
					Subject: "{{.SystemName}} パスワードリセット",
					HTML: resetHTML(
						"パスワードリセット",
						"{{.SystemName}} アカウントのパスワードリセット要求を受け付けました。",
						"パスワードをリセット",
						"このリンクは {{.ValidMinutes}} 分後に期限切れになります。",
						"ボタンが機能しない場合は、次のリンクをブラウザに貼り付けてください:",
						"このリセットに心当たりがない場合は、このメールを無視してください。",
					),
				},
				"vi": {
					Subject: "Đặt lại mật khẩu {{.SystemName}}",
					HTML: resetHTML(
						"Đặt lại mật khẩu",
						"Chúng tôi đã nhận được yêu cầu đặt lại mật khẩu cho tài khoản {{.SystemName}} của bạn.",
						"Đặt lại mật khẩu",
						"Liên kết này hết hạn sau {{.ValidMinutes}} phút.",
						"Nếu nút không hoạt động, hãy sao chép liên kết này vào trình duyệt:",
						"Nếu bạn không yêu cầu đặt lại mật khẩu, bạn có thể bỏ qua email này.",
					),
				},
			},
		},
	}
}

func DefaultSettingsJSONString() string {
	bytes, err := common.Marshal(DefaultSettings())
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func ParseSettings(raw string) (Settings, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultSettings(), nil
	}
	var settings Settings
	if err := common.UnmarshalJsonStr(raw, &settings); err != nil {
		return Settings{}, err
	}
	if NormalizeLocale(settings.DefaultLocale) == "" {
		settings.DefaultLocale = DefaultLocale
	} else {
		settings.DefaultLocale = NormalizeLocale(settings.DefaultLocale)
	}
	if settings.Templates == nil {
		settings.Templates = map[string]map[string]Template{}
	}
	return settings, nil
}

func ValidateSettingsJSON(raw string) error {
	settings, err := ParseSettings(raw)
	if err != nil {
		return err
	}

	sample := RenderData{
		SystemName:   "New API",
		Email:        "user@example.com",
		Code:         "123456",
		ResetLink:    "https://example.com/user/reset?token=example",
		SupportEmail: "support@example.com",
		ValidMinutes: 10,
	}

	for templateName, byLocale := range settings.Templates {
		for locale, tpl := range byLocale {
			if NormalizeLocale(locale) == "" {
				return fmt.Errorf("unsupported locale %q in %s", locale, templateName)
			}
			if strings.TrimSpace(tpl.Subject) == "" && strings.TrimSpace(tpl.HTML) == "" {
				continue
			}
			if strings.TrimSpace(tpl.Subject) == "" || strings.TrimSpace(tpl.HTML) == "" {
				return fmt.Errorf("%s.%s must define both subject and html", templateName, locale)
			}
			if err := validateTemplate(templateName, locale, tpl, sample); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTemplate(templateName string, locale string, tpl Template, sample RenderData) error {
	subject, err := texttemplate.New(templateName + "_subject").Parse(tpl.Subject)
	if err != nil {
		return fmt.Errorf("%s.%s subject template is invalid: %w", templateName, locale, err)
	}
	var subjectOut strings.Builder
	if err = subject.Execute(&subjectOut, sample); err != nil {
		return fmt.Errorf("%s.%s subject template failed: %w", templateName, locale, err)
	}

	body, err := htmltemplate.New(templateName + "_html").Parse(tpl.HTML)
	if err != nil {
		return fmt.Errorf("%s.%s html template is invalid: %w", templateName, locale, err)
	}
	var bodyOut strings.Builder
	if err = body.Execute(&bodyOut, sample); err != nil {
		return fmt.Errorf("%s.%s html template failed: %w", templateName, locale, err)
	}
	return nil
}

func verificationHTML(title, intro, codeLabel, expiry, ignore string) string {
	return emailLayout(title, intro, fmt.Sprintf(
		`<div style="margin:24px 0;">
  <div style="font-size:13px;color:#64748b;margin-bottom:8px;">%s</div>
  <div style="font-size:32px;letter-spacing:6px;font-weight:700;color:#0f172a;">{{.Code}}</div>
</div>
<p style="margin:0 0 12px;color:#475569;line-height:1.6;">%s</p>
<p style="margin:0;color:#64748b;line-height:1.6;">%s</p>`, codeLabel, expiry, ignore))
}

func resetHTML(title, intro, button, expiry, copyHint, ignore string) string {
	return emailLayout(title, intro, fmt.Sprintf(
		`<div style="margin:28px 0;">
  <a href="{{.ResetLink}}" style="display:inline-block;background:#0f172a;color:#ffffff;text-decoration:none;padding:12px 18px;border-radius:8px;font-weight:600;">%s</a>
</div>
<p style="margin:0 0 12px;color:#475569;line-height:1.6;">%s</p>
<p style="word-break:break-all;margin:0 0 12px;color:#334155;line-height:1.6;"><a href="{{.ResetLink}}" style="color:#2563eb;">{{.ResetLink}}</a></p>
<p style="margin:0 0 12px;color:#475569;line-height:1.6;">%s</p>
<p style="margin:0;color:#64748b;line-height:1.6;">%s</p>`, button, copyHint, expiry, ignore))
}

func emailLayout(title, intro, body string) string {
	return fmt.Sprintf(`<!doctype html>
<html>
  <body style="margin:0;background:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;">
    <div style="max-width:560px;margin:0 auto;padding:32px 20px;">
      <div style="background:#ffffff;border:1px solid #e2e8f0;border-radius:12px;padding:28px;">
        <div style="font-size:14px;font-weight:600;color:#0f172a;margin-bottom:20px;">{{.SystemName}}</div>
        <h1 style="font-size:22px;line-height:1.35;color:#0f172a;margin:0 0 14px;">%s</h1>
        <p style="margin:0;color:#475569;line-height:1.6;">%s</p>
        %s
      </div>
      {{if .SupportEmail}}<p style="font-size:12px;color:#64748b;text-align:center;margin:16px 0 0;">{{.SupportEmail}}</p>{{end}}
    </div>
  </body>
</html>`, title, intro, body)
}
