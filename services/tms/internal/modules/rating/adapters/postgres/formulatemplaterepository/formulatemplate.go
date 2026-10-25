package formulatemplaterepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const shipmentUsageSQL = "(SELECT count(*) FROM shipments sp WHERE sp.formula_template_id = ? " +
	"AND sp.organization_id = ? AND sp.business_unit_id = ?)"

var formulaTemplateErrors = dberror.Mapping{Entity: "FormulaTemplate"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) formulatemplateports.FormulaTemplateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-template-repository"),
	}
}

func statusFilter(value string) (keel.Pred, error) {
	if value == "" {
		return nil, nil //nolint:nilnil // a nil predicate leaves the list unfiltered
	}

	status, err := formulatemplate.StatusFromString(value)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Unknown template status: {0}", value,
		)
	}

	return formulatemplatedb.FormulaTemplates.Status.Eq(status), nil
}

func (r *repository) List(
	ctx context.Context,
	req *formulatemplateports.ListFormulaTemplatesRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			log := r.l.With(zap.String("operation", "List"), zap.Any("request", req))

			status, err := statusFilter(req.Status)
			if err != nil {
				log.Warn(
					"rejected unknown template status filter",
					zap.String("status", req.Status),
				)
				return nil, err
			}

			ft := formulatemplatedb.FormulaTemplates
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				ft.Where(status),
				&listquery.Request[formulatemplate.FormulaTemplate]{
					Filter: req.Filter,
					Then:   []keel.Order{ft.CreatedAt.Desc()},
				},
			)
			if err != nil {
				log.Error("failed to list formula templates", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *formulatemplateports.ListFormulaTemplateConnectionRequest,
) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*formulatemplate.FormulaTemplate], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				formulatemplatedb.FormulaTemplates.Query(),
				&listquery.Request[formulatemplate.FormulaTemplate]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.FormulaTemplateColumns,
				},
			)
			if err != nil {
				r.l.Error(
					"failed to list formula templates",
					zap.String("operation", "ListConnection"),
					zap.Error(err),
				)
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplate, error) {
			err := formulatemplatedb.FormulaTemplates.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to create formula template",
					zap.String("name", entity.Name),
					zap.Error(err),
				)
				return nil, mapTemplateError(err, entity)
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplate, error) {
			err := formulatemplatedb.FormulaTemplates.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update formula template",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, mapTemplateError(err, entity)
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplate, error) {
			entity, err := formulatemplatedb.FormulaTemplates.Get(
				ctx,
				r.db.Keel(ctx),
				req.TemplateID,
			)
			if err != nil {
				return nil, formulaTemplateErrors.Map(err, req.TemplateID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req formulatemplateports.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			if len(req.TemplateIDs) == 0 {
				return []*formulatemplate.FormulaTemplate{}, nil
			}

			ft := formulatemplatedb.FormulaTemplates
			entities, err := ft.Where(ft.ID.In(req.TemplateIDs...)).
				Limit(len(req.TemplateIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get formula templates", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) FindByNames(
	ctx context.Context,
	req formulatemplateports.GetFormulaTemplatesByNamesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			if len(req.Names) == 0 {
				return []*formulatemplate.FormulaTemplate{}, nil
			}

			ft := formulatemplatedb.FormulaTemplates
			entities, err := ft.Where(ft.Name.In(req.Names...)).
				Limit(len(req.Names)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to find formula templates by names", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *formulatemplateports.BulkUpdateFormulaTemplateStatusRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			ft := formulatemplatedb.FormulaTemplates
			entities, err := ft.Where(
				ft.ID.In(req.TemplateIDs...),
				ft.Status.In(formulatemplate.StatusActive, formulatemplate.StatusInactive),
			).UpdateReturning(ctx, r.db.Keel(ctx), ft.Status.To(req.Status))
			if err != nil {
				r.l.Error("failed to bulk update formula template status", zap.Error(err))
				return nil, err
			}

			updatedIDs := make([]pulid.ID, 0, len(entities))
			for _, entity := range entities {
				updatedIDs = append(updatedIDs, entity.ID)
			}
			if skipped := sliceutils.Difference(
				sliceutils.Dedupe(req.TemplateIDs),
				updatedIDs,
			); len(skipped) > 0 {
				return nil, errortypes.NewValidationError(
					"templateIds",
					errortypes.ErrInvalid,
					"Templates {0} are no longer Active or Inactive, so none were updated; refresh and try again",
					strings.Join(
						pulid.Map(skipped, func(id pulid.ID) string { return id.String() }),
						", ",
					),
				)
			}

			return entities, nil
		},
	)
}

func (r *repository) BulkCreate(
	ctx context.Context,
	entities []*formulatemplate.FormulaTemplate,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			if len(entities) == 0 {
				return entities, nil
			}

			err := formulatemplatedb.FormulaTemplates.CreateMany(ctx, r.db.Keel(ctx), entities)
			if err != nil {
				r.l.Error("failed to bulk insert formula templates", zap.Error(err))
				return nil, mapTemplateError(err, entities[0])
			}

			return entities, nil
		},
	)
}

func (r *repository) TakenNames(
	ctx context.Context,
	req *formulatemplateports.ListTakenNamesRequest,
) (map[string]struct{}, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]struct{}, error) {
		taken := make(map[string]struct{}, len(req.Bases))
		if len(req.Bases) == 0 {
			return taken, nil
		}

		ft := formulatemplatedb.FormulaTemplates
		prefixes := make([]keel.Pred, 0, len(req.Bases))
		for _, base := range req.Bases {
			prefixes = append(prefixes, ft.Name.HasPrefix(base))
		}

		names, err := keel.Into[string](
			ctx,
			r.db.Keel(ctx),
			ft.Pick(ft.Name).Where(keel.Or(prefixes...)),
		)
		if err != nil {
			return nil, err
		}

		for _, name := range names {
			taken[name] = struct{}{}
		}

		return taken, nil
	})
}

