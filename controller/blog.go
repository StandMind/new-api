package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetPublicBlogPosts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	locale := model.ResolveLocalizedTextLocale(c.Query("lang"), c.GetHeader("Accept-Language"))
	items, total, err := model.ListPublishedBlogPosts(
		locale,
		c.Query("keyword"),
		c.Query("tag"),
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetItems(items)
	pageInfo.SetTotal(int(total))
	common.ApiSuccess(c, pageInfo)
}

func GetPublicBlogPost(c *gin.Context) {
	locale := model.ResolveLocalizedTextLocale(c.Query("lang"), c.GetHeader("Accept-Language"))
	post, err := model.GetPublishedBlogPostBySlug(c.Param("slug"), locale)
	if errors.Is(err, model.ErrBlogPostNotFound) {
		common.ApiErrorMsg(c, "Blog post not found")
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, post)
}

func GetAdminBlogPosts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListBlogPostsAdmin(
		c.Query("keyword"),
		c.Query("status"),
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetItems(items)
	pageInfo.SetTotal(int(total))
	common.ApiSuccess(c, pageInfo)
}

func GetAdminBlogPost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	post, err := model.GetBlogPostAdmin(id)
	if errors.Is(err, model.ErrBlogPostNotFound) {
		common.ApiErrorMsg(c, "Blog post not found")
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, post)
}

func CreateAdminBlogPost(c *gin.Context) {
	var request model.BlogPostAdminDTO
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	post, err := model.CreateBlogPost(request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, post)
}

func UpdateAdminBlogPost(c *gin.Context) {
	var request model.BlogPostAdminDTO
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if request.Id == 0 {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			common.ApiError(c, err)
			return
		}
		request.Id = id
	}
	post, err := model.UpdateBlogPost(request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, post)
}

func DeleteAdminBlogPost(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteBlogPost(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
