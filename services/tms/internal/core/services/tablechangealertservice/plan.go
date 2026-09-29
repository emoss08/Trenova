package tablechangealertservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ownSubscription(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	return s.subRepo.GetByID(ctx, repositories.GetTCASubscriptionByIDRequest{
		SubscriptionID: id,
		TenantInfo:     tenantInfo,
	})
}

func (s *Service) PlanUpdateSubscription(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*services.RecordChange[tablechangealert.TCASubscription], error) {
	original, err := s.ownSubscription(ctx, entity.ID, pagination.TenantInfo{
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: entity.UserID,
	})
	if err != nil {
		return nil, err
	}

	entity.UserID = original.UserID
	entity.CreatedAt = original.CreatedAt
	if err = s.CheckSubscription(ctx, entity); err != nil {
		return nil, err
	}

	return &services.RecordChange[tablechangealert.TCASubscription]{
		Before: original,
		After:  entity,
	}, nil
}

func (s *Service) PlanSetSubscriptionStatus(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	status tablechangealert.SubscriptionStatus,
) (*services.RecordChange[tablechangealert.TCASubscription], error) {
	original, err := s.ownSubscription(ctx, id, tenantInfo)
	if err != nil {
		return nil, err
	}

	changed := *original
	changed.Status = status
	multiErr := errortypes.NewMultiError()
	changed.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &services.RecordChange[tablechangealert.TCASubscription]{
		Before: original,
		After:  &changed,
	}, nil
}

func (s *Service) PlanDeleteSubscription(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	return s.ownSubscription(ctx, id, tenantInfo)
}
