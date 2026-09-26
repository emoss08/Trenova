package captureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

// DeviceIdentity is the device as its own companion sees it: the record, the
// person and organization it acts for, named so the tray can say who it is
// signed in as, and how the organization wants it kept up to date. It carries
// names only, never anything else about the person.
type DeviceIdentity struct {
	Device       *capture.CaptureDevice `json:"device"`
	Person       DevicePerson           `json:"person"`
	Organization DeviceOrganization     `json:"organization"`
	Updates      DeviceUpdatePolicy     `json:"updates"`
}

// DeviceUpdatePolicy is the organization's say over the companion's version.
type DeviceUpdatePolicy struct {
	// MinimumVersion is the oldest companion that may refresh its token;
	// empty when there is none.
	MinimumVersion string `json:"minimumVersion"`
	// AllowAutoUpdate is whether the companion installs a new release by
	// itself. When it is off, IT deploys new versions.
	AllowAutoUpdate bool `json:"allowAutoUpdate"`
}

type DevicePerson struct {
	ID           pulid.ID `json:"id"`
	Name         string   `json:"name"`
	EmailAddress string   `json:"emailAddress"`
}

type DeviceOrganization struct {
	ID   pulid.ID `json:"id"`
	Name string   `json:"name"`
}

// DescribeDevice names the device's person and organization, and the
// organization's update policy.
func (s *Service) DescribeDevice(
	ctx context.Context,
	principal *DevicePrincipal,
) (*DeviceIdentity, error) {
	tenantInfo := principal.TenantInfo()

	person, err := s.users.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   tenantInfo,
		LookupUserID: tenantInfo.UserID,
	})
	if err != nil {
		return nil, err
	}

	org, err := s.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	control, err := s.control(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return &DeviceIdentity{
		Device: principal.Device,
		Person: DevicePerson{
			ID:           person.ID,
			Name:         person.Name,
			EmailAddress: person.EmailAddress,
		},
		Organization: DeviceOrganization{ID: org.ID, Name: org.Name},
		Updates: DeviceUpdatePolicy{
			MinimumVersion:  control.CaptureMinAgentVersion,
			AllowAutoUpdate: control.CaptureAllowAutoUpdate,
		},
	}, nil
}

// DeviceProfiles are the scan profiles the device's person may scan with, for
// the tray's own "Scan to intake". It is the same list the web app offers.
func (s *Service) DeviceProfiles(
	ctx context.Context,
	principal *DevicePrincipal,
) ([]*capture.CaptureProfile, error) {
	if _, err := s.requireEnabled(ctx, principal.TenantInfo()); err != nil {
		return nil, err
	}

	return s.AvailableProfiles(ctx, principal.TenantInfo())
}
