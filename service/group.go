package service

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func GetUserUsableGroups(userLevel string) map[string]string {
	if !model.AccessPolicySnapshotReady() {
		legacyLevel := model.LegacyGroupForUserLevel(userLevel)
		groupsCopy := setting.GetUserUsableGroupsCopy()
		if specialSettings, ok := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Get(legacyLevel); ok {
			for specialGroup, desc := range specialSettings {
				switch {
				case strings.HasPrefix(specialGroup, "-:"):
					delete(groupsCopy, strings.TrimPrefix(specialGroup, "-:"))
				case strings.HasPrefix(specialGroup, "+:"):
					groupsCopy[strings.TrimPrefix(specialGroup, "+:")] = desc
				default:
					groupsCopy[specialGroup] = desc
				}
			}
		}
		if legacyLevel != "" {
			if _, ok := groupsCopy[legacyLevel]; !ok {
				groupsCopy[legacyLevel] = "用户分组"
			}
		}
		return groupsCopy
	}
	groups := model.GetUserLevelRouteGroups(userLevel, false)
	result := make(map[string]string, len(groups))
	for _, group := range groups {
		result[group.Code] = group.Name
	}
	return result
}

func GroupInUserUsableGroups(userLevel, groupName string) bool {
	if !model.AccessPolicySnapshotReady() {
		_, ok := GetUserUsableGroups(userLevel)[groupName]
		return ok
	}
	return model.UserLevelCanAccessRouteGroup(userLevel, groupName)
}

// GetUserGroupRatio 获取用户使用某个分组的倍率
// userGroup 用户分组
// group 需要获取倍率的分组
func GetUserGroupRatio(userLevel, group string) float64 {
	ratio, _ := model.ResolveAccessPolicyRatio(userLevel, group, "")
	return ratio
}
