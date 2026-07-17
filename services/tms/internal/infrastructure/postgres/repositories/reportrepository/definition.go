package reportrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DefinitionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type definitionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDefinitionRepository(p DefinitionParams) repositories.ReportDefinitionRepository {
	return &definitionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-definition-repository"),
	}
}

func (r *definitionRepository) Create(
	ctx context.Context,
	entity *report.ReportDefinition,
	createdBy pulid.ID,
) (*report.ReportDefinition, error) {
	log := r.l.With(zap.String("operation", "Create"), zap.String("name", entity.Name))

	entity.CurrentRevision = 1

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(entity).Exec(txCtx); iErr != nil {
			return iErr
		}

		revision := newRevision(entity, createdBy)
		_, iErr := tx.NewInsert().Model(revision).Exec(txCtx)
		return iErr
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError("ReportDefinition", entity.Name)
		}
		log.Error("failed to create report definition", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *definitionRepository) Update(
	ctx context.Context,
	entity *report.ReportDefinition,
	updatedBy pulid.ID,
) (*report.ReportDefinition, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++
	entity.CurrentRevision++

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		result, uErr := tx.NewUpdate().
			Model(entity).
			WherePK().
			Where(buncolgen.ReportDefinitionColumns.Version.Eq(), ov).
			Returning("*").
			Exec(txCtx)
		if uErr != nil {
			return uErr
		}
		uErr = dberror.CheckRowsAffected(result, "ReportDefinition", entity.ID.String())
		if uErr != nil {
			return uErr
		}

		revision := newRevision(entity, updatedBy)
		_, iErr := tx.NewInsert().Model(revision).Exec(txCtx)
		return iErr
	})
	if err != nil {
		log.Error("failed to update report definition", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func newRevision(
	entity *report.ReportDefinition,
	createdBy pulid.ID,
) *report.ReportDefinitionRevision {
	return &report.ReportDefinitionRevision{
		BusinessUnitID: entity.BusinessUnitID,
		OrganizationID: entity.OrganizationID,
		DefinitionID:   entity.ID,
		RevisionNumber: entity.CurrentRevision,
		CatalogVersion: entity.CatalogVersion,
		Definition:     entity.Definition,
		CreatedByID:    createdBy,
	}
}

func (r *definitionRepository) UpdateStatus(
	ctx context.Context,
	req *repositories.GetReportDefinitionRequest,
	status report.DefinitionStatus,
	diagnostics []string,
) error {
	log := r.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("id", req.DefinitionID.String()),
	)

	cols := buncolgen.ReportDefinitionColumns
	result, err := r.db.DB().
		NewUpdate().
		Model((*report.ReportDefinition)(nil)).
		Set(cols.Status.Set(), status).
		Set(cols.Diagnostics.Set(), pgArray(diagnostics)).
		Set(cols.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ReportDefinitionScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.DefinitionID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update report definition status", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "ReportDefinition", req.DefinitionID.String())
}

func (r *definitionRepository) GetByID(
	ctx context.Context,
	req *repositories.GetReportDefinitionRequest,
) (*report.ReportDefinition, error) {
	entity := new(report.ReportDefinition)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReportDefinitionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ReportDefinitionColumns.ID.Eq(), req.DefinitionID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "ReportDefinition")
	}

	return entity, nil
}

func (r *definitionRepository) List(
	ctx context.Context,
	req *repositories.ListReportDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	cols := buncolgen.ReportDefinitionColumns

	entities := make([]*report.ReportDefinition, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(buncolgen.ReportDefinitionApplyTenant(req.TenantInfo))

	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	if !req.OwnerID.IsNil() {
		q = q.Where(cols.OwnerID.Eq(), req.OwnerID)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}

	if err := q.Order(cols.UpdatedAt.OrderDesc()).Scan(ctx); err != nil {
		r.l.Error("failed to list report definitions", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *definitionRepository) Delete(
	ctx context.Context,
	req *repositories.DeleteReportDefinitionRequest,
) error {
	result, err := r.db.DB().
		NewDelete().
		Model((*report.ReportDefinition)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ReportDefinitionScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.ReportDefinitionColumns.ID.Eq(), req.DefinitionID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete report definition", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "ReportDefinition", req.DefinitionID.String())
}

func (r *definitionRepository) GetRevision(
	ctx context.Context,
	req *repositories.GetReportRevisionRequest,
) (*report.ReportDefinitionRevision, error) {
	entity := new(report.ReportDefinitionRevision)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReportDefinitionRevisionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ReportDefinitionRevisionColumns.ID.Eq(), req.RevisionID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "ReportDefinitionRevision")
	}

	return entity, nil
}

func (r *definitionRepository) ListRevisions(
	ctx context.Context,
	req *repositories.ListReportRevisionsRequest,
) ([]*report.ReportDefinitionRevision, error) {
	cols := buncolgen.ReportDefinitionRevisionColumns

	entities := make([]*report.ReportDefinitionRevision, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(buncolgen.ReportDefinitionRevisionApplyTenant(req.TenantInfo)).
		Where(cols.DefinitionID.Eq(), req.DefinitionID).
		Order(cols.RevisionNumber.OrderDesc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list report definition revisions", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func pgArray(values []string) any {
	if len(values) == 0 {
		return nil
	}
	return pgdialect.Array(values)
}
