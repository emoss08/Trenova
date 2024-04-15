package schema

import (
	"context"
	"errors"
	"fmt"
	"log"

	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/emailprofile"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// EmailProfile holds the schema definition for the EmailProfile entity.
type EmailProfile struct {
	ent.Schema
}

// Fields of the EmailProfile.
func (EmailProfile) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			MaxLen(150),
		field.String("email").
			NotEmpty(),
		field.Enum("protocol").
			Values("TLS", "SSL", "UNENCRYPTED").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(11)",
				dialect.SQLite:   "VARCHAR(11)",
			}).
			StructTag(`json:"protocol"`).
			Optional(),
		field.String("host").
			Optional(),
		field.Int16("port").
			Optional(),
		field.String("username").
			Optional(),
		field.String("password").
			Sensitive().
			Optional(),
		field.Bool("is_default").
			Default(false).
			StructTag(`json:"isDefault"`),
	}
}

// Mixin of the EmailProfile.
func (EmailProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Hooks for the EmailProfile.
func (EmailProfile) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.EmailProfileFunc(func(ctx context.Context, m *gen.EmailProfileMutation) (ent.Value, error) {
					isDefault, isDefaultSet := m.IsDefault()
					if !isDefaultSet {
						// If IsDefault isn't being set in this mutation, no need to check.
						return next.Mutate(ctx, m)
					}

					var organizationID uuid.UUID
					var orgIDExists bool

					if organizationID, orgIDExists = m.OrganizationID(); !orgIDExists {
						if m.Op().Is(ent.OpUpdateOne) || m.Op().Is(ent.OpUpdate) {
							emailProfileID, idExists := m.ID()
							if !idExists {
								return nil, errors.New("email profile ID is required for update operations")
							}

							existingProfile, err := m.Client().EmailProfile.Get(ctx, emailProfileID)
							if err != nil {
								return nil, fmt.Errorf("failed to get the existing email profile: %w", err)
							}
							organizationID = existingProfile.OrganizationID

							// If updating the existing default profile, allow the update without further checks.
							if existingProfile.IsDefault && isDefault {
								return next.Mutate(ctx, m)
							}
						} else {
							return nil, errors.New("organizationID is required")
						}
					}

					// Check for existing default profile only if attempting to set a new default.
					if isDefault {
						existingDefaultCount, err := m.Client().EmailProfile.Query().
							Where(
								emailprofile.IsDefault(true),
								emailprofile.OrganizationID(organizationID),
							).
							Count(ctx)
						if err != nil {
							return nil, fmt.Errorf("failed to query the existing default email profiles: %w", err)
						}

						if existingDefaultCount > 0 {
							log.Println("Default profile is found")
							return nil, errors.New("cannot set multiple default email profiles for the same organization")
						}
					}

					return next.Mutate(ctx, m)
				})
			},
			ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne,
		),
	}
}

// Edges of the EmailProfile.
func (EmailProfile) Edges() []ent.Edge {
	return nil
}
