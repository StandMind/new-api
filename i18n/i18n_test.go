package i18n

import (
	"fmt"
	"net/http/httptest"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
)

var catalogTemplateArgumentPattern = regexp.MustCompile(`\{\{\.[A-Za-z0-9_]+\}\}`)

func loadTestCatalog(t *testing.T, lang string) map[string]string {
	t.Helper()
	data, err := localeFS.ReadFile("locales/" + lang + ".yaml")
	require.NoError(t, err)
	catalog := make(map[string]string)
	require.NoError(t, yaml.Unmarshal(data, &catalog))
	return catalog
}

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "en-US", want: LangEn, ok: true},
		{input: "es_MX", want: LangEs, ok: true},
		{input: "fr-FR", want: LangFr, ok: true},
		{input: "ja", want: LangJa, ok: true},
		{input: "ru-RU", want: LangRu, ok: true},
		{input: "vi-VN", want: LangVi, ok: true},
		{input: "zhCN", want: LangZhCN, ok: true},
		{input: "zh-Hans-SG", want: LangZhCN, ok: true},
		{input: "zhTW", want: LangZhTW, ok: true},
		{input: "zh-Hant-HK", want: LangZhTW, ok: true},
		{input: "de-DE", ok: false},
		{input: "", ok: false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, ok := NormalizeLanguage(test.input)
			assert.Equal(t, test.ok, ok)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestParseAcceptLanguageUsesSupportedWeightedPreference(t *testing.T) {
	assert.Equal(t, LangFr, ParseAcceptLanguage("de-DE;q=1, fr-FR;q=0.9, en;q=0.8"))
	assert.Equal(t, LangJa, ParseAcceptLanguage("en;q=0.2, ja-JP;q=0.9"))
	assert.Equal(t, LangZhTW, ParseAcceptLanguage("zh-Hant-HK,zh;q=0.8"))
	assert.Equal(t, DefaultLang, ParseAcceptLanguage("de-DE,*;q=0.5"))
	assert.Equal(t, DefaultLang, ParseAcceptLanguage("invalid;q=not-a-number"))
	assert.Equal(t, DefaultLang, ParseAcceptLanguage("fr;q=0, en;q=0.5"))
}

func TestParseAcceptLanguageBoundsUntrustedHeaders(t *testing.T) {
	tooManyTags := strings.Repeat("de-DE,", maxAcceptLanguageTags) + "fr"
	assert.Equal(t, DefaultLang, ParseAcceptLanguage(tooManyTags))
	assert.Equal(t, DefaultLang, ParseAcceptLanguage(strings.Repeat("x", maxAcceptLanguageBytes*4)))
}

func TestGetLangFromContextPriorityAndSingleUserLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousLoader := userLangLoaderFunc
	t.Cleanup(func() { userLangLoaderFunc = previousLoader })

	t.Run("user setting wins", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Accept-Language", "fr")
		common.SetContextKey(c, constant.ContextKeyUserSetting, dto.UserSetting{Language: "zhTW"})
		assert.Equal(t, LangZhTW, GetLangFromContext(c))
	})

	t.Run("loaded empty user setting falls back without another lookup", func(t *testing.T) {
		var calls atomic.Int32
		SetUserLangLoader(func(int) string {
			calls.Add(1)
			return LangRu
		})
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Accept-Language", "fr")
		c.Set("id", 42)
		common.SetContextKey(c, constant.ContextKeyUserSetting, dto.UserSetting{})
		common.SetContextKey(c, constant.ContextKeyUserLanguageLookupDone, true)

		assert.Equal(t, LangFr, GetLangFromContext(c))
		assert.Zero(t, calls.Load())
	})

	t.Run("session language wins", func(t *testing.T) {
		var calls atomic.Int32
		SetUserLangLoader(func(int) string {
			calls.Add(1)
			return LangRu
		})
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Accept-Language", "fr")
		common.SetContextKey(c, constant.ContextKeyUserLanguage, "ja")
		common.SetContextKey(c, constant.ContextKeyUserLanguageLookupDone, true)
		assert.Equal(t, LangJa, GetLangFromContext(c))
		assert.Zero(t, calls.Load())
	})

	t.Run("legacy lookup occurs once", func(t *testing.T) {
		var calls atomic.Int32
		SetUserLangLoader(func(userID int) string {
			calls.Add(1)
			assert.Equal(t, 42, userID)
			return "ru"
		})
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Set("id", 42)

		assert.Equal(t, LangRu, GetLangFromContext(c))
		assert.Equal(t, LangRu, GetLangFromContext(c))
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("resolved language is cached for the request", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Accept-Language", "fr")
		common.SetContextKey(c, constant.ContextKeyUserSetting, dto.UserSetting{Language: "ja"})

		assert.Equal(t, LangJa, GetLangFromContext(c))
		common.SetContextKey(c, constant.ContextKeyUserSetting, dto.UserSetting{Language: "ru"})
		c.Request.Header.Set("Accept-Language", "vi")
		assert.Equal(t, LangJa, GetLangFromContext(c))
	})
}

