package db

import (
	"log"
	"os"

	"gorm.io/gorm"
)

func MigrateEnums(db *gorm.DB) {
	dir, err := os.Getwd()

	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	if err := executeSQLFromFile(db, dir+"/db/query/enums.sql"); err != nil {
		log.Fatalf("Failed to execute ENUM migration: %v", err)
	}
}

func executeSQLFromFile(db *gorm.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	sql := string(content)
	return db.Exec(sql).Error
}
