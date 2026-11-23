package desktranscriptservice

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	downloadURLExpiry = 60 * time.Second
	workflowIDPrefix  = "desk-transcript-export/"
)

func errBothShapes() error {
	return errortypes.NewValidationError(
		"conversationIds",
		errortypes.ErrInvalid,
		"Export either a timeframe or the conversations you selected, not both",
	)
}

func errNoShape() error {
	return errortypes.NewValidationError(
		"filter",
		errortypes.ErrRequired,
		"Choose a timeframe or select conversations to export",
	)
}

func errNothingMatches() error {
	return errortypes.NewValidationError(
		"filter",
		errortypes.ErrInvalid,
		"No conversation matches. Choose a longer timeframe, or another person or agent.",
	)
}

func errTooMany() error {
	return errortypes.NewValidationError(
		"filter",
		errortypes.ErrInvalid,
		"More than {0} conversations match, and one export can hold at most {0}. Choose a shorter timeframe, one person or one agent.",
		desktranscript.MaxExportConversations,
	)
}

type exportPlan struct {
	refs     []desktranscriptports.ConversationRef
	coverage *desktranscript.Coverage
	filter   *desktranscript.ConversationFilter
}

func (s *Service) RequestExport(
	ctx context.Context,
	req *desktranscriptports.RequestExportRequest,
) (*desktranscript.Export, error) {
	tenantInfo, err := s.guard(ctx, req.Actor, permission.OpExport)
	if err != nil {
		return nil, err
	}

	var plan *exportPlan
	switch {
	case req.Filter != nil && len(req.ConversationIDs) > 0:
		return nil, errBothShapes()
	case req.Filter != nil:
		plan, err = s.planFilter(ctx, tenantInfo, req.Filter)
	case len(req.ConversationIDs) > 0:
		plan, err = s.planSelection(ctx, tenantInfo, req.ConversationIDs)
	default:
		return nil, errNoShape()
	}
	if err != nil {
		return nil, err
	}

	if err = s.nameOwners(ctx, tenantInfo, plan); err != nil {
		return nil, err
	}

	ids := make([]pulid.ID, len(plan.refs))
	for idx, ref := range plan.refs {
		ids[idx] = ref.ID
	}
	export := &desktranscript.Export{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		RequestedByUserID: req.Actor.UserID,
		Kind:              plan.coverage.Kind,
		Filter:            plan.filter,
		ThreadIDs:         ids,
		Coverage:          plan.coverage,
		ConversationCount: len(ids),
		Status:            desktranscript.ExportStatusPending,
	}
	multiErr := errortypes.NewMultiError()
	export.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.exportRepo.Create(ctx, export)
	if err != nil {
		return nil, err
	}
	s.recordExport(ctx, req.Actor, created, actionExportRequested)

	return s.start(ctx, created)
}

func (s *Service) planFilter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	requested *desktranscript.ConversationFilter,
) (*exportPlan, error) {
	filter := *requested
	window, err := s.window(ctx, tenantInfo, &filter)
	if err != nil {
		return nil, err
	}

	refs, err := s.conversations.Match(ctx, &desktranscriptports.MatchConversationsRequest{
		TenantInfo: tenantInfo,
		Filter:     filter,
		Window:     window,
		Limit:      desktranscript.MaxExportConversations + 1,
	})
	if err != nil {
		return nil, err
	}
	if len(refs) > desktranscript.MaxExportConversations {
		return nil, errTooMany()
	}
	if len(refs) == 0 {
		return nil, errNothingMatches()
	}

	timeframe := filter.Timeframe
	coverage := &desktranscript.Coverage{
		Kind:      desktranscript.ExportKindFilter,
		Timeframe: &timeframe,
		Window:    window,
		Origins:   filter.Origins,
	}
	if filter.AgentDefinitionID.IsNotNil() {
		names, nameErr := s.assembler.AgentNames(
			ctx,
			tenantInfo,
			[]pulid.ID{filter.AgentDefinitionID},
		)
		if nameErr != nil {
			return nil, nameErr
		}
		coverage.AgentName = names[filter.AgentDefinitionID]
	}

	return &exportPlan{refs: refs, coverage: coverage, filter: &filter}, nil
}

func (s *Service) planSelection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	requested []pulid.ID,
) (*exportPlan, error) {
	ids := sliceutils.Dedupe(requested)
	if len(ids) > desktranscript.MaxSelectedConversations {
		return nil, errortypes.NewValidationError(
			"conversationIds",
			errortypes.ErrInvalid,
			"Select at most {0} conversations to download at once, or export a timeframe instead",
			desktranscript.MaxSelectedConversations,
		)
	}
	if slices.ContainsFunc(ids, pulid.ID.IsNil) {
		return nil, errortypes.NewValidationError(
			"conversationIds",
			errortypes.ErrInvalid,
			"A selected conversation is not one Trenova knows",
		)
	}

	threads, err := s.conversations.ListByIDs(ctx, desktranscriptports.ConversationsByIDsRequest{
		TenantInfo: tenantInfo,
		IDs:        ids,
	})
	if err != nil {
		return nil, err
	}
	if len(threads) != len(ids) {
		return nil, errortypes.NewNotFoundError(
			"{0} of the selected conversations could not be found in your organization",
			len(ids)-len(threads),
		)
	}

	byID := make(map[pulid.ID]desktranscriptports.ConversationRef, len(threads))
	for _, thread := range threads {
		byID[thread.ID] = desktranscriptports.ConversationRef{ID: thread.ID, UserID: thread.UserID}
	}
	refs := make([]desktranscriptports.ConversationRef, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, byID[id])
	}

	return &exportPlan{
		refs: refs,
		coverage: &desktranscript.Coverage{
			Kind:          desktranscript.ExportKindSelection,
			SelectedCount: len(refs),
		},
	}, nil
}

