package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDriverSettlementBatch struct {
	settlements map[pulid.ID]*driversettlement.Settlement
	failPerform map[pulid.ID]error
	performed   []pulid.ID
	actions     []settlementshared.Action
	locked      bool
}

func (f *fakeDriverSettlementBatch) PlanAction(
	_ context.Context,
	req *settlementshared.ActionRequest,
) (*settlementshared.ActionPlan[*driversettlement.Settlement], error) {
	entity, ok := f.settlements[req.SettlementID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Driver settlement not found")
	}

	return driverPlanner(entity, nil)(req), nil
}

func (f *fakeDriverSettlementBatch) Perform(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	_ *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if f.locked {
		return nil, errPerformedDuringPreview
	}
	if err := f.failPerform[req.SettlementID]; err != nil {
		return nil, err
	}
	plan, err := f.PlanAction(ctx, req)
	if err != nil {
		return nil, err
	}
	if plan.Refusal != nil {
		return nil, plan.Refusal
	}
	f.performed = append(f.performed, req.SettlementID)
	f.actions = append(f.actions, req.Action)

	return plan.After, nil
}

func (f *fakeDriverSettlementBatch) get(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*driversettlement.Settlement, error) {
	entity, ok := f.settlements[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("Driver settlement not found")
	}

	return entity, nil
}

func driverSettlementBatch(
	n int,
	status driversettlement.Status,
) (*fakeDriverSettlementBatch, []pulid.ID) {
	book := &fakeDriverSettlementBatch{
		settlements: make(map[pulid.ID]*driversettlement.Settlement, n),
		failPerform: map[pulid.ID]error{},
	}
	ids := make([]pulid.ID, 0, n)
	for i := range n {
		entity := pendingDriverSettlement()
		entity.Status = status
		entity.SettlementNumber = fmt.Sprintf("DS-%d", 3000+i)
		book.settlements[entity.ID] = entity
		ids = append(ids, entity.ID)
	}

	return book, ids
}

func driverBulk(
	book *fakeDriverSettlementBatch,
	bulk *settlementBulk,
) *settlementBulkTool[driversettlement.Settlement] {
	tool, _ := newSettlementBulkTool(settlementBulkSpec[driversettlement.Settlement]{
		ledger: driverSettlementLedger(),
		text:   driverLifecycle(),
		book:   book,
		get:    book.get,
	}, bulk).(*settlementBulkTool[driversettlement.Settlement])

	return tool
}

