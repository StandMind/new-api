package openluxsync

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

var mutationMu sync.Mutex

func Apply(ctx context.Context, request ApplyRequest) (*ApplyResponse, error) {
	if err := validateApplyRequest(request); err != nil {
		return nil, err
	}
	source, err := fetchSource(ctx)
	if err != nil {
		return nil, err
	}

	mutationMu.Lock()
	defer mutationMu.Unlock()

	var response *ApplyResponse
	err = model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		snapshot, loadErr := loadSnapshot(tx, true)
		if loadErr != nil {
			return loadErr
		}
		preview, previewErr := buildPreview(tx, source, snapshot, true)
		if previewErr != nil {
			return previewErr
		}
		if snapshot.BindingRevision != request.BindingRevision || preview.BindingRevision != request.BindingRevision {
			return conflict("OpenLux 绑定已变化，请重新检查")
		}
		if source.Hash != request.SourceHash || preview.SourceHash != request.SourceHash {
			return conflict("OpenLux 上游定价已变化，请重新检查")
		}
		if preview.LocalFingerprint != request.LocalFingerprint {
			return conflict("本地渠道、路由或价格配置已变化，请重新检查")
		}

		selected, selectErr := selectChanges(preview, request.ChangeIDs)
		if selectErr != nil {
			return selectErr
		}
		mutation, result, aggregateErr := aggregateChanges(snapshot, selected)
		if aggregateErr != nil {
			return aggregateErr
		}
		newRevision, applyErr := model.ApplyOpenLuxSyncMutation(tx, mutation)
		if applyErr != nil {
			return applyErr
		}
		result.BindingRevision = newRevision
		result.SourceHash = source.Hash
		response = result
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := model.RefreshAccessPolicyCaches(); err != nil {
		return nil, err
	}
	model.InitChannelCache()
	return response, nil
}

