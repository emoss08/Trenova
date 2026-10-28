package workforcetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInjuries struct {
	InjuryKeeper

	guard  *agenttoolstest.WriteGuard
	injury *workersafety.WorkerInjury

	recorded *workersafety.WorkerInjury
	updated  *workerinjuryservice.UpdateInjuryRequest
	deleted  []pulid.ID
}

func newFakeInjuries() *fakeInjuries {
	return &fakeInjuries{
		guard: &agenttoolstest.WriteGuard{},
		injury: &workersafety.WorkerInjury{
			ID:             pulid.MustNew("winj_"),
			WorkerID:       pulid.MustNew("wrk_"),
			CaseYear:       2026,
			CaseNumber:     7,
			Classification: workersafety.CaseOtherRecordable,
			Status:         workersafety.InjuryCaseOpen,
			Description:    "Strained back unloading",
			BodyPart:       "Lower back",
			Version:        2,
		},
	}
}

func (f *fakeInjuries) PlanRecordInjury(
	_ context.Context,
	entity *workersafety.WorkerInjury,
	_ pulid.ID,
) (*workersafety.WorkerInjury, error) {
	if entity.Classification == workersafety.CaseDaysAway && entity.DaysAway == 0 {
		return nil, errortypes.NewValidationError("daysAway", errortypes.ErrRequired,
			"A days-away case counts the days away")
	}
	planned := *entity
	planned.CaseYear = 2026
	planned.CaseNumber = 8
	if planned.Classification == "" {
		planned.Classification = workersafety.CaseFirstAidOnly
	}
	return &planned, nil
}

func (f *fakeInjuries) RecordInjury(
	ctx context.Context,
	entity *workersafety.WorkerInjury,
	userID pulid.ID,
) (*workersafety.WorkerInjury, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	planned, err := f.PlanRecordInjury(ctx, entity, userID)
	if err != nil {
		return nil, err
	}
	planned.ID = pulid.MustNew("winj_")
	f.recorded = planned
	return planned, nil
}

func (f *fakeInjuries) PlanUpdateInjury(
	_ context.Context,
	req *workerinjuryservice.UpdateInjuryRequest,
) (*workerinjuryservice.InjuryChange, error) {
	after := *f.injury
	workerinjuryservice.ApplyInjuryUpdate(&after, req)
	return &workerinjuryservice.InjuryChange{Before: f.injury, After: &after}, nil
}

func (f *fakeInjuries) UpdateInjury(
	_ context.Context,
	req *workerinjuryservice.UpdateInjuryRequest,
) (*workersafety.WorkerInjury, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = req
	return f.injury, nil
}

func (f *fakeInjuries) PlanDeleteInjury(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*workersafety.WorkerInjury, error) {
	return f.injury, nil
}

func (f *fakeInjuries) DeleteInjury(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
	_ pulid.ID,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func TestRecordWorkerInjury_TakesTheNextCaseNumber(t *testing.T) {
	t.Parallel()

	injuries := newFakeInjuries()
	tool := newRecordWorkerInjuryTool(injuries)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamWorkerID:    injuries.injury.WorkerID.String(),
		wfParamOccurred:         "2026-09-22T06:15:00-05:00",
		common.FieldDescription: "Cut hand on a load strap",
		paramTreatment:          "FirstAid",
		paramBodyPart:           "Right hand",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		injuries.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "case 2026-8")
	assert.Contains(t, preview.Summary, "FirstAidOnly")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, injuries.recorded)
	assert.Equal(t, "Right hand", injuries.recorded.BodyPart)
	assert.Equal(t, int64(1_790_075_700), injuries.recorded.OccurredAt)
	assert.Equal(t, permission.ResourceWorkerInjury, tool.Policy().Resource)

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			common.ParamWorkerID:    injuries.injury.WorkerID.String(),
			wfParamOccurred:         "2026-09-22T06:15:00-05:00",
			common.FieldDescription: "Fractured wrist",
			paramClassification:     "DaysAway",
		})), "validation runs the service's plan")
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			common.ParamWorkerID: injuries.injury.WorkerID.String(),
			wfParamOccurred:      "2026-09-22T06:15:00-05:00",
		})), "a case says what happened")
}

func TestUpdateWorkerInjury_ChangesOnlyWhatIsNamed(t *testing.T) {
	t.Parallel()

	injuries := newFakeInjuries()
	tool := newUpdateWorkerInjuryTool(injuries)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramInjuryID:      injuries.injury.ID.String(),
		"returnedToWorkAt": "2026-09-29",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		injuries.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(t, "Would correct case 2026-7.", preview.Summary)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, injuries.updated)
	require.NotNil(t, injuries.updated.ReturnedToWorkAt)
	assert.Nil(t, injuries.updated.BodyPart)
	assert.Nil(t, injuries.updated.Classification)
}

func TestDeleteWorkerInjury_IsOnlyProposed(t *testing.T) {
	t.Parallel()

	injuries := newFakeInjuries()
	tool := newDeleteWorkerInjuryTool(injuries)
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramInjuryID: injuries.injury.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		injuries.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agent.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{injuries.injury.ID}, injuries.deleted)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
}
