package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDQF struct {
	verificationKeeper

	guard        *writeGuard
	verification *worker.WorkerEmploymentVerification

	recorded  *worker.WorkerEmploymentVerification
	updated   *workerdqfservice.UpdateVerificationRequest
	requested []pulid.ID
	followUps []pulid.ID
	deleted   []pulid.ID
}

func newFakeDQF() *fakeDQF {
	return &fakeDQF{
		guard: &writeGuard{},
		verification: &worker.WorkerEmploymentVerification{
			ID:           pulid.MustNew("wev_"),
			WorkerID:     pulid.MustNew("wrk_"),
			EmployerName: "Prairie Freight Lines",
			Status:       worker.VerificationRequested,
			Version:      3,
		},
	}
}

func (f *fakeDQF) PlanRecordVerification(
	_ context.Context,
	entity *worker.WorkerEmploymentVerification,
	_ pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	if entity.EmployedTo != nil && entity.EmployedFrom != nil &&
		*entity.EmployedTo < *entity.EmployedFrom {
		return nil, errortypes.NewValidationError("employedTo", errortypes.ErrInvalid,
			"Employment cannot end before it starts")
	}
	planned := *entity
	return &planned, nil
}

func (f *fakeDQF) RecordVerification(
	ctx context.Context,
	entity *worker.WorkerEmploymentVerification,
	userID pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	planned, err := f.PlanRecordVerification(ctx, entity, userID)
	if err != nil {
		return nil, err
	}
	planned.ID = pulid.MustNew("wev_")
	f.recorded = planned
	return planned, nil
}

func (f *fakeDQF) PlanUpdateVerification(
	_ context.Context,
	req *workerdqfservice.UpdateVerificationRequest,
) (*workerdqfservice.VerificationChange, error) {
	after := *f.verification
	if req.AccidentCount != nil {
		after.AccidentCount = *req.AccidentCount
	}
	return &workerdqfservice.VerificationChange{Before: f.verification, After: &after}, nil
}

func (f *fakeDQF) UpdateVerification(
	_ context.Context,
	req *workerdqfservice.UpdateVerificationRequest,
) (*worker.WorkerEmploymentVerification, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = req
	return f.verification, nil
}

func (f *fakeDQF) PlanMarkRequested(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workerdqfservice.VerificationChange, error) {
	after := *f.verification
	after.Status = worker.VerificationRequested
	return &workerdqfservice.VerificationChange{Before: f.verification, After: &after}, nil
}

func (f *fakeDQF) MarkRequested(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.requested = append(f.requested, id)
	return f.verification, nil
}

func (f *fakeDQF) PlanRecordFollowUp(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workerdqfservice.VerificationChange, error) {
	after := *f.verification
	after.FollowUpCount++
	return &workerdqfservice.VerificationChange{Before: f.verification, After: &after}, nil
}

func (f *fakeDQF) RecordFollowUp(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.followUps = append(f.followUps, id)
	return f.verification, nil
}

func (f *fakeDQF) PlanDeleteVerification(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.WorkerEmploymentVerification, error) {
	return f.verification, nil
}

func (f *fakeDQF) DeleteVerification(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) error {
	if err := f.guard.write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func TestRecordEmploymentVerification_StartsPending(t *testing.T) {
	t.Parallel()

	dqf := newFakeDQF()
	tool := newRecordEmploymentVerificationTool(dqf)
	params := executeParams(map[string]any{
		paramWorkerID:        dqf.verification.WorkerID.String(),
		paramEmployerName:    "Lakeshore Carriers",
		paramEmployerDOT:     "1234567",
		paramEmployedFrom:    "2021-03-01",
		paramEmployedTo:      "2025-06-30",
		paramWasDOTRegulated: true,
	})

	preview := previewWithoutWrites(t, dqf.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Lakeshore Carriers")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, dqf.recorded)
	assert.Equal(t, worker.VerificationPending, dqf.recorded.Status)
	assert.Equal(t, "1234567", dqf.recorded.EmployerDOTNumber)
	assert.True(t, dqf.recorded.WasDOTRegulated)
	assert.Equal(t, permission.ResourceQualification, tool.Policy().Resource)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramWorkerID:     dqf.verification.WorkerID.String(),
			paramEmployerName: "Lakeshore Carriers",
			paramEmployedFrom: "2025-06-30",
			paramEmployedTo:   "2021-03-01",
		})), "validation runs the service's plan")
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramWorkerID: dqf.verification.WorkerID.String()})))
}

func TestUpdateEmploymentVerification_RecordsTheAnswer(t *testing.T) {
	t.Parallel()

	dqf := newFakeDQF()
	tool := newUpdateEmploymentVerificationTool(dqf)
	params := executeParams(map[string]any{
		paramVerificationID: dqf.verification.ID.String(),
		paramResponseOn:     "2026-09-24",
		paramHadAccidents:   true,
		paramAccidentCount:  float64(1),
	})

	preview := previewWithoutWrites(t, dqf.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Prairie Freight Lines")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, dqf.updated)
	require.NotNil(t, dqf.updated.AccidentCount)
	assert.Equal(t, int32(1), *dqf.updated.AccidentCount)
	require.NotNil(t, dqf.updated.ResponseReceivedAt)
	assert.Nil(t, dqf.updated.EmployerName, "what is not named is kept")
}

func TestLogEmploymentVerificationRequest_CountsFollowUps(t *testing.T) {
	t.Parallel()

	dqf := newFakeDQF()
	tool := newLogEmploymentVerificationRequestTool(dqf)
	followUp := executeParams(map[string]any{
		paramVerificationID: dqf.verification.ID.String(),
		paramRequestMove:    "FollowUp",
	})

	preview := previewWithoutWrites(t, dqf.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), followUp)
	})
	assert.Equal(t, "Would count follow-up 1 with Prairie Freight Lines.", preview.Summary)
	require.NoError(t, tool.Execute(t.Context(), followUp))
	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		paramVerificationID: dqf.verification.ID.String(),
		paramRequestMove:    "Requested",
	})))
	assert.Len(t, dqf.followUps, 1)
	assert.Len(t, dqf.requested, 1)
}

func TestDeleteEmploymentVerification_IsOnlyProposed(t *testing.T) {
	t.Parallel()

	dqf := newFakeDQF()
	tool := newDeleteEmploymentVerificationTool(dqf)
	params := executeParams(map[string]any{paramVerificationID: dqf.verification.ID.String()})

	preview := previewWithoutWrites(t, dqf.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationDelete, previewChange(t, preview, 0).Operation)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{dqf.verification.ID}, dqf.deleted)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	assert.Equal(t, permission.OpDelete, tool.Policy().Operation)
}
