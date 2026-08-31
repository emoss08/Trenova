package driverpayservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAssignmentEntity() *driverpay.WorkerPayAssignment {
	return &driverpay.WorkerPayAssignment{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		WorkerID:       pulid.MustNew("wrk_"),
		PayProfileID:   pulid.MustNew("dpp_"),
		EffectiveFrom:  1_700_000_000,
		SplitPercent:   decimal.NewFromInt(100),
	}
}

func profileForAssignment(
	entity *driverpay.WorkerPayAssignment,
	componentIDs ...pulid.ID,
) *driverpay.PayProfile {
	components := make([]*driverpay.PayProfileComponent, 0, len(componentIDs))
	for _, id := range componentIDs {
		components = append(components, &driverpay.PayProfileComponent{
			ID:     id,
			Kind:   driverpay.ComponentKindLinehaul,
			Method: driverpay.CalcMethodPerLoadedMile,
			Rate:   decimal.NewFromFloat(0.55),
		})
	}
	return &driverpay.PayProfile{
		ID:             entity.PayProfileID,
		BusinessUnitID: entity.BusinessUnitID,
		OrganizationID: entity.OrganizationID,
		Name:           "OTR",
		Components:     components,
	}
}

func TestAssignProfileToWorker_RejectsOverrideForForeignComponent(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	entity := newAssignmentEntity()
	entity.RateOverrides = []driverpay.RateOverride{
		{ComponentID: pulid.MustNew("dppc_"), Rate: decimal.NewFromFloat(0.6)},
	}
	profile := profileForAssignment(entity, pulid.MustNew("dppc_"))

	svc.profileRepo = &fakeProfileRepo{
		getByID: func(context.Context, repositories.GetPayProfileByIDRequest) (*driverpay.PayProfile, error) {
			return profile, nil
		},
	}

	_, err := svc.AssignProfileToWorker(t.Context(), entity, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong")
}

func TestAssignProfileToWorker_RejectsLaterExistingAssignment(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	entity := newAssignmentEntity()
	profile := profileForAssignment(entity)

	svc.profileRepo = &fakeProfileRepo{
		getByID: func(context.Context, repositories.GetPayProfileByIDRequest) (*driverpay.PayProfile, error) {
			return profile, nil
		},
	}
	svc.assignmentRepo = &fakeAssignmentRepo{
		listOverlapping: func(context.Context, *driverpay.WorkerPayAssignment) ([]*driverpay.WorkerPayAssignment, error) {
			return []*driverpay.WorkerPayAssignment{
				{EffectiveFrom: entity.EffectiveFrom + 100},
			}, nil
		},
	}

	_, err := svc.AssignProfileToWorker(t.Context(), entity, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "starting on or after")
}

func TestAssignProfileToWorker_EndsEarlierOpenAssignment(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	entity := newAssignmentEntity()
	profile := profileForAssignment(entity)
	existing := &driverpay.WorkerPayAssignment{
		ID:            pulid.MustNew("wpa_"),
		WorkerID:      entity.WorkerID,
		EffectiveFrom: entity.EffectiveFrom - 10_000,
	}

	var endedAssignment *driverpay.WorkerPayAssignment
	var created *driverpay.WorkerPayAssignment
	svc.profileRepo = &fakeProfileRepo{
		getByID: func(context.Context, repositories.GetPayProfileByIDRequest) (*driverpay.PayProfile, error) {
			return profile, nil
		},
	}
	svc.assignmentRepo = &fakeAssignmentRepo{
		listOverlapping: func(context.Context, *driverpay.WorkerPayAssignment) ([]*driverpay.WorkerPayAssignment, error) {
			return []*driverpay.WorkerPayAssignment{existing}, nil
		},
		update: func(_ context.Context, assignment *driverpay.WorkerPayAssignment) (*driverpay.WorkerPayAssignment, error) {
			endedAssignment = assignment
			return assignment, nil
		},
		create: func(_ context.Context, assignment *driverpay.WorkerPayAssignment) (*driverpay.WorkerPayAssignment, error) {
			assignment.ID = pulid.MustNew("wpa_")
			created = assignment
			return assignment, nil
		},
	}

	actor := testActor()
	result, err := svc.AssignProfileToWorker(t.Context(), entity, actor)
	require.NoError(t, err)

	require.NotNil(t, endedAssignment)
	require.NotNil(t, endedAssignment.EffectiveTo)
	assert.Equal(t, entity.EffectiveFrom, *endedAssignment.EffectiveTo)

	require.NotNil(t, created)
	assert.Equal(t, actor.UserID, result.CreatedByID)
	assert.Len(t, audit.logged, 1)
}

func TestAssignProfileToWorker_SkipsAssignmentsAlreadyEnded(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	entity := newAssignmentEntity()
	profile := profileForAssignment(entity)
	endedAt := entity.EffectiveFrom - 100
	existing := &driverpay.WorkerPayAssignment{
		EffectiveFrom: entity.EffectiveFrom - 10_000,
		EffectiveTo:   &endedAt,
	}

	updateCalls := 0
	svc.profileRepo = &fakeProfileRepo{
		getByID: func(context.Context, repositories.GetPayProfileByIDRequest) (*driverpay.PayProfile, error) {
			return profile, nil
		},
	}
	svc.assignmentRepo = &fakeAssignmentRepo{
		listOverlapping: func(context.Context, *driverpay.WorkerPayAssignment) ([]*driverpay.WorkerPayAssignment, error) {
			return []*driverpay.WorkerPayAssignment{existing}, nil
		},
		update: func(_ context.Context, assignment *driverpay.WorkerPayAssignment) (*driverpay.WorkerPayAssignment, error) {
			updateCalls++
			return assignment, nil
		},
		create: func(_ context.Context, assignment *driverpay.WorkerPayAssignment) (*driverpay.WorkerPayAssignment, error) {
			return assignment, nil
		},
	}

	_, err := svc.AssignProfileToWorker(t.Context(), entity, testActor())
	require.NoError(t, err)
	assert.Equal(t, 0, updateCalls)
}

func TestEndAssignment_RejectsEndDateBeforeStart(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	assignment := newAssignmentEntity()
	assignment.ID = pulid.MustNew("wpa_")
	svc.assignmentRepo = &fakeAssignmentRepo{
		getByID: func(context.Context, pagination.TenantInfo, pulid.ID) (*driverpay.WorkerPayAssignment, error) {
			return assignment, nil
		},
	}

	_, err := svc.EndAssignment(
		t.Context(),
		pagination.TenantInfo{},
		assignment.ID,
		assignment.EffectiveFrom,
		testActor(),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "after the assignment")
}

func TestEndAssignment_SetsEffectiveTo(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	assignment := newAssignmentEntity()
	assignment.ID = pulid.MustNew("wpa_")
	svc.assignmentRepo = &fakeAssignmentRepo{
		getByID: func(context.Context, pagination.TenantInfo, pulid.ID) (*driverpay.WorkerPayAssignment, error) {
			return assignment, nil
		},
		update: func(_ context.Context, entity *driverpay.WorkerPayAssignment) (*driverpay.WorkerPayAssignment, error) {
			return entity, nil
		},
	}

	endDate := assignment.EffectiveFrom + 1_000
	updated, err := svc.EndAssignment(
		t.Context(),
		pagination.TenantInfo{},
		assignment.ID,
		endDate,
		testActor(),
	)
	require.NoError(t, err)
	require.NotNil(t, updated.EffectiveTo)
	assert.Equal(t, endDate, *updated.EffectiveTo)
	assert.Len(t, audit.logged, 1)
}

func newAdvanceEntity() *driverpay.PayAdvance {
	return &driverpay.PayAdvance{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		WorkerID:       pulid.MustNew("wrk_"),
		Source:         driverpay.AdvanceSourceCash,
		IssuedDate:     timeutils.NowUnix(),
		AmountMinor:    50_000,
		CurrencyCode:   "USD",
	}
}

func TestIssueAdvance_ForcesCleanRecoveryState(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	var created *driverpay.PayAdvance
	svc.advanceRepo = &fakeAdvanceRepo{
		create: func(_ context.Context, entity *driverpay.PayAdvance) (*driverpay.PayAdvance, error) {
			entity.ID = pulid.MustNew("padv_")
			created = entity
			return entity, nil
		},
	}

	entity := newAdvanceEntity()
	entity.Status = driverpay.AdvanceStatusRecovered
	entity.RecoveredMinor = 10_000
	entity.WrittenOffMinor = 5_000

	actor := testActor()
	result, err := svc.IssueAdvance(t.Context(), entity, actor)
	require.NoError(t, err)
	require.NotNil(t, created)

	assert.Equal(t, driverpay.AdvanceStatusOutstanding, result.Status)
	assert.EqualValues(t, 0, result.RecoveredMinor)
	assert.EqualValues(t, 0, result.WrittenOffMinor)
	assert.Equal(t, actor.UserID, result.CreatedByID)
	assert.Len(t, audit.logged, 1)
}

func TestIssueAdvance_RejectsInvalidEntity(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	entity := newAdvanceEntity()
	entity.AmountMinor = 0

	_, err := svc.IssueAdvance(t.Context(), entity, testActor())
	require.Error(t, err)
	assert.Empty(t, audit.logged)
}

func TestUpdateAdvance_RejectsNonOutstandingAdvance(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	previous := newAdvanceEntity()
	previous.ID = pulid.MustNew("padv_")
	previous.Status = driverpay.AdvanceStatusPartiallyRecovered
	previous.RecoveredMinor = 10_000

	svc.advanceRepo = &fakeAdvanceRepo{
		getByID: func(context.Context, repositories.GetPayAdvanceByIDRequest) (*driverpay.PayAdvance, error) {
			return previous, nil
		},
	}

	entity := newAdvanceEntity()
	entity.ID = previous.ID

	_, err := svc.UpdateAdvance(t.Context(), entity, testActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only outstanding advances")
}

func TestWriteOffAdvance_RequiresReason(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	_, err := svc.WriteOffAdvance(
		t.Context(),
		pagination.TenantInfo{},
		pulid.MustNew("padv_"),
		"",
		testActor(),
	)
	require.Error(t, err)
}

func TestWriteOffAdvance_RejectsFullyRecoveredAdvance(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService()

	advance := newAdvanceEntity()
	advance.ID = pulid.MustNew("padv_")
	advance.RecoveredMinor = advance.AmountMinor
	svc.advanceRepo = &fakeAdvanceRepo{
		getByID: func(context.Context, repositories.GetPayAdvanceByIDRequest) (*driverpay.PayAdvance, error) {
			return advance, nil
		},
	}

	_, err := svc.WriteOffAdvance(
		t.Context(),
		pagination.TenantInfo{},
		advance.ID,
		"uncollectible",
		testActor(),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no outstanding balance")
}

func TestWriteOffAdvance_WritesOffOutstandingBalance(t *testing.T) {
	t.Parallel()
	svc, audit := newTestService()

	advance := newAdvanceEntity()
	advance.ID = pulid.MustNew("padv_")
	advance.RecoveredMinor = 20_000
	svc.advanceRepo = &fakeAdvanceRepo{
		getByID: func(context.Context, repositories.GetPayAdvanceByIDRequest) (*driverpay.PayAdvance, error) {
			return advance, nil
		},
		update: func(_ context.Context, entity *driverpay.PayAdvance) (*driverpay.PayAdvance, error) {
			return entity, nil
		},
	}

	actor := testActor()
	updated, err := svc.WriteOffAdvance(
		t.Context(),
		pagination.TenantInfo{},
		advance.ID,
		"driver terminated",
		actor,
	)
	require.NoError(t, err)

	assert.EqualValues(t, 30_000, updated.WrittenOffMinor)
	assert.EqualValues(t, 0, updated.OutstandingMinor())
	assert.Equal(t, driverpay.AdvanceStatusWrittenOff, updated.Status)
	assert.Equal(t, "driver terminated", updated.WriteOffReason)
	assert.Equal(t, actor.UserID, updated.WrittenOffByID)
	require.NotNil(t, updated.WrittenOffAt)
	assert.Len(t, audit.logged, 1)
}
