package controller

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

type Setup struct {
	Status       bool   `json:"status"`
	RootInit     bool   `json:"root_init"`
	DatabaseType string `json:"database_type"`
}

type SetupRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	ConfirmPassword    string `json:"confirmPassword"`
	SelfUseModeEnabled bool   `json:"SelfUseModeEnabled"`
	DemoSiteEnabled    bool   `json:"DemoSiteEnabled"`
}

func GetSetup(c *gin.Context) {
	setup := Setup{
		Status: constant.Setup,
	}
	if constant.Setup {
		c.JSON(200, gin.H{
			"success": true,
			"data":    setup,
		})
		return
	}
	setup.RootInit = model.RootUserExists()
	setup.DatabaseType = string(common.MainDatabaseType())
	c.JSON(200, gin.H{
		"success": true,
		"data":    setup,
	})
}

func PostSetup(c *gin.Context) {
	// Check if setup is already completed
	if constant.Setup {
		common.ApiErrorI18n(c, i18n.MsgSetupAlreadyCompleted)
		return
	}

	// Check if root user already exists
	rootExists := model.RootUserExists()

	var req SetupRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	// If root doesn't exist, validate and create admin account
	if !rootExists {
		// Validate username length: max 12 characters to align with model.User validation
		if len(req.Username) > 12 {
			common.ApiErrorI18n(c, i18n.MsgSetupUsernameTooLong, map[string]any{"Max": 12})
			return
		}
		// Validate password
		if req.Password != req.ConfirmPassword {
			common.ApiErrorI18n(c, i18n.MsgSetupPasswordMismatch)
			return
		}

		if len(req.Password) < 8 {
			common.ApiErrorI18n(c, i18n.MsgSetupPasswordTooShort, map[string]any{"Min": 8})
			return
		}

		// Create root user
		hashedPassword, err := common.Password2Hash(req.Password)
		if err != nil {
			common.SysError("failed to hash initial root password: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgSetupInitializationFailed)
			return
		}
		rootUser := model.User{
			Username:    req.Username,
			Password:    hashedPassword,
			Role:        common.RoleRootUser,
			Status:      common.UserStatusEnabled,
			DisplayName: "Root User",
			AccessToken: nil,
			Quota:       100000000,
		}
		err = model.DB.Create(&rootUser).Error
		if err != nil {
			common.SysError("failed to create initial root user: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgSetupInitializationFailed)
			return
		}
	}

	// Set operation modes
	operation_setting.SelfUseModeEnabled = req.SelfUseModeEnabled
	operation_setting.DemoSiteEnabled = req.DemoSiteEnabled

	// Save operation modes to database for persistence
	err = model.UpdateOption("SelfUseModeEnabled", boolToString(req.SelfUseModeEnabled))
	if err != nil {
		common.SysError("failed to save initial self-use mode: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgSetupInitializationFailed)
		return
	}

	err = model.UpdateOption("DemoSiteEnabled", boolToString(req.DemoSiteEnabled))
	if err != nil {
		common.SysError("failed to save initial demo mode: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgSetupInitializationFailed)
		return
	}

	// Update setup status
	constant.Setup = true

	setup := model.Setup{
		Version:       common.Version,
		InitializedAt: time.Now().Unix(),
	}
	err = model.DB.Create(&setup).Error
	if err != nil {
		common.SysError("failed to persist initial setup record: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgSetupInitializationFailed)
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": i18n.T(c, i18n.MsgSetupInitializationSuccess),
	})
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
