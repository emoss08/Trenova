package edidocumenttyperepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument/edidocumentdb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
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

func New(p Params) edipartnerports.EDIDocumentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-document-type-repository"),
	}
}

func (r *repository) ListDocumentTypes(
	ctx context.Context,
	req edipartnerports.ListEDIDocumentTypesRequest,
) ([]*edidocument.EDIDocumentType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edidocument.EDIDocumentType, error) {
		dt := edidocumentdb.EDIDocumentTypes
		return dt.Where(documentTypeFilters(&req)...).
			Order(dt.Code.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) SelectDocumentTypeOptions(
	ctx context.Context,
	req *edipartnerports.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edidocument.EDIDocumentType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edidocument.EDIDocumentType], error) {
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
	req *edipartnerports.EDIDocumentTypeSelectOptionsRequest,
) *keel.Query[*edidocument.EDIDocumentType] {
	dt := edidocumentdb.EDIDocumentTypes
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
		Where(documentTypeFilters(&edipartnerports.ListEDIDocumentTypesRequest{
			Standard:       req.Standard,
			TransactionSet: req.TransactionSet,
			Direction:      req.Direction,
			Status:         req.Status,
		})...).
		Order(dt.Code.Asc())
}

func documentTypeSearch(s string) keel.Pred {
	dt := edidocumentdb.EDIDocumentTypes
	return keel.Or(
		dt.Code.ContainsFold(s),
		dt.Name.ContainsFold(s),
		dt.DefaultVersion.ContainsFold(s),
	)
}

func documentTypeFilters(req *edipartnerports.ListEDIDocumentTypesRequest) []keel.Pred {
	dt := edidocumentdb.EDIDocumentTypes
	return []keel.Pred{
		keel.When(req.Standard != "", dt.Standard.Eq(req.Standard)),
		keel.When(req.TransactionSet != "", dt.TransactionSet.Eq(req.TransactionSet)),
		keel.When(req.Direction != "", dt.Direction.Eq(req.Direction)),
		keel.When(req.Status != "", dt.Status.Eq(req.Status)),
	}
}
