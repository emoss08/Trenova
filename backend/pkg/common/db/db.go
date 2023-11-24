package db

import (
	"backend/pkg/common/models"
	"log"

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
}

func Init(url string) *gorm.DB {

	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})

	if err != nil {
		log.Fatalln(err)
	}

	db.AutoMigrate(mods...)

	return db
}
