package schema

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"time"

	gen "github.com/emoss08/trenova/ent"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/emoss08/trenova/ent/hook"
)

// Customer holds the schema definition for the Customer entity.
type Customer struct {
	ent.Schema
}

// Fields of the Customer.
func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(10).
			StructTag(`json:"code" validate:"required,max=10"`),
		field.String("name").
			NotEmpty().
			MaxLen(150).
			StructTag(`json:"name" validate:"required,max=150"`),
		field.String("address_line_1").
			NotEmpty().
			MaxLen(150).
			StructTag(`json:"addressLine1" validate:"required,max=150"`),
		field.String("address_line_2").
			Optional().
			MaxLen(150).
			StructTag(`json:"addressLine2" validate:"omitempty,max=150"`),
		field.String("city").
			NotEmpty().
			MaxLen(150).
			StructTag(`json:"city" validate:"required,max=150"`),
		field.String("state").
			NotEmpty().
			MaxLen(2).
			StructTag(`json:"state" validate:"required,len=2"`),
		field.String("postal_code").
			NotEmpty().
			MaxLen(10).
			StructTag(`json:"postalCode" validate:"required,max=10"`),
		field.Bool("has_customer_portal").
			Default(false).
			StructTag(`json:"hasCustomerPortal" validate:"omitempty"`),
		field.Bool("auto_mark_ready_to_bill").
			Default(false).
			StructTag(`json:"autoMarkReadyToBill" validate:"omitempty"`),
	}
}

// Mixin of the Customer.
func (Customer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the Customer.
func (Customer) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}

// Edges of the Customer.
func (Customer) Edges() []ent.Edge {
	return nil
}

// Hooks for the Customer.
func (Customer) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.CustomerFunc(func(ctx context.Context, m *gen.CustomerMutation) (ent.Value, error) {
					if m.Op().Is(ent.OpCreate) {
						if _, exists := m.Field("code"); !exists {
							name, nameExists := m.Field("name")

							if nameExists {
								// Generate a customer code based on the name and current time
								code := generateCustomerCode(name.(string), time.Now())
								m.SetCode(code)
							}
						}
					}
					return next.Mutate(ctx, m)
				})
			},
			ent.OpCreate,
		),
	}
}

// generateCustomerCode generates a customer code based on the name and current time using crypto/rand for randomness.
func generateCustomerCode(name string, createdAt time.Time) string {
	var initials string
	parts := strings.Fields(strings.ToUpper(name))
	if len(parts) > 0 {
		initials = string(parts[0][0])
	}
	if len(parts) > 1 {
		initials += string(parts[len(parts)-1][0])
	}

	for len(initials) < 2 {
		initials += "X"
	}

	var randSeq uint32
	if err := binary.Read(rand.Reader, binary.LittleEndian, &randSeq); err != nil {
		log.Printf("Error generating random sequence: %v\n", err)
		return ""
	}
	randomSequence := randSeq % 100

	year, day := createdAt.Year(), createdAt.YearDay()
	dateCode := fmt.Sprintf("%03d%d", day, year%10)

	return fmt.Sprintf("%s%02d%s", initials, randomSequence, dateCode)
}
