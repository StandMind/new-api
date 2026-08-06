package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// videoProxyError returns a standardized OpenAI-style error response.
func videoProxyError(c *gin.Context, status int, errType, messageKey string, args ...map[string]any) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": i18n.T(c, messageKey, args...),
			"type":    errType,
		},
	})
}

func logVideoProxyError(c *gin.Context, format string, args ...any) {
	logger.LogError(c.Request.Context(), common.MaskSensitiveInfo(fmt.Sprintf(format, args...)))
}

func VideoProxy(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", i18n.MsgInvalidParams)
		return
	}

	userID := c.GetInt("id")
	task, exists, err := model.GetByTaskId(userID, taskID)
	if err != nil {
		logVideoProxyError(c, "Failed to query task %s: %s", taskID, err.Error())
		videoProxyError(c, http.StatusInternalServerError, "server_error", i18n.MsgOperationFailed)
		return
	}
	if !exists || task == nil {
		videoProxyError(c, http.StatusNotFound, "invalid_request_error", i18n.MsgNotFound)
		return
	}

	if task.Status != model.TaskStatusSuccess {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", i18n.MsgInvalidParams)
		return
	}

	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		logVideoProxyError(c, "Failed to get channel for task %s: %s", taskID, err.Error())
		videoProxyError(c, http.StatusInternalServerError, "server_error", i18n.MsgOperationFailed)
		return
	}
	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	var videoURL string
	proxy := channel.GetSetting().Proxy
	client := service.GetSSRFProtectedHTTPClient()
	if proxy != "" {
		// 渠道代理路径的连接由代理侧建立，无法做拨号时逐 IP 校验，
		// 因此后面对 videoURL 保留请求前的一次性 SSRF 校验。
		client, err = service.GetHttpClientWithProxy(proxy)
		if err != nil {
			logVideoProxyError(c, "Failed to create proxy client for task %s: %s", taskID, err.Error())
			videoProxyError(c, http.StatusInternalServerError, "server_error", i18n.MsgOperationFailed)
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	if err != nil {
		logVideoProxyError(c, "Failed to create request: %s", err.Error())
		videoProxyError(c, http.StatusInternalServerError, "server_error", i18n.MsgOperationFailed)
		return
	}

	switch channel.Type {
	case constant.ChannelTypeGemini:
		apiKey := task.PrivateData.Key
		if apiKey == "" {
			logVideoProxyError(c, "Missing stored API key for Gemini task %s", taskID)
			videoProxyError(c, http.StatusInternalServerError, "server_error", i18n.MsgRelayChannelKeyInvalid)
			return
		}
		videoURL, err = getGeminiVideoURL(channel, task, apiKey)
		if err != nil {
			logVideoProxyError(c, "Failed to resolve Gemini video URL for task %s: %s", taskID, err.Error())
			videoProxyError(c, http.StatusBadGateway, "server_error", i18n.MsgRelayInvalidUpstreamResponse)
			return
		}
		req.Header.Set("x-goog-api-key", apiKey)
	case constant.ChannelTypeVertexAi:
		videoURL, err = getVertexVideoURL(channel, task)
		if err != nil {
			logVideoProxyError(c, "Failed to resolve Vertex video URL for task %s: %s", taskID, err.Error())
			videoProxyError(c, http.StatusBadGateway, "server_error", i18n.MsgRelayInvalidUpstreamResponse)
			return
		}
	case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
		videoURL = fmt.Sprintf("%s/v1/videos/%s/content", baseURL, task.GetUpstreamTaskID())
		req.Header.Set("Authorization", "Bearer "+channel.Key)
	default:
		// Video URL is stored in PrivateData.ResultURL (fallback to FailReason for old data)
		videoURL = task.GetResultURL()
	}

	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		logVideoProxyError(c, "Video URL is empty for task %s", taskID)
		videoProxyError(c, http.StatusBadGateway, "server_error", i18n.MsgRelayUpstreamRequestFailed)
		return
	}

	if strings.HasPrefix(videoURL, "data:") {
		if err := writeVideoDataURL(c, videoURL); err != nil {
			logVideoProxyError(c, "Failed to decode video data URL for task %s: %s", taskID, err.Error())
			videoProxyError(c, http.StatusBadGateway, "server_error", i18n.MsgRelayInvalidUpstreamResponse)
		}
		return
	}

	var validateErr error
	if proxy == "" {
		validateErr = service.ValidateSSRFProtectedFetchURL(videoURL)
	} else {
		fetchSetting := system_setting.GetFetchSetting()
		validateErr = common.ValidateURLWithFetchSetting(videoURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain)
	}
	if validateErr != nil {
		logVideoProxyError(c, "Video URL blocked for task %s: %v", taskID, validateErr)
		videoProxyError(c, http.StatusForbidden, "server_error", i18n.MsgForbidden)
		return
	}

	req.URL, err = url.Parse(videoURL)
	if err != nil {
		logVideoProxyError(c, "Failed to parse URL %s: %s", videoURL, err.Error())
		videoProxyError(c, http.StatusInternalServerError, "server_error", i18n.MsgOperationFailed)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		logVideoProxyError(c, "Failed to fetch video from %s: %s", videoURL, err.Error())
		videoProxyError(c, http.StatusBadGateway, "server_error", i18n.MsgRelayUpstreamRequestFailed)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logVideoProxyError(c, "Upstream returned status %d for %s", resp.StatusCode, videoURL)
		videoProxyError(c, http.StatusBadGateway, "server_error", i18n.MsgRelayInvalidUpstreamResponse)
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		logVideoProxyError(c, "Failed to stream video content: %s", err.Error())
	}
}

func writeVideoDataURL(c *gin.Context, dataURL string) error {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid data url")
	}

	header := parts[0]
	payload := parts[1]
	if !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return fmt.Errorf("unsupported data url")
	}

	mimeType := strings.TrimPrefix(header, "data:")
	mimeType = strings.TrimSuffix(mimeType, ";base64")
	if mimeType == "" {
		mimeType = "video/mp4"
	}

	videoBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		videoBytes, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return err
		}
	}

	c.Writer.Header().Set("Content-Type", mimeType)
	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	c.Writer.WriteHeader(http.StatusOK)
	_, err = c.Writer.Write(videoBytes)
	return err
}
