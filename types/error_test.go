package types

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrappingNewAPIErrorPreservesUpstreamSource(t *testing.T) {
	providerErr := NewError(
		errors.New("provider detail"),
		ErrorCodeBadResponse,
		ErrOptionWithErrorSource(ErrorSourceUpstream),
	)

	wrapped := NewOpenAIError(
		providerErr,
		ErrorCodeDoRequestFailed,
		http.StatusInternalServerError,
	)

	require.Same(t, providerErr, wrapped)
	assert.Equal(t, ErrorSourceUpstream, wrapped.GetSource())
	assert.Equal(t, "provider detail", wrapped.Error())
}
