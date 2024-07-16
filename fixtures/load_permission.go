package fixtures

import (
	"context"
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/casbin/casbin/v2"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

func loadPermissions(ctx context.Context, db *bun.DB, enforcer *casbin.Enforcer) error {
	var resources []*models.Resource
	err := db.NewSelect().Model(&resources).Scan(ctx)
	if err != nil {
		return err
	}

	// Detailed permissions for each action
	actions := []struct {
		action           string
		readDescription  string
		writeDescription string
	}{
		{"view", "Can view all", "Can view all"},
		{"create", "Can view all", "Can create, update, and delete"},
		{"update", "Can view all", "Can create, update, and delete"},
		{"delete", "Can view all", "Can create, update, and delete"},
	}

	for _, resource := range resources {
		resourceTypeLower := toSnakeCase(resource.Type)
		for _, action := range actions {
			codename := fmt.Sprintf("%s:%s", resourceTypeLower, action.action)

			// Add policy for the Admin role instead of "admin" subject
			_, err = enforcer.AddPolicy("Admin", codename, "allow")
			if err != nil {
				return fmt.Errorf("failed to add policy: %w", err)
			}

			log.Printf("Added permission to Casbin: Admin, %s, allow\n", codename)
		}
	}

	return enforcer.SavePolicy()
}

// toSnakeCase converts a string from CamelCase to snake_case
func toSnakeCase(s string) string {
	var result string
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result += "_"
		}
		result += strings.ToLower(string(r))
	}
	return result
}
