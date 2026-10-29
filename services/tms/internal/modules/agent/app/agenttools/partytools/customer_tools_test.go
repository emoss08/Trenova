package partytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCustomers struct {
	*agenttoolstest.FakeMaster[customer.Customer]
}

func (f fakeCustomers) Get(
	context.Context,
	customerports.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	return f.MasterGet()
}

func (f fakeCustomers) PlanCreate(
	ctx context.Context,
	e *customer.Customer,
) (*customer.Customer, error) {
	return f.MasterPlanCreate(ctx, e)
}

func (f fakeCustomers) Create(
	ctx context.Context,
	e *customer.Customer,
	a *permissionports.RequestActor,
) (*customer.Customer, error) {
	return f.MasterCreate(ctx, e, a)
}

func (f fakeCustomers) PlanUpdate(
	ctx context.Context,
	e *customer.Customer,
) (*kernelports.RecordChange[customer.Customer], error) {
	return f.MasterPlanUpdate(ctx, e)
}

func (f fakeCustomers) Update(
	ctx context.Context,
	e *customer.Customer,
	a *permissionports.RequestActor,
) (*customer.Customer, error) {
	return f.MasterUpdate(ctx, e, a)
}

func (f fakeCustomers) PlanBulkUpdateStatus(
	_ context.Context,
	req *customerports.BulkUpdateCustomerStatusRequest,
) ([]kernelports.RecordChange[customer.Customer], error) {
	after := *f.Stored
	after.Status = req.Status

	return []kernelports.RecordChange[customer.Customer]{{Before: f.Stored, After: &after}}, nil
}

func (f fakeCustomers) BulkUpdateStatus(
	_ context.Context,
	req *customerports.BulkUpdateCustomerStatusRequest,
) ([]*customer.Customer, error) {
	f.Status = req

	return []*customer.Customer{f.Stored}, f.Guard.Write()
}

func TestCreateCustomer_StatusEmailRecipientsAreHeldAsLeavingTheOrganization(t *testing.T) {
	t.Parallel()

	states := agenttoolstest.TexasState()
	customers := fakeCustomers{&agenttoolstest.FakeMaster[customer.Customer]{
		Guard: &agenttoolstest.WriteGuard{},
		SetID: func(c *customer.Customer) { c.ID = pulid.MustNew("cus_") },
	}}
	tool := NewCreateCustomerTool(customers, states)
	args := map[string]any{
		masterdatakit.MdCode:         "ACME",
		masterdatakit.MdName:         "Acme Foods",
		masterdatakit.MdAddressLine1: "1 Main St",
		masterdatakit.MdCity:         "Dallas",
		masterdatakit.MdState:        "TX",
		masterdatakit.MdPostalCode:   "75201",
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		customers.Guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(
				t.Context(),
				agenttoolstest.IdempotentParams(args),
			)
		},
	)
	assert.Equal(
		t,
		"TX",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			masterdatakit.MdState,
		).After,
	)
	assert.Equal(
		t,
		agent.EgressInternal,
		tool.Policy().Classify(agenttoolstest.IdempotentParams(args)).Egress,
	)

	args[customerUpdateRecipients] = "ops@acme.example"
	call := tool.Policy().Classify(agenttoolstest.IdempotentParams(args))
	assert.Equal(t, agent.EgressExternalRecipient, call.Egress)
	assert.Equal(t, agent.TierPropose, call.MaxTier)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(),
		agenttoolstest.IdempotentParams(args))
	require.NoError(t, err)
	assert.Equal(t, states.State.ID, customers.Created.StateID)
	assert.Equal(t, customer.StatusUpdateNone, customers.Created.StatusUpdatePreference)
	assert.Equal(t, changepreview.CustomerRecordEntity, result.Record.EntityType)

	delete(args, masterdatakit.MdState)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.IdempotentParams(args)))
}

func TestUpdateCustomer_LeavesTheBillingProfileAlone(t *testing.T) {
	t.Parallel()

	states := agenttoolstest.TexasState()
	stored := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		Code:           "ACME",
		Name:           "Acme Foods",
		StateID:        states.State.ID,
		Status:         domaintypes.StatusActive,
		BillingProfile: &customer.CustomerBillingProfile{},
		Version:        7,
	}
	customers := fakeCustomers{
		&agenttoolstest.FakeMaster[customer.Customer]{
			Guard:  &agenttoolstest.WriteGuard{},
			Stored: stored,
		},
	}
	tool := NewUpdateCustomerTool(customers, states)
	params := agenttoolstest.ExecuteParams(
		map[string]any{
			toolparams.ParamCustomerID: stored.ID.String(),
			masterdatakit.MdCity:       "Plano",
		},
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Plano", customers.Updated.City)
	assert.Nil(t, customers.Updated.BillingProfile)
	assert.Equal(t, int64(7), customers.Updated.Version)

	status := NewUpdateCustomerStatusTool(customers, states)
	require.NoError(t, status.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"customerIds":             []any{stored.ID.String()},
		changepreview.FieldStatus: "Inactive",
	})))
	require.IsType(t, &customerports.BulkUpdateCustomerStatusRequest{}, customers.Status)
}
