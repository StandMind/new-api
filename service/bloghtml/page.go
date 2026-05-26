package bloghtml

import (
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/webbranding"
)

const (
	defaultBlogDescription = "AI updates, model guides, and practical API notes."
)

var (
	markdownLinkPattern   = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	markdownCodePattern   = regexp.MustCompile("`([^`]+)`")
	markdownStrongPattern = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	markdownEmPattern     = regexp.MustCompile(`\*([^*]+)\*`)
	tableSeparatorPattern = regexp.MustCompile(`^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$`)
)

type RenderOptions struct {
	BaseURL  string
	Locale   string
	SiteName string
}

func BuildBlogIndexMeta(options RenderOptions, posts []model.BlogPostPublicDTO) webbranding.PageMeta {
	locale := normalizeLocale(options.Locale)
	title := withSiteName("Blog", options.SiteName)
	canonical := canonicalURL(options.BaseURL, "/blog", locale)

	return webbranding.PageMeta{
		Title:       title,
		Description: defaultBlogDescription,
		Language:    locale,
		HeadHTML:    buildBlogIndexHead(options, canonical, posts),
		RootHTML:    renderBlogIndex(options, posts),
	}
}

func BuildBlogPostMeta(options RenderOptions, post model.BlogPostPublicDTO) webbranding.PageMeta {
	locale := normalizeLocale(options.Locale)
	title := withSiteName(post.Title, options.SiteName)
	canonical := canonicalURL(options.BaseURL, "/blog/"+post.Slug, locale)

	return webbranding.PageMeta{
		Title:       title,
		Description: post.Summary,
		Language:    locale,
		HeadHTML:    buildBlogPostHead(options, canonical, post),
		RootHTML:    renderBlogPost(options, post),
	}
}

func BuildBlogNotFoundMeta(options RenderOptions) webbranding.PageMeta {
	locale := normalizeLocale(options.Locale)
	return webbranding.PageMeta{
		Title:       withSiteName("Article not found", options.SiteName),
		Description: "The requested article was not found.",
		Language:    locale,
		HeadHTML:    `<meta name="robots" content="noindex">`,
		RootHTML:    `<main class="min-h-svh px-6 py-24"><article class="mx-auto max-w-2xl"><h1>Article not found</h1><p>The article may be unpublished or the link may be invalid.</p><p><a href="/blog">Back to blog</a></p></article></main>`,
	}
}

func buildBlogIndexHead(options RenderOptions, canonical string, posts []model.BlogPostPublicDTO) string {
	title := withSiteName("Blog", options.SiteName)
	head := []string{
		canonicalLink(canonical),
		alternateLinks(options.BaseURL, "/blog"),
		metaProperty("og:type", "website"),
		metaProperty("og:title", title),
		metaProperty("og:description", defaultBlogDescription),
		metaProperty("og:url", canonical),
		metaName("twitter:card", "summary"),
		metaName("twitter:title", title),
		metaName("twitter:description", defaultBlogDescription),
		jsonLDScript(map[string]any{
			"@context":    "https://schema.org",
			"@type":       "Blog",
			"name":        title,
			"description": defaultBlogDescription,
			"url":         canonical,
			"blogPost":    blogPostingSummaries(options, posts),
		}),
	}
	return strings.Join(head, "\n")
}

func buildBlogPostHead(options RenderOptions, canonical string, post model.BlogPostPublicDTO) string {
	title := withSiteName(post.Title, options.SiteName)
	head := []string{
		canonicalLink(canonical),
		alternateLinks(options.BaseURL, "/blog/"+post.Slug),
		metaProperty("og:type", "article"),
		metaProperty("og:title", title),
		metaProperty("og:description", post.Summary),
		metaProperty("og:url", canonical),
		metaName("twitter:card", twitterCard(post)),
		metaName("twitter:title", title),
		metaName("twitter:description", post.Summary),
	}
	if post.CoverImage != "" {
		head = append(head, metaProperty("og:image", post.CoverImage), metaName("twitter:image", post.CoverImage))
	}
	if post.PublishedTime > 0 {
		head = append(head, metaProperty("article:published_time", isoTime(post.PublishedTime)))
	}
	if post.UpdatedTime > 0 {
		head = append(head, metaProperty("article:modified_time", isoTime(post.UpdatedTime)))
	}
	for _, tag := range post.Tags {
		head = append(head, metaProperty("article:tag", tag))
	}
	head = append(head, jsonLDScript(blogPostingSchema(options, canonical, post)))
	return strings.Join(head, "\n")
}

