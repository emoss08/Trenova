package accountingtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMappingWriter struct {
	row       *accountingmapping.AccountingMapping
	refs      []*accountingmapping.AccountingReferenceObject
	summary   *accountingmappingports.AccountingMappingSummary
	found     *accountingmappingports.SetAccountingMappingRequest
	set       *accountingmappingports.SetAccountingMappingRequest
	cleared   *accountingmappingports.AccountingMappingActionRequest
	created   *accountingmappingports.CreateAccountingReferenceRecordRequest
	refreshed *accountingmappingports.AccountingSetupRequest
	used      int
	guard     agenttoolstest.WriteGuard
}

func (f *fakeMappingWriter) GuardHistory(
	_ context.Context,
	_ pagination.TenantInfo,
	row *accountingmapping.AccountingMapping,
	acknowledged bool,
) (int, error) {
	if f.used == 0 || acknowledged {
		return f.used, nil
	}

	return f.used, errortypes.NewValidationError(
		"acknowledgeHistory",
		errortypes.ErrInvalidOperation,
		"{0} was used by {1} documents already sent to the accounting system",
		row.TargetLabel,
		f.used,
	)
}

func (f *fakeMappingWriter) FindMapping(
	_ context.Context,
	req *accountingmappingports.SetAccountingMappingRequest,
) (*accountingmapping.AccountingMapping, error) {
	f.found = req
	if f.row == nil {
		return nil, errortypes.NewNotFoundError("Accounting mapping not found")
	}
	out := *f.row
	return &out, nil
}

func (f *fakeMappingWriter) GetReferenceObjects(
	_ context.Context,
	req *accountingmappingports.GetAccountingReferenceObjectsRequest,
) ([]*accountingmapping.AccountingReferenceObject, error) {
	out := make([]*accountingmapping.AccountingReferenceObject, 0, 1)
	for _, ref := range f.refs {
		if ref.Kind == req.Kind && len(req.ExternalIDs) == 1 &&
			ref.ExternalID == req.ExternalIDs[0] {
			out = append(out, ref)
		}
	}
	return out, nil
}

func (f *fakeMappingWriter) Set(
	_ context.Context,
	req *accountingmappingports.SetAccountingMappingRequest,
) (*accountingmapping.AccountingMapping, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.set = req
	for _, ref := range f.refs {
		if ref.ExternalID == req.ExternalID {
			f.row.Confirm(accountingmappingkit.MappingChoice(req, ref, timeutils.NowUnix()))
		}
	}
	return f.row, nil
}

func (f *fakeMappingWriter) Clear(
	_ context.Context,
	req *accountingmappingports.AccountingMappingActionRequest,
) (*accountingmapping.AccountingMapping, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.cleared = req
	if err := f.row.Clear(); err != nil {
		return nil, err
	}
	return f.row, nil
}

func (f *fakeMappingWriter) CreateReferenceRecord(
	_ context.Context,
	req *accountingmappingports.CreateAccountingReferenceRecordRequest,
) (*accountingmapping.AccountingMapping, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.created = req
	name := req.Name
	if name == "" {
		name = f.row.TargetLabel
	}
	f.row.Confirm(accountingmappingkit.CreatedReferenceChoice(
		&accountingmappingkit.CreatedReference{
			ExternalID:      "created-1",
			ExternalName:    name,
			RequestedSource: req.Source,
			IntegrationType: integration.TypeQuickBooksOnline,
			ActorID:         req.UserID,
			At:              timeutils.NowUnix(),
		},
	))
	return f.row, nil
}

func (f *fakeMappingWriter) RequestRefresh(
	_ context.Context,
	req *accountingmappingports.AccountingSetupRequest,
) (*accountingconnection.AccountingConnection, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.refreshed = req
	return f.summary.Connection, nil
}

func (f *fakeMappingWriter) Summary(
	context.Context,
	pagination.TenantInfo,
	integration.Type,
) (*accountingmappingports.AccountingMappingSummary, error) {
	return f.summary, nil
}

func proposedCustomerMapping() *accountingmapping.AccountingMapping {
	return &accountingmapping.AccountingMapping{
		ID:              pulid.MustNew("acctm_"),
		ConnectionID:    pulid.MustNew("acctc_"),
		TargetType:      accountingsync.TargetCustomer,
		TrenovaObjectID: pulid.MustNew("cus_"),
		TargetLabel:     "Acme Logistics",
		ProviderKind:    accountingprovider.ReferenceKindCustomer,
		State:           accountingmapping.MappingStateProposed,
		ExternalID:      "50",
		ExternalName:    "Acme Logistics, Inc.",
	}
}

func customerRef(
	externalID, name string,
	active bool,
) *accountingmapping.AccountingReferenceObject {
	return &accountingmapping.AccountingReferenceObject{
		Kind:       accountingprovider.ReferenceKindCustomer,
		ExternalID: externalID,
		Name:       name,
		Active:     active,
	}
}

