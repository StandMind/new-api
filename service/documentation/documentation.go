package docservice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	documentationsetting "github.com/QuantumNous/new-api/setting/documentation"
)

var (
	ErrDisabled = errors.New("documentation is disabled")
	ErrNotFound = errors.New("documentation page not found")
)

type NavItem struct {
	Slug        string    `json:"slug,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Children    []NavItem `json:"children,omitempty"`
}

type PageItem struct {
	Path        string `json:"path"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type DebugExample struct {
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	PathTemplate string            `json:"path_template,omitempty"`
	Auth         string            `json:"auth,omitempty"`
	Model        string            `json:"model,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         any               `json:"body,omitempty"`
}

type Config struct {
	Enabled          bool       `json:"enabled"`
	Locale           string     `json:"locale"`
	DefaultLocale    string     `json:"default_locale"`
	DefaultSlug      string     `json:"default_slug"`
	SupportedLocales []string   `json:"supported_locales"`
	Nav              []NavItem  `json:"nav"`
	Pages            []PageItem `json:"pages,omitempty"`
}

type Page struct {
	Enabled bool          `json:"enabled"`
	Slug    string        `json:"slug"`
	Title   string        `json:"title"`
	Content string        `json:"content"`
	Locale  string        `json:"locale"`
	Debug   *DebugExample `json:"debug,omitempty"`
}

type loadedPage struct {
	content string
	locale  string
}

var firstHeadingPattern = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)

func GetConfig(lang string) (Config, error) {
	settings, err := loadSettings()
	if err != nil {
		return Config{}, err
	}

	locale := resolveLocale(lang, settings.DefaultLocale)
	return Config{
		Enabled:          settings.Enabled,
		Locale:           locale,
		DefaultLocale:    settings.DefaultLocale,
		DefaultSlug:      resolveDefaultSlug(settings),
		SupportedLocales: documentationsetting.SupportedLocales(),
		Nav:              localizeNav(settings.Nav, locale, settings.DefaultLocale),
		Pages:            localizePages(settings.Pages, locale, settings.DefaultLocale),
	}, nil
}

func GetPage(slug string, lang string) (Page, error) {
	settings, err := loadSettings()
	if err != nil {
		return Page{}, err
	}
	if !settings.Enabled {
		return Page{}, ErrDisabled
	}

	if strings.TrimSpace(slug) == "" || slug == "/" {
		slug = resolveDefaultSlug(settings)
	}
	normalizedSlug, err := documentationsetting.NormalizeSlug(slug)
	if err != nil {
		return Page{}, err
	}
	contentPath := resolveContentPath(settings, normalizedSlug)

	locale := resolveLocale(lang, settings.DefaultLocale)
	page, err := readMarkdown(settings.ContentDir, contentPath, locale, settings.DefaultLocale)
	if err != nil {
		return Page{}, err
	}

	title := findNavTitle(settings.Nav, normalizedSlug, page.locale, settings.DefaultLocale)
	if title == "" {
		title = findPageTitle(settings.Pages, normalizedSlug, page.locale, settings.DefaultLocale)
	}
	if title == "" {
		title = extractFirstHeading(page.content)
	}
	if title == "" {
		title = normalizedSlug
	}

	return Page{
		Enabled: true,
		Slug:    normalizedSlug,
		Title:   title,
		Content: page.content,
		Locale:  page.locale,
		Debug:   localizeDebug(settings.DebugExamples[normalizedSlug], page.locale, settings.DefaultLocale),
	}, nil
}

func loadSettings() (documentationsetting.Settings, error) {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[documentationsetting.OptionKey]
	common.OptionMapRWMutex.RUnlock()
	return documentationsetting.ParseSettings(raw)
}

func resolveDefaultSlug(settings documentationsetting.Settings) string {
	if settings.DefaultSlug != "" {
		return settings.DefaultSlug
	}
	if slug := firstNavSlug(settings.Nav); slug != "" {
		return slug
	}
	return "index"
}

func firstNavSlug(items []documentationsetting.NavItem) string {
	for _, item := range items {
		if item.Slug != "" {
			return item.Slug
		}
		if slug := firstNavSlug(item.Children); slug != "" {
			return slug
		}
	}
	return ""
}

func resolveContentPath(settings documentationsetting.Settings, slug string) string {
	if file := findNavFile(settings.Nav, slug); file != "" {
		return file
	}
	if file := findPageFile(settings.Pages, slug); file != "" {
		return file
	}
	return slug
}

func findNavFile(items []documentationsetting.NavItem, slug string) string {
	for _, item := range items {
		if item.Slug == slug {
			return item.File
		}
		if file := findNavFile(item.Children, slug); file != "" {
			return file
		}
	}
	return ""
}

func findPageFile(items []documentationsetting.PageItem, slug string) string {
	for _, item := range items {
		if item.Slug == slug {
			return item.File
		}
	}
	return ""
}

func resolveLocale(lang string, defaultLocale string) string {
	if locale := documentationsetting.ResolveLocale(lang, defaultLocale); locale != "" {
		return locale
	}
	return documentationsetting.DefaultLocale
}

func localeFallbacks(locale string, defaultLocale string) []string {
	candidates := []string{
		documentationsetting.NormalizeLocale(locale),
		documentationsetting.NormalizeLocale(defaultLocale),
		documentationsetting.DefaultLocale,
		"zh",
	}

	seen := make(map[string]struct{}, len(candidates))
	fallbacks := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		fallbacks = append(fallbacks, candidate)
	}
	return fallbacks
}

func localizeNav(items []documentationsetting.NavItem, locale string, defaultLocale string) []NavItem {
	nav := make([]NavItem, 0, len(items))
	for _, item := range items {
		titleFallback := item.Slug
		if titleFallback == "" {
			titleFallback = "Untitled"
		}
		nav = append(nav, NavItem{
			Slug:        item.Slug,
			Title:       localizeText(item.Title, locale, defaultLocale, titleFallback),
			Description: localizeText(item.Description, locale, defaultLocale, ""),
			Children:    localizeNav(item.Children, locale, defaultLocale),
		})
	}
	return nav
}

func localizePages(items []documentationsetting.PageItem, locale string, defaultLocale string) []PageItem {
	pages := make([]PageItem, 0, len(items))
	for _, item := range items {
		titleFallback := item.Slug
		pages = append(pages, PageItem{
			Path:        item.Path,
			Slug:        item.Slug,
			Title:       localizeText(item.Title, locale, defaultLocale, titleFallback),
			Description: localizeText(item.Description, locale, defaultLocale, ""),
		})
	}
	return pages
}

func findNavTitle(items []documentationsetting.NavItem, slug string, locale string, defaultLocale string) string {
	for _, item := range items {
		if item.Slug == slug {
			return localizeText(item.Title, locale, defaultLocale, "")
		}
		if title := findNavTitle(item.Children, slug, locale, defaultLocale); title != "" {
			return title
		}
	}
	return ""
}

func findPageTitle(items []documentationsetting.PageItem, slug string, locale string, defaultLocale string) string {
	for _, item := range items {
		if item.Slug == slug {
			return localizeText(item.Title, locale, defaultLocale, "")
		}
	}
	return ""
}

func localizeDebug(example documentationsetting.DebugExample, locale string, defaultLocale string) *DebugExample {
	if example.Method == "" {
		return nil
	}

	debug := DebugExample{
		Method:       example.Method,
		Path:         example.Path,
		PathTemplate: example.PathTemplate,
		Auth:         example.Auth,
		Model:        example.Model,
		Headers:      example.Headers,
		Body:         nil,
	}

	for _, candidate := range localeFallbacks(locale, defaultLocale) {
		if value, ok := example.Body[candidate]; ok {
			debug.Body = value
			break
		}
	}

	return &debug
}

func localizeText(text documentationsetting.LocalizedText, locale string, defaultLocale string, fallback string) string {
	for _, candidate := range localeFallbacks(locale, defaultLocale) {
		if value := strings.TrimSpace(text[candidate]); value != "" {
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

func readMarkdown(contentDir string, slug string, locale string, defaultLocale string) (loadedPage, error) {
	baseDir := filepath.Clean(contentDir)
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return loadedPage{}, err
	}

	filename := filepath.FromSlash(slug)
	if strings.ToLower(filepath.Ext(filename)) != ".md" {
		filename += ".md"
	}

	for _, candidateLocale := range localeFallbacks(locale, defaultLocale) {
		candidates := []string{
			filepath.Join(baseDir, candidateLocale, filename),
			filepath.Join(baseDir, "docs", candidateLocale, filename),
		}
		for _, candidate := range candidates {
			candidateAbs, err := filepath.Abs(candidate)
			if err != nil {
				continue
			}
			if !isWithinBase(baseAbs, candidateAbs) {
				continue
			}
			content, err := os.ReadFile(candidateAbs)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return loadedPage{}, err
			}
			return loadedPage{
				content: string(content),
				locale:  candidateLocale,
			}, nil
		}
	}

	return loadedPage{}, fmt.Errorf("%w: %s", ErrNotFound, slug)
}

func isWithinBase(baseAbs string, candidateAbs string) bool {
	rel, err := filepath.Rel(baseAbs, candidateAbs)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func extractFirstHeading(content string) string {
	match := firstHeadingPattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}
