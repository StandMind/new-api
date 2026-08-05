package service

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/gin-gonic/gin"
)

var ErrRouteAttemptPlanExhausted = errors.New("route attempt plan exhausted")

const (
	RouteModeManual       = "manual"
	RouteModeFixedChannel = "fixed_channel"

	RouteAttemptPhaseSelection = "selection"
	RouteAttemptPhaseBilling   = "billing"
	RouteAttemptPhaseRequest   = "request"
	RouteAttemptPhaseUpstream  = "upstream"

	RouteAttemptOutcomeSkipped   = "skipped"
	RouteAttemptOutcomeFailed    = "failed"
	RouteAttemptOutcomeSucceeded = "succeeded"

	RouteRetryDecisionRetry    = "retry"
	RouteRetryDecisionStop     = "stop"
	RouteRetryDecisionComplete = "complete"

	routeDiagnosticMaxEvents       = 256
	routeDiagnosticMaxErrorBytes   = 2 * 1024
	routeDiagnosticMaxReasonBytes  = 128
	routeDiagnosticMaxModelBytes   = 512
	routeDiagnosticMaxRequestBytes = 512
)

type RouteAttempt struct {
	Group       string `json:"group"`
	Model       string `json:"model"`
	Priority    int64  `json:"priority"`
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name,omitempty"`
	Explicit    bool   `json:"explicit"`
}

type RouteAttemptResult struct {
	Phase             string
	Outcome           string
	Reason            string
	StatusCode        int
	ErrorType         string
	ErrorCode         string
	ErrorMessage      string
	DurationMS        int64
	RetryDecision     string
	RetryReason       string
	RetryStopReason   string
	UpstreamModel     string
	RequestFormat     string
	UpstreamRequestID string
}

type RouteAttemptDiagnostic struct {
	Sequence          int    `json:"sequence"`
	Group             string `json:"group"`
	Model             string `json:"model"`
	Priority          int64  `json:"priority"`
	ChannelID         int    `json:"channel_id"`
	ChannelName       string `json:"channel_name,omitempty"`
	Explicit          bool   `json:"explicit"`
	Phase             string `json:"phase"`
	Outcome           string `json:"outcome"`
	Reason            string `json:"reason,omitempty"`
	StatusCode        int    `json:"status_code,omitempty"`
	ErrorType         string `json:"error_type,omitempty"`
	ErrorCode         string `json:"error_code,omitempty"`
	ErrorMessage      string `json:"error_message,omitempty"`
	DurationMS        int64  `json:"duration_ms"`
	RetryDecision     string `json:"retry_decision,omitempty"`
	RetryReason       string `json:"retry_reason,omitempty"`
	RetryStopReason   string `json:"retry_stop_reason,omitempty"`
	UpstreamModel     string `json:"upstream_model,omitempty"`
	RequestFormat     string `json:"request_format,omitempty"`
	UpstreamRequestID string `json:"upstream_request_id,omitempty"`
}

type RoutingDiagnosticSnapshot struct {
	Mode                       string                   `json:"mode"`
	Basis                      string                   `json:"basis,omitempty"`
	Groups                     []string                 `json:"groups,omitempty"`
	Planned                    []RouteAttempt           `json:"planned,omitempty"`
	PlannedTruncated           bool                     `json:"planned_truncated,omitempty"`
	Attempts                   []RouteAttemptDiagnostic `json:"attempts,omitempty"`
	AttemptsTruncated          bool                     `json:"attempts_truncated,omitempty"`
	FinalGroup                 string                   `json:"final_group,omitempty"`
	FinalChannelID             int                      `json:"final_channel_id,omitempty"`
	FinalChannelName           string                   `json:"final_channel_name,omitempty"`
	FinalStopReason            string                   `json:"final_stop_reason,omitempty"`
	UpstreamAttempts           []RouteAttempt           `json:"upstream_attempted,omitempty"`
	UpstreamAttemptedTruncated bool                     `json:"upstream_attempted_truncated,omitempty"`
}

type RouteAttemptPlan struct {
	configuredGroups []string
	attempts         []RouteAttempt
	consumed         map[int]struct{}
	nextIndex        int
	currentIndex     int
	exhaustive       bool
	routingPriority  constant.RoutingPriority
	rankingBasis     string
	diagnostics      []RouteAttemptDiagnostic
	diagnosticsCut   bool
	finalStopReason  string
}

