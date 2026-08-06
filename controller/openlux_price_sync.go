package controller

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/openluxsync"
	"github.com/gin-gonic/gin"
)

func GetOpenLuxPriceSyncBindings(c *gin.Context) {
	response, err := openluxsync.GetBindings(c.Request.Context())
	if err != nil {
		respondOpenLuxSyncError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func PutOpenLuxPriceSyncBindings(c *gin.Context) {
	var request openluxsync.SaveBindingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondOpenLuxSyncRequestError(c)
		return
	}
	response, err := openluxsync.SaveBindings(c.Request.Context(), request)
	if err != nil {
		respondOpenLuxSyncError(c, err)
		return
	}
	channelCount := 0
	for _, binding := range response.Bindings {
		channelCount += len(binding.Channels)
	}
	recordManageAudit(c, "openlux_price_sync.bindings", map[string]interface{}{
		"binding_revision":   response.BindingRevision,
		"source_group_count": len(response.Bindings),
		"channel_count":      channelCount,
	})
	common.ApiSuccess(c, response)
}

func PreviewOpenLuxPriceSync(c *gin.Context) {
	response, err := openluxsync.Preview(c.Request.Context())
	if err != nil {
		respondOpenLuxSyncError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func ApplyOpenLuxPriceSync(c *gin.Context) {
	var request openluxsync.ApplyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondOpenLuxSyncRequestError(c)
		return
	}
	response, err := openluxsync.Apply(c.Request.Context(), request)
	if err != nil {
		respondOpenLuxSyncError(c, err)
		return
	}
	service.ResetProxyClientCache()
	recordManageAudit(c, "openlux_price_sync.apply", map[string]interface{}{
		"applied_count":       response.AppliedCount,
		"binding_revision":    response.BindingRevision,
		"source_hash":         response.SourceHash,
		"updated_models":      response.UpdatedModels,
		"updated_groups":      response.UpdatedGroups,
		"removed_channel_ids": response.RemovedChannels,
	})
	common.ApiSuccess(c, response)
}

func respondOpenLuxSyncRequestError(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"code":    "invalid_request",
		"message": i18n.T(c, i18n.MsgInvalidParams),
	})
}

func respondOpenLuxSyncError(c *gin.Context, err error) {
	var serviceErr *openluxsync.ServiceError
	if errors.As(err, &serviceErr) {
		message := i18n.T(c, i18n.MsgOperationFailed)
		switch serviceErr.Code {
		case "invalid_request":
			message = i18n.T(c, i18n.MsgInvalidParams)
		case "stale_preview":
			message = i18n.T(c, i18n.MsgRetryLater)
		case "upstream_failure":
			message = common.MaskSensitiveInfo(serviceErr.Message)
		case "upstream_request", "upstream_unavailable", "upstream_read", "upstream_too_large", "upstream_json", "upstream_schema", "unsupported_upstream_schema", "upstream_hash":
			message = i18n.T(c, i18n.MsgRelayInvalidUpstreamResponse)
		}
		c.JSON(serviceErr.Status, gin.H{
			"success": false,
			"code":    serviceErr.Code,
			"message": message,
		})
		return
	}
	logger.LogError(c.Request.Context(), "OpenLux price sync failed: "+err.Error())
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"code":    "internal_error",
		"message": i18n.T(c, i18n.MsgOperationFailed),
	})
}
