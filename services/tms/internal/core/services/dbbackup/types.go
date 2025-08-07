/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package dbbackup

import (
	"github.com/emoss08/trenova/shared/pulid"
)

// Response types for the backup API
type (
	ListBackupsRequest struct {
		UserID pulid.ID `json:"userId"`
		BuID   pulid.ID `json:"buId"`
		OrgID  pulid.ID `json:"orgId"`
	}

	CreateBackupRequest struct {
		UserID pulid.ID `json:"userId"`
		BuID   pulid.ID `json:"buId"`
		OrgID  pulid.ID `json:"orgId"`
	}

	DownloadBackupRequest struct {
		UserID   pulid.ID `json:"userId"`
		BuID     pulid.ID `json:"buId"`
		OrgID    pulid.ID `json:"orgId"`
		Filename string   `json:"filename"`
	}

	DeleteBackupRequest struct {
		UserID   pulid.ID `json:"userId"`
		BuID     pulid.ID `json:"buId"`
		OrgID    pulid.ID `json:"orgId"`
		Filename string   `json:"filename"`
	}

	// BackupFileResponse represents a backup file in API responses
	BackupFileResponse struct {
		Filename    string `json:"filename"`
		Size        int64  `json:"size"`
		CreatedAt   int64  `json:"createdAt"`
		Database    string `json:"database"`
		DownloadURL string `json:"downloadUrl"`
	}

	// BackupListResponse represents the response for listing backups
	BackupListResponse struct {
		Backups []BackupFileResponse `json:"backups"`
	}

	// BackupCreateResponse represents the response for creating a backup
	BackupCreateResponse struct {
		Backup  BackupFileResponse `json:"backup"`
		Message string             `json:"message"`
	}

	// BackupDeleteResponse represents the response for deleting a backup
	BackupDeleteResponse struct {
		Message string `json:"message"`
	}

	// BackupRestoreResponse represents the response for restoring a backup
	BackupRestoreResponse struct {
		Message string `json:"message"`
	}

	// RestoreRequest represents the request to restore a backup
	RestoreRequest struct {
		UserID   pulid.ID `json:"userId"`
		BuID     pulid.ID `json:"buId"`
		OrgID    pulid.ID `json:"orgId"`
		Filename string   `json:"filename"`
	}

	// ApplyRetentionPolicyRequest represents the request to apply a retention policy
	ApplyRetentionPolicyRequest struct {
		UserID        pulid.ID `json:"userId"`
		BuID          pulid.ID `json:"buId"`
		OrgID         pulid.ID `json:"orgId"`
		RetentionDays int      `json:"retentionDays"`
	}
)
