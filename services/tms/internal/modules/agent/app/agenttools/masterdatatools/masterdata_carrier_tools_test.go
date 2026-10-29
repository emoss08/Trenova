package masterdatatools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func texasState() *agenttoolstest.FakeStates {
	return &agenttoolstest.FakeStates{State: &usstate.UsState{
		ID:           pulid.MustNew("us_"),
		Abbreviation: "TX",
		Name:         "Texas",
	}}
}

func idempotentParams(params map[string]any) agenttoolports.ToolExecuteParams {
	execute := agenttoolstest.ExecuteParams(params)
	execute.IdempotencyKey = pulid.MustNew("idem_").String()

	return execute
}

type fakeCarriers struct {
	guard      *agenttoolstest.WriteGuard
	stored     *carrier.Carrier
	refusal    error
	created    *carrier.Carrier
	updated    *carrier.Carrier
	getRequest carrierports.GetCarrierByIDRequest
	bulk       *carrierports.BulkUpdateCarrierStatusRequest
}

func (f *fakeCarriers) Get(
	_ context.Context,
	req carrierports.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	f.getRequest = req
	copied := *f.stored

	return &copied, nil
}

func (f *fakeCarriers) PlanCreate(
	_ context.Context,
	entity *carrier.Carrier,
) (*carrier.Carrier, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return entity, nil
}

func (f *fakeCarriers) Create(
	ctx context.Context,
	entity *carrier.Carrier,
	_ *permissionports.RequestActor,
) (*carrier.Carrier, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	planned, err := f.PlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}
	planned.ID = pulid.MustNew("car_")
	f.created = planned

	return planned, nil
}

func (f *fakeCarriers) PlanUpdate(
	_ context.Context,
	entity *carrier.Carrier,
) (*kernelports.RecordChange[carrier.Carrier], error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return &kernelports.RecordChange[carrier.Carrier]{Before: f.stored, After: entity}, nil
}

func (f *fakeCarriers) Update(
	_ context.Context,
	entity *carrier.Carrier,
	_ *permissionports.RequestActor,
) (*carrier.Carrier, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = entity

	return entity, nil
}

func (f *fakeCarriers) PlanBulkUpdateStatus(
	_ context.Context,
	req *carrierports.BulkUpdateCarrierStatusRequest,
) ([]kernelports.RecordChange[carrier.Carrier], error) {
	after := *f.stored
	after.Status = req.Status

	return []kernelports.RecordChange[carrier.Carrier]{{Before: f.stored, After: &after}}, nil
}

func (f *fakeCarriers) BulkUpdateStatus(
	_ context.Context,
	req *carrierports.BulkUpdateCarrierStatusRequest,
) ([]*carrier.Carrier, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.bulk = req
	after := *f.stored
	after.Status = req.Status

	return []*carrier.Carrier{&after}, nil
}

func storedCarrier(states *agenttoolstest.FakeStates) *carrier.Carrier {
	stateID := states.State.ID

	return &carrier.Carrier{
		ID:              pulid.MustNew("car_"),
		Code:            "ACME",
		Name:            "Acme Freight",
		Status:          carrier.StatusActive,
		CarrierType:     carrier.TypeCommon,
		PaymentMethod:   carrier.PaymentMethodCheck,
		PaymentTermDays: 30,
		StateID:         &stateID,
		Contacts:        []*carrier.CarrierContact{{ID: pulid.MustNew("cc_")}},
		EDIChannels:     []*carrier.CarrierEDIChannel{{ID: pulid.MustNew("cedi_")}},
		Version:         4,
	}
}

func TestCreateCarrier_PreviewsTheCarrierAndCreatesIt(t *testing.T) {
	t.Parallel()

	states := texasState()
	carriers := &fakeCarriers{guard: &agenttoolstest.WriteGuard{}}
	tool := NewCreateCarrierTool(carriers, states)
	params := idempotentParams(map[string]any{
		mdCode:      "ROAD",
		MdName:      "Roadrunner Logistics",
		mdDOTNumber: "1234567",
		"scac":      "rrlg",
		mdState:     "tx",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		carriers.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would create the carrier Roadrunner Logistics")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationCreate, change.Operation)
	assert.Equal(t, "TX", agenttoolstest.FieldByPath(t, change, "state").After)
	assert.Equal(t, "RRLG", agenttoolstest.FieldByPath(t, change, "scac").After)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, carriers.created)
	assert.Equal(t, carrier.StatusActive, carriers.created.Status)
	assert.Equal(t, carrier.ComplianceStatusPending, carriers.created.ComplianceStatus)
	assert.Equal(t, states.State.ID, *carriers.created.StateID)
	assert.Equal(t, "created", result.Action)
	assert.Equal(t, changepreview.CarrierRecordEntity, result.Record.EntityType)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceCarrier, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal, agent.EgressMoney}, policy.Egress)
	require.NotNil(t, policy.TaintHold)
	assert.Equal(t, agent.EgressInternal, policy.Classify(params).Egress)
}

