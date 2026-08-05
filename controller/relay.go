package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/samber/lo"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func relayHandler(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	var err *types.NewAPIError
	switch info.RelayMode {
	case relayconstant.RelayModeImagesGenerations, relayconstant.RelayModeImagesEdits:
		err = relay.ImageHelper(c, info)
	case relayconstant.RelayModeAudioSpeech:
		fallthrough
	case relayconstant.RelayModeAudioTranslation:
		fallthrough
	case relayconstant.RelayModeAudioTranscription:
		err = relay.AudioHelper(c, info)
	case relayconstant.RelayModeRerank:
		err = relay.RerankHelper(c, info)
	case relayconstant.RelayModeEmbeddings:
		err = relay.EmbeddingHelper(c, info)
	case relayconstant.RelayModeResponses, relayconstant.RelayModeResponsesCompact:
		err = relay.ResponsesHelper(c, info)
	default:
		err = relay.TextHelper(c, info)
	}
	return err
}

func geminiRelayHandler(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	var err *types.NewAPIError
	if strings.Contains(c.Request.URL.Path, "embed") {
		err = relay.GeminiEmbeddingHandler(c, info)
	} else {
		err = relay.GeminiHelper(c, info)
	}
	return err
}

func Relay(c *gin.Context, relayFormat types.RelayFormat) {

	requestId := c.GetString(common.RequestIdKey)
	//group := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	//originalModel := common.GetContextKeyString(c, constant.ContextKeyOriginalModel)

	var (
		newAPIError *types.NewAPIError
		ws          *websocket.Conn
	)
	defer func() {
		if newAPIError != nil {
			service.EnsureRouteFinalStopReason(c, "request_failed_before_upstream")
			service.SetRequestDetailFailure(c, &service.RequestDetailFailure{
				StatusCode: newAPIError.StatusCode,
				ErrorType:  string(newAPIError.GetErrorType()),
				ErrorCode:  string(newAPIError.GetErrorCode()),
				Message:    newAPIError.MaskSensitiveErrorWithStatusCode(),
			})
		}
	}()

	if relayFormat == types.RelayFormatOpenAIRealtime {
		var err error
		ws, err = upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			newAPIError = types.NewError(err, types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
			helper.WssError(c, ws, newAPIError.ToOpenAIError())
			return
		}
		defer ws.Close()
	}

	defer func() {
		if newAPIError != nil {
			logger.LogError(c, fmt.Sprintf("relay error: %s", common.LocalLogPreview(newAPIError.Error())))
			newAPIError.SetMessage(common.MessageWithRequestId(newAPIError.Error(), requestId))
			switch relayFormat {
			case types.RelayFormatOpenAIRealtime:
				helper.WssError(c, ws, newAPIError.ToOpenAIError())
			case types.RelayFormatClaude:
				c.JSON(newAPIError.StatusCode, gin.H{
					"type":  "error",
					"error": newAPIError.ToClaudeError(),
				})
			default:
				c.JSON(newAPIError.StatusCode, gin.H{
					"error": newAPIError.ToOpenAIError(),
				})
			}
		}
	}()

	request, err := helper.GetAndValidateRequest(c, relayFormat)
	if err != nil {
		// Map "request body too large" to 413 so clients can handle it correctly
		if common.IsRequestBodyTooLargeError(err) || errors.Is(err, common.ErrRequestBodyTooLarge) {
			newAPIError = types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusRequestEntityTooLarge, types.ErrOptionWithSkipRetry())
		} else {
			newAPIError = types.NewError(err, types.ErrorCodeInvalidRequest)
		}
		return
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, request, ws)
	if err != nil {
		newAPIError = types.NewError(err, types.ErrorCodeGenRelayInfoFailed)
		return
	}
	requestBillingRatios, err := helper.NormalizeRequestPricePolicy(relayInfo.OriginModelName, request)
	if err != nil {
		newAPIError = types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		return
	}
	if len(requestBillingRatios) > 0 {
		if err = helper.SyncRequestPricePolicyBody(c, request); err != nil {
			newAPIError = types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			return
		}
	}

	needSensitiveCheck := setting.ShouldCheckPromptSensitive()
	needCountToken := constant.CountToken
	// Avoid building huge CombineText (strings.Join) when token counting and sensitive check are both disabled.
	var meta *types.TokenCountMeta
	if needSensitiveCheck || needCountToken {
		meta = request.GetTokenCountMeta()
	} else {
		meta = fastTokenCountMetaForPricing(request)
	}
	if len(requestBillingRatios) > 0 {
		if meta.BillingRatios == nil {
			meta.BillingRatios = make(map[string]float64, len(requestBillingRatios))
		}
		for name, ratio := range requestBillingRatios {
			meta.BillingRatios[name] = ratio
		}
	}

	if needSensitiveCheck && meta != nil {
		contains, words := service.CheckSensitiveText(meta.CombineText)
		if contains {
			logger.LogWarn(c, fmt.Sprintf("user sensitive words detected: %s", strings.Join(words, ", ")))
			newAPIError = types.NewError(err, types.ErrorCodeSensitiveWordsDetected)
			return
		}
	}

	tokens, err := service.EstimateRequestToken(c, meta, relayInfo)
	if err != nil {
		newAPIError = types.NewError(err, types.ErrorCodeCountTokenFailed)
		return
	}

	relayInfo.SetEstimatePromptTokens(tokens)

	priceData, err := helper.ModelPriceHelper(c, relayInfo, tokens, meta)
	if err != nil {
		newAPIError = types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithStatusCode(http.StatusBadRequest))
		return
	}

	// common.SetContextKey(c, constant.ContextKeyTokenCountMeta, meta)

	if priceData.FreeModel {
		logger.LogInfo(c, fmt.Sprintf("模型 %s 免费，跳过预扣费", relayInfo.OriginModelName))
	} else {
		newAPIError = service.PreConsumeBilling(c, priceData.QuotaToPreConsume, relayInfo)
		if newAPIError != nil {
			return
		}
	}
	relayInfo.PricedGroup = relayInfo.UsingGroup

	defer func() {
		// Only return quota if downstream failed and quota was actually pre-consumed
		if newAPIError != nil {
			newAPIError = service.NormalizeViolationFeeError(newAPIError)
			if relayInfo.Billing != nil {
				relayInfo.Billing.Refund(c)
			}
			service.ChargeViolationFeeIfNeeded(c, relayInfo, newAPIError)
		}
	}()

	retryParam := &service.RetryParam{
		Ctx:         c,
		TokenGroup:  relayInfo.TokenGroup,
		ModelName:   relayInfo.OriginModelName,
		RequestPath: c.Request.URL.Path,
		Retry:       common.GetPointer(0),
	}
	relayInfo.RetryIndex = 0
	relayInfo.LastError = nil

	routePlan := service.GetRouteAttemptPlan(c)
	for attemptIndex := 0; ; attemptIndex++ {
		if (routePlan == nil || !routePlan.IsExhaustive()) && attemptIndex > common.RetryTimes {
			service.SetRouteFinalStopReason(c, "retry_limit_reached")
			break
		}
		retryParam.SetRetry(attemptIndex)
		relayInfo.RetryIndex = attemptIndex
		channel, channelErr := getChannel(c, relayInfo, retryParam)
		if channelErr != nil {
			if errors.Is(channelErr.Err, service.ErrRouteAttemptPlanExhausted) && relayInfo.LastError != nil {
				service.SetRouteFinalStopReason(c, "route_plan_exhausted")
				newAPIError = relayInfo.LastError
				break
			}
			service.EnsureRouteFinalStopReason(c, "get_channel_failed")
			logger.LogError(c, channelErr.Error())
			newAPIError = channelErr
			break
		}
		if relayInfo.PricedGroup != relayInfo.UsingGroup {
			priceData, priceErr := helper.ModelPriceHelper(c, relayInfo, tokens, meta)
			if priceErr != nil {
				newAPIError = types.NewError(priceErr, types.ErrorCodeModelPriceError, types.ErrOptionWithStatusCode(http.StatusBadRequest), types.ErrOptionWithSkipRetry())
				service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
					Phase:           service.RouteAttemptPhaseBilling,
					Outcome:         service.RouteAttemptOutcomeFailed,
					Reason:          "pricing_failed",
					StatusCode:      newAPIError.StatusCode,
					ErrorType:       string(newAPIError.GetErrorType()),
					ErrorCode:       string(newAPIError.GetErrorCode()),
					ErrorMessage:    newAPIError.MaskSensitiveErrorWithStatusCode(),
					RetryDecision:   service.RouteRetryDecisionStop,
					RetryStopReason: "pricing_failed",
				})
				break
			}
			if relayInfo.Billing == nil {
				if !priceData.FreeModel {
					newAPIError = service.PreConsumeBilling(c, priceData.QuotaToPreConsume, relayInfo)
					if newAPIError != nil {
						service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
							Phase:           service.RouteAttemptPhaseBilling,
							Outcome:         service.RouteAttemptOutcomeFailed,
							Reason:          "billing_reserve_failed",
							StatusCode:      newAPIError.StatusCode,
							ErrorType:       string(newAPIError.GetErrorType()),
							ErrorCode:       string(newAPIError.GetErrorCode()),
							ErrorMessage:    newAPIError.MaskSensitiveErrorWithStatusCode(),
							RetryDecision:   service.RouteRetryDecisionStop,
							RetryStopReason: "billing_reserve_failed",
						})
						break
					}
				}
			} else if reserveErr := relayInfo.Billing.Reserve(priceData.QuotaToPreConsume); reserveErr != nil {
				newAPIError = types.NewErrorWithStatusCode(
					reserveErr,
					types.ErrorCodeInsufficientUserQuota,
					http.StatusForbidden,
					types.ErrOptionWithSkipRetry(),
				)
				service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
					Phase:           service.RouteAttemptPhaseBilling,
					Outcome:         service.RouteAttemptOutcomeFailed,
					Reason:          "billing_reserve_failed",
					StatusCode:      newAPIError.StatusCode,
					ErrorType:       string(newAPIError.GetErrorType()),
					ErrorCode:       string(newAPIError.GetErrorCode()),
					ErrorMessage:    newAPIError.MaskSensitiveErrorWithStatusCode(),
					RetryDecision:   service.RouteRetryDecisionStop,
					RetryStopReason: "billing_reserve_failed",
				})
				break
			}
			relayInfo.PricedGroup = relayInfo.UsingGroup
		}

		bodyStorage, bodyErr := common.GetBodyStorage(c)
		if bodyErr != nil {
			// Ensure consistent 413 for oversized bodies even when error occurs later (e.g., retry path)
			if common.IsRequestBodyTooLargeError(bodyErr) || errors.Is(bodyErr, common.ErrRequestBodyTooLarge) {
				newAPIError = types.NewErrorWithStatusCode(bodyErr, types.ErrorCodeReadRequestBodyFailed, http.StatusRequestEntityTooLarge, types.ErrOptionWithSkipRetry())
			} else {
				newAPIError = types.NewErrorWithStatusCode(bodyErr, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
				Phase:           service.RouteAttemptPhaseRequest,
				Outcome:         service.RouteAttemptOutcomeFailed,
				Reason:          "request_body_unavailable",
				StatusCode:      newAPIError.StatusCode,
				ErrorType:       string(newAPIError.GetErrorType()),
				ErrorCode:       string(newAPIError.GetErrorCode()),
				ErrorMessage:    newAPIError.MaskSensitiveErrorWithStatusCode(),
				RetryDecision:   service.RouteRetryDecisionStop,
				RetryStopReason: "request_body_unavailable",
			})
			break
		}
		c.Request.Body = io.NopCloser(bodyStorage)
		service.RecordRouteUpstreamAttempt(c)
		addUsedChannel(c, channel.Id)
		upstreamStartedAt := time.Now()

		switch relayFormat {
		case types.RelayFormatOpenAIRealtime:
			newAPIError = relay.WssHelper(c, relayInfo)
		case types.RelayFormatClaude:
			newAPIError = relay.ClaudeHelper(c, relayInfo)
		case types.RelayFormatGemini:
			newAPIError = geminiRelayHandler(c, relayInfo)
		default:
			newAPIError = relayHandler(c, relayInfo)
		}

		if newAPIError == nil {
			service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
				Phase:             service.RouteAttemptPhaseUpstream,
				Outcome:           service.RouteAttemptOutcomeSucceeded,
				DurationMS:        time.Since(upstreamStartedAt).Milliseconds(),
				RetryDecision:     service.RouteRetryDecisionComplete,
				RetryReason:       "success",
				UpstreamModel:     relayInfo.UpstreamModelName,
				RequestFormat:     string(relayInfo.GetFinalRequestRelayFormat()),
				UpstreamRequestID: c.GetString(common.UpstreamRequestIdKey),
			})
			service.SetRouteFinalStopReason(c, "success")
			relayInfo.LastError = nil
			return
		}

		newAPIError = service.NormalizeViolationFeeError(newAPIError)
		relayInfo.LastError = newAPIError

		decision := relayRetryDecision{}
		if relayInfo.IsStream && (relayInfo.SendResponseCount > 0 || c.Writer.Size() > 0) {
			decision = relayRetryDecision{Reason: "response_started"}
		} else {
			retryAllowance := common.RetryTimes - attemptIndex
			if routePlan != nil && routePlan.IsExhaustive() {
				retryAllowance = 1
			}
			decision = getRelayRetryDecision(c, newAPIError, retryAllowance)
			if decision.Retry && routePlan != nil && !routePlan.HasNextAvailable() {
				decision = relayRetryDecision{Reason: "route_plan_exhausted"}
			}
		}
		retryDecision := service.RouteRetryDecisionStop
		retryStopReason := decision.Reason
		if decision.Retry {
			retryDecision = service.RouteRetryDecisionRetry
			retryStopReason = ""
		}
		service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
			Phase:             service.RouteAttemptPhaseUpstream,
			Outcome:           service.RouteAttemptOutcomeFailed,
			StatusCode:        newAPIError.StatusCode,
			ErrorType:         string(newAPIError.GetErrorType()),
			ErrorCode:         string(newAPIError.GetErrorCode()),
			ErrorMessage:      newAPIError.MaskSensitiveErrorWithStatusCode(),
			DurationMS:        time.Since(upstreamStartedAt).Milliseconds(),
			RetryDecision:     retryDecision,
			RetryReason:       decision.Reason,
			RetryStopReason:   retryStopReason,
			UpstreamModel:     relayInfo.UpstreamModelName,
			RequestFormat:     string(relayInfo.GetFinalRequestRelayFormat()),
			UpstreamRequestID: c.GetString(common.UpstreamRequestIdKey),
		})

		processChannelError(c, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()), newAPIError, !decision.Retry)
		if !decision.Retry {
			break
		}
	}

	useChannel := c.GetStringSlice("use_channel")
	if len(useChannel) > 1 {
		retryLogStr := fmt.Sprintf("重试：%s", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(useChannel)), "->"), "[]"))
		logger.LogInfo(c, retryLogStr)
	}
	if newAPIError != nil {
		perfmetrics.RecordRelaySampleAsync(c, relayInfo, false, 0)
	}
}

