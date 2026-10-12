package triageruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) MatchesThisMonth(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ruleIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	if len(ruleIDs) == 0 {
		return map[pulid.ID]int{}, nil
	}

	loc, err := s.clock.Location(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return s.rules.CountSettledSince(ctx, &triageports.CountRuleSettlementsRequest{
		TenantInfo: tenantInfo,
		RuleIDs:    ruleIDs,
		Since:      timeutils.MonthStart(s.clock.Now(), loc),
	})
}

func (s *Service) Samples(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageIDs []pulid.ID,
) (map[pulid.ID]*inboundmessageports.Sample, error) {
	samples := make(map[pulid.ID]*inboundmessageports.Sample, len(messageIDs))
	if len(messageIDs) == 0 {
		return samples, nil
	}

	rows, err := s.insights.MessageRows(ctx, &triageports.MessageRowsRequest{
		TenantInfo: tenantInfo,
		IDs:        messageIDs,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		samples[row.ID] = triagesupport.SampleOf(row)
	}

	return samples, nil
}

func (s *Service) SenderDomains(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
) ([]string, error) {
	return s.insights.SenderDomains(ctx, &triageports.SenderDomainsRequest{
		TenantInfo: tenantInfo,
		Query:      query,
		Since:      s.clock.DaysAgo(triagesupport.ProfileWindowDays),
		Limit:      maxSenderDomains,
	})
}

func (s *Service) SenderRules(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	message *inboundmessage.InboundMessage,
) ([]*triage.Rule, error) {
	rules, err := s.rules.List(ctx, triageports.ListRulesRequest{TenantInfo: tenantInfo})
	if err != nil {
		return nil, err
	}

	covering := make([]*triage.Rule, 0, len(rules))
	for _, rule := range rules {
		if rule.SenderScope != triage.SenderAnyone && rule.MatchesSender(message) {
			covering = append(covering, rule)
		}
	}

	return covering, nil
}

func (s *Service) MailboxesByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]*inboundmessage.Mailbox, error) {
	found := make(map[pulid.ID]*inboundmessage.Mailbox, len(ids))
	if len(ids) == 0 {
		return found, nil
	}

	result, err := s.mailboxes.List(ctx, &inboundmessageports.ListMailboxesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: maxMailboxes},
		},
	})
	if err != nil {
		return nil, err
	}

	wanted := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	for _, mailbox := range result.Items {
		if _, ok := wanted[mailbox.ID]; ok {
			found[mailbox.ID] = mailbox
		}
	}

	return found, nil
}
