package reportrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func definitionVisibleTo(viewerID pulid.ID) keel.Pred {
	d := reportdb.ReportDefinitions
	return keel.Or(d.Visibility.Eq(report.VisibilityShared), d.OwnerID.Eq(viewerID))
}

func (r *definitionRepository) ListConnection(
	ctx context.Context,
	req *reportports.ListReportDefinitionConnectionRequest,
) (*pagination.CursorListResult[*report.ReportDefinition], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*report.ReportDefinition], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				reportdb.ReportDefinitions.Where(definitionVisibleTo(req.ViewerID)),
				&listquery.Request[report.ReportDefinition]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list report definitions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func runConnectionScope(req *reportports.ListReportRunConnectionRequest) []keel.Pred {
	run := reportdb.ReportRuns
	return []keel.Pred{
		keel.When(!req.DefinitionID.IsNil(), run.DefinitionID.Eq(req.DefinitionID)),
		keel.When(!req.RequestedBy.IsNil(), run.RequestedByID.Eq(req.RequestedBy)),
		keel.When(len(req.Statuses) > 0, run.Status.In(req.Statuses...)),
	}
}

func (r *runRepository) ListConnection(
	ctx context.Context,
	req *reportports.ListReportRunConnectionRequest,
) (*pagination.CursorListResult[*report.ReportRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*report.ReportRun], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				reportdb.ReportRuns.Where(runConnectionScope(req)...),
				&listquery.Request[report.ReportRun]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list report runs", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}
