package i18n

import (
	"embed"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
)

const (
	LangEn      = "en"
	LangEs      = "es"
	LangFr      = "fr"
	LangJa      = "ja"
	LangRu      = "ru"
	LangVi      = "vi"
	LangZhCN    = "zh-CN"
	LangZhTW    = "zh-TW"
	DefaultLang = LangEn

	maxAcceptLanguageBytes = 512
	maxAcceptLanguageTags  = 16
	fallbackMessage        = "Request failed"
)

var supportedLanguages = [...]string{
	LangEn,
	LangEs,
	LangFr,
	LangJa,
	LangRu,
	LangVi,
	LangZhCN,
	LangZhTW,
}

//go:embed locales/*.yaml
var localeFS embed.FS

var (
	bundle     *goi18n.Bundle
	localizers map[string]*goi18n.Localizer
	initOnce   sync.Once
	initErr    error

	missingTranslationCount atomic.Uint64
	missingTranslationLogAt atomic.Int64
)

func init() {
	_ = Init()
}

// Init loads every supported locale before the HTTP server starts. The bundle
// and localizer map are immutable after this function returns.
func Init() error {
	initOnce.Do(func() {
		bundle = goi18n.NewBundle(language.English)
		bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

		for _, lang := range supportedLanguages {
			if _, err := bundle.LoadMessageFileFS(localeFS, "locales/"+lang+".yaml"); err != nil {
				initErr = err
				return
			}
		}

		localizers = make(map[string]*goi18n.Localizer, len(supportedLanguages))
		for _, lang := range supportedLanguages {
			localizers[lang] = goi18n.NewLocalizer(bundle, lang, DefaultLang)
		}
		common.TranslateMessage = T
	})
	return initErr
}

// NormalizeLanguage converts supported aliases and BCP-47 variants to the
// canonical language codes used by the application.
func NormalizeLanguage(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	if idx := strings.IndexByte(value, ';'); idx >= 0 {
		value = value[:idx]
	}
	value = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))

	switch {
	case value == "zh-tw", value == "zhtw", value == "zh-hk", value == "zh-mo", strings.HasPrefix(value, "zh-hant"):
		return LangZhTW, true
	case value == "zh", value == "zh-cn", value == "zhcn", value == "zh-sg", strings.HasPrefix(value, "zh-hans"):
		return LangZhCN, true
	case value == LangEn || strings.HasPrefix(value, LangEn+"-"):
		return LangEn, true
	case value == LangEs || strings.HasPrefix(value, LangEs+"-"):
		return LangEs, true
	case value == LangFr || strings.HasPrefix(value, LangFr+"-"):
		return LangFr, true
	case value == LangJa || strings.HasPrefix(value, LangJa+"-"):
		return LangJa, true
	case value == LangRu || strings.HasPrefix(value, LangRu+"-"):
		return LangRu, true
	case value == LangVi || strings.HasPrefix(value, LangVi+"-"):
		return LangVi, true
	default:
		return "", false
	}
}

// ParseAcceptLanguage returns the highest-priority supported language. The
// standard parser handles q-values; unsupported preferences are skipped.
func ParseAcceptLanguage(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return DefaultLang
	}
	if len(header) > maxAcceptLanguageBytes {
		header = header[:maxAcceptLanguageBytes]
	}

	if !strings.ContainsAny(header, ",;") {
		if lang, ok := NormalizeLanguage(header); ok {
			return lang
		}
		return DefaultLang
	}

	parts := strings.Split(header, ",")
	if len(parts) > maxAcceptLanguageTags {
		parts = parts[:maxAcceptLanguageTags]
	}
	tags, _, err := language.ParseAcceptLanguage(strings.Join(parts, ","))
	if err != nil {
		return DefaultLang
	}
	for _, tag := range tags {
		if lang, ok := NormalizeLanguage(tag.String()); ok {
			return lang
		}
	}
	return DefaultLang
}

