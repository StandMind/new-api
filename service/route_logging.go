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
	plan := GetRouteAttemptPlan(c)
	if plan == nil {
		return
	}

	history, _ := common.GetContextKeyType[[]RouteAttempt](c, constant.ContextKeyRouteAttemptHistory)
	attemptedGroups := make([]string, 0)
	for _, attempt := range history {
		if len(attemptedGroups) == 0 || attemptedGroups[len(attemptedGroups)-1] != attempt.Group {
			attemptedGroups = append(attemptedGroups, attempt.Group)
		}
	}
	other["group_chain"] = plan.ConfiguredGroups()
	other["attempted_groups"] = attemptedGroups
	other["final_group"] = info.UsingGroup
	other["group_ratio_source"] = info.PriceData.GroupRatioInfo.Source
	if priority := plan.RoutingPriority(); priority != constant.RoutingPriorityManual {
		other["routing_priority"] = string(priority)
		other["routing_basis"] = plan.RankingBasis()
	}

	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = make(map[string]interface{})
		other["admin_info"] = adminInfo
	}
	adminInfo["routing"] = map[string]interface{}{
		"mode":      string(plan.RoutingPriority()),
		"basis":     plan.RankingBasis(),
		"planned":   plan.Attempts(),
		"attempted": history,
	}
}

func AppendRoutingAdminInfo(c *gin.Context, adminInfo map[string]interface{}) {
	if c == nil || adminInfo == nil {
		return
	}
	plan := GetRouteAttemptPlan(c)
	if plan == nil {
		return
	}
	history, _ := common.GetContextKeyType[[]RouteAttempt](c, constant.ContextKeyRouteAttemptHistory)
	adminInfo["routing"] = map[string]interface{}{
		"mode":      string(plan.RoutingPriority()),
		"basis":     plan.RankingBasis(),
		"groups":    plan.ConfiguredGroups(),
		"planned":   plan.Attempts(),
		"attempted": history,
	}
}
