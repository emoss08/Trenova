package graphql

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPersistedOperationManifest(t *testing.T) {
	t.Parallel()

	manifest, err := LoadPersistedOperationManifest([]byte(`{"sha256:abc":"query Test { ok }"}`))
	require.NoError(t, err)

	query, ok := manifest.Query("sha256:abc")
	require.True(t, ok)
	assert.Equal(t, "query Test { ok }", query)
	assert.ElementsMatch(t, []string{"sha256:abc"}, manifest.KnownHashes())
}

func TestLoadPersistedOperationManifest_NormalizesBareSHA256Hashes(t *testing.T) {
	t.Parallel()

	manifest, err := LoadPersistedOperationManifest(
		[]byte(`{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":"query Test { ok }"}`),
	)
	require.NoError(t, err)

	query, ok := manifest.Query("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.True(t, ok)
	assert.Equal(t, "query Test { ok }", query)
	assert.ElementsMatch(
		t,
		[]string{"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		manifest.KnownHashes(),
	)
}

func TestNewPersistedOperationManifest_IncludesShipmentOperations(t *testing.T) {
	t.Parallel()

	manifest, err := NewPersistedOperationManifest()
	require.NoError(t, err)

	for _, operation := range []string{
		"ShipmentCommandCenterTable",
		"ShipmentDetail",
		"ShipmentSavedViewCounts",
		"ShipmentPageAnalytics",
		"ShipmentTomorrowsPickups",
		"UnassignedShipments",
		"ExceptionShipments",
		"MapShipments",
		"ShipmentComments",
		"ShipmentCommentCount",
		"ShipmentEvents",
		"ShipmentBillingReadiness",
		"ShipmentUIPolicy",
		"ShipmentPreviousRates",
		"CreateShipment",
		"UpdateShipment",
		"CancelShipment",
		"UncancelShipment",
		"DuplicateShipment",
		"TransferShipmentOwnership",
		"TransferShipmentToBilling",
		"BulkTransferShipmentsToBilling",
		"CalculateShipmentTotals",
		"CalculateShipmentDistance",
		"RecalculateShipmentDistance",
		"CheckShipmentDuplicateBol",
		"CheckShipmentHazmatSegregation",
		"CalculateShipmentLoadingOptimization",
		"CreateShipmentComment",
		"UpdateShipmentComment",
		"DeleteShipmentComment",
	} {
		require.True(
			t,
			manifestIncludesGraphQLOperation(manifest, operation),
			"missing persisted shipment operation %s",
			operation,
		)
	}
}

func manifestIncludesGraphQLOperation(manifest *PersistedOperationManifest, operation string) bool {
	for _, query := range manifest.queries {
		if graphQLOperationNameMatches(query, "query", operation) ||
			graphQLOperationNameMatches(query, "mutation", operation) {
			return true
		}
	}

	return false
}

func graphQLOperationNameMatches(query, operationType, operation string) bool {
	return strings.Contains(query, operationType+" "+operation+"(") ||
		strings.Contains(query, operationType+" "+operation+" {")
}

func TestLoadPersistedOperationManifest_RejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	manifest, err := LoadPersistedOperationManifest([]byte(`{"sha256:abc":`))

	require.Error(t, err)
	assert.Nil(t, manifest)
	assert.Contains(t, err.Error(), "parsing persisted operations")
}

func TestRewritePersistedOperationRequest_AcceptsKnownHash(t *testing.T) {
	t.Parallel()

	manifest, err := LoadPersistedOperationManifest(
		[]byte(`{"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":"query Safelisted($first: Int!) { tractors(first: $first) { totalCount } }"}`),
	)
	require.NoError(t, err)

	req := httptest.NewRequest(
		"POST",
		"/graphql",
		strings.NewReader(`{
			"query":"query Unsafelisted { trailers(first: 1) { totalCount } }",
			"operationName":"Safelisted",
			"variables":{"first":10},
			"extensions":{"persistedQuery":{"version":1,"sha256Hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
		}`),
	)

	require.NoError(t, rewritePersistedOperationRequest(req, manifest, true))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	var rewritten map[string]any
	require.NoError(t, sonic.Unmarshal(body, &rewritten))

	assert.Equal(
		t,
		"query Safelisted($first: Int!) { tractors(first: $first) { totalCount } }",
		rewritten["query"],
	)
	assert.Equal(t, "Safelisted", rewritten["operationName"])
	assert.Equal(t, map[string]any{"first": float64(10)}, rewritten["variables"])
	assert.Equal(t, int64(len(body)), req.ContentLength)
}

func TestRewritePersistedOperationRequest_RejectsUnknownHash(t *testing.T) {
	t.Parallel()

	manifest, err := LoadPersistedOperationManifest([]byte(`{"sha256:abc":"query Test { ok }"}`))
	require.NoError(t, err)

	req := httptest.NewRequest(
		"POST",
		"/graphql",
		strings.NewReader(`{"extensions":{"persistedQuery":{"version":1,"sha256Hash":"sha256:missing"}}}`),
	)

	err = rewritePersistedOperationRequest(req, manifest, true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL persisted operation is not safelisted")
}

func TestRewritePersistedOperationRequest_RejectsRawQueryWhenEnforced(t *testing.T) {
	t.Parallel()

	manifest, err := LoadPersistedOperationManifest([]byte(`{"sha256:abc":"query Test { ok }"}`))
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query":"query Raw { ok }"}`))

	err = rewritePersistedOperationRequest(req, manifest, true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL persisted operation hash is required")
}

func TestRewritePersistedOperationRequest_AllowsRawQueryWhenNotEnforced(t *testing.T) {
	t.Parallel()

	manifest, err := LoadPersistedOperationManifest([]byte(`{"sha256:abc":"query Test { ok }"}`))
	require.NoError(t, err)

	const body = `{"query":"query Raw { ok }","variables":{"first":10}}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))

	require.NoError(t, rewritePersistedOperationRequest(req, manifest, false))

	rewritten, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.JSONEq(t, body, string(rewritten))
}
