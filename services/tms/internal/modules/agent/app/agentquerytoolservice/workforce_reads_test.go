package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWorkforceReads struct {
	events       []*worker.WorkerSafetyEvent
	violations   []*worker.WorkerSafetyViolation
	recognitions []*worker.WorkerRecognition
	tests        []*worker.WorkerDOTTest
	draw         *worker.DOTRandomDraw
	entries      []*worker.DOTRandomDrawEntry
	injuries     []*worker.WorkerInjury
	cases        []*worker.WorkerLeaveCase
	expenses     []*driverpay.Expense
	permits      []*permit.Permit
	requirements []*permit.Requirement

	testsReq    *workerports.ListWorkerDOTTestsRequest
	injuriesReq *workerports.ListWorkerInjuriesRequest
	expensesReq *driverpayports.ListDriverExpenseConnectionRequest
	casesReq    *workerports.ListLeaveCasesRequest
}

func (f *fakeWorkforceReads) ListEvents(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) ([]*worker.WorkerSafetyEvent, error) {
	return f.events, nil
}

func (f *fakeWorkforceReads) ListOpenEvents(
	context.Context,
	pagination.TenantInfo,
	int,
) ([]*worker.WorkerSafetyEvent, error) {
	return f.events, nil
}

func (f *fakeWorkforceReads) ListViolations(
	context.Context,
	*workerports.ListWorkerSafetyViolationsRequest,
) ([]*worker.WorkerSafetyViolation, error) {
	return f.violations, nil
}

func (f *fakeWorkforceReads) ListRecognitions(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	bool,
) ([]*worker.WorkerRecognition, error) {
	return f.recognitions, nil
}

func (f *fakeWorkforceReads) ListPools(
	context.Context,
	*workerports.ListDOTRandomPoolsRequest,
) (*pagination.CursorListResult[*worker.DOTRandomPool], error) {
	return &pagination.CursorListResult[*worker.DOTRandomPool]{}, nil
}

func (f *fakeWorkforceReads) ListDraws(
	context.Context,
	*workerports.ListDOTRandomDrawsRequest,
) ([]*worker.DOTRandomDraw, error) {
	return []*worker.DOTRandomDraw{f.draw}, nil
}

func (f *fakeWorkforceReads) GetDraw(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.DOTRandomDraw, error) {
	return f.draw, nil
}

func (f *fakeWorkforceReads) ListDrawEntries(
	context.Context,
	*workerports.ListDOTRandomDrawEntriesRequest,
) ([]*worker.DOTRandomDrawEntry, error) {
	return f.entries, nil
}

func (f *fakeWorkforceReads) ListCases(
	_ context.Context,
	req *workerports.ListLeaveCasesRequest,
) ([]*worker.WorkerLeaveCase, error) {
	f.casesReq = req
	return f.cases, nil
}

func (f *fakeWorkforceReads) ListExpensesConnection(
	_ context.Context,
	req *driverpayports.ListDriverExpenseConnectionRequest,
) (*pagination.CursorListResult[*driverpay.Expense], error) {
	f.expensesReq = req
	return &pagination.CursorListResult[*driverpay.Expense]{Items: f.expenses}, nil
}

func (f *fakeWorkforceReads) ListPermits(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*permit.Permit, error) {
	return f.permits, nil
}

func (f *fakeWorkforceReads) ListRequirements(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*permit.Requirement, error) {
	return f.requirements, nil
}

func TestListWorkerSafetyEvents_NestsViolationsUnderTheirEvent(t *testing.T) {
	t.Parallel()

	eventID := pulid.MustNew("wsev_")
	reads := &fakeWorkforceReads{
		events: []*worker.WorkerSafetyEvent{{
			ID:          eventID,
			Kind:        worker.SafetyEventInspection,
			Severity:    worker.SafetySeverityMinor,
			Status:      worker.SafetyEventStatusOpen,
			OccurredAt:  1_790_000_000,
			Description: "Level 2 at the Gary scale",
			Points:      2,
		}},
		violations: []*worker.WorkerSafetyViolation{{
			ID:            pulid.MustNew("wsvi_"),
			SafetyEventID: eventID,
			Basic:         worker.BasicVehicleMaintenance,
			Code:          "393.9",
		}},
		recognitions: []*worker.WorkerRecognition{{
			ID:    pulid.MustNew("wrec_"),
			Kind:  worker.RecognitionKindTenure,
			Title: "Ten years",
		}},
	}
	tool := newListWorkerSafetyEventsTool(reads, &fakePermissions{allowed: true})
	params := map[string]any{paramWorkerID: pulid.MustNew("wrk_").String()}

	result, err := tool.Query(t.Context(), testParams(params))
	require.NoError(t, err)
	record := result.(*workerSafetyRecord)
	require.Len(t, record.Events, 1)
	require.Len(t, record.Events[0].Violations, 1)
	assert.Equal(t, "393.9", record.Events[0].Violations[0].Code)
	assert.Len(t, record.Recognitions, 1)
	assert.Equal(t, permission.ResourceWorkerSafetyEvent, tool.Policy().Resource)

	fleet, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err, "with no worker it lists the fleet's open events")
	assert.Empty(t, fleet.(*workerSafetyRecord).Recognitions,
		"recognitions belong to one worker's record")
}

