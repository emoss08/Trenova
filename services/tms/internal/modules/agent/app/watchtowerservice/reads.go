package watchtowerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) readItemIDs(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
	items []*watchtower.Item,
) (map[pulid.ID]struct{}, error) {
	if !actor.IsUser() || len(items) == 0 {
		return map[pulid.ID]struct{}{}, nil
	}

	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return s.repo.ListReadItemIDs(ctx, watchtowerports.ListWatchtowerReadsRequest{
		TenantInfo: tenant,
		UserID:     actor.UserID,
		ItemIDs:    ids,
	})
}

func (s *Service) MarkItemSeen(
	ctx context.Context,
	req watchtowerports.GetWatchtowerItemRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewBusinessError("Only a person has a place in the watchtower")
	}

	item, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = s.access.assertMaySee(ctx, actor, item); err != nil {
		return nil, err
	}

	if err = s.repo.MarkRead(ctx, &watchtower.Read{
		ItemID:         item.ID,
		UserID:         actor.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		SeenAt:         s.now(),
	}); err != nil {
		return nil, err
	}
	item.Seen = true

	return item, nil
}

func (s *Service) Cleared(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
	since int64,
) (int, error) {
	if since <= 0 || since > s.now() {
		return 0, errortypes.NewValidationError(
			"since",
			errortypes.ErrInvalid,
			"Count from a time that has already passed",
		)
	}
	if !actor.IsUser() {
		return 0, nil
	}

	kinds, err := s.access.visibleKinds(ctx, actor, nil)
	if err != nil {
		return 0, err
	}

	return s.repo.CountCleared(ctx, watchtowerports.CountClearedWatchtowerItemsRequest{
		TenantInfo: tenant,
		Kinds:      kinds,
		UserID:     actor.UserID,
		Since:      since,
	})
}
