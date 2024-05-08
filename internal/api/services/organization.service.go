package services

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/minio"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// OrganizationService provides methods for managing organization data in the database.
//
// Fields:
//   - Client: The database client object used to interact with the database.
//   - Logger: The logger object used to log information about the service operations.
type OrganizationService struct {
	Client      *ent.Client
	Logger      *zerolog.Logger
	Minio       *minio.Client
	FileService *util.FileService
}

// NewOrganizationService initializes a new instance of OrganizationService with the provided dependencies.
// this function is typically called during application startup to set up services
// that will be used through the application's lifecycle.
//
// Parameters:
//   - s: The API server object that contains the database client and logger.
//
// Returns:
//   - *OrganizationService: A new instance of the organization service.
//
// Usage:
//
//	orgService := services.NewOrganizationService(s)
func NewOrganizationService(s *api.Server) *OrganizationService {
	return &OrganizationService{
		Client:      s.Client,
		Logger:      s.Logger,
		Minio:       s.Minio,
		FileService: util.NewFileService(s.Logger),
	}
}

// UploadLogo uploads the organization logo file to Minio and updates the organization entity with the logo URL.
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - logo: The file header containing the logo file data.
//   - orgID: The organization ID to update.
//
// Returns:
//   - *ent.Organization: The updated organization entity if the operation is successful.
//   - error: Error object if an error occurs during the database update operation.
func (r *OrganizationService) UploadLogo(ctx context.Context, logo *multipart.FileHeader, orgID uuid.UUID) (*ent.Organization, error) {
	fileData, err := r.FileService.ReadFileData(logo)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to read file data")
		return nil, err
	}

	objectName, err := r.FileService.RenameFile(logo, orgID)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to rename file")
		return nil, err
	}

	org, err := r.updateAndSetLogoURL(ctx, orgID, objectName, logo.Header.Get("Content-Type"), fileData)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// uploadAndSetLogoURL uploads the logo file to Minio and sets the logo URL in the organization entity.
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - orgID: The organization ID to update.
//   - objectName: The name of the object to save in Minio.
//   - contentType: The content type of the file being uploaded.
//   - fileData: The byte slice containing the file data to upload.
//
// Returns:
//   - *ent.Organization: The updated organization entity if the operation is successful.
//   - error: Error object if an error occurs during the database update operation.
func (r *OrganizationService) updateAndSetLogoURL(
	ctx context.Context, orgID uuid.UUID, objectName, contentType string, fileData []byte,
) (*ent.Organization, error) {
	ui, err := r.Minio.SaveFile(ctx, "user-profile-pics", objectName, contentType, fileData)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to upload logo")
		return nil, err
	}

	org, err := r.Client.Organization.UpdateOneID(orgID).SetLogoURL(ui).Save(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to save logo URL")
		return nil, err
	}
	return org, err
}

// GetUserOrganization retrieves the organization information for the given business unit and organization ID.
// This function is used to retrieve the organization information for the currently authenticated user.
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - buID: The business unit ID for the organization.
//   - orgID: The organization ID to retrieve.
//
// Returns:
//   - *ent.Organization: Organization entity if the operation is successful.
//   - error: Error object if an error occurs during the database query operation.
func (r *OrganizationService) GetUserOrganization(ctx context.Context, buID, orgID uuid.UUID) (*ent.Organization, error) {
	org, err := r.Client.Organization.
		Query().
		Where(
			organization.And(
				organization.ID(orgID),
				organization.HasBusinessUnitWith(
					businessunit.ID(buID),
				),
			),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// UpdateOrganization updates the organization information in the database
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - entity: The updated organization entity to save in the database.
//
// Returns:
//   - *ent.Organization: Updated organization entity if the operation is successful.
//   - error: Error object if an error occurs during the database update operation.
func (r *OrganizationService) UpdateOrganization(ctx context.Context, entity *ent.Organization) (*ent.Organization, error) {
	var updatedEntity *ent.Organization

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateOrganizationEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})

	return updatedEntity, err
}

// updateOrganizationEntity updates organization ifnormation in the data within a transaction.
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - tx: The transaction object used to execute the database update operation.
//   - entity: The updated user entity to save in the database.
//
// Returns:
//   - *ent.Organization: Update organization entity if the operation is successful.
//   - error: Error object if an error occurs during the database update operation.
func (r *OrganizationService) updateOrganizationEntity(ctx context.Context, tx *ent.Tx, entity *ent.Organization) (*ent.Organization, error) {
	updateOP := tx.Organization.UpdateOneID(entity.ID).
		SetName(entity.Name).
		SetScacCode(entity.ScacCode).
		SetDotNumber(entity.DotNumber).
		SetOrgType(entity.OrgType).
		SetTimezone(entity.Timezone)

	updatedEntity, err := updateOP.Save(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("failed to update organization")
		return nil, err
	}

	return updatedEntity, nil
}