func NewRouteAttemptPlan(groups []string, exhaustive bool) *RouteAttemptPlan {
	uniqueGroups := make([]string, 0, len(groups))
	seenGroups := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if _, seen := seenGroups[group]; seen {
			continue
		}
		seenGroups[group] = struct{}{}
		uniqueGroups = append(uniqueGroups, group)
	}
	return &RouteAttemptPlan{
		configuredGroups: uniqueGroups,
		attempts:         make([]RouteAttempt, 0),
		consumed:         make(map[int]struct{}),
		currentIndex:     -1,
		exhaustive:       exhaustive,
		diagnostics:      make([]RouteAttemptDiagnostic, 0),
	}
}

func BuildRouteAttemptPlan(groups []string, modelName, requestPath string, exhaustive bool) (*RouteAttemptPlan, error) {
	plan := NewRouteAttemptPlan(groups, exhaustive)
	routePlans, err := model.LoadGroupModelRoutePlans(plan.configuredGroups, modelName, requestPath)
	if err != nil {
		return nil, err
	}
	for _, routePlan := range routePlans {
		group := routePlan.Group
		routeModel := routePlan.RouteModel
		route := routePlan.Route
		explicit := routePlan.Explicit
		candidates := routePlan.Candidates
		if len(candidates) == 0 {
			continue
		}

		available := make(map[int]model.GroupModelRouteCandidate, len(candidates))
		for _, candidate := range candidates {
			available[candidate.ChannelID] = candidate
		}

		tiers := make(model.GroupModelRouteTiers, 0)
		if explicit {
			for _, configuredTier := range route.Tiers {
				tier := model.GroupModelRouteTier{Priority: configuredTier.Priority}
				for _, configuredChannel := range configuredTier.Channels {
					if _, ok := available[configuredChannel.ChannelID]; !ok {
						continue
					}
					tier.Channels = append(tier.Channels, configuredChannel)
				}
				if len(tier.Channels) > 0 {
					tiers = append(tiers, tier)
				}
			}
		} else {
			tierIndexes := make(map[int64]int)
			for _, candidate := range candidates {
				tierIndex, ok := tierIndexes[candidate.Priority]
				if !ok {
					tierIndex = len(tiers)
					tierIndexes[candidate.Priority] = tierIndex
					tiers = append(tiers, model.GroupModelRouteTier{Priority: candidate.Priority})
				}
				tiers[tierIndex].Channels = append(tiers[tierIndex].Channels, model.GroupModelRouteChannel{
					ChannelID: candidate.ChannelID,
					Weight:    boundedRouteWeight(candidate.Weight),
				})
			}
		}

		sort.SliceStable(tiers, func(i, j int) bool {
			return tiers[i].Priority > tiers[j].Priority
		})
		for _, tier := range tiers {
			ordered := weightedRouteChannels(tier.Channels, rand.Int63n)
			for _, channel := range ordered {
				candidate := available[channel.ChannelID]
				plan.attempts = append(plan.attempts, RouteAttempt{
					Group:       group,
					Model:       routeModel,
					Priority:    tier.Priority,
					ChannelID:   channel.ChannelID,
					ChannelName: candidate.ChannelName,
					Explicit:    explicit,
				})
			}
		}
	}
	return plan, nil
}

func weightedRouteChannels(channels []model.GroupModelRouteChannel, randomInt63n func(int64) int64) []model.GroupModelRouteChannel {
	remaining := append([]model.GroupModelRouteChannel(nil), channels...)
	ordered := make([]model.GroupModelRouteChannel, 0, len(channels))
	for len(remaining) > 0 {
		totalWeight := int64(0)
		for _, channel := range remaining {
			totalWeight += normalizedRouteWeight(channel.Weight)
		}

		selectedIndex := 0
		if totalWeight == 0 {
			selectedIndex = int(randomInt63n(int64(len(remaining))))
		} else {
			target := randomInt63n(totalWeight)
			for index, channel := range remaining {
				target -= normalizedRouteWeight(channel.Weight)
				if target < 0 {
					selectedIndex = index
					break
				}
			}
		}
		ordered = append(ordered, remaining[selectedIndex])
		remaining = append(remaining[:selectedIndex], remaining[selectedIndex+1:]...)
	}
	return ordered
}

func boundedRouteWeight(weight uint) int64 {
	if weight > math.MaxInt32 {
		return math.MaxInt32
	}
	return int64(weight)
}

func normalizedRouteWeight(weight int64) int64 {
	if weight <= 0 {
		return 0
	}
	if weight > math.MaxInt32 {
		return math.MaxInt32
	}
	return weight
}

func (p *RouteAttemptPlan) IsExhaustive() bool {
	return p != nil && p.exhaustive
}

