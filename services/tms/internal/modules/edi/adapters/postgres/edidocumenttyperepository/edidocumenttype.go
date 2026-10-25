package edidocumenttyperepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.EDIDocumentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-document-type-repository"),
	}
}

func (r *repository) ListDocumentTypes(
	ctx context.Context,
	req ediports.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIDocumentType, error) {
		dt := edidb.EDIDocumentTypes
		return dt.Where(documentTypeFilters(&req)...).
			Order(dt.Code.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) SelectDocumentTypeOptions(
	ctx context.Context,
	req *ediports.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIDocumentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIDocumentType], error) {
			search := *req.SelectQueryRequest
			search.Query = strings.TrimSpace(search.Query)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				documentTypeOptionsQuery(req),
				&search,
				documentTypeSearch,
			)
		},
	)
}

func documentTypeOptionsQuery(
	req *ediports.EDIDocumentTypeSelectOptionsRequest,
) *keel.Query[edi.EDIDocumentType] {
	dt := edidb.EDIDocumentTypes
	return dt.Pick(
		dt.ID,
		dt.Code,
		dt.Name,
		dt.Standard,
		dt.TransactionSet,
		dt.Direction,
		dt.DefaultVersion,
		dt.Status,
		dt.CreatedAt,
	).
		Where(documentTypeFilters(&ediports.ListEDIDocumentTypesRequest{
			Standard:       req.Standard,
			TransactionSet: req.TransactionSet,
			Direction:      req.Direction,
			Status:         req.Status,
		})...).
		Order(dt.Code.Asc())
}

func documentTypeSearch(s string) keel.Pred {
	dt := edidb.EDIDocumentTypes
	return keel.Or(
		dt.Code.ContainsFold(s),
		dt.Name.ContainsFold(s),
		dt.DefaultVersion.ContainsFold(s),
	)
}

func documentTypeFilters(req *ediports.ListEDIDocumentTypesRequest) []keel.Pred {
	dt := edidb.EDIDocumentTypes
	return []keel.Pred{
		keel.When(req.Standard != "", dt.Standard.Eq(req.Standard)),
		keel.When(req.TransactionSet != "", dt.TransactionSet.Eq(req.TransactionSet)),
		keel.When(req.Direction != "", dt.Direction.Eq(req.Direction)),
		keel.When(req.Status != "", dt.Status.Eq(req.Status)),
	}
}
