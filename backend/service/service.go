package service

import (
	"log"

	"github.com/fatih/color"
	"gorm.io/gorm"
)

type ServiceContainer struct {
	UserService *UserService
}

func InitializeServices(db *gorm.DB) *ServiceContainer {
	serviceMsg := color.New(color.FgHiGreen).SprintfFunc()
	log.Println(serviceMsg("🛠️ Initializing services...."))

	return &ServiceContainer{
		UserService: NewUserService(db),
	}
}
