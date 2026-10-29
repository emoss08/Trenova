package assistantartifacts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/attribution"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/reportquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/docrender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

func (s *Service) exportSourceOf(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
) (*exportSource, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	artifact, err := s.artifacts.GetByID(ctx, assistantartifactports.GetArtifactRequest{
		ID:         artifactID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if artifact.ThreadID != req.ID {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	if !artifactpayload.Rereadable(artifact) {
		return nil, errNotExportable
	}
	arguments, recorded := artifact.Payload[artifactpayload.PayloadReadWith].(map[string]any)
	if !recorded {
		return nil, errNotReread
	}

	return &exportSource{artifact: artifact, arguments: arguments}, nil
}

// ArtifactCSVName is the file a table downloads as, named for its title.
func (s *Service) ArtifactCSVName(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
) (string, error) {
	source, err := s.exportSourceOf(ctx, req, artifactID)
	if err != nil {
		return "", err
	}

	return assistantartifact.SlugBase(source.artifact.Title) + ".csv", nil
}

// ExportArtifactCSV writes a table or a report preview whole. The pane holds
// the rows the agent was shown, cut to what fits; the download is read again
// from the source, so it holds every row the person may see.
func (s *Service) ExportArtifactCSV(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
	sink io.Writer,
) error {
	source, err := s.exportSourceOf(ctx, req, artifactID)
	if err != nil {
		return err
	}
	actor := permissionports.UserActor(req.TenantInfo)

	if source.artifact.Kind == assistantartifact.KindReportPreview {
		return s.exportReport(ctx, actor, source, sink)
	}

	return s.exportTable(ctx, req, actor, source, sink)
}

func (s *Service) exportReport(
	ctx context.Context,
	actor *permissionports.RequestActor,
	source *exportSource,
	sink io.Writer,
) error {
	if s.reports == nil {
		return errortypes.NewBusinessError("Reports cannot be downloaded here")
	}
	previewed, err := reportquerytools.ReportOfPreview(ctx, s.reports, actor, source.arguments)
	if err != nil {
		return errortypes.NewBusinessError("This preview can no longer be run: " + err.Error())
	}

	_, err = s.reports.StreamCSV(ctx, &reporting.StreamCSVRequest{
		Request:    reportquerytools.ReportingRequestFor(actor),
		Title:      source.artifact.Title,
		Definition: previewed.Definition,
		Params:     previewed.Params,
	}, sink)

	return err
}

// exportTable reads the list the table came from again, page by page, with
// the arguments it was asked with and the person's own access.
func (s *Service) exportTable(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	actor *permissionports.RequestActor,
	source *exportSource,
	sink io.Writer,
) error {
	list, err := s.listOf(ctx, &req, actor, source)
	if err != nil {
		return err
	}

	return writeTableCSV(sink, func(at tablePosition) (map[string]any, error) {
		page := map[string]any{argLimit: exportPageRows}
		switch {
		case at.after != "":
			page[artifactpayload.ArgAfter] = at.after
		case at.offset > 0:
			page[artifactpayload.ArgOffset] = at.offset
		}

		return list.read(ctx, page, false)
	}, list.entity)
}

// mayRead checks the person may read what the tool reads, as the runtime
// does before the agent runs it for them.
func (s *Service) mayRead(
	ctx context.Context,
	actor *permissionports.RequestActor,
	policy agenttoolports.ToolPolicy,
) error {
	if policy.Resource == "" {
		return nil
	}
	if s.permissions == nil {
		return errortypes.NewAuthorizationError("This table could not be authorized")
	}
	operation := policy.Operation
	if operation == "" {
		operation = permission.OpRead
	}
	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       policy.Resource.String(),
		Operation:      operation,
	})
	if err != nil {
		return err
	}
	if !result.Allowed {
		return errortypes.NewAuthorizationError(
			fmt.Sprintf("You do not have %s access to %s", operation, policy.Resource.String()),
		)
	}

	return nil
}

// timezoneOf is the person's own timezone, which is what "today" in the
// arguments the agent passed was read in.
func (s *Service) timezoneOf(ctx context.Context, req conversationports.GetThreadRequest) string {
	if s.users == nil {
		return ""
	}
	user, err := s.users.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   req.TenantInfo,
		LookupUserID: req.UserID,
	})
	if err != nil || user == nil {
		return ""
	}

	return user.Timezone
}

