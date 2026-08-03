package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterPricingByUsableGroupsPrunesUnavailableGroups(t *testing.T) {
	pricing := []model.Pricing{
		{ModelName: "shared", EnableGroup: []string{"enabled", "disabled"}},
		{ModelName: "blocked", EnableGroup: []string{"disabled"}},
		{ModelName: "global", EnableGroup: []string{"all", "disabled"}},
	}

	filtered := filterPricingByUsableGroups(pricing, map[string]string{"enabled": ""})

	require.Len(t, filtered, 2)
	assert.Equal(t, "shared", filtered[0].ModelName)
	assert.Equal(t, []string{"enabled"}, filtered[0].EnableGroup)
	assert.Equal(t, "global", filtered[1].ModelName)
	assert.Equal(t, []string{"all"}, filtered[1].EnableGroup)
	assert.Equal(t, []string{"enabled", "disabled"}, pricing[0].EnableGroup)
}

func TestFilterPricingByUsableGroupsRejectsEmptyAccess(t *testing.T) {
	pricing := []model.Pricing{{ModelName: "model", EnableGroup: []string{"group"}}}

	assert.Empty(t, filterPricingByUsableGroups(pricing, nil))
}
