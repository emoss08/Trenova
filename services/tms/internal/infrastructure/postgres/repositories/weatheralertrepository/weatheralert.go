package weatheralertrepository

import (
	"context"
	"database/sql"
	"errors"
	"maps"

	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

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
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type weatherAlertWriteModel struct {
	bun.BaseModel `bun:"table:weather_alerts"`

	ID             pulid.ID                   `bun:"id,pk"`
	OrganizationID pulid.ID                   `bun:"organization_id,pk"`
	BusinessUnitID pulid.ID                   `bun:"business_unit_id,pk"`
	NWSID          string                     `bun:"nws_id"`
	Event          string                     `bun:"event"`
	Severity       string                     `bun:"severity,nullzero"`
	Urgency        string                     `bun:"urgency,nullzero"`
	Certainty      string                     `bun:"certainty,nullzero"`
	Headline       string                     `bun:"headline,nullzero"`
	Description    string                     `bun:"description,nullzero"`
	Instruction    string                     `bun:"instruction,nullzero"`
	AreaDesc       string                     `bun:"area_desc,nullzero"`
	Effective      *int64                     `bun:"effective,nullzero"`
	Expires        *int64                     `bun:"expires,nullzero"`
	Onset          *int64                     `bun:"onset,nullzero"`
	Ends           *int64                     `bun:"ends,nullzero"`
	Status         string                     `bun:"status,nullzero"`
	MessageType    string                     `bun:"message_type,nullzero"`
	SenderName     string                     `bun:"sender_name,nullzero"`
	Response       string                     `bun:"response,nullzero"`
	Category       string                     `bun:"category,nullzero"`
	AlertCategory  weatheralert.AlertCategory `bun:"alert_category"`
	FirstSeenAt    int64                      `bun:"first_seen_at"`
	LastUpdatedAt  int64                      `bun:"last_updated_at"`
	ExpiredAt      *int64                     `bun:"expired_at,nullzero"`
	Version        int64                      `bun:"version"`
	CreatedAt      int64                      `bun:"created_at"`
	UpdatedAt      int64                      `bun:"updated_at"`
}

func New(p Params) repositories.WeatherAlertRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.weather-alert-repository"),
	}
}

func (r *repository) ListTenants(ctx context.Context) ([]pagination.TenantInfo, error) {
	rows := make([]tenantRow, 0)
	if err := r.db.DB().
		NewSelect().
		TableExpr("organizations AS org").
		ColumnExpr("org.id AS organization_id").
		ColumnExpr("org.business_unit_id AS business_unit_id").
		OrderExpr("org.id ASC").
		Scan(ctx, &rows); err != nil {
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
}

func (r *repository) GetActiveAlerts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*weatheralert.WeatherAlert, error) {
	alerts := make([]*weatheralert.WeatherAlert, 0)
	now := timeutils.NowUnix()

	if err := r.db.DB().
		NewSelect().
		Model(&alerts).
		ColumnExpr("wa.*").
		ColumnExpr("wa.geometry").
		Where("wa.organization_id = ?", tenantInfo.OrgID).
		Where("wa.business_unit_id = ?", tenantInfo.BuID).
		Where("wa.expired_at IS NULL").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("wa.expires IS NULL").WhereOr("wa.expires > ?", now)
		}).
		OrderExpr("COALESCE(wa.expires, 9223372036854775807) ASC").
		Scan(ctx); err != nil {
		return nil, err
	}

	return alerts, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetWeatherAlertByIDRequest,
) (*weatheralert.WeatherAlert, error) {
	alert := new(weatheralert.WeatherAlert)
	err := r.db.DB().
		NewSelect().
		Model(alert).
		ColumnExpr("wa.*").
		ColumnExpr("wa.geometry").
		Where("wa.id = ?", req.ID).
		Where("wa.organization_id = ?", req.TenantInfo.OrgID).
		Where("wa.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Weather Alert")
	}

	return alert, nil
}

func (r *repository) GetActivities(
	ctx context.Context,
	req repositories.GetWeatherAlertByIDRequest,
) ([]*weatheralert.Activity, error) {
	activities := make([]*weatheralert.Activity, 0)
	if err := r.db.DB().
		NewSelect().
		Model(&activities).
		Where("waa.weather_alert_id = ?", req.ID).
		Where("waa.organization_id = ?", req.TenantInfo.OrgID).
		Where("waa.business_unit_id = ?", req.TenantInfo.BuID).
		OrderExpr("waa.timestamp DESC").
		Scan(ctx); err != nil {
		return nil, err
	}

	return activities, nil
}

