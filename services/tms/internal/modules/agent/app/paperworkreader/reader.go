package paperworkreader

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/deliveryread"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxTokens = 600

type Params struct {
	fx.In

	Logger       *zap.Logger
	Harness      llmports.TaskRunner
	Documents    documentports.DocumentRepository
	Accessorials accessorialchargeports.AccessorialChargeRepository
}

// Reader is Nova reading a shipment's billing paperwork the moment it is
// uploaded: the file goes to the model as itself, so the facts a biller checks
// are on the document in seconds, while the slower conversion that places them
// on the page runs alongside.
type Reader struct {
	harness      llmports.TaskRunner
	documents    documentports.DocumentRepository
	accessorials accessorialchargeports.AccessorialChargeRepository
	l            *zap.Logger
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Reader {
	return &Reader{
		harness:      p.Harness,
		documents:    p.Documents,
		accessorials: p.Accessorials,
		l:            p.Logger.Named("paperwork-reader"),
	}
}

type answer struct {
	deliveryread.Answer
	DocumentKind string `json:"documentKind"`
}

// Read records what the document says. A document that is not paperwork, a
// file the model cannot read, or facts a person already recorded are left alone.
func (r *Reader) Read(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	doc, err := r.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if !readable(doc) {
		return nil
	}

	catalog, err := r.catalog(ctx, tenantInfo)
	if err != nil {
		return err
	}

	var read answer
	_, err = r.harness.Run(ctx, &llmports.NovaTask{
		Name:         aitask.TaskPaperworkReading,
		Feature:      aiusage.FeaturePaperworkReading,
		Instructions: instructions,
		SchemaName:   "paperwork_reading",
		Schema:       schema(),
		Context: llmports.DelimitedContext{Sections: []llmports.ContextSection{
			{Title: "Accessorials a receipt may bill as", Content: catalogText(catalog)},
		}},
		Files: []llmports.MessageFile{{
			DocumentID:  doc.ID.String(),
			FileName:    doc.OriginalName,
			ContentType: doc.FileType,
			Size:        doc.FileSize,
		}},
		MaxTokens:  maxTokens,
		TenantInfo: tenantInfo,
		UserID:     userID,
	}, &read)
	switch {
	case errors.Is(err, llmports.ErrNoProviderConfigured):
		// Nobody reads paperwork here; the document's conversion still runs.
		return nil
	case errors.Is(err, llmports.ErrOutsideSchema):
		// Asking again pays for the same answer; the person can ask Nova to
		// read it again from the document review.
		r.l.Warn("discarded an unreadable paperwork reading",
			zap.String("documentId", doc.ID.String()), zap.Error(err))

		return nil
	case err != nil:
		return err
	}

	facts := deliveryread.Facts(&read.Answer, charges(catalog), timeutils.NowUnix())
	if facts == nil || deliveryread.Same(doc.DeliveryFacts, facts) {
		return nil
	}
	_, err = r.documents.RecordDeliveryFacts(ctx, &documentports.RecordDeliveryFactsRequest{
		ID:         doc.ID,
		TenantInfo: tenantInfo,
		Facts:      facts,
		KeepPerson: true,
	})

	return err
}

func readable(doc *document.Document) bool {
	if doc.ProcessingProfile != document.ProcessingProfileShipmentPaperwork ||
		deliveryread.Kept(doc.DeliveryFacts) {
		return false
	}
	contentType := strings.ToLower(doc.FileType)

	return contentType == "application/pdf" || strings.HasPrefix(contentType, "image/")
}

func (r *Reader) catalog(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*accessorialcharge.AccessorialCharge, error) {
	return r.accessorials.ListCatalog(ctx, tenantInfo)
}