func TestCatalogsHaveMatchingKeysAndTemplateArguments(t *testing.T) {
	require.NoError(t, Init())
	english := loadTestCatalog(t, LangEn)
	require.NotEmpty(t, english)

	for _, lang := range SupportedLanguages() {
		catalog := loadTestCatalog(t, lang)
		assert.Len(t, catalog, len(english), lang)
		for key, englishMessage := range english {
			localizedMessage, ok := catalog[key]
			if !assert.True(t, ok, "%s is missing %s", lang, key) {
				continue
			}
			assert.NotEmpty(t, localizedMessage, "%s has an empty %s", lang, key)
			assert.ElementsMatch(
				t,
				catalogTemplateArgumentPattern.FindAllString(englishMessage, -1),
				catalogTemplateArgumentPattern.FindAllString(localizedMessage, -1),
				"%s template arguments differ for %s",
				lang,
				key,
			)
		}
	}
}

func TestTranslateIsSafeAcrossLanguages(t *testing.T) {
	require.NoError(t, Init())
	tests := []struct {
		lang     string
		expected string
	}{
		{lang: LangEn, expected: "Invalid parameters"},
		{lang: LangEs, expected: "Parámetros no válidos"},
		{lang: LangFr, expected: "Paramètres invalides"},
		{lang: LangJa, expected: "無効なパラメータ"},
		{lang: LangRu, expected: "Неверные параметры"},
		{lang: LangVi, expected: "Thông số không hợp lệ"},
		{lang: LangZhCN, expected: "无效的参数"},
		{lang: LangZhTW, expected: "無效的參數"},
	}

	var wg sync.WaitGroup
	const goroutineCount = 128
	const iterationsPerGoroutine = 200
	for worker := 0; worker < goroutineCount; worker++ {
		test := tests[worker%len(tests)]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterationsPerGoroutine {
				assert.Equal(t, test.expected, Translate(test.lang, MsgInvalidParams))
			}
		}()
	}
	wg.Wait()
}

func TestUnknownLanguagesDoNotGrowLocalizerSet(t *testing.T) {
	require.NoError(t, Init())
	initialCount := len(localizers)
	require.Equal(t, len(supportedLanguages), initialCount)

	for index := range 10_000 {
		_ = Translate(fmt.Sprintf("unknown-%d", index), MsgInvalidParams)
	}

	assert.Len(t, localizers, initialCount)
}

