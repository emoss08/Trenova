import type { AccessorialChargeMethod, RateUnit } from "@/types/accessorial-charge";
import type { AccountCategory } from "@/types/account-type";
import type { BankReceiptStatus } from "@/types/bank-receipt";
import type { BankReceiptBatchStatus } from "@/types/bank-receipt-batch";
import type { ResolutionType, WorkItemStatus } from "@/types/bank-receipt-work-item";
import type { PaymentMethod } from "@/types/customer-payment";
import type { JournalReversalStatus } from "@/types/journal-reversal";
import type { ManualJournalStatus } from "@/types/manual-journal";
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
import type { BillingQueueStatus, ExceptionReasonCode } from "@/types/billing-queue";
import type {
  BillingExceptionDisposition,
  BillingQueueTransferMode,
  EnforcementLevel,
  InvoiceDraftCreationMode,
  InvoicePostingMode,
  PaymentTerm,
  RateVarianceAutoResolutionMode,
  ReadyToBillAssignmentMode,
  TransferSchedule,
} from "@/types/billing-control";
import type { FreightClass } from "@/types/commodity";
import type { FieldType } from "@/types/custom-field";
import type {
  BillingCycleType,
  ConsolidationGroupBy,
  CreditStatus,
  CustomerPaymentTerm,
  InvoiceAdjustmentSupportingDocumentPolicy,
  InvoiceMethod,
  InvoiceNumberFormat,
} from "@/types/customer";
import type { CaseFormat } from "@/types/data-entry-control";
import type {
  AutoAssignmentStrategy,
  ComplianceEnforcementLevel,
  ServiceIncidentType,
} from "@/types/dispatch-control";
import type { ResourceType } from "@/types/document-packet-rule";
import type { InvoiceStatus } from "@/types/invoice";
import type { DocumentKind } from "@/types/document-parsing-rule";
import type { DocumentCategory, DocumentClassification } from "@/types/document-type";
import type { EquipmentClass } from "@/types/equipment-type";
import type { GenericSelectOption, SelectOption, SelectOptionGroup } from "@/types/fields";
import type { FiscalPeriodStatus, PeriodType } from "@/types/fiscal-period";
import type { FiscalYearStatus } from "@/types/fiscal-year";
import type { FormulaTemplateStatus, FormulaTemplateType } from "@/types/formula-template";
import type { HazardousClass, PackingGroup } from "@/types/hazardous-material";
import type { SegregationDistanceUnit, SegregationType } from "@/types/hazmat-segregation-rule";
import type { EquipmentStatus, Status } from "@/types/helpers";
import type { HoldSeverity, HoldType } from "@/types/hold-reason";
import type {
  AdjustmentAccountingDatePolicy,
  AdjustmentAttachmentPolicy,
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
import type { FacilityType, LocationCategoryType } from "@/types/location-category";
import type { CoreResponsibility, DataScope, FieldSensitivity, Operation } from "@/types/role";
import type {
  MoveStatus,
  ShipmentStatus,
  StopScheduleType,
  StopStatus,
  StopType,
} from "@/types/shipment";
import type { CommentPriority, CommentType, CommentVisibility } from "@/types/shipment-comment";
import type { TimeFormatType } from "@/types/user";
import type {
  CDLClass,
  ComplianceStatus,
  DriverType,
  EndorsementType,
  Gender,
  PTOStatus,
  PTOType,
  WorkerType,
} from "@/types/worker";

export const formulaTemplateStatusChoices = [
  { label: "Active", value: "Active", color: "#15803d" },
  { label: "Inactive", value: "Inactive", color: "#dc2626" },
  { label: "Draft", value: "Draft", color: "#a3a3a3" },
] satisfies ReadonlyArray<GenericSelectOption<FormulaTemplateStatus>>;

export const formulaTemplateTypeChoices = [
  { label: "Freight Charge", value: "FreightCharge" },
  { label: "Accessorial Charge", value: "AccessorialCharge" },
] satisfies ReadonlyArray<GenericSelectOption<FormulaTemplateType>>;

export const statusChoices = [
  { label: "Active", value: "Active", color: "#15803d" },
  { label: "Inactive", value: "Inactive", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<Status>>;

export const shipmentStatusChoices = [
  { label: "New", value: "New", color: "#3b82f6" },
  { label: "Partially Assigned", value: "PartiallyAssigned", color: "#a855f7" },
  { label: "Assigned", value: "Assigned", color: "#16a34a" },
  { label: "In Transit", value: "InTransit", color: "#0891b2" },
  { label: "Delayed", value: "Delayed", color: "#dc2626" },
  { label: "Partially Completed", value: "PartiallyCompleted", color: "#f59e0b" },
  {
    value: "Completed",
    label: "Completed",
    color: "#16a34a",
  },
  { label: "Ready To Invoice", value: "ReadyToInvoice", color: "#0f766e" },
  { label: "Invoiced", value: "Invoiced", color: "#166534" },
  { label: "Canceled", value: "Canceled", color: "#b91c1c" },
] satisfies ReadonlyArray<GenericSelectOption<ShipmentStatus>>;

export const billTypeChoices = [
  { label: "Invoice", value: "Invoice", color: "#3b82f6" },
  { label: "Credit Memo", value: "CreditMemo", color: "#f59e0b" },
  { label: "Debit Memo", value: "DebitMemo", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const billingQueueStatusChoices = [
  { label: "Ready for Review", value: "ReadyForReview", color: "#3b82f6" },
  { label: "In Review", value: "InReview", color: "#0891b2" },
  { label: "Approved", value: "Approved", color: "#16a34a" },
  { label: "On Hold", value: "OnHold", color: "#f59e0b" },
  { label: "Sent Back to Ops", value: "SentBackToOps", color: "#f97316" },
  { label: "Exception", value: "Exception", color: "#dc2626" },
  { label: "Canceled", value: "Canceled", color: "#b91c1c" },
] satisfies ReadonlyArray<GenericSelectOption<BillingQueueStatus>>;

export const billingTransferStatusChoices = [
  { label: "Ready for Review", value: "ReadyForReview", color: "#3b82f6" },
  { label: "In Review", value: "InReview", color: "#0891b2" },
  { label: "Approved", value: "Approved", color: "#16a34a" },
  { label: "On Hold", value: "OnHold", color: "#f59e0b" },
  { label: "Sent Back to Ops", value: "SentBackToOps", color: "#f97316" },
  { label: "Exception", value: "Exception", color: "#dc2626" },
  { label: "Canceled", value: "Canceled", color: "#b91c1c" },
] satisfies ReadonlyArray<GenericSelectOption<string>>;

export const moveStatusChoices = [
  { label: "New", value: "New", color: "#3b82f6" },
  { label: "Assigned", value: "Assigned", color: "#16a34a" },
  { label: "In Transit", value: "InTransit", color: "#0891b2" },
  { label: "Completed", value: "Completed", color: "#15803d" },
  { label: "Canceled", value: "Canceled", color: "#b91c1c" },
] satisfies ReadonlyArray<GenericSelectOption<MoveStatus>>;

export const stopStatusChoices = [
  { label: "New", value: "New", color: "#3b82f6" },
  { label: "In Transit", value: "InTransit", color: "#0891b2" },
  { label: "Completed", value: "Completed", color: "#15803d" },
  { label: "Canceled", value: "Canceled", color: "#b91c1c" },
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

export const formulaTypeChoices: SelectOption[] = [
  { label: "Freight Charge", value: "FreightCharge" },
  { label: "Accessorial Charge", value: "AccessorialCharge" },
];

export const equipmentClassChoices = [
  { value: "Tractor", label: "Tractor", color: "#15803d" },
  { value: "Trailer", label: "Trailer", color: "#7e22ce" },
  { value: "Container", label: "Container", color: "#dc2626" },
  { value: "Other", label: "Other", color: "#f59e0b" },
] satisfies ReadonlyArray<GenericSelectOption<EquipmentClass>>;

export const fieldSensitivityChoices = [
  { value: "public", label: "Public", color: "#15803d", variant: "active" },
  { value: "internal", label: "Internal", color: "#0ea5e9", variant: "info" },
  {
    value: "restricted",
    label: "Restricted",
    color: "#f59e0b",
    variant: "warning",
  },
  {
    value: "confidential",
    label: "Confidential",
    color: "#dc2626",
    variant: "inactive",
  },
] satisfies ReadonlyArray<GenericSelectOption<FieldSensitivity>>;

export const coreResponsibilityChoices = [
  { value: "Billing", label: "Billing", color: "#0ea5e9" },
  { value: "Operations", label: "Operations", color: "#16a34a" },
  { value: "Finance", label: "Finance", color: "#a855f7" },
  { value: "Leadership", label: "Leadership", color: "#f59e0b" },
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
  { value: "Available", label: "Available", color: "#15803d" },
  {
    value: "OutOfService",
    label: "Out of Service",
    color: "#b91c1c",
  },
  {
    value: "AtMaintenance",
    label: "At Maintenance",
    color: "#7e22ce",
  },
  { value: "Sold", label: "Sold", color: "#f59e0b" },
] satisfies ReadonlyArray<GenericSelectOption<EquipmentStatus>>;

export const fieldTypeChoices = [
  { value: "text", label: "Text", color: "#3b82f6" },
  { value: "number", label: "Number", color: "#10b981" },
  { value: "date", label: "Date", color: "#8b5cf6" },
  { value: "boolean", label: "Boolean", color: "#f59e0b" },
  { value: "select", label: "Select", color: "#ef4444" },
  { value: "multiSelect", label: "Multi-Select", color: "#ec4899" },
] satisfies ReadonlyArray<GenericSelectOption<FieldType>>;

export const accessorialChargeMethodChoices = [
  {
    value: "Flat",
    label: "Flat",
    color: "#15803d",
    description: "Fixed amount regardless of shipment details",
  },
  {
    value: "PerUnit",
    label: "Per Unit",
    color: "#7e22ce",
    description: "Rate multiplied by units",
  },
  {
    value: "Percentage",
    label: "Percentage",
    color: "#f59e0b",
    description: "Percentage of linehaul, freight charges, or declared value",
  },
] satisfies ReadonlyArray<GenericSelectOption<AccessorialChargeMethod>>;

export const rateUnitChoices = [
  { value: "Mile", label: "Mile", color: "#15803d" },
  { value: "Hour", label: "Hour", color: "#7e22ce" },
  { value: "Day", label: "Day", color: "#f59e0b" },
  { value: "Stop", label: "Stop", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<RateUnit>>;

export const workerTypeChoices = [
  { value: "Employee", label: "Employee", color: "#15803d" },
  { value: "Contractor", label: "Contractor", color: "#7e22ce" },
] satisfies ReadonlyArray<GenericSelectOption<WorkerType>>;

export const genderChoices = [
  { value: "Male", label: "Male" },
  { value: "Female", label: "Female" },
] satisfies ReadonlyArray<GenericSelectOption<Gender>>;

export const driverTypeChoices = [
  { value: "Local", label: "Local", color: "#3b82f6" },
  { value: "Regional", label: "Regional", color: "#10b981" },
  { value: "OTR", label: "OTR (Over the Road)", color: "#f59e0b" },
  { value: "Team", label: "Team", color: "#8b5cf6" },
] satisfies ReadonlyArray<GenericSelectOption<DriverType>>;

export const cdlClassChoices = [
  { value: "A", label: "Class A", color: "#15803d" },
  { value: "B", label: "Class B", color: "#3b82f6" },
  { value: "C", label: "Class C", color: "#f59e0b" },
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
  { value: "Compliant", label: "Compliant", color: "#15803d" },
  { value: "NonCompliant", label: "Non-Compliant", color: "#dc2626" },
  { value: "Pending", label: "Pending", color: "#f59e0b" },
] satisfies ReadonlyArray<GenericSelectOption<ComplianceStatus>>;

export const ptoStatusChoices = [
  { value: "Requested", label: "Requested", color: "#3b82f6" },
  { value: "Approved", label: "Approved", color: "#15803d" },
  { value: "Rejected", label: "Rejected", color: "#dc2626" },
  { value: "Cancelled", label: "Cancelled", color: "#a3a3a3" },
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
  { value: "Asset", label: "Asset", color: "#3b82f6" },
  { value: "Liability", label: "Liability", color: "#ef4444" },
  { value: "Equity", label: "Equity", color: "#8b5cf6" },
  { value: "Revenue", label: "Revenue", color: "#15803d" },
  { value: "CostOfRevenue", label: "Cost of Revenue", color: "#f59e0b" },
  { value: "Expense", label: "Expense", color: "#dc2626" },
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
  { value: "Disabled", label: "Disabled", color: "#6b7280" },
  { value: "WarnOnly", label: "Warn Only", color: "#f59e0b" },
  { value: "BlockPosting", label: "Block Posting", color: "#dc2626" },
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
  { value: "I", label: "I - High Danger", color: "#dc2626" },
  { value: "II", label: "II - Medium Danger", color: "#f59e0b" },
  { value: "III", label: "III - Low Danger", color: "#15803d" },
] satisfies ReadonlyArray<GenericSelectOption<PackingGroup>>;

export const segregationTypeChoices = [
  { value: "Prohibited", label: "Prohibited", color: "#dc2626" },
  { value: "Separated", label: "Separated", color: "#15803d" },
  { value: "Distance", label: "Distance", color: "#7e22ce" },
  { value: "Barrier", label: "Barrier", color: "#f59e0b" },
] satisfies ReadonlyArray<GenericSelectOption<SegregationType>>;

export const segregationDistanceUnitChoices = [
  { value: "FT", label: "Feet", color: "#15803d" },
  { value: "M", label: "Meters", color: "#7e22ce" },
  { value: "IN", label: "Inches", color: "#f59e0b" },
  { value: "CM", label: "Centimeters", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<SegregationDistanceUnit>>;

export const fiscalYearStatusChoices = [
  { value: "Draft", label: "Draft", color: "#9333ea" },
  { value: "Open", label: "Open", color: "#16a34a" },
  { value: "Closed", label: "Closed", color: "#dc2626" },
  { value: "Locked", label: "Locked", color: "#f59e0b" },
] satisfies ReadonlyArray<GenericSelectOption<FiscalYearStatus>>;

export const fiscalPeriodStatusChoices = [
  { value: "Open", label: "Open", color: "#16a34a" },
  { value: "Closed", label: "Closed", color: "#dc2626" },
  { value: "Locked", label: "Locked", color: "#f59e0b" },
] satisfies ReadonlyArray<GenericSelectOption<FiscalPeriodStatus>>;

export const periodTypeChoices = [
  { value: "Month", label: "Month", color: "#2563eb" },
  { value: "Quarter", label: "Quarter", color: "#4b0082" },
  { value: "Year", label: "Year", color: "#16a34a" },
] satisfies ReadonlyArray<GenericSelectOption<PeriodType>>;

export const documentClassificationChoices = [
  { value: "Public", label: "Public", color: "#15803d" },
  { value: "Private", label: "Private", color: "#7e22ce" },
  { value: "Sensitive", label: "Sensitive", color: "#b91c1c" },
  { value: "Regulatory", label: "Regulatory", color: "#f59e0b" },
] satisfies ReadonlyArray<GenericSelectOption<DocumentClassification>>;

export const documentCategoryChoices = [
  { value: "Shipment", label: "Shipment", color: "#15803d" },
  { value: "Worker", label: "Worker", color: "#7e22ce" },
  { value: "Regulatory", label: "Regulatory", color: "#f59e0b" },
  { value: "Profile", label: "Profile", color: "#0369a1" },
  { value: "Branding", label: "Branding", color: "#10b981" },
  { value: "Invoice", label: "Invoice", color: "#6495ed" },
  { value: "Contract", label: "Contract", color: "#0369a1" },
  { value: "Other", label: "Other", color: "#ec4899" },
] satisfies ReadonlyArray<GenericSelectOption<DocumentCategory>>;

export const locationCategoryTypeChoices = [
  { value: "Terminal", label: "Terminal", color: "#3b82f6" },
  { value: "Warehouse", label: "Warehouse", color: "#10b981" },
  {
    value: "DistributionCenter",
    label: "Distribution Center",
    color: "#8b5cf6",
  },
  { value: "TruckStop", label: "Truck Stop", color: "#f59e0b" },
  { value: "RestArea", label: "Rest Area", color: "#6b7280" },
  { value: "CustomerLocation", label: "Customer Location", color: "#ec4899" },
  { value: "Port", label: "Port", color: "#0ea5e9" },
  { value: "RailYard", label: "Rail Yard", color: "#a855f7" },
  {
    value: "MaintenanceFacility",
    label: "Maintenance Facility",
    color: "#ef4444",
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
  { value: "OperationalHold", label: "Operational", color: "#3b82f6" },
  { value: "ComplianceHold", label: "Compliance", color: "#f59e0b" },
  { value: "CustomerHold", label: "Customer", color: "#8b5cf6" },
  { value: "FinanceHold", label: "Finance", color: "#15803d" },
] satisfies ReadonlyArray<GenericSelectOption<HoldType>>;

export const holdSeverityChoices = [
  { value: "Informational", label: "Informational", color: "#3b82f6" },
  { value: "Advisory", label: "Advisory", color: "#f59e0b" },
  { value: "Blocking", label: "Blocking", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<HoldSeverity>>;

export const transferScheduleChoices = [
  { value: "Continuous", label: "Continuous", color: "#15803d" },
  { value: "Hourly", label: "Hourly", color: "#3b82f6" },
  { value: "Daily", label: "Daily", color: "#7e22ce" },
  { value: "Weekly", label: "Weekly", color: "#f59e0b" },
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
  { value: "Ignore", label: "Ignore", color: "#6b7280" },
  { value: "Warn", label: "Warn", color: "#f59e0b" },
  { value: "RequireReview", label: "Require Review", color: "#3b82f6" },
  { value: "Block", label: "Block", color: "#dc2626" },
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

export const paymentTermChoices = [
  { value: "Net10", label: "Net 10", color: "#0ea5e9" },
  { value: "DueOnReceipt", label: "Due on Receipt", color: "#15803d" },
  { value: "Net15", label: "Net 15", color: "#3b82f6" },
  { value: "Net30", label: "Net 30", color: "#7e22ce" },
  { value: "Net45", label: "Net 45", color: "#f59e0b" },
  { value: "Net60", label: "Net 60", color: "#ef4444" },
  { value: "Net90", label: "Net 90", color: "#6b7280" },
] satisfies ReadonlyArray<GenericSelectOption<PaymentTerm>>;

export const adjustmentEligibilityPolicyChoices = [
  { value: "Disallow", label: "Disallow", color: "#dc2626" },
  { value: "AllowWithApproval", label: "Allow With Approval", color: "#3b82f6" },
  { value: "AllowWithoutApproval", label: "Allow Without Approval", color: "#f59e0b" },
] satisfies ReadonlyArray<GenericSelectOption<AdjustmentEligibilityPolicy>>;

export const adjustmentAccountingDatePolicyChoices = [
  {
    value: "UseOriginalIfOpenElseNextOpen",
    label: "Use Original If Open Else Next Open",
  },
  { value: "AlwaysNextOpen", label: "Always Next Open" },
] satisfies ReadonlyArray<GenericSelectOption<AdjustmentAccountingDatePolicy>>;

export const closedPeriodAdjustmentPolicyChoices = [
  { value: "Disallow", label: "Disallow", color: "#dc2626" },
  { value: "RequireReopen", label: "Require Reopen", color: "#f59e0b" },
  {
    value: "PostInNextOpenPeriodWithApproval",
    label: "Post In Next Open Period With Approval",
    color: "#3b82f6",
  },
] satisfies ReadonlyArray<GenericSelectOption<ClosedPeriodAdjustmentPolicy>>;

export const requirementPolicyChoices = [
  { value: "Optional", label: "Optional", color: "#6b7280" },
  { value: "Required", label: "Required", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<RequirementPolicy>>;

export const adjustmentAttachmentPolicyChoices = [
  { value: "Optional", label: "Optional", color: "#6b7280" },
  {
    value: "RequiredForCreditOrWriteOff",
    label: "Required For Credit Or Write-Off",
    color: "#f59e0b",
  },
  { value: "RequiredForAll", label: "Required For All", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<AdjustmentAttachmentPolicy>>;

export const invoiceAdjustmentSupportingDocumentPolicyChoices = [
  { value: "Inherit", label: "Inherit Organization Default", color: "#3b82f6" },
  { value: "Required", label: "Required", color: "#dc2626" },
  { value: "Optional", label: "Optional", color: "#6b7280" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceAdjustmentSupportingDocumentPolicy>>;

export const approvalPolicyChoices = [
  { value: "None", label: "None", color: "#6b7280" },
  { value: "Always", label: "Always", color: "#dc2626" },
  { value: "AmountThreshold", label: "Amount Threshold", color: "#3b82f6" },
] satisfies ReadonlyArray<GenericSelectOption<ApprovalPolicy>>;

export const writeOffApprovalPolicyChoices = [
  { value: "Disallow", label: "Disallow", color: "#dc2626" },
  { value: "AlwaysRequireApproval", label: "Always Require Approval", color: "#3b82f6" },
  {
    value: "RequireApprovalAboveThreshold",
    label: "Require Approval Above Threshold",
    color: "#f59e0b",
  },
] satisfies ReadonlyArray<GenericSelectOption<WriteOffApprovalPolicy>>;

export const replacementInvoiceReviewPolicyChoices = [
  { value: "NoAdditionalReview", label: "No Additional Review", color: "#6b7280" },
  {
    value: "RequireReviewWhenEconomicTermsChange",
    label: "Require Review When Economic Terms Change",
    color: "#3b82f6",
  },
  { value: "AlwaysRequireReview", label: "Always Require Review", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<ReplacementInvoiceReviewPolicy>>;

export const customerCreditBalancePolicyChoices = [
  { value: "Disallow", label: "Disallow", color: "#dc2626" },
  { value: "AllowUnappliedCredit", label: "Allow Unapplied Credit", color: "#3b82f6" },
] satisfies ReadonlyArray<GenericSelectOption<CustomerCreditBalancePolicy>>;

export const overCreditPolicyChoices = [
  { value: "Block", label: "Block", color: "#dc2626" },
  { value: "AllowWithApproval", label: "Allow With Approval", color: "#3b82f6" },
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
  { value: "Never", label: "Never", color: "#15803d" },
  { value: "Pickup", label: "Pickup", color: "#7e22ce" },
  { value: "Delivery", label: "Delivery", color: "#f59e0b" },
  { value: "PickupDelivery", label: "Pickup/Delivery", color: "#0369a1" },
  {
    value: "AllExceptShipper",
    label: "All Except Shipper",
    color: "#10b981",
  },
] satisfies ReadonlyArray<GenericSelectOption<ServiceIncidentType>>;

export const autoAssignmentStrategyChoices = [
  { value: "Proximity", label: "Proximity", color: "#0369a1" },
  { value: "Availability", label: "Availability", color: "#15803d" },
  { value: "LoadBalancing", label: "Load Balancing", color: "#ec4899" },
] satisfies ReadonlyArray<GenericSelectOption<AutoAssignmentStrategy>>;

export const complianceEnforcementLevelChoices = [
  { value: "Warning", label: "Warning", color: "#f59e0b" },
  { value: "Block", label: "Block", color: "#b91c1c" },
  { value: "Audit", label: "Audit", color: "#7e22ce" },
] satisfies ReadonlyArray<GenericSelectOption<ComplianceEnforcementLevel>>;

export const billingCycleTypeChoices = [
  { value: "Immediate", label: "Immediate" },
  { value: "Daily", label: "Daily" },
  { value: "Weekly", label: "Weekly" },
  { value: "BiWeekly", label: "Bi-Weekly" },
  { value: "Monthly", label: "Monthly" },
  { value: "Quarterly", label: "Quarterly" },
  { value: "PerShipment", label: "Per Shipment" },
] satisfies ReadonlyArray<GenericSelectOption<BillingCycleType>>;

export const customerPaymentTermChoices = [
  { value: "DueOnReceipt", label: "Due on Receipt", color: "#15803d" },
  { value: "Net10", label: "Net 10", color: "#0ea5e9" },
  { value: "Net15", label: "Net 15", color: "#3b82f6" },
  { value: "Net30", label: "Net 30", color: "#7e22ce" },
  { value: "Net45", label: "Net 45", color: "#f59e0b" },
  { value: "Net60", label: "Net 60", color: "#ef4444" },
  { value: "Net90", label: "Net 90", color: "#6b7280" },
] satisfies ReadonlyArray<GenericSelectOption<CustomerPaymentTerm>>;

export const creditStatusChoices = [
  { value: "Active", label: "Active", color: "#15803d" },
  { value: "Warning", label: "Warning", color: "#f59e0b" },
  { value: "Hold", label: "Hold", color: "#dc2626" },
  { value: "Suspended", label: "Suspended", color: "#6b7280" },
  { value: "Review", label: "Review", color: "#7e22ce" },
] satisfies ReadonlyArray<GenericSelectOption<CreditStatus>>;

export const invoiceMethodChoices = [
  { value: "Individual", label: "Individual" },
  { value: "Summary", label: "Summary" },
  { value: "SummaryWithDetail", label: "Summary with Detail" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceMethod>>;

export const consolidationGroupByChoices = [
  { value: "None", label: "None" },
  { value: "Location", label: "Location" },
  { value: "PONumber", label: "PO Number" },
  { value: "BOL", label: "BOL" },
  { value: "Division", label: "Division" },
] satisfies ReadonlyArray<GenericSelectOption<ConsolidationGroupBy>>;

export const invoiceNumberFormatChoices = [
  { value: "Default", label: "Default" },
  { value: "CustomPrefix", label: "Custom Prefix" },
  { value: "POBased", label: "PO Based" },
] satisfies ReadonlyArray<GenericSelectOption<InvoiceNumberFormat>>;

export const commentTypeChoices = [
  { value: "Internal", label: "Internal", color: "#6b7280" },
  { value: "Dispatch", label: "Dispatch", color: "#3b82f6" },
  { value: "DriverUpdate", label: "Driver Update", color: "#0891b2" },
  { value: "PickupInstruction", label: "Pickup Instruction", color: "#16a34a" },
  { value: "DeliveryInstruction", label: "Delivery Instruction", color: "#15803d" },
  { value: "StatusUpdate", label: "Status Update", color: "#6366f1" },
  { value: "Exception", label: "Exception", color: "#dc2626" },
  { value: "CustomerUpdate", label: "Customer Update", color: "#a855f7" },
  { value: "Appointment", label: "Appointment", color: "#f59e0b" },
  { value: "Document", label: "Document", color: "#64748b" },
  { value: "Billing", label: "Billing", color: "#0d9488" },
  { value: "Compliance", label: "Compliance", color: "#db2777" },
] satisfies ReadonlyArray<GenericSelectOption<CommentType>>;

export const commentVisibilityChoices = [
  { value: "Internal", label: "Internal", color: "#6b7280" },
  { value: "Operations", label: "Operations", color: "#3b82f6" },
  { value: "Customer", label: "Customer", color: "#a855f7" },
  { value: "Driver", label: "Driver", color: "#0891b2" },
  { value: "Accounting", label: "Accounting", color: "#0d9488" },
] satisfies ReadonlyArray<GenericSelectOption<CommentVisibility>>;

export const commentPriorityChoices = [
  { value: "Low", label: "Low", color: "#9ca3af" },
  { value: "Normal", label: "Normal", color: "#3b82f6" },
  { value: "High", label: "High", color: "#f59e0b" },
  { value: "Urgent", label: "Urgent", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<CommentPriority>>;

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
] satisfies ReadonlyArray<GenericSelectOption<InvoiceStatus>>;

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
