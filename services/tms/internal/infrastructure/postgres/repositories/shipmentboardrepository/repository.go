package shipmentboardrepository

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentrepository"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrUnknownFacet = errors.New("unknown shipment facet")
	ErrScopeMissing = errors.New("shipment board scope is required")
)

const (
	DefaultFacetLimit  = 50
	maxFacetLimit      = 200
	facetValueAlias    = "value"
	facetLabelAlias    = "label"
	facetCountAlias    = "count"
	stageRankAlias     = "stage_rank"
	stageRevenueAlias  = "revenue"
	quickCountAliasPfx = "qf_"
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

func New(p Params) repositories.ShipmentBoardRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.shipment-board-repository")}
}

func (r *repository) scoped(
	dba bun.IDB,
	scope *repositories.ShipmentBoardScope,
) (*bun.SelectQuery, error) {
	if scope == nil {
		return nil, ErrScopeMissing
	}

	return shipmentrepository.ApplyAggregateScope(
		dba.NewSelect().Model((*shipment.Shipment)(nil)),
		dba,
		scope.Filter,
		scope.Options,
	)
}

func (r *repository) StageSummary(
	ctx context.Context,
	scope *repositories.ShipmentBoardScope,
) ([]*repositories.ShipmentStageSummaryRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*repositories.ShipmentStageSummaryRow, error) {
		sp := buncolgen.ShipmentColumns

		q, err := r.scoped(r.db.DBForContext(ctx), scope)
		if err != nil {
			return nil, err
		}

		rows := make([]*repositories.ShipmentStageSummaryRow, 0, len(shipment.Stages()))
		if err = q.
			ColumnExpr(sp.StageRank.As(stageRankAlias)).
			ColumnExpr(buncolgen.Count(facetCountAlias)).
			ColumnExpr(sp.TotalChargeAmount.Expr("COALESCE(SUM({}), 0) AS " + stageRevenueAlias)).
			GroupExpr(sp.StageRank.Qualified()).
			OrderExpr(sp.StageRank.OrderAsc()).
			Scan(ctx, &rows); err != nil {
			r.l.Error("failed to summarize shipment stages", zap.Error(err))
			return nil, err
		}

		return rows, nil
	})
}

func (r *repository) QuickFilterCounts(
	ctx context.Context,
	req *repositories.CountShipmentQuickFiltersRequest,
) (map[shipment.QuickFilter]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[shipment.QuickFilter]int, error) {
		counts := make(map[shipment.QuickFilter]int, len(req.Filters))
		if len(req.Filters) == 0 {
			return counts, nil
		}

		dba := r.db.DBForContext(ctx)
		q, err := r.scoped(dba, req.Scope)
		if err != nil {
			return nil, err
		}

		values := make([]int, len(req.Filters))
		dest := make([]any, len(req.Filters))
		for i, spec := range req.Filters {
			cond, cErr := shipmentrepository.QuickFilterCondition(
				dba,
				req.Scope.Options.QuickFilterBasis,
				spec,
			)
			if cErr != nil {
				return nil, cErr
			}
			q = q.ColumnExpr("COUNT(*) FILTER (WHERE ?) AS ?", cond,
				bun.Ident(quickCountAliasPfx+string(spec.Filter)))
			dest[i] = &values[i]
		}

		if err = q.Scan(ctx, dest...); err != nil {
			r.l.Error("failed to count shipment quick filters", zap.Error(err))
			return nil, err
		}

		for i, spec := range req.Filters {
			counts[spec.Filter] = values[i]
		}

		return counts, nil
	})
}

type facetDefinition struct {
	field string
	apply func(q *bun.SelectQuery) *bun.SelectQuery
	label func(value string) string
}

