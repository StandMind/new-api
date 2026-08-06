package controller

import (
	"errors"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var accessPolicyCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._() -]{0,63}$`)

var (
	errUserLevelReferenced  = errors.New("user level is referenced")
	errRouteGroupReferenced = errors.New("route group is referenced")
)

type userLevelRequest struct {
	Name                string                  `json:"name"`
	Description         string                  `json:"description"`
	IsDefault           bool                    `json:"is_default"`
	Enabled             bool                    `json:"enabled"`
	TopupRatio          model.AccessPolicyRatio `json:"topup_ratio"`
	RequestLimit        int                     `json:"request_limit"`
	SuccessRequestLimit int                     `json:"success_request_limit"`
}

type createUserLevelRequest struct {
	Code string `json:"code"`
	userLevelRequest
}

type routeGroupRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	BaseRatio   model.AccessPolicyRatio `json:"base_ratio"`
	Enabled     bool                    `json:"enabled"`
}

type createRouteGroupRequest struct {
	Code string `json:"code"`
	routeGroupRequest
}

type routeGroupGrantRequest struct {
	Code       string                   `json:"code"`
	PriceRatio *model.AccessPolicyRatio `json:"price_ratio"`
}

type replaceRouteGroupGrantsRequest struct {
	RouteGroups []routeGroupGrantRequest `json:"route_groups"`
}

func validateAccessPolicyCode(code string) error {
	if code != strings.TrimSpace(code) || code == "auto" || !accessPolicyCodePattern.MatchString(code) {
		return common.NewPublicError(i18n.MsgAccessPolicyCodeInvalid, nil)
	}
	return nil
}

func validateUserLevelRequest(req userLevelRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return common.NewPublicError(i18n.MsgAccessPolicyDisplayNameRequired, nil)
	}
	if req.TopupRatio <= 0 || math.IsNaN(float64(req.TopupRatio)) || math.IsInf(float64(req.TopupRatio), 0) {
		return common.NewPublicError(i18n.MsgAccessPolicyTopupRatioInvalid, nil)
	}
	if req.RequestLimit < 0 || req.SuccessRequestLimit < 0 {
		return common.NewPublicError(i18n.MsgAccessPolicyRateLimitInvalid, nil)
	}
	return nil
}

func validateRouteGroupRequest(req routeGroupRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return common.NewPublicError(i18n.MsgAccessPolicyDisplayNameRequired, nil)
	}
	if req.BaseRatio < 0 || math.IsNaN(float64(req.BaseRatio)) || math.IsInf(float64(req.BaseRatio), 0) {
		return common.NewPublicError(i18n.MsgAccessPolicyBaseRatioInvalid, nil)
	}
	return nil
}

func finishAccessPolicyMutation(c *gin.Context, value interface{}) {
	if err := model.RefreshAccessPolicyCaches(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, value)
}

func ListUserLevels(c *gin.Context) {
	levels, err := model.ListUserLevels()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	type view struct {
		model.UserLevel
		References []model.AccessPolicyReference `json:"references"`
	}
	result := make([]view, 0, len(levels))
	for _, level := range levels {
		refs, refErr := model.UserLevelReferences(level.Code)
		if refErr != nil {
			common.ApiError(c, refErr)
			return
		}
		result = append(result, view{UserLevel: level, References: refs})
	}
	common.ApiSuccess(c, result)
}

func CreateUserLevel(c *gin.Context) {
	var req createUserLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := validateAccessPolicyCode(req.Code); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Code == model.LegacyDefaultGroupCode {
		common.ApiErrorI18n(c, i18n.MsgAccessPolicyReservedUserLevel)
		return
	}
	if err := validateUserLevelRequest(req.userLevelRequest); err != nil {
		common.ApiError(c, err)
		return
	}
	level := model.UserLevel{Code: req.Code, Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), IsDefault: req.IsDefault, Enabled: req.Enabled, TopupRatio: req.TopupRatio, RequestLimit: req.RequestLimit, SuccessRequestLimit: req.SuccessRequestLimit}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.LockAccessPolicyState(tx); err != nil {
			return err
		}
		if level.IsDefault {
			if !level.Enabled {
				return common.NewPublicError(i18n.MsgAccessPolicyDefaultMustBeEnabled, nil)
			}
			if err := tx.Model(&model.UserLevel{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&level).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	finishAccessPolicyMutation(c, level)
}

func UpdateUserLevel(c *gin.Context) {
	code := c.Param("code")
	var req userLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := validateUserLevelRequest(req); err != nil {
		common.ApiError(c, err)
		return
	}
	var updated model.UserLevel
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.LockAccessPolicyState(tx); err != nil {
			return err
		}
		if err := tx.Where("code = ?", code).First(&updated).Error; err != nil {
			return err
		}
		if updated.Enabled && !req.Enabled {
			refs, err := model.InspectUserLevelReferences(tx, code)
			if err != nil {
				return err
			}
			if len(refs) > 0 {
				return common.NewPublicError(i18n.MsgAccessPolicyUserLevelReferenced, nil)
			}
		}
		if req.IsDefault && !req.Enabled {
			return common.NewPublicError(i18n.MsgAccessPolicyDefaultMustBeEnabled, nil)
		}
		if updated.IsDefault && !req.IsDefault {
			var otherDefaults int64
			if err := tx.Model(&model.UserLevel{}).Where("code <> ? AND is_default = ? AND enabled = ?", code, true, true).Count(&otherDefaults).Error; err != nil {
				return err
			}
			if otherDefaults == 0 {
				return common.NewPublicError(i18n.MsgAccessPolicyDefaultRequired, nil)
			}
		}
		if req.IsDefault {
			if err := tx.Model(&model.UserLevel{}).Where("code <> ?", code).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		updates := map[string]interface{}{"name": strings.TrimSpace(req.Name), "description": strings.TrimSpace(req.Description), "is_default": req.IsDefault, "enabled": req.Enabled, "topup_ratio": req.TopupRatio, "request_limit": req.RequestLimit, "success_request_limit": req.SuccessRequestLimit}
		if err := tx.Model(&model.UserLevel{}).Where("code = ?", code).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("code = ?", code).First(&updated).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	finishAccessPolicyMutation(c, updated)
}

func DeleteUserLevel(c *gin.Context) {
	code := c.Param("code")
	refs, err := model.UserLevelReferences(code)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(refs) > 0 {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": common.TranslateMessage(c, i18n.MsgAccessPolicyUserLevelReferenced), "references": refs})
		return
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.LockAccessPolicyState(tx); err != nil {
			return err
		}
		var level model.UserLevel
		if err := tx.Where("code = ?", code).First(&level).Error; err != nil {
			return err
		}
		if level.IsDefault {
			return common.NewPublicError(i18n.MsgAccessPolicyDeleteDefaultDenied, nil)
		}
		refs, err = model.InspectUserLevelReferences(tx, code)
		if err != nil {
			return err
		}
		if len(refs) > 0 {
			return common.NewPublicError(i18n.MsgAccessPolicyUserLevelReferenced, errUserLevelReferenced)
		}
		if err := tx.Where("user_level_code = ?", code).Delete(&model.UserLevelRouteGroup{}).Error; err != nil {
			return err
		}
		if err := tx.Where("code = ?", code).Delete(&model.UserLevel{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errUserLevelReferenced) {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": common.TranslateMessage(c, i18n.MsgAccessPolicyUserLevelReferenced), "references": refs})
			return
		}
		common.ApiError(c, err)
		return
	}
	finishAccessPolicyMutation(c, gin.H{"code": code})
}

func ReplaceUserLevelRouteGroups(c *gin.Context) {
	levelCode := c.Param("code")
	var req replaceRouteGroupGrantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	seen := make(map[string]struct{}, len(req.RouteGroups))
	for _, grant := range req.RouteGroups {
		if _, exists := seen[grant.Code]; exists {
			common.ApiErrorI18n(c, i18n.MsgAccessPolicyRouteGroupDuplicate)
			return
		}
		seen[grant.Code] = struct{}{}
		if grant.Code == model.LegacyDefaultGroupCode {
			common.ApiErrorI18n(c, i18n.MsgAccessPolicyRouteGroupRetired)
			return
		}
		if grant.PriceRatio != nil && (*grant.PriceRatio < 0 || math.IsNaN(float64(*grant.PriceRatio)) || math.IsInf(float64(*grant.PriceRatio), 0)) {
			common.ApiErrorI18n(c, i18n.MsgAccessPolicyPriceRatioInvalid)
			return
		}
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.LockAccessPolicyState(tx); err != nil {
			return err
		}
		var level model.UserLevel
		if err := tx.Where("code = ?", levelCode).First(&level).Error; err != nil {
			return err
		}
		for _, grant := range req.RouteGroups {
			var count int64
			if err := tx.Model(&model.RouteGroup{}).Where("code = ?", grant.Code).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return common.NewPublicError(i18n.MsgAccessPolicyRouteGroupNotFound, nil, map[string]any{"Group": grant.Code})
			}
		}
		if err := tx.Where("user_level_code = ?", levelCode).Delete(&model.UserLevelRouteGroup{}).Error; err != nil {
			return err
		}
		for _, input := range req.RouteGroups {
			grant := model.UserLevelRouteGroup{UserLevelCode: levelCode, RouteGroupCode: input.Code, PriceRatio: input.PriceRatio}
			if err := tx.Create(&grant).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.RefreshAccessPolicyCaches(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, model.GetUserLevelRouteGroups(levelCode, true))
}

func ListRouteGroups(c *gin.Context) {
	groups, err := model.ListRouteGroups()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	type view struct {
		model.RouteGroup
		References []model.AccessPolicyReference `json:"references"`
	}
	result := make([]view, 0, len(groups))
	for _, group := range groups {
		refs, refErr := model.RouteGroupReferences(group.Code)
		if refErr != nil {
			common.ApiError(c, refErr)
			return
		}
		result = append(result, view{RouteGroup: group, References: refs})
	}
	common.ApiSuccess(c, result)
}

func CreateRouteGroup(c *gin.Context) {
	var req createRouteGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := validateAccessPolicyCode(req.Code); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Code == model.LegacyDefaultGroupCode {
		common.ApiErrorI18n(c, i18n.MsgAccessPolicyRouteGroupRetired)
		return
	}
	if err := validateRouteGroupRequest(req.routeGroupRequest); err != nil {
		common.ApiError(c, err)
		return
	}
	group := model.RouteGroup{Code: req.Code, Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), BaseRatio: req.BaseRatio, Enabled: req.Enabled}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.LockAccessPolicyState(tx); err != nil {
			return err
		}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	finishAccessPolicyMutation(c, group)
}

func UpdateRouteGroup(c *gin.Context) {
	code := c.Param("code")
	if code == model.LegacyDefaultGroupCode {
		common.ApiErrorI18n(c, i18n.MsgAccessPolicyRouteGroupRetired)
		return
	}
	var req routeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := validateRouteGroupRequest(req); err != nil {
		common.ApiError(c, err)
		return
	}
	var updated model.RouteGroup
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.LockAccessPolicyState(tx); err != nil {
			return err
		}
		if err := tx.Where("code = ?", code).First(&updated).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{"name": strings.TrimSpace(req.Name), "description": strings.TrimSpace(req.Description), "base_ratio": req.BaseRatio, "enabled": req.Enabled}
		if err := tx.Model(&model.RouteGroup{}).Where("code = ?", code).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("code = ?", code).First(&updated).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	finishAccessPolicyMutation(c, updated)
}

func DeleteRouteGroup(c *gin.Context) {
	code := c.Param("code")
	if code == model.LegacyDefaultGroupCode {
		common.ApiErrorI18n(c, i18n.MsgAccessPolicyRetiredGroupManaged)
		return
	}
	refs, err := model.RouteGroupReferences(code)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(refs) > 0 {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": common.TranslateMessage(c, i18n.MsgAccessPolicyRouteGroupReferenced), "references": refs})
		return
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.LockAccessPolicyState(tx); err != nil {
			return err
		}
		refs, err = model.InspectRouteGroupReferences(tx, code)
		if err != nil {
			return err
		}
		if len(refs) > 0 {
			return common.NewPublicError(i18n.MsgAccessPolicyRouteGroupReferenced, errRouteGroupReferenced)
		}
		result := tx.Where("code = ?", code).Delete(&model.RouteGroup{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		groupModelRatios := ratio_setting.GetGroupModelRatioCopy()
		delete(groupModelRatios, code)
		if err := writeControllerOption(tx, "GroupModelRatio", groupModelRatios); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errRouteGroupReferenced) {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": common.TranslateMessage(c, i18n.MsgAccessPolicyRouteGroupReferenced), "references": refs})
			return
		}
		common.ApiError(c, err)
		return
	}
	finishAccessPolicyMutation(c, gin.H{"code": code})
}

func writeControllerOption(tx *gorm.DB, key string, value interface{}) error {
	bytes, err := common.Marshal(value)
	if err != nil {
		return err
	}
	return tx.Assign(model.Option{Value: string(bytes)}).FirstOrCreate(&model.Option{Key: key}).Error
}

func GetMyRouteGroups(c *gin.Context) {
	level, err := model.GetUserLevel(c.GetInt("id"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"user_level": level, "route_groups": model.GetUserLevelRouteGroups(level, false)})
}
