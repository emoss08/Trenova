package workerresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/authctx"
)

func workerFromInput(
	input *gqlmodel.WorkerInput,
	authCtx *authctx.AuthContext,
) (*worker.Worker, error) {
	entity := &worker.Worker{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyWorkerInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyWorkerInput(entity *worker.Worker, input *gqlmodel.WorkerInput) error {
	stateID, err := gqlid.RequiredID("stateId", input.StateID)
	if err != nil {
		return err
	}
	fleetCodeID, err := gqlid.OptionalScopedID("fleetCodeId", input.FleetCodeID)
	if err != nil {
		return err
	}
	managerID, err := gqlid.OptionalScopedID("managerId", input.ManagerID)
	if err != nil {
		return err
	}
	positionID, err := gqlid.OptionalScopedID("positionId", input.PositionID)
	if err != nil {
		return err
	}

	entity.StateID = stateID
	entity.FleetCodeID = fleetCodeID
	entity.ManagerID = managerID
	entity.PositionID = positionID
	entity.Status = input.Status
	entity.Type = input.Type
	entity.DriverType = input.DriverType
	entity.Gender = input.Gender
	entity.FirstName = input.FirstName
	entity.LastName = input.LastName
	entity.AddressLine1 = input.AddressLine1
	entity.AddressLine2 = gqlscalar.StringValue(input.AddressLine2)
	entity.City = input.City
	entity.PostalCode = input.PostalCode
	entity.Email = gqlscalar.StringValue(input.Email)
	entity.PhoneNumber = gqlscalar.StringValue(input.PhoneNumber)
	entity.EmergencyContactName = gqlscalar.StringValue(input.EmergencyContactName)
	entity.EmergencyContactPhone = gqlscalar.StringValue(input.EmergencyContactPhone)
	entity.ExternalID = gqlscalar.StringValue(input.ExternalID)
	entity.CanBeAssigned = input.CanBeAssigned
	entity.AvailableForDispatch = input.AvailableForDispatch
	entity.CustomFields = input.CustomFields
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	if input.Profile == nil {
		return nil
	}
	if entity.Profile == nil {
		entity.Profile = &worker.WorkerProfile{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
		}
	}

	return applyWorkerProfileInput(entity.Profile, input.Profile)
}

func applyWorkerProfileInput(
	profile *worker.WorkerProfile,
	input *gqlmodel.WorkerProfileInput,
) error {
	licenseStateID, err := gqlid.OptionalScopedID("profile.licenseStateId", input.LicenseStateID)
	if err != nil {
		return err
	}

	profile.LicenseStateID = licenseStateID
	profile.DOB = gqlscalar.Int64Value(input.Dob)
	profile.LicenseNumber = gqlscalar.StringValue(input.LicenseNumber)
	profile.CDLClass = input.CdlClass
	profile.CDLRestrictions = gqlscalar.StringValue(input.CdlRestrictions)
	profile.Endorsement = input.Endorsement
	profile.HazmatExpiry = gqlscalar.Int64Ptr(input.HazmatExpiry)
	profile.LicenseExpiry = gqlscalar.Int64Value(input.LicenseExpiry)
	profile.MedicalCardExpiry = gqlscalar.Int64Ptr(input.MedicalCardExpiry)
	profile.MedicalExaminerName = gqlscalar.StringValue(input.MedicalExaminerName)
	profile.MedicalExaminerNPI = gqlscalar.StringValue(input.MedicalExaminerNPI)
	profile.TWICCardNumber = gqlscalar.StringValue(input.TWICCardNumber)
	profile.TWICExpiry = gqlscalar.Int64Ptr(input.TWICExpiry)
	profile.HireDate = gqlscalar.Int64Value(input.HireDate)
	profile.TerminationDate = gqlscalar.Int64Ptr(input.TerminationDate)
	profile.PhysicalDueDate = gqlscalar.Int64Ptr(input.PhysicalDueDate)
	profile.MVRDueDate = gqlscalar.Int64Ptr(input.MVRDueDate)
	profile.ComplianceStatus = input.ComplianceStatus
	profile.IsQualified = input.IsQualified
	profile.DisqualificationReason = gqlscalar.StringValue(input.DisqualificationReason)
	profile.ELDExempt = input.ELDExempt
	profile.ShortHaulExempt = input.ShortHaulExempt

	return nil
}

func applyWorkerPatch(entity *worker.Worker, input *gqlmodel.WorkerPatchInput) error {
	if err := gqlscalar.ApplyRequired("status", "Status", input.Status, &entity.Status); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("type", "Type", input.Type, &entity.Type); err != nil {
		return err
	}

	return gqlscalar.ApplyRequired(
		"driverType",
		"Driver Type",
		input.DriverType,
		&entity.DriverType,
	)
}
