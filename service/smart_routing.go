package service

import (
	"errors"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const (
	smartRoutingRebuildInterval = 10 * time.Minute
	smartRoutingDebounce        = 500 * time.Millisecond
	smartRoutingMinSamples      = int64(3)
)

type smartRoutingKey struct {
	userGroup string
	model     string
	priority  constant.RoutingPriority
}

type smartRoutingRanking struct {
	groups []string
	basis  string
}

type smartRoutingSnapshot struct {
	createdAt       time.Time
	rankings        map[smartRoutingKey]smartRoutingRanking
	accessibleGroup map[string][]string
}

var (
	smartRoutingSnapshotValue atomic.Pointer[smartRoutingSnapshot]
	smartRoutingRebuildLock   sync.Mutex
	smartRoutingStartOnce     sync.Once
	smartRoutingRebuildSignal = make(chan struct{}, 1)
)

func InitSmartRouting() error {
	model.RegisterRoutingDataChangeHook(TriggerSmartRoutingRebuild)
	if err := model.InitGroupModelRouteIndex(); err != nil {
		return err
	}
	if err := RebuildSmartRoutingSnapshot(); err != nil {
		return err
	}
	smartRoutingStartOnce.Do(func() {
		go smartRoutingRebuildLoop()
	})
	return nil
}

func TriggerSmartRoutingRebuild() {
	select {
	case smartRoutingRebuildSignal <- struct{}{}:
	default:
	}
}

func smartRoutingRebuildLoop() {
	ticker := time.NewTicker(smartRoutingRebuildInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := RebuildSmartRoutingSnapshot(); err != nil {
				common.SysError("failed to rebuild smart routing snapshot: " + err.Error())
			}
		case <-smartRoutingRebuildSignal:
			timer := time.NewTimer(smartRoutingDebounce)
			for {
				select {
				case <-smartRoutingRebuildSignal:
					if !timer.Stop() {
						<-timer.C
					}
					timer.Reset(smartRoutingDebounce)
				case <-timer.C:
					if err := RebuildSmartRoutingSnapshot(); err != nil {
						common.SysError("failed to rebuild smart routing snapshot: " + err.Error())
					}
					goto nextEvent
				}
			}
		}
	nextEvent:
	}
}

func RebuildSmartRoutingSnapshot() error {
	smartRoutingRebuildLock.Lock()
	defer smartRoutingRebuildLock.Unlock()

	groupsByModel := model.GetRouteIndexGroupsByModel()
	if groupsByModel == nil {
		return errors.New("routing index is not initialized")
	}
	stats, err := perfmetrics.QueryRoutingStats(24)
	if err != nil {
		return err
	}

	userGroups := knownSmartRoutingUserGroups()
	allGroups := ratio_setting.GetGroupRatioCopy()
	snapshot := &smartRoutingSnapshot{
		createdAt:       time.Now(),
		rankings:        make(map[smartRoutingKey]smartRoutingRanking),
		accessibleGroup: make(map[string][]string, len(userGroups)),
	}
	for _, userGroup := range userGroups {
		usable := GetUserUsableGroups(userGroup)
		accessible := make([]string, 0, len(usable))
		for group := range usable {
			if _, ok := allGroups[group]; ok {
				accessible = append(accessible, group)
			}
		}
		sort.SliceStable(accessible, func(i, j int) bool {
			left, _ := ratio_setting.ResolveGroupRatio(userGroup, accessible[i], "")
			right, _ := ratio_setting.ResolveGroupRatio(userGroup, accessible[j], "")
			if left != right {
				return left < right
			}
			return accessible[i] < accessible[j]
		})
		snapshot.accessibleGroup[userGroup] = accessible

		accessibleSet := make(map[string]struct{}, len(accessible))
		for _, group := range accessible {
			accessibleSet[group] = struct{}{}
		}
		for modelName, routeGroups := range groupsByModel {
			groups := make([]string, 0, len(routeGroups))
			for _, group := range routeGroups {
				if _, ok := accessibleSet[group]; ok {
					groups = append(groups, group)
				}
			}
			priceOrder := priceOrderedGroups(userGroup, modelName, groups)
			snapshot.rankings[smartRoutingKey{userGroup: userGroup, model: modelName, priority: constant.RoutingPriorityPrice}] = smartRoutingRanking{
				groups: priceOrder, basis: "price",
			}
			for _, priority := range []constant.RoutingPriority{
				constant.RoutingPrioritySpeed,
				constant.RoutingPrioritySuccessRate,
				constant.RoutingPriorityAuto,
			} {
				ordered, basis := performanceOrderedGroups(priority, userGroup, modelName, priceOrder, stats[modelName])
				snapshot.rankings[smartRoutingKey{userGroup: userGroup, model: modelName, priority: priority}] = smartRoutingRanking{
					groups: ordered, basis: basis,
				}
			}
		}
	}

	smartRoutingSnapshotValue.Store(snapshot)
	return nil
}

func knownSmartRoutingUserGroups() []string {
	groups := map[string]struct{}{"": {}}
	for group := range ratio_setting.GetGroupRatioCopy() {
		groups[group] = struct{}{}
	}
	for group := range ratio_setting.GetGroupGroupRatioCopy() {
		groups[group] = struct{}{}
	}
	for group := range ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.ReadAll() {
		groups[group] = struct{}{}
	}
	result := make([]string, 0, len(groups))
	for group := range groups {
		result = append(result, group)
	}
	sort.Strings(result)
	return result
}

