package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func ListGroupModelRoutes(c *gin.Context) {
	routes, err := model.ListGroupModelRoutes()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, routes)
}

func GetGroupModelRoute(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	modelName := strings.TrimSpace(c.Query("model"))
	if group == "" || modelName == "" {
		common.ApiError(c, &routeValidationError{message: "group and model are required"})
		return
	}

	route, explicit, err := model.GetGroupModelRoute(group, modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	candidates, err := model.GetGroupModelRouteCandidates(group, modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if route == nil {
		route = &model.GroupModelRoute{
			Group: group,
			Model: modelName,
			Tiers: inheritedGroupModelRouteTiers(candidates),
		}
	}
	common.ApiSuccess(c, gin.H{
		"explicit":   explicit,
		"route":      route,
		"candidates": candidates,
	})
}

func PutGroupModelRoute(c *gin.Context) {
	var route model.GroupModelRoute
	if err := c.ShouldBindJSON(&route); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveGroupModelRoute(&route); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "group_model_route.update", map[string]interface{}{
		"group":      route.Group,
		"model":      route.Model,
		"tier_count": len(route.Tiers),
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "data": route})
}

func DeleteGroupModelRoute(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	modelName := strings.TrimSpace(c.Query("model"))
	if group == "" || modelName == "" {
		common.ApiError(c, &routeValidationError{message: "group and model are required"})
		return
	}
	deleted, err := model.DeleteGroupModelRoute(group, modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "group_model_route.delete", map[string]interface{}{
		"group": group,
		"model": modelName,
	})
	common.ApiSuccess(c, gin.H{"deleted": deleted})
}

func inheritedGroupModelRouteTiers(candidates []model.GroupModelRouteCandidate) model.GroupModelRouteTiers {
	tiers := make(model.GroupModelRouteTiers, 0)
	tierIndexes := make(map[int64]int)
	for _, candidate := range candidates {
		index, ok := tierIndexes[candidate.Priority]
		if !ok {
			index = len(tiers)
			tierIndexes[candidate.Priority] = index
			tiers = append(tiers, model.GroupModelRouteTier{Priority: candidate.Priority})
		}
		tiers[index].Channels = append(tiers[index].Channels, model.GroupModelRouteChannel{
			ChannelID: candidate.ChannelID,
			Weight:    int64(candidate.Weight),
		})
	}
	return tiers
}

type routeValidationError struct {
	message string
}

func (e *routeValidationError) Error() string {
	return e.message
}