var upgrader = websocket.Upgrader{
	Subprotocols: []string{"realtime"}, // WS 握手支持的协议，如果有使用 Sec-WebSocket-Protocol，则必须在此声明对应的 Protocol TODO add other protocol
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

func addUsedChannel(c *gin.Context, channelId int) {
	useChannel := c.GetStringSlice("use_channel")
	useChannel = append(useChannel, fmt.Sprintf("%d", channelId))
	c.Set("use_channel", useChannel)
}

func fastTokenCountMetaForPricing(request dto.Request) *types.TokenCountMeta {
	if request == nil {
		return &types.TokenCountMeta{}
	}
	meta := &types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
	}
	switch r := request.(type) {
	case *dto.GeneralOpenAIRequest:
		maxCompletionTokens := lo.FromPtrOr(r.MaxCompletionTokens, uint(0))
		maxTokens := lo.FromPtrOr(r.MaxTokens, uint(0))
		if maxCompletionTokens > maxTokens {
			meta.MaxTokens = int(maxCompletionTokens)
		} else {
			meta.MaxTokens = int(maxTokens)
		}
	case *dto.OpenAIResponsesRequest:
		meta.MaxTokens = int(lo.FromPtrOr(r.MaxOutputTokens, uint(0)))
	case *dto.ClaudeRequest:
		meta.MaxTokens = int(lo.FromPtr(r.MaxTokens))
	case *dto.ImageRequest:
		// Pricing for image requests depends on ImagePriceRatio; safe to compute even when CountToken is disabled.
		return r.GetTokenCountMeta()
	default:
		// Best-effort: leave CombineText empty to avoid large allocations.
	}
	return meta
}

