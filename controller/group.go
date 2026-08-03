package controller

import (
	"net/http"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetGroups(c *gin.Context) {
	groups, err := model.ListRouteGroups()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	groupNames := make([]string, 0, len(groups))
	routeGroupNames := make(map[string]string, len(groups))
	for _, group := range groups {
		if group.Enabled {
			groupNames = append(groupNames, group.Code)
			routeGroupNames[group.Code] = group.Name
		}
	}
	levels, err := model.ListUserLevels()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	type userLevelOption struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		IsDefault bool   `json:"is_default"`
	}
	userLevels := make([]userLevelOption, 0, len(levels))
	for _, level := range levels {
		if level.Enabled {
			userLevels = append(userLevels, userLevelOption{Code: level.Code, Name: level.Name, IsDefault: level.IsDefault})
		}
	}
	sort.Strings(groupNames)
	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"message":           "",
		"data":              groupNames,
		"route_group_names": routeGroupNames,
		"user_levels":       userLevels,
	})
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userGroup := ""
	userId := c.GetInt("id")
	userGroup, _ = model.GetUserLevel(userId, false)
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	for groupName, desc := range userUsableGroups {
		usableGroups[groupName] = map[string]interface{}{
			"code": groupName, "name": model.GetRouteGroupDisplayName(groupName),
			"ratio": service.GetUserGroupRatio(userGroup, groupName),
			"desc":  desc,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
