package schema

import (
	"context"
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/util"
	"github.com/pkg/errors"
)

// DocumentClassification holds the schema definition for the DocumentClassification entity.
type DocumentClassification struct {
	ent.Schema
}

// Fields of the DocumentClassification.
func (DocumentClassification) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"code" validate:"required,max=10"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.String("color").
			Optional().
			StructTag(`json:"color" validate:"omitempty"`),
	}
}

// Mixin of the DocumentClassification.
func (DocumentClassification) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the DocumentClassification.
func (DocumentClassification) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("shipment_documentation", ShipmentDocumentation.Type).
			StructTag(`json:"shipmentDocumentation,omitempty"`),
		edge.From("customer_rule_profile", CustomerRuleProfile.Type).
			Ref("document_classifications"),
	}
}

// Hooks for the DocumentClassification.
func (DocumentClassification) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(DocumentClassification{}.validateNameUniqueness, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.DocumentClassificationFunc(func(ctx context.Context, m *gen.DocumentClassificationMutation) (ent.Value, error) {
					// Always uppercase the code value.
					code, codeExists := m.Code()
					codeUpper := strings.ToUpper(code)

					if codeExists {
						m.SetCode(codeUpper)
					}

					return next.Mutate(ctx, m)
				})
			}, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}

// validateNameUniqueness is a hook that validates the uniqueness of the name field.
func (DocumentClassification) validateNameUniqueness(next ent.Mutator) ent.Mutator {
	return hook.DocumentClassificationFunc(func(ctx context.Context, m *gen.DocumentClassificationMutation) (ent.Value, error) {
		code, codeExists := m.Code()
		orgID, orgExists := m.OrganizationID()

		if !codeExists || !orgExists {
			return next.Mutate(ctx, m)
		}

		// Get the current record ID to exclude it from the uniqueness check.
		id, idExists := m.ID()

		conditions := map[string]string{
			"code":            code,
			"organization_id": fmt.Sprint(orgID),
		}

		excludeID := ""
		if idExists {
			excludeID = fmt.Sprint(id)
		}

		err := util.ValidateUniqueness(ctx, m.Client(), "document_classifications", "code", conditions, excludeID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to validate uniqueness of name within organization")
		}

		return next.Mutate(ctx, m)
	})
}
