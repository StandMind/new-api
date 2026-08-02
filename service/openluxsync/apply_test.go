package openluxsync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSelectChangesRequiresCurrentServerGeneratedIDsAndDependencies(t *testing.T) {
	billingID := strings.Repeat("1", 64)
	priceID := strings.Repeat("2", 64)
	blockedID := strings.Repeat("3", 64)
	preview := &PreviewResponse{Changes: []Change{
		{ID: billingID, Kind: "model_billing_update", Model: "model", Actionable: true},
		{ID: priceID, Kind: "group_model_price_update", Model: "model", Actionable: true, Requires: []string{billingID}},
		{ID: blockedID, Kind: "group_model_remove", Model: "blocked", BlockedReasons: []string{"route reference"}},
	}}

	_, err := selectChanges(preview, []string{priceID})
	requireServiceError(t, err, http.StatusUnprocessableEntity, "blocked_change")

	selected, err := selectChanges(preview, []string{priceID, billingID})
	require.NoError(t, err)
	require.Len(t, selected, 2)
	assert.Equal(t, billingID, selected[0].ID)
	assert.Equal(t, priceID, selected[1].ID)

	_, err = selectChanges(preview, []string{strings.Repeat("4", 64)})
	requireServiceError(t, err, http.StatusConflict, "stale_preview")
	_, err = selectChanges(preview, []string{blockedID})
	requireServiceError(t, err, http.StatusUnprocessableEntity, "blocked_change")
}

func TestValidateApplyRequestRejectsMalformedOrDuplicateChangeIDs(t *testing.T) {
	validHash := strings.Repeat("a", 64)
	validID := strings.Repeat("b", 64)
	require.NoError(t, validateApplyRequest(ApplyRequest{
		SourceHash: validHash, LocalFingerprint: validHash, ChangeIDs: []string{validID},
	}))

	tests := []ApplyRequest{
		{SourceHash: "not-a-hash", LocalFingerprint: validHash, ChangeIDs: []string{validID}},
		{SourceHash: validHash, LocalFingerprint: strings.ToUpper(validHash), ChangeIDs: []string{validID}},
		{SourceHash: validHash, LocalFingerprint: validHash},
		{SourceHash: validHash, LocalFingerprint: validHash, ChangeIDs: []string{validID, validID}},
	}
	for _, request := range tests {
		err := validateApplyRequest(request)
		requireServiceError(t, err, http.StatusBadRequest, "invalid_request")
	}
}

func installApplyTestState(t *testing.T, db *gorm.DB, sourceResponse []byte) {
	t.Helper()
	originalDB := model.DB
	originalClient := pricingHTTPClient
	originalMemoryCache := common.MemoryCacheEnabled
	originalModelRatio := ratio_setting.ModelRatio2JSONString()
	originalGroupRatio := ratio_setting.GroupRatio2JSONString()
	originalGroupModelRatio := ratio_setting.GroupModelRatio2JSONString()
	t.Cleanup(func() {
		model.DB = originalDB
		pricingHTTPClient = originalClient
		common.MemoryCacheEnabled = originalMemoryCache
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(originalModelRatio))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatio))
		require.NoError(t, ratio_setting.UpdateGroupModelRatioByJSONString(originalGroupModelRatio))
	})

	model.DB = db
	common.MemoryCacheEnabled = false
	pricingHTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		assert.Equal(t, Endpoint, request.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(sourceResponse))),
			Header:     make(http.Header),
		}, nil
	})}
}

