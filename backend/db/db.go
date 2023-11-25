package db

import (
	"backend/models"
	"log"

	"github.com/fatih/color"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var mods = []interface{}{
	&models.Organization{},
	&models.BusinessUnit{},
	&models.JobTitle{},
	&models.User{},
	&models.UserProfile{},
	&models.Department{},
	&models.EmailControl{},
	&models.EmailProfile{},
	&models.Permission{},
	&models.Role{},
	&models.Permission{},
	&models.AuditLog{},
}

func Init(url string) *gorm.DB {

	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})

	if err != nil {
		log.Fatalln("Failed to connect to database:", err)
	}

	// Log a success message after establishing the connection
	successMsg := color.New(color.FgHiGreen).SprintfFunc()
	log.Println(successMsg("🌟 Successfully connected to the database"))

	// Migrate ENUM Types
	MigrateEnums(db)

	db.AutoMigrate(mods...)

	// Initialize permissions
	InitializePermissions(db)

	return db
}
