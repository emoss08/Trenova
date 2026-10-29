package assistantartifacts

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	payloadRanByPerson = "ranByPerson"
	payloadStatus      = "status"
)

type reportRerunner interface {
	GetRun(ctx context.Context, req *reporting.GetRunRequest) (*report.ReportRun, error)
	RunReport(ctx context.Context, req *reporting.RunReportRequest) (*report.ReportRun, error)
}

func reportRerunnerOf(reports *reporting.Service) reportRerunner {
	if reports == nil {
		return nil
	}

	return reports
}

var errNotRunnable = errortypes.NewValidationError(
	"artifact", errortypes.ErrInvalid, "Only a report run or a report preview can be run",
)

var errPreviewNotSaved = errortypes.NewBusinessError(
	"This preview is of a report that is not saved, so it cannot be run in full. Save it in Reports first",
)
