package openluxsync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	optionModelRatio           = "ModelRatio"
	optionModelPrice           = "ModelPrice"
	optionCompletionRatio      = "CompletionRatio"
	optionCacheRatio           = "CacheRatio"
	optionCreateCacheRatio     = "CreateCacheRatio"
	optionImageRatio           = "ImageRatio"
	optionAudioRatio           = "AudioRatio"
	optionAudioCompletionRatio = "AudioCompletionRatio"
	optionGroupModelRatio      = "GroupModelRatio"
	optionBillingMode          = "billing_setting.billing_mode"
	optionBillingExpr          = "billing_setting.billing_expr"
)

var managedOptionKeys = []string{
	optionModelRatio,
	optionModelPrice,
	optionCompletionRatio,
	optionCacheRatio,
	optionCreateCacheRatio,
	optionImageRatio,
	optionAudioRatio,
	optionAudioCompletionRatio,
	optionGroupModelRatio,
	optionBillingMode,
	optionBillingExpr,
}

type numberMap map[string]json.RawMessage
type nestedNumberMap map[string]numberMap
type stringMap map[string]string
type nestedStringMap map[string]map[string]string

func loadSnapshot(db *gorm.DB, forUpdate bool) (*model.OpenLuxSyncSnapshot, error) {
	snapshot, err := model.LoadOpenLuxSyncSnapshot(db, forUpdate, managedOptionKeys)
	if err != nil {
		return nil, err
	}
	fallbacks, err := effectiveOptionFallbacks()
	if err != nil {
		return nil, err
	}
	for _, key := range managedOptionKeys {
		if strings.TrimSpace(snapshot.Options[key]) == "" {
			snapshot.Options[key] = fallbacks[key]
		}
	}
	return snapshot, nil
}

func effectiveOptionFallbacks() (map[string]string, error) {
	values := map[string]any{
		optionModelRatio:           ratio_setting.GetModelRatioCopy(),
		optionModelPrice:           ratio_setting.GetModelPriceCopy(),
		optionCompletionRatio:      ratio_setting.GetCompletionRatioCopy(),
		optionCacheRatio:           ratio_setting.GetCacheRatioCopy(),
		optionCreateCacheRatio:     ratio_setting.GetCreateCacheRatioCopy(),
		optionImageRatio:           ratio_setting.GetImageRatioCopy(),
		optionAudioRatio:           ratio_setting.GetAudioRatioCopy(),
		optionAudioCompletionRatio: ratio_setting.GetAudioCompletionRatioCopy(),
		optionGroupModelRatio:      ratio_setting.GetGroupModelRatioCopy(),
		optionBillingMode:          billing_setting.GetBillingModeCopy(),
		optionBillingExpr:          billing_setting.GetBillingExprCopy(),
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		encoded, err := common.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode fallback option %s: %w", key, err)
		}
		result[key] = string(encoded)
	}
	return result, nil
}

func parseRawOrEmpty(raw string) any {
	var value any
	if err := common.UnmarshalJsonStr(raw, &value); err != nil {
		return map[string]any{}
	}
	return value
}

func parseNumberMap(raw, key string) (numberMap, error) {
	result := make(numberMap)
	if strings.TrimSpace(raw) == "" {
		return result, nil
	}
	if err := common.UnmarshalJsonStr(raw, &result); err != nil {
		return nil, fmt.Errorf("option %s is not a numeric object: %w", key, err)
	}
	return result, nil
}

func parseNestedNumberMap(raw, key string) (nestedNumberMap, error) {
	result := make(nestedNumberMap)
	if strings.TrimSpace(raw) == "" {
		return result, nil
	}
	if err := common.UnmarshalJsonStr(raw, &result); err != nil {
		return nil, fmt.Errorf("option %s is not a nested numeric object: %w", key, err)
	}
	return result, nil
}

func parseStringMap(raw, key string) (stringMap, error) {
	result := make(stringMap)
	if strings.TrimSpace(raw) == "" {
		return result, nil
	}
	if err := common.UnmarshalJsonStr(raw, &result); err != nil {
		return nil, fmt.Errorf("option %s is not a string object: %w", key, err)
	}
	return result, nil
}

func parseNestedStringMap(raw, key string) (nestedStringMap, error) {
	result := make(nestedStringMap)
	if strings.TrimSpace(raw) == "" {
		return result, nil
	}
	if err := common.UnmarshalJsonStr(raw, &result); err != nil {
		return nil, fmt.Errorf("option %s is not a nested string object: %w", key, err)
	}
	return result, nil
}

func decimalFromRaw(value json.RawMessage) (decimal.Decimal, bool) {
	text := strings.TrimSpace(common.JsonRawMessageToString(value))
	if text == "" {
		return decimal.Zero, false
	}
	parsed, err := decimal.NewFromString(text)
	if err != nil {
		return decimal.Zero, false
	}
	return parsed, true
}

