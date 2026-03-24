package worker

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var usPostalCodeRegex = regexp.MustCompile(`^\d{5}(-\d{4})?$`)

var (
	_ bun.BeforeAppendModelHook          = (*Worker)(nil)
	_ domaintypes.PostgresSearchable     = (*Worker)(nil)
	_ validationframework.TenantedEntity = (*Worker)(nil)
	_ customfield.CustomFieldsSupporter  = (*Worker)(nil)
)

type Worker struct {
	bun.BaseModel `bun:"table:workers,alias:wrk" json:"-"`

	ID                    pulid.ID           `json:"id"                          bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID        pulid.ID           `json:"businessUnitId"              bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID        pulid.ID           `json:"organizationId"              bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	StateID               pulid.ID           `json:"stateId"                     bun:"state_id,type:VARCHAR(100),notnull"`
	FleetCodeID           pulid.ID           `json:"fleetCodeId"                 bun:"fleet_code_id,type:VARCHAR(100),nullzero"`
	ManagerID             pulid.ID           `json:"managerId"                   bun:"manager_id,type:VARCHAR(100),nullzero"`
	Status                domaintypes.Status `json:"status"                      bun:"status,type:status_enum,notnull,default:'Active'"`
	Type                  WorkerType         `json:"type"                        bun:"type,type:worker_type_enum,notnull,default:'Employee'"`
	DriverType            DriverType         `json:"driverType"                  bun:"driver_type,type:driver_type_enum,notnull,default:'OTR'"`
	ProfilePicURL         string             `json:"profilePicUrl"               bun:"profile_pic_url,type:VARCHAR(255),nullzero"`
	FirstName             string             `json:"firstName"                   bun:"first_name,type:VARCHAR(100),notnull"`
	LastName              string             `json:"lastName"                    bun:"last_name,type:VARCHAR(100),notnull"`
	WholeName             string             `json:"wholeName"                   bun:"whole_name,type:VARCHAR(201),scanonly"`
	AddressLine1          string             `json:"addressLine1"                bun:"address_line1,type:VARCHAR(150),notnull"`
	AddressLine2          string             `json:"addressLine2"                bun:"address_line2,type:VARCHAR(150),nullzero"`
	City                  string             `json:"city"                        bun:"city,type:VARCHAR(100),notnull"`
	PostalCode            string             `json:"postalCode"                  bun:"postal_code,type:us_postal_code,notnull"`
	Email                 string             `json:"email"                       bun:"email,type:VARCHAR(255),nullzero"`
	PhoneNumber           string             `json:"phoneNumber"                 bun:"phone_number,type:VARCHAR(20),nullzero"`
	EmergencyContactName  string             `json:"emergencyContactName"        bun:"emergency_contact_name,type:VARCHAR(100),nullzero"`
	EmergencyContactPhone string             `json:"emergencyContactPhone"       bun:"emergency_contact_phone,type:VARCHAR(20),nullzero"`
	ExternalID            string             `json:"externalId"                  bun:"external_id,type:TEXT,nullzero"`
	SearchVector          string             `json:"-"                           bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                  string             `json:"-"                           bun:"rank,type:VARCHAR(100),scanonly"`
	AssignmentBlocked     string             `json:"assignmentBlocked,omitempty" bun:"assignment_blocked,type:VARCHAR(255),nullzero"`
	Gender                Gender             `json:"gender"                      bun:"gender,type:gender_enum,notnull"`
	CanBeAssigned         bool               `json:"canBeAssigned"               bun:"can_be_assigned,type:BOOLEAN,notnull,default:false"`
	AvailableForDispatch  bool               `json:"availableForDispatch"        bun:"available_for_dispatch,type:BOOLEAN,notnull,default:true"`
	Version               int64              `json:"version"                     bun:"version,type:BIGINT"`
	CreatedAt             int64              `json:"createdAt"                   bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64              `json:"updatedAt"                   bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	CustomFields          map[string]any     `json:"customFields,omitempty"      bun:"-"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	State        *usstate.UsState     `json:"state,omitempty"        bun:"rel:belongs-to,join:state_id=id"`
	FleetCode    *fleetcode.FleetCode `json:"fleetCode,omitempty"    bun:"rel:belongs-to,join:fleet_code_id=id"`
	Manager      *tenant.User         `json:"manager,omitempty"      bun:"rel:belongs-to,join:manager_id=id"`
	Profile      *WorkerProfile       `json:"profile,omitempty"      bun:"rel:has-one,join:id=worker_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	PTO          []*WorkerPTO         `json:"pto,omitempty"          bun:"rel:has-many,join:id=worker_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (w *Worker) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(w,
		validation.Field(&w.Type,
			validation.Required.Error("Type is required"),
			validation.By(func(value any) error {
				t, ok := value.(WorkerType)
				if !ok {
					return errors.New("invalid worker type")
				}
				if !t.IsValid() {
					return errors.New("type must be either Employee or Contractor")
				}
				return nil
			}),
		),
		validation.Field(&w.FirstName,
			validation.Required.Error("First Name is required"),
			validation.Length(1, 100).Error("First Name must be between 1 and 100 characters"),
		),
		validation.Field(&w.LastName,
			validation.Required.Error("Last Name is required"),
			validation.Length(1, 100).Error("Last Name must be between 1 and 100 characters"),
		),
		validation.Field(&w.Gender,
			validation.Required.Error("Gender is required"),
			validation.By(func(value any) error {
				g, ok := value.(Gender)
				if !ok {
					return errors.New("invalid gender type")
				}
				if !g.IsValid() {
					return errors.New("Gender must be either Male or Female")
				}
				return nil
			}),
		),
		validation.Field(&w.AddressLine1,
			validation.Required.Error("Address Line 1 is required"),
			validation.Length(1, 150).Error("Address Line 1 must be between 1 and 150 characters"),
		),
		validation.Field(&w.City,
			validation.Required.Error("City is required"),
			validation.Length(1, 100).Error("City must be between 1 and 100 characters"),
		),
		validation.Field(&w.PostalCode,
			validation.Required.Error("Postal Code is required"),
			validation.By(validatePostalCode),
		),
		validation.Field(&w.PhoneNumber,
			is.E164.Error("Phone number must be in E.164 format (e.g., +12025551234)"),
		),
		validation.Field(&w.EmergencyContactPhone,
			is.E164.Error("Phone number must be in E.164 format (e.g., +12025551234)"),
		),
		validation.Field(&w.StateID,
			validation.Required.Error("State is required"),
			validation.By(func(value any) error {
				id, ok := value.(pulid.ID)
				if !ok {
					return errors.New("invalid state ID type")
				}
				if id.IsNil() {
					return errors.New("state is required")
				}
				return nil
			}),
		),
		validation.Field(&w.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
		validation.Field(&w.Profile, validation.Required.Error("Worker profile must be provided")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if w.Profile != nil {
		profileErr := multiErr.WithPrefix("profile")
		w.Profile.Validate(profileErr)
	}
}

func validatePostalCode(value any) error {
	pc, ok := value.(string)
	if !ok {
		return errors.New("postal code must be a string")
	}
	if pc == "" {
		return nil
	}
	if !usPostalCodeRegex.MatchString(pc) {
		return errors.New("postal code must be a valid US postal code (e.g., 12345 or 12345-6789)")
	}
	return nil
}

func (w *Worker) GetTableName() string {
	return "workers"
}

func (w *Worker) GetID() pulid.ID {
	return w.ID
}

func (w *Worker) GetOrganizationID() pulid.ID {
	return w.OrganizationID
}

func (w *Worker) GetBusinessUnitID() pulid.ID {
	return w.BusinessUnitID
}

func (w *Worker) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "wrk",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "first_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "last_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "type",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (w *Worker) FullName() string {
	return fmt.Sprintf("%s %s", w.FirstName, w.LastName)
}

func (w *Worker) GetResourceType() string {
	return "worker"
}

func (w *Worker) GetResourceID() string {
	return w.ID.String()
}

func (w *Worker) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if w.ID.IsNil() {
			w.ID = pulid.MustNew("wrk_")
		}
		w.CreatedAt = now
	case *bun.UpdateQuery:
		w.UpdatedAt = now
	}

	return nil
}
