package seo

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	docservice "github.com/QuantumNous/new-api/service/documentation"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const (
	changeDaily   = "daily"
	changeWeekly  = "weekly"
	changeMonthly = "monthly"
)

type navAccess struct {
	Enabled     bool
	RequireAuth bool
}

type sitemapXML struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type sitemapEntry struct {
	Path       string
	Loc        string
	ChangeFreq string
	Priority   string
}

type sitemapBuilder struct {
	baseURL string
	seen    map[string]struct{}
	entries []sitemapEntry
}

func GenerateSitemapXML(r *http.Request) ([]byte, error) {
	baseURL := ResolveBaseURL(r)
	entries := buildSitemapEntries(baseURL)
	return renderSitemapXML(entries)
}

func BuildRobotsTxt(baseURL string) string {
	baseURL = NormalizeBaseURL(baseURL)
	if baseURL == "" {
		baseURL = "https://example.com"
	}
	return fmt.Sprintf("User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", baseURL)
}

func buildSitemapEntries(baseURL string) []sitemapEntry {
	builder := newSitemapBuilder(baseURL)

	builder.addPath("/", changeDaily, "1.0")

	if isBooleanModulePublic("about", true) {
		builder.addPath("/about", changeMonthly, "0.6")
	}

	if isAccessModulePublic("pricing", navAccess{Enabled: true}) {
		builder.addPath("/pricing", changeDaily, "0.9")
		builder.addModelDetailPages(model.GetPricing())
	}

	if isAccessModulePublic("rankings", navAccess{Enabled: true}) {
		builder.addPath("/rankings", changeDaily, "0.8")
	}

	if isBooleanModulePublic("docs", true) {
		builder.addDocumentationPages()
	}

	builder.addPath("/terms", changeMonthly, "0.5")
	builder.addPath("/privacy-policy", changeMonthly, "0.5")

	builder.sort()
	return builder.entries
}

func renderSitemapXML(entries []sitemapEntry) ([]byte, error) {
	doc := sitemapXML{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]sitemapURL, 0, len(entries)),
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.Loc) == "" {
			continue
		}
		doc.URLs = append(doc.URLs, sitemapURL{
			Loc:        entry.Loc,
			ChangeFreq: entry.ChangeFreq,
			Priority:   entry.Priority,
		})
	}

	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func ResolveBaseURL(r *http.Request) string {
	if configured := configuredBaseURL(); configured != "" {
		return configured
	}
	if r == nil {
		return "http://localhost:3000"
	}

	host := firstHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		return "http://localhost:3000"
	}

	scheme := firstHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = firstHeaderValue(r.Header.Get("X-Forwarded-Protocol"))
	}
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	return NormalizeBaseURL(scheme + "://" + host)
}

func NormalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func configuredBaseURL() string {
	baseURL := NormalizeBaseURL(system_setting.ServerAddress)
	if baseURL == "" {
		return ""
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return ""
	}
	return baseURL
}

func firstHeaderValue(raw string) string {
	if raw == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(raw, ",")[0])
}

func newSitemapBuilder(baseURL string) *sitemapBuilder {
	baseURL = NormalizeBaseURL(baseURL)
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	return &sitemapBuilder{
		baseURL: baseURL,
		seen:    map[string]struct{}{},
		entries: []sitemapEntry{},
	}
}

func (b *sitemapBuilder) addPath(pagePath string, changeFreq string, priority string) {
	normalized := normalizePublicPath(pagePath)
	if normalized == "" {
		return
	}
	if _, ok := b.seen[normalized]; ok {
		return
	}
	b.seen[normalized] = struct{}{}
	b.entries = append(b.entries, sitemapEntry{
		Path:       normalized,
		Loc:        b.baseURL + normalized,
		ChangeFreq: changeFreq,
		Priority:   priority,
	})
}

func (b *sitemapBuilder) addEncodedPath(pagePath string, changeFreq string, priority string) {
	if pagePath == "" || pagePath[0] != '/' {
		return
	}
	if _, ok := b.seen[pagePath]; ok {
		return
	}
	b.seen[pagePath] = struct{}{}
	b.entries = append(b.entries, sitemapEntry{
		Path:       pagePath,
		Loc:        b.baseURL + pagePath,
		ChangeFreq: changeFreq,
		Priority:   priority,
	})
}

