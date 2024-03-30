package services

import (
	"context"

	"github.com/emoss08/trenova/ent/worker"
	"github.com/emoss08/trenova/ent/workerprofile"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type WorkerOps struct {
	ctx    context.Context
	client *ent.Client
}

type WorkerRequest struct {
	BusinessUnitID    uuid.UUID         `json:"businessUnitId"`
	OrganizationID    uuid.UUID         `json:"organizationId"`
	Status            worker.Status     `json:"status" validate:"required,oneof=A I"`
	Code              string            `json:"code" validate:"required,max=10"`
	ProfilePictureURL string            `json:"profilePictureUrl"`
	WorkerType        worker.WorkerType `json:"workerType" validate:"required,oneof=Employee Contractor"`
	FirstName         string            `json:"firstName" validate:"required,max=255"`
	LastName          string            `json:"lastName" validate:"required,max=255"`
	City              string            `json:"city" validate:"omitempty,max=255"`
	PostalCode        string            `json:"postalCode" validate:"omitempty,max=10"`
	StateID           *uuid.UUID        `json:"stateId" validate:"omitempty,uuid"`
	FleetCodeID       *uuid.UUID        `json:"fleetCodeId" validate:"omitempty,uuid"`
	ManagerID         *uuid.UUID        `json:"managerId" validate:"omitempty,uuid"`
	Profile           WorkerProfileRequest
	Comments          []WorkerCommentRequest `json:"comments" validate:"omitempty,dive"`
	Contacts          []WorkerContactRequest `json:"contacts" validate:"omitempty,dive"`
}

type WorkerUpdateRequest struct {
	ID                uuid.UUID         `json:"id,omitempty"`
	BusinessUnitID    uuid.UUID         `json:"businessUnitId"`
	OrganizationID    uuid.UUID         `json:"organizationId"`
	Status            worker.Status     `json:"status" validate:"required,oneof=A I"`
	Code              string            `json:"code" validate:"required,max=10"`
	ProfilePictureURL string            `json:"profilePictureUrl"`
	WorkerType        worker.WorkerType `json:"workerType" validate:"required,oneof=Employee Contractor"`
	FirstName         string            `json:"firstName" validate:"required,max=255"`
	LastName          string            `json:"lastName" validate:"required,max=255"`
	City              string            `json:"city" validate:"omitempty,max=255"`
	PostalCode        string            `json:"postalCode" validate:"omitempty,max=10"`
	StateID           uuid.UUID         `json:"stateId" validate:"omitempty,uuid"`
	FleetCodeID       *uuid.UUID        `json:"fleetCodeId" validate:"omitempty,uuid"`
	ManagerID         *uuid.UUID        `json:"managerId" validate:"omitempty,uuid"`
	Profile           WorkerProfileRequest
}

type WorkerProfileRequest struct {
	WorkerID              uuid.UUID                  `json:"workerId" validate:"required,uuid"`
	Race                  string                     `json:"race" validate:"omitempty"`
	Sex                   string                     `json:"sex,omitempty" validate:"omitempty"`
	DateOfBirth           *pgtype.Date               `json:"dateOfBirth" validate:"omitempty"`
	LicenseNumber         string                     `json:"licenseNumber" validate:"required"`
	LicenseStateID        uuid.UUID                  `json:"licenseStateId" validate:"omitempty,uuid"`
	LicenseExpirationDate *pgtype.Date               `json:"licenseExpirationDate" validate:"omitempty"`
	Endorsements          workerprofile.Endorsements `json:"endorsements" validate:"omitempty"`
	HazmatExpirationDate  *pgtype.Date               `json:"hazmatExpirationDate" validate:"omitempty"`
	HireDate              *pgtype.Date               `json:"hireDate" validate:"omitempty"`
	TerminationDate       *pgtype.Date               `json:"terminationDate" validate:"omitempty"`
	PhysicalDueDate       *pgtype.Date               `json:"physicalDueDate" validate:"omitempty"`
	MedicalCertDate       *pgtype.Date               `json:"medicalCertDate" validate:"omitempty"`
	MvrDueDate            *pgtype.Date               `json:"mvrDueDate" validate:"omitempty"`
}

type WorkerCommentRequest struct {
	WorkerID      uuid.UUID `json:"workerId" validate:"required,uuid"`
	CommentTypeID uuid.UUID `json:"commentTypeId" validate:"required"`
	Comment       string    `json:"comment" validate:"omitempty"`
	EnteredBy     uuid.UUID `json:"enteredBy" validate:"required"`
}

type WorkerContactRequest struct {
	WorkerID     uuid.UUID `json:"workerId" validate:"required"`
	Name         string    `json:"name" validate:"required"`
	Email        string    `json:"email" validate:"required"`
	Phone        string    `json:"phone" validate:"required"`
	Relationship string    `json:"relationship" validate:"omitempty"`
	IsPrimary    bool      `json:"isPrimary" validate:"omitempty"`
}

// NewWorkerOps creates a new tractor service.
func NewWorkerOps(ctx context.Context) *WorkerOps {
	return &WorkerOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetWorkers gets the workers for an organization.
func (r *WorkerOps) GetWorkers(limit, offset int, orgID, buID uuid.UUID) ([]*ent.Worker, int, error) {
	entityCount, countErr := r.client.Worker.Query().Where(
		worker.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.Worker.Query().
		Limit(limit).
		Offset(offset).
		Where(
			worker.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateWorker creates a new worker.
func (r *WorkerOps) CreateWorker(entity WorkerRequest) (*ent.Worker, error) {
	newEntity, err := r.client.Worker.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetProfilePictureURL(entity.ProfilePictureURL).
		SetWorkerType(entity.WorkerType).
		SetFirstName(entity.FirstName).
		SetLastName(entity.LastName).
		SetCity(entity.City).
		SetPostalCode(entity.PostalCode).
		SetNillableStateID(entity.StateID).
		SetNillableFleetCodeID(entity.FleetCodeID).
		SetNillableManagerID(entity.ManagerID).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	// Create the worker profile
	_, err = r.client.WorkerProfile.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetWorkerID(newEntity.ID).
		SetWorker(newEntity).
		SetRace(entity.Profile.Race).
		SetSex(entity.Profile.Sex).
		SetDateOfBirth(entity.Profile.DateOfBirth).
		SetLicenseNumber(entity.Profile.LicenseNumber).
		SetLicenseStateID(entity.Profile.LicenseStateID).
		SetLicenseExpirationDate(entity.Profile.LicenseExpirationDate).
		SetEndorsements(entity.Profile.Endorsements).
		SetHazmatExpirationDate(entity.Profile.HazmatExpirationDate).
		SetHireDate(entity.Profile.HireDate).
		SetTerminationDate(entity.Profile.TerminationDate).
		SetPhysicalDueDate(entity.Profile.PhysicalDueDate).
		SetMedicalCertDate(entity.Profile.MedicalCertDate).
		SetMvrDueDate(entity.Profile.MvrDueDate).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	// Create the worker comments
	for _, comment := range entity.Comments {
		_, err = r.client.WorkerComment.Create().
			SetOrganizationID(entity.OrganizationID).
			SetBusinessUnitID(entity.BusinessUnitID).
			SetWorkerID(newEntity.ID).
			SetWorker(newEntity).
			SetCommentTypeID(comment.CommentTypeID).
			SetComment(comment.Comment).
			SetEnteredBy(comment.EnteredBy).
			Save(r.ctx)
		if err != nil {
			return nil, err
		}
	}

	// Create the worker contacts
	for _, contact := range entity.Contacts {
		_, err = r.client.WorkerContact.Create().
			SetWorkerID(newEntity.ID).
			SetWorker(newEntity).
			SetName(contact.Name).
			SetEmail(contact.Email).
			SetPhone(contact.Phone).
			SetRelationship(contact.Relationship).
			SetIsPrimary(contact.IsPrimary).
			Save(r.ctx)
		if err != nil {
			return nil, err
		}
	}

	return newEntity, nil
}

// UpdateWorker updates a worker.
func (r *WorkerOps) UpdateWorker(entity WorkerUpdateRequest) (*ent.Worker, error) {
	// Start building the update operation
	updateOp := r.client.Worker.UpdateOneID(entity.ID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetProfilePictureURL(entity.ProfilePictureURL).
		SetWorkerType(entity.WorkerType).
		SetFirstName(entity.FirstName).
		SetLastName(entity.LastName).
		SetCity(entity.City).
		SetPostalCode(entity.PostalCode).
		SetStateID(entity.StateID).
		SetNillableFleetCodeID(entity.FleetCodeID).
		SetNillableManagerID(entity.ManagerID)

	// If the worker profile is not nil, update the worker profile.
	if entity.Profile.WorkerID != uuid.Nil {
		_, err := r.client.WorkerProfile.UpdateOneID(entity.Profile.WorkerID).
			SetRace(entity.Profile.Race).
			SetSex(entity.Profile.Sex).
			SetDateOfBirth(entity.Profile.DateOfBirth).
			SetLicenseNumber(entity.Profile.LicenseNumber).
			SetLicenseStateID(entity.Profile.LicenseStateID).
			SetLicenseExpirationDate(entity.Profile.LicenseExpirationDate).
			SetEndorsements(entity.Profile.Endorsements).
			SetHazmatExpirationDate(entity.Profile.HazmatExpirationDate).
			SetHireDate(entity.Profile.HireDate).
			SetTerminationDate(entity.Profile.TerminationDate).
			SetPhysicalDueDate(entity.Profile.PhysicalDueDate).
			SetMedicalCertDate(entity.Profile.MedicalCertDate).
			SetMvrDueDate(entity.Profile.MvrDueDate).
			Save(r.ctx)
		if err != nil {
			return nil, err
		}
	}

	// If the worker manager is nil, clear the assocation.
	if entity.ManagerID == nil {
		updateOp = updateOp.ClearManager()
	}

	// If the fleet code ID is nil, clear the association.
	if entity.FleetCodeID == nil {
		updateOp = updateOp.ClearFleetCode()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
