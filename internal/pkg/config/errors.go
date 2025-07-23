// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package config

import "github.com/rotisserie/eris"

var (
	ErrConfigNotLoaded      = eris.New("config not loaded")
	ErrInvalidAppName       = eris.New("invalid app name")
	ErrInvalidServerAddress = eris.New("invalid server address")

	// Database errors
	ErrInvalidDBHost = eris.New("invalid database host")
	ErrInvalidDBPort = eris.New("invalid database port")
	ErrInvalidDBName = eris.New("invalid database name")
	ErrInvalidDBUser = eris.New("invalid database username")

	// ErrInvalidBackupCompression is returned when the compression level is invalid.
	ErrInvalidBackupCompression = eris.New("invalid backup compression level (must be between 1-9)")

	// ErrInvalidBackupCronSchedule is returned when the cron schedule is invalid.
	ErrInvalidBackupCronSchedule = eris.New("invalid backup cron schedule")
)
