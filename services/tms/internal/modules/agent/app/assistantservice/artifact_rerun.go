package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
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

//nolint:gocritic // AssistantService methods take the thread request by value
func (s *Service) RunArtifactReport(
	ctx context.Context,
	req agentports.GetThreadRequest,
	artifactID pulid.ID,
) (*agentports.AssistantArtifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	if s.reports == nil || s.reportRuns == nil {
		return nil, errortypes.NewBusinessError("Reports cannot be run here")
	}

	versions, err := s.artifacts.ListLineage(ctx, assistantartifactports.LineageRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		ID:         artifactID,
	})
	if err != nil {
		return nil, err
	}
	var shown *assistantartifact.Artifact
	for _, version := range versions {
		if version.ID == artifactID {
			shown = version
		}
	}
	if shown == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	actor := permissionports.UserActor(req.TenantInfo)
	request := agentquerytoolservice.ReportingRequestFor(actor)
	runReq, name, err := s.rerunOf(ctx, &request, actor, shown)
	if err != nil {
		return nil, err
	}

	run, err := s.reportRuns.RunReport(ctx, runReq)
	if err != nil {
		return nil, err
	}

	return s.appendRunVersion(ctx, &req, latestOf(versions), run, name)
}

func (s *Service) rerunOf(
	ctx context.Context,
	request *reporting.Request,
	actor *permissionports.RequestActor,
	shown *assistantartifact.Artifact,
) (*reporting.RunReportRequest, string, error) {
	//nolint:exhaustive // only a report's run or preview names a report to run
	switch shown.Kind {
	case assistantartifact.KindReportRun:
		runID, err := pulid.MustParse(typeutils.StringOfTrimmed(shown.Payload["runId"]))
		if err != nil {
			return nil, "", errNotRunnable
		}
		previous, err := s.reportRuns.GetRun(ctx, &reporting.GetRunRequest{
			Request: *request,
			RunID:   runID,
		})
		if err != nil {
			return nil, "", err
		}

		return &reporting.RunReportRequest{
			Request:      *request,
			DefinitionID: previous.DefinitionID,
			CannedKey:    previous.CannedKey,
			Format:       previous.Format,
			Params:       previous.Params,
			Trigger:      report.RunTriggerManual,
		}, typeutils.StringOfTrimmed(shown.Payload["reportName"]), nil
	case assistantartifact.KindReportPreview:
		arguments, recorded := shown.Payload[payloadReadWith].(map[string]any)
		if !recorded {
			return nil, "", errNotReread
		}
		previewed, err := agentquerytoolservice.ReportOfPreview(ctx, s.reports, actor, arguments)
		if err != nil {
			return nil, "", errortypes.NewBusinessError(
				"This preview can no longer be run: " + err.Error(),
			)
		}
		if previewed.DefinitionID.IsNil() && previewed.Key == "" {
			return nil, "", errPreviewNotSaved
		}
		format := previewed.DefaultFormat
		if !format.IsValid() {
			format = report.DefaultScheduleFormat()
		}

		return &reporting.RunReportRequest{
			Request:      *request,
			DefinitionID: previewed.DefinitionID,
			CannedKey:    previewed.Key,
			Format:       format,
			Params:       previewed.Params,
			Trigger:      report.RunTriggerManual,
		}, previewed.Name, nil
	default:
		return nil, "", errNotRunnable
	}
}

func (s *Service) appendRunVersion(
	ctx context.Context,
	req *agentports.GetThreadRequest,
	latest *assistantartifact.Artifact,
	run *report.ReportRun,
	name string,
) (*agentports.AssistantArtifact, error) {
	result := map[string]any{
		"runId":          run.ID.String(),
		"reportKey":      run.CannedKey,
		"reportName":     name,
		payloadStatus:    string(run.Status),
		"finished":       run.Status.IsTerminal(),
		"format":         string(run.Format),
		payloadRowCount:  run.RowCount,
		payloadTruncated: run.Truncated,
	}
	if !run.DefinitionID.IsNil() {
		result["definitionId"] = run.DefinitionID.String()
	}
	built := runArtifact("", result)
	built.Payload[payloadRanByPerson] = true

	next := &assistantartifact.Artifact{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ThreadID:       req.ID,
		MessageID:      latest.MessageID,
		RunID:          latest.RunID,
		Kind:           built.Kind,
		Status:         built.Status,
		Title:          latest.Title,
		Slug:           latest.Slug,
		Pinned:         latest.Pinned,
		LineageKey:     latest.LineageKey,
		Payload:        built.Payload,
	}
	next.FollowLineage(latest)

	multiErr := errortypes.NewMultiError()
	next.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.artifacts.InsertVersion(ctx, next)
	if err != nil {
		return nil, err
	}
	if s.activity != nil {
		s.activity.ArtifactChanged(ctx, saved, kernelports.AuditActor{
			PrincipalType: permissionports.PrincipalTypeUser,
			PrincipalID:   req.UserID,
			UserID:        req.UserID,
		}, agentports.ActivityUpdated)
	}
	out := toAssistantArtifact(saved)

	return &out, nil
}
