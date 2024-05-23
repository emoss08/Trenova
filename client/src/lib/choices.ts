import { IChoiceProps } from "@/types";
import { TableOptionProps } from "@/types/tables";
import { CircleIcon, MinusCircledIcon } from "@radix-ui/react-icons";

/** Type for Account Type Choices */
export type AccountTypeChoiceProps =
  | "Asset"
  | "Liability"
  | "Equity"
  | "Revenue"
  | "Expense";

export const accountTypeChoices = [
  { value: "Asset", label: "Asset" },
  { value: "Liability", label: "Liability" },
  { value: "Equity", label: "Equity" },
  { value: "Revenue", label: "Revenue" },
  { value: "Expense", label: "Expense" },
] satisfies ReadonlyArray<IChoiceProps<AccountTypeChoiceProps>>;

export const tableAccountTypeChoices = [
  { value: "ASSET", label: "Asset" },
  { value: "LIABILITY", label: "Liability" },
  { value: "EQUITY", label: "Equity" },
  { value: "REVENUE", label: "Revenue" },
  { value: "EXPENSE", label: "Expense" },
] satisfies TableOptionProps[];

/** Type for Cash Flow Type Choices */
export type CashFlowTypeChoiceProps =
  | "Operating"
  | "Investing"
  | "Financing"
  | "";

export const cashFlowTypeChoices = [
  { value: "Operating", label: "Operating" },
  { value: "Investing", label: "Investing" },
  { value: "Financing", label: "Financing" },
] satisfies ReadonlyArray<IChoiceProps<CashFlowTypeChoiceProps>>;

/** Type for Account Sub Type Choices */
export type AccountSubTypeChoiceProps =
  | "CurrentAsset"
  | "FixedAsset"
  | "OtherAsset"
  | "CurrentLiability"
  | "LongTermLiability"
  | "Equity"
  | "Revenue"
  | "CostOfGoodsSold"
  | "Expense"
  | "OtherIncome"
  | "OtherExpense";

export const accountSubTypeChoices = [
  { value: "CurrentAsset", label: "Current Asset" },
  { value: "FixedAsset", label: "Fixed Asset" },
  { value: "OtherAsset", label: "Other Asset" },
  { value: "CurrentLiability", label: "Current Liability" },
  { value: "LongTermLiability", label: "Long Term Liability" },
  { value: "Equity", label: "Equity" },
  { value: "Revenue", label: "Revenue" },
  { value: "CostOfGoodsSold", label: "Cost of Goods Sold" },
  { value: "Expense", label: "Expense" },
  { value: "OtherIncome", label: "Other Income" },
  { value: "OtherExpense", label: "Other Expense" },
] satisfies ReadonlyArray<IChoiceProps<AccountSubTypeChoiceProps>>;

/** Type for Account Classification Choices */
export type AccountClassificationChoiceProps =
  | "AccountClassificationBank"
  | "AccountClassificationCash"
  | "AccountClassificationAR"
  | "AccountClassificationAP"
  | "AccountClassificationINV"
  | "AccountClassificationOCA"
  | "AccountClassificationFA";

export const accountClassificationChoices = [
  { value: "AccountClassificationBank", label: "Bank" },
  { value: "AccountClassificationCash", label: "Cash" },
  { value: "AccountClassificationAR", label: "Accounts Receivable" },
  { value: "AccountClassificationAP", label: "Accounts Payable" },
  { value: "AccountClassificationINV", label: "Inventory" },
  { value: "AccountClassificationOCA", label: "Other Current Asset" },
  { value: "AccountClassificationFA", label: "Fixed Asset" },
] satisfies ReadonlyArray<IChoiceProps<AccountClassificationChoiceProps>>;

/** Types for Job Function Choices */
export type JobFunctionChoiceProps =
  | "MANAGER"
  | "MANAGEMENT_TRAINEE"
  | "SUPERVISOR"
  | "DISPATCHER"
  | "BILLING"
  | "FINANCE"
  | "SAFETY"
  | "SYS_ADMIN"
  | "TEST";

