package aitrainingservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const documentText = `Rate Confirmation Load # AFB-7731902
Carrier: Northstar Hauling  MC# 884213
Dispatcher: Dana Whitfield dana@northstar.com
Shipper: Pacific Coast Produce, 1450 Harbor Industrial Road, Oakland, CA 94607
Total: $2,760.50`

func confirmedSnapshot() *aicorrection.Snapshot {
	return &aicorrection.Snapshot{
		Fields: map[string]string{
			aicorrection.FieldReference: "AFB-7731902",
			aicorrection.FieldShipper:   "Pacific Coast Produce",
			aicorrection.FieldRate:      "2760.50",
		},
		Stops: []aicorrection.StopSnapshot{{
			Role:         aicorrection.RolePickup,
			Name:         "Pacific Coast Produce",
			AddressLine1: "1450 Harbor Industrial Rd",
			City:         "Oakland",
			State:        "CA",
			PostalCode:   "94607",
		}},
	}
}

func exportRequest(f *runnerFixture) *services.ExportTrainingOrganizationRequest {
	return &services.ExportTrainingOrganizationRequest{
		ExportID: f.export.ID,
		Ordinal:  1,
		Consent:  f.exports.consent,
	}
}

func TestExportOrganizationWritesAnonymizedExamples(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.addCorrection(documentText, confirmedSnapshot())
	f.addCorrection(documentText, confirmedSnapshot())

	result, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)

	require.Len(t, result.Parts, 1)
	assert.Equal(t, 2, result.Examples)
	assert.Equal(t, 2, result.Parts[0].Examples)
	assert.Equal(t, aitraining.SplitTrain, result.Parts[0].Split)
	assert.Len(t, result.Parts[0].SHA256, 64)
	assert.Positive(t, result.Parts[0].Bytes)
	assert.NotContains(t, result.Parts[0].Key, testOrg.String())

	lines := f.objects.lines(result.Parts[0].Key)
	require.Len(t, lines, 2)
	body := strings.ToLower(string(f.objects.objects[result.Parts[0].Key]))
	for _, leaked := range []string{
		"northstar", "dana whitfield", "pacific coast", "afb-7731902", "884213", "94607", "oakland",
		"2,760.50", "88123", testOrg.String(),
	} {
		assert.NotContains(t, body, leaked)
	}

	var example aitraining.Example
	require.NoError(t, sonic.Unmarshal(lines[0], &example))
	assert.Equal(t, aitraining.ExampleFormat, example.Format)
	assert.Equal(t, "document.pdf", example.Input.FileName)
	assert.Equal(t, aicorrection.OutcomeCorrect, example.Outcomes[aicorrection.FieldReference])
	assert.Contains(t, example.Input.Pages[0].Text, example.Target.Fields[aicorrection.FieldReference])

	stored := f.exports.items[f.export.ID]
	require.Len(t, stored.Progress, 1)
	assert.Equal(t, 2, stored.ExamplesTotal)
	assert.Equal(t, 1, stored.OrganizationsIncluded)

	records := f.records.records[f.export.ID]
	require.Len(t, records, 2)
	assert.Equal(t, example.ID, records[0].ExampleID)
	assert.Equal(t, f.exports.consent.GrantedAt, records[0].ConsentGrantedAt)
}

func TestExportOrganizationCountsDrops(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.addCorrection(documentText, confirmedSnapshot())
	noText := f.addCorrection("   ", confirmedSnapshot())
	f.contents.pages[*noText.DocumentID] = nil
	f.addCorrection(documentText, &aicorrection.Snapshot{})
	missingDoc := f.addCorrection(documentText, confirmedSnapshot())
	delete(f.documents.docs, *missingDoc.DocumentID)

	result, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)

	assert.Equal(t, 1, result.Examples)
	assert.Equal(t, 1, result.Dropped[aitraining.DropNoDocumentText])
	assert.Equal(t, 1, result.Dropped[aitraining.DropNothingConfirmed])
	assert.Equal(t, 1, result.Dropped[aitraining.DropDocumentUnreadable])
}

func TestExportOrganizationStopsAtTheOrganizationCap(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.export.MaxPerOrganization = 3
	for range 5 {
		f.addCorrection(documentText, confirmedSnapshot())
	}

	result, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)
	assert.Equal(t, 3, result.Examples)
	assert.Len(t, f.records.records[f.export.ID], 3)
}

func TestExportOrganizationSkipsWithdrawnConsent(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.addCorrection(documentText, confirmedSnapshot())
	f.exports.consent.Granted = false

	result, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)
	assert.True(t, result.ConsentWithdrawn)
	assert.Empty(t, result.Parts)
	assert.Empty(t, f.objects.objects)
	assert.Equal(t, 1, f.records.deletes)
}

