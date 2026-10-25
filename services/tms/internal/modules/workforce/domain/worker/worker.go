package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	_ domaintypes.PostgresSearchable     = (*Worker)(nil)
	_ validationframework.TenantedEntity = (*Worker)(nil)
	_ customfield.CustomFieldsSupporter  = (*Worker)(nil)
	_ pagination.CursorEntity            = (*Worker)(nil)
)

// Drivers and operational workers
type Worker struct {
	keel.Table            `keel:"workers,report=Parties"`
	ID                    pulid.ID           `keel:"id,pk,autoid:wrk_"                        json:"id"`
	BusinessUnitID        pulid.ID           `keel:"business_unit_id,pk,tenant"               json:"businessUnitId"`
	OrganizationID        pulid.ID           `keel:"organization_id,pk,tenant"                json:"organizationId"`
	StateID               pulid.ID           `keel:"state_id"                                 json:"stateId"`
	FleetCodeID           pulid.ID           `keel:"fleet_code_id,nullzero"                   json:"fleetCodeId"`
	ManagerID             pulid.ID           `keel:"manager_id,nullzero"                      json:"managerId"`
	PositionID            pulid.ID           `keel:"position_id,nullzero"                     json:"positionId"`
	UserID                pulid.ID           `keel:"user_id,nullzero"                         json:"userId"`
	Status                domaintypes.Status `keel:"status"                                   json:"status"`
	Type                  WorkerType         `keel:"type"                                     json:"type"`
	DriverType            DriverType         `keel:"driver_type"                              json:"driverType"`
	LeaveType             LeaveType          `keel:"leave_type,nullzero"                      json:"leaveType"`
	ProfilePicURL         string             `keel:"profile_pic_url,nullzero"                 json:"profilePicUrl"`
	FirstName             string             `keel:"first_name"                               json:"firstName"`
	LastName              string             `keel:"last_name"                                json:"lastName"`
	WholeName             string             `keel:"whole_name,readonly,label=Full Name"      json:"wholeName"`
	AddressLine1          string             `keel:"address_line1"                            json:"addressLine1"`
	AddressLine2          string             `keel:"address_line2,nullzero"                   json:"addressLine2"`
	City                  string             `keel:"city"                                     json:"city"`
	PostalCode            string             `keel:"postal_code"                              json:"postalCode"`
	Email                 string             `keel:"email,nullzero"                           json:"email"`
	PhoneNumber           string             `keel:"phone_number,nullzero"                    json:"phoneNumber"`
	EmergencyContactName  string             `keel:"emergency_contact_name,nullzero"          json:"emergencyContactName"`
	EmergencyContactPhone string             `keel:"emergency_contact_phone,nullzero"         json:"emergencyContactPhone"`
	ExternalID            string             `keel:"external_id,nullzero"                     json:"externalId"`
	SearchVector          string             `keel:"search_vector,tsvector,readonly,noselect" json:"-"`
	Rank                  string             `keel:"rank,computed"                            json:"-"`
	AssignmentBlocked     string             `keel:"assignment_blocked,nullzero"              json:"assignmentBlocked,omitempty"`
	Gender                Gender             `keel:"gender"                                   json:"gender"`
	CanBeAssigned         bool               `keel:"can_be_assigned"                          json:"canBeAssigned"`
	AvailableForDispatch  bool               `keel:"available_for_dispatch"                   json:"availableForDispatch"`
	Version               int64              `keel:"version,version"                          json:"version"`
	CreatedAt             int64              `keel:"created_at,autocreate"                    json:"createdAt"`
	UpdatedAt             int64              `keel:"updated_at,autoupdate"                    json:"updatedAt"`
	CustomFields          map[string]any     `                                                json:"customFields,omitempty"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" keel:"belongs:business_unit_id,noreport"`
	Organization *tenant.Organization `json:"organization,omitempty" keel:"belongs:organization_id,noreport"`
	State        *usstate.UsState     `json:"state,omitempty"        keel:"belongs:state_id,noreport"`
	FleetCode    *fleetcode.FleetCode `json:"fleetCode,omitempty"    keel:"belongs:fleet_code_id"`
	Manager      *tenant.User         `json:"manager,omitempty"      keel:"belongs:manager_id,noreport"`
	Position     *JobPosition         `json:"position,omitempty"     keel:"belongs:position_id"`
	PortalUser   *tenant.User         `json:"portalUser,omitempty"   keel:"belongs:user_id,noreport"`
	Profile      *WorkerProfile       `json:"profile,omitempty"      keel:"one:worker_id"`
	PTO          []*WorkerPTO         `json:"pto,omitempty"          keel:"many:worker_id,label=Time Off"`
}

func (w *Worker) BeforeCreate(_ context.Context) error {
	if w.Status == "" {
		w.Status = domaintypes.StatusActive
	}
	if w.Type == "" {
		w.Type = WorkerTypeEmployee
	}
	if w.DriverType == "" {
		w.DriverType = DriverTypeOTR
	}

	return nil
}

func (w *Worker) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(w,
		validation.Field(&w.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[WorkerType]("type must be either Employee or Contractor"),
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
			domainvalidation.ValidEnum[Gender]("Gender must be either Male or Female"),
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
			validation.By(domaintypes.ValidatePostalCode),
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
	))

	if w.Profile != nil {
		profileErr := multiErr.WithPrefix("profile")
		w.Profile.Validate(profileErr)
	}
}

func (w *Worker) GetTableName() string {
	return "workers"
}

func (w *Worker) GetID() pulid.ID {
	return w.ID
}

func (w *Worker) GetCreatedAt() int64 {
	return w.CreatedAt
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
		Relationships: []*domaintypes.RelationshipDefintion{
			{Field: "Profile", Queryable: true},
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

func (w *Worker) AfterLoad(_ context.Context) error {
	if w.WholeName == "" && (w.FirstName != "" || w.LastName != "") {
		w.WholeName = strings.TrimSpace(w.FirstName + " " + w.LastName)
	}

	return nil
}

func (w *Worker) SamePartyDetails(other *Worker) bool {
	if w == nil || other == nil {
		return w == other
	}
	return w.FirstName == other.FirstName &&
		w.LastName == other.LastName &&
		w.Email == other.Email &&
		w.AddressLine1 == other.AddressLine1 &&
		w.City == other.City &&
		w.StateID == other.StateID &&
		w.PostalCode == other.PostalCode
}
