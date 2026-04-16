package weatheralert

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type WeatherAlert struct {
	bun.BaseModel `bun:"table:weather_alerts,alias:wa" json:"-"`

	ID             pulid.ID          `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	NWSID          string            `json:"nwsId"          bun:"nws_id,type:TEXT,notnull"`
	Event          string            `json:"event"          bun:"event,type:TEXT,notnull"`
	Severity       string            `json:"severity"       bun:"severity,type:VARCHAR(50),nullzero"`
	Urgency        string            `json:"urgency"        bun:"urgency,type:VARCHAR(50),nullzero"`
	Certainty      string            `json:"certainty"      bun:"certainty,type:VARCHAR(50),nullzero"`
	Headline       string            `json:"headline"       bun:"headline,type:TEXT,nullzero"`
	Description    string            `json:"description"    bun:"description,type:TEXT,nullzero"`
	Instruction    string            `json:"instruction"    bun:"instruction,type:TEXT,nullzero"`
	AreaDesc       string            `json:"areaDesc"       bun:"area_desc,type:TEXT,nullzero"`
	Effective      *int64            `json:"effective"      bun:"effective,type:BIGINT,nullzero"`
	Expires        *int64            `json:"expires"        bun:"expires,type:BIGINT,nullzero"`
	Onset          *int64            `json:"onset"          bun:"onset,type:BIGINT,nullzero"`
	Ends           *int64            `json:"ends"           bun:"ends,type:BIGINT,nullzero"`
	Status         string            `json:"status"         bun:"status,type:VARCHAR(50),nullzero"`
	MessageType    string            `json:"messageType"    bun:"message_type,type:VARCHAR(50),nullzero"`
	SenderName     string            `json:"senderName"     bun:"sender_name,type:TEXT,nullzero"`
	Response       string            `json:"response"       bun:"response,type:VARCHAR(50),nullzero"`
	Category       string            `json:"category"       bun:"category,type:VARCHAR(50),nullzero"`
	AlertCategory  AlertCategory     `json:"alertCategory"  bun:"alert_category,type:VARCHAR(50),notnull"`
	Geometry       *postgis.Geometry `json:"-"              bun:"geometry,type:geometry,scanonly"`
	FirstSeenAt    int64             `json:"firstSeenAt"    bun:"first_seen_at,type:BIGINT,notnull"`
	LastUpdatedAt  int64             `json:"lastUpdatedAt"  bun:"last_updated_at,type:BIGINT,notnull"`
	ExpiredAt      *int64            `json:"expiredAt"      bun:"expired_at,type:BIGINT,nullzero"`
	Version        int64             `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64             `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type Activity struct {
	bun.BaseModel `bun:"table:weather_alert_activities,alias:waa" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	WeatherAlertID pulid.ID       `json:"weatherAlertId" bun:"weather_alert_id,type:VARCHAR(100),notnull"`
	ActivityType   ActivityType   `json:"activityType"   bun:"activity_type,type:VARCHAR(50),notnull"`
	Timestamp      int64          `json:"timestamp"      bun:"timestamp,type:BIGINT,notnull"`
	Details        map[string]any `json:"details"       bun:"details,type:jsonb,nullzero"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (w *WeatherAlert) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(w,
		validation.Field(&w.OrganizationID, validation.Required),
		validation.Field(&w.BusinessUnitID, validation.Required),
		validation.Field(&w.NWSID, validation.Required),
		validation.Field(&w.Event, validation.Required),
		validation.Field(&w.AlertCategory, validation.Required, validation.By(ValidateAlertCategory)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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
