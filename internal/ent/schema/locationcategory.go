package schema

import (
	"context"
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/util"
	"github.com/pkg/errors"
)

// LocationCategory holds the schema definition for the LocationCategory entity.
type LocationCategory struct {
	ent.Schema
}

// Fields of the LocationCategory.
func (LocationCategory) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(100)",
				dialect.SQLite:   "VARCHAR(100)",
			}).
			StructTag(`json:"name" validate:"required"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.String("color").
			Optional().
			StructTag(`json:"color" validate:"omitempty"`),
	}
}

// Edges of the LocationCategory.
func (LocationCategory) Edges() []ent.Edge {
	return nil
}

// Mixin of the LocationCategory.
func (LocationCategory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Hooks for the LocationCategory.
func (LocationCategory) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(validateNameUniqueness, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}

// validateNameUniqueness is a hook that validates the uniqueness of the name field.
func validateNameUniqueness(next ent.Mutator) ent.Mutator {
	return hook.LocationCategoryFunc(func(ctx context.Context, m *gen.LocationCategoryMutation) (ent.Value, error) {
		name, nameExists := m.Name()
		orgID, orgExists := m.OrganizationID() // Assuming you have OrganizationID field in your mutation

		if !nameExists || !orgExists {
			return next.Mutate(ctx, m)
		}

		// Get the current record ID to exclude it from the uniqueness check.
		id, idExists := m.ID()

		conditions := map[string]string{
			"name":            name,
			"organization_id": fmt.Sprint(orgID),
		}

		excludeID := ""
		if idExists {
			excludeID = fmt.Sprint(id)
		}

		err := util.ValidateUniqueness(ctx, m.Client(), "location_categories", "name", conditions, excludeID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to validate uniqueness of name within organization")
		}

		return next.Mutate(ctx, m)
	})
}
