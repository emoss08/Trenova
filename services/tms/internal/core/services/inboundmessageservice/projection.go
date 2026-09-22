package inboundmessageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/pagination"
)

// project puts a message on the watchtower, or takes it off.
//
// Only a message waiting on a person belongs there. One the desk answered by
// itself is work that happened rather than work owed, and leaving it on the
// feed would bury the ones that are actually waiting — so settling a message
// resolves its item as readily as it raises one.
func (s *Service) project(ctx context.Context, message *inboundmessage.InboundMessage) {
	if s.watchtower == nil {
		return
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: message.OrganizationID,
		BuID:  message.BusinessUnitID,
	}

	if !message.NeedsReview() {
		s.watchtower.Resolve(
			ctx, tenantInfo, watchtower.SourceInboundMessage, message.ID.String(),
		)

		return
	}

	s.watchtower.Upsert(ctx, watchtowersources.DescribeInboundMessage(message))
}

// announce tells the desks a message was read.
//
// The event fires once the message is settled rather than when it arrived,
// because a desk asked to act on a message nobody has read yet has nothing to
// act on. It fires whatever the outcome: a desk that subscribes to intake is
// as interested in the tender it may take over as in the one it may not.
func (s *Service) announce(ctx context.Context, message *inboundmessage.InboundMessage) {
	services.PublishAgentEvent(ctx, s.events, services.AgentEvent{
		Kind:      agent.EventInboundMessageClassified,
		SubjectID: message.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: message.OrganizationID,
			BuID:  message.BusinessUnitID,
		},
	})
}