func getChannel(c *gin.Context, info *relaycommon.RelayInfo, retryParam *service.RetryParam) (*model.Channel, *types.NewAPIError) {
	if info.ChannelMeta == nil {
		autoBan := c.GetBool("auto_ban")
		autoBanInt := 1
		if !autoBan {
			autoBanInt = 0
		}
		return &model.Channel{
			Id:      c.GetInt("channel_id"),
			Type:    c.GetInt("channel_type"),
			Name:    c.GetString("channel_name"),
			AutoBan: &autoBanInt,
		}, nil
	}
	if routePlan := service.GetRouteAttemptPlan(c); routePlan != nil {
		for {
			attempt, ok := routePlan.NextAfterFailure()
			if !ok {
				service.SetRouteFinalStopReason(c, "route_plan_exhausted")
				return nil, types.NewError(service.ErrRouteAttemptPlanExhausted, types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
			}
			selectionStartedAt := time.Now()
			channel, err := service.ApplyRouteAttempt(c, attempt)
			if err != nil {
				service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
					Phase:        service.RouteAttemptPhaseSelection,
					Outcome:      service.RouteAttemptOutcomeSkipped,
					Reason:       "channel_unavailable",
					ErrorType:    "selection_error",
					ErrorCode:    "channel_unavailable",
					ErrorMessage: err.Error(),
					DurationMS:   time.Since(selectionStartedAt).Milliseconds(),
				})
				continue
			}
			info.UsingGroup = attempt.Group
			newAPIError := middleware.SetupContextForSelectedChannel(c, channel, info.OriginModelName)
			if newAPIError != nil {
				if types.IsChannelError(newAPIError) {
					service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
						Phase:        service.RouteAttemptPhaseSelection,
						Outcome:      service.RouteAttemptOutcomeSkipped,
						Reason:       "channel_initialization_failed",
						StatusCode:   newAPIError.StatusCode,
						ErrorType:    string(newAPIError.GetErrorType()),
						ErrorCode:    string(newAPIError.GetErrorCode()),
						ErrorMessage: newAPIError.MaskSensitiveErrorWithStatusCode(),
						DurationMS:   time.Since(selectionStartedAt).Milliseconds(),
					})
					continue
				}
				service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
					Phase:           service.RouteAttemptPhaseSelection,
					Outcome:         service.RouteAttemptOutcomeFailed,
					Reason:          "channel_initialization_failed",
					StatusCode:      newAPIError.StatusCode,
					ErrorType:       string(newAPIError.GetErrorType()),
					ErrorCode:       string(newAPIError.GetErrorCode()),
					ErrorMessage:    newAPIError.MaskSensitiveErrorWithStatusCode(),
					DurationMS:      time.Since(selectionStartedAt).Milliseconds(),
					RetryDecision:   service.RouteRetryDecisionStop,
					RetryStopReason: "channel_initialization_failed",
				})
				return nil, newAPIError
			}
			return channel, nil
		}
	}
	channel, selectGroup, err := service.CacheGetRandomSatisfiedChannel(retryParam)

	info.PriceData.GroupRatioInfo = helper.HandleGroupRatio(c, info)

	if err != nil {
		return nil, types.NewError(fmt.Errorf("获取分组 %s 下模型 %s 的可用渠道失败（retry）: %s", selectGroup, info.OriginModelName, err.Error()), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}
	if channel == nil {
		return nil, types.NewError(fmt.Errorf("分组 %s 下模型 %s 的可用渠道不存在（retry）", selectGroup, info.OriginModelName), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}

	newAPIError := middleware.SetupContextForSelectedChannel(c, channel, info.OriginModelName)
	if newAPIError != nil {
		return nil, newAPIError
	}
	return channel, nil
}

type relayRetryDecision struct {
	Retry  bool
	Reason string
}

func shouldRetry(c *gin.Context, openaiErr *types.NewAPIError, retryTimes int) bool {
	return getRelayRetryDecision(c, openaiErr, retryTimes).Retry
}

func getRelayRetryDecision(c *gin.Context, openaiErr *types.NewAPIError, retryTimes int) relayRetryDecision {
	if openaiErr == nil {
		return relayRetryDecision{Reason: "no_error"}
	}
	if c != nil && c.Request != nil && c.Request.Context().Err() != nil {
		if errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			return relayRetryDecision{Reason: "client_context_deadline_exceeded"}
		}
		return relayRetryDecision{Reason: "client_context_canceled"}
	}
	if service.ShouldSkipRetryAfterChannelAffinityFailure(c) {
		return relayRetryDecision{Reason: "channel_affinity_retry_suppressed"}
	}
	if retryTimes <= 0 {
		return relayRetryDecision{Reason: "retry_limit_reached"}
	}
	if _, ok := c.Get("specific_channel_id"); ok {
		return relayRetryDecision{Reason: "specific_channel"}
	}
	if types.IsChannelError(openaiErr) {
		return relayRetryDecision{Retry: true, Reason: "channel_error"}
	}
	if types.IsSkipRetryError(openaiErr) {
		return relayRetryDecision{Reason: "skip_retry_error"}
	}
	code := openaiErr.StatusCode
	if code >= 200 && code < 300 {
		return relayRetryDecision{Reason: "successful_status"}
	}
	if code < 100 || code > 599 {
		return relayRetryDecision{Retry: true, Reason: "invalid_status_code"}
	}
	if operation_setting.IsAlwaysSkipRetryCode(openaiErr.GetErrorCode()) {
		return relayRetryDecision{Reason: "non_retryable_error_code"}
	}
	if operation_setting.ShouldRetryByStatusCode(code) {
		return relayRetryDecision{Retry: true, Reason: "retryable_status"}
	}
	return relayRetryDecision{Reason: "non_retryable_status"}
}

