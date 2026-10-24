package recurringshipment

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*RecurringShipment)(nil)
	_ pagination.CursorEntity            = (*RecurringShipment)(nil)
)

const (
	MaxLeadTimeDays  = 60
	MaxBlackoutDates = 100
	blackoutDateFmt  = "2006-01-02"
)

type RecurringShipment struct {
	keel.Table `keel:"recurring_shipments"`

	ID                      pulid.ID        `json:"id" keel:"id,pk,autoid:rsh_"`
	BusinessUnitID          pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID          pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	SourceShipmentID        pulid.ID        `json:"sourceShipmentId" keel:"source_shipment_id"`
	CustomerID              pulid.ID        `json:"customerId" keel:"customer_id,nullzero"`
	OriginLocationID        pulid.ID        `json:"originLocationId" keel:"origin_location_id,nullzero"`
	DestinationLocationID   pulid.ID        `json:"destinationLocationId" keel:"destination_location_id,nullzero"`
	EnteredByID             pulid.ID        `json:"enteredById" keel:"entered_by_id,nullzero"`
	LastGeneratedShipmentID pulid.ID        `json:"lastGeneratedShipmentId" keel:"last_generated_shipment_id,nullzero"`
	Name                    string          `json:"name" keel:"name"`
	Description             string          `json:"description" keel:"description,nullzero"`
	Status                  Status          `json:"status" keel:"status"`
	CronExpression          string          `json:"cronExpression" keel:"cron_expression"`
	Timezone                string          `json:"timezone" keel:"timezone"`
	StartDate               int64           `json:"startDate" keel:"start_date,nullzero"`
	EndDate                 *int64          `json:"endDate" keel:"end_date"`
	MaxOccurrences          *int32          `json:"maxOccurrences" keel:"max_occurrences"`
	LeadTimeDays            int16           `json:"leadTimeDays" keel:"lead_time_days"`
	SkipWeekends            bool            `json:"skipWeekends" keel:"skip_weekends"`
	ExceptionPolicy         ExceptionPolicy `json:"exceptionPolicy" keel:"exception_policy"`
	BlackoutDates           []string        `json:"blackoutDates" keel:"blackout_dates,nullzero"`
	AutoGenerate            bool            `json:"autoGenerate" keel:"auto_generate"`
	NextOccurrenceAt        *int64          `json:"nextOccurrenceAt" keel:"next_occurrence_at"`
	NextOccurrenceSourceAt  *int64          `json:"nextOccurrenceSourceAt" keel:"next_occurrence_source_at"`
	LastOccurrenceAt        *int64          `json:"lastOccurrenceAt" keel:"last_occurrence_at"`
	LastRunAt               *int64          `json:"lastRunAt" keel:"last_run_at"`
	GenerationCount         int64           `json:"generationCount" keel:"generation_count"`
	ConsecutiveFailures     int32           `json:"consecutiveFailures" keel:"consecutive_failures"`
	Version                 int64           `json:"version" keel:"version,version"`
	CreatedAt               int64           `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt               int64           `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit        *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization        *tenant.Organization `json:"organization,omitempty"`
	SourceShipment      *shipment.Shipment   `json:"sourceShipment,omitempty" keel:"belongs:source_shipment_id"`
	Customer            *customer.Customer   `json:"customer,omitempty" keel:"belongs:customer_id"`
	OriginLocation      *location.Location   `json:"originLocation,omitempty" keel:"belongs:origin_location_id"`
	DestinationLocation *location.Location   `json:"destinationLocation,omitempty" keel:"belongs:destination_location_id"`
	EnteredBy           *tenant.User         `json:"enteredBy,omitempty" keel:"belongs:entered_by_id"`
}

