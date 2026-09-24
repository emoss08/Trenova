package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSubjectRepository struct {
	exists bool
	err    error
	asked  []repositories.AgentSubjectExistsRequest
}

func (f *fakeSubjectRepository) Exists(
	_ context.Context,
	req repositories.AgentSubjectExistsRequest,
) (bool, error) {
	f.asked = append(f.asked, req)

	return f.exists, f.err
}

func raiseExceptionParams(subjectType string, subjectID pulid.ID) serviceports.ToolExecuteParams {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return serviceports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor:          agentActorFor(orgID, buID),
		RunID:          pulid.MustNew("arun_"),
		Params: map[string]any{
			"subjectType":    subjectType,
			"subjectId":      subjectID.String(),
			"category":       "Other",
			"severity":       "Medium",
			"attemptSummary": "The report cannot compute an average dwell.",
		},
	}
}

// The transcript behind this: a saved report's id filed as an Insight. The
// person approved it and the exception pointed at an insight that does not
// exist. The refusal names what the id is and the subject type to use, and
// nothing is read or recorded.
func TestRaiseException_RefusesAnIDOfAnotherKind(t *testing.T) {
	t.Parallel()

	exceptions := &fakeExceptionService{}
	subjects := &fakeSubjectRepository{exists: true}
	tool := newRaiseExceptionTool(exceptions, subjects)
	reportID := pulid.MustNew("rd_")
	params := raiseExceptionParams("Insight", reportID)

	validator, ok := tool.(serviceports.ToolValidator)
	require.True(t, ok, "raise_exception checks its subject before a proposal is raised")

	for name, run := range map[string]func() error{
		"validate": func() error { return validator.Validate(t.Context(), params) },
		"execute":  func() error { return tool.Execute(t.Context(), params) },
	} {
		err := run()
		require.Errorf(t, err, name)
		assert.Equalf(
			t,
			reportID.String()+" is a report, not an insight; use subjectType Report",
			err.Error(),
			name,
		)
	}

	assert.Empty(t, subjects.asked, "a mismatched prefix is refused before any read")
	assert.Empty(t, exceptions.flagged)
}

// An approval runs Execute long after the proposal was validated, and the
// record may be gone by then, or never have been the tenant's.
func TestRaiseException_RefusesARecordTheTenantDoesNotHold(t *testing.T) {
	t.Parallel()

	exceptions := &fakeExceptionService{}
	subjects := &fakeSubjectRepository{exists: false}
	tool := newRaiseExceptionTool(exceptions, subjects)
	reportID := pulid.MustNew("rd_")
	params := raiseExceptionParams("Report", reportID)

	err := tool.Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "there is no report "+reportID.String())
	assert.Empty(t, exceptions.flagged)

	require.Len(t, subjects.asked, 1)
	asked := subjects.asked[0]
	assert.Equal(t, agent.SubjectReport, asked.SubjectType)
	assert.Equal(t, reportID, asked.SubjectID)
	assert.Equal(t, params.OrganizationID, asked.TenantInfo.OrgID, "the tenant is the actor's")
	assert.Equal(t, params.BusinessUnitID, asked.TenantInfo.BuID)
}

func TestRaiseException_FilesACaseAboutASavedReport(t *testing.T) {
	t.Parallel()

	exceptions := &fakeExceptionService{}
	tool := newRaiseExceptionTool(exceptions, &fakeSubjectRepository{exists: true})
	reportID := pulid.MustNew("rd_")

	require.NoError(t, tool.Execute(t.Context(), raiseExceptionParams("Report", reportID)))

	require.Len(t, exceptions.flagged, 1)
	assert.Equal(t, agent.SubjectReport, exceptions.flagged[0].SubjectType)
	assert.Equal(t, reportID, exceptions.flagged[0].SubjectID)
}

func TestRaiseException_ReportsAFailedLookupRatherThanFilingBlind(t *testing.T) {
	t.Parallel()

	exceptions := &fakeExceptionService{}
	lookup := errors.New("connection reset")
	tool := newRaiseExceptionTool(
		exceptions,
		&fakeSubjectRepository{err: lookup},
	)

	err := tool.Execute(t.Context(), raiseExceptionParams("Shipment", pulid.MustNew("shp_")))
	require.ErrorIs(t, err, lookup)
	assert.Empty(t, exceptions.flagged)
}

func TestRaiseException_NamesTheKindsItAccepts(t *testing.T) {
	t.Parallel()

	tool := newRaiseExceptionTool(&fakeExceptionService{}, &fakeSubjectRepository{exists: true})

	err := tool.Execute(t.Context(), raiseExceptionParams("Spreadsheet", pulid.MustNew("shp_")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Report")
	assert.Contains(t, err.Error(), "Shipment")
}

// A feature request is not a case a person can resolve, and the description
// is where the model learns that.
func TestRaiseException_DescriptionTurnsAwayFeatureRequests(t *testing.T) {
	t.Parallel()

	description := newRaiseExceptionTool(nil, nil).Description()
	assert.Contains(t, description, "not for requesting a product feature")
	assert.Contains(t, description, "tell the person plainly what cannot be done")
}