func numberMapValue(values numberMap, key string, fallback decimal.Decimal) decimal.Decimal {
	if value, ok := values[key]; ok {
		if parsed, valid := decimalFromRaw(value); valid {
			return parsed
		}
	}
	return fallback
}

func setNumberMapValue(values numberMap, key string, value decimal.Decimal) {
	values[key] = json.RawMessage(value.String())
}

func encodeOption(value any, key string) (string, error) {
	encoded, err := common.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode option %s: %w", key, err)
	}
	return string(encoded), nil
}

type localFingerprintChannel struct {
	ID                 int               `json:"id"`
	Type               int               `json:"type"`
	Name               string            `json:"name"`
	Status             int               `json:"status"`
	Group              string            `json:"group"`
	Models             string            `json:"models"`
	BaseURL            string            `json:"base_url"`
	KeyHash            string            `json:"key_hash"`
	OpenAIOrganization *string           `json:"openai_organization"`
	TestModel          *string           `json:"test_model"`
	Weight             *uint             `json:"weight"`
	Priority           *int64            `json:"priority"`
	Other              string            `json:"other"`
	ModelMapping       *string           `json:"model_mapping"`
	StatusCodeMapping  *string           `json:"status_code_mapping"`
	AutoBan            *int              `json:"auto_ban"`
	OtherInfo          string            `json:"other_info"`
	Tag                *string           `json:"tag"`
	Setting            *string           `json:"setting"`
	ParamOverride      *string           `json:"param_override"`
	HeaderOverride     *string           `json:"header_override"`
	Remark             *string           `json:"remark"`
	ChannelInfo        model.ChannelInfo `json:"channel_info"`
	OtherSettings      string            `json:"settings"`
}

type localFingerprintView struct {
	Bindings    []model.OpenLuxPriceSyncBinding `json:"bindings"`
	RouteGroups []model.RouteGroup              `json:"route_groups"`
	Channels    []localFingerprintChannel       `json:"channels"`
	Abilities   []model.Ability                 `json:"abilities"`
	Routes      []model.GroupModelRoute         `json:"routes"`
	Options     map[string]json.RawMessage      `json:"options"`
}

