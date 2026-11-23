package desktranscriptservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/desktranscriptexport"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationtranscript"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const maxOpenedMessages = 10_000

type Service struct {
	conversations desktranscriptports.ConversationRepository
	exportRepo    desktranscriptports.ExportRepository
	assembler     *desktranscriptexport.Assembler
	exports       *desktranscriptexport.Exports
	organizations tenantports.OrganizationReader
	permissions   permissionports.PermissionEngine
	auditor       permissionports.SecurityAuditor
	storage       kernelports.Client
	workflows     kernelports.WorkflowStarter
	now           func() time.Time
	l             *zap.Logger
}

type ServiceParams struct {
	Conversations desktranscriptports.ConversationRepository
	ExportRepo    desktranscriptports.ExportRepository
	Assembler     *desktranscriptexport.Assembler
	Exports       *desktranscriptexport.Exports
	Organizations tenantports.OrganizationReader
	Permissions   permissionports.PermissionEngine
	Auditor       permissionports.SecurityAuditor
	Storage       kernelports.Client
	Workflows     kernelports.WorkflowStarter
	Now           func() time.Time
	Logger        *zap.Logger
}

func NewService(p *ServiceParams) *Service {
	now := p.Now
	if now == nil {
		now = time.Now
	}

	return &Service{
		conversations: p.Conversations,
		exportRepo:    p.ExportRepo,
		assembler:     p.Assembler,
		exports:       p.Exports,
		organizations: p.Organizations,
		permissions:   p.Permissions,
		auditor:       p.Auditor,
		storage:       p.Storage,
		workflows:     p.Workflows,
		now:           now,
		l:             p.Logger.Named("service.desk-transcripts"),
	}
}

func errNotAPerson() error {
	return errortypes.NewAuthorizationError(
		"Only a person signed in to this organization can read other people's Desk conversations",
	)
}

func (s *Service) guard(
	ctx context.Context,
	actor *permissionports.RequestActor,
	op permission.Operation,
) (pagination.TenantInfo, error) {
	if actor == nil || !actor.IsUser() || actor.IsSupport() || actor.PersonUserID().IsNil() ||
		actor.OrganizationID.IsNil() || actor.BusinessUnitID.IsNil() {
		return pagination.TenantInfo{}, errNotAPerson()
	}

	result, err := s.permissions.Check(
		ctx,
		actor.PermissionCheck(permission.ResourceDeskTranscript, op),
	)
	if err != nil {
		return pagination.TenantInfo{}, err
	}
	if result == nil || !result.Allowed {
		message := "You do not have permission to read other people's Desk conversations"
		if op == permission.OpExport {
			message = "You do not have permission to take Desk conversations out of Trenova"
		}

		return pagination.TenantInfo{}, permissionports.DeniedError(
			result,
			errortypes.NewAuthorizationError(message),
		)
	}

	return pagination.TenantInfo{OrgID: actor.OrganizationID, BuID: actor.BusinessUnitID}, nil
}

func (s *Service) window(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	filter *desktranscript.ConversationFilter,
) (desktranscript.Window, error) {
	filter.Normalize()
	multiErr := errortypes.NewMultiError()
	filter.Validate("filter", multiErr)
	if multiErr.HasErrors() {
		return desktranscript.Window{}, multiErr
	}

	loc, err := tenantports.OrganizationLocation(ctx, s.organizations, tenantInfo)
	if err != nil {
		return desktranscript.Window{}, err
	}

	return filter.Timeframe.Resolve(s.now(), loc), nil
}

func (s *Service) ListConversations(
	ctx context.Context,
	req *desktranscriptports.ListRequest,
) (*pagination.CursorListResult[*desktranscript.Conversation], error) {
	tenantInfo, err := s.guard(ctx, req.Actor, permission.OpRead)
	if err != nil {
		return nil, err
	}
	filter := req.Filter
	window, err := s.window(ctx, tenantInfo, &filter)
	if err != nil {
		return nil, err
	}

	page, err := s.conversations.List(ctx, &desktranscriptports.ListConversationsRequest{
		TenantInfo: tenantInfo,
		Filter:     filter,
		Window:     window,
		Query:      req.Query,
		Cursor:     req.Cursor,
	})
	if err != nil {
		return nil, err
	}

	items, err := s.assembler.Describe(ctx, tenantInfo, page.Items)
	if err != nil {
		return nil, err
	}

	return &pagination.CursorListResult[*desktranscript.Conversation]{
		Items:           items,
		HasNextPage:     page.HasNextPage,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalExact,
		EdgeCursors:     page.EdgeCursors,
	}, nil
}

