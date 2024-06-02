package schema

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/ent/rate"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Rate holds the schema definition for the Rate entity.
type Rate struct {
	ent.Schema
}

// Fields of the Rate.
func (Rate) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("rate_number").
			NotEmpty().
			MaxLen(22).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(22)",
				dialect.SQLite:   "VARCHAR(22)",
			}).
			StructTag(`json:"rateNumber" validate:"omitempty"`),
		field.UUID("customer_id", uuid.UUID{}).
			Unique().
			StructTag(`json:"customerId" validate:"required"`),
		field.Other("effective_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"effectiveDate"`),
		field.Other("expiration_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"expirationDate"`),
		field.UUID("commodity_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"commodityId" validate:"omitempty"`),
		field.UUID("shipment_type_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"shipmentTypeId" validate:"omitempty"`),
		field.UUID("origin_location_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"originLocationId" validate:"omitempty"`),
		field.UUID("destination_location_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"destinationLocationId" validate:"omitempty"`),
		field.Enum("rating_method").
			Values("FlatRate", "PerMile", "PerHundredWeight", "PerStop", "PerPound", "Other").
			Default("FlatRate").
			StructTag(`json:"ratingMethod" validate:"omitempty"`),
		field.Float("rate_amount").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			StructTag(`json:"rateAmount" validate:"required"`),
		field.Text("comment").
			Optional().
			StructTag(`json:"comment" validate:"omitempty"`),
		field.UUID("approved_by_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"approvedById" validate:"omitempty"`),
		field.Other("approved_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"approvedDate"`),
		field.Int("usage_count").
			Optional().
			Default(0).
			StructTag(`json:"usageCount" validate:"omitempty"`),
		field.Float("minimum_charge").
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			StructTag(`json:"minimumCharge" validate:"omitempty"`),
		field.Float("maximum_charge").
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			StructTag(`json:"maximumCharge" validate:"omitempty"`),
	}
}

// Mixin of the Rate.
func (Rate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the Rate.
func (Rate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Ref("rates").
			Field("customer_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("commodity", Commodity.Type).
			Ref("rates").
			Field("commodity_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("shipment_type", ShipmentType.Type).
			Ref("rates").
			Field("shipment_type_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("origin_location", Location.Type).
			Ref("rates_origin").
			Field("origin_location_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("destination_location", Location.Type).
			Ref("rates_destination").
			Field("destination_location_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("approved_by", User.Type).
			Ref("rates_approved").
			Field("approved_by_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Rate) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.RateFunc(func(ctx context.Context, m *gen.RateMutation) (ent.Value, error) {
					if m.Op().Is(ent.OpCreate) {
						if _, exists := m.Field("rate_number"); !exists {
							customerID, _ := m.CustomerID()
							originLocationID, _ := m.OriginLocationID()
							destinationLocationID, _ := m.DestinationLocationID()
							effectiveDate, _ := m.EffectiveDate()

							rateNumber, err := generateUniqueRateNumber(ctx, m.Client(), customerID, originLocationID, destinationLocationID, effectiveDate)
							if err != nil {
								return nil, err
							}
							m.SetRateNumber(rateNumber)
						}
					}
					return next.Mutate(ctx, m)
				})
			},
			ent.OpCreate,
		),
	}
}

func generateUniqueRateNumber(ctx context.Context, client *gen.Client, customerID, originLocationID, destinationLocationID uuid.UUID, effectiveDate *pgtype.Date) (string, error) {
	var rateNumber string
	var exists bool
	var err error

	for {
		rateNumber = generateRateNumber(customerID, originLocationID, destinationLocationID, effectiveDate)
		exists, err = client.Rate.Query().Where(rate.RateNumberEQ(rateNumber)).Exist(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to check if rate number exists: %w", err)
		}
		if !exists {
			break
		}
	}

	return rateNumber, nil
}

func generateRateNumber(customerID, originLocationID, destinationLocationID uuid.UUID, effectiveDate *pgtype.Date) string {
	var parts []string

	if customerID != uuid.Nil {
		parts = append(parts, getCompactCode(customerID, 1, 3))
	} else {
		parts = append(parts, "CXXX")
	}

	if originLocationID != uuid.Nil {
		parts = append(parts, getCompactCode(originLocationID, 1, 2))
	} else {
		parts = append(parts, "OXX")
	}

	if destinationLocationID != uuid.Nil {
		parts = append(parts, getCompactCode(destinationLocationID, 1, 2))
	} else {
		parts = append(parts, "DXX")
	}

	if effectiveDate != nil {
		parts = append(parts, effectiveDate.Time.Format("010206")) // MMDDYY format
	} else {
		parts = append(parts, "000000")
	}

	// Add a random 2-character suffix to ensure uniqueness
	parts = append(parts, randomString(2))

	return strings.Join(parts, "-")
}

// getCompactCode creates a short code from a UUID
func getCompactCode(id uuid.UUID, prefixLength int, suffixLength int) string {
	idStr := strings.ToUpper(id.String())
	prefix := idStr[:prefixLength]
	suffix := idStr[len(idStr)-suffixLength:]
	return fmt.Sprintf("%s%s", prefix, suffix)
}

// randomString generates a random alphanumeric string of the given length
func randomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
