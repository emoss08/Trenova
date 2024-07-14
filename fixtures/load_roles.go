package fixtures

import (
	"context"
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/uptrace/bun"
)

func loadRoles(_ context.Context, _ *bun.DB, enforcer *casbin.Enforcer) error {
	roles := []string{"Admin", "Dispatcher", "Billing", "Safety", "Maintenance"}

	for _, role := range roles {
		// We don't need to add grouping policies here
		log.Printf("Role defined: %s\n", role)
	}

	return nil
}