func renderBlogIndex(options RenderOptions, posts []model.BlogPostPublicDTO) string {
	var builder strings.Builder
	builder.WriteString(`<main class="min-h-svh bg-background text-foreground px-4 py-16 sm:px-6 lg:px-8">`)
	builder.WriteString(`<section class="mx-auto max-w-7xl space-y-8">`)
	builder.WriteString(`<header class="border-b pb-6">`)
	builder.WriteString(`<h1 class="text-4xl font-semibold tracking-tight">Blog</h1>`)
	builder.WriteString(`<p class="mt-3 max-w-2xl text-muted-foreground">`)
	builder.WriteString(html.EscapeString(defaultBlogDescription))
	builder.WriteString(`</p></header>`)

	if len(posts) == 0 {
		builder.WriteString(`<section class="rounded-lg border border-dashed p-8 text-center"><h2>No blog posts yet</h2><p>Published articles will appear here.</p></section>`)
	} else {
		builder.WriteString(`<section class="grid gap-5 md:grid-cols-2 xl:grid-cols-3">`)
		for _, post := range posts {
			builder.WriteString(`<article class="rounded-lg border p-5">`)
			if post.CoverImage != "" {
				builder.WriteString(`<a href="/blog/` + attr(post.Slug) + `"><img loading="lazy" alt="" src="` + attr(post.CoverImage) + `" class="mb-4 aspect-video w-full rounded-md object-cover"></a>`)
			}
			builder.WriteString(`<p class="text-sm text-muted-foreground">`)
			builder.WriteString(html.EscapeString(formatDate(post.PublishedTime, options.Locale)))
			if post.ReadingMinutes > 0 {
				builder.WriteString(` · ` + strconv.Itoa(post.ReadingMinutes) + ` min read`)
			}
			builder.WriteString(`</p>`)
			builder.WriteString(`<h2 class="mt-2 text-xl font-semibold"><a href="/blog/` + attr(post.Slug) + `">` + html.EscapeString(post.Title) + `</a></h2>`)
			if post.Summary != "" {
				builder.WriteString(`<p class="mt-3 text-muted-foreground">` + html.EscapeString(post.Summary) + `</p>`)
			}
			builder.WriteString(renderTags(post.Tags))
			builder.WriteString(`</article>`)
		}
		builder.WriteString(`</section>`)
	}

	builder.WriteString(`</section></main>`)
	return builder.String()
}

func renderBlogPost(options RenderOptions, post model.BlogPostPublicDTO) string {
	var builder strings.Builder
	builder.WriteString(`<main class="min-h-svh bg-background text-foreground px-4 py-16 sm:px-6 lg:px-8">`)
	builder.WriteString(`<article class="mx-auto max-w-4xl">`)
	builder.WriteString(`<p><a href="/blog">Back to blog</a></p>`)
	builder.WriteString(`<header class="border-b pb-6">`)
	builder.WriteString(`<p class="text-sm text-muted-foreground">`)
	builder.WriteString(html.EscapeString(formatDate(post.PublishedTime, options.Locale)))
	if post.Author != "" {
		builder.WriteString(` · ` + html.EscapeString(post.Author))
	}
	if post.ReadingMinutes > 0 {
		builder.WriteString(` · ` + strconv.Itoa(post.ReadingMinutes) + ` min read`)
	}
	builder.WriteString(`</p>`)
	builder.WriteString(`<h1 class="mt-3 text-4xl font-semibold tracking-tight">` + html.EscapeString(post.Title) + `</h1>`)
	if post.Summary != "" {
		builder.WriteString(`<p class="mt-4 text-lg text-muted-foreground">` + html.EscapeString(post.Summary) + `</p>`)
	}
	builder.WriteString(renderTags(post.Tags))
	builder.WriteString(`</header>`)
	if post.CoverImage != "" {
		builder.WriteString(`<img alt="" src="` + attr(post.CoverImage) + `" class="my-8 w-full rounded-lg border object-cover">`)
	}
	builder.WriteString(`<section class="prose prose-neutral dark:prose-invert mt-8">`)
	builder.WriteString(markdownToHTML(post.Content))
	builder.WriteString(`</section>`)
	builder.WriteString(`</article></main>`)
	return builder.String()
}

