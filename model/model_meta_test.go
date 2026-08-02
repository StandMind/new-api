package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeModelReleaseDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		shouldErr bool
	}{
		{name: "empty", input: "", expected: ""},
		{name: "trimmed", input: " 2026-07-31 ", expected: "2026-07-31"},
		{name: "invalid format", input: "2026-7-31", shouldErr: true},
		{name: "invalid date", input: "2026-02-30", shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := normalizeModelReleaseDate(tt.input)
			if tt.shouldErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
