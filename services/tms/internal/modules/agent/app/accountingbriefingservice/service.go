package accountingbriefingservice

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/shipmentboardbrief"
	"github.com/emoss08/trenova/internal/modules/agent/app/shipmentnarration"
	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/accountingbriefports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxOutputTokens   = 700
	maxNarrationChars = 560
	maxSegments       = accountingbrief.MaxSegments
	schemaName        = "accounting_dashboard_briefing"
	refreshAfter      = 30 * time.Minute
)

type Params struct {
	fx.In

	Receivables   customerledgerports.ARSnapshotReader
	Collections   collectionports.Service
	Briefs        accountingbriefports.AccountingBriefRepository
	Organizations tenantports.OrganizationRepository
	Completion    llmports.CompletionService
	Logger        *zap.Logger
}

type Dependencies struct {
	Receivables   customerledgerports.ARSnapshotReader
	Collections   collectionports.Service
	Briefs        accountingbriefports.AccountingBriefRepository
	Organizations tenantports.OrganizationRepository
	Completion    llmports.CompletionService
	Logger        *zap.Logger
	Now           func() time.Time
}

type Service struct {
	receivables   customerledgerports.ARSnapshotReader
	collections   collectionports.Service
	briefs        accountingbriefports.AccountingBriefRepository
	organizations tenantports.OrganizationRepository
	completion    llmports.CompletionService
	l             *zap.Logger
	now           func() time.Time
}

var (
	_ accountingbriefports.AccountingBriefingReader = (*Service)(nil)
	_ accountingbriefports.AccountingBriefWriter    = (*Service)(nil)
)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return NewWithDependencies(&Dependencies{
		Receivables:   p.Receivables,
		Collections:   p.Collections,
		Briefs:        p.Briefs,
		Organizations: p.Organizations,
		Completion:    p.Completion,
		Logger:        p.Logger,
		Now:           time.Now,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	return &Service{
		receivables:   d.Receivables,
		collections:   d.Collections,
		briefs:        d.Briefs,
		organizations: d.Organizations,
		completion:    d.Completion,
		l:             d.Logger.Named("service.accounting-briefing"),
		now:           d.Now,
	}
}

func NewReader(s *Service) accountingbriefports.AccountingBriefingReader { return s }

func NewWriter(s *Service) accountingbriefports.AccountingBriefWriter { return s }

func (s *Service) Briefing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*accountingbriefports.AccountingBriefing, error) {
	day, err := shipmentboardbrief.Today(ctx, s.organizations, tenantInfo)
	if err != nil {
		return nil, err
	}
	brief, err := s.latest(ctx, tenantInfo, day)
	if err != nil {
		return nil, err
	}

	if brief == nil {
		brief, err = s.write(ctx, tenantInfo, day, accountingbrief.TriggerOnDemand, nil, nil)
	} else if live := s.outdated(ctx, tenantInfo, day, brief); live != nil {
		brief, err = s.write(ctx, tenantInfo, day, accountingbrief.TriggerRefreshed, brief, live)
	}
	if err != nil {
		return nil, err
	}

	return toBriefing(brief), nil
}

func (s *Service) WriteBrief(
	ctx context.Context,
	req *accountingbriefports.WriteAccountingBriefRequest,
) (*accountingbrief.Brief, error) {
	day, err := shipmentboardbrief.Today(ctx, s.organizations, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	previous, err := s.latest(ctx, req.TenantInfo, day)
	if err != nil {
		return nil, err
	}
	if previous != nil && req.Trigger == accountingbrief.TriggerScheduled &&
		previous.Trigger == accountingbrief.TriggerScheduled {
		return previous, nil
	}

	return s.write(ctx, req.TenantInfo, day, req.Trigger, previous, nil)
}

func (s *Service) latest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	day shipmentboardbrief.Day,
) (*accountingbrief.Brief, error) {
	return s.briefs.Latest(ctx, &accountingbriefports.GetLatestAccountingBriefRequest{
		TenantInfo: tenantInfo,
		BriefDate:  day.Date,
	})
}

func (s *Service) outdated(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	day shipmentboardbrief.Day,
	brief *accountingbrief.Brief,
) *accountingbrief.Facts {
	now := s.now()
	if !accountingbrief.CanRefresh(brief.Generation) ||
		now.Sub(time.Unix(brief.CheckedAt(), 0)) < refreshAfter {
		return nil
	}

	live, err := s.Facts(ctx, tenantInfo, day.Timezone)
	if err != nil {
		s.l.Warn("could not read the receivables figures", zap.Error(err))
		return nil
	}
	if !live.Same(brief.Facts) {
		return live
	}

	if err = s.briefs.MarkVerified(ctx, &accountingbriefports.MarkAccountingBriefVerifiedRequest{
		TenantInfo: tenantInfo,
		ID:         brief.ID,
		VerifiedAt: now.Unix(),
	}); err != nil {
		s.l.Warn("could not record that the accounting brief still holds", zap.Error(err))
	}

	return nil
}

