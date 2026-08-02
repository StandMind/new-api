package openluxsync

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

func GetBindings(ctx context.Context) (*BindingsResponse, error) {
	source, err := fetchSource(ctx)
	if err != nil {
		return nil, err
	}
	snapshot, err := loadSnapshot(model.DB, false)
	if err != nil {
		return nil, err
	}
	return buildBindingsResponse(source, snapshot), nil
}

func SaveBindings(ctx context.Context, request SaveBindingsRequest) (*BindingsResponse, error) {
	if request.ExpectedRevision < 0 {
		return nil, badRequest("expected_revision 不能为负数")
	}
	source, err := fetchSource(ctx)
	if err != nil {
		return nil, err
	}
	var savedSnapshot *model.OpenLuxSyncSnapshot
	mutationMu.Lock()
	defer mutationMu.Unlock()
	err = model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		snapshot, loadErr := loadSnapshot(tx, true)
		if loadErr != nil {
			return loadErr
		}
		bindings, validateErr := validateBindings(request.Bindings, source, snapshot)
		if validateErr != nil {
			return validateErr
		}
		newRevision, replaceErr := model.ReplaceOpenLuxPriceSyncBindings(tx, request.ExpectedRevision, bindings)
		if errors.Is(replaceErr, model.ErrOpenLuxPriceSyncRevisionConflict) {
			return conflict("OpenLux 绑定已被其他操作修改，请刷新后重试")
		}
		if replaceErr != nil {
			return replaceErr
		}
		snapshot.BindingRevision = newRevision
		snapshot.Bindings = bindings
		savedSnapshot = snapshot
		return nil
	})
	if err != nil {
		return nil, err
	}
	return buildBindingsResponse(source, savedSnapshot), nil
}

func validateBindings(requests []SaveBinding, source *sourceSnapshot, snapshot *model.OpenLuxSyncSnapshot) ([]model.OpenLuxPriceSyncBinding, error) {
	channelByID := make(map[int]model.Channel, len(snapshot.Channels))
	for _, channel := range snapshot.Channels {
		channelByID[channel.Id] = channel
	}
	groupRatios, err := parseNumberMap(snapshot.Options[optionGroupRatio], optionGroupRatio)
	if err != nil {
		return nil, unprocessable(err.Error())
	}
	seenSources := make(map[string]struct{}, len(requests))
	seenChannels := make(map[int]struct{})
	seenLocalGroups := make(map[string]string)
	result := make([]model.OpenLuxPriceSyncBinding, 0)
	for _, request := range requests {
		sourceGroup := strings.TrimSpace(request.SourceGroup)
		if sourceGroup == "" || sourceGroup == "default" {
			return nil, badRequest("不能绑定空来源组或 default")
		}
		if _, ok := source.Groups[sourceGroup]; !ok {
			return nil, badRequest("OpenLux 来源组不存在: " + sourceGroup)
		}
		if _, duplicate := seenSources[sourceGroup]; duplicate {
			return nil, badRequest("来源组重复: " + sourceGroup)
		}
		seenSources[sourceGroup] = struct{}{}
		if len(request.ChannelIDs) == 0 {
			return nil, badRequest("来源组 " + sourceGroup + " 至少需要一个渠道")
		}
		localGroup := ""
		for _, channelID := range request.ChannelIDs {
			if _, duplicate := seenChannels[channelID]; duplicate {
				return nil, badRequest(fmt.Sprintf("渠道 #%d 被重复绑定", channelID))
			}
			channel, ok := channelByID[channelID]
			if !ok {
				return nil, badRequest(fmt.Sprintf("渠道 #%d 不存在", channelID))
			}
			baseURL := ""
			if channel.BaseURL != nil {
				baseURL = *channel.BaseURL
			}
			if !isOpenLuxBaseURL(baseURL) {
				return nil, badRequest(fmt.Sprintf("渠道 #%d 不是 OpenLux 渠道", channelID))
			}
			groups := splitCSV(channel.Group)
			if len(groups) != 1 || groups[0] == "default" {
				return nil, badRequest(fmt.Sprintf("渠道 #%d 必须属于一个非 default 本地组", channelID))
			}
			if localGroup == "" {
				localGroup = groups[0]
			} else if localGroup != groups[0] {
				return nil, badRequest("同一来源组的绑定渠道必须属于同一本地组")
			}
			seenChannels[channelID] = struct{}{}
			result = append(result, model.OpenLuxPriceSyncBinding{SourceGroup: sourceGroup, ChannelID: channelID})
		}
		if _, exists := groupRatios[localGroup]; !exists {
			return nil, badRequest("本地组未配置 GroupRatio: " + localGroup)
		}
		if previous, duplicate := seenLocalGroups[localGroup]; duplicate && previous != sourceGroup {
			return nil, badRequest("本地组 " + localGroup + " 不能同时绑定多个 OpenLux 来源组")
		}
		seenLocalGroups[localGroup] = sourceGroup
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SourceGroup != result[j].SourceGroup {
			return result[i].SourceGroup < result[j].SourceGroup
		}
		return result[i].ChannelID < result[j].ChannelID
	})
	return result, nil
}

