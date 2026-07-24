package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ProviderVehicle struct {
	ID           string
	Name         string
	VIN          string
	LicensePlate string
}

type ProviderPosition struct {
	VehicleID         string
	Latitude          float64
	Longitude         float64
	HeadingDegrees    float64
	SpeedMph          float64
	EngineState       telematics.EngineState
	FuelPercent       *float64
	OdometerMeters    *int64
	FormattedLocation string
	RecordedAt        int64
}

type ProviderRuleset struct {
	Cycle        string
	Shift        string
	Restart      string
	Break        string
	Jurisdiction string
}

type ProviderDriverProfile struct {
	DriverID string
	Name     string
	Ruleset  *ProviderRuleset
}

type ProviderHOSClocks struct {
	DriverID                string
	DutyStatus              telematics.DutyStatus
	DriveRemainingMs        int64
	ShiftRemainingMs        int64
	CycleRemainingMs        int64
	CycleTomorrowMs         int64
	BreakRemainingMs        int64
	CycleStartedAt          *int64
	ShiftDrivingViolationMs int64
	CycleViolationMs        int64
	CurrentVehicleID        string
}

type ProviderViolation struct {
	DriverID    string
	Type        string
	Description string
	DurationMs  int64
	StartAt     int64
	DayStartAt  *int64
	DayEndAt    *int64
}

type ProviderHOSLogEntry struct {
	HosStatusType string   `json:"hosStatusType"`
	LogStartAt    int64    `json:"logStartAt"`
	LogEndAt      *int64   `json:"logEndAt,omitempty"`
	Remark        string   `json:"remark,omitempty"`
	VehicleID     string   `json:"vehicleId,omitempty"`
	VehicleName   string   `json:"vehicleName,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	Codrivers     []string `json:"codrivers,omitempty"`
}

type ProviderHOSDailyLog struct {
	StartAt                      int64    `json:"startAt"`
	EndAt                        int64    `json:"endAt"`
	DriveDistanceMeters          int64    `json:"driveDistanceMeters"`
	ActiveDurationMs             int64    `json:"activeDurationMs"`
	DriveDurationMs              int64    `json:"driveDurationMs"`
	OnDutyDurationMs             int64    `json:"onDutyDurationMs"`
	OffDutyDurationMs            int64    `json:"offDutyDurationMs"`
	SleeperBerthDurationMs       int64    `json:"sleeperBerthDurationMs"`
	PersonalConveyanceDurationMs int64    `json:"personalConveyanceDurationMs"`
	YardMoveDurationMs           int64    `json:"yardMoveDurationMs"`
	IsCertified                  bool     `json:"isCertified"`
	CertifiedAt                  *int64   `json:"certifiedAt,omitempty"`
	ShippingDocs                 string   `json:"shippingDocs,omitempty"`
	VehicleNames                 []string `json:"vehicleNames,omitempty"`
}

type ProviderDVIRDefect struct {
	ID         string
	DefectType string
	Comment    string
	Resolved   bool
	ResolvedAt *int64
}

type ProviderDVIR struct {
	ID             string
	Type           string
	SafetyStatus   string
	DriverID       string
	DriverName     string
	VehicleID      string
	TrailerID      string
	TrailerName    string
	StartAt        int64
	EndAt          int64
	OdometerMeters *int64
	Location       string
	Signed         bool
	Defects        []ProviderDVIRDefect
}

type ProviderFormField struct {
	Label string
	Type  string
	Value string
}

type ProviderFormSubmission struct {
	ID           string
	TemplateID   string
	TemplateName string
	DriverID     string
	RouteStopID  string
	ExternalIDs  map[string]string
	SubmittedAt  int64
	Fields       []ProviderFormField
}

type ProviderGeofenceEvent struct {
	AddressName        string
	AddressExternalIDs map[string]string
	VehicleID          string
	VehicleVIN         string
	DriverID           string
}

type ProviderStopEvent struct {
	RouteStopID        string
	StopExternalIDs    map[string]string
	AddressExternalIDs map[string]string
	VehicleID          string
	VehicleVIN         string
	DriverID           string
	OccurredAt         int64
}

type ProviderFormEvent struct {
	SubmissionID string
	TemplateID   string
	TemplateName string
	RouteStopID  string
	ExternalIDs  map[string]string
	DriverID     string
	VehicleID    string
	SubmittedAt  int64
	Fields       []ProviderFormField
}

type ProviderEventKind string

const (
	ProviderEventKindGeofenceEntry  = ProviderEventKind("geofenceEntry")
	ProviderEventKindGeofenceExit   = ProviderEventKind("geofenceExit")
	ProviderEventKindStopArrival    = ProviderEventKind("stopArrival")
	ProviderEventKindStopDeparture  = ProviderEventKind("stopDeparture")
	ProviderEventKindFormSubmission = ProviderEventKind("formSubmission")
	ProviderEventKindOther          = ProviderEventKind("other")
)

type ProviderWebhookEvent struct {
	EventID    string
	EventType  string
	Kind       ProviderEventKind
	OccurredAt int64
	Payload    []byte
	Geofence   *ProviderGeofenceEvent
	Stop       *ProviderStopEvent
	Form       *ProviderFormEvent
	VehicleID  string
	DriverID   string
}

type TelematicsProvider interface {
	Type() integration.Type
	ListVehicles(ctx context.Context) ([]ProviderVehicle, error)
	ListPositions(ctx context.Context) ([]ProviderPosition, error)
	ListHOSClocks(ctx context.Context) ([]ProviderHOSClocks, error)
	ListDriverProfiles(ctx context.Context) ([]ProviderDriverProfile, error)
	ListHOSViolations(
		ctx context.Context,
		startAt int64,
		endAt int64,
	) ([]ProviderViolation, error)
	ListHOSLogs(
		ctx context.Context,
		driverID string,
		startAt int64,
		endAt int64,
	) ([]ProviderHOSLogEntry, error)
	ListHOSDailyLogs(
		ctx context.Context,
		driverID string,
		startDate string,
		endDate string,
	) ([]ProviderHOSDailyLog, error)
	ListTrailers(ctx context.Context) ([]ProviderVehicle, error)
	ListDVIRs(
		ctx context.Context,
		startAt int64,
		endAt int64,
	) ([]ProviderDVIR, error)
	ListFormSubmissions(
		ctx context.Context,
		driverID string,
		startAt int64,
		endAt int64,
	) ([]ProviderFormSubmission, error)
	VerifyWebhookSignature(
		secret string,
		timestamp string,
		body []byte,
		signature string,
		now time.Time,
		maxSkew time.Duration,
	) error
	ParseWebhookEvent(body []byte) (*ProviderWebhookEvent, error)
}

type TelematicsProviderFactory interface {
	ProviderFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (TelematicsProvider, error)
	ProviderOfType(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (TelematicsProvider, error)
}
