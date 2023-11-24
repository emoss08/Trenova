package db

import (
	"backend/models"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

func GenerateStandardPermissions(modelName string) []models.Permission {
	return []models.Permission{
		{Name: fmt.Sprintf("view_%s", modelName), HelpText: fmt.Sprintf("Can view %s", modelName)},
		{Name: fmt.Sprintf("add_%s", modelName), HelpText: fmt.Sprintf("Can add %s", modelName)},
		{Name: fmt.Sprintf("change_%s", modelName), HelpText: fmt.Sprintf("Can change %s", modelName)},
		{Name: fmt.Sprintf("delete_%s", modelName), HelpText: fmt.Sprintf("Can delete %s", modelName)},
	}
}

func InitializePermissions(db *gorm.DB) {
	for _, model := range mods {
		// Get the type of the model
		modelType := reflect.TypeOf(model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}

		// Generate a lowercase model name
		modelName := strings.ToLower(modelType.Name())

		// Generate standard permissions for the model
		permissions := GenerateStandardPermissions(modelName)

		// Create permissions in the database
		for _, perm := range permissions {
			// Check if permission already exists to avoid duplication
			var count int64
			db.Model(&models.Permission{}).Where("name = ?", perm.Name).Count(&count)
			if count == 0 {
				db.Create(&perm)
			}
		}
	}
}
