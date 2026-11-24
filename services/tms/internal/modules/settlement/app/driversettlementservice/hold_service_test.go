package driversettlementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type batchTotals struct {
	driversettlementports.SettlementBatchRepository
	recalculated []pulid.ID
}

func (b *batchTotals) RecalculateAggregates(
	_ context.Context,
	_ pagination.TenantInfo,
	batchID pulid.ID,
) (*driversettlement.SettlementBatch, error) {
	b.recalculated = append(b.recalculated, batchID)

	return &driversettlement.SettlementBatch{ID: batchID}, nil
}

type recordedAudit struct {
	auditports.AuditService
	operations []permission.Operation
}

func (r *recordedAudit) LogAction(
	_ context.Context,
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	r.operations = append(r.operations, params.Operation)

	return nil
}

type holdRig struct {
	svc     *Service
	store   *settlementStore
	batches *batchTotals
	audit   *recordedAudit
	entity  *driversettlement.Settlement
	tenant  pagination.TenantInfo
	actor   *permissionports.RequestActor
}

func newHoldRig(status driversettlement.Status) *holdRig {
	batchID := pulid.MustNew("dstlb_")
	entity := planFixture(status)
	entity.OrganizationID = pulid.MustNew("org_")
	entity.BusinessUnitID = pulid.MustNew("bu_")
	entity.BatchID = &batchID
	rig := &holdRig{
		store:   &settlementStore{current: entity},
		batches: &batchTotals{},
		audit:   &recordedAudit{},
		entity:  entity,
		tenant:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		actor:   &permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	}
	rig.svc = &Service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		realtime:       &mocks.NoopRealtimeService{},
		settlementRepo: rig.store,
		batchRepo:      rig.batches,
		auditService:   rig.audit,
	}

	return rig
}

func TestHoldingASettlementSavesItRefreshesItsBatchAndAuditsIt(t *testing.T) {
	t.Parallel()

	rig := newHoldRig(driversettlement.StatusApproved)
	held, err := rig.svc.ChangeHold(t.Context(), &HoldRequest{
		TenantInfo:   rig.tenant,
		SettlementID: rig.entity.ID,
		Reason:       driversettlement.HoldReasonMissingPaperwork,
		Note:         "Signed BOL not in yet",
	}, rig.actor)
	require.NoError(t, err)

	require.NotNil(t, rig.store.saved)
	assert.Equal(t, driversettlement.HoldReasonMissingPaperwork, held.HoldReason)
	assert.Equal(t, rig.actor.UserID, held.HeldByID)
	assert.Equal(t, driversettlement.StatusApproved, held.Status)
	assert.Equal(t, []pulid.ID{*rig.entity.BatchID}, rig.batches.recalculated)
	assert.Equal(t, []permission.Operation{permission.OpHold}, rig.audit.operations)
}

func TestReleasingAHoldIsAuditedAsAHoldOperation(t *testing.T) {
	t.Parallel()

	rig := newHoldRig(driversettlement.StatusApproved)
	require.NoError(t, rig.entity.PlaceHold(&driversettlement.HoldInput{
		Reason: driversettlement.HoldReasonRateDispute, UserID: pulid.MustNew("usr_"), At: 1,
	}))
	released, err := rig.svc.ChangeHold(t.Context(), &HoldRequest{
		TenantInfo:   rig.tenant,
		SettlementID: rig.entity.ID,
		Release:      true,
	}, rig.actor)
	require.NoError(t, err)

	assert.False(t, released.IsHeld())
	assert.Equal(t, rig.actor.UserID, released.HoldReleasedByID)
	assert.Len(t, rig.batches.recalculated, 1)
	assert.Equal(t, []permission.Operation{permission.OpHold}, rig.audit.operations)
}

func TestAHoldIsRefusedBeforeItIsReadWhenItsReasonIsWrong(t *testing.T) {
	t.Parallel()

	rig := newHoldRig(driversettlement.StatusApproved)
	_, err := rig.svc.ChangeHold(t.Context(), &HoldRequest{
		TenantInfo:   rig.tenant,
		SettlementID: rig.entity.ID,
		Reason:       driversettlement.HoldReasonOther,
	}, rig.actor)
	require.Error(t, err)
	assert.Nil(t, rig.store.saved)
	assert.Empty(t, rig.audit.operations)

	_, err = rig.svc.ChangeHold(t.Context(), &HoldRequest{
		TenantInfo:   rig.tenant,
		SettlementID: rig.entity.ID,
		Reason:       driversettlement.HoldReasonRateDispute,
	}, nil)
	require.Error(t, err, "a hold needs a person or a system actor behind it")
}

func TestApprovingFromTheBulkBarOrPostingAHeldSettlementSavesNothing(t *testing.T) {
	t.Parallel()

	for _, route := range []approvalRoute{{}, {fromDraft: true}, {fromDraft: true, byPolicy: true}} {
		rig := newHoldRig(driversettlement.StatusDraft)
		if !route.fromDraft {
			rig.entity.Status = driversettlement.StatusPendingApproval
		}
		require.NoError(t, rig.entity.PlaceHold(&driversettlement.HoldInput{
			Reason: driversettlement.HoldReasonRateDispute, UserID: pulid.MustNew("usr_"), At: 1,
		}))
		_, err := rig.svc.approveInternal(t.Context(), rig.tenant, rig.entity.ID, rig.actor, route)
		requireHeldRefusal(t, err)
		assert.Nil(t, rig.store.saved, "nothing is submitted or approved on the way to the refusal")
		assert.Empty(t, rig.audit.operations)
	}

	rig := newHoldRig(driversettlement.StatusApproved)
	require.NoError(t, rig.entity.PlaceHold(&driversettlement.HoldInput{
		Reason: driversettlement.HoldReasonRateDispute, UserID: pulid.MustNew("usr_"), At: 1,
	}))
	_, err := rig.svc.Post(t.Context(), rig.tenant, rig.entity.ID, rig.actor)
	requireHeldRefusal(t, err)
	assert.Nil(t, rig.store.saved)
}

func TestPlanningAHoldChangeReadsTheSettlementAndLeavesItAlone(t *testing.T) {
	t.Parallel()

	rig := newHoldRig(driversettlement.StatusPendingApproval)
	plan, err := rig.svc.PlanHoldChange(t.Context(), &HoldRequest{
		TenantInfo:   rig.tenant,
		SettlementID: rig.entity.ID,
		Reason:       driversettlement.HoldReasonDeductionReview,
	})
	require.NoError(t, err)
	require.False(t, plan.Refused())
	assert.False(t, plan.Before.IsHeld())
	assert.True(t, plan.After.IsHeld())
	assert.Nil(t, rig.store.saved)

	plan, err = rig.svc.PlanHoldChange(t.Context(), &HoldRequest{
		TenantInfo:   rig.tenant,
		SettlementID: rig.entity.ID,
		Release:      true,
	})
	require.NoError(t, err)
	assert.True(t, plan.Refused(), "a settlement that is not held cannot be released")
}
