package volumeseed

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type customerRecord struct {
	ID           pulid.ID
	Code         string
	Name         string
	AddressLine1 string
	City         string
	StateAbbr    string
	PostalCode   string
	PaymentTerm  invoice.PaymentTerm
	BillingEmail string
	Weight       int
	Home         *cityRecord
	Facilities   []*locationRecord
	Factoring    []*facPeriod
}

type locationRecord struct {
	ID        pulid.ID
	Code      string
	Name      string
	City      *cityRecord
	Address   string
	Shipper   bool
	Consignee bool
}

type commodityRecord struct {
	ID             pulid.ID
	Name           string
	WeightPerPiece int64
	Hazmat         bool
	TempMin        *int16
	TempMax        *int16
}

type crew struct {
	WorkerID          pulid.ID
	SecondaryWorkerID *pulid.ID
	TractorID         pulid.ID
	TractorTypeID     pulid.ID
	TrailerID         pulid.ID
	TrailerTypeID     pulid.ID
	TrailerKind       trailerKind
	Hazmat            bool
	HomeState         string
}

type trailerKind string

const (
	trailerKindDry     = trailerKind("dry")
	trailerKindReefer  = trailerKind("reefer")
	trailerKindFlatbed = trailerKind("flatbed")
	trailerKindOther   = trailerKind("other")
)

func trailerKindOf(et *equipmenttype.EquipmentType) trailerKind {
	text := strings.ToUpper(et.Code + " " + et.Description)
	switch {
	case strings.Contains(text, "REEF"), strings.Contains(text, "REFRIGERAT"):
		return trailerKindReefer
	case strings.Contains(text, "FLAT"), strings.Contains(text, "STEP DECK"):
		return trailerKindFlatbed
	case strings.Contains(text, "DRY"), strings.Contains(text, "VAN"),
		strings.EqualFold(et.Code, "TRAILER"):
		return trailerKindDry
	default:
		return trailerKindOther
	}
}

type driverRecord struct {
	WorkerID    pulid.ID
	FleetCodeID pulid.ID
	Type        worker.WorkerType
	DriverType  worker.DriverType
	Active      bool
	HireDate    int64
	Home        *cityRecord
	TractorID   pulid.ID
}

type moveRecord struct {
	ID                pulid.ID
	Sequence          int64
	Status            shipment.MoveStatus
	Miles             float64
	Cities            []*cityRecord
	StartedAt         int64
	CompletedAt       int64
	Crew              *crew
	AssignmentID      pulid.ID
	CarrierAssignment *shipment.CarrierAssignment
}

type billingStage int

const (
	billingStageNone billingStage = iota
	billingStageQueued
	billingStageInvoiced
)

type chargeRecord struct {
	AccessorialChargeID pulid.ID
	AdditionalChargeID  pulid.ID
	Code                string
	Description         string
	Method              accessorialcharge.Method
	Unit                int64
	UnitPrice           decimal.Decimal
	Amount              decimal.Decimal
}

type shipmentRecord struct {
	ID                pulid.ID
	OrderID           pulid.ID
	ProNumber         string
	BOL               string
	OrderNumber       string
	PONumber          string
	Customer          *customerRecord
	Status            shipment.Status
	FormulaTemplateID pulid.ID
	FormulaName       string
	CreatedAt         int64
	PickupAt          int64
	DeliveredAt       int64
	Miles             float64
	Freight           decimal.Decimal
	FreightLines      []shipment.FreightLine
	Charges           []chargeRecord
	Total             decimal.Decimal
	Stage             billingStage
	QueueStatus       billingqueue.Status
	ReadyToBillAt     int64
	TransferredAt     int64
	InvoiceDate       int64
	Moves             []*moveRecord
	Invoice           *invoice.Invoice
}
