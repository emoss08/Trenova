package desktranscriptrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript/desktranscriptdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const oneLiveExportConstraint = "uq_desk_transcript_exports_one_live"

var exportErrors = dberror.Mapping{Entity: "Desk transcript export"}

type exports struct {
	entitlements entitlementports.Filter
	db           *postgres.Connection
	l            *zap.Logger
}

func NewExports(p Params) desktranscriptports.ExportRepository {
	return &exports{
		entitlements: p.Entitlements,
		db:           p.DB,
		l:            p.Logger.Named("postgres.desk-transcript-exports"),
	}
}

func ErrExportUnderWay() error {
	return errortypes.NewBusinessError(
		"An export of Desk conversations is already being written for your organization. Wait for it to finish, then export again.",
	)
}

func (r *exports) Create(
	ctx context.Context,
	export *desktranscript.Export,
) (*desktranscript.Export, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*desktranscript.Export, error) {
		if err := desktranscriptdb.Exports.Create(ctx, r.db.Keel(ctx), export); err != nil {
			if keel.IsUnique(err, oneLiveExportConstraint) {
				return nil, ErrExportUnderWay()
			}
			r.l.Error("failed to create a desk transcript export", zap.Error(err))

			return nil, exportErrors.Map(err, export.ID.String())
		}

		return export, nil
	})
}

func (r *exports) Update(
	ctx context.Context,
	export *desktranscript.Export,
) (*desktranscript.Export, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*desktranscript.Export, error) {
		x := desktranscriptdb.Exports
		if err := x.SaveExcept(ctx, r.db.Keel(ctx), export,
			x.RequestedByUserID,
			x.Kind,
			x.Filter,
			x.ThreadIDs,
			x.Coverage,
		); err != nil {
			r.l.Error("failed to update a desk transcript export",
				zap.String("exportId", export.ID.String()), zap.Error(err))

			return nil, exportErrors.Map(err, export.ID.String())
		}

		return export, nil
	})
}

func (r *exports) GetByID(
	ctx context.Context,
	req desktranscriptports.GetExportRequest,
) (*desktranscript.Export, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*desktranscript.Export, error) {
		entity, err := desktranscriptdb.Exports.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, exportErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *exports) ListConnection(
	ctx context.Context,
	req *desktranscriptports.ListExportsRequest,
) (*pagination.CursorListResult[*desktranscript.Export], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*desktranscript.Export], error) {
			x := desktranscriptdb.Exports
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				x.Query(),
				&listquery.Request[desktranscript.Export]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q keel.Query[*desktranscript.Export]) keel.Query[*desktranscript.Export] {
						return q.Omit(x.ThreadIDs)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list desk transcript exports", zap.Error(err))

				return nil, fmt.Errorf("list desk transcript exports: %w", err)
			}

			return result, nil
		},
	)
}

func (r *exports) ListExpired(
	ctx context.Context,
	before int64,
	limit int,
) ([]*desktranscript.Export, error) {
	ctx = dbscope.WithSystem(ctx, "list expired desk transcript exports across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*desktranscript.Export, error) {
		x := desktranscriptdb.Exports
		entitled, entitledErr := r.entitlements.ForWork(ctx, x.OrganizationID.Col)
		if entitledErr != nil {
			return nil, entitledErr
		}
		rows, err := x.AllTenants().
			Omit(x.ThreadIDs).
			Where(
				entitled,
				x.ArtifactKey.Ne(""),
				x.ArtifactExpiresAt.Lte(before),
				x.Status.Eq(desktranscript.ExportStatusSucceeded),
			).
			Order(x.ArtifactExpiresAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list expired desk transcript exports: %w", err)
		}

		return rows, nil
	})
}

func (r *exports) ListStale(
	ctx context.Context,
	updatedBefore int64,
	limit int,
) ([]*desktranscript.Export, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list abandoned desk transcript exports across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*desktranscript.Export, error) {
		x := desktranscriptdb.Exports
		entitled, entitledErr := r.entitlements.ForWork(ctx, x.OrganizationID.Col)
		if entitledErr != nil {
			return nil, entitledErr
		}
		rows, err := x.AllTenants().
			Omit(x.ThreadIDs).
			Where(
				entitled,
				x.Status.In(desktranscript.ExportStatusPending, desktranscript.ExportStatusRunning),
				x.UpdatedAt.Lt(updatedBefore),
			).
			Order(x.UpdatedAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list abandoned desk transcript exports: %w", err)
		}

		return rows, nil
	})
}
