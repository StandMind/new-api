package model

import (
	"testing"
	"time"

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

func TestBlogPostStatsSummaries(t *testing.T) {
	truncateTables(t)

	created, err := CreateBlogPost(BlogPostAdminDTO{
		Slug:   "stats-guide",
		Status: BlogStatusPublished,
		Translations: map[string]BlogPostTranslationPayload{
			"en": {
				Title:   "Stats guide",
				Content: "Content",
			},
		},
	})
	require.NoError(t, err)

	today := time.Now().UTC()
	require.NoError(t, incrementBlogPostView(created.Id, today))
	require.NoError(t, incrementBlogPostView(created.Id, today))
	require.NoError(t, incrementBlogPostView(created.Id, today.AddDate(0, 0, -2)))
	require.NoError(t, DB.Create(&BlogPostDailyView{
		PostID: created.Id,
		Date:   today.AddDate(0, 0, -40).Format(blogStatsDateLayout),
		Views:  99,
	}).Error)

	stats, err := GetBlogPostStats(created.Id, 30)
	require.NoError(t, err)
	require.Equal(t, 30, stats.Days)
	require.Equal(t, today.AddDate(0, 0, -29).Format(blogStatsDateLayout), stats.StartDate)
	require.Equal(t, today.Format(blogStatsDateLayout), stats.EndDate)
	require.EqualValues(t, 2, stats.ViewsToday)
	require.EqualValues(t, 3, stats.TotalViews)
	require.EqualValues(t, 102, stats.LifetimeViews)
	require.Len(t, stats.Daily, 30)
	require.Equal(t, today.AddDate(0, 0, -29).Format(blogStatsDateLayout), stats.Daily[0].Date)
	require.Equal(t, today.Format(blogStatsDateLayout), stats.Daily[29].Date)

	adminItems, _, err := ListBlogPostsAdmin("", "", 0, 10)
	require.NoError(t, err)
	require.Len(t, adminItems, 1)
	require.EqualValues(t, 3, adminItems[0].Stats.TotalViews)
	require.EqualValues(t, 102, adminItems[0].Stats.LifetimeViews)
	require.Empty(t, adminItems[0].Stats.Daily)

	rangeStats, err := GetBlogPostStatsByDateRange(
		created.Id,
		today.AddDate(0, 0, -1).Format(blogStatsDateLayout),
		today.Format(blogStatsDateLayout),
	)
	require.NoError(t, err)
	require.Equal(t, 2, rangeStats.Days)
	require.EqualValues(t, 2, rangeStats.TotalViews)
	require.EqualValues(t, 102, rangeStats.LifetimeViews)
	require.Len(t, rangeStats.Daily, 2)
}
