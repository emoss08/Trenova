package triageruleservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MinSuggestionCount = 5
	MinSuggestionShare = 0.9
	representative     = pulid.ID("representative")
)

type suggestionResult struct {
	Suggestion *inboundmessageports.RuleSuggestion `json:"suggestion"`
}

func (s *Service) Suggest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*inboundmessageports.RuleSuggestion, error) {
	result, err := triagesupport.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&triagesupport.Request{
			Section:    triageports.CacheSuggestion,
			TenantInfo: tenantInfo,
			TTL:        suggestionTTL,
		},
		func(ctx context.Context) (*suggestionResult, error) {
			rows, err := s.insights.ManualOutcomes(ctx, &triageports.ManualOutcomesRequest{
				TenantInfo: tenantInfo,
				Since:      s.clock.DaysAgo(triagesupport.PreviewWindowDays),
			})
			if err != nil {
				return nil, err
			}

			rules, err := s.rules.List(ctx, triageports.ListRulesRequest{TenantInfo: tenantInfo})
			if err != nil {
				return nil, err
			}

			return &suggestionResult{Suggestion: BestSuggestion(rows, rules)}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return result.Suggestion, nil
}

type outcomeKey struct {
	domain string
	class  inboundmessage.Classification
}

type outcomeGroup struct {
	byHand int
	top    *triageports.ManualOutcomeRow
}

func BestSuggestion(
	rows []*triageports.ManualOutcomeRow,
	rules []*triage.Rule,
) *inboundmessageports.RuleSuggestion {
	groups := make(map[outcomeKey]*outcomeGroup, len(rows))
	order := make([]outcomeKey, 0, len(rows))
	for _, row := range rows {
		key := outcomeKey{domain: row.SenderDomain, class: row.Classification}
		group, ok := groups[key]
		if !ok {
			group = &outcomeGroup{}
			groups[key] = group
			order = append(order, key)
		}
		group.byHand += row.Count
		if group.top == nil || row.Count > group.top.Count {
			group.top = row
		}
	}

	candidates := make([]*inboundmessageports.RuleSuggestion, 0, len(order))
	for _, key := range order {
		if suggestion := suggestionFor(groups[key], rules); suggestion != nil {
			candidates = append(candidates, suggestion)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	slices.SortStableFunc(candidates, func(a, b *inboundmessageports.RuleSuggestion) int {
		if c := cmp.Compare(b.HandledByHand, a.HandledByHand); c != 0 {
			return c
		}
		if c := cmp.Compare(b.Total, a.Total); c != 0 {
			return c
		}

		return cmp.Compare(a.Sender, b.Sender)
	})

	return candidates[0]
}

func suggestionFor(
	group *outcomeGroup,
	rules []*triage.Rule,
) *inboundmessageports.RuleSuggestion {
	top := group.top
	action, ok := actionFor(top.Outcome)
	if !ok || top.Count < MinSuggestionCount || group.byHand == 0 ||
		float64(top.Count)/float64(group.byHand) < MinSuggestionShare {
		return nil
	}
	if emailutils.ValidateDomain(top.SenderDomain) != nil || !top.Classification.IsValid() {
		return nil
	}

	suggestion := &inboundmessageports.RuleSuggestion{
		Classification: top.Classification,
		SenderScope:    triage.SenderDomain,
		Sender:         top.SenderDomain,
		Action:         action,
		HandledByHand:  top.Count,
		Total:          max(top.Arrived, group.byHand),
	}
	if top.Mailboxes == 1 && top.MailboxID.IsNotNil() {
		mailboxID := top.MailboxID
		suggestion.MailboxID = &mailboxID
	}
	if covered(suggestion, top, rules) {
		return nil
	}

	return suggestion
}

func actionFor(outcome triage.ActivityKind) (inboundmessage.NovaAction, bool) {
	switch outcome { //nolint:exhaustive // only filing and ignoring settle a message a rule can repeat
	case triage.ActivityFiled:
		return inboundmessage.NovaActionFileOnShipment, true
	case triage.ActivityIgnored:
		return inboundmessage.NovaActionIgnore, true
	default:
		return "", false
	}
}

func covered(
	suggestion *inboundmessageports.RuleSuggestion,
	top *triageports.ManualOutcomeRow,
	rules []*triage.Rule,
) bool {
	message := &inboundmessage.InboundMessage{
		Classification: suggestion.Classification,
		FromAddress:    "sender@" + suggestion.Sender,
		SenderDomain:   suggestion.Sender,
	}
	if suggestion.MailboxID != nil {
		message.MailboxID = *suggestion.MailboxID
	}
	if top.AllCustomers {
		message.MatchedCustomerID = representative
	}
	if top.AllCarriers {
		message.MatchedCarrierID = representative
	}

	for _, rule := range rules {
		if rule.Covers(message) {
			return true
		}
	}

	return false
}
