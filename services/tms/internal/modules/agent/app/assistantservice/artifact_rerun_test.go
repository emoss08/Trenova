package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubReportRuns struct {
	previous *report.ReportRun
	asked    *reporting.GetRunRequest
	started  *reporting.RunReportRequest
	err      error
}

func (r *stubReportRuns) GetRun(
	_ context.Context,
	req *reporting.GetRunRequest,
) (*report.ReportRun, error) {
	r.asked = req
	return r.previous, nil
}

func (r *stubReportRuns) RunReport(
	_ context.Context,
	req *reporting.RunReportRequest,
) (*report.ReportRun, error) {
	r.started = req
	if r.err != nil {
		return nil, r.err
	}

	return &report.ReportRun{
		ID:           pulid.MustNew("rrun_"),
		DefinitionID: req.DefinitionID,
		CannedKey:    req.CannedKey,
		Format:       req.Format,
		Params:       req.Params,
		Status:       report.RunStatusQueued,
	}, nil
}

func rerunFixture(t *testing.T, runs *stubReportRuns) *workspaceFixture {
	t.Helper()

	f := newWorkspaceFixture(t)
	f.svc.reports = &reporting.Service{}
	f.svc.reportRuns = runs

	return f
}

func (f *workspaceFixture) reportArtifact(
	kind assistantartifact.Kind,
	payload map[string]any,
) *assistantartifact.Artifact {
	artifact := &assistantartifact.Artifact{
		ID:         pulid.MustNew("art_"),
		ThreadID:   f.thread.ID,
		MessageID:  pulid.MustNew("amsg_"),
		Kind:       kind,
		Status:     assistantartifact.StatusReady,
		Title:      "Revenue by customer",
		Slug:       "revenue-by-customer",
		LineageKey: "lineage-revenue",
		LineageSeq: 1,
		Payload:    payload,
	}
	f.repo.stored[artifact.ID] = artifact

	return artifact
}

func TestRunArtifactReport_RunsTheReportAgainAsTheNextVersion(t *testing.T) {
	t.Parallel()

	previous := &report.ReportRun{
		ID:        pulid.MustNew("rrun_"),
		CannedKey: "revenue-by-customer",
		Format:    report.FormatXLSX,
		Params:    map[string]any{"windowDays": 30.0},
		Status:    report.RunStatusSucceeded,
	}
	runs := &stubReportRuns{previous: previous}
	f := rerunFixture(t, runs)
	first := f.reportArtifact(assistantartifact.KindReportRun, map[string]any{
		"runId":      previous.ID.String(),
		"reportKey":  "revenue-by-customer",
		"reportName": "Revenue by customer",
		"readBy":     artifactReader{Name: "Billing Desk"},
	})

	saved, err := f.svc.RunArtifactReport(t.Context(), f.req, first.ID)
	require.NoError(t, err)

	require.NotNil(t, runs.asked)
	assert.Equal(t, previous.ID, runs.asked.RunID)
	assert.Equal(t, f.req.UserID, runs.asked.TenantInfo.UserID)
	require.NotNil(t, runs.started)
	assert.Equal(t, "revenue-by-customer", runs.started.CannedKey)
	assert.Equal(t, report.FormatXLSX, runs.started.Format)
	assert.Equal(t, map[string]any{"windowDays": 30.0}, runs.started.Params)
	assert.Equal(t, report.RunTriggerManual, runs.started.Trigger)
	assert.Equal(t, f.req.UserID, runs.started.TenantInfo.UserID)

	require.Len(t, f.repo.inserted, 1)
	next := f.repo.inserted[0]
	assert.Equal(t, assistantartifact.KindReportRun, next.Kind)
	assert.Equal(t, assistantartifact.StatusPending, next.Status)
	assert.Equal(t, first.ID, next.LineageID)
	assert.Equal(t, 2, next.LineageSeq)
	assert.Equal(t, "revenue-by-customer", next.Slug)
	assert.Equal(t, "lineage-revenue", next.LineageKey)
	assert.Equal(t, first.MessageID, next.MessageID)
	assert.Equal(t, "Revenue by customer", next.Title)
	assert.NotEqual(t, previous.ID.String(), next.Payload["runId"])
	assert.Equal(t, "Revenue by customer", next.Payload["reportName"])
	assert.Equal(t, "queued", next.Payload["status"])
	assert.Equal(t, false, next.Payload["finished"])
	assert.Equal(t, true, next.Payload[payloadRanByPerson])
	assert.NotContains(t, next.Payload, payloadReadBy)
	assert.Equal(t, 2, saved.LineageSeq)
}

func TestRunArtifactReport_APreviewOfAnUnsavedReportCannotBeRunInFull(t *testing.T) {
	t.Parallel()

	runs := &stubReportRuns{}
	f := rerunFixture(t, runs)
	preview := f.reportArtifact(assistantartifact.KindReportPreview, map[string]any{
		payloadReadWith: map[string]any{
			"definition": map[string]any{
				"version": 2,
				"entity":  "shipment",
				"columns": []any{map[string]any{
					"id":        "c_status",
					"dimension": map[string]any{"field": "status"},
				}},
			},
		},
	})

	_, err := f.svc.RunArtifactReport(t.Context(), f.req, preview.ID)
	require.ErrorIs(t, err, errPreviewNotSaved)
	assert.Nil(t, runs.started)
	assert.Empty(t, f.repo.inserted)
}

func TestRunArtifactReport_RefusesWhatIsNotAReport(t *testing.T) {
	t.Parallel()

	runs := &stubReportRuns{}
	f := rerunFixture(t, runs)
	doc := f.document("Three loads will be late.")

	_, err := f.svc.RunArtifactReport(t.Context(), f.req, doc.ID)
	require.ErrorIs(t, err, errNotRunnable)
	assert.Nil(t, runs.started)
}

func TestRunArtifactReport_KeepsNoVersionWhenTheRunCannotStart(t *testing.T) {
	t.Parallel()

	runs := &stubReportRuns{
		previous: &report.ReportRun{ID: pulid.MustNew("rrun_"), CannedKey: "revenue-by-customer"},
		err:      errortypes.NewRateLimitError("report", "Too many reports are generating"),
	}
	f := rerunFixture(t, runs)
	first := f.reportArtifact(assistantartifact.KindReportRun, map[string]any{
		"runId": runs.previous.ID.String(),
	})

	_, err := f.svc.RunArtifactReport(t.Context(), f.req, first.ID)
	require.Error(t, err)
	assert.Empty(t, f.repo.inserted)
}
