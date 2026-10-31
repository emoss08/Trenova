package documentreader

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/extractionartifact"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger    *zap.Logger
	Documents documentports.DocumentRepository
	Contents  documentports.DocumentContentService
}

type Reader struct {
	reads extractionartifact.SourceReads
}

var _ documentports.DocumentReader = (*Reader)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) documentports.DocumentReader {
	return &Reader{reads: extractionartifact.SourceReads{
		Documents: p.Documents,
		Contents:  p.Contents,
		Logger:    p.Logger.Named("document-reader"),
	}}
}

func (r *Reader) ReadDocument(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (map[string]any, error) {
	if _, err := r.reads.Documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
	}

	return extractionartifact.ReadDocument(ctx, &r.reads, documentID, tenantInfo), nil
}
