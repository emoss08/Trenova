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
	keel.Table            `                                       keel:"workers,report=Parties"`
	ID                    pulid.ID           `json:"id" keel:"id,pk,autoid:wrk_"`
	BusinessUnitID        pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID        pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	StateID               pulid.ID           `json:"stateId" keel:"state_id"`
	FleetCodeID           pulid.ID           `json:"fleetCodeId" keel:"fleet_code_id,nullzero"`
	ManagerID             pulid.ID           `json:"managerId" keel:"manager_id,nullzero"`
	PositionID            pulid.ID           `json:"positionId" keel:"position_id,nullzero"`
	UserID                pulid.ID           `json:"userId" keel:"user_id,nullzero"`
	Status                domaintypes.Status `json:"status" keel:"status"`
	Type                  WorkerType         `json:"type" keel:"type"`
	DriverType            DriverType         `json:"driverType" keel:"driver_type"`
	LeaveType             LeaveType          `json:"leaveType" keel:"leave_type,nullzero"`
	ProfilePicURL         string             `json:"profilePicUrl" keel:"profile_pic_url,nullzero"`
	FirstName             string             `json:"firstName" keel:"first_name"`
	LastName              string             `json:"lastName" keel:"last_name"`
	WholeName             string             `json:"wholeName" keel:"whole_name,readonly,label=Full Name"`
	AddressLine1          string             `json:"addressLine1" keel:"address_line1"`
	AddressLine2          string             `json:"addressLine2" keel:"address_line2,nullzero"`
	City                  string             `json:"city" keel:"city"`
	PostalCode            string             `json:"postalCode" keel:"postal_code"`
	Email                 string             `json:"email" keel:"email,nullzero"`
	PhoneNumber           string             `json:"phoneNumber" keel:"phone_number,nullzero"`
	EmergencyContactName  string             `json:"emergencyContactName" keel:"emergency_contact_name,nullzero"`
	EmergencyContactPhone string             `json:"emergencyContactPhone" keel:"emergency_contact_phone,nullzero"`
	ExternalID            string             `json:"externalId" keel:"external_id,nullzero"`
	SearchVector          string             `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank                  string             `json:"-" keel:"rank,computed"`
	AssignmentBlocked     string             `json:"assignmentBlocked,omitempty" keel:"assignment_blocked,nullzero"`
	Gender                Gender             `json:"gender" keel:"gender"`
	CanBeAssigned         bool               `json:"canBeAssigned" keel:"can_be_assigned"`
	AvailableForDispatch  bool               `json:"availableForDispatch" keel:"available_for_dispatch"`
	Version               int64              `json:"version" keel:"version,version"`
	CreatedAt             int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt             int64              `json:"updatedAt" keel:"updated_at,autoupdate"`
	CustomFields          map[string]any     `json:"customFields,omitempty"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" keel:"belongs:business_unit_id,noreport"`
	Organization *tenant.Organization `json:"organization,omitempty" keel:"belongs:organization_id,noreport"`
	State        *usstate.UsState     `json:"state,omitempty" keel:"belongs:state_id,noreport"`
	FleetCode    *fleetcode.FleetCode `json:"fleetCode,omitempty" keel:"belongs:fleet_code_id"`
	Manager      *tenant.User         `json:"manager,omitempty" keel:"belongs:manager_id,noreport"`
	Position     *JobPosition         `json:"position,omitempty" keel:"belongs:position_id"`
	PortalUser   *tenant.User         `json:"portalUser,omitempty" keel:"belongs:user_id,noreport"`
	Profile      *WorkerProfile       `json:"profile,omitempty" keel:"one:worker_id"`
	PTO          []*WorkerPTO         `json:"pto,omitempty" keel:"many:worker_id,label=Time Off"`
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
