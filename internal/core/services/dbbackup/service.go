// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package dbbackup

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger            *logger.Logger
	BackupService     services.BackupService
	BackupScheduler   services.BackupScheduler
	PermissionService services.PermissionService
}

type Service struct {
	l   *zerolog.Logger
	bs  services.BackupService
	bss services.BackupScheduler
	ps  services.PermissionService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "dbbackup").
		Logger()

	return &Service{
		l:   &log,
		bs:  p.BackupService,
		bss: p.BackupScheduler,
		ps:  p.PermissionService,
	}
}

func (s *Service) ListBackups(
	ctx context.Context,
	req ListBackupsRequest,
) ([]BackupFileResponse, error) {
	log := s.l.With().Str("operation", "ListBackups").Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceBackup,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
				UserID:         req.UserID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read backups")
	}

	// * Get backup files
	backupFiles, err := s.bs.ListBackups()
	if err != nil {
		return nil, err
	}

	// * Format response with download URLs
	backups := make([]BackupFileResponse, 0, len(backupFiles))
	for _, file := range backupFiles {
		backups = append(backups, BackupFileResponse{
			Filename:    file.Filename,
			Size:        file.SizeBytes,
			CreatedAt:   file.CreatedAt.Unix(),
			Database:    file.Database,
			DownloadURL: "/api/v1/backups/" + file.Filename,
		})
	}

	return backups, nil
}

func (s *Service) CreateBackup(
	ctx context.Context,
	req CreateBackupRequest,
) (*BackupCreateResponse, error) {
	log := s.l.With().Str("operation", "CreateBackup").Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceBackup,
				Action:         permission.ActionCreate,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
				UserID:         req.UserID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read backups")
	}

	// * Create backup
	backupPath, err := s.bs.CreateBackup(ctx)
	if err != nil {
		return nil, err
	}

	// * Get file info from backup service
	backupFiles, err := s.bs.ListBackups()
	if err != nil {
		return nil, err
	}

	// * Find the created backup in the list
	var backup BackupFileResponse
	for _, file := range backupFiles {
		if file.Path == backupPath {
			backup = BackupFileResponse{
				Filename:    file.Filename,
				Size:        file.SizeBytes,
				CreatedAt:   file.CreatedAt.Unix(),
				Database:    file.Database,
				DownloadURL: "/api/v1/backups/" + file.Filename,
			}
			break
		}
	}

	return &BackupCreateResponse{
		Backup:  backup,
		Message: "Backup created successfully",
	}, nil
}

func (s *Service) DownloadBackup(ctx context.Context, req *DownloadBackupRequest) (string, error) {
	log := s.l.With().Str("operation", "DownloadBackup").Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceBackup,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
				UserID:         req.UserID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return "", eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return "", errors.NewAuthorizationError("You do not have permission to read backups")
	}

	// * Get the backup directory
	backupDir := s.bs.GetBackupDir()
	backupPath := backupDir + "/" + req.Filename

	return backupPath, nil
}

func (s *Service) DeleteBackup(ctx context.Context, req *DeleteBackupRequest) error {
	log := s.l.With().Str("operation", "DeleteBackup").Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceBackup,
				Action:         permission.ActionDelete,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
				UserID:         req.UserID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to delete backups")
	}

	// * Get the backup directory
	backupDir := s.bs.GetBackupDir()
	backupPath := backupDir + "/" + req.Filename

	if err = s.bs.DeleteBackup(backupPath); err != nil {
		return eris.Wrap(err, "delete backup")
	}

	return nil
}

func (s *Service) RestoreBackup(ctx context.Context, req *RestoreRequest) error {
	log := s.l.With().Str("operation", "RestoreBackup").Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceBackup,
				Action:         permission.ActionRestore,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
				UserID:         req.UserID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to restore backups")
	}

	// * Get the backup directory
	backupDir := s.bs.GetBackupDir()
	backupPath := backupDir + "/" + req.Filename

	// * Restore the backup
	if err = s.bs.RestoreBackup(ctx, backupPath); err != nil {
		return eris.Wrap(err, "restore backup")
	}

	return nil
}

func (s *Service) ApplyRetentionPolicy(
	ctx context.Context,
	req *ApplyRetentionPolicyRequest,
) error {
	log := s.l.With().Str("operation", "ApplyRetentionPolicy").Logger()

	// *s Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceBackup,
				Action:         permission.ActionDelete,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
				UserID:         req.UserID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to delete backups")
	}

	var retentionDays int
	if req.RetentionDays <= 0 {
		// * Use default retention days from service
		retentionDays = s.bs.GetRetentionDays()
	} else {
		retentionDays = req.RetentionDays
	}

	// * Apply the retention policy
	if err = s.bs.ApplyRetentionPolicy(retentionDays); err != nil {
		return eris.Wrap(err, "apply retention policy")
	}

	return nil
}
