package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *fakeStates) GetByIDs(_ context.Context, ids []pulid.ID) ([]*usstate.UsState, error) {
	found := make([]*usstate.UsState, 0, len(ids))
	for _, id := range ids {
		if f.state != nil && f.state.ID == id {
			found = append(found, f.state)
		}
	}

	return found, nil
}

func texasState() *fakeStates {
	return &fakeStates{state: &usstate.UsState{
		ID:           pulid.MustNew("us_"),
		Abbreviation: "TX",
		Name:         "Texas",
	}}
}

func idempotentParams(params map[string]any) serviceports.ToolExecuteParams {
	execute := executeParams(params)
	execute.IdempotencyKey = pulid.MustNew("idem_").String()

	return execute
}

type fakeCarriers struct {
	guard      *writeGuard
	stored     *carrier.Carrier
	refusal    error
	created    *carrier.Carrier
	updated    *carrier.Carrier
	getRequest repositories.GetCarrierByIDRequest
	bulk       *repositories.BulkUpdateCarrierStatusRequest
}

func (f *fakeCarriers) Get(
	_ context.Context,
	req repositories.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	f.getRequest = req
	copied := *f.stored

	return &copied, nil
}

func (f *fakeCarriers) PlanCreate(_ context.Context, entity *carrier.Carrier) (*carrier.Carrier, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return entity, nil
}

func (f *fakeCarriers) Create(
	ctx context.Context,
	entity *carrier.Carrier,
	_ *serviceports.RequestActor,
) (*carrier.Carrier, error) {
	if err := f.guard.write(); err != nil {
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
) (*serviceports.RecordChange[carrier.Carrier], error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return &serviceports.RecordChange[carrier.Carrier]{Before: f.stored, After: entity}, nil
}

func (f *fakeCarriers) Update(
	_ context.Context,
	entity *carrier.Carrier,
	_ *serviceports.RequestActor,
) (*carrier.Carrier, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = entity

	return entity, nil
}

func (f *fakeCarriers) PlanBulkUpdateStatus(
	_ context.Context,
	req *repositories.BulkUpdateCarrierStatusRequest,
) ([]serviceports.RecordChange[carrier.Carrier], error) {
	after := *f.stored
	after.Status = req.Status

	return []serviceports.RecordChange[carrier.Carrier]{{Before: f.stored, After: &after}}, nil
}

func (f *fakeCarriers) BulkUpdateStatus(
	_ context.Context,
	req *repositories.BulkUpdateCarrierStatusRequest,
) ([]*carrier.Carrier, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.bulk = req
	after := *f.stored
	after.Status = req.Status

	return []*carrier.Carrier{&after}, nil
}

func storedCarrier(states *fakeStates) *carrier.Carrier {
	stateID := states.state.ID

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
	carriers := &fakeCarriers{guard: &writeGuard{}}
	tool := newCreateCarrierTool(carriers, states)
	params := idempotentParams(map[string]any{
		mdCode:      "ROAD",
		mdName:      "Roadrunner Logistics",
		mdDOTNumber: "1234567",
		"scac":      "rrlg",
		mdState:     "tx",
	})

	preview := previewWithoutWrites(t, carriers.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would create the carrier Roadrunner Logistics")
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, "TX", fieldByPath(t, change, "state").After)
	assert.Equal(t, "RRLG", fieldByPath(t, change, "scac").After)

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, carriers.created)
	assert.Equal(t, carrier.StatusActive, carriers.created.Status)
	assert.Equal(t, carrier.ComplianceStatusPending, carriers.created.ComplianceStatus)
	assert.Equal(t, states.state.ID, *carriers.created.StateID)
	assert.Equal(t, "created", result.Action)
	assert.Equal(t, carrierRecordEntity, result.Record.EntityType)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceCarrier, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal, agent.EgressMoney}, policy.Egress)
	require.NotNil(t, policy.TaintHold)
	assert.Equal(t, agent.EgressInternal, policy.Classify(params).Egress)
}

func TestCreateCarrier_PaymentDetailsAreMoneyAndWaitForAPerson(t *testing.T) {
	t.Parallel()

	carriers := &fakeCarriers{guard: &writeGuard{}}
	tool := newCreateCarrierTool(carriers, texasState())
	args := map[string]any{
		mdCode:            "ROAD",
		mdName:            "Roadrunner Logistics",
		carrierRemitLine1: "PO Box 9",
	}

	call := tool.Policy().Classify(idempotentParams(args))
	assert.Equal(t, agent.EgressMoney, call.Egress)
	assert.Equal(t, agent.TierPropose, call.MaxTier)

	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(),
		idempotentParams(args))
	require.ErrorIs(t, err, ErrNeedsAPersonsApproval)
	assert.Nil(t, carriers.created)

	approved := approvedParams(args)
	approved.IdempotencyKey = pulid.MustNew("idem_").String()
	_, err = tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), approved)
	require.NoError(t, err)
	assert.Equal(t, "PO Box 9", carriers.created.RemitAddressLine1)
}

func TestCreateCarrier_RefusesWhatTheServiceRefuses(t *testing.T) {
	t.Parallel()

	for name, args := range map[string]map[string]any{
		"no name":            {mdCode: "ROAD"},
		"an unknown state":   {mdCode: "ROAD", mdName: "Road", mdState: "ZZ"},
		"an unknown type":    {mdCode: "ROAD", mdName: "Road", "carrierType": "Tramp"},
		"a payment term out": {mdCode: "ROAD", mdName: "Road", carrierPaymentTerm: 900},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tool := newCreateCarrierTool(&fakeCarriers{}, texasState())
			require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
				idempotentParams(args)))
		})
	}

	refused := newCreateCarrierTool(&fakeCarriers{refusal: errortypes.NewValidationError(
		mdCode, errortypes.ErrDuplicate, "Carrier with this code already exists",
	)}, texasState())
	preview, err := refused.(serviceports.ToolPreviewer).Preview(t.Context(),
		idempotentParams(map[string]any{mdCode: "ROAD", mdName: "Road"}))
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestUpdateCarrier_KeepsWhatItIsNotGivenAndTheCarriersChildren(t *testing.T) {
	t.Parallel()

	states := texasState()
	carriers := &fakeCarriers{guard: &writeGuard{}, stored: storedCarrier(states)}
	tool := newUpdateCarrierTool(carriers, states)
	params := executeParams(map[string]any{
		paramCarrierID: carriers.stored.ID.String(),
		mdPhone:        "555-0100",
	})

	preview := previewWithoutWrites(t, carriers.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "555-0100", fieldByPath(t, change, mdPhone).After)
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

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceCarrier, target.Resource)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(
		map[string]any{paramCarrierID: carriers.stored.ID.String()},
	)))
}

func TestUpdateCarrierStatus_PreviewsEachCarrierAndMovesThem(t *testing.T) {
	t.Parallel()

	states := texasState()
	carriers := &fakeCarriers{guard: &writeGuard{}, stored: storedCarrier(states)}
	tool := newUpdateCarrierStatusTool(carriers, states)
	params := executeParams(map[string]any{
		"carrierIds": []any{carriers.stored.ID.String()},
		fieldStatus:  "DoNotUse",
	})

	preview := previewWithoutWrites(t, carriers.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "DoNotUse", fieldByPath(t, previewChange(t, preview, 0), fieldStatus).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, carriers.bulk)
	assert.Equal(t, carrier.StatusDoNotUse, carriers.bulk.Status)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(
		map[string]any{"carrierIds": []any{carriers.stored.ID.String()}, fieldStatus: "Gone"},
	)))
}