func (r *repository) UpsertAlert(
	ctx context.Context,
	alert *weatheralert.WeatherAlert,
) (*repositories.UpsertWeatherAlertResult, error) {
	result := new(repositories.UpsertWeatherAlertResult)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		existing, err := r.getByNWSID(c, tx, alert.OrganizationID.String(), alert.BusinessUnitID.String(), alert.NWSID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		if errors.Is(err, sql.ErrNoRows) {
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
			if err = r.insertAlert(c, tx, alert); err != nil {
				return err
			}

			activity := newActivity(alert, weatheralert.ActivityTypeIssued, nil)
			if err = r.insertActivity(c, tx, activity); err != nil {
				return err
			}

			result.Alert = alert
			result.Activity = activity
			result.Created = true
			result.Changed = true
			result.ActivityType = weatheralert.ActivityTypeIssued
			return nil
		}

		diff, err := buildDiff(existing, alert)
		if err != nil {
			return err
		}

		alert.ID = existing.ID
		alert.CreatedAt = existing.CreatedAt
		alert.FirstSeenAt = existing.FirstSeenAt
		alert.Version = existing.Version + 1
		if alert.LastUpdatedAt == 0 {
			alert.LastUpdatedAt = timeutils.NowUnix()
		}
		alert.UpdatedAt = alert.LastUpdatedAt

		changed := len(diff) > 0 || stateTransition(existing, alert)
		if !changed {
			result.Alert = existing
			return nil
		}

		if err = r.updateAlert(c, tx, alert, existing.Version); err != nil {
			return err
		}

		activityType := weatheralert.ActivityTypeUpdated
		if isCancelled(alert) {
			activityType = weatheralert.ActivityTypeCancelled
		}
		activity := newActivity(alert, activityType, map[string]any{"changes": diff})
		if err = r.insertActivity(c, tx, activity); err != nil {
			return err
		}

		result.Alert = alert
		result.Activity = activity
		result.Changed = true
		result.ActivityType = activityType
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repository) ExpireStaleAlerts(
	ctx context.Context,
) (*repositories.ExpireWeatherAlertsResult, error) {
	now := timeutils.NowUnix()
	result := &repositories.ExpireWeatherAlertsResult{}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		alerts := make([]*weatheralert.WeatherAlert, 0)
		if err := tx.NewSelect().
			Model(&alerts).
			ColumnExpr("wa.*").
			ColumnExpr("wa.geometry").
			Where("wa.expired_at IS NULL").
			Where("wa.expires IS NOT NULL").
			Where("wa.expires <= ?", now).
			Scan(c); err != nil {
			return err
		}

		for _, alert := range alerts {
			alert.ExpiredAt = &now
			alert.LastUpdatedAt = now
			alert.Version++

			if _, err := tx.NewUpdate().
				Model(alert).
				Column("expired_at", "last_updated_at", "version", "updated_at").
				WherePK().
				Where("version = ?", alert.Version-1).
				Exec(c); err != nil {
				return err
			}

			if err := r.insertActivity(c, tx, newActivity(alert, weatheralert.ActivityTypeExpired, nil)); err != nil {
				return err
			}
		}

		result.ExpiredCount = len(alerts)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repository) getByNWSID(
	ctx context.Context,
	db bun.IDB,
	orgID, buID, nwsID string,
) (*weatheralert.WeatherAlert, error) {
	alert := new(weatheralert.WeatherAlert)
	err := db.NewSelect().
		Model(alert).
		ColumnExpr("wa.*").
		ColumnExpr("wa.geometry").
		Where("wa.organization_id = ?", orgID).
		Where("wa.business_unit_id = ?", buID).
		Where("wa.nws_id = ?", nwsID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

func (r *repository) insertAlert(
	ctx context.Context,
	db bun.IDB,
	alert *weatheralert.WeatherAlert,
) error {
	geometryJSON, err := alert.Geometry.GeoJSONString()
	if err != nil {
		return err
	}

	writeModel := toWriteModel(alert)
	_, err = db.NewInsert().
		Model(writeModel).
		Value("geometry", "ST_SetSRID(ST_GeomFromGeoJSON(?), 4326)", geometryJSON).
		Exec(ctx)
	return err
}

func (r *repository) updateAlert(
	ctx context.Context,
	db bun.IDB,
	alert *weatheralert.WeatherAlert,
	previousVersion int64,
) error {
	geometryJSON, err := alert.Geometry.GeoJSONString()
	if err != nil {
		return err
	}

	writeModel := toWriteModel(alert)
	result, err := db.NewUpdate().
		Model(writeModel).
		WherePK().
		ExcludeColumn("created_at").
		Set("geometry = ST_SetSRID(ST_GeomFromGeoJSON(?), 4326)", geometryJSON).
		Where("version = ?", previousVersion).
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(result, "Weather Alert", alert.ID.String())
}

func toWriteModel(alert *weatheralert.WeatherAlert) *weatherAlertWriteModel {
	return &weatherAlertWriteModel{
		ID:             alert.ID,
		OrganizationID: alert.OrganizationID,
		BusinessUnitID: alert.BusinessUnitID,
		NWSID:          alert.NWSID,
		Event:          alert.Event,
		Severity:       alert.Severity,
		Urgency:        alert.Urgency,
		Certainty:      alert.Certainty,
		Headline:       alert.Headline,
		Description:    alert.Description,
		Instruction:    alert.Instruction,
		AreaDesc:       alert.AreaDesc,
		Effective:      alert.Effective,
		Expires:        alert.Expires,
		Onset:          alert.Onset,
		Ends:           alert.Ends,
		Status:         alert.Status,
		MessageType:    alert.MessageType,
		SenderName:     alert.SenderName,
		Response:       alert.Response,
		Category:       alert.Category,
		AlertCategory:  alert.AlertCategory,
		FirstSeenAt:    alert.FirstSeenAt,
		LastUpdatedAt:  alert.LastUpdatedAt,
		ExpiredAt:      alert.ExpiredAt,
		Version:        alert.Version,
		CreatedAt:      alert.CreatedAt,
		UpdatedAt:      alert.UpdatedAt,
	}
}

func (r *repository) insertActivity(
	ctx context.Context,
	db bun.IDB,
	activity *weatheralert.Activity,
) error {
	_, err := db.NewInsert().Model(activity).Exec(ctx)
	return err
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
