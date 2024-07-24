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
	"fmt"
	"mime/multipart"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/file"
	"github.com/emoss08/trenova/pkg/minio"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/redis"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type OrganizationService struct {
	db          *bun.DB
	logger      *config.ServerLogger
	minio       minio.MinioClient
	cache       *redis.Client
	fileService *file.FileService
}

func NewOrganizationService(s *server.Server) *OrganizationService {
	return &OrganizationService{
		db:          s.DB,
		logger:      s.Logger,
		minio:       s.Minio,
		cache:       s.Cache,
		fileService: file.NewFileService(s.Logger, s.FileHandler),
	}
}

func (s *OrganizationService) organizationCacheKey(orgID uuid.UUID) string {
	return fmt.Sprintf("organization:%s", orgID.String())
}

func (s *OrganizationService) GetOrganization(ctx context.Context, buID, orgID uuid.UUID) (*models.Organization, error) {
	cacheKey := s.organizationCacheKey(orgID)

	// Try to fetch the organization from the cache
	cachedOrg, err := s.cache.FetchFromCacheByKey(ctx, cacheKey)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch organization from cache")
		return nil, err
	}

	if cachedOrg != "" {
		org := new(models.Organization)

		if err = sonic.Unmarshal([]byte(cachedOrg), org); err != nil {
			s.logger.Error().Err(err).Str("cacheKey", cacheKey).Msg("Failed to unmarshal organization from cache")
			return nil, err
		}

		return org, nil
	}

	// If not in cache then fetch from the database
	org := new(models.Organization)
	err = s.db.NewSelect().
		Model(org).
		Relation("State").
		Where("organization.business_unit_id = ?", buID).
		Where("organization.id = ?", orgID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the organization
	orgJSON, err := sonic.Marshal(org)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to marshal organization")
		return nil, err
	}

	if err = s.cache.CacheByKey(ctx, cacheKey, string(orgJSON)); err != nil {
		s.logger.Error().Err(err).Str("cacheKey", cacheKey).Msg("Failed to cache organization")
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

	// invalidate cache
	cacheKey := s.organizationCacheKey(entity.ID)
	if err = s.cache.InvalidateCacheByKey(ctx, cacheKey); err != nil {
		s.logger.Error().Err(err).Str("cacheKey", cacheKey).Msg("Failed to invalidate cache")
		return nil, err
	}

	return entity, nil
}

func (s *OrganizationService) UploadLogo(ctx context.Context, logo *multipart.FileHeader, orgID uuid.UUID) (*models.Organization, error) {
	fileData, err := s.fileService.ReadFileData(logo)
	if err != nil {
		return nil, err
	}

	objectName, err := s.fileService.RenameFile(logo, orgID)
	if err != nil {
		return nil, err
	}

	params := minio.SaveFileOptions{
		BucketName:  "organization-logo",
		ObjectName:  objectName,
		ContentType: logo.Header.Get("Content-Type"),
		FileData:    fileData,
	}

	return s.updateAndSetLogoURL(ctx, orgID, params)
}

func (s *OrganizationService) updateAndSetLogoURL(ctx context.Context, orgID uuid.UUID, params minio.SaveFileOptions) (*models.Organization, error) {
	org := new(models.Organization)

	ui, err := s.minio.SaveFile(ctx, params)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	// invalidate cache
	cacheKey := s.organizationCacheKey(orgID)
	if err = s.cache.InvalidateCacheByKey(ctx, cacheKey); err != nil {
		s.logger.Error().Err(err).Str("cacheKey", cacheKey).Msg("Failed to invalidate cache")
		return nil, err
	}

	return org, nil
}

func (s *OrganizationService) ClearLogo(ctx context.Context, orgID uuid.UUID) (*models.Organization, error) {
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
		return nil, err
	}

	// invalidate cache
	cacheKey := s.organizationCacheKey(orgID)
	if err = s.cache.InvalidateCacheByKey(ctx, cacheKey); err != nil {
		s.logger.Error().Err(err).Str("cacheKey", cacheKey).Msg("Failed to invalidate cache")
		return nil, err
	}

	return org, nil
}
