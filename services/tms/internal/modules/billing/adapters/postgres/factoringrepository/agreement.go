package factoringrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	agreementEntity    = "FactoringAgreement"
	activeOverlapIndex = "ex_factoring_agreements_active_overlap"
)

type Params struct {
	fx.In

	DB     kernelports.DBConnection
	Logger *zap.Logger
}

type agreementRepository struct {
	db kernelports.DBConnection
	l  *zap.Logger
}

func NewAgreementRepository(p Params) factoringports.AgreementRepository {
	return &agreementRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-agreement"),
	}
}

func (r *agreementRepository) Create(
	ctx context.Context,
	entity *factoring.Agreement,
) (*factoring.Agreement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Agreement, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, translateAgreementError(err, "create factoring agreement")
		}
		if err := r.replaceChildren(ctx, entity); err != nil {
			return nil, err
		}

		return r.load(ctx, entity.ID, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
	})
}

func (r *agreementRepository) Update(
	ctx context.Context,
	entity *factoring.Agreement,
) (*factoring.Agreement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Agreement, error) {
		previousVersion := entity.Version
		entity.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			ExcludeColumn(
				factoringcol.AgreementColumns.CreatedAt.Bare(),
				factoringcol.AgreementColumns.FactoringCompanyID.Bare(),
			).
			WherePK().
			Where(factoringcol.AgreementColumns.Version.Eq(), previousVersion).
			Exec(ctx)
		if err != nil {
			entity.Version = previousVersion
			return nil, translateAgreementError(err, "update factoring agreement")
		}
		if err = dberror.CheckRowsAffected(res, agreementEntity, entity.ID.String()); err != nil {
			entity.Version = previousVersion
			return nil, err
		}
		if err = r.replaceChildren(ctx, entity); err != nil {
			return nil, err
		}

		return r.load(ctx, entity.ID, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
	})
}

func (r *agreementRepository) GetByID(
	ctx context.Context,
	req factoringports.GetAgreementRequest,
) (*factoring.Agreement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Agreement, error) {
		return r.load(ctx, req.ID, req.TenantInfo)
	})
}

func (r *agreementRepository) ListByFactoringCompany(
	ctx context.Context,
	req factoringports.ListAgreementsRequest,
) ([]*factoring.Agreement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.Agreement, error) {
		cols := factoringcol.AgreementColumns
		rows := make([]*factoring.Agreement, 0, 4)
		if err := r.withChildren(r.db.DBForContext(ctx).NewSelect().Model(&rows)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.AgreementScopeTenant(sq, req.TenantInfo).
					Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID)
			}).
			Order(cols.EffectiveFrom.OrderDesc(), cols.ID.OrderDesc()).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list factoring agreements: %w", err)
		}

		return rows, nil
	})
}

func (r *agreementRepository) ListByFactoringCompanyIDs(
	ctx context.Context,
	req *factoringports.ListAgreementsByFactoringCompanyIDsRequest,
) (map[pulid.ID][]*factoring.Agreement, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*factoring.Agreement, error) {
			result := make(map[pulid.ID][]*factoring.Agreement, len(req.FactoringCompanyIDs))
			if len(req.FactoringCompanyIDs) == 0 {
				return result, nil
			}
			cols := factoringcol.AgreementColumns
			rows := make([]*factoring.Agreement, 0, len(req.FactoringCompanyIDs))
			if err := r.withChildren(r.db.DBForContext(ctx).NewSelect().Model(&rows)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return factoringcol.AgreementScopeTenant(sq, req.TenantInfo).
						Where(cols.FactoringCompanyID.In(), bun.List(req.FactoringCompanyIDs))
				}).
				Order(cols.EffectiveFrom.OrderDesc(), cols.ID.OrderDesc()).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("list factoring agreements by factor: %w", err)
			}
			for _, row := range rows {
				result[row.FactoringCompanyID] = append(result[row.FactoringCompanyID], row)
			}
			return result, nil
		},
	)
}

func (r *agreementRepository) GetInForce(
	ctx context.Context,
	req factoringports.GetAgreementInForceRequest,
) (*factoring.Agreement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Agreement, error) {
		entity := new(factoring.Agreement)
		if err := r.withChildren(r.db.DBForContext(ctx).NewSelect().Model(entity)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return inForce(
					factoringcol.AgreementScopeTenant(sq, req.TenantInfo),
					req.At,
				).Where(factoringcol.AgreementColumns.FactoringCompanyID.Eq(), req.FactoringCompanyID)
			}).
			Limit(1).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, agreementEntity)
		}

		return entity, nil
	})
}

