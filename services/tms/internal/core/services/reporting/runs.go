package reporting

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/reportjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type RunReportRequest struct {
	Request

	DefinitionID pulid.ID
	CannedKey    string
	Format       report.Format
	Params       map[string]any
	Trigger      report.RunTrigger
}

func (s *Service) RunReport(ctx context.Context, req *RunReportRequest) (*report.ReportRun, error) {
	log := s.l.With(zap.String("operation", "RunReport"))

	if !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Report generation is temporarily unavailable — the background worker is not connected",
		)
	}

	format := req.Format
	if !format.IsValid() {
		return nil, errortypes.NewValidationError(
			"format", errortypes.ErrInvalid, "Format must be csv, xlsx, pdf, or json",
		)
	}

	if err := s.enforceEnqueueGate(ctx, req); err != nil {
		return nil, err
	}

	run, err := s.buildRun(ctx, req, format)
	if err != nil {
		return nil, err
	}

	created, err := s.runRepo.Create(ctx, run)
	if err != nil {
		log.Error("failed to create report run", zap.Error(err))
		return nil, err
	}

	workflowID := "report-run/" + created.ID.String()
	if _, err = s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                    workflowID,
			TaskQueue:             temporaltype.ReportTaskQueue,
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		},
		reportjobs.RunReportWorkflow,
		&reportjobs.RunReportPayload{
			RunID:          created.ID,
			OrganizationID: created.OrganizationID,
			BusinessUnitID: created.BusinessUnitID,
		},
	); err != nil {
		log.Error("failed to start report workflow", zap.Error(err))
		created.Status = report.RunStatusFailed
		created.Error = &report.RunError{
			Code:    "ENQUEUE_FAILED",
			Message: "The report could not be queued for generation",
		}
		if _, updateErr := s.runRepo.Update(ctx, created); updateErr != nil {
			log.Error("failed to mark unqueued run as failed", zap.Error(updateErr))
		}
		return nil, errortypes.NewBusinessError(
			"The report could not be queued for generation — try again shortly",
		)
	}

	return created, nil
}

func (s *Service) enforceEnqueueGate(ctx context.Context, req *RunReportRequest) error {
	counts, err := s.runRepo.CountActive(ctx, &repositories.CountActiveReportRunsRequest{
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if counts.Running >= s.cfg.GetMaxConcurrentRunsPerOrg() {
		if s.metrics != nil {
			s.metrics.RecordEnqueueRejection("concurrency")
		}
		return errortypes.NewRateLimitError("report",
			fmt.Sprintf(
				"Your organization already has %d reports generating — wait for one to finish",
				counts.Running,
			),
		)
	}
	if counts.Queued >= s.cfg.GetMaxQueuedRunsPerOrg() {
		if s.metrics != nil {
			s.metrics.RecordEnqueueRejection("queue_depth")
		}
		return errortypes.NewRateLimitError("report",
			fmt.Sprintf(
				"Your organization already has %d reports queued — wait for the queue to drain",
				counts.Queued,
			),
		)
	}

	return nil
}

func (s *Service) buildRun(
	ctx context.Context,
	req *RunReportRequest,
	format report.Format,
) (*report.ReportRun, error) {
	trigger := req.Trigger
	if !trigger.IsValid() {
		trigger = report.RunTriggerManual
	}

	run := &report.ReportRun{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		RequestedByID:  req.TenantInfo.UserID,
		Trigger:        trigger,
		Params:         req.Params,
		Format:         format,
		Status:         report.RunStatusQueued,
	}

	switch {
	case !req.DefinitionID.IsNil():
		definition, err := s.GetDefinition(ctx, &GetDefinitionRequest{
			Request:      req.Request,
			DefinitionID: req.DefinitionID,
		})
		if err != nil {
			return nil, err
		}
		if definition.Status == report.DefinitionStatusNeedsAttention {
			return nil, errortypes.NewBusinessError(
				"This report references fields that no longer exist and needs to be repaired before it can run",
			)
		}
		if definition.Status == report.DefinitionStatusArchived {
			return nil, errortypes.NewBusinessError("Archived reports cannot be run")
		}

		revision, err := s.headRevision(ctx, req.TenantInfo, definition.ID)
		if err != nil {
			return nil, err
		}

		run.DefinitionID = definition.ID
		run.RevisionID = revision.ID
	case req.CannedKey != "":
		entry, err := s.GetCanned(req.CannedKey)
		if err != nil {
			return nil, err
		}
		if _, err = s.compiler.ValidateAndAuthorize(ctx, &services.ReportCompileRequest{
			Definition:  entry.Definition,
			Tenant:      req.TenantInfo,
			Params:      req.Params,
			OrgTimezone: s.orgTimezone(ctx, req.TenantInfo),
			NowUnix:     timeutils.NowUnix(),
		}); err != nil {
			if s.metrics != nil {
				s.metrics.RecordCompileError("validation")
			}
			return nil, err
		}
		run.CannedKey = entry.Key
		run.CannedVersion = entry.Version
	default:
		return nil, errortypes.NewValidationError(
			"definitionId", errortypes.ErrRequired,
			"A report definition or canned report key is required",
		)
	}

	return run, nil
}

type GetRunRequest struct {
	Request

	RunID pulid.ID
}

func (s *Service) GetRun(ctx context.Context, req *GetRunRequest) (*report.ReportRun, error) {
	return s.runRepo.GetByID(ctx, &repositories.GetReportRunRequest{
		TenantInfo: req.TenantInfo,
		RunID:      req.RunID,
	})
}

type ListRunsRequest struct {
	Request

	DefinitionID pulid.ID
	MineOnly     bool
	Statuses     []report.RunStatus
	Limit        int
	Offset       int
}

func (s *Service) ListRuns(
	ctx context.Context,
	req *ListRunsRequest,
) ([]*report.ReportRun, error) {
	listReq := &repositories.ListReportRunsRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
		Statuses:     req.Statuses,
		Limit:        req.Limit,
		Offset:       req.Offset,
	}
	if req.MineOnly {
		listReq.RequestedBy = req.TenantInfo.UserID
	}

	return s.runRepo.List(ctx, listReq)
}

func (s *Service) CancelRun(ctx context.Context, req *GetRunRequest) (*report.ReportRun, error) {
	run, err := s.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}

	if run.RequestedByID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the user who requested a report run can cancel it",
		)
	}
	if run.Status.IsTerminal() {
		return nil, errortypes.NewBusinessError("This report run has already finished")
	}

	if run.TemporalWorkflowID != "" && s.workflows.Enabled() {
		if err = s.workflows.CancelWorkflow(
			ctx, run.TemporalWorkflowID, run.TemporalRunID,
		); err != nil {
			s.l.Warn("failed to cancel report workflow",
				zap.String("runId", run.ID.String()), zap.Error(err))
		}
		return run, nil
	}

	run.Status = report.RunStatusCanceled
	run.Error = &report.RunError{Code: "CANCELED", Message: "The run was canceled"}
	return s.runRepo.Update(ctx, run)
}
