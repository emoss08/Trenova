package accountingsyncrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	mappingEntity     = "Accounting mapping"
	mappingLabelField = "targetLabel"
)

var mappingErrors = dberror.Mapping{Entity: mappingEntity}

type MappingParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type mappingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewMappingRepository(p MappingParams) accountingmappingports.AccountingMappingRepository {
	return &mappingRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-mapping-repository"),
	}
}

func (r *mappingRepository) ListByConnection(
	ctx context.Context,
	req *accountingmappingports.ListAccountingMappingsQuery,
) ([]*accountingsync.AccountingMapping, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingMapping, error) {
			am := accountingsyncdb.AccountingMappings
			return am.Where(
				am.ConnectionID.Eq(req.ConnectionID),
				keel.When(len(req.TargetTypes) > 0, am.TargetType.In(req.TargetTypes...)),
			).
				Order(am.TargetType.Asc(), am.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func mappingFilters(
	req *accountingmappingports.ListAccountingMappingsConnectionRequest,
) []keel.Pred {
	am := accountingsyncdb.AccountingMappings
	term := stringutils.NormalizeName(req.Search)

	return []keel.Pred{
		am.ConnectionID.Eq(req.ConnectionID),
		keel.When(len(req.TargetTypes) > 0, am.TargetType.In(req.TargetTypes...)),
		keel.When(len(req.States) > 0, am.State.In(req.States...)),
		keel.When(req.RequiredOnly, requiredTargets()),
		keel.When(term != "", am.SearchLabel.Contains(term)),
	}
}

func requiredTargets() keel.Pred {
	am := accountingsyncdb.AccountingMappings
	var required []keel.Pred
	for _, target := range accountingsync.AllMappingTargetTypes() {
		for _, key := range target.Keys() {
			if !accountingsync.IsRequiredTarget(target, key) {
				continue
			}
			required = append(required, keel.And(am.TargetType.Eq(target), am.TrenovaKey.Eq(key)))
		}
	}
	return keel.Or(required...)
}

func (r *mappingRepository) ListConnection(
	ctx context.Context,
	req *accountingmappingports.ListAccountingMappingsConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error) {
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: mappingLabelField, Direction: dbtype.SortDirectionAsc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				accountingsyncdb.AccountingMappings.Where(mappingFilters(req)...),
				&listquery.Request[accountingsync.AccountingMapping]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list accounting mappings", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *mappingRepository) GetByID(
	ctx context.Context,
	req accountingmappingports.GetAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingMapping, error) {
			entity, err := accountingsyncdb.AccountingMappings.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, mappingErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *mappingRepository) GetByIDs(
	ctx context.Context,
	req accountingmappingports.GetAccountingMappingsByIDsRequest,
) ([]*accountingsync.AccountingMapping, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingMapping, error) {
			if len(req.IDs) == 0 {
				return []*accountingsync.AccountingMapping{}, nil
			}

			am := accountingsyncdb.AccountingMappings
			return am.Where(am.ID.In(req.IDs...)).Limit(len(req.IDs)).List(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *mappingRepository) GetByTarget(
	ctx context.Context,
	req *accountingmappingports.GetAccountingMappingByTargetRequest,
) (*accountingsync.AccountingMapping, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingMapping, error) {
			am := accountingsyncdb.AccountingMappings
			key := am.TrenovaKey.Eq(req.TrenovaKey)
			if req.TargetType.KeyedByObject() {
				key = am.TrenovaObjectID.Eq(req.TrenovaObjectID)
			}

			entity, err := am.Where(
				am.ConnectionID.Eq(req.ConnectionID),
				am.TargetType.Eq(req.TargetType),
				key,
			).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, mappingErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *mappingRepository) CreateMissing(
	ctx context.Context,
	entities []*accountingsync.AccountingMapping,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		am := accountingsyncdb.AccountingMappings
		created, err := am.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			entities,
			am.OrganizationID,
			am.BusinessUnitID,
			am.ConnectionID,
			am.TargetType,
			am.TrenovaObjectID.Coalesce(keel.ConstText("")),
			am.TrenovaKey.Coalesce(keel.ConstText("")),
		)
		if err != nil {
			return created, fmt.Errorf("create accounting mappings: %w", err)
		}

		return created, nil
	})
}