export const jobFunctionChoices = [
  { value: "MANAGER", label: "Manager" },
  { value: "MANAGEMENT_TRAINEE", label: "Management Trainee" },
  { value: "SUPERVISOR", label: "Supervisor" },
  { value: "DISPATCHER", label: "Dispatcher" },
  { value: "BILLING", label: "Billing" },
  { value: "FINANCE", label: "Finance" },
  { value: "SAFETY", label: "Safety" },
  { value: "SYS_ADMIN", label: "System Administrator" },
  { value: "TEST", label: "Test Job Function" },
] satisfies ReadonlyArray<IChoiceProps<JobFunctionChoiceProps>>;

export type ServiceIncidentControlChoiceProps =
  | "Never"
  | "Pickup"
  | "Delivery"
  | "PickupAndDelivery"
  | "AllExceptShipper";

export enum ServiceIncidentControlEnum {
  Never = "Never",
  Pickup = "Pickup",
  Delivery = "Delivery",
  PickupAndDelivery = "PickupAndDelivery",
  AllExceptShipper = "AllExceptShipper",
}

export const serviceIncidentControlChoices = [
  { value: "Never", label: "Never" },
  { value: "Pickup", label: "Pickup" },
  { value: "Delivery", label: "Delivery" },
  { value: "PickupAndDelivery", label: "Pickup and Delivery" },
  { value: "AllExceptShipper", label: "All except shipper" },
] satisfies ReadonlyArray<IChoiceProps<ServiceIncidentControlChoiceProps>>;

/** Type for Date Format Choices */
export type DateFormatChoiceProps =
  | "InvoiceDateFormatMDY"
  | "InvoiceDateFormatDMY"
  | "InvoiceDateFormatYMD"
  | "InvoiceDateFormatYDM"
  | "";

export const dateFormatChoices = [
  { value: "InvoiceDateFormatMDY", label: "01/02/2006" },
  { value: "InvoiceDateFormatDMY", label: "02/01/2006" },
  { value: "InvoiceDateFormatYMD", label: "2006/02/01" },
  { value: "InvoiceDateFormatYDM", label: "2006/01/02" },
] satisfies ReadonlyArray<IChoiceProps<DateFormatChoiceProps>>;

/** Type for Route Avoidance Choices */
type RouteAvoidanceChoiceProps = "tolls" | "highways" | "ferries";

export const routeAvoidanceChoices = [
  { value: "tolls", label: "Tolls" },
  { value: "highways", label: "Highways" },
  { value: "ferries", label: "Ferries" },
] satisfies ReadonlyArray<IChoiceProps<RouteAvoidanceChoiceProps>>;

/** Type for Route Model Choices */
export type RouteModelChoiceProps = "BestGuess" | "Optimistic" | "Pessimistic";

export const routeModelChoices = [
  { value: "BestGuess", label: "Best Guess" },
  { value: "Optimistic", label: "Optimistic" },
  { value: "Pessimistic", label: "Pessimistic" },
] satisfies ReadonlyArray<IChoiceProps<RouteModelChoiceProps>>;

/** Type for Route Distance Unit Choices */
export type RouteDistanceUnitProps = "UnitsMetric" | "UnitsImperial";

export const routeDistanceUnitChoices = [
  { value: "UnitsMetric", label: "Metric" },
  { value: "UnitsImperial", label: "Imperial" },
] satisfies ReadonlyArray<IChoiceProps<RouteDistanceUnitProps>>;

/** Type for Distance Method Choices */
export type DistanceMethodChoiceProps = "Google" | "Trenova" | "PCMiler";

export const distanceMethodChoices = [
  { value: "Google", label: "Google" },
  { value: "PCMiler", label: "PCMiler" },
  { value: "Trenova", label: "Trenova" },
] satisfies ReadonlyArray<IChoiceProps<DistanceMethodChoiceProps>>;