// ceilingOf is how sensitive a field the conversation's agent may read, which
// bounds what its table showed and so what its download may hold.
func (s *Service) ceilingOf(
	ctx context.Context,
	req conversationports.GetThreadRequest,
) permission.FieldSensitivity {
	definition := s.threadDefinition(ctx, req)
	if definition == nil {
		return permission.SensitivityPublic
	}

	return definition.DataAccessSensitivity()
}

// threadDefinition is the agent the conversation is with, or nil.
func (s *Service) threadDefinition(
	ctx context.Context,
	req conversationports.GetThreadRequest,
) *agentdefinition.Definition {
	if s.definitions == nil {
		return nil
	}
	thread, err := s.conversations.GetThread(ctx, req)
	if err != nil || thread.AgentDefinitionID.IsNil() {
		return nil
	}
	definition, err := s.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         thread.AgentDefinitionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil
	}

	return definition
}

// ListThreadArtifacts reads one page of a thread's artifacts by lineage, with
// each draft's and plan's status read from the proposal or plan it views, so
// the pane shows a draft as sent the moment the decision ran.
func (s *Service) ListThreadArtifacts(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	opts assistantartifactports.ListArtifactsOptions,
) (*assistantartifactports.AssistantArtifactPage, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if opts.Family != "" && !opts.Family.IsValid() {
		return nil, errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "There is no artifact kind by that name",
		)
	}
	empty := &assistantartifactports.AssistantArtifactPage{
		Results: []assistantartifactports.AssistantArtifact{},
		Counts: assistantartifactports.ArtifactCounts{
			Families: map[assistantartifact.Family]int{},
		},
	}
	if s.artifacts == nil {
		return empty, nil
	}

	page, err := s.artifacts.ListPage(ctx, assistantartifactports.ListArtifactsRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		Limit:      opts.Limit,
		Cursor:     opts.Cursor,
		Query:      opts.Query,
		Family:     opts.Family,
		PinnedOnly: opts.PinnedOnly,
		Summary:    opts.Summary,
	})
	if err != nil {
		return nil, err
	}

	return &assistantartifactports.AssistantArtifactPage{
		Results:    s.presentArtifacts(ctx, req, page.Artifacts),
		Total:      page.Total,
		NextCursor: page.NextCursor,
		Counts:     page.Counts,
	}, nil
}

func (s *Service) ArtifactLineage(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
) ([]assistantartifactports.AssistantArtifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	versions, err := s.artifacts.ListLineage(ctx, assistantartifactports.LineageRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		ID:         artifactID,
	})
	if err != nil {
		return nil, err
	}

	return s.presentArtifacts(ctx, req, versions), nil
}

func (s *Service) ArtifactBySlug(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	slug string,
) ([]assistantartifactports.AssistantArtifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	slug = strings.ToLower(strings.TrimSpace(slug))
	root, err := s.artifacts.FindBySlug(ctx, assistantartifactports.SlugRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		Slug:       slug,
	})
	if err != nil {
		// An old link names an artifact by its id rather than its slug.
		id, parseErr := pulid.Parse(slug)
		if parseErr != nil {
			return nil, err
		}

		return s.ArtifactLineage(ctx, req, id)
	}

	return s.ArtifactLineage(ctx, req, root.ID)
}

// presentArtifacts is what the pane reads: statuses that follow their
// decisions, and the question of the turn each artifact came from.
func (s *Service) presentArtifacts(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifacts []*assistantartifact.Artifact,
) []assistantartifactports.AssistantArtifact {
	s.followDecisions(ctx, req, artifacts)

	messageIDs := make([]pulid.ID, 0, len(artifacts))
	seen := make(map[pulid.ID]bool, len(artifacts))
	for _, artifact := range artifacts {
		if !artifact.MessageID.IsNil() && !seen[artifact.MessageID] {
			seen[artifact.MessageID] = true
			messageIDs = append(messageIDs, artifact.MessageID)
		}
	}
	questions, err := s.artifacts.TurnQuestions(ctx, req.ID, req.TenantInfo, messageIDs)
	if err != nil {
		s.logger.Warn("artifact turns could not be read", zap.Error(err))
	}

	out := make([]assistantartifactports.AssistantArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		shown := artifactpayload.ToAssistantArtifact(artifact)
		shown.Turn = turnLabel(questions[artifact.MessageID])
		out = append(out, shown)
	}

	return out
}

