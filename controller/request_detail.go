package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/request_detail_setting"

	"github.com/gin-gonic/gin"
)

func GetRequestDetails(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	outcome := strings.TrimSpace(c.Query("outcome"))
	if outcome != "" && outcome != model.RequestDetailOutcomeSuccess && outcome != model.RequestDetailOutcomeFailed {
		common.ApiErrorMsg(c, "invalid request detail outcome")
		return
	}
	details, total, err := model.GetRequestDetails(model.RequestDetailFilter{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Username:       strings.TrimSpace(c.Query("username")),
		ModelName:      strings.TrimSpace(c.Query("model_name")),
		Outcome:        outcome,
		RequestID:      strings.TrimSpace(c.Query("request_id")),
		StartIndex:     pageInfo.GetStartIdx(),
		Limit:          pageInfo.GetPageSize(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(details)
	common.ApiSuccess(c, pageInfo)
}

func GetRequestDetail(c *gin.Context) {
	requestID := strings.TrimSpace(c.Param("request_id"))
	if requestID == "" {
		common.ApiErrorMsg(c, "request id is required")
		return
	}
	detail, err := model.GetRequestDetailByRequestID(requestID)
	if err != nil {
		if model.IsRequestDetailNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "request detail not found or expired",
			})
			return
		}
		common.ApiError(c, err)
		return
	}
	payload, err := service.DecodeRequestDetailPayload(detail.Payload)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"detail":  detail,
		"payload": payload,
	})
}

func GetRequestDetailStats(c *gin.Context) {
	stats, err := model.GetRequestDetailStorageStats(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	settings := request_detail_setting.GetSetting()
	maxStorageBytes := int64(settings.MaxStorageMB) << 20
	usagePercent := float64(0)
	if maxStorageBytes > 0 {
		usagePercent = float64(stats.StorageBytes) / float64(maxStorageBytes) * 100
	}
	common.ApiSuccess(c, gin.H{
		"storage":           stats,
		"runtime":           service.GetRequestDetailRuntimeStats(),
		"mode":              settings.Mode,
		"retention_days":    settings.RetentionDays,
		"max_storage_bytes": maxStorageBytes,
		"usage_percent":     usagePercent,
	})
}
