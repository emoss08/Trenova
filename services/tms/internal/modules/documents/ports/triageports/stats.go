package triageports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TriageRuleStats interface {
	MatchesThisMonth(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ruleIDs []pulid.ID,
	) (map[pulid.ID]int, error)
	Samples(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		messageIDs []pulid.ID,
	) (map[pulid.ID]*inboundmessageports.Sample, error)
	MailboxesByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]*inboundmessage.Mailbox, error)
}
