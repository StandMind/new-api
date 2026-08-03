package service

import "github.com/QuantumNous/new-api/model"

func GetUserUsableGroups(userLevel string) map[string]string {
	groups := model.GetUserLevelRouteGroups(userLevel, false)
	result := make(map[string]string, len(groups))
	for _, group := range groups {
		result[group.Code] = group.Name
	}
	return result
}

func GroupInUserUsableGroups(userLevel, groupName string) bool {
	return model.UserLevelCanAccessRouteGroup(userLevel, groupName)
}

// GetUserGroupRatio 获取用户使用某个分组的倍率
// userGroup 用户分组
// group 需要获取倍率的分组
func GetUserGroupRatio(userLevel, group string) float64 {
	ratio, _ := model.ResolveAccessPolicyRatio(userLevel, group, "")
	return ratio
}
