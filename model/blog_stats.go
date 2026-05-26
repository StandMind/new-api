package model

import (
	"errors"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultBlogStatsDays = 30
	maxBlogStatsDays     = 365
	blogStatsDateLayout  = "2006-01-02"
)

type BlogPostDailyView struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	PostID      int    `json:"post_id" gorm:"not null;index;uniqueIndex:uk_blog_post_daily_view,priority:1"`
	Date        string `json:"date" gorm:"size:10;not null;index;uniqueIndex:uk_blog_post_daily_view,priority:2"`
	Views       int64  `json:"views" gorm:"not null;default:0"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

func (BlogPostDailyView) TableName() string {
	return "blog_post_daily_views"
}

type BlogPostDailyStatsDTO struct {
	Date  string `json:"date"`
	Views int64  `json:"views"`
}

type BlogPostStatsSummaryDTO struct {
	Days          int                     `json:"days"`
	StartDate     string                  `json:"start_date"`
	EndDate       string                  `json:"end_date"`
	ViewsToday    int64                   `json:"views_today"`
	TotalViews    int64                   `json:"total_views"`
	LifetimeViews int64                   `json:"lifetime_views"`
	Daily         []BlogPostDailyStatsDTO `json:"daily,omitempty"`
}

type blogPostViewTotalRow struct {
	PostID int   `json:"post_id"`
	Views  int64 `json:"views"`
}

func RecordPublishedBlogPostView(slug string) error {
	slug = NormalizeBlogSlug(slug)
	if slug == "" {
		return ErrBlogPostNotFound
	}

	var post BlogPost
	err := blogPostPublicQuery().
		Select("id").
		Where("slug = ?", slug).
		First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrBlogPostNotFound
	}
	if err != nil {
		return err
	}
	return incrementBlogPostView(post.Id, time.Now().UTC())
}

func GetBlogPostStats(postID int, days int) (BlogPostStatsSummaryDTO, error) {
	if postID <= 0 {
		return BlogPostStatsSummaryDTO{}, ErrBlogPostNotFound
	}
	summaries, err := GetBlogPostStatsSummaries([]int{postID}, days, true)
	if err != nil {
		return BlogPostStatsSummaryDTO{}, err
	}
	return summaries[postID], nil
}

func GetBlogPostStatsByDateRange(postID int, startDate string, endDate string) (BlogPostStatsSummaryDTO, error) {
	if postID <= 0 {
		return BlogPostStatsSummaryDTO{}, ErrBlogPostNotFound
	}
	dates, err := normalizeBlogStatsDateRange(startDate, endDate)
	if err != nil {
		return BlogPostStatsSummaryDTO{}, err
	}
	summaries, err := getBlogPostStatsSummariesByDates([]int{postID}, dates, true)
	if err != nil {
		return BlogPostStatsSummaryDTO{}, err
	}
	return summaries[postID], nil
}

func GetBlogPostStatsSummaries(postIDs []int, days int, includeDaily bool) (map[int]BlogPostStatsSummaryDTO, error) {
	postIDs = cleanBlogPostIDs(postIDs)
	days = normalizeBlogStatsDays(days)
	dates := blogStatsDates(time.Now().UTC(), days)
	return getBlogPostStatsSummariesByDates(postIDs, dates, includeDaily)
}

func getBlogPostStatsSummariesByDates(postIDs []int, dates []string, includeDaily bool) (map[int]BlogPostStatsSummaryDTO, error) {
	postIDs = cleanBlogPostIDs(postIDs)
	summaries := make(map[int]BlogPostStatsSummaryDTO, len(postIDs))
	if len(postIDs) == 0 {
		return summaries, nil
	}
	if len(dates) == 0 {
		dates = blogStatsDates(time.Now().UTC(), defaultBlogStatsDays)
	}

	days := len(dates)
	endDate := dates[len(dates)-1]
	todayDate := time.Now().UTC().Format(blogStatsDateLayout)
	for _, postID := range postIDs {
		summary := BlogPostStatsSummaryDTO{
			Days:      days,
			StartDate: dates[0],
			EndDate:   endDate,
		}
		if includeDaily {
			summary.Daily = make([]BlogPostDailyStatsDTO, 0, len(dates))
			for _, date := range dates {
				summary.Daily = append(summary.Daily, BlogPostDailyStatsDTO{Date: date})
			}
		}
		summaries[postID] = summary
	}

	var totalRows []blogPostViewTotalRow
	if err := DB.Model(&BlogPostDailyView{}).
		Select("post_id, SUM(views) as views").
		Where("post_id IN ?", postIDs).
		Group("post_id").
		Find(&totalRows).Error; err != nil {
		return nil, err
	}
	for _, row := range totalRows {
		summary, ok := summaries[row.PostID]
		if !ok {
			continue
		}
		summary.LifetimeViews = row.Views
		summaries[row.PostID] = summary
	}

	var todayRows []blogPostViewTotalRow
	if err := DB.Model(&BlogPostDailyView{}).
		Select("post_id, SUM(views) as views").
		Where("post_id IN ? AND date = ?", postIDs, todayDate).
		Group("post_id").
		Find(&todayRows).Error; err != nil {
		return nil, err
	}
	for _, row := range todayRows {
		summary, ok := summaries[row.PostID]
		if !ok {
			continue
		}
		summary.ViewsToday = row.Views
		summaries[row.PostID] = summary
	}

	var rows []BlogPostDailyView
	if err := DB.Model(&BlogPostDailyView{}).
		Where("post_id IN ? AND date >= ? AND date <= ?", postIDs, dates[0], endDate).
		Order("date ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	dateIndex := make(map[string]int, len(dates))
	for index, date := range dates {
		dateIndex[date] = index
	}

	for _, row := range rows {
		summary, ok := summaries[row.PostID]
		if !ok {
			continue
		}
		summary.TotalViews += row.Views
		if includeDaily {
			if index, ok := dateIndex[row.Date]; ok {
				summary.Daily[index].Views += row.Views
			}
		}
		summaries[row.PostID] = summary
	}
	return summaries, nil
}

func incrementBlogPostView(postID int, at time.Time) error {
	if postID <= 0 {
		return ErrBlogPostNotFound
	}
	now := common.GetTimestamp()
	row := BlogPostDailyView{
		PostID:      postID,
		Date:        at.UTC().Format(blogStatsDateLayout),
		Views:       1,
		CreatedTime: now,
		UpdatedTime: now,
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "post_id"},
			{Name: "date"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"views":        gorm.Expr("blog_post_daily_views.views + ?", 1),
			"updated_time": now,
		}),
	}).Create(&row).Error
}

func cleanBlogPostIDs(postIDs []int) []int {
	seen := map[int]struct{}{}
	cleaned := make([]int, 0, len(postIDs))
	for _, postID := range postIDs {
		if postID <= 0 {
			continue
		}
		if _, ok := seen[postID]; ok {
			continue
		}
		seen[postID] = struct{}{}
		cleaned = append(cleaned, postID)
	}
	sort.Ints(cleaned)
	return cleaned
}

func normalizeBlogStatsDays(days int) int {
	if days <= 0 {
		return defaultBlogStatsDays
	}
	if days > maxBlogStatsDays {
		return maxBlogStatsDays
	}
	return days
}

func normalizeBlogStatsDateRange(startDate string, endDate string) ([]string, error) {
	now := time.Now().UTC()
	if endDate == "" {
		endDate = now.Format(blogStatsDateLayout)
	}
	end, err := time.Parse(blogStatsDateLayout, endDate)
	if err != nil {
		return nil, errors.New("invalid end_date")
	}
	var start time.Time
	if startDate == "" {
		start = end.AddDate(0, 0, -defaultBlogStatsDays+1)
	} else {
		start, err = time.Parse(blogStatsDateLayout, startDate)
		if err != nil {
			return nil, errors.New("invalid start_date")
		}
	}
	if start.After(end) {
		start, end = end, start
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days > maxBlogStatsDays {
		start = end.AddDate(0, 0, -maxBlogStatsDays+1)
		days = maxBlogStatsDays
	}
	return blogStatsDates(start.AddDate(0, 0, days-1), days), nil
}

func blogStatsDates(today time.Time, days int) []string {
	days = normalizeBlogStatsDays(days)
	today = today.UTC()
	start := today.AddDate(0, 0, -days+1)
	dates := make([]string, 0, days)
	for index := 0; index < days; index++ {
		dates = append(dates, start.AddDate(0, 0, index).Format(blogStatsDateLayout))
	}
	return dates
}