func (s *Service) write(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	day shipmentboardbrief.Day,
	trigger accountingbrief.Trigger,
	previous *accountingbrief.Brief,
	facts *accountingbrief.Facts,
) (*accountingbrief.Brief, error) {
	if facts == nil {
		computed, err := s.Facts(ctx, tenantInfo, day.Timezone)
		if err != nil {
			return nil, err
		}
		facts = computed
	}

	segments := Deterministic(*facts)
	narrated, model := false, ""
	if worded, wordedBy, ok := s.narrate(ctx, tenantInfo, *facts, segments); ok {
		segments, narrated, model = worded, true, wordedBy
	}

	generation := 1
	if previous != nil {
		generation = previous.Generation + 1
	}

	generatedAt := s.now().Unix()
	brief := &accountingbrief.Brief{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		BriefDate:       day.Date,
		Generation:      generation,
		Trigger:         trigger,
		Segments:        toStored(segments),
		Facts:           *facts,
		Narrated:        narrated,
		ModelIdentifier: model,
		GeneratedAt:     generatedAt,
		VerifiedAt:      generatedAt,
	}
	brief.Normalize()

	multiErr := errortypes.NewMultiError()
	brief.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.briefs.Insert(ctx, brief)
}

type narrationDraft struct {
	Segments []struct {
		Text string `json:"text"`
		Link string `json:"link"`
	} `json:"segments"`
}

func (s *Service) narrate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	facts accountingbrief.Facts,
	plain []accountingbriefports.AccountingBriefingSegment,
) ([]accountingbriefports.AccountingBriefingSegment, string, bool) {
	if !facts.HasReceivables() {
		return nil, "", false
	}

	allowed := facts.Links()
	narration, ok := shipmentnarration.Narrate[narrationDraft](
		ctx,
		s.completion,
		s.l,
		&llmports.StructuredCompletionRequest{
			TenantInfo:   tenantInfo,
			Task:         aitask.TaskDailyBriefing,
			System:       systemPrompt,
			Context:      buildContext(facts, plain),
			OutputSchema: outputSchema(allowed),
			SchemaName:   schemaName,
			MaxTokens:    maxOutputTokens,
			Attribution:  aiusageports.AIUsageAttribution{UserID: tenantInfo.UserID},
		},
	)
	if !ok {
		return nil, "", false
	}

	segments, accepted := AcceptNarration(narration.Draft.toSegments(), facts, allowed)
	if !accepted {
		return nil, "", false
	}
	return segments, narration.Model, true
}

func (d *narrationDraft) toSegments() []accountingbriefports.AccountingBriefingSegment {
	out := make([]accountingbriefports.AccountingBriefingSegment, 0, len(d.Segments))
	for _, segment := range d.Segments {
		item := accountingbriefports.AccountingBriefingSegment{Text: segment.Text}
		if segment.Link != "" {
			link := accountingbrief.Link(segment.Link)
			item.Link = &link
		}
		out = append(out, item)
	}
	return out
}

func AcceptNarration(
	segments []accountingbriefports.AccountingBriefingSegment,
	facts accountingbrief.Facts,
	allowed []accountingbrief.Link,
) ([]accountingbriefports.AccountingBriefingSegment, bool) {
	if len(segments) == 0 || len(segments) > maxSegments {
		return nil, false
	}

	var builder strings.Builder
	for _, segment := range segments {
		if link := segment.Link; link != nil {
			if !slices.Contains(allowed, *link) {
				return nil, false
			}
			if !namesItsCustomer(segment.Text, *link, facts) {
				return nil, false
			}
		}
		builder.WriteString(segment.Text)
	}

	prose := strings.TrimSpace(builder.String())
	if prose == "" || len(prose) > maxNarrationChars {
		return nil, false
	}
	if !shipmentnarration.Supported(prose, supportedValues(facts)) {
		return nil, false
	}

	return segments, true
}

func toStored(segments []accountingbriefports.AccountingBriefingSegment) []accountingbrief.Segment {
	out := make([]accountingbrief.Segment, 0, len(segments))
	for _, segment := range segments {
		out = append(out, accountingbrief.Segment{Text: segment.Text, Link: segment.Link})
	}
	return out
}

func namesItsCustomer(text string, link accountingbrief.Link, facts accountingbrief.Facts) bool {
	switch link {
	case accountingbrief.LinkTopCustomer, accountingbrief.LinkBrokenPromise:
		_, name, ok := facts.CustomerFor(link)
		return !ok || strings.Contains(text, name)
	default:
		return true
	}
}

func customerPointer(facts accountingbrief.Facts, link *accountingbrief.Link) *pulid.ID {
	if link == nil {
		return nil
	}
	id, _, ok := facts.CustomerFor(*link)
	if !ok {
		return nil
	}
	return &id
}

func toBriefing(brief *accountingbrief.Brief) *accountingbriefports.AccountingBriefing {
	segments := make([]accountingbriefports.AccountingBriefingSegment, 0, len(brief.Segments))
	for _, segment := range brief.Segments {
		segments = append(segments, accountingbriefports.AccountingBriefingSegment{
			Text:       segment.Text,
			Link:       segment.Link,
			CustomerID: customerPointer(brief.Facts, segment.Link),
		})
	}

	return &accountingbriefports.AccountingBriefing{
		Segments:    segments,
		Actions:     toActions(brief.Facts.Actions()),
		Narrated:    brief.Narrated,
		GeneratedAt: brief.GeneratedAt,
		Generation:  brief.Generation,
	}
}

func toActions(actions []accountingbrief.Action) []accountingbriefports.AccountingBriefingAction {
	out := make([]accountingbriefports.AccountingBriefingAction, 0, len(actions))
	for _, action := range actions {
		item := accountingbriefports.AccountingBriefingAction{
			Link:         action.Link,
			Count:        action.Count,
			AmountMinor:  action.AmountMinor,
			CustomerName: action.CustomerName,
		}
		if !action.CustomerID.IsNil() {
			id := action.CustomerID
			item.CustomerID = &id
		}
		out = append(out, item)
	}
	return out
}
