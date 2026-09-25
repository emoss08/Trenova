package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMappingReader struct {
	summary  *serviceports.AccountingMappingSummary
	rows     []*accountingsync.AccountingMapping
	refs     []*accountingsync.AccountingReferenceObject
	listed   *serviceports.ListAccountingMappingsRequest
	found    *serviceports.SetAccountingMappingRequest
	searched *serviceports.SearchAccountingReferenceRequest
}

func (f *fakeMappingReader) Summary(
	context.Context,
	pagination.TenantInfo,
	integration.Type,
) (*serviceports.AccountingMappingSummary, error) {
	return f.summary, nil
}

func (f *fakeMappingReader) ListMappings(
	_ context.Context,
	req *serviceports.ListAccountingMappingsRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error) {
	f.listed = req
	return &pagination.CursorListResult[*accountingsync.AccountingMapping]{Items: f.rows, HasNextPage: true}, nil
}

func (f *fakeMappingReader) FindMapping(
	_ context.Context,
	req *serviceports.SetAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	f.found = req
	return f.rows[0], nil
}

func (f *fakeMappingReader) SearchReference(
	_ context.Context,
	req *serviceports.SearchAccountingReferenceRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	f.searched = req
	return f.refs, nil
}

func unmatchedRoleMapping() *accountingsync.AccountingMapping {
	confidence := 0.62
	return &accountingsync.AccountingMapping{
		ID:           pulid.MustNew("acctm_"),
		TargetType:   accountingsync.TargetAccountRole,
		TrenovaKey:   accountingsync.AccountRoleRevenue,
		TargetLabel:  "Revenue",
		ProviderKind: accountingsync.ReferenceKindAccount,
		State:        accountingsync.MappingStateUnmatched,
		Confidence:   &confidence,
		Signals: accountingsync.MappingSignals{Candidates: []accountingsync.MappingCandidate{
			{ExternalID: "79", Name: "Freight Income", Score: 0.62, Reason: "name"},
		}},
	}
}

func TestListAccountingMappingGaps_ListsOpenMappingsWithTheSummary(t *testing.T) {
	t.Parallel()

	reader := &fakeMappingReader{
		summary: &serviceports.AccountingMappingSummary{
			IntegrationType:   integration.TypeQuickBooksOnline,
			ProviderName:      "QuickBooks Online",
			RequiredTotal:     4,
			RequiredConfirmed: 1,
			Connection: &accountingsync.AccountingConnection{
				Status:    accountingsync.ConnectionStatusConnected,
				SetupStep: accountingsync.SetupStepMappings,
			},
			Groups: []serviceports.AccountingMappingGroup{
				{TargetType: accountingsync.TargetAccountRole, Unmatched: 5, Proposed: 1},
			},
		},
		rows: []*accountingsync.AccountingMapping{unmatchedRoleMapping()},
	}
	tool := newListAccountingMappingGapsTool(reader)
	params := accountingQueryParams("QuickBooksOnline")
	params.Params["targetType"] = "AccountRole"
	params.Params["limit"] = 500.0

	out, err := tool.Query(t.Context(), params)
	require.NoError(t, err)
	result, ok := out.(*accountingMappingGapsResult)
	require.True(t, ok)

	assert.True(t, result.Connected)
	assert.Equal(t, "Mappings", result.SetupStep)
	assert.Equal(t, 1, result.RequiredConfirmed)
	assert.True(t, result.HasMore)
	require.Len(t, result.Gaps, 1)
	assert.Equal(t, "RevenueAccount", result.Gaps[0].Key)
	require.Len(t, result.Gaps[0].Candidates, 1)
	assert.Equal(t, "79", result.Gaps[0].Candidates[0].ExternalID)

	require.NotNil(t, reader.listed)
	assert.Equal(t, mappingGapsMaxLimit, reader.listed.Cursor.Limit)
	assert.Equal(t, []accountingsync.MappingTargetType{accountingsync.TargetAccountRole}, reader.listed.TargetTypes)
	assert.ElementsMatch(t, []accountingsync.MappingState{
		accountingsync.MappingStateUnmatched,
		accountingsync.MappingStateProposed,
	}, reader.listed.States)
}

func TestListAccountingMappingGaps_BeforeConnecting(t *testing.T) {
	t.Parallel()

	reader := &fakeMappingReader{summary: &serviceports.AccountingMappingSummary{
		ProviderName:  "QuickBooks Online",
		RequiredTotal: 4,
	}}
	out, err := newListAccountingMappingGapsTool(reader).Query(t.Context(), accountingQueryParams("QuickBooksOnline"))
	require.NoError(t, err)
	result, ok := out.(*accountingMappingGapsResult)
	require.True(t, ok)
	assert.False(t, result.Connected)
	assert.Empty(t, result.Gaps)
	assert.Nil(t, reader.listed, "nothing is listed before a connection exists")
}

func TestGetAccountingMapping_SearchesTheRightKindOfRecord(t *testing.T) {
	t.Parallel()

	reader := &fakeMappingReader{
		rows: []*accountingsync.AccountingMapping{unmatchedRoleMapping()},
		refs: []*accountingsync.AccountingReferenceObject{{
			Kind:               accountingsync.ReferenceKindAccount,
			ExternalID:         "79",
			Name:               "Freight Income",
			FullyQualifiedName: "Income:Freight Income",
			AccountType:        "Income",
			Active:             true,
		}},
	}
	params := accountingQueryParams("QuickBooksOnline")
	params.Params["targetType"] = "AccountRole"
	params.Params["key"] = "RevenueAccount"
	params.Params["search"] = "freight"

	out, err := newGetAccountingMappingTool(reader).Query(t.Context(), params)
	require.NoError(t, err)
	detail, ok := out.(*accountingMappingDetail)
	require.True(t, ok)

	assert.Equal(t, "RevenueAccount", reader.found.TrenovaKey)
	require.NotNil(t, reader.searched)
	assert.Equal(t, accountingsync.ReferenceKindAccount, reader.searched.Kind)
	assert.True(t, reader.searched.UsableOnly)
	require.Len(t, detail.Options, 1)
	assert.Equal(t, "Income:Freight Income", detail.Options[0].Name)
	assert.Equal(t, "Income", detail.Options[0].Type)
}

func TestGetAccountingMapping_RefusesAnUnnamedTarget(t *testing.T) {
	t.Parallel()

	reader := &fakeMappingReader{rows: []*accountingsync.AccountingMapping{unmatchedRoleMapping()}}
	tool := newGetAccountingMappingTool(reader)

	for _, extra := range []map[string]any{
		{},
		{"targetType": "Customer"},
		{"targetType": "PaymentTerm", "key": "ACH"},
		{"targetType": "Nonsense", "key": "ARAccount"},
	} {
		params := accountingQueryParams("QuickBooksOnline")
		for key, value := range extra {
			params.Params[key] = value
		}
		_, err := tool.Query(t.Context(), params)
		require.Error(t, err, extra)
	}
	assert.Nil(t, reader.found)
}
