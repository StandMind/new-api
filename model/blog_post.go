package model

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	BlogStatusDraft     = "draft"
	BlogStatusPublished = "published"
)

var (
	ErrBlogPostNotFound = errors.New("blog post not found")
	blogSlugPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
)

type BlogPost struct {
	Id            int                   `json:"id"`
	Slug          string                `json:"slug" gorm:"size:128;not null;uniqueIndex:uk_blog_post_slug_deleted_at,priority:1"`
	Status        string                `json:"status" gorm:"size:20;not null;default:draft;index"`
	Tags          string                `json:"-" gorm:"type:text"`
	CoverImage    string                `json:"cover_image,omitempty" gorm:"type:text"`
	Author        string                `json:"author,omitempty" gorm:"size:128"`
	PublishedTime int64                 `json:"published_time" gorm:"bigint;index"`
	CreatedTime   int64                 `json:"created_time" gorm:"bigint"`
	UpdatedTime   int64                 `json:"updated_time" gorm:"bigint"`
	DeletedAt     gorm.DeletedAt        `json:"-" gorm:"index;uniqueIndex:uk_blog_post_slug_deleted_at,priority:2"`
	Translations  []BlogPostTranslation `json:"translations,omitempty" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

type BlogPostTranslation struct {
	Id          int    `json:"id"`
	PostID      int    `json:"post_id" gorm:"not null;index;uniqueIndex:uk_blog_post_translation_locale,priority:1"`
	Locale      string `json:"locale" gorm:"size:16;not null;uniqueIndex:uk_blog_post_translation_locale,priority:2"`
	Title       string `json:"title" gorm:"type:text"`
	Summary     string `json:"summary,omitempty" gorm:"type:text"`
	Content     string `json:"content" gorm:"type:text"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

type BlogPostTranslationPayload struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

type BlogPostAdminDTO struct {
	Id            int                                   `json:"id"`
	Slug          string                                `json:"slug"`
	Status        string                                `json:"status"`
	Tags          []string                              `json:"tags"`
	CoverImage    string                                `json:"cover_image"`
	Author        string                                `json:"author"`
	PublishedTime int64                                 `json:"published_time"`
	CreatedTime   int64                                 `json:"created_time"`
	UpdatedTime   int64                                 `json:"updated_time"`
	Translations  map[string]BlogPostTranslationPayload `json:"translations"`
}

type BlogPostPublicDTO struct {
	Id             int      `json:"id"`
	Slug           string   `json:"slug"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Content        string   `json:"content,omitempty"`
	Tags           []string `json:"tags"`
	CoverImage     string   `json:"cover_image,omitempty"`
	Author         string   `json:"author,omitempty"`
	PublishedTime  int64    `json:"published_time"`
	UpdatedTime    int64    `json:"updated_time"`
	Locale         string   `json:"locale"`
	ReadingMinutes int      `json:"reading_minutes"`
}

func migrateBlogTables() error {
	return DB.AutoMigrate(&BlogPost{}, &BlogPostTranslation{})
}

func NormalizeBlogSlug(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.Trim(value, "/")
	value = strings.ReplaceAll(value, " ", "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return value
}

func ValidateBlogSlug(value string) error {
	if value = NormalizeBlogSlug(value); value == "" {
		return errors.New("slug is required")
	}
	if !blogSlugPattern.MatchString(value) {
		return errors.New("slug can only contain lowercase letters, numbers, dots, underscores, and hyphens")
	}
	return nil
}

func NormalizeBlogStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case BlogStatusPublished:
		return BlogStatusPublished
	default:
		return BlogStatusDraft
	}
}

func EncodeBlogTags(tags []string) (string, error) {
	cleaned := CleanBlogTags(tags)
	if len(cleaned) == 0 {
		return "", nil
	}
	bytes, err := common.Marshal(cleaned)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func DecodeBlogTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var tags []string
	if err := common.Unmarshal([]byte(raw), &tags); err == nil {
		return CleanBlogTags(tags)
	}
	return CleanBlogTags(strings.Split(raw, ","))
}

func CleanBlogTags(tags []string) []string {
	seen := map[string]struct{}{}
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		tag = strings.Trim(tag, "#")
		if tag == "" {
			continue
		}
		if len([]rune(tag)) > 32 {
			tag = string([]rune(tag)[:32])
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		cleaned = append(cleaned, tag)
		if len(cleaned) >= 20 {
			break
		}
	}
	return cleaned
}

func ListPublishedBlogPosts(locale string, keyword string, tag string, offset int, limit int) ([]BlogPostPublicDTO, int64, error) {
	posts, total, err := listBlogPosts(blogPostPublicQuery(), keyword, tag, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	items := make([]BlogPostPublicDTO, 0, len(posts))
	for _, post := range posts {
		items = append(items, post.toPublicDTO(locale, false))
	}
	return items, total, nil
}

func GetPublishedBlogPostBySlug(slug string, locale string) (*BlogPostPublicDTO, error) {
	slug = NormalizeBlogSlug(slug)
	if slug == "" {
		return nil, ErrBlogPostNotFound
	}
	var post BlogPost
	err := blogPostPublicQuery().
		Preload("Translations").
		Where("slug = ?", slug).
		First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBlogPostNotFound
	}
	if err != nil {
		return nil, err
	}
	item := post.toPublicDTO(locale, true)
	return &item, nil
}

func ListBlogPostsAdmin(keyword string, status string, offset int, limit int) ([]BlogPostAdminDTO, int64, error) {
	query := DB.Model(&BlogPost{})
	status = strings.TrimSpace(status)
	if status != "" {
		query = query.Where("status = ?", NormalizeBlogStatus(status))
	}
	posts, total, err := listBlogPosts(query, keyword, "", offset, limit)
	if err != nil {
		return nil, 0, err
	}
	items := make([]BlogPostAdminDTO, 0, len(posts))
	for _, post := range posts {
		items = append(items, post.toAdminDTO())
	}
	return items, total, nil
}

func GetBlogPostAdmin(id int) (*BlogPostAdminDTO, error) {
	if id <= 0 {
		return nil, ErrBlogPostNotFound
	}
	var post BlogPost
	err := DB.Preload("Translations").First(&post, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBlogPostNotFound
	}
	if err != nil {
		return nil, err
	}
	item := post.toAdminDTO()
	return &item, nil
}

func CreateBlogPost(input BlogPostAdminDTO) (*BlogPostAdminDTO, error) {
	if err := normalizeBlogPostInput(&input); err != nil {
		return nil, err
	}
	if duplicated, err := isBlogSlugDuplicated(0, input.Slug); err != nil {
		return nil, err
	} else if duplicated {
		return nil, errors.New("slug already exists")
	}

	tags, err := EncodeBlogTags(input.Tags)
	if err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	if input.Status == BlogStatusPublished && input.PublishedTime == 0 {
		input.PublishedTime = now
	}
	post := BlogPost{
		Slug:          input.Slug,
		Status:        input.Status,
		Tags:          tags,
		CoverImage:    input.CoverImage,
		Author:        input.Author,
		PublishedTime: input.PublishedTime,
		CreatedTime:   now,
		UpdatedTime:   now,
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		return replaceBlogPostTranslations(tx, post.Id, input.Translations, now)
	})
	if err != nil {
		return nil, err
	}
	return GetBlogPostAdmin(post.Id)
}

func UpdateBlogPost(input BlogPostAdminDTO) (*BlogPostAdminDTO, error) {
	if input.Id <= 0 {
		return nil, errors.New("blog post id is required")
	}
	if err := normalizeBlogPostInput(&input); err != nil {
		return nil, err
	}
	if duplicated, err := isBlogSlugDuplicated(input.Id, input.Slug); err != nil {
		return nil, err
	} else if duplicated {
		return nil, errors.New("slug already exists")
	}

	tags, err := EncodeBlogTags(input.Tags)
	if err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	if input.Status == BlogStatusPublished && input.PublishedTime == 0 {
		input.PublishedTime = now
	}
	post := BlogPost{
		Id:            input.Id,
		Slug:          input.Slug,
		Status:        input.Status,
		Tags:          tags,
		CoverImage:    input.CoverImage,
		Author:        input.Author,
		PublishedTime: input.PublishedTime,
		UpdatedTime:   now,
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&BlogPost{}).Where("id = ?", input.Id).
			Select("slug", "status", "tags", "cover_image", "author", "published_time", "updated_time").
			Updates(&post).Error; err != nil {
			return err
		}
		return replaceBlogPostTranslations(tx, input.Id, input.Translations, now)
	})
	if err != nil {
		return nil, err
	}
	return GetBlogPostAdmin(input.Id)
}

func DeleteBlogPost(id int) error {
	if id <= 0 {
		return ErrBlogPostNotFound
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", id).Delete(&BlogPostTranslation{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&BlogPost{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrBlogPostNotFound
		}
		return nil
	})
}

func GetPublishedBlogSitemapSlugs() ([]string, error) {
	var posts []BlogPost
	if err := blogPostPublicQuery().Select("slug").Order("published_time DESC, id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	slugs := make([]string, 0, len(posts))
	for _, post := range posts {
		if post.Slug != "" {
			slugs = append(slugs, post.Slug)
		}
	}
	sort.Strings(slugs)
	return slugs, nil
}

func GetPublishedBlogPostSEOMeta(slug string, locale string) (*BlogPostPublicDTO, error) {
	return GetPublishedBlogPostBySlug(slug, locale)
}

func blogPostPublicQuery() *gorm.DB {
	now := common.GetTimestamp()
	return DB.Model(&BlogPost{}).Where("status = ? AND (published_time = ? OR published_time <= ?)", BlogStatusPublished, 0, now)
}

func listBlogPosts(query *gorm.DB, keyword string, tag string, offset int, limit int) ([]BlogPost, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Joins("LEFT JOIN blog_post_translations ON blog_post_translations.post_id = blog_posts.id").
			Where("blog_posts.slug LIKE ? OR blog_post_translations.title LIKE ? OR blog_post_translations.summary LIKE ?", like, like, like)
	}
	if tag = strings.TrimSpace(strings.ToLower(tag)); tag != "" {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("blog_posts.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ids []int
	if err := query.Session(&gorm.Session{}).
		Distinct("blog_posts.id").
		Order("blog_posts.published_time DESC").
		Order("blog_posts.id DESC").
		Offset(offset).
		Limit(limit).
		Pluck("blog_posts.id", &ids).Error; err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []BlogPost{}, total, nil
	}

	var posts []BlogPost
	if err := DB.Preload("Translations").Where("id IN ?", ids).Find(&posts).Error; err != nil {
		return nil, 0, err
	}
	sortBlogPostsByIDs(posts, ids)
	return posts, total, nil
}

func sortBlogPostsByIDs(posts []BlogPost, ids []int) {
	order := make(map[int]int, len(ids))
	for index, id := range ids {
		order[id] = index
	}
	sort.SliceStable(posts, func(i int, j int) bool {
		return order[posts[i].Id] < order[posts[j].Id]
	})
}

func normalizeBlogPostInput(input *BlogPostAdminDTO) error {
	input.Slug = NormalizeBlogSlug(input.Slug)
	if err := ValidateBlogSlug(input.Slug); err != nil {
		return err
	}
	input.Status = NormalizeBlogStatus(input.Status)
	input.Tags = CleanBlogTags(input.Tags)
	input.CoverImage = strings.TrimSpace(input.CoverImage)
	input.Author = strings.TrimSpace(input.Author)
	input.Translations = cleanBlogTranslations(input.Translations)
	if len(input.Translations) == 0 {
		return errors.New("at least one localized title or content is required")
	}
	return nil
}

func cleanBlogTranslations(translations map[string]BlogPostTranslationPayload) map[string]BlogPostTranslationPayload {
	cleaned := map[string]BlogPostTranslationPayload{}
	for locale, translation := range translations {
		locale = NormalizeLocalizedTextLocale(locale)
		if locale == "" {
			continue
		}
		item := BlogPostTranslationPayload{
			Title:   strings.TrimSpace(translation.Title),
			Summary: strings.TrimSpace(translation.Summary),
			Content: strings.TrimSpace(translation.Content),
		}
		if item.Title == "" && item.Summary == "" && item.Content == "" {
			continue
		}
		cleaned[locale] = item
	}
	return cleaned
}

func replaceBlogPostTranslations(tx *gorm.DB, postID int, translations map[string]BlogPostTranslationPayload, now int64) error {
	if err := tx.Where("post_id = ?", postID).Delete(&BlogPostTranslation{}).Error; err != nil {
		return err
	}
	rows := make([]BlogPostTranslation, 0, len(translations))
	for locale, translation := range translations {
		rows = append(rows, BlogPostTranslation{
			PostID:      postID,
			Locale:      locale,
			Title:       translation.Title,
			Summary:     translation.Summary,
			Content:     translation.Content,
			CreatedTime: now,
			UpdatedTime: now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	sort.Slice(rows, func(i int, j int) bool {
		return rows[i].Locale < rows[j].Locale
	})
	return tx.Create(&rows).Error
}

func isBlogSlugDuplicated(id int, slug string) (bool, error) {
	var count int64
	err := DB.Model(&BlogPost{}).Where("slug = ? AND id <> ?", slug, id).Count(&count).Error
	return count > 0, err
}

func (post BlogPost) toAdminDTO() BlogPostAdminDTO {
	translations := map[string]BlogPostTranslationPayload{}
	for _, translation := range post.Translations {
		locale := NormalizeLocalizedTextLocale(translation.Locale)
		if locale == "" {
			continue
		}
		translations[locale] = BlogPostTranslationPayload{
			Title:   translation.Title,
			Summary: translation.Summary,
			Content: translation.Content,
		}
	}
	return BlogPostAdminDTO{
		Id:            post.Id,
		Slug:          post.Slug,
		Status:        NormalizeBlogStatus(post.Status),
		Tags:          DecodeBlogTags(post.Tags),
		CoverImage:    post.CoverImage,
		Author:        post.Author,
		PublishedTime: post.PublishedTime,
		CreatedTime:   post.CreatedTime,
		UpdatedTime:   post.UpdatedTime,
		Translations:  translations,
	}
}

func (post BlogPost) toPublicDTO(locale string, includeContent bool) BlogPostPublicDTO {
	locale = ResolveLocalizedTextLocale(locale)
	title := post.localizedTranslationValue(locale, func(item BlogPostTranslation) string { return item.Title })
	summary := post.localizedTranslationValue(locale, func(item BlogPostTranslation) string { return item.Summary })
	content := post.localizedTranslationValue(locale, func(item BlogPostTranslation) string { return item.Content })
	if summary == "" {
		summary = makeBlogSummary(content)
	}
	readingMinutes := estimateBlogReadingMinutes(content)
	if !includeContent {
		content = ""
	}
	return BlogPostPublicDTO{
		Id:             post.Id,
		Slug:           post.Slug,
		Title:          title,
		Summary:        summary,
		Content:        content,
		Tags:           DecodeBlogTags(post.Tags),
		CoverImage:     post.CoverImage,
		Author:         post.Author,
		PublishedTime:  post.PublishedTime,
		UpdatedTime:    post.UpdatedTime,
		Locale:         locale,
		ReadingMinutes: readingMinutes,
	}
}

func (post BlogPost) localizedTranslationValue(locale string, pick func(BlogPostTranslation) string) string {
	locale = NormalizeLocalizedTextLocale(locale)
	if locale == "" {
		locale = "en"
	}
	translationsByLocale := make(map[string]BlogPostTranslation, len(post.Translations))
	for _, translation := range post.Translations {
		normalized := NormalizeLocalizedTextLocale(translation.Locale)
		if normalized != "" {
			translationsByLocale[normalized] = translation
		}
	}
	for _, candidate := range []string{locale, "en", "zh"} {
		if translation, ok := translationsByLocale[candidate]; ok {
			if value := strings.TrimSpace(pick(translation)); value != "" {
				return value
			}
		}
	}
	locales := make([]string, 0, len(translationsByLocale))
	for candidate := range translationsByLocale {
		locales = append(locales, candidate)
	}
	sort.Strings(locales)
	for _, candidate := range locales {
		if value := strings.TrimSpace(pick(translationsByLocale[candidate])); value != "" {
			return value
		}
	}
	return ""
}

func makeBlogSummary(content string) string {
	plain := strings.TrimSpace(content)
	replacements := []string{"#", "", "`", "", "*", "", "_", "", "[", "", "]", "", "(", " ", ")", " "}
	replacer := strings.NewReplacer(replacements...)
	plain = strings.Join(strings.Fields(replacer.Replace(plain)), " ")
	runes := []rune(plain)
	if len(runes) > 180 {
		return string(runes[:180])
	}
	return plain
}

func estimateBlogReadingMinutes(content string) int {
	count := len([]rune(strings.TrimSpace(content)))
	if count == 0 {
		return 0
	}
	minutes := count / 600
	if count%600 != 0 {
		minutes++
	}
	if minutes < 1 {
		return 1
	}
	return minutes
}
