package toolparams

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const ParamShipmentIDs = "shipmentIds"

const ParamBillType = "billType"

const ParamCarrierID = "carrierId"

const ParamTractorID = "tractorId"

const ParamFuelType = "fuelType"

const KeepWhenLeftOut = " Leave it out to keep it."

var FuelTypes = agenttoolschema.Source("ifta.fuelType", domaintypes.IFTAFuelTypeValues())

const ParamShipmentMoveID = "shipmentMoveId"

const ParamDisputeID = "disputeId"

const ParamOrderID = "orderId"

const ParamForceResend = "force"

const ParamCustomerID = "customerId"

const ParamCustomerIDs = "customerIds"

const ParamPeriodStart = "periodStart"

const ParamPeriodEnd = "periodEnd"

const ParamInvoiceDate = "invoiceDate"

const ParamItemID = "itemId"

const DayLayout = "2006-01-02"

const ParamInvoiceID = "invoiceId"

const MaxOperationNoteChars = 2000

const ParamStopID = "stopId"

const ParamAmount = "amount"

const ParamNote = "note"

const CurrencyCodeLength = 3

const ParamWorkerID = "workerId"

const SecondsPerDay = int64(24 * time.Hour / time.Second)

const ParamShipmentID = "shipmentId"

const ParamOwnerID = "ownerId"

const ParamEffectiveFrom = "effectiveFrom"

const ParamEffectiveTo = "effectiveTo"

const ParamAccountingDate = "accountingDate"

const ParamReason = "reason"

var ReportFormats = agenttoolschema.Source("report.format", report.AllFormats())

const ParamPayCodeID = "payCodeId"

const ParamGLAccountID = "glAccountId"

const MaxAdjustmentDescription = 255

const ParamJurisdictionID = "jurisdictionId"

const SearchTermIFTA = "ifta"

const ParamCurrencyCode = "currencyCode"

const SchemaItems = "items"

const ParamAdjustments = "adjustments"

const ParamAdjustmentLines = "lines"

const ParamLineDescription = "description"

const ToolInvoiceGet = "invoice_get"

const ToolShipmentGetTracking = "shipment_get_tracking"