func processChannelError(c *gin.Context, channelError types.ChannelError, err *types.NewAPIError, finalFailure bool) {
	logger.LogError(c, fmt.Sprintf("channel error (channel #%d, status code: %d): %s", channelError.ChannelId, err.StatusCode, common.LocalLogPreview(err.Error())))
	// 不要使用context获取渠道信息，异步处理时可能会出现渠道信息不一致的情况
	// do not use context to get channel info, there may be inconsistent channel info when processing asynchronously
	if service.ShouldDisableChannel(err) && channelError.AutoBan {
		gopool.Go(func() {
			service.DisableChannel(channelError, err.ErrorWithStatusCode())
		})
	}

	if constant.ErrorLogEnabled && types.IsRecordErrorLog(err) {
		// 保存错误日志到mysql中
		userId := c.GetInt("id")
		tokenName := c.GetString("token_name")
		modelName := c.GetString("original_model")
		tokenId := c.GetInt("token_id")
		userGroup := c.GetString("group")
		channelId := c.GetInt("channel_id")
		other := make(map[string]interface{})
		if c.Request != nil && c.Request.URL != nil {
			other["request_path"] = c.Request.URL.Path
		}
		other["error_type"] = err.GetErrorType()
		other["error_code"] = err.GetErrorCode()
		other["status_code"] = err.StatusCode
		other["channel_id"] = channelId
		other["channel_name"] = c.GetString("channel_name")
		other["channel_type"] = c.GetInt("channel_type")
		adminInfo := make(map[string]interface{})
		adminInfo["use_channel"] = c.GetStringSlice("use_channel")
		isMultiKey := common.GetContextKeyBool(c, constant.ContextKeyChannelIsMultiKey)
		if isMultiKey {
			adminInfo["is_multi_key"] = true
			adminInfo["multi_key_index"] = common.GetContextKeyInt(c, constant.ContextKeyChannelMultiKeyIndex)
		}
		service.AppendChannelAffinityAdminInfo(c, adminInfo)
		other["admin_info"] = adminInfo
		service.AppendRoutingErrorLogInfo(c, other, finalFailure)
		startTime := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
		if startTime.IsZero() {
			startTime = time.Now()
		}
		useTimeSeconds := int(time.Since(startTime).Seconds())
		model.RecordErrorLog(c, userId, channelId, modelName, tokenName, err.MaskSensitiveErrorWithStatusCode(), tokenId, useTimeSeconds, common.GetContextKeyBool(c, constant.ContextKeyIsStream), userGroup, other)
	}

}

