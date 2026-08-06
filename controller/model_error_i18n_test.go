package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrieveModelLocalizesNotFoundWithoutChangingErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models/does-not-exist", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")
	c.Params = gin.Params{{Key: "model", Value: "does-not-exist"}}

	RetrieveModel(c, 0)

	var response struct {
		Error types.OpenAIError `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, i18n.Translate(i18n.LangZhCN, i18n.MsgRelayModelNotFound), response.Error.Message)
	assert.Equal(t, "invalid_request_error", response.Error.Type)
	assert.Equal(t, "model_not_found", response.Error.Code)
}