func buildBindingsResponse(source *sourceSnapshot, snapshot *model.OpenLuxSyncSnapshot) *BindingsResponse {
	response := &BindingsResponse{BindingRevision: snapshot.BindingRevision}
	boundSourceByChannel := make(map[int]string, len(snapshot.Bindings))
	bindingsBySource := make(map[string][]int)
	for _, binding := range snapshot.Bindings {
		boundSourceByChannel[binding.ChannelID] = binding.SourceGroup
		bindingsBySource[binding.SourceGroup] = append(bindingsBySource[binding.SourceGroup], binding.ChannelID)
	}
	channelByID := make(map[int]model.Channel, len(snapshot.Channels))
	for _, channel := range snapshot.Channels {
		channelByID[channel.Id] = channel
		baseURL := ""
		if channel.BaseURL != nil {
			baseURL = *channel.BaseURL
		}
		if !isOpenLuxBaseURL(baseURL) {
			continue
		}
		response.OpenLuxChannels = append(response.OpenLuxChannels, summarizeChannel(channel, boundSourceByChannel[channel.Id]))
	}
	sort.Slice(response.OpenLuxChannels, func(i, j int) bool { return response.OpenLuxChannels[i].ID < response.OpenLuxChannels[j].ID })

	sources := make([]string, 0, len(bindingsBySource))
	for sourceGroup := range bindingsBySource {
		sources = append(sources, sourceGroup)
	}
	sort.Strings(sources)
	for _, sourceGroup := range sources {
		response.Bindings = append(response.Bindings, bindingView(sourceGroup, bindingsBySource[sourceGroup], channelByID, snapshot.Channels, false))
	}

	candidateGroups := make(map[string]map[string][]int)
	for _, channel := range snapshot.Channels {
		if _, bound := boundSourceByChannel[channel.Id]; bound {
			continue
		}
		baseURL := ""
		if channel.BaseURL != nil {
			baseURL = *channel.BaseURL
		}
		if !isOpenLuxBaseURL(baseURL) {
			continue
		}
		groups := splitCSV(channel.Group)
		if len(groups) != 1 || groups[0] == "default" {
			continue
		}
		candidateSource := ""
		if _, ok := source.Groups[groups[0]]; ok && groups[0] != "default" {
			candidateSource = groups[0]
		}
		for _, prefix := range []string{"OpenLux/", "OpenLux-Ext/"} {
			if strings.HasPrefix(channel.Name, prefix) {
				nameSource := strings.TrimSpace(strings.TrimPrefix(channel.Name, prefix))
				if _, ok := source.Groups[nameSource]; ok && nameSource != "default" {
					candidateSource = nameSource
				}
			}
		}
		if candidateSource == "" {
			continue
		}
		if _, alreadyBound := bindingsBySource[candidateSource]; alreadyBound {
			continue
		}
		if candidateGroups[candidateSource] == nil {
			candidateGroups[candidateSource] = make(map[string][]int)
		}
		candidateGroups[candidateSource][groups[0]] = append(candidateGroups[candidateSource][groups[0]], channel.Id)
	}
	candidateSources := make([]string, 0, len(candidateGroups))
	for sourceGroup := range candidateGroups {
		candidateSources = append(candidateSources, sourceGroup)
	}
	sort.Strings(candidateSources)
	for _, sourceGroup := range candidateSources {
		localGroups := make([]string, 0, len(candidateGroups[sourceGroup]))
		for localGroup := range candidateGroups[sourceGroup] {
			localGroups = append(localGroups, localGroup)
		}
		sort.Strings(localGroups)
		for _, localGroup := range localGroups {
			response.Candidates = append(response.Candidates, bindingView(sourceGroup, candidateGroups[sourceGroup][localGroup], channelByID, snapshot.Channels, true))
		}
	}

	groupNames := make([]string, 0, len(source.Groups))
	for name := range source.Groups {
		if name != "default" {
			groupNames = append(groupNames, name)
		}
	}
	sort.Strings(groupNames)
	for _, name := range groupNames {
		group := source.Groups[name]
		response.SourceGroups = append(response.SourceGroups, SourceGroupView{Name: name, Description: group.Description})
	}
	return response
}