func RelayMidjourney(c *gin.Context) {
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatMjProxy, nil, nil)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"description": fmt.Sprintf("failed to generate relay info: %s", err.Error()),
			"type":        "upstream_error",
			"code":        4,
		})
		return
	}

	var mjErr *dto.MidjourneyResponse
	switch relayInfo.RelayMode {
	case relayconstant.RelayModeMidjourneyNotify:
		mjErr = relay.RelayMidjourneyNotify(c)
	case relayconstant.RelayModeMidjourneyTaskFetch, relayconstant.RelayModeMidjourneyTaskFetchByCondition:
		mjErr = relay.RelayMidjourneyTask(c, relayInfo.RelayMode)
	case relayconstant.RelayModeMidjourneyTaskImageSeed:
		mjErr = relay.RelayMidjourneyTaskImageSeed(c)
	case relayconstant.RelayModeSwapFace:
		mjErr = relay.RelaySwapFace(c, relayInfo)
	default:
		mjErr = relay.RelayMidjourneySubmit(c, relayInfo)
	}
	//err = relayMidjourneySubmit(c, relayMode)
	log.Println(mjErr)
	if mjErr != nil {
		statusCode := http.StatusBadRequest
		if mjErr.Code == 30 {
			mjErr.Result = "当前分组负载已饱和，请稍后再试，或升级账户以提升服务质量。"
			statusCode = http.StatusTooManyRequests
		}
		c.JSON(statusCode, gin.H{
			"description": fmt.Sprintf("%s %s", mjErr.Description, mjErr.Result),
			"type":        "upstream_error",
			"code":        mjErr.Code,
		})
		channelId := c.GetInt("channel_id")
		logger.LogError(c, fmt.Sprintf("relay error (channel #%d, status code %d): %s", channelId, statusCode, fmt.Sprintf("%s %s", mjErr.Description, mjErr.Result)))
	}
}

