package openluxsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
)

const maxPricingResponseBytes = 8 << 20

type stepRatio struct {
	StepSize                    decimal.Decimal `json:"step_size"`
	CompletionStepSize          int64           `json:"completion_step_size"`
	PromptStepRatio             decimal.Decimal `json:"prompt_step_ratio"`
	CompletionStepRatio         decimal.Decimal `json:"completion_step_ratio"`
	CacheStepRatio              decimal.Decimal `json:"cache_step_ratio"`
	PromptThinkingStepRatio     decimal.Decimal `json:"prompt_thinking_step_ratio"`
	CompletionThinkingStepRatio decimal.Decimal `json:"completion_thinking_step_ratio"`
}

type sourceModel struct {
	Name                 string           `json:"model_name"`
	QuotaType            int              `json:"quota_type"`
	ModelRatio           decimal.Decimal  `json:"model_ratio"`
	ModelPrice           decimal.Decimal  `json:"model_price"`
	CompletionRatio      *decimal.Decimal `json:"completion_ratio"`
	CacheRatio           *decimal.Decimal `json:"cache_ratio"`
	CacheCreationRatio   *decimal.Decimal `json:"cache_creation_ratio"`
	CacheCreation5mRatio *decimal.Decimal `json:"cache_creation_5m_ratio"`
	CacheCreation1hRatio *decimal.Decimal `json:"cache_creation_1h_ratio"`
	ImageRatio           *decimal.Decimal `json:"image_ratio"`
	AudioRatio           *decimal.Decimal `json:"audio_ratio"`
	AudioCompletionRatio *decimal.Decimal `json:"audio_completion_ratio"`
	EnableGroups         []string         `json:"enable_groups"`
	StepRatios           []stepRatio      `json:"step_ratios"`
}

type pricingResponse struct {
	Success         bool                                  `json:"success"`
	Message         string                                `json:"message"`
	Data            []sourceModel                         `json:"data"`
	GroupRatio      map[string]decimal.Decimal            `json:"group_ratio"`
	GroupModelRatio map[string]map[string]decimal.Decimal `json:"group_model_ratio"`
	UsableGroup     map[string]json.RawMessage            `json:"usable_group"`
}

type sourceGroup struct {
	Name        string
	Description string
	Ratio       decimal.Decimal
}

type sourceSnapshot struct {
	Models           map[string]sourceModel
	OrderedModels    []sourceModel
	Groups           map[string]sourceGroup
	ReferencedGroups map[string]struct{}
	Hash             string
	FetchedAt        int64
}

type sourceHashView struct {
	Models []sourceModel         `json:"models"`
	Groups []sourceGroupHashView `json:"groups"`
}

type sourceGroupHashView struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Ratio       decimal.Decimal `json:"ratio"`
}

var pricingHTTPClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		if request.URL.Scheme != "https" || !strings.EqualFold(request.URL.Hostname(), "api.openlux.ai") {
			return errors.New("OpenLux redirected outside the configured endpoint")
		}
		return nil
	},
}

func fetchSource(ctx context.Context) (*sourceSnapshot, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, Endpoint, nil)
	if err != nil {
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_request", Message: "无法创建 OpenLux 定价请求", Err: err}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Aivrae-OpenLux-Price-Sync/1.0")

	response, err := pricingHTTPClient.Do(request)
	if err != nil {
		status := http.StatusBadGateway
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			status = http.StatusGatewayTimeout
		}
		return nil, &ServiceError{Status: status, Code: "upstream_unavailable", Message: "无法获取 OpenLux 定价", Err: err}
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, &ServiceError{
			Status:  http.StatusBadGateway,
			Code:    "upstream_status",
			Message: fmt.Sprintf("OpenLux 定价接口返回 HTTP %d", response.StatusCode),
		}
	}

	limited := io.LimitReader(response.Body, maxPricingResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_read", Message: "读取 OpenLux 定价失败", Err: err}
	}
	if len(body) > maxPricingResponseBytes {
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_too_large", Message: "OpenLux 定价响应超过大小限制"}
	}
	return parseSource(body)
}

