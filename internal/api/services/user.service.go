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
		Relation("Roles.Permissions").
		Where("u.id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s UserService) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := user.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, err
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
