package workerptoservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPlanCreate_CountsTheDaysWithoutFiling(t *testing.T) {
	t.Parallel()

	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			create: func(context.Context, *worker.WorkerPTO) (*worker.WorkerPTO, error) {
				t.Fatal("a plan must not file the request")
				return nil, nil
			},
		},
	}

	entity := validPTO()
	planned, err := svc.PlanCreate(t.Context(), entity, pulid.MustNew("usr_"))
	require.NoError(t, err)
	assert.True(t, planned.Days.IsPositive())
	assert.Equal(t, worker.PTOStatusRequested, planned.Status)
	assert.True(t, entity.Days.IsZero(), "the caller's request is left as it was")

	overlapping := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			hasOverlap: func(context.Context, *repositories.PTOOverlapRequest) (bool, error) {
				return true, nil
			},
		},
	}
	_, err = overlapping.PlanCreate(t.Context(), validPTO(), pulid.MustNew("usr_"))
	require.Error(t, err)
}

func TestPlanUpdate_OnlyWhileAwaitingADecision(t *testing.T) {
	t.Parallel()

	current := validPTO()
	current.ID = pulid.MustNew("wrkpto_")
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(context.Context, *repositories.GetPTOByIDRequest) (*worker.WorkerPTO, error) {
				copied := *current
				return &copied, nil
			},
		},
	}

	edit := *current
	edit.Reason = "Moved a day"
	change, err := svc.PlanUpdate(t.Context(), &edit)
	require.NoError(t, err)
	assert.Equal(t, "Moved a day", change.After.Reason)
	assert.Equal(t, "Family vacation", change.Before.Reason)

	current.Status = worker.PTOStatusApproved
	_, err = svc.PlanUpdate(t.Context(), &edit)
	require.Error(t, err, "decided time off is not edited")
}
