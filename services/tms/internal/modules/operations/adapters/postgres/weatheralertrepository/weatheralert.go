package weatheralertrepository

import (
	"context"
	"maps"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/weatheralert"
	"github.com/emoss08/trenova/internal/modules/operations/domain/weatheralert/weatheralertdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var weatherAlertErrors = dberror.Mapping{
	Entity: "Weather Alert",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_weather_alerts_tenant_nws_id",
			Field:   "nwsId",
			Message: "Weather alert with this NWS ID already exists in your organization",
		},
	},
}

var activityErrors = dberror.Mapping{
	Entity: "Weather Alert Activity",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_weather_alert_activities_alert",
			Field:   "weatherAlertId",
			Message: "Weather alert does not exist in your organization",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func New(p Params) weatheralertports.WeatherAlertRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.weather-alert-repository"),
	}
}

func (r *repository) ListTenants(ctx context.Context) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations for weather alert polling")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		o := tenantdb.Organizations
		rows, err := keel.Into[tenantRow](
			ctx,
			r.db.Keel(ctx),
			o.AllTenants().Pick(o.ID, o.BusinessUnitID).Order(o.ID.Asc()),
		)
		if err != nil {
			return nil, err
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}

		return tenants, nil
	})
}

func (r *repository) GetActiveAlerts(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*weatheralert.WeatherAlert, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*weatheralert.WeatherAlert, error) {
		wa := weatheralertdb.WeatherAlerts
		now := timeutils.NowUnix()

		return wa.Where(
			wa.ExpiredAt.IsNull(),
			keel.Or(wa.Expires.IsNull(), wa.Expires.Gt(now)),
		).
			Order(wa.Expires.Asc().NullsLast()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req weatheralertports.GetWeatherAlertByIDRequest,
) (*weatheralert.WeatherAlert, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*weatheralert.WeatherAlert, error) {
		alert, err := weatheralertdb.WeatherAlerts.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, weatherAlertErrors.Map(err, req.ID.String())
		}

		return alert, nil
	})
}

func (r *repository) GetActivities(
	ctx context.Context,
	req weatheralertports.GetWeatherAlertByIDRequest,
) ([]*weatheralert.Activity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*weatheralert.Activity, error) {
		a := weatheralertdb.Activities

		return a.Where(a.WeatherAlertID.Eq(req.ID)).
			Order(a.Timestamp.Desc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListByNWSIDs(
	ctx context.Context,
	req weatheralertports.ListWeatherAlertsByNWSIDsRequest,
) ([]*weatheralert.WeatherAlert, error) {
	if len(req.NWSIDs) == 0 {
		return []*weatheralert.WeatherAlert{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*weatheralert.WeatherAlert, error) {
		wa := weatheralertdb.WeatherAlerts

		return wa.Pick(
			wa.NWSID,
			wa.MessageType,
			wa.AlertCategory,
			wa.Effective,
			wa.Onset,
			wa.Expires,
			wa.Ends,
		).
			Where(wa.NWSID.In(req.NWSIDs...)).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) UpsertAlert(
	ctx context.Context,
	alert *weatheralert.WeatherAlert,
) (*weatheralertports.UpsertWeatherAlertResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*weatheralertports.UpsertWeatherAlertResult, error) {
			db := r.db.Keel(ctx)
			wa := weatheralertdb.WeatherAlerts
			existing, err := wa.Where(wa.NWSID.Eq(alert.NWSID)).One(ctx, db)
			if err != nil && !dberror.IsNotFoundError(err) {
				return nil, err
			}

			if existing == nil {
				return r.createAlert(ctx, db, alert)
			}

			return r.updateAlert(ctx, db, existing, alert)
		},
	)
}

func (r *repository) createAlert(
	ctx context.Context,
	db *keel.DB,
	alert *weatheralert.WeatherAlert,
) (*weatheralertports.UpsertWeatherAlertResult, error) {
	prepareNewAlert(alert)
	if err := weatheralertdb.WeatherAlerts.Create(ctx, db, alert); err != nil {
		return nil, weatherAlertErrors.Map(err, alert.ID.String())
	}

	activity := newActivity(alert, weatheralert.ActivityTypeIssued, nil)
	if err := weatheralertdb.Activities.Create(ctx, db, activity); err != nil {
		return nil, activityErrors.Map(err, activity.ID.String())
	}

	return &weatheralertports.UpsertWeatherAlertResult{
		Alert:        alert,
		Activity:     activity,
		Created:      true,
		Changed:      true,
		ActivityType: weatheralert.ActivityTypeIssued,
	}, nil
}

func (r *repository) updateAlert(
	ctx context.Context,
	db *keel.DB,
	existing, alert *weatheralert.WeatherAlert,
) (*weatheralertports.UpsertWeatherAlertResult, error) {
	diff, err := buildDiff(existing, alert)
	if err != nil {
		return nil, err
	}

	alert.ID = existing.ID
	alert.CreatedAt = existing.CreatedAt
	alert.FirstSeenAt = existing.FirstSeenAt
	alert.Version = existing.Version
	if alert.LastUpdatedAt == 0 {
		alert.LastUpdatedAt = timeutils.NowUnix()
	}

	if len(diff) == 0 && !stateTransition(existing, alert) {
		return &weatheralertports.UpsertWeatherAlertResult{Alert: existing}, nil
	}

	if err = weatheralertdb.WeatherAlerts.Save(ctx, db, alert); err != nil {
		return nil, weatherAlertErrors.Map(err, alert.ID.String())
	}

	activityType := weatheralert.ActivityTypeUpdated
	if isCancelled(alert) {
		activityType = weatheralert.ActivityTypeCancelled
	}
	activity := newActivity(alert, activityType, map[string]any{"changes": diff})
	if err = weatheralertdb.Activities.Create(ctx, db, activity); err != nil {
		return nil, activityErrors.Map(err, activity.ID.String())
	}

	return &weatheralertports.UpsertWeatherAlertResult{
		Alert:        alert,
		Activity:     activity,
		Changed:      true,
		ActivityType: activityType,
	}, nil
}

func prepareNewAlert(alert *weatheralert.WeatherAlert) {
	if alert.ID.IsNil() {
		alert.ID = pulid.MustNew("walt_")
	}
	if alert.FirstSeenAt == 0 {
		alert.FirstSeenAt = timeutils.NowUnix()
	}
	if alert.LastUpdatedAt == 0 {
		alert.LastUpdatedAt = alert.FirstSeenAt
	}
	if alert.CreatedAt == 0 {
		alert.CreatedAt = alert.FirstSeenAt
	}
	if alert.UpdatedAt == 0 {
		alert.UpdatedAt = alert.LastUpdatedAt
	}
}

func (r *repository) ExpireStaleAlerts(
	ctx context.Context,
) (*weatheralertports.ExpireWeatherAlertsResult, error) {
	ctx = dbscope.WithSystem(ctx, "expire stale weather alerts across every organization")
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*weatheralertports.ExpireWeatherAlertsResult, error) {
			now := timeutils.NowUnix()
			db := r.db.Keel(ctx)
			wa := weatheralertdb.WeatherAlerts
			expired, err := wa.AllTenants().
				Pick(wa.ID, wa.LastUpdatedAt).
				Where(
					wa.ExpiredAt.IsNull(),
					wa.Expires.IsNotNull(),
					wa.Expires.Lte(now),
				).
				UpdateReturning(ctx, db, wa.ExpiredAt.To(now), wa.LastUpdatedAt.To(now))
			if err != nil {
				return nil, err
			}

			activities := make([]*weatheralert.Activity, 0, len(expired))
			for _, alert := range expired {
				activities = append(
					activities,
					newActivity(alert, weatheralert.ActivityTypeExpired, nil),
				)
			}
			if err = weatheralertdb.Activities.CreateMany(ctx, db, activities); err != nil {
				return nil, activityErrors.Map(err, "")
			}

			return &weatheralertports.ExpireWeatherAlertsResult{ExpiredCount: len(expired)}, nil
		},
	)
}

