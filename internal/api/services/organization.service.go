// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
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
