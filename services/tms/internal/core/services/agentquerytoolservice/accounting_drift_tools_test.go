package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDriftReader struct {
	findings []*accountingsync.AccountingDriftFinding
	listed   *serviceports.ListAccountingDriftFindingsRequest
}

func (f *fakeDriftReader) List(
	_ context.Context,
	req *serviceports.ListAccountingDriftFindingsRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingDriftFinding], error) {
	f.listed = req
	return &pagination.CursorListResult[*accountingsync.AccountingDriftFinding]{
		Items: f.findings,
	}, nil
}

func (f *fakeDriftReader) Overview(
	_ context.Context,
	_ *serviceports.AccountingDriftOverviewRequest,
) (*serviceports.AccountingDriftOverview, error) {
	checked := int64(1_790_208_000)
	return &serviceports.AccountingDriftOverview{
		ToleranceMinor: 500,
		CurrencyCode:   "CAD",
		CheckedAt:      &checked,
	}, nil
}

func driftFindingOf(
	kind accountingsync.DriftKind,
	trenova, provider int64,
) *accountingsync.AccountingDriftFinding {
	modified := int64(1_790_100_000)
	return accountingsync.NewAccountingDriftFinding(&accountingsync.DriftObservation{
		ConnectionID:       pulid.MustNew("acctc_"),
		ObjectType:         accountingsync.SyncObjectInvoice,
		ObjectID:           pulid.MustNew("inv_"),
		ObjectNumber:       "INV-1001",
		PartyName:          "Acme Foods",
		ExternalURL:        "https://books.example/invoice/145",
		Kind:               kind,
		CurrencyCode:       "CAD",
		TrenovaMinor:       &trenova,
		ProviderMinor:      &provider,
		ProviderModifiedAt: &modified,
		ProviderModifiedBy: "J Doe",
		At:                 1_790_150_000,
	})
}

func TestListAccountingDriftFindings_FiltersAndShowsBothSidesAndTheTolerance(t *testing.T) {
	t.Parallel()

	small := driftFindingOf(accountingsync.DriftAmountMismatch, 125_000, 124_700)
	large := driftFindingOf(accountingsync.DriftAmountMismatch, 125_000, 120_000)
	dismissed := driftFindingOf(accountingsync.DriftAmountMismatch, 125_000, 124_900)
	require.NoError(t, dismissed.Dismiss(pulid.MustNew("usr_"), "Rounding", 1_790_200_000))
	reader := &fakeDriftReader{findings: []*accountingsync.AccountingDriftFinding{small, large, dismissed}}
	params := accountingQueryParams("QuickBooksOnline")
	params.Params["status"] = []any{"Open"}
	params.Params["kind"] = []any{"AmountMismatch"}
	params.Params["objectType"] = []any{"Invoice", "Customer"}
	params.Params["search"] = " Acme "
	params.Params["limit"] = 500.0

	out, err := newListAccountingDriftFindingsTool(reader).Query(t.Context(), params)
	require.NoError(t, err)
	result, ok := out.(*accountingDriftResult)
	require.True(t, ok)

	require.NotNil(t, reader.listed)
	assert.Equal(t, []accountingsync.DriftStatus{accountingsync.DriftStatusOpen}, reader.listed.Statuses)
	assert.Equal(t, []accountingsync.DriftKind{accountingsync.DriftAmountMismatch}, reader.listed.Kinds)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectCustomer,
	}, reader.listed.ObjectTypes)
	assert.Equal(t, "Acme", reader.listed.Search)
	assert.Equal(t, driftMaxLimit, reader.listed.Cursor.Limit)
	assert.Equal(t, params.OrganizationID, reader.listed.TenantInfo.OrgID)
	assert.Equal(t, accountingDriftPath, result.PagePath)
	assert.Equal(t, "5.00 CAD", result.Tolerance)

	require.Len(t, result.Findings, 3)
	row := result.Findings[0]
	assert.Equal(t, "1250.00 CAD", row.InTrenova)
	assert.Equal(t, "1247.00 CAD", row.InProvider)
	assert.Equal(t, "-3.00 CAD", row.Difference)
	assert.True(t, row.WithinTolerance)
	assert.Equal(t, "J Doe", row.ChangedBy)
	assert.Equal(t, []string{"PushTrenovaValue", "AdjustTrenova"}, row.Fixes)
	assert.False(t, result.Findings[1].WithinTolerance)
	assert.Empty(t, result.Findings[2].Fixes, "a dismissed finding offers no fix")
	assert.Equal(t, "Rounding", result.Findings[2].Note)
}

func TestListAccountingDriftFindings_RejectsUnknownValues(t *testing.T) {
	t.Parallel()

	for _, bad := range []map[string]any{
		{"status": []any{"Pending"}},
		{"kind": []any{"Typo"}},
		{"objectType": []any{"Shipment"}},
	} {
		params := accountingQueryParams("QuickBooksOnline")
		for key, value := range bad {
			params.Params[key] = value
		}
		_, err := newListAccountingDriftFindingsTool(&fakeDriftReader{}).Query(t.Context(), params)
		require.Error(t, err, bad)
	}
}