func markdownToHTML(markdown string) string {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	var builder strings.Builder
	var paragraph []string
	inCode := false
	inUL := false
	inOL := false
	codeLang := ""

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		builder.WriteString(`<p>`)
		builder.WriteString(renderInline(strings.Join(paragraph, " ")))
		builder.WriteString(`</p>`)
		paragraph = nil
	}
	closeLists := func() {
		if inUL {
			builder.WriteString(`</ul>`)
			inUL = false
		}
		if inOL {
			builder.WriteString(`</ol>`)
			inOL = false
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if inCode {
				builder.WriteString(`</code></pre>`)
				inCode = false
				codeLang = ""
				continue
			}
			flushParagraph()
			closeLists()
			codeLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			builder.WriteString(`<pre><code`)
			if codeLang != "" {
				builder.WriteString(` class="language-` + attr(codeLang) + `"`)
			}
			builder.WriteString(`>`)
			inCode = true
			continue
		}
		if inCode {
			builder.WriteString(html.EscapeString(line))
			builder.WriteByte('\n')
			continue
		}
		if trimmed == "" {
			flushParagraph()
			closeLists()
			continue
		}
		if isTableStart(lines, i) {
			flushParagraph()
			closeLists()
			tableHTML, nextIndex := renderTable(lines, i)
			builder.WriteString(tableHTML)
			i = nextIndex
			continue
		}
		if depth, title := heading(trimmed); depth > 0 {
			flushParagraph()
			closeLists()
			builder.WriteString(`<h` + strconv.Itoa(depth) + `>`)
			builder.WriteString(renderInline(title))
			builder.WriteString(`</h` + strconv.Itoa(depth) + `>`)
			continue
		}
		if strings.HasPrefix(trimmed, ">") {
			flushParagraph()
			closeLists()
			builder.WriteString(`<blockquote><p>`)
			builder.WriteString(renderInline(strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))))
			builder.WriteString(`</p></blockquote>`)
			continue
		}
		if item, ok := unorderedItem(trimmed); ok {
			flushParagraph()
			if inOL {
				builder.WriteString(`</ol>`)
				inOL = false
			}
			if !inUL {
				builder.WriteString(`<ul>`)
				inUL = true
			}
			builder.WriteString(`<li>` + renderInline(item) + `</li>`)
			continue
		}
		if item, ok := orderedItem(trimmed); ok {
			flushParagraph()
			if inUL {
				builder.WriteString(`</ul>`)
				inUL = false
			}
			if !inOL {
				builder.WriteString(`<ol>`)
				inOL = true
			}
			builder.WriteString(`<li>` + renderInline(item) + `</li>`)
			continue
		}
		paragraph = append(paragraph, trimmed)
	}
	flushParagraph()
	closeLists()
	if inCode {
		builder.WriteString(`</code></pre>`)
	}
	return builder.String()
}

func renderInline(value string) string {
	matches := markdownLinkPattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return renderInlineNoLinks(value)
	}
	var builder strings.Builder
	last := 0
	for _, match := range matches {
		builder.WriteString(renderInlineNoLinks(value[last:match[0]]))
		text := renderInlineNoLinks(value[match[2]:match[3]])
		href := sanitizeHref(value[match[4]:match[5]])
		builder.WriteString(`<a href="` + attr(href) + `">` + text + `</a>`)
		last = match[1]
	}
	builder.WriteString(renderInlineNoLinks(value[last:]))
	return builder.String()
}

func renderInlineNoLinks(value string) string {
	escaped := html.EscapeString(value)
	escaped = markdownCodePattern.ReplaceAllString(escaped, `<code>$1</code>`)
	escaped = markdownStrongPattern.ReplaceAllString(escaped, `<strong>$1</strong>`)
	escaped = markdownEmPattern.ReplaceAllString(escaped, `<em>$1</em>`)
	return escaped
}

func heading(trimmed string) (int, string) {
	if !strings.HasPrefix(trimmed, "#") {
		return 0, ""
	}
	depth := 0
	for depth < len(trimmed) && trimmed[depth] == '#' {
		depth++
	}
	if depth == 0 || depth > 6 || depth >= len(trimmed) || trimmed[depth] != ' ' {
		return 0, ""
	}
	return depth, strings.TrimSpace(trimmed[depth:])
}

func unorderedItem(trimmed string) (string, bool) {
	for _, prefix := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), true
		}
	}
	return "", false
}

func orderedItem(trimmed string) (string, bool) {
	dot := strings.Index(trimmed, ". ")
	if dot <= 0 || dot > 3 {
		return "", false
	}
	if _, err := strconv.Atoi(trimmed[:dot]); err != nil {
		return "", false
	}
	return strings.TrimSpace(trimmed[dot+2:]), true
}

func isTableStart(lines []string, index int) bool {
	if index+1 >= len(lines) || !strings.Contains(lines[index], "|") {
		return false
	}
	return tableSeparatorPattern.MatchString(lines[index+1])
}