/** Type for Email Protocol Choices */
export type EmailProtocolChoiceProps = "TLS" | "SSL" | "UNENCRYPTED";

export const emailProtocolChoices = [
  { value: "TLS", label: "TLS" },
  { value: "SSL", label: "SSL" },
  { value: "UNENCRYPTED", label: "Unencrypted" },
] satisfies ReadonlyArray<IChoiceProps<EmailProtocolChoiceProps>>;

/** Type for Feasibility Operator Choices */
export type FeasibilityOperatorChoiceProps =
  | "Eq"
  | "Ne"
  | "Gt"
  | "Gte"
  | "Lt"
  | "Lte";

export const feasibilityOperatorChoices = [
  { value: "Eq", label: "Equals" },
  { value: "Ne", label: "Not Equals" },
  { value: "Gt", label: "Greater Than" },
  { value: "Gte", label: "Greater Than or Equal To" },
  { value: "Lt", label: "Less Than" },
  { value: "Lte", label: "Less Than or Equal To" },
] satisfies ReadonlyArray<IChoiceProps<FeasibilityOperatorChoiceProps>>;

/** Type for Equipment Class Choices */
export type EquipmentClassChoiceProps =
  | "Undefined"
  | "Car"
  | "Van"
  | "Pickup"
  | "Straight"
  | "Tractor"
  | "Trailer"
  | "Container"
  | "Chassis"
  | "Other";

export const equipmentClassChoices = [
  { value: "Undefined", label: "Undefined" },
  { value: "Car", label: "Car" },
  { value: "Van", label: "Van" },
  { value: "Pickup", label: "Pickup" },
  { value: "Straight", label: "Straight" },
  { value: "Tractor", label: "Tractor" },
  { value: "Trailer", label: "Trailer" },
  { value: "Container", label: "Container" },
  { value: "Chassis", label: "Chassis" },
  { value: "Other", label: "Other" },
] satisfies ReadonlyArray<IChoiceProps<EquipmentClassChoiceProps>>;

/** Type for Unit of Measure Choices */
export type UnitOfMeasureChoiceProps =
  | "Pallet"
  | "Tote"
  | "Drum"
  | "Cylinder"
  | "Case"
  | "Ampule"
  | "Bag"
  | "Bottle"
  | "Pail"
  | "Pieces"
  | "IsoTank";

export const UnitOfMeasureChoices = [
  { value: "Pallet", label: "Pallet" },
  { value: "Tote", label: "Tote" },
  { value: "Drum", label: "Drum" },
  { value: "Cylinder", label: "Cylinder" },
  { value: "Case", label: "Case" },
  { value: "Ampule", label: "Ampule" },
  { value: "Bag", label: "Bag" },
  { value: "Bottle", label: "Bottle" },
  { value: "Pail", label: "Pail" },
  { value: "Pieces", label: "Pieces" },
  { value: "IsoTank", label: "ISO Tank" },
] satisfies ReadonlyArray<IChoiceProps<UnitOfMeasureChoiceProps>>;

/** Type for Hazardous Class Choices */
export type PackingGroupChoiceProps = "I" | "II" | "III";

export const packingGroupChoices = [
  { value: "I", label: "I" },
  { value: "II", label: "II" },
  { value: "III", label: "III" },
] satisfies ReadonlyArray<IChoiceProps<PackingGroupChoiceProps>>;

/** Type for Hazardous Class Choices */
export type HazardousClassChoiceProps =
  | "HazardClass1And1"
  | "HazardClass1And2"
  | "HazardClass1And3"
  | "HazardClass1And4"
  | "HazardClass1And5"
  | "HazardClass1And6"
  | "HazardClass2And1"
  | "HazardClass2And2"
  | "HazardClass2And3"
  | "HazardClass3"
  | "HazardClass4And1"
  | "HazardClass4And2"
  | "HazardClass4And3"
  | "HazardClass5And1"
  | "HazardClass5And2"
  | "HazardClass6And1"
  | "HazardClass6And2"
  | "HazardClass7"
  | "HazardClass8"
  | "HazardClass9";

