package webbranding

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func withBrand(t *testing.T, systemName string, logo string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	previousName := common.SystemName
	previousLogo := common.Logo
	common.SystemName = systemName
	common.Logo = logo
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		common.SystemName = previousName
		common.Logo = previousLogo
	})
}

func TestApplyIndexPageBrandingReplacesTitleAndMetaTitle(t *testing.T) {
	withBrand(t, "Example Gateway", "")

	page := []byte(`<!doctype html><html><head><title>New API</title><meta name="title" content="New API" /></head></html>`)

	body := string(ApplyIndexPageBranding(page))

	require.Contains(t, body, "<title>Example Gateway</title>")
	require.Contains(t, body, `<meta name="title" content="Example Gateway" />`)
}

func TestApplyIndexPageBrandingEscapesTitleAndIconURL(t *testing.T) {
	withBrand(t, `A&B "Gateway"`, `/assets/logo.png?v=1&brand=a`)

	page := []byte(`<!doctype html><html><head><title>New API</title><meta name="title" content="New API" /><link rel="icon" href="/favicon.ico"><link rel="apple-touch-icon" href="/apple.png"></head></html>`)

	body := string(ApplyIndexPageBranding(page))

	require.Contains(t, body, "<title>A&amp;B &#34;Gateway&#34;</title>")
	require.Contains(t, body, `content="A&amp;B &#34;Gateway&#34;"`)
	require.Contains(t, body, `href="/assets/logo.png?v=1&amp;brand=a"`)
	require.NotContains(t, body, `href="/favicon.ico"`)
	require.NotContains(t, body, `href="/apple.png"`)
}

func TestApplyIndexPageBrandingLeavesNonIconLinksUnchanged(t *testing.T) {
	withBrand(t, "", "/assets/logo.png")

	page := []byte(`<!doctype html><html><head><link rel="stylesheet" href="/style.css"><link rel="shortcut icon" href="/favicon.ico"></head></html>`)

	body := string(ApplyIndexPageBranding(page))

	require.Contains(t, body, `rel="stylesheet" href="/style.css"`)
	require.Contains(t, body, `rel="shortcut icon" href="/assets/logo.png"`)
}

func TestApplyIndexPageBrandingWithMetaInjectsSEOAndRootHTML(t *testing.T) {
	withBrand(t, "", "")

	page := []byte(`<!doctype html><html lang="en"><head><title>New API</title><meta name="title" content="New API"><meta name="description" content="Default"></head><body><div id="root"></div></body></html>`)

	body := string(ApplyIndexPageBrandingWithMeta(page, PageMeta{
		Title:       `Article & Guide`,
		Description: `Useful <summary>`,
		Language:    "zh",
		HeadHTML:    `<link rel="canonical" href="https://example.com/blog/a">`,
		RootHTML:    `<main><article>Full article</article></main>`,
	}))

	require.Contains(t, body, `<html lang="zh">`)
	require.Contains(t, body, `<title>Article &amp; Guide</title>`)
	require.Contains(t, body, `content="Useful &lt;summary&gt;"`)
	require.Contains(t, body, `<link rel="canonical" href="https://example.com/blog/a">`)
	require.Contains(t, body, `<div id="root"><main><article>Full article</article></main></div>`)
}
