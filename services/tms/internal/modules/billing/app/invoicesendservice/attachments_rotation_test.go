package invoicesendservice

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicepacket"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type storedFiles struct {
	docs    map[pulid.ID]*document.Document
	content map[pulid.ID][]byte
}

func (s *storedFiles) Get(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return s.docs[req.ID], nil
}

func (s *storedFiles) GetMany(
	context.Context,
	documentports.GetDocumentsByIDsRequest,
) ([]*document.Document, error) {
	return nil, nil
}

func (s *storedFiles) GetDownloadContent(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	return &documentports.DocumentContent{
		Document: s.docs[req.ID],
		Body:     io.NopCloser(bytes.NewReader(s.content[req.ID])),
	}, nil
}

func TestAnEmailedDocumentWithTurnedPagesGoesAsATurnedPDF(t *testing.T) {
	turned := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "scale-ticket.jpg",
		FileType:     "image/jpeg",
	}
	plain := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "bol.pdf",
		FileType:     "application/pdf",
	}
	files := &storedFiles{
		docs:    map[pulid.ID]*document.Document{turned.ID: turned, plain.ID: plain},
		content: map[pulid.ID][]byte{turned.ID: []byte("jpeg"), plain.ID: []byte("%PDF bol")},
	}
	imaging := imagingfake.New()
	var rotation int
	imaging.AssembleFunc = func(
		_ context.Context,
		req *kernelports.ImagingAssembleRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) ([]int, error) {
		rotation = req.Outputs[0][0].Rotation
		data := []byte("%PDF turned")
		if err := yield(&kernelports.ImagingArtifact{
			ContentType: "application/pdf",
			Size:        int64(len(data)),
			Body:        bytes.NewReader(data),
		}); err != nil {
			return nil, err
		}
		return []int{1}, nil
	}
	svc := &Service{
		documentService: files,
		packets:         invoicepacket.NewAssembler(imaging, files, zap.NewNop()),
	}

	attachments, err := svc.emailAttachmentsForPlan(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		[]*invoiceports.InvoiceSendPlanAttachment{
			{
				DocumentID:    turned.ID,
				FileName:      turned.OriginalName,
				ContentType:   turned.FileType,
				PageRotations: []billingvocab.PageRotation{{Page: 1, Rotation: 90}},
			},
			{DocumentID: plain.ID, FileName: plain.OriginalName, ContentType: plain.FileType},
		},
	)

	require.NoError(t, err)
	require.Len(t, attachments, 2)
	assert.Equal(t, "scale-ticket.pdf", attachments[0].FileName)
	assert.Equal(t, "application/pdf", attachments[0].ContentType)
	assert.Equal(t, []byte("%PDF turned"), attachments[0].Content)
	assert.Equal(t, 90, rotation)
	assert.Equal(t, "bol.pdf", attachments[1].FileName)
	assert.Equal(t, []byte("%PDF bol"), attachments[1].Content)
}