export const hazardousClassChoices = [
  { value: "HazardClass1And1", label: "Division 1.1: Mass Explosive Hazard" },
  { value: "HazardClass1And2", label: "Division 1.2: Projection Hazard" },
  {
    value: "HazardClass1And3",
    label: "Division 1.3: Fire and/or Minor Blast/Minor Projection Hazard",
  },
  { value: "HazardClass1And4", label: "Division 1.4: Minor Explosion Hazard" },
  {
    value: "HazardClass1And5",
    label: "Division 1.5: Very Insensitive With Mass Explosion Hazard",
  },
  {
    value: "HazardClass1And6",
    label: "Division 1.6: Extremely Insensitive; No Mass Explosion Hazard",
  },
  { value: "HazardClass2And1", label: "Division 2.1: Flammable Gases" },
  { value: "HazardClass2And2", label: "Division 2.2: Non-Flammable Gases" },
  { value: "HazardClass2And3", label: "Division 2.3: Poisonous Gases" },
  { value: "HazardClass3", label: "Division 3: Flammable Liquids" },
  { value: "HazardClass4And1", label: "Division 4.1: Flammable Solids" },
  {
    value: "HazardClass4And2",
    label: "Division 4.2: Spontaneously Combustible Solids",
  },
  { value: "HazardClass4And3", label: "Division 4.3: Dangerous When Wet" },
  { value: "HazardClass5And1", label: "Division 5.1: Oxidizing Substances" },
  { value: "HazardClass5And2", label: "Division 5.2: Organic Peroxides" },
  { value: "HazardClass6And1", label: "Division 6.1: Toxic Substances" },
  { value: "HazardClass6And2", label: "Division 6.2: Infectious Substances" },
  { value: "HazardClass7", label: "Division 7: Radioactive Material" },
  { value: "HazardClass8", label: "Division 8: Corrosive Substances" },
  {
    value: "HazardClass9",
    label: "Division 9: Miscellaneous Hazardous Substances and Articles",
  },
] satisfies ReadonlyArray<IChoiceProps<HazardousClassChoiceProps>>;

export function getHazardousClassLabel(
  value: HazardousClassChoiceProps,
): string {
  return (
    hazardousClassChoices.find((choice) => choice.value === value)?.label ?? ""
  );
}

/* Transaction Type Choice Type */
export type TransactionTypeChoiceType = "REVENUE" | "EXPENSE";
export const transactionTypeChoices = [
  { value: "REVENUE", label: "Revenue" },
  { value: "EXPENSE", label: "Expense" },
] satisfies ReadonlyArray<IChoiceProps<TransactionTypeChoiceType>>;

/* Transaction Status Choice Type */
export type TransactionStatusChoiceType =
  | "PENDING"
  | "COMPLETED"
  | "FAILED"
  | "PENDING_RECON";

export const transactionStatusChoices = [
  { value: "PENDING", label: "Pending" },
  { value: "PENDING_RECON", label: "Pending Reconciliation" },
  { value: "COMPLETED", label: "Completed" },
  { value: "FAILED", label: "Failed" },
] satisfies ReadonlyArray<IChoiceProps<TransactionStatusChoiceType>>;

/* Threshold Action Choice Type */
export type ThresholdActionChoiceType = "HALT" | "WARN";
export const thresholdActionChoices = [
  { value: "HALT", label: "Halt" },
  { value: "WARN", label: "Warn" },
] satisfies ReadonlyArray<IChoiceProps<ThresholdActionChoiceType>>;

/** Automatic Journal Entry Choices */
export type AutomaticJournalEntryChoiceType =
  | "OnShipmentBill"
  | "OnReceiptOfPayment"
  | "OnExpenseRecognition"
  | "";