func validateApplyRequest(request ApplyRequest) error {
	if request.BindingRevision < 0 {
		return badRequest("binding_revision 不能为负数")
	}
	if !validSHA256(request.SourceHash) {
		return badRequest("source_hash 无效")
	}
	if !validSHA256(request.LocalFingerprint) {
		return badRequest("local_fingerprint 无效")
	}
	if len(request.ChangeIDs) == 0 {
		return badRequest("至少选择一个可执行变更")
	}
	if len(request.ChangeIDs) > 5000 {
		return badRequest("选择的变更数量过多")
	}
	seen := make(map[string]struct{}, len(request.ChangeIDs))
	for _, id := range request.ChangeIDs {
		if !validSHA256(id) {
			return badRequest("change_ids 包含无效 ID")
		}
		if _, exists := seen[id]; exists {
			return badRequest("change_ids 包含重复 ID")
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func selectChanges(preview *PreviewResponse, requestedIDs []string) ([]Change, error) {
	requested := make(map[string]struct{}, len(requestedIDs))
	for _, id := range requestedIDs {
		requested[id] = struct{}{}
	}
	available := make(map[string]Change, len(preview.Changes))
	for _, change := range preview.Changes {
		available[change.ID] = change
	}
	for id := range requested {
		change, exists := available[id]
		if !exists {
			return nil, conflict("预览变更已不存在，请重新检查")
		}
		if !change.Actionable {
			return nil, unprocessable("所选变更当前不可执行: " + strings.Join(change.BlockedReasons, "；"))
		}
		for _, dependency := range change.Requires {
			if _, selected := requested[dependency]; !selected {
				return nil, unprocessable("所选价格变更缺少必需的模型计费变更")
			}
		}
	}
	selected := make([]Change, 0, len(requested))
	for _, change := range preview.Changes {
		if _, ok := requested[change.ID]; ok {
			selected = append(selected, change)
		}
	}
	return selected, nil
}

func aggregateChanges(snapshot *model.OpenLuxSyncSnapshot, changes []Change) (model.OpenLuxSyncMutation, *ApplyResponse, error) {
	optionMutations := make([]optionMutation, 0)
	abilityByKey := make(map[string]abilityRemoval)
	deleteChannels := make(map[int]struct{})
	deleteBindings := make(map[string]struct{})
	deleteRouteGroups := make(map[string]struct{})
	updatedModels := make(map[string]struct{})
	updatedGroups := make(map[string]struct{})

	for _, change := range changes {
		optionMutations = append(optionMutations, change.mutations...)
		for _, removal := range change.abilityRemovals {
			abilityByKey[associationKey(removal.ChannelID, removal.Group, removal.Model)] = removal
		}
		for _, channelID := range change.deleteChannelIDs {
			deleteChannels[channelID] = struct{}{}
		}
		for _, sourceGroup := range change.deleteBindings {
			deleteBindings[sourceGroup] = struct{}{}
		}
		if change.Kind == "source_group_remove" && change.RemoveMode == "delete_local_group" && change.LocalGroup != "" {
			deleteRouteGroups[change.LocalGroup] = struct{}{}
		}
		if change.Model != "" {
			updatedModels[change.Model] = struct{}{}
		}
		if change.LocalGroup != "" {
			updatedGroups[change.LocalGroup] = struct{}{}
		}
	}

	optionValues, err := aggregateOptionMutations(snapshot.Options, optionMutations)
	if err != nil {
		return model.OpenLuxSyncMutation{}, nil, fmt.Errorf("aggregate OpenLux option changes: %w", err)
	}

	channelByID := make(map[int]model.Channel, len(snapshot.Channels))
	for _, channel := range snapshot.Channels {
		channelByID[channel.Id] = channel
	}
	channelModelLists := make(map[int][]string)
	abilityRemovals := make([]model.OpenLuxAbilityRemoval, 0, len(abilityByKey))
	for _, removal := range abilityByKey {
		if _, deleting := deleteChannels[removal.ChannelID]; deleting {
			continue
		}
		channel, exists := channelByID[removal.ChannelID]
		if !exists {
			return model.OpenLuxSyncMutation{}, nil, conflict(fmt.Sprintf("渠道 #%d 已不存在，请重新检查", removal.ChannelID))
		}
		models, loaded := channelModelLists[removal.ChannelID]
		if !loaded {
			models = splitCSV(channel.Models)
		}
		filtered := models[:0]
		found := false
		for _, modelName := range models {
			if modelName == removal.Model {
				found = true
				continue
			}
			filtered = append(filtered, modelName)
		}
		if !found {
			return model.OpenLuxSyncMutation{}, nil, conflict(fmt.Sprintf("渠道 #%d 的模型列表已变化，请重新检查", removal.ChannelID))
		}
		channelModelLists[removal.ChannelID] = append([]string(nil), filtered...)
		abilityRemovals = append(abilityRemovals, model.OpenLuxAbilityRemoval{
			ChannelID: removal.ChannelID,
			Group:     removal.Group,
			Model:     removal.Model,
		})
	}

	channelModels := make(map[int]string, len(channelModelLists))
	for channelID, models := range channelModelLists {
		channelModels[channelID] = joinCSV(models)
	}
	deleteChannelIDs := intSetValues(deleteChannels)
	deleteBindingGroups := stringSetValues(deleteBindings)
	sort.Slice(abilityRemovals, func(i, j int) bool {
		if abilityRemovals[i].ChannelID != abilityRemovals[j].ChannelID {
			return abilityRemovals[i].ChannelID < abilityRemovals[j].ChannelID
		}
		if abilityRemovals[i].Group != abilityRemovals[j].Group {
			return abilityRemovals[i].Group < abilityRemovals[j].Group
		}
		return abilityRemovals[i].Model < abilityRemovals[j].Model
	})

	mutation := model.OpenLuxSyncMutation{
		Options:                   optionValues,
		ChannelModels:             channelModels,
		AbilityRemovals:           abilityRemovals,
		DeleteChannelIDs:          deleteChannelIDs,
		DeleteBindingSourceGroups: deleteBindingGroups,
		DeleteRouteGroupCodes:     stringSetValues(deleteRouteGroups),
	}
	response := &ApplyResponse{
		AppliedCount:    len(changes),
		UpdatedModels:   stringSetValues(updatedModels),
		UpdatedGroups:   stringSetValues(updatedGroups),
		RemovedChannels: deleteChannelIDs,
	}
	return mutation, response, nil
}

func intSetValues(values map[int]struct{}) []int {
	result := make([]int, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}

func stringSetValues(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
