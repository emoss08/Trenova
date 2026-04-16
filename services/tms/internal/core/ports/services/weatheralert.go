package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type WeatherAlertGeometry map[string]any

type WeatherAlertFeatureCollection struct {
	Type     string                 `json:"type"`
	Features []*WeatherAlertFeature `json:"features"`
}

type WeatherAlertFeature struct {
	Type       string                        `json:"type"`
	ID         pulid.ID                      `json:"id"`
	Geometry   WeatherAlertGeometry          `json:"geometry"`
	Properties WeatherAlertFeatureProperties `json:"properties"`
}

type WeatherAlertFeatureProperties struct {
	ID            pulid.ID                   `json:"id"`
	NWSID         string                     `json:"nwsId"`
	Event         string                     `json:"event"`
	Severity      string                     `json:"severity,omitempty"`
	Urgency       string                     `json:"urgency,omitempty"`
	Certainty     string                     `json:"certainty,omitempty"`
	Headline      string                     `json:"headline,omitempty"`
	Description   string                     `json:"description,omitempty"`
	Instruction   string                     `json:"instruction,omitempty"`
	AreaDesc      string                     `json:"areaDesc,omitempty"`
	Effective     *int64                     `json:"effective,omitempty"`
	Expires       *int64                     `json:"expires,omitempty"`
	Onset         *int64                     `json:"onset,omitempty"`
	Ends          *int64                     `json:"ends,omitempty"`
	Status        string                     `json:"status,omitempty"`
	MessageType   string                     `json:"messageType,omitempty"`
	SenderName    string                     `json:"senderName,omitempty"`
	Response      string                     `json:"response,omitempty"`
	Category      string                     `json:"category,omitempty"`
	AlertCategory weatheralert.AlertCategory `json:"alertCategory"`
	FirstSeenAt   int64                      `json:"firstSeenAt"`
	LastUpdatedAt int64                      `json:"lastUpdatedAt"`
	ExpiredAt     *int64                     `json:"expiredAt,omitempty"`
}

type WeatherAlertDetail struct {
	Feature    *WeatherAlertFeature     `json:"feature"`
	Activities []*weatheralert.Activity `json:"activities"`
}

type GetWeatherAlertDetailRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type WeatherAlertService interface {
	PollNWSAlerts(ctx context.Context) error
	GetActiveAlerts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*WeatherAlertFeatureCollection, error)
	GetAlertDetail(
		ctx context.Context,
		req *GetWeatherAlertDetailRequest,
	) (*WeatherAlertDetail, error)
}
