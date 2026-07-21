package recurringshipment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*RecurringShipment)(nil)
	_ validationframework.TenantedEntity = (*RecurringShipment)(nil)
	_ pagination.CursorEntity            = (*RecurringShipment)(nil)
)

const (
	MaxLeadTimeDays  = 60
	MaxBlackoutDates = 100
	blackoutDateFmt  = "2006-01-02"
)

type RecurringShipment struct {
	bun.BaseModel             `json:"-" bun:"table:recurring_shipments,alias:rsh"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                      pulid.ID        `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID        `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID        `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SourceShipmentID        pulid.ID        `json:"sourceShipmentId"        bun:"source_shipment_id,type:VARCHAR(100),notnull"`
	CustomerID              pulid.ID        `json:"customerId"              bun:"customer_id,type:VARCHAR(100),nullzero"`
	OriginLocationID        pulid.ID        `json:"originLocationId"        bun:"origin_location_id,type:VARCHAR(100),nullzero"`
	DestinationLocationID   pulid.ID        `json:"destinationLocationId"   bun:"destination_location_id,type:VARCHAR(100),nullzero"`
	EnteredByID             pulid.ID        `json:"enteredById"             bun:"entered_by_id,type:VARCHAR(100),nullzero"`
	LastGeneratedShipmentID pulid.ID        `json:"lastGeneratedShipmentId" bun:"last_generated_shipment_id,type:VARCHAR(100),nullzero"`
	Name                    string          `json:"name"                    bun:"name,type:VARCHAR(100),notnull"`
	Description             string          `json:"description"             bun:"description,type:TEXT,nullzero"`
	Status                  Status          `json:"status"                  bun:"status,type:recurring_shipment_status_enum,notnull,default:'Active'"`
	CronExpression          string          `json:"cronExpression"          bun:"cron_expression,type:VARCHAR(100),notnull"`
	Timezone                string          `json:"timezone"                bun:"timezone,type:VARCHAR(64),notnull"`
	StartDate               int64           `json:"startDate"               bun:"start_date,type:BIGINT,nullzero"`
	EndDate                 *int64          `json:"endDate"                 bun:"end_date,type:BIGINT,nullzero"`
	MaxOccurrences          *int32          `json:"maxOccurrences"          bun:"max_occurrences,type:INTEGER,nullzero"`
	LeadTimeDays            int16           `json:"leadTimeDays"            bun:"lead_time_days,type:SMALLINT,notnull,default:1"`
	SkipWeekends            bool            `json:"skipWeekends"            bun:"skip_weekends,type:BOOLEAN,notnull,default:false"`
	ExceptionPolicy         ExceptionPolicy `json:"exceptionPolicy"         bun:"exception_policy,type:recurring_shipment_exception_policy_enum,notnull,default:'Skip'"`
	BlackoutDates           []string        `json:"blackoutDates"           bun:"blackout_dates,type:TEXT[],array,nullzero"`
	AutoGenerate            bool            `json:"autoGenerate"            bun:"auto_generate,type:BOOLEAN,notnull,default:true"`
	NextOccurrenceAt        *int64          `json:"nextOccurrenceAt"        bun:"next_occurrence_at,type:BIGINT,nullzero"`
	NextOccurrenceSourceAt  *int64          `json:"nextOccurrenceSourceAt"  bun:"next_occurrence_source_at,type:BIGINT,nullzero"`
	LastOccurrenceAt        *int64          `json:"lastOccurrenceAt"        bun:"last_occurrence_at,type:BIGINT,nullzero"`
	LastRunAt               *int64          `json:"lastRunAt"               bun:"last_run_at,type:BIGINT,nullzero"`
	GenerationCount         int64           `json:"generationCount"         bun:"generation_count,type:BIGINT,notnull,default:0"`
	ConsecutiveFailures     int32           `json:"consecutiveFailures"     bun:"consecutive_failures,type:INTEGER,notnull,default:0"`
	Version                 int64           `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt               int64           `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64           `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit        *tenant.BusinessUnit `json:"businessUnit,omitempty"        bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization        *tenant.Organization `json:"organization,omitempty"        bun:"rel:belongs-to,join:organization_id=id"`
	SourceShipment      *shipment.Shipment   `json:"sourceShipment,omitempty"      bun:"rel:belongs-to,join:source_shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Customer            *customer.Customer   `json:"customer,omitempty"            bun:"rel:belongs-to,join:customer_id=id"`
	OriginLocation      *location.Location   `json:"originLocation,omitempty"      bun:"rel:belongs-to,join:origin_location_id=id"`
	DestinationLocation *location.Location   `json:"destinationLocation,omitempty" bun:"rel:belongs-to,join:destination_location_id=id"`
	EnteredBy           *tenant.User         `json:"enteredBy,omitempty"           bun:"rel:belongs-to,join:entered_by_id=id"`
}

func (rs *RecurringShipment) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		rs,
		validation.Field(&rs.SourceShipmentID, validation.Required.Error("Source shipment is required")),
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	rs.validateSchedule(multiErr)
}

func (rs *RecurringShipment) validateSchedule(multiErr *errortypes.MultiError) {
	if rs.CronExpression != "" {
		if err := cronutils.Validate(rs.CronExpression); err != nil {
			multiErr.Add("cronExpression", errortypes.ErrInvalid, "Schedule is not a valid cron expression")
		}
	}

	if rs.Timezone != "" {
		if _, err := time.LoadLocation(rs.Timezone); err != nil {
			multiErr.Add("timezone", errortypes.ErrInvalid, "Timezone must be a valid IANA timezone")
		}
	}

	if rs.LeadTimeDays < 0 || rs.LeadTimeDays > MaxLeadTimeDays {
		multiErr.Add(
			"leadTimeDays",
			errortypes.ErrInvalid,
			fmt.Sprintf("Lead time must be between 0 and %d days", MaxLeadTimeDays),
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
			fmt.Sprintf("A series supports at most %d blackout dates", MaxBlackoutDates),
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
			{
				Field:        "customer",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*customer.Customer)(nil),
				TargetTable:  "customers",
				ForeignKey:   "customer_id",
				ReferenceKey: "id",
				Alias:        "cus",
				Queryable:    true,
			},
			{
				Field:        "originLocation",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*location.Location)(nil),
				TargetTable:  "locations",
				ForeignKey:   "origin_location_id",
				ReferenceKey: "id",
				Alias:        "orig_loc",
				Queryable:    true,
			},
			{
				Field:        "destinationLocation",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*location.Location)(nil),
				TargetTable:  "locations",
				ForeignKey:   "destination_location_id",
				ReferenceKey: "id",
				Alias:        "dest_loc",
				Queryable:    true,
			},
		},
	}
}

func (rs *RecurringShipment) IsActive() bool {
	return rs.Status == StatusActive
}

func (rs *RecurringShipment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rs.ID.IsNil() {
			rs.ID = pulid.MustNew("rsh_")
		}

		rs.CreatedAt = now
	case *bun.UpdateQuery:
		rs.UpdatedAt = now
	}

	return nil
}
