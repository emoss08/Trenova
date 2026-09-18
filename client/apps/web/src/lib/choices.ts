import { LOCALE_NAMES, LOCALES, type Locale } from "@trenova/shared/i18n/generated/locales";
import type { AccountCategory } from "@/types/account-type";
import type {
  AccountingBasis,
  ClosedPeriodPostingPolicy,
  CurrencyMode,
  ExchangeRateDatePolicy,
  ExchangeRateOverridePolicy,
  ExpenseRecognitionPolicy,
  JournalPostingMode,
  JournalReversalPolicy,
  JournalSourceEvent,
  LockedPeriodPostingPolicy,
  ManualJournalEntryPolicy,
  PeriodCloseMode,
  ReconciliationMode,
  RevenueRecognitionPolicy,
} from "@/types/accounting-control";
import type { BankReceiptStatus } from "@/types/bank-receipt";
import type { BankReceiptBatchStatus } from "@/types/bank-receipt-batch";
import type { ResolutionType, WorkItemStatus } from "@/types/bank-receipt-work-item";
import type {
  BillingExceptionDisposition,
  BillingQueueTransferMode,
  EnforcementLevel,
  InvoiceDraftCreationMode,
  InvoicePostingMode,
  LateChargeAssessmentMode,
  PaymentTerm,
  RateVarianceAutoResolutionMode,
  ReadyToBillAssignmentMode,
  TransferSchedule,
  UnratedShipmentDisposition,
} from "@/types/billing-control";
import type { FieldType } from "@/types/custom-field";
import type { CaseFormat } from "@/types/data-entry-control";
import type {
  AutoAssignmentStrategy,
  ComplianceEnforcementLevel,
  ServiceIncidentType,
} from "@/types/dispatch-control";
import type { DistanceProfile } from "@/types/distance-profile";
import type { ResourceType } from "@/types/document-packet-rule";
import type { DocumentKind } from "@/types/document-parsing-rule";
import type { EquipmentClass } from "@/types/equipment-type";
import type { FiscalPeriodStatus, PeriodType } from "@/types/fiscal-period";
import type { FiscalYearStatus } from "@/types/fiscal-year";
import type {
  FuelIndexSource,
  FuelSurchargeDateBasis,
  FuelSurchargeFallback,
  FuelSurchargePercentBasis,
  FuelSurchargeProgramMethod,
  FuelSurchargeProgramStatus,
  FuelSurchargeRateRounding,
  FuelSurchargeStepRounding,
  FuelType,
} from "@/types/fuel-surcharge";
import type { HazardousClass, PackingGroup } from "@/types/hazardous-material";
import type { SegregationDistanceUnit, SegregationType } from "@/types/hazmat-segregation-rule";
import type { HoldSeverity, HoldType } from "@/types/hold-reason";
import type {
  AdjustmentAccountingDatePolicy,
  AdjustmentEligibilityPolicy,
  ApprovalPolicy,
  ClosedPeriodAdjustmentPolicy,
  CustomerCreditBalancePolicy,
  OverCreditPolicy,
  ReplacementInvoiceReviewPolicy,
  RequirementPolicy,
  SupersededInvoiceVisibilityPolicy,
  WriteOffApprovalPolicy,
} from "@/types/invoice-adjustment-control";
import type { JournalReversalStatus } from "@/types/journal-reversal";
import type { FacilityType, LocationCategoryType } from "@/types/location-category";
import type { ManualJournalStatus } from "@/types/manual-journal";
import type {
  RecurringShipmentExceptionPolicy,
  RecurringShipmentStatus,
} from "@/types/recurring-shipment";
import type {
  ServiceFailureSource,
  ServiceFailureStatus,
  ServiceFailureType,
} from "@/types/service-failure";
import type {
  ServiceFailureReasonCategory,
  ServiceFailureReasonCodeAppliesTo,
} from "@/types/service-failure-reason-code";
import type { CarrierIntelRiskLevel } from "@trenova/graphql/generated/graphql";
import type { AccessorialChargeMethod, RateUnit } from "@trenova/shared/types/accessorial-charge";
import type { BillingQueueStatus, ExceptionReasonCode } from "@trenova/shared/types/billing-queue";
import type {
  CarrierComplianceStatus,
  CarrierInsurancePolicyType,
  CarrierPaymentMethod,
  CarrierSafetyRating,
  CarrierStatus,
  CarrierTaxIdType,
  CarrierType,
} from "@trenova/shared/types/carrier";
import type {
  CarrierCostEventStatus,
  CarrierCostEventType,
  CarrierInvoiceMatchStatus,
  CarrierLedgerEntryType,
  CarrierSettlementBatchStatus,
  CarrierSettlementStatus,
} from "@trenova/shared/types/carrier-settlement";
import type { FreightClass } from "@trenova/shared/types/commodity";
import type {
  BillingCycle,
  InvoiceDelivery,
  CreditStatus,
  CustomerFuelSurchargeMode,
  CustomerPaymentTerm,
  InvoiceDetail,
  InvoiceSectionKey,
  InvoiceSplitKey,
  InvoiceNumberFormat,
} from "@trenova/shared/types/customer";
import type { PaymentMethod } from "@trenova/shared/types/customer-payment";
import type { DocumentCategory, DocumentClassification } from "@trenova/shared/types/document-type";
import type {
  PayAdvanceSource,
  PayCalcMethod,
  PayCodeDirection,
  PayComponentKind,
  PayPeriodFrequency,
  PayRevenueBasis,
  PayeeClassification,
  RecurringDeductionFrequency,
  RecurringDeductionStatus,
  RecurringEarningFrequency,
  RecurringEarningStatus,
  SettlementPayTrigger,
} from "@trenova/shared/types/driver-pay";
import type {
  EDIInboundFileStatus,
  EDIMessageAcknowledgmentStatus,
  EDIMessageDeliveryStatus,
  EDITransferStatus,
} from "@trenova/shared/types/edi";
import type {
  GenericSelectOption,
  SelectOption,
  SelectOptionGroup,
} from "@trenova/shared/types/fields";
import type {
  FormulaTemplateStatus,
  FormulaTemplateType,
} from "@trenova/shared/types/formula-template";
import {
  FUEL_CARD_PROVIDER_LABELS,
  FUEL_CARD_STATUS_LABELS,
  FUEL_PURCHASE_IMPORT_STATUS_LABELS,
  FUEL_PURCHASE_SOURCE_LABELS,
  FUEL_QUANTITY_UNIT_LABELS,
  IFTA_FUEL_TYPE_LABELS,
  IFTA_MILEAGE_SOURCE_LABELS,
  IFTA_QUARTER_LABELS,
  IFTA_RETURN_STATUS_LABELS,
  fuelCardProviderSchema,
  fuelPurchaseSourceSchema,
  fuelQuantityUnitSchema,
  iftaFuelTypeSchema,
  iftaMileageSourceSchema,
  iftaQuarterSchema,
  type FuelCardProvider,
  type FuelCardStatus,
  type FuelPurchaseImportStatus,
  type FuelPurchaseSource,
  type FuelQuantityUnit,
  type IftaFuelType,
  type IftaMileageSource,
  type IftaQuarter,
  type IftaReturnStatus,
} from "@trenova/shared/types/fuel-ifta-enums";
import type { EquipmentStatus, Status } from "@trenova/shared/types/helpers";
import type {
  InvoiceDisputeReasonCode,
  InvoiceDisputeResolution,
  InvoiceEdiSendStatus,
  InvoiceScope,
  InvoiceStatus,
  InvoiceVoidDisposition,
} from "@trenova/shared/types/invoice";
import type { LocationGeofenceType } from "@trenova/shared/types/location";
import type { OrderStatus } from "@trenova/shared/types/order";
import type { RateConfirmationStatus } from "@trenova/shared/types/rate-confirmation";
import type {
  CoreResponsibility,
  DataScope,
  FieldSensitivity,
  Operation,
} from "@trenova/shared/types/role";
import type {
  CarrierAssignmentStatus,
  CarrierRateMethod,
  MoveStatus,
  ShipmentStatus,
  ShipmentTenderStatus,
  StopScheduleType,
  StopStatus,
  StopType,
} from "@trenova/shared/types/shipment";
import type { SpotTenderMode, TenderChannel } from "@trenova/shared/types/tender";
import type { TimeFormatType } from "@trenova/shared/types/user";
import type {
  CDLClass,
  ComplianceStatus,
  DriverType,
  EndorsementType,
  Gender,
  PTOStatus,
  PTOType,
  WorkerType,
} from "@trenova/shared/types/worker";
import type { DrugAlcoholStatus } from "@trenova/shared/types/worker-drug-alcohol-status";
import type { SafetyRating } from "@trenova/shared/types/worker-safety";
import type { WorkerTrainingHealth } from "@trenova/shared/types/worker-training";

