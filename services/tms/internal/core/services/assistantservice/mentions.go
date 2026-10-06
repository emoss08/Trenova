package assistantservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	mentionLimitOneKind  = 8
	mentionLimitAllKinds = 3
	maxMentionQueryLen   = 100
)

type mentionKind struct {
	kind     string
	tab      string
	resource permission.Resource
}

var mentionKinds = []mentionKind{
	{kind: "shipment", tab: "shipment", resource: permission.ResourceShipment},
	{kind: "customer", tab: "customer", resource: permission.ResourceCustomer},
	{kind: "invoice", tab: "invoice", resource: permission.ResourceInvoice},
	{kind: "billing_queue_item", tab: "invoice", resource: permission.ResourceBillingQueue},
	{kind: "worker", tab: "worker", resource: permission.ResourceWorker},
	{kind: "carrier", tab: "carrier", resource: permission.ResourceCarrier},
}

func (s *Service) SearchMentions(
	ctx context.Context,
	actor serviceports.RequestActor,
	req serviceports.MentionSearchRequest,
) ([]serviceports.MentionCandidate, error) {
	tab := strings.TrimSpace(req.Kind)
	if tab == "all" {
		tab = ""
	}
	query := strings.TrimSpace(req.Query)
	if len(query) > maxMentionQueryLen {
		query = query[:maxMentionQueryLen]
	}

	wanted := make([]mentionKind, 0, len(mentionKinds))
	for _, candidate := range mentionKinds {
		if tab == "" || candidate.tab == tab {
			wanted = append(wanted, candidate)
		}
	}
	if len(wanted) == 0 || s.permissions == nil {
		return []serviceports.MentionCandidate{}, nil
	}

	checks := make([]serviceports.ResourceOperationCheck, 0, len(wanted))
	for _, candidate := range wanted {
		checks = append(checks, serviceports.ResourceOperationCheck{
			Resource:  candidate.resource.String(),
			Operation: permission.OpRead,
		})
	}
	result, err := s.permissions.CheckBatch(ctx, &serviceports.BatchPermissionCheckRequest{
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

	kinds := make([]string, 0, len(wanted))
	for idx, candidate := range wanted {
		if idx < len(result.Results) && result.Results[idx].Allowed {
			kinds = append(kinds, candidate.kind)
		}
	}
	if len(kinds) == 0 {
		return []serviceports.MentionCandidate{}, nil
	}

	limit := mentionLimitOneKind
	if tab == "" {
		limit = mentionLimitAllKinds
	}
	rows, err := s.conversations.SearchMentions(ctx, repositories.SearchMentionsRequest{
		TenantInfo:   actor.TenantInfo(),
		Query:        query,
		Kinds:        kinds,
		LimitPerKind: limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]serviceports.MentionCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, serviceports.MentionCandidate{
			Type:     row.Type,
			ID:       row.ID,
			Label:    row.Label,
			Subtitle: row.Subtitle,
		})
	}

	return out, nil
}
