package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestDetailResponseWriterCapturesWithoutChangingResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	writer := &requestDetailResponseWriter{
		ResponseWriter: context.Writer,
		maxBodyBytes:   5,
		captureAll:     true,
	}
	writer.Header().Set("Content-Type", "application/json")

	written, err := writer.WriteString("{\"answer\":\"complete\"}")
	require.NoError(t, err)
	assert.Equal(t, len("{\"answer\":\"complete\"}"), written)
	assert.Equal(t, "{\"answer\":\"complete\"}", recorder.Body.String())
	assert.Equal(t, []byte("{\"ans"), writer.body)
	assert.Equal(t, int64(len("{\"answer\":\"complete\"}")), writer.bodySize)
	assert.True(t, writer.truncated)
	assert.Empty(t, writer.omittedReason)
}

func TestRequestDetailResponseWriterSkipsBinaryContent(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	writer := &requestDetailResponseWriter{
		ResponseWriter: context.Writer,
		maxBodyBytes:   1024,
		captureAll:     true,
	}
	writer.Header().Set("Content-Type", "image/png")

	written, err := writer.Write([]byte{0x89, 0x50, 0x4e, 0x47})
	require.NoError(t, err)
	assert.Equal(t, 4, written)
	assert.Equal(t, []byte{0x89, 0x50, 0x4e, 0x47}, recorder.Body.Bytes())
	assert.Empty(t, writer.body)
	assert.Equal(t, int64(4), writer.bodySize)
	assert.Equal(t, "unsupported_content_type", writer.omittedReason)
}

func TestRequestDetailResponseWriterFailureModeSkipsSuccessfulBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	writer := &requestDetailResponseWriter{
		ResponseWriter: context.Writer,
		maxBodyBytes:   1024,
	}
	writer.Header().Set("Content-Type", "application/json")

	_, err := writer.WriteString("{\"ok\":true}")
	require.NoError(t, err)
	assert.Equal(t, "{\"ok\":true}", recorder.Body.String())
	assert.Empty(t, writer.body)
	assert.Equal(t, int64(len("{\"ok\":true}")), writer.bodySize)
}

func TestRequestDetailResponseWriterFailureModeCapturesErrorBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	writer := &requestDetailResponseWriter{
		ResponseWriter: context.Writer,
		maxBodyBytes:   1024,
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(503)

	_, err := writer.WriteString("{\"error\":\"unavailable\"}")
	require.NoError(t, err)
	assert.Equal(t, "{\"error\":\"unavailable\"}", recorder.Body.String())
	assert.Equal(t, []byte("{\"error\":\"unavailable\"}"), writer.body)
}
