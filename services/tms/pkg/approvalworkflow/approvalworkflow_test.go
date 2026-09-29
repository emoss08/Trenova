package approvalworkflow_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/approvalworkflow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type record struct {
	ID      pulid.ID
	Status  string
	Comment string
}

type store struct {
	stored  *record
	saves   int
	audited int
}

func (s *store) engine() approvalworkflow.Engine[*record, string] {
	return approvalworkflow.Engine[*record, string]{
		Label: "record",
		Load: func(context.Context, pulid.ID, pagination.TenantInfo) (*record, error) {
			return s.stored, nil
		},
		Save: func(_ context.Context, entity *record) (*record, error) {
			s.saves++
			s.stored = entity

			return entity, nil
		},
		StatusOf:      func(entity *record) string { return entity.Status },
		SetStatus:     func(entity *record, status string) { entity.Status = status },
		CanTransition: func(from, to string) bool { return from == "Draft" && to == "InReview" },
		Snapshot: func(entity *record) *record {
			copied := *entity

			return &copied
		},
		Audit: func(_, _ *record, _ permission.Operation, _ *approvalworkflow.Request, _ string) {
			s.audited++
		},
		Now: func() int64 { return 42 },
	}
}

func submit() approvalworkflow.Transition[*record, string] {
	return approvalworkflow.Transition[*record, string]{
		Operation:    "Submit",
		From:         "Draft",
		To:           "InReview",
		PermissionOp: permission.OpSubmit,
		Apply: func(entity *record, req *approvalworkflow.Request, _ int64) {
			entity.Comment = req.Comment
		},
	}
}

func TestPlan_ChangesACopyAndSavesNothing(t *testing.T) {
	t.Parallel()

	loaded := &record{ID: pulid.MustNew("rec_"), Status: "Draft"}
	s := &store{stored: loaded}
	req := &approvalworkflow.Request{EntityID: loaded.ID, Comment: "Ready"}

	change, err := s.engine().Plan(t.Context(), req, submit())
	require.NoError(t, err)
	assert.Equal(t, "Draft", change.Before.Status)
	assert.Equal(t, "InReview", change.After.Status)
	assert.Equal(t, "Ready", change.After.Comment)
	assert.Equal(t, "Draft", loaded.Status)
	assert.Zero(t, s.saves)
	assert.Zero(t, s.audited)
}

func TestPlan_RefusesAnIllegalMove(t *testing.T) {
	t.Parallel()

	loaded := &record{ID: pulid.MustNew("rec_"), Status: "InReview"}
	s := &store{stored: loaded}

	_, err := s.engine().Plan(t.Context(), &approvalworkflow.Request{EntityID: loaded.ID}, submit())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot transition record status")
}

func TestApply_SavesThePlannedChange(t *testing.T) {
	t.Parallel()

	loaded := &record{ID: pulid.MustNew("rec_"), Status: "Draft"}
	s := &store{stored: loaded}
	req := &approvalworkflow.Request{EntityID: loaded.ID, Comment: "Ready"}

	updated, err := s.engine().Apply(t.Context(), req, submit())
	require.NoError(t, err)
	assert.Equal(t, "InReview", updated.Status)
	assert.Equal(t, 1, s.saves)
	assert.Equal(t, 1, s.audited)
}
