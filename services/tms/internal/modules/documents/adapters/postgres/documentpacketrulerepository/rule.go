package documentpacketrulerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentpacketrule/documentpacketruledb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentpacketruleports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ruleErrors = dberror.Mapping{Entity: "DocumentPacketRule"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) documentpacketruleports.DocumentPacketRuleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-packet-rule-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *documentpacketruleports.ListDocumentPacketRulesRequest,
) (*pagination.ListResult[*documentpacketrule.DocumentPacketRule], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documentpacketrule.DocumentPacketRule], error) {
			dpr := documentpacketruledb.DocumentPacketRules
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				dpr.Query(),
				&listquery.Request[documentpacketrule.DocumentPacketRule]{Filter: req.Filter},
			)
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *documentpacketruleports.ListDocumentPacketRuleConnectionRequest,
) (*pagination.CursorListResult[*documentpacketrule.DocumentPacketRule], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*documentpacketrule.DocumentPacketRule], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				documentpacketruledb.DocumentPacketRules.Query(),
				&listquery.Request[documentpacketrule.DocumentPacketRule]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.DocumentPacketRuleColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list document packet rules", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req documentpacketruleports.GetDocumentPacketRuleByIDRequest,
) (*documentpacketrule.DocumentPacketRule, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documentpacketrule.DocumentPacketRule, error) {
			entity, err := documentpacketruledb.DocumentPacketRules.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, ruleErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListByResourceType(
	ctx context.Context,
	req *documentpacketruleports.ListDocumentPacketRulesByResourceRequest,
) ([]*documentpacketrule.DocumentPacketRule, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*documentpacketrule.DocumentPacketRule, error) {
			dpr := documentpacketruledb.DocumentPacketRules
			items, err := dpr.Where(dpr.ResourceType.Eq(req.ResourceType)).
				Order(dpr.DisplayOrder.Asc(), dpr.CreatedAt.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}
			if items == nil {
				items = make([]*documentpacketrule.DocumentPacketRule, 0)
			}

			return items, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) (*documentpacketrule.DocumentPacketRule, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentpacketrule.DocumentPacketRule, error) {
			err := documentpacketruledb.DocumentPacketRules.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create document packet rule", zap.Error(err))
				return nil, ruleErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) (*documentpacketrule.DocumentPacketRule, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentpacketrule.DocumentPacketRule, error) {
			err := documentpacketruledb.DocumentPacketRules.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update document packet rule",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, ruleErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	req documentpacketruleports.GetDocumentPacketRuleByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := documentpacketruledb.DocumentPacketRules.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&documentpacketrule.DocumentPacketRule{ID: req.ID},
		)
		if err != nil {
			r.l.Error(
				"failed to delete document packet rule",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return ruleErrors.Map(err, req.ID.String())
		}

		return nil
	})
}