func (p *RouteAttemptPlan) SetSmartRouting(priority constant.RoutingPriority, basis string) {
	if p == nil {
		return
	}
	p.routingPriority = priority
	p.rankingBasis = basis
}

func (p *RouteAttemptPlan) RoutingPriority() constant.RoutingPriority {
	if p == nil {
		return constant.RoutingPriorityManual
	}
	return p.routingPriority
}

func (p *RouteAttemptPlan) RankingBasis() string {
	if p == nil {
		return ""
	}
	return p.rankingBasis
}

func (p *RouteAttemptPlan) Mode() string {
	if p == nil || p.routingPriority == constant.RoutingPriorityManual {
		return RouteModeManual
	}
	return string(p.routingPriority)
}

func (p *RouteAttemptPlan) ConfiguredGroups() []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.configuredGroups...)
}

func (p *RouteAttemptPlan) Attempts() []RouteAttempt {
	if p == nil {
		return nil
	}
	return append([]RouteAttempt(nil), p.attempts...)
}

func (p *RouteAttemptPlan) Next() (RouteAttempt, bool) {
	index, ok := p.nextAvailableIndex()
	if !ok {
		return RouteAttempt{}, false
	}
	attempt := p.attempts[index]
	p.consumed[index] = struct{}{}
	p.currentIndex = index
	p.nextIndex = index + 1
	return attempt, true
}

func (p *RouteAttemptPlan) NextAfterFailure() (RouteAttempt, bool) {
	if p == nil || p.currentIndex < 0 {
		return RouteAttempt{}, false
	}
	return p.Next()
}

func (p *RouteAttemptPlan) Current() (RouteAttempt, bool) {
	if p == nil || p.currentIndex < 0 || p.currentIndex >= len(p.attempts) {
		return RouteAttempt{}, false
	}
	return p.attempts[p.currentIndex], true
}

func (p *RouteAttemptPlan) HasNextAvailable() bool {
	_, ok := p.nextAvailableIndex()
	return ok
}

func (p *RouteAttemptPlan) TakePreferred(channelID int) (RouteAttempt, bool) {
	if p == nil || p.nextIndex != 0 {
		return RouteAttempt{}, false
	}
	for index, attempt := range p.attempts {
		if attempt.ChannelID != channelID {
			continue
		}
		p.consumed[index] = struct{}{}
		p.currentIndex = index
		return attempt, true
	}
	return RouteAttempt{}, false
}

func (p *RouteAttemptPlan) nextAvailableIndex() (int, bool) {
	if p == nil {
		return 0, false
	}
	for index := p.nextIndex; index < len(p.attempts); index++ {
		if _, used := p.consumed[index]; !used {
			return index, true
		}
	}
	return 0, false
}

func SetRouteAttemptPlan(c *gin.Context, plan *RouteAttemptPlan) {
	common.SetContextKey(c, constant.ContextKeyRouteAttemptPlan, plan)
}

func GetRouteAttemptPlan(c *gin.Context) *RouteAttemptPlan {
	plan, _ := common.GetContextKeyType[*RouteAttemptPlan](c, constant.ContextKeyRouteAttemptPlan)
	return plan
}

func RecordCurrentRouteAttemptResult(c *gin.Context, result RouteAttemptResult) {
	plan := GetRouteAttemptPlan(c)
	if plan == nil {
		return
	}
	attempt, ok := plan.Current()
	if !ok {
		return
	}
	plan.recordDiagnostic(attempt, result)
}

func SetRouteFinalStopReason(c *gin.Context, reason string) {
	if plan := GetRouteAttemptPlan(c); plan != nil {
		plan.SetFinalStopReason(reason)
	}
}

func EnsureRouteFinalStopReason(c *gin.Context, reason string) {
	if plan := GetRouteAttemptPlan(c); plan != nil && plan.FinalStopReason() == "" {
		plan.SetFinalStopReason(reason)
	}
}

func (p *RouteAttemptPlan) SetFinalStopReason(reason string) {
	if p == nil {
		return
	}
	p.finalStopReason = boundedRouteDiagnosticString(reason, routeDiagnosticMaxReasonBytes, false)
}

func (p *RouteAttemptPlan) FinalStopReason() string {
	if p == nil {
		return ""
	}
	return p.finalStopReason
}

func (p *RouteAttemptPlan) Diagnostics() ([]RouteAttemptDiagnostic, bool) {
	if p == nil {
		return nil, false
	}
	return append([]RouteAttemptDiagnostic(nil), p.diagnostics...), p.diagnosticsCut
}

