package assistantthreads

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
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

// mentionKind is a kind of record the @ search offers and the tab it sits
// under, its own name unless it is filed under another's; the permission that
// guards it is the kind's own. The client's tabs and glyphs are generated from
// this table (client/packages/graphql/scripts/generate-record-kinds.mjs), so
// it stays a list of plain literals.
type mentionKind struct {
	kind recordkind.Kind
	// under is the tab the kind is filed under when it is not its own.
	under string
	// askedFor keeps a kind out of the search across every kind: it is
	// offered only when its own tab is asked for.
	askedFor bool
}

func (m mentionKind) tab() string {
	if m.under != "" {
		return m.under
	}

	return string(m.kind)
}

var mentionKinds = []mentionKind{
	{kind: "shipment"},
	{kind: "customer"},
	{kind: "invoice"},
	{kind: "billing_queue_item", under: "invoice"},
	{kind: "worker"},
	{kind: "carrier"},
	{kind: "invoice_dispute", under: "dispute", askedFor: true},
	{kind: "location", askedFor: true},
}

func (s *Service) SearchMentions(
	ctx context.Context,
	actor *permissionports.RequestActor,
	req assistantports.MentionSearchRequest,
) ([]assistantports.MentionCandidate, error) {
	tab := strings.TrimSpace(req.Kind)
	if tab == "all" {
		tab = ""
	}

	wanted := make([]mentionKind, 0, len(mentionKinds))
	for _, candidate := range mentionKinds {
		if (tab == "" && !candidate.askedFor) || candidate.tab() == tab {
			wanted = append(wanted, candidate)
		}
	}

	limit := mentionLimitOneKind
	if tab == "" {
		limit = mentionLimitAllKinds
	}

	return s.searchKinds(ctx, actor, &mentionSearch{
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
	actor *permissionports.RequestActor,
	req assistantports.MentionPageRequest,
) (*assistantports.MentionPage, error) {
	kind := strings.TrimSpace(req.Kind)
	idx := slices.IndexFunc(mentionKinds, func(candidate mentionKind) bool {
		return string(candidate.kind) == kind
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

	rows, err := s.searchKinds(ctx, actor, &mentionSearch{
		query:  req.Query,
		wanted: mentionKinds[idx : idx+1],
		limit:  limit + 1,
		offset: max(req.Offset, 0),
	})
	if err != nil {
		return nil, err
	}

	page := &assistantports.MentionPage{Results: rows, HasMore: len(rows) > limit}
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
) ([]assistantports.MentionCandidate, error) {
	query := strings.TrimSpace(search.query)
	if len(query) > maxMentionQueryLen {
		query = query[:maxMentionQueryLen]
	}
	if len(search.wanted) == 0 || s.permissions == nil {
		return []assistantports.MentionCandidate{}, nil
	}

	checks := make([]permissionports.ResourceOperationCheck, 0, len(search.wanted))
	for _, candidate := range search.wanted {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  candidate.kind.Resource().String(),
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
			kinds = append(kinds, string(candidate.kind))
		}
	}
	if len(kinds) == 0 {
		return []assistantports.MentionCandidate{}, nil
	}

	rows, err := s.conversations.SearchMentions(ctx, &conversationports.SearchMentionsRequest{
		TenantInfo:   actor.TenantInfo(),
		Query:        query,
		Kinds:        kinds,
		LimitPerKind: search.limit,
		Offset:       search.offset,
	})
	if err != nil {
		return nil, err
	}

	out := make([]assistantports.MentionCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, assistantports.MentionCandidate(row))
	}

	return out, nil
}
