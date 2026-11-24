package boardcardservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/planwording"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardmarkdown"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Permissions   permissionports.PermissionEngine
	Fields        permissionports.FieldAccess
	Readers       *boardreaderports.Index
	Artifacts     assistantartifactports.AssistantArtifactRepository
	Conversations conversationports.ConversationRepository
	Audit         auditports.AuditService
	Organizations tenantports.OrganizationRepository
	Attention     deskattentionports.DeskAttentionService
	// Plan withholds a figure whose feature the plan leaves out: the costing
	// figures are Accounting & Cash.
	Plan   permissionports.PlanFeatures
	Scopes permissionports.ReadScopes
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
		req *conversationports.GetThreadRequest,
	) (*conversation.Thread, error)
}

type activityReader interface {
	ListConnection(
		ctx context.Context,
		req *auditports.ListAuditEntriesRequest,
	) (*pagination.CursorListResult[*audit.Entry], error)
}

type Service struct {
	l             *zap.Logger
	permissions   permissionChecker
	engine        permissionports.PermissionEngine
	registry      *permissioncatalog.Registry
	fields        permissionports.FieldAccess
	readers       *boardreaderports.Index
	artifacts     artifactReader
	threads       threadReader
	activity      activityReader
	organizations tenantports.OrganizationReader
	attention     deskattentionports.DeskAttentionService
	plan          planFeatures
	scopes        permissionports.ReadScopes
	lists         assistantports.ListSizer
	now           func() time.Time
}

type planFeatures interface {
	Feature(
		ctx context.Context,
		feature platformcatalog.FeatureKey,
	) (permissionports.PlanDenial, bool, error)
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:             p.Logger.Named("service.board-card"),
		permissions:   p.Permissions,
		engine:        p.Permissions,
		registry:      fieldsensitivity.DefaultRegistry(),
		fields:        p.Fields,
		readers:       p.Readers,
		artifacts:     p.Artifacts,
		threads:       p.Conversations,
		activity:      p.Audit,
		organizations: p.Organizations,
		attention:     p.Attention,
		plan:          p.Plan,
		scopes:        p.Scopes,
		now:           time.Now,
	}
}

var errNotABoard = errortypes.NewNotFoundError("Board not found")