//nolint:gocritic // AssistantService methods take the thread request by value
func (s *Service) RunArtifactReport(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
) (*assistantartifactports.AssistantArtifact, error) {
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
	request := reportquerytools.ReportingRequestFor(actor)
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
		arguments, recorded := shown.Payload[artifactpayload.PayloadReadWith].(map[string]any)
		if !recorded {
			return nil, "", errNotReread
		}
		previewed, err := reportquerytools.ReportOfPreview(ctx, s.reports, actor, arguments)
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
	req *conversationports.GetThreadRequest,
	latest *assistantartifact.Artifact,
	run *report.ReportRun,
	name string,
) (*assistantartifactports.AssistantArtifact, error) {
	result := map[string]any{
		"runId":                          run.ID.String(),
		"reportKey":                      run.CannedKey,
		"reportName":                     name,
		payloadStatus:                    string(run.Status),
		"finished":                       run.Status.IsTerminal(),
		"format":                         string(run.Format),
		artifactpayload.PayloadRowCount:  run.RowCount,
		artifactpayload.PayloadTruncated: run.Truncated,
	}
	if !run.DefinitionID.IsNil() {
		result["definitionId"] = run.DefinitionID.String()
	}
	built := toolartifacts.RunArtifact("", result)
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
	out := artifactpayload.ToAssistantArtifact(saved)

	return &out, nil
}

func (s *Service) listOf(
	ctx context.Context,
	req *conversationports.GetThreadRequest,
	actor *permissionports.RequestActor,
	source *exportSource,
) (*listReader, error) {
	toolName := typeutils.StringOfTrimmed(source.artifact.Payload[artifactpayload.PayloadTool])
	if s.queries == nil || toolName == "" {
		return nil, errNotExportable
	}
	tool, ok := s.queries.Get(toolName)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"The list this table came from is no longer available",
		)
	}
	if err := s.mayRead(ctx, actor, tool.Policy()); err != nil {
		return nil, err
	}

	arguments := maps.Clone(source.arguments)
	delete(arguments, artifactpayload.ArgAfter)
	delete(arguments, artifactpayload.ArgOffset)
	delete(arguments, argLimit)

	return &listReader{
		tool: tool,
		params: agenttoolports.QueryToolParams{
			OrganizationID:    req.TenantInfo.OrgID,
			BusinessUnitID:    req.TenantInfo.BuID,
			Actor:             actor,
			Timezone:          s.timezoneOf(ctx, *req),
			DataAccessCeiling: s.ceilingOf(ctx, *req),
		},
		arguments: arguments,
		entity: typeutils.StringOfTrimmed(
			source.artifact.Payload[artifactpayload.PayloadEntity],
		),
	}, nil
}

