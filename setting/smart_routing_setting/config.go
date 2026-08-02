package smart_routing_setting

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/config"
)

const ConfigName = "smart_routing_setting"

type SmartRoutingSetting struct {
	DefaultPriority string `json:"default_priority"`
}

var smartRoutingSetting = SmartRoutingSetting{
	DefaultPriority: string(constant.RoutingPriorityPrice),
}

func init() {
	config.GlobalConfig.Register(ConfigName, &smartRoutingSetting)
}

func GetDefaultPriority() constant.RoutingPriority {
	priority := constant.NormalizeRoutingPriority(smartRoutingSetting.DefaultPriority)
	if !constant.IsValidRoutingPriority(priority, false) {
		return constant.RoutingPriorityPrice
	}
	return priority
}

func ValidateDefaultPriority(value string) bool {
	return constant.IsValidRoutingPriority(constant.NormalizeRoutingPriority(value), false)
}
