package middleware

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func validUserInfo(username string, role int) bool {
	// check username is empty
	if strings.TrimSpace(username) == "" {
		return false
	}
	if !common.IsValidateRole(role) {
		return false
	}
	return true
}

func applySessionLanguage(c *gin.Context, session sessions.Session) {
	value := session.Get("language")
	if value == nil {
		return
	}
	common.SetContextKey(c, constant.ContextKeyUserLanguageLookupDone, true)
	if sessionLanguage, ok := value.(string); ok {
		if lang, valid := i18n.NormalizeLanguage(sessionLanguage); valid {
			common.SetContextKey(c, constant.ContextKeyUserLanguage, lang)
		}
	}
}

func authHelper(c *gin.Context, minRole int) {
	session := sessions.Default(c)
	username := session.Get("username")
	role := session.Get("role")
	id := session.Get("id")
	status := session.Get("status")
	if sessionID, ok := id.(int); ok && sessionID > 0 {
		// Legacy sessions do not contain a language. Expose the signed session ID
		// early so an authentication error can load that preference once.
		c.Set("id", sessionID)
	}
	applySessionLanguage(c, session)
	useAccessToken := false
	accessTokenUserLevel := ""
	accessTokenLanguage := ""
	if username == nil {
		// Check access token
		accessToken := c.Request.Header.Get("Authorization")
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgAuthNotLoggedIn),
			})
			c.Abort()
			return
		}
		user, authErr := model.ValidateAccessToken(accessToken)
		if authErr != nil {
			if errors.Is(authErr, model.ErrDatabase) {
				common.SysLog("ValidateAccessToken database error: " + authErr.Error())
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": common.TranslateMessage(c, i18n.MsgDatabaseError),
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": common.TranslateMessage(c, i18n.MsgAuthAccessTokenInvalid),
				})
			}
			c.Abort()
			return
		}
		if user != nil && user.Username != "" {
			if !validUserInfo(user.Username, user.Role) {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": common.TranslateMessage(c, i18n.MsgAuthUserInfoInvalid),
				})
				c.Abort()
				return
			}
			// Token is valid
			username = user.Username
			role = user.Role
			id = user.Id
			status = user.Status
			model.PopulateUserLevelDisplay(user)
			accessTokenUserLevel = user.UserLevel
			accessTokenLanguage = user.GetSetting().Language
			common.SetContextKey(c, constant.ContextKeyUserLanguageLookupDone, true)
			if lang, ok := i18n.NormalizeLanguage(accessTokenLanguage); ok {
				common.SetContextKey(c, constant.ContextKeyUserLanguage, lang)
			}
			useAccessToken = true
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgAuthAccessTokenInvalid),
			})
			c.Abort()
			return
		}
	}
	// get header New-Api-User
	apiUserIdStr := c.Request.Header.Get("New-Api-User")
	if apiUserIdStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgAuthUserIdNotProvided),
		})
		c.Abort()
		return
	}
	apiUserId, err := strconv.Atoi(apiUserIdStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgAuthUserIdFormatError),
		})
		c.Abort()
		return

	}
	if id != apiUserId {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgAuthUserIdMismatch),
		})
		c.Abort()
		return
	}
	if status.(int) == common.UserStatusDisabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgAuthUserBanned),
		})
		c.Abort()
		return
	}
	if role.(int) < minRole {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgAuthInsufficientPrivilege),
		})
		c.Abort()
		return
	}
	if !validUserInfo(username.(string), role.(int)) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgAuthUserInfoInvalid),
		})
		c.Abort()
		return
	}
	// 防止不同newapi版本冲突，导致数据不通用
	c.Header("Auth-Version", "864b7076dbcd0a3c01b5520316720ebf")
	c.Set("username", username)
	c.Set("role", role)
	c.Set("id", id)
	userLevel := accessTokenUserLevel
	if !useAccessToken {
		userLevel, _ = session.Get("user_level").(string)
		if userLevel == "" {
			legacyGroup, _ := session.Get("group").(string)
			userLevel = model.UserLevelForLegacyGroup(legacyGroup)
		}
	}
	c.Set("group", userLevel)
	c.Set("user_group", userLevel)
	c.Set("user_level", userLevel)
	c.Set("use_access_token", useAccessToken)
	if lang, ok := i18n.NormalizeLanguage(accessTokenLanguage); useAccessToken && ok {
		common.SetContextKey(c, constant.ContextKeyUserLanguage, lang)
	}

	// 管理/root 写操作审计兜底：内聚在鉴权链路里，保证任何经过 AdminAuth/RootAuth
	// 的写接口都会自动留痕（无需在路由上单独挂审计中间件，避免漏挂）。
	// handler 内手动埋点者会设置 ContextKeyAuditLogged，finishAdminAudit 据此跳过。
	var auditWriter *auditResponseWriter
	if minRole >= common.RoleAdminUser {
		auditWriter = beginAdminAudit(c)
	}

	c.Next()

	finishAdminAudit(c, auditWriter)
}

func TryUserAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		id := session.Get("id")
		if id != nil {
			c.Set("id", id)
		}
		applySessionLanguage(c, session)
		c.Next()
	}
}

func UserAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, common.RoleCommonUser)
	}
}

func AdminAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, common.RoleAdminUser)
	}
}

func RootAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, common.RoleRootUser)
	}
}

func RequirePermission(permission authz.Permission) func(c *gin.Context) {
	return func(c *gin.Context) {
		role := c.GetInt("role")
		userID := c.GetInt("id")
		if authz.Can(userID, role, permission) {
			c.Next()
			return
		}
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgAuthInsufficientPrivilege),
		})
		c.Abort()
	}
}

func WssAuth(c *gin.Context) {

}

// TokenOrUserAuth allows either session-based user auth or API token auth.
// Used for endpoints that need to be accessible from both the dashboard and API clients.
func TokenOrUserAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		// Try session auth first (dashboard users)
		session := sessions.Default(c)
		if id := session.Get("id"); id != nil {
			if status, ok := session.Get("status").(int); ok && status == common.UserStatusEnabled {
				c.Set("id", id)
				applySessionLanguage(c, session)
				c.Next()
				return
			}
		}
		// Fall back to token auth (API clients)
		TokenAuth()(c)
	}
}

// TokenAuthReadOnly 宽松版本的令牌认证中间件，用于只读查询接口。
// 只验证令牌 key 是否存在，不检查令牌状态、过期时间和额度。
// 即使令牌已过期、已耗尽或已禁用，也允许访问。
// 仍然检查用户是否被封禁。
func TokenAuthReadOnly() func(c *gin.Context) {
	return func(c *gin.Context) {
		key := c.Request.Header.Get("Authorization")
		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgTokenNotProvided),
			})
			c.Abort()
			return
		}
		if strings.HasPrefix(key, "Bearer ") || strings.HasPrefix(key, "bearer ") {
			key = strings.TrimSpace(key[7:])
		}
		key = strings.TrimPrefix(key, "sk-")
		parts := strings.Split(key, "-")
		key = parts[0]

		token, err := model.GetTokenByKey(key, false)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"message": common.TranslateMessage(c, i18n.MsgTokenInvalid),
				})
			} else {
				common.SysLog("TokenAuthReadOnly GetTokenByKey database error: " + err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": common.TranslateMessage(c, i18n.MsgDatabaseError),
				})
			}
			c.Abort()
			return
		}

		// TokenAuthReadOnly must keep allowing other token states to query read-only
		// data, such as token usage logs; only explicitly disabled tokens are denied.
		if token.Status == common.TokenStatusDisabled {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgTokenStatusUnavailable),
			})
			c.Abort()
			return
		}

		userCache, err := model.GetUserCache(token.UserId)
		if err != nil {
			common.SysLog(fmt.Sprintf("TokenAuthReadOnly GetUserCache error for user %d: %v", token.UserId, err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgDatabaseError),
			})
			c.Abort()
			return
		}
		if userCache.Status != common.UserStatusEnabled {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgAuthUserBanned),
			})
			c.Abort()
			return
		}

		userCache.WriteContext(c)
		c.Set("id", token.UserId)
		c.Set("token_id", token.Id)
		c.Set("token_key", token.Key)
		c.Next()
	}
}

func TokenAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 先检测是否为ws
		if c.Request.Header.Get("Sec-WebSocket-Protocol") != "" {
			// Sec-WebSocket-Protocol: realtime, openai-insecure-api-key.sk-xxx, openai-beta.realtime-v1
			// read sk from Sec-WebSocket-Protocol
			key := c.Request.Header.Get("Sec-WebSocket-Protocol")
			parts := strings.Split(key, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "openai-insecure-api-key") {
					key = strings.TrimPrefix(part, "openai-insecure-api-key.")
					break
				}
			}
			c.Request.Header.Set("Authorization", "Bearer "+key)
		}
		// 检查path包含/v1/messages 或 /v1/models
		if strings.Contains(c.Request.URL.Path, "/v1/messages") || strings.Contains(c.Request.URL.Path, "/v1/models") {
			anthropicKey := c.Request.Header.Get("x-api-key")
			if anthropicKey != "" {
				c.Request.Header.Set("Authorization", "Bearer "+anthropicKey)
			}
		}
		// gemini api 从query中获取key
		if strings.HasPrefix(c.Request.URL.Path, "/v1beta/models") ||
			strings.HasPrefix(c.Request.URL.Path, "/v1beta/openai/models") ||
			strings.HasPrefix(c.Request.URL.Path, "/v1/models/") {
			skKey := c.Query("key")
			if skKey != "" {
				c.Request.Header.Set("Authorization", "Bearer "+skKey)
			}
			// 从x-goog-api-key header中获取key
			xGoogKey := c.Request.Header.Get("x-goog-api-key")
			if xGoogKey != "" {
				c.Request.Header.Set("Authorization", "Bearer "+xGoogKey)
			}
		}
		key := c.Request.Header.Get("Authorization")
		parts := make([]string, 0)
		if strings.HasPrefix(key, "Bearer ") || strings.HasPrefix(key, "bearer ") {
			key = strings.TrimSpace(key[7:])
		}
		if key == "" || key == "midjourney-proxy" {
			key = c.Request.Header.Get("mj-api-secret")
			if strings.HasPrefix(key, "Bearer ") || strings.HasPrefix(key, "bearer ") {
				key = strings.TrimSpace(key[7:])
			}
			key = strings.TrimPrefix(key, "sk-")
			parts = strings.Split(key, "-")
			key = parts[0]
		} else {
			key = strings.TrimPrefix(key, "sk-")
			parts = strings.Split(key, "-")
			key = parts[0]
		}
		token, err := model.ValidateUserToken(key)
		if token != nil {
			id := c.GetInt("id")
			if id == 0 {
				c.Set("id", token.UserId)
			}
		}
		if err != nil {
			if errors.Is(err, model.ErrDatabase) {
				common.SysLog("TokenAuth ValidateUserToken database error: " + err.Error())
				abortWithOpenAIMessageKey(c, http.StatusInternalServerError, i18n.MsgDatabaseError, nil)
			} else {
				abortWithOpenAIMessageKey(c, http.StatusUnauthorized, i18n.MsgTokenInvalid, nil)
			}
			return
		}

		userCache, err := model.GetUserCache(token.UserId)
		if err != nil {
			common.SysLog(fmt.Sprintf("TokenAuth GetUserCache error for user %d: %v", token.UserId, err))
			abortWithOpenAIMessageKey(c, http.StatusInternalServerError, i18n.MsgDatabaseError, nil)
			return
		}
		userCache.WriteContext(c)
		if userCache.Status != common.UserStatusEnabled {
			abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthUserBanned, nil)
			return
		}

		allowIps := token.GetIpLimits()
		if len(allowIps) > 0 {
			clientIp := c.ClientIP()
			logger.LogDebug(c, "Token has IP restrictions, checking client IP %s", clientIp)
			ip := net.ParseIP(clientIp)
			if ip == nil {
				abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthClientIPInvalid, nil)
				return
			}
			if common.IsIpInCIDRList(ip, allowIps) == false {
				abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthIPNotAllowed, nil, types.ErrorCodeAccessDenied)
				return
			}
			logger.LogDebug(c, "Client IP %s passed the token IP restrictions check", clientIp)
		}

		userGroup := userCache.UserLevel
		routingPriority := constant.NormalizeRoutingPriority(token.RoutingPriority)
		if !constant.IsValidRoutingPriority(routingPriority, true) {
			abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthRoutingModeInvalid, nil)
			return
		}
		tokenGroup := token.Group
		if tokenGroup == "auto" {
			abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthLegacyAutoToken, nil)
			return
		}
		if routingPriority != constant.RoutingPriorityManual {
			groups := service.GetSmartRoutingAccessibleGroups(userGroup)
			if len(groups) == 0 {
				abortWithOpenAIMessageKey(c, http.StatusServiceUnavailable, i18n.MsgAuthNoSmartRoutingGroup, nil)
				return
			}
			common.SetContextKey(c, constant.ContextKeyUsingGroup, groups[0])
			if err = SetupContextForToken(c, token, parts...); err != nil {
				return
			}
			common.SetContextKey(c, constant.ContextKeyTokenGroupChain, groups)
			common.SetContextKey(c, constant.ContextKeyTokenRoutingPriority, string(routingPriority))
			c.Next()
			return
		}
		groupChain := token.GetGroupChain()
		if len(groupChain) == 0 {
			abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthGroupChainMissing, nil)
			return
		}
		if tokenGroup != groupChain[0] {
			abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthGroupChainMismatch, nil)
			return
		}
		usableGroups := service.GetUserUsableGroups(userGroup)
		seenGroups := make(map[string]struct{}, len(groupChain))
		for _, group := range groupChain {
			if group == "auto" {
				abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthGroupChainContainsAuto, nil)
				return
			}
			if _, seen := seenGroups[group]; seen {
				abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthGroupChainDuplicate, map[string]any{"Group": group})
				return
			}
			seenGroups[group] = struct{}{}
			if _, ok := usableGroups[group]; !ok {
				abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthGroupNamedAccessDenied, map[string]any{"Group": group})
				return
			}
			routeGroup, exists := model.GetRouteGroupFromSnapshot(group)
			if !exists || !routeGroup.Enabled {
				abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthGroupRetired, map[string]any{"Group": group})
				return
			}
		}
		common.SetContextKey(c, constant.ContextKeyUsingGroup, groupChain[0])

		err = SetupContextForToken(c, token, parts...)
		if err != nil {
			return
		}
		c.Next()
	}
}