func TestSettlementBulkTools_CopyTheirSingleToolsPolicy(t *testing.T) {
	t.Parallel()

	twins := map[string]any{
		"approve_driver_settlement":  provideApproveDriverSettlementTool,
		"post_driver_settlement":     providePostDriverSettlementTool,
		"approve_carrier_settlement": provideApproveCarrierSettlementTool,
		"post_carrier_settlement":    providePostCarrierSettlementTool,
	}
	resources := map[string]permission.Resource{
		"approve_driver_settlements":  permission.ResourceDriverSettlement,
		"post_driver_settlements":     permission.ResourceDriverSettlement,
		"approve_carrier_settlements": permission.ResourceCarrierSettlement,
		"post_carrier_settlements":    permission.ResourceCarrierSettlement,
	}

	seen := map[string]bool{}
	for _, provider := range settlementBulkProviders() {
		tool := buildTool(t, provider)
		name := tool.Name()
		seen[name] = true
		singleName := name[:len(name)-1]
		singleProvider, ok := twins[singleName]
		require.Truef(t, ok, "%s has no single twin", name)
		single := buildTool(t, singleProvider)

		policy := tool.Policy()
		want := single.Policy()
		assert.Equal(t, name, policy.Name)
		assert.Equal(t, want.Kind, policy.Kind, name)
		assert.Equal(t, want.Resource, policy.Resource, name)
		assert.Equal(t, resources[name], policy.Resource, name)
		assert.Equal(t, want.Operation, policy.Operation, name)
		assert.Equal(t, permission.OpApprove, policy.Operation, name)
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, name)
		assert.Equal(t, agent.TierPropose, policy.MaxTier, name)
		assert.Equal(t, want.Egress, policy.Egress, name)
		assert.True(t, policy.HasEgress(agent.EgressMoney), name)
		assert.False(t, policy.Reversible, name)
		assert.Equal(t, want.Classify != nil, policy.Classify != nil, name)
		assert.NotEqual(t, want.Rationale, policy.Rationale, name)
		assert.Equal(t, resources[name].String(), policy.Artifact,
			"%s reports the settlements it changed as record links", name)

		properties := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
		ids := properties[paramSettlementIDs].(map[string]any)
		assert.Equal(t, resources[name].String(), toolschema.SubsetResource(ids), name)
		assert.Equal(t, maxBulkRecords, ids[toolschema.KeyMaxItems], name)
		assert.Equal(t, 1, ids[toolschema.KeyMinItems], name)
		assert.Equal(t, []string{paramSettlementIDs},
			tool.ParamSchema()[toolschema.KeyRequired], name)

		_, targeted := tool.(serviceports.TargetedTool)
		assert.False(t, targeted, "%s acts on a set, not one record", name)
		assert.Contains(t, single.Description(), name,
			"%s points to its bulk twin", singleName)
	}
	assert.Len(t, seen, len(twins))

	driverPost := buildTool(t, providePostDriverSettlementsTool).Policy()
	assert.True(t, driverPost.HasEgress(agent.EgressDriverVisible))
	require.NotNil(t, driverPost.Classify)
	assert.Equal(t, agent.EgressMoney, driverPost.Classify(serviceports.ToolExecuteParams{}).Egress)
}

func TestApproveDriverSettlements_PreviewsEachWithRefusalsFirst(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(4, driversettlement.StatusPendingApproval)
	book.settlements[ids[2]].Status = driversettlement.StatusDraft
	book.locked = true
	tool := driverBulk(book, &approveSettlements)
	params := executeParams(map[string]any{paramSettlementIDs: idList(ids)})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Empty(t, preview.Warnings, "one refusal does not refuse the whole batch")
	assert.False(t, preview.Partial)
	assert.Equal(t,
		"Would approve 4 driver settlements: 3 would be approved, 1 would be refused.",
		preview.Summary)
	require.Len(t, preview.Changes, 4)
	first := previewChange(t, preview, 0)
	assert.Equal(t, ids[2], first.EntityID, "the refused settlement leads")
	assert.Contains(t, fieldByPath(t, first, bulkOutcomeField).After, "Refused:")
	for idx := 1; idx < 4; idx++ {
		change := previewChange(t, preview, idx)
		assert.Equal(t, permission.ResourceDriverSettlement, change.Resource)
		assert.Equal(t, "Approved", fieldByPath(t, change, "status").After)
		assert.Contains(t, fieldByPath(t, change, bulkOutcomeField).After,
			"Would approve driver settlement DS-")
		require.NotNil(t, change.Money)
	}
	require.NoError(t, tool.Validate(t.Context(), params))
}

func TestApproveDriverSettlements_ABatchNothingOfWhichWouldGoIsRefused(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(2, driversettlement.StatusDraft)
	book.locked = true
	tool := driverBulk(book, &approveSettlements)
	params := executeParams(map[string]any{paramSettlementIDs: idList(ids)})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, preview.Refusal())

	err = tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "none of the driver settlements would be approved")
}

