package customertrackingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// GetProfile returns the customer's tracking settings, or the defaults a
// customer has until someone changes them. Defaults carry no id.
func (s *Service) GetProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*customertracking.Profile, error) {
	if customerID.IsNil() {
		return nil, errortypes.NewValidationError(
			"customerId",
			errortypes.ErrRequired,
			"Customer ID is required",
		)
	}
	return s.profileOrDefault(ctx, tenantInfo, customerID)
}

func (s *Service) profileOrDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*customertracking.Profile, error) {
	profile, err := s.profiles.GetByCustomer(ctx, tenantInfo, customerID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return customertracking.DefaultProfile(tenantInfo.OrgID, tenantInfo.BuID, customerID), nil
	}
	return profile, nil
}

func (s *Service) UpdateProfile(
	ctx context.Context,
	req *customertrackingports.UpdateProfileRequest,
) (*customertracking.Profile, error) {
	if req == nil || req.Profile == nil || req.Profile.CustomerID.IsNil() {
		return nil, errortypes.NewValidationError(
			"customerId",
			errortypes.ErrRequired,
			"Customer ID is required",
		)
	}
	if _, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         req.Profile.CustomerID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	previous, err := s.profiles.GetByCustomer(ctx, req.TenantInfo, req.Profile.CustomerID)
	if err != nil {
		return nil, err
	}
	if previous != nil && req.ExpectedVersion != nil && *req.ExpectedVersion != previous.Version {
		return nil, errortypes.NewBusinessError(
			"These tracking settings changed since you opened them; refresh and try again",
		)
	}

	now := s.now()
	next := *req.Profile
	next.OrganizationID = req.TenantInfo.OrgID
	next.BusinessUnitID = req.TenantInfo.BuID
	next.Normalize(previous, now)
	multiErr := errortypes.NewMultiError()
	next.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	next.UpdatedAt = now
	if previous == nil {
		next.ID = customertracking.NewProfileID()
		next.Version = 0
		next.CreatedAt = now
		err = s.profiles.Create(ctx, &next)
	} else {
		next.ID = previous.ID
		next.Version = previous.Version
		next.CreatedAt = previous.CreatedAt
		err = s.profiles.Update(ctx, &next)
	}
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		tenant:     req.TenantInfo,
		userID:     req.UpdatedByID,
		resource:   permission.ResourceCustomer,
		resourceID: next.CustomerID.String(),
		operation:  permission.OpUpdate,
		comment:    "Changed the customer's tracking link and status email settings",
		previous:   profileState(previous),
		current:    profileState(&next),
	})
	return &next, nil
}

func profileState(p *customertracking.Profile) map[string]any {
	if p == nil {
		return nil
	}
	return map[string]any{
		"linkEnabled":     p.LinkEnabled,
		"visibleFields":   p.VisibleFields,
		"linkTtlDays":     p.LinkTTLDays,
		"notifyPickedUp":  p.NotifyPickedUp,
		"notifyDelayed":   p.NotifyDelayed,
		"notifyDelivered": p.NotifyDelivered,
		"notifyPodReady":  p.NotifyPodReady,
		"includeLink":     p.IncludeLink,
		"recipients":      p.Recipients,
	}
}