func priceOrderedGroups(userGroup, modelName string, groups []string) []string {
	ordered := append([]string(nil), groups...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left, _ := ratio_setting.ResolveGroupRatio(userGroup, ordered[i], modelName)
		right, _ := ratio_setting.ResolveGroupRatio(userGroup, ordered[j], modelName)
		if left != right {
			return left < right
		}
		return ordered[i] < ordered[j]
	})
	return ordered
}

func performanceOrderedGroups(
	priority constant.RoutingPriority,
	userGroup string,
	modelName string,
	priceOrder []string,
	stats map[string]perfmetrics.RoutingStat,
) ([]string, string) {
	sufficient := make([]string, 0, len(priceOrder))
	positions := make([]int, 0, len(priceOrder))
	for position, group := range priceOrder {
		stat := stats[group]
		valid := stat.RequestCount >= smartRoutingMinSamples
		if priority == constant.RoutingPrioritySpeed {
			valid = stat.SuccessCount >= smartRoutingMinSamples
		} else if priority == constant.RoutingPriorityAuto {
			valid = valid && stat.SuccessCount >= smartRoutingMinSamples
		}
		if valid {
			sufficient = append(sufficient, group)
			positions = append(positions, position)
		}
	}
	if len(sufficient) < 2 {
		return append([]string(nil), priceOrder...), "price_fallback"
	}

	switch priority {
	case constant.RoutingPrioritySpeed:
		sort.SliceStable(sufficient, func(i, j int) bool {
			left := float64(stats[sufficient[i]].SuccessLatencyMs) / float64(stats[sufficient[i]].SuccessCount)
			right := float64(stats[sufficient[j]].SuccessLatencyMs) / float64(stats[sufficient[j]].SuccessCount)
			return left < right
		})
	case constant.RoutingPrioritySuccessRate:
		sort.SliceStable(sufficient, func(i, j int) bool {
			left := float64(stats[sufficient[i]].SuccessCount) / float64(stats[sufficient[i]].RequestCount)
			right := float64(stats[sufficient[j]].SuccessCount) / float64(stats[sufficient[j]].RequestCount)
			return left > right
		})
	case constant.RoutingPriorityAuto:
		scores := autoRoutingScores(userGroup, modelName, sufficient, stats)
		sort.SliceStable(sufficient, func(i, j int) bool {
			return scores[sufficient[i]] > scores[sufficient[j]]
		})
	}

	ordered := append([]string(nil), priceOrder...)
	for index, position := range positions {
		ordered[position] = sufficient[index]
	}
	return ordered, string(priority)
}

func autoRoutingScores(userGroup, modelName string, groups []string, stats map[string]perfmetrics.RoutingStat) map[string]float64 {
	prices := make(map[string]float64, len(groups))
	speeds := make(map[string]float64, len(groups))
	minPrice, maxPrice := math.Inf(1), math.Inf(-1)
	minSpeed, maxSpeed := math.Inf(1), math.Inf(-1)
	for _, group := range groups {
		price, _ := ratio_setting.ResolveGroupRatio(userGroup, group, modelName)
		speed := float64(stats[group].SuccessLatencyMs) / float64(stats[group].SuccessCount)
		prices[group] = price
		speeds[group] = speed
		minPrice, maxPrice = math.Min(minPrice, price), math.Max(maxPrice, price)
		minSpeed, maxSpeed = math.Min(minSpeed, speed), math.Max(maxSpeed, speed)
	}
	scores := make(map[string]float64, len(groups))
	for _, group := range groups {
		stat := stats[group]
		successScore := float64(stat.SuccessCount) / float64(stat.RequestCount)
		priceScore := inverseNormalizedScore(prices[group], minPrice, maxPrice)
		speedScore := inverseNormalizedScore(speeds[group], minSpeed, maxSpeed)
		scores[group] = successScore*0.5 + priceScore*0.3 + speedScore*0.2
	}
	return scores
}

func inverseNormalizedScore(value, minimum, maximum float64) float64 {
	if maximum <= minimum {
		return 1
	}
	return 1 - (value-minimum)/(maximum-minimum)
}

func GetSmartRoutingGroups(userGroup, modelName, requestPath string, priority constant.RoutingPriority) ([]string, string) {
	snapshot := smartRoutingSnapshotValue.Load()
	if snapshot == nil || !constant.IsValidRoutingPriority(priority, false) {
		return nil, "unavailable"
	}
	ranking, ok := snapshot.rankings[smartRoutingKey{userGroup: userGroup, model: modelName, priority: priority}]
	if !ok {
		normalizedModel := ratio_setting.FormatMatchingModelName(modelName)
		ranking, ok = snapshot.rankings[smartRoutingKey{userGroup: userGroup, model: normalizedModel, priority: priority}]
	}
	if !ok {
		ranking, ok = snapshot.rankings[smartRoutingKey{userGroup: "", model: modelName, priority: priority}]
	}
	if !ok {
		return nil, "unavailable"
	}
	return model.FilterRouteGroupsForRequest(ranking.groups, modelName, requestPath), ranking.basis
}

func GetSmartRoutingAccessibleGroups(userGroup string) []string {
	snapshot := smartRoutingSnapshotValue.Load()
	if snapshot == nil {
		return nil
	}
	groups, ok := snapshot.accessibleGroup[userGroup]
	if !ok {
		groups = snapshot.accessibleGroup[""]
	}
	return append([]string(nil), groups...)
}
