package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/official_price_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func callUpdateOption(t *testing.T, key string, value string) map[string]any {
	t.Helper()
	body, err := common.Marshal(OptionUpdateRequest{Key: key, Value: value})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/option/", bytes.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")

	UpdateOption(context)

	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestUpdateOptionRejectsInvalidOfficialPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := callUpdateOption(
		t,
		official_price_setting.OptionKey,
		`{"model":{"unit":"usd_per_request","tiers":[{"price":0}]}}`,
	)

	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid setting value", response["message"])
}

func TestUpdateOptionRejectsDeprecatedAutomaticGroupSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, key := range []string{
		"AutoGroups",
		"DefaultUseAutoGroup",
		"routing_setting.user_group_chain_enabled",
	} {
		t.Run(key, func(t *testing.T) {
			response := callUpdateOption(t, key, "true")
			assert.Equal(t, false, response["success"])
			assert.Equal(t, "This setting has been deprecated", response["message"])
		})
	}
}
