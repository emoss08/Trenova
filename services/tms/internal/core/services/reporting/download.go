package reporting

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	downloadURLExpiry  = 60 * time.Second
	maxDownloadNameLen = 120
)

type RunDownload struct {
	URL      string
	FileName string
}

func (s *Service) DownloadRun(ctx context.Context, req *GetRunRequest) (*RunDownload, error) {
	run, err := s.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}

	if run.Status != report.RunStatusSucceeded {
		return nil, errortypes.NewBusinessError(
			"This report run has no downloadable artifact",
		)
	}
	if run.ArtifactKey == "" {
		return nil, errortypes.NewNotFoundError("The report artifact is no longer available")
	}
	if run.ArtifactExpiresAt > 0 && run.ArtifactExpiresAt < timeutils.NowUnix() {
		return nil, errortypes.NewBusinessError(
			"This report artifact has expired — run the report again",
		)
	}

	fileName := s.downloadFileName(ctx, req, run)

	url, err := s.storage.GetPresignedURL(ctx, &storage.PresignedURLParams{
		Key:                run.ArtifactKey,
		Expiry:             downloadURLExpiry,
		ContentDisposition: fmt.Sprintf("attachment; filename=%q", fileName),
	})
	if err != nil {
		s.l.Error("failed to presign report artifact",
			zap.String("runId", run.ID.String()), zap.Error(err))
		return nil, errortypes.NewBusinessError(
			"The report artifact could not be prepared for download — try again shortly",
		)
	}

	s.auditDownload(run)

	return &RunDownload{URL: url, FileName: fileName}, nil
}

func (s *Service) downloadFileName(
	ctx context.Context,
	req *GetRunRequest,
	run *report.ReportRun,
) string {
	base := "report"
	switch {
	case !run.DefinitionID.IsNil():
		if def, err := s.GetDefinition(ctx, &GetDefinitionRequest{
			Request:      req.Request,
			DefinitionID: run.DefinitionID,
		}); err == nil {
			base = def.Name
		}
	case run.CannedKey != "":
		if entry, ok := s.canned.Get(run.CannedKey); ok {
			base = entry.Name
		}
	}

	stamp := time.Unix(run.CreatedAt, 0).UTC().Format("2006-01-02")
	return fmt.Sprintf(
		"%s %s.%s",
		fileutils.SanitizeDisplayFilename(base, "report", maxDownloadNameLen),
		stamp,
		run.Format.Extension(),
	)
}

func (s *Service) auditDownload(run *report.ReportRun) {
	if s.audit == nil {
		return
	}
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceReport,
		ResourceID:     run.ID.String(),
		Operation:      permission.OpExport,
		UserID:         run.RequestedByID,
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
		CurrentState: map[string]any{
			"event":  "artifact_downloaded",
			"format": string(run.Format),
		},
	}); err != nil {
		s.l.Warn("failed to audit report download",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}
