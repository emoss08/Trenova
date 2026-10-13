package watchtowerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type DismisserParams struct {
	fx.In

	Exceptions *Exceptions
}

type Dismisser struct {
	exceptions *Exceptions
}

func NewDismisser(p DismisserParams) *Dismisser {
	return &Dismisser{exceptions: p.Exceptions}
}

type kindAccess struct {
	permissions permissionports.PermissionEngine
}

// visibleKinds is the kinds this reader may be shown: those whose source
// resource they may read. A request naming kinds is narrowed to the ones
// they may see; naming none means all of them.
func (a kindAccess) visibleKinds(
	ctx context.Context,
	actor *permissionports.RequestActor,
	requested []watchtower.SourceKind,
) ([]watchtower.SourceKind, error) {
	candidates := requested
	if len(candidates) == 0 {
		candidates = watchtower.AllSourceKinds()
	}

	allowed := make(map[permission.Resource]bool, len(candidates))
	kinds := make([]watchtower.SourceKind, 0, len(candidates))
	for _, kind := range candidates {
		if !kind.IsValid() {
			return nil, errortypes.NewValidationError(
				"kinds",
				errortypes.ErrInvalid,
				"Unknown watchtower kind",
			)
		}
		resource := kind.ReadResource()
		ok, seen := allowed[resource]
		if !seen {
			result, err := a.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
				PrincipalType:  actor.PrincipalType,
				PrincipalID:    actor.PrincipalID,
				UserID:         actor.UserID,
				APIKeyID:       actor.APIKeyID,
				BusinessUnitID: actor.BusinessUnitID,
				OrganizationID: actor.OrganizationID,
				Resource:       resource.String(),
				Operation:      permission.OpRead,
			})
			if err != nil {
				return nil, err
			}
			ok = result != nil && result.Allowed
			allowed[resource] = ok
		}
		if ok {
			kinds = append(kinds, kind)
		}
	}

	return kinds, nil
}

// Dismiss resolves an item by hand as needing no action. The source is
// untouched: a dismissed failed run is still a failed run, it is just no
// longer on the tower.
func (d *Dismisser) Dismiss(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	change, err := d.PlanDismiss(ctx, tenant, id, actor)
	if err != nil {
		return nil, err
	}
	if change.Before.IsResolved() {
		return change.Before, nil
	}

	return d.exceptions.ResolveException(ctx, &watchtowerports.ResolveExceptionRequest{
		ExceptionTarget: watchtowerports.ExceptionTarget{
			TenantInfo:      tenant,
			ItemID:          id,
			ExpectedVersion: change.Before.Version,
		},
		Code: watchtower.ResolutionNoActionNeeded,
	}, actor)
}

func (d *Dismisser) PlanDismiss(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *permissionports.RequestActor,
) (*kernelports.RecordChange[watchtower.Item], error) {
	item, err := d.exceptions.GetException(
		ctx,
		watchtowerports.GetWatchtowerItemRequest{ID: id, TenantInfo: tenant},
		actor,
	)
	if err != nil {
		return nil, err
	}

	dismissed := *item
	if !dismissed.IsResolved() {
		if err = dismissed.Resolve(
			watchtower.ResolutionNoActionNeeded,
			"",
			actorUserID(actor),
			d.exceptions.now(),
		); err != nil {
			return nil, err
		}
	}

	return &kernelports.RecordChange[watchtower.Item]{Before: item, After: &dismissed}, nil
}

func (a kindAccess) assertMaySee(
	ctx context.Context,
	actor *permissionports.RequestActor,
	item *watchtower.Item,
) error {
	kinds, err := a.visibleKinds(ctx, actor, []watchtower.SourceKind{item.SourceKind})
	if err != nil {
		return err
	}
	if len(kinds) == 0 {
		return errortypes.NewAuthorizationError("You cannot see this watchtower item")
	}

	return nil
}