func TestExportOrganizationDiscardsWhenConsentChangesMidway(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.addCorrection(documentText, confirmedSnapshot())
	f.addCorrection(documentText, confirmedSnapshot())
	withdrawn := f.exports.consent
	withdrawn.Granted = false
	f.exports.consentAfter = &withdrawn

	result, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)
	assert.True(t, result.ConsentWithdrawn)
	assert.Empty(t, result.Parts)
	assert.Equal(t, 2, result.Dropped[aitraining.DropConsentWithdrawn])
	assert.Empty(t, f.objects.objects)
	assert.Contains(t, f.objects.deleted, f.export.PartKey(1, aitraining.SplitTrain))
	assert.Empty(t, f.records.records[f.export.ID])
	assert.Equal(t, 2, f.exports.items[f.export.ID].Dropped[aitraining.DropConsentWithdrawn])
	assert.Zero(t, f.exports.items[f.export.ID].OrganizationsIncluded)
}

func TestExportOrganizationRetryReplacesItsProgress(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.addCorrection(documentText, confirmedSnapshot())

	_, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)
	f.exports.consentReads = 0
	_, err = f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)

	stored := f.exports.items[f.export.ID]
	assert.Len(t, stored.Progress, 1)
	assert.Equal(t, 1, stored.ExamplesTotal)
}

func TestExportOrganizationRegrantIsTreatedAsWithdrawal(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.addCorrection(documentText, confirmedSnapshot())
	regranted := f.exports.consent
	regranted.GrantedAt++
	f.exports.consentAfter = &regranted

	result, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.NoError(t, err)
	assert.True(t, result.ConsentWithdrawn)
	assert.Empty(t, f.objects.objects)
}

func TestExportOrganizationStopsWhenCanceled(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.addCorrection(documentText, confirmedSnapshot())
	reads := 0
	f.exports.onStatusRead = func(entity *aitraining.TrainingExport) {
		reads++
		if reads > 1 {
			entity.Status = aitraining.ExportStatusCanceled
		}
	}

	_, err := f.runner.ExportOrganization(t.Context(), exportRequest(f))
	require.ErrorIs(t, err, aitraining.ErrExportInactive)
	assert.Empty(t, f.objects.objects)
}

func TestFinishWritesManifestAndIsIdempotent(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.export.RecordProgress(aitraining.OrganizationProgress{
		Ordinal:  1,
		Examples: 5,
		Dropped:  map[aitraining.DropReason]int{aitraining.DropResidualIdentifier: 2},
		Parts: []aitraining.Part{{
			Ordinal: 1, Split: aitraining.SplitTrain, Key: f.export.PartKey(1, aitraining.SplitTrain), Examples: 5,
		}},
	})
	f.export.RecordProgress(aitraining.OrganizationProgress{Ordinal: 2, ConsentWithdrawn: true})
	f.export.RecordProgress(aitraining.OrganizationProgress{Ordinal: 3})
	finished, err := f.runner.Finish(t.Context(), &services.FinishAITrainingExportRequest{ExportID: f.export.ID})
	require.NoError(t, err)

	assert.Equal(t, aitraining.ExportStatusCompleted, finished.Status)
	assert.Equal(t, 5, finished.ExamplesTotal)
	assert.Len(t, finished.ManifestSHA256, 64)

	var manifest aitraining.Manifest
	require.NoError(t, sonic.Unmarshal(f.objects.objects[finished.ManifestKey], &manifest))
	assert.Equal(t, aitraining.ManifestFormat, manifest.Format)
	assert.Equal(t, 2, manifest.Dropped[aitraining.DropResidualIdentifier])
	assert.Equal(t, 3, manifest.OrganizationsConsidered)

	again, err := f.runner.Finish(t.Context(), &services.FinishAITrainingExportRequest{ExportID: f.export.ID})
	require.NoError(t, err)
	assert.Equal(t, finished.Version, again.Version)
}

func TestFinishKeepsCanceledStatus(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.export.Status = aitraining.ExportStatusCanceled
	canceledAt := testNow - 5
	f.export.FinishedAt = &canceledAt

	finished, err := f.runner.Finish(t.Context(), &services.FinishAITrainingExportRequest{ExportID: f.export.ID})
	require.NoError(t, err)
	assert.Equal(t, aitraining.ExportStatusCanceled, finished.Status)
	assert.Equal(t, canceledAt, *finished.FinishedAt)
	assert.NotEmpty(t, finished.ManifestKey)
}

