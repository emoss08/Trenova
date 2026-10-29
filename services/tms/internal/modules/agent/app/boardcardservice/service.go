package boardcardservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardmarkdown"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Permissions   permissionports.PermissionEngine
	Summaries     agentports.RecordSummarizer
	Artifacts     assistantartifactports.AssistantArtifactRepository
	Conversations conversationports.ConversationRepository
	Analytics     kernelports.AnalyticsService
	Receivables   customerledgerports.ARSnapshotReader
	Audit         auditports.AuditService
	Organizations tenantports.OrganizationRepository
}

type permissionChecker interface {
	Check(
		ctx context.Context,
		req *permissionports.PermissionCheckRequest,
	) (*permissionports.PermissionCheckResult, error)
}

type artifactReader interface {
	GetByID(
		ctx context.Context,
		req assistantartifactports.GetArtifactRequest,
	) (*assistantartifact.Artifact, error)
	TurnQuestions(
		ctx context.Context,
		threadID pulid.ID,
		tenant pagination.TenantInfo,
		messageIDs []pulid.ID,
	) (map[pulid.ID]string, error)
}

type threadReader interface {
	GetThread(
		ctx context.Context,
		req conversationports.GetThreadRequest,
	) (*conversation.Thread, error)
}

type analyticsReader interface {
	GetAnalytics(
		ctx context.Context,
		opts *kernelports.AnalyticsRequestOptions,
	) (kernelports.AnalyticsData, error)
}

type receivablesReader interface {
	GetDashboardKPIs(
		ctx context.Context,
		req customerledgerports.GetARAnalyticsRequest,
	) (*customerledgerports.ARDashboardKPIs, error)
}

type activityReader interface {
	ListByResourceID(
		ctx context.Context,
		req *auditports.ListByResourceIDRequest,
	) (*pagination.ListResult[*audit.Entry], error)
}

type Service struct {
	l             *zap.Logger
	permissions   permissionChecker
	summaries     agentports.RecordSummarizer
	artifacts     artifactReader
	threads       threadReader
	analytics     analyticsReader
	receivables   receivablesReader
	activity      activityReader
	organizations orgzone.OrganizationReader
	now           func() time.Time
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:             p.Logger.Named("service.board-card"),
		permissions:   p.Permissions,
		summaries:     p.Summaries,
		artifacts:     p.Artifacts,
		threads:       p.Conversations,
		analytics:     p.Analytics,
		receivables:   p.Receivables,
		activity:      p.Audit,
		organizations: p.Organizations,
		now:           time.Now,
	}
}

var errNotABoard = errortypes.NewNotFoundError("Board not found")

