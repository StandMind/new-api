package webbranding

import (
	"html"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var (
	titleTagPattern      = regexp.MustCompile(`(?is)<title>.*?</title>`)
	metaTitlePattern     = regexp.MustCompile(`(?is)<meta\s+[^>]*\bname=["']title["'][^>]*>`)
	linkRelPattern       = regexp.MustCompile(`(?is)<link\s+[^>]*\brel\s*=\s*("[^"]*"|'[^']*')[^>]*>`)
	relAttributePattern  = regexp.MustCompile(`(?is)\brel\s*=\s*("[^"]*"|'[^']*')`)
	hrefAttributePattern = regexp.MustCompile(`(?is)\bhref\s*=\s*("[^"]*"|'[^']*')`)
)

type brandConfig struct {
	systemName string
	logo       string
}

func ApplyIndexPageBranding(page []byte) []byte {
	if len(page) == 0 {
		return page
	}

	config := currentBrandConfig()
	if config.systemName == "" && config.logo == "" {
		return page
	}

	body := string(page)
	if config.systemName != "" {
		body = replaceTitle(body, config.systemName)
		body = replaceMetaTitle(body, config.systemName)
	}
	if config.logo != "" {
		body = replaceIconLinks(body, config.logo)
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

func replaceMetaTitle(body string, systemName string) string {
	escaped := html.EscapeString(systemName)
	return metaTitlePattern.ReplaceAllStringFunc(body, func(tag string) string {
		return replaceAttribute(tag, "content", escaped)
	})
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
