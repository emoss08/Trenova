package settlementtools

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
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
	_ *permissionports.RequestActor,
) (*driversettlement.Settlement, error) {
	if f.locked {
		return nil, agenttoolstest.ErrPerformedDuringPreview
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
		entity := agenttoolstest.PendingDriverSettlement()
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
) *SettlementBulkTool[driversettlement.Settlement] {
	tool, _ := newSettlementBulkTool(settlementBulkSpec[driversettlement.Settlement]{
		ledger: common.DriverSettlementLedger(),
		text:   driverLifecycle(),
		book:   book,
		get:    book.get,
	}, bulk).(*SettlementBulkTool[driversettlement.Settlement])

	return tool
}

func TestSettlementBulkTools_CopyTheirSingleToolsPolicy(t *testing.T) {
	t.Parallel()

	twins := map[string]any{
		"settlement_approve_driver":  approveDriverSingle,
		"settlement_post_driver":     postDriverSingle,
		"settlement_approve_carrier": approveCarrierSingle,
		"settlement_post_carrier":    postCarrierSingle,
	}
	resources := map[string]permission.Resource{
		"settlement_approve_driver":  permission.ResourceDriverSettlement,
		"settlement_post_driver":     permission.ResourceDriverSettlement,
		"settlement_approve_carrier": permission.ResourceCarrierSettlement,
		"settlement_post_carrier":    permission.ResourceCarrierSettlement,
	}

	seen := map[string]bool{}
	for _, provider := range SettlementBulkProviders() {
		tool := agenttoolstest.BuildTool(t, provider)
		name := tool.Name()
		seen[name] = true
		singleProvider, ok := twins[name]
		require.Truef(t, ok, "%s runs no single-settlement tool", name)
		single := agenttoolstest.BuildTool(t, singleProvider)

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
		assert.Equal(t, common.MaxBulkRecords, ids[toolschema.KeyMaxItems], name)
		assert.Equal(t, 1, ids[toolschema.KeyMinItems], name)
		assert.Equal(t, []string{paramSettlementIDs},
			tool.ParamSchema()[toolschema.KeyRequired], name)

		targeted, ok := tool.(agentports.TargetedTool)
		require.Truef(t, ok, "%s pins a call naming one settlement", name)
		id := pulid.MustNew("dstl_").String()
		_, one := targeted.Target(map[string]any{paramSettlementIDs: []any{id}})
		assert.True(t, one, "%s pins one settlement", name)
		_, several := targeted.Target(map[string]any{
			paramSettlementIDs: []any{id, pulid.MustNew("dstl_").String()},
		})
		assert.False(t, several, "%s pins nothing for a set", name)
	}
	assert.Len(t, seen, len(twins))

	driverPost := agenttoolstest.BuildTool(t, providePostDriverSettlementsTool).Policy()
	assert.True(t, driverPost.HasEgress(agent.EgressDriverVisible))
	require.NotNil(t, driverPost.Classify)
	assert.Equal(t, agent.EgressMoney, driverPost.Classify(agentports.ToolExecuteParams{}).Egress)
}

func TestApproveDriverSettlements_PreviewsEachWithRefusalsFirst(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(4, driversettlement.StatusPendingApproval)
	book.settlements[ids[2]].Status = driversettlement.StatusDraft
	book.locked = true
	tool := driverBulk(book, &approveSettlements)
	params := agenttoolstest.ExecuteParams(map[string]any{paramSettlementIDs: agenttoolstest.IdList(ids)})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Empty(t, preview.Warnings, "one refusal does not refuse the whole batch")
	assert.False(t, preview.Partial)
	assert.Equal(t,
		"Would approve 4 driver settlements: 3 would be approved, 1 would be refused.",
		preview.Summary)
	require.Len(t, preview.Changes, 4)
	first := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, ids[2], first.EntityID, "the refused settlement leads")
	assert.Contains(t, agenttoolstest.FieldByPath(t, first, common.BulkOutcomeField).After, "Refused:")
	for idx := 1; idx < 4; idx++ {
		change := agenttoolstest.PreviewChange(t, preview, idx)
		assert.Equal(t, permission.ResourceDriverSettlement, change.Resource)
		assert.Equal(t, "Approved", agenttoolstest.FieldByPath(t, change, "status").After)
		assert.Contains(t, agenttoolstest.FieldByPath(t, change, common.BulkOutcomeField).After,
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
	params := agenttoolstest.ExecuteParams(map[string]any{paramSettlementIDs: agenttoolstest.IdList(ids)})

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
	params := agenttoolstest.ExecuteParams(map[string]any{paramSettlementIDs: agenttoolstest.IdList(append(ids, foreign))})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t,
		"Would post 3 driver settlements: 2 would be posted, 1 would be refused.",
		preview.Summary)
	assert.Equal(t, foreign, preview.Changes[0].EntityID)
	assert.Contains(t, agenttoolstest.FieldByPath(t, &preview.Changes[0], common.BulkOutcomeField).After, "not found")
	assert.Equal(t, "Posted", agenttoolstest.FieldByPath(t, &preview.Changes[1], "status").After)
}

func TestPostDriverSettlements_PreviewChecksTheFirstTwentyAndSaysSo(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(agent.MaxPreviewRecords+3, driversettlement.StatusApproved)
	book.locked = true
	tool := driverBulk(book, &postSettlements)

	preview, err := tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		paramSettlementIDs: agenttoolstest.IdList(ids),
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

	many := make([]any, 0, common.MaxBulkRecords+1)
	for range common.MaxBulkRecords + 1 {
		many = append(many, pulid.MustNew("dstl_").String())
	}
	err := tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{paramSettlementIDs: many}))
	require.ErrorContains(t, err, "more than the 50")

	err = tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{paramSettlementIDs: []any{}}))
	require.Error(t, err)

	params := agenttoolstest.ApprovedParams(map[string]any{paramSettlementIDs: agenttoolstest.IdList(append(ids, ids[0]))})
	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "1 of 1 posted", result.Name)
	assert.Equal(t, ids, book.performed)
}

