package triagebriefingservice

import (
	"context"
	"math"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type SenderProfileRequest struct {
	TenantInfo pagination.TenantInfo
	MessageID  pulid.ID
}

func (s *Service) SenderProfile(
	ctx context.Context,
	req SenderProfileRequest,
) (*inboundmessageports.SenderProfile, error) {
	message, err := s.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	domain := stringutils.FirstNonEmpty(message.SenderDomain, emailutils.Domain(message.FromAddress))
	stats, err := s.insights.SenderStats(ctx, triageports.SenderStatsRequest{
		TenantInfo:   req.TenantInfo,
		SenderDomain: domain,
		Since:        s.clock.DaysAgo(triagesupport.ProfileWindowDays),
		ExcludeID:    message.ID,
		RecentLimit:  maxProfileSamples,
	})
	if err != nil {
		return nil, err
	}

	party, err := s.insights.SenderParty(ctx, triageports.SenderPartyRequest{
		TenantInfo:   req.TenantInfo,
		Address:      message.FromAddress,
		SenderDomain: domain,
		CustomerID:   pulid.FirstNotNil(message.MatchedCustomerID, stats.CustomerID),
		CarrierID:    pulid.FirstNotNil(message.MatchedCarrierID, stats.CarrierID),
	})
	if err != nil {
		return nil, err
	}

	rules, err := s.rules.SenderRules(ctx, req.TenantInfo, message)
	if err != nil {
		return nil, err
	}

	return Profile(ProfileInputs{
		Address: message.FromAddress,
		Name:    message.FromName,
		Domain:  domain,
		Stats:   stats,
		Party:   party,
		Rules:   rules,
	}), nil
}

type ProfileInputs struct {
	Address string
	Name    string
	Domain  string
	Stats   *triageports.SenderStats
	Party   *triageports.SenderParty
	Rules   []*triage.Rule
}

func Profile(in ProfileInputs) *inboundmessageports.SenderProfile {
	profile := &inboundmessageports.SenderProfile{
		Address:            in.Address,
		Name:               stringutils.FirstNonEmpty(in.Name, emailutils.LocalPart(in.Address)),
		Domain:             in.Domain,
		Role:               in.Party.Role,
		MessagesLast90Days: in.Stats.Messages,
		Kinds:              make([]*inboundmessageports.SenderKindCount, 0, len(in.Stats.Kinds)),
		Rules:              in.Rules,
		Recent:             triagesupport.SamplesOf(in.Stats.Recent),
	}
	if in.Party.Since > 0 {
		since := in.Party.Since
		profile.Since = &since
	}
	if in.Stats.UsualSettleSeconds != nil && *in.Stats.UsualSettleSeconds >= 0 {
		minutes := max(1, int(math.Round(*in.Stats.UsualSettleSeconds/secondsPerMinute)))
		profile.UsualReplyMinutes = &minutes
	}
	for _, kind := range in.Stats.Kinds {
		profile.Kinds = append(profile.Kinds, &inboundmessageports.SenderKindCount{
			Classification: kind.Classification,
			Count:          kind.Count,
		})
	}

	return profile
}
