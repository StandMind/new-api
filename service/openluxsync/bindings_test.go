package openluxsync

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindingsResolveRenamedGroupsAllowMixedChannelsAndKeepCandidatesUnpersisted(t *testing.T) {
	const (
		sourceA = "Source-A"
		sourceB = "Source-B"
		localA  = "renamed-local-a"
		localB  = "renamed-local-b"
	)
	boundA := previewChannel(1, "OpenLux/Source-A", localA, "model-a", "https://api.openlux.ai")
	selfHosted := previewChannel(2, "self-hosted", localA, "model-a", "https://pool.internal.example")
	candidateB := previewChannel(3, "OpenLux-Ext/Source-B", localB, "model-b", "https://api.openlux.ai")
	snapshot := &model.OpenLuxSyncSnapshot{
		BindingRevision: 9,
		Bindings: []model.OpenLuxPriceSyncBinding{{
			SourceGroup: sourceA, ChannelID: boundA.Id,
		}},
		RouteGroups: []model.RouteGroup{
			{Code: localA, Name: localA, BaseRatio: 1, Enabled: true},
			{Code: localB, Name: localB, BaseRatio: 1, Enabled: true},
		},
		Channels: []model.Channel{boundA, selfHosted, candidateB},
	}
	source := previewSource(nil, map[string]decimal.Decimal{
		sourceA: decimal.NewFromInt(1),
		sourceB: decimal.NewFromInt(1),
	})

	response := buildBindingsResponse(source, snapshot)
	assert.Equal(t, int64(9), response.BindingRevision)
	require.Len(t, response.Bindings, 1)
	assert.Equal(t, localA, response.Bindings[0].LocalGroup)
	assert.Equal(t, 1, response.Bindings[0].MixedChannelCount)
	require.Len(t, response.Candidates, 1)
	assert.Equal(t, sourceB, response.Candidates[0].SourceGroup)
	assert.Equal(t, localB, response.Candidates[0].LocalGroup)
	assert.True(t, response.Candidates[0].Candidate)
	assert.Len(t, response.OpenLuxChannels, 2)

	validated, err := validateBindings([]SaveBinding{
		{SourceGroup: sourceA, ChannelIDs: []int{boundA.Id}},
		{SourceGroup: sourceB, ChannelIDs: []int{candidateB.Id}},
	}, source, snapshot)
	require.NoError(t, err)
	require.Len(t, validated, 2)
	assert.Equal(t, sourceA, validated[0].SourceGroup)
	assert.Equal(t, boundA.Id, validated[0].ChannelID)
	assert.Equal(t, sourceB, validated[1].SourceGroup)
	assert.Equal(t, candidateB.Id, validated[1].ChannelID)
}
