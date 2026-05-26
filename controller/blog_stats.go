package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func RecordPublicBlogPostView(c *gin.Context) {
	err := model.RecordPublishedBlogPostView(c.Param("slug"))
	if errors.Is(err, model.ErrBlogPostNotFound) {
		common.ApiErrorMsg(c, "Blog post not found")
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetAdminBlogPostStats(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	if startDate != "" || endDate != "" {
		stats, err := model.GetBlogPostStatsByDateRange(id, startDate, endDate)
		if errors.Is(err, model.ErrBlogPostNotFound) {
			common.ApiErrorMsg(c, "Blog post not found")
			return
		}
		if err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, stats)
		return
	}
	days, _ := strconv.Atoi(c.Query("days"))
	stats, err := model.GetBlogPostStats(id, days)
	if errors.Is(err, model.ErrBlogPostNotFound) {
		common.ApiErrorMsg(c, "Blog post not found")
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}
