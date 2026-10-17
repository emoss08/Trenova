package extractionevalservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
fakeRefreezer keeps one re-freeze per organization, as the workflow ID does:
a start while one is running returns that one.
*/
type fakeRefreezer struct {
	running *extractionevalports.ExtractionEvalRefreezeProgress
	starts  []*extractionevalports.ExtractionEvalRefreezeStart
}

func (f *fakeRefreezer) StartExtractionEvalRefreeze(
	_ context.Context,
	start *extractionevalports.ExtractionEvalRefreezeStart,
) (*extractionevalports.ExtractionEvalRefreezeStarted, error) {
	workflowID := "extraction-eval-refreeze:" + start.TenantInfo.OrgID.String() + ":" +
		start.TenantInfo.BuID.String()
	if f.running != nil && f.running.Status == extractioneval.RefreezeStatusRunning {
		return &extractionevalports.ExtractionEvalRefreezeStarted{
			WorkflowID:     workflowID,
			AlreadyRunning: true,
		}, nil
	}
	f.starts = append(f.starts, start)
	f.running = &extractionevalports.ExtractionEvalRefreezeProgress{
		WorkflowID: workflowID,
		Status:     extractioneval.RefreezeStatusRunning,
		Total:      len(start.CaseIDs),
	}

	return &extractionevalports.ExtractionEvalRefreezeStarted{WorkflowID: workflowID}, nil
}

func (f *fakeRefreezer) ExtractionEvalRefreezeProgress(
	context.Context,
	pagination.TenantInfo,
) (*extractionevalports.ExtractionEvalRefreezeProgress, error) {
	if f.running == nil {
		return nil, nil //nolint:nilnil // none has run
	}
	copied := *f.running

	return &copied, nil
}

func (w *world) withRefreezer() *fakeRefreezer {
	refreezer := &fakeRefreezer{}
	w.svc.refreezes = refreezer
	return refreezer
}

func TestRefreezeCases_ASecondClickReturnsTheRunningRefreeze(t *testing.T) {
	t.Parallel()

	w := newWorld()
	refreezer := w.withRefreezer()
	seeded := w.seedActiveCase(t)
	require.NotNil(t, seeded.SourceDocumentID)

	started, err := w.svc.RefreezeCases(
		t.Context(),
		&extractionevalports.RefreezeExtractionEvalCasesRequest{TenantInfo: testTenant()},
		testActor(),
	)
	require.NoError(t, err)
	assert.False(t, started.AlreadyRunning)
	assert.Equal(t, 1, started.Queued)
	refreezer.running.Total = 7

	again, err := w.svc.RefreezeCases(
		t.Context(),
		&extractionevalports.RefreezeExtractionEvalCasesRequest{TenantInfo: testTenant()},
		testActor(),
	)
	require.NoError(t, err)
	assert.True(t, again.AlreadyRunning)
	assert.Equal(t, started.WorkflowID, again.WorkflowID)
	assert.Equal(t, 7, again.Queued, "the running re-freeze's own size")
	assert.Len(t, refreezer.starts, 1, "a second click starts nothing")
}

func TestRefreezeCases_StartsAgainOnceTheLastOneFinished(t *testing.T) {
	t.Parallel()

	w := newWorld()
	refreezer := w.withRefreezer()
	w.seedActiveCase(t)
	refreezer.running = &extractionevalports.ExtractionEvalRefreezeProgress{
		Status: extractioneval.RefreezeStatusCompleted,
		Total:  4,
		Frozen: 4,
	}

	started, err := w.svc.RefreezeCases(
		t.Context(),
		&extractionevalports.RefreezeExtractionEvalCasesRequest{TenantInfo: testTenant()},
		testActor(),
	)
	require.NoError(t, err)
	assert.False(t, started.AlreadyRunning)
	assert.Len(t, refreezer.starts, 1)
}

func TestRefreezeProgress_ReportsTheOrganizationsLatestRefreeze(t *testing.T) {
	t.Parallel()

	w := newWorld()
	none, err := w.svc.RefreezeProgress(t.Context(), testTenant())
	require.NoError(t, err)
	assert.Nil(t, none, "an installation without a re-freeze starter has none")

	refreezer := w.withRefreezer()
	none, err = w.svc.RefreezeProgress(t.Context(), testTenant())
	require.NoError(t, err)
	assert.Nil(t, none)

	refreezer.running = &extractionevalports.ExtractionEvalRefreezeProgress{
		WorkflowID: "extraction-eval-refreeze:org:bu",
		Status:     extractioneval.RefreezeStatusRunning,
		Total:      5,
		Frozen:     2,
		Skipped:    1,
	}
	progress, err := w.svc.RefreezeProgress(t.Context(), testTenant())
	require.NoError(t, err)
	assert.Equal(t, refreezer.running, progress)
}
