package constant

import "strings"

type RoutingPriority string

const (
	RoutingPriorityManual      RoutingPriority = ""
	RoutingPriorityAuto        RoutingPriority = "auto"
	RoutingPriorityPrice       RoutingPriority = "price"
	RoutingPrioritySpeed       RoutingPriority = "speed"
	RoutingPrioritySuccessRate RoutingPriority = "success_rate"
)

var SmartRoutingPriorities = []RoutingPriority{
	RoutingPriorityAuto,
	RoutingPriorityPrice,
	RoutingPrioritySpeed,
	RoutingPrioritySuccessRate,
}

func NormalizeRoutingPriority(value string) RoutingPriority {
	return RoutingPriority(strings.ToLower(strings.TrimSpace(value)))
}

func IsValidRoutingPriority(priority RoutingPriority, allowManual bool) bool {
	if priority == RoutingPriorityManual {
		return allowManual
	}
	for _, candidate := range SmartRoutingPriorities {
		if priority == candidate {
			return true
		}
	}
	return false
}