func localFingerprint(snapshot *model.OpenLuxSyncSnapshot) (string, error) {
	view := localFingerprintView{
		Bindings:    append([]model.OpenLuxPriceSyncBinding(nil), snapshot.Bindings...),
		RouteGroups: append([]model.RouteGroup(nil), snapshot.RouteGroups...),
		Options:     make(map[string]json.RawMessage, len(snapshot.Options)),
	}
	for _, channel := range snapshot.Channels {
		keyDigest := sha256.Sum256([]byte(channel.Key))
		baseURL := ""
		if channel.BaseURL != nil {
			baseURL = *channel.BaseURL
		}
		view.Channels = append(view.Channels, localFingerprintChannel{
			ID: channel.Id, Type: channel.Type, Name: channel.Name, Status: channel.Status,
			Group: channel.Group, Models: channel.Models, BaseURL: baseURL,
			KeyHash: hex.EncodeToString(keyDigest[:]), OpenAIOrganization: channel.OpenAIOrganization,
			TestModel: channel.TestModel, Weight: channel.Weight, Priority: channel.Priority,
			Other: channel.Other, ModelMapping: channel.ModelMapping, StatusCodeMapping: channel.StatusCodeMapping,
			AutoBan: channel.AutoBan, OtherInfo: channel.OtherInfo, Tag: channel.Tag, Setting: channel.Setting,
			ParamOverride: channel.ParamOverride, HeaderOverride: channel.HeaderOverride, Remark: channel.Remark,
			ChannelInfo: channel.ChannelInfo, OtherSettings: channel.OtherSettings,
		})
	}
	view.Abilities = append(view.Abilities, snapshot.Abilities...)
	view.Routes = append(view.Routes, snapshot.Routes...)
	sort.Slice(view.Channels, func(i, j int) bool { return view.Channels[i].ID < view.Channels[j].ID })
	sort.Slice(view.RouteGroups, func(i, j int) bool { return view.RouteGroups[i].Code < view.RouteGroups[j].Code })
	sort.Slice(view.Abilities, func(i, j int) bool {
		if view.Abilities[i].ChannelId != view.Abilities[j].ChannelId {
			return view.Abilities[i].ChannelId < view.Abilities[j].ChannelId
		}
		if view.Abilities[i].Group != view.Abilities[j].Group {
			return view.Abilities[i].Group < view.Abilities[j].Group
		}
		return view.Abilities[i].Model < view.Abilities[j].Model
	})
	for key, raw := range snapshot.Options {
		var normalized any
		if err := common.UnmarshalJsonStr(raw, &normalized); err != nil {
			return "", fmt.Errorf("option %s is invalid JSON: %w", key, err)
		}
		encoded, err := common.Marshal(normalized)
		if err != nil {
			return "", err
		}
		view.Options[key] = encoded
	}
	encoded, err := common.Marshal(view)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func joinCSV(values []string) string {
	return strings.Join(values, ",")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func aggregateOptionMutations(options map[string]string, mutations []optionMutation) (map[string]string, error) {
	byKey := make(map[string][]optionMutation)
	for _, mutation := range mutations {
		byKey[mutation.Key] = append(byKey[mutation.Key], mutation)
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make(map[string]string, len(keys))
	for _, key := range keys {
		var (
			value string
			err   error
		)
		switch key {
		case optionGroupModelRatio:
			value, err = mutateNestedNumberOption(options[key], key, byKey[key])
		case optionBillingMode, optionBillingExpr:
			value, err = mutateStringOption(options[key], key, byKey[key])
		case optionModelRatio, optionModelPrice, optionCompletionRatio, optionCacheRatio,
			optionCreateCacheRatio, optionImageRatio, optionAudioRatio, optionAudioCompletionRatio:
			value, err = mutateNumberOption(options[key], key, byKey[key])
		default:
			err = fmt.Errorf("unsupported OpenLux option mutation key %s", key)
		}
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

func mutateNumberOption(raw, key string, mutations []optionMutation) (string, error) {
	values, err := parseNumberMap(raw, key)
	if err != nil {
		return "", err
	}
	for _, mutation := range mutations {
		switch mutation.Action {
		case "set":
			value, parseErr := decimal.NewFromString(mutation.Value)
			if parseErr != nil || value.IsNegative() {
				return "", fmt.Errorf("invalid %s value for %s", key, mutation.Model)
			}
			setNumberMapValue(values, mutation.Model, value)
		case "delete":
			delete(values, mutation.Model)
		default:
			return "", fmt.Errorf("unsupported %s mutation action %s", key, mutation.Action)
		}
	}
	return encodeOption(values, key)
}

func mutateStringOption(raw, key string, mutations []optionMutation) (string, error) {
	values, err := parseStringMap(raw, key)
	if err != nil {
		return "", err
	}
	for _, mutation := range mutations {
		switch mutation.Action {
		case "set_string":
			values[mutation.Model] = mutation.Value
		case "delete":
			delete(values, mutation.Model)
		default:
			return "", fmt.Errorf("unsupported %s mutation action %s", key, mutation.Action)
		}
	}
	return encodeOption(values, key)
}

func mutateNestedNumberOption(raw, key string, mutations []optionMutation) (string, error) {
	values, err := parseNestedNumberMap(raw, key)
	if err != nil {
		return "", err
	}
	for _, mutation := range mutations {
		switch mutation.Action {
		case "set_nested":
			value, parseErr := decimal.NewFromString(mutation.Value)
			if parseErr != nil || value.IsNegative() {
				return "", fmt.Errorf("invalid %s value for %s/%s", key, mutation.Group, mutation.Model)
			}
			if values[mutation.Group] == nil {
				values[mutation.Group] = make(numberMap)
			}
			setNumberMapValue(values[mutation.Group], mutation.Model, value)
		case "delete_nested":
			if nested := values[mutation.Group]; nested != nil {
				delete(nested, mutation.Model)
				if len(nested) == 0 {
					delete(values, mutation.Group)
				}
			}
		case "delete":
			delete(values, mutation.Model)
		default:
			return "", fmt.Errorf("unsupported %s mutation action %s", key, mutation.Action)
		}
	}
	return encodeOption(values, key)
}

func mutateNestedNumberReferences(raw, key string, mutations []optionMutation) (string, error) {
	values, err := parseNestedNumberMap(raw, key)
	if err != nil {
		return "", err
	}
	for _, mutation := range mutations {
		if mutation.Action != "delete_references" {
			return "", fmt.Errorf("unsupported %s mutation action %s", key, mutation.Action)
		}
		delete(values, mutation.Model)
		for outer, nested := range values {
			delete(nested, mutation.Model)
			if len(nested) == 0 {
				delete(values, outer)
			}
		}
	}
	return encodeOption(values, key)
}

func mutateNestedStringReferences(raw, key string, mutations []optionMutation) (string, error) {
	values, err := parseNestedStringMap(raw, key)
	if err != nil {
		return "", err
	}
	for _, mutation := range mutations {
		if mutation.Action != "delete_references" {
			return "", fmt.Errorf("unsupported %s mutation action %s", key, mutation.Action)
		}
		delete(values, mutation.Model)
		for outer, nested := range values {
			delete(nested, mutation.Model)
			if len(nested) == 0 {
				delete(values, outer)
			}
		}
	}
	return encodeOption(values, key)
}
