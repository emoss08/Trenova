package invoiceadjustmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newAttachmentPreview() *serviceports.InvoiceAdjustmentPreview {
	return &serviceports.InvoiceAdjustmentPreview{Errors: make(map[string][]string)}
}

func TestValidateAttachmentsNeverRequiresDocuments(t *testing.T) {
	kinds := []invoiceadjustment.Kind{
		invoiceadjustment.KindCreditOnly,
		invoiceadjustment.KindCreditRebill,
		invoiceadjustment.KindFullReversal,
		invoiceadjustment.KindWriteOff,
	}
	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			svc := &Service{documentRepo: mocks.NewMockDocumentRepository(t)}
			preview := newAttachmentPreview()

			svc.validateAttachments(t.Context(), &serviceports.InvoiceAdjustmentRequest{
				Kind:       kind,
				TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
			}, preview)

			assert.Empty(t, preview.Errors)
		})
	}
}

func TestValidateAttachmentsRejectsArchivedDocuments(t *testing.T) {
	docID := pulid.MustNew("doc_")
	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.BulkDeleteDocumentRequest) bool {
			return len(req.IDs) == 1 && req.IDs[0] == docID
		})).
		Return([]*document.Document{{ID: docID, Status: document.StatusArchived}}, nil)
	svc := &Service{documentRepo: repo}
	preview := newAttachmentPreview()

	svc.validateAttachments(t.Context(), &serviceports.InvoiceAdjustmentRequest{
		Kind:          invoiceadjustment.KindCreditOnly,
		AttachmentIDs: []pulid.ID{docID},
	}, preview)

	assert.Contains(t, preview.Errors, "attachmentIds")
}

func TestValidateAttachmentsAcceptsActiveDocuments(t *testing.T) {
	docID := pulid.MustNew("doc_")
	repo := mocks.NewMockDocumentRepository(t)
	repo.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return([]*document.Document{{ID: docID, Status: document.StatusActive}}, nil)
	svc := &Service{documentRepo: repo}
	preview := newAttachmentPreview()

	svc.validateAttachments(t.Context(), &serviceports.InvoiceAdjustmentRequest{
		Kind:          invoiceadjustment.KindWriteOff,
		AttachmentIDs: []pulid.ID{docID},
	}, preview)

	assert.Empty(t, preview.Errors)
}
