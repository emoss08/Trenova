package tenantprovisioningservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Config *config.Config
	Repo   repositories.TenantProvisioningRepository
}

type Service struct {
	cfg  *config.Config
	repo repositories.TenantProvisioningRepository
	now  func() time.Time
}

func New(p Params) services.TenantProvisioningService {
	return &Service{
		cfg:  p.Config,
		repo: p.Repo,
		now:  time.Now,
	}
}

func (s *Service) ProvisionTenant(
	ctx context.Context,
	req *services.TenantProvisioningRequest,
) (*services.TenantProvisioningResult, error) {
	if err := s.validate(req); err != nil {
		return nil, err
	}

	result, err := s.repo.UpsertProvisioningSnapshot(ctx, req)
	if err != nil {
		return nil, err
	}

	result.Accepted = true
	result.ReceivedAt = s.now().Unix()

	return result, nil
}

func (s *Service) validate(req *services.TenantProvisioningRequest) error {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("payload", errortypes.ErrRequired, "Provisioning payload is required")
		return multiErr
	}

	if strings.TrimSpace(s.cfg.Platform.InstanceID) != "" &&
		req.InstanceID != s.cfg.Platform.InstanceID {
		multiErr.Add(
			"instanceId",
			errortypes.ErrInvalid,
			"Provisioning payload is not targeted to this instance",
		)
	}
	if strings.TrimSpace(req.InstanceID) == "" {
		multiErr.Add("instanceId", errortypes.ErrRequired, "Instance ID is required")
	}
	validateCustomer(multiErr, req.Customer)
	validateWorkspace(multiErr, req.Workspace)
	if req.Customer.ID.IsNotNil() &&
		req.Workspace.BusinessUnitID.IsNotNil() &&
		req.Customer.ID != req.Workspace.BusinessUnitID {
		multiErr.Add(
			"workspace.businessUnitId",
			errortypes.ErrInvalid,
			"Workspace business unit ID must match the customer ID",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func validateCustomer(multiErr *errortypes.MultiError, customer services.TenantProvisioningCustomer) {
	if customer.ID.IsNil() {
		multiErr.Add("customer.id", errortypes.ErrRequired, "Customer ID is required")
	}
	if strings.TrimSpace(customer.Name) == "" {
		multiErr.Add("customer.name", errortypes.ErrRequired, "Customer name is required")
	}
	if strings.TrimSpace(customer.Code) == "" {
		multiErr.Add("customer.code", errortypes.ErrRequired, "Customer code is required")
	}
}

func validateWorkspace(multiErr *errortypes.MultiError, workspace services.TenantProvisioningWorkspace) {
	if workspace.ID.IsNil() {
		multiErr.Add("workspace.id", errortypes.ErrRequired, "Workspace ID is required")
	}
	if workspace.BusinessUnitID.IsNil() {
		multiErr.Add(
			"workspace.businessUnitId",
			errortypes.ErrRequired,
			"Workspace business unit ID is required",
		)
	}
	if strings.TrimSpace(workspace.Name) == "" {
		multiErr.Add("workspace.name", errortypes.ErrRequired, "Workspace name is required")
	}
	if workspace.StateID.IsNil() && strings.TrimSpace(workspace.State) == "" {
		multiErr.Add("workspace.state", errortypes.ErrRequired, "Workspace state is required")
	}
	if strings.TrimSpace(workspace.AddressLine1) == "" {
		multiErr.Add("workspace.addressLine1", errortypes.ErrRequired, "Address line 1 is required")
	}
	if strings.TrimSpace(workspace.City) == "" {
		multiErr.Add("workspace.city", errortypes.ErrRequired, "City is required")
	}
	if strings.TrimSpace(workspace.PostalCode) == "" {
		multiErr.Add("workspace.postalCode", errortypes.ErrRequired, "Postal code is required")
	}
	if strings.TrimSpace(workspace.Timezone) == "" {
		multiErr.Add("workspace.timezone", errortypes.ErrRequired, "Timezone is required")
	} else if err := domainvalidation.ValidateTimezone(workspace.Timezone); err != nil {
		multiErr.Add("workspace.timezone", errortypes.ErrInvalid, "Timezone is invalid")
	}
	if strings.TrimSpace(workspace.BucketName) == "" {
		multiErr.Add("workspace.bucketName", errortypes.ErrRequired, "Bucket name is required")
	}
	if strings.TrimSpace(workspace.TaxID) == "" {
		multiErr.Add("workspace.taxId", errortypes.ErrRequired, "Tax ID is required")
	}
	if strings.TrimSpace(workspace.ScacCode) == "" {
		multiErr.Add("workspace.scacCode", errortypes.ErrRequired, "SCAC code is required")
	} else if len(strings.TrimSpace(workspace.ScacCode)) != 4 {
		multiErr.Add("workspace.scacCode", errortypes.ErrInvalid, "SCAC code must be 4 characters")
	}
	if strings.TrimSpace(workspace.DOTNumber) == "" {
		multiErr.Add("workspace.dotNumber", errortypes.ErrRequired, "DOT number is required")
	} else if !isDOTNumber(workspace.DOTNumber) {
		multiErr.Add(
			"workspace.dotNumber",
			errortypes.ErrInvalid,
			"DOT number must contain 1 to 8 digits",
		)
	}
	if strings.TrimSpace(workspace.LoginSlug) == "" {
		multiErr.Add("workspace.loginSlug", errortypes.ErrRequired, "Login slug is required")
	}
}

func isDOTNumber(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > 8 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