func RelayNotImplemented(c *gin.Context) {
	err := types.OpenAIError{
		Message: "API not implemented",
		Type:    "new_api_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := types.OpenAIError{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}

func RelayTaskFetch(c *gin.Context) {
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, &dto.TaskError{
			Code:       "gen_relay_info_failed",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}
	if taskErr := relay.RelayTaskFetch(c, relayInfo.RelayMode); taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

func RelayTask(c *gin.Context) {
	var taskErr *dto.TaskError
	defer func() {
		if taskErr != nil {
			service.EnsureRouteFinalStopReason(c, "request_failed_before_upstream")
			service.SetRequestDetailFailure(c, &service.RequestDetailFailure{
				StatusCode: taskErr.StatusCode,
				ErrorType:  "task_error",
				ErrorCode:  taskErr.Code,
				Message:    taskErr.Message,
			})
		}
	}()

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		taskErr = &dto.TaskError{
			Code:       "gen_relay_info_failed",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		}
		respondTaskError(c, taskErr)
		return
	}

	if taskErr = relay.ResolveOriginTask(c, relayInfo); taskErr != nil {
		respondTaskError(c, taskErr)
		return
	}

	var result *relay.TaskSubmitResult
	defer func() {
		if taskErr != nil && relayInfo.Billing != nil {
			relayInfo.Billing.Refund(c)
		}
	}()

	retryParam := &service.RetryParam{
		Ctx:         c,
		TokenGroup:  relayInfo.TokenGroup,
		ModelName:   relayInfo.OriginModelName,
		RequestPath: c.Request.URL.Path,
		Retry:       common.GetPointer(0),
	}

	routePlan := service.GetRouteAttemptPlan(c)
	if relayInfo.LockedChannel != nil {
		routePlan = nil
		c.Set("specific_channel_id", "locked-task-channel")
	}
	for attemptIndex := 0; ; attemptIndex++ {
		if (routePlan == nil || !routePlan.IsExhaustive()) && attemptIndex > common.RetryTimes {
			service.SetRouteFinalStopReason(c, "retry_limit_reached")
			break
		}
		retryParam.SetRetry(attemptIndex)
		var channel *model.Channel

		if lockedCh, ok := relayInfo.LockedChannel.(*model.Channel); ok && lockedCh != nil {
			channel = lockedCh
			if attemptIndex > 0 {
				if setupErr := middleware.SetupContextForSelectedChannel(c, channel, relayInfo.OriginModelName); setupErr != nil {
					taskErr = service.TaskErrorWrapperLocal(setupErr.Err, "setup_locked_channel_failed", http.StatusInternalServerError)
					service.SetRouteFinalStopReason(c, "channel_initialization_failed")
					break
				}
			}
		} else {
			var channelErr *types.NewAPIError
			channel, channelErr = getChannel(c, relayInfo, retryParam)
			if channelErr != nil {
				if errors.Is(channelErr.Err, service.ErrRouteAttemptPlanExhausted) && taskErr != nil {
					service.SetRouteFinalStopReason(c, "route_plan_exhausted")
					break
				}
				service.EnsureRouteFinalStopReason(c, "get_channel_failed")
				logger.LogError(c, channelErr.Error())
				taskErr = service.TaskErrorWrapperLocal(channelErr.Err, "get_channel_failed", http.StatusInternalServerError)
				break
			}
		}

		bodyStorage, bodyErr := common.GetBodyStorage(c)
		if bodyErr != nil {
			if common.IsRequestBodyTooLargeError(bodyErr) || errors.Is(bodyErr, common.ErrRequestBodyTooLarge) {
				taskErr = service.TaskErrorWrapperLocal(bodyErr, "read_request_body_failed", http.StatusRequestEntityTooLarge)
			} else {
				taskErr = service.TaskErrorWrapperLocal(bodyErr, "read_request_body_failed", http.StatusBadRequest)
			}
			service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
				Phase:           service.RouteAttemptPhaseRequest,
				Outcome:         service.RouteAttemptOutcomeFailed,
				Reason:          "request_body_unavailable",
				StatusCode:      taskErr.StatusCode,
				ErrorType:       "task_error",
				ErrorCode:       taskErr.Code,
				ErrorMessage:    taskErr.Message,
				RetryDecision:   service.RouteRetryDecisionStop,
				RetryStopReason: "request_body_unavailable",
			})
			break
		}
		c.Request.Body = io.NopCloser(bodyStorage)
		service.RecordRouteUpstreamAttempt(c)
		addUsedChannel(c, channel.Id)
		upstreamStartedAt := time.Now()

		result, taskErr = relay.RelayTaskSubmit(c, relayInfo)
		if taskErr == nil {
			service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
				Phase:             service.RouteAttemptPhaseUpstream,
				Outcome:           service.RouteAttemptOutcomeSucceeded,
				DurationMS:        time.Since(upstreamStartedAt).Milliseconds(),
				RetryDecision:     service.RouteRetryDecisionComplete,
				RetryReason:       "success",
				UpstreamModel:     relayInfo.UpstreamModelName,
				RequestFormat:     string(relayInfo.GetFinalRequestRelayFormat()),
				UpstreamRequestID: c.GetString(common.UpstreamRequestIdKey),
			})
			service.SetRouteFinalStopReason(c, "success")
			break
		}

		retryAllowance := common.RetryTimes - attemptIndex
		if routePlan != nil && routePlan.IsExhaustive() {
			retryAllowance = 1
		}
		decision := getTaskRelayRetryDecision(c, taskErr, retryAllowance)
		if decision.Retry && routePlan != nil && !routePlan.HasNextAvailable() {
			decision = relayRetryDecision{Reason: "route_plan_exhausted"}
		}
		retryDecision := service.RouteRetryDecisionStop
		retryStopReason := decision.Reason
		if decision.Retry {
			retryDecision = service.RouteRetryDecisionRetry
			retryStopReason = ""
		}
		taskErrorMessage := taskErr.Message
		if taskErr.Error != nil {
			taskErrorMessage = taskErr.Error.Error()
		}
		service.RecordCurrentRouteAttemptResult(c, service.RouteAttemptResult{
			Phase:             service.RouteAttemptPhaseUpstream,
			Outcome:           service.RouteAttemptOutcomeFailed,
			StatusCode:        taskErr.StatusCode,
			ErrorType:         "task_error",
			ErrorCode:         taskErr.Code,
			ErrorMessage:      taskErrorMessage,
			DurationMS:        time.Since(upstreamStartedAt).Milliseconds(),
			RetryDecision:     retryDecision,
			RetryReason:       decision.Reason,
			RetryStopReason:   retryStopReason,
			UpstreamModel:     relayInfo.UpstreamModelName,
			RequestFormat:     string(relayInfo.GetFinalRequestRelayFormat()),
			UpstreamRequestID: c.GetString(common.UpstreamRequestIdKey),
		})

		if !taskErr.LocalError {
			processChannelError(c,
				*types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey,
					common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()),
				types.NewOpenAIError(taskErr.Error, types.ErrorCodeBadResponseStatusCode, taskErr.StatusCode),
				!decision.Retry)
		}

		if !decision.Retry {
			break
		}
	}

	useChannel := c.GetStringSlice("use_channel")
	if len(useChannel) > 1 {
		retryLogStr := fmt.Sprintf("重试：%s", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(useChannel)), "->"), "[]"))
		logger.LogInfo(c, retryLogStr)
	}
	perfmetrics.RecordRelaySampleAsync(c, relayInfo, taskErr == nil, 0)

	// ── 成功：结算 + 日志 + 插入任务 ──
	if taskErr == nil {
		if settleErr := service.SettleBilling(c, relayInfo, result.Quota); settleErr != nil {
			common.SysError("settle task billing error: " + settleErr.Error())
		}
		service.LogTaskConsumption(c, relayInfo)

		task := model.InitTask(result.Platform, relayInfo)
		task.PrivateData.UpstreamTaskID = result.UpstreamTaskID
		task.PrivateData.BillingSource = relayInfo.BillingSource
		task.PrivateData.SubscriptionId = relayInfo.SubscriptionId
		task.PrivateData.TokenId = relayInfo.TokenId
		task.PrivateData.NodeName = common.NodeName
		task.PrivateData.BillingContext = &model.TaskBillingContext{
			ModelPrice:      relayInfo.PriceData.ModelPrice,
			GroupRatio:      relayInfo.PriceData.GroupRatioInfo.GroupRatio,
			ModelRatio:      relayInfo.PriceData.ModelRatio,
			OtherRatios:     relayInfo.PriceData.OtherRatios(),
			OriginModelName: relayInfo.OriginModelName,
			PerCallBilling:  common.StringsContains(constant.TaskPricePatches, relayInfo.OriginModelName) || relayInfo.PriceData.UsePrice,
		}
		task.Quota = result.Quota
		task.Data = result.TaskData
		task.Action = relayInfo.Action
		if insertErr := task.Insert(); insertErr != nil {
			common.SysError("insert task error: " + insertErr.Error())
		}
	}

	if taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

