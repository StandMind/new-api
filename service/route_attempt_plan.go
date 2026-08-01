package service

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

var ErrRouteAttemptPlanExhausted = errors.New("route attempt plan exhausted")

type RouteAttempt struct {
	Group     string `json:"group"`
	Model     string `json:"model"`
	Priority  int64  `json:"priority"`
	ChannelID int    `json:"channel_id"`
	Explicit  bool   `json:"explicit"`
}

type RouteAttemptPlan struct {
	configuredGroups []string
	attempts         []RouteAttempt
	consumed         map[int]struct{}
	nextIndex        int
	currentIndex     int
	exhaustive       bool
}

func BuildRouteAttemptPlan(groups []string, modelName, requestPath string, exhaustive bool) (*RouteAttemptPlan, error) {
	uniqueGroups := make([]string, 0, len(groups))
	seenGroups := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if _, seen := seenGroups[group]; seen {
			continue
		}
		seenGroups[group] = struct{}{}
		uniqueGroups = append(uniqueGroups, group)
	}
	plan := &RouteAttemptPlan{
		configuredGroups: uniqueGroups,
		attempts:         make([]RouteAttempt, 0),
		consumed:         make(map[int]struct{}),
		currentIndex:     -1,
		exhaustive:       exhaustive,
	}
	routePlans, err := model.LoadGroupModelRoutePlans(uniqueGroups, modelName, requestPath)
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
				plan.attempts = append(plan.attempts, RouteAttempt{
					Group:     group,
					Model:     routeModel,
					Priority:  tier.Priority,
					ChannelID: channel.ChannelID,
					Explicit:  explicit,
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

func ApplyRouteAttempt(c *gin.Context, attempt RouteAttempt) (*model.Channel, error) {
	channel, err := model.CacheGetChannel(attempt.ChannelID)
	if err != nil {
		return nil, err
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
}
