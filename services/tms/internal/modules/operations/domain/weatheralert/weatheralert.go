package weatheralert

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type WeatherAlert struct {
	keel.Table `keel:"weather_alerts"`

	ID             pulid.ID          `json:"id"             keel:"id,pk,autoid:walt_"`
	OrganizationID pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	NWSID          string            `json:"nwsId"          keel:"nws_id"`
	Event          string            `json:"event"          keel:"event"`
	Severity       string            `json:"severity"       keel:"severity,nullzero"`
	Urgency        string            `json:"urgency"        keel:"urgency,nullzero"`
	Certainty      string            `json:"certainty"      keel:"certainty,nullzero"`
	Headline       string            `json:"headline"       keel:"headline,nullzero"`
	Description    string            `json:"description"    keel:"description,nullzero"`
	Instruction    string            `json:"instruction"    keel:"instruction,nullzero"`
	AreaDesc       string            `json:"areaDesc"       keel:"area_desc,nullzero"`
	Effective      *int64            `json:"effective"      keel:"effective"`
	Expires        *int64            `json:"expires"        keel:"expires"`
	Onset          *int64            `json:"onset"          keel:"onset"`
	Ends           *int64            `json:"ends"           keel:"ends"`
	Status         string            `json:"status"         keel:"status,nullzero"`
	MessageType    string            `json:"messageType"    keel:"message_type,nullzero"`
	SenderName     string            `json:"senderName"     keel:"sender_name,nullzero"`
	Response       string            `json:"response"       keel:"response,nullzero"`
	Category       string            `json:"category"       keel:"category,nullzero"`
	AlertCategory  AlertCategory     `json:"alertCategory"  keel:"alert_category"`
	Geometry       *postgis.Geometry `json:"-"              keel:"geometry"`
	FirstSeenAt    int64             `json:"firstSeenAt"    keel:"first_seen_at"`
	LastUpdatedAt  int64             `json:"lastUpdatedAt"  keel:"last_updated_at"`
	ExpiredAt      *int64            `json:"expiredAt"      keel:"expired_at"`
	Version        int64             `json:"version"        keel:"version,version"`
	CreatedAt      int64             `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64             `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

type Activity struct {
	keel.Table `keel:"weather_alert_activities"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:waac_"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	WeatherAlertID pulid.ID       `json:"weatherAlertId" keel:"weather_alert_id"`
	ActivityType   ActivityType   `json:"activityType"   keel:"activity_type"`
	Timestamp      int64          `json:"timestamp"      keel:"timestamp"`
	Details        map[string]any `json:"details"        keel:"details,jsonb,nullzero"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (w *WeatherAlert) IsSameMessage(stored *WeatherAlert) bool {
	if stored == nil {
		return false
	}

	return w.NWSID == stored.NWSID &&
		w.MessageType == stored.MessageType &&
		w.AlertCategory == stored.AlertCategory &&
		typeutils.EqualPtr(w.Effective, stored.Effective) &&
		typeutils.EqualPtr(w.Onset, stored.Onset) &&
		typeutils.EqualPtr(w.Expires, stored.Expires) &&
		typeutils.EqualPtr(w.Ends, stored.Ends)
}

func (w *WeatherAlert) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		w,
		validation.Field(&w.OrganizationID, validation.Required),
		validation.Field(&w.BusinessUnitID, validation.Required),
		validation.Field(&w.NWSID, validation.Required),
		validation.Field(&w.Event, validation.Required),
		validation.Field(
			&w.AlertCategory,
			validation.Required,
			validation.By(ValidateAlertCategory),
		),
	))

	if w.Geometry == nil || w.Geometry.Geometry == nil {
		multiErr.Add("geometry", errortypes.ErrRequired, "Geometry is required")
	}
}

func (WeatherAlert) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_weather_alerts_tenant_nws_id",
			Message: "Weather alert with this NWS ID already exists in your organization",
		},
	}
}

func (a *Activity) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&a.WeatherAlertID,
			validation.Required.Error("Weather alert is required"),
		),
		validation.Field(&a.ActivityType,
			validation.Required.Error("Activity type is required"),
			domainvalidation.ValidEnum[ActivityType]("Activity type is invalid"),
		),
		validation.Field(&a.Timestamp,
			validation.Required.Error("Timestamp is required"),
			validation.Min(int64(1)).Error("Timestamp must be a valid timestamp"),
		),
	))
}

func (Activity) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_weather_alert_activities_alert",
			Message: "Weather alert does not exist in your organization",
		},
	}
}
