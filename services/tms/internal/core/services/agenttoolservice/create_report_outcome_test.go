package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// namingReportWriter saves a report under a known id and the name it was
// asked for, so a test can check what the tool reports back.
type namingReportWriter struct {
	fakeReportWriter

	id  pulid.ID
	err error
}

func (w *namingReportWriter) CreateDefinition(
	_ context.Context,
	req *reporting.SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	w.created = req
	if w.err != nil {
		return nil, w.err
	}

	return &report.ReportDefinition{ID: w.id, Name: req.Name}, nil
}

func personParams(args map[string]any) serviceports.ToolExecuteParams {
	params := executeParams(args)
	params.Actor.PrincipalType = serviceports.PrincipalTypeUser
	params.Actor.PrincipalID = params.Actor.UserID

	return params
}

/*
A private report is the person's own saved query, so it is saved as asked.

The owner decided that saving a report nobody else sees needs no approval: a
card in front of a report the person just asked for only stood between them
and it. A report shared with the whole organization still waits for a person,
and so does a call no person is driving, because "private" is private to
someone.
*/
func TestCreateReport_TierLimitRunsPrivateReportsAndHoldsSharedOnes(t *testing.T) {
	t.Parallel()

	tool := newCreateReportTool(&fakeReportWriter{})
	limiter, ok := tool.(serviceports.ToolTierLimiter)
	require.True(t, ok)

	base := func() map[string]any {
		return map[string]any{"name": "Lane revenue", "definition": definitionArgument()}
	}
	withVisibility := func(visibility any) map[string]any {
		args := base()
		args["visibility"] = visibility

		return args
	}

	cases := []struct {
		name   string
		params serviceports.ToolExecuteParams
		want   agent.AutonomyTier
	}{
		{
			name:   "no visibility is private by default",
			params: personParams(base()),
			want:   agent.TierAutoExecute,
		},
		{
			name:   "private",
			params: personParams(withVisibility(string(report.VisibilityPrivate))),
			want:   agent.TierAutoExecute,
		},
		{
			name:   "shared with the organization",
			params: personParams(withVisibility(string(report.VisibilityShared))),
			want:   agent.TierActWithApproval,
		},
		{
			name:   "visibility that cannot be read",
			params: personParams(withVisibility("everyone")),
			want:   agent.TierActWithApproval,
		},
		{
			name: "an agent principal, not a person",
			params: func() serviceports.ToolExecuteParams {
				params := personParams(base())
				params.Actor.PrincipalType = serviceports.PrincipalTypeAgent

				return params
			}(),
			want: agent.TierActWithApproval,
		},
		{
			name: "a person with no user",
			params: func() serviceports.ToolExecuteParams {
				params := personParams(base())
				params.Actor.UserID = pulid.Nil

				return params
			}(),
			want: agent.TierActWithApproval,
		},
		{
			name: "no actor",
			params: func() serviceports.ToolExecuteParams {
				params := personParams(base())
				params.Actor = nil

				return params
			}(),
			want: agent.TierActWithApproval,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, limiter.TierLimit(t.Context(), tc.params))
		})
	}
}

/*
Saving a report says which report it saved.

An approved create_report used to leave only "it ran", and the agent then
passed the proposal's id to describe_report. The result names the report and
its id under the parameter describe_report, run_report and update_report take.
*/
func TestCreateReport_ExecuteWithResultNamesTheSavedReport(t *testing.T) {
	t.Parallel()

	writer := &namingReportWriter{id: pulid.MustNew("rd_")}
	tool := newCreateReportTool(writer)

	result, err := serviceports.ExecuteTool(t.Context(), tool, personParams(map[string]any{
		"name":       "Shipments for Peak Distributing",
		"definition": definitionArgument(),
	}))

	require.NoError(t, err)
	require.NotNil(t, writer.created)
	require.NotNil(t, result)
	assert.Equal(t, "created", result.Action)
	assert.Equal(t, "report", result.Kind)
	assert.Equal(t, "Shipments for Peak Distributing", result.Name)
	assert.Equal(t, map[string]string{"definitionId": writer.id.String()}, result.IDs)
}

func TestCreateReport_ExecuteWithResultReportsNothingWhenTheSaveFails(t *testing.T) {
	t.Parallel()

	writer := &namingReportWriter{id: pulid.MustNew("rd_"), err: errors.New("name taken")}

	result, err := serviceports.ExecuteTool(
		t.Context(),
		newCreateReportTool(writer),
		personParams(map[string]any{"name": "Lane revenue", "definition": definitionArgument()}),
	)

	require.ErrorContains(t, err, "name taken")
	assert.Nil(t, result)
}