func bindingView(sourceGroup string, channelIDs []int, channelByID map[int]model.Channel, allChannels []model.Channel, candidate bool) BindingView {
	view := BindingView{SourceGroup: sourceGroup, Healthy: true, Candidate: candidate}
	localGroups := make(map[string]struct{})
	bound := make(map[int]struct{}, len(channelIDs))
	for _, channelID := range channelIDs {
		bound[channelID] = struct{}{}
		channel, ok := channelByID[channelID]
		if !ok {
			view.Healthy = false
			view.Issues = append(view.Issues, fmt.Sprintf("绑定渠道 #%d 已不存在", channelID))
			continue
		}
		baseURL := ""
		if channel.BaseURL != nil {
			baseURL = *channel.BaseURL
		}
		if !isOpenLuxBaseURL(baseURL) {
			view.Healthy = false
			view.Issues = append(view.Issues, fmt.Sprintf("渠道 #%d 已不再指向 OpenLux", channelID))
		}
		groups := splitCSV(channel.Group)
		if len(groups) != 1 || groups[0] == "default" {
			view.Healthy = false
			view.Issues = append(view.Issues, fmt.Sprintf("渠道 #%d 不属于单一非 default 组", channelID))
		} else {
			localGroups[groups[0]] = struct{}{}
		}
		view.Channels = append(view.Channels, summarizeChannel(channel, sourceGroup))
	}
	if len(localGroups) == 1 {
		for localGroup := range localGroups {
			view.LocalGroup = localGroup
		}
	} else if len(localGroups) > 1 {
		view.Healthy = false
		view.Issues = append(view.Issues, "绑定渠道当前属于不同本地组")
	}
	if view.LocalGroup != "" {
		for _, channel := range allChannels {
			if _, isBound := bound[channel.Id]; isBound {
				continue
			}
			if containsString(splitCSV(channel.Group), view.LocalGroup) {
				view.MixedChannelCount++
			}
		}
	}
	sort.Slice(view.Channels, func(i, j int) bool { return view.Channels[i].ID < view.Channels[j].ID })
	return view
}

func summarizeChannel(channel model.Channel, boundSource string) ChannelSummary {
	localGroup := ""
	groups := splitCSV(channel.Group)
	if len(groups) == 1 {
		localGroup = groups[0]
	}
	return ChannelSummary{
		ID: channel.Id, Name: channel.Name, LocalGroup: localGroup, Status: channel.Status,
		ModelCount: len(splitCSV(channel.Models)), BoundSource: boundSource,
	}
}
