package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleHTTPPanicLocalizesAndHidesCause(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	c.Request.Header.Set("Accept-Language", "zh-CN")

	handleHTTPPanic(c, "database password secret")

	var response struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, i18n.Translate(i18n.LangZhCN, i18n.MsgRelayInternalPanic), response.Error.Message)
	assert.Equal(t, "new_api_panic", response.Error.Type)
	assert.NotContains(t, recorder.Body.String(), "database password secret")
}

func TestMigrationOnlyRequested(t *testing.T) {
	previous := os.Args
	t.Cleanup(func() {
		os.Args = previous
	})

	os.Args = []string{"new-api", "--log-dir", "/tmp/logs"}
	require.False(t, migrationOnlyRequested())

	os.Args = []string{"new-api", "--migrate-only", "--log-dir", "/tmp/logs"}
	require.True(t, migrationOnlyRequested())
}

func TestValidateMigrationNode(t *testing.T) {
	previous := common.IsMasterNode
	t.Cleanup(func() {
		common.IsMasterNode = previous
	})

	common.IsMasterNode = false
	require.ErrorContains(t, validateMigrationNode(), "NODE_TYPE=slave")

	common.IsMasterNode = true
	require.NoError(t, validateMigrationNode())
}
