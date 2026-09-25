package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingmappingservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMappingWriter struct {
	row       *accountingsync.AccountingMapping
	refs      []*accountingsync.AccountingReferenceObject
	summary   *serviceports.AccountingMappingSummary
	found     *serviceports.SetAccountingMappingRequest
	set       *serviceports.SetAccountingMappingRequest
	cleared   *serviceports.AccountingMappingActionRequest
	created   *serviceports.CreateAccountingReferenceRecordRequest
	refreshed *serviceports.AccountingSetupRequest
	guard     writeGuard
}

func (f *fakeMappingWriter) FindMapping(
	_ context.Context,
	req *serviceports.SetAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	f.found = req
	if f.row == nil {
		return nil, errortypes.NewNotFoundError("Accounting mapping not found")
	}
	out := *f.row
	return &out, nil
}

func (f *fakeMappingWriter) GetReferenceObjects(
	_ context.Context,
	req *serviceports.GetAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	out := make([]*accountingsync.AccountingReferenceObject, 0, 1)
	for _, ref := range f.refs {
		if ref.Kind == req.Kind && len(req.ExternalIDs) == 1 && ref.ExternalID == req.ExternalIDs[0] {
			out = append(out, ref)
		}
	}
	return out, nil
}

func (f *fakeMappingWriter) Set(
	_ context.Context,
	req *serviceports.SetAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.set = req
	for _, ref := range f.refs {
		if ref.ExternalID == req.ExternalID {
			f.row.Confirm(accountingmappingservice.MappingChoice(req, ref, timeutils.NowUnix()))
		}
	}
	return f.row, nil
}

