package middleware

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func abortWithOpenAIMessageKey(c *gin.Context, statusCode int, key string, args map[string]any, code ...types.ErrorCode) {
	errorCode := types.ErrorCode("")
	if len(code) > 0 && code[0] != "" {
		errorCode = code[0]
	}
	err := types.NewErrorWithStatusCode(
		errors.New(key),
		errorCode,
		statusCode,
		types.ErrOptionWithPublicMessage(key, args),
	)
	abortWithNewAPIError(c, err)
}

func abortWithNewAPIError(c *gin.Context, err *types.NewAPIError) {
	if err == nil {
		return
	}
	openAIError := service.OpenAIErrorForResponse(c, err, c.GetString(common.RequestIdKey))
	service.SetRequestDetailFailure(c, &service.RequestDetailFailure{
		StatusCode: err.StatusCode,
		ErrorType:  string(err.GetErrorType()),
		ErrorCode:  string(err.GetErrorCode()),
		Message:    err.MaskSensitiveErrorWithStatusCode(),
	})
	userId := c.GetInt("id")
	c.JSON(err.StatusCode, gin.H{
		"error": openAIError,
	})
	c.Abort()
	logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | %s", userId, err.MaskSensitiveError()))
}

func abortWithMidjourneyMessage(c *gin.Context, statusCode int, code int, description string) {
	service.SetRequestDetailFailure(c, &service.RequestDetailFailure{
		StatusCode: statusCode,
		ErrorType:  "new_api_error",
		ErrorCode:  fmt.Sprintf("%d", code),
		Message:    description,
	})
	c.JSON(statusCode, gin.H{
		"description": description,
		"type":        "new_api_error",
		"code":        code,
	})
	c.Abort()
	logger.LogError(c.Request.Context(), description)
}
