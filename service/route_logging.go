package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func AppendRoutingLogInfo(c *gin.Context, info *relaycommon.RelayInfo, other map[string]interface{}) {
	if c == nil || info == nil || other == nil {
		return
	}
	appendRoutingSafeInfo(c, info.UsingGroup, other)
	other["final_group"] = info.UsingGroup
	other["group_ratio_source"] = info.PriceData.GroupRatioInfo.Source
}

func AppendRoutingErrorLogInfo(c *gin.Context, other map[string]interface{}, finalFailure bool) {
	if c == nil || other == nil {
		return
	}
	finalGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	appendRoutingSafeInfo(c, finalGroup, other)
	other["final_group"] = finalGroup
	if !finalFailure {
		return
	}
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = make(map[string]interface{})
		other["admin_info"] = adminInfo
	}
	if snapshot := BuildRoutingDiagnosticSnapshot(c, finalGroup); snapshot != nil {
		adminInfo["routing"] = snapshot
	}
}

func BuildRoutingDiagnosticSnapshot(c *gin.Context, finalGroup string) *RoutingDiagnosticSnapshot {
	if c == nil {
		return nil
	}
	plan := GetRouteAttemptPlan(c)
	if plan == nil {
		if !isFixedChannelRoute(c) {
			return nil
		}
		return &RoutingDiagnosticSnapshot{
			Mode:             RouteModeFixedChannel,
			FinalGroup:       finalGroup,
			FinalChannelID:   common.GetContextKeyInt(c, constant.ContextKeyChannelId),
			FinalChannelName: common.GetContextKeyString(c, constant.ContextKeyChannelName),
		}
	}

	planned := plan.Attempts()
	plannedTruncated := false
	if len(planned) > routeDiagnosticMaxEvents {
		planned = append([]RouteAttempt(nil), planned[:routeDiagnosticMaxEvents]...)
		plannedTruncated = true
	}
	diagnostics, diagnosticsTruncated := plan.Diagnostics()
	history, _ := common.GetContextKeyType[[]RouteAttempt](c, constant.ContextKeyRouteAttemptHistory)
	historyTruncated := false
	if len(history) > routeDiagnosticMaxEvents {
		history = append([]RouteAttempt(nil), history[:routeDiagnosticMaxEvents]...)
		historyTruncated = true
	} else {
		history = append([]RouteAttempt(nil), history...)
	}
	return &RoutingDiagnosticSnapshot{
		Mode:                       plan.Mode(),
		Basis:                      plan.RankingBasis(),
		Groups:                     plan.ConfiguredGroups(),
		Planned:                    planned,
		PlannedTruncated:           plannedTruncated,
		Attempts:                   diagnostics,
		AttemptsTruncated:          diagnosticsTruncated,
		FinalGroup:                 finalGroup,
		FinalChannelID:             common.GetContextKeyInt(c, constant.ContextKeyChannelId),
		FinalChannelName:           common.GetContextKeyString(c, constant.ContextKeyChannelName),
		FinalStopReason:            plan.FinalStopReason(),
		UpstreamAttempts:           history,
		UpstreamAttemptedTruncated: historyTruncated,
	}
}

func appendRoutingSafeInfo(c *gin.Context, finalGroup string, other map[string]interface{}) {
	if c == nil || other == nil {
		return
	}
	plan := GetRouteAttemptPlan(c)
	if plan == nil {
		if isFixedChannelRoute(c) {
			other["routing_mode"] = RouteModeFixedChannel
		}
		return
	}

	history, _ := common.GetContextKeyType[[]RouteAttempt](c, constant.ContextKeyRouteAttemptHistory)
	attemptedGroups := make([]string, 0)
	for _, attempt := range history {
		if len(attemptedGroups) == 0 || attemptedGroups[len(attemptedGroups)-1] != attempt.Group {
			attemptedGroups = append(attemptedGroups, attempt.Group)
		}
	}
	other["routing_mode"] = plan.Mode()
	other["group_chain"] = plan.ConfiguredGroups()
	other["attempted_groups"] = attemptedGroups
	if finalGroup != "" {
		other["final_group"] = finalGroup
	}
	if priority := plan.RoutingPriority(); priority != constant.RoutingPriorityManual {
		other["routing_priority"] = string(priority)
		other["routing_basis"] = plan.RankingBasis()
	}
}

func isFixedChannelRoute(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if _, fixed := c.Get(string(constant.ContextKeyTokenSpecificChannelId)); fixed {
		return true
	}
	_, fixed := c.Get("specific_channel_id")
	return fixed
}
