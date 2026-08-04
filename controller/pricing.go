package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/official_price_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func filterPricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string) []model.Pricing {
	if len(pricing) == 0 {
		return pricing
	}
	if len(usableGroup) == 0 {
		return []model.Pricing{}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if common.StringsContains(item.EnableGroup, "all") {
			item.EnableGroup = []string{"all"}
			filtered = append(filtered, item)
			continue
		}
		availableGroups := make([]string, 0, len(item.EnableGroup))
		for _, group := range item.EnableGroup {
			if _, ok := usableGroup[group]; ok {
				availableGroups = append(availableGroups, group)
			}
		}
		if len(availableGroups) == 0 {
			continue
		}
		item.EnableGroup = availableGroups
		filtered = append(filtered, item)
	}
	return filtered
}

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	vendors := model.GetVendors()
	userId, exists := c.Get("id")
	usableGroup := map[string]string{}
	groupRatio := map[string]float64{}
	group := model.GetDefaultUserLevelFromSnapshot()
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil {
			group = user.UserLevel
		}
	}

	usableGroup = service.GetUserUsableGroups(group)
	type usableRouteGroupView struct {
		Code        string  `json:"code"`
		Name        string  `json:"name"`
		Description string  `json:"desc"`
		Ratio       float64 `json:"ratio"`
	}
	usableGroupView := make(map[string]usableRouteGroupView, len(usableGroup))
	for routeGroup := range usableGroup {
		ratio, _ := model.ResolveAccessPolicyRatio(group, routeGroup, "")
		groupRatio[routeGroup] = ratio
		name := model.GetRouteGroupDisplayName(routeGroup)
		description := usableGroup[routeGroup]
		if route, ok := model.GetRouteGroupFromSnapshot(routeGroup); ok {
			if route.Description != "" {
				description = route.Description
			}
			name = route.Name
		}
		usableGroupView[routeGroup] = usableRouteGroupView{Code: routeGroup, Name: name, Description: description, Ratio: ratio}
	}
	pricing = filterPricingByUsableGroups(pricing, usableGroup)
	locale := model.ResolveLocalizedTextLocale(c.Query("lang"), c.GetHeader("Accept-Language"))
	pricing, vendors = model.LocalizePricingData(pricing, vendors, locale)
	// check groupRatio contains usableGroup
	for group := range groupRatio {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}
	rawGroupModelRatio := ratio_setting.GetGroupModelRatioCopy()
	for ratioGroup := range rawGroupModelRatio {
		if _, ok := usableGroup[ratioGroup]; !ok {
			delete(rawGroupModelRatio, ratioGroup)
		}
	}
	effectiveGroupModelRatio := make(map[string]map[string]float64, len(pricing))
	for index := range pricing {
		effective := make(map[string]float64)
		pricing[index].RouteGroupNames = make(map[string]string)
		for usableGroupName := range usableGroup {
			if !common.StringsContains(pricing[index].EnableGroup, "all") &&
				!common.StringsContains(pricing[index].EnableGroup, usableGroupName) {
				continue
			}
			ratio, _ := model.ResolveAccessPolicyRatio(group, usableGroupName, pricing[index].ModelName)
			effective[usableGroupName] = ratio
			pricing[index].RouteGroupNames[usableGroupName] = model.GetRouteGroupDisplayName(usableGroupName)
		}
		pricing[index].EffectiveGroupRatio = effective
		if officialPrice, ok := official_price_setting.GetModelPrice(pricing[index].ModelName); ok {
			pricing[index].OfficialPrice = officialPrice
		}
		effectiveGroupModelRatio[pricing[index].ModelName] = effective
	}

	c.JSON(200, gin.H{
		"success":                     true,
		"data":                        pricing,
		"vendors":                     vendors,
		"group_ratio":                 groupRatio,
		"group_model_ratio":           rawGroupModelRatio,
		"effective_group_model_ratio": effectiveGroupModelRatio,
		"usable_group":                usableGroupView,
		"supported_endpoint":          model.GetSupportedEndpointMap(),
		"pricing_version":             "c8fcbfc881ac3680c5019bb680fb4b3d",
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "重置模型倍率成功",
	})
}
