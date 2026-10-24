package checkcall

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CheckCall struct {
	keel.Table `keel:"check_calls"`

	ID               pulid.ID `json:"id" keel:"id,pk"`
	OrganizationID   pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ShipmentID       pulid.ID `json:"shipmentId" keel:"shipment_id"`
	ShipmentMoveID   pulid.ID `json:"shipmentMoveId" keel:"shipment_move_id"`
	StopID           pulid.ID `json:"stopId" keel:"stop_id,nullzero"`
	Source           Source   `json:"source" keel:"source"`
	SourceKey        string   `json:"sourceKey" keel:"source_key"`
	Kind             Kind     `json:"kind" keel:"kind"`
	ReportedAt       int64    `json:"reportedAt" keel:"reported_at"`
	ReceivedAt       int64    `json:"receivedAt" keel:"received_at"`
	Latitude         *float64 `json:"latitude" keel:"latitude"`
	Longitude        *float64 `json:"longitude" keel:"longitude"`
	City             string   `json:"city" keel:"city,nullzero"`
	StateCode        string   `json:"stateCode" keel:"state_code,nullzero"`
	Delayed          bool     `json:"delayed" keel:"delayed"`
	DelayReason      string   `json:"delayReason" keel:"delay_reason,nullzero"`
	EstimatedArrival *int64   `json:"estimatedArrival" keel:"estimated_arrival"`
	Note             string   `json:"note" keel:"note,nullzero"`
	ContactName      string   `json:"contactName" keel:"contact_name,nullzero"`
	ReportedByID     pulid.ID `json:"reportedById" keel:"reported_by_id,nullzero"`
	TrackingLinkID   pulid.ID `json:"trackingLinkId" keel:"tracking_link_id,nullzero"`
	TrackingEventID  pulid.ID `json:"trackingEventId" keel:"tracking_event_id,nullzero"`
	CreatedAt        int64    `json:"createdAt" keel:"created_at"`
}

func NewID() pulid.ID {
	return pulid.MustNew("chkc_")
}

func (c *CheckCall) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&c.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&c.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&c.ShipmentMoveID, validation.Required.Error("Move is required")),
		validation.Field(&c.StopID,
			validation.When(c.Kind.IsStopVisit(),
				validation.Required.Error("Choose the stop that was reached or left"),
			),
		),
		validation.Field(&c.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[Source]("Source is invalid"),
		),
		validation.Field(&c.SourceKey,
			validation.Required.Error("Source key is required"),
			validation.RuneLength(1, MaxSourceKeyLength).
				Error("Source key cannot be longer than 255 characters"),
		),
		validation.Field(&c.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[Kind]("Kind must be Update, Arrival or Departure"),
		),
		validation.Field(&c.ReportedAt,
			validation.Required.Error("Reported time is required"),
			validation.Min(int64(1)).Error("Reported time must be a valid timestamp"),
		),
		validation.Field(&c.ReceivedAt,
			validation.Required.Error("Received time is required"),
			validation.Min(int64(1)).Error("Received time must be a valid timestamp"),
		),
		validation.Field(&c.Latitude,
			validation.When(c.Latitude != nil,
				validation.Min(-90.0).Error("Latitude must be between -90 and 90"),
				validation.Max(90.0).Error("Latitude must be between -90 and 90"),
			),
			validation.When(c.Longitude != nil && c.Latitude == nil,
				validation.Required.Error("Latitude is required with a longitude"),
			),
		),
		validation.Field(&c.Longitude,
			validation.When(c.Longitude != nil,
				validation.Min(-180.0).Error("Longitude must be between -180 and 180"),
				validation.Max(180.0).Error("Longitude must be between -180 and 180"),
			),
			validation.When(c.Latitude != nil && c.Longitude == nil,
				validation.Required.Error("Longitude is required with a latitude"),
			),
		),
		validation.Field(&c.City,
			validation.RuneLength(0, MaxCityLength).
				Error("City cannot be longer than 100 characters"),
		),
		validation.Field(&c.StateCode,
			validation.RuneLength(0, MaxStateCodeLength).
				Error("State cannot be longer than 10 characters"),
		),
		validation.Field(&c.DelayReason,
			validation.When(c.Delayed,
				validation.Required.Error("Say why the load is delayed"),
			),
			validation.RuneLength(0, MaxDelayReasonLength).
				Error("Delay reason cannot be longer than 255 characters"),
		),
		validation.Field(&c.EstimatedArrival,
			validation.When(c.EstimatedArrival != nil,
				validation.Min(int64(1)).Error("Estimated arrival must be a valid timestamp"),
			),
		),
		validation.Field(&c.Note,
			validation.RuneLength(0, MaxNoteLength).
				Error("Note cannot be longer than 1000 characters"),
		),
		validation.Field(&c.ContactName,
			validation.RuneLength(0, MaxContactNameLength).
				Error("Contact name cannot be longer than 100 characters"),
		),
	))
}
