package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"

	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsQuotaSaturation(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*dto.UserLog{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}

func TestFormatUserLogsOmitsChannelMetadataAndNormalizesRouting(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price":      0.004,
		"routing_mode":     "fixed_channel",
		"group_chain":      []string{"primary", "fallback"},
		"attempted_groups": []string{"primary"},
		"final_group":      "primary",
		"channel_id":       9411,
		"channel_name":     "openlux-secret-channel",
		"channel_type":     1,
		"unknown":          "must-not-be-public",
		"billing_ratios": map[string]interface{}{
			"image_resolution:4K": 1.79,
			"channel_id":          9411,
		},
		"op": map[string]interface{}{
			"action": "login",
			"params": map[string]interface{}{
				"method":       "password",
				"channel_name": "openlux-secret-channel",
			},
		},
		"admin_info": map[string]interface{}{
			"use_channel": []int{9411, 9412},
			"routing": map[string]interface{}{
				"final_channel_id":   9411,
				"final_channel_name": "openlux-secret-channel",
			},
		},
	})
	logs := []*dto.UserLog{{Id: 99, Other: other}}

	formatUserLogs(logs, 20)

	require.Equal(t, 21, logs[0].Id)
	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	require.Equal(t, "manual", parsed["routing_mode"])
	require.Equal(t, []interface{}{"primary", "fallback"}, parsed["group_chain"])
	require.Equal(t, "primary", parsed["final_group"])
	require.NotContains(t, parsed, "channel_id")
	require.NotContains(t, parsed, "channel_name")
	require.NotContains(t, parsed, "channel_type")
	require.NotContains(t, parsed, "admin_info")
	require.NotContains(t, parsed, "unknown")
	require.NotContains(t, logs[0].Other, "openlux-secret-channel")
	require.NotContains(t, logs[0].Other, "9411")

	ratios := parsed["billing_ratios"].(map[string]interface{})
	require.Contains(t, ratios, "image_resolution:4K")
	require.NotContains(t, ratios, "channel_id")
	params := parsed["op"].(map[string]interface{})["params"].(map[string]interface{})
	require.Equal(t, "password", params["method"])
	require.NotContains(t, params, "channel_name")
}

func TestSanitizeUserLogOtherDropsMalformedAndStructuredTypeMismatches(t *testing.T) {
	malformed, _ := sanitizeUserLogOther(`{"model_price":`)
	wrongType, _ := sanitizeUserLogOther(`{"group_chain":{"channel_id":1}}`)
	allowed, _ := sanitizeUserLogOther(`{"model_price":1,"admin_info":{"routing":{"channel_id":1}}}`)
	require.JSONEq(t, `{}`, malformed)
	require.JSONEq(t, `{}`, wrongType)
	require.JSONEq(t, `{"model_price":1}`, allowed)
}

func TestSanitizeUserLogOperationUsesPublicActionAndParamWhitelist(t *testing.T) {
	login, replacement := sanitizeUserLogOther(`{"op":{"action":"login","params":{"method":"passkey","route":"/api/channel/1","channel_id":99403}}}`)
	require.JSONEq(t, `{"op":{"action":"login","params":{"method":"passkey"}}}`, login)
	require.Empty(t, replacement)

	channelAudit, replacement := sanitizeUserLogOther(`{"op":{"action":"channel.update","params":{"id":99403,"name":"openlux-secret-channel"}}}`)
	require.JSONEq(t, `{}`, channelAudit)
	require.Equal(t, constant.UserLogContentHidden, replacement)

	unknownAction, replacement := sanitizeUserLogOther(`{"op":{"action":"future.action","params":{"safe":"value"}}}`)
	require.JSONEq(t, `{}`, unknownAction)
	require.Equal(t, constant.UserLogContentHidden, replacement)
}

func TestFormatUserLogsHidesNonPublicOperationContent(t *testing.T) {
	logs := []*dto.UserLog{{
		Content: "Updated channel openlux-secret-channel (ID: 99403)",
		Other:   `{"op":{"action":"channel.update","params":{"id":99403,"name":"openlux-secret-channel"}}}`,
	}}

	formatUserLogs(logs, 0)

	require.Equal(t, constant.UserLogContentHidden, logs[0].Content)
	require.NotContains(t, logs[0].Other, "openlux-secret-channel")
	require.NotContains(t, logs[0].Other, "99403")
}

func TestUserLogQueriesProjectOutChannelColumns(t *testing.T) {
	const (
		userID    = 99401
		tokenID   = 99402
		channelID = 99403
	)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:      userID,
		CreatedAt:   123456789,
		Type:        LogTypeConsume,
		TokenId:     tokenID,
		ChannelId:   channelID,
		ChannelName: "openlux-secret-channel",
		ModelName:   "user-log-projection-test",
		Group:       "visible-group",
		Other:       `{"channel_id":99403,"final_group":"visible-group"}`,
	}).Error)

	userLogs, total, err := GetUserLogs(userID, LogTypeConsume, 0, 0, "user-log-projection-test", "", 0, 10, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, userLogs, 1)
	encoded, err := common.Marshal(userLogs[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"channel"`)
	require.NotContains(t, string(encoded), "channel_name")
	require.NotContains(t, string(encoded), "99403")
	require.Contains(t, string(encoded), "visible-group")

	tokenLogs, err := GetLogByTokenId(tokenID)
	require.NoError(t, err)
	require.Len(t, tokenLogs, 1)
	encoded, err = common.Marshal(tokenLogs[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"channel"`)
	require.NotContains(t, string(encoded), "99403")

	adminLogs, total, err := GetAllLogs(LogTypeConsume, 0, 0, "user-log-projection-test", "", "", 0, 10, channelID, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, adminLogs, 1)
	require.Equal(t, channelID, adminLogs[0].ChannelId)
}

