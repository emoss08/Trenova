package messagefileservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type pdfFiles struct {
	reads int
}

func (f *pdfFiles) ReadConversationFile(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.ConversationFile, error) {
	f.reads++
	return &documentports.ConversationFile{
		Document: &document.Document{ID: req.ID, FileType: "application/pdf"},
		Data:     []byte("%PDF-1.7"),
	}, nil
}

func reader(probeErr error) (*Service, *pdfFiles, *imagingfake.Fake) {
	files := &pdfFiles{}
	imaging := imagingfake.New()
	imaging.ProbeFunc = func(
		context.Context,
		*kernelports.ImagingProbeRequest,
	) (*kernelports.ImagingProbe, error) {
		if probeErr != nil {
			return nil, probeErr
		}
		return &kernelports.ImagingProbe{PageCount: 3}, nil
	}
	svc, ok := New(Params{Logger: zap.NewNop(), Files: files, Imaging: imaging}).(*Service)
	if !ok {
		panic("New returns *Service")
	}
	return svc, files, imaging
}

func request() *assistantports.ReadMessageFileRequest {
	return &assistantports.ReadMessageFileRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		DocumentID: pulid.MustNew("doc_"),
	}
}

func TestReadMessageFile_ADamagedOrEncryptedPDFIsUnreadable(t *testing.T) {
	t.Parallel()

	for _, cause := range []error{
		kernelports.ErrImagingInvalidDocument,
		kernelports.ErrImagingEncrypted,
	} {
		svc, files, _ := reader(fmt.Errorf("%w: refused", cause))
		req := request()

		content, err := svc.ReadMessageFile(t.Context(), req)
		require.NoError(t, err)
		assert.True(t, content.Unreadable, "%v", cause)

		_, err = svc.ReadMessageFile(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, 1, files.reads, "an unreadable file is kept, as it will not change")
	}
}

func TestReadMessageFile_AWorkerThatCannotAnswerLeavesThePageCountUnknown(t *testing.T) {
	t.Parallel()

	for _, cause := range []error{
		kernelports.ErrImagingUnavailable,
		kernelports.ErrImagingTimeout,
		kernelports.ErrImagingTooLarge,
	} {
		probeErr := fmt.Errorf("%w: no worker", cause)
		svc, files, imaging := reader(nil)
		imaging.ProbeFunc = func(
			context.Context,
			*kernelports.ImagingProbeRequest,
		) (*kernelports.ImagingProbe, error) {
			if probeErr != nil {
				return nil, probeErr
			}
			return &kernelports.ImagingProbe{PageCount: 3}, nil
		}
		req := request()

		content, err := svc.ReadMessageFile(t.Context(), req)
		require.NoError(t, err, "%v", cause)
		assert.Equal(t, []byte("%PDF-1.7"), content.Data, "%v", cause)
		assert.False(t, content.Unreadable, "%v", cause)
		assert.Zero(t, content.PageCount, "%v", cause)

		probeErr = nil
		content, err = svc.ReadMessageFile(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, 3, content.PageCount, "the file without its count was not cached")
		assert.Equal(t, 2, files.reads)

		_, err = svc.ReadMessageFile(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, 2, files.reads, "once counted, the file is cached")
	}
}