func (p *RouteAttemptPlan) recordDiagnostic(attempt RouteAttempt, result RouteAttemptResult) {
	if p == nil {
		return
	}
	if len(p.diagnostics) >= routeDiagnosticMaxEvents {
		p.diagnosticsCut = true
		return
	}
	if result.DurationMS < 0 {
		result.DurationMS = 0
	}
	diagnostic := RouteAttemptDiagnostic{
		Sequence:          len(p.diagnostics) + 1,
		Group:             boundedRouteDiagnosticString(attempt.Group, routeDiagnosticMaxReasonBytes, false),
		Model:             boundedRouteDiagnosticString(attempt.Model, routeDiagnosticMaxModelBytes, false),
		Priority:          attempt.Priority,
		ChannelID:         attempt.ChannelID,
		ChannelName:       boundedRouteDiagnosticString(attempt.ChannelName, routeDiagnosticMaxModelBytes, false),
		Explicit:          attempt.Explicit,
		Phase:             boundedRouteDiagnosticString(result.Phase, routeDiagnosticMaxReasonBytes, false),
		Outcome:           boundedRouteDiagnosticString(result.Outcome, routeDiagnosticMaxReasonBytes, false),
		Reason:            boundedRouteDiagnosticString(result.Reason, routeDiagnosticMaxReasonBytes, false),
		StatusCode:        result.StatusCode,
		ErrorType:         boundedRouteDiagnosticString(result.ErrorType, routeDiagnosticMaxReasonBytes, false),
		ErrorCode:         boundedRouteDiagnosticString(result.ErrorCode, routeDiagnosticMaxReasonBytes, false),
		ErrorMessage:      boundedRouteDiagnosticString(result.ErrorMessage, routeDiagnosticMaxErrorBytes, true),
		DurationMS:        result.DurationMS,
		RetryDecision:     boundedRouteDiagnosticString(result.RetryDecision, routeDiagnosticMaxReasonBytes, false),
		RetryReason:       boundedRouteDiagnosticString(result.RetryReason, routeDiagnosticMaxReasonBytes, false),
		RetryStopReason:   boundedRouteDiagnosticString(result.RetryStopReason, routeDiagnosticMaxReasonBytes, false),
		UpstreamModel:     boundedRouteDiagnosticString(result.UpstreamModel, routeDiagnosticMaxModelBytes, false),
		RequestFormat:     boundedRouteDiagnosticString(result.RequestFormat, routeDiagnosticMaxRequestBytes, false),
		UpstreamRequestID: boundedRouteDiagnosticString(result.UpstreamRequestID, routeDiagnosticMaxRequestBytes, true),
	}
	p.diagnostics = append(p.diagnostics, diagnostic)
	if diagnostic.RetryStopReason != "" {
		p.finalStopReason = diagnostic.RetryStopReason
	}
}

func boundedRouteDiagnosticString(value string, maxBytes int, mask bool) string {
	if mask {
		value = common.MaskSensitiveInfo(value)
	}
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.RuneStart(value[maxBytes]) {
		maxBytes--
	}
	return value[:maxBytes]
}

func ApplyRouteAttempt(c *gin.Context, attempt RouteAttempt) (*model.Channel, error) {
	metricModel := common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
	if metricModel == "" {
		metricModel = attempt.Model
	}
	perfmetrics.BeginGroupAttempt(c, metricModel, attempt.Group)
	channel, ok := model.GetGroupModelRouteChannel(attempt.ChannelID)
	if !ok {
		return nil, fmt.Errorf("channel %d is unavailable", attempt.ChannelID)
	}
	if channel == nil || channel.Status != common.ChannelStatusEnabled {
		return nil, fmt.Errorf("channel %d is unavailable", attempt.ChannelID)
	}
	common.SetContextKey(c, constant.ContextKeyUsingGroup, attempt.Group)
	common.SetContextKey(c, constant.ContextKeyRouteAttempt, attempt)
	return channel, nil
}

func RecordRouteUpstreamAttempt(c *gin.Context) {
	attempt, ok := common.GetContextKeyType[RouteAttempt](c, constant.ContextKeyRouteAttempt)
	if !ok {
		return
	}
	history, _ := common.GetContextKeyType[[]RouteAttempt](c, constant.ContextKeyRouteAttemptHistory)
	history = append(history, attempt)
	common.SetContextKey(c, constant.ContextKeyRouteAttemptHistory, history)
	c.Set(common.UpstreamRequestIdKey, "")
}
