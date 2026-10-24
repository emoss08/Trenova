package weatheralert

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type WeatherAlert struct {
	bun.BaseModel `bun:"table:weather_alerts,alias:wa" json:"-"`
	keel.Table    `keel:"weather_alerts"`

	ID             pulid.ID          `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:walt_"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	NWSID          string            `json:"nwsId"          bun:"nws_id,type:TEXT,notnull" keel:"nws_id"`
	Event          string            `json:"event"          bun:"event,type:TEXT,notnull" keel:"event"`
	Severity       string            `json:"severity"       bun:"severity,type:VARCHAR(50),nullzero" keel:"severity,nullzero"`
	Urgency        string            `json:"urgency"        bun:"urgency,type:VARCHAR(50),nullzero" keel:"urgency,nullzero"`
	Certainty      string            `json:"certainty"      bun:"certainty,type:VARCHAR(50),nullzero" keel:"certainty,nullzero"`
	Headline       string            `json:"headline"       bun:"headline,type:TEXT,nullzero" keel:"headline,nullzero"`
	Description    string            `json:"description"    bun:"description,type:TEXT,nullzero" keel:"description,nullzero"`
	Instruction    string            `json:"instruction"    bun:"instruction,type:TEXT,nullzero" keel:"instruction,nullzero"`
	AreaDesc       string            `json:"areaDesc"       bun:"area_desc,type:TEXT,nullzero" keel:"area_desc,nullzero"`
	Effective      *int64            `json:"effective"      bun:"effective,type:BIGINT,nullzero" keel:"effective"`
	Expires        *int64            `json:"expires"        bun:"expires,type:BIGINT,nullzero" keel:"expires"`
	Onset          *int64            `json:"onset"          bun:"onset,type:BIGINT,nullzero" keel:"onset"`
	Ends           *int64            `json:"ends"           bun:"ends,type:BIGINT,nullzero" keel:"ends"`
	Status         string            `json:"status"         bun:"status,type:VARCHAR(50),nullzero" keel:"status,nullzero"`
	MessageType    string            `json:"messageType"    bun:"message_type,type:VARCHAR(50),nullzero" keel:"message_type,nullzero"`
	SenderName     string            `json:"senderName"     bun:"sender_name,type:TEXT,nullzero" keel:"sender_name,nullzero"`
	Response       string            `json:"response"       bun:"response,type:VARCHAR(50),nullzero" keel:"response,nullzero"`
	Category       string            `json:"category"       bun:"category,type:VARCHAR(50),nullzero" keel:"category,nullzero"`
	AlertCategory  AlertCategory     `json:"alertCategory"  bun:"alert_category,type:VARCHAR(50),notnull" keel:"alert_category"`
	Geometry       *postgis.Geometry `json:"-"              bun:"geometry,type:geometry,scanonly" keel:"geometry,readonly"`
	FirstSeenAt    int64             `json:"firstSeenAt"    bun:"first_seen_at,type:BIGINT,notnull" keel:"first_seen_at"`
	LastUpdatedAt  int64             `json:"lastUpdatedAt"  bun:"last_updated_at,type:BIGINT,notnull" keel:"last_updated_at"`
	ExpiredAt      *int64            `json:"expiredAt"      bun:"expired_at,type:BIGINT,nullzero" keel:"expired_at"`
	Version        int64             `json:"version"        bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64             `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

type Activity struct {
	bun.BaseModel `bun:"table:weather_alert_activities,alias:waa" json:"-"`
	keel.Table    `keel:"weather_alert_activities"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:waac_"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	WeatherAlertID pulid.ID       `json:"weatherAlertId" bun:"weather_alert_id,type:VARCHAR(100),notnull" keel:"weather_alert_id"`
	ActivityType   ActivityType   `json:"activityType"   bun:"activity_type,type:VARCHAR(50),notnull" keel:"activity_type"`
	Timestamp      int64          `json:"timestamp"      bun:"timestamp,type:BIGINT,notnull" keel:"timestamp"`
	Details        map[string]any `json:"details"        bun:"details,type:jsonb,nullzero" keel:"details,jsonb,nullzero"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (w *WeatherAlert) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if w.ID.IsNil() {
			w.ID = pulid.MustNew("walt_")
		}
		w.CreatedAt = now
	case *bun.UpdateQuery:
		w.UpdatedAt = now
	}

	return nil
}

func (a *Activity) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("waac_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
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