func (r *agreementRepository) ListInForce(
	ctx context.Context,
	req *factoringports.ListAgreementsInForceRequest,
) (map[pulid.ID]*factoring.Agreement, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoring.Agreement, error) {
			result := make(map[pulid.ID]*factoring.Agreement, len(req.FactoringCompanyIDs))
			if len(req.FactoringCompanyIDs) == 0 {
				return result, nil
			}

			rows := make([]*factoring.Agreement, 0, len(req.FactoringCompanyIDs))
			if err := r.withChildren(r.db.DBForContext(ctx).NewSelect().Model(&rows)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return inForce(
						factoringcol.AgreementScopeTenant(sq, req.TenantInfo),
						req.At,
					).Where(
						factoringcol.AgreementColumns.FactoringCompanyID.In(),
						bun.List(req.FactoringCompanyIDs),
					)
				}).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("list factoring agreements in force: %w", err)
			}

			for _, row := range rows {
				result[row.FactoringCompanyID] = row
			}

			return result, nil
		},
	)
}

func (r *agreementRepository) load(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*factoring.Agreement, error) {
	entity := new(factoring.Agreement)
	if err := r.withChildren(r.db.DBForContext(ctx).NewSelect().Model(entity)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return factoringcol.AgreementScopeTenant(sq, tenantInfo).
				Where(factoringcol.AgreementColumns.ID.Eq(), id)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, agreementEntity)
	}

	return entity, nil
}

func (r *agreementRepository) withChildren(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		Relation(factoringcol.AgreementRelations.FeeTiers, func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order(factoringcol.FeeTierColumns.DaysFrom.OrderAsc())
		}).
		Relation("DocumentTypes")
}

func (r *agreementRepository) replaceChildren(
	ctx context.Context,
	entity *factoring.Agreement,
) error {
	dba := r.db.DBForContext(ctx)
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	if _, err := dba.NewDelete().
		Model((*factoring.FeeTier)(nil)).
		Apply(func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return factoringcol.FeeTierScopeTenantDelete(dq, tenant).
				Where(factoringcol.FeeTierColumns.AgreementID.Eq(), entity.ID)
		}).
		Exec(ctx); err != nil {
		return fmt.Errorf("clear factoring fee tiers: %w", err)
	}

	if len(entity.FeeTiers) > 0 {
		tiers := make([]*factoring.FeeTier, 0, len(entity.FeeTiers))
		for _, tier := range factoring.SortedFeeTiers(entity.FeeTiers) {
			tiers = append(tiers, &factoring.FeeTier{
				OrganizationID: entity.OrganizationID,
				BusinessUnitID: entity.BusinessUnitID,
				AgreementID:    entity.ID,
				DaysFrom:       tier.DaysFrom,
				DaysTo:         tier.DaysTo,
				RateBP:         tier.RateBP,
				FlatMinor:      tier.FlatMinor,
			})
		}
		if _, err := dba.NewInsert().Model(&tiers).Exec(ctx); err != nil {
			return fmt.Errorf("insert factoring fee tiers: %w", err)
		}
	}

	if _, err := dba.NewDelete().
		Model((*factoring.AgreementDocumentType)(nil)).
		Apply(func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return factoringcol.AgreementDocumentTypeScopeTenantDelete(dq, tenant).
				Where(factoringcol.AgreementDocumentTypeColumns.AgreementID.Eq(), entity.ID)
		}).
		Exec(ctx); err != nil {
		return fmt.Errorf("clear factoring agreement document types: %w", err)
	}

	typeIDs := entity.DocumentTypeIDs()
	if len(typeIDs) == 0 {
		return nil
	}
	links := make([]*factoring.AgreementDocumentType, 0, len(typeIDs))
	for _, typeID := range typeIDs {
		links = append(links, &factoring.AgreementDocumentType{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			AgreementID:    entity.ID,
			DocumentTypeID: typeID,
		})
	}
	if _, err := dba.NewInsert().Model(&links).Exec(ctx); err != nil {
		if dberror.IsForeignKeyConstraintViolation(err) {
			return errortypes.NewValidationError(
				"documentTypeIds",
				errortypes.ErrInvalid,
				"One of the required document types does not exist",
			)
		}
		return fmt.Errorf("insert factoring agreement document types: %w", err)
	}

	return nil
}

func inForce(q *bun.SelectQuery, at int64) *bun.SelectQuery {
	cols := factoringcol.AgreementColumns
	return q.
		Where(cols.Status.Eq(), factoring.AgreementStatusActive).
		Where(cols.EffectiveFrom.Lte(), at).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.EffectiveTo.IsNull()).WhereOr(cols.EffectiveTo.Gt(), at)
		})
}

func translateAgreementError(err error, action string) error {
	if dberror.IsExclusionConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == activeOverlapIndex {
		return errortypes.NewValidationError(
			"effectiveFrom",
			errortypes.ErrInvalid,
			"Another active agreement with this factor covers part of these dates",
		)
	}
	if dberror.IsCheckConstraintViolation(err) {
		return errortypes.NewValidationError(
			"agreement",
			errortypes.ErrInvalid,
			"The agreement terms are not valid",
		)
	}
	return fmt.Errorf("%s: %w", action, err)
}