// respondTaskError 统一输出 Task 错误响应（含 429 限流提示改写）
func respondTaskError(c *gin.Context, taskErr *dto.TaskError) {
	if taskErr.StatusCode == http.StatusTooManyRequests {
		taskErr.Message = "当前分组上游负载已饱和，请稍后再试"
	}
	c.JSON(taskErr.StatusCode, taskErr)
}

func shouldRetryTaskRelay(c *gin.Context, taskErr *dto.TaskError, retryTimes int) bool {
	return getTaskRelayRetryDecision(c, taskErr, retryTimes).Retry
}

func getTaskRelayRetryDecision(c *gin.Context, taskErr *dto.TaskError, retryTimes int) relayRetryDecision {
	if taskErr == nil {
		return relayRetryDecision{Reason: "no_error"}
	}
	if c != nil && c.Request != nil && c.Request.Context().Err() != nil {
		if errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			return relayRetryDecision{Reason: "client_context_deadline_exceeded"}
		}
		return relayRetryDecision{Reason: "client_context_canceled"}
	}
	if service.ShouldSkipRetryAfterChannelAffinityFailure(c) {
		return relayRetryDecision{Reason: "channel_affinity_retry_suppressed"}
	}
	if retryTimes <= 0 {
		return relayRetryDecision{Reason: "retry_limit_reached"}
	}
	if _, ok := c.Get("specific_channel_id"); ok {
		return relayRetryDecision{Reason: "specific_channel"}
	}
	if taskErr.LocalError {
		return relayRetryDecision{Reason: "local_task_error"}
	}
	if !taskErr.RetrySafe {
		return relayRetryDecision{Reason: "task_retry_not_safe"}
	}
	return relayRetryDecision{Retry: true, Reason: "retry_safe_task_error"}
}