// T translates a message key using the language resolved for this request.
func T(c *gin.Context, key string, args ...map[string]any) string {
	return Translate(GetLangFromContext(c), key, args...)
}

// Translate translates a message key for a canonical or aliased language.
func Translate(lang, key string, args ...map[string]any) string {
	if err := Init(); err != nil {
		return fallbackMessage
	}
	canonical, ok := NormalizeLanguage(lang)
	if !ok {
		canonical = DefaultLang
	}
	loc := localizers[canonical]
	if loc == nil {
		loc = localizers[DefaultLang]
	}

	config := &goi18n.LocalizeConfig{MessageID: key}
	if len(args) > 0 && args[0] != nil {
		config.TemplateData = args[0]
	}
	message, err := loc.Localize(config)
	if err != nil || message == "" {
		missingTranslationCount.Add(1)
		now := time.Now().Unix()
		previous := missingTranslationLogAt.Load()
		if now-previous >= 60 && missingTranslationLogAt.CompareAndSwap(previous, now) {
			common.SysError(fmt.Sprintf("missing public translation language=%s key=%s", canonical, common.LocalLogPreview(key)))
		}
		return fallbackMessage
	}
	return message
}

// userLangLoaderFunc loads the persisted preference for legacy sessions that
// do not yet contain a language. It is registered once during startup.
var userLangLoaderFunc func(userID int) string

func SetUserLangLoader(loader func(userID int) string) {
	userLangLoaderFunc = loader
}

// GetLangFromContext resolves language without performing more than one user
// cache lookup per request. A user setting always wins over the request header.
func GetLangFromContext(c *gin.Context) string {
	if c == nil {
		return DefaultLang
	}
	if common.GetContextKeyBool(c, constant.ContextKeyLanguageResolved) {
		if lang := common.GetContextKeyString(c, constant.ContextKeyLanguage); lang != "" {
			return lang
		}
		return DefaultLang
	}

	if userSetting, ok := common.GetContextKeyType[dto.UserSetting](c, constant.ContextKeyUserSetting); ok {
		if lang, valid := NormalizeLanguage(userSetting.Language); valid {
			return cacheResolvedLanguage(c, lang)
		}
	}
	if lang := common.GetContextKeyString(c, constant.ContextKeyUserLanguage); lang != "" {
		if canonical, valid := NormalizeLanguage(lang); valid {
			return cacheResolvedLanguage(c, canonical)
		}
	}

	if !common.GetContextKeyBool(c, constant.ContextKeyUserLanguageLookupDone) {
		common.SetContextKey(c, constant.ContextKeyUserLanguageLookupDone, true)
		if userLangLoaderFunc != nil {
			if userID := c.GetInt("id"); userID > 0 {
				if lang, valid := NormalizeLanguage(userLangLoaderFunc(userID)); valid {
					common.SetContextKey(c, constant.ContextKeyUserLanguage, lang)
					return cacheResolvedLanguage(c, lang)
				}
			}
		}
	}

	if lang := common.GetContextKeyString(c, constant.ContextKeyLanguage); lang != "" {
		if canonical, valid := NormalizeLanguage(lang); valid {
			return cacheResolvedLanguage(c, canonical)
		}
	}
	lang := ParseAcceptLanguage(c.GetHeader("Accept-Language"))
	return cacheResolvedLanguage(c, lang)
}

func cacheResolvedLanguage(c *gin.Context, lang string) string {
	common.SetContextKey(c, constant.ContextKeyLanguage, lang)
	common.SetContextKey(c, constant.ContextKeyLanguageResolved, true)
	return lang
}

func SupportedLanguages() []string {
	result := make([]string, len(supportedLanguages))
	copy(result, supportedLanguages[:])
	return result
}

func IsSupported(lang string) bool {
	_, ok := NormalizeLanguage(lang)
	return ok
}

func MissingTranslationCount() uint64 {
	return missingTranslationCount.Load()
}
