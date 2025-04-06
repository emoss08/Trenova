package services

import (
	"context"
	"time"
)

// BackupFileInfo contains information about a backup file
type BackupFileInfo struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
	Database  string    `json:"database"`
}

type BackupService interface {
	CreateBackup(ctx context.Context) (string, error)
	RestoreBackup(ctx context.Context, backupFile string) error
	ScheduledBackup(ctx context.Context, retentionDays int) error
	ApplyRetentionPolicy(retentionDays int) error
	ListBackups() ([]BackupFileInfo, error)
	DeleteBackup(backupPath string) error
	GetRetentionDays() int
	GetBackupDir() string
}

type BackupScheduler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	RunNow(ctx context.Context) error
}