func TestPostDriverSettlements_AForeignSettlementIsRefusedOnItsOwn(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(2, driversettlement.StatusApproved)
	book.locked = true
	foreign := pulid.MustNew("dstl_")
	tool := driverBulk(book, &postSettlements)
	params := executeParams(map[string]any{paramSettlementIDs: idList(append(ids, foreign))})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t,
		"Would post 3 driver settlements: 2 would be posted, 1 would be refused.",
		preview.Summary)
	assert.Equal(t, foreign, preview.Changes[0].EntityID)
	assert.Contains(t, fieldByPath(t, &preview.Changes[0], bulkOutcomeField).After, "not found")
	assert.Equal(t, "Posted", fieldByPath(t, &preview.Changes[1], "status").After)
}

func TestPostDriverSettlements_PreviewChecksTheFirstTwentyAndSaysSo(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(agent.MaxPreviewRecords+3, driversettlement.StatusApproved)
	book.locked = true
	tool := driverBulk(book, &postSettlements)

	preview, err := tool.Preview(t.Context(), executeParams(map[string]any{
		paramSettlementIDs: idList(ids),
	}))
	require.NoError(t, err)

	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary,
		"Would post 23 driver settlements. Of the first 20 checked")
	assert.Len(t, preview.Changes, agent.MaxPreviewRecords)
}

func TestSettlementBulk_RefusesMoreThanFiftyAndDropsRepeats(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(1, driversettlement.StatusApproved)
	tool := driverBulk(book, &postSettlements)

	many := make([]any, 0, maxBulkRecords+1)
	for range maxBulkRecords + 1 {
		many = append(many, pulid.MustNew("dstl_").String())
	}
	err := tool.Validate(t.Context(), executeParams(map[string]any{paramSettlementIDs: many}))
	require.ErrorContains(t, err, "more than the 50")

	err = tool.Validate(t.Context(), executeParams(map[string]any{paramSettlementIDs: []any{}}))
	require.Error(t, err)

	params := approvedParams(map[string]any{paramSettlementIDs: idList(append(ids, ids[0]))})
	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "1 of 1 posted", result.Name)
	assert.Equal(t, ids, book.performed)
}

func TestSettlementBulk_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(2, driversettlement.StatusPendingApproval)
	tool := driverBulk(book, &approveSettlements)

	_, err := tool.ExecuteWithResult(t.Context(), executeParams(map[string]any{
		paramSettlementIDs: idList(ids),
	}))
	require.ErrorIs(t, err, ErrSettlementNeedsAPerson)
	assert.Empty(t, book.performed)

	agentParams := approvedParams(map[string]any{paramSettlementIDs: idList(ids)})
	agentParams.Actor.PrincipalType = serviceports.PrincipalTypeAgent
	_, err = tool.ExecuteWithResult(t.Context(), agentParams)
	require.ErrorIs(t, err, ErrAgentCannotApprove)
	_, err = tool.Preview(t.Context(), agentParams)
	require.ErrorIs(t, err, ErrAgentCannotApprove)
	assert.Empty(t, book.performed)

	foreign := approvedParams(map[string]any{paramSettlementIDs: idList(ids)})
	foreign.OrganizationID = pulid.MustNew("org_")
	_, err = tool.ExecuteWithResult(t.Context(), foreign)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Empty(t, book.performed)
}

func TestApproveDriverSettlements_ApprovesEveryApprovedSettlementAndReportsTheRefusals(
	t *testing.T,
) {
	t.Parallel()

	book, ids := driverSettlementBatch(5, driversettlement.StatusPendingApproval)
	book.failPerform[ids[3]] = errors.New("the settlement changed since it was read")
	tool := driverBulk(book, &approveSettlements)

	approved := approvedParams(map[string]any{paramSettlementIDs: idList(ids[:4])})
	result, err := tool.ExecuteWithResult(t.Context(), approved)
	require.NoError(t, err)

	assert.Equal(t, ids[:3], book.performed, "only the approved settlements are approved")
	assert.NotContains(t, book.performed, ids[4], "an unticked settlement is never approved")
	for _, action := range book.actions {
		assert.Equal(t, settlementshared.ActionApprove, action)
	}
	assert.Equal(t, "approved", result.Action)
	assert.Equal(t, "driver settlements", result.Kind)
	assert.Equal(t,
		"3 of 4 approved; refused: DS-3003 (the settlement changed since it was read)",
		result.Name)
}