func TestSettlementBulk_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	book, ids := driverSettlementBatch(2, driversettlement.StatusPendingApproval)
	tool := driverBulk(book, &approveSettlements)

	_, err := tool.ExecuteWithResult(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		paramSettlementIDs: agenttoolstest.IdList(ids),
	}))
	require.ErrorIs(t, err, common.ErrSettlementNeedsAPerson)
	assert.Empty(t, book.performed)

	agentParams := agenttoolstest.ApprovedParams(map[string]any{paramSettlementIDs: agenttoolstest.IdList(ids)})
	agentParams.Actor.PrincipalType = permissionports.PrincipalTypeAgent
	_, err = tool.ExecuteWithResult(t.Context(), agentParams)
	require.ErrorIs(t, err, common.ErrAgentCannotApprove)
	_, err = tool.Preview(t.Context(), agentParams)
	require.ErrorIs(t, err, common.ErrAgentCannotApprove)
	assert.Empty(t, book.performed)

	foreign := agenttoolstest.ApprovedParams(map[string]any{paramSettlementIDs: agenttoolstest.IdList(ids)})
	foreign.OrganizationID = pulid.MustNew("org_")
	_, err = tool.ExecuteWithResult(t.Context(), foreign)
	require.ErrorIs(t, err, common.ErrTenantMismatch)
	assert.Empty(t, book.performed)
}

func TestApproveDriverSettlements_ApprovesEveryApprovedSettlementAndReportsTheRefusals(
	t *testing.T,
) {
	t.Parallel()

	book, ids := driverSettlementBatch(5, driversettlement.StatusPendingApproval)
	book.failPerform[ids[3]] = errors.New("the settlement changed since it was read")
	tool := driverBulk(book, &approveSettlements)

	approved := agenttoolstest.ApprovedParams(map[string]any{paramSettlementIDs: agenttoolstest.IdList(ids[:4])})
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

	_, err := tool.ExecuteWithResult(t.Context(), agenttoolstest.ApprovedParams(map[string]any{
		paramSettlementIDs: agenttoolstest.IdList(ids),
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
	_ *permissionports.RequestActor,
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
	}, &postSettlements).(*SettlementBulkTool[carriersettlement.CarrierSettlement])
	require.True(t, ok)
	assert.Equal(t, "settlement_post_carrier", tool.Name())

	params := agenttoolstest.ExecuteParams(map[string]any{paramSettlementIDs: agenttoolstest.IdList(ids)})
	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t,
		"Would post 3 carrier settlements: 2 would be posted, 1 would be refused.",
		preview.Summary)
	assert.Equal(t, ids[1], preview.Changes[0].EntityID)
	assert.Equal(t, permission.ResourceCarrierSettlement, preview.Changes[1].Resource)

	result, err := tool.ExecuteWithResult(t.Context(), agenttoolstest.ApprovedParams(map[string]any{
		paramSettlementIDs: agenttoolstest.IdList(ids),
	}))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{ids[0], ids[2]}, book.performed)
	assert.Contains(t, result.Name, "2 of 3 posted; refused: CS-501")
}