type usageRow struct {
	Shipments int64
	Rules     int64
	Fallbacks int64
}

func usageQuery(templateID pulid.ID) *keel.Query[formulatemplate.FormulaTemplate] {
	ft := formulatemplatedb.FormulaTemplates
	rr := rateagreementdb.RateAgreementRules
	bc := tenantdb.BillingControls

	return ft.Pick(
		shipmentUsage(ft),
		keel.Scalar(rr.Pick(keel.Count()).Where(rr.FormulaTemplateID.EqCol(ft.ID))),
		keel.Scalar(bc.Pick(keel.Count()).Where(bc.FallbackFormulaTemplateID.EqCol(ft.ID))),
	).Where(ft.ID.Eq(templateID))
}

func shipmentUsage(ft *formulatemplatedb.FormulaTemplatesTable) keel.Expr {
	return keel.Raw(shipmentUsageSQL, ft.ID, ft.OrganizationID, ft.BusinessUnitID)
}

func (r *repository) CountUsages(
	ctx context.Context,
	req *formulatemplateports.GetTemplateUsageRequest,
) (*formulatemplateports.GetTemplateUsageResponse, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplateports.GetTemplateUsageResponse, error) {
			row, err := keel.IntoOne[usageRow](ctx, r.db.Keel(ctx), usageQuery(req.TemplateID))
			if err != nil && !dberror.IsNotFoundError(err) {
				r.l.Error(
					"failed to count usages",
					zap.String("templateID", req.TemplateID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			counts := []formulatemplateports.TemplateUsageCount{
				{Type: "shipment", Count: int(row.Shipments)},
				{Type: "rate_agreement_rule", Count: int(row.Rules)},
				{Type: "billing_control_fallback", Count: int(row.Fallbacks)},
			}
			usages := make([]formulatemplateports.TemplateUsageCount, 0, len(counts))
			for _, usage := range counts {
				if usage.Count > 0 {
					usages = append(usages, usage)
				}
			}

			return &formulatemplateports.GetTemplateUsageResponse{
				InUse:  len(usages) > 0,
				Usages: usages,
			}, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *formulatemplateports.FormulaTemplateSelectOptionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			ft := formulatemplatedb.FormulaTemplates
			q := ft.Pick(ft.ID, ft.Name, ft.Description).
				Where(ft.Status.Eq(formulatemplate.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(ft.Name.ContainsFold(s), ft.Description.ContainsFold(s))
				},
			)
		},
	)
}

func mapTemplateError(err error, entity *formulatemplate.FormulaTemplate) error {
	if dberror.IsUniqueConstraintViolation(err) {
		return duplicateTemplateName(entity.Name)
	}

	return formulaTemplateErrors.Map(err, entity.ID.String())
}

func duplicateTemplateName(name string) error {
	return errortypes.NewValidationError(
		"name",
		errortypes.ErrDuplicate,
		"A formula template named \"{0}\" already exists", name,
	)
}
