package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetLogsSelfStatRejectsChannelFilterBeforeQuery(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/log/self/stat?channel=9411", nil)

	GetLogsSelfStat(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"success":false,"message":"Invalid parameters"}`, recorder.Body.String())
}
