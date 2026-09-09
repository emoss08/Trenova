package fuelpurchaseservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func cardTenant(card *fuelpurchase.FuelCard) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: card.OrganizationID, BuID: card.BusinessUnitID}
}

func (s *Service) ListCards(
	ctx context.Context,
	req *repositories.ListFuelCardsRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error) {
	return s.repo.ListCards(ctx, req)
}

func (s *Service) ListActiveCards(
	ctx context.Context,
	req *repositories.ListActiveFuelCardsRequest,
) ([]*fuelpurchase.FuelCard, error) {
	return s.repo.ListActiveCards(ctx, req)
}

func (s *Service) GetCard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*fuelpurchase.FuelCard, error) {
	return s.repo.GetCardByID(ctx, &repositories.GetFuelCardByIDRequest{
		ID:                 id,
		TenantInfo:         tenantInfo,
		IncludeAssignments: true,
	})
}

func (s *Service) GetCardsByIDs(
	ctx context.Context,
	req *repositories.GetFuelCardsByIDsRequest,
) ([]*fuelpurchase.FuelCard, error) {
	return s.repo.GetCardsByIDs(ctx, req)
}

func (s *Service) CreateCard(
	ctx context.Context,
	card *fuelpurchase.FuelCard,
	userID pulid.ID,
) (*fuelpurchase.FuelCard, error) {
	card.Normalize()
	if card.Status == fuelpurchase.CardStatusCancelled {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A card cannot be created already cancelled; create it and cancel it with a reason",
		)
	}
	card.CancelledAt = nil
	card.CancelReason = ""

	if s.validator != nil {
		if multiErr := s.validator.ValidateCreate(ctx, card); multiErr != nil {
			return nil, multiErr
		}
	} else if err := validateEntity(card); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateCard(ctx, card)
	if err != nil {
		return nil, err
	}

	tenantInfo := cardTenant(created)
	s.audit(&auditParams{
		resource:   permission.ResourceFuelCard,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    created,
		comment:    "Added " + created.Provider.Label() + " card ending " + created.LastFour,
	})
	s.publish(ctx, tenantInfo, realtimeCard, permission.OpCreate, created.ID, userID)

	return created, nil
}

func (s *Service) UpdateCard(
	ctx context.Context,
	card *fuelpurchase.FuelCard,
	userID pulid.ID,
) (*fuelpurchase.FuelCard, error) {
	tenantInfo := cardTenant(card)
	stored, err := s.repo.GetCardByID(ctx, &repositories.GetFuelCardByIDRequest{
		ID:         card.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if stored.IsCancelled() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A cancelled card cannot be edited",
		)
	}

	card.Normalize()
	if card.Status == fuelpurchase.CardStatusCancelled {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Cancel the card with a reason instead of setting its status",
		)
	}
	if card.Status != stored.Status && !stored.Status.CanTransitionTo(card.Status) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A card that is "+strings.ToLower(stored.Status.Label())+
				" cannot be made "+strings.ToLower(card.Status.Label()),
		)
	}

	card.CancelledAt = nil
	card.CancelReason = ""
	card.CreatedAt = stored.CreatedAt

	if s.validator != nil {
		if multiErr := s.validator.ValidateUpdate(ctx, card); multiErr != nil {
			return nil, multiErr
		}
	} else if vErr := validateEntity(card); vErr != nil {
		return nil, vErr
	}

	previous := *stored
	updated, err := s.repo.UpdateCard(ctx, card)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceFuelCard,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Updated card ending " + updated.LastFour,
	})
	s.publish(ctx, tenantInfo, realtimeCard, permission.OpUpdate, updated.ID, userID)

	return updated, nil
}

type CancelCardRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	Reason     string
	UserID     pulid.ID
}

func (s *Service) CancelCard(
	ctx context.Context,
	req *CancelCardRequest,
) (*fuelpurchase.FuelCard, error) {
	reason := strings.TrimSpace(req.Reason)
	if len(reason) < fuelpurchase.MinCancelReasonLength {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A cancellation reason of at least 10 characters is required",
		)
	}

	card, err := s.repo.GetCardByID(ctx, &repositories.GetFuelCardByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if card.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("FuelCard", card.ID.String())
	}
	if !card.Status.CanTransitionTo(fuelpurchase.CardStatusCancelled) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This card is already cancelled",
		)
	}

	previous := *card
	card.Cancel(s.now(), reason)
	if vErr := validateEntity(card); vErr != nil {
		return nil, vErr
	}

	updated, err := s.repo.UpdateCard(ctx, card)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceFuelCard,
		resourceID: updated.ID.String(),
		operation:  permission.OpCancel,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Cancelled card ending " + updated.LastFour + ": " + reason,
	})
	s.publish(ctx, req.TenantInfo, realtimeCard, permission.OpCancel, updated.ID, req.UserID)

	return updated, nil
}

func validateEntity(entity interface {
	Validate(multiErr *errortypes.MultiError)
}) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