func (b *sitemapBuilder) addDocumentationPages() {
	config, err := docservice.GetConfig("")
	if err != nil || !config.Enabled {
		return
	}

	b.addPath("/docs", changeWeekly, "0.9")
	for _, slug := range flattenDocumentationSlugs(config.Nav) {
		b.addPath("/docs/"+slug, changeWeekly, "0.8")
	}
	for _, page := range config.Pages {
		b.addPath(page.Path, changeMonthly, "0.5")
	}
}

func (b *sitemapBuilder) addModelDetailPages(pricing []model.Pricing) {
	names := make([]string, 0, len(pricing))
	for _, item := range pricing {
		name := strings.TrimSpace(item.ModelName)
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		b.addEncodedPath("/pricing/"+url.PathEscape(name), changeWeekly, "0.7")
	}
}

func (b *sitemapBuilder) sort() {
	sort.SliceStable(b.entries, func(i int, j int) bool {
		if b.entries[i].Priority == b.entries[j].Priority {
			return b.entries[i].Path < b.entries[j].Path
		}
		return b.entries[i].Priority > b.entries[j].Priority
	})
}

func normalizePublicPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	cleaned := path.Clean(value)
	if cleaned == "." {
		return ""
	}
	if cleaned == "/" {
		return "/"
	}
	if strings.HasPrefix(cleaned, "/api/") || strings.HasPrefix(cleaned, "/v1/") || strings.HasPrefix(cleaned, "/v1beta/") {
		return ""
	}
	if strings.Contains(cleaned, "\x00") || strings.Contains(cleaned, "\\") {
		return ""
	}
	return cleaned
}

func flattenDocumentationSlugs(items []docservice.NavItem) []string {
	slugs := make([]string, 0)
	var walk func([]docservice.NavItem)
	walk = func(nav []docservice.NavItem) {
		for _, item := range nav {
			if slug := strings.TrimSpace(item.Slug); slug != "" {
				slugs = append(slugs, slug)
			}
			walk(item.Children)
		}
	}
	walk(items)
	sort.Strings(slugs)
	return slugs
}

func isBooleanModulePublic(module string, fallback bool) bool {
	raw, ok := headerNavModuleValue(module)
	if !ok {
		return fallback
	}
	return parseNavBool(raw, fallback)
}

func isAccessModulePublic(module string, fallback navAccess) bool {
	raw, ok := headerNavModuleValue(module)
	if !ok {
		return fallback.Enabled && !fallback.RequireAuth
	}
	access := parseNavAccess(raw, fallback)
	return access.Enabled && !access.RequireAuth
}

func headerNavModuleValue(module string) (any, bool) {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap["HeaderNavModules"]
	common.OptionMapRWMutex.RUnlock()
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}

	var parsed map[string]any
	if err := common.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, false
	}
	value, ok := parsed[module]
	return value, ok
}

func parseNavAccess(raw any, fallback navAccess) navAccess {
	switch value := raw.(type) {
	case map[string]any:
		access := fallback
		if enabled, ok := value["enabled"]; ok {
			access.Enabled = parseNavBool(enabled, fallback.Enabled)
		}
		if requireAuth, ok := value["requireAuth"]; ok {
			access.RequireAuth = parseNavBool(requireAuth, fallback.RequireAuth)
		}
		return access
	default:
		return navAccess{
			Enabled:     parseNavBool(raw, fallback.Enabled),
			RequireAuth: fallback.RequireAuth,
		}
	}
}

func parseNavBool(raw any, fallback bool) bool {
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1":
			return true
		case "false", "0":
			return false
		default:
			return fallback
		}
	case float64:
		if value == 1 {
			return true
		}
		if value == 0 {
			return false
		}
		return fallback
	case int:
		if value == 1 {
			return true
		}
		if value == 0 {
			return false
		}
		return fallback
	default:
		return fallback
	}
}
