package editransactionsetrepository

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

func New(p Params) edipartnerports.EDITransactionSetRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-transaction-set-repository"),
	}
}

func (r *repository) SelectTransactionSetOptions(
	ctx context.Context,
	req *edipartnerports.EDITransactionSetSelectOptionsRequest,
) (*pagination.ListResult[*edidocument.EDITransactionSet], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edidocument.EDITransactionSet], error) {
			search := *req.SelectQueryRequest
			search.Query = strings.TrimSpace(search.Query)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				transactionSetOptionsQuery(req),
				&search,
				transactionSetSearch,
			)
		},
	)
}

func transactionSetOptionsQuery(
	req *edipartnerports.EDITransactionSetSelectOptionsRequest,
) *keel.Query[*edidocument.EDITransactionSet] {
	ts := edidocumentdb.EDITransactionSets
	return ts.Pick(
		ts.ID,
		ts.Standard,
		ts.Code,
		ts.Name,
		ts.Description,
		ts.DefaultVersion,
		ts.Status,
		ts.CreatedAt,
	).Where(
		keel.When(len(req.IDs) > 0, ts.ID.In(req.IDs...)),
		keel.When(req.Standard != "", ts.Standard.Eq(req.Standard)),
		keel.When(req.Status != "", ts.Status.Eq(req.Status)),
	).Order(ts.Code.Asc())
}

func transactionSetSearch(s string) keel.Pred {
	ts := edidocumentdb.EDITransactionSets
	return keel.Or(
		ts.Code.ContainsFold(s),
		ts.Name.ContainsFold(s),
		ts.DefaultVersion.ContainsFold(s),
	)
}
