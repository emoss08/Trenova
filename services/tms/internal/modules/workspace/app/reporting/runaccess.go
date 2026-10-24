package reporting

import (
	"context"
	"errors"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

// authorizeStoredRun checks a finished run's report against the requester's
// grants as they are now, as running it again would: a run is the requester's
// export, so access withdrawn since it ran withdraws its artifact and rows too.
func (s *Service) authorizeStoredRun(
	ctx context.Context,
	req *GetRunRequest,
	run *report.ReportRun,
) error {
	definition, err := s.storedRunDefinition(ctx, req, run)
	if err != nil {
		return err
	}

	err = s.Check(ctx, &PrepareParams{
		Tenant:     req.TenantInfo,
		Definition: definition,
		Params:     run.Params,
		Mode:       reportaccess.ModeExport,
	})
	if err == nil {
		return nil
	}

	var authzErr *errortypes.AuthorizationError
	if errors.As(err, &authzErr) {
		return authzErr
	}
	s.l.Warn("stored report run no longer compiles",
		zap.String("runId", run.ID.String()), zap.Error(err))

	return errortypes.NewBusinessError(
		"This report has changed since it ran, so its result cannot be served — run it again",
	)
}

func (s *Service) storedRunDefinition(
	ctx context.Context,
	req *GetRunRequest,
	run *report.ReportRun,
) (*ir.Definition, error) {
	switch {
	case !run.RevisionID.IsNil():
		revision, err := s.defRepo.GetRevision(ctx, &reportports.GetReportRevisionRequest{
			TenantInfo: req.TenantInfo,
			RevisionID: run.RevisionID,
		})
		if err != nil {
			return nil, err
		}
		return revision.Definition, nil
	case run.CannedKey != "":
		entry, ok := s.canned.Get(run.CannedKey)
		if !ok {
			return nil, errortypes.NewBusinessError(
				"The report this run came from is no longer available — run it again",
			)
		}
		return entry.Definition, nil
	default:
		return nil, errortypes.NewBusinessError(
			"This report run is bound to no report, so its result cannot be served",
		)
	}
}
