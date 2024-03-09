package database

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

func MigrateTypes(db *gorm.DB, migrationsPath string) error {
	path := filepath.Join(migrationsPath, "types.sql")

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	if err := executeFromSQLFile(tx, path); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute types migration: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit types migration transaction: %w", err)
	}

	return nil
}

func executeFromSQLFile(db *gorm.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	statements := string(content)
	if err := db.Exec(statements).Error; err != nil {
		return fmt.Errorf("failed to execute SQL from file %s: %w", path, err)
	}

	return nil
}
