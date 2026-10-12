package triagebriefingservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/triageruleservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	BriefingTTL       = 30 * time.Second
	maxMailboxes      = 100
	maxProfileSamples = 3
	weekSeconds       = 7 * triagesupport.DaySeconds
)

type ruleReader interface {
	Suggest(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*inboundmessageports.RuleSuggestion, error)
	SenderRules(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		message *inboundmessage.InboundMessage,
	) ([]*triage.Rule, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Insights      triageports.TriageInsightRepository
	Messages      inboundmessageports.InboundMessageRepository
	Mailboxes     inboundmessageports.InboundMailboxRepository
	Settings      inboundmessageports.TriageSettingsRepository
	Organizations tenantports.OrganizationRepository
	Rules         *triageruleservice.Service
	Cache         triageports.TriageCache                `optional:"true"`
	Models        triageports.InboundModelAvailability   `optional:"true"`
	Replies       inboundmessageports.InboundReplyReader `optional:"true"`
}

type Service struct {
	l         *zap.Logger
	insights  triageports.TriageInsightRepository
	messages  inboundmessageports.InboundMessageRepository
	mailboxes inboundmessageports.InboundMailboxRepository
	settings  inboundmessageports.TriageSettingsRepository
	rules     ruleReader
	cache     triageports.TriageCache
	models    triageports.InboundModelAvailability
	replies   inboundmessageports.InboundReplyReader
	clock     *triagesupport.Clock
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:         p.Logger.Named("service.triage-briefing"),
		insights:  p.Insights,
		messages:  p.Messages,
		mailboxes: p.Mailboxes,
		settings:  p.Settings,
		rules:     p.Rules,
		cache:     p.Cache,
		models:    p.Models,
		replies:   p.Replies,
		clock:     triagesupport.NewClock(p.Organizations, nil),
	}
}

func (s *Service) Briefing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*inboundmessageports.Briefing, error) {
	return triagesupport.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&triagesupport.Request{
			Section:    triageports.CacheBriefing,
			TenantInfo: tenantInfo,
			TTL:        BriefingTTL,
		},
		func(ctx context.Context) (*inboundmessageports.Briefing, error) {
			inputs, err := s.gather(ctx, tenantInfo)
			if err != nil {
				return nil, err
			}

			return Assemble(inputs), nil
		},
	)
}

func (s *Service) gather(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*Inputs, error) {
	loc, err := s.clock.Location(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	dayStart := timeutils.DayStart(now, loc)
	window := triageports.BriefingWindow{
		TenantInfo: tenantInfo,
		DayStart:   dayStart,
		Now:        now,
		Limit:      MaxRiverPoints,
	}

	inputs := &Inputs{Now: now, DayStart: dayStart, Location: loc}
	if inputs.Settings, err = s.settings.Get(ctx, tenantInfo); err != nil {
		return nil, fmt.Errorf("read the triage settings: %w", err)
	}
	if inputs.Counts, err = s.insights.BriefingCounts(ctx, window); err != nil {
		return nil, fmt.Errorf("count the queue: %w", err)
	}
	if inputs.Mailboxes, err = s.listMailboxes(ctx, tenantInfo); err != nil {
		return nil, err
	}
	if inputs.Handled, err = s.insights.HandledByNova(ctx, window); err != nil {
		return nil, fmt.Errorf("read what Nova handled: %w", err)
	}
	if inputs.Arrivals, err = s.insights.Arrivals(ctx, window); err != nil {
		return nil, fmt.Errorf("read the day's arrivals: %w", err)
	}
	if inputs.River, err = s.insights.River(ctx, window); err != nil {
		return nil, fmt.Errorf("read the day's messages: %w", err)
	}
	if err = s.gatherWaiting(ctx, tenantInfo, inputs); err != nil {
		return nil, err
	}

	weekStart := timeutils.WeekStart(now, loc)
	if inputs.Corrections, err = s.insights.CorrectionTotals(ctx, triageports.CorrectionTotalsRequest{
		TenantInfo:    tenantInfo,
		LastWeekStart: timeutils.WeekStart(weekStart-weekSeconds/2, loc),
		WeekStart:     weekStart,
	}); err != nil {
		return nil, fmt.Errorf("count the week's corrections: %w", err)
	}
	if inputs.Suggestion, err = s.rules.Suggest(ctx, tenantInfo); err != nil {
		return nil, fmt.Errorf("find a rule to suggest: %w", err)
	}
	inputs.NovaEnabled = s.novaEnabled(ctx, tenantInfo)

	return inputs, nil
}

func (s *Service) gatherWaiting(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	inputs *Inputs,
) error {
	waiting, err := s.insights.Waiting(ctx, triageports.WaitingRequest{
		TenantInfo: tenantInfo,
		Limit:      MaxWaitingScanned,
	})
	if err != nil {
		return fmt.Errorf("read the waiting messages: %w", err)
	}
	inputs.Waiting = waiting
	inputs.Replies = map[pulid.ID]*triage.Reply{}
	if len(waiting) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(waiting))
	for _, message := range waiting {
		ids = append(ids, message.ID)
	}

	attachments, err := s.messages.ListAttachmentsByMessageIDs(
		ctx,
		inboundmessageports.ListAttachmentsByMessageIDsRequest{
			TenantInfo: tenantInfo,
			MessageIDs: ids,
		},
	)
	if err != nil {
		return fmt.Errorf("read the waiting messages' attachments: %w", err)
	}
	for _, message := range waiting {
		message.Attachments = attachments[message.ID]
	}

	if s.replies != nil {
		if inputs.Replies, err = s.replies.LatestByMessageIDs(ctx, tenantInfo, ids); err != nil {
			return fmt.Errorf("read the waiting messages' replies: %w", err)
		}
	}

	return nil
}

func (s *Service) listMailboxes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*inboundmessage.Mailbox, error) {
	result, err := s.mailboxes.List(ctx, &inboundmessageports.ListMailboxesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: maxMailboxes},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("read the sources: %w", err)
	}

	return result.Items, nil
}

func (s *Service) novaEnabled(ctx context.Context, tenantInfo pagination.TenantInfo) bool {
	if s.models == nil {
		return false
	}

	enabled, err := s.models.ReadsInboundMail(ctx, tenantInfo)
	if err != nil {
		s.l.Warn("could not tell whether Nova reads mail",
			zap.String("organizationId", tenantInfo.OrgID.String()), zap.Error(err))

		return false
	}

	return enabled
}

func (s *Service) UnhealthySources(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*inboundmessage.Mailbox, error) {
	if len(ids) == 0 {
		return []*inboundmessage.Mailbox{}, nil
	}

	mailboxes, err := s.listMailboxes(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	wanted := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}

	unhealthy := make([]*inboundmessage.Mailbox, 0, len(ids))
	for _, mailbox := range mailboxes {
		if _, ok := wanted[mailbox.ID]; ok {
			unhealthy = append(unhealthy, mailbox)
		}
	}

	return unhealthy, nil
}