func (s *Service) nameOwners(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	plan *exportPlan,
) error {
	userIDs := make([]pulid.ID, 0, len(plan.refs)+1)
	for _, ref := range plan.refs {
		userIDs = append(userIDs, ref.UserID)
	}
	userIDs = sliceutils.Dedupe(userIDs)
	if plan.filter != nil && plan.filter.UserID.IsNotNil() &&
		!slices.Contains(userIDs, plan.filter.UserID) {
		userIDs = append(userIDs, plan.filter.UserID)
	}

	owners, err := s.assembler.Owners(ctx, tenantInfo, userIDs)
	if err != nil {
		return err
	}

	owned := make(map[pulid.ID]struct{}, len(plan.refs))
	named := make(map[string]struct{}, len(owners))
	names := make([]string, 0, len(owners))
	for _, ref := range plan.refs {
		owned[ref.UserID] = struct{}{}
		owner, ok := owners[ref.UserID]
		if _, seen := named[owner.Name]; !ok || owner.Name == "" || seen {
			continue
		}
		named[owner.Name] = struct{}{}
		names = append(names, owner.Name)
	}
	slices.Sort(names)
	plan.coverage.OwnerCount = len(owned)
	plan.coverage.OwnerNames = names[:min(len(names), desktranscript.MaxOwnersNamed)]
	if plan.filter != nil && plan.filter.UserID.IsNotNil() {
		plan.coverage.PersonName = owners[plan.filter.UserID].Name
	}

	return nil
}

func (s *Service) start(
	ctx context.Context,
	export *desktranscript.Export,
) (*desktranscript.Export, error) {
	workflowID := workflowIDPrefix + export.ID.String()
	if _, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                    workflowID,
			TaskQueue:             temporaltype.ReportTaskQueue,
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		},
		desktranscriptports.ExportWorkflowName,
		&desktranscriptports.ExportPayload{
			ExportID:       export.ID,
			OrganizationID: export.OrganizationID,
			BusinessUnitID: export.BusinessUnitID,
		},
	); err != nil {
		s.l.Error("failed to start a desk transcript export", zap.Error(err))
		if _, failErr := s.exports.Fail(
			ctx,
			export,
			i18n.T(ctx, "The export could not be queued. Export again."),
		); failErr != nil {
			s.l.Error(
				"failed to mark an unqueued desk transcript export failed",
				zap.Error(failErr),
			)
		}

		return nil, errortypes.NewBusinessError(
			"The export could not be queued. Try again shortly.",
		)
	}

	export.Status = desktranscript.ExportStatusRunning
	export.WorkflowID = workflowID
	started := s.now().Unix()
	export.StartedAt = &started

	return s.exportRepo.Update(ctx, export)
}

func (s *Service) ListExports(
	ctx context.Context,
	req *desktranscriptports.ListExportsServiceRequest,
) (*pagination.CursorListResult[*desktranscript.Export], error) {
	tenantInfo, err := s.guard(ctx, req.Actor, permission.OpRead)
	if err != nil {
		return nil, err
	}
	filter := req.Filter
	if filter == nil {
		filter = &pagination.QueryOptions{}
	}
	filter.TenantInfo = tenantInfo

	return s.exportRepo.ListConnection(ctx, &desktranscriptports.ListExportsRequest{
		Filter: filter,
		Cursor: req.Cursor,
	})
}

func exportFileName(export *desktranscript.Export) string {
	return fmt.Sprintf(
		"desk-conversations-%s.zip",
		time.Unix(export.CreatedAt, 0).UTC().Format("2006-01-02-150405"),
	)
}

func (s *Service) ExportDownload(
	ctx context.Context,
	req desktranscriptports.ExportDownloadRequest,
) (*desktranscriptports.ExportDownload, error) {
	tenantInfo, err := s.guard(ctx, req.Actor, permission.OpExport)
	if err != nil {
		return nil, err
	}

	export, err := s.exportRepo.GetByID(ctx, desktranscriptports.GetExportRequest{
		TenantInfo: tenantInfo,
		ID:         req.ExportID,
	})
	if err != nil {
		return nil, err
	}
	if export.RequestedByUserID != req.Actor.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the person who asked for an export may download it",
		)
	}

	now := s.now()
	if export.Status == desktranscript.ExportStatusExpired ||
		(export.ArtifactExpiresAt != nil && *export.ArtifactExpiresAt <= now.Unix()) {
		return nil, errortypes.NewBusinessError(
			"This export has expired. Export the conversations again.",
		)
	}
	if !export.Downloadable(now.Unix()) {
		return nil, errortypes.NewBusinessError("This export has no file to download yet")
	}

	fileName := exportFileName(export)
	url, err := s.storage.GetPresignedURL(ctx, &kernelports.PresignedURLParams{
		Key:                export.ArtifactKey,
		Expiry:             downloadURLExpiry,
		ContentDisposition: fmt.Sprintf("attachment; filename=%q", fileName),
	})
	if err != nil {
		s.l.Error("failed to presign a desk transcript export",
			zap.String("exportId", export.ID.String()), zap.Error(err))

		return nil, errortypes.NewBusinessError(
			"The export could not be prepared for download. Try again shortly.",
		)
	}

	s.recordExport(ctx, req.Actor, export, actionExportDownloaded)

	return &desktranscriptports.ExportDownload{
		URL:       url,
		FileName:  fileName,
		ExpiresAt: now.Add(downloadURLExpiry).Unix(),
	}, nil
}

var _ desktranscriptports.Service = (*Service)(nil)
