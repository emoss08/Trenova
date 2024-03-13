package models

import (
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AutoBillShipmentType string

const (
	// AutoBillShipmentDelivery is a constant for the AutoBillShipmentType enum. When a shipment is delivered.
	AutoBillShipmentDelivery AutoBillShipmentType = "D"

	// AutoBillShipmentTransferred is a constant for the AutoBillShipmentType enum. When a shipment is transferred to billing queue.
	AutoBillShipmentTransferred AutoBillShipmentType = "T"

	// AutoBillingMarkedReady is a constant for the AutoBillShipmentType enum. When a shipment is marked ready to bill.
	AutoBillingMarkedReady AutoBillShipmentType = "MR"
)

type ShipmentTransferCriteriaType string

const (
	// ShipmentTransferCriteriaRAndC  is a constant for the ShipmentTransferCriteriaType enum. When a shipment is ready to bill and confirmed.
	ShipmentTransferCriteriaRAndC ShipmentTransferCriteriaType = "RC"

	// ShipmentTransferCriteriaCompleted is a constant for the ShipmentTransferCriteriaType enum. When a shipment is completed.
	ShipmentTransferCriteriaCompleted ShipmentTransferCriteriaType = "C"

	// ShipmentTransferCriteriaReady is a constant for the ShipmentTransferCriteriaType enum. When a shipment is ready to bill.
	ShipmentTransferCriteriaReady ShipmentTransferCriteriaType = "RTB"
)

type BillingControl struct {
	BaseModel
	BusinessUnitID           uuid.UUID                    `gorm:"type:uuid;not null;index"                          json:"businessUnitId"`
	OrganizationID           uuid.UUID                    `gorm:"type:uuid;not null;unique"                         json:"organizationId"`
	RemoveBillingHistory     bool                         `gorm:"type:boolean;not null;default:false"               json:"removeBillingHistory"     validate:"omitempty"`
	AutoBillShipment         bool                         `gorm:"type:boolean;not null;default:false"               json:"autoBillShipment"         validate:"omitempty"`
	AutoMarkReadyToBill      bool                         `gorm:"type:boolean;not null;default:false"               json:"autoMarkReadyToBill"      validate:"omitempty"`
	ValidateCustomerRates    bool                         `gorm:"type:boolean;not null;default:false"               json:"validateCustomerRates"    validate:"omitempty"`
	EnforceCustomerBilling   bool                         `gorm:"type:boolean;not null;default:false"               json:"enforceCustomerBilling"   validate:"omitempty"`
	AutoBillCriteria         AutoBillShipmentType         `gorm:"type:auto_billing_shipment_type;default:'MR'"      json:"autoBillCriteria"         validate:"omitempty,oneof=D T MR"`
	ShipmentTransferCriteria ShipmentTransferCriteriaType `gorm:"type:shipment_transfer_criteria_type;default:'RC'" json:"shipmentTransferCriteria" validate:"omitempty,oneof=RC C RTB"`
}

func (bc *BillingControl) BeforeCreate(_ *gorm.DB) error {
	return bc.validateBillingControl()
}

func (bc *BillingControl) BeforeUpdate(_ *gorm.DB) error {
	return bc.validateBillingControl()
}

var errAutoBillCriteria = errors.New("AutoBillCriteria must be set if AutoBillShipment is true")

func (bc *BillingControl) validateBillingControl() error {
	// if autoBillShipment is true and not AutoBillCriteria is set, return an error
	if bc.AutoBillShipment && bc.AutoBillCriteria == "" {
		return errAutoBillCriteria
	}

	return nil
}

type (
	AutomaticJournalEntryType string
	AccountingAccountType     string
	ThresholdActionType       string
)

const (
	RevenueAccountType AccountingAccountType = "REVENUE"
	ExpenseAccountType AccountingAccountType = "EXPENSE"
)

type AccountingControl struct {
	BaseModel
	BusinessUnitID               uuid.UUID                  `gorm:"type:uuid;not null;index"                              json:"businessUnitId"`
	OrganizationID               uuid.UUID                  `gorm:"type:uuid;not null;unique"                             json:"organizationId"`
	RecThreshold                 int64                      `gorm:"type:int;not null;default:50"                          json:"recThreshold"                 validate:"required"`
	DefaultRevenueAccountID      *uuid.UUID                 `gorm:"type:uuid"                                             json:"defaultRevenueAccountId"      validate:"omitempty"`
	DefaultExpenseAccountID      *uuid.UUID                 `gorm:"type:uuid"                                             json:"defaultExpenseAccountId"      validate:"omitempty"`
	BusinessUnit                 BusinessUnit               `json:"-" validate:"omitempty"`
	JournalEntryCriteria         *AutomaticJournalEntryType `gorm:"type:varchar(50);default:'ON_SHIPMENT_BILL'"           json:"journalEntryCriteria"         validate:"omitempty,oneof=ON_SHIPMENT_BILL ON_RECEIPT_OF_PAYMENT ON_EXPENSE_RECOGNITION"`
	RecThresholdAction           ThresholdActionType        `gorm:"type:ac_threshold_action_type;not null;default:'HALT'" json:"recThresholdAction"           validate:"required,oneof=HALT WARN"`
	DefaultRevenueAccount        *GeneralLedgerAccount      `gorm:"foreignKey:DefaultRevenueAccountID;references:ID"      json:"-"                            validate:"omitempty"`
	DefaultExpenseAccount        *GeneralLedgerAccount      `gorm:"foreignKey:DefaultExpenseAccountID;references:ID"      json:"-"                            validate:"omitempty"`
	AutoCreateJournalEntries     bool                       `gorm:"type:boolean;not null;default:false"                   json:"autoCreateJournalEntries"     validate:"omitempty"`
	RestrictManualJournalEntries bool                       `gorm:"type:boolean;not null;default:false"                   json:"restrictManualJournalEntries" validate:"omitempty"`
	RequireJournalEntryApproval  bool                       `gorm:"type:boolean;not null;default:false"                   json:"requireJournalEntryApproval"  validate:"omitempty"`
	EnableRecNotifications       bool                       `gorm:"type:boolean;not null;default:true"                    json:"enableRecNotifications"       validate:"omitempty"`
	HaltOnPendingRec             bool                       `gorm:"type:boolean;not null;default:false"                   json:"haltOnPendingRec"             validate:"omitempty"`
	CriticalProcesses            *string                    `gorm:"type:text"                                             json:"criticalProcesses"            validate:"omitempty"`
}

var (
	ErrExpenseAccount = errors.New("default expense account must be an expense account")
	ErrRevenueAccount = errors.New("default revenue account must be a revenue account")
)

func (ac *AccountingControl) validateAccountingControl() error {
	if ac.DefaultExpenseAccountID != nil && ac.DefaultExpenseAccount.AccountType != ExpenseAccountType {
		return ErrExpenseAccount
	}

	if ac.DefaultRevenueAccountID != nil && ac.DefaultRevenueAccount.AccountType != RevenueAccountType {
		return ErrRevenueAccount
	}

	return nil
}

func (ac *AccountingControl) BeforeCreate(_ *gorm.DB) error {
	return ac.validateAccountingControl()
}

func (ac *AccountingControl) BeforeUpdate(_ *gorm.DB) error {
	return ac.validateAccountingControl()
}

type ServiceIncidentType string

type DispatchControl struct {
	BaseModel
	BusinessUnitID               uuid.UUID           `gorm:"type:uuid;not null;index"                                                 json:"businessUnitId"`
	OrganizationID               uuid.UUID           `gorm:"type:uuid;not null;unique"                                                json:"organizationId"`
	RecordServiceIncident        ServiceIncidentType `gorm:"type:varchar(3);not null;default:'N'"                                     json:"recordServiceIncident"        validate:"required,oneof=N P PD AEP"`
	DeadheadTarget               *float64            `gorm:"type:numeric(5,2);default:0.00"                                           json:"deadheadTarget"               validate:"omitempty"`
	MaxShipmentWeightLimit       int                 `gorm:"type:integer;check:max_shipment_weight_limit >= 0;not null;default:80000" json:"maxShipmentWeightLimit"       validate:"required"`
	GracePeriod                  uint8               `gorm:"type:smallint;check:grace_period >= 0;not null;default:0"                 json:"gracePeriod"                  validate:"required"`
	EnforceWorkerAssign          bool                `gorm:"type:boolean;not null;default:true"                                       json:"enforceWorkerAssign"          validate:"omitempty"`
	TrailerContinuity            bool                `gorm:"type:boolean;not null;default:false"                                      json:"trailerContinuity"            validate:"omitempty"`
	DupeTrailerCheck             bool                `gorm:"type:boolean;not null;default:false"                                      json:"dupeTrailerCheck"             validate:"omitempty"`
	MaintenanceCompliance        bool                `gorm:"type:boolean;not null;default:true"                                       json:"maintenanceCompliance"        validate:"omitempty"`
	RegulatoryCheck              bool                `gorm:"type:boolean;not null;default:false"                                      json:"regulatoryCheck"              validate:"omitempty"`
	PrevShipmentOnHold           bool                `gorm:"type:boolean;not null;default:false"                                      json:"prevShipmentOnHold"           validate:"omitempty"`
	WorkerTimeAwayRestriction    bool                `gorm:"type:boolean;not null;default:true"                                       json:"workerTimeAwayRestriction"    validate:"omitempty"`
	TractorWorkerFleetConstraint bool                `gorm:"type:boolean;not null;default:false"                                      json:"tractorWorkerFleetConstraint" validate:"omitempty"`
}

type OperatorChoices string

type FeasibilityToolControl struct {
	BaseModel
	BusinessUnitID uuid.UUID       `gorm:"type:uuid;not null;index"                                         json:"businessUnitId"`
	OrganizationID uuid.UUID       `gorm:"type:uuid;not null;unique"                                        json:"organizationId"`
	OtpOperator    OperatorChoices `gorm:"type:varchar(3);not null;default:'EQ'"                            json:"otpOperator" validate:"required,oneof=EQ NE GT LT LTE"`
	MpwOperator    OperatorChoices `gorm:"type:varchar(3);not null;default:'EQ'"                            json:"mpwOperator" validate:"required,oneof=EQ NE GT LT LTE"`
	MpdOperator    OperatorChoices `gorm:"type:varchar(3);not null;default:'EQ'"                            json:"mpdOperator" validate:"required,oneof=EQ NE GT LT LTE"`
	MpgOperator    OperatorChoices `gorm:"type:varchar(3);not null;default:'EQ'"                            json:"mpgOperator" validate:"required,oneof=EQ NE GT LT LTE"`
	MpwCriteria    float64         `gorm:"type:decimal(10,2);check:mpw_criteria >= 0;not null;default:100"  json:"mpwCriteria" validate:"required,gt=0"`
	MpdCriteria    float64         `gorm:"type:decimal(10,2);check:mpd_criteria >= 0;not null;default:100"  json:"mpdCriteria" validate:"required,gt=0"`
	MpgCriteria    float64         `gorm:"type:decimal(10,2);check:mpg_criteria >= 0;not null;default:100"  json:"mpgCriteria" validate:"required,gt=0"`
	OtpCriteria    float64         `gorm:"type:decimal(10,2);check:otp_criteria >= 0;not null;default:100"  json:"otpCriteria" validate:"required,gt=0"`
}

type DateFormatType string

type InvoiceControl struct {
	BaseModel
	BusinessUnitID         uuid.UUID      `gorm:"type:uuid;not null;index"                                            json:"businessUnitId"`
	OrganizationID         uuid.UUID      `gorm:"type:uuid;not null;unique"                                           json:"organizationId"`
	InvoiceNumberPrefix    string         `gorm:"type:varchar(10);not null;default:'INV-'"                            json:"invoiceNumberPrefix"    validate:"required,max=10"`
	CreditMemoNumberPrefix string         `gorm:"type:varchar(10);not null;default:'CM-'"                             json:"creditMemoNumberPrefix" validate:"required,max=10"`
	InvoiceTerms           string         `gorm:"type:text"                                                           json:"invoiceTerms"           validate:"omitempty"`
	InvoiceFooter          string         `gorm:"type:text"                                                           json:"invoiceFooter"          validate:"omitempty"`
	InvoiceLogoURL         string         `gorm:"type:varchar(255);"                                                  json:"invoiceLogoUrl"         validate:"omitempty,url"`
	InvoiceDateFormat      DateFormatType `gorm:"type:varchar(10);not null;default:'01/02/2006'"                      json:"invoiceDateFormat"      validate:"required,oneof=01/02/2006 02/01/2006 2006/02/01 2006/01/02"`
	InvoiceDueAfterDays    uint8          `gorm:"type:smallint;check:invoice_due_after_days >= 0;not null;default:30" json:"invoiceDueAfterDays"    validate:"required"`
	InvoiceLogoWidth       uint16         `gorm:"type:smallint;check:invoice_logo_width >= 0;not null;default:100"    json:"invoiceLogoWidth"       validate:"required"`
	ShowAmountDue          bool           `gorm:"type:boolean;not null;default:true"                                  json:"showAmountDue"          validate:"omitempty"`
	AttachPDF              bool           `gorm:"type:boolean;not null;default:true"                                  json:"attachPdf"              validate:"omitempty"`
	ShowInvoiceDueDate     bool           `gorm:"type:boolean;not null;default:true"                                  json:"showInvoiceDueDate"     validate:"omitempty"`
}

type (
	RouteDistanceUnitType string
	DistanceMethodType    string
)

const TrenovaDistanceMethod DistanceMethodType = "T"

type RouteControl struct {
	BaseModel
	BusinessUnitID uuid.UUID             `gorm:"type:uuid;not null;index"             json:"businessUnitId"`
	OrganizationID uuid.UUID             `gorm:"type:uuid;not null;unique"            json:"organizationId"`
	DistanceMethod DistanceMethodType    `gorm:"type:varchar(1);not null;default:'T'" json:"distanceMethod" validate:"required,oneof=T G"`
	MileageUnit    RouteDistanceUnitType `gorm:"type:varchar(1);not null;default:'M'" json:"mileageUnit"    validate:"required,oneof=M I"`
	GenerateRoutes bool                  `gorm:"type:boolean;not null;default:false"  json:"generateRoutes" validate:"omitempty"`
}

func (rc *RouteControl) BeforeCreate(_ *gorm.DB) error {
	return rc.validateRouteControl()
}

func (rc *RouteControl) BeforeUpdate(_ *gorm.DB) error {
	return rc.validateRouteControl()
}

var errGenerateRoutesWithTrenova = errors.New("cannot use generate routes with Trenova distance method")

func (rc *RouteControl) validateRouteControl() error {
	// Disallow using the generate routes if the distance method is set to Trenova
	if rc.DistanceMethod == TrenovaDistanceMethod && rc.GenerateRoutes {
		return &AttributeError{
			Attr:    "generateRoutes",
			Message: errGenerateRoutesWithTrenova.Error(),
		}
	}

	return nil
}

type ShipmentControl struct {
	BaseModel
	BusinessUnitID           uuid.UUID `gorm:"type:uuid;not null;index"            json:"businessUnitId"`
	OrganizationID           uuid.UUID `gorm:"type:uuid;not null;unique"           json:"organizationId"`
	AutoRateShipment         bool      `gorm:"type:boolean;not null;default:true"  json:"autoRateShipment"         validate:"omitempty"`
	CalculateDistance        bool      `gorm:"type:boolean;not null;default:true"  json:"calculateDistance"        validate:"omitempty"`
	EnforceRevCode           bool      `gorm:"type:boolean;not null;default:false" json:"enforceRevCode"           validate:"omitempty"`
	EnforceVoidedComm        bool      `gorm:"type:boolean;not null;default:false" json:"enforceVoidedComm"        validate:"omitempty"`
	GenerateRoutes           bool      `gorm:"type:boolean;not null;default:false" json:"generateRoutes"           validate:"omitempty"`
	EnforceCommodity         bool      `gorm:"type:boolean;not null;default:false" json:"enforceCommodity"         validate:"omitempty"`
	AutoSequenceStops        bool      `gorm:"type:boolean;not null;default:true"  json:"autoSequenceStops"        validate:"omitempty"`
	AutoShipmentTotal        bool      `gorm:"type:boolean;not null;default:true"  json:"autoShipmentTotal"        validate:"omitempty"`
	EnforceOriginDestination bool      `gorm:"type:boolean;not null;default:false" json:"enforceOriginDestination" validate:"omitempty"`
	CheckForDuplicateBOL     bool      `gorm:"type:boolean;not null;default:false" json:"checkForDuplicateBol"     validate:"omitempty"`
	SendPlacardInfo          bool      `gorm:"type:boolean;not null;default:false" json:"sendPlacardInfo"          validate:"omitempty"`
	EnforceHazmatSegRules    bool      `gorm:"type:boolean;not null;default:true"  json:"enforceHazmatSegRules"    validate:"omitempty"`
}