func (s *Service) Card(
	ctx context.Context,
	req *assistantports.BoardCardRequest,
) (*assistantports.BoardCard, error) {
	if _, err := s.threads.GetThread(ctx, req.Thread); err != nil {
		return nil, err
	}
	artifact, err := s.artifacts.GetByID(ctx, assistantartifactports.GetArtifactRequest{
		ID:         req.ArtifactID,
		TenantInfo: req.Thread.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if artifact.ThreadID != req.Thread.ID || artifact.Kind != assistantartifact.KindBoard {
		return nil, errNotABoard
	}
	payload, err := board.PayloadOf(artifact.Payload)
	if err != nil {
		return nil, fmt.Errorf("read the board's payload: %w", err)
	}

	filled, err := s.Fill(ctx, &assistantports.BoardFillRequest{
		Actor:    req.Actor,
		ThreadID: artifact.ThreadID,
		Board:    &payload.Board,
	})
	if err != nil {
		return nil, err
	}
	filled.Compare(&payload.Snapshot)

	card := &assistantports.BoardCard{
		ArtifactID: artifact.ID,
		ThreadID:   artifact.ThreadID,
		Title:      artifact.Title,
		Author:     payload.Author,
		AnsweredAt: artifact.CreatedAt,
		Filled:     filled,
	}
	if artifact.MessageID.IsNotNil() {
		questions, qErr := s.artifacts.TurnQuestions(
			ctx, artifact.ThreadID, req.Thread.TenantInfo, []pulid.ID{artifact.MessageID},
		)
		if qErr != nil {
			s.l.Warn("the board's question could not be read", zap.Error(qErr))
		}
		card.Question = questions[artifact.MessageID]
	}
	card.Markdown = boardmarkdown.Markdown(filled, boardmarkdown.MarkdownMeta{
		Question:   card.Question,
		Author:     card.Author,
		AnsweredAt: card.AnsweredAt,
		Location:   s.Location(ctx, req.Thread.TenantInfo),
	})

	return card, nil
}

func (s *Service) Location(ctx context.Context, tenant pagination.TenantInfo) *time.Location {
	location, err := orgzone.Location(ctx, s.organizations, tenant)
	if err != nil {
		s.l.Warn("the organization's time zone could not be read", zap.Error(err))
		return time.UTC
	}

	return location
}

func (s *Service) Fill(
	ctx context.Context,
	req *assistantports.BoardFillRequest,
) (*board.Filled, error) {
	now := s.now().Unix()
	tenant := req.Actor.TenantInfo()
	reads := &boardassemble.Reads{
		Now:      now,
		Readable: make(map[permission.Resource]bool, 4),
		Metrics:  map[string]*board.Value{},
		Analyses: map[pulid.ID]*boardassemble.AnalysisRead{},
		Tables:   map[pulid.ID]*boardassemble.TableRead{},
		Events:   map[pulid.ID][]board.Event{},
	}

	refs := req.Board.Records()
	readable := make(map[permission.Resource][]pulid.ID, len(refs))
	for resource, ids := range refs {
		allowed, err := s.allowed(ctx, req.Actor, resource, permission.OpRead)
		if err != nil {
			return nil, err
		}
		reads.Readable[resource] = allowed
		if allowed {
			readable[resource] = ids
		}
	}
	summaries, err := s.summaries.Summaries(ctx, &agentports.RecordSummaryRequest{
		Tenant: tenant,
		Refs:   readable,
		Now:    now,
	})
	if err != nil {
		return nil, err
	}
	reads.Summaries = summaries

	if err = s.readMetrics(ctx, req.Actor, req.Board.Metrics(), reads); err != nil {
		return nil, err
	}
	s.readArtifacts(ctx, req, reads)
	if err = s.readEvents(ctx, req, reads); err != nil {
		return nil, err
	}

	return boardassemble.Assemble(req.Board, reads), nil
}

func (s *Service) allowed(
	ctx context.Context,
	actor *permissionports.RequestActor,
	resource permission.Resource,
	operation permission.Operation,
) (bool, error) {
	result, err := s.permissions.Check(ctx, actor.PermissionCheck(resource, operation))
	if err != nil {
		return false, fmt.Errorf("check %s %s: %w", resource, operation, err)
	}

	return result != nil && result.Allowed, nil
}

func mistake(format string, args ...any) error {
	return &agentruntimeports.PublishMistake{Reason: fmt.Sprintf(format, args...)}
}

func (s *Service) Check(ctx context.Context, req *assistantports.BoardFillRequest) error {
	for resource := range req.Board.Records() {
		allowed, err := s.allowed(ctx, req.Actor, resource, permission.OpRead)
		if err != nil {
			return err
		}
		if !allowed {
			return mistake("the person may not read %s, so leave the blocks that name them out",
				inflect.PluralNoun(resource.Noun()))
		}
	}
	for _, key := range req.Board.Metrics() {
		allowed, err := s.metricAllowed(ctx, req.Actor, key)
		if err != nil {
			return err
		}
		if !allowed {
			return mistake("the person may not read the %s metric, so leave it out", key)
		}
	}
	for idx := range req.Board.Blocks {
		if req.Board.Blocks[idx].Type != board.BlockTimeline {
			continue
		}
		allowed, err := s.allowed(ctx, req.Actor, permission.ResourceAuditLog, permission.OpRead)
		if err != nil {
			return err
		}
		if !allowed {
			return mistake("the person may not read a record's history, so leave the " +
				"timeline out")
		}
		break
	}

	return s.checkArtifacts(ctx, req)
}

var errArtifactGone = errors.New("artifact not in this conversation")