export const automaticJournalEntryChoices = [
  {
    value: "OnShipmentBill",
    label: "Auto create entry when shipment is billed",
  },
  {
    value: "OnReceiptOfPayment",
    label: "Auto create entry on receipt of payment",
  },
  {
    value: "OnExpenseRecognition",
    label: "Auto create entry when an expense is recognized",
  },
] satisfies ReadonlyArray<IChoiceProps<AutomaticJournalEntryChoiceType>>;

/**
 * Returns status choices for a select input.
 */
type TStatusChoiceProps = "A" | "I";

/**
 * Returns status choices for a select input.
 * @returns An array of status choices.
 */
export const statusChoices = [
  { value: "A", label: "Active", color: "#15803d" },
  { value: "I", label: "Inactive", color: "#b91c1c" },
] satisfies ReadonlyArray<IChoiceProps<TStatusChoiceProps>>;

/** Entry methods for shipments */
export type EntryMethodChoiceProps = "MANUAL" | "EDI" | "API";

export const entryMethodChoices = [
  { value: "MANUAL", label: "Manual" },
  { value: "EDI", label: "EDI" },
  { value: "API", label: "API" },
] satisfies ReadonlyArray<IChoiceProps<EntryMethodChoiceProps>>;

/** Returns status choices for a select input. */
export type ShipmentStatusChoiceProps = "N" | "P" | "C" | "H" | "B" | "V";

/**
 * Returns shipment status choices for a select input.
 * @returns An array of shipment status choices.
 */
export const shipmentStatusChoices = [
  { value: "N", label: "New", color: "#16a34a" },
  { value: "P", label: "In Progress", color: "#ca8a04" },
  { value: "C", label: "Completed", color: "#9333ea" },
  { value: "H", label: "On Hold", color: "#2563eb" },
  { value: "B", label: "Billed", color: "#0891b2" },
  { value: "V", label: "Voided", color: "#dc2626" },
] satisfies ReadonlyArray<IChoiceProps<ShipmentStatusChoiceProps>>;

export type ShipmentEntryMethodChoices = "MANUAL" | "EDI" | "API";

export const shipmentSourceChoices = [
  { value: "MANUAL", label: "Manual" },
  { value: "EDI", label: "EDI" },
  { value: "API", label: "API" },
] satisfies ReadonlyArray<IChoiceProps<ShipmentEntryMethodChoices>>;

/**
 * Returns code type choices for a select input.
 */
export type CodeTypeProps = "Voided" | "Cancelled";

/**
 * Returns code type choices for a select input.
 * @returns An array of code type choices.
 */
export const codeTypeChoices = [
  { value: "Voided", label: "Voided", color: "#b9ac1c" },
  {
    value: "Cancelled",
    label: "Cancelled",
    color: "#b91c1c",
  },
] satisfies ReadonlyArray<IChoiceProps<CodeTypeProps>>;

/**
 * Returns boolean yes & no choices for a select input.
 * @returns An array of yes & no choices.
 */
export const booleanStatusChoices: ReadonlyArray<IChoiceProps<boolean>> = [
  { value: true, label: "Active" },
  { value: false, label: "Inactive" },
];

/* Type for Database Actions */
export type DatabaseActionChoicesProps = "Insert" | "Update" | "Delete" | "All";

export const databaseActionChoices = [
  {
    value: "Insert",
    label: "Insert",
    color: "#15803d",
    description: "Only receive alerts for new records.",
  },
  {
    value: "Update",
    label: "Update",
    color: "#2563eb",
    description: "Only receive alerts for updated records.",
  },
  {
    value: "Delete",
    label: "Delete",
    color: "#b91c1c",
    description: "Only receive alerts for deleted records.",
  },
  {
    value: "All",
    label: "All",
    color: "#9c25eb",
    description: "Receive alerts for all actions.",
  },
] satisfies ReadonlyArray<IChoiceProps<DatabaseActionChoicesProps>>;