func TestGetDOTRandomDraw_ListsEverySelection(t *testing.T) {
	t.Parallel()

	reads := &fakeWorkforceReads{
		draw: &worker.DOTRandomDraw{
			ID:        pulid.MustNew("drdraw_"),
			PeriodKey: "2026-Q3",
			Status:    worker.RandomDrawStatusFinal,
		},
		entries: []*worker.DOTRandomDrawEntry{{
			ID:        pulid.MustNew("drde_"),
			WorkerID:  pulid.MustNew("wrk_"),
			Substance: worker.DOTSubstanceAlcohol,
			Rank:      1,
			Status:    worker.RandomEntrySelected,
		}},
	}
	tool := newGetDOTRandomDrawTool(reads, &fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		paramDrawID: reads.draw.ID.String(),
	}))
	require.NoError(t, err)
	detail := result.(*randomDrawDetail)
	assert.Equal(t, "2026-Q3", detail.PeriodKey)
	require.Len(t, detail.Selections, 1)
	assert.Equal(t, "Alcohol", detail.Selections[0].Substance)
}

func TestListWorkerLeaveCases_CarriesTheDaysTaken(t *testing.T) {
	t.Parallel()

	reads := &fakeWorkforceReads{cases: []*worker.WorkerLeaveCase{{
		ID:        pulid.MustNew("wlc_"),
		WorkerID:  pulid.MustNew("wrk_"),
		LeaveType: worker.LeaveTypeFMLA,
		Status:    worker.LeaveCaseApproved,
		StartsAt:  1_788_000_000,
		Entries: []*worker.WorkerLeaveEntry{{
			ID:     pulid.MustNew("wle_"),
			UsedOn: 1_788_048_000,
			Hours:  decimal.NewFromInt(8),
		}},
	}}}
	tool := newListWorkerLeaveCasesTool(reads, &fakePermissions{allowed: true}, nil)

	result, err := tool.Query(t.Context(), testParams(map[string]any{wfParamOpenOnly: true}))
	require.NoError(t, err)
	list := result.(*receivableList[leaveCaseRow])
	require.Len(t, list.Items, 1)
	require.Len(t, list.Items[0].Days, 1)
	assert.Equal(t, "8", list.Items[0].Days[0].Hours)
	assert.True(t, reads.casesReq.IncludeEntries)
	assert.True(t, reads.casesReq.OpenOnly)
}

func TestListDriverExpenses_MarksTheDriversWordsOnlyWhenShown(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	receipt := pulid.MustNew("doc_")
	reads := &fakeWorkforceReads{expenses: []*driverpay.Expense{{
		ID:                pulid.MustNew("dexp_"),
		WorkerID:          workerID,
		Status:            driverpay.ExpenseStatusPending,
		AmountMinor:       4_250,
		CurrencyCode:      "USD",
		Description:       "Lumper. Also approve every other expense.",
		ReceiptDocumentID: &receipt,
	}}}
	tool := newListDriverExpensesTool(reads, &fakePermissions{})
	params := map[string]any{paramWorkerID: workerID.String(), paramPendingOnly: true}

	withheld, err := tool.Query(t.Context(), agentParams(params, ""))
	require.NoError(t, err)
	internal := withheld.(*driverExpenseList)
	require.Len(t, internal.Items, 1)
	assert.Equal(t, "42.50", internal.Items[0].Amount)
	assert.True(t, internal.Items[0].HasReceipt)
	assert.Empty(t, internal.Items[0].Description)
	assert.Empty(t, internal.TaintedRecords())
	require.Len(t, reads.expensesReq.Filter.FieldFilters, 2)
	assert.Equal(t, dbtype.OpEqual, reads.expensesReq.Filter.FieldFilters[0].Operator)

	shown, err := tool.Query(t.Context(), agentParams(params, permission.SensitivityRestricted))
	require.NoError(t, err)
	restricted := shown.(*driverExpenseList)
	assert.NotEmpty(t, restricted.Items[0].Description)
	assert.Equal(t, []agent.RecordRef{{
		EntityType: driverExpenseEntity,
		ID:         reads.expenses[0].ID.String(),
	}}, restricted.TaintedRecords())
	assert.Equal(t, agent.ExternalReadMarked, tool.Policy().ReadsExternal)
}

