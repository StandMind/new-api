package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlogPostCreateListAndLocalize(t *testing.T) {
	truncateTables(t)

	created, err := CreateBlogPost(BlogPostAdminDTO{
		Slug:   "model-pricing-guide",
		Status: BlogStatusPublished,
		Tags:   []string{"AI", "api", "AI"},
		Translations: map[string]BlogPostTranslationPayload{
			"en": {
				Title:   "Model pricing guide",
				Summary: "English summary",
				Content: "## Intro\nEnglish content",
			},
			"zh": {
				Title:   "模型价格指南",
				Summary: "中文摘要",
				Content: "## 介绍\n中文正文",
			},
		},
	})
	require.NoError(t, err)
	require.NotZero(t, created.Id)
	require.Equal(t, []string{"ai", "api"}, created.Tags)

	zhItems, total, err := ListPublishedBlogPosts("zh-CN", "", "", 0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, zhItems, 1)
	require.Equal(t, "模型价格指南", zhItems[0].Title)
	require.Empty(t, zhItems[0].Content)
	require.Equal(t, 1, zhItems[0].ReadingMinutes)

	searchedItems, total, err := ListPublishedBlogPosts("zh-CN", "价格", "", 0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, searchedItems, 1)
	require.Equal(t, "model-pricing-guide", searchedItems[0].Slug)

	taggedItems, total, err := ListPublishedBlogPosts("en", "", "api", 0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, taggedItems, 1)
	require.Equal(t, "model-pricing-guide", taggedItems[0].Slug)

	enPost, err := GetPublishedBlogPostBySlug("model-pricing-guide", "es")
	require.NoError(t, err)
	require.Equal(t, "Model pricing guide", enPost.Title)
	require.Contains(t, enPost.Content, "English content")
}