func TestPostDriverSettlements_ABatchThatPostsNothingFails(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(2, driversettlement.StatusDraft)
	tool := driverBulk(book, &postSettlements)

	_, err := tool.ExecuteWithResult(t.Context(), approvedParams(map[string]any{
		paramSettlementIDs: idList(ids),
	}))
	require.ErrorContains(t, err, "0 of 2 posted")
	assert.Empty(t, book.performed)
}

type fakeCarrierSettlementBatch struct {
	settlements map[pulid.ID]*carriersettlement.CarrierSettlement
	performed   []pulid.ID
}

func (f *fakeCarrierSettlementBatch) PlanAction(
	_ context.Context,
	req *settlementshared.ActionRequest,
) (*settlementshared.ActionPlan[*carriersettlement.CarrierSettlement], error) {
	entity, ok := f.settlements[req.SettlementID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Carrier settlement not found")
	}
	plan := &settlementshared.ActionPlan[*carriersettlement.CarrierSettlement]{
		Before: entity,
		After:  carriersettlementservice.CloneSettlement(entity),
	}
	plan.Refusal = carriersettlementservice.PlanPost(
		plan.After,
		req.TenantInfo.UserID,
		1_790_000_000,
	)

	return plan, nil
}

func (f *fakeCarrierSettlementBatch) Perform(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	_ *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
	plan, err := f.PlanAction(ctx, req)
	if err != nil {
		return nil, err
	}
	if plan.Refusal != nil {
		return nil, plan.Refusal
	}
	f.performed = append(f.performed, req.SettlementID)

	return plan.After, nil
}

func TestPostCarrierSettlements_PostsEachAsPostCarrierSettlementWould(t *testing.T) {
	t.Parallel()

	book := &fakeCarrierSettlementBatch{
		settlements: map[pulid.ID]*carriersettlement.CarrierSettlement{},
	}
	ids := make([]pulid.ID, 0, 3)
	for i := range 3 {
		entity := &carriersettlement.CarrierSettlement{
			ID:               pulid.MustNew("carstl_"),
			SettlementNumber: fmt.Sprintf("CS-%d", 500+i),
			Status:           carriersettlement.StatusApproved,
			CurrencyCode:     "USD",
			GrossCostMinor:   250000,
		}
		book.settlements[entity.ID] = entity
		ids = append(ids, entity.ID)
	}
	book.settlements[ids[1]].Status = carriersettlement.StatusPaid
	tool, ok := newSettlementBulkTool(settlementBulkSpec[carriersettlement.CarrierSettlement]{
		ledger: carrierSettlementLedger(),
		text:   carrierLifecycle(),
		book:   book,
		get: func(
			_ context.Context,
			_ pagination.TenantInfo,
			id pulid.ID,
		) (*carriersettlement.CarrierSettlement, error) {
			return book.settlements[id], nil
		},
	}, &postSettlements).(*settlementBulkTool[carriersettlement.CarrierSettlement])
	require.True(t, ok)
	assert.Equal(t, "post_carrier_settlements", tool.Name())

	params := executeParams(map[string]any{paramSettlementIDs: idList(ids)})
	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t,
		"Would post 3 carrier settlements: 2 would be posted, 1 would be refused.",
		preview.Summary)
	assert.Equal(t, ids[1], preview.Changes[0].EntityID)
	assert.Equal(t, permission.ResourceCarrierSettlement, preview.Changes[1].Resource)

	result, err := tool.ExecuteWithResult(t.Context(), approvedParams(map[string]any{
		paramSettlementIDs: idList(ids),
	}))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{ids[0], ids[2]}, book.performed)
	assert.Contains(t, result.Name, "2 of 3 posted; refused: CS-501")
}