func TestBeginAndFail(t *testing.T) {
	t.Parallel()

	f := newRunnerFixture()
	f.export.Status = aitraining.ExportStatusQueued

	begun, err := f.runner.Begin(t.Context(), f.export.ID)
	require.NoError(t, err)
	assert.Equal(t, aitraining.ExportStatusRunning, begun.Status)

	again, err := f.runner.Begin(t.Context(), f.export.ID)
	require.NoError(t, err)
	assert.Equal(t, begun.Version, again.Version)

	require.NoError(t, f.runner.Fail(t.Context(), f.export.ID, "boom"))
	failed, err := f.runner.exports.GetByID(t.Context(), f.export.ID)
	require.NoError(t, err)
	assert.Equal(t, aitraining.ExportStatusFailed, failed.Status)

	_, err = f.runner.Begin(t.Context(), f.export.ID)
	require.ErrorIs(t, err, aitraining.ErrExportInactive)
	require.NoError(t, f.runner.Fail(t.Context(), f.export.ID, "again"))
}

type fakeStarter struct {
	err     error
	started []pulid.ID
}

func (s *fakeStarter) StartAITrainingExport(_ context.Context, id pulid.ID) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	s.started = append(s.started, id)
	return aitraining.ExportWorkflowID(id), nil
}

func newOperator(starter services.AITrainingExportStarter) (*Operator, *exportStore) {
	exports := newExportStore()
	return &Operator{
		l:       zap.NewNop(),
		exports: exports,
		records: newRecordStore(),
		starter: starter,
		now:     func() int64 { return testNow },
	}, exports
}

func TestOperatorStart(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{}
	operator, exports := newOperator(starter)
	created, err := operator.Start(t.Context(), &services.StartAITrainingExportRequest{
		CapturedFrom: testNow - 86_400,
		RequestedBy:  "  ops  ",
	})
	require.NoError(t, err)

	assert.Equal(t, aitraining.ExportStatusQueued, created.Status)
	assert.Equal(t, testNow, created.CapturedTo)
	assert.Equal(t, aitraining.DefaultMaxPerOrganization, created.MaxPerOrganization)
	assert.Equal(t, "ops", created.RequestedBy)
	assert.Equal(t, aitraining.ExportWorkflowID(created.ID), exports.items[created.ID].WorkflowID)
	assert.Equal(t, []pulid.ID{created.ID}, starter.started)

	_, err = operator.Start(t.Context(), &services.StartAITrainingExportRequest{
		CapturedFrom: testNow - 86_400,
		RequestedBy:  "ops",
	})
	require.Error(t, err)
}

func TestOperatorStartValidates(t *testing.T) {
	t.Parallel()

	operator, _ := newOperator(&fakeStarter{})
	_, err := operator.Start(t.Context(), &services.StartAITrainingExportRequest{
		CapturedFrom: testNow,
		CapturedTo:   testNow + 10,
		RequestedBy:  "",
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.GreaterOrEqual(t, len(multiErr.Errors), 2)
}

func TestOperatorStartMarksFailedWhenWorkflowCannotStart(t *testing.T) {
	t.Parallel()

	operator, exports := newOperator(&fakeStarter{err: errors.New("temporal down")})
	_, err := operator.Start(t.Context(), &services.StartAITrainingExportRequest{
		CapturedFrom: testNow - 86_400,
		RequestedBy:  "ops",
	})
	require.Error(t, err)
	require.Len(t, exports.items, 1)
	for _, item := range exports.items {
		assert.Equal(t, aitraining.ExportStatusFailed, item.Status)
	}
}

func TestOperatorStartWithoutStarter(t *testing.T) {
	t.Parallel()

	operator, _ := newOperator(nil)
	_, err := operator.Start(t.Context(), &services.StartAITrainingExportRequest{})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestOperatorCancel(t *testing.T) {
	t.Parallel()

	operator, _ := newOperator(&fakeStarter{})
	created, err := operator.Start(t.Context(), &services.StartAITrainingExportRequest{
		CapturedFrom: testNow - 86_400,
		RequestedBy:  "ops",
	})
	require.NoError(t, err)

	canceled, err := operator.Cancel(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, aitraining.ExportStatusCanceled, canceled.Status)
	require.NotNil(t, canceled.FinishedAt)

	_, err = operator.Cancel(t.Context(), created.ID)
	require.Error(t, err)
}

func TestOperatorWithdrawnRequiresExport(t *testing.T) {
	t.Parallel()

	operator, _ := newOperator(&fakeStarter{})
	_, err := operator.WithdrawnExamples(t.Context(), repositories.ListWithdrawnTrainingExamplesRequest{
		ExportID: pulid.MustNew("aitx_"),
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}
