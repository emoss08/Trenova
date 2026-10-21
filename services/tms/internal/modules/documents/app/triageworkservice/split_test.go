package triageworkservice

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triageworkjobs/triageworkcontract"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitFilesEachRangeInline(t *testing.T) {
	t.Parallel()

	f := newFixture()

	_, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:   testTenant,
		UserID:       testUserID,
		AttachmentID: f.attachment.ID,
		ShipmentID:   f.shipmentID,
		Pages: []PageRange{
			{FromPage: 3, ToPage: 4, DocumentTypeID: f.bol.ID},
			{FromPage: 1, ToPage: 2, DocumentTypeID: f.pod.ID},
		},
	})
	require.NoError(t, err)

	require.Len(t, f.pdf.assembled, 2)
	assert.Len(t, f.uploads.sessions, 2)
	names := make([]string, 0, 2)
	for id, session := range f.uploads.sessions {
		names = append(names, session.FileName)
		assert.Equal(t, f.shipmentID.String(), session.ResourceID)
		assert.Len(t, f.uploads.parts[id], 2)
	}
	assert.ElementsMatch(t, []string{"paperwork (pages 1-2).pdf", "paperwork (pages 3-4).pdf"}, names)

	stored := f.storedAttachment()
	assert.Equal(t, "2 documents (Proof of Delivery, Bill of Lading)", stored.FiledAs)
	require.Len(t, stored.SplitDocuments, 2)
	assert.Equal(t, 1, stored.SplitDocuments[0].FromPage)
	assert.Equal(t, 4, stored.SplitDocuments[1].ToPage)
	assert.NotNil(t, f.recorder.find(triage.ActivitySplit))
	assert.ElementsMatch(t, []pulid.ID{f.pod.ID, f.bol.ID}, f.events.attached)

	_, err = f.service.Unfile(t.Context(), &UnfileRequest{
		TenantInfo:   testTenant,
		UserID:       testUserID,
		AttachmentID: f.attachment.ID,
	})
	require.NoError(t, err)
	assert.Len(t, f.documents.deleted, 2)
	assert.False(t, f.storedAttachment().Filed())
}

func TestSplitValidatesRanges(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		pages []PageRange
		want  string
	}{
		{name: "past the end", pages: []PageRange{{FromPage: 3, ToPage: 5}}, want: "pages[0].fromPage"},
		{name: "backwards", pages: []PageRange{{FromPage: 2, ToPage: 1}}, want: "pages[0].fromPage"},
		{name: "no type", pages: []PageRange{{FromPage: 1, ToPage: 1, DocumentTypeID: pulid.Nil}}, want: "pages[0].documentTypeId"},
		{name: "overlap", pages: []PageRange{{FromPage: 1, ToPage: 2}, {FromPage: 2, ToPage: 3}}, want: "overlap"},
		{name: "unknown type", pages: []PageRange{{FromPage: 1, ToPage: 1, DocumentTypeID: pulid.MustNew("dt_")}}, want: "pages[0].documentTypeId"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture()
			for i := range tc.pages {
				if tc.pages[i].DocumentTypeID.IsNil() && tc.name != "no type" {
					tc.pages[i].DocumentTypeID = f.pod.ID
				}
			}

			_, err := f.service.File(t.Context(), &FileRequest{
				TenantInfo:   testTenant,
				AttachmentID: f.attachment.ID,
				ShipmentID:   f.shipmentID,
				Pages:        tc.pages,
			})
			require.Error(t, err)
			if strings.Contains(tc.want, "[") {
				assert.Contains(t, errorFields(err), tc.want)
			} else {
				assert.Contains(t, err.Error(), tc.want)
			}
			assert.Empty(t, f.uploads.sessions)
			assert.False(t, f.storedAttachment().Filed())
		})
	}
}