func parseSource(body []byte) (*sourceSnapshot, error) {
	var raw pricingResponse
	if err := common.Unmarshal(body, &raw); err != nil {
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_json", Message: "OpenLux 定价响应不是有效 JSON", Err: err}
	}
	if !raw.Success {
		message := strings.TrimSpace(raw.Message)
		if message == "" {
			message = "OpenLux 定价接口返回失败"
		}
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_failure", Message: message}
	}
	if len(raw.Data) == 0 || len(raw.GroupRatio) == 0 || len(raw.UsableGroup) == 0 {
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_schema", Message: "OpenLux 定价响应缺少模型或分组数据"}
	}
	if len(raw.GroupModelRatio) > 0 {
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "unsupported_upstream_schema", Message: "OpenLux 已启用模型级来源组倍率，当前同步器无法无损处理"}
	}

	source := &sourceSnapshot{
		Models:           make(map[string]sourceModel, len(raw.Data)),
		Groups:           make(map[string]sourceGroup, len(raw.GroupRatio)),
		ReferencedGroups: make(map[string]struct{}),
		FetchedAt:        time.Now().Unix(),
	}
	for name, ratio := range raw.GroupRatio {
		name = strings.TrimSpace(name)
		if name == "" || ratio.IsNegative() {
			return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_schema", Message: "OpenLux 包含无效分组倍率"}
		}
		descriptionRaw, ok := raw.UsableGroup[name]
		if !ok {
			continue
		}
		description := name
		if value := strings.TrimSpace(common.JsonRawMessageToString(descriptionRaw)); value != "" {
			description = value
		}
		source.Groups[name] = sourceGroup{Name: name, Description: description, Ratio: ratio}
	}
	if len(source.Groups) == 0 {
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_schema", Message: "OpenLux 没有完整注册的来源组"}
	}

	for _, item := range raw.Data {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_schema", Message: "OpenLux 包含空模型名称"}
		}
		if _, exists := source.Models[item.Name]; exists {
			return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_schema", Message: "OpenLux 包含重复模型 " + item.Name}
		}
		if item.QuotaType != 0 && item.QuotaType != 1 {
			return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_schema", Message: "OpenLux 模型 " + item.Name + " 使用未知计费类型"}
		}
		if item.ModelRatio.IsNegative() || item.ModelPrice.IsNegative() {
			return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_schema", Message: "OpenLux 模型 " + item.Name + " 包含负价格"}
		}
		for field, value := range map[string]*decimal.Decimal{
			"completion_ratio":        item.CompletionRatio,
			"cache_ratio":             item.CacheRatio,
			"cache_creation_ratio":    item.CacheCreationRatio,
			"cache_creation_5m_ratio": item.CacheCreation5mRatio,
			"cache_creation_1h_ratio": item.CacheCreation1hRatio,
			"image_ratio":             item.ImageRatio,
			"audio_ratio":             item.AudioRatio,
			"audio_completion_ratio":  item.AudioCompletionRatio,
		} {
			if value != nil && value.IsNegative() {
				return nil, &ServiceError{
					Status: http.StatusBadGateway, Code: "upstream_schema",
					Message: "OpenLux 模型 " + item.Name + " 包含负倍率 " + field,
				}
			}
		}
		seenGroups := make(map[string]struct{}, len(item.EnableGroups))
		cleanGroups := make([]string, 0, len(item.EnableGroups))
		for _, group := range item.EnableGroups {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}
			if _, exists := seenGroups[group]; exists {
				return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_schema", Message: "OpenLux 模型 " + item.Name + " 包含重复来源组 " + group}
			}
			seenGroups[group] = struct{}{}
			source.ReferencedGroups[group] = struct{}{}
			cleanGroups = append(cleanGroups, group)
		}
		sort.Strings(cleanGroups)
		item.EnableGroups = cleanGroups
		source.Models[item.Name] = item
		source.OrderedModels = append(source.OrderedModels, item)
	}
	sort.Slice(source.OrderedModels, func(i, j int) bool {
		return source.OrderedModels[i].Name < source.OrderedModels[j].Name
	})

	groupViews := make([]sourceGroupHashView, 0, len(source.Groups))
	for _, group := range source.Groups {
		groupViews = append(groupViews, sourceGroupHashView{Name: group.Name, Description: group.Description, Ratio: group.Ratio})
	}
	sort.Slice(groupViews, func(i, j int) bool { return groupViews[i].Name < groupViews[j].Name })
	hashInput := sourceHashView{Models: source.OrderedModels, Groups: groupViews}
	encoded, err := common.Marshal(hashInput)
	if err != nil {
		return nil, &ServiceError{Status: http.StatusBadGateway, Code: "upstream_hash", Message: "无法规范化 OpenLux 定价", Err: err}
	}
	digest := sha256.Sum256(encoded)
	source.Hash = hex.EncodeToString(digest[:])
	return source, nil
}

func isOpenLuxBaseURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" && strings.EqualFold(parsed.Hostname(), "api.openlux.ai") &&
		(parsed.Port() == "" || parsed.Port() == "443") && (parsed.Path == "" || parsed.Path == "/") &&
		parsed.RawQuery == "" && parsed.Fragment == "" && parsed.User == nil
}

func decimalValue(value *decimal.Decimal, fallback decimal.Decimal) decimal.Decimal {
	if value == nil {
		return fallback
	}
	return *value
}

func cacheCreationRatios(item sourceModel) (fiveMinutes, oneHour decimal.Decimal, fiveConfigured, oneHourConfigured bool, err error) {
	legacyConfigured := item.CacheCreationRatio != nil
	fiveConfigured = item.CacheCreation5mRatio != nil || legacyConfigured
	oneHourConfigured = item.CacheCreation1hRatio != nil
	legacy := decimalValue(item.CacheCreationRatio, decimal.Zero)
	fiveMinutes = decimalValue(item.CacheCreation5mRatio, decimal.Zero)
	oneHour = decimalValue(item.CacheCreation1hRatio, decimal.Zero)
	if legacy.IsNegative() || fiveMinutes.IsNegative() || oneHour.IsNegative() {
		return decimal.Zero, decimal.Zero, false, false, fmt.Errorf("缓存写入倍率不能为负数")
	}
	if legacyConfigured && item.CacheCreation5mRatio != nil && !legacy.Equal(fiveMinutes) {
		return decimal.Zero, decimal.Zero, false, false, fmt.Errorf("cache_creation_ratio 与 cache_creation_5m_ratio 不一致")
	}
	if item.CacheCreation5mRatio == nil {
		fiveMinutes = legacy
	}
	return fiveMinutes, oneHour, fiveConfigured, oneHourConfigured, nil
}
