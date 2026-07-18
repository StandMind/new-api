package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetReadiness(t *testing.T) {
	previous := common.IsProcessReady()
	t.Cleanup(func() {
		common.SetProcessReady(previous)
	})

	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		ready      bool
		statusCode int
	}{
		{name: "ready", ready: true, statusCode: http.StatusOK},
		{name: "not ready", ready: false, statusCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.SetProcessReady(tt.ready)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			GetReadiness(ctx)

			require.Equal(t, tt.statusCode, recorder.Code)
			var body struct {
				Success bool `json:"success"`
				Ready   bool `json:"ready"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
			require.Equal(t, tt.ready, body.Success)
			require.Equal(t, tt.ready, body.Ready)
		})
	}
}