func (rs *RecurringShipment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		rs,
		validation.Field(
			&rs.SourceShipmentID,
			validation.Required.Error("Source shipment is required"),
		),
		validation.Field(
			&rs.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(
			&rs.Status,
			validation.Required.Error("Status is required"),
			validation.In(StatusActive, StatusPaused, StatusExpired).
				Error("Status must be a valid status"),
		),
		validation.Field(
			&rs.CronExpression,
			validation.Required.Error("Schedule is required"),
		),
		validation.Field(&rs.Timezone, validation.Required.Error("Timezone is required")),
		validation.Field(
			&rs.ExceptionPolicy,
			validation.Required.Error("Exception policy is required"),
			validation.In(
				ExceptionPolicySkip,
				ExceptionPolicyPreviousBusinessDay,
				ExceptionPolicyNextBusinessDay,
			).Error("Exception policy must be a valid policy"),
		),
	))

	rs.validateSchedule(multiErr)
}

func (rs *RecurringShipment) validateSchedule(multiErr *errortypes.MultiError) {
	if rs.CronExpression != "" {
		if err := cronutils.Validate(rs.CronExpression); err != nil {
			multiErr.Add(
				"cronExpression",
				errortypes.ErrInvalid,
				"Schedule is not a valid cron expression",
			)
		}
	}

	if rs.Timezone != "" {
		if _, err := time.LoadLocation(rs.Timezone); err != nil {
			multiErr.Add(
				"timezone",
				errortypes.ErrInvalid,
				"Timezone must be a valid IANA timezone",
			)
		}
	}

	if rs.LeadTimeDays < 0 || rs.LeadTimeDays > MaxLeadTimeDays {
		multiErr.Add(
			"leadTimeDays",
			errortypes.ErrInvalid,
			"Lead time must be between 0 and {0} days", MaxLeadTimeDays,
		)
	}

	if rs.EndDate != nil && rs.StartDate > 0 && *rs.EndDate <= rs.StartDate {
		multiErr.Add("endDate", errortypes.ErrInvalid, "End date must be after the start date")
	}

	if rs.MaxOccurrences != nil && *rs.MaxOccurrences < 1 {
		multiErr.Add("maxOccurrences", errortypes.ErrInvalid, "Max occurrences must be at least 1")
	}

	if len(rs.BlackoutDates) > MaxBlackoutDates {
		multiErr.Add(
			"blackoutDates",
			errortypes.ErrInvalid,
			"A series supports at most {0} blackout dates", MaxBlackoutDates,
		)
	}

	for i, blackoutDate := range rs.BlackoutDates {
		if _, err := time.Parse(blackoutDateFmt, blackoutDate); err != nil {
			multiErr.Add(
				fmt.Sprintf("blackoutDates[%d]", i),
				errortypes.ErrInvalid,
				"Blackout dates must use the YYYY-MM-DD format",
			)
		}
	}
}

func (rs *RecurringShipment) GetID() pulid.ID {
	return rs.ID
}

func (rs *RecurringShipment) GetCreatedAt() int64 {
	return rs.CreatedAt
}

func (rs *RecurringShipment) GetTableName() string {
	return "recurring_shipments"
}

func (rs *RecurringShipment) GetOrganizationID() pulid.ID {
	return rs.OrganizationID
}

func (rs *RecurringShipment) GetBusinessUnitID() pulid.ID {
	return rs.BusinessUnitID
}

func (rs *RecurringShipment) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rsh",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{Field: "customer", Queryable: true},
			{Field: "originLocation", Queryable: true},
			{Field: "destinationLocation", Queryable: true},
		},
	}
}

func (rs *RecurringShipment) IsActive() bool {
	return rs.Status == StatusActive
}

func (rs *RecurringShipment) BeforeCreate(_ context.Context) error {
	if rs.Status == "" {
		rs.Status = StatusActive
	}
	if rs.LeadTimeDays == 0 {
		rs.LeadTimeDays = 1
	}
	if rs.ExceptionPolicy == "" {
		rs.ExceptionPolicy = ExceptionPolicySkip
	}

	return nil
}
