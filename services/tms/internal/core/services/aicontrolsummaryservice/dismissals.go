package aicontrolsummaryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// VisibleFailures are the failing providers a person has not put away, or
// has put away before the failure they would now see.
func (s *Service) VisibleFailures(
	ctx context.Context,
	req *services.VisibleFailuresRequest,
) ([]aicontrolsummary.ProviderFailure, error) {
	if len(req.Failing) == 0 {
		return []aicontrolsummary.ProviderFailure{}, nil
	}

	dismissals, err := s.dismissals.List(ctx, &repositories.ListFailureDismissalsRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
	})
	if err != nil {
		return nil, err
	}

	byProvider := make(map[string]*aiprovider.FailureDismissal, len(dismissals))
	for _, dismissal := range dismissals {
		byProvider[dismissal.ProviderID.String()] = dismissal
	}

	visible := make([]aicontrolsummary.ProviderFailure, 0, len(req.Failing))
	for _, failure := range req.Failing {
		if byProvider[failure.ProviderID.String()].Covers(failure.LastFailureAt) {
			continue
		}
		visible = append(visible, failure)
	}
	return visible, nil
}

// DismissFailure puts away the notice of a provider's failure for one person,
// up to the failure they saw.
func (s *Service) DismissFailure(ctx context.Context, req *services.ProviderFailureDismissal) error {
	if req.UserID.IsNil() {
		return errortypes.NewBusinessError("Only a person can put a notice away")
	}
	if _, err := s.providers.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         req.ProviderID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return err
	}

	return s.dismissals.Upsert(ctx, &aiprovider.FailureDismissal{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		UserID:         req.UserID,
		ProviderID:     req.ProviderID,
		FailureAt:      req.FailureAt,
	})
}

// RestoreFailure brings a dismissed notice back, for the undo after a dismissal.
func (s *Service) RestoreFailure(ctx context.Context, req *services.ProviderFailureDismissal) error {
	return s.dismissals.Delete(ctx, &repositories.DeleteFailureDismissalRequest{
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
		ProviderID: req.ProviderID,
	})
}