func TestSustainedLocalizationKeepsHeapAndGoroutinesStable(t *testing.T) {
	require.NoError(t, Init())
	languages := SupportedLanguages()
	args := map[string]any{"Model": "gpt-5", "Group": "default"}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	beforeGoroutines := runtime.NumGoroutine()

	for wave := range 5 {
		for index := range 20_000 {
			message := Translate(languages[(wave+index)%len(languages)], MsgDistributorNoAvailableChannel, args)
			if message == "" {
				t.Fatal("localized message is empty")
			}
		}
		runtime.GC()
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	afterGoroutines := runtime.NumGoroutine()
	assert.LessOrEqual(t, after.HeapAlloc, before.HeapAlloc+4<<20)
	assert.LessOrEqual(t, afterGoroutines, beforeGoroutines+2)
	assert.Len(t, localizers, len(supportedLanguages))
	t.Logf(
		"heap_before=%d heap_after=%d gc_delta=%d goroutines_before=%d goroutines_after=%d",
		before.HeapAlloc,
		after.HeapAlloc,
		after.NumGC-before.NumGC,
		beforeGoroutines,
		afterGoroutines,
	)
}

func TestMissingTranslationReturnsGenericEnglishMessage(t *testing.T) {
	assert.Equal(t, fallbackMessage, Translate(LangZhCN, "missing.public.key"))
}

func BenchmarkTranslate(b *testing.B) {
	require.NoError(b, Init())
	args := map[string]any{"Model": "gpt-5", "Group": "default"}
	b.ReportAllocs()
	for b.Loop() {
		_ = Translate(LangEn, MsgDistributorNoAvailableChannel, args)
	}
}

func BenchmarkParseAcceptLanguage(b *testing.B) {
	for b.Loop() {
		_ = ParseAcceptLanguage("de-DE;q=1, fr-FR;q=0.9, en;q=0.8")
	}
}

func legacyNormalizeLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(value, "zh-tw") {
		return LangZhTW
	}
	if strings.HasPrefix(value, "zh") {
		return LangZhCN
	}
	if strings.HasPrefix(value, "en") {
		return LangEn
	}
	return LangEn
}

// legacyDetectLanguage mirrors the complete pre-change middleware path so the
// benchmark can compare success requests without a separate checkout.
func legacyDetectLanguage(header string) string {
	if header == "" {
		return LangEn
	}
	parts := strings.Split(header, ",")
	if len(parts) == 0 {
		return LangEn
	}
	value := strings.TrimSpace(parts[0])
	if index := strings.IndexByte(value, ';'); index > 0 {
		value = value[:index]
	}
	lang := legacyNormalizeLanguage(value)
	normalized := legacyNormalizeLanguage(lang)
	for _, supported := range []string{LangZhCN, LangZhTW, LangEn} {
		if normalized == supported {
			return lang
		}
	}
	return LangEn
}

func BenchmarkSuccessLanguageDetection(b *testing.B) {
	b.Run("before", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = legacyDetectLanguage("en-US")
		}
	})
	b.Run("after", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = ParseAcceptLanguage("en-US")
		}
	})
}

var benchmarkTrafficBytes uint64

func BenchmarkLocalizedErrorTrafficProfiles(b *testing.B) {
	headers := [...]string{LangEn, LangEs, LangFr, LangJa, LangRu, LangVi, LangZhCN, LangZhTW}
	args := map[string]any{"Model": "gpt-5", "Group": "default"}

	runProfile := func(b *testing.B, errorEvery int) {
		b.Helper()
		b.ReportAllocs()
		var total uint64
		index := 0
		for b.Loop() {
			lang := ParseAcceptLanguage(headers[index%len(headers)])
			if errorEvery > 0 && index%errorEvery == 0 {
				total += uint64(len(Translate(lang, MsgDistributorNoAvailableChannel, args)))
			}
			index++
		}
		benchmarkTrafficBytes = total
	}

	b.Run("normal_success", func(b *testing.B) { runProfile(b, 0) })
	b.Run("ten_percent_errors", func(b *testing.B) { runProfile(b, 10) })
	b.Run("sustained_errors", func(b *testing.B) { runProfile(b, 1) })
	b.Run("parallel_eight_languages", func(b *testing.B) {
		b.ReportAllocs()
		var total atomic.Uint64
		b.RunParallel(func(pb *testing.PB) {
			var localTotal uint64
			index := 0
			for pb.Next() {
				lang := ParseAcceptLanguage(headers[index%len(headers)])
				localTotal += uint64(len(Translate(lang, MsgDistributorNoAvailableChannel, args)))
				index++
			}
			total.Add(localTotal)
		})
		benchmarkTrafficBytes = total.Load()
	})
}