func TestSetAccountingMapping_SetsAnExistingRecordAsTheAgent(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	writer := &fakeMappingWriter{row: row, refs: []*accountingmapping.AccountingReferenceObject{
		customerRef("51", "Acme Logistics LLC", true),
	}}
	tool := newSetAccountingMappingTool(writer)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"system":     "QuickBooksOnline",
		"mappingId":  row.ID.String(),
		"externalId": "51",
		"reason":     "  The LLC is the entity Acme invoices under.  ",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.set)
	assert.Equal(t, row.ID, writer.set.MappingID)
	assert.Equal(t, "51", writer.set.ExternalID)
	assert.Equal(t, accountingmapping.MappingSourceAgent, writer.set.Source)
	assert.Equal(t, "The LLC is the entity Acme invoices under.", writer.set.Reason)
	assert.Equal(t, params.Actor.UserID, writer.set.UserID)
}

func TestSetAccountingMapping_RefusesAnInventedOrInactiveRecord(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	writer := &fakeMappingWriter{row: row, refs: []*accountingmapping.AccountingReferenceObject{
		customerRef("52", "Acme (old)", false),
	}}
	tool := newSetAccountingMappingTool(writer)

	for _, externalID := range []string{"999", "52"} {
		err := tool.(agenttoolports.ToolValidator).Validate(
			t.Context(),
			agenttoolstest.ExecuteParams(map[string]any{
				"system":     "QuickBooksOnline",
				"mappingId":  row.ID.String(),
				"externalId": externalID,
				"reason":     "guess",
			}),
		)
		require.Error(t, err, externalID)
	}
	assert.Nil(t, writer.set)
}

func TestSetAccountingMapping_NamesATargetByKey(t *testing.T) {
	t.Parallel()

	writer := &fakeMappingWriter{}
	tool := newSetAccountingMappingTool(writer)

	err := tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"system":     "QuickBooksOnline",
		"targetType": "PaymentTerm",
		"key":        "ACH",
		"externalId": "3",
		"reason":     "wrong key",
	}))
	require.Error(t, err)
	assert.Nil(t, writer.found, "a key the target type does not accept never reaches the service")

	_ = tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"system":     "QuickBooksOnline",
		"targetType": "AccountRole",
		"key":        "ARAccount",
		"externalId": "3",
		"reason":     "the only receivable account",
	}))
	require.NotNil(t, writer.found)
	assert.Equal(t, accountingsync.TargetAccountRole, writer.found.TargetType)
	assert.Equal(t, "ARAccount", writer.found.TrenovaKey)
}

func TestClearAccountingMapping_RefusesAnUnmatchedMapping(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	writer := &fakeMappingWriter{row: row}
	tool := newClearAccountingMappingTool(writer)
	params := agenttoolstest.ExecuteParams(
		map[string]any{"system": "QuickBooksOnline", "mappingId": row.ID.String()},
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.cleared)
	assert.Equal(t, row.ID, writer.cleared.ID)
	assert.Equal(t, accountingmapping.MappingSourceAgent, writer.cleared.Source)

	writer.cleared = nil
	writer.row.State = accountingmapping.MappingStateUnmatched
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, writer.cleared)
}

func TestCreateAccountingReferenceRecord_OnlyCreatesWhatTrenovaMayCreate(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	writer := &fakeMappingWriter{row: row}
	tool := newCreateAccountingReferenceRecordTool(writer)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"system":    "QuickBooksOnline",
		"mappingId": row.ID.String(),
		"name":      " Acme Logistics ",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.created)
	assert.Equal(t, "Acme Logistics", writer.created.Name)
	assert.Equal(t, accountingmapping.MappingSourceAgent, writer.created.Source)

	writer.created = nil
	writer.row.ProviderKind = accountingprovider.ReferenceKindAccount
	writer.row.TargetType = accountingsync.TargetAccountRole
	require.Error(t, tool.Execute(t.Context(), params))

	writer.row.ProviderKind = accountingprovider.ReferenceKindCustomer
	writer.row.State = accountingmapping.MappingStateConfirmed
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, writer.created)
}

func TestRefreshAccountingReferenceData_NeedsAConnection(t *testing.T) {
	t.Parallel()

	writer := &fakeMappingWriter{summary: &accountingmappingports.AccountingMappingSummary{
		ProviderName: "QuickBooks Online",
	}}
	tool := newRefreshAccountingReferenceDataTool(writer)
	params := agenttoolstest.ExecuteParams(map[string]any{"system": "QuickBooksOnline"})

	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, writer.refreshed)

	writer.summary.Connection = &accountingconnection.AccountingConnection{
		ID:                  pulid.MustNew("acctc_"),
		Status:              accountingconnection.ConnectionStatusConnected,
		ExternalCompanyName: "Acme Freight",
	}
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.refreshed)
	assert.Equal(t, integration.TypeQuickBooksOnline, writer.refreshed.IntegrationType)
}

func TestAccountingMappingTools_AskAPersonBeforeChangingWhatPosts(t *testing.T) {
	t.Parallel()

	writer := &fakeMappingWriter{}
	for _, tool := range []agenttoolports.AgentTool{
		newSetAccountingMappingTool(writer),
		newClearAccountingMappingTool(writer),
		newCreateAccountingReferenceRecordTool(writer),
	} {
		policy := tool.Policy()
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
		assert.Equal(t, agent.TierActWithApproval, policy.MaxTier, tool.Name())
	}
	assert.False(t, newCreateAccountingReferenceRecordTool(writer).Policy().Reversible)
}
