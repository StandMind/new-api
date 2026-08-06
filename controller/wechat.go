package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type wechatLoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

func getWeChatIdByCode(code string) (string, error) {
	if code == "" {
		return "", common.NewPublicError(i18n.MsgOAuthInvalidCode, nil)
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/wechat/user?code=%s", common.WeChatServerAddress, url.QueryEscape(code)), nil)
	if err != nil {
		return "", common.NewPublicError(i18n.MsgOAuthConnectFailed, err, map[string]any{"Provider": "WeChat"})
	}
	req.Header.Set("Authorization", common.WeChatServerToken)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	httpResponse, err := client.Do(req)
	if err != nil {
		return "", common.NewPublicError(i18n.MsgOAuthConnectFailed, err, map[string]any{"Provider": "WeChat"})
	}
	defer httpResponse.Body.Close()
	var res wechatLoginResponse
	err = common.DecodeJson(httpResponse.Body, &res)
	if err != nil {
		return "", common.NewPublicError(i18n.MsgRelayInvalidUpstreamResponse, err)
	}
	if !res.Success {
		return "", common.NewUpstreamError(res.Message, nil)
	}
	if res.Data == "" {
		return "", common.NewPublicError(i18n.MsgOAuthInvalidCode, nil)
	}
	return res.Data, nil
}

func WeChatAuth(c *gin.Context) {
	if !common.WeChatAuthEnabled {
		common.ApiErrorI18n(c, i18n.MsgOAuthNotEnabled, map[string]any{"Provider": "WeChat"})
		return
	}
	code := c.Query("code")
	wechatId, err := getWeChatIdByCode(code)
	if err != nil {
		if message, ok := common.UpstreamErrorMessage(err); ok {
			common.ApiUpstreamError(c, http.StatusOK, message)
		} else {
			common.ApiError(c, err)
		}
		return
	}
	user := model.User{
		WeChatId: wechatId,
	}
	if model.IsWeChatIdAlreadyTaken(wechatId) {
		err := user.FillUserByWeChatId()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if user.Id == 0 {
			common.ApiErrorI18n(c, i18n.MsgOAuthUserDeleted)
			return
		}
	} else {
		if common.RegisterEnabled {
			user.Username = "wechat_" + strconv.Itoa(model.GetMaxUserId()+1)
			user.DisplayName = "WeChat User"
			user.Role = common.RoleCommonUser
			user.Status = common.UserStatusEnabled

			if err := user.Insert(0); err != nil {
				common.ApiError(c, err)
				return
			}
		} else {
			common.ApiErrorI18n(c, i18n.MsgUserRegisterDisabled)
			return
		}
	}

	if user.Status != common.UserStatusEnabled {
		common.ApiErrorI18n(c, i18n.MsgOAuthUserBanned)
		return
	}
	setupLogin(&user, c)
}

type wechatBindRequest struct {
	Code string `json:"code"`
}

func WeChatBind(c *gin.Context) {
	if !common.WeChatAuthEnabled {
		common.ApiErrorI18n(c, i18n.MsgOAuthNotEnabled, map[string]any{"Provider": "WeChat"})
		return
	}
	var req wechatBindRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	code := req.Code
	wechatId, err := getWeChatIdByCode(code)
	if err != nil {
		if message, ok := common.UpstreamErrorMessage(err); ok {
			common.ApiUpstreamError(c, http.StatusOK, message)
		} else {
			common.ApiError(c, err)
		}
		return
	}
	if model.IsWeChatIdAlreadyTaken(wechatId) {
		common.ApiErrorI18n(c, i18n.MsgOAuthAlreadyBound, map[string]any{"Provider": "WeChat"})
		return
	}
	session := sessions.Default(c)
	id := session.Get("id")
	user := model.User{
		Id: id.(int),
	}
	err = user.FillUserById()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user.WeChatId = wechatId
	err = user.Update(false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}