func SetupContextForToken(c *gin.Context, token *model.Token, parts ...string) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}
	c.Set("id", token.UserId)
	c.Set("token_id", token.Id)
	c.Set("token_key", token.Key)
	c.Set("token_name", token.Name)
	c.Set("token_unlimited_quota", token.UnlimitedQuota)
	if !token.UnlimitedQuota {
		c.Set("token_quota", token.RemainQuota)
	}
	if token.ModelLimitsEnabled {
		c.Set("token_model_limit_enabled", true)
		c.Set("token_model_limit", token.GetModelLimitsMap())
	} else {
		c.Set("token_model_limit_enabled", false)
	}
	common.SetContextKey(c, constant.ContextKeyTokenGroup, token.Group)
	common.SetContextKey(c, constant.ContextKeyTokenGroupChain, token.GetGroupChain())
	common.SetContextKey(c, constant.ContextKeyTokenRoutingPriority, token.RoutingPriority)
	if len(parts) > 1 {
		if model.IsAdmin(token.UserId) {
			c.Set("specific_channel_id", parts[1])
		} else {
			c.Header("specific_channel_version", "701e3ae1dc3f7975556d354e0675168d004891c8")
			abortWithOpenAIMessageKey(c, http.StatusForbidden, i18n.MsgAuthSpecificChannelDenied, nil)
			return fmt.Errorf("普通用户不支持指定渠道")
		}
	}
	return nil
}
