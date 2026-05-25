package seo

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	docsetting "github.com/QuantumNous/new-api/setting/documentation"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/require"
)

func withOption(t *testing.T, key string, value string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	previous, existed := common.OptionMap[key]
	common.OptionMap[key] = value
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if existed {
			common.OptionMap[key] = previous
			return
		}
		delete(common.OptionMap, key)
	})
}

func TestResolveBaseURLUsesForwardedHeadersWhenConfiguredAddressIsLocal(t *testing.T) {
	previous := system_setting.ServerAddress
	system_setting.ServerAddress = "http://localhost:3000"
	t.Cleanup(func() {
		system_setting.ServerAddress = previous
	})

	request := httptest.NewRequest("GET", "http://internal/sitemap.xml", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Forwarded-Host", "example.com")

	require.Equal(t, "https://example.com", ResolveBaseURL(request))
}

func TestBuildRobotsTxtReferencesCanonicalSitemap(t *testing.T) {
	body := BuildRobotsTxt("https://example.com/")

	require.Contains(t, body, "User-agent: *")
	require.Contains(t, body, "Allow: /")
	require.Contains(t, body, "Sitemap: https://example.com/sitemap.xml")
}

func TestRenderSitemapXMLEscapesLocations(t *testing.T) {
	body, err := renderSitemapXML([]sitemapEntry{{
		Loc:        "https://example.com/docs/a&b",
		ChangeFreq: changeWeekly,
		Priority:   "0.8",
	}})

	require.NoError(t, err)
	xmlBody := string(body)
	require.True(t, strings.HasPrefix(xmlBody, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"))
	require.Contains(t, xmlBody, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	require.Contains(t, xmlBody, "https://example.com/docs/a&amp;b")
}

func TestBuilderAddsDocumentationConfiguredPages(t *testing.T) {
	withOption(t, docsetting.OptionKey, `{
		"enabled": true,
		"content_dir": "data/docs",
		"default_locale": "en",
		"default_slug": "chat",
		"nav": [
			{"slug": "chat", "title": {"en": "Chat"}},
			{"title": {"en": "Gemini"}, "children": [
				{"slug": "gemini/generate", "title": {"en": "Generate"}}
			]}
		],
		"pages": [
			{"path": "/contact", "slug": "contact", "title": {"en": "Contact"}},
			{"path": "/terms", "slug": "terms", "title": {"en": "Terms"}}
		]
	}`)

	builder := newSitemapBuilder("https://example.com")
	builder.addDocumentationPages()

	locations := make(map[string]bool, len(builder.entries))
	for _, entry := range builder.entries {
		locations[entry.Loc] = true
	}

	require.True(t, locations["https://example.com/docs"])
	require.True(t, locations["https://example.com/docs/chat"])
	require.True(t, locations["https://example.com/docs/gemini/generate"])
	require.True(t, locations["https://example.com/contact"])
	require.True(t, locations["https://example.com/terms"])
}

func TestBuilderEscapesModelDetailPathSegments(t *testing.T) {
	builder := newSitemapBuilder("https://example.com")
	builder.addModelDetailPages([]model.Pricing{{ModelName: "provider/model"}})

	require.Len(t, builder.entries, 1)
	require.Equal(t, "https://example.com/pricing/provider%2Fmodel", builder.entries[0].Loc)
}

func TestAccessModuleRequiresPublicVisibility(t *testing.T) {
	withOption(t, "HeaderNavModules", `{"pricing":{"enabled":true,"requireAuth":true},"rankings":false}`)

	require.False(t, isAccessModulePublic("pricing", navAccess{Enabled: true}))
	require.False(t, isAccessModulePublic("rankings", navAccess{Enabled: true}))
	require.True(t, isAccessModulePublic("unknown", navAccess{Enabled: true}))
}
