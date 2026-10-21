package captureservice

import (
	"bytes"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnInfectedPageIsRefusedBeforeItIsReadOrStored(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	gate := contentscanportsmock.NewInfectedGate("Win.Test.EICAR_HDB-1")
	s.contentGate = gate
	principal := principalFor(t, s, pair(t, w, s))
	batch, err := s.OpenBatch(
		t.Context(),
		principal,
		&OpenBatchInput{ClientKey: "k", Source: capture.SourceScan},
	)
	require.NoError(t, err)

	body := []byte("not even a pdf")
	_, err = s.PutPage(t.Context(), principal, &PutPageInput{
		BatchID: batch.ID, Sequence: 1, Body: bytes.NewReader(body),
	})

	var refusal *errortypes.Error
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Error(), "Win.Test.EICAR_HDB-1")
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, contentscanports.SourceCapturePage, gate.Requests()[0].Source)
	assert.Equal(t, body, gate.Requests()[0].Content)
	assert.Empty(t, w.objects)
	assert.Zero(t, w.batches[batch.ID].ReceivedPageCount)
}

func TestAPrintJobIsScannedWholeBeforeItIsSplit(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	gate := contentscanportsmock.NewRefusingGate(contentscanports.ErrScannerUnavailable)
	s.contentGate = gate
	principal := principalFor(t, s, pair(t, w, s))
	batch, err := s.OpenBatch(
		t.Context(),
		principal,
		&OpenBatchInput{ClientKey: "p", Source: capture.SourcePrint},
	)
	require.NoError(t, err)

	job := pdfPage(t, 10)
	_, err = s.PutPrintJob(
		t.Context(),
		principal,
		&PutPrintJobInput{BatchID: batch.ID, Body: bytes.NewReader(job)},
	)

	require.ErrorIs(t, err, contentscanports.ErrScannerUnavailable)
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, job, gate.Requests()[0].Content)
	assert.Empty(t, w.objects)
}
