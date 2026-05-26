package webbranding

import (
	"html"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var (
	titleTagPattern        = regexp.MustCompile(`(?is)<title>.*?</title>`)
	htmlLangPattern        = regexp.MustCompile(`(?is)<html\b([^>]*)\blang\s*=\s*("[^"]*"|'[^']*')([^>]*)>`)
	metaTitlePattern       = regexp.MustCompile(`(?is)<meta\s+[^>]*\bname=["']title["'][^>]*>`)
	metaDescriptionPattern = regexp.MustCompile(`(?is)<meta\s+[^>]*\bname=["']description["'][^>]*>`)
	headEndPattern         = regexp.MustCompile(`(?is)</head>`)
	rootPattern            = regexp.MustCompile(`(?is)<div\s+id\s*=\s*["']root["']\s*></div>`)
	linkRelPattern         = regexp.MustCompile(`(?is)<link\s+[^>]*\brel\s*=\s*("[^"]*"|'[^']*')[^>]*>`)
	relAttributePattern    = regexp.MustCompile(`(?is)\brel\s*=\s*("[^"]*"|'[^']*')`)
	hrefAttributePattern   = regexp.MustCompile(`(?is)\bhref\s*=\s*("[^"]*"|'[^']*')`)
)

type PageMeta struct {
	Title       string
	Description string
	Language    string
	HeadHTML    string
	RootHTML    string
}

type brandConfig struct {
	systemName string
	logo       string
}

func ApplyIndexPageBranding(page []byte) []byte {
	return ApplyIndexPageBrandingWithMeta(page, PageMeta{})
}

func ApplyIndexPageBrandingWithMeta(page []byte, meta PageMeta) []byte {
	if len(page) == 0 {
		return page
	}

	config := currentBrandConfig()
	if config.systemName == "" && config.logo == "" && strings.TrimSpace(meta.Title) == "" && strings.TrimSpace(meta.Description) == "" && strings.TrimSpace(meta.Language) == "" && strings.TrimSpace(meta.HeadHTML) == "" && strings.TrimSpace(meta.RootHTML) == "" {
		return page
	}

	body := string(page)
	if language := strings.TrimSpace(meta.Language); language != "" {
		body = replaceHTMLLanguage(body, language)
	}
	title := strings.TrimSpace(meta.Title)
	if title == "" {
		title = config.systemName
	}
	if title != "" {
		body = replaceTitle(body, title)
		body = replaceMetaTitle(body, title)
	}
	if description := strings.TrimSpace(meta.Description); description != "" {
		body = replaceMetaDescription(body, description)
	}
	if config.logo != "" {
		body = replaceIconLinks(body, config.logo)
	}
	if headHTML := strings.TrimSpace(meta.HeadHTML); headHTML != "" {
		body = injectHeadHTML(body, headHTML)
	}
	if rootHTML := strings.TrimSpace(meta.RootHTML); rootHTML != "" {
		body = replaceRootHTML(body, rootHTML)
	}

	return []byte(body)
}

func currentBrandConfig() brandConfig {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	return brandConfig{
		systemName: strings.TrimSpace(common.SystemName),
		logo:       strings.TrimSpace(common.Logo),
	}
}

func replaceTitle(body string, systemName string) string {
	title := "<title>" + html.EscapeString(systemName) + "</title>"
	return titleTagPattern.ReplaceAllString(body, title)
}

func replaceHTMLLanguage(body string, language string) string {
	escaped := html.EscapeString(language)
	if htmlLangPattern.MatchString(body) {
		return htmlLangPattern.ReplaceAllStringFunc(body, func(tag string) string {
			return replaceAttribute(tag, "lang", escaped)
		})
	}
	return strings.Replace(body, "<html", `<html lang="`+escaped+`"`, 1)
}

func replaceMetaTitle(body string, systemName string) string {
	escaped := html.EscapeString(systemName)
	return metaTitlePattern.ReplaceAllStringFunc(body, func(tag string) string {
		return replaceAttribute(tag, "content", escaped)
	})
}

func replaceMetaDescription(body string, description string) string {
	escaped := html.EscapeString(description)
	return metaDescriptionPattern.ReplaceAllStringFunc(body, func(tag string) string {
		return replaceAttribute(tag, "content", escaped)
	})
}

func injectHeadHTML(body string, headHTML string) string {
	return headEndPattern.ReplaceAllString(body, headHTML+"\n  </head>")
}

func replaceRootHTML(body string, rootHTML string) string {
	if rootPattern.MatchString(body) {
		return rootPattern.ReplaceAllString(body, `<div id="root">`+rootHTML+`</div>`)
	}
	return body
}

func replaceIconLinks(body string, logo string) string {
	escaped := html.EscapeString(logo)
	return linkRelPattern.ReplaceAllStringFunc(body, func(tag string) string {
		rel := attributeValue(relAttributePattern.FindString(tag))
		if !isIconRel(rel) {
			return tag
		}
		return replaceAttribute(tag, "href", escaped)
	})
}

func isIconRel(rel string) bool {
	for _, token := range strings.Fields(strings.ToLower(rel)) {
		if token == "icon" || token == "apple-touch-icon" {
			return true
		}
	}
	return false
}

func replaceAttribute(tag string, name string, value string) string {
	pattern := hrefAttributePattern
	if name != "href" {
		pattern = regexp.MustCompile(`(?is)\b` + regexp.QuoteMeta(name) + `\s*=\s*("[^"]*"|'[^']*')`)
	}
	if !pattern.MatchString(tag) {
		return strings.TrimRight(tag, ">") + ` ` + name + `="` + value + `">`
	}
	return pattern.ReplaceAllStringFunc(tag, func(attr string) string {
		quote := attributeQuote(attr)
		return name + "=" + quote + value + quote
	})
}

func attributeValue(attr string) string {
	idx := strings.Index(attr, "=")
	if idx < 0 {
		return ""
	}
	value := strings.TrimSpace(attr[idx+1:])
	return strings.Trim(value, `"'`)
}

func attributeQuote(attr string) string {
	idx := strings.Index(attr, "=")
	if idx < 0 {
		return `"`
	}
	value := strings.TrimSpace(attr[idx+1:])
	if strings.HasPrefix(value, `'`) {
		return `'`
	}
	return `"`
}
