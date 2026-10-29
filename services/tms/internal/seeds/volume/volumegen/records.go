package volumegen

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type CustomerRecord struct {
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
	Home         *CityRecord
	Facilities   []*LocationRecord
	Factoring    []*FactorPeriod
}

type LocationRecord struct {
	ID        pulid.ID
	Code      string
	Name      string
	City      *CityRecord
	Address   string
	Shipper   bool
	Consignee bool
}

type CommodityRecord struct {
	ID             pulid.ID
	Name           string
	WeightPerPiece int64
	Hazmat         bool
	TempMin        *int16
	TempMax        *int16
}

type Crew struct {
	WorkerID          pulid.ID
	SecondaryWorkerID *pulid.ID
	TractorID         pulid.ID
	TractorTypeID     pulid.ID
	TrailerID         pulid.ID
	TrailerTypeID     pulid.ID
	TrailerKind       TrailerKind
	Hazmat            bool
	HomeState         string
}

type TrailerKind string

const (
	TrailerKindDry     = TrailerKind("dry")
	TrailerKindReefer  = TrailerKind("reefer")
	TrailerKindFlatbed = TrailerKind("flatbed")
	TrailerKindOther   = TrailerKind("other")
)

func TrailerKindOf(et *equipmenttype.EquipmentType) TrailerKind {
	text := strings.ToUpper(et.Code + " " + et.Description)
	switch {
	case strings.Contains(text, "REEF"), strings.Contains(text, "REFRIGERAT"):
		return TrailerKindReefer
	case strings.Contains(text, "FLAT"), strings.Contains(text, "STEP DECK"):
		return TrailerKindFlatbed
	case strings.Contains(text, "DRY"), strings.Contains(text, "VAN"),
		strings.EqualFold(et.Code, "TRAILER"):
		return TrailerKindDry
	default:
		return TrailerKindOther
	}
}

type DriverRecord struct {
	WorkerID    pulid.ID
	FleetCodeID pulid.ID
	Type        worker.WorkerType
	DriverType  worker.DriverType
	Active      bool
	HireDate    int64
	Home        *CityRecord
	TractorID   pulid.ID
}

type StopRecord struct {
	ID              pulid.ID
	Sequence        int64
	Type            shipmentvocab.StopType
	Status          shipmentvocab.StopStatus
	Location        *LocationRecord
	Appointment     bool
	WindowStart     int64
	WindowEnd       int64
	ActualArrival   int64
	ActualDeparture int64
	Miles           float64
}

func (s *StopRecord) IsOrigin() bool {
	return s.Type == shipmentvocab.StopTypePickup || s.Type == shipmentvocab.StopTypeSplitPickup
}

func (s *StopRecord) Arrived() bool {
	return s.ActualArrival > 0
}

type MoveRecord struct {
	ID                pulid.ID
	Sequence          int64
	Status            shipmentvocab.MoveStatus
	Miles             float64
	Cities            []*CityRecord
	Stops             []*StopRecord
	StartedAt         int64
	CompletedAt       int64
	Crew              *Crew
	AssignmentID      pulid.ID
	CarrierAssignment *shipment.CarrierAssignment
}

type billingStage int

const (
	BillingStageNone billingStage = iota
	BillingStageQueued
	BillingStageInvoiced
)

type ChargeRecord struct {
	AccessorialChargeID pulid.ID
	AdditionalChargeID  pulid.ID
	Code                string
	Description         string
	Method              accessorialcharge.Method
	Unit                int64
	UnitPrice           decimal.Decimal
	Amount              decimal.Decimal
}

type ShipmentRecord struct {
	ID                pulid.ID
	OrderID           pulid.ID
	ProNumber         string
	BOL               string
	OrderNumber       string
	PONumber          string
	Customer          *CustomerRecord
	Status            shipmentvocab.Status
	FormulaTemplateID pulid.ID
	FormulaName       string
	CreatedAt         int64
	PickupAt          int64
	DeliveredAt       int64
	Miles             float64
	Freight           decimal.Decimal
	FreightLines      []shipment.FreightLine
	Charges           []ChargeRecord
	Total             decimal.Decimal
	Stage             billingStage
	QueueStatus       billingqueue.Status
	ReadyToBillAt     int64
	TransferredAt     int64
	InvoiceDate       int64
	Moves             []*MoveRecord
	Invoice           *invoice.Invoice
}
