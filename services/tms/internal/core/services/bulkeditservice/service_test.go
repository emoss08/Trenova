package bulkeditservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/bulkedit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingEditor struct {
	calls   []services.BulkEditApplyRequest
	outcome func(id pulid.ID, value string) services.BulkEditOutcome
}

func (e *recordingEditor) Resource() permission.Resource { return permission.ResourceCustomer }

func (e *recordingEditor) Fields() []services.BulkEditField {
	return []services.BulkEditField{{Name: "status"}}
}

func (e *recordingEditor) Apply(
	_ context.Context,
	req *services.BulkEditApplyRequest,
) ([]services.BulkEditOutcome, error) {
	e.calls = append(e.calls, *req)
	outcomes := make([]services.BulkEditOutcome, 0, len(req.IDs))
	for _, id := range req.IDs {
		outcomes = append(outcomes, e.outcome(id, req.Value))
	}
	return outcomes, nil
}

func TestStatusEditor_ChangesOnlyRowsThatDifferAndReportsMissingOnes(t *testing.T) {
	var applied []pulid.ID
	editor := &statusEditor[domaintypes.Status]{
		resource: permission.ResourceCustomer,
		valid:    domaintypes.Status.IsValid,
		load: func(_ context.Context, _ pagination.TenantInfo, _ []pulid.ID) (map[pulid.ID]domaintypes.Status, error) {
			return map[pulid.ID]domaintypes.Status{
				"cus_active":   domaintypes.StatusActive,
				"cus_inactive": domaintypes.StatusInactive,
			}, nil
		},
		apply: func(_ context.Context, _ pagination.TenantInfo, ids []pulid.ID, _ domaintypes.Status) error {
			applied = append(applied, ids...)
			return nil
		},
	}

	outcomes, err := editor.Apply(t.Context(), &services.BulkEditApplyRequest{
		IDs:   []pulid.ID{"cus_active", "cus_inactive", "cus_gone"},
		Field: "status",
		Value: string(domaintypes.StatusInactive),
	})

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{"cus_active"}, applied)
	byID := map[pulid.ID]services.BulkEditOutcome{}
	for _, outcome := range outcomes {
		byID[outcome.ID] = outcome
	}
	assert.True(t, byID["cus_active"].Changed)
	assert.Equal(t, "Active", byID["cus_active"].Previous)
	assert.False(t, byID["cus_inactive"].Changed)
	assert.NoError(t, byID["cus_inactive"].Err)
	assert.ErrorIs(t, byID["cus_gone"].Err, ErrRecordNotFound)
}

func TestStatusEditor_RefusesAStatusTheRecordCannotHave(t *testing.T) {
	editor := &statusEditor[domaintypes.Status]{valid: domaintypes.Status.IsValid}

	_, err := editor.Apply(t.Context(), &services.BulkEditApplyRequest{Field: "status", Value: "Archived"})

	require.Error(t, err)
}

func TestStatusEditor_MarksEveryRowOfAFailedChunk(t *testing.T) {
	boom := errors.New("boom")
	editor := &statusEditor[domaintypes.Status]{
		valid: domaintypes.Status.IsValid,
		load: func(_ context.Context, _ pagination.TenantInfo, ids []pulid.ID) (map[pulid.ID]domaintypes.Status, error) {
			current := map[pulid.ID]domaintypes.Status{}
			for _, id := range ids {
				current[id] = domaintypes.StatusActive
			}
			return current, nil
		},
		apply: func(context.Context, pagination.TenantInfo, []pulid.ID, domaintypes.Status) error {
			return boom
		},
	}

	outcomes, err := editor.Apply(t.Context(), &services.BulkEditApplyRequest{
		IDs:   []pulid.ID{"cus_1", "cus_2"},
		Field: "status",
		Value: string(domaintypes.StatusInactive),
	})

	require.NoError(t, err)
	for _, outcome := range outcomes {
		assert.ErrorIs(t, outcome.Err, boom)
		assert.False(t, outcome.Changed)
	}
}

func TestService_UndoPutsEachRowBackToItsOwnPreviousValue(t *testing.T) {
	editor := &recordingEditor{outcome: func(id pulid.ID, _ string) services.BulkEditOutcome {
		return services.BulkEditOutcome{ID: id, Changed: true}
	}}
	edit := &bulkedit.BulkEdit{
		Field:  "status",
		Status: bulkedit.StatusUndoing,
		Targets: []bulkedit.Target{
			{ID: "cus_1", Previous: "Active"},
			{ID: "cus_2", Previous: "Inactive"},
			{ID: "cus_3", Previous: "Active"},
		},
	}

	(&Service{}).undoBatch(t.Context(), editor, edit, edit.Targets, pagination.TenantInfo{}, nil)

	require.Len(t, editor.calls, 2)
	assert.Equal(t, "Active", editor.calls[0].Value)
	assert.Equal(t, []pulid.ID{"cus_1", "cus_3"}, editor.calls[0].IDs)
	assert.Equal(t, "Inactive", editor.calls[1].Value)
	assert.Equal(t, []pulid.ID{"cus_2"}, editor.calls[1].IDs)
	assert.Equal(t, 3, edit.ChangedCount)
	for _, target := range edit.Targets {
		assert.True(t, target.Done)
		assert.NotEmpty(t, target.Previous, "undo keeps what the row was put back to")
	}
}

func TestRecordOutcomes_CountsChangesAndFailuresAndKeepsPreviousValues(t *testing.T) {
	edit := &bulkedit.BulkEdit{
		Targets: []bulkedit.Target{{ID: "cus_1"}, {ID: "cus_2"}, {ID: "cus_3"}, {ID: "cus_4"}},
	}

	recordOutcomes(edit, edit.Targets, []services.BulkEditOutcome{
		{ID: "cus_1", Previous: "Active", Changed: true},
		{ID: "cus_2", Previous: "Inactive"},
		{ID: "cus_3", Err: errors.New("locked")},
	}, nil, true)

	assert.Equal(t, 1, edit.ChangedCount)
	assert.Equal(t, 2, edit.FailedCount)
	assert.Equal(t, "Active", edit.Targets[0].Previous)
	assert.False(t, edit.Targets[1].Changed)
	assert.Equal(t, "locked", edit.Targets[2].Error)
	assert.Equal(t, "The record was not changed", edit.Targets[3].Error)
}