func TestSplitOnlyPDF(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.attachments[f.attachment.ID].ContentType = "image/png"
	f.messages.attachments[f.attachment.ID].FileName = "scan.png"

	_, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:   testTenant,
		AttachmentID: f.attachment.ID,
		ShipmentID:   f.shipmentID,
		Pages:        []PageRange{{FromPage: 1, ToPage: 1, DocumentTypeID: f.pod.ID}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only a PDF")
}

func TestRecordSplitWithNoDocumentsClearsFiling(t *testing.T) {
	t.Parallel()

	f := newFixture()
	stored := f.messages.attachments[f.attachment.ID]
	stored.FiledShipmentID = f.shipmentID
	stored.FiledAt = 1_791_000_000

	err := f.service.RecordSplit(t.Context(), &triageworkcontract.RecordSplitPayload{
		SplitFilingPayload: triageworkcontract.SplitFilingPayload{
			MessageID:    f.message.ID,
			AttachmentID: f.attachment.ID,
			ShipmentID:   f.shipmentID,
			FiledAt:      1_791_000_000,
		},
		Failed: 2,
	})
	require.NoError(t, err)
	assert.False(t, f.storedAttachment().Filed())
	assert.Equal(t, splitFailedMessage, f.recorder.find(triage.ActivityUnfiled).Detail)
}

func TestRecordSplitIgnoresStaleFiling(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.attachments[f.attachment.ID].FiledAt = 5

	err := f.service.RecordSplit(t.Context(), &triageworkcontract.RecordSplitPayload{
		SplitFilingPayload: triageworkcontract.SplitFilingPayload{
			MessageID:    f.message.ID,
			AttachmentID: f.attachment.ID,
			FiledAt:      4,
		},
		Documents: []triageworkcontract.SplitDocument{{DocumentID: pulid.MustNew("doc_")}},
	})
	require.NoError(t, err)
	assert.Empty(t, f.storedAttachment().SplitDocuments)
}

func TestRangeFileName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "bundle (page 2).pdf", rangeFileName("bundle.pdf", PageRange{FromPage: 2, ToPage: 2}))
	assert.Equal(t, "Attachment (pages 1-3).pdf", rangeFileName(".pdf", PageRange{FromPage: 1, ToPage: 3}))
}

func TestSplitTooManyRanges(t *testing.T) {
	t.Parallel()

	f := newFixture()
	pages := make([]PageRange, maxSplitRanges+1)
	_, err := f.service.validateRanges(t.Context(), testTenant, pages, 100)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err) || errortypes.IsMultiError(err))
}

func splitAll(t *testing.T, f *fixture) error {
	t.Helper()
	_, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:   testTenant,
		UserID:       testUserID,
		AttachmentID: f.attachment.ID,
		ShipmentID:   f.shipmentID,
		Pages:        []PageRange{{FromPage: 1, ToPage: 2, DocumentTypeID: f.pod.ID}},
	})
	return err
}

func TestSplitRefusesOnlyAFileTheWorkerCannotRead(t *testing.T) {
	t.Parallel()

	for _, cause := range []error{
		kernelports.ErrImagingInvalidDocument,
		kernelports.ErrImagingEncrypted,
	} {
		probing := newFixture()
		probing.pdf.ProbeFunc = func(context.Context, *kernelports.ImagingProbeRequest) (*kernelports.ImagingProbe, error) {
			return nil, fmt.Errorf("%w: refused", cause)
		}
		err := splitAll(t, probing)
		require.True(t, errortypes.IsError(err), "probe %v: %v", cause, err)
		assert.Contains(t, err.Error(), "not a PDF that can be split")

		assembling := newFixture()
		assembling.pdf.AssembleFunc = func(
			context.Context,
			*kernelports.ImagingAssembleRequest,
			func(*kernelports.ImagingArtifact) error,
		) ([]int, error) {
			return nil, fmt.Errorf("%w: refused", cause)
		}
		err = splitAll(t, assembling)
		require.True(t, errortypes.IsError(err), "assemble %v: %v", cause, err)
		assert.Contains(t, err.Error(), "not a PDF that can be split")
	}
}

func TestSplitReturnsAWorkerThatCannotAnswerAsItIs(t *testing.T) {
	t.Parallel()

	for _, cause := range []error{
		kernelports.ErrImagingUnavailable,
		kernelports.ErrImagingTimeout,
	} {
		probing := newFixture()
		probing.pdf.ProbeFunc = func(context.Context, *kernelports.ImagingProbeRequest) (*kernelports.ImagingProbe, error) {
			return nil, fmt.Errorf("%w: no worker", cause)
		}
		err := splitAll(t, probing)
		require.ErrorIs(t, err, cause, "probe")
		assert.False(t, errortypes.IsError(err), "probe %v is not a 422", cause)

		assembling := newFixture()
		assembling.pdf.AssembleFunc = func(
			context.Context,
			*kernelports.ImagingAssembleRequest,
			func(*kernelports.ImagingArtifact) error,
		) ([]int, error) {
			return nil, fmt.Errorf("%w: no worker", cause)
		}
		err = splitAll(t, assembling)
		require.ErrorIs(t, err, cause, "assemble")
		assert.False(t, errortypes.IsError(err), "assemble %v is not a 422", cause)
	}
}
