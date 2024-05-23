package schema

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/ent/shipment"
	"github.com/emoss08/trenova/internal/queries"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/emoss08/trenova/internal/validators"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

// Shipment holds the schema definition for the Shipment entity.
type Shipment struct {
	ent.Schema
}

// Fields of the Shipment.
func (Shipment) Fields() []ent.Field { //nolint:funlen // This is a schema definition.
	return []ent.Field{
		field.String("pro_number").
			NotEmpty().
			MaxLen(20).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(20)",
				dialect.SQLite:   "VARCHAR(20)",
			}).
			StructTag(`json:"pro_number" validate:"required,max=20"`),
		field.Enum("status").
			Values("New",
				"InProgress",
				"Completed",
				"Hold",
				"Billed",
				"Voided").
			Default("New").
			StructTag(`json:"status" validate:"required"`),
		field.UUID("origin_location_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"originLocationId" validate:"required"`),
		field.String("origin_address_line").
			Optional().
			StructTag(`json:"originAddressLine" validate:"omitempty"`),
		field.Time("origin_appointment_start").
			Optional().
			Nillable().
			StructTag(`json:"originAppointmentStart" validate:"required"`),
		field.Time("origin_appointment_end").
			Optional().
			Nillable().
			StructTag(`json:"originAppointmentEnd" validate:"required"`),
		field.UUID("destination_location_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"destinationLocationId" validate:"required"`),
		field.String("destination_address_line").
			Optional().
			StructTag(`json:"destinationAddressLine" validate:"omitempty"`),
		field.Time("destination_appointment_start").
			Optional().
			Nillable().
			StructTag(`json:"destinationAppointmentStart" validate:"required"`),
		field.Time("destination_appointment_end").
			Optional().
			Nillable().
			StructTag(`json:"destinationAppointmentEnd" validate:"required"`),
		field.UUID("shipment_type_id", uuid.UUID{}).
			StructTag(`json:"shipmentTypeId" validate:"required"`),
		field.UUID("revenue_code_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"revenueCodeId" validate:"omitempty"`),
		field.UUID("service_type_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"serviceTypeId" validate:"omitempty"`),
		// Billing Information for the shipment.
		field.Int("rating_unit").
			Positive().
			Default(1).
			Comment("The rating unit for the shipment.").
			StructTag(`json:"ratingUnit" validate:"omitempty"`),
		field.Float("mileage").
			Positive().
			Optional().
			StructTag(`json:"mileage" validate:"omitempty"`),
		field.Float("other_charge_amount").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			Optional().
			StructTag(`json:"otherChargeAmount" validate:"omitempty"`),
		field.Float("freight_charge_amount").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			Optional().
			StructTag(`json:"freightChargeAmount" validate:"omitempty"`),
		field.Enum("rating_method").
			Values("FlatRate", "PerMile", "PerHundredWeight", "PerStop", "PerPound", "Other").
			Default("FlatRate").
			StructTag(`json:"ratingMethod" validate:"omitempty"`),
		field.UUID("customer_id", uuid.UUID{}).
			StructTag(`json:"customerId" validate:"required"`),
		field.Float("pieces").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"pieces" validate:"omitempty"`),
		field.Float("weight").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Optional().
			StructTag(`json:"weight" validate:"omitempty"`),
		field.Bool("ready_to_bill").
			Default(false).
			StructTag(`json:"readyToBill" validate:"omitempty"`),
		field.Other("bill_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"billDate" validate:"omitempty"`),
		field.Other("ship_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"shipDate" validate:"omitempty"`),
		field.Bool("billed").
			Default(false).
			StructTag(`json:"billed" validate:"omitempty"`),
		field.Bool("transferred_to_billing").
			Default(false).
			StructTag(`json:"transferredToBilling" validate:"omitempty"`),
		field.Other("transferred_to_billing_date", &pgtype.Date{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"transferredToBillingDate" validate:"omitempty"`),
		field.Float("total_charge_amount").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			Optional().
			StructTag(`json:"totalChargeAmount" validate:"omitempty"`),
		field.UUID("trailer_type_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"trailerTypeId" validate:"omitempty"`),
		field.UUID("tractor_type_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"tractorTypeId" validate:"omitempty"`),
		field.Int("temperature_min").
			Optional().
			StructTag(`json:"temperatureMin" validate:"omitempty"`),
		field.Int("temperature_max").
			Optional().
			StructTag(`json:"temperatureMax" validate:"omitempty"`),
		field.String("bill_of_lading_number").
			Optional().
			StructTag(`json:"billOfLadingNumber" validate:"omitempty"`),
		field.String("consignee_reference_number").
			Optional().
			StructTag(`json:"consigneeReferenceNumber" validate:"omitempty"`),
		field.Text("comment").
			Optional().
			StructTag(`json:"comment" validate:"omitempty"`),
		field.String("voided_comment").
			MaxLen(100).
			Comment("The comment for voiding the shipment.").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(100)",
				dialect.SQLite:   "VARCHAR(100)",
			}).
			Optional().
			StructTag(`json:"voidedComment" validate:"omitempty"`),
		field.Bool("auto_rated").
			Default(false).
			Comment("Indicates if the shipment was auto rated.").
			StructTag(`json:"autoRated" validate:"omitempty"`),
		field.String("current_suffix").
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(2)",
				dialect.SQLite:   "VARCHAR(2)",
			}).
			StructTag(`json:"currentSuffix" validate:"omitempty"`),
		field.Enum("entry_method").
			Values("Manual", "EDI", "Web", "Mobile", "API").
			Default("Manual").
			StructTag(`json:"entryMethod" validate:"omitempty"`),
		field.UUID("created_by", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"createdBy" validate:"omitempty"`),
		field.Bool("is_hazardous").
			Default(false).
			Comment("Indicates if the shipment is hazardous.").
			StructTag(`json:"isHazardous" validate:"omitempty"`),
	}
}

// Mixin of the Shipment.
func (Shipment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the Shipment.
func (Shipment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("bill_date", "organization_id"),
		index.Fields("ship_date", "organization_id"),
		index.Fields("bill_of_lading_number", "organization_id"),
	}
}

// Annotations of the Shipment.
func (Shipment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.WithComments(true),
		schema.Comment("Shipment holds the schema definition for the Shipment entity."),
	}
}

// Edges of the Shipment.
func (Shipment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("shipment_type", ShipmentType.Type).
			Field("shipment_type_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"shipmentType"`),
		edge.To("service_type", ServiceType.Type).
			Field("service_type_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"serviceType"`),
		edge.To("revenue_code", ServiceType.Type).
			Field("revenue_code_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"revenueCode"`),
		edge.To("origin_location", Location.Type).
			Field("origin_location_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"originLocation"`),
		edge.To("destination_location", Location.Type).
			Field("destination_location_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"destinationLocation"`),
		edge.To("trailer_type", EquipmentType.Type).
			Field("trailer_type_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"trailerType"`),
		edge.To("tractor_type", EquipmentType.Type).
			Field("tractor_type_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"tractorType"`),
		edge.To("shipment_documentation", ShipmentDocumentation.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"shipmentDocumentation"`),
		edge.To("shipment_comments", ShipmentComment.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"shipmentComments"`),
		edge.To("shipment_charges", ShipmentCharges.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"shipmentCharges"`),
		edge.To("shipment_commodities", ShipmentCommodity.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"shipmentCommodities"`),
		edge.From("created_by_user", User.Type).
			Ref("shipments").
			Field("created_by").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"createdByUser"`),
		edge.From("customer", Customer.Type).
			Ref("shipments").
			Field("customer_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"customer"`),
		edge.To("shipment_moves", ShipmentMove.Type).
			StructTag(`json:"shipmentMoves`),
	}
}

// Hooks for the Shipment.
func (Shipment) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(ShipmentHook, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}

// handleCreateShipment handles the creation of a shipment including generating a pro number.
func handleCreateShipment(ctx context.Context, m *gen.ShipmentMutation, client *gen.Client) error {
	proNumber, err := generateProNumber(ctx, m, client)
	if err != nil {
		return err
	}
	m.SetProNumber(proNumber)
	return nil
}

// handleUpdateShipment handles updates to a shipment, including voiding the shipment.
func handleUpdateShipment(ctx context.Context, m *gen.ShipmentMutation, client *gen.Client) error {
	return queries.HandleVoidedShipment(ctx, m, client)
}

// validateShipmentControl validates various controls related to a shipment.
func validateShipmentControl(ctx context.Context, m *gen.ShipmentMutation, client *gen.Client) error {
	// Initialize the query service.
	queryService := queries.QueryService{Client: client, Logger: &zerolog.Logger{}}

	// Get the organization and business unit IDs from the mutation.
	orgID, _ := m.OrganizationID()
	buID, _ := m.BusinessUnitID()

	shipmentControl, err := queryService.GetShipmentControlByOrganization(ctx, client, orgID, buID)
	if err != nil {
		return err
	}

	billingControl, err := queryService.GetBillingControlByOrganization(ctx, client, orgID, buID)
	if err != nil {
		return err
	}

	dispatchControl, err := queryService.GetDispatchControlByOrganization(ctx, client, orgID, buID)
	if err != nil {
		return err
	}

	validationErrs, err := validators.ValidateShipment(ctx, m, shipmentControl, billingControl, dispatchControl)
	if err != nil {
		return err
	}

	if len(validationErrs) > 0 {
		return &types.ValidationErrorResponse{
			Type:   "validationError",
			Errors: validationErrs,
		}
	}
	return nil
}

// ShipmentHook main hook for processing shipments.
func ShipmentHook(next ent.Mutator) ent.Mutator {
	return hook.ShipmentFunc(func(ctx context.Context, m *gen.ShipmentMutation) (ent.Value, error) {
		client := m.Client()
		var err error

		if m.Op().Is(ent.OpCreate) {
			err = handleCreateShipment(ctx, m, client)
			if err != nil {
				return nil, err
			}
		}

		if m.Op().Is(ent.OpUpdate) {
			err = handleUpdateShipment(ctx, m, client)
			if err != nil {
				return nil, err
			}
		}

		err = validateShipmentControl(ctx, m, client)
		if err != nil {
			return nil, err
		}

		return next.Mutate(ctx, m)
	})
}

// generateProNumber facilitates the generation of a pro number for a shipment.
func generateProNumber(
	ctx context.Context, m *gen.ShipmentMutation, client *gen.Client,
) (string, error) {
	today := time.Now().Format("060102") // YYMMDD
	organizationID, _ := m.OrganizationID()

	// Count the number shipments for today and the same organization.
	countForToday, err := client.Shipment.Query().
		Where(
			shipment.ProNumberHasPrefix(today),
			shipment.OrganizationIDEQ(organizationID),
		).
		Count(ctx)
	if err != nil {
		return "", err
	}

	countForToday++ // Increment the count by 1 to get the next number.

	// Generate the pronumber with zero-padded count.
	proNumber := today + fmt.Sprintf("%s-%04d", today, countForToday)
	return proNumber, nil
}
