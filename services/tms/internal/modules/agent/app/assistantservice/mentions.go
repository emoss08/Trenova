package assistantservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	mentionLimitOneKind  = 8
	mentionLimitAllKinds = 3
	maxMentionQueryLen   = 100
	mentionPageDefault   = 25
	mentionPageMax       = 50
)

type mentionKind struct {
	kind     string
	tab      string
	resource permission.Resource
	// askedFor keeps a kind out of the search across every kind: it is
	// offered only when its own tab is asked for.
	askedFor bool
}

var mentionKinds = []mentionKind{
	{kind: "shipment", tab: "shipment", resource: permission.ResourceShipment},
	{kind: "customer", tab: "customer", resource: permission.ResourceCustomer},
	{kind: "invoice", tab: "invoice", resource: permission.ResourceInvoice},
	{kind: "billing_queue_item", tab: "invoice", resource: permission.ResourceBillingQueue},
	{kind: "worker", tab: "worker", resource: permission.ResourceWorker},
	{kind: "carrier", tab: "carrier", resource: permission.ResourceCarrier},
	{
		kind:     "invoice_dispute",
		tab:      "dispute",
		resource: permission.ResourceInvoiceDispute,
		askedFor: true,
	},
}

func (s *Service) SearchMentions(
	ctx context.Context,
	actor permissionports.RequestActor,
	req agentports.MentionSearchRequest,
) ([]agentports.MentionCandidate, error) {
	tab := strings.TrimSpace(req.Kind)
	if tab == "all" {
		tab = ""
	}

	wanted := make([]mentionKind, 0, len(mentionKinds))
	for _, candidate := range mentionKinds {
		if (tab == "" && !candidate.askedFor) || candidate.tab == tab {
			wanted = append(wanted, candidate)
		}
	}

	limit := mentionLimitOneKind
	if tab == "" {
		limit = mentionLimitAllKinds
	}

	return s.searchKinds(ctx, &actor, &mentionSearch{
		query:  req.Query,
		wanted: wanted,
		limit:  limit,
	})
}

// SearchMentionPage is one page of the records of one kind, for a list that
// scrolls through all of them. One row past the page says whether another
// follows without counting the rest.
func (s *Service) SearchMentionPage(
	ctx context.Context,
	actor permissionports.RequestActor,
	req agentports.MentionPageRequest,
) (*agentports.MentionPage, error) {
	kind := strings.TrimSpace(req.Kind)
	idx := slices.IndexFunc(mentionKinds, func(candidate mentionKind) bool {
		return candidate.kind == kind
	})
	if idx < 0 {
		return nil, errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "Search one kind of record at a time",
		)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = mentionPageDefault
	}
	limit = min(limit, mentionPageMax)

	rows, err := s.searchKinds(ctx, &actor, &mentionSearch{
		query:  req.Query,
		wanted: mentionKinds[idx : idx+1],
		limit:  limit + 1,
		offset: max(req.Offset, 0),
	})
	if err != nil {
		return nil, err
	}

	page := &agentports.MentionPage{Results: rows, HasMore: len(rows) > limit}
	if page.HasMore {
		page.Results = rows[:limit]
	}

	return page, nil
}

type mentionSearch struct {
	query  string
	wanted []mentionKind
	limit  int
	offset int
}

// searchKinds searches the kinds the person may read, each kind's rows
// capped at the limit.
func (s *Service) searchKinds(
	ctx context.Context,
	actor *permissionports.RequestActor,
	search *mentionSearch,
) ([]agentports.MentionCandidate, error) {
	query := strings.TrimSpace(search.query)
	if len(query) > maxMentionQueryLen {
		query = query[:maxMentionQueryLen]
	}
	if len(search.wanted) == 0 || s.permissions == nil {
		return []agentports.MentionCandidate{}, nil
	}

	checks := make([]permissionports.ResourceOperationCheck, 0, len(search.wanted))
	for _, candidate := range search.wanted {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  candidate.resource.String(),
			Operation: permission.OpRead,
		})
	}
	result, err := s.permissions.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Checks:         checks,
	})
	if err != nil {
		return nil, fmt.Errorf("check which records may be mentioned: %w", err)
	}

	kinds := make([]string, 0, len(search.wanted))
	for idx, candidate := range search.wanted {
		if idx < len(result.Results) && result.Results[idx].Allowed {
			kinds = append(kinds, candidate.kind)
		}
	}
	if len(kinds) == 0 {
		return []agentports.MentionCandidate{}, nil
	}

	rows, err := s.conversations.SearchMentions(ctx, agentports.SearchMentionsRequest{
		TenantInfo:   actor.TenantInfo(),
		Query:        query,
		Kinds:        kinds,
		LimitPerKind: search.limit,
		Offset:       search.offset,
	})
	if err != nil {
		return nil, err
	}

	out := make([]agentports.MentionCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentports.MentionCandidate(row))
	}

	return out, nil
}
