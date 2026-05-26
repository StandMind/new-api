package bloghtml

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestBuildBlogPostMetaIncludesFullArticleHTMLAndSEO(t *testing.T) {
	meta := BuildBlogPostMeta(RenderOptions{
		BaseURL:  "https://example.com",
		Locale:   "zh",
		SiteName: "Gateway",
	}, model.BlogPostPublicDTO{
		Slug:          "model-guide",
		Title:         "Model Guide",
		Summary:       "Short summary",
		Content:       "## Intro\nFull **article** body with [link](https://example.com/docs).",
		Tags:          []string{"ai", "api"},
		CoverImage:    "https://example.com/cover.png",
		Author:        "Admin",
		PublishedTime: 1779550641,
		UpdatedTime:   1779550641,
		Locale:        "zh",
	})

	require.Equal(t, "Model Guide | Gateway", meta.Title)
	require.Equal(t, "Short summary", meta.Description)
	require.Equal(t, "zh", meta.Language)
	require.Contains(t, meta.HeadHTML, `<link rel="canonical" href="https://example.com/blog/model-guide?lang=zh">`)
	require.Contains(t, meta.HeadHTML, `application/ld+json`)
	require.Contains(t, meta.HeadHTML, `"@type":"BlogPosting"`)
	require.Contains(t, meta.RootHTML, `<h1`)
	require.Contains(t, meta.RootHTML, `Model Guide`)
	require.Contains(t, meta.RootHTML, `<h2>Intro</h2>`)
	require.Contains(t, meta.RootHTML, `<strong>article</strong>`)
	require.Contains(t, meta.RootHTML, `<a href="https://example.com/docs">link</a>`)
}

func TestBuildBlogPostMetaEscapesMarkdownHTML(t *testing.T) {
	meta := BuildBlogPostMeta(RenderOptions{
		BaseURL: "https://example.com",
		Locale:  "en",
	}, model.BlogPostPublicDTO{
		Slug:    "safe",
		Title:   "Safe",
		Summary: "Summary",
		Content: `<script>alert(1)</script>

[bad](javascript:alert(1))`,
	})

	require.Contains(t, meta.RootHTML, `&lt;script&gt;alert(1)&lt;/script&gt;`)
	require.NotContains(t, strings.ToLower(meta.RootHTML), `<script>alert`)
	require.Contains(t, meta.RootHTML, `<a href="#">bad</a>`)
}