func (r *mappingRepository) ApplyScoring(
	ctx context.Context,
	entities []*accountingsync.AccountingMapping,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		am := accountingsyncdb.AccountingMappings
		db := r.db.Keel(ctx)
		var applied int64

		for _, entity := range entities {
			entity.SearchLabel = stringutils.NormalizeName(entity.TargetLabel)
			affected, err := am.Where(
				am.ID.Eq(entity.ID),
				am.Version.Eq(entity.Version),
				am.State.Ne(accountingsync.MappingStateConfirmed),
			).Update(
				ctx,
				db,
				am.TargetLabel.To(entity.TargetLabel),
				am.SearchLabel.To(entity.SearchLabel),
				stringOrNull(am.ExternalID, entity.ExternalID),
				stringOrNull(am.ExternalName, entity.ExternalName),
				am.State.To(entity.State),
				stringOrNull(am.Source.StringCol, string(entity.Source)),
				float64OrNull(am.Confidence, entity.Confidence),
				stringOrNull(am.Reason, entity.Reason),
				am.Signals.To(entity.Signals),
			)
			if err != nil {
				return applied, fmt.Errorf("apply accounting mapping scores: %w", err)
			}
			if affected == 0 {
				continue
			}
			entity.Version++
			applied += affected
		}

		return applied, nil
	})
}

func (r *mappingRepository) UpdateLabels(
	ctx context.Context,
	entities []*accountingsync.AccountingMapping,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		am := accountingsyncdb.AccountingMappings
		db := r.db.Keel(ctx)
		for _, entity := range entities {
			entity.SearchLabel = stringutils.NormalizeName(entity.TargetLabel)
			if _, err := am.Where(am.ID.Eq(entity.ID)).Update(
				ctx,
				db,
				am.TargetLabel.To(entity.TargetLabel),
				am.SearchLabel.To(entity.SearchLabel),
				am.Version.ToExpr(am.Version),
			); err != nil {
				return fmt.Errorf("update accounting mapping labels: %w", err)
			}
		}
		return nil
	})
}

func (r *mappingRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingMapping,
) (*accountingsync.AccountingMapping, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingMapping, error) {
			ov := entity.Version
			err := accountingsyncdb.AccountingMappings.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				entity.Version = ov
				return nil, mappingErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *mappingRepository) CountByState(
	ctx context.Context,
	_ pagination.TenantInfo,
	connectionID pulid.ID,
) ([]accountingmappingports.AccountingMappingCount, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingmappingports.AccountingMappingCount, error) {
			am := accountingsyncdb.AccountingMappings
			counts, err := keel.Into[accountingmappingports.AccountingMappingCount](
				ctx,
				r.db.Keel(ctx),
				am.Group(am.TargetType, am.State).
					Pick(am.TargetType, am.State, keel.Count()).
					Where(am.ConnectionID.Eq(connectionID)),
			)
			if err != nil {
				return nil, err
			}
			if counts == nil {
				counts = []accountingmappingports.AccountingMappingCount{}
			}

			return counts, nil
		},
	)
}

func (r *mappingRepository) ListOwnerOperators(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*worker.Worker, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.Worker, error) {
		ds := driversettlementdb.Settlements
		settled := ds.Pick(ds.WorkerID).Where(
			ds.Classification.Eq(driverpay.PayeeClassificationOwnerOperator),
			ds.Status.In(driversettlement.StatusPosted, driversettlement.StatusPaid),
		)

		wrk := workerdb.Workers
		workers, err := wrk.Where(wrk.ID.InQuery(settled)).
			Load(wrk.State()).
			Order(wrk.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list owner-operators", zap.Error(err))
			return nil, fmt.Errorf("list owner-operators: %w", err)
		}
		return workers, nil
	})
}

func float64OrNull(col keel.Float64Col, value *float64) keel.Assignment {
	if value == nil {
		return col.ToNull()
	}
	return col.To(*value)
}