//nolint:gocritic // hugeParam: GetThreadRequest is passed by value across AssistantService.
func (s *Service) ArtifactRows(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
	query *assistantartifactports.ArtifactRowsQuery,
) (*assistantartifactports.ArtifactRows, error) {
	source, err := s.exportSourceOf(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	if source.artifact.Kind != assistantartifact.KindTableView {
		return nil, errNotPageable
	}
	actor := permissionports.UserActor(req.TenantInfo)
	list, err := s.listOf(ctx, &req, actor, source)
	if err != nil {
		return nil, err
	}

	columns := artifactpayload.ColumnsFrom(source.artifact.Payload[artifactpayload.PayloadColumns])
	sortable := list.sortFields(columns)
	searchable := list.searchable()
	overrides, err := rowsArguments(query, sortable, searchable)
	if err != nil {
		return nil, err
	}

	page, err := list.read(ctx, overrides, query.IncludeTotal)
	if errors.Is(err, querytool.ErrStaleCursor) {
		return nil, errStalePage
	}
	if err != nil {
		return nil, err
	}

	return &assistantartifactports.ArtifactRows{
		Rows: artifactpayload.ProjectRows(
			columns,
			typeutils.StringOfTrimmed(source.artifact.Payload["recordEntity"]),
			artifactpayload.RecordsOf(artifactpayload.RowsFrom(page[artifactpayload.ResultItems])),
		),
		HasMore:        typeutils.BoolOf(page[artifactpayload.ResultHasMore]),
		NextCursor:     typeutils.StringOfTrimmed(page[artifactpayload.ResultNextCursor]),
		NextOffset:     optionalInt(page[artifactpayload.ResultNextOffset]),
		Total:          optionalInt(page[artifactpayload.ResultTotal]),
		TotalEstimated: typeutils.BoolOf(page[artifactpayload.ResultTotalEstimated]),
		Withheld: withheldColumns(
			columns,
			artifactpayload.StringsOf(page["withheldByAccess"]),
		),
		Searchable: searchable,
		Sortable:   slices.Sorted(maps.Keys(sortable)),
	}, nil
}

// followDecisions overlays each draft's and plan's status from the record
// that carries its decision. A read that fails leaves the stored status,
// which is what the pane showed last time.
func (s *Service) followDecisions(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifacts []*assistantartifact.Artifact,
) {
	var wantProposals, wantPlans bool
	for _, artifact := range artifacts {
		wantProposals = wantProposals || !artifact.ProposalID.IsNil()
		wantPlans = wantPlans ||
			(artifact.Kind == assistantartifact.KindPlan && !artifact.PlanID.IsNil())
	}

	if wantProposals && s.proposals != nil {
		proposals, err := s.proposals.ListByThread(
			ctx,
			agentports.ListAgentProposalsByThreadRequest{
				ThreadID:   req.ID,
				TenantInfo: req.TenantInfo,
			},
		)
		if err != nil {
			s.logger.Warn("artifact statuses could not follow proposals", zap.Error(err))
		} else {
			byID := make(map[pulid.ID]agent.ProposalStatus, len(proposals))
			for _, proposal := range proposals {
				byID[proposal.ID] = proposal.Status
			}
			for _, artifact := range artifacts {
				status, ok := byID[artifact.ProposalID]
				switch {
				case !ok:
				case artifact.Kind == assistantartifact.KindEmailDraft:
					artifact.Status = artifactpayload.DraftStatus(status)
				case artifact.Kind == assistantartifact.KindDecisionRequest:
					artifact.Status = artifactpayload.DecisionStatus(status)
				}
			}
		}
	}

	if wantPlans && s.plans != nil {
		plans, err := s.plans.ListByThread(ctx, agentports.ListAgentPlansByThreadRequest{
			ThreadID:   req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			s.logger.Warn("artifact statuses could not follow plans", zap.Error(err))
		} else {
			byID := make(map[pulid.ID]agentplan.PlanStatus, len(plans))
			for _, plan := range plans {
				byID[plan.ID] = plan.Status
			}
			for _, artifact := range artifacts {
				if status, ok := byID[artifact.PlanID]; ok &&
					artifact.Kind == assistantartifact.KindPlan {
					artifact.Status = artifactpayload.PlanStatus(status)
				}
			}
		}
	}
}

func (s *Service) PinArtifact(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
	pinned bool,
) (*assistantartifactports.AssistantArtifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}

	artifact, err := s.artifacts.SetPinned(ctx, assistantartifactports.SetArtifactPinnedRequest{
		ID:         artifactID,
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		Pinned:     pinned,
	})
	if err != nil {
		return nil, err
	}

	out := artifactpayload.ToAssistantArtifact(artifact)

	return &out, nil
}

func (s *Service) boardDocument(
	ctx context.Context,
	req *conversationports.GetThreadRequest,
	artifactID pulid.ID,
) (*docrender.Document, bool, error) {
	if s.boards == nil || s.artifacts == nil {
		return nil, false, nil
	}
	artifact, err := s.artifacts.GetByID(ctx, assistantartifactports.GetArtifactRequest{
		ID:         artifactID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, true, err
	}
	if artifact.Kind != assistantartifact.KindBoard {
		return nil, false, nil
	}

	card, err := s.boards.Card(ctx, &assistantports.BoardCardRequest{
		Thread:     *req,
		ArtifactID: artifactID,
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    req.UserID,
			UserID:         req.UserID,
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, true, err
	}
	_, body, _ := strings.Cut(card.Markdown, "\n")

	return &docrender.Document{
		Title:  card.Title,
		Kicker: "Board",
		Byline: strings.Join(stringutils.NonEmptyStrings(
			card.Author,
			"answered "+time.Unix(card.AnsweredAt, 0).UTC().Format("Jan 2, 2006"),
		), " · "),
		Body: strings.TrimSpace(body),
	}, true, nil
}

// ExportDocument renders the version of a document a person is reading as a
// PDF, printed by the same renderer as invoices, or as a Word file.
func (s *Service) ExportDocument(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
	format string,
) (*assistantartifactports.ArtifactFile, error) {
	if doc, isBoard, err := s.boardDocument(ctx, &req, artifactID); isBoard {
		if err != nil {
			return nil, err
		}
		return s.renderDocument(ctx, doc, format)
	}
	versions, err := s.documentLineage(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	version := latestOf(versions)
	for _, candidate := range versions {
		if candidate.ID == artifactID {
			version = candidate
		}
	}
	return s.renderDocument(ctx, renderableDocument(version), format)
}

func (s *Service) renderDocument(
	ctx context.Context,
	doc *docrender.Document,
	format string,
) (*assistantartifactports.ArtifactFile, error) {
	name := assistantartifact.SlugBase(doc.Title)

	switch format {
	case DocumentExportPDF:
		if s.pdfs == nil {
			return nil, errortypes.NewBusinessError("PDF export is not available here")
		}
		body, renderErr := s.pdfs.Render(ctx, &documenttemplateports.PDFRenderRequest{
			HTML:  docrender.HTML(doc),
			Title: doc.Title,
		})
		if renderErr != nil {
			if errors.Is(renderErr, documenttemplateports.ErrPDFRendererUnavailable) {
				return nil, errortypes.NewBusinessError("PDF export is not available right now")
			}
			return nil, renderErr
		}

		return &assistantartifactports.ArtifactFile{
			FileName: name + ".pdf", ContentType: "application/pdf", Body: body,
		}, nil
	case DocumentExportDOCX:
		body, renderErr := docrender.DOCX(doc)
		if renderErr != nil {
			return nil, renderErr
		}

		return &assistantartifactports.ArtifactFile{
			FileName: name + ".docx", ContentType: docxContentType, Body: body,
		}, nil
	default:
		return nil, errortypes.NewValidationError(
			"format", errortypes.ErrInvalid, "A document downloads as pdf or docx",
		)
	}
}

// documentLineage reads a document's versions by any of their ids, checking
// the conversation is the person's and the artifact a document in it.
func (s *Service) documentLineage(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
) ([]*assistantartifact.Artifact, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	versions, err := s.artifacts.ListLineage(ctx, assistantartifactports.LineageRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		ID:         artifactID,
	})
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 || versions[0].Kind != assistantartifact.KindDocument {
		return nil, errortypes.NewValidationError(
			"artifact", errortypes.ErrInvalid, "Only a document has versions to edit",
		)
	}

	return versions, nil
}

// appendDocumentVersion keeps a person's text as the document's next version:
// the same title, sources and link, credited to them, with a note of what
// changed. Earlier versions stay as they were.
func (s *Service) appendDocumentVersion(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	versions []*assistantartifact.Artifact,
	body, note string,
) (*assistantartifactports.AssistantArtifact, error) {
	body = strings.TrimSpace(body)
	switch {
	case body == "":
		return nil, errortypes.NewValidationError(
			"body", errortypes.ErrRequired, "A document cannot be saved empty",
		)
	case len(body) > assistantartifact.MaxDocumentBodyBytes:
		return nil, errortypes.NewValidationError(
			"body", errortypes.ErrInvalid, "This document is too long to save",
		)
	}
	note = strings.TrimSpace(note)
	if runes := []rune(note); len(runes) > maxVersionNote {
		note = string(runes[:maxVersionNote])
	}

	latest := latestOf(versions)
	next := &assistantartifact.Artifact{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ThreadID:       req.ID,
		MessageID:      latest.MessageID,
		RunID:          latest.RunID,
		Kind:           assistantartifact.KindDocument,
		Status:         assistantartifact.StatusReady,
		Title:          latest.Title,
		Slug:           latest.Slug,
		Pinned:         latest.Pinned,
		Payload: assistantartifact.DocumentVersion(
			latest.Payload, body, assistantartifact.DocumentEditedByPerson,
			s.attribution().PersonName(ctx, &req.UserID, req.TenantInfo), note,
		),
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
	out := artifactpayload.ToAssistantArtifact(saved)

	return &out, nil
}

// SaveDocumentVersion keeps a person's edit of a document as its next version.
func (s *Service) SaveDocumentVersion(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
	version assistantartifactports.SaveDocumentVersionRequest,
) (*assistantartifactports.AssistantArtifact, error) {
	versions, err := s.documentLineage(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	note := version.Note
	if strings.TrimSpace(note) == "" {
		note = "Your edits"
	}

	return s.appendDocumentVersion(ctx, req, versions, version.Body, note)
}

// RestoreDocumentVersion makes an earlier version the latest again, as a new
// version, so nothing written since is lost.
func (s *Service) RestoreDocumentVersion(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
) (*assistantartifactports.AssistantArtifact, error) {
	versions, err := s.documentLineage(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	var restored *assistantartifact.Artifact
	for _, version := range versions {
		if version.ID == artifactID {
			restored = version
		}
	}
	if restored == nil {
		return nil, errortypes.NewNotFoundError("Version not found")
	}

	return s.appendDocumentVersion(
		ctx, req, versions,
		typeutils.StringOfTrimmed(restored.Payload[assistantartifact.DocumentBody]),
		fmt.Sprintf("Restored v%d", max(restored.LineageSeq, 1)),
	)
}

// RewriteDocument suggests a new wording for a passage of a document. Nothing
// is saved: the person reads the suggestion against the original and accepts
// it as a new version or keeps what was there.
func (s *Service) RewriteDocument(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	artifactID pulid.ID,
	rewrite assistantartifactports.DocumentRewriteRequest,
) (*assistantartifactports.DocumentRewriteSuggestion, error) {
	versions, err := s.documentLineage(ctx, req, artifactID)
	if err != nil {
		return nil, err
	}
	instruction, err := rewriteInstruction(rewrite)
	if err != nil {
		return nil, err
	}
	latest := latestOf(versions)
	body := typeutils.StringOfTrimmed(latest.Payload[assistantartifact.DocumentBody])
	passage := strings.TrimSpace(rewrite.Text)
	if !strings.Contains(body, passage) {
		return nil, errortypes.NewValidationError(
			"text", errortypes.ErrInvalid, "That passage is not in the latest version",
		)
	}
	if s.completion == nil {
		return nil, errortypes.NewBusinessError("Rewriting is not available here")
	}

	thread, err := s.conversations.GetThread(ctx, req)
	if err != nil {
		return nil, err
	}
	definition := s.threadDefinition(ctx, req)
	attribution := aiusageports.AIUsageAttribution{
		UserID:   req.UserID,
		ThreadID: req.ID,
		Feature:  aiusage.FeatureAgentTurn,
	}
	preferred := thread.PreferredProviderID
	if definition != nil {
		attribution.AgentDefinitionID = definition.ID
		version := definition.Version
		attribution.DefinitionVersion = &version
		if preferred.IsNil() {
			preferred = definition.PreferredProviderID
		}
	}

	result, err := s.completion.CompleteStructured(ctx, &llmports.StructuredCompletionRequest{
		TenantInfo: req.TenantInfo,
		Task:       aitask.TaskAssistantChat,
		System:     rewriteSystem,
		Context: llmports.DelimitedContext{Sections: []llmports.ContextSection{
			{Title: "Document title", Trusted: true, Content: latest.Title},
			{Title: "Passage to rewrite", Content: passage},
			{
				Title:   "How to rewrite it",
				Trusted: rewrite.Mode != assistantartifactports.DocumentRewriteAsk,
				Content: instruction,
			},
		}},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{"type": "string"},
			},
			"required":             []string{"text"},
			"additionalProperties": false,
		},
		SchemaName:          "document_rewrite",
		MaxTokens:           rewriteMaxTokens,
		PreferredProviderID: preferred,
		Attribution:         attribution,
	})
	if err != nil {
		s.logger.Warn("document rewrite failed", zap.Error(err))
		return nil, errortypes.NewBusinessError("The rewrite could not be written. Try again.")
	}

	var out assistantartifactports.DocumentRewriteSuggestion
	if err = sonic.UnmarshalString(result.Text, &out); err != nil ||
		strings.TrimSpace(out.Text) == "" {
		return nil, errortypes.NewBusinessError("The rewrite came back empty. Try again.")
	}
	out.Text = strings.TrimSpace(out.Text)

	return &out, nil
}

type Service struct {
	logger        *zap.Logger
	conversations conversationports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	artifacts     assistantartifactports.AssistantArtifactRepository
	permissions   permissionports.PermissionEngine
	activity      agentports.AgentActivityPublisher
	proposals     proposalstores.ProposalStore
	plans         proposalstores.PlanStore
	queries       agenttoolports.AgentQueryToolRegistry
	reports       *reporting.Service
	reportRuns    reportRerunner
	boards        assistantports.BoardFiller
	pdfs          documenttemplateports.PDFRenderer
	completion    llmports.StructuredCompleter
	runs          agentruntimeports.AgentRunRepository
	users         tenantports.UserRepository
}

func (s *Service) attribution() *attribution.Attributor {
	return &attribution.Attributor{
		Logger:      s.logger,
		Definitions: s.definitions,
		Runs:        s.runs,
		Users:       s.users,
	}
}