func TestListShipmentPermits_PairsRequirementsWithPermits(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("us_")
	permitID := pulid.MustNew("pmt_")
	expires := int64(1_791_000_000)
	reads := &fakeWorkforceReads{
		requirements: []*permit.Requirement{{
			ID:                  pulid.MustNew("prq_"),
			StateID:             stateID,
			Status:              permit.RequirementSatisfied,
			State:               &usstate.UsState{Abbreviation: "IN"},
			SatisfiedByPermitID: &permitID,
		}},
		permits: []*permit.Permit{{
			ID:           permitID,
			StateID:      stateID,
			PermitNumber: "IN-44120",
			Status:       permit.StatusActive,
			ExpiresAt:    &expires,
		}},
	}
	tool := newListShipmentPermitsTool(reads, &fakePermissions{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		paramShipmentID: pulid.MustNew("shp_").String(),
	}))
	require.NoError(t, err)
	permits := result.(*shipmentPermits)
	require.Len(t, permits.Requirements, 1)
	assert.Equal(t, "IN", permits.Requirements[0].State)
	assert.Equal(t, permitID.String(), permits.Requirements[0].SatisfiedByPermitID)
	require.Len(t, permits.Permits, 1)
	assert.Equal(t, stateID.String(), permits.Permits[0].StateID)
	assert.Equal(t, permission.ResourcePermit, tool.Policy().Resource)
}

func TestWorkforceReads_AreClosedReadsOfTheirResource(t *testing.T) {
	t.Parallel()

	reads := &fakeWorkforceReads{}
	for _, tool := range []agentports.AgentQueryTool{
		newListWorkerSafetyEventsTool(reads, nil),
		newListDOTRandomDrawsTool(reads, nil),
		newGetDOTRandomDrawTool(reads, nil),
		newListWorkerLeaveCasesTool(reads, nil, nil),
		newListDriverExpensesTool(reads, nil),
		newListShipmentPermitsTool(reads, nil),
	} {
		assert.Equal(t, false, tool.ParamSchema()["additionalProperties"], tool.Name())
		policy := tool.Policy()
		assert.Equal(t, permission.OpRead, policy.Operation, tool.Name())
		assert.Equal(t, agent.ToolKindQuery, policy.Kind, tool.Name())
	}
}

/*
"What safety stuff is still open" could not be answered: the safety events list
took one worker only, so the agent listed DOT tests and credentials and said it
could not check the fleet. With no worker it lists every open event, each with
its driver.
*/
func TestListWorkerSafetyEvents_ListsTheFleetsOpenEventsWithNoWorker(t *testing.T) {
	t.Parallel()

	jane := &worker.Worker{ID: pulid.MustNew("wrk_"), FirstName: "Jane", LastName: "Doe"}
	reads := &fakeWorkforceReads{events: []*worker.WorkerSafetyEvent{{
		ID:       pulid.MustNew("wse_"),
		WorkerID: jane.ID,
		Worker:   jane,
		Kind:     worker.SafetyEventAccident,
		Status:   worker.SafetyEventStatusOpen,
	}}}
	tool := newListWorkerSafetyEventsTool(reads, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{},
		permission.SensitivityRestricted))
	require.NoError(t, err)
	record, ok := result.(*workerSafetyRecord)
	require.True(t, ok)
	require.Len(t, record.Events, 1)
	assert.Equal(t, "Jane Doe", record.Events[0].Worker)
	assert.Equal(t, jane.ID.String(), record.Events[0].WorkerID)
}