func TestCreateCarrier_PaymentDetailsAreMoneyAndWaitForAPerson(t *testing.T) {
	t.Parallel()

	carriers := &fakeCarriers{guard: &agenttoolstest.WriteGuard{}}
	tool := NewCreateCarrierTool(carriers, texasState())
	args := map[string]any{
		mdCode:            "ROAD",
		MdName:            "Roadrunner Logistics",
		carrierRemitLine1: "PO Box 9",
	}

	call := tool.Policy().Classify(idempotentParams(args))
	assert.Equal(t, agent.EgressMoney, call.Egress)
	assert.Equal(t, agent.TierPropose, call.MaxTier)

	_, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(),
		idempotentParams(args))
	require.ErrorIs(t, err, toolspec.ErrNeedsAPersonsApproval)
	assert.Nil(t, carriers.created)

	approved := agenttoolstest.ApprovedParams(args)
	approved.IdempotencyKey = pulid.MustNew("idem_").String()
	_, err = tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), approved)
	require.NoError(t, err)
	assert.Equal(t, "PO Box 9", carriers.created.RemitAddressLine1)
}

func TestCreateCarrier_RefusesWhatTheServiceRefuses(t *testing.T) {
	t.Parallel()

	for name, args := range map[string]map[string]any{
		"no name":            {mdCode: "ROAD"},
		"an unknown state":   {mdCode: "ROAD", MdName: "Road", mdState: "ZZ"},
		"an unknown type":    {mdCode: "ROAD", MdName: "Road", "carrierType": "Tramp"},
		"a payment term out": {mdCode: "ROAD", MdName: "Road", carrierPaymentTerm: 900},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tool := NewCreateCarrierTool(&fakeCarriers{}, texasState())
			require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
				idempotentParams(args)))
		})
	}

	refused := NewCreateCarrierTool(&fakeCarriers{refusal: errortypes.NewValidationError(
		mdCode, errortypes.ErrDuplicate, "Carrier with this code already exists",
	)}, texasState())
	preview, err := refused.(agenttoolports.ToolPreviewer).Preview(t.Context(),
		idempotentParams(map[string]any{mdCode: "ROAD", MdName: "Road"}))
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

func TestUpdateCarrier_KeepsWhatItIsNotGivenAndTheCarriersChildren(t *testing.T) {
	t.Parallel()

	states := texasState()
	carriers := &fakeCarriers{guard: &agenttoolstest.WriteGuard{}, stored: storedCarrier(states)}
	tool := NewUpdateCarrierTool(carriers, states)
	params := agenttoolstest.ExecuteParams(map[string]any{
		toolparams.ParamCarrierID: carriers.stored.ID.String(),
		mdPhone:                   "555-0100",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		carriers.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "555-0100", agenttoolstest.FieldByPath(t, change, mdPhone).After)
	require.NotNil(t, change.Version)
	assert.Equal(t, int64(4), *change.Version)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, carriers.updated)
	assert.Equal(t, "555-0100", carriers.updated.Phone)
	assert.Equal(t, "Acme Freight", carriers.updated.Name)
	assert.Equal(t, int64(4), carriers.updated.Version)
	assert.Len(t, carriers.updated.Contacts, 1)
	assert.Len(t, carriers.updated.EDIChannels, 1)
	assert.True(t, carriers.getRequest.IncludeEDIChannels)
	assert.True(t, carriers.getRequest.IncludeInsurancePolicies)

	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceCarrier, target.Resource)

	require.Error(
		t,
		tool.(agenttoolports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{toolparams.ParamCarrierID: carriers.stored.ID.String()},
		)),
	)
}

func TestUpdateCarrierStatus_PreviewsEachCarrierAndMovesThem(t *testing.T) {
	t.Parallel()

	states := texasState()
	carriers := &fakeCarriers{guard: &agenttoolstest.WriteGuard{}, stored: storedCarrier(states)}
	tool := NewUpdateCarrierStatusTool(carriers, states)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"carrierIds":              []any{carriers.stored.ID.String()},
		changepreview.FieldStatus: "DoNotUse",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		carriers.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		"DoNotUse",
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			changepreview.FieldStatus,
		).After,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, carriers.bulk)
	assert.Equal(t, carrier.StatusDoNotUse, carriers.bulk.Status)

	require.Error(
		t,
		tool.(agenttoolports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{
				"carrierIds":              []any{carriers.stored.ID.String()},
				changepreview.FieldStatus: "Gone",
			},
		)),
	)
}