func (s *Service) conversation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*conversation.Thread, error) {
	if id.IsNil() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrRequired,
			"Choose a conversation",
		)
	}

	return s.conversations.Get(ctx, desktranscriptports.GetConversationRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (s *Service) OpenConversation(
	ctx context.Context,
	req desktranscriptports.ConversationRequest,
) (*desktranscriptports.OpenedConversation, error) {
	tenantInfo, err := s.guard(ctx, req.Actor, permission.OpRead)
	if err != nil {
		return nil, err
	}
	thread, err := s.conversation(ctx, tenantInfo, req.ID)
	if err != nil {
		return nil, err
	}

	described, err := s.assembler.Describe(ctx, tenantInfo, []*conversation.Thread{thread})
	if err != nil {
		return nil, err
	}
	messages, err := s.conversations.ListMessages(ctx, &desktranscriptports.ThreadMessagesRequest{
		TenantInfo: tenantInfo,
		ThreadIDs:  []pulid.ID{thread.ID},
		Limit:      maxOpenedMessages,
	})
	if err != nil {
		return nil, err
	}

	opened := &desktranscriptports.OpenedConversation{
		Conversation: described[0],
		Transcript:   conversationtranscript.RunTranscriptOf(messages),
	}
	s.recordLook(ctx, req.Actor, subjectOf(described[0]), lookOpened)

	return opened, nil
}

func (s *Service) DownloadConversation(
	ctx context.Context,
	req desktranscriptports.ConversationRequest,
) (*kernelports.TranscriptFile, error) {
	tenantInfo, err := s.guard(ctx, req.Actor, permission.OpExport)
	if err != nil {
		return nil, err
	}
	thread, err := s.conversation(ctx, tenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	loc, err := tenantports.OrganizationLocation(ctx, s.organizations, tenantInfo)
	if err != nil {
		return nil, err
	}
	var file *kernelports.TranscriptFile
	var subject lookSubject
	if err = s.assembler.Render(ctx, &desktranscriptexport.RenderBatch{
		TenantInfo: tenantInfo,
		Threads:    []*conversation.Thread{thread},
		Location:   loc,
		ExportedAt: s.now().Unix(),
	}, func(_ *conversation.Thread, rendered *desktranscriptports.RenderedConversation) error {
		file = &kernelports.TranscriptFile{FileName: rendered.FileName, Body: rendered.Markdown}
		subject = lookSubject{
			ThreadID:    thread.ID,
			OwnerUserID: thread.UserID,
			OwnerName:   rendered.Record.Owner.Name,
			AgentName:   rendered.Record.Agent,
			Title:       thread.Title,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if file == nil {
		return nil, errortypes.NewNotFoundError("Conversation not found")
	}

	s.recordLook(ctx, req.Actor, subject, lookDownloaded)

	return file, nil
}

func (s *Service) PreviewExport(
	ctx context.Context,
	req *desktranscriptports.PreviewRequest,
) (*desktranscriptports.ExportPreview, error) {
	tenantInfo, err := s.guard(ctx, req.Actor, permission.OpExport)
	if err != nil {
		return nil, err
	}
	filter := req.Filter
	window, err := s.window(ctx, tenantInfo, &filter)
	if err != nil {
		return nil, err
	}

	n, err := s.conversations.CountUpTo(ctx, &desktranscriptports.MatchConversationsRequest{
		TenantInfo: tenantInfo,
		Filter:     filter,
		Window:     window,
		Limit:      desktranscript.MaxExportConversations + 1,
	})
	if err != nil {
		return nil, err
	}

	return &desktranscriptports.ExportPreview{
		Count:       n,
		Exact:       n <= desktranscript.MaxExportConversations,
		Limit:       desktranscript.MaxExportConversations,
		WithinLimit: n <= desktranscript.MaxExportConversations,
	}, nil
}
