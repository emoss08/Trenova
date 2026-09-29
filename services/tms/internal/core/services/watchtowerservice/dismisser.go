package watchtowerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type DismisserParams struct {
	fx.In

	Repo        repositories.WatchtowerRepository
	Projector   *Projector
	Permissions services.PermissionEngine
}

type Dismisser struct {
	repo      repositories.WatchtowerRepository
	projector *Projector
	access    kindAccess
	now       func() int64
}

func NewDismisser(p DismisserParams) *Dismisser {
	return &Dismisser{
		repo:      p.Repo,
		projector: p.Projector,
		access:    kindAccess{permissions: p.Permissions},
		now:       timeutils.NowUnix,
	}
}

type kindAccess struct {
	permissions services.PermissionEngine
}

// visibleKinds is the kinds this reader may be shown: those whose source
// resource they may read. A request naming kinds is narrowed to the ones
// they may see; naming none means all of them.
func (a kindAccess) visibleKinds(
	ctx context.Context,
	actor *services.RequestActor,
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
			result, err := a.permissions.Check(ctx, &services.PermissionCheckRequest{
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

// Dismiss resolves an item by hand. The source is untouched: a dismissed
// failed run is still a failed run, it is just no longer on the tower.
func (d *Dismisser) Dismiss(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *services.RequestActor,
) (*watchtower.Item, error) {
	change, err := d.PlanDismiss(ctx, tenant, id, actor)
	if err != nil {
		return nil, err
	}
	if change.Before.IsResolved() {
		return change.Before, nil
	}

	resolved, err := d.repo.ResolveByID(
		ctx,
		repositories.GetWatchtowerItemRequest{ID: id, TenantInfo: tenant},
		d.now(),
	)
	if err != nil {
		return nil, err
	}
	d.projector.publish(ctx, resolved, "resolved")

	return resolved, nil
}

func (d *Dismisser) PlanDismiss(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *services.RequestActor,
) (*services.RecordChange[watchtower.Item], error) {
	item, err := d.repo.GetByID(
		ctx,
		repositories.GetWatchtowerItemRequest{ID: id, TenantInfo: tenant},
	)
	if err != nil {
		return nil, err
	}
	if err = d.access.assertMaySee(ctx, actor, item); err != nil {
		return nil, err
	}

	dismissed := *item
	if !dismissed.IsResolved() {
		now := d.now()
		dismissed.ResolvedAt = &now
	}

	return &services.RecordChange[watchtower.Item]{Before: item, After: &dismissed}, nil
}

func (a kindAccess) assertMaySee(
	ctx context.Context,
	actor *services.RequestActor,
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
