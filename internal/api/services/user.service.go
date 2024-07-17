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

type UserService struct {
	db          *bun.DB
	logger      *zerolog.Logger
	minio       minio.MinioClient
	fileService *file.FileService
}

func NewUserService(s *server.Server) *UserService {
	return &UserService{
		db:          s.DB,
		logger:      s.Logger,
		minio:       s.Minio,
		fileService: file.NewFileService(s.Logger, s.FileHandler),
	}
}

func (s UserService) GetAuthenticatedUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	u := new(models.User)

	err := s.db.NewSelect().
		Model(u).
		Where("u.id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s UserService) UpdateUser(ctx context.Context, entity *models.User) (*models.User, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entity, err
}

func (s UserService) ClearProfilePic(ctx context.Context, userID uuid.UUID) error {
	user := new(models.User)
	user.ProfilePicURL = ""
	user.ThumbnailURL = ""

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model(user).
			Set("profile_pic_url = '', thumbnail_url = ''").
			Where("id = ?", userID).
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

func (s UserService) UploadProfilePicture(ctx context.Context, pic *multipart.FileHeader, userID uuid.UUID) (*models.User, error) {
	fileData, err := s.fileService.ReadFileData(pic)
	if err != nil {
		s.logger.Error().Err(err).Msg("UserService: Failed to read file data")
		return nil, err
	}

	objectName, err := s.fileService.RenameFile(pic, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("UserService: Failed to rename file")
		return nil, err
	}

	params := minio.SaveFileOptions{
		BucketName:  "user-profile-pics",
		ObjectName:  objectName,
		ContentType: pic.Header.Get("Content-Type"),
		FileData:    fileData,
	}

	return s.uploadAndSetProfilePicURL(ctx, userID, params)
}

func (s UserService) uploadAndSetProfilePicURL(ctx context.Context, userID uuid.UUID, params minio.SaveFileOptions) (*models.User, error) {
	user := new(models.User)

	ui, err := s.minio.SaveFile(ctx, params)
	if err != nil {
		s.logger.Error().Err(err).Msg("UserService: Failed to save file")
		return nil, err
	}

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err = tx.NewUpdate().
			Model(user).
			Set("profile_pic_url = ?", ui).
			Where("id = ?", userID).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, err
}

func (s UserService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user := new(models.User)

	err := s.db.NewSelect().
		Model(user).
		Where("id = ?", userID).
		Scan(ctx)
	if err != nil {
		return err
	}

	if err = user.VerifyPassword(oldPassword); err != nil {
		return err
	}

	hash := user.GeneratePassword(newPassword)

	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err = tx.NewUpdate().
			Model(user).
			Set("password = ?", hash).
			Where("id = ?", userID).
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
