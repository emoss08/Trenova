package dedicatedlane

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*DedicatedLane)(nil)
	_ domain.Validatable             = (*DedicatedLane)(nil)
	_ framework.TenantedEntity       = (*DedicatedLane)(nil)
	_ domaintypes.PostgresSearchable = (*DedicatedLane)(nil)
)

type DedicatedLane struct {
	bun.BaseModel `bun:"table:dedicated_lanes,alias:dl" json:"-"`

	ID                    pulid.ID      `json:"id"                         bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID        pulid.ID      `json:"businessUnitId"             bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID        pulid.ID      `json:"organizationId"             bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Name                  string        `json:"name"                       bun:"name,type:VARCHAR(100),notnull"`
	Status                domain.Status `json:"status"                     bun:"status,type:status_enum,notnull,default:'Active'"`
	CustomerID            pulid.ID      `json:"customerId"                 bun:"customer_id,type:VARCHAR(100),notnull"`
	OriginLocationID      pulid.ID      `json:"originLocationId"           bun:"origin_location_id,type:VARCHAR(100),notnull"`
	DestinationLocationID pulid.ID      `json:"destinationLocationId"      bun:"destination_location_id,type:VARCHAR(100),notnull"`
	ServiceTypeID         pulid.ID      `json:"serviceTypeId"              bun:"service_type_id,type:VARCHAR(100),notnull"`
	ShipmentTypeID        pulid.ID      `json:"shipmentTypeId"             bun:"shipment_type_id,type:VARCHAR(100),notnull"`
	PrimaryWorkerID       *pulid.ID     `json:"primaryWorkerId,omitzero"   bun:"primary_worker_id,type:VARCHAR(100),nullzero"`
	SecondaryWorkerID     *pulid.ID     `json:"secondaryWorkerId,omitzero" bun:"secondary_worker_id,type:VARCHAR(100),nullzero"`
	TrailerTypeID         *pulid.ID     `json:"trailerTypeId,omitzero"     bun:"trailer_type_id,type:VARCHAR(100),nullzero"`
	TractorTypeID         *pulid.ID     `json:"tractorTypeId,omitzero"     bun:"tractor_type_id,type:VARCHAR(100),nullzero"`
	AutoAssign            bool          `json:"autoAssign"                 bun:"auto_assign,type:BOOLEAN,notnull,default:false"`
	Version               int64         `json:"version"                    bun:"version,type:BIGINT"`
	CreatedAt             int64         `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64         `json:"updatedAt"                  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit        *tenant.BusinessUnit         `json:"businessUnit,omitzero"        bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization        *tenant.Organization         `json:"organization,omitzero"        bun:"rel:belongs-to,join:organization_id=id"`
	ShipmentType        *shipmenttype.ShipmentType   `json:"shipmentType,omitzero"        bun:"rel:belongs-to,join:shipment_type_id=id"`
	ServiceType         *servicetype.ServiceType     `json:"serviceType,omitzero"         bun:"rel:belongs-to,join:service_type_id=id"`
	Customer            *customer.Customer           `json:"customer,omitzero"            bun:"rel:belongs-to,join:customer_id=id"`
	TractorType         *equipmenttype.EquipmentType `json:"tractorType,omitzero"         bun:"rel:belongs-to,join:tractor_type_id=id"`
	TrailerType         *equipmenttype.EquipmentType `json:"trailerType,omitzero"         bun:"rel:belongs-to,join:trailer_type_id=id"`
	OriginLocation      *location.Location           `json:"originLocation,omitzero"      bun:"rel:belongs-to,join:origin_location_id=id"`
	DestinationLocation *location.Location           `json:"destinationLocation,omitzero" bun:"rel:belongs-to,join:destination_location_id=id"`
	PrimaryWorker       *worker.Worker               `json:"primaryWorker,omitzero"       bun:"rel:belongs-to,join:primary_worker_id=id"`
	SecondaryWorker     *worker.Worker               `json:"secondaryWorker,omitzero"     bun:"rel:belongs-to,join:secondary_worker_id=id"`
}

func (d *DedicatedLane) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		d,
		validation.Field(&d.Name,
			validation.Required.Error("Name is required"),
			validation.Length(2, 100).Error("Name must be between 2 and 100 characters"),
		),
		validation.Field(&d.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(
			&d.DestinationLocationID,
			validation.Required.Error("Destination Location is required"),
			validation.When(
				pulid.Equals(d.OriginLocationID, d.DestinationLocationID),
				validation.Required.Error("Origin and Destination cannot be the same location"),
			),
		),
		validation.Field(
			&d.AutoAssign,
			validation.When(
				d.AutoAssign && d.PrimaryWorkerID.IsNil(),
				validation.Required.Error("Primary worker is required when auto assign is true"),
			),
		),
		validation.Field(&d.SecondaryWorkerID,
			validation.When(
				d.SecondaryWorkerID != nil && d.PrimaryWorkerID != nil &&
					pulid.Equals(*d.PrimaryWorkerID, *d.SecondaryWorkerID),
				validation.Required.Error("Primary and Secondary Workers cannot be the same"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (d *DedicatedLane) GetID() string {
	return d.ID.String()
}

func (d *DedicatedLane) GetTableName() string {
	return "dedicated_lanes"
}

func (d *DedicatedLane) GetOrganizationID() pulid.ID {
	return d.OrganizationID
}

func (d *DedicatedLane) GetBusinessUnitID() pulid.ID {
	return d.BusinessUnitID
}

func (d *DedicatedLane) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dl",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (d *DedicatedLane) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("dl_")
		}

		d.CreatedAt = now

	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}
