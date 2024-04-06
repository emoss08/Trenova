package services

import (
	"context"

	"github.com/emoss08/trenova/ent/worker"
	"github.com/emoss08/trenova/ent/workercomment"
	"github.com/emoss08/trenova/ent/workercontact"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/logger"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

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
	Version           int               `json:"version" validate:"omitempty"`
	FleetCodeID       *uuid.UUID        `json:"fleetCodeId" validate:"omitempty,uuid"`
	ManagerID         *uuid.UUID        `json:"managerId" validate:"omitempty,uuid"`
	Profile           ent.WorkerProfile
	Comments          []ent.WorkerComment `json:"comments" validate:"omitempty,dive"`
	Contacts          []ent.WorkerContact `json:"contacts" validate:"omitempty,dive"`
}

type WorkerUpdateRequest struct {
	ID uuid.UUID `json:"id,omitempty"`
	WorkerRequest
}

type WorkerOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewWorkerOps creates a new tractor service.
func NewWorkerOps() *WorkerOps {
	return &WorkerOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetWorkers gets the workers for an organization.
func (r *WorkerOps) GetWorkers(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID, fleetCodeID uuid.UUID,
) ([]*ent.Worker, int, error) {
	query := r.client.Worker.Query().Where(
		worker.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitID(buID),
		),
	)

	// Add conditions based on query params.
	if fleetCodeID != uuid.Nil {
		query = query.Where(
			worker.FleetCodeID(fleetCodeID),
		)
	}

	entityCount, countErr := query.Count(ctx)
	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := query.Limit(limit).Offset(offset).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateWorker creates a new worker.
func (r *WorkerOps) CreateWorker(ctx context.Context, newEntity WorkerRequest) (*ent.Worker, error) {
	var createdEntity *ent.Worker

	err := tools.WithTx(ctx, r.client, func(tx *ent.Tx) error {
		var err error
		createdEntity, err = r.createWorkerEntity(ctx, tx, newEntity)
		if err != nil {
			return err
		}

		// Create the profile for the worker.
		if err = r.createWorkerProfileEntity(ctx, tx, createdEntity.ID, newEntity); err != nil {
			return err
		}

		// If comments are provided, create them and associate them with the location.
		if len(newEntity.Comments) > 0 {
			if err = r.createWorkerComments(ctx, tx, createdEntity.ID, newEntity); err != nil {
				return nil
			}
		}

		// If contacts are provided, create them and associate them with the location.
		if len(newEntity.Contacts) > 0 {
			if err = r.createWorkerContacts(ctx, tx, createdEntity.ID, newEntity); err != nil {
				return nil
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

func (r *WorkerOps) createWorkerEntity(
	ctx context.Context, tx *ent.Tx, newEntity WorkerRequest,
) (*ent.Worker, error) {
	return tx.Worker.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetCode(newEntity.Code).
		SetStatus(newEntity.Status).
		SetProfilePictureURL(newEntity.ProfilePictureURL).
		SetWorkerType(newEntity.WorkerType).
		SetFirstName(newEntity.FirstName).
		SetLastName(newEntity.LastName).
		SetCity(newEntity.City).
		SetPostalCode(newEntity.PostalCode).
		SetNillableStateID(newEntity.StateID).
		SetNillableFleetCodeID(newEntity.FleetCodeID).
		SetNillableManagerID(newEntity.ManagerID).
		Save(ctx)
}

func (r *WorkerOps) createWorkerProfileEntity(
	ctx context.Context, tx *ent.Tx, workerID uuid.UUID, newEntity WorkerRequest,
) error {
	return tx.WorkerProfile.Create().
		SetWorkerID(workerID).
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetRace(newEntity.Profile.Race).
		SetSex(newEntity.Profile.Sex).
		SetDateOfBirth(newEntity.Profile.DateOfBirth).
		SetLicenseNumber(newEntity.Profile.LicenseNumber).
		SetLicenseStateID(newEntity.Profile.LicenseStateID).
		SetLicenseExpirationDate(newEntity.Profile.LicenseExpirationDate).
		SetEndorsements(newEntity.Profile.Endorsements).
		SetHazmatExpirationDate(newEntity.Profile.HazmatExpirationDate).
		SetHireDate(newEntity.Profile.HireDate).
		SetTerminationDate(newEntity.Profile.TerminationDate).
		SetPhysicalDueDate(newEntity.Profile.PhysicalDueDate).
		SetMedicalCertDate(newEntity.Profile.MedicalCertDate).
		SetMvrDueDate(newEntity.Profile.MvrDueDate).
		Exec(ctx)
}

func (r *WorkerOps) createWorkerComments(
	ctx context.Context, tx *ent.Tx, workerID uuid.UUID, newEntity WorkerRequest,
) error {
	for _, comment := range newEntity.Comments {
		_, err := r.client.WorkerComment.Create().
			SetOrganizationID(newEntity.OrganizationID).
			SetBusinessUnitID(newEntity.BusinessUnitID).
			SetWorkerID(workerID).
			SetCommentTypeID(comment.CommentTypeID).
			SetComment(comment.Comment).
			SetUserID(comment.UserID).
			Save(ctx)
		if err != nil {
			wrappedErr := eris.Wrap(err, "failed to create worker comment")
			r.logger.WithField("error", wrappedErr).Error("failed to create worker comment")
			return wrappedErr
		}
	}

	return nil
}

func (r *WorkerOps) createWorkerContacts(
	ctx context.Context, tx *ent.Tx, workerID uuid.UUID, newEntity WorkerRequest,
) error {
	for _, contact := range newEntity.Contacts {
		_, err := r.client.WorkerContact.Create().
			SetWorkerID(workerID).
			SetName(contact.Name).
			SetEmail(contact.Email).
			SetPhone(contact.Phone).
			SetRelationship(contact.Relationship).
			SetIsPrimary(contact.IsPrimary).
			Save(ctx)
		if err != nil {
			wrappedErr := eris.Wrap(err, "failed to create worker contact")
			r.logger.WithField("error", wrappedErr).Error("failed to create worker contact")
			return wrappedErr
		}
	}

	return nil
}

// UpdateWorker updates a worker.
func (r *WorkerOps) UpdateWorker(ctx context.Context, entity WorkerUpdateRequest) (*ent.Worker, error) {
	var updatedEntity *ent.Worker

	err := tools.WithTx(ctx, r.client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateWorkerEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		// Update the profile ID if it is not nil
		if entity.Profile.WorkerID != uuid.Nil {
			_, err = r.updateWorkerProfileEntity(ctx, tx, entity)
			if err != nil {
				return err
			}
		}

		// If the worker manager is nil, clear the association.
		if entity.ManagerID == nil {
			updatedEntity.Update().ClearManager()
		}
		// If the fleet code ID is nil, clear the association.
		if entity.FleetCodeID == nil {
			updatedEntity.Update().ClearFleetCode()
		}

		if err = r.syncContacts(ctx, tx, entity, updatedEntity); err != nil {
			return err
		}

		return r.syncComments(ctx, tx, entity, updatedEntity)
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *WorkerOps) updateWorkerEntity(
	ctx context.Context, tx *ent.Tx, entity WorkerUpdateRequest,
) (*ent.Worker, error) {
	current, err := tx.Worker.Get(ctx, entity.ID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to retrieve requested entity")
	}

	if current.Version != entity.Version {
		return nil, tools.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	return tx.Worker.UpdateOneID(entity.ID).
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
		SetVersion(entity.Version + 1). // Increment the version
		Save(ctx)
}

func (r *WorkerOps) updateWorkerProfileEntity(
	ctx context.Context, tx *ent.Tx, entity WorkerUpdateRequest,
) (*ent.WorkerProfile, error) {
	current, err := tx.WorkerProfile.Get(ctx, entity.Profile.ID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to retrieve requested entity")
	}

	if current.Version != entity.Version {
		return nil, tools.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	return tx.WorkerProfile.UpdateOneID(entity.Profile.WorkerID).
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
		SetVersion(entity.Profile.Version + 1). // Increment the version
		Save(ctx)
}

func (r *WorkerOps) syncComments(
	ctx context.Context, tx *ent.Tx, entity WorkerUpdateRequest, updatedEntity *ent.Worker,
) error {
	existingComments, err := tx.Worker.QueryWorkerComments(updatedEntity).Where(
		workercomment.HasWorkerWith(worker.IDEQ(entity.ID)),
	).All(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to fetch existing comments")
	}

	// Delete unmatched comments
	if err = r.deleteUnmatchedComments(ctx, tx, entity, existingComments); err != nil {
		return err
	}

	// Update or create new comments
	return r.updateOrCreateComments(ctx, tx, entity)
}

func (r *WorkerOps) deleteUnmatchedComments(
	ctx context.Context, tx *ent.Tx, entity WorkerUpdateRequest, existingComments []*ent.WorkerComment,
) error {
	commentPresnet := make(map[uuid.UUID]bool)
	for _, comment := range entity.Comments {
		if comment.ID != uuid.Nil {
			commentPresnet[comment.ID] = true
		}
	}

	for _, existingComment := range existingComments {
		if !commentPresnet[existingComment.ID] {
			if err := tx.WorkerComment.DeleteOneID(existingComment.ID).Exec(ctx); err != nil {
				wrappedErr := eris.Wrap(err, "failed to delete worker comment")
				r.logger.WithField("error", wrappedErr).Error("failed to delete worker comment")
				return wrappedErr
			}
		}
	}

	return nil
}

func (r *WorkerOps) updateOrCreateComments(ctx context.Context, tx *ent.Tx, entity WorkerUpdateRequest) error {
	for _, comment := range entity.Comments {
		if comment.ID != uuid.Nil {
			if _, err := tx.WorkerComment.UpdateOneID(comment.ID).
				SetCommentTypeID(comment.CommentTypeID).
				SetComment(comment.Comment).
				SetVersion(comment.Version + 1). // Increment the version
				SetUserID(comment.UserID).
				Save(ctx); err != nil {
				wrappedErr := eris.Wrap(err, "failed to update worker comment")
				r.logger.WithField("error", wrappedErr).Error("failed to update worker comment")
				return wrappedErr
			}
		} else {
			if _, err := tx.WorkerComment.Create().
				SetWorkerID(comment.WorkerID).
				SetOrganizationID(comment.OrganizationID).
				SetBusinessUnitID(comment.BusinessUnitID).
				SetCommentTypeID(comment.CommentTypeID).
				SetUserID(comment.UserID).
				SetComment(comment.Comment).
				Save(ctx); err != nil {
				wrappedErr := eris.Wrap(err, "failed to create worker comment")
				r.logger.WithField("error", wrappedErr).Error("failed to create worker comment")
				return wrappedErr
			}
		}
	}

	return nil
}

func (r *WorkerOps) syncContacts(
	ctx context.Context, tx *ent.Tx, entity WorkerUpdateRequest, updatedEntity *ent.Worker,
) error {
	existingContacts, err := tx.Worker.QueryWorkerContacts(updatedEntity).Where(
		workercontact.HasWorkerWith(worker.IDEQ(entity.ID)),
	).All(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to featch existing worker contacts")
	}

	// Delete unmatched contacts
	if err = r.deleteUnmatchedContacts(ctx, tx, entity, existingContacts); err != nil {
		return err
	}

	// Update or create new contacts
	return r.updateOrCreateContacts(ctx, tx, entity)
}

func (r *WorkerOps) deleteUnmatchedContacts(
	ctx context.Context, tx *ent.Tx, entity WorkerUpdateRequest, existingContacts []*ent.WorkerContact,
) error {
	contactPresent := make(map[uuid.UUID]bool)
	for _, contact := range entity.Contacts {
		if contact.ID != uuid.Nil {
			contactPresent[contact.ID] = true
		}
	}

	for _, existingContact := range existingContacts {
		if !contactPresent[existingContact.ID] {
			if err := tx.WorkerContact.DeleteOneID(existingContact.ID).Exec(ctx); err != nil {
				wrappedErr := eris.Wrap(err, "failed to delete worker contact")
				r.logger.WithField("error", wrappedErr).Error("failed to delete worker contact")
				return wrappedErr
			}
		}
	}

	return nil
}

func (r *WorkerOps) updateOrCreateContacts(
	ctx context.Context, tx *ent.Tx, entity WorkerUpdateRequest,
) error {
	for _, contact := range entity.Contacts {
		if contact.ID != uuid.Nil {
			if _, err := tx.WorkerContact.UpdateOneID(contact.ID).
				SetName(contact.Name).
				SetEmail(contact.Email).
				SetPhone(contact.Phone).
				SetRelationship(contact.Relationship).
				SetIsPrimary(contact.IsPrimary).
				Save(ctx); err != nil {
				wrappedErr := eris.Wrap(err, "failed to update worker contact")
				r.logger.WithField("error", wrappedErr).Error("failed to update worker contact")
				return wrappedErr
			}
		} else {
			if _, err := tx.WorkerContact.Create().
				SetWorkerID(entity.ID).
				SetBusinessUnitID(entity.BusinessUnitID).
				SetOrganizationID(entity.OrganizationID).
				SetName(contact.Name).
				SetEmail(contact.Email).
				SetPhone(contact.Phone).
				SetRelationship(contact.Relationship).
				SetIsPrimary(contact.IsPrimary).
				Save(ctx); err != nil {
				wrappedErr := eris.Wrap(err, "failed to create worker contact")
				r.logger.WithField("error", wrappedErr).Error("failed to create worker contact")
				return wrappedErr
			}
		}
	}

	return nil
}
