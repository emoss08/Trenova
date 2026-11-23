package documentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"go.uber.org/fx"
)

type CounterParams struct {
	fx.In

	DB *postgres.Connection
}

type resourceCounter struct {
	db *postgres.Connection
}

func NewResourceCounter(p CounterParams) documentports.ResourceDocumentCounter {
	return &resourceCounter{db: p.DB}
}

func (r *resourceCounter) CountByResource(
	ctx context.Context,
	req *documentports.CountDocumentsByResourceRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		d := documentdb.Documents
		count, err := d.Where(
			d.ResourceType.Eq(req.ResourceType),
			d.ResourceID.Eq(req.ResourceID),
			d.IsCurrentVersion.IsTrue(),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count documents by resource: %w", err)
		}
		return int(count), nil
	})
}