func newActivity(
	alert *weatheralert.WeatherAlert,
	activityType weatheralert.ActivityType,
	details map[string]any,
) *weatheralert.Activity {
	return &weatheralert.Activity{
		OrganizationID: alert.OrganizationID,
		BusinessUnitID: alert.BusinessUnitID,
		WeatherAlertID: alert.ID,
		ActivityType:   activityType,
		Timestamp:      alert.LastUpdatedAt,
		Details:        details,
	}
}

func isCancelled(alert *weatheralert.WeatherAlert) bool {
	return alert.MessageType == "Cancel" && alert.ExpiredAt != nil
}

func stateTransition(existing, incoming *weatheralert.WeatherAlert) bool {
	return existing.ExpiredAt == nil && incoming.ExpiredAt != nil
}

func buildDiff(
	existing, incoming *weatheralert.WeatherAlert,
) (map[string]any, error) {
	diff, err := jsonutils.JSONDiff(snapshot(existing), snapshot(incoming), nil)
	if err != nil {
		return nil, err
	}

	result := make(map[string]any, len(diff))
	for key, change := range diff {
		result[key] = change
	}

	return result, nil
}

func snapshot(alert *weatheralert.WeatherAlert) map[string]any {
	geometryJSON := ""
	if alert.Geometry != nil {
		if raw, err := alert.Geometry.GeoJSONString(); err == nil {
			geometryJSON = raw
		}
	}

	result := map[string]any{
		"nwsId":         alert.NWSID,
		"event":         alert.Event,
		"severity":      alert.Severity,
		"urgency":       alert.Urgency,
		"certainty":     alert.Certainty,
		"headline":      alert.Headline,
		"description":   alert.Description,
		"instruction":   alert.Instruction,
		"areaDesc":      alert.AreaDesc,
		"effective":     alert.Effective,
		"expires":       alert.Expires,
		"onset":         alert.Onset,
		"ends":          alert.Ends,
		"status":        alert.Status,
		"messageType":   alert.MessageType,
		"senderName":    alert.SenderName,
		"response":      alert.Response,
		"category":      alert.Category,
		"alertCategory": alert.AlertCategory,
		"geometry":      geometryJSON,
		"expiredAt":     alert.ExpiredAt,
	}

	return maps.Clone(result)
}
