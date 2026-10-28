package reporttools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/productguide"
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

/*
A private report is the person's own saved query, so it is saved as asked.

The owner decided that saving a report nobody else sees needs no approval: a
card in front of a report the person just asked for only stood between them
and it. A report shared with the whole organization still waits for a person,
and so does a call no person is driving, because "private" is private to
someone.
*/
func TestCreateReport_ClassifiesPrivateReportsAsPersonalAndHoldsSharedOnes(t *testing.T) {
	t.Parallel()

	policy := NewCreateReportTool(&fakeReportWriter{}).Policy()
	require.True(t, policy.PersonalRunsUnasked)

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
		params agenttoolports.ToolExecuteParams
		want   agent.AutonomyTier
	}{
		{
			name:   "no visibility is private by default",
			params: agenttoolstest.PersonParams(base()),
			want:   agent.TierAutoExecute,
		},
		{
			name:   "private",
			params: agenttoolstest.PersonParams(withVisibility(string(report.VisibilityPrivate))),
			want:   agent.TierAutoExecute,
		},
		{
			name:   "shared with the organization",
			params: agenttoolstest.PersonParams(withVisibility(string(report.VisibilityShared))),
			want:   agent.TierActWithApproval,
		},
		{
			name:   "visibility that cannot be read",
			params: agenttoolstest.PersonParams(withVisibility("everyone")),
			want:   agent.TierActWithApproval,
		},
		{
			name: "an agent principal, not a person",
			params: func() agenttoolports.ToolExecuteParams {
				params := agenttoolstest.PersonParams(base())
				params.Actor.PrincipalType = permissionports.PrincipalTypeAgent

				return params
			}(),
			want: agent.TierActWithApproval,
		},
		{
			name: "a person with no user",
			params: func() agenttoolports.ToolExecuteParams {
				params := agenttoolstest.PersonParams(base())
				params.Actor.UserID = pulid.Nil

				return params
			}(),
			want: agent.TierActWithApproval,
		},
		{
			name: "no actor",
			params: func() agenttoolports.ToolExecuteParams {
				params := agenttoolstest.PersonParams(base())
				params.Actor = nil

				return params
			}(),
			want: agent.TierActWithApproval,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			call := policy.Classified(tc.params)
			assert.Equal(t, tc.want, agenttoolstest.CallLimit(call))
			assert.Equal(t, tc.want == agent.TierAutoExecute,
				call.Egress == agent.EgressPersonal)
		})
	}
}

/*
Saving a report says which report it saved.

An approved report_create used to leave only "it ran", and the agent then
passed the proposal's id to report_get. The result names the report and
its id under the parameter report_get, report_run and report_update take.
*/
func TestCreateReport_ExecuteWithResultNamesTheSavedReport(t *testing.T) {
	t.Parallel()

	writer := &namingReportWriter{id: pulid.MustNew("rd_")}
	tool := NewCreateReportTool(writer)

	result, err := agenttoolports.ExecuteTool(
		t.Context(),
		tool,
		agenttoolstest.PersonParams(map[string]any{
			"name":       "Shipments for Peak Distributing",
			"definition": definitionArgument(),
		}),
	)

	require.NoError(t, err)
	require.NotNil(t, writer.created)
	require.NotNil(t, result)
	assert.Equal(t, "created", result.Action)
	assert.Equal(t, "report", result.Kind)
	assert.Equal(t, "Shipments for Peak Distributing", result.Name)
	assert.Equal(t, map[string]string{"definitionId": writer.id.String()}, result.IDs)
	assert.Equal(t, &agent.RecordRef{EntityType: "report", ID: writer.id.String()}, result.Record,
		"the saved report is named as the record it opens as")
}

// The record a report tool names must be one the app can open: an entity the
// record-link registry holds.
func TestReportResult_NamesARecordTheAppOpens(t *testing.T) {
	t.Parallel()

	result := reportResult("rd_1", "Lane revenue")

	require.NotNil(t, result.Record)
	path, ok := productguide.RecordPath(result.Record.EntityType, result.Record.ID)
	require.True(t, ok, "the registry has no %q entity", result.Record.EntityType)
	assert.Contains(t, path, "rd_1")
	assert.Equal(t, map[string]string{"definitionId": "rd_1"}, result.IDs,
		"the model still reads the id under the parameter the report tools take")
}

func TestCreateReport_ExecuteWithResultReportsNothingWhenTheSaveFails(t *testing.T) {
	t.Parallel()

	writer := &namingReportWriter{id: pulid.MustNew("rd_"), err: errors.New("name taken")}

	result, err := agenttoolports.ExecuteTool(
		t.Context(),
		NewCreateReportTool(writer),
		agenttoolstest.PersonParams(
			map[string]any{"name": "Lane revenue", "definition": definitionArgument()},
		),
	)

	require.ErrorContains(t, err, "name taken")
	assert.Nil(t, result)
}