func facetDefinitionOf(facet repositories.ShipmentFacet) (facetDefinition, error) {
	sp := buncolgen.ShipmentColumns
	filters := buncolgen.ShipmentFilter

	switch facet {
	case repositories.ShipmentFacetStatus:
		return facetDefinition{
			field: filters.Status(dbtype.OpEqual, nil).Field,
			apply: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.ColumnExpr(sp.Status.As(facetValueAlias)).
					GroupExpr(sp.Status.Qualified())
			},
			label: stringutils.HumanizeCamelCaseSentence,
		}, nil
	case repositories.ShipmentFacetTenderStatus:
		return facetDefinition{
			field: filters.TenderStatus(dbtype.OpEqual, nil).Field,
			apply: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.ColumnExpr(sp.TenderStatus.As(facetValueAlias)).
					Where(sp.TenderStatus.IsNotNull()).
					GroupExpr(sp.TenderStatus.Qualified())
			},
			label: stringutils.HumanizeCamelCaseSentence,
		}, nil
	case repositories.ShipmentFacetCustomer:
		cus := buncolgen.CustomerColumns
		return facetDefinition{
			field: filters.CustomerID(dbtype.OpEqual, nil).Field,
			apply: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Join("JOIN "+buncolgen.CustomerTable.As(buncolgen.CustomerTable.Alias)).
					JoinOn(cus.ID.EqColumn(sp.CustomerID)).
					JoinOn(cus.OrganizationID.EqColumn(sp.OrganizationID)).
					JoinOn(cus.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
					ColumnExpr(sp.CustomerID.As(facetValueAlias)).
					ColumnExpr(cus.Name.As(facetLabelAlias)).
					GroupExpr(sp.CustomerID.Qualified()).
					GroupExpr(cus.Name.Qualified())
			},
		}, nil
	case repositories.ShipmentFacetEquipment:
		et := buncolgen.EquipmentTypeColumns
		return facetDefinition{
			field: filters.TrailerTypeID(dbtype.OpEqual, nil).Field,
			apply: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Join("JOIN "+buncolgen.EquipmentTypeTable.As(buncolgen.EquipmentTypeTable.Alias)).
					JoinOn(et.ID.EqColumn(sp.TrailerTypeID)).
					JoinOn(et.OrganizationID.EqColumn(sp.OrganizationID)).
					JoinOn(et.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
					ColumnExpr(sp.TrailerTypeID.As(facetValueAlias)).
					ColumnExpr(et.Code.As(facetLabelAlias)).
					GroupExpr(sp.TrailerTypeID.Qualified()).
					GroupExpr(et.Code.Qualified())
			},
		}, nil
	default:
		return facetDefinition{}, ErrUnknownFacet
	}
}

func withoutFieldFilters(filter *pagination.QueryOptions, field string) *pagination.QueryOptions {
	if filter == nil {
		return nil
	}

	clone := *filter
	clone.FieldFilters = make([]domaintypes.FieldFilter, 0, len(filter.FieldFilters))
	for _, ff := range filter.FieldFilters {
		if ff.Field != field {
			clone.FieldFilters = append(clone.FieldFilters, ff)
		}
	}

	return &clone
}

func (r *repository) FacetCounts(
	ctx context.Context,
	req *repositories.CountShipmentFacetRequest,
) (*repositories.ShipmentFacetCounts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*repositories.ShipmentFacetCounts, error) {
		if req.Scope == nil {
			return nil, ErrScopeMissing
		}

		def, err := facetDefinitionOf(req.Facet)
		if err != nil {
			return nil, err
		}

		limit := req.Limit
		if limit <= 0 {
			limit = DefaultFacetLimit
		}
		limit = min(limit, maxFacetLimit)

		scope := &repositories.ShipmentBoardScope{
			Filter:  withoutFieldFilters(req.Scope.Filter, def.field),
			Options: req.Scope.Options,
		}
		q, err := r.scoped(r.db.DBForContext(ctx), scope)
		if err != nil {
			return nil, err
		}

		values := make([]*repositories.ShipmentFacetValue, 0, limit)
		if err = def.apply(q).
			ColumnExpr(buncolgen.Count(facetCountAlias)).
			OrderExpr(facetCountAlias + " DESC").
			OrderExpr(facetValueAlias + " ASC").
			Limit(limit).
			Scan(ctx, &values); err != nil {
			r.l.Error("failed to count shipment facet", zap.Error(err))
			return nil, err
		}

		if def.label != nil {
			for _, value := range values {
				value.Label = def.label(value.Value)
			}
		}

		return &repositories.ShipmentFacetCounts{
			Facet:  req.Facet,
			Field:  def.field,
			Values: values,
		}, nil
	})
}