export const formulaTemplateStatusChoices = [
  { label: "Active", value: "Active", color: "var(--success)" },
  { label: "Inactive", value: "Inactive", color: "var(--danger)" },
  { label: "Draft", value: "Draft", color: "var(--foreground-subtle)" },
  { label: "In Review", value: "InReview", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<FormulaTemplateStatus>>;

export const formulaTemplateTypeChoices = [
  { label: "Freight Charge", value: "FreightCharge" },
  { label: "Accessorial Charge", value: "AccessorialCharge" },
] satisfies ReadonlyArray<GenericSelectOption<FormulaTemplateType>>;

export const fuelIndexSourceChoices = [
  { label: "DOE / EIA", value: "EIA", color: "var(--info)" },
  { label: "Custom", value: "Custom", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<FuelIndexSource>>;

export const fuelTypeChoices = [
  { label: "Diesel", value: "Diesel", color: "var(--info)" },
  { label: "Gasoline", value: "Gasoline", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<FuelType>>;

export const fuelSurchargeMethodChoices = [
  {
    label: "Per Mile (Peg + Increment)",
    value: "PerMileStep",
    color: "var(--info)",
    description:
      "The classic truckload formula — the rate rises a set amount for every price step above your base price. No table to maintain.",
  },
  {
    label: "Per Mile (MPG Formula)",
    value: "PerMileMPG",
    color: "var(--accent-teal)",
    description:
      "Recovers actual fuel cost: (price − base price) ÷ fleet MPG. Common for owner-operators and cost-plus contracts.",
  },
  {
    label: "Custom Table: $ per Mile",
    value: "TablePerMile",
    color: "var(--accent-violet)",
    description:
      "Your own price bands, each with its own per-mile rate. Bands can be irregular — use this for customer-supplied fuel tables.",
  },
  {
    label: "Custom Table: % of Charge",
    value: "TablePercent",
    color: "var(--accent-violet-on-subtle)",
    description:
      "Price bands map to a percentage of the freight charge — the standard LTL and brokerage style.",
  },
  {
    label: "Custom Table: Flat Amount",
    value: "TableFlat",
    color: "var(--warning)",
    description: "Price bands map to a fixed dollar amount per shipment.",
  },
] satisfies ReadonlyArray<GenericSelectOption<FuelSurchargeProgramMethod>>;

export const fuelSurchargePercentBasisChoices = [
  {
    label: "Linehaul only",
    value: "Linehaul",
    description:
      "Percentage applies to the base freight charge only — the most common contract term.",
  },
  {
    label: "Linehaul + accessorials",
    value: "LinehaulPlusAccessorials",
    description:
      "Percentage applies to the freight charge plus other accessorial charges (gross basis).",
  },
] satisfies ReadonlyArray<GenericSelectOption<FuelSurchargePercentBasis>>;

export const customerFuelSurchargeModeChoices = [
  {
    label: "No fuel surcharge",
    value: "None",
    description: "Shipments for this customer never get an automatic fuel surcharge line.",
  },
  {
    label: "Fuel surcharge program",
    value: "Program",
    description: "Apply a fuel surcharge program — the correct week's rate is added automatically.",
  },
  {
    label: "Fuel included in rates",
    value: "FuelIncluded",
    description:
      "All-in pricing: fuel is baked into the negotiated rates, so no separate surcharge is billed.",
  },
] satisfies ReadonlyArray<GenericSelectOption<CustomerFuelSurchargeMode>>;

export const fuelSurchargeDateBasisChoices = [
  { label: "Pickup Date", value: "PickupDate" },
  { label: "Tender Date", value: "TenderDate" },
] satisfies ReadonlyArray<GenericSelectOption<FuelSurchargeDateBasis>>;

export const fuelSurchargeStepRoundingChoices = [
  { label: "Round Up (any partial step counts)", value: "Up" },
  { label: "Round Down", value: "Down" },
  { label: "Round Nearest", value: "Nearest" },
] satisfies ReadonlyArray<GenericSelectOption<FuelSurchargeStepRounding>>;

export const fuelSurchargeRateRoundingChoices = [
  { label: "Half Up", value: "HalfUp" },
  { label: "Always Up", value: "Up" },
  { label: "Always Down", value: "Down" },
] satisfies ReadonlyArray<GenericSelectOption<FuelSurchargeRateRounding>>;

export const fuelSurchargeFallbackChoices = [
  { label: "Use latest available price", value: "UseLatestAvailable" },
  { label: "Skip surcharge until price arrives", value: "Skip" },
] satisfies ReadonlyArray<GenericSelectOption<FuelSurchargeFallback>>;

export const fuelSurchargeEffectiveDayChoices = [
  { label: "Sunday", value: 0 },
  { label: "Monday", value: 1 },
  { label: "Tuesday", value: 2 },
  { label: "Wednesday (industry standard)", value: 3 },
  { label: "Thursday", value: 4 },
  { label: "Friday", value: 5 },
  { label: "Saturday", value: 6 },
] satisfies ReadonlyArray<GenericSelectOption<number>>;

export const fuelSurchargeProgramStatusChoices = [
  { label: "Active", value: "Active", color: "var(--success)" },
  { label: "Inactive", value: "Inactive", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<FuelSurchargeProgramStatus>>;

export const statusChoices = [
  { label: "Active", value: "Active", color: "var(--success)" },
  { label: "Inactive", value: "Inactive", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<Status>>;

export const carrierStatusChoices = [
  { label: "Active", value: "Active", color: "var(--success)" },
  { label: "Inactive", value: "Inactive", color: "var(--danger)" },
  { label: "Do Not Use", value: "DoNotUse", color: "var(--danger-foreground)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierStatus>>;

export const carrierTypeChoices = [
  { label: "Common", value: "Common", color: "var(--info)" },
  { label: "Contract", value: "Contract", color: "var(--accent-violet)" },
  { label: "Broker", value: "Broker", color: "var(--warning)" },
  { label: "Exempt", value: "Exempt", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierType>>;

export const carrierComplianceStatusChoices = [
  { label: "Pending", value: "Pending", color: "var(--warning)" },
  { label: "Qualified", value: "Qualified", color: "var(--success)" },
  { label: "Disqualified", value: "Disqualified", color: "var(--danger)" },
  { label: "Expired", value: "Expired", color: "var(--warning-foreground)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierComplianceStatus>>;

export const carrierSafetyRatingChoices = [
  { label: "Satisfactory", value: "Satisfactory", color: "var(--success)" },
  { label: "Conditional", value: "Conditional", color: "var(--warning)" },
  { label: "Unsatisfactory", value: "Unsatisfactory", color: "var(--danger)" },
  { label: "Not Rated", value: "NotRated", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierSafetyRating>>;

export const carrierIntelRiskLevelChoices = [
  { label: "Low", value: "Low", color: "var(--success)" },
  { label: "Moderate", value: "Moderate", color: "var(--info)" },
  { label: "Elevated", value: "Elevated", color: "var(--warning)" },
  { label: "High", value: "High", color: "var(--warning-foreground)" },
  { label: "Very High", value: "VeryHigh", color: "var(--danger)" },
  { label: "Unknown", value: "Unknown", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierIntelRiskLevel>>;

export const carrierIntelReviewRequiredChoices = [
  { label: "Review required", value: true, color: "var(--warning)" },
  { label: "No review required", value: false, color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<boolean>>;

export const carrierTaxIdTypeChoices = [
  { label: "EIN", value: "EIN" },
  { label: "SSN", value: "SSN" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierTaxIdType>>;

export const carrierPaymentMethodChoices = [
  { label: "Check", value: "Check", color: "var(--info)" },
  { label: "ACH (Manual)", value: "ACHManual", color: "var(--accent-teal)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierPaymentMethod>>;

export const carrierRateMethodChoices = [
  { label: "Flat", value: "Flat", color: "var(--info)" },
  { label: "Per Mile", value: "PerMile", color: "var(--accent-teal)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierRateMethod>>;

export const tenderChannelChoices = [
  { label: "Email", value: "Email", color: "var(--info)" },
  { label: "EDI", value: "EDI", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<TenderChannel>>;

export const spotTenderModeChoices = [
  { label: "Broadcast to all at once", value: "SpotBroadcast" },
  { label: "Offer one at a time", value: "SpotSequential" },
] satisfies ReadonlyArray<GenericSelectOption<SpotTenderMode>>;

/**
 * Friendly offer-expiry presets. The server accepts any value between 5
 * minutes and 7 days; a guide loaded with a non-preset value gets an ad-hoc
 * option appended by the form so the stored value survives an edit.
 */
export const offerTtlChoices = [
  { label: "15 minutes", value: 900 },
  { label: "30 minutes", value: 1800 },
  { label: "1 hour", value: 3600 },
  { label: "2 hours", value: 7200 },
  { label: "4 hours", value: 14400 },
  { label: "8 hours", value: 28800 },
  { label: "12 hours", value: 43200 },
  { label: "24 hours", value: 86400 },
  { label: "48 hours", value: 172800 },
  { label: "3 days", value: 259200 },
  { label: "7 days", value: 604800 },
] satisfies ReadonlyArray<GenericSelectOption<number>>;

export const usStateAbbreviationChoices = [
  { label: "Alabama (AL)", value: "AL" },
  { label: "Alaska (AK)", value: "AK" },
  { label: "Arizona (AZ)", value: "AZ" },
  { label: "Arkansas (AR)", value: "AR" },
  { label: "California (CA)", value: "CA" },
  { label: "Colorado (CO)", value: "CO" },
  { label: "Connecticut (CT)", value: "CT" },
  { label: "Delaware (DE)", value: "DE" },
  { label: "District of Columbia (DC)", value: "DC" },
  { label: "Florida (FL)", value: "FL" },
  { label: "Georgia (GA)", value: "GA" },
  { label: "Hawaii (HI)", value: "HI" },
  { label: "Idaho (ID)", value: "ID" },
  { label: "Illinois (IL)", value: "IL" },
  { label: "Indiana (IN)", value: "IN" },
  { label: "Iowa (IA)", value: "IA" },
  { label: "Kansas (KS)", value: "KS" },
  { label: "Kentucky (KY)", value: "KY" },
  { label: "Louisiana (LA)", value: "LA" },
  { label: "Maine (ME)", value: "ME" },
  { label: "Maryland (MD)", value: "MD" },
  { label: "Massachusetts (MA)", value: "MA" },
  { label: "Michigan (MI)", value: "MI" },
  { label: "Minnesota (MN)", value: "MN" },
  { label: "Mississippi (MS)", value: "MS" },
  { label: "Missouri (MO)", value: "MO" },
  { label: "Montana (MT)", value: "MT" },
  { label: "Nebraska (NE)", value: "NE" },
  { label: "Nevada (NV)", value: "NV" },
  { label: "New Hampshire (NH)", value: "NH" },
  { label: "New Jersey (NJ)", value: "NJ" },
  { label: "New Mexico (NM)", value: "NM" },
  { label: "New York (NY)", value: "NY" },
  { label: "North Carolina (NC)", value: "NC" },
  { label: "North Dakota (ND)", value: "ND" },
  { label: "Ohio (OH)", value: "OH" },
  { label: "Oklahoma (OK)", value: "OK" },
  { label: "Oregon (OR)", value: "OR" },
  { label: "Pennsylvania (PA)", value: "PA" },
  { label: "Rhode Island (RI)", value: "RI" },
  { label: "South Carolina (SC)", value: "SC" },
  { label: "South Dakota (SD)", value: "SD" },
  { label: "Tennessee (TN)", value: "TN" },
  { label: "Texas (TX)", value: "TX" },
  { label: "Utah (UT)", value: "UT" },
  { label: "Vermont (VT)", value: "VT" },
  { label: "Virginia (VA)", value: "VA" },
  { label: "Washington (WA)", value: "WA" },
  { label: "West Virginia (WV)", value: "WV" },
  { label: "Wisconsin (WI)", value: "WI" },
  { label: "Wyoming (WY)", value: "WY" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const carrierAssignmentStatusChoices = [
  { label: "Pending", value: "Pending", color: "var(--warning)" },
  { label: "Confirmed", value: "Confirmed", color: "var(--success)" },
  { label: "Canceled", value: "Canceled", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierAssignmentStatus>>;

export const carrierInsurancePolicyTypeChoices = [
  { label: "Auto Liability", value: "AutoLiability", color: "var(--info)" },
  { label: "Cargo Liability", value: "CargoLiability", color: "var(--accent-teal)" },
  { label: "General Liability", value: "GeneralLiability", color: "var(--accent-violet)" },
  { label: "Workers Comp", value: "WorkersComp", color: "var(--warning)" },
  { label: "Umbrella", value: "Umbrella", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierInsurancePolicyType>>;

export const carrierSettlementStatusChoices = [
  { label: "Draft", value: "Draft", color: "var(--foreground-subtle)" },
  { label: "Pending Approval", value: "PendingApproval", color: "var(--warning)" },
  { label: "Approved", value: "Approved", color: "var(--info)" },
  { label: "Posted", value: "Posted", color: "var(--accent-violet)" },
  { label: "Paid", value: "Paid", color: "var(--success)" },
  { label: "Voided", value: "Voided", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierSettlementStatus>>;

export const carrierCostEventTypeChoices = [
  { label: "Linehaul Cost", value: "LinehaulCost", color: "var(--info)" },
  { label: "Fuel Surcharge", value: "FuelSurcharge", color: "var(--warning)" },
  { label: "Accessorial", value: "Accessorial", color: "var(--accent-teal)" },
  { label: "Adjustment", value: "Adjustment", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierCostEventType>>;

export const carrierSettlementBatchStatusChoices = [
  { label: "Open", value: "Open", color: "var(--info)" },
  { label: "Completed", value: "Completed", color: "var(--success)" },
  { label: "Canceled", value: "Canceled", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierSettlementBatchStatus>>;

export const carrierLedgerEntryTypeChoices = [
  { label: "Bill", value: "Bill", color: "var(--info)" },
  { label: "Payment", value: "Payment", color: "var(--success)" },
  { label: "Adjustment", value: "Adjustment", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierLedgerEntryType>>;

export const carrierCostEventStatusChoices = [
  { label: "Pending", value: "Pending", color: "var(--info)" },
  { label: "Attached", value: "Attached", color: "var(--warning)" },
  { label: "Settled", value: "Settled", color: "var(--success)" },
  { label: "Voided", value: "Voided", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierCostEventStatus>>;

export const carrierInvoiceMatchStatusChoices = [
  { label: "Suggested", value: "Suggested", color: "var(--foreground-subtle)" },
  { label: "Matched", value: "Matched", color: "var(--info)" },
  { label: "Variance", value: "Variance", color: "var(--warning)" },
  { label: "Resolved", value: "Resolved", color: "var(--success)" },
  { label: "Rejected", value: "Rejected", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<CarrierInvoiceMatchStatus>>;

export const rateConfirmationStatusChoices = [
  { label: "Generated", value: "Generated", color: "var(--foreground-subtle)" },
  { label: "Sent", value: "Sent", color: "var(--info)" },
  { label: "Confirmed", value: "Confirmed", color: "var(--success)" },
  { label: "Voided", value: "Voided", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<RateConfirmationStatus>>;

export const orderStatusChoices = [
  { label: "Draft", value: "Draft", color: "var(--foreground-subtle)" },
  { label: "Confirmed", value: "Confirmed", color: "var(--accent-violet)" },
  { label: "In Progress", value: "InProgress", color: "var(--info)" },
  { label: "Completed", value: "Completed", color: "var(--success)" },
  { label: "Billed", value: "Billed", color: "var(--accent-teal)" },
  { label: "Closed", value: "Closed", color: "var(--foreground-muted)" },
  { label: "Canceled", value: "Canceled", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<OrderStatus>>;

export const distanceProfileProviderChoices = [
  {
    label: "PC*Miler",
    value: "PCMiler",
    color: "var(--info)",
    description: "Use Trimble PC*Miler Route Reports for mileage calculations.",
  },
] satisfies ReadonlyArray<GenericSelectOption<DistanceProfile["provider"]>>;

export const distanceProfileStatusChoices = [
  {
    label: "Active",
    value: "Active",
    color: "var(--success)",
    description: "Available for default selection and shipment mileage calculations.",
  },
  {
    label: "Inactive",
    value: "Inactive",
    color: "var(--danger)",
    description: "Hidden from default selection while preserving historical audit data.",
  },
] satisfies ReadonlyArray<GenericSelectOption<DistanceProfile["status"]>>;

export const distanceProfileRegionChoices = [
  {
    label: "North America",
    value: "NA",
    description: "Default PC*Miler region for North American routing.",
  },
] satisfies ReadonlyArray<GenericSelectOption<DistanceProfile["region"]>>;

export const distanceProfileRoutingTypeChoices = [
  {
    label: "Practical",
    value: "Practical",
    color: "var(--accent-teal)",
    description: "Preferred freight policy for balanced mileage and road suitability.",
  },
  {
    label: "Shortest",
    value: "Shortest",
    color: "var(--warning)",
    description: "Minimizes distance and may use less operationally preferred roads.",
  },
  {
    label: "Fastest",
    value: "Fastest",
    color: "var(--info)",
    description: "Prioritizes travel time where PC*Miler data supports it.",
  },
] satisfies ReadonlyArray<GenericSelectOption<DistanceProfile["routingType"]>>;

export const distanceProfileDistanceUnitChoices = [
  { label: "Miles", value: "Miles", description: "Store and return mileage in miles." },
  {
    label: "Kilometers",
    value: "Kilometers",
    description: "Store and return mileage in kilometers.",
  },
] satisfies ReadonlyArray<GenericSelectOption<DistanceProfile["distanceUnits"]>>;

export const distanceProfileLocationGranularityChoices = [
  {
    label: "Postal Code",
    value: "PostalCode",
    description: "Use postal codes from stop locations for stable mileage lookups.",
  },
  {
    label: "City / State",
    value: "CityState",
    description: "Use city and state when postal codes are unavailable or inconsistent.",
  },
  {
    label: "Street Address",
    value: "StreetAddress",
    description: "Use street-level addresses for more precise routing.",
  },
  {
    label: "Coordinates",
    value: "Coordinates",
    description: "Use latitude and longitude from geocoded locations.",
  },
  {
    label: "Trimble Place ID",
    value: "TrimblePlaceId",
    description: "Use Trimble place identifiers stored on locations.",
  },
] satisfies ReadonlyArray<GenericSelectOption<DistanceProfile["locationGranularity"]>>;

export const locationGeofenceTypeChoices = [
  { label: "Auto", value: "auto" },
  { label: "Circle", value: "circle" },
  { label: "Rectangle", value: "rectangle" },
  { label: "Draw", value: "draw" },
] satisfies ReadonlyArray<GenericSelectOption<LocationGeofenceType>>;

export const shipmentStatusChoices = [
  { label: "New", value: "New", color: "var(--info)" },
  { label: "Partially Assigned", value: "PartiallyAssigned", color: "var(--accent-violet)" },
  { label: "Assigned", value: "Assigned", color: "var(--success)" },
  { label: "In Transit", value: "InTransit", color: "var(--accent-teal)" },
  { label: "Delayed", value: "Delayed", color: "var(--danger)" },
  { label: "Partially Completed", value: "PartiallyCompleted", color: "var(--warning)" },
  {
    value: "Completed",
    label: "Completed",
    color: "var(--success)",
  },
  { label: "Ready To Invoice", value: "ReadyToInvoice", color: "var(--accent-teal-on-subtle)" },
  { label: "Invoiced", value: "Invoiced", color: "var(--success-foreground)" },
  { label: "Canceled", value: "Canceled", color: "var(--danger-foreground)" },
] satisfies ReadonlyArray<GenericSelectOption<ShipmentStatus>>;

export const shipmentTenderStatusChoices = [
  { label: "Tendered", value: "Tendered", color: "var(--info)" },
  { label: "Accepted", value: "Accepted", color: "var(--success)" },
  { label: "Rejected", value: "Rejected", color: "var(--danger)" },
  { label: "Expired", value: "Expired", color: "var(--warning)" },
  { label: "Canceled", value: "Canceled", color: "var(--danger-foreground)" },
] satisfies ReadonlyArray<GenericSelectOption<ShipmentTenderStatus>>;

export const freightTermsChoices = [
  {
    value: "Prepaid",
    label: "Prepaid",
    color: "var(--success)",
    description: "The shipper pays the carrier",
  },
  {
    value: "Collect",
    label: "Collect",
    color: "var(--accent-teal)",
    description: "The consignee pays the carrier on delivery",
  },
  {
    value: "ThirdParty",
    label: "Third Party",
    color: "var(--accent-violet)",
    description: "A party other than shipper or consignee pays",
  },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const allocationMethodChoices = [
  { value: "Percent", label: "Percent", color: "var(--info)" },
  { value: "Amount", label: "Amount", color: "var(--success)" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const billTypeChoices = [
  { label: "Invoice", value: "Invoice", color: "var(--info)" },
  { label: "Credit Memo", value: "CreditMemo", color: "var(--warning)" },
  { label: "Debit Memo", value: "DebitMemo", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const billingQueueStatusChoices = [
  { label: "Ready for Review", value: "ReadyForReview", color: "var(--info)" },
  { label: "In Review", value: "InReview", color: "var(--accent-teal)" },
  { label: "Approved", value: "Approved", color: "var(--success)" },
  { label: "On Hold", value: "OnHold", color: "var(--warning)" },
  { label: "Sent Back to Ops", value: "SentBackToOps", color: "var(--warning-foreground)" },
  { label: "Exception", value: "Exception", color: "var(--danger)" },
  { label: "Canceled", value: "Canceled", color: "var(--danger-foreground)" },
] satisfies ReadonlyArray<GenericSelectOption<BillingQueueStatus>>;

// The queue hides posted items behind its own toggle, so its status filter omits
// Posted; a shipment's billing state includes it.
export const shipmentBillingStatusChoices = [
  ...billingQueueStatusChoices,
  { label: "Posted", value: "Posted", color: "var(--accent-teal)" },
] satisfies ReadonlyArray<GenericSelectOption<BillingQueueStatus>>;

export const moveStatusChoices = [
  { label: "New", value: "New", color: "var(--info)" },
  { label: "Assigned", value: "Assigned", color: "var(--success)" },
  { label: "In Transit", value: "InTransit", color: "var(--accent-teal)" },
  { label: "Completed", value: "Completed", color: "var(--success-foreground)" },
  { label: "Canceled", value: "Canceled", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<MoveStatus>>;

export const stopStatusChoices = [
  { label: "New", value: "New", color: "var(--info)" },
  { label: "In Transit", value: "InTransit", color: "var(--accent-teal)" },
  { label: "Completed", value: "Completed", color: "var(--success)" },
  { label: "Canceled", value: "Canceled", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<StopStatus>>;

export const stopTypeChoices = [
  { label: "Pickup", value: "Pickup" },
  { label: "Delivery", value: "Delivery" },
  { label: "Split Delivery", value: "SplitDelivery" },
  { label: "Split Pickup", value: "SplitPickup" },
] satisfies ReadonlyArray<GenericSelectOption<StopType>>;

export const stopScheduleTypeChoices = [
  { label: "Open", value: "Open" },
  { label: "Appointment", value: "Appointment" },
] satisfies ReadonlyArray<GenericSelectOption<StopScheduleType>>;

export const equipmentClassChoices = [
  { value: "Tractor", label: "Tractor", color: "var(--success)" },
  { value: "Trailer", label: "Trailer", color: "var(--accent-violet)" },
  { value: "Container", label: "Container", color: "var(--danger)" },
  { value: "Other", label: "Other", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<EquipmentClass>>;

export const fieldSensitivityChoices = [
  { value: "public", label: "Public", color: "var(--success)", variant: "success" },
  { value: "internal", label: "Internal", color: "var(--info)", variant: "info" },
  {
    value: "restricted",
    label: "Restricted",
    color: "var(--warning)",
    variant: "warning",
  },
  {
    value: "confidential",
    label: "Confidential",
    color: "var(--danger)",
    variant: "danger",
  },
] satisfies ReadonlyArray<GenericSelectOption<FieldSensitivity>>;

export const coreResponsibilityChoices = [
  { value: "Billing", label: "Billing", color: "var(--info)" },
  { value: "Operations", label: "Operations", color: "var(--success)" },
  { value: "Finance", label: "Finance", color: "var(--accent-violet)" },
  { value: "Leadership", label: "Leadership", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<CoreResponsibility>>;

export const dataScopeChoices = [
  { value: "own", label: "Own Data Only" },
  { value: "organization", label: "Organization" },
  { value: "all", label: "All Data" },
] satisfies ReadonlyArray<GenericSelectOption<DataScope>>;

export const operationChoices: SelectOption[] = [
  { value: "read", label: "Read" },
  { value: "create", label: "Create" },
  { value: "update", label: "Update" },
  { value: "export", label: "Export" },
  { value: "import", label: "Import" },
  { value: "approve", label: "Approve" },
  { value: "reject", label: "Reject" },
  { value: "assign", label: "Assign" },
  { value: "unassign", label: "Unassign" },
  { value: "archive", label: "Archive" },
  { value: "restore", label: "Restore" },
  { value: "submit", label: "Submit" },
  { value: "cancel", label: "Cancel" },
  { value: "duplicate", label: "Duplicate" },
] satisfies ReadonlyArray<GenericSelectOption<Operation>>;

// Derived from i18n/locales.json via the generated module, so shipping a new language does
// not also require editing this list. Each option is labelled in its own language, which is
// how a user who cannot read the current one finds theirs.
export const localeChoices = LOCALES.map((locale) => ({
  value: locale,
  label: LOCALE_NAMES[locale],
})) satisfies ReadonlyArray<GenericSelectOption<Locale>>;

export const timeFormatChoices = [
  { label: "12 Hour", value: "12-hour" },
  { label: "24 Hour", value: "24-hour" },
] satisfies ReadonlyArray<GenericSelectOption<TimeFormatType>>;

export const timezoneChoices = [
  { value: "America/New_York", label: "Eastern Time (US)" },
  { value: "America/Chicago", label: "Central Time (US)" },
  { value: "America/Denver", label: "Mountain Time (US)" },
  { value: "America/Los_Angeles", label: "Pacific Time (US)" },
  { value: "America/Phoenix", label: "Arizona Time (US)" },
  { value: "America/Anchorage", label: "Alaska Time" },
  { value: "Pacific/Honolulu", label: "Hawaii Time" },
  { value: "America/Toronto", label: "Eastern Time (Canada)" },
  { value: "America/Vancouver", label: "Pacific Time (Canada)" },
  { value: "Europe/London", label: "London (GMT)" },
  { value: "Europe/Paris", label: "Paris (CET)" },
  { value: "Europe/Berlin", label: "Berlin (CET)" },
  { value: "Asia/Tokyo", label: "Tokyo (JST)" },
  { value: "Asia/Shanghai", label: "Shanghai (CST)" },
  { value: "Australia/Sydney", label: "Sydney (AEST)" },
  { value: "UTC", label: "UTC" },
] satisfies ReadonlyArray<SelectOption>;

export const timezoneGroupedChoices: SelectOptionGroup[] = [
  {
    label: "Americas",
    options: [
      { value: "America/New_York", label: "Eastern Time (US)", description: "UTC-5" },
      { value: "America/Chicago", label: "Central Time (US)", description: "UTC-6" },
      { value: "America/Denver", label: "Mountain Time (US)", description: "UTC-7" },
      { value: "America/Los_Angeles", label: "Pacific Time (US)", description: "UTC-8" },
      { value: "America/Phoenix", label: "Arizona Time (US)", description: "UTC-7" },
      { value: "America/Anchorage", label: "Alaska Time", description: "UTC-9" },
      { value: "Pacific/Honolulu", label: "Hawaii Time", description: "UTC-10" },
      { value: "America/Toronto", label: "Eastern Time (Canada)", description: "UTC-5" },
      { value: "America/Vancouver", label: "Pacific Time (Canada)", description: "UTC-8" },
    ],
  },
  {
    label: "Europe",
    options: [
      { value: "Europe/London", label: "London (GMT)", description: "UTC+0" },
      { value: "Europe/Paris", label: "Paris (CET)", description: "UTC+1" },
      { value: "Europe/Berlin", label: "Berlin (CET)", description: "UTC+1" },
    ],
  },
  {
    label: "Asia & Pacific",
    options: [
      { value: "Asia/Tokyo", label: "Tokyo (JST)", description: "UTC+9" },
      { value: "Asia/Shanghai", label: "Shanghai (CST)", description: "UTC+8" },
      { value: "Australia/Sydney", label: "Sydney (AEST)", description: "UTC+10" },
    ],
  },
  {
    label: "Other",
    options: [{ value: "UTC", label: "UTC", description: "UTC+0" }],
  },
];

export const equipmentStatusChoices = [
  { value: "Available", label: "Available", color: "var(--success)" },
  {
    value: "OutOfService",
    label: "Out of Service",
    color: "var(--danger)",
  },
  {
    value: "AtMaintenance",
    label: "At Maintenance",
    color: "var(--accent-violet)",
  },
  { value: "Sold", label: "Sold", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<EquipmentStatus>>;

export const fieldTypeChoices = [
  { value: "text", label: "Text", color: "var(--info)" },
  { value: "number", label: "Number", color: "var(--success)" },
  { value: "date", label: "Date", color: "var(--accent-violet)" },
  { value: "boolean", label: "Boolean", color: "var(--warning)" },
  { value: "select", label: "Select", color: "var(--danger)" },
  { value: "multiSelect", label: "Multi-Select", color: "var(--accent-rose)" },
] satisfies ReadonlyArray<GenericSelectOption<FieldType>>;

export const accessorialChargeMethodChoices = [
  {
    value: "Flat",
    label: "Flat",
    color: "var(--success)",
    description: "Fixed amount regardless of shipment details",
  },
  {
    value: "PerUnit",
    label: "Per Unit",
    color: "var(--accent-violet)",
    description: "Rate multiplied by units",
  },
  {
    value: "Percentage",
    label: "Percentage",
    color: "var(--warning)",
    description: "Percentage of linehaul, freight charges, or declared value",
  },
] satisfies ReadonlyArray<GenericSelectOption<AccessorialChargeMethod>>;

export const rateUnitChoices = [
  { value: "Mile", label: "Mile", color: "var(--success)" },
  { value: "Hour", label: "Hour", color: "var(--accent-violet)" },
  { value: "Day", label: "Day", color: "var(--warning)" },
  { value: "Stop", label: "Stop", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<RateUnit>>;

export const workerTypeChoices = [
  { value: "Employee", label: "Employee", color: "var(--success)" },
  { value: "Contractor", label: "Contractor", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<WorkerType>>;

export const genderChoices = [
  { value: "Male", label: "Male" },
  { value: "Female", label: "Female" },
] satisfies ReadonlyArray<GenericSelectOption<Gender>>;

export const driverTypeChoices = [
  { value: "Local", label: "Local", color: "var(--info)" },
  { value: "Regional", label: "Regional", color: "var(--success)" },
  { value: "OTR", label: "OTR (Over the Road)", color: "var(--warning)" },
  { value: "Team", label: "Team", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<DriverType>>;

export const cdlClassChoices = [
  { value: "A", label: "Class A", color: "var(--success)" },
  { value: "B", label: "Class B", color: "var(--info)" },
  { value: "C", label: "Class C", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<CDLClass>>;

export const endorsementTypeChoices = [
  { value: "O", label: "None (O)" },
  { value: "N", label: "Tanker (N)" },
  { value: "H", label: "Hazmat (H)" },
  { value: "X", label: "Tanker + Hazmat (X)" },
  { value: "P", label: "Passenger (P)" },
  { value: "T", label: "Double/Triple (T)" },
] satisfies ReadonlyArray<GenericSelectOption<EndorsementType>>;

export const complianceStatusChoices = [
  { value: "Compliant", label: "Compliant", color: "var(--success)" },
  { value: "NonCompliant", label: "Non-Compliant", color: "var(--danger)" },
  { value: "Pending", label: "Pending", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<ComplianceStatus>>;

export const trainingHealthChoices = [
  { value: "Current", label: "Current", color: "var(--success)" },
  { value: "Scheduled", label: "Scheduled", color: "var(--info)" },
  { value: "DueSoon", label: "Due soon", color: "var(--warning)" },
  { value: "ExpiringSoon", label: "Expiring soon", color: "var(--warning)" },
  { value: "Overdue", label: "Overdue", color: "var(--danger)" },
  { value: "Expired", label: "Expired", color: "var(--danger)" },
  { value: "Failed", label: "Failed", color: "var(--danger)" },
  { value: "Missing", label: "Never assigned", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<WorkerTrainingHealth>>;

export const safetyRatingChoices = [
  { value: "Excellent", label: "Excellent", color: "var(--success)" },
  { value: "Good", label: "Good", color: "var(--info)" },
  { value: "Watch", label: "Watch", color: "var(--warning)" },
  { value: "AtRisk", label: "At risk", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<SafetyRating>>;

export const drugAlcoholStatusChoices = [
  { value: "Clear", label: "Clear", color: "var(--success)" },
  { value: "Pending", label: "Awaiting result", color: "var(--warning)" },
  { value: "Prohibited", label: "Prohibited", color: "var(--danger)" },
  { value: "Unknown", label: "Not on file", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<DrugAlcoholStatus>>;

export const ptoStatusChoices = [
  { value: "Requested", label: "Requested", color: "var(--info)" },
  { value: "Approved", label: "Approved", color: "var(--success)" },
  { value: "Rejected", label: "Rejected", color: "var(--danger)" },
  { value: "Cancelled", label: "Cancelled", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<PTOStatus>>;

export const ptoTypeChoices = [
  { value: "Personal", label: "Personal" },
  { value: "Vacation", label: "Vacation" },
  { value: "Sick", label: "Sick" },
  { value: "Holiday", label: "Holiday" },
  { value: "Bereavement", label: "Bereavement" },
  { value: "Maternity", label: "Maternity" },
  { value: "Paternity", label: "Paternity" },
] satisfies ReadonlyArray<GenericSelectOption<PTOType>>;

export const hazardousClassChoices = [
  { value: "HazardClass1", label: "Division 1: Explosives" },
  { value: "HazardClass1And1", label: "Division 1.1: Mass Explosion Hazard" },
  { value: "HazardClass1And2", label: "Division 1.2: Projection Hazard" },
  { value: "HazardClass1And3", label: "Division 1.3: Fire Hazard" },
  { value: "HazardClass1And4", label: "Division 1.4: Minor Hazard" },
  { value: "HazardClass1And5", label: "Division 1.5: Insensitive Explosives" },
  { value: "HazardClass1And6", label: "Division 1.6: Extremely Insensitive" },
  { value: "HazardClass2And1", label: "Division 2.1: Flammable Gas" },
  { value: "HazardClass2And2", label: "Division 2.2: Non-Flammable Gas" },
  { value: "HazardClass2And3", label: "Division 2.3: Toxic Gas" },
  { value: "HazardClass3", label: "Class 3: Flammable Liquids" },
  { value: "HazardClass4And1", label: "Division 4.1: Flammable Solids" },
  { value: "HazardClass4And2", label: "Division 4.2: Spontaneous Combustion" },
  { value: "HazardClass4And3", label: "Division 4.3: Dangerous When Wet" },
  { value: "HazardClass5And1", label: "Division 5.1: Oxidizers" },
  { value: "HazardClass5And2", label: "Division 5.2: Organic Peroxides" },
  { value: "HazardClass6And1", label: "Division 6.1: Toxic Substances" },
  { value: "HazardClass6And2", label: "Division 6.2: Infectious Substances" },
  { value: "HazardClass7", label: "Class 7: Radioactive Materials" },
  { value: "HazardClass8", label: "Class 8: Corrosives" },
  { value: "HazardClass9", label: "Class 9: Miscellaneous" },
] satisfies ReadonlyArray<GenericSelectOption<HazardousClass>>;

export const freightClassChoices = [
  { value: "Class50", label: "Class 50" },
  { value: "Class55", label: "Class 55" },
  { value: "Class60", label: "Class 60" },
  { value: "Class65", label: "Class 65" },
  { value: "Class70", label: "Class 70" },
  { value: "Class77_5", label: "Class 77.5" },
  { value: "Class85", label: "Class 85" },
  { value: "Class92_5", label: "Class 92.5" },
  { value: "Class100", label: "Class 100" },
  { value: "Class110", label: "Class 110" },
  { value: "Class125", label: "Class 125" },
  { value: "Class150", label: "Class 150" },
  { value: "Class175", label: "Class 175" },
  { value: "Class200", label: "Class 200" },
  { value: "Class250", label: "Class 250" },
  { value: "Class300", label: "Class 300" },
  { value: "Class400", label: "Class 400" },
  { value: "Class500", label: "Class 500" },
] satisfies ReadonlyArray<GenericSelectOption<FreightClass>>;

export const accountCategoryChoices = [
  { value: "Asset", label: "Asset", color: "var(--info)" },
  { value: "Liability", label: "Liability", color: "var(--danger)" },
  { value: "Equity", label: "Equity", color: "var(--accent-violet)" },
  { value: "Revenue", label: "Revenue", color: "var(--success)" },
  { value: "CostOfRevenue", label: "Cost of Revenue", color: "var(--warning)" },
  { value: "Expense", label: "Expense", color: "var(--danger-foreground)" },
] satisfies ReadonlyArray<GenericSelectOption<AccountCategory>>;

export const accountingBasisChoices = [
  {
    value: "Accrual",
    label: "Accrual",
    description: "Recognize revenue and expense from non-cash posting events",
  },
  {
    value: "Cash",
    label: "Cash",
    description: "Recognize revenue and expense only from cash settlement events",
  },
] satisfies ReadonlyArray<GenericSelectOption<AccountingBasis>>;

export const journalPostingModeChoices = [
  { value: "Manual", label: "Manual" },
  { value: "Automatic", label: "Automatic" },
] satisfies ReadonlyArray<GenericSelectOption<JournalPostingMode>>;

export const journalSourceEventChoices = [
  { value: "InvoicePosted", label: "Invoice Posted" },
  { value: "CreditMemoPosted", label: "Credit Memo Posted" },
  { value: "DebitMemoPosted", label: "Debit Memo Posted" },
  { value: "CustomerPaymentPosted", label: "Customer Payment Posted" },
  { value: "VendorBillPosted", label: "Vendor Bill Posted" },
  { value: "VendorPaymentPosted", label: "Vendor Payment Posted" },
  { value: "CarrierSettlementPosted", label: "Carrier Settlement Posted" },
  { value: "CarrierSettlementVoided", label: "Carrier Settlement Voided" },
  { value: "CarrierSettlementPaid", label: "Carrier Settlement Paid" },
] satisfies ReadonlyArray<GenericSelectOption<JournalSourceEvent>>;

export const manualJournalEntryPolicyChoices = [
  { value: "AllowAll", label: "Allow All" },
  { value: "AdjustmentOnly", label: "Adjustment Only" },
  { value: "Disallow", label: "Disallow" },
] satisfies ReadonlyArray<GenericSelectOption<ManualJournalEntryPolicy>>;

export const journalReversalPolicyChoices = [
  { value: "Disallow", label: "Disallow" },
  { value: "NextOpenPeriod", label: "Next Open Period" },
] satisfies ReadonlyArray<GenericSelectOption<JournalReversalPolicy>>;

export const revenueRecognitionPolicyChoices = [
  { value: "OnInvoicePost", label: "On Invoice Post" },
  { value: "OnCashReceipt", label: "On Cash Receipt" },
] satisfies ReadonlyArray<GenericSelectOption<RevenueRecognitionPolicy>>;

export const expenseRecognitionPolicyChoices = [
  { value: "OnVendorBillPost", label: "On Vendor Bill Post" },
  { value: "OnCashDisbursement", label: "On Cash Disbursement" },
] satisfies ReadonlyArray<GenericSelectOption<ExpenseRecognitionPolicy>>;

export const periodCloseModeChoices = [
  { value: "ManualOnly", label: "Manual Only" },
  { value: "SystemScheduled", label: "System Scheduled" },
] satisfies ReadonlyArray<GenericSelectOption<PeriodCloseMode>>;

export const lockedPeriodPostingPolicyChoices = [
  { value: "BlockSubledgerAllowManualJe", label: "Block Subledger, Allow Manual JE" },
] satisfies ReadonlyArray<GenericSelectOption<LockedPeriodPostingPolicy>>;

export const closedPeriodPostingPolicyChoices = [
  { value: "RequireReopen", label: "Require Reopen" },
  { value: "PostToNextOpen", label: "Post To Next Open" },
] satisfies ReadonlyArray<GenericSelectOption<ClosedPeriodPostingPolicy>>;

export const reconciliationModeChoices = [
  { value: "Disabled", label: "Disabled", color: "var(--foreground-subtle)" },
  { value: "WarnOnly", label: "Warn Only", color: "var(--warning)" },
  { value: "BlockPosting", label: "Block Posting", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<ReconciliationMode>>;

export const currencyModeChoices = [
  { value: "SingleCurrency", label: "Single Currency" },
  { value: "MultiCurrency", label: "Multi Currency" },
] satisfies ReadonlyArray<GenericSelectOption<CurrencyMode>>;

export const exchangeRateDatePolicyChoices = [
  { value: "DocumentDate", label: "Document Date" },
  { value: "AccountingDate", label: "Accounting Date" },
] satisfies ReadonlyArray<GenericSelectOption<ExchangeRateDatePolicy>>;

export const exchangeRateOverridePolicyChoices = [
  { value: "Allow", label: "Allow" },
  { value: "RequireApproval", label: "Require Approval" },
  { value: "Disallow", label: "Disallow" },
] satisfies ReadonlyArray<GenericSelectOption<ExchangeRateOverridePolicy>>;

export const packingGroupChoices = [
  { value: "I", label: "I - High Danger", color: "var(--danger)" },
  { value: "II", label: "II - Medium Danger", color: "var(--warning)" },
  { value: "III", label: "III - Low Danger", color: "var(--success)" },
] satisfies ReadonlyArray<GenericSelectOption<PackingGroup>>;

export const segregationTypeChoices = [
  { value: "Prohibited", label: "Prohibited", color: "var(--danger)" },
  { value: "Separated", label: "Separated", color: "var(--success)" },
  { value: "Distance", label: "Distance", color: "var(--accent-violet)" },
  { value: "Barrier", label: "Barrier", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<SegregationType>>;

export const segregationDistanceUnitChoices = [
  { value: "FT", label: "Feet", color: "var(--success)" },
  { value: "M", label: "Meters", color: "var(--accent-violet)" },
  { value: "IN", label: "Inches", color: "var(--warning)" },
  { value: "CM", label: "Centimeters", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<SegregationDistanceUnit>>;

export const fiscalYearStatusChoices = [
  { value: "Draft", label: "Draft", color: "var(--accent-violet)" },
  { value: "Open", label: "Open", color: "var(--success)" },
  { value: "Closed", label: "Closed", color: "var(--danger)" },
  { value: "PermanentlyClosed", label: "Permanently Closed", color: "var(--danger-foreground)" },
] satisfies ReadonlyArray<GenericSelectOption<FiscalYearStatus>>;

export const fiscalPeriodStatusChoices = [
  { value: "Inactive", label: "Inactive", color: "var(--foreground-subtle)" },
  { value: "Open", label: "Open", color: "var(--success)" },
  { value: "Locked", label: "Locked", color: "var(--warning)" },
  { value: "Closed", label: "Closed", color: "var(--danger)" },
  { value: "PermanentlyClosed", label: "Permanently Closed", color: "var(--danger-foreground)" },
] satisfies ReadonlyArray<GenericSelectOption<FiscalPeriodStatus>>;

export const periodTypeChoices = [
  { value: "Month", label: "Month", color: "var(--info)" },
  { value: "Quarter", label: "Quarter", color: "var(--accent-violet)" },
  { value: "Week", label: "Week", color: "var(--accent-teal)" },
  { value: "Adjusting", label: "Adjusting", color: "var(--accent-violet-on-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<PeriodType>>;

export const documentClassificationChoices = [
  { value: "Public", label: "Public", color: "var(--success)" },
  { value: "Private", label: "Private", color: "var(--accent-violet)" },
  { value: "Sensitive", label: "Sensitive", color: "var(--danger)" },
  { value: "Regulatory", label: "Regulatory", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<DocumentClassification>>;

export const documentCategoryChoices = [
  { value: "Shipment", label: "Shipment", color: "var(--success)" },
  { value: "Worker", label: "Worker", color: "var(--accent-violet)" },
  { value: "Regulatory", label: "Regulatory", color: "var(--warning)" },
  { value: "Profile", label: "Profile", color: "var(--info)" },
  { value: "Branding", label: "Branding", color: "var(--success-foreground)" },
  { value: "Invoice", label: "Invoice", color: "var(--info-foreground)" },
  { value: "Contract", label: "Contract", color: "var(--info)" },
  { value: "Other", label: "Other", color: "var(--accent-rose)" },
] satisfies ReadonlyArray<GenericSelectOption<DocumentCategory>>;

export const locationCategoryTypeChoices = [
  { value: "Terminal", label: "Terminal", color: "var(--info)" },
  { value: "Warehouse", label: "Warehouse", color: "var(--success)" },
  {
    value: "DistributionCenter",
    label: "Distribution Center",
    color: "var(--accent-violet)",
  },
  { value: "TruckStop", label: "Truck Stop", color: "var(--warning)" },
  { value: "RestArea", label: "Rest Area", color: "var(--foreground-subtle)" },
  { value: "CustomerLocation", label: "Customer Location", color: "var(--accent-rose)" },
  { value: "Port", label: "Port", color: "var(--info-foreground)" },
  { value: "RailYard", label: "Rail Yard", color: "var(--accent-violet-on-subtle)" },
  {
    value: "MaintenanceFacility",
    label: "Maintenance Facility",
    color: "var(--danger)",
  },
] satisfies ReadonlyArray<GenericSelectOption<LocationCategoryType>>;

export const facilityTypeChoices = [
  { value: "CrossDock", label: "Cross Dock" },
  { value: "StorageWarehouse", label: "Storage Warehouse" },
  { value: "ColdStorage", label: "Cold Storage" },
  { value: "HazmatFacility", label: "Hazmat Facility" },
  { value: "IntermodalFacility", label: "Intermodal Facility" },
] satisfies ReadonlyArray<GenericSelectOption<FacilityType>>;

export const holdTypeChoices = [
  { value: "OperationalHold", label: "Operational", color: "var(--info)" },
  { value: "ComplianceHold", label: "Compliance", color: "var(--warning)" },
  { value: "CustomerHold", label: "Customer", color: "var(--accent-violet)" },
  { value: "FinanceHold", label: "Finance", color: "var(--success)" },
] satisfies ReadonlyArray<GenericSelectOption<HoldType>>;

export const holdSeverityChoices = [
  { value: "Informational", label: "Informational", color: "var(--info)" },
  { value: "Advisory", label: "Advisory", color: "var(--warning)" },
  { value: "Blocking", label: "Blocking", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<HoldSeverity>>;

export const serviceFailureStatusChoices = [
  { value: "Open", label: "Open", color: "var(--danger)" },
  { value: "Reviewed", label: "Reviewed", color: "var(--warning)" },
  { value: "Resolved", label: "Resolved", color: "var(--success)" },
  { value: "Voided", label: "Voided", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<ServiceFailureStatus>>;

export const serviceFailureSourceChoices = [
  { value: "Detected", label: "Detected", color: "var(--info)" },
  { value: "Manual", label: "Manual", color: "var(--accent-violet)" },
  { value: "EDI", label: "EDI", color: "var(--info-foreground)" },
  { value: "Integration", label: "Integration", color: "var(--accent-teal)" },
] satisfies ReadonlyArray<GenericSelectOption<ServiceFailureSource>>;

export const serviceFailureTypeChoices = [
  { value: "LatePickup", label: "Late Pickup", color: "var(--warning)" },
  { value: "LateDelivery", label: "Late Delivery", color: "var(--danger)" },
  { value: "MissedPickup", label: "Missed Pickup", color: "var(--warning-foreground)" },
  { value: "MissedDelivery", label: "Missed Delivery", color: "var(--danger-foreground)" },
  { value: "AppointmentMissed", label: "Appointment Missed", color: "var(--accent-violet)" },
  { value: "Other", label: "Other", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<ServiceFailureType>>;

export const serviceFailureReasonCategoryChoices = [
  { value: "Carrier", label: "Carrier", color: "var(--info)" },
  { value: "Customer", label: "Customer", color: "var(--accent-violet)" },
  { value: "Facility", label: "Facility", color: "var(--warning)" },
  { value: "Weather", label: "Weather", color: "var(--info-foreground)" },
  { value: "Equipment", label: "Equipment", color: "var(--danger)" },
  { value: "Documentation", label: "Documentation", color: "var(--foreground-subtle)" },
  { value: "Driver", label: "Driver", color: "var(--accent-indigo)" },
  { value: "Shipper", label: "Shipper", color: "var(--accent-teal)" },
  { value: "Consignee", label: "Consignee", color: "var(--info-subtle-foreground)" },
  { value: "Appointment", label: "Appointment", color: "var(--accent-violet-on-subtle)" },
  { value: "Other", label: "Other", color: "var(--foreground-muted)" },
] satisfies ReadonlyArray<GenericSelectOption<ServiceFailureReasonCategory>>;

export const serviceFailureReasonCodeAppliesToChoices = [
  { value: "Pickup", label: "Pickup" },
  { value: "Delivery", label: "Delivery" },
  { value: "Both", label: "Pickup & Delivery" },
  { value: "All", label: "All Stops" },
] satisfies ReadonlyArray<GenericSelectOption<ServiceFailureReasonCodeAppliesTo>>;

export function findChoice<TValue extends string | boolean | number>(
  choices: ReadonlyArray<GenericSelectOption<TValue>>,
  value: TValue,
) {
  return choices.find((choice) => choice.value === value);
}

export const transferScheduleChoices = [
  { value: "Continuous", label: "Continuous", color: "var(--success)" },
  { value: "Hourly", label: "Hourly", color: "var(--info)" },
  { value: "Daily", label: "Daily", color: "var(--accent-violet)" },
  { value: "Weekly", label: "Weekly", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<TransferSchedule>>;

export const readyToBillAssignmentModeChoices = [
  { value: "ManualOnly", label: "Manual Only" },
  { value: "AutomaticWhenEligible", label: "Automatic When Eligible" },
] satisfies ReadonlyArray<GenericSelectOption<ReadyToBillAssignmentMode>>;

export const billingQueueTransferModeChoices = [
  { value: "ManualOnly", label: "Manual Only" },
  { value: "AutomaticWhenReady", label: "Automatic When Ready" },
] satisfies ReadonlyArray<GenericSelectOption<BillingQueueTransferMode>>;

export const invoiceDraftCreationModeChoices = [
  { value: "ManualOnly", label: "Manual Only" },
  { value: "AutomaticWhenTransferred", label: "Automatic When Transferred" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceDraftCreationMode>>;

export const invoicePostingModeChoices = [
  { value: "ManualReviewRequired", label: "Manual Review Required" },
  {
    value: "AutomaticWhenNoBlockingExceptions",
    label: "Automatic When No Blocking Exceptions",
  },
] satisfies ReadonlyArray<GenericSelectOption<InvoicePostingMode>>;

export const enforcementLevelChoices = [
  { value: "Ignore", label: "Ignore", color: "var(--foreground-subtle)" },
  { value: "Warn", label: "Warn", color: "var(--warning)" },
  { value: "RequireReview", label: "Require Review", color: "var(--info)" },
  { value: "Block", label: "Block", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<EnforcementLevel>>;

export const billingExceptionDispositionChoices = [
  { value: "RouteToBillingReview", label: "Route To Billing Review" },
  { value: "ReturnToOperations", label: "Return To Operations" },
] satisfies ReadonlyArray<GenericSelectOption<BillingExceptionDisposition>>;

export const rateVarianceAutoResolutionModeChoices = [
  { value: "Disabled", label: "Disabled" },
  {
    value: "BypassReviewWithinTolerance",
    label: "Bypass Review Within Tolerance",
  },
] satisfies ReadonlyArray<GenericSelectOption<RateVarianceAutoResolutionMode>>;

export const unratedShipmentDispositionChoices = [
  { value: "FallbackFormulaTemplate", label: "Fall Back to Formula Template" },
  { value: "ZeroAndFlag", label: "Zero the Rate and Flag for Review" },
  { value: "Block", label: "Block the Save" },
] satisfies ReadonlyArray<GenericSelectOption<UnratedShipmentDisposition>>;

export const paymentTermChoices = [
  { value: "Net10", label: "Net 10", color: "var(--info)" },
  { value: "DueOnReceipt", label: "Due on Receipt", color: "var(--success)" },
  { value: "Net15", label: "Net 15", color: "var(--info-foreground)" },
  { value: "Net30", label: "Net 30", color: "var(--accent-violet)" },
  { value: "Net45", label: "Net 45", color: "var(--warning)" },
  { value: "Net60", label: "Net 60", color: "var(--danger)" },
  { value: "Net90", label: "Net 90", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<PaymentTerm>>;

export const adjustmentEligibilityPolicyChoices = [
  { value: "Disallow", label: "Disallow", color: "var(--danger)" },
  { value: "AllowWithApproval", label: "Allow With Approval", color: "var(--info)" },
  { value: "AllowWithoutApproval", label: "Allow Without Approval", color: "var(--warning)" },
] satisfies ReadonlyArray<GenericSelectOption<AdjustmentEligibilityPolicy>>;

export const adjustmentAccountingDatePolicyChoices = [
  {
    value: "UseOriginalIfOpenElseNextOpen",
    label: "Use Original If Open Else Next Open",
  },
  { value: "AlwaysNextOpen", label: "Always Next Open" },
] satisfies ReadonlyArray<GenericSelectOption<AdjustmentAccountingDatePolicy>>;

export const closedPeriodAdjustmentPolicyChoices = [
  { value: "Disallow", label: "Disallow", color: "var(--danger)" },
  { value: "RequireReopen", label: "Require Reopen", color: "var(--warning)" },
  {
    value: "PostInNextOpenPeriodWithApproval",
    label: "Post In Next Open Period With Approval",
    color: "var(--info)",
  },
] satisfies ReadonlyArray<GenericSelectOption<ClosedPeriodAdjustmentPolicy>>;

export const requirementPolicyChoices = [
  { value: "Optional", label: "Optional", color: "var(--foreground-subtle)" },
  { value: "Required", label: "Required", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<RequirementPolicy>>;

export const approvalPolicyChoices = [
  { value: "None", label: "None", color: "var(--foreground-subtle)" },
  { value: "Always", label: "Always", color: "var(--danger)" },
  { value: "AmountThreshold", label: "Amount Threshold", color: "var(--info)" },
] satisfies ReadonlyArray<GenericSelectOption<ApprovalPolicy>>;

export const writeOffApprovalPolicyChoices = [
  { value: "Disallow", label: "Disallow", color: "var(--danger)" },
  { value: "AlwaysRequireApproval", label: "Always Require Approval", color: "var(--info)" },
  {
    value: "RequireApprovalAboveThreshold",
    label: "Require Approval Above Threshold",
    color: "var(--warning)",
  },
] satisfies ReadonlyArray<GenericSelectOption<WriteOffApprovalPolicy>>;

export const replacementInvoiceReviewPolicyChoices = [
  { value: "NoAdditionalReview", label: "No Additional Review", color: "var(--foreground-subtle)" },
  {
    value: "RequireReviewWhenEconomicTermsChange",
    label: "Require Review When Economic Terms Change",
    color: "var(--info)",
  },
  { value: "AlwaysRequireReview", label: "Always Require Review", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<ReplacementInvoiceReviewPolicy>>;

export const customerCreditBalancePolicyChoices = [
  { value: "Disallow", label: "Disallow", color: "var(--danger)" },
  { value: "AllowUnappliedCredit", label: "Allow Unapplied Credit", color: "var(--info)" },
] satisfies ReadonlyArray<GenericSelectOption<CustomerCreditBalancePolicy>>;

export const overCreditPolicyChoices = [
  { value: "Block", label: "Block", color: "var(--danger)" },
  { value: "AllowWithApproval", label: "Allow With Approval", color: "var(--info)" },
] satisfies ReadonlyArray<GenericSelectOption<OverCreditPolicy>>;

export const supersededInvoiceVisibilityPolicyChoices = [
  {
    value: "ShowCurrentOnlyExternally",
    label: "Show Current Only Externally",
  },
  {
    value: "ShowCurrentAndSupersededExternally",
    label: "Show Current And Superseded Externally",
  },
] satisfies ReadonlyArray<GenericSelectOption<SupersededInvoiceVisibilityPolicy>>;

export const serviceIncidentTypeChoices = [
  { value: "Never", label: "Never", color: "var(--success)" },
  { value: "Pickup", label: "Pickup", color: "var(--accent-violet)" },
  { value: "Delivery", label: "Delivery", color: "var(--warning)" },
  { value: "PickupDelivery", label: "Pickup/Delivery", color: "var(--info)" },
  {
    value: "AllExceptShipper",
    label: "All Except Shipper",
    color: "var(--success-foreground)",
  },
] satisfies ReadonlyArray<GenericSelectOption<ServiceIncidentType>>;

export const autoAssignmentStrategyChoices = [
  { value: "Proximity", label: "Proximity", color: "var(--info)" },
  { value: "Availability", label: "Availability", color: "var(--success)" },
  { value: "LoadBalancing", label: "Load Balancing", color: "var(--accent-rose)" },
  { value: "Performance", label: "Performance", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<AutoAssignmentStrategy>>;

export const complianceEnforcementLevelChoices = [
  { value: "Warning", label: "Warning", color: "var(--warning)" },
  { value: "Block", label: "Block", color: "var(--danger)" },
  { value: "Audit", label: "Audit", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<ComplianceEnforcementLevel>>;

export const invoiceDeliveryChoices = [
  { value: "PerShipment", label: "Per shipment" },
  { value: "PerOrder", label: "Per order" },
  { value: "Consolidated", label: "Statement (consolidated)" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceDelivery>>;

export const billingCycleChoices = [
  { value: "Immediate", label: "Immediate" },
  { value: "Daily", label: "Daily" },
  { value: "Weekly", label: "Weekly" },
  { value: "BiWeekly", label: "Bi-weekly" },
  { value: "SemiMonthly", label: "Semi-monthly" },
  { value: "Monthly", label: "Monthly" },
  { value: "Quarterly", label: "Quarterly" },
] satisfies ReadonlyArray<GenericSelectOption<BillingCycle>>;

export const customerPaymentTermChoices = [
  { value: "DueOnReceipt", label: "Due on Receipt", color: "var(--success)" },
  { value: "Net10", label: "Net 10", color: "var(--info)" },
  { value: "Net15", label: "Net 15", color: "var(--info-foreground)" },
  { value: "Net30", label: "Net 30", color: "var(--accent-violet)" },
  { value: "Net45", label: "Net 45", color: "var(--warning)" },
  { value: "Net60", label: "Net 60", color: "var(--danger)" },
  { value: "Net90", label: "Net 90", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<CustomerPaymentTerm>>;

export const creditStatusChoices = [
  { value: "Active", label: "Active", color: "var(--success)" },
  { value: "Warning", label: "Warning", color: "var(--warning)" },
  { value: "Hold", label: "Hold", color: "var(--danger)" },
  { value: "Suspended", label: "Suspended", color: "var(--foreground-subtle)" },
  { value: "Review", label: "Review", color: "var(--accent-violet)" },
] satisfies ReadonlyArray<GenericSelectOption<CreditStatus>>;

export const invoiceSplitKeyChoices = [
  { value: "Customer", label: "Nothing — one invoice for the period" },
  { value: "CustomerAndPONumber", label: "PO number" },
  { value: "CustomerAndShipmentBOL", label: "BOL" },
  { value: "CustomerAndOrder", label: "Order" },
  { value: "CustomerAndOrigin", label: "Pickup location" },
  { value: "CustomerAndDestination", label: "Delivery location" },
  { value: "CustomerAndServiceType", label: "Service type" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceSplitKey>>;

export const invoiceSectionKeyChoices = [
  { value: "Shipment", label: "Shipment" },
  { value: "PONumber", label: "PO number" },
  { value: "Origin", label: "Pickup location" },
  { value: "Destination", label: "Delivery location" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceSectionKey>>;

export const invoiceDetailChoices = [
  { value: "Detailed", label: "Itemised — every charge line" },
  { value: "Summary", label: "Summary — one line per shipment" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceDetail>>;

export const invoiceNumberFormatChoices = [
  { value: "Default", label: "Default" },
  { value: "CustomPrefix", label: "Custom Prefix" },
  { value: "POBased", label: "PO Based" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceNumberFormat>>;

export {
  commentPriorityChoices,
  commentTypeChoices,
  commentVisibilityChoices,
} from "@trenova/shared/lib/comment-choices";

export const currencyChoices = [
  { value: "USD", label: "USD - US Dollar" },
  { value: "CAD", label: "CAD - Canadian Dollar" },
  { value: "MXN", label: "MXN - Mexican Peso" },
  { value: "EUR", label: "EUR - Euro" },
  { value: "GBP", label: "GBP - British Pound" },
] satisfies ReadonlyArray<SelectOption>;

export const caseFormatChoices = [
  { value: "AsEntered", label: "As Entered" },
  { value: "Upper", label: "UPPER" },
  { value: "Lower", label: "lower" },
  { value: "TitleCase", label: "Title Case" },
] satisfies ReadonlyArray<GenericSelectOption<CaseFormat>>;
export const resourceTypeChoices = [
  { value: "Shipment", label: "Shipment" },
  { value: "Trailer", label: "Trailer" },
  { value: "Tractor", label: "Tractor" },
  { value: "Worker", label: "Worker" },
] satisfies ReadonlyArray<GenericSelectOption<ResourceType>>;

export const documentKindChoices = [
  { value: "RateConfirmation", label: "Rate Confirmation" },
  { value: "BillOfLading", label: "Bill of Lading" },
  { value: "ProofOfDelivery", label: "Proof of Delivery" },
  { value: "Invoice", label: "Invoice" },
] satisfies ReadonlyArray<GenericSelectOption<DocumentKind>>;

export const invoiceStatusChoices = [
  { value: "Draft", label: "Draft" },
  { value: "Posted", label: "Posted" },
  { value: "Voided", label: "Voided" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceStatus>>;

export const invoiceVoidDispositionChoices = [
  {
    value: "Rebill",
    label: "Release freight for rebilling",
    description: "The billing queue items go back to Approved with a fresh number.",
  },
  {
    value: "DoNotRebill",
    label: "Do not rebill",
    description: "The billing queue items are canceled and the shipments settle as completed.",
  },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceVoidDisposition>>;

export const memoBillTypeChoices = [
  { value: "CreditMemo", label: "Credit Memo" },
  { value: "DebitMemo", label: "Debit Memo" },
] satisfies ReadonlyArray<GenericSelectOption<"CreditMemo" | "DebitMemo">>;

export const invoiceDisputeReasonCodeChoices = [
  { value: "RateDiscrepancy", label: "Rate discrepancy" },
  { value: "AccessorialDisputed", label: "Accessorial disputed" },
  { value: "ServiceFailure", label: "Service failure" },
  { value: "DuplicateBilling", label: "Duplicate billing" },
  { value: "WrongBillTo", label: "Wrong bill-to" },
  { value: "MissingDocumentation", label: "Missing documentation" },
  { value: "Other", label: "Other" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceDisputeReasonCode>>;

export const invoiceDisputeResolutionChoices = [
  { value: "CreditIssued", label: "Credit issued" },
  { value: "InvoiceUpheld", label: "Invoice upheld" },
  { value: "Rebilled", label: "Rebilled" },
  { value: "WrittenOff", label: "Written off" },
  { value: "CustomerWithdrew", label: "Customer withdrew" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceDisputeResolution>>;

export const invoiceEdiSendStatusChoices = [
  { value: "NotSent", label: "Not sent" },
  { value: "NotConfigured", label: "Not configured" },
  { value: "Queued", label: "Queued" },
  { value: "Generated", label: "Generated" },
  { value: "Sending", label: "Sending" },
  { value: "Sent", label: "Sent" },
  { value: "Failed", label: "Failed" },
  { value: "DeadLettered", label: "Dead-lettered" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceEdiSendStatus>>;

export const lateChargeAssessmentModeChoices = [
  { value: "Disabled", label: "Disabled" },
  { value: "Preview", label: "Preview only" },
  { value: "Automatic", label: "Automatic" },
] satisfies ReadonlyArray<GenericSelectOption<LateChargeAssessmentMode>>;

export const invoiceScopeChoices = [
  { value: "Shipment", label: "Single Shipment" },
  { value: "Order", label: "Order" },
  { value: "Consolidated", label: "Consolidated" },
  { value: "Adjustment", label: "Adjustment" },
  { value: "Memo", label: "Memo" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceScope>>;

export const exceptionReasonLabels: Record<ExceptionReasonCode, string> = {
  MissingDocumentation: "Missing Documentation",
  IncorrectRates: "Incorrect Rates",
  WeightDiscrepancy: "Weight Discrepancy",
  AccessorialDispute: "Accessorial Dispute",
  DuplicateCharge: "Duplicate Charge",
  MissingReferenceNumber: "Missing Reference Number",
  CustomerInformationError: "Customer Information Error",
  ServiceFailure: "Service Failure",
  RateNotOnFile: "Rate Not On File",
  Other: "Other",
};

export const manualJournalStatusChoices = [
  { label: "Draft", value: "Draft" },
  { label: "Pending Approval", value: "PendingApproval" },
  { label: "Approved", value: "Approved" },
  { label: "Rejected", value: "Rejected" },
  { label: "Cancelled", value: "Cancelled" },
  { label: "Posted", value: "Posted" },
] satisfies ReadonlyArray<GenericSelectOption<ManualJournalStatus>>;

export const journalReversalStatusChoices = [
  { label: "Requested", value: "Requested" },
  { label: "Pending Approval", value: "PendingApproval" },
  { label: "Approved", value: "Approved" },
  { label: "Rejected", value: "Rejected" },
  { label: "Cancelled", value: "Cancelled" },
  { label: "Posted", value: "Posted" },
] satisfies ReadonlyArray<GenericSelectOption<JournalReversalStatus>>;

export const bankReceiptBatchStatusChoices = [
  { label: "Processing", value: "Processing" },
  { label: "Completed", value: "Completed" },
] satisfies ReadonlyArray<GenericSelectOption<BankReceiptBatchStatus>>;

export const bankReceiptStatusChoices = [
  { label: "Imported", value: "Imported" },
  { label: "Matched", value: "Matched" },
  { label: "Exception", value: "Exception" },
] satisfies ReadonlyArray<GenericSelectOption<BankReceiptStatus>>;

export const workItemStatusChoices = [
  { label: "Open", value: "Open" },
  { label: "Assigned", value: "Assigned" },
  { label: "In Review", value: "InReview" },
  { label: "Resolved", value: "Resolved" },
  { label: "Dismissed", value: "Dismissed" },
] satisfies ReadonlyArray<GenericSelectOption<WorkItemStatus>>;

export const paymentMethodChoices = [
  { label: "ACH", value: "ACH" },
  { label: "Check", value: "Check" },
  { label: "Wire", value: "Wire" },
  { label: "Card", value: "Card" },
  { label: "Cash", value: "Cash" },
  { label: "Other", value: "Other" },
] satisfies ReadonlyArray<GenericSelectOption<PaymentMethod>>;

export const resolutionTypeChoices = [
  { label: "Matched to Payment", value: "MatchedToPayment" },
  { label: "Marked False Positive", value: "MarkedFalsePositive" },
  { label: "Requires External Follow-Up", value: "RequiresExternalFollowUp" },
  { label: "Superseded", value: "Superseded" },
] satisfies ReadonlyArray<GenericSelectOption<ResolutionType>>;

export const ediTransferStatusChoices = [
  { label: "Submitted", value: "Submitted" },
  { label: "Mapping Required", value: "MappingRequired" },
  { label: "Pending Approval", value: "PendingApproval" },
  { label: "Processing", value: "Processing" },
  { label: "Approved", value: "Approved" },
  { label: "Rejected", value: "Rejected" },
  { label: "Expired", value: "Expired" },
  { label: "Canceled", value: "Canceled" },
  { label: "Failed", value: "Failed" },
] satisfies ReadonlyArray<GenericSelectOption<EDITransferStatus>>;

export const ediMessageDeliveryStatusChoices = [
  { label: "Queued", value: "Queued" },
  { label: "Sending", value: "Sending" },
  { label: "Sent", value: "Sent" },
  { label: "Failed", value: "Failed" },
  { label: "Dead Lettered", value: "DeadLettered" },
] satisfies ReadonlyArray<GenericSelectOption<EDIMessageDeliveryStatus>>;

export const ediAckStatusChoices = [
  { label: "Not Expected", value: "NotExpected" },
  { label: "Pending", value: "Pending" },
  { label: "Accepted", value: "Accepted" },
  { label: "Rejected", value: "Rejected" },
  { label: "Failed", value: "Failed" },
] satisfies ReadonlyArray<GenericSelectOption<EDIMessageAcknowledgmentStatus>>;

export const ediInboundFileStatusChoices = [
  { label: "Received", value: "Received" },
  { label: "Parsed", value: "Parsed" },
  { label: "Processed", value: "Processed" },
  { label: "Partially Processed", value: "PartiallyProcessed" },
  { label: "Quarantined", value: "Quarantined" },
  { label: "Duplicate", value: "Duplicate" },
] satisfies ReadonlyArray<GenericSelectOption<EDIInboundFileStatus>>;

export const ediConnectionMethodChoices = [
  { label: "Internal", value: "Internal" },
  { label: "AS2", value: "AS2" },
  { label: "SFTP", value: "SFTP" },
  { label: "VAN", value: "VAN" },
];

export const ediTransactionSetChoices = [
  { label: "204 Load Tender", value: "204" },
  { label: "210 Freight Invoice", value: "210" },
  { label: "214 Shipment Status", value: "214" },
  { label: "990 Tender Response", value: "990" },
  { label: "997 Functional Ack", value: "997" },
  { label: "999 Implementation Ack", value: "999" },
];

export const ediDocumentDirectionChoices = [
  { label: "Inbound", value: "Inbound" },
  { label: "Outbound", value: "Outbound" },
];

export const payeeClassificationChoices = [
  {
    label: "Company Driver",
    value: "CompanyDriver",
    color: "var(--info)",
    description: "W-2 employee paid through driver pay expense.",
  },
  {
    label: "Owner-Operator",
    value: "OwnerOperator",
    color: "var(--accent-violet)",
    description: "1099 contractor paid through purchased transportation.",
  },
] satisfies ReadonlyArray<GenericSelectOption<PayeeClassification>>;

export const payComponentKindChoices = [
  {
    label: "Linehaul",
    value: "Linehaul",
    color: "var(--info)",
    description: "Base haul pay — per-mile or percent-of-revenue.",
  },
  {
    label: "Fuel Surcharge",
    value: "FuelSurcharge",
    color: "var(--warning)",
    description: "Passes a share of the shipment's fuel surcharge to the driver.",
  },
  {
    label: "Stop Pay",
    value: "StopPay",
    color: "var(--accent-teal)",
    description: "Pays for each extra stop beyond pickup and delivery.",
  },
  {
    label: "Detention",
    value: "Detention",
    color: "var(--warning-foreground)",
    description: "Hourly pay for dwell time beyond the free-time allowance.",
  },
  {
    label: "Layover",
    value: "Layover",
    color: "var(--accent-violet)",
    description: "Per-day pay when the driver is held over away from home.",
  },
  {
    label: "Breakdown",
    value: "Breakdown",
    color: "var(--danger)",
    description: "Pay while the truck is down for repairs.",
  },
  {
    label: "Tarp",
    value: "Tarp",
    color: "var(--accent-indigo)",
    description: "Flat pay for tarping flatbed loads.",
  },
  {
    label: "Hazmat",
    value: "Hazmat",
    color: "var(--accent-violet-on-subtle)",
    description: "Premium applied when the shipment carries hazardous materials.",
  },
  {
    label: "Bonus",
    value: "Bonus",
    color: "var(--success)",
    description: "Discretionary or program bonus tied to the move.",
  },
  {
    label: "Custom",
    value: "Custom",
    color: "var(--foreground-subtle)",
    description: "Carrier-defined component — describe it so drivers recognize it.",
  },
] satisfies ReadonlyArray<GenericSelectOption<PayComponentKind>>;

export const payCalcMethodChoices = [
  {
    label: "Per Loaded Mile",
    value: "PerLoadedMile",
    color: "var(--info)",
    description: "Rate × the move's loaded miles; supports mileage bands.",
  },
  {
    label: "Per Empty Mile",
    value: "PerEmptyMile",
    color: "var(--accent-teal)",
    description: "Rate × the move's empty (deadhead) miles.",
  },
  {
    label: "Per Total Mile",
    value: "PerTotalMile",
    color: "var(--accent-indigo)",
    description: "Rate × all dispatched miles, loaded or empty.",
  },
  {
    label: "Percent of Revenue",
    value: "PercentOfRevenue",
    color: "var(--accent-violet)",
    description: "Share of shipment revenue, allocated to the move by distance.",
  },
  {
    label: "Flat per Shipment",
    value: "FlatPerShipment",
    color: "var(--success)",
    description: "Fixed amount for each shipment regardless of miles.",
  },
  {
    label: "Per Stop",
    value: "PerStop",
    color: "var(--warning)",
    description: "Rate × extra stops beyond pickup and delivery.",
  },
  {
    label: "Per Hour",
    value: "PerHour",
    color: "var(--warning-foreground)",
    description: "Rate × hours — used for detention beyond free time.",
  },
  {
    label: "Per Day",
    value: "PerDay",
    color: "var(--accent-violet-on-subtle)",
    description: "Rate × days — used for layover and similar daily pay.",
  },
  {
    label: "Per Event",
    value: "PerEvent",
    color: "var(--foreground-subtle)",
    description: "Fixed amount per occurrence (breakdown, tarp, etc.).",
  },
] satisfies ReadonlyArray<GenericSelectOption<PayCalcMethod>>;

export const payRevenueBasisChoices = [
  {
    label: "Linehaul",
    value: "Linehaul",
    color: "var(--info)",
    description: "Percentage applies to freight charges only.",
  },
  {
    label: "Linehaul + Fuel Surcharge",
    value: "LinehaulPlusFuelSurcharge",
    color: "var(--warning)",
    description: "Percentage applies to freight charges plus fuel surcharge.",
  },
  {
    label: "Total Revenue",
    value: "TotalRevenue",
    color: "var(--success)",
    description: "Percentage applies to every charge on the shipment.",
  },
] satisfies ReadonlyArray<GenericSelectOption<PayRevenueBasis>>;

export const payCodeDirectionChoices = [
  {
    label: "Earning",
    value: "Earning",
    color: "var(--success)",
    description: "Adds pay to settlements — bonuses, per diem, stipends.",
  },
  {
    label: "Deduction",
    value: "Deduction",
    color: "var(--danger)",
    description: "Withholds pay from settlements — leases, insurance, repayments.",
  },
] satisfies ReadonlyArray<GenericSelectOption<PayCodeDirection>>;

export const recurringDeductionFrequencyChoices = [
  {
    label: "Every Settlement",
    value: "EverySettlement",
    color: "var(--info)",
    description: "Withheld from every qualifying settlement.",
  },
  {
    label: "Monthly",
    value: "Monthly",
    color: "var(--accent-violet)",
    description: "Withheld only from the first settlement of each month.",
  },
] satisfies ReadonlyArray<GenericSelectOption<RecurringDeductionFrequency>>;

export const recurringDeductionStatusChoices = [
  {
    label: "Active",
    value: "Active",
    color: "var(--success)",
    description: "Withheld automatically from each qualifying settlement.",
  },
  {
    label: "Paused",
    value: "Paused",
    color: "var(--warning)",
    description: "Temporarily skipped; history is kept and it can resume anytime.",
  },
  {
    label: "Completed",
    value: "Completed",
    color: "var(--foreground-subtle)",
    description: "Reached its lifetime cap and stopped permanently.",
  },
] satisfies ReadonlyArray<GenericSelectOption<RecurringDeductionStatus>>;

export const recurringEarningFrequencyChoices = [
  {
    label: "Every Settlement",
    value: "EverySettlement",
    color: "var(--info)",
    description: "Added to every qualifying settlement.",
  },
  {
    label: "Monthly",
    value: "Monthly",
    color: "var(--accent-violet)",
    description: "Added only to the first settlement of each month.",
  },
] satisfies ReadonlyArray<GenericSelectOption<RecurringEarningFrequency>>;

export const recurringEarningStatusChoices = [
  {
    label: "Active",
    value: "Active",
    color: "var(--success)",
    description: "Added automatically to each qualifying settlement.",
  },
  {
    label: "Paused",
    value: "Paused",
    color: "var(--warning)",
    description: "Temporarily skipped; history is kept and it can resume anytime.",
  },
  {
    label: "Completed",
    value: "Completed",
    color: "var(--foreground-subtle)",
    description: "Reached its lifetime cap and stopped permanently.",
  },
] satisfies ReadonlyArray<GenericSelectOption<RecurringEarningStatus>>;

export const recurringShipmentStatusChoices = [
  {
    label: "Active",
    value: "Active",
    color: "var(--success)",
    description: "The schedule runs and shipments generate on their own.",
  },
  {
    label: "Paused",
    value: "Paused",
    color: "var(--warning)",
    description:
      "Nothing generates until resumed; the schedule restarts from the next future pickup.",
  },
  {
    label: "Expired",
    value: "Expired",
    color: "var(--foreground-subtle)",
    description: "The end date or occurrence cap was reached and the series stopped for good.",
  },
] satisfies ReadonlyArray<GenericSelectOption<RecurringShipmentStatus>>;

export const recurringShipmentExceptionPolicyChoices = [
  {
    label: "Skip the occurrence",
    value: "Skip",
    color: "var(--foreground-subtle)",
    description: "No shipment is created and the series waits for the next scheduled pickup.",
  },
  {
    label: "Move to previous business day",
    value: "PreviousBusinessDay",
    color: "var(--info)",
    description: "Pulls the pickup earlier — use when the freight cannot ship late.",
  },
  {
    label: "Move to next business day",
    value: "NextBusinessDay",
    color: "var(--accent-violet)",
    description: "Pushes the pickup later — the common choice for holiday closures.",
  },
] satisfies ReadonlyArray<GenericSelectOption<RecurringShipmentExceptionPolicy>>;

export const payAdvanceSourceChoices = [
  {
    label: "Cash",
    value: "Cash",
    color: "var(--success)",
    description: "Cash handed to the driver directly.",
  },
  {
    label: "EFS Money Code",
    value: "EFSMoneyCode",
    color: "var(--info)",
    description: "EFS code the driver cashes at a truck stop.",
  },
  {
    label: "Comdata Code",
    value: "ComdataCode",
    color: "var(--accent-indigo)",
    description: "Comdata Comchek code issued to the driver.",
  },
  {
    label: "Fuel Card",
    value: "FuelCard",
    color: "var(--warning)",
    description: "Cash advance drawn on the driver's fuel card.",
  },
  {
    label: "Other",
    value: "Other",
    color: "var(--foreground-subtle)",
    description: "Any other advance mechanism — note the details on the record.",
  },
] satisfies ReadonlyArray<GenericSelectOption<PayAdvanceSource>>;

export const payPeriodFrequencyChoices = [
  {
    label: "Weekly",
    value: "Weekly",
    color: "var(--info)",
    description: "Settlements cover one week, ending on the configured weekday.",
  },
  {
    label: "Biweekly",
    value: "Biweekly",
    color: "var(--accent-indigo)",
    description: "Settlements cover two weeks, ending on the configured weekday.",
  },
  {
    label: "Monthly",
    value: "Monthly",
    color: "var(--accent-violet)",
    description: "Settlements cover one month, ending on the configured weekday.",
  },
] satisfies ReadonlyArray<GenericSelectOption<PayPeriodFrequency>>;

export const settlementPayTriggerChoices = [
  {
    label: "Move Completed",
    value: "MoveCompleted",
    color: "var(--success)",
    description: "Pay accrues the moment a driver finishes their own move — best for split loads.",
  },
  {
    label: "Shipment Delivered",
    value: "ShipmentDelivered",
    color: "var(--info)",
    description: "Pay accrues when the whole shipment reaches Completed.",
  },
  {
    label: "POD Received (Ready to Invoice)",
    value: "PODReceived",
    color: "var(--warning)",
    description: "Pay accrues once paperwork is in and the shipment is ready to invoice.",
  },
  {
    label: "Shipment Invoiced",
    value: "ShipmentInvoiced",
    color: "var(--accent-violet)",
    description: "Pay accrues only after the customer has been invoiced.",
  },
] satisfies ReadonlyArray<GenericSelectOption<SettlementPayTrigger>>;

export const settlementDisputeStatusChoices = [
  {
    label: "Open",
    value: "Open",
    color: "var(--info)",
    description: "Newly submitted by the driver and waiting for a first look.",
  },
  {
    label: "In Review",
    value: "InReview",
    color: "var(--warning)",
    description: "Being investigated by payroll or the fleet manager.",
  },
  {
    label: "Resolved",
    value: "Resolved",
    color: "var(--success)",
    description: "Closed in the driver's favor, optionally with a correcting adjustment.",
  },
  {
    label: "Denied",
    value: "Denied",
    color: "var(--danger)",
    description: "Closed with an explanation; the original settlement stands.",
  },
  {
    label: "Withdrawn",
    value: "Withdrawn",
    color: "var(--foreground-subtle)",
    description: "Pulled back by the driver before a decision was made.",
  },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const settlementDisputeCategoryChoices = [
  {
    label: "Missing Pay",
    value: "MissingPay",
    color: "var(--danger)",
    description: "A load or accessorial the driver ran isn't on the statement.",
  },
  {
    label: "Incorrect Rate",
    value: "IncorrectRate",
    color: "var(--warning)",
    description: "Pay was calculated with the wrong rate or mileage.",
  },
  {
    label: "Incorrect Deduction",
    value: "IncorrectDeduction",
    color: "var(--accent-violet)",
    description: "A deduction is wrong, duplicated, or shouldn't apply.",
  },
  {
    label: "Missing Reimbursement",
    value: "MissingReimbursement",
    color: "var(--info)",
    description: "An expense the carrier owes back wasn't reimbursed.",
  },
  {
    label: "Other",
    value: "Other",
    color: "var(--foreground-subtle)",
    description: "Anything else about the statement that looks off.",
  },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const detentionPolicyStatusChoices = [
  { label: "Active", value: "Active", color: "var(--success)" },
  { label: "Draft", value: "Draft", color: "var(--accent-violet)" },
  { label: "Inactive", value: "Inactive", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const detentionClockStartBasisChoices = [
  { label: "Later of arrival or appointment", value: "LaterOfArrivalOrAppointment" },
  { label: "Actual arrival", value: "Arrival" },
  { label: "Appointment, regardless of arrival", value: "Appointment" },
  { label: "Earlier of arrival or appointment", value: "EarlierOfArrivalOrAppointment" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const detentionLateArrivalRuleChoices = [
  { label: "No effect on entitlement", value: "NoEffect" },
  { label: "Forfeit detention entirely", value: "Forfeit" },
  { label: "Anchor the clock to the appointment", value: "ClockFromAppointment" },
  { label: "Subtract lateness from free time", value: "ReduceFreeTime" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const detentionRoundingModeChoices = [
  { label: "Round up", value: "Up" },
  { label: "Round down", value: "Down" },
  { label: "Round to nearest", value: "Nearest" },
  { label: "Bill exact minutes", value: "Exact" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const detentionRateSourceChoices = [
  { label: "Flat accessorial rate", value: "Accessorial" },
  { label: "Graduated tiers", value: "Tiers" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const detentionNotificationRequirementChoices = [
  { label: "No notice required", value: "None" },
  { label: "Advisory only", value: "Advisory" },
  { label: "Required to bill", value: "Required" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const detentionUnnotifiedBehaviorChoices = [
  { label: "Bill anyway", value: "Bill" },
  { label: "Hold for review", value: "Flag" },
  { label: "Suppress the charge", value: "Suppress" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const detentionWaiverReasonChoices = [
  {
    label: "Weather",
    value: "Weather",
    description: "Weather kept the facility from working the truck.",
  },
  {
    label: "Facility Closure",
    value: "FacilityClosure",
    description: "The facility was closed or unable to receive during the stay.",
  },
  {
    label: "Carrier Fault",
    value: "CarrierFault",
    description: "Our own error caused the delay, so the charge is not defensible.",
  },
  {
    label: "Equipment Issue",
    value: "EquipmentIssue",
    description: "An equipment problem on our side extended the dwell.",
  },
  {
    label: "Customer Goodwill",
    value: "CustomerGoodwill",
    description: "A commercial concession to preserve the relationship.",
  },
  {
    label: "Data Correction",
    value: "DataCorrection",
    description: "The underlying timestamps were wrong; the charge should not stand.",
  },
  {
    label: "Force Majeure",
    value: "ForceMajeure",
    description: "An event outside anyone's control caused the detention.",
  },
  {
    label: "Other",
    value: "Other",
    description: "Anything else — explain it in the note.",
  },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const weekdayChoices = [
  { label: "Sunday", value: 0 },
  { label: "Monday", value: 1 },
  { label: "Tuesday", value: 2 },
  { label: "Wednesday", value: 3 },
  { label: "Thursday", value: 4 },
  { label: "Friday", value: 5 },
  { label: "Saturday", value: 6 },
] satisfies ReadonlyArray<GenericSelectOption<number>>;

/* -------------------------------------------------------------------------- */
/*                              Rate agreements                                */
/* -------------------------------------------------------------------------- */

export const rateAgreementStatusChoices = [
  { label: "Active", value: "Active", color: "var(--success)" },
  { label: "In review", value: "InReview", color: "var(--warning)" },
  { label: "Draft", value: "Draft", color: "var(--accent-violet)" },
  { label: "Suspended", value: "Suspended", color: "var(--warning-foreground)" },
  { label: "Expired", value: "Expired", color: "var(--foreground-subtle)" },
  { label: "Archived", value: "Archived", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const ratePartyTypeChoices = [
  { label: "Customer", value: "Customer", color: "var(--info)" },
  { label: "Carrier", value: "Carrier", color: "var(--accent-teal)" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateAgreementTypeChoices = [
  { label: "Contract", value: "Contract" },
  { label: "Tariff", value: "Tariff" },
  { label: "Spot", value: "Spot" },
  { label: "Project", value: "Project" },
  { label: "Dedicated", value: "Dedicated" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

/**
 * Ordered widest to narrowest, which is the order the resolver ranks lanes in.
 * A rule written at a narrower scope beats a wider one covering the same load.
 */
export const rateScopeTypeChoices = [
  { label: "Anywhere", value: "Any" },
  { label: "Country", value: "Country" },
  { label: "State", value: "State" },
  { label: "Zone", value: "Zone" },
  { label: "Radius", value: "Radius" },
  { label: "City", value: "CityState" },
  { label: "Postal prefix", value: "Zip3" },
  { label: "Postal code", value: "Zip5" },
  { label: "Location", value: "Location" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateDirectionChoices = [
  { label: "One way", value: "Directional" },
  { label: "Both ways", value: "Bidirectional" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateRoundingModeChoices = [
  { label: "Half up", value: "HalfUp" },
  { label: "Half even", value: "HalfEven" },
  { label: "Up", value: "Up" },
  { label: "Down", value: "Down" },
  { label: "None", value: "None" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateZoneKindChoices = [
  { label: "Custom", value: "Custom" },
  { label: "Market area (KMA)", value: "KMA" },
  { label: "Region", value: "Regional" },
  { label: "Metro", value: "Metro" },
  { label: "Country", value: "Country" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const freightClassSourceChoices = [
  { label: "From the commodity", value: "Commodity" },
  { label: "Fixed on the lane", value: "Fixed" },
  { label: "Derived from density", value: "Density" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateQuoteOutcomeChoices = [
  { label: "Rated", value: "Rated", color: "var(--success)" },
  { label: "Formula fallback", value: "FormulaFallback", color: "var(--accent-teal)" },
  { label: "Manual override", value: "ManualOverride", color: "var(--warning)" },
  { label: "No rate found", value: "NoRateFound", color: "var(--warning-foreground)" },
  { label: "Error", value: "Error", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

/** The bounds the rate engine records as guardrails when one changes a price. */
export const rateGuardrailKindChoices = [
  { label: "Minimum charge", value: "MinimumCharge" },
  { label: "Maximum charge", value: "MaximumCharge" },
  { label: "Absolute minimum charge", value: "AbsoluteMinCharge" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateQuotePurposeChoices = [
  { label: "Rating", value: "Rating" },
  { label: "Quote", value: "Quote" },
  { label: "Shopping", value: "Shopping" },
  { label: "Simulation", value: "Simulation" },
  { label: "What if", value: "WhatIf" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateMatrixDimensionKindChoices = [
  { label: "Zone", value: "Zone" },
  { label: "Postal prefix", value: "Zip3" },
  { label: "Postal code", value: "Zip5" },
  { label: "State", value: "State" },
  { label: "Country", value: "Country" },
  { label: "Weight break", value: "WeightBreak" },
  { label: "Distance", value: "Distance" },
  { label: "Piece count", value: "PieceCount" },
  { label: "Linear feet", value: "LinearFeet" },
  { label: "Freight class", value: "FreightClass" },
  { label: "Equipment type", value: "EquipmentType" },
  { label: "Service type", value: "ServiceType" },
  { label: "Custom", value: "Custom" },
  { label: "Quantity", value: "Quantity" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateMatrixMatchModeChoices = [
  { label: "Exact key", value: "Exact" },
  { label: "Band", value: "Range" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateMatrixKeyNormalizationChoices = [
  { label: "As entered", value: "None" },
  { label: "Trim whitespace", value: "Trim" },
  { label: "Upper case", value: "Upper" },
  { label: "ZIP3 (first three characters)", value: "Zip3" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const rateMatrixRangeOverflowChoices = [
  { label: "No match (strict)", value: "Error" },
  { label: "Clamp to top band", value: "ClampToTopBand" },
  { label: "Nearest band", value: "Nearest" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const fuelCardProviderChoices = fuelCardProviderSchema.options.map((value) => ({
  label: FUEL_CARD_PROVIDER_LABELS[value],
  value,
})) satisfies ReadonlyArray<GenericSelectOption<FuelCardProvider>>;

export const fuelCardStatusChoices = [
  { label: FUEL_CARD_STATUS_LABELS.Active, value: "Active", color: "var(--success)" },
  { label: FUEL_CARD_STATUS_LABELS.Suspended, value: "Suspended", color: "var(--warning)" },
  { label: FUEL_CARD_STATUS_LABELS.Cancelled, value: "Cancelled", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<FuelCardStatus>>;

export const fuelPurchaseSourceChoices = fuelPurchaseSourceSchema.options.map((value) => ({
  label: FUEL_PURCHASE_SOURCE_LABELS[value],
  value,
})) satisfies ReadonlyArray<GenericSelectOption<FuelPurchaseSource>>;

export const iftaFuelTypeChoices = iftaFuelTypeSchema.options.map((value) => ({
  label: IFTA_FUEL_TYPE_LABELS[value],
  value,
})) satisfies ReadonlyArray<GenericSelectOption<IftaFuelType>>;

export const fuelQuantityUnitChoices = fuelQuantityUnitSchema.options.map((value) => ({
  label: FUEL_QUANTITY_UNIT_LABELS[value],
  value,
})) satisfies ReadonlyArray<GenericSelectOption<FuelQuantityUnit>>;

export const iftaReturnStatusChoices = [
  { label: IFTA_RETURN_STATUS_LABELS.Draft, value: "Draft", color: "var(--foreground-subtle)" },
  { label: IFTA_RETURN_STATUS_LABELS.Finalized, value: "Finalized", color: "var(--info)" },
  { label: IFTA_RETURN_STATUS_LABELS.Filed, value: "Filed", color: "var(--success)" },
] satisfies ReadonlyArray<GenericSelectOption<IftaReturnStatus>>;

export const iftaQuarterChoices = iftaQuarterSchema.options.map((value) => ({
  label: IFTA_QUARTER_LABELS[value],
  value,
})) satisfies ReadonlyArray<GenericSelectOption<IftaQuarter>>;

export const iftaMileageSourceChoices = iftaMileageSourceSchema.options.map((value) => ({
  label: IFTA_MILEAGE_SOURCE_LABELS[value],
  value,
})) satisfies ReadonlyArray<GenericSelectOption<IftaMileageSource>>;

export const fuelPurchaseImportStatusChoices = [
  { label: FUEL_PURCHASE_IMPORT_STATUS_LABELS.Pending, value: "Pending", color: "var(--warning)" },
  { label: FUEL_PURCHASE_IMPORT_STATUS_LABELS.Parsed, value: "Parsed", color: "var(--info)" },
  { label: FUEL_PURCHASE_IMPORT_STATUS_LABELS.Committed, value: "Committed", color: "var(--success)" },
  { label: FUEL_PURCHASE_IMPORT_STATUS_LABELS.Failed, value: "Failed", color: "var(--danger)" },
  { label: FUEL_PURCHASE_IMPORT_STATUS_LABELS.Discarded, value: "Discarded", color: "var(--foreground-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<FuelPurchaseImportStatus>>;
