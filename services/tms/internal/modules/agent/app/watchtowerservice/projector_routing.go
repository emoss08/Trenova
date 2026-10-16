package watchtowerservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (p *Projector) route(
	ctx context.Context,
	saved *watchtower.Item,
	ownerID pulid.ID,
	inserted bool,
) *watchtower.Item {
	if saved == nil || saved.IsResolved() {
		return saved
	}
	needsDue := saved.DueAt == nil && saved.DueSource != watchtower.DueSourcePerson
	needsAssignee := saved.AssigneeID.IsNil() && (inserted || saved.Reopened)
	if !needsDue && !needsAssignee {
		return saved
	}

	tenant := pagination.TenantInfo{OrgID: saved.OrganizationID, BuID: saved.BusinessUnitID}
	policy := p.router.Policy(ctx, tenant)
	now := p.now()
	routed := *saved
	events := make([]*watchtower.Event, 0, 2)

	if needsDue {
		due := DueAt(policy, routed.Severity, now)
		routed.DueAt = &due
		routed.DueLabel = ""
		routed.DueSource = watchtower.DueSourcePolicy
		events = append(events, watchtower.NewEvent(&routed, watchtower.EventDueChanged, pulid.Nil).
			WithChange("", strconv.FormatInt(due, 10)))
	}
	if needsAssignee {
		if assignee := p.pickAssignee(
			ctx,
			&routed,
			tenant,
			ownerID,
			policy.DefaultAssigneeID,
		); assignee.IsNotNil() {
			routed.AssigneeID = assignee
			events = append(
				events,
				watchtower.NewEvent(&routed, watchtower.EventAssigned, pulid.Nil).
					WithChange("", assignee.String()),
			)
		}
	}

	updated, err := p.repo.UpdateWorkflow(ctx, &watchtowerports.UpdateWatchtowerWorkflowRequest{
		Item:            &routed,
		ExpectedVersion: saved.Version,
		Events:          events,
	})
	if err != nil {
		p.l.Warn("could not route an exception",
			zap.String("item", saved.ID.String()), zap.Error(err))
		return saved
	}
	updated.Reopened = saved.Reopened

	if updated.AssigneeID.IsNotNil() && updated.AssigneeID != saved.AssigneeID &&
		p.notifications != nil {
		sendExceptionNotice(ctx, p.notifications, p.l, &exceptionNotice{
			item:      updated,
			userID:    updated.AssigneeID,
			eventType: EventAssigned,
			title:     "Exception assigned to you: " + updated.Title,
			priority:  noticePriority(updated.Severity),
		})
	}

	return updated
}

func (p *Projector) pickAssignee(
	ctx context.Context,
	item *watchtower.Item,
	tenant pagination.TenantInfo,
	candidates ...pulid.ID,
) pulid.ID {
	for _, candidate := range candidates {
		if candidate.IsNil() {
			continue
		}
		if p.mayWork(ctx, item, tenant, candidate) {
			return candidate
		}
	}
	return pulid.Nil
}

func (p *Projector) mayWork(
	ctx context.Context,
	item *watchtower.Item,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) bool {
	if p.access.permissions == nil {
		return false
	}
	ok, err := mayWorkException(ctx, p.access, item, pagination.TenantInfo{
		OrgID:  tenant.OrgID,
		BuID:   tenant.BuID,
		UserID: userID,
	})
	if err != nil {
		p.l.Warn("could not check who may own an exception",
			zap.String("item", item.ID.String()),
			zap.String("user", userID.String()),
			zap.Error(err))
		return false
	}
	return ok
}

func mayWorkException(
	ctx context.Context,
	access kindAccess,
	item *watchtower.Item,
	tenant pagination.TenantInfo,
) (bool, error) {
	actor := permissionports.UserActor(tenant)
	result, err := access.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceWatchtower.String(),
		Operation:      permission.OpRead,
	})
	if err != nil {
		return false, err
	}
	if result == nil || !result.Allowed {
		return false, nil
	}
	kinds, err := access.visibleKinds(ctx, actor, []watchtower.SourceKind{item.SourceKind})
	if err != nil {
		return false, err
	}
	return len(kinds) > 0, nil
}

func noticePriority(severity watchtower.Severity) notification.Priority {
	switch severity {
	case watchtower.SeverityCritical:
		return notification.PriorityHigh
	case watchtower.SeverityWarning:
		return notification.PriorityMedium
	case watchtower.SeverityInfo:
		return notification.PriorityLow
	}
	return notification.PriorityMedium
}
