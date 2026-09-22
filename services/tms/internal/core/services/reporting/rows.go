package reporting

import (
	"context"
	"io"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// maxRunRowsBytes bounds what a single read will pull into memory.
//
// It is deliberately smaller than the artifact cap: a person downloading a
// report streams it to a file, while a caller reading rows back holds the whole
// thing to compare it, and doing that inside a request is what turns one large
// report into an outage.
const maxRunRowsBytes = 24 * 1024 * 1024

// ReadRunRows returns a finished run's rows as they were produced — raw
// values, not formatted cells.
//
// It is the only way to read a run's data on the server. DownloadRun mints a
// presigned URL to the artifact, which for CSV and XLSX holds formatted text
// and for PDF holds no rows at all; comparing those would report a currency
// change as a price change.
func (s *Service) ReadRunRows(
	ctx context.Context,
	req *GetRunRequest,
) (*reportrows.Envelope, error) {
	run, err := s.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}

	if err = rowsReadable(run); err != nil {
		return nil, err
	}

	info, err := s.storage.GetFileInfo(ctx, run.RowsKey)
	if err != nil {
		s.l.Warn("failed to stat report rows sidecar",
			zap.String("runId", run.ID.String()),
			zap.String("key", run.RowsKey),
			zap.Error(err))

		return nil, errortypes.NewNotFoundError(
			"The stored rows for this report run are no longer available",
		)
	}
	if info.Size > maxRunRowsBytes {
		return nil, errortypes.NewBusinessError(
			"This report run is too large to read back — narrow its filters and run it again",
		)
	}

	data, err := s.downloadRunRows(ctx, run)
	if err != nil {
		return nil, err
	}

	envelope, err := reportrows.Decode(data)
	if err != nil {
		s.l.Error("stored report rows could not be decoded",
			zap.String("runId", run.ID.String()),
			zap.String("key", run.RowsKey),
			zap.Error(err))

		return nil, errortypes.NewBusinessError(
			"The stored rows for this report run could not be read — run the report again",
		)
	}

	s.auditRowsRead(run)

	return envelope, nil
}

// rowsReadable names the reason a run's rows cannot be read, separately for
// each case: a caller deciding whether two runs can be compared needs to know
// which of them is the problem and why.
func rowsReadable(run *report.ReportRun) error {
	switch {
	case run.Status != report.RunStatusSucceeded:
		return errortypes.NewBusinessError(
			"This report run did not finish successfully, so it has no rows to read",
		)
	case run.RowsKey == "":
		return errortypes.NewBusinessError(
			"This report run was generated before its rows were stored, so it cannot be compared",
		)
	case run.ArtifactExpiresAt > 0 && run.ArtifactExpiresAt < timeutils.NowUnix():
		return errortypes.NewBusinessError(
			"The stored rows for this report run have expired — run the report again",
		)
	}

	return nil
}

func (s *Service) downloadRunRows(
	ctx context.Context,
	run *report.ReportRun,
) ([]byte, error) {
	result, err := s.storage.Download(ctx, run.RowsKey)
	if err != nil {
		s.l.Warn("failed to download report rows sidecar",
			zap.String("runId", run.ID.String()),
			zap.String("key", run.RowsKey),
			zap.Error(err))

		return nil, errortypes.NewNotFoundError(
			"The stored rows for this report run are no longer available",
		)
	}
	defer result.Body.Close()

	// The stat above can be stale or, on some backends, unset, so the cap is
	// enforced again on the bytes themselves. Reading one byte past the limit
	// is what tells a file at the limit from one over it.
	data, err := io.ReadAll(io.LimitReader(result.Body, maxRunRowsBytes+1))
	if err != nil {
		s.l.Error("failed to read report rows sidecar",
			zap.String("runId", run.ID.String()), zap.Error(err))

		return nil, errortypes.NewBusinessError(
			"The stored rows for this report run could not be read — try again shortly",
		)
	}
	if len(data) > maxRunRowsBytes {
		return nil, errortypes.NewBusinessError(
			"This report run is too large to read back — narrow its filters and run it again",
		)
	}

	return data, nil
}

func (s *Service) auditRowsRead(run *report.ReportRun) {
	if s.audit == nil {
		return
	}
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceReport,
		ResourceID:     run.ID.String(),
		Operation:      permission.OpRead,
		UserID:         run.RequestedByID,
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
		CurrentState: map[string]any{
			"event":  "rows_read",
			"format": string(run.Format),
		},
	}); err != nil {
		s.l.Warn("failed to audit report rows read",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}
