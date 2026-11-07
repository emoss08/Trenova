package demandforecast

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type Rate struct {
	keel.Table `keel:"demand_forecast_rates"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:dfr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	CustomerID     pulid.ID `json:"customerId"     keel:"customer_id,nullzero"`
	Cell           int64    `json:"cell"           keel:"cell"`
	Slot           int16    `json:"slot"           keel:"slot"`
	Alpha          float64  `json:"alpha"          keel:"alpha"`
	Beta           float64  `json:"beta"           keel:"beta"`
	Loads          int32    `json:"loads"          keel:"loads"`
	Recent         []int32  `json:"recent"         keel:"recent,jsonb"`
	FittedAt       int64    `json:"fittedAt"       keel:"fitted_at"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
}

type Day struct {
	keel.Table `keel:"demand_forecast_days"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:dfd_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	DayStart       int64    `json:"dayStart"       keel:"day_start"`
	Low            int32    `json:"low"            keel:"low"`
	High           int32    `json:"high"           keel:"high"`
	Mean           float64  `json:"mean"           keel:"mean"`
	Actual         int32    `json:"actual"         keel:"actual"`
	FittedAt       int64    `json:"fittedAt"       keel:"fitted_at"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
}

type Area struct {
	keel.Table `keel:"demand_forecast_areas"`

	ID                 pulid.ID `json:"id"                 keel:"id,pk,autoid:dfa_"`
	BusinessUnitID     pulid.ID `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID `json:"organizationId"     keel:"organization_id,pk,tenant"`
	Cell               int64    `json:"cell"               keel:"cell"`
	Label              string   `json:"label"              keel:"label"`
	MedianRevenueCents int64    `json:"medianRevenueCents" keel:"median_revenue_cents"`
	Loads              int32    `json:"loads"              keel:"loads"`
	FittedAt           int64    `json:"fittedAt"           keel:"fitted_at"`
	CreatedAt          int64    `json:"createdAt"          keel:"created_at,autocreate"`
}

type BookingCurve struct {
	keel.Table `keel:"demand_booking_curves"`

	ID             pulid.ID  `json:"id"             keel:"id,pk,autoid:dbc_"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	CustomerID     pulid.ID  `json:"customerId"     keel:"customer_id,nullzero"`
	Shares         []float64 `json:"shares"         keel:"shares,jsonb"`
	Loads          int32     `json:"loads"          keel:"loads"`
	FittedAt       int64     `json:"fittedAt"       keel:"fitted_at"`
	CreatedAt      int64     `json:"createdAt"      keel:"created_at,autocreate"`
}

type Status string

const (
	StatusInsufficientHistory = Status("insufficientHistory")
	StatusNoSkill             = Status("noSkill")
	StatusReady               = Status("ready")
)

type Quality struct {
	keel.Table `keel:"demand_forecast_quality"`

	BusinessUnitID   pulid.ID `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID `json:"organizationId"   keel:"organization_id,pk,tenant"`
	Status           Status   `json:"status"           keel:"status"`
	HistoryWeeks     int32    `json:"historyWeeks"     keel:"history_weeks"`
	HistoryLoads     int64    `json:"historyLoads"     keel:"history_loads"`
	WeeksTested      int32    `json:"weeksTested"      keel:"weeks_tested"`
	LoadsTested      int64    `json:"loadsTested"      keel:"loads_tested"`
	IntervalCoverage float64  `json:"intervalCoverage" keel:"interval_coverage"`
	MAE              float64  `json:"mae"              keel:"mae"`
	NaiveMAE         float64  `json:"naiveMae"         keel:"naive_mae"`
	Skill            float64  `json:"skill"            keel:"skill"`
	FittedAt         int64    `json:"fittedAt"         keel:"fitted_at"`
	Version          int64    `json:"version"          keel:"version,version"`
	CreatedAt        int64    `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64    `json:"updatedAt"        keel:"updated_at,autoupdate"`
}

func (q *Quality) Ready() bool {
	return q != nil && q.Status == StatusReady
}