func renderTable(lines []string, index int) (string, int) {
	header := splitTableRow(lines[index])
	var builder strings.Builder
	builder.WriteString(`<table><thead><tr>`)
	for _, cell := range header {
		builder.WriteString(`<th>` + renderInline(cell) + `</th>`)
	}
	builder.WriteString(`</tr></thead><tbody>`)
	i := index + 2
	for ; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" || !strings.Contains(lines[i], "|") {
			break
		}
		builder.WriteString(`<tr>`)
		for _, cell := range splitTableRow(lines[i]) {
			builder.WriteString(`<td>` + renderInline(cell) + `</td>`)
		}
		builder.WriteString(`</tr>`)
	}
	builder.WriteString(`</tbody></table>`)
	return builder.String(), i - 1
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func sanitizeHref(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "#") || strings.HasPrefix(value, "/") {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "#"
	}
	switch parsed.Scheme {
	case "http", "https", "mailto":
		return value
	default:
		return "#"
	}
}

func canonicalURL(baseURL string, pagePath string, locale string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://example.com"
	}
	u := baseURL + pagePath
	q := url.Values{}
	q.Set("lang", normalizeLocale(locale))
	return u + "?" + q.Encode()
}

func alternateLinks(baseURL string, pagePath string) string {
	locales := model.SupportedLocalizedTextLocales()
	lines := make([]string, 0, len(locales)+1)
	for _, locale := range locales {
		lines = append(lines, `<link rel="alternate" hreflang="`+attr(locale)+`" href="`+attr(canonicalURL(baseURL, pagePath, locale))+`">`)
	}
	lines = append(lines, `<link rel="alternate" hreflang="x-default" href="`+attr(canonicalURL(baseURL, pagePath, "en"))+`">`)
	return strings.Join(lines, "\n")
}

func canonicalLink(href string) string {
	return `<link rel="canonical" href="` + attr(href) + `">`
}

func metaName(name string, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	return `<meta name="` + attr(name) + `" content="` + attr(content) + `">`
}

func metaProperty(property string, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	return `<meta property="` + attr(property) + `" content="` + attr(content) + `">`
}

func jsonLDScript(data map[string]any) string {
	body, err := common.Marshal(data)
	if err != nil {
		return ""
	}
	value := strings.ReplaceAll(string(body), "</script", "<\\/script")
	return `<script type="application/ld+json">` + value + `</script>`
}

func blogPostingSchema(options RenderOptions, canonical string, post model.BlogPostPublicDTO) map[string]any {
	data := map[string]any{
		"@context":         "https://schema.org",
		"@type":            "BlogPosting",
		"headline":         post.Title,
		"description":      post.Summary,
		"url":              canonical,
		"mainEntityOfPage": canonical,
		"inLanguage":       normalizeLocale(options.Locale),
		"keywords":         strings.Join(post.Tags, ", "),
	}
	if post.Author != "" {
		data["author"] = map[string]any{"@type": "Person", "name": post.Author}
	}
	if options.SiteName != "" {
		data["publisher"] = map[string]any{"@type": "Organization", "name": options.SiteName}
	}
	if post.PublishedTime > 0 {
		data["datePublished"] = isoTime(post.PublishedTime)
	}
	if post.UpdatedTime > 0 {
		data["dateModified"] = isoTime(post.UpdatedTime)
	}
	if post.CoverImage != "" {
		data["image"] = []string{post.CoverImage}
	}
	return data
}

func blogPostingSummaries(options RenderOptions, posts []model.BlogPostPublicDTO) []map[string]any {
	items := make([]map[string]any, 0, len(posts))
	for _, post := range posts {
		items = append(items, map[string]any{
			"@type":    "BlogPosting",
			"headline": post.Title,
			"url":      canonicalURL(options.BaseURL, "/blog/"+post.Slug, options.Locale),
		})
	}
	return items
}

func renderTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(`<ul class="mt-4 flex flex-wrap gap-2" aria-label="Tags">`)
	for _, tag := range tags {
		builder.WriteString(`<li class="rounded-md border px-2 py-1 text-sm">` + html.EscapeString(tag) + `</li>`)
	}
	builder.WriteString(`</ul>`)
	return builder.String()
}

func twitterCard(post model.BlogPostPublicDTO) string {
	if post.CoverImage != "" {
		return "summary_large_image"
	}
	return "summary"
}

func withSiteName(title string, siteName string) string {
	title = strings.TrimSpace(title)
	siteName = strings.TrimSpace(siteName)
	if title == "" {
		title = "Blog"
	}
	if siteName == "" {
		return title
	}
	return title + " | " + siteName
}

func normalizeLocale(locale string) string {
	normalized := model.NormalizeLocalizedTextLocale(locale)
	if normalized == "" {
		return "en"
	}
	return normalized
}

func formatDate(timestamp int64, locale string) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).UTC().Format("2006-01-02")
}

func isoTime(timestamp int64) string {
	return time.Unix(timestamp, 0).UTC().Format(time.RFC3339)
}

func attr(value string) string {
	return html.EscapeString(value)
}
