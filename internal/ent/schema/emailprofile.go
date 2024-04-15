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
		hook.On(emailProfileHook, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}

// emailProfileHook centralizes the logic for mutating email profiles.
func emailProfileHook(next ent.Mutator) ent.Mutator {
	return hook.EmailProfileFunc(func(ctx context.Context, m *gen.EmailProfileMutation) (ent.Value, error) {
		if shouldProceed, err := checkDefaultStatus(ctx, m); !shouldProceed {
			return nil, err
		}
		return next.Mutate(ctx, m)
	})
}

// checkDefaultStatus checks if the isDefault flag is set and processes accordingly.
func checkDefaultStatus(ctx context.Context, m *gen.EmailProfileMutation) (bool, error) {
	isDefault, isDefaultSet := m.IsDefault()
	if !isDefaultSet {
		// If IsDefault isn't being set in this mutation, no need to check.
		return true, nil
	}

	organizationID, err := validateOrganizationID(ctx, m)
	if err != nil {
		return false, err
	}

	if isDefault {
		return checkExistingDefaultProfile(ctx, m, organizationID)
	}

	return true, nil
}

// validateOrganizationID ensures that a valid organization ID is provided for the operation.
func validateOrganizationID(ctx context.Context, m *gen.EmailProfileMutation) (uuid.UUID, error) {
	organizationID, orgIDExists := m.OrganizationID()
	if !orgIDExists {
		return handleMissingOrganizationID(ctx, m)
	}
	return organizationID, nil
}

// handleMissingOrganizationID handles cases where no organization ID is present.
func handleMissingOrganizationID(ctx context.Context, m *gen.EmailProfileMutation) (uuid.UUID, error) {
	if m.Op().Is(ent.OpUpdateOne) || m.Op().Is(ent.OpUpdate) {
		return fetchExistingOrganizationID(ctx, m)
	}
	return uuid.Nil, errors.New("organizationID is required")
}

// fetchExistingOrganizationID fetches the organization ID from an existing profile.
func fetchExistingOrganizationID(ctx context.Context, m *gen.EmailProfileMutation) (uuid.UUID, error) {
	emailProfileID, idExists := m.ID()
	if !idExists {
		return uuid.Nil, errors.New("email profile ID is required for update operations")
	}
	existingProfile, err := m.Client().EmailProfile.Get(ctx, emailProfileID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get the existing email profile: %w", err)
	}
	return existingProfile.OrganizationID, nil
}

// checkExistingDefaultProfile ensures that there can only be one default profile per organization.
func checkExistingDefaultProfile(ctx context.Context, m *gen.EmailProfileMutation, organizationID uuid.UUID) (bool, error) {
	existingDefaultCount, err := m.Client().EmailProfile.Query().
		Where(
			emailprofile.IsDefault(true),
			emailprofile.OrganizationID(organizationID),
		).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to query the existing default email profiles: %w", err)
	}
	if existingDefaultCount > 0 {
		log.Println("Default profile is found")
		return false, errors.New("cannot set multiple default email profiles for the same organization")
	}
	return true, nil
}

// Edges of the EmailProfile.
func (EmailProfile) Edges() []ent.Edge {
	return nil
}
