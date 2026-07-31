package ratio_setting

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
)

var defaultGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

var groupRatioMap = types.NewRWMap[string, float64]()

var defaultGroupGroupRatio = map[string]map[string]float64{
	"vip": {
		"edit_this": 0.9,
	},
}

var groupGroupRatioMap = types.NewRWMap[string, map[string]float64]()

var groupModelRatioMap = types.NewRWMap[string, map[string]float64]()

var defaultGroupSpecialUsableGroup = map[string]map[string]string{}

type GroupRatioSetting struct {
	GroupRatio              *types.RWMap[string, float64]            `json:"group_ratio"`
	GroupGroupRatio         *types.RWMap[string, map[string]float64] `json:"group_group_ratio"`
	GroupModelRatio         *types.RWMap[string, map[string]float64] `json:"group_model_ratio"`
	GroupSpecialUsableGroup *types.RWMap[string, map[string]string]  `json:"group_special_usable_group"`
}

var groupRatioSetting GroupRatioSetting

func init() {
	groupSpecialUsableGroup := types.NewRWMap[string, map[string]string]()
	groupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)

	groupRatioMap.AddAll(defaultGroupRatio)
	groupGroupRatioMap.AddAll(defaultGroupGroupRatio)

	groupRatioSetting = GroupRatioSetting{
		GroupSpecialUsableGroup: groupSpecialUsableGroup,
		GroupRatio:              groupRatioMap,
		GroupGroupRatio:         groupGroupRatioMap,
		GroupModelRatio:         groupModelRatioMap,
	}

	config.GlobalConfig.Register("group_ratio_setting", &groupRatioSetting)
}

func GetGroupRatioSetting() *GroupRatioSetting {
	if groupRatioSetting.GroupSpecialUsableGroup == nil {
		groupRatioSetting.GroupSpecialUsableGroup = types.NewRWMap[string, map[string]string]()
		groupRatioSetting.GroupSpecialUsableGroup.AddAll(defaultGroupSpecialUsableGroup)
	}
	return &groupRatioSetting
}

func GetGroupRatioCopy() map[string]float64 {
	return groupRatioMap.ReadAll()
}

func ContainsGroupRatio(name string) bool {
	_, ok := groupRatioMap.Get(name)
	return ok
}

func GroupRatio2JSONString() string {
	return groupRatioMap.MarshalJSONString()
}

func UpdateGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupRatioMap, jsonStr)
}

func GetGroupRatio(name string) float64 {
	ratio, ok := groupRatioMap.Get(name)
	if !ok {
		common.SysLog("group ratio not found: " + name)
		return 1
	}
	return ratio
}

func GetGroupGroupRatio(userGroup, usingGroup string) (float64, bool) {
	gp, ok := groupGroupRatioMap.Get(userGroup)
	if !ok {
		return -1, false
	}
	ratio, ok := gp[usingGroup]
	if !ok {
		return -1, false
	}
	return ratio, true
}

func GroupGroupRatio2JSONString() string {
	return groupGroupRatioMap.MarshalJSONString()
}

func UpdateGroupGroupRatioByJSONString(jsonStr string) error {
	return types.LoadFromJsonString(groupGroupRatioMap, jsonStr)
}

func GetGroupModelRatioCopy() map[string]map[string]float64 {
	source := groupModelRatioMap.ReadAll()
	result := make(map[string]map[string]float64, len(source))
	for group, ratios := range source {
		result[group] = make(map[string]float64, len(ratios))
		for model, ratio := range ratios {
			result[group][model] = ratio
		}
	}
	return result
}

func GroupModelRatio2JSONString() string {
	return groupModelRatioMap.MarshalJSONString()
}

func UpdateGroupModelRatioByJSONString(jsonStr string) error {
	if err := CheckGroupModelRatio(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonString(groupModelRatioMap, jsonStr)
}

// ResolveGroupRatio applies model-specific overrides before the existing
// user-group and ordinary group ratios. Wildcards are trailing-* prefixes and
// the longest matching prefix wins.
func ResolveGroupRatio(userGroup, usingGroup, originalModel string) (float64, string) {
	if modelRatios, ok := groupModelRatioMap.Get(usingGroup); ok {
		if ratio, found := modelRatios[originalModel]; found {
			return ratio, "group_model_ratio.exact"
		}
		longestPrefix := ""
		wildcardRatio := 0.0
		wildcardFound := false
		for pattern, ratio := range modelRatios {
			if pattern != "*" && (len(pattern) < 2 || !strings.HasSuffix(pattern, "*")) {
				continue
			}
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(originalModel, prefix) &&
				(!wildcardFound || len(prefix) > len(longestPrefix)) {
				longestPrefix = prefix
				wildcardRatio = ratio
				wildcardFound = true
			}
		}
		if wildcardFound {
			return wildcardRatio, "group_model_ratio.prefix"
		}
	}
	if ratio, ok := GetGroupGroupRatio(userGroup, usingGroup); ok {
		return ratio, "group_group_ratio"
	}
	return GetGroupRatio(usingGroup), "group_ratio"
}

func CheckGroupModelRatio(jsonStr string) error {
	check := make(map[string]map[string]float64)
	if err := common.UnmarshalJsonStr(jsonStr, &check); err != nil {
		return err
	}
	for group, ratios := range check {
		if strings.TrimSpace(group) == "" {
			return errors.New("group model ratio contains an empty group")
		}
		for model, ratio := range ratios {
			if strings.TrimSpace(model) == "" {
				return fmt.Errorf("group model ratio contains an empty model in group %s", group)
			}
			if strings.Contains(strings.TrimSuffix(model, "*"), "*") {
				return fmt.Errorf("group model ratio wildcard must be a trailing *: %s/%s", group, model)
			}
			if ratio < 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
				return fmt.Errorf("group model ratio must be finite and not less than 0: %s/%s", group, model)
			}
		}
	}
	return nil
}

func CheckGroupRatio(jsonStr string) error {
	checkGroupRatio := make(map[string]float64)
	err := common.UnmarshalJsonStr(jsonStr, &checkGroupRatio)
	if err != nil {
		return err
	}
	for name, ratio := range checkGroupRatio {
		if ratio < 0 {
			return errors.New("group ratio must be not less than 0: " + name)
		}
	}
	return nil
}