func (s *Service) Card(
	ctx context.Context,
	req *assistantports.BoardCardRequest,
) (*assistantports.BoardCard, error) {
	if _, err := s.threads.GetThread(ctx, &req.Thread); err != nil {
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
	location, err := tenantports.OrganizationLocation(ctx, s.organizations, tenant)
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
	tenant := req.Actor.TenantInfo()
	ctx = s.scoped(ctx, req.Actor)
	scope := &readScope{
		actor:    req.Actor,
		tenant:   tenant,
		now:      s.now().Unix(),
		location: s.Location(ctx, tenant),
		reach:    s.reachOf(ctx, req),
	}
	reads := &boardassemble.Reads{
		Now:       scope.now,
		Readable:  make(map[recordkind.Kind]bool, 4),
		Summaries: make(map[recordkind.Kind]map[pulid.ID]boardreading.Summary, 4),
		Readings:  make(map[board.MeasureKey]map[pulid.ID]boardreading.Reading, 2),
		Withheld:  make(map[board.MeasureKey]bool, 1),
		Metrics:   make(map[board.MetricAsk]*boardcatalog.MetricRead, 2),
		Analyses:  map[pulid.ID]*boardassemble.AnalysisRead{},
		Tables:    map[pulid.ID]*boardassemble.TableRead{},
		Lists:     map[pulid.ID]*boardassemble.ListRead{},
		Events:    map[pulid.ID][]board.Event{},
	}

	if err := s.readMetrics(ctx, scope, req.Board.Metrics(), reads); err != nil {
		return nil, err
	}
	refs := req.Board.Records()
	for _, read := range reads.Metrics {
		for _, ref := range boardassemble.MetricRefs(read) {
			if !slices.Contains(refs[ref.Kind], ref.ID) {
				refs[ref.Kind] = append(refs[ref.Kind], ref.ID)
			}
		}
	}
	if err := s.readRecords(ctx, scope, refs, reads); err != nil {
		return nil, err
	}
	if err := s.readMeasures(ctx, scope, req.Board.Measures(), refs, reads); err != nil {
		return nil, err
	}
	s.readArtifacts(ctx, req, reads)
	s.readLists(ctx, req, reads)
	if err := s.readWork(ctx, req, reads); err != nil {
		return nil, err
	}
	if err := s.readEvents(ctx, req, reads); err != nil {
		return nil, err
	}

	return boardassemble.Assemble(req.Board, reads), nil
}

func (s *Service) mayRead(
	ctx context.Context,
	actor *permissionports.RequestActor,
	resource permission.Resource,
) (bool, error) {
	result, err := s.permissions.Check(ctx, actor.PermissionCheck(resource, permission.OpRead))
	if err != nil {
		return false, fmt.Errorf("check %s %s: %w", resource, permission.OpRead, err)
	}

	return result != nil && result.Allowed, nil
}

func mistake(format string, args ...any) error {
	return &agentruntimeports.PublishMistake{Reason: fmt.Sprintf(format, args...)}
}

func (s *Service) Check(ctx context.Context, req *assistantports.BoardFillRequest) error {
	ctx = s.scoped(ctx, req.Actor)
	reach := s.reachOf(ctx, req)
	checks := []func() error{
		func() error { return s.checkKinds(ctx, req) },
		func() error { return s.checkMeasures(ctx, req, reach) },
		func() error { return s.checkMetrics(ctx, req, reach) },
		func() error { return s.checkFigures(ctx, req, reach) },
		func() error { return s.checkTimeline(ctx, req) },
		func() error { return s.checkArtifacts(ctx, req) },
		func() error { return s.checkLists(ctx, req) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) checkKinds(ctx context.Context, req *assistantports.BoardFillRequest) error {
	for kind := range req.Board.Records() {
		allowed, err := s.mayRead(ctx, req.Actor, kind.Resource())
		if err != nil {
			return err
		}
		if !allowed {
			return mistake("the person may not read %s, so leave out the blocks and readings "+
				"that name them", pluralOf(kind))
		}
	}

	return nil
}

func (s *Service) checkMeasures(
	ctx context.Context,
	req *assistantports.BoardFillRequest,
	reach *fieldReach,
) error {
	for kind, measures := range req.Board.Measures() {
		for _, measure := range measures {
			spec, known := measure.Spec(kind)
			if !known {
				continue
			}
			allowed, err := s.allAllowed(ctx, req.Actor, spec.Requires)
			if err != nil {
				return err
			}
			if !allowed || !reach.shows(ctx, measureFields(kind, &spec)) {
				return mistake("the person may not read the %s of %s, so leave that measure "+
					"out", spec.Name, pluralOf(kind))
			}
		}
	}

	return nil
}

func (s *Service) checkMetrics(
	ctx context.Context,
	req *assistantports.BoardFillRequest,
	reach *fieldReach,
) error {
	for _, ask := range req.Board.Metrics() {
		allowed, denial, err := s.metricAllowed(ctx, req.Actor, reach, ask.Metric)
		if err != nil {
			return err
		}
		if denial != nil {
			return mistake("the %s metric is not available: %s, so leave it out and say "+
				"so if the person asked for it", ask.Metric, planwording.Restriction(denial))
		}
		if !allowed {
			return mistake("the person may not read the %s metric, so leave it out", ask.Metric)
		}
	}

	return nil
}

func (s *Service) checkTimeline(ctx context.Context, req *assistantports.BoardFillRequest) error {
	if !slices.ContainsFunc(req.Board.Blocks, func(block board.Block) bool {
		return block.Type == board.BlockTimeline
	}) {
		return nil
	}
	allowed, err := s.mayRead(ctx, req.Actor, permission.ResourceAuditLog)
	if err != nil {
		return err
	}
	if !allowed {
		return mistake("the person may not read a record's history, so leave the timeline out")
	}

	return nil
}

var errArtifactGone = errors.New("artifact not in this conversation")
