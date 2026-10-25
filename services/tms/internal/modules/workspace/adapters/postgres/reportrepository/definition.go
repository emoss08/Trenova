package reportrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const definitionEntityName = "ReportDefinition"

var (
	definitionErrors = dberror.Mapping{Entity: definitionEntityName}
	revisionErrors   = dberror.Mapping{Entity: "ReportDefinitionRevision"}
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

func NewDefinitionRepository(p DefinitionParams) reportports.ReportDefinitionRepository {
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
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.ReportDefinition, error) {
		db := r.db.Keel(ctx)
		entity.CurrentRevision = 1

		err := reportdb.ReportDefinitions.Create(ctx, db, entity)
		if err == nil {
			err = reportdb.ReportDefinitionRevisions.Create(ctx, db, newRevision(entity, createdBy))
		}
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError(definitionEntityName, entity.Name)
		}
		if err != nil {
			r.l.Error(
				"failed to create report definition",
				zap.String("name", entity.Name),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *definitionRepository) Update(
	ctx context.Context,
	entity *report.ReportDefinition,
	updatedBy pulid.ID,
) (*report.ReportDefinition, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.ReportDefinition, error) {
		db := r.db.Keel(ctx)
		entity.CurrentRevision++

		err := reportdb.ReportDefinitions.Save(ctx, db, entity)
		if err == nil {
			err = reportdb.ReportDefinitionRevisions.Create(ctx, db, newRevision(entity, updatedBy))
		}
		if err != nil {
			entity.CurrentRevision--
			r.l.Error(
				"failed to update report definition",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, definitionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
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
	req *reportports.GetReportDefinitionRequest,
	status report.DefinitionStatus,
	diagnostics []string,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := reportdb.ReportDefinitions
		rows, err := d.Where(d.ID.Eq(req.DefinitionID)).Update(
			ctx,
			r.db.Keel(ctx),
			d.Status.To(status),
			d.Diagnostics.To(diagnostics),
			d.UpdatedAt.ToExpr(d.UpdatedAt),
		)
		if err != nil {
			r.l.Error(
				"failed to update report definition status",
				zap.String("id", req.DefinitionID.String()),
				zap.Error(err),
			)
			return err
		}

		return dberror.CheckAffected(rows, definitionEntityName, req.DefinitionID.String())
	})
}

func (r *definitionRepository) GetByID(
	ctx context.Context,
	req *reportports.GetReportDefinitionRequest,
) (*report.ReportDefinition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*report.ReportDefinition, error) {
		entity, err := reportdb.ReportDefinitions.Get(ctx, r.db.Keel(ctx), req.DefinitionID)
		if err != nil {
			return nil, definitionErrors.Map(err, req.DefinitionID.String())
		}

		return entity, nil
	})
}

func (r *definitionRepository) List(
	ctx context.Context,
	req *reportports.ListReportDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportDefinition, error) {
		d := reportdb.ReportDefinitions
		q := d.Where(
			definitionVisibleTo(req.ViewerID),
			keel.When(len(req.Statuses) > 0, d.Status.In(req.Statuses...)),
			keel.When(!req.OwnerID.IsNil(), d.OwnerID.Eq(req.OwnerID)),
		).Order(d.UpdatedAt.Desc())
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}
		if req.Offset > 0 {
			q = q.Offset(req.Offset)
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list report definitions", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *definitionRepository) Delete(
	ctx context.Context,
	req *reportports.DeleteReportDefinitionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := reportdb.ReportDefinitions
		rows, err := d.Where(d.ID.Eq(req.DefinitionID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete report definition", zap.Error(err))
			return err
		}

		return dberror.CheckAffected(rows, definitionEntityName, req.DefinitionID.String())
	})
}

func (r *definitionRepository) GetRevision(
	ctx context.Context,
	req *reportports.GetReportRevisionRequest,
) (*report.ReportDefinitionRevision, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*report.ReportDefinitionRevision, error) {
			entity, err := reportdb.ReportDefinitionRevisions.Get(
				ctx,
				r.db.Keel(ctx),
				req.RevisionID,
			)
			if err != nil {
				return nil, revisionErrors.Map(err, req.RevisionID.String())
			}

			return entity, nil
		},
	)
}

func (r *definitionRepository) ListRevisions(
	ctx context.Context,
	req *reportports.ListReportRevisionsRequest,
) ([]*report.ReportDefinitionRevision, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*report.ReportDefinitionRevision, error) {
			rev := reportdb.ReportDefinitionRevisions
			q := rev.Where(rev.DefinitionID.Eq(req.DefinitionID)).Order(rev.RevisionNumber.Desc())
			if req.Limit > 0 {
				q = q.Limit(req.Limit)
			}

			entities, err := q.All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list report definition revisions", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}
