package openluxsync

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func sourceBody(t *testing.T, models []sourceModel, groups map[string]decimal.Decimal) []byte {
	t.Helper()
	usableGroups := make(map[string]json.RawMessage, len(groups))
	for group := range groups {
		usableGroups[group] = json.RawMessage(`"` + group + `"`)
	}
	body, err := common.Marshal(pricingResponse{
		Success:     true,
		Data:        models,
		GroupRatio:  groups,
		UsableGroup: usableGroups,
	})
	require.NoError(t, err)
	return body
}

func requireServiceError(t *testing.T, err error, status int, code string) *ServiceError {
	t.Helper()
	var serviceErr *ServiceError
	require.ErrorAs(t, err, &serviceErr)
	assert.Equal(t, status, serviceErr.Status)
	assert.Equal(t, code, serviceErr.Code)
	return serviceErr
}

func TestParseSourceCanonicalizesAndTracksReferencedGroups(t *testing.T) {
	groups := map[string]decimal.Decimal{
		"source-b": decimal.RequireFromString("0.5"),
		"source-a": decimal.RequireFromString("0.25"),
	}
	models := []sourceModel{
		{Name: "model-b", QuotaType: 0, ModelRatio: decimal.NewFromInt(2), EnableGroups: []string{"ghost", "source-b"}},
		{Name: "model-a", QuotaType: 1, ModelPrice: decimal.RequireFromString("0.02"), EnableGroups: []string{"source-a"}},
	}

	first, err := parseSource(sourceBody(t, models, groups))
	require.NoError(t, err)
	second, err := parseSource(sourceBody(t, []sourceModel{models[1], models[0]}, map[string]decimal.Decimal{
		"source-a": groups["source-a"],
		"source-b": groups["source-b"],
	}))
	require.NoError(t, err)

	assert.Equal(t, first.Hash, second.Hash)
	assert.Len(t, first.Hash, 64)
	assert.Equal(t, []string{"model-a", "model-b"}, []string{first.OrderedModels[0].Name, first.OrderedModels[1].Name})
	assert.Contains(t, first.ReferencedGroups, "ghost")
	assert.NotContains(t, first.Groups, "ghost")
}

func TestParseSourceRejectsUnsafeSchemas(t *testing.T) {
	validModel := sourceModel{Name: "model", QuotaType: 0, ModelRatio: decimal.NewFromInt(1), EnableGroups: []string{"source"}}
	validGroups := map[string]decimal.Decimal{"source": decimal.NewFromInt(1)}

	tests := []struct {
		name string
		body []byte
		code string
	}{
		{name: "invalid json", body: []byte(`{`), code: "upstream_json"},
		{name: "reported failure", body: []byte(`{"success":false,"message":"maintenance"}`), code: "upstream_failure"},
		{name: "missing data", body: []byte(`{"success":true,"data":[],"group_ratio":{"source":1},"usable_group":{"source":"source"}}`), code: "upstream_schema"},
		{name: "model group override", body: []byte(`{"success":true,"data":[{"model_name":"model","quota_type":0,"model_ratio":1}],"group_ratio":{"source":1},"group_model_ratio":{"source":{"model":2}},"usable_group":{"source":"source"}}`), code: "unsupported_upstream_schema"},
		{name: "unknown quota type", body: sourceBody(t, []sourceModel{{Name: "model", QuotaType: 7, ModelRatio: decimal.NewFromInt(1)}}, validGroups), code: "upstream_schema"},
		{name: "negative billing ratio", body: sourceBody(t, []sourceModel{{Name: "model", QuotaType: 0, ModelRatio: decimal.NewFromInt(1), CacheRatio: decimalPointer("-0.1")}}, validGroups), code: "upstream_schema"},
		{name: "duplicate model", body: sourceBody(t, []sourceModel{validModel, validModel}, validGroups), code: "upstream_schema"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseSource(test.body)
			requireServiceError(t, err, http.StatusBadGateway, test.code)
		})
	}
}

func TestFetchSourceMapsUpstreamFailures(t *testing.T) {
	originalClient := pricingHTTPClient
	t.Cleanup(func() { pricingHTTPClient = originalClient })

	t.Run("timeout", func(t *testing.T) {
		pricingHTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			assert.Equal(t, Endpoint, request.URL.String())
			return nil, context.DeadlineExceeded
		})}
		_, err := fetchSource(context.Background())
		requireServiceError(t, err, http.StatusGatewayTimeout, "upstream_unavailable")
	})

	t.Run("status", func(t *testing.T) {
		pricingHTTPClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("unavailable")),
				Header:     make(http.Header),
			}, nil
		})}
		_, err := fetchSource(context.Background())
		requireServiceError(t, err, http.StatusBadGateway, "upstream_status")
	})

	t.Run("transport error", func(t *testing.T) {
		pricingHTTPClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})}
		_, err := fetchSource(context.Background())
		requireServiceError(t, err, http.StatusBadGateway, "upstream_unavailable")
	})
}

func TestOpenLuxURLRecognitionIsExact(t *testing.T) {
	assert.True(t, isOpenLuxBaseURL("https://api.openlux.ai"))
	assert.True(t, isOpenLuxBaseURL("https://API.OPENLUX.AI:443/"))
	assert.False(t, isOpenLuxBaseURL("http://api.openlux.ai"))
	assert.False(t, isOpenLuxBaseURL("https://api.openlux.ai/v1"))
	assert.False(t, isOpenLuxBaseURL("https://api.openlux.ai.evil.example"))
}
