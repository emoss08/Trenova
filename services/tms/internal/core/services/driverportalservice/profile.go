package driverportalservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PortalComplianceProfile struct {
	WorkerID              pulid.ID `json:"workerId"`
	LicenseNumber         string   `json:"licenseNumber"`
	LicenseState          string   `json:"licenseState"`
	CDLClass              string   `json:"cdlClass"`
	Endorsement           string   `json:"endorsement"`
	LicenseExpiry         int64    `json:"licenseExpiry"`
	HazmatExpiry          *int64   `json:"hazmatExpiry"`
	MedicalCardExpiry     *int64   `json:"medicalCardExpiry"`
	PhysicalDueDate       *int64   `json:"physicalDueDate"`
	MVRDueDate            *int64   `json:"mvrDueDate"`
	TWICExpiry            *int64   `json:"twicExpiry"`
	ComplianceStatus      string   `json:"complianceStatus"`
	IsQualified           bool     `json:"isQualified"`
	HireDate              int64    `json:"hireDate"`
	AddressLine1          string   `json:"addressLine1"`
	AddressLine2          string   `json:"addressLine2"`
	City                  string   `json:"city"`
	StateAbbreviation     string   `json:"stateAbbreviation"`
	PostalCode            string   `json:"postalCode"`
	PhoneNumber           string   `json:"phoneNumber"`
	EmergencyContactName  string   `json:"emergencyContactName"`
	EmergencyContactPhone string   `json:"emergencyContactPhone"`
}

func (s *Service) MyComplianceProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PortalComplianceProfile, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	view := &PortalComplianceProfile{
		WorkerID:              wrk.ID,
		AddressLine1:          wrk.AddressLine1,
		AddressLine2:          wrk.AddressLine2,
		City:                  wrk.City,
		PostalCode:            wrk.PostalCode,
		PhoneNumber:           wrk.PhoneNumber,
		EmergencyContactName:  wrk.EmergencyContactName,
		EmergencyContactPhone: wrk.EmergencyContactPhone,
	}
	if wrk.State != nil {
		view.StateAbbreviation = wrk.State.Abbreviation
	}
	if profile := wrk.Profile; profile != nil {
		view.LicenseNumber = profile.LicenseNumber
		view.CDLClass = string(profile.CDLClass)
		view.Endorsement = string(profile.Endorsement)
		view.LicenseExpiry = profile.LicenseExpiry
		view.HazmatExpiry = profile.HazmatExpiry
		view.MedicalCardExpiry = profile.MedicalCardExpiry
		view.PhysicalDueDate = profile.PhysicalDueDate
		view.MVRDueDate = profile.MVRDueDate
		view.TWICExpiry = profile.TWICExpiry
		view.ComplianceStatus = string(profile.ComplianceStatus)
		view.IsQualified = profile.IsQualified
		view.HireDate = profile.HireDate
		if profile.LicenseState != nil {
			view.LicenseState = profile.LicenseState.Abbreviation
		}
	}
	return view, nil
}

type UpdateMyContactInfoRequest struct {
	PhoneNumber           string
	AddressLine1          string
	AddressLine2          string
	City                  string
	PostalCode            string
	EmergencyContactName  string
	EmergencyContactPhone string
}

// UpdateMyContactInfo lets a driver keep their own contact details current.
// Compliance fields (license, medical, endorsements) stay carrier-controlled
// and are intentionally not updatable from the portal.
func (s *Service) UpdateMyContactInfo(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *UpdateMyContactInfoRequest,
	actor *serviceports.RequestActor,
) (*PortalComplianceProfile, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Contact details are required",
		)
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.requireFeature(ctx, tenantInfo,
		func(control *tenant.DashControl) bool { return control.AllowContactInfoEdit },
		"Your carrier manages contact details in the office — ask them to update your record.",
	); err != nil {
		return nil, err
	}

	wrk.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	wrk.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	wrk.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	wrk.City = strings.TrimSpace(req.City)
	wrk.PostalCode = strings.TrimSpace(req.PostalCode)
	wrk.EmergencyContactName = strings.TrimSpace(req.EmergencyContactName)
	wrk.EmergencyContactPhone = strings.TrimSpace(req.EmergencyContactPhone)

	if _, err = s.workerService.Update(ctx, wrk, actor); err != nil {
		return nil, err
	}
	return s.MyComplianceProfile(ctx, tenantInfo)
}
