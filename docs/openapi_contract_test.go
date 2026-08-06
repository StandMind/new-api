package docs

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

type openAPIDocument struct {
	Paths      map[string]map[string]any `json:"paths"`
	Components struct {
		Schemas map[string]map[string]any `json:"schemas"`
	} `json:"components"`
}

func loadOpenAPIDocument(t *testing.T, path string) openAPIDocument {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	var document openAPIDocument
	require.NoError(t, common.Unmarshal(content, &document))
	return document
}

func schemaProperties(document openAPIDocument, name string) map[string]any {
	properties := make(map[string]any)
	var visit func(map[string]any)
	visit = func(schema map[string]any) {
		if ref, ok := schema["$ref"].(string); ok {
			const prefix = "#/components/schemas/"
			if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
				visit(document.Components.Schemas[ref[len(prefix):]])
			}
		}
		if current, ok := schema["properties"].(map[string]any); ok {
			for key, value := range current {
				properties[key] = value
			}
		}
		if allOf, ok := schema["allOf"].([]any); ok {
			for _, item := range allOf {
				if nested, ok := item.(map[string]any); ok {
					visit(nested)
				}
			}
		}
	}
	visit(document.Components.Schemas[name])
	return properties
}

func responseSchemaRef(t *testing.T, document openAPIDocument, path, method string) string {
	t.Helper()
	operation := document.Paths[path][method].(map[string]any)
	responses := operation["responses"].(map[string]any)
	okResponse := responses["200"].(map[string]any)
	content := okResponse["content"].(map[string]any)
	applicationJSON := content["application/json"].(map[string]any)
	schema := applicationJSON["schema"].(map[string]any)
	return schema["$ref"].(string)
}

func TestManagementOpenAPISeparatesUserAndAdminChannelFields(t *testing.T) {
	document := loadOpenAPIDocument(t, "openapi/api.json")

	for _, schemaName := range []string{"UserLog", "UserTask", "UserMidjourneyTask", "UserQuotaData", "UserFlowQuotaData"} {
		properties := schemaProperties(document, schemaName)
		require.NotContains(t, properties, "channel")
		require.NotContains(t, properties, "channel_id")
		require.NotContains(t, properties, "channel_name")
	}
	require.Contains(t, schemaProperties(document, "AdminLog"), "channel")
	require.Contains(t, schemaProperties(document, "AdminTask"), "channel_id")
	require.Contains(t, schemaProperties(document, "AdminMidjourneyTask"), "channel_id")
	require.Contains(t, schemaProperties(document, "AdminFlowQuotaData"), "channel_id")

	require.Equal(t, "#/components/schemas/UserLogPageResponse", responseSchemaRef(t, document, "/api/log/self", "get"))
	require.Equal(t, "#/components/schemas/UserLogListResponse", responseSchemaRef(t, document, "/api/log/token", "get"))
	require.Equal(t, "#/components/schemas/UserTaskPageResponse", responseSchemaRef(t, document, "/api/task/self", "get"))
	require.Equal(t, "#/components/schemas/UserMidjourneyPageResponse", responseSchemaRef(t, document, "/api/mj/self", "get"))
	require.Equal(t, "#/components/schemas/UserQuotaDataListResponse", responseSchemaRef(t, document, "/api/data/self", "get"))
	require.Equal(t, "#/components/schemas/UserFlowQuotaListResponse", responseSchemaRef(t, document, "/api/data/flow/self", "get"))
	require.Equal(t, "#/components/schemas/AdminFlowQuotaListResponse", responseSchemaRef(t, document, "/api/data/flow", "get"))
}

func TestRelayOpenAPIUserTasksExcludeChannelFields(t *testing.T) {
	document := loadOpenAPIDocument(t, "openapi/relay.json")

	for _, schemaName := range []string{"UserAsyncTask", "UserMidjourneyTask", "VideoTaskResponse"} {
		properties := schemaProperties(document, schemaName)
		require.NotContains(t, properties, "channel")
		require.NotContains(t, properties, "channel_id")
		require.NotContains(t, properties, "channel_name")
	}
	require.Equal(t, "#/components/schemas/UserAsyncTaskResponse", responseSchemaRef(t, document, "/suno/fetch/{id}", "get"))
	require.Equal(t, "#/components/schemas/UserMidjourneyTask", responseSchemaRef(t, document, "/mj/task/{id}/fetch", "get"))
}
