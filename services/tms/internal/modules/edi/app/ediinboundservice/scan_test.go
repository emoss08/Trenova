package ediinboundservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProcessInboundFile_QuarantinesAnInfectedFileBeforeParsing(t *testing.T) {
	t.Parallel()

	fixture := newInboundFixture(t, "ISA*00*infected")
	gate := contentscanportsmock.NewInfectedGate("Win.Test.EICAR_HDB-1")
	fixture.service.contentGate = gate
	fixture.expectFileLoadAndUpdates(t)

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})

	require.NoError(t, err)
	assert.Equal(t, ediinboundfile.InboundFileStatusQuarantined, file.Status)
	assert.Contains(t, file.FailureReason, "Win.Test.EICAR_HDB-1")
	assert.Empty(t, file.InterchangeControlNumber)
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, contentscanports.SourceEDIInboundFile, gate.Requests()[0].Source)
	assert.Equal(t, []byte("ISA*00*infected"), gate.Requests()[0].Content)
}

func TestProcessInboundFile_HoldsTheFileWhenTheScannerCannotCheckIt(t *testing.T) {
	t.Parallel()

	fixture := newInboundFixture(t, "this is not x12")
	fixture.service.contentGate = contentscanportsmock.NewRefusingGate(
		contentscanports.ErrScannerUnavailable,
	)
	fixture.inboundFileRepo.EXPECT().
		GetInboundFileByID(mock.Anything, edimessageports.GetEDIInboundFileByIDRequest{
			ID:         fixture.file.ID,
			TenantInfo: fixture.tenantInfo(),
		}).
		Return(fixture.file, nil).
		Once()

	_, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})

	require.ErrorIs(t, err, contentscanports.ErrScannerUnavailable)
	assert.Equal(t, ediinboundfile.InboundFileStatusReceived, fixture.file.Status)
}

func TestProcessInboundFile_QuarantinesAFileTooLargeToScanInsteadOfRetryingIt(t *testing.T) {
	t.Parallel()

	fixture := newInboundFixture(t, "ISA*00*large")
	fixture.service.contentGate = contentscanportsmock.NewRefusingGate(
		&contentscanports.TooLargeError{Size: 12, Limit: 4},
	)
	fixture.expectFileLoadAndUpdates(t)

	file, err := fixture.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     fixture.file.ID,
		TenantInfo: fixture.tenantInfo(),
	})

	require.NoError(t, err)
	assert.Equal(t, ediinboundfile.InboundFileStatusQuarantined, file.Status)
	assert.Contains(t, file.FailureReason, "larger than the content scanner can check")
}