/* Type for Table Change Alert Source */
export type SourceChoicesProps = "Kafka" | "Database";

export const sourceChoices = [
  { value: "Kafka", label: "Kafka" },
  { value: "Database", label: "Database" },
] satisfies ReadonlyArray<IChoiceProps<SourceChoicesProps>>;

/** Type for RatingMethodW for Shipment */
export type RatingMethodChoiceProps = "F" | "PM" | "PS" | "PP" | "O";

export const ratingMethodChoices = [
  { value: "F", label: "Flat" },
  { value: "PM", label: "Per Mile" },
  { value: "PS", label: "Per Stop" },
  { value: "PP", label: "Per Pound" },
  { value: "O", label: "Other" },
] satisfies ReadonlyArray<IChoiceProps<RatingMethodChoiceProps>>;

/** Type for Shipment Stop Choices */
export type ShipmentStopChoices = "P" | "SP" | "SD" | "D" | "DO";

/** Returns shipment stop choices for a select input. */
export const shipmentStopChoices = [
  { value: "P", label: "Pickup" },
  { value: "SP", label: "Stop Pickup" },
  { value: "SD", label: "Stop Delivery" },
  { value: "D", label: "Delivery" },
  { value: "DO", label: "Drop Off" },
] satisfies ReadonlyArray<IChoiceProps<ShipmentStopChoices>>;

/** Type for Comment Type Severity */
export type SeverityChoiceProps = "High" | "Medium" | "Low";

/** Returns comment type severity choices for a select input. */
export const severityChoices = [
  { value: "High", label: "High", color: "#dc2626" },
  { value: "Medium", label: "Medium", color: "#2563eb" },
  { value: "Low", label: "Low", color: "#15803d" },
] satisfies ReadonlyArray<IChoiceProps<SeverityChoiceProps>>;

/**
 * Type for timezone choices
 */
export type TimezoneChoices =
  | "AmericaLosAngeles"
  | "AmericaDenver"
  | "AmericaChicago"
  | "AmericaNewYork";

/**
 * Returns timezone choices for a select input
 * @returns An array of timezone choices.
 */
export const timezoneChoices: ReadonlyArray<IChoiceProps<TimezoneChoices>> = [
  { value: "AmericaLosAngeles", label: "America/Los_Angeles" },
  { value: "AmericaDenver", label: "America/Denver" },
  { value: "AmericaChicago", label: "America/Chicago" },
  { value: "AmericaNewYork", label: "America/New_York" },
];

/**
 * Returns status choices when using TableFacetedFilters.
 * @returns An array of table faceted filter choices.
 */
export const tableStatusChoices = [
  {
    value: "A",
    label: "Active",
    icon: CircleIcon,
  },
  {
    value: "I",
    label: "Inactive",
    icon: MinusCircledIcon,
  },
] satisfies TableOptionProps[];

/**
 * Returns yes & no choices for a select input.
 * @returns An array of yes & no choices.
 */
export const yesAndNoChoices = [
  { value: "Y", label: "Yes" },
  { value: "N", label: "No" },
];

/**
 * Returns yes and no choices as boolean for a select input
 * @returns An array of yes and no choices as boolean.
 */
export const yesAndNoBooleanChoices = [
  { value: true, label: "Yes" },
  { value: false, label: "No" },
];

export type SegregationTypeChoiceProps = "O" | "X";
/**
 * Returns Segregation Type choices when using TableFacetedFilters.
 * @returns An array of table faceted filter choices.
 */
export const segregationTypeChoices = [
  {
    value: "O",
    label: "Allowed With Conditions",
    icon: CircleIcon,
  },
  {
    value: "X",
    label: "Not Allowed",
    icon: MinusCircledIcon,
  },
] satisfies TableOptionProps[];