func (f *fakeMappingWriter) Clear(
	_ context.Context,
	req *serviceports.AccountingMappingActionRequest,
) (*accountingsync.AccountingMapping, error) {
	if err := f.guard.write(); err != nil {
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
	req *serviceports.CreateAccountingReferenceRecordRequest,
) (*accountingsync.AccountingMapping, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = req
	name := req.Name
	if name == "" {
		name = f.row.TargetLabel
	}
	f.row.Confirm(accountingmappingservice.CreatedReferenceChoice(
		&accountingmappingservice.CreatedReference{
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
	req *serviceports.AccountingSetupRequest,
) (*accountingsync.AccountingConnection, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.refreshed = req
	return f.summary.Connection, nil
}

func (f *fakeMappingWriter) Summary(
	context.Context,
	pagination.TenantInfo,
	integration.Type,
) (*serviceports.AccountingMappingSummary, error) {
	return f.summary, nil
}

func proposedCustomerMapping() *accountingsync.AccountingMapping {
	return &accountingsync.AccountingMapping{
		ID:              pulid.MustNew("acctm_"),
		ConnectionID:    pulid.MustNew("acctc_"),
		TargetType:      accountingsync.TargetCustomer,
		TrenovaObjectID: pulid.MustNew("cus_"),
		TargetLabel:     "Acme Logistics",
		ProviderKind:    accountingsync.ReferenceKindCustomer,
		State:           accountingsync.MappingStateProposed,
		ExternalID:      "50",
		ExternalName:    "Acme Logistics, Inc.",
	}
}

func customerRef(externalID, name string, active bool) *accountingsync.AccountingReferenceObject {
	return &accountingsync.AccountingReferenceObject{
		Kind:       accountingsync.ReferenceKindCustomer,
		ExternalID: externalID,
		Name:       name,
		Active:     active,
	}
}

func TestSetAccountingMapping_SetsAnExistingRecordAsTheAgent(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	writer := &fakeMappingWriter{row: row, refs: []*accountingsync.AccountingReferenceObject{
		customerRef("51", "Acme Logistics LLC", true),
	}}
	tool := newSetAccountingMappingTool(writer)
	params := executeParams(map[string]any{
		"system":     "QuickBooksOnline",
		"mappingId":  row.ID.String(),
		"externalId": "51",
		"reason":     "  The LLC is the entity Acme invoices under.  ",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.set)
	assert.Equal(t, row.ID, writer.set.MappingID)
	assert.Equal(t, "51", writer.set.ExternalID)
	assert.Equal(t, accountingsync.MappingSourceAgent, writer.set.Source)
	assert.Equal(t, "The LLC is the entity Acme invoices under.", writer.set.Reason)
	assert.Equal(t, params.Actor.UserID, writer.set.UserID)
}

func TestSetAccountingMapping_RefusesAnInventedOrInactiveRecord(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	writer := &fakeMappingWriter{row: row, refs: []*accountingsync.AccountingReferenceObject{
		customerRef("52", "Acme (old)", false),
	}}
	tool := newSetAccountingMappingTool(writer)

	for _, externalID := range []string{"999", "52"} {
		err := tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(map[string]any{
			"system":     "QuickBooksOnline",
			"mappingId":  row.ID.String(),
			"externalId": externalID,
			"reason":     "guess",
		}))
		require.Error(t, err, externalID)
	}
	assert.Nil(t, writer.set)
}

func TestSetAccountingMapping_NamesATargetByKey(t *testing.T) {
	t.Parallel()

	writer := &fakeMappingWriter{}
	tool := newSetAccountingMappingTool(writer)

	err := tool.Execute(t.Context(), executeParams(map[string]any{
		"system":     "QuickBooksOnline",
		"targetType": "PaymentTerm",
		"key":        "ACH",
		"externalId": "3",
		"reason":     "wrong key",
	}))
	require.Error(t, err)
	assert.Nil(t, writer.found, "a key the target type does not accept never reaches the service")

	_ = tool.Execute(t.Context(), executeParams(map[string]any{
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
	params := executeParams(map[string]any{"system": "QuickBooksOnline", "mappingId": row.ID.String()})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.cleared)
	assert.Equal(t, row.ID, writer.cleared.ID)
	assert.Equal(t, accountingsync.MappingSourceAgent, writer.cleared.Source)

	writer.cleared = nil
	writer.row.State = accountingsync.MappingStateUnmatched
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, writer.cleared)
}

func TestCreateAccountingReferenceRecord_OnlyCreatesWhatTrenovaMayCreate(t *testing.T) {
	t.Parallel()

	row := proposedCustomerMapping()
	writer := &fakeMappingWriter{row: row}
	tool := newCreateAccountingReferenceRecordTool(writer)
	params := executeParams(map[string]any{
		"system":    "QuickBooksOnline",
		"mappingId": row.ID.String(),
		"name":      " Acme Logistics ",
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.created)
	assert.Equal(t, "Acme Logistics", writer.created.Name)
	assert.Equal(t, accountingsync.MappingSourceAgent, writer.created.Source)

	writer.created = nil
	writer.row.ProviderKind = accountingsync.ReferenceKindAccount
	writer.row.TargetType = accountingsync.TargetAccountRole
	require.Error(t, tool.Execute(t.Context(), params))

	writer.row.ProviderKind = accountingsync.ReferenceKindCustomer
	writer.row.State = accountingsync.MappingStateConfirmed
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, writer.created)
}

func TestRefreshAccountingReferenceData_NeedsAConnection(t *testing.T) {
	t.Parallel()

	writer := &fakeMappingWriter{summary: &serviceports.AccountingMappingSummary{
		ProviderName: "QuickBooks Online",
	}}
	tool := newRefreshAccountingReferenceDataTool(writer)
	params := executeParams(map[string]any{"system": "QuickBooksOnline"})

	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, writer.refreshed)

	writer.summary.Connection = &accountingsync.AccountingConnection{
		ID:                  pulid.MustNew("acctc_"),
		Status:              accountingsync.ConnectionStatusConnected,
		ExternalCompanyName: "Acme Freight",
	}
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.refreshed)
	assert.Equal(t, integration.TypeQuickBooksOnline, writer.refreshed.IntegrationType)
}

func TestAccountingMappingTools_AskAPersonBeforeChangingWhatPosts(t *testing.T) {
	t.Parallel()

	writer := &fakeMappingWriter{}
	for _, tool := range []serviceports.AgentTool{
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
