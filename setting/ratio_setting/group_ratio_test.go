package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGroupRatioPrecedence(t *testing.T) {
	originalGroupRatio := GroupRatio2JSONString()
	originalGroupGroupRatio := GroupGroupRatio2JSONString()
	originalGroupModelRatio := GroupModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupRatioByJSONString(originalGroupRatio))
		require.NoError(t, UpdateGroupGroupRatioByJSONString(originalGroupGroupRatio))
		require.NoError(t, UpdateGroupModelRatioByJSONString(originalGroupModelRatio))
	})

	require.NoError(t, UpdateGroupRatioByJSONString(`{"fallback":1.25}`))
	require.NoError(t, UpdateGroupGroupRatioByJSONString(`{"member":{"fallback":0.9}}`))
	require.NoError(t, UpdateGroupModelRatioByJSONString(
		`{"fallback":{"*":1.5,"gpt-*":2,"gpt-4*":3,"gpt-4o":4}}`,
	))

	modelSpecificTests := []struct {
		name       string
		userGroup  string
		model      string
		wantRatio  float64
		wantSource string
	}{
		{
			name:       "exact model wins",
			userGroup:  "member",
			model:      "gpt-4o",
			wantRatio:  4,
			wantSource: "group_model_ratio.exact",
		},
		{
			name:       "longest prefix wins",
			userGroup:  "member",
			model:      "gpt-4.1",
			wantRatio:  3,
			wantSource: "group_model_ratio.prefix",
		},
		{
			name:       "shorter prefix remains available",
			userGroup:  "member",
			model:      "gpt-3.5",
			wantRatio:  2,
			wantSource: "group_model_ratio.prefix",
		},
		{
			name:       "global wildcard precedes user group override",
			userGroup:  "member",
			model:      "claude",
			wantRatio:  1.5,
			wantSource: "group_model_ratio.prefix",
		},
	}

	for _, test := range modelSpecificTests {
		t.Run(test.name, func(t *testing.T) {
			ratio, source := ResolveGroupRatio(test.userGroup, "fallback", test.model)
			assert.Equal(t, test.wantRatio, ratio)
			assert.Equal(t, test.wantSource, source)
		})
	}

	require.NoError(t, UpdateGroupModelRatioByJSONString(
		`{"fallback":{"gpt-*":2,"gpt-4*":3,"gpt-4o":4}}`,
	))

	ratio, source := ResolveGroupRatio("member", "fallback", "claude")
	assert.Equal(t, 0.9, ratio)
	assert.Equal(t, "group_group_ratio", source)

	ratio, source = ResolveGroupRatio("other", "fallback", "claude")
	assert.Equal(t, 1.25, ratio)
	assert.Equal(t, "group_ratio", source)
}

func TestCheckGroupModelRatioRejectsInvalidConfiguration(t *testing.T) {
	tests := []string{
		`{"":{"gpt-4o":1}}`,
		`{"default":{"":1}}`,
		`{"default":{"gpt-*-mini":1}}`,
		`{"default":{"gpt-4o":-1}}`,
	}
	for _, value := range tests {
		assert.Error(t, CheckGroupModelRatio(value))
	}

	assert.NoError(t, CheckGroupModelRatio(`{"default":{"*":0.5,"gpt-*":0,"gpt-4o":1.5}}`))
}