func seedApplyPriceState(t *testing.T, db *gorm.DB, sourceGroup, localGroup, modelName string, currentRatio float64) model.Channel {
	t.Helper()
	bound := createReferenceChannel(t, db, "OpenLux/"+sourceGroup, localGroup, modelName, "https://api.openlux.ai")
	require.NoError(t, db.Create(&model.Ability{
		ChannelId: bound.Id, Group: localGroup, Model: modelName, Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&model.OpenLuxPriceSyncState{ID: 1, BindingRevision: 7}).Error)
	require.NoError(t, db.Create(&model.OpenLuxPriceSyncBinding{
		SourceGroup: sourceGroup, ChannelID: bound.Id,
	}).Error)
	options := map[string]string{
		optionModelRatio: optionJSON(t, map[string]float64{modelName: 1}),
		optionGroupRatio: optionJSON(t, map[string]float64{localGroup: 1}),
		optionGroupModelRatio: optionJSON(t, map[string]map[string]float64{
			localGroup: {modelName: currentRatio},
		}),
	}
	for key, value := range options {
		require.NoError(t, db.Create(&model.Option{Key: key, Value: value}).Error)
	}
	return bound
}

func TestConcurrentApplyAllowsOnlyOneCommit(t *testing.T) {
	const (
		sourceGroup = "OpenLux-Source"
		localGroup  = "apply-local"
		modelName   = "openlux-sync-concurrent-model"
	)
	db := openLuxTestDB(t)
	seedApplyPriceState(t, db, sourceGroup, localGroup, modelName, 1)
	body := sourceBody(t, []sourceModel{{
		Name: modelName, QuotaType: 0, ModelRatio: decimal.NewFromInt(2), EnableGroups: []string{sourceGroup},
	}}, map[string]decimal.Decimal{sourceGroup: decimal.NewFromInt(1)})
	installApplyTestState(t, db, body)

	preview, err := Preview(context.Background())
	require.NoError(t, err)
	change := findPreviewChange(t, preview, "group_model_price_update", modelName)
	request := ApplyRequest{
		SourceHash:       preview.SourceHash,
		LocalFingerprint: preview.LocalFingerprint,
		BindingRevision:  preview.BindingRevision,
		ChangeIDs:        []string{change.ID},
	}

	type applyResult struct {
		response *ApplyResponse
		err      error
	}
	results := make(chan applyResult, 2)
	for range 2 {
		go func() {
			response, applyErr := Apply(context.Background(), request)
			results <- applyResult{response: response, err: applyErr}
		}()
	}
	first := <-results
	second := <-results
	close(results)

	resultList := []applyResult{first, second}
	successCount := 0
	conflictCount := 0
	for _, result := range resultList {
		if result.err == nil {
			successCount++
			require.NotNil(t, result.response)
			assert.Equal(t, 1, result.response.AppliedCount)
			continue
		}
		var serviceErr *ServiceError
		require.ErrorAs(t, result.err, &serviceErr)
		if serviceErr.Status == http.StatusConflict {
			conflictCount++
		}
	}
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, conflictCount)

	var option model.Option
	require.NoError(t, db.First(&option, "key = ?", optionGroupModelRatio).Error)
	var ratios map[string]map[string]json.RawMessage
	require.NoError(t, common.UnmarshalJsonStr(option.Value, &ratios))
	assert.Equal(t, "2.6", strings.Trim(string(ratios[localGroup][modelName]), `"`))

	var modelRatioOption model.Option
	require.NoError(t, db.First(&modelRatioOption, "key = ?", optionModelRatio).Error)
	assert.JSONEq(t, optionJSON(t, map[string]float64{modelName: 1}), modelRatioOption.Value)

	nextPreview, err := Preview(context.Background())
	require.NoError(t, err)
	for _, nextChange := range nextPreview.Changes {
		assert.False(t,
			nextChange.Kind == "group_model_price_update" && nextChange.Model == modelName,
			"an unchanged OpenLux price must not drift on the next preview",
		)
	}
}

func TestApplyRollsBackOptionsChannelsAndAbilitiesOnWriteFailure(t *testing.T) {
	const (
		sourceGroup = "OpenLux-Source"
		localGroup  = "rollback-local"
		modelName   = "openlux-sync-rollback-model"
	)
	db := openLuxTestDB(t)
	bound := seedApplyPriceState(t, db, sourceGroup, localGroup, modelName, 2)
	body := sourceBody(t, []sourceModel{{
		Name: modelName, QuotaType: 0, ModelRatio: decimal.NewFromInt(1), EnableGroups: []string{"another-source"},
	}}, map[string]decimal.Decimal{sourceGroup: decimal.NewFromInt(1)})
	installApplyTestState(t, db, body)

	preview, err := Preview(context.Background())
	require.NoError(t, err)
	change := findPreviewChange(t, preview, "group_model_remove", modelName)
	require.True(t, change.Actionable)
	require.NoError(t, db.Exec(`
		CREATE TRIGGER fail_openlux_channel_model_update
		BEFORE UPDATE OF models ON channels
		BEGIN
			SELECT RAISE(ABORT, 'forced OpenLux rollback');
		END
	`).Error)

	_, err = Apply(context.Background(), ApplyRequest{
		SourceHash:       preview.SourceHash,
		LocalFingerprint: preview.LocalFingerprint,
		BindingRevision:  preview.BindingRevision,
		ChangeIDs:        []string{change.ID},
	})
	require.Error(t, err)

	var option model.Option
	require.NoError(t, db.First(&option, "key = ?", optionGroupModelRatio).Error)
	assert.JSONEq(t, optionJSON(t, map[string]map[string]float64{
		localGroup: {modelName: 2},
	}), option.Value)

	var persistedChannel model.Channel
	require.NoError(t, db.First(&persistedChannel, "id = ?", bound.Id).Error)
	assert.Equal(t, modelName, persistedChannel.Models)
	var abilityCount int64
	require.NoError(t, db.Model(&model.Ability{}).
		Where("channel_id = ? AND model = ?", bound.Id, modelName).
		Count(&abilityCount).Error)
	assert.EqualValues(t, 1, abilityCount)
}
