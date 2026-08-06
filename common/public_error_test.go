package common

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyPublicMessageArgsBoundsAndDetachesValues(t *testing.T) {
	longValue := strings.Repeat("x", maxPublicMessageArgValue+32)
	input := map[string]any{
		"String":      longValue,
		"Number":      int64(42),
		"Boolean":     true,
		"Unsupported": []string{"request", "body"},
	}

	copied := CopyPublicMessageArgs(input)
	input["String"] = "changed"

	assert.Equal(t, strings.Repeat("x", maxPublicMessageArgValue), copied["String"])
	assert.Equal(t, int64(42), copied["Number"])
	assert.NotContains(t, copied, "Boolean")
	assert.NotContains(t, copied, "Unsupported")
}

func TestCopyPublicMessageArgsLimitsEntryCountAndKeyLength(t *testing.T) {
	input := map[string]any{
		strings.Repeat("k", maxPublicMessageArgKey+1): "ignored",
	}
	for index := 0; index < maxPublicMessageArgs+4; index++ {
		input[string(rune('a'+index))] = index
	}

	copied := CopyPublicMessageArgs(input)
	assert.Len(t, copied, maxPublicMessageArgs)
	assert.NotContains(t, copied, strings.Repeat("k", maxPublicMessageArgKey+1))
}

func TestCopyPublicMessageArgsKeepsTruncatedStringsValidUTF8(t *testing.T) {
	copied := CopyPublicMessageArgs(map[string]any{
		"Message": strings.Repeat("界", maxPublicMessageArgValue),
	})
	message, ok := copied["Message"].(string)
	require.True(t, ok)
	assert.LessOrEqual(t, len(message), maxPublicMessageArgValue)
	assert.True(t, utf8.ValidString(message))
}

func TestApiErrorHidesUnclassifiedCause(t *testing.T) {
	previousTranslate := TranslateMessage
	TranslateMessage = func(_ *gin.Context, key string, _ ...map[string]any) string {
		assert.Equal(t, defaultPublicErrorKey, key)
		return "Operation failed"
	}
	t.Cleanup(func() { TranslateMessage = previousTranslate })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	ApiErrorStatus(c, http.StatusInternalServerError, errors.New("database password secret"))

	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Operation failed", response["message"])
	assert.NotContains(t, recorder.Body.String(), "database password secret")
}

func TestApiUpstreamErrorPreservesMessageAndMasksSecrets(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	ApiUpstreamError(c, http.StatusBadGateway, "provider rejected https://api.example.com/v1?key=secret")

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "provider rejected")
	assert.NotContains(t, recorder.Body.String(), "api.example.com")
	assert.NotContains(t, recorder.Body.String(), "key=secret")
}

func BenchmarkCopyPublicMessageArgs(b *testing.B) {
	args := map[string]any{
		"Model": "gpt-5",
		"Group": "default",
		"Max":   100,
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = CopyPublicMessageArgs(args)
	}
}
