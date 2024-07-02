package services

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/file"
	"github.com/emoss08/trenova/pkg/minio"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type OrganizationService struct {
	db          *bun.DB
	logger      *zerolog.Logger
	minio       minio.MinioClient
	fileService *file.FileService
}

func NewOrganizationService(s *server.Server) *OrganizationService {
	return &OrganizationService{
		db:          s.DB,
		logger:      s.Logger,
		minio:       s.Minio,
		fileService: file.NewFileService(s.Logger, s.FileHandler),
	}
}

func (s *OrganizationService) GetUserOrganization(ctx context.Context, buID, orgID uuid.UUID) (*models.Organization, error) {
	org := new(models.Organization)

	err := s.db.NewSelect().
		Model(org).
		Relation("State").
		Where("organization.business_unit_id = ?", buID).
		Where("organization.id = ?", orgID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}

func (s *OrganizationService) GetOrganization(ctx context.Context, buID, orgID uuid.UUID) (*models.Organization, error) {
	org := new(models.Organization)

	err := s.db.NewSelect().
		Model(org).
		Relation("State").
		Where("organization.business_unit_id = ?", buID).
		Where("organization.id = ?", orgID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, entity *models.Organization) (*models.Organization, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model(entity).WherePK().Returning("*").Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *OrganizationService) UploadLogo(ctx context.Context, logo *multipart.FileHeader, orgID uuid.UUID) error {
	fileData, err := s.fileService.ReadFileData(logo)
	if err != nil {
		return err
	}

	objectName, err := s.fileService.RenameFile(logo, orgID)
	if err != nil {
		return err
	}

	params := minio.SaveFileOptions{
		BucketName:  "organization-logo",
		ObjectName:  objectName,
		ContentType: logo.Header.Get("Content-Type"),
		FileData:    fileData,
	}

	return s.updateAndSetLogoURL(ctx, orgID, params)
}

func (s *OrganizationService) updateAndSetLogoURL(ctx context.Context, orgID uuid.UUID, params minio.SaveFileOptions) error {
	org := new(models.Organization)

	ui, err := s.minio.SaveFile(ctx, params)
	if err != nil {
		return err
	}

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err = tx.NewUpdate().
			Model(org).
			Set("logo_url = ?", ui).
			Where("id = ?", orgID).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *OrganizationService) ClearLogo(ctx context.Context, orgID uuid.UUID) error {
	org := new(models.Organization)

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model(org).
			Set("logo_url = ?", "").
			Where("id = ?", orgID).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