func TestFormatUserLogsSanitizesInternalRouteErrorContent(t *testing.T) {
	const (
		userID        = 99501
		channelID     = 99502
		modelName     = "user-log-route-error-test"
		secretContent = "status_code=500, channel ID is 0: openlux-secret-channel"
	)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    userID,
		CreatedAt: 123456790,
		Type:      LogTypeError,
		Content:   secretContent,
		ModelName: modelName,
		ChannelId: channelID,
		Group:     "primary",
		Other:     `{"error_code":"channel:no_available_key","final_group":"primary"}`,
	}).Error)

	userLogs, _, err := GetUserLogs(userID, LogTypeError, 0, 0, modelName, "", 0, 10, "", "", "")
	require.NoError(t, err)
	require.Len(t, userLogs, 1)
	require.Equal(t, constant.TaskFailReasonRouteUnavailable, userLogs[0].Content)
	require.NotContains(t, userLogs[0].Other, "openlux")
	parsed, err := common.StrToMap(userLogs[0].Other)
	require.NoError(t, err)
	require.Equal(t, "route_unavailable", parsed["error_code"])

	adminLogs, _, err := GetAllLogs(LogTypeError, 0, 0, modelName, "", "", 0, 10, channelID, "", "", "")
	require.NoError(t, err)
	require.Len(t, adminLogs, 1)
	require.Equal(t, secretContent, adminLogs[0].Content)
	require.Equal(t, channelID, adminLogs[0].ChannelId)
}

func legacyFormatUserLogsForBenchmark(logs []*dto.UserLog) {
	for i := range logs {
		other, _ := common.StrToMap(logs[i].Other)
		if other != nil {
			delete(other, "admin_info")
			delete(other, "audit_info")
			delete(other, "stream_status")
		}
		logs[i].Other = common.MapToJsonStr(other)
	}
}

func benchmarkFormatUserLogs(b *testing.B, count int, withRoutingSnapshot bool, legacy bool) {
	base := map[string]interface{}{
		"model_price":      0.004,
		"group_ratio":      1.2,
		"routing_mode":     "price",
		"group_chain":      []string{"primary", "fallback"},
		"attempted_groups": []string{"primary", "fallback"},
		"final_group":      "fallback",
	}
	if withRoutingSnapshot {
		attempts := make([]map[string]interface{}, 256)
		for i := range attempts {
			attempts[i] = map[string]interface{}{
				"channel_id": 9000 + i,
				"group":      fmt.Sprintf("group-%d", i%4),
				"error":      "upstream unavailable",
			}
		}
		base["admin_info"] = map[string]interface{}{"routing": map[string]interface{}{"attempts": attempts}}
	}
	raw := common.MapToJsonStr(base)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logs := make([]*dto.UserLog, count)
		for j := range logs {
			logs[j] = &dto.UserLog{Other: raw}
		}
		if legacy {
			legacyFormatUserLogsForBenchmark(logs)
		} else {
			formatUserLogs(logs, 0)
		}
	}
}

func BenchmarkFormatUserLogs100(b *testing.B) {
	b.Run("current", func(b *testing.B) { benchmarkFormatUserLogs(b, 100, false, false) })
	b.Run("legacy", func(b *testing.B) { benchmarkFormatUserLogs(b, 100, false, true) })
}

func BenchmarkFormatUserLogs1000(b *testing.B) {
	b.Run("current", func(b *testing.B) { benchmarkFormatUserLogs(b, 1000, false, false) })
	b.Run("legacy", func(b *testing.B) { benchmarkFormatUserLogs(b, 1000, false, true) })
}

func BenchmarkFormatUserLogsLargeFailure(b *testing.B) {
	b.Run("current", func(b *testing.B) { benchmarkFormatUserLogs(b, 100, true, false) })
	b.Run("legacy", func(b *testing.B) { benchmarkFormatUserLogs(b, 100, true, true) })
}

func BenchmarkSanitizeUserLogOtherParallel(b *testing.B) {
	raw := `{"model_price":0.004,"routing_mode":"price","group_chain":["primary","fallback"],"admin_info":{"routing":{"final_channel_id":9411}}}`
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = sanitizeUserLogOther(raw)
		}
	})
}

func FuzzSanitizeUserLogOtherNeverEmitsChannelKeys(f *testing.F) {
	f.Add(`{"channel_id":1,"routing_mode":"fixed_channel","final_group":"default"}`)
	f.Add(`{"admin_info":{"routing":{"final_channel_name":"secret"}},"model_price":1}`)
	f.Add(`{"op":{"action":"login","params":{"channelIds":[1,2],"method":"password"}}}`)

	f.Fuzz(func(t *testing.T, input string) {
		output, _ := sanitizeUserLogOther(input)
		if output == "" {
			return
		}
		var value any
		require.NoError(t, common.Unmarshal([]byte(output), &value))
		var inspect func(any)
		inspect = func(current any) {
			switch typed := current.(type) {
			case map[string]any:
				for key, nested := range typed {
					normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
					require.NotContains(t, normalized, "channel")
					require.NotEqual(t, "admininfo", normalized)
					inspect(nested)
				}
			case []any:
				for _, nested := range typed {
					inspect(nested)
				}
			}
		}
		inspect(value)
		require.NotContains(t, output, `"fixed_channel"`)
	})
}
