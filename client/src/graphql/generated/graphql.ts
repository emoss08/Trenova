/* eslint-disable */
/** Internal type. DO NOT USE DIRECTLY. */
type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
/** Internal type. DO NOT USE DIRECTLY. */
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
import type { DocumentTypeDecoration } from '@graphql-typed-document-node/core';
export type AccessorialMethod =
  | 'Flat'
  | 'PerUnit'
  | 'Percentage';

export type AccountCategory =
  | 'Asset'
  | 'CostOfRevenue'
  | 'Equity'
  | 'Expense'
  | 'Liability'
  | 'Revenue';

export type AddSettlementAdjustmentInput = {
  amountMinor: number;
  description: string;
  /** Optional pay code — the adjustment posts to the code's GL account when mapped. */
  payCodeId?: string | number | null | undefined;
  quantity?: string | null | undefined;
  rate?: string | null | undefined;
  settlementId: string | number;
};

export type AdjustEscrowAccountInput = {
  accountId: string | number;
  amountMinor: number;
  description: string;
  occurredDate?: number | null | undefined;
};

export type ApplyCustomerPaymentInput = {
  accountingDate: number;
  applications: Array<CustomerPaymentApplicationInput>;
  paymentId: string | number;
};

/**
 * Assigns a pay profile to a worker. Any currently-open assignment for the worker
 * is automatically ended on the new effective date — no manual cleanup needed.
 */
export type AssignPayProfileInput = {
  effectiveFrom: number;
  effectiveTo?: number | null | undefined;
  notes?: string | null | undefined;
  payProfileId: string | number;
  /** Optional per-component rate overrides for this driver. */
  rateOverrides?: Array<PayRateOverrideInput> | null | undefined;
  /** Defaults to 100. Use 50 for an even team split. */
  splitPercent?: string | null | undefined;
  workerId: string | number;
};

export type AssignmentStatus =
  | 'Canceled'
  | 'Completed'
  | 'InProgress'
  | 'New';

export type AttachPayEventsInput = {
  payEventIds: Array<string | number>;
  settlementId: string | number;
};

export type AuditCategory =
  | 'System'
  | 'User';

export type BillType =
  | 'CreditMemo'
  | 'DebitMemo'
  | 'Invoice';

export type BillingQueueAssignInput = {
  billerId: string | number;
};

export type BillingQueueExceptionReasonCode =
  | 'AccessorialDispute'
  | 'CustomerInformationError'
  | 'DuplicateCharge'
  | 'IncorrectRates'
  | 'MissingDocumentation'
  | 'MissingReferenceNumber'
  | 'Other'
  | 'RateNotOnFile'
  | 'ServiceFailure'
  | 'WeightDiscrepancy';

export type BillingQueueStatus =
  | 'Approved'
  | 'Canceled'
  | 'Exception'
  | 'InReview'
  | 'OnHold'
  | 'Posted'
  | 'ReadyForReview'
  | 'SentBackToOps';

export type BillingQueueUpdateStatusInput = {
  cancelReason?: string | null | undefined;
  exceptionNotes?: string | null | undefined;
  exceptionReasonCode?: BillingQueueExceptionReasonCode | null | undefined;
  reviewNotes?: string | null | undefined;
  status: BillingQueueStatus;
};

export type BulkSettlementActionInput = {
  action: BulkSettlementActionType;
  /** Required when action is MarkPaid. */
  paymentMethod?: string | null | undefined;
  paymentReference?: string | null | undefined;
  settlementIds: Array<string | number>;
};

export type BulkSettlementActionType =
  | 'Approve'
  | 'MarkPaid'
  | 'Post'
  | 'Submit';

export type BulkUpdateEquipmentTypeStatusInput = {
  equipmentTypeIds: Array<string | number>;
  status: EntityStatus;
};

export type CdlClass =
  | 'A'
  | 'B'
  | 'C';

export type ComplianceStatus =
  | 'Compliant'
  | 'NonCompliant'
  | 'Pending';

export type ConfigurationVisibility =
  | 'Private'
  | 'Public'
  | 'Shared';

export type CostBehavior =
  | 'Fixed'
  | 'Variable';

export type CostCategoryType =
  | 'Custom'
  | 'DriverBenefits'
  | 'DriverWages'
  | 'EquipmentPayments'
  | 'Fuel'
  | 'Insurance'
  | 'Maintenance'
  | 'Overhead'
  | 'PermitsLicenses'
  | 'Tires'
  | 'Tolls';

export type CostCategoryUpdateInput = {
  glAccountIds: Array<string | number>;
  id: string | number;
  isActive: boolean;
  overrideRatePerMile?: string | null | undefined;
  rateSource: CostRateSource;
  version: number;
};

export type CostRateSource =
  | 'Benchmark'
  | 'GLActual'
  | 'Override';

export type CostingControlInput = {
  fuelIndexId?: string | number | null | undefined;
  glActualsEnabled: boolean;
  glRollingMonths: number;
  includeDeadheadMiles: boolean;
  milesPerGallon: string;
  plannedMonthlyMiles?: number | null | undefined;
  targetMarginPercent?: string | null | undefined;
  useLiveFuelPrice: boolean;
  version: number;
};

export type CreateMyLoadCommentInput = {
  comment: string;
  shipmentId: string | number;
};

export type CreatePayCodeInput = {
  code: string;
  countsTowardGuarantee?: boolean | null | undefined;
  defaultAmountMinor?: number | null | undefined;
  description?: string | null | undefined;
  direction: PayCodeDirection;
  glAccountId?: string | number | null | undefined;
  name: string;
  taxable?: boolean | null | undefined;
};

export type CreatePayProfileInput = {
  classification: PayeeClassification;
  components: Array<PayProfileComponentInput>;
  currencyCode?: string | null | undefined;
  description?: string | null | undefined;
  guaranteedPeriodMinimumMinor?: number | null | undefined;
  name: string;
  perDiemDailyCapMinor?: number | null | undefined;
  perDiemRatePerMile?: string | null | undefined;
  status?: EntityStatus | null | undefined;
};

export type CreateRecurringDeductionInput = {
  amountMinor: number;
  currencyCode?: string | null | undefined;
  description: string;
  endDate?: number | null | undefined;
  escrowAccountId?: string | number | null | undefined;
  /**
   * When true and no escrow account is given, the deduction links to the driver's
   * active escrow account and posts as an escrow contribution.
   */
  escrowContribution?: boolean | null | undefined;
  frequency?: RecurringDeductionFrequency | null | undefined;
  payCodeId: string | number;
  startDate: number;
  totalCapMinor?: number | null | undefined;
  workerId: string | number;
};

export type CreateRecurringEarningInput = {
  amountMinor: number;
  currencyCode?: string | null | undefined;
  description: string;
  endDate?: number | null | undefined;
  frequency?: RecurringEarningFrequency | null | undefined;
  payCodeId: string | number;
  startDate: number;
  totalCapMinor?: number | null | undefined;
  workerId: string | number;
};

export type CreateReportScheduleInput = {
  cronExpression: string;
  definitionId: string | number;
  emailAttach?: boolean | null | undefined;
  emailRecipients?: Array<string> | null | undefined;
  enabled: boolean;
  formats: Array<string>;
  notifyUserIds?: Array<string | number> | null | undefined;
  timezone?: string | null | undefined;
};

export type CreateSettlementDisputeInput = {
  category: SettlementDisputeCategory;
  description: string;
  settlementId: string | number;
  settlementLineId?: string | number | null | undefined;
};

export type CustomerBillingCycleType =
  | 'BiWeekly'
  | 'Daily'
  | 'Immediate'
  | 'Monthly'
  | 'PerShipment'
  | 'Quarterly'
  | 'Weekly';

export type CustomerConsolidationGroupBy =
  | 'BOL'
  | 'Division'
  | 'Location'
  | 'None'
  | 'PONumber';

export type CustomerCreditStatus =
  | 'Active'
  | 'Hold'
  | 'Review'
  | 'Suspended'
  | 'Warning';

export type CustomerFuelSurchargeMode =
  | 'FuelIncluded'
  | 'None'
  | 'Program';

export type CustomerInvoiceAdjustmentSupportingDocumentPolicy =
  | 'Inherit'
  | 'Optional'
  | 'Required';

export type CustomerInvoiceMethod =
  | 'Individual'
  | 'Summary'
  | 'SummaryWithDetail';

export type CustomerInvoiceNumberFormat =
  | 'CustomPrefix'
  | 'Default'
  | 'POBased';

export type CustomerPaymentApplicationInput = {
  appliedAmountMinor: number;
  invoiceId: string | number;
  shortPayAmountMinor?: number | null | undefined;
};

export type CustomerPaymentMethod =
  | 'ACH'
  | 'Card'
  | 'Cash'
  | 'Check'
  | 'Other'
  | 'Wire';

export type CustomerPaymentStatus =
  | 'Posted'
  | 'Reversed';

export type CustomerPaymentTerm =
  | 'DueOnReceipt'
  | 'Net10'
  | 'Net15'
  | 'Net30'
  | 'Net45'
  | 'Net60'
  | 'Net90';

export type DataTableConnectionInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
};

export type DetachPayEventInput = {
  payEventId: string | number;
  settlementId: string | number;
};

export type DisputeAdjustmentInput = {
  amountMinor: number;
  description: string;
  payCodeId?: string | number | null | undefined;
};

export type DocumentCategory =
  | 'Branding'
  | 'Contract'
  | 'Invoice'
  | 'Other'
  | 'Profile'
  | 'Regulatory'
  | 'Shipment'
  | 'Worker';

export type DocumentClassification =
  | 'Private'
  | 'Public'
  | 'Regulatory'
  | 'Sensitive';

export type DriverExpenseStatus =
  | 'Approved'
  | 'Cancelled'
  | 'Pending'
  | 'Reimbursed'
  | 'Rejected';

export type DriverPayEventStatus =
  | 'Accrued'
  | 'Settled'
  | 'Voided';

export type DriverSettlementActionInput = {
  reason?: string | null | undefined;
  settlementId: string | number;
};

export type DriverSettlementStatus =
  | 'Approved'
  | 'Draft'
  | 'Paid'
  | 'PendingApproval'
  | 'Posted'
  | 'Voided';

export type DriverType =
  | 'Local'
  | 'OTR'
  | 'Regional'
  | 'Team';

export type EdiConnectionMethod =
  | 'AS2'
  | 'Internal'
  | 'SFTP'
  | 'VAN';

export type EdiConnectionStatus =
  | 'Active'
  | 'PendingAcceptance'
  | 'Rejected'
  | 'Revoked'
  | 'Suspended';

export type EdiDocumentDirection =
  | 'Inbound'
  | 'Outbound';

export type EdiInboundFileStatus =
  | 'Duplicate'
  | 'Parsed'
  | 'PartiallyProcessed'
  | 'Processed'
  | 'Quarantined'
  | 'Received';

export type EdiMappingEntityType =
  | 'AccessorialCharge'
  | 'Commodity'
  | 'Customer'
  | 'FormulaTemplate'
  | 'Location'
  | 'ServiceFailureReasonCode'
  | 'ServiceType'
  | 'ShipmentType';

export type EdiMessageAckStatus =
  | 'Accepted'
  | 'Failed'
  | 'NotExpected'
  | 'Pending'
  | 'Rejected';

export type EdiMessageDeliveryStatus =
  | 'DeadLettered'
  | 'Failed'
  | 'Queued'
  | 'Sending'
  | 'Sent';

export type EdiMessageStatus =
  | 'Failed'
  | 'Generated';

export type EdiPartnerKind =
  | 'External'
  | 'Internal';

export type EdiStandard =
  | 'X12';

export type EdiSummaryAttentionKind =
  | 'InboundFile'
  | 'Message';

export type EdiTemplateStatus =
  | 'Active'
  | 'Archived'
  | 'Certified'
  | 'Deprecated'
  | 'Draft'
  | 'Superseded';

export type EdiTransferDirection =
  | 'Inbound'
  | 'Outbound';

export type EdiTransferStatus =
  | 'Approved'
  | 'Canceled'
  | 'Expired'
  | 'Failed'
  | 'MappingRequired'
  | 'PendingApproval'
  | 'Processing'
  | 'Rejected'
  | 'Submitted';

export type EffectiveRateSource =
  | 'Benchmark'
  | 'GLActual'
  | 'LiveIndex'
  | 'Override';

export type EmailProfileStatus =
  | 'Active'
  | 'Inactive';

export type EmailProvider =
  | 'Postmark'
  | 'Resend';

export type EndWorkerPayAssignmentInput = {
  assignmentId: string | number;
  endDate: number;
};

export type EndorsementType =
  | 'H'
  | 'N'
  | 'O'
  | 'P'
  | 'T'
  | 'X';

export type EntityStatus =
  | 'Active'
  | 'Inactive';

export type EquipmentClass =
  | 'Container'
  | 'Other'
  | 'Tractor'
  | 'Trailer';

export type EquipmentStatus =
  | 'AtMaintenance'
  | 'Available'
  | 'OutOfService'
  | 'Sold';

export type EquipmentTypeInput = {
  class: EquipmentClass;
  code: string;
  color?: string | null | undefined;
  description?: string | null | undefined;
  interiorLength?: number | null | undefined;
  status?: EntityStatus | null | undefined;
  version?: number | null | undefined;
};

export type EquipmentTypePatchInput = {
  class?: EquipmentClass | null | undefined;
  code?: string | null | undefined;
  color?: string | null | undefined;
  description?: string | null | undefined;
  interiorLength?: number | null | undefined;
  status?: EntityStatus | null | undefined;
  version?: number | null | undefined;
};

export type EscrowAccountStatus =
  | 'Active'
  | 'Closed';

export type EscrowTransactionType =
  | 'Adjustment'
  | 'Application'
  | 'Contribution'
  | 'InterestAccrual'
  | 'Refund';

export type FacilityType =
  | 'ColdStorage'
  | 'CrossDock'
  | 'HazmatFacility'
  | 'IntermodalFacility'
  | 'StorageWarehouse';

export type FieldFilterInput = {
  field: string;
  operator: string;
  value?: unknown;
};

export type FieldType =
  | 'boolean'
  | 'date'
  | 'multiSelect'
  | 'number'
  | 'select'
  | 'text';

export type FilterGroupInput = {
  filters: Array<FieldFilterInput>;
};

export type FiscalPeriodStatus =
  | 'Closed'
  | 'Inactive'
  | 'Locked'
  | 'Open'
  | 'PermanentlyClosed';

export type FiscalYearStatus =
  | 'Closed'
  | 'Draft'
  | 'Open'
  | 'PermanentlyClosed';

export type ForkCannedReportInput = {
  cannedKey: string;
  name?: string | null | undefined;
};

export type FormulaTemplateStatus =
  | 'Active'
  | 'Draft'
  | 'Inactive';

export type FormulaTemplateType =
  | 'AccessorialCharge'
  | 'FreightCharge';

export type FreightClass =
  | 'Class50'
  | 'Class55'
  | 'Class60'
  | 'Class65'
  | 'Class70'
  | 'Class77_5'
  | 'Class85'
  | 'Class92_5'
  | 'Class100'
  | 'Class110'
  | 'Class125'
  | 'Class150'
  | 'Class175'
  | 'Class200'
  | 'Class250'
  | 'Class300'
  | 'Class400'
  | 'Class500';

export type FuelIndexInput = {
  code: string;
  currency?: string | null | undefined;
  description?: string | null | undefined;
  eiaSeriesId?: string | null | undefined;
  fuelType?: FuelType | null | undefined;
  isActive?: boolean | null | undefined;
  name: string;
  region?: string | null | undefined;
  source: FuelIndexSource;
};

export type FuelIndexPriceInput = {
  fuelIndexId: string | number;
  price: string;
  priceDate: string;
};

export type FuelIndexSource =
  | 'Custom'
  | 'EIA';

export type FuelSurchargeDateBasis =
  | 'PickupDate'
  | 'TenderDate';

export type FuelSurchargeMissingPriceFallback =
  | 'Skip'
  | 'UseLatestAvailable';

export type FuelSurchargePercentBasis =
  | 'Linehaul'
  | 'LinehaulPlusAccessorials';

export type FuelSurchargeProgramInput = {
  accessorialChargeId: string | number;
  code: string;
  dateBasis?: FuelSurchargeDateBasis | null | undefined;
  description?: string | null | undefined;
  effectiveEndDate?: number | null | undefined;
  effectiveStartDate?: number | null | undefined;
  fuelIndexId: string | number;
  increment?: string | null | undefined;
  incrementRate?: string | null | undefined;
  maxAmount?: string | null | undefined;
  method: FuelSurchargeProgramMethod;
  milesPerGallon?: string | null | undefined;
  minAmount?: string | null | undefined;
  missingPriceFallback?: FuelSurchargeMissingPriceFallback | null | undefined;
  name: string;
  pegPrice?: string | null | undefined;
  percentBasis?: FuelSurchargePercentBasis | null | undefined;
  priceEffectiveDay?: number | null | undefined;
  ratePrecision?: number | null | undefined;
  rateRounding?: FuelSurchargeRateRounding | null | undefined;
  serviceTypeIds?: Array<string | number> | null | undefined;
  shipmentTypeIds?: Array<string | number> | null | undefined;
  status?: FuelSurchargeProgramStatus | null | undefined;
  stepRounding?: FuelSurchargeStepRounding | null | undefined;
  tableRows?: Array<FuelSurchargeTableRowInput> | null | undefined;
  tractorTypeIds?: Array<string | number> | null | undefined;
  trailerTypeIds?: Array<string | number> | null | undefined;
};

export type FuelSurchargeProgramMethod =
  | 'PerMileMPG'
  | 'PerMileStep'
  | 'TableFlat'
  | 'TablePerMile'
  | 'TablePercent';

export type FuelSurchargeProgramStatus =
  | 'Active'
  | 'Inactive';

export type FuelSurchargeRateRounding =
  | 'Down'
  | 'HalfUp'
  | 'Up';

export type FuelSurchargeStepRounding =
  | 'Down'
  | 'Nearest'
  | 'Up';

export type FuelSurchargeTableRowInput = {
  priceMax?: string | null | undefined;
  priceMin?: string | null | undefined;
  sortOrder?: number | null | undefined;
  value: string;
};

export type FuelType =
  | 'Diesel'
  | 'Gasoline';

export type GenerateDriverSettlementInput = {
  batchId?: string | number | null | undefined;
  payDate: number;
  periodEnd: number;
  periodStart: number;
  workerId: string | number;
};

export type GenerateFuelTableInput = {
  increment: string;
  maxPrice: string;
  minPrice: string;
  openEnded?: boolean | null | undefined;
  startValue: string;
  valueStep: string;
};

export type GenerateSettlementBatchInput = {
  name?: string | null | undefined;
  notes?: string | null | undefined;
  periodEnd?: number | null | undefined;
  periodStart?: number | null | undefined;
};

export type HazardousClass =
  | 'HazardClass1'
  | 'HazardClass1And1'
  | 'HazardClass1And2'
  | 'HazardClass1And3'
  | 'HazardClass1And4'
  | 'HazardClass1And5'
  | 'HazardClass1And6'
  | 'HazardClass2And1'
  | 'HazardClass2And2'
  | 'HazardClass2And3'
  | 'HazardClass3'
  | 'HazardClass4And1'
  | 'HazardClass4And2'
  | 'HazardClass4And3'
  | 'HazardClass5And1'
  | 'HazardClass5And2'
  | 'HazardClass6And1'
  | 'HazardClass6And2'
  | 'HazardClass7'
  | 'HazardClass8'
  | 'HazardClass9';

export type HoldPayEventInput = {
  payEventId: string | number;
  /** Why pay is being deferred — shown to whoever reviews the held event. */
  reason: string;
};

export type HoldSeverity =
  | 'Advisory'
  | 'Blocking'
  | 'Informational';

export type HoldType =
  | 'ComplianceHold'
  | 'CustomerHold'
  | 'FinanceHold'
  | 'OperationalHold';

export type InviteWorkerToPortalInput = {
  /** Overrides the email on the worker record when provided. */
  email?: string | null | undefined;
  workerId: string | number;
};

export type InvoiceDisputeStatus =
  | 'Disputed'
  | 'None';

export type InvoicePaymentTerm =
  | 'DueOnReceipt'
  | 'Net10'
  | 'Net15'
  | 'Net30'
  | 'Net45'
  | 'Net60'
  | 'Net90';

export type InvoiceSendStatus =
  | 'Failed'
  | 'NotSent'
  | 'PartiallySent'
  | 'Sending'
  | 'Sent';

export type InvoiceSettlementStatus =
  | 'Paid'
  | 'PartiallyPaid'
  | 'Unpaid';

export type InvoiceStatus =
  | 'Draft'
  | 'Posted';

export type IssuePayAdvanceInput = {
  amountMinor: number;
  currencyCode?: string | null | undefined;
  issuedDate: number;
  notes?: string | null | undefined;
  reference?: string | null | undefined;
  source: PayAdvanceSource;
  workerId: string | number;
};

export type JournalReversalStatus =
  | 'Approved'
  | 'Cancelled'
  | 'PendingApproval'
  | 'Posted'
  | 'Rejected'
  | 'Requested';

export type LocationCategoryType =
  | 'CustomerLocation'
  | 'DistributionCenter'
  | 'MaintenanceFacility'
  | 'Port'
  | 'RailYard'
  | 'RestArea'
  | 'Terminal'
  | 'TruckStop'
  | 'Warehouse';

export type ManualJournalStatus =
  | 'Approved'
  | 'Cancelled'
  | 'Draft'
  | 'PendingApproval'
  | 'Posted'
  | 'Rejected';

export type MarkDriverSettlementPaidInput = {
  paymentMethod: string;
  paymentReference?: string | null | undefined;
  settlementId: string | number;
};

export type MoveStatus =
  | 'Assigned'
  | 'Canceled'
  | 'Completed'
  | 'InTransit'
  | 'New';

export type NotificationChannel =
  | 'global'
  | 'role'
  | 'user';

export type NotificationFilterInput = {
  state?: NotificationState | null | undefined;
  unreadOnly?: boolean | null | undefined;
};

export type NotificationPriority =
  | 'critical'
  | 'high'
  | 'low'
  | 'medium';

export type NotificationState =
  | 'archived'
  | 'inbox';

export type OpenEscrowAccountInput = {
  annualInterestRate?: string | null | undefined;
  currencyCode?: string | null | undefined;
  openedDate?: number | null | undefined;
  targetAmountMinor: number;
  workerId: string | number;
};

export type OrderInput = {
  baseAmount?: string | null | undefined;
  bol?: string | null | undefined;
  currencyCode?: string | null | undefined;
  customerId: string | number;
  ownerId?: string | number | null | undefined;
  poNumber?: string | null | undefined;
  quotedAmount?: string | null | undefined;
  version?: number | null | undefined;
};

export type OrderStatus =
  | 'Billed'
  | 'Canceled'
  | 'Closed'
  | 'Completed'
  | 'Confirmed'
  | 'Draft'
  | 'InProgress';

export type OrganizationInput = {
  addressLine1: string;
  addressLine2?: string | null | undefined;
  bucketName?: string | null | undefined;
  city: string;
  dotNumber: string;
  loginSlug?: string | null | undefined;
  logoUrl?: string | null | undefined;
  name: string;
  postalCode: string;
  scacCode: string;
  stateId: string | number;
  taxId?: string | null | undefined;
  timezone: string;
  version: number;
};

export type PtoStatus =
  | 'Approved'
  | 'Cancelled'
  | 'Rejected'
  | 'Requested';

export type PtoType =
  | 'Bereavement'
  | 'Holiday'
  | 'Maternity'
  | 'Paternity'
  | 'Personal'
  | 'Sick'
  | 'Vacation';

export type PackingGroup =
  | 'I'
  | 'II'
  | 'III';

export type PayAdvanceSource =
  | 'Cash'
  | 'ComdataCode'
  | 'EFSMoneyCode'
  | 'FuelCard'
  | 'Other';

export type PayAdvanceStatus =
  | 'Outstanding'
  | 'PartiallyRecovered'
  | 'Recovered'
  | 'WrittenOff';

export type PayCalcMethod =
  | 'FlatPerShipment'
  | 'PerDay'
  | 'PerEmptyMile'
  | 'PerEvent'
  | 'PerHour'
  | 'PerLoadedMile'
  | 'PerStop'
  | 'PerTotalMile'
  | 'PercentOfRevenue';

export type PayCodeDirection =
  | 'Deduction'
  | 'Earning';

export type PayComponentKind =
  | 'Bonus'
  | 'Breakdown'
  | 'Custom'
  | 'Detention'
  | 'FuelSurcharge'
  | 'Hazmat'
  | 'Layover'
  | 'Linehaul'
  | 'StopPay'
  | 'Tarp';

export type PayMileageBandInput = {
  maxMiles: number;
  minMiles: number;
  rate: string;
};

export type PayPeriodFrequency =
  | 'Biweekly'
  | 'Monthly'
  | 'Weekly';

export type PayProfileComponentInput = {
  bands?: Array<PayMileageBandInput> | null | undefined;
  description?: string | null | undefined;
  freeTimeMinutes?: number | null | undefined;
  isActive?: boolean | null | undefined;
  kind: PayComponentKind;
  maxAmountMinor?: number | null | undefined;
  method: PayCalcMethod;
  minAmountMinor?: number | null | undefined;
  rate: string;
  revenueBasis?: PayRevenueBasis | null | undefined;
};

export type PayRateOverrideInput = {
  componentId: string | number;
  rate: string;
};

export type PayRevenueBasis =
  | 'Linehaul'
  | 'LinehaulPlusFuelSurcharge'
  | 'TotalRevenue';

export type PayWorkerNowInput = {
  /**
   * Also apply recurring deductions, escrow, advance recovery, and carry-forward.
   * Off by default so the instant payout doesn't double-dip items the regular
   * period settlement will take.
   */
  applyRecurring?: boolean | null | undefined;
  /** Specific accrued events to pay; omit to pay everything accrued and unheld. */
  payEventIds?: Array<string | number> | null | undefined;
  paymentMethod: string;
  paymentReference?: string | null | undefined;
  workerId: string | number;
};

export type PayeeClassification =
  | 'CompanyDriver'
  | 'OwnerOperator';

export type PeriodType =
  | 'Adjusting'
  | 'Month'
  | 'Quarter'
  | 'Week';

export type PortalInvitationStatus =
  | 'Accepted'
  | 'Pending'
  | 'Revoked';

export type PortalLoadScope =
  | 'Active'
  | 'History';

export type PortalPtoStatus =
  | 'Approved'
  | 'Cancelled'
  | 'Rejected'
  | 'Requested';

export type PortalPtoType =
  | 'Bereavement'
  | 'Holiday'
  | 'Maternity'
  | 'Paternity'
  | 'Personal'
  | 'Sick'
  | 'Vacation';

export type PortalStopAction =
  | 'Arrive'
  | 'Depart';

export type PostCustomerPaymentInput = {
  accountingDate: number;
  amountMinor: number;
  applications?: Array<CustomerPaymentApplicationInput> | null | undefined;
  currencyCode?: string | null | undefined;
  customerId: string | number;
  memo?: string | null | undefined;
  paymentDate: number;
  paymentMethod: CustomerPaymentMethod;
  referenceNumber?: string | null | undefined;
};

export type RateTableLookupType =
  | 'Exact'
  | 'Range';

export type RateUnit =
  | 'Day'
  | 'Hour'
  | 'Mile'
  | 'Stop';

export type RecordMyStopActionInput = {
  action: PortalStopAction;
  moveId: string | number;
  stopId: string | number;
};

export type RecurringDeductionFrequency =
  | 'EverySettlement'
  | 'Monthly';

export type RecurringDeductionStatus =
  | 'Active'
  | 'Completed'
  | 'Paused';

export type RecurringEarningFrequency =
  | 'EverySettlement'
  | 'Monthly';

export type RecurringEarningStatus =
  | 'Active'
  | 'Completed'
  | 'Paused';

export type RecurringShipmentExceptionPolicy =
  | 'NextBusinessDay'
  | 'PreviousBusinessDay'
  | 'Skip';

export type RecurringShipmentStatus =
  | 'Active'
  | 'Expired'
  | 'Paused';

export type RemoveOrderChargeInput = {
  chargeId: string | number;
  orderId: string | number;
};

export type RemoveSettlementAdjustmentInput = {
  lineId: string | number;
  settlementId: string | number;
};

export type ReportColumnInput = {
  agg?: string | null | undefined;
  bucket?: string | null | undefined;
  computed?: ReportComputedInput | null | undefined;
  id: string;
  kind: string;
  label?: string | null | undefined;
  ref?: ReportFieldRefInput | null | undefined;
};

export type ReportComputedInput = {
  format?: string | null | undefined;
  leftId: string;
  op: string;
  rightId: string;
};

export type ReportFieldRefInput = {
  field: string;
  path?: Array<string> | null | undefined;
};

export type ReportFilterGroupInput = {
  filters?: Array<ReportFilterInput> | null | undefined;
  groups?: Array<ReportFilterGroupInput> | null | undefined;
  op: string;
};

export type ReportFilterInput = {
  agg?: string | null | undefined;
  operator: string;
  param?: string | null | undefined;
  ref: ReportFieldRefInput;
  value?: unknown;
};

export type ReportIrInput = {
  columns: Array<ReportColumnInput>;
  entity: string;
  filters?: ReportFilterGroupInput | null | undefined;
  having?: ReportFilterGroupInput | null | undefined;
  limit?: number | null | undefined;
  parameters?: Array<ReportParameterDefInput> | null | undefined;
  pivot?: ReportPivotInput | null | undefined;
  sort?: Array<ReportSortInput> | null | undefined;
};

export type ReportParameterDefInput = {
  allowedValues?: Array<string> | null | undefined;
  default?: unknown;
  label?: string | null | undefined;
  multi?: boolean | null | undefined;
  name: string;
  refEntity?: string | null | undefined;
  required?: boolean | null | undefined;
  type: string;
};

export type ReportPivotInput = {
  includeOther?: boolean | null | undefined;
  measureIds: Array<string>;
  ref: ReportFieldRefInput;
  values: Array<string>;
};

export type ReportRunsFilterInput = {
  definitionId?: string | number | null | undefined;
  mineOnly?: boolean | null | undefined;
  statuses?: Array<string> | null | undefined;
};

export type ReportSortInput = {
  columnId: string;
  direction: string;
};

export type RequestMyPtoInput = {
  endDate: number;
  reason: string;
  startDate: number;
  type: PortalPtoType;
};

export type ResolveSettlementDisputeInput = {
  /**
   * Optional correcting adjustment applied to the driver's open settlement (one is
   * generated off-cycle when none exists). Only valid when approving.
   */
  adjustment?: DisputeAdjustmentInput | null | undefined;
  approve: boolean;
  disputeId: string | number;
  resolutionNote: string;
};

export type RespondToMyAssignmentInput = {
  accept: boolean;
  assignmentId: string | number;
  reason?: string | null | undefined;
};

export type ReverseCustomerPaymentInput = {
  accountingDate: number;
  paymentId: string | number;
  reason?: string | null | undefined;
};

export type ReviewDriverExpenseInput = {
  approve: boolean;
  expenseId: string | number;
  note?: string | null | undefined;
};

export type RunReportInput = {
  cannedKey?: string | null | undefined;
  definitionId?: string | number | null | undefined;
  format: string;
  params?: unknown;
};

export type SaveReportDefinitionInput = {
  category?: string | null | undefined;
  defaultFormat?: string | null | undefined;
  definition: ReportIrInput;
  description?: string | null | undefined;
  name: string;
  status?: string | null | undefined;
  tags?: Array<string> | null | undefined;
  visibility?: string | null | undefined;
};

export type SegregationType =
  | 'Barrier'
  | 'Distance'
  | 'Prohibited'
  | 'Separated';

export type SelectOptionResource =
  | 'CUSTOMER'
  | 'EDI_TRANSFER'
  | 'EQUIPMENT_MANUFACTURER'
  | 'EQUIPMENT_TYPE'
  | 'FISCAL_PERIOD'
  | 'FISCAL_YEAR'
  | 'FUEL_INDEX'
  | 'FUEL_SURCHARGE_PROGRAM'
  | 'GL_ACCOUNT'
  | 'ORDER'
  | 'SHIPMENT'
  | 'TRACTOR'
  | 'TRAILER'
  | 'US_STATE'
  | 'WORKER';

export type SelectOptionsInput = {
  filters?: unknown;
  first?: number | null | undefined;
  ids?: Array<string | number> | null | undefined;
  offset?: number | null | undefined;
  query?: string | null | undefined;
  resource: SelectOptionResource;
};

export type ServiceFailureReasonCategory =
  | 'Appointment'
  | 'Carrier'
  | 'Consignee'
  | 'Customer'
  | 'Documentation'
  | 'Driver'
  | 'Equipment'
  | 'Facility'
  | 'Other'
  | 'Shipper'
  | 'Weather';

export type ServiceFailureReasonCodeAppliesTo =
  | 'All'
  | 'Both'
  | 'Delivery'
  | 'Pickup';

export type ServiceFailureSource =
  | 'Detected'
  | 'EDI'
  | 'Integration'
  | 'Manual';

export type ServiceFailureStatus =
  | 'Open'
  | 'Resolved'
  | 'Reviewed'
  | 'Voided';

export type ServiceFailureType =
  | 'AppointmentMissed'
  | 'LateDelivery'
  | 'LatePickup'
  | 'MissedDelivery'
  | 'MissedPickup'
  | 'Other';

export type SettlementBatchStatus =
  | 'Canceled'
  | 'Completed'
  | 'Open';

export type SettlementDisputeCategory =
  | 'IncorrectDeduction'
  | 'IncorrectRate'
  | 'MissingPay'
  | 'MissingReimbursement'
  | 'Other';

export type SettlementDisputeStatus =
  | 'Denied'
  | 'InReview'
  | 'Open'
  | 'Resolved'
  | 'Withdrawn';

export type SettlementLineCategory =
  | 'Adjustment'
  | 'AdvanceRecovery'
  | 'CarryForward'
  | 'Deduction'
  | 'Earning'
  | 'EscrowContribution'
  | 'GuaranteeTopUp'
  | 'Reimbursement';

export type SettlementPayTrigger =
  /** Pay accrues the moment a driver completes their move — even before the full shipment delivers. */
  | 'MoveCompleted'
  | 'PODReceived'
  | 'ShipmentDelivered'
  | 'ShipmentInvoiced';

export type ShipmentAdditionalChargeInput = {
  accessorialChargeId: string | number;
  amount?: string | null | undefined;
  fuelSurchargeProgramId?: string | number | null | undefined;
  id?: string | number | null | undefined;
  isSystemGenerated?: boolean | null | undefined;
  method?: string | null | undefined;
  shipmentId?: string | number | null | undefined;
  unit?: number | null | undefined;
  version?: number | null | undefined;
};

export type ShipmentAnalyticsInput = {
  endDate?: number | null | undefined;
  include?: string | null | undefined;
  limit?: number | null | undefined;
  offset?: number | null | undefined;
  startDate?: number | null | undefined;
  timezone?: string | null | undefined;
  windowDays?: number | null | undefined;
};

export type ShipmentBulkTransferToBillingInput = {
  billType?: BillType | null | undefined;
  shipmentIds: Array<string | number>;
};

export type ShipmentCancelInput = {
  cancelReason?: string | null | undefined;
};

export type ShipmentCommentInput = {
  comment: string;
  mentionedUserIds?: Array<string | number> | null | undefined;
  priority?: ShipmentCommentPriority | null | undefined;
  type?: ShipmentCommentType | null | undefined;
  visibility?: ShipmentCommentVisibility | null | undefined;
};

export type ShipmentCommentPriority =
  | 'High'
  | 'Low'
  | 'Normal'
  | 'Urgent';

export type ShipmentCommentSource =
  | 'AI'
  | 'Integration'
  | 'System'
  | 'User';

export type ShipmentCommentType =
  | 'Appointment'
  | 'Billing'
  | 'Compliance'
  | 'CustomerUpdate'
  | 'DeliveryInstruction'
  | 'Dispatch'
  | 'Document'
  | 'DriverUpdate'
  | 'Exception'
  | 'Internal'
  | 'PickupInstruction'
  | 'StatusUpdate';

export type ShipmentCommentUpdateInput = {
  comment: string;
  id: string | number;
  mentionedUserIds?: Array<string | number> | null | undefined;
  priority?: ShipmentCommentPriority | null | undefined;
  type?: ShipmentCommentType | null | undefined;
  version: number;
  visibility?: ShipmentCommentVisibility | null | undefined;
};

export type ShipmentCommentVisibility =
  | 'Accounting'
  | 'Customer'
  | 'Driver'
  | 'Internal'
  | 'Operations';

export type ShipmentCommodityInput = {
  commodityId: string | number;
  id?: string | number | null | undefined;
  pieces?: number | null | undefined;
  shipmentId?: string | number | null | undefined;
  version?: number | null | undefined;
  weight?: number | null | undefined;
};

export type ShipmentDuplicateBolInput = {
  bol: string;
  shipmentId?: string | number | null | undefined;
};

export type ShipmentDuplicateInput = {
  count?: number | null | undefined;
  overrideDates?: boolean | null | undefined;
  shipmentId: string | number;
};

export type ShipmentEntryMethod =
  | 'EDI'
  | 'Manual';

export type ShipmentEventActorType =
  | 'apikey'
  | 'edi'
  | 'system'
  | 'user';

export type ShipmentEventSeverity =
  | 'brand'
  | 'danger'
  | 'info'
  | 'muted'
  | 'success';

export type ShipmentEventType =
  | 'CommentPosted'
  | 'DriverAssigned'
  | 'DriverReassigned'
  | 'DriverUnassigned'
  | 'HoldPlaced'
  | 'HoldReleased'
  | 'HoldUpdated'
  | 'MoveArrived'
  | 'MoveDeparted'
  | 'MoveStatusChanged'
  | 'OwnershipTransferred'
  | 'ShipmentCanceled'
  | 'ShipmentCreated'
  | 'ShipmentUncanceled'
  | 'ShipmentUpdated'
  | 'StatusChanged'
  | 'StopCompleted';

export type ShipmentEventsInput = {
  before?: number | null | undefined;
  limit?: number | null | undefined;
  shipmentId?: string | number | null | undefined;
  types?: Array<ShipmentEventType> | null | undefined;
};

export type ShipmentHazmatInput = {
  commodityIds: Array<string | number>;
};

export type ShipmentInput = {
  actualDeliveryDate?: number | null | undefined;
  actualShipDate?: number | null | undefined;
  additionalCharges?: Array<ShipmentAdditionalChargeInput> | null | undefined;
  baseRate?: string | null | undefined;
  billedAt?: number | null | undefined;
  billingTransferStatus?: string | null | undefined;
  bol?: string | null | undefined;
  cancelReason?: string | null | undefined;
  canceledAt?: number | null | undefined;
  canceledById?: string | number | null | undefined;
  commodities?: Array<ShipmentCommodityInput> | null | undefined;
  consolidationGroupId?: string | number | null | undefined;
  customerId: string | number;
  enteredById?: string | number | null | undefined;
  entryMethod?: ShipmentEntryMethod | null | undefined;
  formulaTemplateId: string | number;
  freightChargeAmount?: string | null | undefined;
  fuelSurchargeLocked?: boolean | null | undefined;
  markedReadyToBillAt?: number | null | undefined;
  moves?: Array<ShipmentMoveInput> | null | undefined;
  orderId?: string | number | null | undefined;
  otherChargeAmount?: string | null | undefined;
  ownerId?: string | number | null | undefined;
  pieces?: number | null | undefined;
  proNumber?: string | null | undefined;
  ratingDetail?: ShipmentRatingDetailInput | null | undefined;
  ratingUnit?: number | null | undefined;
  serviceTypeId: string | number;
  shipmentTypeId: string | number;
  sourceDocumentId?: string | null | undefined;
  status?: ShipmentStatus | null | undefined;
  temperatureMax?: number | null | undefined;
  temperatureMin?: number | null | undefined;
  tenderStatus?: ShipmentTenderStatus | null | undefined;
  totalChargeAmount?: string | null | undefined;
  tractorTypeId?: string | number | null | undefined;
  trailerTypeId?: string | number | null | undefined;
  transferredToBillingAt?: number | null | undefined;
  version?: number | null | undefined;
  weight?: number | null | undefined;
};

export type ShipmentLoadingCommodityInput = {
  commodityId: string | number;
  pieces: number;
  weight: number;
};

export type ShipmentLoadingOptimizationInput = {
  commodities: Array<ShipmentLoadingCommodityInput>;
  equipmentTypeId?: string | number | null | undefined;
  stops?: Array<ShipmentLoadingStopInput> | null | undefined;
};

export type ShipmentLoadingStopInput = {
  locationCity: string;
  locationName: string;
  sequence: number;
};

export type ShipmentMoveInput = {
  distance?: number | null | undefined;
  distanceCalculatedAt?: number | null | undefined;
  distanceDataVersion?: string | null | undefined;
  distanceMetadata?: unknown;
  distanceProvider?: string | null | undefined;
  distanceRouteSignature?: string | null | undefined;
  distanceRoutingType?: string | null | undefined;
  distanceSource?: string | null | undefined;
  distanceUnits?: string | null | undefined;
  id?: string | number | null | undefined;
  loaded?: boolean | null | undefined;
  sequence?: number | null | undefined;
  shipmentId?: string | number | null | undefined;
  status?: MoveStatus | null | undefined;
  stops?: Array<ShipmentStopInput> | null | undefined;
  version?: number | null | undefined;
};

export type ShipmentPreviousRatesInput = {
  customerId?: string | number | null | undefined;
  destinationLocationId: string | number;
  excludeShipmentId?: string | number | null | undefined;
  originLocationId: string | number;
  serviceTypeId: string | number;
  shipmentTypeId: string | number;
};

export type ShipmentRatingDetailInput = {
  expression: string;
  formulaTemplateId: string;
  formulaTemplateName: string;
  ratedAt: number;
  resolvedVariables: unknown;
  result: number;
};

export type ShipmentStatus =
  | 'Assigned'
  | 'Canceled'
  | 'Completed'
  | 'Delayed'
  | 'InTransit'
  | 'Invoiced'
  | 'New'
  | 'PartiallyAssigned'
  | 'PartiallyCompleted'
  | 'ReadyToInvoice';

export type ShipmentStopInput = {
  actualArrival?: number | null | undefined;
  actualDeparture?: number | null | undefined;
  addressLine?: string | null | undefined;
  countDetentionOverride?: boolean | null | undefined;
  countLateOverride?: boolean | null | undefined;
  id?: string | number | null | undefined;
  locationId: string | number;
  pieces?: number | null | undefined;
  scheduleType?: StopScheduleType | null | undefined;
  scheduledWindowEnd?: number | null | undefined;
  scheduledWindowStart?: number | null | undefined;
  sequence?: number | null | undefined;
  shipmentMoveId?: string | number | null | undefined;
  status?: StopStatus | null | undefined;
  type?: StopType | null | undefined;
  version?: number | null | undefined;
  weight?: number | null | undefined;
};

export type ShipmentTenderStatus =
  | 'Accepted'
  | 'Canceled'
  | 'Expired'
  | 'Rejected'
  | 'Tendered';

export type ShipmentTransferOwnershipInput = {
  ownerId: string | number;
};

export type ShipmentTransferToBillingInput = {
  billType?: BillType | null | undefined;
  shipmentId: string | number;
};

export type ShipmentsInput = {
  activityWindowEnd?: number | null | undefined;
  activityWindowStart?: number | null | undefined;
  after?: string | null | undefined;
  expandShipmentDetails?: boolean | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: string | null | undefined;
};

export type SidebarActivityPreferenceInput = {
  defaultOpen: boolean;
  pageSize: number;
};

export type SidebarPreferencesInput = {
  activity: SidebarActivityPreferenceInput;
  attentionMetrics: Array<string>;
  quickActionIds: Array<string>;
  sections: Array<SidebarSectionPreferenceInput>;
  version: number;
};

export type SidebarSectionPreferenceInput = {
  hidden: boolean;
  key: string;
};

export type SortFieldInput = {
  direction: string;
  field: string;
};

export type StopScheduleType =
  | 'Appointment'
  | 'Open';

export type StopStatus =
  | 'Canceled'
  | 'Completed'
  | 'InTransit'
  | 'New';

export type StopType =
  | 'Delivery'
  | 'Pickup'
  | 'SplitDelivery'
  | 'SplitPickup';

export type SubmitMyExpenseInput = {
  amountMinor: number;
  description: string;
  incurredDate?: number | null | undefined;
  payCodeId?: string | number | null | undefined;
  shipmentId?: string | number | null | undefined;
};

export type TableConfigurationInput = {
  description?: string | null | undefined;
  isDefault?: boolean | null | undefined;
  name: string;
  resource: string;
  tableConfig: unknown;
  visibility?: ConfigurationVisibility | null | undefined;
};

export type UpcomingWorkerPtoInput = {
  after?: string | null | undefined;
  endDate?: number | null | undefined;
  first?: number | null | undefined;
  fleetCodeId?: string | number | null | undefined;
  startDate?: number | null | undefined;
  status?: PtoStatus | null | undefined;
  timezone?: string | null | undefined;
  type?: PtoType | null | undefined;
  workerId?: string | number | null | undefined;
};

export type UpdateDashControlInput = {
  allowContactInfoEdit: boolean;
  allowExpenseSubmission: boolean;
  allowLoadComments: boolean;
  allowLoadDocumentUpload: boolean;
  allowLoadRefusals: boolean;
  allowProfileDocumentUpload: boolean;
  allowPtoRequests: boolean;
  allowSettlementDisputes: boolean;
  allowStopActions: boolean;
  detentionAlertThresholdMinutes: number;
  enableDetentionAlerts: boolean;
  requireExpenseReceipt: boolean;
  requireLoadAcknowledgment: boolean;
  sendCredentialReminders: boolean;
  showLoadPay: boolean;
  showPayEstimates: boolean;
  version: number;
};

export type UpdateEscrowAccountInput = {
  annualInterestRate: string;
  id: string | number;
  targetAmountMinor: number;
  version: number;
  workerId: string | number;
};

export type UpdateFuelIndexPriceInput = {
  id: string | number;
  price: string;
  priceDate: string;
};

export type UpdateMyContactInfoInput = {
  addressLine1: string;
  addressLine2?: string | null | undefined;
  city: string;
  emergencyContactName?: string | null | undefined;
  emergencyContactPhone?: string | null | undefined;
  phoneNumber: string;
  postalCode: string;
};

export type UpdateOrderChargeInput = {
  amount: string;
  chargeId: string | number;
  description: string;
  orderId: string | number;
  version: number;
};

export type UpdatePayCodeInput = {
  code: string;
  countsTowardGuarantee: boolean;
  defaultAmountMinor?: number | null | undefined;
  description?: string | null | undefined;
  glAccountId?: string | number | null | undefined;
  id: string | number;
  name: string;
  status: EntityStatus;
  taxable: boolean;
  version: number;
};

export type UpdatePayProfileInput = {
  classification: PayeeClassification;
  components: Array<PayProfileComponentInput>;
  currencyCode?: string | null | undefined;
  description?: string | null | undefined;
  guaranteedPeriodMinimumMinor?: number | null | undefined;
  id: string | number;
  name: string;
  perDiemDailyCapMinor?: number | null | undefined;
  perDiemRatePerMile?: string | null | undefined;
  status?: EntityStatus | null | undefined;
  version: number;
};

export type UpdateRecurringDeductionInput = {
  amountMinor: number;
  currencyCode?: string | null | undefined;
  description: string;
  endDate?: number | null | undefined;
  escrowAccountId?: string | number | null | undefined;
  frequency: RecurringDeductionFrequency;
  id: string | number;
  payCodeId: string | number;
  startDate: number;
  status: RecurringDeductionStatus;
  totalCapMinor?: number | null | undefined;
  version: number;
  workerId: string | number;
};

export type UpdateRecurringEarningInput = {
  amountMinor: number;
  currencyCode?: string | null | undefined;
  description: string;
  endDate?: number | null | undefined;
  frequency: RecurringEarningFrequency;
  id: string | number;
  payCodeId: string | number;
  startDate: number;
  status: RecurringEarningStatus;
  totalCapMinor?: number | null | undefined;
  version: number;
  workerId: string | number;
};

export type UpdateReportDefinitionInput = {
  category?: string | null | undefined;
  defaultFormat?: string | null | undefined;
  definition: ReportIrInput;
  description?: string | null | undefined;
  id: string | number;
  name: string;
  status?: string | null | undefined;
  tags?: Array<string> | null | undefined;
  version: number;
  visibility?: string | null | undefined;
};

export type UpdateReportScheduleInput = {
  cronExpression: string;
  definitionId: string | number;
  emailAttach?: boolean | null | undefined;
  emailRecipients?: Array<string> | null | undefined;
  enabled: boolean;
  formats: Array<string>;
  id: string | number;
  notifyUserIds?: Array<string | number> | null | undefined;
  timezone?: string | null | undefined;
  version: number;
};

export type UpdateSettlementControlInput = {
  allowNegativeNet: boolean;
  autoApproveClean: boolean;
  autoAttachAccruals: boolean;
  autoGenerateBatches: boolean;
  autoPostOnApprove: boolean;
  defaultEscrowInterestRate: string;
  escrowInterestFrequencyMonths: number;
  payDelayDays: number;
  payPeriodFrequency: PayPeriodFrequency;
  payTrigger: SettlementPayTrigger;
  periodEndDayOfWeek: number;
  varianceLookbackWeeks: number;
  varianceThresholdPct: string;
  version: number;
};

export type WorkerGender =
  | 'Female'
  | 'Male';

export type WorkerPtoChartInput = {
  startDateFrom: number;
  startDateTo: number;
  timezone?: string | null | undefined;
  type?: PtoType | null | undefined;
  workerId?: string | number | null | undefined;
};

export type WorkerPtoEntriesInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  includeWorker?: boolean | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  startDateFrom?: number | null | undefined;
  startDateTo?: number | null | undefined;
  status?: PtoStatus | null | undefined;
  type?: PtoType | null | undefined;
  workerId?: string | number | null | undefined;
};

export type WorkerPatchInput = {
  driverType?: DriverType | null | undefined;
  status?: EntityStatus | null | undefined;
  type?: WorkerType | null | undefined;
};

export type WorkerType =
  | 'Contractor'
  | 'Employee';

export type WriteOffPayAdvanceInput = {
  advanceId: string | number;
  reason: string;
};

export type AccessorialChargeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, description: string, method: AccessorialMethod, rateUnit: RateUnit | null, amount: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'AccessorialChargeTableRowFieldsFragment' };

export type AccessorialChargeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type AccessorialChargeTableQuery = { accessorialCharges: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AccessorialChargeTableRowFieldsFragment': AccessorialChargeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type AccountTypeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, name: string, description: string | null, category: AccountCategory, color: string | null, isSystem: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'AccountTypeTableRowFieldsFragment' };

export type AccountTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type AccountTypeTableQuery = { accountTypes: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AccountTypeTableRowFieldsFragment': AccountTypeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ArAgingSummaryQueryVariables = Exact<{
  asOfDate?: number | null | undefined;
}>;


export type ArAgingSummaryQuery = { arAgingSummary: { asOfDate: number, totals: { currentMinor: number, days1To30Minor: number, days31To60Minor: number, days61To90Minor: number, daysOver90Minor: number, totalOpenMinor: number }, rows: Array<{ customerId: string, customerName: string, buckets: { currentMinor: number, days1To30Minor: number, days31To60Minor: number, days61To90Minor: number, daysOver90Minor: number, totalOpenMinor: number } }> } };

export type ArOpenItemsQueryVariables = Exact<{
  customerId?: string | number | null | undefined;
  asOfDate?: number | null | undefined;
}>;


export type ArOpenItemsQuery = { arOpenItems: Array<{ invoiceId: string, customerId: string, customerName: string, invoiceNumber: string, billType: string, invoiceDate: number, dueDate: number, currencyCode: string, shipmentProNumber: string, shipmentBol: string, totalAmountMinor: number, appliedAmountMinor: number, openAmountMinor: number, daysPastDue: number, settlementStatus: string, disputeStatus: string, hasShortPay: boolean }> };

export type ArCustomerLedgerQueryVariables = Exact<{
  customerId: string | number;
}>;


export type ArCustomerLedgerQuery = { arCustomerLedger: Array<{ customerId: string, transactionDate: number, eventType: string, documentNumber: string, sourceObjectType: string, sourceObjectId: string, amountMinor: number, relatedInvoiceId: string | null }> };

export type ArCustomerStatementQueryVariables = Exact<{
  customerId: string | number;
  startDate?: number | null | undefined;
  asOfDate?: number | null | undefined;
}>;


export type ArCustomerStatementQuery = { arCustomerStatement: { customerId: string, customerName: string, statementDate: number, startDate: number, openingBalanceMinor: number, totalChargesMinor: number, totalPaymentsMinor: number, endingBalanceMinor: number, aging: { currentMinor: number, days1To30Minor: number, days31To60Minor: number, days61To90Minor: number, daysOver90Minor: number, totalOpenMinor: number }, transactions: Array<{ transactionDate: number, eventType: string, documentNumber: string, sourceObjectId: string, amountMinor: number, chargeMinor: number, paymentMinor: number, runningBalanceMinor: number }>, openItems: Array<{ invoiceId: string, customerId: string, customerName: string, invoiceNumber: string, billType: string, invoiceDate: number, dueDate: number, currencyCode: string, shipmentProNumber: string, shipmentBol: string, totalAmountMinor: number, appliedAmountMinor: number, openAmountMinor: number, daysPastDue: number, settlementStatus: string, disputeStatus: string, hasShortPay: boolean }> } };

export type ArDashboardKpisQueryVariables = Exact<{ [key: string]: never; }>;


export type ArDashboardKpisQuery = { arDashboardKpis: { asOfDate: number, currentDsoDays: number, dsoDeltaDays: number, cei: number, avgDaysToPay: number, overduePercent: number, writeOffRatio: number, disputeRate: number, shortPayRate: number, overview: { totalOpenMinor: number, overdueMinor: number, unappliedCashMinor: number, disputedOpenMinor: number, openInvoiceCount: number, overdueInvoiceCount: number, disputedInvoiceCount: number, avgDaysPastDue: number, buckets: { currentMinor: number, days1To30Minor: number, days31To60Minor: number, days61To90Minor: number, daysOver90Minor: number, totalOpenMinor: number } } } };

export type ArDsoTrendQueryVariables = Exact<{
  weeks?: number | null | undefined;
}>;


export type ArDsoTrendQuery = { arDsoTrend: Array<{ periodEnd: number, dsoDays: number, arBalanceMinor: number, billedMinor: number }> };

export type ArAgingTrendQueryVariables = Exact<{
  weeks?: number | null | undefined;
}>;


export type ArAgingTrendQuery = { arAgingTrend: Array<{ periodEnd: number, buckets: { currentMinor: number, days1To30Minor: number, days31To60Minor: number, days61To90Minor: number, daysOver90Minor: number, totalOpenMinor: number } }> };

export type ArCashFlowForecastQueryVariables = Exact<{
  pastWeeks?: number | null | undefined;
  futureWeeks?: number | null | undefined;
}>;


export type ArCashFlowForecastQuery = { arCashFlowForecast: Array<{ weekStart: number, expectedMinor: number, openDueMinor: number, actualMinor: number, isForecast: boolean }> };

export type ArCollectionPerformanceQueryVariables = Exact<{
  periodDays?: number | null | undefined;
}>;


export type ArCollectionPerformanceQuery = { arCollectionPerformance: { cei: number, writeOffRatio: number, disputeRate: number, shortPayRate: number, totals: { periodStart: number, periodEnd: number, beginningOpenMinor: number, endingOpenMinor: number, endingCurrentMinor: number, creditSalesMinor: number, collectedMinor: number, avgDaysToPay: number, shortPayMinor: number, shortPayApplicationCount: number, applicationCount: number, disputedInvoiceCount: number, postedInvoiceCount: number } } };

export type ArTopOverdueCustomersQueryVariables = Exact<{
  limit?: number | null | undefined;
}>;


export type ArTopOverdueCustomersQuery = { arTopOverdueCustomers: Array<{ customerId: string, customerName: string, overdueMinor: number, totalOpenMinor: number, oldestDaysPastDue: number, openInvoiceCount: number }> };

export type ArCollectionsWorklistQueryVariables = Exact<{
  limit?: number | null | undefined;
}>;


export type ArCollectionsWorklistQuery = { arCollectionsWorklist: Array<{ invoiceId: string, customerId: string, customerName: string, invoiceNumber: string, dueDate: number, openAmountMinor: number, daysPastDue: number, isDisputed: boolean, hasShortPay: boolean, severity: string }> };

export type ArPaymentStatsQueryVariables = Exact<{ [key: string]: never; }>;


export type ArPaymentStatsQuery = { arPaymentStats: { postedTodayMinor: number, postedTodayCount: number, unappliedCashMinor: number, unappliedPaymentCount: number, reversedLast30Minor: number, reversedLast30Count: number } };

export type ArCustomerProfileQueryVariables = Exact<{
  customerId: string | number;
}>;


export type ArCustomerProfileQuery = { arCustomerProfile: { dsoDays: number, creditUtilization: number, delinquencyScore: number, snapshot: { customerId: string, customerName: string, totalOpenMinor: number, overdueMinor: number, unappliedCashMinor: number, creditLimitMinor: number, hasCreditLimit: boolean, openInvoiceCount: number, oldestOpenInvoiceDate: number, oldestDaysPastDue: number, lastPaymentDate: number, lastPaymentMinor: number, avgDaysToPay: number, billedTrailing91Minor: number, buckets: { currentMinor: number, days1To30Minor: number, days31To60Minor: number, days61To90Minor: number, daysOver90Minor: number, totalOpenMinor: number }, monthlyCollections: Array<{ monthStart: number, amountMinor: number }> } } };

export type ApiKeyTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, keyPrefix: string, status: string, expiresAt: number, lastUsedAt: number, permissionScope: string, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ApiKeyTableRowFieldsFragment' };

export type ApiKeyTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type ApiKeyTableQuery = { apiKeys: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ApiKeyTableRowFieldsFragment': ApiKeyTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type AttentionSummaryQueryVariables = Exact<{ [key: string]: never; }>;


export type AttentionSummaryQuery = { attentionSummary: { billingQueue: number | null, pendingApprovals: number | null, reconciliationExceptions: number | null, serviceFailures: number | null, ediAttention: number | null } };

export type RecentActivityQueryVariables = Exact<{
  first: number;
  after?: string | null | undefined;
}>;


export type RecentActivityQuery = { auditEntries: { edges: Array<{ node: { id: string, resource: string, operation: string, resourceId: string, timestamp: number, comment: string | null, entityRef: string | null, user: { id: string, name: string, username: string, profilePicUrl: string, thumbnailUrl: string } | null } }>, pageInfo: { endCursor: string | null, hasNextPage: boolean } } };

export type AuditLogTableRowFieldsFragment = { id: string, userId: string | null, businessUnitId: string, organizationId: string, timestamp: number, changes: unknown, previousState: unknown, currentState: unknown, metadata: unknown, resource: string, operation: string, resourceId: string, correlationId: string | null, userAgent: string | null, comment: string | null, ipAddress: string | null, category: AuditCategory, sensitiveData: boolean, critical: boolean, user: { id: string, name: string, username: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null } & { ' $fragmentName'?: 'AuditLogTableRowFieldsFragment' };

export type AuditLogTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type AuditLogTableQuery = { auditEntries: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AuditLogTableRowFieldsFragment': AuditLogTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type BillingQueueActionFieldsFragment = { id: string, organizationId: string, businessUnitId: string, shipmentId: string | null, assignedBillerId: string | null, number: string, status: BillingQueueStatus, billType: BillType, exceptionReasonCode: BillingQueueExceptionReasonCode | null, reviewNotes: string, exceptionNotes: string, reviewStartedAt: number | null, reviewCompletedAt: number | null, canceledById: string | null, canceledAt: number | null, cancelReason: string, isAdjustmentOrigin: boolean, sourceInvoiceId: string | null, sourceInvoiceAdjustmentId: string | null, sourceCreditMemoInvoiceId: string | null, correctionGroupId: string | null, rebillStrategy: string | null, requiresReplacementReview: boolean, rerateVariancePercent: string, adjustmentContext: unknown, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'BillingQueueActionFieldsFragment' };

export type UpdateBillingQueueStatusMutationVariables = Exact<{
  id: string | number;
  input: BillingQueueUpdateStatusInput;
}>;


export type UpdateBillingQueueStatusMutation = { updateBillingQueueStatus: { ' $fragmentRefs'?: { 'BillingQueueActionFieldsFragment': BillingQueueActionFieldsFragment } } };

export type AssignBillingQueueBillerMutationVariables = Exact<{
  id: string | number;
  input: BillingQueueAssignInput;
}>;


export type AssignBillingQueueBillerMutation = { assignBillingQueueBiller: { ' $fragmentRefs'?: { 'BillingQueueActionFieldsFragment': BillingQueueActionFieldsFragment } } };

export type CommodityTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, hazardousMaterialId: string | null, status: EntityStatus, name: string, description: string, minTemperature: number | null, maxTemperature: number | null, weightPerUnit: number | null, linearFeetPerUnit: number | null, maxQuantityPerShipment: number | null, freightClass: FreightClass | null, loadingInstructions: string | null, stackable: boolean, fragile: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CommodityTableRowFieldsFragment' };

export type CommodityTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type CommodityTableQuery = { commodities: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'CommodityTableRowFieldsFragment': CommodityTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CostingControlPageQueryVariables = Exact<{ [key: string]: never; }>;


export type CostingControlPageQuery = { costingControl: { id: string, businessUnitId: string, organizationId: string, fuelIndexId: string | null, useLiveFuelPrice: boolean, milesPerGallon: string, includeDeadheadMiles: boolean, glActualsEnabled: boolean, glRollingMonths: number, plannedMonthlyMiles: number | null, targetMarginPercent: string | null, version: number, createdAt: number, updatedAt: number, fuelIndex: { id: string, name: string, code: string, source: FuelIndexSource, fuelType: FuelType, isActive: boolean } | null, categories: Array<{ id: string, category: CostCategoryType, name: string, costBehavior: CostBehavior, rateSource: CostRateSource, benchmarkRatePerMile: string, overrideRatePerMile: string | null, isActive: boolean, sortOrder: number, version: number, glAccounts: Array<{ id: string, glAccountId: string, accountCode: string, accountName: string }> }> } };

export type ResolvedCostProfilePageQueryVariables = Exact<{
  asOfDate?: string | null | undefined;
}>;


export type ResolvedCostProfilePageQuery = { resolvedCostProfile: { totalCpm: string, variableCpm: string, fixedCpm: string, targetMarginPercent: string | null, includeDeadheadMiles: boolean, asOfDate: string, fuel: { pricePerGallon: string | null, priceDate: string, fuelIndexId: string | null, milesPerGallon: string, source: EffectiveRateSource } | null, categories: Array<{ category: CostCategoryType, name: string, costBehavior: CostBehavior, ratePerMile: string, effectiveSource: EffectiveRateSource }>, glWindow: { fromDate: number, toDate: number, fleetMiles: number, hasPostings: boolean } | null } };

export type UpdateCostingControlMutationVariables = Exact<{
  input: CostingControlInput;
}>;


export type UpdateCostingControlMutation = { updateCostingControl: { id: string, version: number } };

export type UpdateCostCategoryMutationVariables = Exact<{
  input: CostCategoryUpdateInput;
}>;


export type UpdateCostCategoryMutation = { updateCostCategory: { id: string, version: number } };

export type CustomFieldDefinitionTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, resourceType: string, name: string, label: string, description: string | null, fieldType: FieldType, isRequired: boolean, isActive: boolean, displayOrder: number, color: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CustomFieldDefinitionTableRowFieldsFragment' };

export type CustomFieldDefinitionTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type CustomFieldDefinitionTableQuery = { customFieldDefinitions: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'CustomFieldDefinitionTableRowFieldsFragment': CustomFieldDefinitionTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CustomerPaymentTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type CustomerPaymentTableQuery = { customerPayments: { totalCount: number | null, edges: Array<{ node: { id: string, organizationId: string, businessUnitId: string, customerId: string, paymentDate: number, accountingDate: number, amountMinor: number, appliedAmountMinor: number, unappliedAmountMinor: number, status: CustomerPaymentStatus, paymentMethod: CustomerPaymentMethod, referenceNumber: string, memo: string, currencyCode: string, postedBatchId: string | null, reversalBatchId: string | null, reversedById: string | null, reversedAt: number | null, reversalReason: string, createdById: string, updatedById: string | null, version: number, createdAt: number, updatedAt: number, customer: { id: string, code: string, name: string } | null, applications: Array<{ id: string, customerPaymentId: string, invoiceId: string, appliedAmountMinor: number, shortPayAmountMinor: number, lineNumber: number, createdAt: number, updatedAt: number }> | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CustomerPaymentDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type CustomerPaymentDetailQuery = { customerPayment: { id: string, organizationId: string, businessUnitId: string, customerId: string, paymentDate: number, accountingDate: number, amountMinor: number, appliedAmountMinor: number, unappliedAmountMinor: number, status: CustomerPaymentStatus, paymentMethod: CustomerPaymentMethod, referenceNumber: string, memo: string, currencyCode: string, postedBatchId: string | null, reversalBatchId: string | null, reversedById: string | null, reversedAt: number | null, reversalReason: string, createdById: string, updatedById: string | null, version: number, createdAt: number, updatedAt: number, customer: { id: string, code: string, name: string } | null, applications: Array<{ id: string, customerPaymentId: string, invoiceId: string, appliedAmountMinor: number, shortPayAmountMinor: number, lineNumber: number, createdAt: number, updatedAt: number, invoice: { id: string, number: string, invoiceDate: number, dueDate: number | null, totalAmount: string, appliedAmount: string, settlementStatus: InvoiceSettlementStatus, disputeStatus: InvoiceDisputeStatus, billToName: string } | null }> | null } | null };

export type PostAndApplyCustomerPaymentMutationVariables = Exact<{
  input: PostCustomerPaymentInput;
}>;


export type PostAndApplyCustomerPaymentMutation = { postAndApplyCustomerPayment: { id: string, customerId: string, paymentDate: number, accountingDate: number, amountMinor: number, appliedAmountMinor: number, unappliedAmountMinor: number, status: CustomerPaymentStatus, paymentMethod: CustomerPaymentMethod, referenceNumber: string, memo: string, currencyCode: string, postedBatchId: string | null, createdAt: number, updatedAt: number, applications: Array<{ id: string, invoiceId: string, appliedAmountMinor: number, shortPayAmountMinor: number, lineNumber: number }> | null } };

export type ApplyUnappliedCustomerPaymentMutationVariables = Exact<{
  input: ApplyCustomerPaymentInput;
}>;


export type ApplyUnappliedCustomerPaymentMutation = { applyUnappliedCustomerPayment: { id: string, customerId: string, amountMinor: number, appliedAmountMinor: number, unappliedAmountMinor: number, status: CustomerPaymentStatus, updatedAt: number, applications: Array<{ id: string, invoiceId: string, appliedAmountMinor: number, shortPayAmountMinor: number, lineNumber: number }> | null } };

export type ReverseCustomerPaymentMutationVariables = Exact<{
  input: ReverseCustomerPaymentInput;
}>;


export type ReverseCustomerPaymentMutation = { reverseCustomerPayment: { id: string, customerId: string, amountMinor: number, appliedAmountMinor: number, unappliedAmountMinor: number, status: CustomerPaymentStatus, reversalBatchId: string | null, reversedById: string | null, reversedAt: number | null, reversalReason: string, updatedAt: number, applications: Array<{ id: string, invoiceId: string, appliedAmountMinor: number, shortPayAmountMinor: number, lineNumber: number }> | null } };

export type CustomerBillingProfileFieldsFragment = { id: string, businessUnitId: string, organizationId: string, customerId: string, billingCycleType: CustomerBillingCycleType, billingCycleDayOfWeek: number | null, paymentTerm: CustomerPaymentTerm, hasBillingControlOverrides: boolean, creditLimit: string | null, creditBalance: string, creditStatus: CustomerCreditStatus, enforceCreditLimit: boolean, autoCreditHold: boolean, creditHoldReason: string, invoiceMethod: CustomerInvoiceMethod, autoSendInvoiceOnGeneration: boolean, allowInvoiceConsolidation: boolean, consolidationPeriodDays: number, consolidationGroupBy: CustomerConsolidationGroupBy, invoiceNumberFormat: CustomerInvoiceNumberFormat, customerInvoicePrefix: string, invoiceCopies: number, revenueAccountId: string | null, arAccountId: string | null, applyLateCharges: boolean, lateChargeRate: string | null, gracePeriodDays: number, taxExempt: boolean, taxExemptNumber: string, enforceCustomerBillingReq: boolean, validateCustomerRates: boolean, autoTransfer: boolean, autoMarkReadyToBill: boolean, autoBill: boolean, detentionBillingEnabled: boolean, detentionFreeMinutes: number, detentionRatePerHour: string | null, countLateOnlyOnAppointmentStops: boolean, countDetentionOnlyOnAppointmentStops: boolean, autoApplyAccessorials: boolean, billingCurrency: string, requirePONumber: boolean, requireBOLNumber: boolean, requireDeliveryNumber: boolean, invoiceAdjustmentSupportingDocumentPolicy: CustomerInvoiceAdjustmentSupportingDocumentPolicy, defaultBillerId: string | null, billingNotes: string, fuelSurchargeMode: CustomerFuelSurchargeMode, fuelSurchargeProgramId: string | null, version: number, createdAt: number, updatedAt: number, documentTypes: Array<{ id: string, code: string, name: string, color: string, documentClassification: DocumentClassification, documentCategory: DocumentCategory }> | null } & { ' $fragmentName'?: 'CustomerBillingProfileFieldsFragment' };

export type CustomerEmailProfileFieldsFragment = { id: string, businessUnitId: string, organizationId: string, customerId: string, subject: string, comment: string, fromEmail: string, toRecipients: string, ccRecipients: string, bccRecipients: string, attachmentName: string, readReceipt: boolean, includeShipmentDetail: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CustomerEmailProfileFieldsFragment' };

export type CustomerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, stateId: string, status: EntityStatus, code: string, name: string, addressLine1: string | null, addressLine2: string | null, city: string | null, postalCode: string, isGeocoded: boolean, longitude: number | null, latitude: number | null, placeId: string | null, externalId: string | null, allowConsolidation: boolean, exclusiveConsolidation: boolean, consolidationPriority: number, version: number, createdAt: number, updatedAt: number, billingProfile: { ' $fragmentRefs'?: { 'CustomerBillingProfileFieldsFragment': CustomerBillingProfileFieldsFragment } } | null, emailProfile: { ' $fragmentRefs'?: { 'CustomerEmailProfileFieldsFragment': CustomerEmailProfileFieldsFragment } } | null } & { ' $fragmentName'?: 'CustomerTableRowFieldsFragment' };

export type CustomerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type CustomerTableQuery = { customers: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'CustomerTableRowFieldsFragment': CustomerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DistanceOverrideLocationFieldsFragment = { id: string, name: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, state: { id: string, abbreviation: string } | null } & { ' $fragmentName'?: 'DistanceOverrideLocationFieldsFragment' };

export type DistanceOverrideTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, originLocationId: string, destinationLocationId: string, customerId: string | null, distance: number, version: number, createdAt: number, updatedAt: number, originLocation: { ' $fragmentRefs'?: { 'DistanceOverrideLocationFieldsFragment': DistanceOverrideLocationFieldsFragment } } | null, destinationLocation: { ' $fragmentRefs'?: { 'DistanceOverrideLocationFieldsFragment': DistanceOverrideLocationFieldsFragment } } | null, customer: { id: string, name: string } | null, intermediateStops: Array<{ locationId: string, stopOrder: number }> | null } & { ' $fragmentName'?: 'DistanceOverrideTableRowFieldsFragment' };

export type DistanceOverrideTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type DistanceOverrideTableQuery = { distanceOverrides: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DistanceOverrideTableRowFieldsFragment': DistanceOverrideTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DistanceProfileTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, status: string, isDefault: boolean, provider: string, dataVersion: string, region: string, routingType: string, distanceUnits: string, locationGranularity: string, profileName: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'DistanceProfileTableRowFieldsFragment' };

export type DistanceProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type DistanceProfileTableQuery = { distanceProfiles: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DistanceProfileTableRowFieldsFragment': DistanceProfileTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DocumentPacketRuleTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, resourceType: string, documentTypeId: string, required: boolean, allowMultiple: boolean, displayOrder: number, expirationRequired: boolean, expirationWarningDays: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'DocumentPacketRuleTableRowFieldsFragment' };

export type DocumentPacketRuleTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type DocumentPacketRuleTableQuery = { documentPacketRules: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DocumentPacketRuleTableRowFieldsFragment': DocumentPacketRuleTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DocumentTypeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string, color: string, documentClassification: DocumentClassification, documentCategory: DocumentCategory, isSystem: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'DocumentTypeTableRowFieldsFragment' };

export type DocumentTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type DocumentTypeTableQuery = { documentTypes: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DocumentTypeTableRowFieldsFragment': DocumentTypeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type WorkerPortalStatusQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerPortalStatusQuery = { workerPortalStatus: { linked: boolean, portalUser: { id: string, name: string, emailAddress: string, status: EntityStatus, lastLoginAt: number | null } | null, pendingInvitation: { id: string, email: string, status: PortalInvitationStatus, expiresAt: number, createdAt: number } | null, invitations: Array<{ id: string, email: string, status: PortalInvitationStatus, expiresAt: number, acceptedAt: number | null, createdAt: number, invitedBy: { id: string, name: string } | null }> } };

export type InviteWorkerToPortalMutationVariables = Exact<{
  input: InviteWorkerToPortalInput;
}>;


export type InviteWorkerToPortalMutation = { inviteWorkerToPortal: { inviteUrl: string, emailSent: boolean, invitation: { id: string, email: string, status: PortalInvitationStatus, expiresAt: number } } };

export type RevokeWorkerPortalAccessMutationVariables = Exact<{
  workerId: string | number;
}>;


export type RevokeWorkerPortalAccessMutation = { revokeWorkerPortalAccess: boolean };

export type SettlementDisputeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type SettlementDisputeTableQuery = { settlementDisputes: { totalCount: number | null, edges: Array<{ node: { id: string, settlementId: string, settlementLineId: string | null, workerId: string, status: SettlementDisputeStatus, category: SettlementDisputeCategory, description: string, resolutionNote: string, resolvedAt: number | null, createdAt: number, updatedAt: number, version: number, worker: { id: string, firstName: string, lastName: string } | null, settlement: { id: string, settlementNumber: string, netPayMinor: number, currencyCode: string, status: DriverSettlementStatus } | null, resolvedBy: { id: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type SettlementDisputeDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type SettlementDisputeDetailQuery = { settlementDispute: { id: string, settlementId: string, settlementLineId: string | null, workerId: string, status: SettlementDisputeStatus, category: SettlementDisputeCategory, description: string, submittedByUserId: string, resolutionNote: string, resolutionLineId: string | null, resolvedById: string | null, resolvedAt: number | null, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null, settlement: { id: string, settlementNumber: string, status: DriverSettlementStatus, periodStart: number, periodEnd: number, netPayMinor: number, grossEarningsMinor: number, deductionsMinor: number, currencyCode: string } | null, settlementLine: { id: string, lineNumber: number, category: SettlementLineCategory, description: string, amountMinor: number, proNumber: string } | null, resolvedBy: { id: string, name: string } | null } };

export type OpenSettlementDisputeCountQueryVariables = Exact<{ [key: string]: never; }>;


export type OpenSettlementDisputeCountQuery = { openSettlementDisputeCount: number };

export type StartSettlementDisputeReviewMutationVariables = Exact<{
  id: string | number;
}>;


export type StartSettlementDisputeReviewMutation = { startSettlementDisputeReview: { id: string, status: SettlementDisputeStatus, version: number } };

export type ResolveSettlementDisputeMutationVariables = Exact<{
  input: ResolveSettlementDisputeInput;
}>;


export type ResolveSettlementDisputeMutation = { resolveSettlementDispute: { id: string, status: SettlementDisputeStatus, resolutionNote: string, resolutionLineId: string | null, resolvedAt: number | null, version: number } };

export type MyPortalProfileQueryVariables = Exact<{ [key: string]: never; }>;


export type MyPortalProfileQuery = { myPortalProfile: { workerId: string, firstName: string, lastName: string, email: string, phoneNumber: string, workerType: string, driverType: string, fleetCodeName: string, organizationName: string } };

export type MyLoadsQueryVariables = Exact<{
  scope: PortalLoadScope;
  limit?: number | null | undefined;
}>;


export type MyLoadsQuery = { myLoads: Array<{ assignmentId: string, moveId: string, shipmentId: string, proNumber: string, bol: string, status: string, isPrimary: boolean, tractorCode: string, trailerCode: string, pieces: number | null, weight: number | null, distanceMiles: number | null, payGrossMinor: number | null, payStatus: string, payOnHold: boolean, ackStatus: string, stops: Array<{ id: string, type: string, status: string, sequence: number, locationName: string, addressLine: string, scheduledWindowStart: number, scheduledWindowEnd: number | null, actualArrival: number | null, actualDeparture: number | null }> }> };

export type MyLoadCommentsQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type MyLoadCommentsQuery = { myLoadComments: Array<{ id: string, type: string, priority: string, comment: string, authorName: string, createdAt: number }> };

export type RecordMyStopActionMutationVariables = Exact<{
  input: RecordMyStopActionInput;
}>;


export type RecordMyStopActionMutation = { recordMyStopAction: boolean };

export type CreateMyLoadCommentMutationVariables = Exact<{
  input: CreateMyLoadCommentInput;
}>;


export type CreateMyLoadCommentMutation = { createMyLoadComment: { id: string, type: string, priority: string, comment: string, authorName: string, createdAt: number } };

export type MyPeriodSummaryQueryVariables = Exact<{ [key: string]: never; }>;


export type MyPeriodSummaryQuery = { myPeriodSummary: { periodStart: number, periodEnd: number, payDate: number, accruedGrossMinor: number, eventCount: number } };

export type MyRecentPayEventsQueryVariables = Exact<{
  limit?: number | null | undefined;
}>;


export type MyRecentPayEventsQuery = { myRecentPayEvents: Array<{ id: string, status: DriverPayEventStatus, eventDate: number, proNumber: string, grossAmountMinor: number, totalMiles: string, currencyCode: string, onHold: boolean, holdReason: string }> };

export type MySettlementsQueryVariables = Exact<{
  limit?: number | null | undefined;
  offset?: number | null | undefined;
}>;


export type MySettlementsQuery = { mySettlements: { total: number, items: Array<{ id: string, settlementNumber: string, status: DriverSettlementStatus, periodStart: number, periodEnd: number, payDate: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, netPayMinor: number, currencyCode: string, paidAt: number | null, paymentMethod: string, paymentReference: string }> } };

export type MySettlementQueryVariables = Exact<{
  id: string | number;
}>;


export type MySettlementQuery = { mySettlement: { id: string, settlementNumber: string, status: DriverSettlementStatus, classification: PayeeClassification, payProfileName: string, periodStart: number, periodEnd: number, payDate: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, carryForwardInMinor: number, carryForwardOutMinor: number, netPayMinor: number, totalMiles: string, shipmentCount: number, currencyCode: string, paidAt: number | null, paymentMethod: string, paymentReference: string, createdAt: number, lines: Array<{ id: string, lineNumber: number, category: SettlementLineCategory, componentKind: PayComponentKind | null, method: PayCalcMethod | null, description: string, quantity: string, rate: string, amountMinor: number, proNumber: string }> | null } };

export type MyEscrowQueryVariables = Exact<{ [key: string]: never; }>;


export type MyEscrowQuery = { myEscrow: { account: { id: string, status: EscrowAccountStatus, targetAmountMinor: number, balanceMinor: number, currencyCode: string, createdAt: number } | null, transactions: Array<{ id: string, type: EscrowTransactionType, amountMinor: number, balanceAfterMinor: number, description: string, occurredDate: number, createdAt: number }> } };

export type MyAdvancesQueryVariables = Exact<{ [key: string]: never; }>;


export type MyAdvancesQuery = { myAdvances: Array<{ id: string, status: PayAdvanceStatus, source: PayAdvanceSource, reference: string, amountMinor: number, recoveredMinor: number, outstandingMinor: number, currencyCode: string, issuedDate: number }> };

export type MyDisputesQueryVariables = Exact<{ [key: string]: never; }>;


export type MyDisputesQuery = { myDisputes: Array<{ id: string, settlementId: string, settlementLineId: string | null, status: SettlementDisputeStatus, category: SettlementDisputeCategory, description: string, resolutionNote: string, resolvedAt: number | null, createdAt: number, settlement: { id: string, settlementNumber: string, periodStart: number, periodEnd: number } | null, settlementLine: { id: string, description: string, amountMinor: number, category: SettlementLineCategory } | null }> };

export type CreateSettlementDisputeMutationVariables = Exact<{
  input: CreateSettlementDisputeInput;
}>;


export type CreateSettlementDisputeMutation = { createSettlementDispute: { id: string, status: SettlementDisputeStatus, category: SettlementDisputeCategory, description: string, createdAt: number } };

export type WithdrawSettlementDisputeMutationVariables = Exact<{
  id: string | number;
}>;


export type WithdrawSettlementDisputeMutation = { withdrawSettlementDispute: { id: string, status: SettlementDisputeStatus } };

export type MyComplianceProfileQueryVariables = Exact<{ [key: string]: never; }>;


export type MyComplianceProfileQuery = { myComplianceProfile: { workerId: string, licenseNumber: string, licenseState: string, cdlClass: string, endorsement: string, licenseExpiry: number, hazmatExpiry: number | null, medicalCardExpiry: number | null, physicalDueDate: number | null, mvrDueDate: number | null, twicExpiry: number | null, complianceStatus: string, isQualified: boolean, hireDate: number, addressLine1: string, addressLine2: string, city: string, stateAbbreviation: string, postalCode: string, phoneNumber: string, emergencyContactName: string, emergencyContactPhone: string } };

export type UpdateMyContactInfoMutationVariables = Exact<{
  input: UpdateMyContactInfoInput;
}>;


export type UpdateMyContactInfoMutation = { updateMyContactInfo: { workerId: string, addressLine1: string, addressLine2: string, city: string, stateAbbreviation: string, postalCode: string, phoneNumber: string, emergencyContactName: string, emergencyContactPhone: string } };

export type MyPtoQueryVariables = Exact<{ [key: string]: never; }>;


export type MyPtoQuery = { myPto: Array<{ id: string, status: PortalPtoStatus, type: PortalPtoType, startDate: number, endDate: number, reason: string, createdAt: number }> };

export type RequestMyPtoMutationVariables = Exact<{
  input: RequestMyPtoInput;
}>;


export type RequestMyPtoMutation = { requestMyPto: { id: string, status: PortalPtoStatus, type: PortalPtoType, startDate: number, endDate: number, reason: string, createdAt: number } };

export type CancelMyPtoMutationVariables = Exact<{
  id: string | number;
}>;


export type CancelMyPtoMutation = { cancelMyPto: { id: string, status: PortalPtoStatus } };

export type MyExpensesQueryVariables = Exact<{ [key: string]: never; }>;


export type MyExpensesQuery = { myExpenses: Array<{ id: string, shipmentId: string | null, payCodeId: string | null, status: DriverExpenseStatus, amountMinor: number, currencyCode: string, description: string, incurredDate: number, receiptDocumentId: string | null, reviewNote: string, reviewedAt: number | null, createdAt: number, payCode: { id: string, code: string, description: string } | null }> };

export type SubmitMyExpenseMutationVariables = Exact<{
  input: SubmitMyExpenseInput;
}>;


export type SubmitMyExpenseMutation = { submitMyExpense: { id: string, status: DriverExpenseStatus, amountMinor: number, description: string, incurredDate: number, createdAt: number } };

export type CancelMyExpenseMutationVariables = Exact<{
  id: string | number;
}>;


export type CancelMyExpenseMutation = { cancelMyExpense: { id: string, status: DriverExpenseStatus } };

export type RespondToMyAssignmentMutationVariables = Exact<{
  input: RespondToMyAssignmentInput;
}>;


export type RespondToMyAssignmentMutation = { respondToMyAssignment: boolean };

export type MyLoadPayEstimateQueryVariables = Exact<{
  shipmentId: string | number;
  moveId: string | number;
}>;


export type MyLoadPayEstimateQuery = { myLoadPayEstimate: { grossMinor: number, currencyCode: string } };

export type MyYtdPayQueryVariables = Exact<{
  year: number;
}>;


export type MyYtdPayQuery = { myYtdPay: { workerId: string, year: number, settlementCount: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, netPayMinor: number } };

export type DriverExpenseTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type DriverExpenseTableQuery = { driverExpenses: { totalCount: number | null, edges: Array<{ node: { id: string, workerId: string, shipmentId: string | null, status: DriverExpenseStatus, amountMinor: number, currencyCode: string, description: string, incurredDate: number, receiptDocumentId: string | null, reviewNote: string, reviewedAt: number | null, settlementLineId: string | null, createdAt: number, version: number, worker: { id: string, firstName: string, lastName: string } | null, payCode: { id: string, code: string, description: string } | null, reviewedBy: { id: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DriverExpenseDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type DriverExpenseDetailQuery = { driverExpense: { id: string, workerId: string, shipmentId: string | null, payCodeId: string | null, status: DriverExpenseStatus, amountMinor: number, currencyCode: string, description: string, incurredDate: number, receiptDocumentId: string | null, reviewNote: string, reviewedById: string | null, reviewedAt: number | null, settlementLineId: string | null, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string, email: string, phoneNumber: string } | null, payCode: { id: string, code: string, description: string } | null, reviewedBy: { id: string, name: string } | null } };

export type PendingDriverExpenseCountQueryVariables = Exact<{ [key: string]: never; }>;


export type PendingDriverExpenseCountQuery = { pendingDriverExpenseCount: number };

export type ReviewDriverExpenseMutationVariables = Exact<{
  input: ReviewDriverExpenseInput;
}>;


export type ReviewDriverExpenseMutation = { reviewDriverExpense: { id: string, status: DriverExpenseStatus, reviewNote: string, reviewedAt: number | null, settlementLineId: string | null, version: number } };

export type DashControlQueryVariables = Exact<{ [key: string]: never; }>;


export type DashControlQuery = { dashControl: { id: string, requireLoadAcknowledgment: boolean, allowLoadRefusals: boolean, allowStopActions: boolean, allowLoadDocumentUpload: boolean, allowLoadComments: boolean, showLoadPay: boolean, showPayEstimates: boolean, allowExpenseSubmission: boolean, requireExpenseReceipt: boolean, allowSettlementDisputes: boolean, allowProfileDocumentUpload: boolean, allowContactInfoEdit: boolean, allowPtoRequests: boolean, sendCredentialReminders: boolean, enableDetentionAlerts: boolean, detentionAlertThresholdMinutes: number, version: number } };

export type UpdateDashControlMutationVariables = Exact<{
  input: UpdateDashControlInput;
}>;


export type UpdateDashControlMutation = { updateDashControl: { id: string, requireLoadAcknowledgment: boolean, allowLoadRefusals: boolean, allowStopActions: boolean, allowLoadDocumentUpload: boolean, allowLoadComments: boolean, showLoadPay: boolean, showPayEstimates: boolean, allowExpenseSubmission: boolean, requireExpenseReceipt: boolean, allowSettlementDisputes: boolean, allowProfileDocumentUpload: boolean, allowContactInfoEdit: boolean, allowPtoRequests: boolean, sendCredentialReminders: boolean, enableDetentionAlerts: boolean, detentionAlertThresholdMinutes: number, version: number } };

export type MyPortalFeaturesQueryVariables = Exact<{ [key: string]: never; }>;


export type MyPortalFeaturesQuery = { myPortalFeatures: { requireLoadAcknowledgment: boolean, allowLoadRefusals: boolean, allowStopActions: boolean, allowLoadDocumentUpload: boolean, allowLoadComments: boolean, showLoadPay: boolean, showPayEstimates: boolean, allowExpenseSubmission: boolean, requireExpenseReceipt: boolean, allowSettlementDisputes: boolean, allowProfileDocumentUpload: boolean, allowContactInfoEdit: boolean, allowPtoRequests: boolean } };

export type PayProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type PayProfileTableQuery = { payProfiles: { totalCount: number | null, edges: Array<{ node: { id: string, organizationId: string, businessUnitId: string, status: EntityStatus, name: string, description: string, classification: PayeeClassification, currencyCode: string, guaranteedPeriodMinimumMinor: number, perDiemRatePerMile: string, perDiemDailyCapMinor: number, version: number, createdAt: number, updatedAt: number, activeAssignmentCount: number, components: Array<{ id: string, kind: PayComponentKind, method: PayCalcMethod, description: string, rate: string, revenueBasis: PayRevenueBasis | null, freeTimeMinutes: number, minAmountMinor: number | null, maxAmountMinor: number | null, sequence: number, isActive: boolean, bands: Array<{ minMiles: number, maxMiles: number, rate: string }> | null }> | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type PayProfileOptionsQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type PayProfileOptionsQuery = { payProfiles: { totalCount: number | null, edges: Array<{ node: { id: string, name: string, classification: PayeeClassification, status: EntityStatus } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type WorkerPayAssignmentsQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerPayAssignmentsQuery = { workerPayAssignments: Array<{ id: string, workerId: string, payProfileId: string, effectiveFrom: number, effectiveTo: number | null, splitPercent: string, notes: string, version: number, createdAt: number, rateOverrides: Array<{ componentId: string, rate: string }> | null, payProfile: { id: string, name: string, classification: PayeeClassification, components: Array<{ id: string, kind: PayComponentKind, method: PayCalcMethod, description: string, rate: string }> | null } | null }> };

export type EffectiveWorkerPayAssignmentQueryVariables = Exact<{
  workerId: string | number;
}>;


export type EffectiveWorkerPayAssignmentQuery = { effectiveWorkerPayAssignment: { id: string, workerId: string, payProfileId: string, effectiveFrom: number, effectiveTo: number | null, splitPercent: string, notes: string, rateOverrides: Array<{ componentId: string, rate: string }> | null, payProfile: { id: string, name: string, classification: PayeeClassification, guaranteedPeriodMinimumMinor: number, components: Array<{ id: string, kind: PayComponentKind, method: PayCalcMethod, description: string, rate: string, revenueBasis: PayRevenueBasis | null, isActive: boolean, bands: Array<{ minMiles: number, maxMiles: number, rate: string }> | null }> | null } | null } | null };

export type PayProfileAssignmentsQueryVariables = Exact<{
  payProfileId: string | number;
}>;


export type PayProfileAssignmentsQuery = { payProfileAssignments: Array<{ id: string, workerId: string, effectiveFrom: number, effectiveTo: number | null, splitPercent: string, rateOverrides: Array<{ componentId: string, rate: string }> | null, worker: { id: string, firstName: string, lastName: string } | null }> };

export type PayProfileDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type PayProfileDetailQuery = { payProfile: { id: string, name: string, classification: PayeeClassification, currencyCode: string, components: Array<{ id: string, kind: PayComponentKind, method: PayCalcMethod, description: string, rate: string, revenueBasis: PayRevenueBasis | null, isActive: boolean }> | null } | null };

export type RecurringDeductionTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type RecurringDeductionTableQuery = { recurringDeductions: { totalCount: number | null, edges: Array<{ node: { id: string, workerId: string, payCodeId: string, escrowAccountId: string | null, status: RecurringDeductionStatus, frequency: RecurringDeductionFrequency, description: string, amountMinor: number, totalCapMinor: number | null, deductedToDateMinor: number, startDate: number, endDate: number | null, currencyCode: string, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null, payCode: { id: string, code: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RecurringEarningTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type RecurringEarningTableQuery = { recurringEarnings: { totalCount: number | null, edges: Array<{ node: { id: string, workerId: string, payCodeId: string, status: RecurringEarningStatus, frequency: RecurringEarningFrequency, description: string, amountMinor: number, totalCapMinor: number | null, paidToDateMinor: number, startDate: number, endDate: number | null, currencyCode: string, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null, payCode: { id: string, code: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type PayCodeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type PayCodeTableQuery = { payCodes: { totalCount: number | null, edges: Array<{ node: { id: string, status: EntityStatus, direction: PayCodeDirection, code: string, name: string, description: string, taxable: boolean, countsTowardGuarantee: boolean, glAccountId: string | null, defaultAmountMinor: number | null, isSystem: boolean, version: number, createdAt: number, updatedAt: number, glAccount: { id: string, accountCode: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type PayCodeOptionsQueryVariables = Exact<{
  direction?: PayCodeDirection | null | undefined;
}>;


export type PayCodeOptionsQuery = { payCodeOptions: Array<{ id: string, direction: PayCodeDirection, code: string, name: string, taxable: boolean, defaultAmountMinor: number | null }> };

export type PayAdvanceTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type PayAdvanceTableQuery = { payAdvances: { totalCount: number | null, edges: Array<{ node: { id: string, workerId: string, status: PayAdvanceStatus, source: PayAdvanceSource, reference: string, issuedDate: number, amountMinor: number, recoveredMinor: number, writtenOffMinor: number, outstandingMinor: number, writeOffReason: string, notes: string, currencyCode: string, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EscrowAccountTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type EscrowAccountTableQuery = { escrowAccounts: { totalCount: number | null, edges: Array<{ node: { id: string, workerId: string, status: EscrowAccountStatus, targetAmountMinor: number, balanceMinor: number, annualInterestRate: string, lastInterestAccrualDate: number | null, openedDate: number, closedDate: number | null, currencyCode: string, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EscrowAccountDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type EscrowAccountDetailQuery = { escrowAccount: { id: string, workerId: string, status: EscrowAccountStatus, targetAmountMinor: number, balanceMinor: number, annualInterestRate: string, lastInterestAccrualDate: number | null, openedDate: number, closedDate: number | null, currencyCode: string, version: number, worker: { id: string, firstName: string, lastName: string } | null, transactions: Array<{ id: string, type: EscrowTransactionType, amountMinor: number, balanceAfterMinor: number, occurredDate: number, description: string, settlementId: string | null, createdAt: number }> | null } | null };

export type DriverSettlementTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type DriverSettlementTableQuery = { driverSettlements: { totalCount: number | null, edges: Array<{ node: { id: string, workerId: string, batchId: string | null, settlementNumber: string, status: DriverSettlementStatus, classification: PayeeClassification, payProfileName: string, periodStart: number, periodEnd: number, payDate: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, carryForwardInMinor: number, carryForwardOutMinor: number, netPayMinor: number, totalMiles: string, shipmentCount: number, currencyCode: string, hasExceptions: boolean, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DriverSettlementDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type DriverSettlementDetailQuery = { driverSettlement: { id: string, workerId: string, batchId: string | null, payProfileId: string | null, settlementNumber: string, status: DriverSettlementStatus, classification: PayeeClassification, payProfileName: string, periodStart: number, periodEnd: number, payDate: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, carryForwardInMinor: number, carryForwardOutMinor: number, netPayMinor: number, totalMiles: string, shipmentCount: number, currencyCode: string, hasExceptions: boolean, notes: string, submittedById: string | null, submittedAt: number | null, approvedById: string | null, approvedAt: number | null, postedById: string | null, postedAt: number | null, paidAt: number | null, paymentMethod: string, paymentReference: string, voidedById: string | null, voidedAt: number | null, voidReason: string, version: number, createdAt: number, updatedAt: number, exceptions: Array<{ code: string, severity: string, message: string }> | null, worker: { id: string, firstName: string, lastName: string } | null, lines: Array<{ id: string, lineNumber: number, category: SettlementLineCategory, componentKind: PayComponentKind | null, method: PayCalcMethod | null, description: string, quantity: string, rate: string, amountMinor: number, shipmentId: string | null, moveId: string | null, payEventId: string | null, recurringDeductionId: string | null, advanceId: string | null, escrowAccountId: string | null, proNumber: string }> | null } | null };

export type SettlementBatchTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type SettlementBatchTableQuery = { settlementBatches: { totalCount: number | null, edges: Array<{ node: { id: string, status: SettlementBatchStatus, name: string, periodStart: number, periodEnd: number, payDate: number, settlementCount: number, exceptionCount: number, totalGrossMinor: number, totalNetMinor: number, currencyCode: string, notes: string, generatedById: string | null, generatedAt: number | null, completedAt: number | null, canceledAt: number | null, version: number, createdAt: number, updatedAt: number } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DriverPayEventTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type DriverPayEventTableQuery = { driverPayEvents: { totalCount: number | null, edges: Array<{ node: { id: string, workerId: string, shipmentId: string, moveId: string | null, settlementId: string | null, status: DriverPayEventStatus, eventDate: number, grossAmountMinor: number, totalMiles: string, currencyCode: string, proNumber: string, onHold: boolean, holdReason: string, voidedAt: number | null, voidReason: string, version: number, createdAt: number, updatedAt: number, components: Array<{ kind: PayComponentKind, method: PayCalcMethod, description: string, quantity: string, rate: string, amountMinor: number }> | null, worker: { id: string, firstName: string, lastName: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type WorkerEarningsSummaryQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerEarningsSummaryQuery = { workerEarningsSummary: { workerId: string, accruedEventCount: number, accruedGrossMinor: number, outstandingAdvances: number, escrowBalanceMinor: number } };

export type WorkerYtdPaySummariesQueryVariables = Exact<{
  year: number;
  classification?: PayeeClassification | null | undefined;
}>;


export type WorkerYtdPaySummariesQuery = { workerYtdPaySummaries: Array<{ workerId: string, workerName: string, classification: PayeeClassification, year: number, settlementCount: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, netPayMinor: number }> };

export type SettlementControlQueryVariables = Exact<{ [key: string]: never; }>;


export type SettlementControlQuery = { settlementControl: { id: string, organizationId: string, businessUnitId: string, payPeriodFrequency: PayPeriodFrequency, periodEndDayOfWeek: number, payDelayDays: number, payTrigger: SettlementPayTrigger, autoGenerateBatches: boolean, autoApproveClean: boolean, autoAttachAccruals: boolean, autoPostOnApprove: boolean, allowNegativeNet: boolean, varianceThresholdPct: string, varianceLookbackWeeks: number, defaultEscrowInterestRate: string, escrowInterestFrequencyMonths: number, version: number } };

export type SettlementWorkspaceSummaryQueryVariables = Exact<{
  periodStart?: number | null | undefined;
  periodEnd?: number | null | undefined;
}>;


export type SettlementWorkspaceSummaryQuery = { settlementWorkspaceSummary: { periodStart: number, periodEnd: number, payDate: number, draftCount: number, pendingApprovalCount: number, approvedCount: number, postedCount: number, paidCount: number, exceptionCount: number, totalNetMinor: number, totalGrossMinor: number, unsettledEventCount: number, unsettledGrossMinor: number, heldEventCount: number, heldGrossMinor: number, unsettledWorkerCount: number, openBatchId: string | null } };

export type UnsettledWorkerSummariesQueryVariables = Exact<{
  periodStart?: number | null | undefined;
  periodEnd?: number | null | undefined;
}>;


export type UnsettledWorkerSummariesQuery = { unsettledWorkerSummaries: Array<{ workerId: string, workerName: string, eventCount: number, grossAmountMinor: number, heldCount: number, heldGrossMinor: number, hasSettlement: boolean }> };

export type CurrentSettlementPeriodQueryVariables = Exact<{ [key: string]: never; }>;


export type CurrentSettlementPeriodQuery = { currentSettlementPeriod: { periodStart: number, periodEnd: number, payDate: number } };

export type PreviewDriverSettlementQueryVariables = Exact<{
  workerId: string | number;
  periodStart?: number | null | undefined;
  periodEnd?: number | null | undefined;
}>;


export type PreviewDriverSettlementQuery = { previewDriverSettlement: { id: string, workerId: string, settlementNumber: string, status: DriverSettlementStatus, classification: PayeeClassification, payProfileName: string, periodStart: number, periodEnd: number, payDate: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, carryForwardInMinor: number, carryForwardOutMinor: number, netPayMinor: number, totalMiles: string, shipmentCount: number, currencyCode: string, hasExceptions: boolean, exceptions: Array<{ code: string, severity: string, message: string }> | null, lines: Array<{ lineNumber: number, category: SettlementLineCategory, componentKind: PayComponentKind | null, method: PayCalcMethod | null, description: string, quantity: string, rate: string, amountMinor: number, proNumber: string }> | null } };

export type ExportSettlementBatchCsvQueryVariables = Exact<{
  batchId: string | number;
}>;


export type ExportSettlementBatchCsvQuery = { exportSettlementBatchCsv: string };

export type CreatePayProfileMutationVariables = Exact<{
  input: CreatePayProfileInput;
}>;


export type CreatePayProfileMutation = { createPayProfile: { id: string, name: string, version: number } };

export type UpdatePayProfileMutationVariables = Exact<{
  input: UpdatePayProfileInput;
}>;


export type UpdatePayProfileMutation = { updatePayProfile: { id: string, name: string, version: number } };

export type AssignPayProfileToWorkerMutationVariables = Exact<{
  input: AssignPayProfileInput;
}>;


export type AssignPayProfileToWorkerMutation = { assignPayProfileToWorker: { id: string, workerId: string, payProfileId: string, effectiveFrom: number, effectiveTo: number | null } };

export type EndWorkerPayAssignmentMutationVariables = Exact<{
  input: EndWorkerPayAssignmentInput;
}>;


export type EndWorkerPayAssignmentMutation = { endWorkerPayAssignment: { id: string, effectiveTo: number | null } };

export type CreateRecurringDeductionMutationVariables = Exact<{
  input: CreateRecurringDeductionInput;
}>;


export type CreateRecurringDeductionMutation = { createRecurringDeduction: { id: string, version: number } };

export type UpdateRecurringDeductionMutationVariables = Exact<{
  input: UpdateRecurringDeductionInput;
}>;


export type UpdateRecurringDeductionMutation = { updateRecurringDeduction: { id: string, version: number } };

export type CreatePayCodeMutationVariables = Exact<{
  input: CreatePayCodeInput;
}>;


export type CreatePayCodeMutation = { createPayCode: { id: string, version: number } };

export type UpdatePayCodeMutationVariables = Exact<{
  input: UpdatePayCodeInput;
}>;


export type UpdatePayCodeMutation = { updatePayCode: { id: string, version: number } };

export type CreateRecurringEarningMutationVariables = Exact<{
  input: CreateRecurringEarningInput;
}>;


export type CreateRecurringEarningMutation = { createRecurringEarning: { id: string, version: number } };

export type UpdateRecurringEarningMutationVariables = Exact<{
  input: UpdateRecurringEarningInput;
}>;


export type UpdateRecurringEarningMutation = { updateRecurringEarning: { id: string, version: number } };

export type IssuePayAdvanceMutationVariables = Exact<{
  input: IssuePayAdvanceInput;
}>;


export type IssuePayAdvanceMutation = { issuePayAdvance: { id: string, version: number } };

export type WriteOffPayAdvanceMutationVariables = Exact<{
  input: WriteOffPayAdvanceInput;
}>;


export type WriteOffPayAdvanceMutation = { writeOffPayAdvance: { id: string, status: PayAdvanceStatus, version: number } };

export type OpenEscrowAccountMutationVariables = Exact<{
  input: OpenEscrowAccountInput;
}>;


export type OpenEscrowAccountMutation = { openEscrowAccount: { id: string, version: number } };

export type UpdateEscrowAccountMutationVariables = Exact<{
  input: UpdateEscrowAccountInput;
}>;


export type UpdateEscrowAccountMutation = { updateEscrowAccount: { id: string, version: number } };

export type AdjustEscrowAccountMutationVariables = Exact<{
  input: AdjustEscrowAccountInput;
}>;


export type AdjustEscrowAccountMutation = { adjustEscrowAccount: { id: string, balanceMinor: number, version: number } };

export type CloseEscrowAccountMutationVariables = Exact<{
  accountId: string | number;
}>;


export type CloseEscrowAccountMutation = { closeEscrowAccount: { id: string, status: EscrowAccountStatus, version: number } };

export type GenerateSettlementBatchMutationVariables = Exact<{
  input: GenerateSettlementBatchInput;
}>;


export type GenerateSettlementBatchMutation = { generateSettlementBatch: { id: string, name: string, settlementCount: number, exceptionCount: number, totalGrossMinor: number, totalNetMinor: number } };

export type GenerateDriverSettlementMutationVariables = Exact<{
  input: GenerateDriverSettlementInput;
}>;


export type GenerateDriverSettlementMutation = { generateDriverSettlement: { id: string, settlementNumber: string } | null };

export type SubmitDriverSettlementMutationVariables = Exact<{
  input: DriverSettlementActionInput;
}>;


export type SubmitDriverSettlementMutation = { submitDriverSettlement: { id: string, status: DriverSettlementStatus, version: number } };

export type ApproveDriverSettlementMutationVariables = Exact<{
  input: DriverSettlementActionInput;
}>;


export type ApproveDriverSettlementMutation = { approveDriverSettlement: { id: string, status: DriverSettlementStatus, version: number } };

export type RejectDriverSettlementMutationVariables = Exact<{
  input: DriverSettlementActionInput;
}>;


export type RejectDriverSettlementMutation = { rejectDriverSettlement: { id: string, status: DriverSettlementStatus, version: number } };

export type PostDriverSettlementMutationVariables = Exact<{
  input: DriverSettlementActionInput;
}>;


export type PostDriverSettlementMutation = { postDriverSettlement: { id: string, status: DriverSettlementStatus, version: number } };

export type MarkDriverSettlementPaidMutationVariables = Exact<{
  input: MarkDriverSettlementPaidInput;
}>;


export type MarkDriverSettlementPaidMutation = { markDriverSettlementPaid: { id: string, status: DriverSettlementStatus, version: number } };

export type VoidDriverSettlementMutationVariables = Exact<{
  input: DriverSettlementActionInput;
}>;


export type VoidDriverSettlementMutation = { voidDriverSettlement: { id: string, status: DriverSettlementStatus, version: number } };

export type RecalculateDriverSettlementMutationVariables = Exact<{
  input: DriverSettlementActionInput;
}>;


export type RecalculateDriverSettlementMutation = { recalculateDriverSettlement: { id: string, version: number } };

export type AddDriverSettlementAdjustmentMutationVariables = Exact<{
  input: AddSettlementAdjustmentInput;
}>;


export type AddDriverSettlementAdjustmentMutation = { addDriverSettlementAdjustment: { id: string, version: number } };

export type RemoveDriverSettlementAdjustmentMutationVariables = Exact<{
  input: RemoveSettlementAdjustmentInput;
}>;


export type RemoveDriverSettlementAdjustmentMutation = { removeDriverSettlementAdjustment: { id: string, version: number } };

export type HoldDriverPayEventMutationVariables = Exact<{
  input: HoldPayEventInput;
}>;


export type HoldDriverPayEventMutation = { holdDriverPayEvent: { id: string, status: DriverPayEventStatus, onHold: boolean, holdReason: string, version: number } };

export type ReleaseDriverPayEventMutationVariables = Exact<{
  payEventId: string | number;
}>;


export type ReleaseDriverPayEventMutation = { releaseDriverPayEvent: { id: string, status: DriverPayEventStatus, onHold: boolean, holdReason: string, version: number } };

export type AttachPayEventsToSettlementMutationVariables = Exact<{
  input: AttachPayEventsInput;
}>;


export type AttachPayEventsToSettlementMutation = { attachPayEventsToSettlement: { id: string, status: DriverSettlementStatus, grossEarningsMinor: number, netPayMinor: number, version: number } };

export type DetachPayEventFromSettlementMutationVariables = Exact<{
  input: DetachPayEventInput;
}>;


export type DetachPayEventFromSettlementMutation = { detachPayEventFromSettlement: { id: string, status: DriverSettlementStatus, grossEarningsMinor: number, netPayMinor: number, version: number } };

export type BulkDriverSettlementActionMutationVariables = Exact<{
  input: BulkSettlementActionInput;
}>;


export type BulkDriverSettlementActionMutation = { bulkDriverSettlementAction: { successCount: number, failureCount: number, results: Array<{ settlementId: string, success: boolean, error: string }> } };

export type UpdateSettlementControlMutationVariables = Exact<{
  input: UpdateSettlementControlInput;
}>;


export type UpdateSettlementControlMutation = { updateSettlementControl: { id: string, version: number } };

export type SettlementBatchDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type SettlementBatchDetailQuery = { settlementBatch: { id: string, status: SettlementBatchStatus, name: string, periodStart: number, periodEnd: number, payDate: number, settlementCount: number, exceptionCount: number, totalGrossMinor: number, totalNetMinor: number, currencyCode: string, notes: string, version: number, settlements: Array<{ id: string, settlementNumber: string, status: DriverSettlementStatus, classification: PayeeClassification, grossEarningsMinor: number, deductionsMinor: number, netPayMinor: number, currencyCode: string, hasExceptions: boolean, worker: { id: string, firstName: string, lastName: string } | null }> | null } | null };

export type UnsettledPayEventsQueryVariables = Exact<{
  workerId: string | number;
}>;


export type UnsettledPayEventsQuery = { unsettledPayEvents: Array<{ id: string, shipmentId: string, moveId: string | null, eventDate: number, grossAmountMinor: number, totalMiles: string, currencyCode: string, proNumber: string }> };

export type PayWorkerNowMutationVariables = Exact<{
  input: PayWorkerNowInput;
}>;


export type PayWorkerNowMutation = { payWorkerNow: { id: string, settlementNumber: string, status: DriverSettlementStatus, netPayMinor: number, currencyCode: string, paidAt: number | null, paymentMethod: string, paymentReference: string } };

export type EdiPartnerScorecardsQueryVariables = Exact<{
  sinceHours?: number | null | undefined;
}>;


export type EdiPartnerScorecardsQuery = { ediPartnerScorecards: Array<{ partnerId: string, partnerName: string, partnerCode: string, outboundTotal: number, sentCount: number, failedCount: number, deadLetteredCount: number, receivedCount: number, deliverySuccessRate: number | null, avgAckSeconds: number | null, p95AckSeconds: number | null, overdueAckCount: number, pendingOver4hCount: number, pendingOver24hCount: number, oldestPendingAgeSeconds: number | null }> };

export type EdiVolumeSeriesQueryVariables = Exact<{
  sinceHours?: number | null | undefined;
}>;


export type EdiVolumeSeriesQuery = { ediVolumeSeries: Array<{ bucketStart: number, bucketSeconds: number, outboundCount: number, sentCount: number, failedCount: number, receivedCount: number }> };

export type EdiTemplateVersionSummaryFieldsFragment = { id: string, businessUnitId: string, organizationId: string, templateId: string, sourceVersionId: string | null, versionNumber: number, x12Version: string, functionalGroupId: string, status: EdiTemplateStatus, isActive: boolean, notes: string | null, certifiedAt: number | null, activatedAt: number | null, archivedAt: number | null, deprecatedAt: number | null, supersededAt: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'EdiTemplateVersionSummaryFieldsFragment' };

export type EdiTemplateListFieldsFragment = { id: string, businessUnitId: string, organizationId: string, documentTypeId: string, name: string, description: string | null, direction: EdiDocumentDirection, standard: EdiStandard, transactionSet: string, status: EdiTemplateStatus, version: number, createdAt: number, updatedAt: number, versions: Array<{ ' $fragmentRefs'?: { 'EdiTemplateVersionSummaryFieldsFragment': EdiTemplateVersionSummaryFieldsFragment } }> | null } & { ' $fragmentName'?: 'EdiTemplateListFieldsFragment' };

export type EdiTemplateListQueryVariables = Exact<{
  input: DataTableConnectionInput;
  status?: EdiTemplateStatus | null | undefined;
  transactionSet?: string | null | undefined;
  direction?: EdiDocumentDirection | null | undefined;
}>;


export type EdiTemplateListQuery = { ediTemplates: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiTemplateListFieldsFragment': EdiTemplateListFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiPartnerReadinessQueryVariables = Exact<{
  partnerIds: Array<string | number> | string | number;
}>;


export type EdiPartnerReadinessQuery = { ediPartnerReadiness: Array<{ partnerId: string, ready: boolean, completedCount: number, totalCount: number }> };

export type EdiSummaryQueryVariables = Exact<{
  sinceHours?: number | null | undefined;
}>;


export type EdiSummaryQuery = { ediSummary: { overdueAckCount: number, deliveryStatusCounts: Array<{ status: string, count: number }>, ackStatusCounts: Array<{ status: string, count: number }>, inboundFileStatusCounts: Array<{ status: string, count: number }>, inboundTransferStatusCounts: Array<{ status: string, count: number }>, attentionItems: Array<{ kind: EdiSummaryAttentionKind, id: string, partnerId: string | null, partnerName: string | null, partnerCode: string | null, reference: string | null, error: string | null, occurredAt: number }> } };

export type EdiPartnerRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, kind: EdiPartnerKind, status: EntityStatus, code: string, name: string, description: string | null, internalOrganizationId: string | null, customerId: string | null, defaultTransportId: string | null, defaultMappingProfileId: string | null, country: string, timezone: string | null, contactName: string | null, contactEmail: string | null, contactPhone: string | null, enabledForInbound: boolean, enabledForOutbound: boolean, version: number, createdAt: number, updatedAt: number, internalOrganization: { id: string, name: string } | null, connection: { id: string, method: EdiConnectionMethod, status: EdiConnectionStatus } | null, defaultTransport: { id: string, name: string, method: EdiConnectionMethod } | null } & { ' $fragmentName'?: 'EdiPartnerRowFieldsFragment' };

export type EdiPartnerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type EdiPartnerTableQuery = { ediPartners: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiPartnerRowFieldsFragment': EdiPartnerRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiCommunicationProfileRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, ediPartnerId: string | null, ediConnectionId: string | null, method: EdiConnectionMethod, status: EntityStatus, name: string, description: string | null, version: number, createdAt: number, updatedAt: number, secretState: Array<{ key: string }> | null, partner: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'EdiCommunicationProfileRowFieldsFragment' };

export type EdiCommunicationProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type EdiCommunicationProfileTableQuery = { ediCommunicationProfiles: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiCommunicationProfileRowFieldsFragment': EdiCommunicationProfileRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiTransferRowFieldsFragment = { id: string, sourceOrganizationId: string, sourceBusinessUnitId: string, targetOrganizationId: string, targetBusinessUnitId: string, sourcePartnerId: string, targetPartnerId: string, sourceShipmentId: string | null, targetShipmentId: string | null, inboundMessageId: string | null, status: EdiTransferStatus, tenderPayload: unknown, mappingSnapshot: unknown, rejectionReason: string | null, failureReason: string | null, submittedAt: number, processedAt: number | null, version: number, createdAt: number, updatedAt: number, sourcePartner: { id: string, code: string, name: string } | null, targetPartner: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'EdiTransferRowFieldsFragment' };

export type EdiTransferTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  direction: EdiTransferDirection;
}>;


export type EdiTransferTableQuery = { ediTransfers: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiTransferRowFieldsFragment': EdiTransferRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiMessageRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, ediPartnerId: string, documentTypeId: string, partnerDocumentProfileId: string | null, shipmentId: string | null, transferId: string | null, inboundFileId: string | null, direction: EdiDocumentDirection, transactionSet: string, x12Version: string, status: EdiMessageStatus, interchangeControlNumber: string, groupControlNumber: string, transactionControlNumber: string, segmentCount: number, deliveryStatus: EdiMessageDeliveryStatus | null, deliveryRemotePath: string | null, deliveryAttempts: number, deliveryLastAttemptAt: number | null, deliverySentAt: number | null, deliveryLastError: string | null, ackStatus: EdiMessageAckStatus | null, ackMessageId: string | null, ackReceivedAt: number | null, ackLastError: string | null, generatedAt: number, version: number, partner: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'EdiMessageRowFieldsFragment' };

export type EdiMessageTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type EdiMessageTableQuery = { ediMessages: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiMessageRowFieldsFragment': EdiMessageRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiInboundFileRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, communicationProfileId: string, ediPartnerId: string | null, method: EdiConnectionMethod, remotePath: string, fileName: string, checksum: string, sizeBytes: number, interchangeControlNumber: string | null, isaSenderQualifier: string | null, isaSenderId: string | null, isaReceiverQualifier: string | null, isaReceiverId: string | null, status: EdiInboundFileStatus, failureReason: string | null, transactionCount: number, receivedAt: number, processedAt: number | null, version: number, partner: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'EdiInboundFileRowFieldsFragment' };

export type EdiInboundFileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type EdiInboundFileTableQuery = { ediInboundFiles: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiInboundFileRowFieldsFragment': EdiInboundFileRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiMappingProfileRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, ediPartnerId: string, name: string, description: string | null, version: number, createdAt: number, updatedAt: number, partner: { id: string, code: string, name: string } | null, entries: Array<{ id: string, entityType: EdiMappingEntityType, sourceId: string, sourceLabel: string | null, targetId: string, targetLabel: string | null }> | null } & { ' $fragmentName'?: 'EdiMappingProfileRowFieldsFragment' };

export type EdiMappingProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type EdiMappingProfileTableQuery = { ediMappingProfiles: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiMappingProfileRowFieldsFragment': EdiMappingProfileRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiTestCaseRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, partnerDocumentProfileId: string, name: string, description: string | null, expectedWarnings: number, expectedErrors: number, version: number, createdAt: number, updatedAt: number, documentProfile: { id: string, name: string, direction: EdiDocumentDirection, transactionSet: string, partner: { id: string, code: string, name: string } | null } | null } & { ' $fragmentName'?: 'EdiTestCaseRowFieldsFragment' };

export type EdiTestCaseTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  partnerDocumentProfileId?: string | number | null | undefined;
}>;


export type EdiTestCaseTableQuery = { ediTestCases: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiTestCaseRowFieldsFragment': EdiTestCaseRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EmailProfileTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, senderName: string, senderEmail: string, replyToEmail: string, provider: EmailProvider, status: EmailProfileStatus, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'EmailProfileTableRowFieldsFragment' };

export type EmailProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type EmailProfileTableQuery = { emailProfiles: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EmailProfileTableRowFieldsFragment': EmailProfileTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EquipmentManufacturerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, name: string, description: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'EquipmentManufacturerTableRowFieldsFragment' };

export type EquipmentManufacturerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type EquipmentManufacturerTableQuery = { equipmentManufacturers: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EquipmentManufacturerTableRowFieldsFragment': EquipmentManufacturerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EquipmentTypeTableFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'EquipmentTypeTableFieldsFragment' };

export type EquipmentTypeConfigurationRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, description: string, class: EquipmentClass, color: string, interiorLength: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'EquipmentTypeConfigurationRowFieldsFragment' };

export type EquipmentManufacturerTableFieldsFragment = { id: string, name: string } & { ' $fragmentName'?: 'EquipmentManufacturerTableFieldsFragment' };

export type FleetCodeTableFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'FleetCodeTableFieldsFragment' };

export type UsStateTableFieldsFragment = { id: string, name: string, abbreviation: string } & { ' $fragmentName'?: 'UsStateTableFieldsFragment' };

export type WorkerTableReferenceFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string } & { ' $fragmentName'?: 'WorkerTableReferenceFieldsFragment' };

export type DataTablePageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'DataTablePageInfoFieldsFragment' };

export type TractorTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, primaryWorkerId: string, equipmentTypeId: string, equipmentManufacturerId: string, stateId: string | null, fleetCodeId: string | null, secondaryWorkerId: string | null, status: EquipmentStatus, code: string, model: string, make: string, year: number | null, licensePlateNumber: string, registrationNumber: string, registrationExpiry: number | null, vin: string, lastKnownLocationId: string | null, lastKnownLocationName: string, version: number, createdAt: number, updatedAt: number, customFields: unknown, equipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeTableFieldsFragment': EquipmentTypeTableFieldsFragment } } | null, equipmentManufacturer: { ' $fragmentRefs'?: { 'EquipmentManufacturerTableFieldsFragment': EquipmentManufacturerTableFieldsFragment } } | null, fleetCode: { ' $fragmentRefs'?: { 'FleetCodeTableFieldsFragment': FleetCodeTableFieldsFragment } } | null, state: { ' $fragmentRefs'?: { 'UsStateTableFieldsFragment': UsStateTableFieldsFragment } } | null, primaryWorker: { ' $fragmentRefs'?: { 'WorkerTableReferenceFieldsFragment': WorkerTableReferenceFieldsFragment } } | null, secondaryWorker: { ' $fragmentRefs'?: { 'WorkerTableReferenceFieldsFragment': WorkerTableReferenceFieldsFragment } } | null } & { ' $fragmentName'?: 'TractorTableRowFieldsFragment' };

export type TrailerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, equipmentTypeId: string, equipmentManufacturerId: string, registrationStateId: string | null, fleetCodeId: string | null, status: EquipmentStatus, code: string, model: string, make: string, year: number | null, licensePlateNumber: string, vin: string, registrationNumber: string, maxLoadWeight: number | null, lastInspectionDate: number | null, registrationExpiry: number | null, lastKnownLocationId: string | null, lastKnownLocationName: string, version: number, createdAt: number, updatedAt: number, customFields: unknown, equipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeTableFieldsFragment': EquipmentTypeTableFieldsFragment } } | null, equipmentManufacturer: { ' $fragmentRefs'?: { 'EquipmentManufacturerTableFieldsFragment': EquipmentManufacturerTableFieldsFragment } } | null, fleetCode: { ' $fragmentRefs'?: { 'FleetCodeTableFieldsFragment': FleetCodeTableFieldsFragment } } | null, registrationState: { ' $fragmentRefs'?: { 'UsStateTableFieldsFragment': UsStateTableFieldsFragment } } | null } & { ' $fragmentName'?: 'TrailerTableRowFieldsFragment' };

export type TractorTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeEquipmentDetails?: boolean | null | undefined;
  includeFleetDetails?: boolean | null | undefined;
  includeWorkerDetails?: boolean | null | undefined;
}>;


export type TractorTableQuery = { tractors: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TractorTableRowFieldsFragment': TractorTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type TrailerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeEquipmentDetails?: boolean | null | undefined;
  includeFleetDetails?: boolean | null | undefined;
}>;


export type TrailerTableQuery = { trailers: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TrailerTableRowFieldsFragment': TrailerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EquipmentTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  classes?: Array<EquipmentClass> | EquipmentClass | null | undefined;
}>;


export type EquipmentTypeTableQuery = { equipmentTypes: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EquipmentTypeConfigurationRowFieldsFragment': EquipmentTypeConfigurationRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EquipmentTypeQueryVariables = Exact<{
  id: string | number;
}>;


export type EquipmentTypeQuery = { equipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeConfigurationRowFieldsFragment': EquipmentTypeConfigurationRowFieldsFragment } } | null };

export type CreateEquipmentTypeMutationVariables = Exact<{
  input: EquipmentTypeInput;
}>;


export type CreateEquipmentTypeMutation = { createEquipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeConfigurationRowFieldsFragment': EquipmentTypeConfigurationRowFieldsFragment } } };

export type UpdateEquipmentTypeMutationVariables = Exact<{
  id: string | number;
  input: EquipmentTypeInput;
}>;


export type UpdateEquipmentTypeMutation = { updateEquipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeConfigurationRowFieldsFragment': EquipmentTypeConfigurationRowFieldsFragment } } };

export type PatchEquipmentTypeMutationVariables = Exact<{
  id: string | number;
  input: EquipmentTypePatchInput;
}>;


export type PatchEquipmentTypeMutation = { patchEquipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeConfigurationRowFieldsFragment': EquipmentTypeConfigurationRowFieldsFragment } } };

export type BulkUpdateEquipmentTypeStatusMutationVariables = Exact<{
  input: BulkUpdateEquipmentTypeStatusInput;
}>;


export type BulkUpdateEquipmentTypeStatusMutation = { bulkUpdateEquipmentTypeStatus: Array<{ ' $fragmentRefs'?: { 'EquipmentTypeConfigurationRowFieldsFragment': EquipmentTypeConfigurationRowFieldsFragment } }> };

export type FiscalPeriodFieldsFragment = { id: string, businessUnitId: string, organizationId: string, fiscalYearId: string, periodNumber: number, periodType: PeriodType, status: FiscalPeriodStatus, name: string, startDate: number, endDate: number, closedAt: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'FiscalPeriodFieldsFragment' };

export type FiscalYearTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: FiscalYearStatus, year: number, name: string, description: string, startDate: number, endDate: number, isCurrent: boolean, isCalendarYear: boolean, allowAdjustingEntries: boolean, version: number, createdAt: number, updatedAt: number, periods: Array<{ ' $fragmentRefs'?: { 'FiscalPeriodFieldsFragment': FiscalPeriodFieldsFragment } }> } & { ' $fragmentName'?: 'FiscalYearTableRowFieldsFragment' };

export type FiscalYearTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type FiscalYearTableQuery = { fiscalYears: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FiscalYearTableRowFieldsFragment': FiscalYearTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FleetCodeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, managerId: string, status: EntityStatus, code: string, description: string, revenueGoal: number | null, deadheadGoal: number | null, mileageGoal: number | null, color: string, version: number, createdAt: number, updatedAt: number, manager: { id: string, name: string } | null } & { ' $fragmentName'?: 'FleetCodeTableRowFieldsFragment' };

export type FleetCodeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type FleetCodeTableQuery = { fleetCodes: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FleetCodeTableRowFieldsFragment': FleetCodeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FormulaTemplateTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, type: FormulaTemplateType, expression: string, status: FormulaTemplateStatus, schemaId: string, version: number, currentVersionNumber: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'FormulaTemplateTableRowFieldsFragment' };

export type FormulaTemplateTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type FormulaTemplateTableQuery = { formulaTemplates: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FormulaTemplateTableRowFieldsFragment': FormulaTemplateTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FuelIndexFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, code: string, description: string, source: FuelIndexSource, fuelType: FuelType, region: string, eiaSeriesId: string, currency: string, isActive: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'FuelIndexFieldsFragment' };

export type FuelSurchargeProgramFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, code: string, description: string, status: FuelSurchargeProgramStatus, fuelIndexId: string, accessorialChargeId: string, method: FuelSurchargeProgramMethod, pegPrice: string | null, increment: string | null, incrementRate: string | null, milesPerGallon: string | null, percentBasis: FuelSurchargePercentBasis, stepRounding: FuelSurchargeStepRounding, rateRounding: FuelSurchargeRateRounding, ratePrecision: number, minAmount: string | null, maxAmount: string | null, dateBasis: FuelSurchargeDateBasis, priceEffectiveDay: number, missingPriceFallback: FuelSurchargeMissingPriceFallback, effectiveStartDate: number | null, effectiveEndDate: number | null, shipmentTypeIds: Array<string> | null, serviceTypeIds: Array<string> | null, tractorTypeIds: Array<string> | null, trailerTypeIds: Array<string> | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'FuelSurchargeProgramFieldsFragment' };

export type FuelIndexTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type FuelIndexTableQuery = { fuelIndexes: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FuelIndexFieldsFragment': FuelIndexFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FuelSurchargeProgramTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type FuelSurchargeProgramTableQuery = { fuelSurchargePrograms: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FuelSurchargeProgramFieldsFragment': FuelSurchargeProgramFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FuelSurchargeProgramDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type FuelSurchargeProgramDetailQuery = { fuelSurchargeProgram: { id: string, name: string, code: string, description: string, status: FuelSurchargeProgramStatus, fuelIndexId: string, accessorialChargeId: string, method: FuelSurchargeProgramMethod, pegPrice: string | null, increment: string | null, incrementRate: string | null, milesPerGallon: string | null, percentBasis: FuelSurchargePercentBasis, stepRounding: FuelSurchargeStepRounding, rateRounding: FuelSurchargeRateRounding, ratePrecision: number, minAmount: string | null, maxAmount: string | null, dateBasis: FuelSurchargeDateBasis, priceEffectiveDay: number, missingPriceFallback: FuelSurchargeMissingPriceFallback, effectiveStartDate: number | null, effectiveEndDate: number | null, shipmentTypeIds: Array<string> | null, serviceTypeIds: Array<string> | null, tractorTypeIds: Array<string> | null, trailerTypeIds: Array<string> | null, version: number, fuelIndex: { id: string, name: string, code: string, source: FuelIndexSource, fuelType: FuelType, region: string } | null, accessorialCharge: { id: string, code: string, description: string } | null, tableRows: Array<{ id: string, priceMin: string | null, priceMax: string | null, value: string, sortOrder: number }> | null } | null };

export type FuelDashboardQueryVariables = Exact<{ [key: string]: never; }>;


export type FuelDashboardQuery = { fuelDashboard: Array<{ delta: string | null, index: { id: string, name: string, code: string, description: string, source: FuelIndexSource, fuelType: FuelType, region: string, eiaSeriesId: string, currency: string, isActive: boolean }, latest: { id: string, priceDate: string, price: string, currency: string, isManual: boolean } | null, previous: { id: string, priceDate: string, price: string, currency: string, isManual: boolean } | null }> };

export type FuelIndexPriceHistoryQueryVariables = Exact<{
  indexId: string | number;
  from?: string | null | undefined;
  to?: string | null | undefined;
  limit?: number | null | undefined;
}>;


export type FuelIndexPriceHistoryQuery = { fuelIndexPriceHistory: Array<{ id: string, fuelIndexId: string, priceDate: string, price: string, currency: string, isManual: boolean, sourceRaw: string, fetchedAt: string }> };

export type FuelProgramCurrentRatesQueryVariables = Exact<{ [key: string]: never; }>;


export type FuelProgramCurrentRatesQuery = { fuelProgramCurrentRates: Array<{ ratePerMile: string | null, percent: string | null, flatAmount: string | null, usedFallback: boolean, program: { id: string, name: string, code: string, description: string, status: FuelSurchargeProgramStatus, method: FuelSurchargeProgramMethod, fuelIndexId: string, priceEffectiveDay: number, dateBasis: FuelSurchargeDateBasis, fuelIndex: { id: string, name: string, code: string, source: FuelIndexSource, fuelType: FuelType, region: string } | null }, price: { id: string, priceDate: string, price: string, currency: string } | null, matchedRow: { id: string, priceMin: string | null, priceMax: string | null, value: string } | null }> };

export type GenerateFuelSurchargeTableQueryVariables = Exact<{
  input: GenerateFuelTableInput;
}>;


export type GenerateFuelSurchargeTableQuery = { generateFuelSurchargeTable: Array<{ priceMin: string | null, priceMax: string | null, value: string }> };

export type EiaSeriesOptionsQueryVariables = Exact<{ [key: string]: never; }>;


export type EiaSeriesOptionsQuery = { eiaSeriesOptions: Array<{ seriesId: string, code: string, name: string, region: string, fuelType: FuelType }> };

export type CreateFuelIndexMutationVariables = Exact<{
  input: FuelIndexInput;
}>;


export type CreateFuelIndexMutation = { createFuelIndex: { id: string, name: string, code: string } };

export type UpdateFuelIndexMutationVariables = Exact<{
  id: string | number;
  input: FuelIndexInput;
}>;


export type UpdateFuelIndexMutation = { updateFuelIndex: { id: string, name: string, code: string } };

export type DeleteFuelIndexMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteFuelIndexMutation = { deleteFuelIndex: boolean };

export type AddFuelIndexPriceMutationVariables = Exact<{
  input: FuelIndexPriceInput;
}>;


export type AddFuelIndexPriceMutation = { addFuelIndexPrice: { id: string, fuelIndexId: string, priceDate: string, price: string } };

export type UpdateFuelIndexPriceMutationVariables = Exact<{
  input: UpdateFuelIndexPriceInput;
}>;


export type UpdateFuelIndexPriceMutation = { updateFuelIndexPrice: { id: string, fuelIndexId: string, priceDate: string, price: string } };

export type DeleteFuelIndexPriceMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteFuelIndexPriceMutation = { deleteFuelIndexPrice: boolean };

export type CreateFuelSurchargeProgramMutationVariables = Exact<{
  input: FuelSurchargeProgramInput;
}>;


export type CreateFuelSurchargeProgramMutation = { createFuelSurchargeProgram: { id: string, name: string, code: string } };

export type UpdateFuelSurchargeProgramMutationVariables = Exact<{
  id: string | number;
  input: FuelSurchargeProgramInput;
}>;


export type UpdateFuelSurchargeProgramMutation = { updateFuelSurchargeProgram: { id: string, name: string, code: string } };

export type DeleteFuelSurchargeProgramMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteFuelSurchargeProgramMutation = { deleteFuelSurchargeProgram: boolean };

export type HazardousMaterialTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, name: string, description: string, class: HazardousClass, unNumber: string, packingGroup: PackingGroup, subsidiaryHazardClass: string, ergGuideNumber: string, labelCodes: string, specialProvisions: string, properShippingName: string, handlingInstructions: string, emergencyContact: string, emergencyContactPhoneNumber: string, quantityThreshold: string, placardRequired: boolean, isReportableQuantity: boolean, marinePollutant: boolean, inhalationHazard: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'HazardousMaterialTableRowFieldsFragment' };

export type HazardousMaterialTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type HazardousMaterialTableQuery = { hazardousMaterials: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'HazardousMaterialTableRowFieldsFragment': HazardousMaterialTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type HazmatSegregationRuleTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, name: string, description: string, exceptionNotes: string, referenceCode: string, regulationSource: string, distanceUnit: string, classA: HazardousClass, classB: HazardousClass, segregationType: SegregationType, hasExceptions: boolean, hazmatAId: string | null, hazmatBId: string | null, minimumDistance: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'HazmatSegregationRuleTableRowFieldsFragment' };

export type HazmatSegregationRuleTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type HazmatSegregationRuleTableQuery = { hazmatSegregationRules: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'HazmatSegregationRuleTableRowFieldsFragment': HazmatSegregationRuleTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type HoldReasonTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, type: HoldType, code: string, label: string, description: string, active: boolean, defaultSeverity: HoldSeverity, defaultBlocksDispatch: boolean, defaultBlocksDelivery: boolean, defaultBlocksBilling: boolean, defaultVisibleToCustomer: boolean, sortOrder: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'HoldReasonTableRowFieldsFragment' };

export type HoldReasonTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type HoldReasonTableQuery = { holdReasons: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'HoldReasonTableRowFieldsFragment': HoldReasonTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type InvoiceTableRowFieldsFragment = { id: string, billingQueueItemId: string, shipmentId: string | null, customerId: string, number: string, billType: BillType, status: InvoiceStatus, paymentTerm: InvoicePaymentTerm, currencyCode: string, invoiceDate: number, dueDate: number | null, billToName: string, subtotalAmount: string, otherAmount: string, totalAmount: string, appliedAmount: string, settlementStatus: InvoiceSettlementStatus, disputeStatus: InvoiceDisputeStatus, sendStatus: InvoiceSendStatus, isAdjustmentArtifact: boolean, version: number, createdAt: number, updatedAt: number, customer: { id: string, name: string, code: string } | null } & { ' $fragmentName'?: 'InvoiceTableRowFieldsFragment' };

export type InvoiceTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type InvoiceTableQuery = { invoices: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'InvoiceTableRowFieldsFragment': InvoiceTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type JournalEntryDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type JournalEntryDetailQuery = { journalEntry: { id: string, organizationId: string, businessUnitId: string, batchId: string, fiscalYearId: string, fiscalPeriodId: string, entryNumber: string, entryType: string, status: string, accountingDate: number, description: string, referenceType: string, referenceId: string, totalDebit: number, totalCredit: number, isPosted: boolean, isReversal: boolean, reversalOfId: string | null, reversedById: string | null, reversalDate: number | null, reversalReason: string, lines: Array<{ id: string, journalEntryId: string, glAccountId: string, lineNumber: number, description: string, debitAmount: number, creditAmount: number, netAmount: number, customerId: string | null, locationId: string | null, glAccount: { id: string, accountCode: string, name: string } | null }> | null } | null };

export type JournalSourceByObjectQueryVariables = Exact<{
  sourceType: string;
  sourceId: string;
}>;


export type JournalSourceByObjectQuery = { journalSourceByObject: { id: string, sourceObjectType: string, sourceObjectId: string, sourceEventType: string, sourceDocumentNumber: string, status: string } | null };

export type JournalEntriesBySourceQueryVariables = Exact<{
  sourceType: string;
  sourceId: string;
}>;


export type JournalEntriesBySourceQuery = { journalEntriesBySource: Array<{ id: string, batchId: string, entryNumber: string, entryType: string, status: string, accountingDate: number, description: string, referenceType: string, referenceId: string, totalDebit: number, totalCredit: number, isPosted: boolean, isReversal: boolean, lines: Array<{ id: string, journalEntryId: string, glAccountId: string, lineNumber: number, description: string, debitAmount: number, creditAmount: number, netAmount: number, customerId: string | null, locationId: string | null, glAccount: { id: string, accountCode: string, name: string } | null }> | null }> };

export type JournalReversalTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, originalJournalEntryId: string, reversalJournalEntryId: string | null, postedBatchId: string | null, status: JournalReversalStatus, requestedAccountingDate: number, resolvedFiscalYearId: string, resolvedFiscalPeriodId: string, reasonCode: string, reasonText: string, requestedById: string, approvedById: string | null, approvedAt: number | null, rejectedById: string | null, rejectedAt: number | null, rejectionReason: string | null, cancelledById: string | null, cancelledAt: number | null, cancelReason: string | null, postedById: string | null, postedAt: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'JournalReversalTableRowFieldsFragment' };

export type JournalReversalTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type JournalReversalTableQuery = { journalReversals: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'JournalReversalTableRowFieldsFragment': JournalReversalTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type LocationCategoryTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, type: LocationCategoryType, facilityType: FacilityType | null, color: string, hasSecureParking: boolean, requiresAppointment: boolean, allowsOvernight: boolean, hasRestroom: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'LocationCategoryTableRowFieldsFragment' };

export type LocationCategoryTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type LocationCategoryTableQuery = { locationCategories: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'LocationCategoryTableRowFieldsFragment': LocationCategoryTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type LocationTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, locationCategoryId: string, stateId: string, status: EntityStatus, code: string, name: string, description: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, version: number, createdAt: number, updatedAt: number, state: { id: string, name: string, abbreviation: string } | null, locationCategory: { id: string, name: string, color: string } | null } & { ' $fragmentName'?: 'LocationTableRowFieldsFragment' };

export type LocationTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type LocationTableQuery = { locations: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'LocationTableRowFieldsFragment': LocationTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ManualJournalTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, requestNumber: string, status: ManualJournalStatus, description: string, reason: string | null, accountingDate: number, requestedFiscalYearId: string, requestedFiscalPeriodId: string, currencyCode: string, totalDebit: number, totalCredit: number, approvedAt: number | null, approvedById: string | null, rejectedAt: number | null, rejectedById: string | null, rejectionReason: string | null, cancelledAt: number | null, cancelledById: string | null, cancelReason: string | null, postedBatchId: string | null, createdById: string | null, updatedById: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ManualJournalTableRowFieldsFragment' };

export type ManualJournalTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type ManualJournalTableQuery = { manualJournals: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ManualJournalTableRowFieldsFragment': ManualJournalTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type NotificationFieldsFragment = { id: string, organizationId: string, businessUnitId: string | null, targetUserId: string | null, eventType: string, priority: NotificationPriority, channel: NotificationChannel, title: string, message: string, data: unknown, relatedEntities: unknown, source: string, readAt: number | null, dismissedAt: number | null, createdAt: number } & { ' $fragmentName'?: 'NotificationFieldsFragment' };

export type NotificationListQueryVariables = Exact<{
  input: DataTableConnectionInput;
  filter?: NotificationFilterInput | null | undefined;
}>;


export type NotificationListQuery = { notifications: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'NotificationFieldsFragment': NotificationFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type NotificationUnreadCountQueryVariables = Exact<{ [key: string]: never; }>;


export type NotificationUnreadCountQuery = { notificationUnreadCount: number };

export type MarkNotificationsReadMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type MarkNotificationsReadMutation = { markNotificationsRead: boolean };

export type MarkNotificationsUnreadMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type MarkNotificationsUnreadMutation = { markNotificationsUnread: boolean };

export type MarkAllNotificationsReadMutationVariables = Exact<{ [key: string]: never; }>;


export type MarkAllNotificationsReadMutation = { markAllNotificationsRead: boolean };

export type DismissNotificationsMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type DismissNotificationsMutation = { dismissNotifications: boolean };

export type RestoreNotificationsMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type RestoreNotificationsMutation = { restoreNotifications: boolean };

export type OrderDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type OrderDetailQuery = { order: { id: string, orderNumber: string, status: OrderStatus, customerId: string, ownerId: string | null, poNumber: string | null, bol: string | null, currencyCode: string, quotedAmount: string | null, baseAmount: string | null, totalAmount: string | null, version: number, createdAt: number, updatedAt: number, customer: { id: string, name: string, code: string } | null, legs: Array<{ id: string, proNumber: string, status: ShipmentStatus, bol: string | null, freightChargeAmount: string, totalChargeAmount: string }>, charges: Array<{ id: string, description: string, amount: string, invoiceId: string | null, version: number, createdAt: number }> } | null };

export type OrderMutationResultFragment = { id: string, orderNumber: string, status: OrderStatus, totalAmount: string | null, version: number } & { ' $fragmentName'?: 'OrderMutationResultFragment' };

export type AttachOrderShipmentsMutationVariables = Exact<{
  orderId: string | number;
  shipmentIds: Array<string | number> | string | number;
}>;


export type AttachOrderShipmentsMutation = { attachOrderShipments: { ' $fragmentRefs'?: { 'OrderMutationResultFragment': OrderMutationResultFragment } } };

export type DetachOrderShipmentMutationVariables = Exact<{
  orderId: string | number;
  shipmentId: string | number;
}>;


export type DetachOrderShipmentMutation = { detachOrderShipment: { ' $fragmentRefs'?: { 'OrderMutationResultFragment': OrderMutationResultFragment } } };

export type CreateInvoiceFromOrderMutationVariables = Exact<{
  orderId: string | number;
}>;


export type CreateInvoiceFromOrderMutation = { createInvoiceFromOrder: { id: string, number: string } };

export type CreateOrderMutationVariables = Exact<{
  input: OrderInput;
}>;


export type CreateOrderMutation = { createOrder: { id: string, orderNumber: string, status: OrderStatus, version: number } };

export type UpdateOrderMutationVariables = Exact<{
  id: string | number;
  input: OrderInput;
}>;


export type UpdateOrderMutation = { updateOrder: { id: string, orderNumber: string, status: OrderStatus, version: number } };

export type AddOrderChargeMutationVariables = Exact<{
  orderId: string | number;
  description: string;
  amount: string;
}>;


export type AddOrderChargeMutation = { addOrderCharge: { ' $fragmentRefs'?: { 'OrderMutationResultFragment': OrderMutationResultFragment } } };

export type UpdateOrderChargeMutationVariables = Exact<{
  input: UpdateOrderChargeInput;
}>;


export type UpdateOrderChargeMutation = { updateOrderCharge: { ' $fragmentRefs'?: { 'OrderMutationResultFragment': OrderMutationResultFragment } } };

export type RemoveOrderChargeMutationVariables = Exact<{
  input: RemoveOrderChargeInput;
}>;


export type RemoveOrderChargeMutation = { removeOrderCharge: { ' $fragmentRefs'?: { 'OrderMutationResultFragment': OrderMutationResultFragment } } };

export type CloseOrderMutationVariables = Exact<{
  id: string | number;
}>;


export type CloseOrderMutation = { closeOrder: { ' $fragmentRefs'?: { 'OrderMutationResultFragment': OrderMutationResultFragment } } };

export type CancelOrderMutationVariables = Exact<{
  id: string | number;
  cancelReason: string;
}>;


export type CancelOrderMutation = { cancelOrder: { ' $fragmentRefs'?: { 'OrderMutationResultFragment': OrderMutationResultFragment } } };

export type OrderTableRowFieldsFragment = { id: string, ownerId: string | null, businessUnitId: string, organizationId: string, customerId: string, status: OrderStatus, orderNumber: string, poNumber: string | null, bol: string | null, currencyCode: string, quotedAmount: string | null, baseAmount: string | null, totalAmount: string | null, version: number, createdAt: number, updatedAt: number, customer: { id: string, name: string, code: string } | null } & { ' $fragmentName'?: 'OrderTableRowFieldsFragment' };

export type OrderTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type OrderTableQuery = { orders: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'OrderTableRowFieldsFragment': OrderTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type OrganizationSettingsStateFieldsFragment = { id: string, name: string, abbreviation: string } & { ' $fragmentName'?: 'OrganizationSettingsStateFieldsFragment' };

export type OrganizationSettingsFieldsFragment = { id: string, version: number, createdAt: number, updatedAt: number, bucketName: string, businessUnitId: string, loginSlug: string, name: string, scacCode: string, dotNumber: string, logoUrl: string, addressLine1: string, addressLine2: string, city: string, stateId: string, postalCode: string, timezone: string, taxId: string, state: { ' $fragmentRefs'?: { 'OrganizationSettingsStateFieldsFragment': OrganizationSettingsStateFieldsFragment } } | null } & { ' $fragmentName'?: 'OrganizationSettingsFieldsFragment' };

export type OrganizationSettingsQueryVariables = Exact<{
  id: string | number;
  includeState?: boolean | null | undefined;
  includeBu?: boolean | null | undefined;
}>;


export type OrganizationSettingsQuery = { organization: { ' $fragmentRefs'?: { 'OrganizationSettingsFieldsFragment': OrganizationSettingsFieldsFragment } } };

export type UpdateOrganizationSettingsMutationVariables = Exact<{
  id: string | number;
  input: OrganizationInput;
}>;


export type UpdateOrganizationSettingsMutation = { updateOrganization: { ' $fragmentRefs'?: { 'OrganizationSettingsFieldsFragment': OrganizationSettingsFieldsFragment } } };

export type RateTableTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, key: string, description: string, lookupType: RateTableLookupType, active: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'RateTableTableRowFieldsFragment' };

export type RateTableTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type RateTableTableQuery = { rateTables: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RateTableTableRowFieldsFragment': RateTableTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RecurringShipmentTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, sourceShipmentId: string, customerId: string | null, originLocationId: string | null, destinationLocationId: string | null, name: string, description: string, status: RecurringShipmentStatus, cronExpression: string, timezone: string, startDate: number | null, endDate: number | null, maxOccurrences: number | null, leadTimeDays: number, skipWeekends: boolean, exceptionPolicy: RecurringShipmentExceptionPolicy, blackoutDates: Array<string> | null, autoGenerate: boolean, nextOccurrenceAt: number | null, lastOccurrenceAt: number | null, lastRunAt: number | null, generationCount: number, consecutiveFailures: number, version: number, createdAt: number, updatedAt: number, customer: { id: string, name: string, code: string } | null, originLocation: { id: string, name: string, code: string } | null, destinationLocation: { id: string, name: string, code: string } | null } & { ' $fragmentName'?: 'RecurringShipmentTableRowFieldsFragment' };

export type RecurringShipmentTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type RecurringShipmentTableQuery = { recurringShipments: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RecurringShipmentTableRowFieldsFragment': RecurringShipmentTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CannedReportsQueryVariables = Exact<{ [key: string]: never; }>;


export type CannedReportsQuery = { cannedReports: Array<{ key: string, version: string, name: string, description: string, category: string, tags: Array<string>, defaultFormat: string, definition: unknown }> };

export type ReportCatalogQueryVariables = Exact<{ [key: string]: never; }>;


export type ReportCatalogQuery = { reportCatalog: { version: string, entities: Array<{ key: string, resource: string, label: string, pluralLabel: string, description: string | null, category: string, ownScopeSupported: boolean, fields: Array<{ key: string, label: string, description: string | null, type: string, format: string | null, nullable: boolean, aggregations: Array<string>, filterable: boolean, groupable: boolean, accessible: boolean, sensitivity: string, enumValues: Array<{ value: string, label: string }> }>, edges: Array<{ name: string, label: string, target: string, cardinality: string, traversable: boolean }> }> } };

export type ReportDefinitionFieldsFragment = { id: string, name: string, description: string, category: string, tags: Array<string>, kind: string, cannedKey: string | null, cannedVersion: string | null, ownerId: string, visibility: string, status: string, diagnostics: Array<string>, catalogVersion: string, definition: unknown, defaultFormat: string, currentRevision: number, lastRunAt: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ReportDefinitionFieldsFragment' };

export type ReportDefinitionsTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type ReportDefinitionsTableQuery = { reportDefinitions: { totalCount: number, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'ReportDefinitionFieldsFragment': ReportDefinitionFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ReportDefinitionByIdQueryVariables = Exact<{
  id: string | number;
}>;


export type ReportDefinitionByIdQuery = { reportDefinition: { ' $fragmentRefs'?: { 'ReportDefinitionFieldsFragment': ReportDefinitionFieldsFragment } } };

export type ReportDefinitionRevisionsQueryVariables = Exact<{
  definitionId: string | number;
  limit?: number | null | undefined;
}>;


export type ReportDefinitionRevisionsQuery = { reportDefinitionRevisions: Array<{ id: string, definitionId: string, revisionNumber: number, catalogVersion: string, definition: unknown, createdById: string, createdAt: number }> };

export type CreateReportDefinitionMutationVariables = Exact<{
  input: SaveReportDefinitionInput;
}>;


export type CreateReportDefinitionMutation = { createReportDefinition: { ' $fragmentRefs'?: { 'ReportDefinitionFieldsFragment': ReportDefinitionFieldsFragment } } };

export type UpdateReportDefinitionMutationVariables = Exact<{
  input: UpdateReportDefinitionInput;
}>;


export type UpdateReportDefinitionMutation = { updateReportDefinition: { ' $fragmentRefs'?: { 'ReportDefinitionFieldsFragment': ReportDefinitionFieldsFragment } } };

export type DeleteReportDefinitionMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteReportDefinitionMutation = { deleteReportDefinition: boolean };

export type ForkCannedReportMutationVariables = Exact<{
  input: ForkCannedReportInput;
}>;


export type ForkCannedReportMutation = { forkCannedReport: { ' $fragmentRefs'?: { 'ReportDefinitionFieldsFragment': ReportDefinitionFieldsFragment } } };

export type ResetCannedForkMutationVariables = Exact<{
  id: string | number;
}>;


export type ResetCannedForkMutation = { resetCannedFork: { ' $fragmentRefs'?: { 'ReportDefinitionFieldsFragment': ReportDefinitionFieldsFragment } } };

export type PreviewReportQueryVariables = Exact<{
  definition: ReportIrInput;
  params?: unknown;
}>;


export type PreviewReportQuery = { previewReport: { rows: unknown, truncated: boolean, columns: Array<{ id: string, label: string, type: string, format: string | null }> } };

export type ReportRunFieldsFragment = { id: string, definitionId: string | null, revisionId: string | null, cannedKey: string | null, cannedVersion: string | null, requestedById: string, trigger: string, params: unknown, format: string, status: string, rowCount: number, byteSize: number, durationMs: number, truncated: boolean, artifactExpiresAt: number | null, cacheHit: boolean, queuedAt: number | null, startedAt: number | null, completedAt: number | null, version: number, createdAt: number, error: { code: string, message: string, detail: string | null } | null } & { ' $fragmentName'?: 'ReportRunFieldsFragment' };

export type ReportRunsTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  filter?: ReportRunsFilterInput | null | undefined;
}>;


export type ReportRunsTableQuery = { reportRuns: { totalCount: number, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'ReportRunFieldsFragment': ReportRunFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ReportRunByIdQueryVariables = Exact<{
  id: string | number;
}>;


export type ReportRunByIdQuery = { reportRun: { ' $fragmentRefs'?: { 'ReportRunFieldsFragment': ReportRunFieldsFragment } } };

export type RunReportMutationVariables = Exact<{
  input: RunReportInput;
}>;


export type RunReportMutation = { runReport: { ' $fragmentRefs'?: { 'ReportRunFieldsFragment': ReportRunFieldsFragment } } };

export type CancelReportRunMutationVariables = Exact<{
  id: string | number;
}>;


export type CancelReportRunMutation = { cancelReportRun: { ' $fragmentRefs'?: { 'ReportRunFieldsFragment': ReportRunFieldsFragment } } };

export type ReportScheduleFieldsFragment = { id: string, definitionId: string, cronExpression: string, timezone: string, formats: Array<string>, emailRecipients: Array<string>, emailAttach: boolean, notifyUserIds: Array<string>, enabled: boolean, runAsId: string, lastRunId: string | null, nextRunAt: number | null, consecutiveFailures: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ReportScheduleFieldsFragment' };

export type ReportSchedulesQueryVariables = Exact<{
  definitionId?: string | number | null | undefined;
}>;


export type ReportSchedulesQuery = { reportSchedules: Array<{ ' $fragmentRefs'?: { 'ReportScheduleFieldsFragment': ReportScheduleFieldsFragment } }> };

export type CreateReportScheduleMutationVariables = Exact<{
  input: CreateReportScheduleInput;
}>;


export type CreateReportScheduleMutation = { createReportSchedule: { ' $fragmentRefs'?: { 'ReportScheduleFieldsFragment': ReportScheduleFieldsFragment } } };

export type UpdateReportScheduleMutationVariables = Exact<{
  input: UpdateReportScheduleInput;
}>;


export type UpdateReportScheduleMutation = { updateReportSchedule: { ' $fragmentRefs'?: { 'ReportScheduleFieldsFragment': ReportScheduleFieldsFragment } } };

export type DeleteReportScheduleMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteReportScheduleMutation = { deleteReportSchedule: boolean };

export type RoleTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, coreResponsibility: string | null, parentRoleIds: Array<string> | null, maxSensitivity: string, isSystem: boolean, createdBy: string, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'RoleTableRowFieldsFragment' };

export type RoleTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type RoleTableQuery = { roles: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RoleTableRowFieldsFragment': RoleTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ScimGroupRoleMappingTableRowFieldsFragment = { id: string, directoryId: string, externalGroupId: string, displayName: string, roleId: string, version: number, role: { id: string, name: string } | null } & { ' $fragmentName'?: 'ScimGroupRoleMappingTableRowFieldsFragment' };

export type ScimGroupRoleMappingsTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  directoryId: string | number;
}>;


export type ScimGroupRoleMappingsTableQuery = { scimGroupRoleMappings: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ScimGroupRoleMappingTableRowFieldsFragment': ScimGroupRoleMappingTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type SelectOptionsQueryVariables = Exact<{
  input: SelectOptionsInput;
}>;


export type SelectOptionsQuery = { selectOptions: { totalCount: number | null, edges: Array<{ cursor: string, node: { id: string, label: string, description: string | null, meta: unknown } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type ServiceFailureReasonCodeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, label: string, description: string, category: ServiceFailureReasonCategory, appliesTo: ServiceFailureReasonCodeAppliesTo, defaultStatusCode: string, defaultReasonCode: string, defaultExceptionCode: string, defaultNote: string, active: boolean, sortOrder: number, externalMap: unknown, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ServiceFailureReasonCodeTableRowFieldsFragment' };

export type ServiceFailureReasonCodeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type ServiceFailureReasonCodeTableQuery = { serviceFailureReasonCodes: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ServiceFailureReasonCodeTableRowFieldsFragment': ServiceFailureReasonCodeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ServiceFailureTableRowFieldsFragment = { id: string, shipmentId: string, number: string, type: ServiceFailureType, source: ServiceFailureSource, status: ServiceFailureStatus, stopType: StopType, stopId: string, scheduledCutoff: number, actualArrival: number, gracePeriodMinutes: number, lateMinutes: number, reasonCodeId: string | null, notes: string, detectedAt: number, version: number, shipment: { id: string, proNumber: string, bol: string | null } | null, stop: { id: string, type: StopType, sequence: number, locationId: string, location: { id: string, name: string, code: string, city: string, state: { abbreviation: string } | null } | null } | null, reasonCode: { id: string, code: string, label: string } | null } & { ' $fragmentName'?: 'ServiceFailureTableRowFieldsFragment' };

export type ServiceFailureTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  shipmentId?: string | number | null | undefined;
}>;


export type ServiceFailureTableQuery = { serviceFailures: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ServiceFailureTableRowFieldsFragment': ServiceFailureTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ServiceTypeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, description: string | null, color: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ServiceTypeTableRowFieldsFragment' };

export type ServiceTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type ServiceTypeTableQuery = { serviceTypes: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ServiceTypeTableRowFieldsFragment': ServiceTypeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ShipmentTypeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, description: string | null, color: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ShipmentTypeTableRowFieldsFragment' };

export type ShipmentTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type ShipmentTypeTableQuery = { shipmentTypes: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentTypeTableRowFieldsFragment': ShipmentTypeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ShipmentUserFieldsFragment = { id: string, name: string, username: string, emailAddress: string, timezone: string, status: EntityStatus, profilePicUrl: string, thumbnailUrl: string } & { ' $fragmentName'?: 'ShipmentUserFieldsFragment' };

export type ShipmentLocationFieldsFragment = { id: string, name: string, code: string, status: EntityStatus, locationCategoryId: string, stateId: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, longitude: number | null, latitude: number | null } & { ' $fragmentName'?: 'ShipmentLocationFieldsFragment' };

export type ShipmentWorkerFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string, profilePicUrl: string } & { ' $fragmentName'?: 'ShipmentWorkerFieldsFragment' };

export type ShipmentTractorFieldsFragment = { id: string, code: string } & { ' $fragmentName'?: 'ShipmentTractorFieldsFragment' };

export type ShipmentTrailerFieldsFragment = { id: string, code: string, equipmentTypeId: string } & { ' $fragmentName'?: 'ShipmentTrailerFieldsFragment' };

export type ShipmentAssignmentFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentMoveId: string | null, primaryWorkerId: string | null, tractorId: string | null, trailerId: string | null, secondaryWorkerId: string | null, status: AssignmentStatus, archivedAt: number | null, version: number, createdAt: number, updatedAt: number, tractor: { ' $fragmentRefs'?: { 'ShipmentTractorFieldsFragment': ShipmentTractorFieldsFragment } } | null, trailer: { ' $fragmentRefs'?: { 'ShipmentTrailerFieldsFragment': ShipmentTrailerFieldsFragment } } | null, primaryWorker: { ' $fragmentRefs'?: { 'ShipmentWorkerFieldsFragment': ShipmentWorkerFieldsFragment } } | null, secondaryWorker: { ' $fragmentRefs'?: { 'ShipmentWorkerFieldsFragment': ShipmentWorkerFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentAssignmentFieldsFragment' };

export type ShipmentStopFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentMoveId: string | null, locationId: string, status: StopStatus, type: StopType, scheduleType: StopScheduleType, sequence: number, pieces: number | null, weight: number | null, scheduledWindowStart: number, scheduledWindowEnd: number | null, actualArrival: number | null, actualDeparture: number | null, countLateOverride: boolean | null, countDetentionOverride: boolean | null, addressLine: string, version: number, createdAt: number, updatedAt: number, location: { ' $fragmentRefs'?: { 'ShipmentLocationFieldsFragment': ShipmentLocationFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentStopFieldsFragment' };

export type ShipmentMoveFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string | null, status: MoveStatus, loaded: boolean, sequence: number, distance: number | null, distanceSource: string | null, distanceProvider: string | null, distanceCalculatedAt: number | null, distanceRouteSignature: string | null, distanceDataVersion: string | null, distanceRoutingType: string | null, distanceUnits: string | null, distanceMetadata: unknown, version: number, createdAt: number, updatedAt: number, stops: Array<{ ' $fragmentRefs'?: { 'ShipmentStopFieldsFragment': ShipmentStopFieldsFragment } }>, assignment: { ' $fragmentRefs'?: { 'ShipmentAssignmentFieldsFragment': ShipmentAssignmentFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentMoveFieldsFragment' };

export type ShipmentAdditionalChargeFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string, accessorialChargeId: string, isSystemGenerated: boolean, method: string, amount: string, unit: number, fuelSurchargeProgramId: string | null, fuelSurchargeDetail: unknown, version: number, createdAt: number, updatedAt: number, accessorialCharge: { id: string, businessUnitId: string, organizationId: string, code: string, description: string, status: EntityStatus, method: string, rateUnit: string, amount: string, version: number, createdAt: number, updatedAt: number } | null } & { ' $fragmentName'?: 'ShipmentAdditionalChargeFieldsFragment' };

export type ShipmentCommodityFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string, commodityId: string, pieces: number, weight: number, version: number, createdAt: number, updatedAt: number, commodity: { id: string, businessUnitId: string, organizationId: string, hazardousMaterialId: string | null, status: EntityStatus, name: string, description: string, minTemperature: number | null, maxTemperature: number | null, weightPerUnit: number | null, linearFeetPerUnit: number | null, maxQuantityPerShipment: number | null, freightClass: string, loadingInstructions: string, stackable: boolean, fragile: boolean, version: number, createdAt: number, updatedAt: number } | null } & { ' $fragmentName'?: 'ShipmentCommodityFieldsFragment' };

export type ShipmentRatingDetailFieldsFragment = { formulaTemplateId: string, formulaTemplateName: string, expression: string, resolvedVariables: unknown, result: number, ratedAt: number } & { ' $fragmentName'?: 'ShipmentRatingDetailFieldsFragment' };

export type ShipmentFieldsFragment = { id: string, businessUnitId: string, organizationId: string, sourceDocumentId: string | null, serviceTypeId: string, shipmentTypeId: string, customerId: string, tractorTypeId: string | null, trailerTypeId: string | null, ownerId: string | null, enteredById: string | null, canceledById: string | null, formulaTemplateId: string, consolidationGroupId: string | null, orderId: string | null, orderNumber: string | null, orderStatus: OrderStatus | null, status: ShipmentStatus, tenderStatus: ShipmentTenderStatus | null, entryMethod: ShipmentEntryMethod | null, proNumber: string, bol: string | null, cancelReason: string, otherChargeAmount: string, freightChargeAmount: string, baseRate: string, totalChargeAmount: string, pieces: number | null, weight: number | null, temperatureMin: number | null, temperatureMax: number | null, actualDeliveryDate: number | null, actualShipDate: number | null, canceledAt: number | null, billingTransferStatus: string | null, transferredToBillingAt: number | null, markedReadyToBillAt: number | null, billedAt: number | null, ratingUnit: number, fuelSurchargeLocked: boolean, version: number, createdAt: number, updatedAt: number, profitabilityEstimate: { shipmentId: string, loadedMiles: number, deadheadMiles: number, totalMiles: number, costPerMile: string, estimatedCost: string, profit: string, marginPercent: string | null, breakEvenRpm: string | null, targetMarginPercent: string | null, missingDistance: boolean } | null, ratingDetail: { ' $fragmentRefs'?: { 'ShipmentRatingDetailFieldsFragment': ShipmentRatingDetailFieldsFragment } } | null, moves: Array<{ ' $fragmentRefs'?: { 'ShipmentMoveFieldsFragment': ShipmentMoveFieldsFragment } }>, additionalCharges: Array<{ ' $fragmentRefs'?: { 'ShipmentAdditionalChargeFieldsFragment': ShipmentAdditionalChargeFieldsFragment } }>, commodities: Array<{ ' $fragmentRefs'?: { 'ShipmentCommodityFieldsFragment': ShipmentCommodityFieldsFragment } }>, customer: { id: string, businessUnitId: string, organizationId: string, stateId: string, status: EntityStatus, code: string, name: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, isGeocoded: boolean, longitude: number | null, latitude: number | null, placeId: string, externalId: string, allowConsolidation: boolean, exclusiveConsolidation: boolean, consolidationPriority: number, version: number, createdAt: number, updatedAt: number } | null, owner: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, formulaTemplate: { id: string, organizationId: string, businessUnitId: string, name: string, description: string, type: string, expression: string, status: string, schemaId: string, metadata: unknown, version: number, sourceTemplateId: string | null, sourceVersionNumber: number | null, currentVersionNumber: number, createdAt: number, updatedAt: number, variableDefinitions: Array<{ name: string, type: string, description: string, required: boolean, defaultValue: unknown, source: string | null }> } | null } & { ' $fragmentName'?: 'ShipmentFieldsFragment' };

export type ShipmentPageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'ShipmentPageInfoFieldsFragment' };

export type ShipmentCommentMentionFieldsFragment = { id: string, commentId: string, mentionedUserId: string, organizationId: string | null, businessUnitId: string | null, shipmentId: string | null, createdAt: number, mentionedUser: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentCommentMentionFieldsFragment' };

export type ShipmentCommentFieldsFragment = { id: string, businessUnitId: string | null, organizationId: string | null, shipmentId: string, userId: string | null, comment: string, type: ShipmentCommentType, visibility: ShipmentCommentVisibility, priority: ShipmentCommentPriority, source: ShipmentCommentSource, metadata: unknown, editedAt: number | null, version: number, createdAt: number, updatedAt: number, mentionedUserIds: Array<string>, user: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, mentionedUsers: Array<{ ' $fragmentRefs'?: { 'ShipmentCommentMentionFieldsFragment': ShipmentCommentMentionFieldsFragment } }> | null } & { ' $fragmentName'?: 'ShipmentCommentFieldsFragment' };

export type ShipmentEventFieldsFragment = { id: string, organizationId: string, businessUnitId: string, shipmentId: string, moveId: string | null, stopId: string | null, assignmentId: string | null, commentId: string | null, holdId: string | null, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, proNumber: string | null, previousStatus: string | null, newStatus: string | null, reason: string | null, previousOwnerId: string | null, newOwnerId: string | null, primaryWorkerId: string | null, secondaryWorkerId: string | null, tractorId: string | null, trailerId: string | null, driverName: string | null, holdType: string | null, holdSeverity: string | null, holdSource: string | null, commentBody: string | null, commentType: string | null, commentVisibility: string | null, commentPriority: string | null, mentionedUserIds: Array<string>, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFieldsFragment' };

export type ShipmentCommandCenterTableQueryVariables = Exact<{
  input: ShipmentsInput;
}>;


export type ShipmentCommandCenterTableQuery = { shipments: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

export type ShipmentDetailQueryVariables = Exact<{
  id: string | number;
  expandShipmentDetails?: boolean | null | undefined;
}>;


export type ShipmentDetailQuery = { shipment: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } | null };

export type ShipmentSavedViewCountsQueryVariables = Exact<{
  timezone: string;
}>;


export type ShipmentSavedViewCountsQuery = { shipmentAnalytics: { page: string, savedViewCounts: { all: number | null, transit: number | null, atRisk: number | null, unassigned: number | null, deliveringToday: number | null } | null } };

export type ShipmentPageAnalyticsQueryVariables = Exact<{
  input: ShipmentAnalyticsInput;
}>;


export type ShipmentPageAnalyticsQuery = { shipmentAnalytics: { page: string, savedViewCounts: { all: number | null, transit: number | null, atRisk: number | null, unassigned: number | null, deliveringToday: number | null } | null, activeShipments: { count: number, changeFromYesterday: number, sparkline: Array<{ hour: string, value: number }>, breakdown: { inTransit: number, atRisk: number, loading: number, done: number } } | null, onTimePercent: { percent: number, onTimeCount: number, totalCount: number, target: number | null, deltaPp: number, sevenDayPercent: number } | null, profitability: { avgCpm: number, avgMarginPct: number, hasMargin: boolean, unprofitableCount: number, shipmentCount: number, totalMiles: number } | null, revenueToday: { total: number, deltaPct: number, rpm: number, sparkline: Array<{ hour: string, value: number }> } | null, emptyMilePercent: { percent: number, emptyMiles: number, totalMiles: number, deltaPp: number } | null, atRisk: { count: number, delta: number, etaSlip: number, weather: number, reefer: number } | null, unassigned: { count: number, delta: number, revenueWaiting: number } | null, readyToDispatch: { count: number, delta: number, unassigned: number, driverReady: number } | null, detentionWatchlist: { items: Array<{ shipmentId: string, customer: string, dwellLabel: string, tone: string }> } | null, customerMix: { windowDays: number, entries: Array<{ customerId: string, name: string, revenue: number, share: number, loads: number, trend: number }> } | null, tomorrowsPickups: { date: string, pickups: Array<{ shipmentId: string, proNumber: string, pickupWindowStart: number, customer: string, origin: string, destination: string, driver: string, status: string }> } | null, laneHeatmap: { windowDays: number, total: number, cells: Array<{ origin: string, destination: string, count: number }> } | null } };

export type ShipmentTomorrowsPickupsQueryVariables = Exact<{
  limit?: number | null | undefined;
  offset?: number | null | undefined;
  timezone?: string | null | undefined;
}>;


export type ShipmentTomorrowsPickupsQuery = { shipmentAnalytics: { page: string, tomorrowsPickups: { date: string, pickups: Array<{ shipmentId: string, proNumber: string, pickupWindowStart: number, customer: string, origin: string, destination: string, driver: string, status: string }> } | null } };

export type UnassignedShipmentsQueryVariables = Exact<{
  first: number;
  after?: string | null | undefined;
}>;


export type UnassignedShipmentsQuery = { unassignedShipments: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

export type ExceptionShipmentsQueryVariables = Exact<{
  input: ShipmentsInput;
}>;


export type ExceptionShipmentsQuery = { shipments: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

export type MapShipmentsQueryVariables = Exact<{
  input: ShipmentsInput;
}>;


export type MapShipmentsQuery = { shipments: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

export type ShipmentCommentsQueryVariables = Exact<{
  shipmentId: string | number;
  first: number;
  after?: string | null | undefined;
}>;


export type ShipmentCommentsQuery = { shipmentComments: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

export type ShipmentCommentCountQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type ShipmentCommentCountQuery = { shipmentCommentCount: { count: number } };

export type ShipmentEventsQueryVariables = Exact<{
  input: ShipmentEventsInput;
}>;


export type ShipmentEventsQuery = { shipmentEvents: Array<{ ' $fragmentRefs'?: { 'ShipmentEventFieldsFragment': ShipmentEventFieldsFragment } }> };

export type ShipmentBillingReadinessQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type ShipmentBillingReadinessQuery = { shipmentBillingReadiness: { shipmentId: string, shipmentStatus: ShipmentStatus, canMarkReadyToInvoice: boolean, shouldAutoMarkReadyToInvoice: boolean, shouldAutoTransferToBilling: boolean, policy: { shipmentBillingRequirementEnforcement: string, rateValidationEnforcement: string, billingExceptionDisposition: string, notifyOnBillingExceptions: boolean, readyToBillAssignmentMode: string, billingQueueTransferMode: string }, requirements: Array<{ documentTypeId: string, documentTypeCode: string, documentTypeName: string, satisfied: boolean, documentCount: number, documentIds: Array<string> }>, missingRequirements: Array<{ documentTypeId: string, documentTypeCode: string, documentTypeName: string, satisfied: boolean, documentCount: number, documentIds: Array<string> }>, validationFailures: Array<{ field: string, code: string, message: string }>, warnings: Array<{ code: string, message: string, context: { documentTypeId: string | null, documentTypeCode: string | null, documentTypeName: string | null, documentCount: number | null, requirementCount: number | null, missingRequirementCount: number | null, serviceFailureIds: Array<string> | null, unresolvedCount: number | null } | null }>, serviceFailureContext: { hasUnresolved: boolean, unresolvedCount: number, serviceFailureIds: Array<string> } } };

export type ShipmentUiPolicyQueryVariables = Exact<{ [key: string]: never; }>;


export type ShipmentUiPolicyQuery = { shipmentUIPolicy: { allowMoveRemovals: boolean, checkForDuplicateBols: boolean, checkHazmatSegregation: boolean, maxShipmentWeightLimit: number } };

export type ShipmentPreviousRatesQueryVariables = Exact<{
  input: ShipmentPreviousRatesInput;
}>;


export type ShipmentPreviousRatesQuery = { shipmentPreviousRates: { total: number, items: Array<{ shipmentId: string, proNumber: string, customerId: string, serviceTypeId: string, shipmentTypeId: string, formulaTemplateId: string, freightChargeAmount: string, otherChargeAmount: string, totalChargeAmount: string, ratingUnit: number, pieces: number | null, weight: number | null, createdAt: number }> } };

export type CreateShipmentMutationVariables = Exact<{
  input: ShipmentInput;
}>;


export type CreateShipmentMutation = { createShipment: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } };

export type UpdateShipmentMutationVariables = Exact<{
  id: string | number;
  input: ShipmentInput;
}>;


export type UpdateShipmentMutation = { updateShipment: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } };

export type CancelShipmentMutationVariables = Exact<{
  id: string | number;
  input?: ShipmentCancelInput | null | undefined;
}>;


export type CancelShipmentMutation = { cancelShipment: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } };

export type UncancelShipmentMutationVariables = Exact<{
  id: string | number;
}>;


export type UncancelShipmentMutation = { uncancelShipment: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } };

export type DuplicateShipmentMutationVariables = Exact<{
  input: ShipmentDuplicateInput;
}>;


export type DuplicateShipmentMutation = { duplicateShipment: { workflowId: string, runId: string, taskQueue: string, status: string, submittedAt: number } };

export type TransferShipmentOwnershipMutationVariables = Exact<{
  id: string | number;
  input: ShipmentTransferOwnershipInput;
}>;


export type TransferShipmentOwnershipMutation = { transferShipmentOwnership: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } };

export type TransferShipmentToBillingMutationVariables = Exact<{
  input: ShipmentTransferToBillingInput;
}>;


export type TransferShipmentToBillingMutation = { transferShipmentToBilling: { id: string, organizationId: string, businessUnitId: string, shipmentId: string | null, assignedBillerId: string | null, number: string, status: BillingQueueStatus, billType: BillType, exceptionReasonCode: BillingQueueExceptionReasonCode | null, reviewNotes: string, exceptionNotes: string, reviewStartedAt: number | null, reviewCompletedAt: number | null, canceledById: string | null, canceledAt: number | null, cancelReason: string, isAdjustmentOrigin: boolean, sourceInvoiceId: string | null, sourceInvoiceAdjustmentId: string | null, sourceCreditMemoInvoiceId: string | null, correctionGroupId: string | null, rebillStrategy: string | null, requiresReplacementReview: boolean, rerateVariancePercent: string, adjustmentContext: unknown, version: number, createdAt: number, updatedAt: number } };

export type BulkTransferShipmentsToBillingMutationVariables = Exact<{
  input: ShipmentBulkTransferToBillingInput;
}>;


export type BulkTransferShipmentsToBillingMutation = { bulkTransferShipmentsToBilling: { totalCount: number, successCount: number, errorCount: number, results: Array<{ shipmentId: string, success: boolean, error: string | null }> } };

export type CalculateShipmentTotalsMutationVariables = Exact<{
  input: ShipmentInput;
}>;


export type CalculateShipmentTotalsMutation = { calculateShipmentTotals: { freightChargeAmount: string, otherChargeAmount: string, totalChargeAmount: string, fuelSurcharge: { accessorialChargeId: string, isSystemGenerated: boolean, method: string, amount: string, unit: number, fuelSurchargeProgramId: string | null, fuelSurchargeDetail: unknown } | null } };

export type CalculateShipmentDistanceMutationVariables = Exact<{
  input: ShipmentInput;
}>;


export type CalculateShipmentDistanceMutation = { calculateShipmentDistance: { shipmentId: string | null, totalDistance: number, moves: Array<{ moveId: string | null, moveIndex: number, distance: number, source: string, provider: string | null, routingType: string | null, dataVersion: string | null, distanceUnits: string | null, distanceProfileId: string | null, distanceProfileName: string | null, warnings: Array<string> | null, calculatedAt: number }> } };

export type RecalculateShipmentDistanceMutationVariables = Exact<{
  shipmentId: string | number;
}>;


export type RecalculateShipmentDistanceMutation = { recalculateShipmentDistance: { shipmentId: string | null, totalDistance: number, moves: Array<{ moveId: string | null, moveIndex: number, distance: number, source: string, provider: string | null, routingType: string | null, dataVersion: string | null, distanceUnits: string | null, distanceProfileId: string | null, distanceProfileName: string | null, warnings: Array<string> | null, calculatedAt: number }> } };

export type CheckShipmentDuplicateBolMutationVariables = Exact<{
  input: ShipmentDuplicateBolInput;
}>;


export type CheckShipmentDuplicateBolMutation = { checkShipmentDuplicateBol: { valid: boolean } };

export type CheckShipmentHazmatSegregationMutationVariables = Exact<{
  input: ShipmentHazmatInput;
}>;


export type CheckShipmentHazmatSegregationMutation = { checkShipmentHazmatSegregation: { valid: boolean } };

export type CalculateShipmentLoadingOptimizationMutationVariables = Exact<{
  input: ShipmentLoadingOptimizationInput;
}>;


export type CalculateShipmentLoadingOptimizationMutation = { calculateShipmentLoadingOptimization: { trailerLengthFeet: number, totalLinearFeet: number, totalWeight: number, maxWeight: number, linearFeetUtil: number, weightUtil: number, utilizationScore: number, utilizationGrade: string, aiAnalysis: string | null, placements: Array<{ commodityId: string, commodityName: string, positionFeet: number, lengthFeet: number, weight: number, pieces: number, stackable: boolean, fragile: boolean, isHazmat: boolean, hazmatClass: string | null, minTemp: number | null, maxTemp: number | null, loadingInstructions: string | null, estimatedLength: boolean, stopNumber: number | null }>, hazmatZones: Array<{ commodityAId: string, commodityBId: string, commodityAName: string, commodityBName: string, ruleName: string, segregationType: string, requiredDistanceFeet: number | null, actualDistanceFeet: number, satisfied: boolean }>, warnings: Array<{ type: string, message: string, severity: string, commodityIds: Array<string> | null }>, axleWeights: Array<{ axle: string, weight: number, limit: number, percentage: number, compliant: boolean }>, recommendations: Array<{ type: string, priority: string, title: string, description: string, impact: string | null, commodityIds: Array<string> | null }>, stopDividers: Array<{ positionFeet: number, stopNumber: number, label: string }> | null } };

export type CreateShipmentCommentMutationVariables = Exact<{
  shipmentId: string | number;
  input: ShipmentCommentInput;
}>;


export type CreateShipmentCommentMutation = { createShipmentComment: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } };

export type UpdateShipmentCommentMutationVariables = Exact<{
  shipmentId: string | number;
  commentId: string | number;
  input: ShipmentCommentUpdateInput;
}>;


export type UpdateShipmentCommentMutation = { updateShipmentComment: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } };

export type DeleteShipmentCommentMutationVariables = Exact<{
  shipmentId: string | number;
  commentId: string | number;
}>;


export type DeleteShipmentCommentMutation = { deleteShipmentComment: boolean };

export type ShipmentProfitabilityQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type ShipmentProfitabilityQuery = { shipmentProfitability: { shipmentId: string, loadedMiles: number, deadheadMiles: number, totalMiles: number, revenue: string, estimatedCost: string, profit: string, marginPercent: string | null, revenuePerLoadedMile: string | null, breakEvenRpm: string | null, missingDistance: boolean, breakdown: Array<{ category: CostCategoryType, name: string, costBehavior: CostBehavior, ratePerMile: string, amount: string, effectiveSource: EffectiveRateSource }>, profile: { totalCpm: string, variableCpm: string, fixedCpm: string, targetMarginPercent: string | null, includeDeadheadMiles: boolean, asOfDate: string, fuel: { pricePerGallon: string | null, priceDate: string, fuelIndexId: string | null, milesPerGallon: string, source: EffectiveRateSource } | null, glWindow: { fromDate: number, toDate: number, fleetMiles: number, hasPostings: boolean } | null } } };

export type UpdateSidebarPreferencesMutationVariables = Exact<{
  input: SidebarPreferencesInput;
}>;


export type UpdateSidebarPreferencesMutation = { updateSidebarPreferences: { schemaVersion: number, version: number, attentionMetrics: Array<string>, quickActionIds: Array<string>, sections: Array<{ key: string, hidden: boolean }>, activity: { pageSize: number, defaultOpen: boolean } } };

export type SidebarPreferencesQueryVariables = Exact<{ [key: string]: never; }>;


export type SidebarPreferencesQuery = { sidebarPreferences: { schemaVersion: number, version: number, attentionMetrics: Array<string>, quickActionIds: Array<string>, sections: Array<{ key: string, hidden: boolean }>, activity: { pageSize: number, defaultOpen: boolean } } };

export type SidebarCustomizationOptionsQueryVariables = Exact<{ [key: string]: never; }>;


export type SidebarCustomizationOptionsQuery = { sidebarCustomizationOptions: { maxQuickActions: number, activityPageSizes: Array<number>, sections: Array<{ key: string, label: string, hideable: boolean }>, attentionMetrics: Array<{ key: string, label: string }>, quickActions: Array<{ id: string, label: string }> } };

export type StoredMileageStopKeyFieldsFragment = { method: string, key: string, city: string, state: string, postalCode: string, placeId: string, coordinates: Array<number> | null } & { ' $fragmentName'?: 'StoredMileageStopKeyFieldsFragment' };

export type StoredMileageTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: string, routeSignature: string, routeHash: string, distance: number, distanceUnits: string, provider: string, source: string, routingType: string, method: string, distanceProfileId: string, distanceProfileName: string, hitCount: number, lastCalculatedAt: number, version: number, createdAt: number, updatedAt: number, originKey: { ' $fragmentRefs'?: { 'StoredMileageStopKeyFieldsFragment': StoredMileageStopKeyFieldsFragment } }, destinationKey: { ' $fragmentRefs'?: { 'StoredMileageStopKeyFieldsFragment': StoredMileageStopKeyFieldsFragment } }, intermediateKeys: Array<{ ' $fragmentRefs'?: { 'StoredMileageStopKeyFieldsFragment': StoredMileageStopKeyFieldsFragment } }> | null } & { ' $fragmentName'?: 'StoredMileageTableRowFieldsFragment' };

export type StoredMileageTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type StoredMileageTableQuery = { storedMileages: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'StoredMileageTableRowFieldsFragment': StoredMileageTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type TcaSubscriptionTableRowFieldsFragment = { id: string, organizationId: string, businessUnitId: string, userId: string, name: string, tableName: string, recordId: string | null, eventTypes: Array<string>, conditions: Array<unknown>, conditionMatch: string, watchedColumns: Array<string>, customTitle: string, customMessage: string, topic: string, priority: string, status: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'TcaSubscriptionTableRowFieldsFragment' };

export type TcaSubscriptionTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type TcaSubscriptionTableQuery = { tcaSubscriptions: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TcaSubscriptionTableRowFieldsFragment': TcaSubscriptionTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type TableConfigurationFieldsFragment = { id: string, organizationId: string, businessUnitId: string, userId: string, name: string, description: string, resource: string, tableConfig: unknown, visibility: ConfigurationVisibility, isDefault: boolean, isOrgDefault: boolean, version: number, createdAt: number, updatedAt: number, user: { id: string, name: string, profilePicUrl: string } | null } & { ' $fragmentName'?: 'TableConfigurationFieldsFragment' };

export type TableConfigurationTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  resource?: string | null | undefined;
  visibility?: ConfigurationVisibility | null | undefined;
}>;


export type TableConfigurationTableQuery = { tableConfigurations: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TableConfigurationFieldsFragment': TableConfigurationFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DefaultTableConfigurationQueryVariables = Exact<{
  resource: string;
}>;


export type DefaultTableConfigurationQuery = { defaultTableConfiguration: { ' $fragmentRefs'?: { 'TableConfigurationFieldsFragment': TableConfigurationFieldsFragment } } | null };

export type TableConfigurationDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type TableConfigurationDetailQuery = { tableConfiguration: { ' $fragmentRefs'?: { 'TableConfigurationFieldsFragment': TableConfigurationFieldsFragment } } | null };

export type CreateTableConfigurationMutationVariables = Exact<{
  input: TableConfigurationInput;
}>;


export type CreateTableConfigurationMutation = { createTableConfiguration: { ' $fragmentRefs'?: { 'TableConfigurationFieldsFragment': TableConfigurationFieldsFragment } } };

export type UpdateTableConfigurationMutationVariables = Exact<{
  id: string | number;
  input: TableConfigurationInput;
}>;


export type UpdateTableConfigurationMutation = { updateTableConfiguration: { ' $fragmentRefs'?: { 'TableConfigurationFieldsFragment': TableConfigurationFieldsFragment } } };

export type DeleteTableConfigurationMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteTableConfigurationMutation = { deleteTableConfiguration: boolean };

export type SetDefaultTableConfigurationMutationVariables = Exact<{
  id: string | number;
}>;


export type SetDefaultTableConfigurationMutation = { setDefaultTableConfiguration: { ' $fragmentRefs'?: { 'TableConfigurationFieldsFragment': TableConfigurationFieldsFragment } } };

export type SetOrgDefaultTableConfigurationMutationVariables = Exact<{
  id: string | number;
  enabled: boolean;
}>;


export type SetOrgDefaultTableConfigurationMutation = { setOrgDefaultTableConfiguration: { ' $fragmentRefs'?: { 'TableConfigurationFieldsFragment': TableConfigurationFieldsFragment } } };

export type UserTableRowFieldsFragment = { id: string, businessUnitId: string, currentOrganizationId: string, status: EntityStatus, name: string, username: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string, timezone: string, isLocked: boolean, mustChangePassword: boolean, version: number, lastLoginAt: number | null, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'UserTableRowFieldsFragment' };

export type UserTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type UserTableQuery = { users: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'UserTableRowFieldsFragment': UserTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type WorkerFleetCodeFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'WorkerFleetCodeFieldsFragment' };

export type WorkerUsStateFieldsFragment = { id: string, name: string, abbreviation: string } & { ' $fragmentName'?: 'WorkerUsStateFieldsFragment' };

export type WorkerProfileTableFieldsFragment = { id: string, workerId: string, businessUnitId: string, organizationId: string, licenseStateId: string | null, dob: number, licenseNumber: string, cdlClass: CdlClass, cdlRestrictions: string, endorsement: EndorsementType, hazmatExpiry: number | null, licenseExpiry: number, medicalCardExpiry: number | null, medicalExaminerName: string, medicalExaminerNpi: string, twicCardNumber: string, twicExpiry: number | null, hireDate: number, terminationDate: number | null, physicalDueDate: number | null, mvrDueDate: number | null, complianceStatus: ComplianceStatus, isQualified: boolean, disqualificationReason: string, lastComplianceCheck: number, lastMvrCheck: number, lastDrugTest: number, eldExempt: boolean, shortHaulExempt: boolean, version: number, createdAt: number, updatedAt: number, licenseState: { ' $fragmentRefs'?: { 'WorkerUsStateFieldsFragment': WorkerUsStateFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerProfileTableFieldsFragment' };

export type WorkerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, stateId: string, fleetCodeId: string | null, managerId: string | null, status: EntityStatus, type: WorkerType, driverType: DriverType, profilePicUrl: string, firstName: string, lastName: string, wholeName: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, email: string, phoneNumber: string, emergencyContactName: string, emergencyContactPhone: string, externalId: string, assignmentBlocked: string, gender: WorkerGender, canBeAssigned: boolean, availableForDispatch: boolean, version: number, createdAt: number, updatedAt: number, customFields: unknown, fleetCode: { ' $fragmentRefs'?: { 'WorkerFleetCodeFieldsFragment': WorkerFleetCodeFieldsFragment } } | null, state: { ' $fragmentRefs'?: { 'WorkerUsStateFieldsFragment': WorkerUsStateFieldsFragment } } | null, profile: { ' $fragmentRefs'?: { 'WorkerProfileTableFieldsFragment': WorkerProfileTableFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerTableRowFieldsFragment' };

export type WorkerPtoWorkerFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string, profilePicUrl: string } & { ' $fragmentName'?: 'WorkerPtoWorkerFieldsFragment' };

export type WorkerPtoRowFieldsFragment = { id: string, workerId: string, organizationId: string, businessUnitId: string, approverId: string | null, rejectorId: string | null, status: PtoStatus, type: PtoType, startDate: number, endDate: number, reason: string, version: number, createdAt: number, updatedAt: number, worker: { ' $fragmentRefs'?: { 'WorkerPtoWorkerFieldsFragment': WorkerPtoWorkerFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerPtoRowFieldsFragment' };

export type WorkerDataTablePageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'WorkerDataTablePageInfoFieldsFragment' };

export type WorkerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type WorkerTableQuery = { workers: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerTableRowFieldsFragment': WorkerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'WorkerDataTablePageInfoFieldsFragment': WorkerDataTablePageInfoFieldsFragment } } } };

export type WorkerPtoTableQueryVariables = Exact<{
  input: WorkerPtoEntriesInput;
}>;


export type WorkerPtoTableQuery = { workerPTOEntries: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'WorkerDataTablePageInfoFieldsFragment': WorkerDataTablePageInfoFieldsFragment } } } };

export type UpcomingWorkerPtoQueryVariables = Exact<{
  input: UpcomingWorkerPtoInput;
}>;


export type UpcomingWorkerPtoQuery = { upcomingWorkerPTO: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'WorkerDataTablePageInfoFieldsFragment': WorkerDataTablePageInfoFieldsFragment } } } };

export type WorkerPtoChartDataQueryVariables = Exact<{
  input: WorkerPtoChartInput;
}>;


export type WorkerPtoChartDataQuery = { workerPTOChartData: Array<{ date: string, vacation: number, sick: number, holiday: number, bereavement: number, maternity: number, paternity: number, personal: number, workers: unknown }> };

export type PatchWorkerMutationVariables = Exact<{
  id: string | number;
  input: WorkerPatchInput;
}>;


export type PatchWorkerMutation = { patchWorker: { ' $fragmentRefs'?: { 'WorkerTableRowFieldsFragment': WorkerTableRowFieldsFragment } } };

export type ApproveWorkerPtoMutationVariables = Exact<{
  id: string | number;
}>;


export type ApproveWorkerPtoMutation = { approveWorkerPTO: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } };

export type RejectWorkerPtoMutationVariables = Exact<{
  id: string | number;
  reason: string;
}>;


export type RejectWorkerPtoMutation = { rejectWorkerPTO: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } };

export class TypedDocumentString<TResult, TVariables>
  extends String
  implements DocumentTypeDecoration<TResult, TVariables>
{
  __apiType?: NonNullable<DocumentTypeDecoration<TResult, TVariables>['__apiType']>;
  private value: string;
  public __meta__?: Record<string, any> | undefined;

  constructor(value: string, __meta__?: Record<string, any> | undefined) {
    super(value);
    this.value = value;
    this.__meta__ = __meta__;
  }

  override toString(): string & DocumentTypeDecoration<TResult, TVariables> {
    return this.value;
  }
}
export const AccessorialChargeTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment AccessorialChargeTableRowFields on AccessorialCharge {
  id
  businessUnitId
  organizationId
  status
  code
  description
  method
  rateUnit
  amount
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"AccessorialChargeTableRowFields"}) as unknown as TypedDocumentString<AccessorialChargeTableRowFieldsFragment, unknown>;
export const AccountTypeTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment AccountTypeTableRowFields on AccountType {
  id
  businessUnitId
  organizationId
  status
  code
  name
  description
  category
  color
  isSystem
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"AccountTypeTableRowFields"}) as unknown as TypedDocumentString<AccountTypeTableRowFieldsFragment, unknown>;
export const ApiKeyTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment ApiKeyTableRowFields on ApiKey {
  id
  businessUnitId
  organizationId
  name
  description
  keyPrefix
  status
  expiresAt
  lastUsedAt
  permissionScope
  createdAt
  updatedAt
}
    `, {"fragmentName":"ApiKeyTableRowFields"}) as unknown as TypedDocumentString<ApiKeyTableRowFieldsFragment, unknown>;
export const AuditLogTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment AuditLogTableRowFields on AuditEntry {
  id
  userId
  businessUnitId
  organizationId
  timestamp
  changes
  previousState
  currentState
  metadata
  resource
  operation
  resourceId
  correlationId
  userAgent
  comment
  ipAddress
  category
  sensitiveData
  critical
  user {
    id
    name
    username
    emailAddress
    profilePicUrl
    thumbnailUrl
  }
}
    `, {"fragmentName":"AuditLogTableRowFields"}) as unknown as TypedDocumentString<AuditLogTableRowFieldsFragment, unknown>;
export const BillingQueueActionFieldsFragmentDoc = new TypedDocumentString(`
    fragment BillingQueueActionFields on BillingQueueItem {
  id
  organizationId
  businessUnitId
  shipmentId
  assignedBillerId
  number
  status
  billType
  exceptionReasonCode
  reviewNotes
  exceptionNotes
  reviewStartedAt
  reviewCompletedAt
  canceledById
  canceledAt
  cancelReason
  isAdjustmentOrigin
  sourceInvoiceId
  sourceInvoiceAdjustmentId
  sourceCreditMemoInvoiceId
  correctionGroupId
  rebillStrategy
  requiresReplacementReview
  rerateVariancePercent
  adjustmentContext
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"BillingQueueActionFields"}) as unknown as TypedDocumentString<BillingQueueActionFieldsFragment, unknown>;
export const CommodityTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment CommodityTableRowFields on Commodity {
  id
  businessUnitId
  organizationId
  hazardousMaterialId
  status
  name
  description
  minTemperature
  maxTemperature
  weightPerUnit
  linearFeetPerUnit
  maxQuantityPerShipment
  freightClass
  loadingInstructions
  stackable
  fragile
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"CommodityTableRowFields"}) as unknown as TypedDocumentString<CommodityTableRowFieldsFragment, unknown>;
export const CustomFieldDefinitionTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment CustomFieldDefinitionTableRowFields on CustomFieldDefinition {
  id
  businessUnitId
  organizationId
  resourceType
  name
  label
  description
  fieldType
  isRequired
  isActive
  displayOrder
  color
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"CustomFieldDefinitionTableRowFields"}) as unknown as TypedDocumentString<CustomFieldDefinitionTableRowFieldsFragment, unknown>;
export const CustomerBillingProfileFieldsFragmentDoc = new TypedDocumentString(`
    fragment CustomerBillingProfileFields on CustomerBillingProfile {
  id
  businessUnitId
  organizationId
  customerId
  billingCycleType
  billingCycleDayOfWeek
  paymentTerm
  hasBillingControlOverrides
  creditLimit
  creditBalance
  creditStatus
  enforceCreditLimit
  autoCreditHold
  creditHoldReason
  invoiceMethod
  autoSendInvoiceOnGeneration
  allowInvoiceConsolidation
  consolidationPeriodDays
  consolidationGroupBy
  invoiceNumberFormat
  customerInvoicePrefix
  invoiceCopies
  revenueAccountId
  arAccountId
  applyLateCharges
  lateChargeRate
  gracePeriodDays
  taxExempt
  taxExemptNumber
  enforceCustomerBillingReq
  validateCustomerRates
  autoTransfer
  autoMarkReadyToBill
  autoBill
  detentionBillingEnabled
  detentionFreeMinutes
  detentionRatePerHour
  countLateOnlyOnAppointmentStops
  countDetentionOnlyOnAppointmentStops
  autoApplyAccessorials
  billingCurrency
  requirePONumber
  requireBOLNumber
  requireDeliveryNumber
  invoiceAdjustmentSupportingDocumentPolicy
  defaultBillerId
  billingNotes
  fuelSurchargeMode
  fuelSurchargeProgramId
  documentTypes {
    id
    code
    name
    color
    documentClassification
    documentCategory
  }
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"CustomerBillingProfileFields"}) as unknown as TypedDocumentString<CustomerBillingProfileFieldsFragment, unknown>;
export const CustomerEmailProfileFieldsFragmentDoc = new TypedDocumentString(`
    fragment CustomerEmailProfileFields on CustomerEmailProfile {
  id
  businessUnitId
  organizationId
  customerId
  subject
  comment
  fromEmail
  toRecipients
  ccRecipients
  bccRecipients
  attachmentName
  readReceipt
  includeShipmentDetail
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"CustomerEmailProfileFields"}) as unknown as TypedDocumentString<CustomerEmailProfileFieldsFragment, unknown>;
export const CustomerTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment CustomerTableRowFields on Customer {
  id
  businessUnitId
  organizationId
  stateId
  status
  code
  name
  addressLine1
  addressLine2
  city
  postalCode
  isGeocoded
  longitude
  latitude
  placeId
  externalId
  allowConsolidation
  exclusiveConsolidation
  consolidationPriority
  version
  createdAt
  updatedAt
  billingProfile {
    ...CustomerBillingProfileFields
  }
  emailProfile {
    ...CustomerEmailProfileFields
  }
}
    fragment CustomerBillingProfileFields on CustomerBillingProfile {
  id
  businessUnitId
  organizationId
  customerId
  billingCycleType
  billingCycleDayOfWeek
  paymentTerm
  hasBillingControlOverrides
  creditLimit
  creditBalance
  creditStatus
  enforceCreditLimit
  autoCreditHold
  creditHoldReason
  invoiceMethod
  autoSendInvoiceOnGeneration
  allowInvoiceConsolidation
  consolidationPeriodDays
  consolidationGroupBy
  invoiceNumberFormat
  customerInvoicePrefix
  invoiceCopies
  revenueAccountId
  arAccountId
  applyLateCharges
  lateChargeRate
  gracePeriodDays
  taxExempt
  taxExemptNumber
  enforceCustomerBillingReq
  validateCustomerRates
  autoTransfer
  autoMarkReadyToBill
  autoBill
  detentionBillingEnabled
  detentionFreeMinutes
  detentionRatePerHour
  countLateOnlyOnAppointmentStops
  countDetentionOnlyOnAppointmentStops
  autoApplyAccessorials
  billingCurrency
  requirePONumber
  requireBOLNumber
  requireDeliveryNumber
  invoiceAdjustmentSupportingDocumentPolicy
  defaultBillerId
  billingNotes
  fuelSurchargeMode
  fuelSurchargeProgramId
  documentTypes {
    id
    code
    name
    color
    documentClassification
    documentCategory
  }
  version
  createdAt
  updatedAt
}
fragment CustomerEmailProfileFields on CustomerEmailProfile {
  id
  businessUnitId
  organizationId
  customerId
  subject
  comment
  fromEmail
  toRecipients
  ccRecipients
  bccRecipients
  attachmentName
  readReceipt
  includeShipmentDetail
  version
  createdAt
  updatedAt
}`, {"fragmentName":"CustomerTableRowFields"}) as unknown as TypedDocumentString<CustomerTableRowFieldsFragment, unknown>;
export const DistanceOverrideLocationFieldsFragmentDoc = new TypedDocumentString(`
    fragment DistanceOverrideLocationFields on Location {
  id
  name
  addressLine1
  addressLine2
  city
  postalCode
  state {
    id
    abbreviation
  }
}
    `, {"fragmentName":"DistanceOverrideLocationFields"}) as unknown as TypedDocumentString<DistanceOverrideLocationFieldsFragment, unknown>;
export const DistanceOverrideTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment DistanceOverrideTableRowFields on DistanceOverride {
  id
  businessUnitId
  organizationId
  originLocationId
  destinationLocationId
  customerId
  distance
  version
  createdAt
  updatedAt
  originLocation {
    ...DistanceOverrideLocationFields
  }
  destinationLocation {
    ...DistanceOverrideLocationFields
  }
  customer {
    id
    name
  }
  intermediateStops {
    locationId
    stopOrder
  }
}
    fragment DistanceOverrideLocationFields on Location {
  id
  name
  addressLine1
  addressLine2
  city
  postalCode
  state {
    id
    abbreviation
  }
}`, {"fragmentName":"DistanceOverrideTableRowFields"}) as unknown as TypedDocumentString<DistanceOverrideTableRowFieldsFragment, unknown>;
export const DistanceProfileTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment DistanceProfileTableRowFields on DistanceProfile {
  id
  businessUnitId
  organizationId
  name
  description
  status
  isDefault
  provider
  dataVersion
  region
  routingType
  distanceUnits
  locationGranularity
  profileName
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"DistanceProfileTableRowFields"}) as unknown as TypedDocumentString<DistanceProfileTableRowFieldsFragment, unknown>;
export const DocumentPacketRuleTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment DocumentPacketRuleTableRowFields on DocumentPacketRule {
  id
  businessUnitId
  organizationId
  resourceType
  documentTypeId
  required
  allowMultiple
  displayOrder
  expirationRequired
  expirationWarningDays
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"DocumentPacketRuleTableRowFields"}) as unknown as TypedDocumentString<DocumentPacketRuleTableRowFieldsFragment, unknown>;
export const DocumentTypeTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment DocumentTypeTableRowFields on DocumentType {
  id
  businessUnitId
  organizationId
  code
  name
  description
  color
  documentClassification
  documentCategory
  isSystem
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"DocumentTypeTableRowFields"}) as unknown as TypedDocumentString<DocumentTypeTableRowFieldsFragment, unknown>;
export const EdiTemplateVersionSummaryFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiTemplateVersionSummaryFields on EdiTemplateVersion {
  id
  businessUnitId
  organizationId
  templateId
  sourceVersionId
  versionNumber
  x12Version
  functionalGroupId
  status
  isActive
  notes
  certifiedAt
  activatedAt
  archivedAt
  deprecatedAt
  supersededAt
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"EdiTemplateVersionSummaryFields"}) as unknown as TypedDocumentString<EdiTemplateVersionSummaryFieldsFragment, unknown>;
export const EdiTemplateListFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiTemplateListFields on EdiTemplate {
  id
  businessUnitId
  organizationId
  documentTypeId
  name
  description
  direction
  standard
  transactionSet
  status
  version
  createdAt
  updatedAt
  versions {
    ...EdiTemplateVersionSummaryFields
  }
}
    fragment EdiTemplateVersionSummaryFields on EdiTemplateVersion {
  id
  businessUnitId
  organizationId
  templateId
  sourceVersionId
  versionNumber
  x12Version
  functionalGroupId
  status
  isActive
  notes
  certifiedAt
  activatedAt
  archivedAt
  deprecatedAt
  supersededAt
  version
  createdAt
  updatedAt
}`, {"fragmentName":"EdiTemplateListFields"}) as unknown as TypedDocumentString<EdiTemplateListFieldsFragment, unknown>;
export const EdiPartnerRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiPartnerRowFields on EdiPartner {
  id
  businessUnitId
  organizationId
  kind
  status
  code
  name
  description
  internalOrganizationId
  customerId
  defaultTransportId
  defaultMappingProfileId
  country
  timezone
  contactName
  contactEmail
  contactPhone
  enabledForInbound
  enabledForOutbound
  version
  createdAt
  updatedAt
  internalOrganization {
    id
    name
  }
  connection {
    id
    method
    status
  }
  defaultTransport {
    id
    name
    method
  }
}
    `, {"fragmentName":"EdiPartnerRowFields"}) as unknown as TypedDocumentString<EdiPartnerRowFieldsFragment, unknown>;
export const EdiCommunicationProfileRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiCommunicationProfileRowFields on EdiCommunicationProfile {
  id
  businessUnitId
  organizationId
  ediPartnerId
  ediConnectionId
  method
  status
  name
  description
  secretState {
    key
  }
  version
  createdAt
  updatedAt
  partner {
    id
    code
    name
  }
}
    `, {"fragmentName":"EdiCommunicationProfileRowFields"}) as unknown as TypedDocumentString<EdiCommunicationProfileRowFieldsFragment, unknown>;
export const EdiTransferRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiTransferRowFields on EdiTransfer {
  id
  sourceOrganizationId
  sourceBusinessUnitId
  targetOrganizationId
  targetBusinessUnitId
  sourcePartnerId
  targetPartnerId
  sourceShipmentId
  targetShipmentId
  inboundMessageId
  status
  tenderPayload
  mappingSnapshot
  rejectionReason
  failureReason
  submittedAt
  processedAt
  version
  createdAt
  updatedAt
  sourcePartner {
    id
    code
    name
  }
  targetPartner {
    id
    code
    name
  }
}
    `, {"fragmentName":"EdiTransferRowFields"}) as unknown as TypedDocumentString<EdiTransferRowFieldsFragment, unknown>;
export const EdiMessageRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiMessageRowFields on EdiMessage {
  id
  businessUnitId
  organizationId
  ediPartnerId
  documentTypeId
  partnerDocumentProfileId
  shipmentId
  transferId
  inboundFileId
  direction
  transactionSet
  x12Version
  status
  interchangeControlNumber
  groupControlNumber
  transactionControlNumber
  segmentCount
  deliveryStatus
  deliveryRemotePath
  deliveryAttempts
  deliveryLastAttemptAt
  deliverySentAt
  deliveryLastError
  ackStatus
  ackMessageId
  ackReceivedAt
  ackLastError
  generatedAt
  version
  partner {
    id
    code
    name
  }
}
    `, {"fragmentName":"EdiMessageRowFields"}) as unknown as TypedDocumentString<EdiMessageRowFieldsFragment, unknown>;
export const EdiInboundFileRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiInboundFileRowFields on EdiInboundFile {
  id
  businessUnitId
  organizationId
  communicationProfileId
  ediPartnerId
  method
  remotePath
  fileName
  checksum
  sizeBytes
  interchangeControlNumber
  isaSenderQualifier
  isaSenderId
  isaReceiverQualifier
  isaReceiverId
  status
  failureReason
  transactionCount
  receivedAt
  processedAt
  version
  partner {
    id
    code
    name
  }
}
    `, {"fragmentName":"EdiInboundFileRowFields"}) as unknown as TypedDocumentString<EdiInboundFileRowFieldsFragment, unknown>;
export const EdiMappingProfileRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiMappingProfileRowFields on EdiMappingProfile {
  id
  businessUnitId
  organizationId
  ediPartnerId
  name
  description
  version
  createdAt
  updatedAt
  partner {
    id
    code
    name
  }
  entries {
    id
    entityType
    sourceId
    sourceLabel
    targetId
    targetLabel
  }
}
    `, {"fragmentName":"EdiMappingProfileRowFields"}) as unknown as TypedDocumentString<EdiMappingProfileRowFieldsFragment, unknown>;
export const EdiTestCaseRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EdiTestCaseRowFields on EdiTestCase {
  id
  businessUnitId
  organizationId
  partnerDocumentProfileId
  name
  description
  expectedWarnings
  expectedErrors
  version
  createdAt
  updatedAt
  documentProfile {
    id
    name
    direction
    transactionSet
    partner {
      id
      code
      name
    }
  }
}
    `, {"fragmentName":"EdiTestCaseRowFields"}) as unknown as TypedDocumentString<EdiTestCaseRowFieldsFragment, unknown>;
export const EmailProfileTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EmailProfileTableRowFields on EmailProfile {
  id
  businessUnitId
  organizationId
  name
  description
  senderName
  senderEmail
  replyToEmail
  provider
  status
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"EmailProfileTableRowFields"}) as unknown as TypedDocumentString<EmailProfileTableRowFieldsFragment, unknown>;
export const EquipmentManufacturerTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EquipmentManufacturerTableRowFields on EquipmentManufacturer {
  id
  businessUnitId
  organizationId
  status
  name
  description
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"EquipmentManufacturerTableRowFields"}) as unknown as TypedDocumentString<EquipmentManufacturerTableRowFieldsFragment, unknown>;
export const EquipmentTypeConfigurationRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment EquipmentTypeConfigurationRowFields on EquipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  class
  color
  interiorLength
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"EquipmentTypeConfigurationRowFields"}) as unknown as TypedDocumentString<EquipmentTypeConfigurationRowFieldsFragment, unknown>;
export const DataTablePageInfoFieldsFragmentDoc = new TypedDocumentString(`
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
    `, {"fragmentName":"DataTablePageInfoFields"}) as unknown as TypedDocumentString<DataTablePageInfoFieldsFragment, unknown>;
export const EquipmentTypeTableFieldsFragmentDoc = new TypedDocumentString(`
    fragment EquipmentTypeTableFields on EquipmentType {
  id
  code
  color
}
    `, {"fragmentName":"EquipmentTypeTableFields"}) as unknown as TypedDocumentString<EquipmentTypeTableFieldsFragment, unknown>;
export const EquipmentManufacturerTableFieldsFragmentDoc = new TypedDocumentString(`
    fragment EquipmentManufacturerTableFields on EquipmentManufacturer {
  id
  name
}
    `, {"fragmentName":"EquipmentManufacturerTableFields"}) as unknown as TypedDocumentString<EquipmentManufacturerTableFieldsFragment, unknown>;
export const FleetCodeTableFieldsFragmentDoc = new TypedDocumentString(`
    fragment FleetCodeTableFields on FleetCode {
  id
  code
  color
}
    `, {"fragmentName":"FleetCodeTableFields"}) as unknown as TypedDocumentString<FleetCodeTableFieldsFragment, unknown>;
export const UsStateTableFieldsFragmentDoc = new TypedDocumentString(`
    fragment UsStateTableFields on UsState {
  id
  name
  abbreviation
}
    `, {"fragmentName":"UsStateTableFields"}) as unknown as TypedDocumentString<UsStateTableFieldsFragment, unknown>;
export const WorkerTableReferenceFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerTableReferenceFields on Worker {
  id
  firstName
  lastName
  wholeName
}
    `, {"fragmentName":"WorkerTableReferenceFields"}) as unknown as TypedDocumentString<WorkerTableReferenceFieldsFragment, unknown>;
export const TractorTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment TractorTableRowFields on Tractor {
  id
  businessUnitId
  organizationId
  primaryWorkerId
  equipmentTypeId
  equipmentManufacturerId
  stateId
  fleetCodeId
  secondaryWorkerId
  status
  code
  model
  make
  year
  licensePlateNumber
  registrationNumber
  registrationExpiry
  vin
  lastKnownLocationId
  lastKnownLocationName
  version
  createdAt
  updatedAt
  customFields
  equipmentType {
    ...EquipmentTypeTableFields
  }
  equipmentManufacturer {
    ...EquipmentManufacturerTableFields
  }
  fleetCode {
    ...FleetCodeTableFields
  }
  state {
    ...UsStateTableFields
  }
  primaryWorker {
    ...WorkerTableReferenceFields
  }
  secondaryWorker {
    ...WorkerTableReferenceFields
  }
}
    fragment EquipmentTypeTableFields on EquipmentType {
  id
  code
  color
}
fragment EquipmentManufacturerTableFields on EquipmentManufacturer {
  id
  name
}
fragment FleetCodeTableFields on FleetCode {
  id
  code
  color
}
fragment UsStateTableFields on UsState {
  id
  name
  abbreviation
}
fragment WorkerTableReferenceFields on Worker {
  id
  firstName
  lastName
  wholeName
}`, {"fragmentName":"TractorTableRowFields"}) as unknown as TypedDocumentString<TractorTableRowFieldsFragment, unknown>;
export const TrailerTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment TrailerTableRowFields on Trailer {
  id
  businessUnitId
  organizationId
  equipmentTypeId
  equipmentManufacturerId
  registrationStateId
  fleetCodeId
  status
  code
  model
  make
  year
  licensePlateNumber
  vin
  registrationNumber
  maxLoadWeight
  lastInspectionDate
  registrationExpiry
  lastKnownLocationId
  lastKnownLocationName
  version
  createdAt
  updatedAt
  customFields
  equipmentType {
    ...EquipmentTypeTableFields
  }
  equipmentManufacturer {
    ...EquipmentManufacturerTableFields
  }
  fleetCode {
    ...FleetCodeTableFields
  }
  registrationState {
    ...UsStateTableFields
  }
}
    fragment EquipmentTypeTableFields on EquipmentType {
  id
  code
  color
}
fragment EquipmentManufacturerTableFields on EquipmentManufacturer {
  id
  name
}
fragment FleetCodeTableFields on FleetCode {
  id
  code
  color
}
fragment UsStateTableFields on UsState {
  id
  name
  abbreviation
}`, {"fragmentName":"TrailerTableRowFields"}) as unknown as TypedDocumentString<TrailerTableRowFieldsFragment, unknown>;
export const FiscalPeriodFieldsFragmentDoc = new TypedDocumentString(`
    fragment FiscalPeriodFields on FiscalPeriod {
  id
  businessUnitId
  organizationId
  fiscalYearId
  periodNumber
  periodType
  status
  name
  startDate
  endDate
  closedAt
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"FiscalPeriodFields"}) as unknown as TypedDocumentString<FiscalPeriodFieldsFragment, unknown>;
export const FiscalYearTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment FiscalYearTableRowFields on FiscalYear {
  id
  businessUnitId
  organizationId
  status
  year
  name
  description
  startDate
  endDate
  isCurrent
  isCalendarYear
  allowAdjustingEntries
  version
  createdAt
  updatedAt
  periods {
    ...FiscalPeriodFields
  }
}
    fragment FiscalPeriodFields on FiscalPeriod {
  id
  businessUnitId
  organizationId
  fiscalYearId
  periodNumber
  periodType
  status
  name
  startDate
  endDate
  closedAt
  version
  createdAt
  updatedAt
}`, {"fragmentName":"FiscalYearTableRowFields"}) as unknown as TypedDocumentString<FiscalYearTableRowFieldsFragment, unknown>;
export const FleetCodeTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment FleetCodeTableRowFields on FleetCode {
  id
  businessUnitId
  organizationId
  managerId
  status
  code
  description
  revenueGoal
  deadheadGoal
  mileageGoal
  color
  version
  createdAt
  updatedAt
  manager {
    id
    name
  }
}
    `, {"fragmentName":"FleetCodeTableRowFields"}) as unknown as TypedDocumentString<FleetCodeTableRowFieldsFragment, unknown>;
export const FormulaTemplateTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment FormulaTemplateTableRowFields on FormulaTemplate {
  id
  businessUnitId
  organizationId
  name
  description
  type
  expression
  status
  schemaId
  version
  currentVersionNumber
  createdAt
  updatedAt
}
    `, {"fragmentName":"FormulaTemplateTableRowFields"}) as unknown as TypedDocumentString<FormulaTemplateTableRowFieldsFragment, unknown>;
export const FuelIndexFieldsFragmentDoc = new TypedDocumentString(`
    fragment FuelIndexFields on FuelIndex {
  id
  businessUnitId
  organizationId
  name
  code
  description
  source
  fuelType
  region
  eiaSeriesId
  currency
  isActive
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"FuelIndexFields"}) as unknown as TypedDocumentString<FuelIndexFieldsFragment, unknown>;
export const FuelSurchargeProgramFieldsFragmentDoc = new TypedDocumentString(`
    fragment FuelSurchargeProgramFields on FuelSurchargeProgram {
  id
  businessUnitId
  organizationId
  name
  code
  description
  status
  fuelIndexId
  accessorialChargeId
  method
  pegPrice
  increment
  incrementRate
  milesPerGallon
  percentBasis
  stepRounding
  rateRounding
  ratePrecision
  minAmount
  maxAmount
  dateBasis
  priceEffectiveDay
  missingPriceFallback
  effectiveStartDate
  effectiveEndDate
  shipmentTypeIds
  serviceTypeIds
  tractorTypeIds
  trailerTypeIds
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"FuelSurchargeProgramFields"}) as unknown as TypedDocumentString<FuelSurchargeProgramFieldsFragment, unknown>;
export const HazardousMaterialTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment HazardousMaterialTableRowFields on HazardousMaterial {
  id
  businessUnitId
  organizationId
  status
  code
  name
  description
  class
  unNumber
  packingGroup
  subsidiaryHazardClass
  ergGuideNumber
  labelCodes
  specialProvisions
  properShippingName
  handlingInstructions
  emergencyContact
  emergencyContactPhoneNumber
  quantityThreshold
  placardRequired
  isReportableQuantity
  marinePollutant
  inhalationHazard
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"HazardousMaterialTableRowFields"}) as unknown as TypedDocumentString<HazardousMaterialTableRowFieldsFragment, unknown>;
export const HazmatSegregationRuleTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment HazmatSegregationRuleTableRowFields on HazmatSegregationRule {
  id
  businessUnitId
  organizationId
  status
  name
  description
  exceptionNotes
  referenceCode
  regulationSource
  distanceUnit
  classA
  classB
  segregationType
  hasExceptions
  hazmatAId
  hazmatBId
  minimumDistance
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"HazmatSegregationRuleTableRowFields"}) as unknown as TypedDocumentString<HazmatSegregationRuleTableRowFieldsFragment, unknown>;
export const HoldReasonTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment HoldReasonTableRowFields on HoldReason {
  id
  businessUnitId
  organizationId
  type
  code
  label
  description
  active
  defaultSeverity
  defaultBlocksDispatch
  defaultBlocksDelivery
  defaultBlocksBilling
  defaultVisibleToCustomer
  sortOrder
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"HoldReasonTableRowFields"}) as unknown as TypedDocumentString<HoldReasonTableRowFieldsFragment, unknown>;
export const InvoiceTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment InvoiceTableRowFields on Invoice {
  id
  billingQueueItemId
  shipmentId
  customerId
  number
  billType
  status
  paymentTerm
  currencyCode
  invoiceDate
  dueDate
  billToName
  subtotalAmount
  otherAmount
  totalAmount
  appliedAmount
  settlementStatus
  disputeStatus
  sendStatus
  isAdjustmentArtifact
  version
  createdAt
  updatedAt
  customer {
    id
    name
    code
  }
}
    `, {"fragmentName":"InvoiceTableRowFields"}) as unknown as TypedDocumentString<InvoiceTableRowFieldsFragment, unknown>;
export const JournalReversalTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment JournalReversalTableRowFields on JournalReversal {
  id
  businessUnitId
  organizationId
  originalJournalEntryId
  reversalJournalEntryId
  postedBatchId
  status
  requestedAccountingDate
  resolvedFiscalYearId
  resolvedFiscalPeriodId
  reasonCode
  reasonText
  requestedById
  approvedById
  approvedAt
  rejectedById
  rejectedAt
  rejectionReason
  cancelledById
  cancelledAt
  cancelReason
  postedById
  postedAt
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"JournalReversalTableRowFields"}) as unknown as TypedDocumentString<JournalReversalTableRowFieldsFragment, unknown>;
export const LocationCategoryTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment LocationCategoryTableRowFields on LocationCategory {
  id
  businessUnitId
  organizationId
  name
  description
  type
  facilityType
  color
  hasSecureParking
  requiresAppointment
  allowsOvernight
  hasRestroom
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"LocationCategoryTableRowFields"}) as unknown as TypedDocumentString<LocationCategoryTableRowFieldsFragment, unknown>;
export const LocationTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment LocationTableRowFields on Location {
  id
  businessUnitId
  organizationId
  locationCategoryId
  stateId
  status
  code
  name
  description
  addressLine1
  addressLine2
  city
  postalCode
  version
  createdAt
  updatedAt
  state {
    id
    name
    abbreviation
  }
  locationCategory {
    id
    name
    color
  }
}
    `, {"fragmentName":"LocationTableRowFields"}) as unknown as TypedDocumentString<LocationTableRowFieldsFragment, unknown>;
export const ManualJournalTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment ManualJournalTableRowFields on ManualJournal {
  id
  businessUnitId
  organizationId
  requestNumber
  status
  description
  reason
  accountingDate
  requestedFiscalYearId
  requestedFiscalPeriodId
  currencyCode
  totalDebit
  totalCredit
  approvedAt
  approvedById
  rejectedAt
  rejectedById
  rejectionReason
  cancelledAt
  cancelledById
  cancelReason
  postedBatchId
  createdById
  updatedById
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ManualJournalTableRowFields"}) as unknown as TypedDocumentString<ManualJournalTableRowFieldsFragment, unknown>;
export const NotificationFieldsFragmentDoc = new TypedDocumentString(`
    fragment NotificationFields on Notification {
  id
  organizationId
  businessUnitId
  targetUserId
  eventType
  priority
  channel
  title
  message
  data
  relatedEntities
  source
  readAt
  dismissedAt
  createdAt
}
    `, {"fragmentName":"NotificationFields"}) as unknown as TypedDocumentString<NotificationFieldsFragment, unknown>;
export const OrderMutationResultFragmentDoc = new TypedDocumentString(`
    fragment OrderMutationResult on Order {
  id
  orderNumber
  status
  totalAmount
  version
}
    `, {"fragmentName":"OrderMutationResult"}) as unknown as TypedDocumentString<OrderMutationResultFragment, unknown>;
export const OrderTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment OrderTableRowFields on Order {
  id
  ownerId
  businessUnitId
  organizationId
  customerId
  status
  orderNumber
  poNumber
  bol
  currencyCode
  quotedAmount
  baseAmount
  totalAmount
  version
  createdAt
  updatedAt
  customer {
    id
    name
    code
  }
}
    `, {"fragmentName":"OrderTableRowFields"}) as unknown as TypedDocumentString<OrderTableRowFieldsFragment, unknown>;
export const OrganizationSettingsStateFieldsFragmentDoc = new TypedDocumentString(`
    fragment OrganizationSettingsStateFields on UsState {
  id
  name
  abbreviation
}
    `, {"fragmentName":"OrganizationSettingsStateFields"}) as unknown as TypedDocumentString<OrganizationSettingsStateFieldsFragment, unknown>;
export const OrganizationSettingsFieldsFragmentDoc = new TypedDocumentString(`
    fragment OrganizationSettingsFields on Organization {
  id
  version
  createdAt
  updatedAt
  bucketName
  businessUnitId
  loginSlug
  name
  scacCode
  dotNumber
  logoUrl
  addressLine1
  addressLine2
  city
  stateId
  postalCode
  timezone
  taxId
  state {
    ...OrganizationSettingsStateFields
  }
}
    fragment OrganizationSettingsStateFields on UsState {
  id
  name
  abbreviation
}`, {"fragmentName":"OrganizationSettingsFields"}) as unknown as TypedDocumentString<OrganizationSettingsFieldsFragment, unknown>;
export const RateTableTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment RateTableTableRowFields on RateTable {
  id
  businessUnitId
  organizationId
  name
  key
  description
  lookupType
  active
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"RateTableTableRowFields"}) as unknown as TypedDocumentString<RateTableTableRowFieldsFragment, unknown>;
export const RecurringShipmentTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment RecurringShipmentTableRowFields on RecurringShipment {
  id
  businessUnitId
  organizationId
  sourceShipmentId
  customerId
  originLocationId
  destinationLocationId
  name
  description
  status
  cronExpression
  timezone
  startDate
  endDate
  maxOccurrences
  leadTimeDays
  skipWeekends
  exceptionPolicy
  blackoutDates
  autoGenerate
  nextOccurrenceAt
  lastOccurrenceAt
  lastRunAt
  generationCount
  consecutiveFailures
  version
  createdAt
  updatedAt
  customer {
    id
    name
    code
  }
  originLocation {
    id
    name
    code
  }
  destinationLocation {
    id
    name
    code
  }
}
    `, {"fragmentName":"RecurringShipmentTableRowFields"}) as unknown as TypedDocumentString<RecurringShipmentTableRowFieldsFragment, unknown>;
export const ReportDefinitionFieldsFragmentDoc = new TypedDocumentString(`
    fragment ReportDefinitionFields on ReportDefinition {
  id
  name
  description
  category
  tags
  kind
  cannedKey
  cannedVersion
  ownerId
  visibility
  status
  diagnostics
  catalogVersion
  definition
  defaultFormat
  currentRevision
  lastRunAt
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ReportDefinitionFields"}) as unknown as TypedDocumentString<ReportDefinitionFieldsFragment, unknown>;
export const ReportRunFieldsFragmentDoc = new TypedDocumentString(`
    fragment ReportRunFields on ReportRun {
  id
  definitionId
  revisionId
  cannedKey
  cannedVersion
  requestedById
  trigger
  params
  format
  status
  rowCount
  byteSize
  durationMs
  truncated
  error {
    code
    message
    detail
  }
  artifactExpiresAt
  cacheHit
  queuedAt
  startedAt
  completedAt
  version
  createdAt
}
    `, {"fragmentName":"ReportRunFields"}) as unknown as TypedDocumentString<ReportRunFieldsFragment, unknown>;
export const ReportScheduleFieldsFragmentDoc = new TypedDocumentString(`
    fragment ReportScheduleFields on ReportSchedule {
  id
  definitionId
  cronExpression
  timezone
  formats
  emailRecipients
  emailAttach
  notifyUserIds
  enabled
  runAsId
  lastRunId
  nextRunAt
  consecutiveFailures
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ReportScheduleFields"}) as unknown as TypedDocumentString<ReportScheduleFieldsFragment, unknown>;
export const RoleTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment RoleTableRowFields on Role {
  id
  businessUnitId
  organizationId
  name
  description
  coreResponsibility
  parentRoleIds
  maxSensitivity
  isSystem
  createdBy
  createdAt
  updatedAt
}
    `, {"fragmentName":"RoleTableRowFields"}) as unknown as TypedDocumentString<RoleTableRowFieldsFragment, unknown>;
export const ScimGroupRoleMappingTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment SCIMGroupRoleMappingTableRowFields on SCIMGroupRoleMapping {
  id
  directoryId
  externalGroupId
  displayName
  roleId
  version
  role {
    id
    name
  }
}
    `, {"fragmentName":"SCIMGroupRoleMappingTableRowFields"}) as unknown as TypedDocumentString<ScimGroupRoleMappingTableRowFieldsFragment, unknown>;
export const ServiceFailureReasonCodeTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment ServiceFailureReasonCodeTableRowFields on ServiceFailureReasonCode {
  id
  businessUnitId
  organizationId
  code
  label
  description
  category
  appliesTo
  defaultStatusCode
  defaultReasonCode
  defaultExceptionCode
  defaultNote
  active
  sortOrder
  externalMap
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ServiceFailureReasonCodeTableRowFields"}) as unknown as TypedDocumentString<ServiceFailureReasonCodeTableRowFieldsFragment, unknown>;
export const ServiceFailureTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment ServiceFailureTableRowFields on ServiceFailure {
  id
  shipmentId
  number
  type
  source
  status
  stopType
  stopId
  scheduledCutoff
  actualArrival
  gracePeriodMinutes
  lateMinutes
  reasonCodeId
  notes
  detectedAt
  version
  shipment {
    id
    proNumber
    bol
  }
  stop {
    id
    type
    sequence
    locationId
    location {
      id
      name
      code
      city
      state {
        abbreviation
      }
    }
  }
  reasonCode {
    id
    code
    label
  }
}
    `, {"fragmentName":"ServiceFailureTableRowFields"}) as unknown as TypedDocumentString<ServiceFailureTableRowFieldsFragment, unknown>;
export const ServiceTypeTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment ServiceTypeTableRowFields on ServiceType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  color
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ServiceTypeTableRowFields"}) as unknown as TypedDocumentString<ServiceTypeTableRowFieldsFragment, unknown>;
export const ShipmentTypeTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentTypeTableRowFields on ShipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  color
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ShipmentTypeTableRowFields"}) as unknown as TypedDocumentString<ShipmentTypeTableRowFieldsFragment, unknown>;
export const ShipmentRatingDetailFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
    `, {"fragmentName":"ShipmentRatingDetailFields"}) as unknown as TypedDocumentString<ShipmentRatingDetailFieldsFragment, unknown>;
export const ShipmentLocationFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
    `, {"fragmentName":"ShipmentLocationFields"}) as unknown as TypedDocumentString<ShipmentLocationFieldsFragment, unknown>;
export const ShipmentStopFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
    fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}`, {"fragmentName":"ShipmentStopFields"}) as unknown as TypedDocumentString<ShipmentStopFieldsFragment, unknown>;
export const ShipmentTractorFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentTractorFields on Tractor {
  id
  code
}
    `, {"fragmentName":"ShipmentTractorFields"}) as unknown as TypedDocumentString<ShipmentTractorFieldsFragment, unknown>;
export const ShipmentTrailerFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
    `, {"fragmentName":"ShipmentTrailerFields"}) as unknown as TypedDocumentString<ShipmentTrailerFieldsFragment, unknown>;
export const ShipmentWorkerFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
    `, {"fragmentName":"ShipmentWorkerFields"}) as unknown as TypedDocumentString<ShipmentWorkerFieldsFragment, unknown>;
export const ShipmentAssignmentFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
    fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}`, {"fragmentName":"ShipmentAssignmentFields"}) as unknown as TypedDocumentString<ShipmentAssignmentFieldsFragment, unknown>;
export const ShipmentMoveFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
    fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}`, {"fragmentName":"ShipmentMoveFields"}) as unknown as TypedDocumentString<ShipmentMoveFieldsFragment, unknown>;
export const ShipmentAdditionalChargeFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
    `, {"fragmentName":"ShipmentAdditionalChargeFields"}) as unknown as TypedDocumentString<ShipmentAdditionalChargeFieldsFragment, unknown>;
export const ShipmentCommodityFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
    `, {"fragmentName":"ShipmentCommodityFields"}) as unknown as TypedDocumentString<ShipmentCommodityFieldsFragment, unknown>;
export const ShipmentUserFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
    `, {"fragmentName":"ShipmentUserFields"}) as unknown as TypedDocumentString<ShipmentUserFieldsFragment, unknown>;
export const ShipmentFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}`, {"fragmentName":"ShipmentFields"}) as unknown as TypedDocumentString<ShipmentFieldsFragment, unknown>;
export const ShipmentPageInfoFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentPageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
    `, {"fragmentName":"ShipmentPageInfoFields"}) as unknown as TypedDocumentString<ShipmentPageInfoFieldsFragment, unknown>;
export const ShipmentCommentMentionFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentCommentMentionFields on ShipmentCommentMention {
  id
  commentId
  mentionedUserId
  organizationId
  businessUnitId
  shipmentId
  createdAt
  mentionedUser {
    ...ShipmentUserFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}`, {"fragmentName":"ShipmentCommentMentionFields"}) as unknown as TypedDocumentString<ShipmentCommentMentionFieldsFragment, unknown>;
export const ShipmentCommentFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentCommentFields on ShipmentComment {
  id
  businessUnitId
  organizationId
  shipmentId
  userId
  comment
  type
  visibility
  priority
  source
  metadata
  editedAt
  version
  createdAt
  updatedAt
  mentionedUserIds
  user {
    ...ShipmentUserFields
  }
  mentionedUsers {
    ...ShipmentCommentMentionFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentCommentMentionFields on ShipmentCommentMention {
  id
  commentId
  mentionedUserId
  organizationId
  businessUnitId
  shipmentId
  createdAt
  mentionedUser {
    ...ShipmentUserFields
  }
}`, {"fragmentName":"ShipmentCommentFields"}) as unknown as TypedDocumentString<ShipmentCommentFieldsFragment, unknown>;
export const ShipmentEventFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentEventFields on ShipmentEvent {
  id
  organizationId
  businessUnitId
  shipmentId
  moveId
  stopId
  assignmentId
  commentId
  holdId
  type
  severity
  actorType
  actorId
  actorLabel
  summary
  proNumber
  previousStatus
  newStatus
  reason
  previousOwnerId
  newOwnerId
  primaryWorkerId
  secondaryWorkerId
  tractorId
  trailerId
  driverName
  holdType
  holdSeverity
  holdSource
  commentBody
  commentType
  commentVisibility
  commentPriority
  mentionedUserIds
  metadata
  occurredAt
  correlationId
  actor {
    id
    name
    emailAddress
    profilePicUrl
    thumbnailUrl
  }
  shipment {
    id
    proNumber
  }
}
    `, {"fragmentName":"ShipmentEventFields"}) as unknown as TypedDocumentString<ShipmentEventFieldsFragment, unknown>;
export const StoredMileageStopKeyFieldsFragmentDoc = new TypedDocumentString(`
    fragment StoredMileageStopKeyFields on StopKey {
  method
  key
  city
  state
  postalCode
  placeId
  coordinates
}
    `, {"fragmentName":"StoredMileageStopKeyFields"}) as unknown as TypedDocumentString<StoredMileageStopKeyFieldsFragment, unknown>;
export const StoredMileageTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment StoredMileageTableRowFields on StoredMileage {
  id
  businessUnitId
  organizationId
  status
  originKey {
    ...StoredMileageStopKeyFields
  }
  destinationKey {
    ...StoredMileageStopKeyFields
  }
  intermediateKeys {
    ...StoredMileageStopKeyFields
  }
  routeSignature
  routeHash
  distance
  distanceUnits
  provider
  source
  routingType
  method
  distanceProfileId
  distanceProfileName
  hitCount
  lastCalculatedAt
  version
  createdAt
  updatedAt
}
    fragment StoredMileageStopKeyFields on StopKey {
  method
  key
  city
  state
  postalCode
  placeId
  coordinates
}`, {"fragmentName":"StoredMileageTableRowFields"}) as unknown as TypedDocumentString<StoredMileageTableRowFieldsFragment, unknown>;
export const TcaSubscriptionTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment TCASubscriptionTableRowFields on TCASubscription {
  id
  organizationId
  businessUnitId
  userId
  name
  tableName
  recordId
  eventTypes
  conditions
  conditionMatch
  watchedColumns
  customTitle
  customMessage
  topic
  priority
  status
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"TCASubscriptionTableRowFields"}) as unknown as TypedDocumentString<TcaSubscriptionTableRowFieldsFragment, unknown>;
export const TableConfigurationFieldsFragmentDoc = new TypedDocumentString(`
    fragment TableConfigurationFields on TableConfiguration {
  id
  organizationId
  businessUnitId
  userId
  name
  description
  resource
  tableConfig
  visibility
  isDefault
  isOrgDefault
  version
  createdAt
  updatedAt
  user {
    id
    name
    profilePicUrl
  }
}
    `, {"fragmentName":"TableConfigurationFields"}) as unknown as TypedDocumentString<TableConfigurationFieldsFragment, unknown>;
export const UserTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment UserTableRowFields on User {
  id
  businessUnitId
  currentOrganizationId
  status
  name
  username
  emailAddress
  profilePicUrl
  thumbnailUrl
  timezone
  isLocked
  mustChangePassword
  version
  lastLoginAt
  createdAt
  updatedAt
}
    `, {"fragmentName":"UserTableRowFields"}) as unknown as TypedDocumentString<UserTableRowFieldsFragment, unknown>;
export const WorkerFleetCodeFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerFleetCodeFields on FleetCode {
  id
  code
  color
}
    `, {"fragmentName":"WorkerFleetCodeFields"}) as unknown as TypedDocumentString<WorkerFleetCodeFieldsFragment, unknown>;
export const WorkerUsStateFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerUsStateFields on UsState {
  id
  name
  abbreviation
}
    `, {"fragmentName":"WorkerUsStateFields"}) as unknown as TypedDocumentString<WorkerUsStateFieldsFragment, unknown>;
export const WorkerProfileTableFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerProfileTableFields on WorkerProfile {
  id
  workerId
  businessUnitId
  organizationId
  licenseStateId
  dob
  licenseNumber
  cdlClass
  cdlRestrictions
  endorsement
  hazmatExpiry
  licenseExpiry
  medicalCardExpiry
  medicalExaminerName
  medicalExaminerNpi
  twicCardNumber
  twicExpiry
  hireDate
  terminationDate
  physicalDueDate
  mvrDueDate
  complianceStatus
  isQualified
  disqualificationReason
  lastComplianceCheck
  lastMvrCheck
  lastDrugTest
  eldExempt
  shortHaulExempt
  version
  createdAt
  updatedAt
  licenseState {
    ...WorkerUsStateFields
  }
}
    fragment WorkerUsStateFields on UsState {
  id
  name
  abbreviation
}`, {"fragmentName":"WorkerProfileTableFields"}) as unknown as TypedDocumentString<WorkerProfileTableFieldsFragment, unknown>;
export const WorkerTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerTableRowFields on Worker {
  id
  businessUnitId
  organizationId
  stateId
  fleetCodeId
  managerId
  status
  type
  driverType
  profilePicUrl
  firstName
  lastName
  wholeName
  addressLine1
  addressLine2
  city
  postalCode
  email
  phoneNumber
  emergencyContactName
  emergencyContactPhone
  externalId
  assignmentBlocked
  gender
  canBeAssigned
  availableForDispatch
  version
  createdAt
  updatedAt
  customFields
  fleetCode {
    ...WorkerFleetCodeFields
  }
  state {
    ...WorkerUsStateFields
  }
  profile {
    ...WorkerProfileTableFields
  }
}
    fragment WorkerFleetCodeFields on FleetCode {
  id
  code
  color
}
fragment WorkerUsStateFields on UsState {
  id
  name
  abbreviation
}
fragment WorkerProfileTableFields on WorkerProfile {
  id
  workerId
  businessUnitId
  organizationId
  licenseStateId
  dob
  licenseNumber
  cdlClass
  cdlRestrictions
  endorsement
  hazmatExpiry
  licenseExpiry
  medicalCardExpiry
  medicalExaminerName
  medicalExaminerNpi
  twicCardNumber
  twicExpiry
  hireDate
  terminationDate
  physicalDueDate
  mvrDueDate
  complianceStatus
  isQualified
  disqualificationReason
  lastComplianceCheck
  lastMvrCheck
  lastDrugTest
  eldExempt
  shortHaulExempt
  version
  createdAt
  updatedAt
  licenseState {
    ...WorkerUsStateFields
  }
}`, {"fragmentName":"WorkerTableRowFields"}) as unknown as TypedDocumentString<WorkerTableRowFieldsFragment, unknown>;
export const WorkerPtoWorkerFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerPtoWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
    `, {"fragmentName":"WorkerPtoWorkerFields"}) as unknown as TypedDocumentString<WorkerPtoWorkerFieldsFragment, unknown>;
export const WorkerPtoRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerPtoRowFields on WorkerPTO {
  id
  workerId
  organizationId
  businessUnitId
  approverId
  rejectorId
  status
  type
  startDate
  endDate
  reason
  version
  createdAt
  updatedAt
  worker {
    ...WorkerPtoWorkerFields
  }
}
    fragment WorkerPtoWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}`, {"fragmentName":"WorkerPtoRowFields"}) as unknown as TypedDocumentString<WorkerPtoRowFieldsFragment, unknown>;
export const WorkerDataTablePageInfoFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerDataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
    `, {"fragmentName":"WorkerDataTablePageInfoFields"}) as unknown as TypedDocumentString<WorkerDataTablePageInfoFieldsFragment, unknown>;
export const AccessorialChargeTableDocument = new TypedDocumentString(`
    query AccessorialChargeTable($input: DataTableConnectionInput!) {
  accessorialCharges(input: $input) {
    edges {
      node {
        ...AccessorialChargeTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment AccessorialChargeTableRowFields on AccessorialCharge {
  id
  businessUnitId
  organizationId
  status
  code
  description
  method
  rateUnit
  amount
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:27a2e818673246434e1ff9e5eeee1b5d8636cd03e000b1687a499cb7ac62e03e"}) as unknown as TypedDocumentString<AccessorialChargeTableQuery, AccessorialChargeTableQueryVariables>;
export const AccountTypeTableDocument = new TypedDocumentString(`
    query AccountTypeTable($input: DataTableConnectionInput!) {
  accountTypes(input: $input) {
    edges {
      node {
        ...AccountTypeTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment AccountTypeTableRowFields on AccountType {
  id
  businessUnitId
  organizationId
  status
  code
  name
  description
  category
  color
  isSystem
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:2822d802104dfabd7fe2f2d1ca2f602dc8252429dbef72cd86a9cb81d97c1a11"}) as unknown as TypedDocumentString<AccountTypeTableQuery, AccountTypeTableQueryVariables>;
export const ArAgingSummaryDocument = new TypedDocumentString(`
    query ArAgingSummary($asOfDate: Int) {
  arAgingSummary(asOfDate: $asOfDate) {
    asOfDate
    totals {
      currentMinor
      days1To30Minor
      days31To60Minor
      days61To90Minor
      daysOver90Minor
      totalOpenMinor
    }
    rows {
      customerId
      customerName
      buckets {
        currentMinor
        days1To30Minor
        days31To60Minor
        days61To90Minor
        daysOver90Minor
        totalOpenMinor
      }
    }
  }
}
    `, {"hash":"sha256:e15d59a908dce2b711f04ab748e41238803ed73f0e5a3566b1adfb553521f532"}) as unknown as TypedDocumentString<ArAgingSummaryQuery, ArAgingSummaryQueryVariables>;
export const ArOpenItemsDocument = new TypedDocumentString(`
    query ArOpenItems($customerId: ID, $asOfDate: Int) {
  arOpenItems(customerId: $customerId, asOfDate: $asOfDate) {
    invoiceId
    customerId
    customerName
    invoiceNumber
    billType
    invoiceDate
    dueDate
    currencyCode
    shipmentProNumber
    shipmentBol
    totalAmountMinor
    appliedAmountMinor
    openAmountMinor
    daysPastDue
    settlementStatus
    disputeStatus
    hasShortPay
  }
}
    `, {"hash":"sha256:03f9416ae24afd23531a73b23ace5f76aaea88bd287a1aae0b12bd75424fedc2"}) as unknown as TypedDocumentString<ArOpenItemsQuery, ArOpenItemsQueryVariables>;
export const ArCustomerLedgerDocument = new TypedDocumentString(`
    query ArCustomerLedger($customerId: ID!) {
  arCustomerLedger(customerId: $customerId) {
    customerId
    transactionDate
    eventType
    documentNumber
    sourceObjectType
    sourceObjectId
    amountMinor
    relatedInvoiceId
  }
}
    `, {"hash":"sha256:d9554d04015d108ca655b857f923eae260ae36f143a509025f79717a4fdf65c4"}) as unknown as TypedDocumentString<ArCustomerLedgerQuery, ArCustomerLedgerQueryVariables>;
export const ArCustomerStatementDocument = new TypedDocumentString(`
    query ArCustomerStatement($customerId: ID!, $startDate: Int, $asOfDate: Int) {
  arCustomerStatement(
    customerId: $customerId
    startDate: $startDate
    asOfDate: $asOfDate
  ) {
    customerId
    customerName
    statementDate
    startDate
    openingBalanceMinor
    totalChargesMinor
    totalPaymentsMinor
    endingBalanceMinor
    aging {
      currentMinor
      days1To30Minor
      days31To60Minor
      days61To90Minor
      daysOver90Minor
      totalOpenMinor
    }
    transactions {
      transactionDate
      eventType
      documentNumber
      sourceObjectId
      amountMinor
      chargeMinor
      paymentMinor
      runningBalanceMinor
    }
    openItems {
      invoiceId
      customerId
      customerName
      invoiceNumber
      billType
      invoiceDate
      dueDate
      currencyCode
      shipmentProNumber
      shipmentBol
      totalAmountMinor
      appliedAmountMinor
      openAmountMinor
      daysPastDue
      settlementStatus
      disputeStatus
      hasShortPay
    }
  }
}
    `, {"hash":"sha256:02fa7018fe90124e84ec3909ad97282fdad2a2ee737eea5465e6ca2ca52a0b70"}) as unknown as TypedDocumentString<ArCustomerStatementQuery, ArCustomerStatementQueryVariables>;
export const ArDashboardKpisDocument = new TypedDocumentString(`
    query ArDashboardKpis {
  arDashboardKpis {
    asOfDate
    overview {
      totalOpenMinor
      overdueMinor
      unappliedCashMinor
      disputedOpenMinor
      openInvoiceCount
      overdueInvoiceCount
      disputedInvoiceCount
      avgDaysPastDue
      buckets {
        currentMinor
        days1To30Minor
        days31To60Minor
        days61To90Minor
        daysOver90Minor
        totalOpenMinor
      }
    }
    currentDsoDays
    dsoDeltaDays
    cei
    avgDaysToPay
    overduePercent
    writeOffRatio
    disputeRate
    shortPayRate
  }
}
    `, {"hash":"sha256:a05fd02645fddb941d688972550b27be3ce6665510aecb4170c44dd11c491452"}) as unknown as TypedDocumentString<ArDashboardKpisQuery, ArDashboardKpisQueryVariables>;
export const ArDsoTrendDocument = new TypedDocumentString(`
    query ArDsoTrend($weeks: Int) {
  arDsoTrend(weeks: $weeks) {
    periodEnd
    dsoDays
    arBalanceMinor
    billedMinor
  }
}
    `, {"hash":"sha256:031255e1f9c64413b438a2b3825d9dd8c2101fb1705a6c0e6e66175243334571"}) as unknown as TypedDocumentString<ArDsoTrendQuery, ArDsoTrendQueryVariables>;
export const ArAgingTrendDocument = new TypedDocumentString(`
    query ArAgingTrend($weeks: Int) {
  arAgingTrend(weeks: $weeks) {
    periodEnd
    buckets {
      currentMinor
      days1To30Minor
      days31To60Minor
      days61To90Minor
      daysOver90Minor
      totalOpenMinor
    }
  }
}
    `, {"hash":"sha256:ef0336444abe3d8b2ab3101645e9f24367fbb81bda8e4c66f902a2fbfedda191"}) as unknown as TypedDocumentString<ArAgingTrendQuery, ArAgingTrendQueryVariables>;
export const ArCashFlowForecastDocument = new TypedDocumentString(`
    query ArCashFlowForecast($pastWeeks: Int, $futureWeeks: Int) {
  arCashFlowForecast(pastWeeks: $pastWeeks, futureWeeks: $futureWeeks) {
    weekStart
    expectedMinor
    openDueMinor
    actualMinor
    isForecast
  }
}
    `, {"hash":"sha256:a5a6d234e6dcbd2be9e4244023746ee4c4644b2f93108d908e5f457f74031246"}) as unknown as TypedDocumentString<ArCashFlowForecastQuery, ArCashFlowForecastQueryVariables>;
export const ArCollectionPerformanceDocument = new TypedDocumentString(`
    query ArCollectionPerformance($periodDays: Int) {
  arCollectionPerformance(periodDays: $periodDays) {
    totals {
      periodStart
      periodEnd
      beginningOpenMinor
      endingOpenMinor
      endingCurrentMinor
      creditSalesMinor
      collectedMinor
      avgDaysToPay
      shortPayMinor
      shortPayApplicationCount
      applicationCount
      disputedInvoiceCount
      postedInvoiceCount
    }
    cei
    writeOffRatio
    disputeRate
    shortPayRate
  }
}
    `, {"hash":"sha256:d3e0b01af564df96113a703515f4c80fdefdbd7b2e70914bd8b7567153fc7e66"}) as unknown as TypedDocumentString<ArCollectionPerformanceQuery, ArCollectionPerformanceQueryVariables>;
export const ArTopOverdueCustomersDocument = new TypedDocumentString(`
    query ArTopOverdueCustomers($limit: Int) {
  arTopOverdueCustomers(limit: $limit) {
    customerId
    customerName
    overdueMinor
    totalOpenMinor
    oldestDaysPastDue
    openInvoiceCount
  }
}
    `, {"hash":"sha256:dddd0e21b1ac01157ee2f9d7b763f1f83ec482078ae54b30d159aa1e3641e408"}) as unknown as TypedDocumentString<ArTopOverdueCustomersQuery, ArTopOverdueCustomersQueryVariables>;
export const ArCollectionsWorklistDocument = new TypedDocumentString(`
    query ArCollectionsWorklist($limit: Int) {
  arCollectionsWorklist(limit: $limit) {
    invoiceId
    customerId
    customerName
    invoiceNumber
    dueDate
    openAmountMinor
    daysPastDue
    isDisputed
    hasShortPay
    severity
  }
}
    `, {"hash":"sha256:a72cdc4147d001e5acaecc0659d81becc1a7fb642c63c8e3f79240651ffadb40"}) as unknown as TypedDocumentString<ArCollectionsWorklistQuery, ArCollectionsWorklistQueryVariables>;
export const ArPaymentStatsDocument = new TypedDocumentString(`
    query ArPaymentStats {
  arPaymentStats {
    postedTodayMinor
    postedTodayCount
    unappliedCashMinor
    unappliedPaymentCount
    reversedLast30Minor
    reversedLast30Count
  }
}
    `, {"hash":"sha256:a4fe33f6233932aadde3e5ec2e4dc656c78b2e73188bab35638f674c3045bbeb"}) as unknown as TypedDocumentString<ArPaymentStatsQuery, ArPaymentStatsQueryVariables>;
export const ArCustomerProfileDocument = new TypedDocumentString(`
    query ArCustomerProfile($customerId: ID!) {
  arCustomerProfile(customerId: $customerId) {
    snapshot {
      customerId
      customerName
      totalOpenMinor
      overdueMinor
      unappliedCashMinor
      creditLimitMinor
      hasCreditLimit
      openInvoiceCount
      oldestOpenInvoiceDate
      oldestDaysPastDue
      lastPaymentDate
      lastPaymentMinor
      avgDaysToPay
      billedTrailing91Minor
      buckets {
        currentMinor
        days1To30Minor
        days31To60Minor
        days61To90Minor
        daysOver90Minor
        totalOpenMinor
      }
      monthlyCollections {
        monthStart
        amountMinor
      }
    }
    dsoDays
    creditUtilization
    delinquencyScore
  }
}
    `, {"hash":"sha256:b82086fc8a84f2dcc1c322b26634a1465bf4d240b6a5ff5f9bfd36300fbe7b37"}) as unknown as TypedDocumentString<ArCustomerProfileQuery, ArCustomerProfileQueryVariables>;
export const ApiKeyTableDocument = new TypedDocumentString(`
    query ApiKeyTable($input: DataTableConnectionInput!) {
  apiKeys(input: $input) {
    edges {
      node {
        ...ApiKeyTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment ApiKeyTableRowFields on ApiKey {
  id
  businessUnitId
  organizationId
  name
  description
  keyPrefix
  status
  expiresAt
  lastUsedAt
  permissionScope
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:eedf9586fa2250cef08f3db2d0bdd0e4626ef2ba3a1bfa718672099b06537d89"}) as unknown as TypedDocumentString<ApiKeyTableQuery, ApiKeyTableQueryVariables>;
export const AttentionSummaryDocument = new TypedDocumentString(`
    query AttentionSummary {
  attentionSummary {
    billingQueue
    pendingApprovals
    reconciliationExceptions
    serviceFailures
    ediAttention
  }
}
    `, {"hash":"sha256:f60e116af6655582ac87c443bb9a03a7990af7fdd7671aa37452ea7bc48074b9"}) as unknown as TypedDocumentString<AttentionSummaryQuery, AttentionSummaryQueryVariables>;
export const RecentActivityDocument = new TypedDocumentString(`
    query RecentActivity($first: Int!, $after: String) {
  auditEntries(input: { first: $first, after: $after }) {
    edges {
      node {
        id
        resource
        operation
        resourceId
        timestamp
        comment
        entityRef
        user {
          id
          name
          username
          profilePicUrl
          thumbnailUrl
        }
      }
    }
    pageInfo {
      endCursor
      hasNextPage
    }
  }
}
    `, {"hash":"sha256:3fe2bf53bf715a8f3c32bf9ed4a7f61e822b4d79f54f0564b6b647cf54b13407"}) as unknown as TypedDocumentString<RecentActivityQuery, RecentActivityQueryVariables>;
export const AuditLogTableDocument = new TypedDocumentString(`
    query AuditLogTable($input: DataTableConnectionInput!) {
  auditEntries(input: $input) {
    edges {
      node {
        ...AuditLogTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment AuditLogTableRowFields on AuditEntry {
  id
  userId
  businessUnitId
  organizationId
  timestamp
  changes
  previousState
  currentState
  metadata
  resource
  operation
  resourceId
  correlationId
  userAgent
  comment
  ipAddress
  category
  sensitiveData
  critical
  user {
    id
    name
    username
    emailAddress
    profilePicUrl
    thumbnailUrl
  }
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:38a7fba8573aabdc8c3200c54ad7144482fa3304ff8e2094264b45ada14a4e59"}) as unknown as TypedDocumentString<AuditLogTableQuery, AuditLogTableQueryVariables>;
export const UpdateBillingQueueStatusDocument = new TypedDocumentString(`
    mutation UpdateBillingQueueStatus($id: ID!, $input: BillingQueueUpdateStatusInput!) {
  updateBillingQueueStatus(id: $id, input: $input) {
    ...BillingQueueActionFields
  }
}
    fragment BillingQueueActionFields on BillingQueueItem {
  id
  organizationId
  businessUnitId
  shipmentId
  assignedBillerId
  number
  status
  billType
  exceptionReasonCode
  reviewNotes
  exceptionNotes
  reviewStartedAt
  reviewCompletedAt
  canceledById
  canceledAt
  cancelReason
  isAdjustmentOrigin
  sourceInvoiceId
  sourceInvoiceAdjustmentId
  sourceCreditMemoInvoiceId
  correctionGroupId
  rebillStrategy
  requiresReplacementReview
  rerateVariancePercent
  adjustmentContext
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:941445cb1c9ee5677f5b115525133c96fcdac96cbfa2591e8f587620cba61272"}) as unknown as TypedDocumentString<UpdateBillingQueueStatusMutation, UpdateBillingQueueStatusMutationVariables>;
export const AssignBillingQueueBillerDocument = new TypedDocumentString(`
    mutation AssignBillingQueueBiller($id: ID!, $input: BillingQueueAssignInput!) {
  assignBillingQueueBiller(id: $id, input: $input) {
    ...BillingQueueActionFields
  }
}
    fragment BillingQueueActionFields on BillingQueueItem {
  id
  organizationId
  businessUnitId
  shipmentId
  assignedBillerId
  number
  status
  billType
  exceptionReasonCode
  reviewNotes
  exceptionNotes
  reviewStartedAt
  reviewCompletedAt
  canceledById
  canceledAt
  cancelReason
  isAdjustmentOrigin
  sourceInvoiceId
  sourceInvoiceAdjustmentId
  sourceCreditMemoInvoiceId
  correctionGroupId
  rebillStrategy
  requiresReplacementReview
  rerateVariancePercent
  adjustmentContext
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:9850f8da7824976fc1edee6d78fb22738914fc8ffefb4d3588e54e6c6f66bd9f"}) as unknown as TypedDocumentString<AssignBillingQueueBillerMutation, AssignBillingQueueBillerMutationVariables>;
export const CommodityTableDocument = new TypedDocumentString(`
    query CommodityTable($input: DataTableConnectionInput!) {
  commodities(input: $input) {
    edges {
      node {
        ...CommodityTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment CommodityTableRowFields on Commodity {
  id
  businessUnitId
  organizationId
  hazardousMaterialId
  status
  name
  description
  minTemperature
  maxTemperature
  weightPerUnit
  linearFeetPerUnit
  maxQuantityPerShipment
  freightClass
  loadingInstructions
  stackable
  fragile
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:041b888950efa127280047210c6b31f1c0cc7acae490838d5dbe274e6f5c9e82"}) as unknown as TypedDocumentString<CommodityTableQuery, CommodityTableQueryVariables>;
export const CostingControlPageDocument = new TypedDocumentString(`
    query CostingControlPage {
  costingControl {
    id
    businessUnitId
    organizationId
    fuelIndexId
    fuelIndex {
      id
      name
      code
      source
      fuelType
      isActive
    }
    useLiveFuelPrice
    milesPerGallon
    includeDeadheadMiles
    glActualsEnabled
    glRollingMonths
    plannedMonthlyMiles
    targetMarginPercent
    version
    createdAt
    updatedAt
    categories {
      id
      category
      name
      costBehavior
      rateSource
      benchmarkRatePerMile
      overrideRatePerMile
      isActive
      sortOrder
      version
      glAccounts {
        id
        glAccountId
        accountCode
        accountName
      }
    }
  }
}
    `, {"hash":"sha256:a85cccb870b7669eca888497e484d84fee24b403957e9d3bbf4ff03b337551ea"}) as unknown as TypedDocumentString<CostingControlPageQuery, CostingControlPageQueryVariables>;
export const ResolvedCostProfilePageDocument = new TypedDocumentString(`
    query ResolvedCostProfilePage($asOfDate: String) {
  resolvedCostProfile(asOfDate: $asOfDate) {
    totalCpm
    variableCpm
    fixedCpm
    targetMarginPercent
    includeDeadheadMiles
    asOfDate
    fuel {
      pricePerGallon
      priceDate
      fuelIndexId
      milesPerGallon
      source
    }
    categories {
      category
      name
      costBehavior
      ratePerMile
      effectiveSource
    }
    glWindow {
      fromDate
      toDate
      fleetMiles
      hasPostings
    }
  }
}
    `, {"hash":"sha256:0b2352614b5935706f571748ef919218386d7f44cee104d0a0382467511caf67"}) as unknown as TypedDocumentString<ResolvedCostProfilePageQuery, ResolvedCostProfilePageQueryVariables>;
export const UpdateCostingControlDocument = new TypedDocumentString(`
    mutation UpdateCostingControl($input: CostingControlInput!) {
  updateCostingControl(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:c1105bb2e20563d6d25437d41bf5dc0dbe04cd9acf23164e4782478cbd49fa0d"}) as unknown as TypedDocumentString<UpdateCostingControlMutation, UpdateCostingControlMutationVariables>;
export const UpdateCostCategoryDocument = new TypedDocumentString(`
    mutation UpdateCostCategory($input: CostCategoryUpdateInput!) {
  updateCostCategory(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:2c74749981a6ed8680896dcf651f47a87e3aa698c2d3bf3c5c9b717e359ca828"}) as unknown as TypedDocumentString<UpdateCostCategoryMutation, UpdateCostCategoryMutationVariables>;
export const CustomFieldDefinitionTableDocument = new TypedDocumentString(`
    query CustomFieldDefinitionTable($input: DataTableConnectionInput!) {
  customFieldDefinitions(input: $input) {
    edges {
      node {
        ...CustomFieldDefinitionTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment CustomFieldDefinitionTableRowFields on CustomFieldDefinition {
  id
  businessUnitId
  organizationId
  resourceType
  name
  label
  description
  fieldType
  isRequired
  isActive
  displayOrder
  color
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:8adb1344e082bff954969a271a629448765d0bf043201cbc840773ef215a8e11"}) as unknown as TypedDocumentString<CustomFieldDefinitionTableQuery, CustomFieldDefinitionTableQueryVariables>;
export const CustomerPaymentTableDocument = new TypedDocumentString(`
    query CustomerPaymentTable($input: DataTableConnectionInput!) {
  customerPayments(input: $input) {
    edges {
      node {
        id
        organizationId
        businessUnitId
        customerId
        paymentDate
        accountingDate
        amountMinor
        appliedAmountMinor
        unappliedAmountMinor
        status
        paymentMethod
        referenceNumber
        memo
        currencyCode
        postedBatchId
        reversalBatchId
        reversedById
        reversedAt
        reversalReason
        createdById
        updatedById
        version
        createdAt
        updatedAt
        customer {
          id
          code
          name
        }
        applications {
          id
          customerPaymentId
          invoiceId
          appliedAmountMinor
          shortPayAmountMinor
          lineNumber
          createdAt
          updatedAt
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:95620bd474574dcb0e33efe78fadb92dd80a9a653322950286efc88f36eeb35d"}) as unknown as TypedDocumentString<CustomerPaymentTableQuery, CustomerPaymentTableQueryVariables>;
export const CustomerPaymentDetailDocument = new TypedDocumentString(`
    query CustomerPaymentDetail($id: ID!) {
  customerPayment(id: $id) {
    id
    organizationId
    businessUnitId
    customerId
    paymentDate
    accountingDate
    amountMinor
    appliedAmountMinor
    unappliedAmountMinor
    status
    paymentMethod
    referenceNumber
    memo
    currencyCode
    postedBatchId
    reversalBatchId
    reversedById
    reversedAt
    reversalReason
    createdById
    updatedById
    version
    createdAt
    updatedAt
    customer {
      id
      code
      name
    }
    applications {
      id
      customerPaymentId
      invoiceId
      appliedAmountMinor
      shortPayAmountMinor
      lineNumber
      createdAt
      updatedAt
      invoice {
        id
        number
        invoiceDate
        dueDate
        totalAmount
        appliedAmount
        settlementStatus
        disputeStatus
        billToName
      }
    }
  }
}
    `, {"hash":"sha256:63ba0ffb6ef2656f7dc2721a0f6fc0da8dc0d184ee08e3871f26e1048b6a9c3f"}) as unknown as TypedDocumentString<CustomerPaymentDetailQuery, CustomerPaymentDetailQueryVariables>;
export const PostAndApplyCustomerPaymentDocument = new TypedDocumentString(`
    mutation PostAndApplyCustomerPayment($input: PostCustomerPaymentInput!) {
  postAndApplyCustomerPayment(input: $input) {
    id
    customerId
    paymentDate
    accountingDate
    amountMinor
    appliedAmountMinor
    unappliedAmountMinor
    status
    paymentMethod
    referenceNumber
    memo
    currencyCode
    postedBatchId
    createdAt
    updatedAt
    applications {
      id
      invoiceId
      appliedAmountMinor
      shortPayAmountMinor
      lineNumber
    }
  }
}
    `, {"hash":"sha256:8509b32952e2ba614257d3189c57cbd58da45afbb0dafb31f17958e117f62ea6"}) as unknown as TypedDocumentString<PostAndApplyCustomerPaymentMutation, PostAndApplyCustomerPaymentMutationVariables>;
export const ApplyUnappliedCustomerPaymentDocument = new TypedDocumentString(`
    mutation ApplyUnappliedCustomerPayment($input: ApplyCustomerPaymentInput!) {
  applyUnappliedCustomerPayment(input: $input) {
    id
    customerId
    amountMinor
    appliedAmountMinor
    unappliedAmountMinor
    status
    updatedAt
    applications {
      id
      invoiceId
      appliedAmountMinor
      shortPayAmountMinor
      lineNumber
    }
  }
}
    `, {"hash":"sha256:1c0798232c1c035894870e85c421a9f0f214ff7407eb9f35155cfd9b87a9b4e0"}) as unknown as TypedDocumentString<ApplyUnappliedCustomerPaymentMutation, ApplyUnappliedCustomerPaymentMutationVariables>;
export const ReverseCustomerPaymentDocument = new TypedDocumentString(`
    mutation ReverseCustomerPayment($input: ReverseCustomerPaymentInput!) {
  reverseCustomerPayment(input: $input) {
    id
    customerId
    amountMinor
    appliedAmountMinor
    unappliedAmountMinor
    status
    reversalBatchId
    reversedById
    reversedAt
    reversalReason
    updatedAt
    applications {
      id
      invoiceId
      appliedAmountMinor
      shortPayAmountMinor
      lineNumber
    }
  }
}
    `, {"hash":"sha256:fe84be95798f92734cec53df3348b8593909fafde378deb329d583affe25145f"}) as unknown as TypedDocumentString<ReverseCustomerPaymentMutation, ReverseCustomerPaymentMutationVariables>;
export const CustomerTableDocument = new TypedDocumentString(`
    query CustomerTable($input: DataTableConnectionInput!) {
  customers(input: $input) {
    edges {
      node {
        ...CustomerTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment CustomerBillingProfileFields on CustomerBillingProfile {
  id
  businessUnitId
  organizationId
  customerId
  billingCycleType
  billingCycleDayOfWeek
  paymentTerm
  hasBillingControlOverrides
  creditLimit
  creditBalance
  creditStatus
  enforceCreditLimit
  autoCreditHold
  creditHoldReason
  invoiceMethod
  autoSendInvoiceOnGeneration
  allowInvoiceConsolidation
  consolidationPeriodDays
  consolidationGroupBy
  invoiceNumberFormat
  customerInvoicePrefix
  invoiceCopies
  revenueAccountId
  arAccountId
  applyLateCharges
  lateChargeRate
  gracePeriodDays
  taxExempt
  taxExemptNumber
  enforceCustomerBillingReq
  validateCustomerRates
  autoTransfer
  autoMarkReadyToBill
  autoBill
  detentionBillingEnabled
  detentionFreeMinutes
  detentionRatePerHour
  countLateOnlyOnAppointmentStops
  countDetentionOnlyOnAppointmentStops
  autoApplyAccessorials
  billingCurrency
  requirePONumber
  requireBOLNumber
  requireDeliveryNumber
  invoiceAdjustmentSupportingDocumentPolicy
  defaultBillerId
  billingNotes
  fuelSurchargeMode
  fuelSurchargeProgramId
  documentTypes {
    id
    code
    name
    color
    documentClassification
    documentCategory
  }
  version
  createdAt
  updatedAt
}
fragment CustomerEmailProfileFields on CustomerEmailProfile {
  id
  businessUnitId
  organizationId
  customerId
  subject
  comment
  fromEmail
  toRecipients
  ccRecipients
  bccRecipients
  attachmentName
  readReceipt
  includeShipmentDetail
  version
  createdAt
  updatedAt
}
fragment CustomerTableRowFields on Customer {
  id
  businessUnitId
  organizationId
  stateId
  status
  code
  name
  addressLine1
  addressLine2
  city
  postalCode
  isGeocoded
  longitude
  latitude
  placeId
  externalId
  allowConsolidation
  exclusiveConsolidation
  consolidationPriority
  version
  createdAt
  updatedAt
  billingProfile {
    ...CustomerBillingProfileFields
  }
  emailProfile {
    ...CustomerEmailProfileFields
  }
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:5a193fd06a8c6ee5b581c9fad145441766cfbb218821bc3ee33b0b4dfefb4322"}) as unknown as TypedDocumentString<CustomerTableQuery, CustomerTableQueryVariables>;
export const DistanceOverrideTableDocument = new TypedDocumentString(`
    query DistanceOverrideTable($input: DataTableConnectionInput!) {
  distanceOverrides(input: $input) {
    edges {
      node {
        ...DistanceOverrideTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DistanceOverrideLocationFields on Location {
  id
  name
  addressLine1
  addressLine2
  city
  postalCode
  state {
    id
    abbreviation
  }
}
fragment DistanceOverrideTableRowFields on DistanceOverride {
  id
  businessUnitId
  organizationId
  originLocationId
  destinationLocationId
  customerId
  distance
  version
  createdAt
  updatedAt
  originLocation {
    ...DistanceOverrideLocationFields
  }
  destinationLocation {
    ...DistanceOverrideLocationFields
  }
  customer {
    id
    name
  }
  intermediateStops {
    locationId
    stopOrder
  }
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:64a708bb1057e5597b782607190db2f5813c3a2913a09ed3a77489508b6c2c3b"}) as unknown as TypedDocumentString<DistanceOverrideTableQuery, DistanceOverrideTableQueryVariables>;
export const DistanceProfileTableDocument = new TypedDocumentString(`
    query DistanceProfileTable($input: DataTableConnectionInput!) {
  distanceProfiles(input: $input) {
    edges {
      node {
        ...DistanceProfileTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DistanceProfileTableRowFields on DistanceProfile {
  id
  businessUnitId
  organizationId
  name
  description
  status
  isDefault
  provider
  dataVersion
  region
  routingType
  distanceUnits
  locationGranularity
  profileName
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:9b47b610d1875f28f673905f7bec10c67fe6f178b06ce0af560b2fe8b758967f"}) as unknown as TypedDocumentString<DistanceProfileTableQuery, DistanceProfileTableQueryVariables>;
export const DocumentPacketRuleTableDocument = new TypedDocumentString(`
    query DocumentPacketRuleTable($input: DataTableConnectionInput!) {
  documentPacketRules(input: $input) {
    edges {
      node {
        ...DocumentPacketRuleTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DocumentPacketRuleTableRowFields on DocumentPacketRule {
  id
  businessUnitId
  organizationId
  resourceType
  documentTypeId
  required
  allowMultiple
  displayOrder
  expirationRequired
  expirationWarningDays
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:4ca290e5de0116cb3c75bdb1c763213bbce015c05fcd9a4ed64a072c73cedc67"}) as unknown as TypedDocumentString<DocumentPacketRuleTableQuery, DocumentPacketRuleTableQueryVariables>;
export const DocumentTypeTableDocument = new TypedDocumentString(`
    query DocumentTypeTable($input: DataTableConnectionInput!) {
  documentTypes(input: $input) {
    edges {
      node {
        ...DocumentTypeTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DocumentTypeTableRowFields on DocumentType {
  id
  businessUnitId
  organizationId
  code
  name
  description
  color
  documentClassification
  documentCategory
  isSystem
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:ae55c419637d48ea77823cabaa483d3c741cc6b0bcc07b86e2bd25f4fce11b82"}) as unknown as TypedDocumentString<DocumentTypeTableQuery, DocumentTypeTableQueryVariables>;
export const WorkerPortalStatusDocument = new TypedDocumentString(`
    query WorkerPortalStatus($workerId: ID!) {
  workerPortalStatus(workerId: $workerId) {
    linked
    portalUser {
      id
      name
      emailAddress
      status
      lastLoginAt
    }
    pendingInvitation {
      id
      email
      status
      expiresAt
      createdAt
    }
    invitations {
      id
      email
      status
      expiresAt
      acceptedAt
      createdAt
      invitedBy {
        id
        name
      }
    }
  }
}
    `, {"hash":"sha256:747e063d2865387a9bfee8d33322637bfb155421a1605b8b7bee154804b3c4a1"}) as unknown as TypedDocumentString<WorkerPortalStatusQuery, WorkerPortalStatusQueryVariables>;
export const InviteWorkerToPortalDocument = new TypedDocumentString(`
    mutation InviteWorkerToPortal($input: InviteWorkerToPortalInput!) {
  inviteWorkerToPortal(input: $input) {
    invitation {
      id
      email
      status
      expiresAt
    }
    inviteUrl
    emailSent
  }
}
    `, {"hash":"sha256:aca1f784efb7ea2322d18e6ef69d1981810ffac214ea568ae44b56a0cf721a48"}) as unknown as TypedDocumentString<InviteWorkerToPortalMutation, InviteWorkerToPortalMutationVariables>;
export const RevokeWorkerPortalAccessDocument = new TypedDocumentString(`
    mutation RevokeWorkerPortalAccess($workerId: ID!) {
  revokeWorkerPortalAccess(workerId: $workerId)
}
    `, {"hash":"sha256:f155b1032fe0e180152934a5fb457e75109136e2ffbde362d474828e42dabd7b"}) as unknown as TypedDocumentString<RevokeWorkerPortalAccessMutation, RevokeWorkerPortalAccessMutationVariables>;
export const SettlementDisputeTableDocument = new TypedDocumentString(`
    query SettlementDisputeTable($input: DataTableConnectionInput!) {
  settlementDisputes(input: $input) {
    edges {
      node {
        id
        settlementId
        settlementLineId
        workerId
        status
        category
        description
        resolutionNote
        resolvedAt
        createdAt
        updatedAt
        version
        worker {
          id
          firstName
          lastName
        }
        settlement {
          id
          settlementNumber
          netPayMinor
          currencyCode
          status
        }
        resolvedBy {
          id
          name
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:33d72f67dfa456db1b8de861e84b3e16173a8ba5f5fea3cf7a71bf3f9ddaef93"}) as unknown as TypedDocumentString<SettlementDisputeTableQuery, SettlementDisputeTableQueryVariables>;
export const SettlementDisputeDetailDocument = new TypedDocumentString(`
    query SettlementDisputeDetail($id: ID!) {
  settlementDispute(id: $id) {
    id
    settlementId
    settlementLineId
    workerId
    status
    category
    description
    submittedByUserId
    resolutionNote
    resolutionLineId
    resolvedById
    resolvedAt
    version
    createdAt
    updatedAt
    worker {
      id
      firstName
      lastName
    }
    settlement {
      id
      settlementNumber
      status
      periodStart
      periodEnd
      netPayMinor
      grossEarningsMinor
      deductionsMinor
      currencyCode
    }
    settlementLine {
      id
      lineNumber
      category
      description
      amountMinor
      proNumber
    }
    resolvedBy {
      id
      name
    }
  }
}
    `, {"hash":"sha256:ccd8fbb15a3ec77478291e05d6dd931d3c71923ffea585cd51b02ecc3964cd26"}) as unknown as TypedDocumentString<SettlementDisputeDetailQuery, SettlementDisputeDetailQueryVariables>;
export const OpenSettlementDisputeCountDocument = new TypedDocumentString(`
    query OpenSettlementDisputeCount {
  openSettlementDisputeCount
}
    `, {"hash":"sha256:d8f8fb203d4850d7671c64c9b442e243f87771ccdb245289725411c8fc1bbbbe"}) as unknown as TypedDocumentString<OpenSettlementDisputeCountQuery, OpenSettlementDisputeCountQueryVariables>;
export const StartSettlementDisputeReviewDocument = new TypedDocumentString(`
    mutation StartSettlementDisputeReview($id: ID!) {
  startSettlementDisputeReview(id: $id) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:d9763f9364a8c8ffb2df5481c6da57a9824953a6f9e7fcca26cde2d132c12028"}) as unknown as TypedDocumentString<StartSettlementDisputeReviewMutation, StartSettlementDisputeReviewMutationVariables>;
export const ResolveSettlementDisputeDocument = new TypedDocumentString(`
    mutation ResolveSettlementDispute($input: ResolveSettlementDisputeInput!) {
  resolveSettlementDispute(input: $input) {
    id
    status
    resolutionNote
    resolutionLineId
    resolvedAt
    version
  }
}
    `, {"hash":"sha256:0a9ff3966e92976c95b3815f2820954520d283fd44748dceeb18629e05758401"}) as unknown as TypedDocumentString<ResolveSettlementDisputeMutation, ResolveSettlementDisputeMutationVariables>;
export const MyPortalProfileDocument = new TypedDocumentString(`
    query MyPortalProfile {
  myPortalProfile {
    workerId
    firstName
    lastName
    email
    phoneNumber
    workerType
    driverType
    fleetCodeName
    organizationName
  }
}
    `, {"hash":"sha256:d9f1923f100755639ac4dcb381b4d12d4ec966896dc62f6feeace051bbb313f8"}) as unknown as TypedDocumentString<MyPortalProfileQuery, MyPortalProfileQueryVariables>;
export const MyLoadsDocument = new TypedDocumentString(`
    query MyLoads($scope: PortalLoadScope!, $limit: Int) {
  myLoads(scope: $scope, limit: $limit) {
    assignmentId
    moveId
    shipmentId
    proNumber
    bol
    status
    isPrimary
    tractorCode
    trailerCode
    pieces
    weight
    distanceMiles
    payGrossMinor
    payStatus
    payOnHold
    ackStatus
    stops {
      id
      type
      status
      sequence
      locationName
      addressLine
      scheduledWindowStart
      scheduledWindowEnd
      actualArrival
      actualDeparture
    }
  }
}
    `, {"hash":"sha256:234ab5affc2ec541d377704704289944b19ac2f12553fdb404bce4323e11ba7c"}) as unknown as TypedDocumentString<MyLoadsQuery, MyLoadsQueryVariables>;
export const MyLoadCommentsDocument = new TypedDocumentString(`
    query MyLoadComments($shipmentId: ID!) {
  myLoadComments(shipmentId: $shipmentId) {
    id
    type
    priority
    comment
    authorName
    createdAt
  }
}
    `, {"hash":"sha256:bb8f38084baf00216f6da811c1d01e04b2dfdcf4ac579b48771e2c0103e3e490"}) as unknown as TypedDocumentString<MyLoadCommentsQuery, MyLoadCommentsQueryVariables>;
export const RecordMyStopActionDocument = new TypedDocumentString(`
    mutation RecordMyStopAction($input: RecordMyStopActionInput!) {
  recordMyStopAction(input: $input)
}
    `, {"hash":"sha256:2e9bf84e0ce7dbfd352cba8d60db65f4912a71052aff8f90c7ef530277a91e24"}) as unknown as TypedDocumentString<RecordMyStopActionMutation, RecordMyStopActionMutationVariables>;
export const CreateMyLoadCommentDocument = new TypedDocumentString(`
    mutation CreateMyLoadComment($input: CreateMyLoadCommentInput!) {
  createMyLoadComment(input: $input) {
    id
    type
    priority
    comment
    authorName
    createdAt
  }
}
    `, {"hash":"sha256:d1ae76e14b0a4562299fb803f6445db0ba915c29e9de6be29855c42edf501d17"}) as unknown as TypedDocumentString<CreateMyLoadCommentMutation, CreateMyLoadCommentMutationVariables>;
export const MyPeriodSummaryDocument = new TypedDocumentString(`
    query MyPeriodSummary {
  myPeriodSummary {
    periodStart
    periodEnd
    payDate
    accruedGrossMinor
    eventCount
  }
}
    `, {"hash":"sha256:300c67ba34b8708f1e1352f6bf455729f24ad3f18ba8f7a97d2886d2ba12e382"}) as unknown as TypedDocumentString<MyPeriodSummaryQuery, MyPeriodSummaryQueryVariables>;
export const MyRecentPayEventsDocument = new TypedDocumentString(`
    query MyRecentPayEvents($limit: Int) {
  myRecentPayEvents(limit: $limit) {
    id
    status
    eventDate
    proNumber
    grossAmountMinor
    totalMiles
    currencyCode
    onHold
    holdReason
  }
}
    `, {"hash":"sha256:b21ae5d25eaa70e20606dd2bc6decf9b6be568727a1fd3069ffcd9e43a2ffc1b"}) as unknown as TypedDocumentString<MyRecentPayEventsQuery, MyRecentPayEventsQueryVariables>;
export const MySettlementsDocument = new TypedDocumentString(`
    query MySettlements($limit: Int, $offset: Int) {
  mySettlements(limit: $limit, offset: $offset) {
    items {
      id
      settlementNumber
      status
      periodStart
      periodEnd
      payDate
      grossEarningsMinor
      reimbursementsMinor
      deductionsMinor
      netPayMinor
      currencyCode
      paidAt
      paymentMethod
      paymentReference
    }
    total
  }
}
    `, {"hash":"sha256:2ee91e8c25e4027169a740ef45aaf59ebab0fdba2f9fe2c1616d4a5f6a5bb491"}) as unknown as TypedDocumentString<MySettlementsQuery, MySettlementsQueryVariables>;
export const MySettlementDocument = new TypedDocumentString(`
    query MySettlement($id: ID!) {
  mySettlement(id: $id) {
    id
    settlementNumber
    status
    classification
    payProfileName
    periodStart
    periodEnd
    payDate
    grossEarningsMinor
    reimbursementsMinor
    deductionsMinor
    carryForwardInMinor
    carryForwardOutMinor
    netPayMinor
    totalMiles
    shipmentCount
    currencyCode
    paidAt
    paymentMethod
    paymentReference
    createdAt
    lines {
      id
      lineNumber
      category
      componentKind
      method
      description
      quantity
      rate
      amountMinor
      proNumber
    }
  }
}
    `, {"hash":"sha256:13d06f21c46b4eeeb624270d875cdccc3574ce029799208b15d05c3cc38e499d"}) as unknown as TypedDocumentString<MySettlementQuery, MySettlementQueryVariables>;
export const MyEscrowDocument = new TypedDocumentString(`
    query MyEscrow {
  myEscrow {
    account {
      id
      status
      targetAmountMinor
      balanceMinor
      currencyCode
      createdAt
    }
    transactions {
      id
      type
      amountMinor
      balanceAfterMinor
      description
      occurredDate
      createdAt
    }
  }
}
    `, {"hash":"sha256:d6e92e5dd91225da94c876899ec76b6539d6b200adc66d8f5c9f16743631c4e7"}) as unknown as TypedDocumentString<MyEscrowQuery, MyEscrowQueryVariables>;
export const MyAdvancesDocument = new TypedDocumentString(`
    query MyAdvances {
  myAdvances {
    id
    status
    source
    reference
    amountMinor
    recoveredMinor
    outstandingMinor
    currencyCode
    issuedDate
  }
}
    `, {"hash":"sha256:4ad05d0be74ab6dec588aa0b7db8ab77d9cd5d605096e13a97cb20bce852f47f"}) as unknown as TypedDocumentString<MyAdvancesQuery, MyAdvancesQueryVariables>;
export const MyDisputesDocument = new TypedDocumentString(`
    query MyDisputes {
  myDisputes {
    id
    settlementId
    settlementLineId
    status
    category
    description
    resolutionNote
    resolvedAt
    createdAt
    settlement {
      id
      settlementNumber
      periodStart
      periodEnd
    }
    settlementLine {
      id
      description
      amountMinor
      category
    }
  }
}
    `, {"hash":"sha256:ce60a2cceb081d891881972f30a664ff30829c3bee0d0ea86b2d4bb9214d78f6"}) as unknown as TypedDocumentString<MyDisputesQuery, MyDisputesQueryVariables>;
export const CreateSettlementDisputeDocument = new TypedDocumentString(`
    mutation CreateSettlementDispute($input: CreateSettlementDisputeInput!) {
  createSettlementDispute(input: $input) {
    id
    status
    category
    description
    createdAt
  }
}
    `, {"hash":"sha256:369c2c8a4105f0c1c8e19d5e899d99997ba5449ea24316561c28e415cc008123"}) as unknown as TypedDocumentString<CreateSettlementDisputeMutation, CreateSettlementDisputeMutationVariables>;
export const WithdrawSettlementDisputeDocument = new TypedDocumentString(`
    mutation WithdrawSettlementDispute($id: ID!) {
  withdrawSettlementDispute(id: $id) {
    id
    status
  }
}
    `, {"hash":"sha256:8ac805417a3dde06fdd223772563f68688bdb73966c0424fcfc441df9db0c205"}) as unknown as TypedDocumentString<WithdrawSettlementDisputeMutation, WithdrawSettlementDisputeMutationVariables>;
export const MyComplianceProfileDocument = new TypedDocumentString(`
    query MyComplianceProfile {
  myComplianceProfile {
    workerId
    licenseNumber
    licenseState
    cdlClass
    endorsement
    licenseExpiry
    hazmatExpiry
    medicalCardExpiry
    physicalDueDate
    mvrDueDate
    twicExpiry
    complianceStatus
    isQualified
    hireDate
    addressLine1
    addressLine2
    city
    stateAbbreviation
    postalCode
    phoneNumber
    emergencyContactName
    emergencyContactPhone
  }
}
    `, {"hash":"sha256:ac5af8e8d15be26168448355e9771b11417890d634039dbdf7f39317d739b532"}) as unknown as TypedDocumentString<MyComplianceProfileQuery, MyComplianceProfileQueryVariables>;
export const UpdateMyContactInfoDocument = new TypedDocumentString(`
    mutation UpdateMyContactInfo($input: UpdateMyContactInfoInput!) {
  updateMyContactInfo(input: $input) {
    workerId
    addressLine1
    addressLine2
    city
    stateAbbreviation
    postalCode
    phoneNumber
    emergencyContactName
    emergencyContactPhone
  }
}
    `, {"hash":"sha256:e81e7bb10b0ba505ac01858582f241192fb8f9552a6ea9ef910afba79ff73f1d"}) as unknown as TypedDocumentString<UpdateMyContactInfoMutation, UpdateMyContactInfoMutationVariables>;
export const MyPtoDocument = new TypedDocumentString(`
    query MyPto {
  myPto {
    id
    status
    type
    startDate
    endDate
    reason
    createdAt
  }
}
    `, {"hash":"sha256:9fcfb3c6f94692c333a8f9b3f6074f033ace1ec340b3c7ffc601c00618a7e264"}) as unknown as TypedDocumentString<MyPtoQuery, MyPtoQueryVariables>;
export const RequestMyPtoDocument = new TypedDocumentString(`
    mutation RequestMyPto($input: RequestMyPtoInput!) {
  requestMyPto(input: $input) {
    id
    status
    type
    startDate
    endDate
    reason
    createdAt
  }
}
    `, {"hash":"sha256:b1ad80c4985bc9a0b66fe33cb36e37d957a107db4c25fe06a4154b0e76ed7be6"}) as unknown as TypedDocumentString<RequestMyPtoMutation, RequestMyPtoMutationVariables>;
export const CancelMyPtoDocument = new TypedDocumentString(`
    mutation CancelMyPto($id: ID!) {
  cancelMyPto(id: $id) {
    id
    status
  }
}
    `, {"hash":"sha256:f6bff19ede90794d5d921c0277bb7254497e80ce7fb32e10870af9dffc87759c"}) as unknown as TypedDocumentString<CancelMyPtoMutation, CancelMyPtoMutationVariables>;
export const MyExpensesDocument = new TypedDocumentString(`
    query MyExpenses {
  myExpenses {
    id
    shipmentId
    payCodeId
    status
    amountMinor
    currencyCode
    description
    incurredDate
    receiptDocumentId
    reviewNote
    reviewedAt
    createdAt
    payCode {
      id
      code
      description
    }
  }
}
    `, {"hash":"sha256:a37af7272e3a8791c6ea1257e9786bfdb352b5c0c58e29ff8cb01fb72de1145f"}) as unknown as TypedDocumentString<MyExpensesQuery, MyExpensesQueryVariables>;
export const SubmitMyExpenseDocument = new TypedDocumentString(`
    mutation SubmitMyExpense($input: SubmitMyExpenseInput!) {
  submitMyExpense(input: $input) {
    id
    status
    amountMinor
    description
    incurredDate
    createdAt
  }
}
    `, {"hash":"sha256:a7598d8fcd7abe6245cd2dfa7858051fc11048da1632d7f04fbf0914e4dc5ebc"}) as unknown as TypedDocumentString<SubmitMyExpenseMutation, SubmitMyExpenseMutationVariables>;
export const CancelMyExpenseDocument = new TypedDocumentString(`
    mutation CancelMyExpense($id: ID!) {
  cancelMyExpense(id: $id) {
    id
    status
  }
}
    `, {"hash":"sha256:ff0f019a13713002ba005dabcfbaaf348ab036dbb4461aa7f8ad6393d208cb3a"}) as unknown as TypedDocumentString<CancelMyExpenseMutation, CancelMyExpenseMutationVariables>;
export const RespondToMyAssignmentDocument = new TypedDocumentString(`
    mutation RespondToMyAssignment($input: RespondToMyAssignmentInput!) {
  respondToMyAssignment(input: $input)
}
    `, {"hash":"sha256:211393bb6c83113d7199803c0c269021ebfac1ecf2a65eb46143051c7e3fb3f2"}) as unknown as TypedDocumentString<RespondToMyAssignmentMutation, RespondToMyAssignmentMutationVariables>;
export const MyLoadPayEstimateDocument = new TypedDocumentString(`
    query MyLoadPayEstimate($shipmentId: ID!, $moveId: ID!) {
  myLoadPayEstimate(shipmentId: $shipmentId, moveId: $moveId) {
    grossMinor
    currencyCode
  }
}
    `, {"hash":"sha256:153135d927a73aaa32b52850ee67d3f02242447d08c58e4f0058d052f85f32b2"}) as unknown as TypedDocumentString<MyLoadPayEstimateQuery, MyLoadPayEstimateQueryVariables>;
export const MyYtdPayDocument = new TypedDocumentString(`
    query MyYtdPay($year: Int!) {
  myYtdPay(year: $year) {
    workerId
    year
    settlementCount
    grossEarningsMinor
    reimbursementsMinor
    deductionsMinor
    netPayMinor
  }
}
    `, {"hash":"sha256:3020bc2943b245e09336415b3901f2c59161751172c3a12e1b721972bb8ed133"}) as unknown as TypedDocumentString<MyYtdPayQuery, MyYtdPayQueryVariables>;
export const DriverExpenseTableDocument = new TypedDocumentString(`
    query DriverExpenseTable($input: DataTableConnectionInput!) {
  driverExpenses(input: $input) {
    edges {
      node {
        id
        workerId
        shipmentId
        status
        amountMinor
        currencyCode
        description
        incurredDate
        receiptDocumentId
        reviewNote
        reviewedAt
        settlementLineId
        createdAt
        version
        worker {
          id
          firstName
          lastName
        }
        payCode {
          id
          code
          description
        }
        reviewedBy {
          id
          name
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:871d4b9aaf068f47c56778c4af2cb6063c875805bc156d90119fca05a718da58"}) as unknown as TypedDocumentString<DriverExpenseTableQuery, DriverExpenseTableQueryVariables>;
export const DriverExpenseDetailDocument = new TypedDocumentString(`
    query DriverExpenseDetail($id: ID!) {
  driverExpense(id: $id) {
    id
    workerId
    shipmentId
    payCodeId
    status
    amountMinor
    currencyCode
    description
    incurredDate
    receiptDocumentId
    reviewNote
    reviewedById
    reviewedAt
    settlementLineId
    version
    createdAt
    updatedAt
    worker {
      id
      firstName
      lastName
      email
      phoneNumber
    }
    payCode {
      id
      code
      description
    }
    reviewedBy {
      id
      name
    }
  }
}
    `, {"hash":"sha256:b6dfe3c16c23e841da42029ab4276a328018d85777371bfc36808ae4868cee2d"}) as unknown as TypedDocumentString<DriverExpenseDetailQuery, DriverExpenseDetailQueryVariables>;
export const PendingDriverExpenseCountDocument = new TypedDocumentString(`
    query PendingDriverExpenseCount {
  pendingDriverExpenseCount
}
    `, {"hash":"sha256:ea576d13a22021a5d5e9559b60c5a2082502418662f1ab9e11591b1a0af0e4de"}) as unknown as TypedDocumentString<PendingDriverExpenseCountQuery, PendingDriverExpenseCountQueryVariables>;
export const ReviewDriverExpenseDocument = new TypedDocumentString(`
    mutation ReviewDriverExpense($input: ReviewDriverExpenseInput!) {
  reviewDriverExpense(input: $input) {
    id
    status
    reviewNote
    reviewedAt
    settlementLineId
    version
  }
}
    `, {"hash":"sha256:a7977fe40219d006a216e85a7db70e407be08880a8b5b0fd602d53c4603f6bef"}) as unknown as TypedDocumentString<ReviewDriverExpenseMutation, ReviewDriverExpenseMutationVariables>;
export const DashControlDocument = new TypedDocumentString(`
    query DashControl {
  dashControl {
    id
    requireLoadAcknowledgment
    allowLoadRefusals
    allowStopActions
    allowLoadDocumentUpload
    allowLoadComments
    showLoadPay
    showPayEstimates
    allowExpenseSubmission
    requireExpenseReceipt
    allowSettlementDisputes
    allowProfileDocumentUpload
    allowContactInfoEdit
    allowPtoRequests
    sendCredentialReminders
    enableDetentionAlerts
    detentionAlertThresholdMinutes
    version
  }
}
    `, {"hash":"sha256:a7fa2c52c009be34795526a62d7bd524f6238be8da67be763802bcbaad6ac77a"}) as unknown as TypedDocumentString<DashControlQuery, DashControlQueryVariables>;
export const UpdateDashControlDocument = new TypedDocumentString(`
    mutation UpdateDashControl($input: UpdateDashControlInput!) {
  updateDashControl(input: $input) {
    id
    requireLoadAcknowledgment
    allowLoadRefusals
    allowStopActions
    allowLoadDocumentUpload
    allowLoadComments
    showLoadPay
    showPayEstimates
    allowExpenseSubmission
    requireExpenseReceipt
    allowSettlementDisputes
    allowProfileDocumentUpload
    allowContactInfoEdit
    allowPtoRequests
    sendCredentialReminders
    enableDetentionAlerts
    detentionAlertThresholdMinutes
    version
  }
}
    `, {"hash":"sha256:6240832c7e2db613007b7c3e093f36d779100155ba80adc5a2df5c2c6e6c4e74"}) as unknown as TypedDocumentString<UpdateDashControlMutation, UpdateDashControlMutationVariables>;
export const MyPortalFeaturesDocument = new TypedDocumentString(`
    query MyPortalFeatures {
  myPortalFeatures {
    requireLoadAcknowledgment
    allowLoadRefusals
    allowStopActions
    allowLoadDocumentUpload
    allowLoadComments
    showLoadPay
    showPayEstimates
    allowExpenseSubmission
    requireExpenseReceipt
    allowSettlementDisputes
    allowProfileDocumentUpload
    allowContactInfoEdit
    allowPtoRequests
  }
}
    `, {"hash":"sha256:dce4827f759a822d4b074209578feb2d3f12294a2fb26e9d0184fd89c8183571"}) as unknown as TypedDocumentString<MyPortalFeaturesQuery, MyPortalFeaturesQueryVariables>;
export const PayProfileTableDocument = new TypedDocumentString(`
    query PayProfileTable($input: DataTableConnectionInput!) {
  payProfiles(input: $input) {
    edges {
      node {
        id
        organizationId
        businessUnitId
        status
        name
        description
        classification
        currencyCode
        guaranteedPeriodMinimumMinor
        perDiemRatePerMile
        perDiemDailyCapMinor
        version
        createdAt
        updatedAt
        activeAssignmentCount
        components {
          id
          kind
          method
          description
          rate
          revenueBasis
          bands {
            minMiles
            maxMiles
            rate
          }
          freeTimeMinutes
          minAmountMinor
          maxAmountMinor
          sequence
          isActive
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:1e5c03f4818797042043e0fb555d6554e4651086b3312226c4981f680c5995fe"}) as unknown as TypedDocumentString<PayProfileTableQuery, PayProfileTableQueryVariables>;
export const PayProfileOptionsDocument = new TypedDocumentString(`
    query PayProfileOptions($input: DataTableConnectionInput!) {
  payProfiles(input: $input) {
    edges {
      node {
        id
        name
        classification
        status
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:0f1fc94a1c413422c5dea0282e0cc1a022769625469f09634c7ad5051cf29790"}) as unknown as TypedDocumentString<PayProfileOptionsQuery, PayProfileOptionsQueryVariables>;
export const WorkerPayAssignmentsDocument = new TypedDocumentString(`
    query WorkerPayAssignments($workerId: ID!) {
  workerPayAssignments(workerId: $workerId) {
    id
    workerId
    payProfileId
    effectiveFrom
    effectiveTo
    splitPercent
    rateOverrides {
      componentId
      rate
    }
    notes
    version
    createdAt
    payProfile {
      id
      name
      classification
      components {
        id
        kind
        method
        description
        rate
      }
    }
  }
}
    `, {"hash":"sha256:1bf635cd5fa402d1768672762dcc0cca45e44bfd28abb08b7c7caa1a7efc6528"}) as unknown as TypedDocumentString<WorkerPayAssignmentsQuery, WorkerPayAssignmentsQueryVariables>;
export const EffectiveWorkerPayAssignmentDocument = new TypedDocumentString(`
    query EffectiveWorkerPayAssignment($workerId: ID!) {
  effectiveWorkerPayAssignment(workerId: $workerId) {
    id
    workerId
    payProfileId
    effectiveFrom
    effectiveTo
    splitPercent
    rateOverrides {
      componentId
      rate
    }
    notes
    payProfile {
      id
      name
      classification
      guaranteedPeriodMinimumMinor
      components {
        id
        kind
        method
        description
        rate
        revenueBasis
        bands {
          minMiles
          maxMiles
          rate
        }
        isActive
      }
    }
  }
}
    `, {"hash":"sha256:2249cf1dac4bf029650c70b40a3b51522bdfb085d8d4b0783a8e58b649b5524b"}) as unknown as TypedDocumentString<EffectiveWorkerPayAssignmentQuery, EffectiveWorkerPayAssignmentQueryVariables>;
export const PayProfileAssignmentsDocument = new TypedDocumentString(`
    query PayProfileAssignments($payProfileId: ID!) {
  payProfileAssignments(payProfileId: $payProfileId) {
    id
    workerId
    effectiveFrom
    effectiveTo
    splitPercent
    rateOverrides {
      componentId
      rate
    }
    worker {
      id
      firstName
      lastName
    }
  }
}
    `, {"hash":"sha256:a5cd25fafd7706fafd7a770aec517285d5ed7b8ab090cbd5756fb66f7e528221"}) as unknown as TypedDocumentString<PayProfileAssignmentsQuery, PayProfileAssignmentsQueryVariables>;
export const PayProfileDetailDocument = new TypedDocumentString(`
    query PayProfileDetail($id: ID!) {
  payProfile(id: $id) {
    id
    name
    classification
    currencyCode
    components {
      id
      kind
      method
      description
      rate
      revenueBasis
      isActive
    }
  }
}
    `, {"hash":"sha256:7c7934aa1908ddd5d2d39e488200ef13338e7cdbd5eb59f8a8489cb7a835be3b"}) as unknown as TypedDocumentString<PayProfileDetailQuery, PayProfileDetailQueryVariables>;
export const RecurringDeductionTableDocument = new TypedDocumentString(`
    query RecurringDeductionTable($input: DataTableConnectionInput!) {
  recurringDeductions(input: $input) {
    edges {
      node {
        id
        workerId
        payCodeId
        escrowAccountId
        status
        frequency
        description
        amountMinor
        totalCapMinor
        deductedToDateMinor
        startDate
        endDate
        currencyCode
        version
        createdAt
        updatedAt
        worker {
          id
          firstName
          lastName
        }
        payCode {
          id
          code
          name
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:b08b1dd23e5a56af87307e351290b4c8e611b257939e1e5fe3eadc77d6331cb8"}) as unknown as TypedDocumentString<RecurringDeductionTableQuery, RecurringDeductionTableQueryVariables>;
export const RecurringEarningTableDocument = new TypedDocumentString(`
    query RecurringEarningTable($input: DataTableConnectionInput!) {
  recurringEarnings(input: $input) {
    edges {
      node {
        id
        workerId
        payCodeId
        status
        frequency
        description
        amountMinor
        totalCapMinor
        paidToDateMinor
        startDate
        endDate
        currencyCode
        version
        createdAt
        updatedAt
        worker {
          id
          firstName
          lastName
        }
        payCode {
          id
          code
          name
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:fedf545334de1148057d2bde0e8a58f94e3c6b9ad09a4a29bcd17c1cb68a2272"}) as unknown as TypedDocumentString<RecurringEarningTableQuery, RecurringEarningTableQueryVariables>;
export const PayCodeTableDocument = new TypedDocumentString(`
    query PayCodeTable($input: DataTableConnectionInput!) {
  payCodes(input: $input) {
    edges {
      node {
        id
        status
        direction
        code
        name
        description
        taxable
        countsTowardGuarantee
        glAccountId
        defaultAmountMinor
        isSystem
        version
        createdAt
        updatedAt
        glAccount {
          id
          accountCode
          name
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:1cd027118971feb07a21cdf553bb50a716510e6882471adf6031858296275211"}) as unknown as TypedDocumentString<PayCodeTableQuery, PayCodeTableQueryVariables>;
export const PayCodeOptionsDocument = new TypedDocumentString(`
    query PayCodeOptions($direction: PayCodeDirection) {
  payCodeOptions(direction: $direction) {
    id
    direction
    code
    name
    taxable
    defaultAmountMinor
  }
}
    `, {"hash":"sha256:936a60ce21a22bf3cb1ec09b8e12a1d12b6cffb98550835e69d0255f713dfd72"}) as unknown as TypedDocumentString<PayCodeOptionsQuery, PayCodeOptionsQueryVariables>;
export const PayAdvanceTableDocument = new TypedDocumentString(`
    query PayAdvanceTable($input: DataTableConnectionInput!) {
  payAdvances(input: $input) {
    edges {
      node {
        id
        workerId
        status
        source
        reference
        issuedDate
        amountMinor
        recoveredMinor
        writtenOffMinor
        outstandingMinor
        writeOffReason
        notes
        currencyCode
        version
        createdAt
        updatedAt
        worker {
          id
          firstName
          lastName
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:72e0fa0b3fc41df7a92208a1575f5a3242fa80666502710659439dc96338d8dd"}) as unknown as TypedDocumentString<PayAdvanceTableQuery, PayAdvanceTableQueryVariables>;
export const EscrowAccountTableDocument = new TypedDocumentString(`
    query EscrowAccountTable($input: DataTableConnectionInput!) {
  escrowAccounts(input: $input) {
    edges {
      node {
        id
        workerId
        status
        targetAmountMinor
        balanceMinor
        annualInterestRate
        lastInterestAccrualDate
        openedDate
        closedDate
        currencyCode
        version
        createdAt
        updatedAt
        worker {
          id
          firstName
          lastName
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:1c0df4bc500406a888b423333eb4e66c48185efcc23a8d73fc55a3171d3a8a63"}) as unknown as TypedDocumentString<EscrowAccountTableQuery, EscrowAccountTableQueryVariables>;
export const EscrowAccountDetailDocument = new TypedDocumentString(`
    query EscrowAccountDetail($id: ID!) {
  escrowAccount(id: $id) {
    id
    workerId
    status
    targetAmountMinor
    balanceMinor
    annualInterestRate
    lastInterestAccrualDate
    openedDate
    closedDate
    currencyCode
    version
    worker {
      id
      firstName
      lastName
    }
    transactions {
      id
      type
      amountMinor
      balanceAfterMinor
      occurredDate
      description
      settlementId
      createdAt
    }
  }
}
    `, {"hash":"sha256:57e8a0b5d1c97b85fcb0dc2aa0b40ffaec3b9fbfaf20e3d00e4983912db939b9"}) as unknown as TypedDocumentString<EscrowAccountDetailQuery, EscrowAccountDetailQueryVariables>;
export const DriverSettlementTableDocument = new TypedDocumentString(`
    query DriverSettlementTable($input: DataTableConnectionInput!) {
  driverSettlements(input: $input) {
    edges {
      node {
        id
        workerId
        batchId
        settlementNumber
        status
        classification
        payProfileName
        periodStart
        periodEnd
        payDate
        grossEarningsMinor
        reimbursementsMinor
        deductionsMinor
        carryForwardInMinor
        carryForwardOutMinor
        netPayMinor
        totalMiles
        shipmentCount
        currencyCode
        hasExceptions
        version
        createdAt
        updatedAt
        worker {
          id
          firstName
          lastName
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:a5b74fc45a1800bf202a35cef1731d1c2edb39f201dadc92d0171276cadeee6c"}) as unknown as TypedDocumentString<DriverSettlementTableQuery, DriverSettlementTableQueryVariables>;
export const DriverSettlementDetailDocument = new TypedDocumentString(`
    query DriverSettlementDetail($id: ID!) {
  driverSettlement(id: $id) {
    id
    workerId
    batchId
    payProfileId
    settlementNumber
    status
    classification
    payProfileName
    periodStart
    periodEnd
    payDate
    grossEarningsMinor
    reimbursementsMinor
    deductionsMinor
    carryForwardInMinor
    carryForwardOutMinor
    netPayMinor
    totalMiles
    shipmentCount
    currencyCode
    hasExceptions
    exceptions {
      code
      severity
      message
    }
    notes
    submittedById
    submittedAt
    approvedById
    approvedAt
    postedById
    postedAt
    paidAt
    paymentMethod
    paymentReference
    voidedById
    voidedAt
    voidReason
    version
    createdAt
    updatedAt
    worker {
      id
      firstName
      lastName
    }
    lines {
      id
      lineNumber
      category
      componentKind
      method
      description
      quantity
      rate
      amountMinor
      shipmentId
      moveId
      payEventId
      recurringDeductionId
      advanceId
      escrowAccountId
      proNumber
    }
  }
}
    `, {"hash":"sha256:0f96d6515e17c15b83b8b0e56cf436a5626dd09d6bbb87efbcfbdbddcf0bf68b"}) as unknown as TypedDocumentString<DriverSettlementDetailQuery, DriverSettlementDetailQueryVariables>;
export const SettlementBatchTableDocument = new TypedDocumentString(`
    query SettlementBatchTable($input: DataTableConnectionInput!) {
  settlementBatches(input: $input) {
    edges {
      node {
        id
        status
        name
        periodStart
        periodEnd
        payDate
        settlementCount
        exceptionCount
        totalGrossMinor
        totalNetMinor
        currencyCode
        notes
        generatedById
        generatedAt
        completedAt
        canceledAt
        version
        createdAt
        updatedAt
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:dba0d6a1fc477db6f5c73a71fda39edbabffa3e00b3be0d7acce5e330ee540eb"}) as unknown as TypedDocumentString<SettlementBatchTableQuery, SettlementBatchTableQueryVariables>;
export const DriverPayEventTableDocument = new TypedDocumentString(`
    query DriverPayEventTable($input: DataTableConnectionInput!) {
  driverPayEvents(input: $input) {
    edges {
      node {
        id
        workerId
        shipmentId
        moveId
        settlementId
        status
        eventDate
        grossAmountMinor
        totalMiles
        currencyCode
        proNumber
        onHold
        holdReason
        voidedAt
        voidReason
        version
        createdAt
        updatedAt
        components {
          kind
          method
          description
          quantity
          rate
          amountMinor
        }
        worker {
          id
          firstName
          lastName
        }
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:45a28f87675baa79d6a95a77ba6f7f23c9aac18453cb78d5ac2586cc79bc30fa"}) as unknown as TypedDocumentString<DriverPayEventTableQuery, DriverPayEventTableQueryVariables>;
export const WorkerEarningsSummaryDocument = new TypedDocumentString(`
    query WorkerEarningsSummary($workerId: ID!) {
  workerEarningsSummary(workerId: $workerId) {
    workerId
    accruedEventCount
    accruedGrossMinor
    outstandingAdvances
    escrowBalanceMinor
  }
}
    `, {"hash":"sha256:604360bedc59b36d762fa6780dbfe073bfbfeb71015006a8a74717663ea17f9a"}) as unknown as TypedDocumentString<WorkerEarningsSummaryQuery, WorkerEarningsSummaryQueryVariables>;
export const WorkerYtdPaySummariesDocument = new TypedDocumentString(`
    query WorkerYtdPaySummaries($year: Int!, $classification: PayeeClassification) {
  workerYtdPaySummaries(year: $year, classification: $classification) {
    workerId
    workerName
    classification
    year
    settlementCount
    grossEarningsMinor
    reimbursementsMinor
    deductionsMinor
    netPayMinor
  }
}
    `, {"hash":"sha256:f7d96ae4a669343c9c3f27a40e57063c9565295d8c2fa4924e44a2643b5ae67b"}) as unknown as TypedDocumentString<WorkerYtdPaySummariesQuery, WorkerYtdPaySummariesQueryVariables>;
export const SettlementControlDocument = new TypedDocumentString(`
    query SettlementControl {
  settlementControl {
    id
    organizationId
    businessUnitId
    payPeriodFrequency
    periodEndDayOfWeek
    payDelayDays
    payTrigger
    autoGenerateBatches
    autoApproveClean
    autoAttachAccruals
    autoPostOnApprove
    allowNegativeNet
    varianceThresholdPct
    varianceLookbackWeeks
    defaultEscrowInterestRate
    escrowInterestFrequencyMonths
    version
  }
}
    `, {"hash":"sha256:000046c1cbdafc864b73f222fca8fad96ed5a4ff21e87d5807bae7f56f070673"}) as unknown as TypedDocumentString<SettlementControlQuery, SettlementControlQueryVariables>;
export const SettlementWorkspaceSummaryDocument = new TypedDocumentString(`
    query SettlementWorkspaceSummary($periodStart: Int, $periodEnd: Int) {
  settlementWorkspaceSummary(periodStart: $periodStart, periodEnd: $periodEnd) {
    periodStart
    periodEnd
    payDate
    draftCount
    pendingApprovalCount
    approvedCount
    postedCount
    paidCount
    exceptionCount
    totalNetMinor
    totalGrossMinor
    unsettledEventCount
    unsettledGrossMinor
    heldEventCount
    heldGrossMinor
    unsettledWorkerCount
    openBatchId
  }
}
    `, {"hash":"sha256:c557e4cfa2aec767c55a4a6eb7a14fd4a506c9d1808bd909a714beee10e5a731"}) as unknown as TypedDocumentString<SettlementWorkspaceSummaryQuery, SettlementWorkspaceSummaryQueryVariables>;
export const UnsettledWorkerSummariesDocument = new TypedDocumentString(`
    query UnsettledWorkerSummaries($periodStart: Int, $periodEnd: Int) {
  unsettledWorkerSummaries(periodStart: $periodStart, periodEnd: $periodEnd) {
    workerId
    workerName
    eventCount
    grossAmountMinor
    heldCount
    heldGrossMinor
    hasSettlement
  }
}
    `, {"hash":"sha256:5f51e5c0fc01343e23452380d8711b2fb47fe90d380317889a987f512d8c023a"}) as unknown as TypedDocumentString<UnsettledWorkerSummariesQuery, UnsettledWorkerSummariesQueryVariables>;
export const CurrentSettlementPeriodDocument = new TypedDocumentString(`
    query CurrentSettlementPeriod {
  currentSettlementPeriod {
    periodStart
    periodEnd
    payDate
  }
}
    `, {"hash":"sha256:18a696ad99213f20822751f7dd47c0e146724a9a15cd4acbf0725a1964074680"}) as unknown as TypedDocumentString<CurrentSettlementPeriodQuery, CurrentSettlementPeriodQueryVariables>;
export const PreviewDriverSettlementDocument = new TypedDocumentString(`
    query PreviewDriverSettlement($workerId: ID!, $periodStart: Int, $periodEnd: Int) {
  previewDriverSettlement(
    workerId: $workerId
    periodStart: $periodStart
    periodEnd: $periodEnd
  ) {
    id
    workerId
    settlementNumber
    status
    classification
    payProfileName
    periodStart
    periodEnd
    payDate
    grossEarningsMinor
    reimbursementsMinor
    deductionsMinor
    carryForwardInMinor
    carryForwardOutMinor
    netPayMinor
    totalMiles
    shipmentCount
    currencyCode
    hasExceptions
    exceptions {
      code
      severity
      message
    }
    lines {
      lineNumber
      category
      componentKind
      method
      description
      quantity
      rate
      amountMinor
      proNumber
    }
  }
}
    `, {"hash":"sha256:c3cea3211e8cdd412f9df5f2ec682195587c30756286ccaba39b69e7f65b7660"}) as unknown as TypedDocumentString<PreviewDriverSettlementQuery, PreviewDriverSettlementQueryVariables>;
export const ExportSettlementBatchCsvDocument = new TypedDocumentString(`
    query ExportSettlementBatchCsv($batchId: ID!) {
  exportSettlementBatchCsv(batchId: $batchId)
}
    `, {"hash":"sha256:09b35d0d3a76868967b6a609edee59aab83a14a47829d873be063ad8bafd1dd4"}) as unknown as TypedDocumentString<ExportSettlementBatchCsvQuery, ExportSettlementBatchCsvQueryVariables>;
export const CreatePayProfileDocument = new TypedDocumentString(`
    mutation CreatePayProfile($input: CreatePayProfileInput!) {
  createPayProfile(input: $input) {
    id
    name
    version
  }
}
    `, {"hash":"sha256:057e95826edc7c7f2e812cc37894744703aee39c278c09bde038098c6dc7e632"}) as unknown as TypedDocumentString<CreatePayProfileMutation, CreatePayProfileMutationVariables>;
export const UpdatePayProfileDocument = new TypedDocumentString(`
    mutation UpdatePayProfile($input: UpdatePayProfileInput!) {
  updatePayProfile(input: $input) {
    id
    name
    version
  }
}
    `, {"hash":"sha256:2aa4747470ca1fb5b86dfdf513276a06a579f785fb0ffc605e38f60feca010c7"}) as unknown as TypedDocumentString<UpdatePayProfileMutation, UpdatePayProfileMutationVariables>;
export const AssignPayProfileToWorkerDocument = new TypedDocumentString(`
    mutation AssignPayProfileToWorker($input: AssignPayProfileInput!) {
  assignPayProfileToWorker(input: $input) {
    id
    workerId
    payProfileId
    effectiveFrom
    effectiveTo
  }
}
    `, {"hash":"sha256:81e72cc2a26f0710b7df1d899b7ff8e97f26b9ab1d5f37d8c99876073c9f5bf0"}) as unknown as TypedDocumentString<AssignPayProfileToWorkerMutation, AssignPayProfileToWorkerMutationVariables>;
export const EndWorkerPayAssignmentDocument = new TypedDocumentString(`
    mutation EndWorkerPayAssignment($input: EndWorkerPayAssignmentInput!) {
  endWorkerPayAssignment(input: $input) {
    id
    effectiveTo
  }
}
    `, {"hash":"sha256:971182611f2f9ad0a2c0ee7a12060d646bce07472d790f6d363790dedb7ae52a"}) as unknown as TypedDocumentString<EndWorkerPayAssignmentMutation, EndWorkerPayAssignmentMutationVariables>;
export const CreateRecurringDeductionDocument = new TypedDocumentString(`
    mutation CreateRecurringDeduction($input: CreateRecurringDeductionInput!) {
  createRecurringDeduction(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:27048f4cb91e9ceb4cc32391c58f7012d07253930cd32f1377de5a33a5eb6c67"}) as unknown as TypedDocumentString<CreateRecurringDeductionMutation, CreateRecurringDeductionMutationVariables>;
export const UpdateRecurringDeductionDocument = new TypedDocumentString(`
    mutation UpdateRecurringDeduction($input: UpdateRecurringDeductionInput!) {
  updateRecurringDeduction(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:60ff79afe87b8907fa14e6470065770f4022043a122db9866b9138d2b42ca15a"}) as unknown as TypedDocumentString<UpdateRecurringDeductionMutation, UpdateRecurringDeductionMutationVariables>;
export const CreatePayCodeDocument = new TypedDocumentString(`
    mutation CreatePayCode($input: CreatePayCodeInput!) {
  createPayCode(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:aa0e0bad8977697bc927a0e8dc931a1e29d5699e278af3e61a2b70feb5ea08ee"}) as unknown as TypedDocumentString<CreatePayCodeMutation, CreatePayCodeMutationVariables>;
export const UpdatePayCodeDocument = new TypedDocumentString(`
    mutation UpdatePayCode($input: UpdatePayCodeInput!) {
  updatePayCode(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:54054095fdcf12e64ef6dd8c31e280536a250d01bca95292636c708c73cbfd4e"}) as unknown as TypedDocumentString<UpdatePayCodeMutation, UpdatePayCodeMutationVariables>;
export const CreateRecurringEarningDocument = new TypedDocumentString(`
    mutation CreateRecurringEarning($input: CreateRecurringEarningInput!) {
  createRecurringEarning(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:f92e8c58da59d89b21355bb1378503a710fccf747afea2987da9b5c11ea5aaab"}) as unknown as TypedDocumentString<CreateRecurringEarningMutation, CreateRecurringEarningMutationVariables>;
export const UpdateRecurringEarningDocument = new TypedDocumentString(`
    mutation UpdateRecurringEarning($input: UpdateRecurringEarningInput!) {
  updateRecurringEarning(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:15d49ce8f2769edc8dc6c562aef844ce80a3c2fb015a337f1a57bf16b2377ffc"}) as unknown as TypedDocumentString<UpdateRecurringEarningMutation, UpdateRecurringEarningMutationVariables>;
export const IssuePayAdvanceDocument = new TypedDocumentString(`
    mutation IssuePayAdvance($input: IssuePayAdvanceInput!) {
  issuePayAdvance(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:a473c5b8196a7a4b792c124cd2f065f1361bf5327ff39363222ead87010fd6f5"}) as unknown as TypedDocumentString<IssuePayAdvanceMutation, IssuePayAdvanceMutationVariables>;
export const WriteOffPayAdvanceDocument = new TypedDocumentString(`
    mutation WriteOffPayAdvance($input: WriteOffPayAdvanceInput!) {
  writeOffPayAdvance(input: $input) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:1e61ca8e3b303b5eeede39215a299509c6d83213342e1d13138c19d24c4c09f2"}) as unknown as TypedDocumentString<WriteOffPayAdvanceMutation, WriteOffPayAdvanceMutationVariables>;
export const OpenEscrowAccountDocument = new TypedDocumentString(`
    mutation OpenEscrowAccount($input: OpenEscrowAccountInput!) {
  openEscrowAccount(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:44f99ba015deaaba7ba8a36c235f1b9cd31960106e649665df8597d4104f2646"}) as unknown as TypedDocumentString<OpenEscrowAccountMutation, OpenEscrowAccountMutationVariables>;
export const UpdateEscrowAccountDocument = new TypedDocumentString(`
    mutation UpdateEscrowAccount($input: UpdateEscrowAccountInput!) {
  updateEscrowAccount(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:7c8e094b2dff1858e448e77b9315e02eabe34847e1c4bb36aaaa9da9afb8b791"}) as unknown as TypedDocumentString<UpdateEscrowAccountMutation, UpdateEscrowAccountMutationVariables>;
export const AdjustEscrowAccountDocument = new TypedDocumentString(`
    mutation AdjustEscrowAccount($input: AdjustEscrowAccountInput!) {
  adjustEscrowAccount(input: $input) {
    id
    balanceMinor
    version
  }
}
    `, {"hash":"sha256:67d1baa098e04646476ef4bec1bb7633ac6caac81c71f4b71b7b54b539c60a21"}) as unknown as TypedDocumentString<AdjustEscrowAccountMutation, AdjustEscrowAccountMutationVariables>;
export const CloseEscrowAccountDocument = new TypedDocumentString(`
    mutation CloseEscrowAccount($accountId: ID!) {
  closeEscrowAccount(accountId: $accountId) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:4058437f8024641e9e3da7a2d98b8655beb6dee382b4b9c3e77f255c6da27328"}) as unknown as TypedDocumentString<CloseEscrowAccountMutation, CloseEscrowAccountMutationVariables>;
export const GenerateSettlementBatchDocument = new TypedDocumentString(`
    mutation GenerateSettlementBatch($input: GenerateSettlementBatchInput!) {
  generateSettlementBatch(input: $input) {
    id
    name
    settlementCount
    exceptionCount
    totalGrossMinor
    totalNetMinor
  }
}
    `, {"hash":"sha256:08e2f9acd12b2117789bfffc7bc36d074c3c14ae66935f1953cd371093439277"}) as unknown as TypedDocumentString<GenerateSettlementBatchMutation, GenerateSettlementBatchMutationVariables>;
export const GenerateDriverSettlementDocument = new TypedDocumentString(`
    mutation GenerateDriverSettlement($input: GenerateDriverSettlementInput!) {
  generateDriverSettlement(input: $input) {
    id
    settlementNumber
  }
}
    `, {"hash":"sha256:0d8cbbe97e9f00c3c4c3e36e9bcc4c027760a26fc34c241cc15df6742056a4a2"}) as unknown as TypedDocumentString<GenerateDriverSettlementMutation, GenerateDriverSettlementMutationVariables>;
export const SubmitDriverSettlementDocument = new TypedDocumentString(`
    mutation SubmitDriverSettlement($input: DriverSettlementActionInput!) {
  submitDriverSettlement(input: $input) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:690603bb04ff7ceab1c1ac6949e3e5f932152daab6396a7f36fa8d72396e4aaf"}) as unknown as TypedDocumentString<SubmitDriverSettlementMutation, SubmitDriverSettlementMutationVariables>;
export const ApproveDriverSettlementDocument = new TypedDocumentString(`
    mutation ApproveDriverSettlement($input: DriverSettlementActionInput!) {
  approveDriverSettlement(input: $input) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:8256dbbc6c4af57dfaafe586fbc3cbc789baaa24e23ff173c0251b0ad208d7c0"}) as unknown as TypedDocumentString<ApproveDriverSettlementMutation, ApproveDriverSettlementMutationVariables>;
export const RejectDriverSettlementDocument = new TypedDocumentString(`
    mutation RejectDriverSettlement($input: DriverSettlementActionInput!) {
  rejectDriverSettlement(input: $input) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:655f7d88cbe5f76b89bc3ec04a1dde34465fefbdf00304559620b0b20dcc64f2"}) as unknown as TypedDocumentString<RejectDriverSettlementMutation, RejectDriverSettlementMutationVariables>;
export const PostDriverSettlementDocument = new TypedDocumentString(`
    mutation PostDriverSettlement($input: DriverSettlementActionInput!) {
  postDriverSettlement(input: $input) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:528e783cf5406e6b3c351f88e4deb5941a3bfc5b90c5d547e7c8e07b28ba136c"}) as unknown as TypedDocumentString<PostDriverSettlementMutation, PostDriverSettlementMutationVariables>;
export const MarkDriverSettlementPaidDocument = new TypedDocumentString(`
    mutation MarkDriverSettlementPaid($input: MarkDriverSettlementPaidInput!) {
  markDriverSettlementPaid(input: $input) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:52293ccd71fa3f9f752a2c6ee79ece56b5b121240379e0a777dbf2dd1b11788e"}) as unknown as TypedDocumentString<MarkDriverSettlementPaidMutation, MarkDriverSettlementPaidMutationVariables>;
export const VoidDriverSettlementDocument = new TypedDocumentString(`
    mutation VoidDriverSettlement($input: DriverSettlementActionInput!) {
  voidDriverSettlement(input: $input) {
    id
    status
    version
  }
}
    `, {"hash":"sha256:f5a663c3939fddfbe2d23987cad6a6ca7fd8a03df0e933270cfe021136c166e8"}) as unknown as TypedDocumentString<VoidDriverSettlementMutation, VoidDriverSettlementMutationVariables>;
export const RecalculateDriverSettlementDocument = new TypedDocumentString(`
    mutation RecalculateDriverSettlement($input: DriverSettlementActionInput!) {
  recalculateDriverSettlement(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:527b34093726f31a9b0c76a39a26b9776e811990ecdf563159d42623c9947350"}) as unknown as TypedDocumentString<RecalculateDriverSettlementMutation, RecalculateDriverSettlementMutationVariables>;
export const AddDriverSettlementAdjustmentDocument = new TypedDocumentString(`
    mutation AddDriverSettlementAdjustment($input: AddSettlementAdjustmentInput!) {
  addDriverSettlementAdjustment(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:847f907ae868014335a3a237a6ce3f3b7a0a0b56eea44d3b8e278bdc7b3fe2cb"}) as unknown as TypedDocumentString<AddDriverSettlementAdjustmentMutation, AddDriverSettlementAdjustmentMutationVariables>;
export const RemoveDriverSettlementAdjustmentDocument = new TypedDocumentString(`
    mutation RemoveDriverSettlementAdjustment($input: RemoveSettlementAdjustmentInput!) {
  removeDriverSettlementAdjustment(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:fa1ff7af4e0d35dbb7c44913e146dbfa4c122f02807028dfb69ce7d2710bee46"}) as unknown as TypedDocumentString<RemoveDriverSettlementAdjustmentMutation, RemoveDriverSettlementAdjustmentMutationVariables>;
export const HoldDriverPayEventDocument = new TypedDocumentString(`
    mutation HoldDriverPayEvent($input: HoldPayEventInput!) {
  holdDriverPayEvent(input: $input) {
    id
    status
    onHold
    holdReason
    version
  }
}
    `, {"hash":"sha256:19b550da25ad6ee05d644fc426fb0cb4eed437c28373ed7a55847e5218753a97"}) as unknown as TypedDocumentString<HoldDriverPayEventMutation, HoldDriverPayEventMutationVariables>;
export const ReleaseDriverPayEventDocument = new TypedDocumentString(`
    mutation ReleaseDriverPayEvent($payEventId: ID!) {
  releaseDriverPayEvent(payEventId: $payEventId) {
    id
    status
    onHold
    holdReason
    version
  }
}
    `, {"hash":"sha256:74e127862ab228018eb6817bcf2dfe47df3616ff045b776f9ceb7fb7768f8875"}) as unknown as TypedDocumentString<ReleaseDriverPayEventMutation, ReleaseDriverPayEventMutationVariables>;
export const AttachPayEventsToSettlementDocument = new TypedDocumentString(`
    mutation AttachPayEventsToSettlement($input: AttachPayEventsInput!) {
  attachPayEventsToSettlement(input: $input) {
    id
    status
    grossEarningsMinor
    netPayMinor
    version
  }
}
    `, {"hash":"sha256:ce8dcbcff0fbfc094af72f65403e51dd3e05b3e66be8f6e093daf7020d5647e6"}) as unknown as TypedDocumentString<AttachPayEventsToSettlementMutation, AttachPayEventsToSettlementMutationVariables>;
export const DetachPayEventFromSettlementDocument = new TypedDocumentString(`
    mutation DetachPayEventFromSettlement($input: DetachPayEventInput!) {
  detachPayEventFromSettlement(input: $input) {
    id
    status
    grossEarningsMinor
    netPayMinor
    version
  }
}
    `, {"hash":"sha256:30116a97824390c55a39f8976f0816dda38f06d8e92b2d06a87d7fb8c697672c"}) as unknown as TypedDocumentString<DetachPayEventFromSettlementMutation, DetachPayEventFromSettlementMutationVariables>;
export const BulkDriverSettlementActionDocument = new TypedDocumentString(`
    mutation BulkDriverSettlementAction($input: BulkSettlementActionInput!) {
  bulkDriverSettlementAction(input: $input) {
    results {
      settlementId
      success
      error
    }
    successCount
    failureCount
  }
}
    `, {"hash":"sha256:400632fe9026dfd69abf6dfee45c166104d0d028b6c3f13ab32b90a73da8eda6"}) as unknown as TypedDocumentString<BulkDriverSettlementActionMutation, BulkDriverSettlementActionMutationVariables>;
export const UpdateSettlementControlDocument = new TypedDocumentString(`
    mutation UpdateSettlementControl($input: UpdateSettlementControlInput!) {
  updateSettlementControl(input: $input) {
    id
    version
  }
}
    `, {"hash":"sha256:d1cd27e9fd5f72b0c09c7000850c3749c37cdb3b1e2eefa5ec9244a59b47d002"}) as unknown as TypedDocumentString<UpdateSettlementControlMutation, UpdateSettlementControlMutationVariables>;
export const SettlementBatchDetailDocument = new TypedDocumentString(`
    query SettlementBatchDetail($id: ID!) {
  settlementBatch(id: $id) {
    id
    status
    name
    periodStart
    periodEnd
    payDate
    settlementCount
    exceptionCount
    totalGrossMinor
    totalNetMinor
    currencyCode
    notes
    version
    settlements {
      id
      settlementNumber
      status
      classification
      grossEarningsMinor
      deductionsMinor
      netPayMinor
      currencyCode
      hasExceptions
      worker {
        id
        firstName
        lastName
      }
    }
  }
}
    `, {"hash":"sha256:f417f184b9f58856ebe25866495f696c2fa0cc4b56bf14615f9c74ed3a5d3250"}) as unknown as TypedDocumentString<SettlementBatchDetailQuery, SettlementBatchDetailQueryVariables>;
export const UnsettledPayEventsDocument = new TypedDocumentString(`
    query UnsettledPayEvents($workerId: ID!) {
  unsettledPayEvents(workerId: $workerId) {
    id
    shipmentId
    moveId
    eventDate
    grossAmountMinor
    totalMiles
    currencyCode
    proNumber
  }
}
    `, {"hash":"sha256:f9b3d5579995e1ea62171d1cd9c081bd5018e816c58f119b94589db91905ef98"}) as unknown as TypedDocumentString<UnsettledPayEventsQuery, UnsettledPayEventsQueryVariables>;
export const PayWorkerNowDocument = new TypedDocumentString(`
    mutation PayWorkerNow($input: PayWorkerNowInput!) {
  payWorkerNow(input: $input) {
    id
    settlementNumber
    status
    netPayMinor
    currencyCode
    paidAt
    paymentMethod
    paymentReference
  }
}
    `, {"hash":"sha256:db65206e7e8cc3bd6558487534d20c043d05e59b9bbef71d89b006fd38580973"}) as unknown as TypedDocumentString<PayWorkerNowMutation, PayWorkerNowMutationVariables>;
export const EdiPartnerScorecardsDocument = new TypedDocumentString(`
    query EdiPartnerScorecards($sinceHours: Int) {
  ediPartnerScorecards(sinceHours: $sinceHours) {
    partnerId
    partnerName
    partnerCode
    outboundTotal
    sentCount
    failedCount
    deadLetteredCount
    receivedCount
    deliverySuccessRate
    avgAckSeconds
    p95AckSeconds
    overdueAckCount
    pendingOver4hCount
    pendingOver24hCount
    oldestPendingAgeSeconds
  }
}
    `, {"hash":"sha256:f96494e915c8a2ece90302cd0a0fc58743dacddaa6685b3cabdae9b07c4cbda4"}) as unknown as TypedDocumentString<EdiPartnerScorecardsQuery, EdiPartnerScorecardsQueryVariables>;
export const EdiVolumeSeriesDocument = new TypedDocumentString(`
    query EdiVolumeSeries($sinceHours: Int) {
  ediVolumeSeries(sinceHours: $sinceHours) {
    bucketStart
    bucketSeconds
    outboundCount
    sentCount
    failedCount
    receivedCount
  }
}
    `, {"hash":"sha256:8021ec391401918feed133209023963adf7ea69667f15ceb36ea26dd898e5c74"}) as unknown as TypedDocumentString<EdiVolumeSeriesQuery, EdiVolumeSeriesQueryVariables>;
export const EdiTemplateListDocument = new TypedDocumentString(`
    query EdiTemplateList($input: DataTableConnectionInput!, $status: EdiTemplateStatus, $transactionSet: String, $direction: EdiDocumentDirection) {
  ediTemplates(
    input: $input
    status: $status
    transactionSet: $transactionSet
    direction: $direction
  ) {
    edges {
      node {
        ...EdiTemplateListFields
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
    fragment EdiTemplateVersionSummaryFields on EdiTemplateVersion {
  id
  businessUnitId
  organizationId
  templateId
  sourceVersionId
  versionNumber
  x12Version
  functionalGroupId
  status
  isActive
  notes
  certifiedAt
  activatedAt
  archivedAt
  deprecatedAt
  supersededAt
  version
  createdAt
  updatedAt
}
fragment EdiTemplateListFields on EdiTemplate {
  id
  businessUnitId
  organizationId
  documentTypeId
  name
  description
  direction
  standard
  transactionSet
  status
  version
  createdAt
  updatedAt
  versions {
    ...EdiTemplateVersionSummaryFields
  }
}`, {"hash":"sha256:f412648ecffa3ce3939f151f43e74eebb77d0278b8fc405c4a091eee7ae6298d"}) as unknown as TypedDocumentString<EdiTemplateListQuery, EdiTemplateListQueryVariables>;
export const EdiPartnerReadinessDocument = new TypedDocumentString(`
    query EdiPartnerReadiness($partnerIds: [ID!]!) {
  ediPartnerReadiness(partnerIds: $partnerIds) {
    partnerId
    ready
    completedCount
    totalCount
  }
}
    `, {"hash":"sha256:6a4869ef7f675e627e9080ece4fbf1ef01427e11cc848095267bbc312d95db31"}) as unknown as TypedDocumentString<EdiPartnerReadinessQuery, EdiPartnerReadinessQueryVariables>;
export const EdiSummaryDocument = new TypedDocumentString(`
    query EdiSummary($sinceHours: Int) {
  ediSummary(sinceHours: $sinceHours) {
    deliveryStatusCounts {
      status
      count
    }
    ackStatusCounts {
      status
      count
    }
    inboundFileStatusCounts {
      status
      count
    }
    inboundTransferStatusCounts {
      status
      count
    }
    overdueAckCount
    attentionItems {
      kind
      id
      partnerId
      partnerName
      partnerCode
      reference
      error
      occurredAt
    }
  }
}
    `, {"hash":"sha256:3ac851e5c896dbfb25ef2c5275379c815430951cf1f410cb48d6587ccbf2ce0a"}) as unknown as TypedDocumentString<EdiSummaryQuery, EdiSummaryQueryVariables>;
export const EdiPartnerTableDocument = new TypedDocumentString(`
    query EdiPartnerTable($input: DataTableConnectionInput!) {
  ediPartners(input: $input) {
    edges {
      node {
        ...EdiPartnerRowFields
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
    fragment EdiPartnerRowFields on EdiPartner {
  id
  businessUnitId
  organizationId
  kind
  status
  code
  name
  description
  internalOrganizationId
  customerId
  defaultTransportId
  defaultMappingProfileId
  country
  timezone
  contactName
  contactEmail
  contactPhone
  enabledForInbound
  enabledForOutbound
  version
  createdAt
  updatedAt
  internalOrganization {
    id
    name
  }
  connection {
    id
    method
    status
  }
  defaultTransport {
    id
    name
    method
  }
}`, {"hash":"sha256:c3d724f49cd3db7e385b862d3c367b5e262b45a37c949b38f79b9daf5d487b49"}) as unknown as TypedDocumentString<EdiPartnerTableQuery, EdiPartnerTableQueryVariables>;
export const EdiCommunicationProfileTableDocument = new TypedDocumentString(`
    query EdiCommunicationProfileTable($input: DataTableConnectionInput!) {
  ediCommunicationProfiles(input: $input) {
    edges {
      node {
        ...EdiCommunicationProfileRowFields
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
    fragment EdiCommunicationProfileRowFields on EdiCommunicationProfile {
  id
  businessUnitId
  organizationId
  ediPartnerId
  ediConnectionId
  method
  status
  name
  description
  secretState {
    key
  }
  version
  createdAt
  updatedAt
  partner {
    id
    code
    name
  }
}`, {"hash":"sha256:f223a5cffeab0423e08214971f5dff307b73c42cc7859433e0a64b375b3b999d"}) as unknown as TypedDocumentString<EdiCommunicationProfileTableQuery, EdiCommunicationProfileTableQueryVariables>;
export const EdiTransferTableDocument = new TypedDocumentString(`
    query EdiTransferTable($input: DataTableConnectionInput!, $direction: EdiTransferDirection!) {
  ediTransfers(input: $input, direction: $direction) {
    edges {
      node {
        ...EdiTransferRowFields
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
    fragment EdiTransferRowFields on EdiTransfer {
  id
  sourceOrganizationId
  sourceBusinessUnitId
  targetOrganizationId
  targetBusinessUnitId
  sourcePartnerId
  targetPartnerId
  sourceShipmentId
  targetShipmentId
  inboundMessageId
  status
  tenderPayload
  mappingSnapshot
  rejectionReason
  failureReason
  submittedAt
  processedAt
  version
  createdAt
  updatedAt
  sourcePartner {
    id
    code
    name
  }
  targetPartner {
    id
    code
    name
  }
}`, {"hash":"sha256:79fb13e369231b4319fbdc4fd4e5b47859a7a3b548834f5eb3ce9b8cd9414c65"}) as unknown as TypedDocumentString<EdiTransferTableQuery, EdiTransferTableQueryVariables>;
export const EdiMessageTableDocument = new TypedDocumentString(`
    query EdiMessageTable($input: DataTableConnectionInput!) {
  ediMessages(input: $input) {
    edges {
      node {
        ...EdiMessageRowFields
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
    fragment EdiMessageRowFields on EdiMessage {
  id
  businessUnitId
  organizationId
  ediPartnerId
  documentTypeId
  partnerDocumentProfileId
  shipmentId
  transferId
  inboundFileId
  direction
  transactionSet
  x12Version
  status
  interchangeControlNumber
  groupControlNumber
  transactionControlNumber
  segmentCount
  deliveryStatus
  deliveryRemotePath
  deliveryAttempts
  deliveryLastAttemptAt
  deliverySentAt
  deliveryLastError
  ackStatus
  ackMessageId
  ackReceivedAt
  ackLastError
  generatedAt
  version
  partner {
    id
    code
    name
  }
}`, {"hash":"sha256:af442ce1cb8573365cec4e594d4d3c656fc188d44a16e63f20a545c2dcc6e937"}) as unknown as TypedDocumentString<EdiMessageTableQuery, EdiMessageTableQueryVariables>;
export const EdiInboundFileTableDocument = new TypedDocumentString(`
    query EdiInboundFileTable($input: DataTableConnectionInput!) {
  ediInboundFiles(input: $input) {
    edges {
      node {
        ...EdiInboundFileRowFields
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
    fragment EdiInboundFileRowFields on EdiInboundFile {
  id
  businessUnitId
  organizationId
  communicationProfileId
  ediPartnerId
  method
  remotePath
  fileName
  checksum
  sizeBytes
  interchangeControlNumber
  isaSenderQualifier
  isaSenderId
  isaReceiverQualifier
  isaReceiverId
  status
  failureReason
  transactionCount
  receivedAt
  processedAt
  version
  partner {
    id
    code
    name
  }
}`, {"hash":"sha256:23a72ad0db50e1988827a9b85f280f720f93562e70933de12af52b544d0bebd4"}) as unknown as TypedDocumentString<EdiInboundFileTableQuery, EdiInboundFileTableQueryVariables>;
export const EdiMappingProfileTableDocument = new TypedDocumentString(`
    query EdiMappingProfileTable($input: DataTableConnectionInput!) {
  ediMappingProfiles(input: $input) {
    edges {
      node {
        ...EdiMappingProfileRowFields
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
    fragment EdiMappingProfileRowFields on EdiMappingProfile {
  id
  businessUnitId
  organizationId
  ediPartnerId
  name
  description
  version
  createdAt
  updatedAt
  partner {
    id
    code
    name
  }
  entries {
    id
    entityType
    sourceId
    sourceLabel
    targetId
    targetLabel
  }
}`, {"hash":"sha256:b8dd0f903ccc5fe99d5b781c747d43a7112d8356e9eb02049c5617a766dfe198"}) as unknown as TypedDocumentString<EdiMappingProfileTableQuery, EdiMappingProfileTableQueryVariables>;
export const EdiTestCaseTableDocument = new TypedDocumentString(`
    query EdiTestCaseTable($input: DataTableConnectionInput!, $partnerDocumentProfileId: ID) {
  ediTestCases(input: $input, partnerDocumentProfileId: $partnerDocumentProfileId) {
    edges {
      node {
        ...EdiTestCaseRowFields
      }
    }
    totalCount
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
    fragment EdiTestCaseRowFields on EdiTestCase {
  id
  businessUnitId
  organizationId
  partnerDocumentProfileId
  name
  description
  expectedWarnings
  expectedErrors
  version
  createdAt
  updatedAt
  documentProfile {
    id
    name
    direction
    transactionSet
    partner {
      id
      code
      name
    }
  }
}`, {"hash":"sha256:5eddd01690b554a6b32398b3bbb4d397ef7b355a3a0f988818783ad41a79bd43"}) as unknown as TypedDocumentString<EdiTestCaseTableQuery, EdiTestCaseTableQueryVariables>;
export const EmailProfileTableDocument = new TypedDocumentString(`
    query EmailProfileTable($input: DataTableConnectionInput!) {
  emailProfiles(input: $input) {
    edges {
      node {
        ...EmailProfileTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment EmailProfileTableRowFields on EmailProfile {
  id
  businessUnitId
  organizationId
  name
  description
  senderName
  senderEmail
  replyToEmail
  provider
  status
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:c038fb1fcdf44c087999c4ece3b1bf633228536561dd6e8f1b9e135a33f971c3"}) as unknown as TypedDocumentString<EmailProfileTableQuery, EmailProfileTableQueryVariables>;
export const EquipmentManufacturerTableDocument = new TypedDocumentString(`
    query EquipmentManufacturerTable($input: DataTableConnectionInput!) {
  equipmentManufacturers(input: $input) {
    edges {
      node {
        ...EquipmentManufacturerTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment EquipmentManufacturerTableRowFields on EquipmentManufacturer {
  id
  businessUnitId
  organizationId
  status
  name
  description
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:c2afeb6f14e3169daa15b0187712b44c1bf698a30b68fc0b76636537842844a6"}) as unknown as TypedDocumentString<EquipmentManufacturerTableQuery, EquipmentManufacturerTableQueryVariables>;
export const TractorTableDocument = new TypedDocumentString(`
    query TractorTable($input: DataTableConnectionInput!, $includeEquipmentDetails: Boolean = true, $includeFleetDetails: Boolean = true, $includeWorkerDetails: Boolean = true) {
  tractors(
    input: $input
    includeEquipmentDetails: $includeEquipmentDetails
    includeFleetDetails: $includeFleetDetails
    includeWorkerDetails: $includeWorkerDetails
  ) {
    edges {
      node {
        ...TractorTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment EquipmentTypeTableFields on EquipmentType {
  id
  code
  color
}
fragment EquipmentManufacturerTableFields on EquipmentManufacturer {
  id
  name
}
fragment FleetCodeTableFields on FleetCode {
  id
  code
  color
}
fragment UsStateTableFields on UsState {
  id
  name
  abbreviation
}
fragment WorkerTableReferenceFields on Worker {
  id
  firstName
  lastName
  wholeName
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment TractorTableRowFields on Tractor {
  id
  businessUnitId
  organizationId
  primaryWorkerId
  equipmentTypeId
  equipmentManufacturerId
  stateId
  fleetCodeId
  secondaryWorkerId
  status
  code
  model
  make
  year
  licensePlateNumber
  registrationNumber
  registrationExpiry
  vin
  lastKnownLocationId
  lastKnownLocationName
  version
  createdAt
  updatedAt
  customFields
  equipmentType {
    ...EquipmentTypeTableFields
  }
  equipmentManufacturer {
    ...EquipmentManufacturerTableFields
  }
  fleetCode {
    ...FleetCodeTableFields
  }
  state {
    ...UsStateTableFields
  }
  primaryWorker {
    ...WorkerTableReferenceFields
  }
  secondaryWorker {
    ...WorkerTableReferenceFields
  }
}`, {"hash":"sha256:221694538edc13a9273b326e0a39921877c7dbd0af16166a2415150aadc1c61e"}) as unknown as TypedDocumentString<TractorTableQuery, TractorTableQueryVariables>;
export const TrailerTableDocument = new TypedDocumentString(`
    query TrailerTable($input: DataTableConnectionInput!, $includeEquipmentDetails: Boolean = true, $includeFleetDetails: Boolean = true) {
  trailers(
    input: $input
    includeEquipmentDetails: $includeEquipmentDetails
    includeFleetDetails: $includeFleetDetails
  ) {
    edges {
      node {
        ...TrailerTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment EquipmentTypeTableFields on EquipmentType {
  id
  code
  color
}
fragment EquipmentManufacturerTableFields on EquipmentManufacturer {
  id
  name
}
fragment FleetCodeTableFields on FleetCode {
  id
  code
  color
}
fragment UsStateTableFields on UsState {
  id
  name
  abbreviation
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment TrailerTableRowFields on Trailer {
  id
  businessUnitId
  organizationId
  equipmentTypeId
  equipmentManufacturerId
  registrationStateId
  fleetCodeId
  status
  code
  model
  make
  year
  licensePlateNumber
  vin
  registrationNumber
  maxLoadWeight
  lastInspectionDate
  registrationExpiry
  lastKnownLocationId
  lastKnownLocationName
  version
  createdAt
  updatedAt
  customFields
  equipmentType {
    ...EquipmentTypeTableFields
  }
  equipmentManufacturer {
    ...EquipmentManufacturerTableFields
  }
  fleetCode {
    ...FleetCodeTableFields
  }
  registrationState {
    ...UsStateTableFields
  }
}`, {"hash":"sha256:a03e38be977d09fb8f9c421a646adde0cf7b54ed10970765f43e353ca61ae4fe"}) as unknown as TypedDocumentString<TrailerTableQuery, TrailerTableQueryVariables>;
export const EquipmentTypeTableDocument = new TypedDocumentString(`
    query EquipmentTypeTable($input: DataTableConnectionInput!, $classes: [EquipmentClass!]) {
  equipmentTypes(input: $input, classes: $classes) {
    edges {
      node {
        ...EquipmentTypeConfigurationRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment EquipmentTypeConfigurationRowFields on EquipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  class
  color
  interiorLength
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:eb3d22a152d723f6080cf09beda77d7de525d386bafd988d2fc5432d4244dbee"}) as unknown as TypedDocumentString<EquipmentTypeTableQuery, EquipmentTypeTableQueryVariables>;
export const EquipmentTypeDocument = new TypedDocumentString(`
    query EquipmentType($id: ID!) {
  equipmentType(id: $id) {
    ...EquipmentTypeConfigurationRowFields
  }
}
    fragment EquipmentTypeConfigurationRowFields on EquipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  class
  color
  interiorLength
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:77492fa4f96133c985d9c81eff2aea6ca53852b002fa122718aeae37e4fd5b06"}) as unknown as TypedDocumentString<EquipmentTypeQuery, EquipmentTypeQueryVariables>;
export const CreateEquipmentTypeDocument = new TypedDocumentString(`
    mutation CreateEquipmentType($input: EquipmentTypeInput!) {
  createEquipmentType(input: $input) {
    ...EquipmentTypeConfigurationRowFields
  }
}
    fragment EquipmentTypeConfigurationRowFields on EquipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  class
  color
  interiorLength
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:804bf1016e1197c35723926d418334090e7ae5a6d68fe1e8a31517d60b87796f"}) as unknown as TypedDocumentString<CreateEquipmentTypeMutation, CreateEquipmentTypeMutationVariables>;
export const UpdateEquipmentTypeDocument = new TypedDocumentString(`
    mutation UpdateEquipmentType($id: ID!, $input: EquipmentTypeInput!) {
  updateEquipmentType(id: $id, input: $input) {
    ...EquipmentTypeConfigurationRowFields
  }
}
    fragment EquipmentTypeConfigurationRowFields on EquipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  class
  color
  interiorLength
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:05012e23f44106760ea0b9c4574517ce5095c9de4eddae711e8582b983f1d849"}) as unknown as TypedDocumentString<UpdateEquipmentTypeMutation, UpdateEquipmentTypeMutationVariables>;
export const PatchEquipmentTypeDocument = new TypedDocumentString(`
    mutation PatchEquipmentType($id: ID!, $input: EquipmentTypePatchInput!) {
  patchEquipmentType(id: $id, input: $input) {
    ...EquipmentTypeConfigurationRowFields
  }
}
    fragment EquipmentTypeConfigurationRowFields on EquipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  class
  color
  interiorLength
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:c5c81ee0f81421708ff81f3e3e2c42c61d42b40f5a5cee5993a778df6b36ca1d"}) as unknown as TypedDocumentString<PatchEquipmentTypeMutation, PatchEquipmentTypeMutationVariables>;
export const BulkUpdateEquipmentTypeStatusDocument = new TypedDocumentString(`
    mutation BulkUpdateEquipmentTypeStatus($input: BulkUpdateEquipmentTypeStatusInput!) {
  bulkUpdateEquipmentTypeStatus(input: $input) {
    ...EquipmentTypeConfigurationRowFields
  }
}
    fragment EquipmentTypeConfigurationRowFields on EquipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  class
  color
  interiorLength
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:a7e10803dec124d60c6f74fbce991a84c22d9e6dd45781e8bdc63de2ac10b9b3"}) as unknown as TypedDocumentString<BulkUpdateEquipmentTypeStatusMutation, BulkUpdateEquipmentTypeStatusMutationVariables>;
export const FiscalYearTableDocument = new TypedDocumentString(`
    query FiscalYearTable($input: DataTableConnectionInput!) {
  fiscalYears(input: $input) {
    edges {
      node {
        ...FiscalYearTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment FiscalPeriodFields on FiscalPeriod {
  id
  businessUnitId
  organizationId
  fiscalYearId
  periodNumber
  periodType
  status
  name
  startDate
  endDate
  closedAt
  version
  createdAt
  updatedAt
}
fragment FiscalYearTableRowFields on FiscalYear {
  id
  businessUnitId
  organizationId
  status
  year
  name
  description
  startDate
  endDate
  isCurrent
  isCalendarYear
  allowAdjustingEntries
  version
  createdAt
  updatedAt
  periods {
    ...FiscalPeriodFields
  }
}`, {"hash":"sha256:159f55ea593ba3c8d3ad7f9c22cbdc9f4faf731f2b1dd2ebe34da83b1eac82b3"}) as unknown as TypedDocumentString<FiscalYearTableQuery, FiscalYearTableQueryVariables>;
export const FleetCodeTableDocument = new TypedDocumentString(`
    query FleetCodeTable($input: DataTableConnectionInput!) {
  fleetCodes(input: $input) {
    edges {
      node {
        ...FleetCodeTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment FleetCodeTableRowFields on FleetCode {
  id
  businessUnitId
  organizationId
  managerId
  status
  code
  description
  revenueGoal
  deadheadGoal
  mileageGoal
  color
  version
  createdAt
  updatedAt
  manager {
    id
    name
  }
}`, {"hash":"sha256:9e60d0f64fda83c8ea362f780adbecf5c40a3746eaae58f5e4697f7f61da6c68"}) as unknown as TypedDocumentString<FleetCodeTableQuery, FleetCodeTableQueryVariables>;
export const FormulaTemplateTableDocument = new TypedDocumentString(`
    query FormulaTemplateTable($input: DataTableConnectionInput!) {
  formulaTemplates(input: $input) {
    edges {
      node {
        ...FormulaTemplateTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment FormulaTemplateTableRowFields on FormulaTemplate {
  id
  businessUnitId
  organizationId
  name
  description
  type
  expression
  status
  schemaId
  version
  currentVersionNumber
  createdAt
  updatedAt
}`, {"hash":"sha256:c01b733598294ed6c946e930dbe90a2c6ac8b264ef6cce8c9e19a4f7260c9722"}) as unknown as TypedDocumentString<FormulaTemplateTableQuery, FormulaTemplateTableQueryVariables>;
export const FuelIndexTableDocument = new TypedDocumentString(`
    query FuelIndexTable($input: DataTableConnectionInput!) {
  fuelIndexes(input: $input) {
    edges {
      node {
        ...FuelIndexFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment FuelIndexFields on FuelIndex {
  id
  businessUnitId
  organizationId
  name
  code
  description
  source
  fuelType
  region
  eiaSeriesId
  currency
  isActive
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:3d9de58edce377a1196d022cadeed9fdad99c2f1ab648d46185115a65dfc82ac"}) as unknown as TypedDocumentString<FuelIndexTableQuery, FuelIndexTableQueryVariables>;
export const FuelSurchargeProgramTableDocument = new TypedDocumentString(`
    query FuelSurchargeProgramTable($input: DataTableConnectionInput!) {
  fuelSurchargePrograms(input: $input) {
    edges {
      node {
        ...FuelSurchargeProgramFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment FuelSurchargeProgramFields on FuelSurchargeProgram {
  id
  businessUnitId
  organizationId
  name
  code
  description
  status
  fuelIndexId
  accessorialChargeId
  method
  pegPrice
  increment
  incrementRate
  milesPerGallon
  percentBasis
  stepRounding
  rateRounding
  ratePrecision
  minAmount
  maxAmount
  dateBasis
  priceEffectiveDay
  missingPriceFallback
  effectiveStartDate
  effectiveEndDate
  shipmentTypeIds
  serviceTypeIds
  tractorTypeIds
  trailerTypeIds
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:d3b91b52c41c370b27562468cf3e00dee15ba60342572a7b0397447d370030d9"}) as unknown as TypedDocumentString<FuelSurchargeProgramTableQuery, FuelSurchargeProgramTableQueryVariables>;
export const FuelSurchargeProgramDetailDocument = new TypedDocumentString(`
    query FuelSurchargeProgramDetail($id: ID!) {
  fuelSurchargeProgram(id: $id) {
    id
    name
    code
    description
    status
    fuelIndexId
    accessorialChargeId
    method
    pegPrice
    increment
    incrementRate
    milesPerGallon
    percentBasis
    stepRounding
    rateRounding
    ratePrecision
    minAmount
    maxAmount
    dateBasis
    priceEffectiveDay
    missingPriceFallback
    effectiveStartDate
    effectiveEndDate
    shipmentTypeIds
    serviceTypeIds
    tractorTypeIds
    trailerTypeIds
    version
    fuelIndex {
      id
      name
      code
      source
      fuelType
      region
    }
    accessorialCharge {
      id
      code
      description
    }
    tableRows {
      id
      priceMin
      priceMax
      value
      sortOrder
    }
  }
}
    `, {"hash":"sha256:668cfbb25c0bc4ff5599aa92de019bbabb47ff8af16e9eb7f9a53e59698287a9"}) as unknown as TypedDocumentString<FuelSurchargeProgramDetailQuery, FuelSurchargeProgramDetailQueryVariables>;
export const FuelDashboardDocument = new TypedDocumentString(`
    query FuelDashboard {
  fuelDashboard {
    index {
      id
      name
      code
      description
      source
      fuelType
      region
      eiaSeriesId
      currency
      isActive
    }
    latest {
      id
      priceDate
      price
      currency
      isManual
    }
    previous {
      id
      priceDate
      price
      currency
      isManual
    }
    delta
  }
}
    `, {"hash":"sha256:eb8d81ae4caebe6f5986cc6160e44c6c083ea25481fb4f66856d980d1b92ad43"}) as unknown as TypedDocumentString<FuelDashboardQuery, FuelDashboardQueryVariables>;
export const FuelIndexPriceHistoryDocument = new TypedDocumentString(`
    query FuelIndexPriceHistory($indexId: ID!, $from: String, $to: String, $limit: Int) {
  fuelIndexPriceHistory(indexId: $indexId, from: $from, to: $to, limit: $limit) {
    id
    fuelIndexId
    priceDate
    price
    currency
    isManual
    sourceRaw
    fetchedAt
  }
}
    `, {"hash":"sha256:2642b855e49ae205529b62bbdaa39e0c30f1476c31773ad47b63b0995208912c"}) as unknown as TypedDocumentString<FuelIndexPriceHistoryQuery, FuelIndexPriceHistoryQueryVariables>;
export const FuelProgramCurrentRatesDocument = new TypedDocumentString(`
    query FuelProgramCurrentRates {
  fuelProgramCurrentRates {
    program {
      id
      name
      code
      description
      status
      method
      fuelIndexId
      priceEffectiveDay
      dateBasis
      fuelIndex {
        id
        name
        code
        source
        fuelType
        region
      }
    }
    price {
      id
      priceDate
      price
      currency
    }
    ratePerMile
    percent
    flatAmount
    usedFallback
    matchedRow {
      id
      priceMin
      priceMax
      value
    }
  }
}
    `, {"hash":"sha256:b6ded035e6dea96b6e99b7d7515aa56dab0a13682606b38c7374f980be5a15f0"}) as unknown as TypedDocumentString<FuelProgramCurrentRatesQuery, FuelProgramCurrentRatesQueryVariables>;
export const GenerateFuelSurchargeTableDocument = new TypedDocumentString(`
    query GenerateFuelSurchargeTable($input: GenerateFuelTableInput!) {
  generateFuelSurchargeTable(input: $input) {
    priceMin
    priceMax
    value
  }
}
    `, {"hash":"sha256:a9038137e3854fa509ebd9e1ffaead13acfb73244483ca6a157c1676bab27dc1"}) as unknown as TypedDocumentString<GenerateFuelSurchargeTableQuery, GenerateFuelSurchargeTableQueryVariables>;
export const EiaSeriesOptionsDocument = new TypedDocumentString(`
    query EIASeriesOptions {
  eiaSeriesOptions {
    seriesId
    code
    name
    region
    fuelType
  }
}
    `, {"hash":"sha256:aaf292fcd2d43d06a7f9694efaa21c08892e926da551fec3ee3747349ca474ee"}) as unknown as TypedDocumentString<EiaSeriesOptionsQuery, EiaSeriesOptionsQueryVariables>;
export const CreateFuelIndexDocument = new TypedDocumentString(`
    mutation CreateFuelIndex($input: FuelIndexInput!) {
  createFuelIndex(input: $input) {
    id
    name
    code
  }
}
    `, {"hash":"sha256:c319990c4b5f3d40c1860bbed53325ee8253ff65822862a15d29417e5eb7c576"}) as unknown as TypedDocumentString<CreateFuelIndexMutation, CreateFuelIndexMutationVariables>;
export const UpdateFuelIndexDocument = new TypedDocumentString(`
    mutation UpdateFuelIndex($id: ID!, $input: FuelIndexInput!) {
  updateFuelIndex(id: $id, input: $input) {
    id
    name
    code
  }
}
    `, {"hash":"sha256:4f399ea1e3726ba5b46b72cceedf871a79d6c5491ce72140b22f0e3ab382cf5e"}) as unknown as TypedDocumentString<UpdateFuelIndexMutation, UpdateFuelIndexMutationVariables>;
export const DeleteFuelIndexDocument = new TypedDocumentString(`
    mutation DeleteFuelIndex($id: ID!) {
  deleteFuelIndex(id: $id)
}
    `, {"hash":"sha256:2755ccaf08a669e4ef6d4431518c9b88f6050aac4722e8317ec51c0ce60a5b5b"}) as unknown as TypedDocumentString<DeleteFuelIndexMutation, DeleteFuelIndexMutationVariables>;
export const AddFuelIndexPriceDocument = new TypedDocumentString(`
    mutation AddFuelIndexPrice($input: FuelIndexPriceInput!) {
  addFuelIndexPrice(input: $input) {
    id
    fuelIndexId
    priceDate
    price
  }
}
    `, {"hash":"sha256:df875f75a8e1490c5ede2d25401a7f4532db9104475770a27e64e660735d7f51"}) as unknown as TypedDocumentString<AddFuelIndexPriceMutation, AddFuelIndexPriceMutationVariables>;
export const UpdateFuelIndexPriceDocument = new TypedDocumentString(`
    mutation UpdateFuelIndexPrice($input: UpdateFuelIndexPriceInput!) {
  updateFuelIndexPrice(input: $input) {
    id
    fuelIndexId
    priceDate
    price
  }
}
    `, {"hash":"sha256:edc0d10d2ca1db5814601c0fbd342451e82e378fe16c5543c3b815a412a1e255"}) as unknown as TypedDocumentString<UpdateFuelIndexPriceMutation, UpdateFuelIndexPriceMutationVariables>;
export const DeleteFuelIndexPriceDocument = new TypedDocumentString(`
    mutation DeleteFuelIndexPrice($id: ID!) {
  deleteFuelIndexPrice(id: $id)
}
    `, {"hash":"sha256:8903bc47ee21d85ead19724a808087144465a1468b0323a72386d1ee5b627b20"}) as unknown as TypedDocumentString<DeleteFuelIndexPriceMutation, DeleteFuelIndexPriceMutationVariables>;
export const CreateFuelSurchargeProgramDocument = new TypedDocumentString(`
    mutation CreateFuelSurchargeProgram($input: FuelSurchargeProgramInput!) {
  createFuelSurchargeProgram(input: $input) {
    id
    name
    code
  }
}
    `, {"hash":"sha256:c38694d28544cc10ca5710f466a1932d5a860625706c97f7fee1ea1257db38b0"}) as unknown as TypedDocumentString<CreateFuelSurchargeProgramMutation, CreateFuelSurchargeProgramMutationVariables>;
export const UpdateFuelSurchargeProgramDocument = new TypedDocumentString(`
    mutation UpdateFuelSurchargeProgram($id: ID!, $input: FuelSurchargeProgramInput!) {
  updateFuelSurchargeProgram(id: $id, input: $input) {
    id
    name
    code
  }
}
    `, {"hash":"sha256:2d15f5ba8184f8be6fdf1628b9d674c4c98f117fa8c36e0cabd847068c7cd45c"}) as unknown as TypedDocumentString<UpdateFuelSurchargeProgramMutation, UpdateFuelSurchargeProgramMutationVariables>;
export const DeleteFuelSurchargeProgramDocument = new TypedDocumentString(`
    mutation DeleteFuelSurchargeProgram($id: ID!) {
  deleteFuelSurchargeProgram(id: $id)
}
    `, {"hash":"sha256:29699115bac5fef098c018951b45e9a68e8f1ab0fd98c45f23d69cfc60885a4b"}) as unknown as TypedDocumentString<DeleteFuelSurchargeProgramMutation, DeleteFuelSurchargeProgramMutationVariables>;
export const HazardousMaterialTableDocument = new TypedDocumentString(`
    query HazardousMaterialTable($input: DataTableConnectionInput!) {
  hazardousMaterials(input: $input) {
    edges {
      node {
        ...HazardousMaterialTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment HazardousMaterialTableRowFields on HazardousMaterial {
  id
  businessUnitId
  organizationId
  status
  code
  name
  description
  class
  unNumber
  packingGroup
  subsidiaryHazardClass
  ergGuideNumber
  labelCodes
  specialProvisions
  properShippingName
  handlingInstructions
  emergencyContact
  emergencyContactPhoneNumber
  quantityThreshold
  placardRequired
  isReportableQuantity
  marinePollutant
  inhalationHazard
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:9d47f5a414bcb4f1a88805ae94dc4768f7f859ca59ebd101fb9934a6ad5a5701"}) as unknown as TypedDocumentString<HazardousMaterialTableQuery, HazardousMaterialTableQueryVariables>;
export const HazmatSegregationRuleTableDocument = new TypedDocumentString(`
    query HazmatSegregationRuleTable($input: DataTableConnectionInput!) {
  hazmatSegregationRules(input: $input) {
    edges {
      node {
        ...HazmatSegregationRuleTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment HazmatSegregationRuleTableRowFields on HazmatSegregationRule {
  id
  businessUnitId
  organizationId
  status
  name
  description
  exceptionNotes
  referenceCode
  regulationSource
  distanceUnit
  classA
  classB
  segregationType
  hasExceptions
  hazmatAId
  hazmatBId
  minimumDistance
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:936ac7914a10d12b383f19c58fae243218ae68c137bc3d09cb7dc8f92a4ec676"}) as unknown as TypedDocumentString<HazmatSegregationRuleTableQuery, HazmatSegregationRuleTableQueryVariables>;
export const HoldReasonTableDocument = new TypedDocumentString(`
    query HoldReasonTable($input: DataTableConnectionInput!) {
  holdReasons(input: $input) {
    edges {
      node {
        ...HoldReasonTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment HoldReasonTableRowFields on HoldReason {
  id
  businessUnitId
  organizationId
  type
  code
  label
  description
  active
  defaultSeverity
  defaultBlocksDispatch
  defaultBlocksDelivery
  defaultBlocksBilling
  defaultVisibleToCustomer
  sortOrder
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:366e81c2432a4eafc200d3719e3b3917a9ac0fccc30df96abd7a3450fe189e31"}) as unknown as TypedDocumentString<HoldReasonTableQuery, HoldReasonTableQueryVariables>;
export const InvoiceTableDocument = new TypedDocumentString(`
    query InvoiceTable($input: DataTableConnectionInput!) {
  invoices(input: $input) {
    edges {
      node {
        ...InvoiceTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment InvoiceTableRowFields on Invoice {
  id
  billingQueueItemId
  shipmentId
  customerId
  number
  billType
  status
  paymentTerm
  currencyCode
  invoiceDate
  dueDate
  billToName
  subtotalAmount
  otherAmount
  totalAmount
  appliedAmount
  settlementStatus
  disputeStatus
  sendStatus
  isAdjustmentArtifact
  version
  createdAt
  updatedAt
  customer {
    id
    name
    code
  }
}`, {"hash":"sha256:c0a125f36486b5046c94215debaab6cc1d2f63ff57cfdfed8c9d9ceadb3c1d3c"}) as unknown as TypedDocumentString<InvoiceTableQuery, InvoiceTableQueryVariables>;
export const JournalEntryDetailDocument = new TypedDocumentString(`
    query JournalEntryDetail($id: ID!) {
  journalEntry(id: $id) {
    id
    organizationId
    businessUnitId
    batchId
    fiscalYearId
    fiscalPeriodId
    entryNumber
    entryType
    status
    accountingDate
    description
    referenceType
    referenceId
    totalDebit
    totalCredit
    isPosted
    isReversal
    reversalOfId
    reversedById
    reversalDate
    reversalReason
    lines {
      id
      journalEntryId
      glAccountId
      lineNumber
      description
      debitAmount
      creditAmount
      netAmount
      customerId
      locationId
      glAccount {
        id
        accountCode
        name
      }
    }
  }
}
    `, {"hash":"sha256:9115c76311ea912a3c9abf400bf6fc66c811ec2227b1500769f88a829782a646"}) as unknown as TypedDocumentString<JournalEntryDetailQuery, JournalEntryDetailQueryVariables>;
export const JournalSourceByObjectDocument = new TypedDocumentString(`
    query JournalSourceByObject($sourceType: String!, $sourceId: String!) {
  journalSourceByObject(sourceType: $sourceType, sourceId: $sourceId) {
    id
    sourceObjectType
    sourceObjectId
    sourceEventType
    sourceDocumentNumber
    status
  }
}
    `, {"hash":"sha256:9fc6924e799999cbc0d1e413752b76e244eb6f3d8f0cef4eec55f1c5d2b785af"}) as unknown as TypedDocumentString<JournalSourceByObjectQuery, JournalSourceByObjectQueryVariables>;
export const JournalEntriesBySourceDocument = new TypedDocumentString(`
    query JournalEntriesBySource($sourceType: String!, $sourceId: String!) {
  journalEntriesBySource(sourceType: $sourceType, sourceId: $sourceId) {
    id
    batchId
    entryNumber
    entryType
    status
    accountingDate
    description
    referenceType
    referenceId
    totalDebit
    totalCredit
    isPosted
    isReversal
    lines {
      id
      journalEntryId
      glAccountId
      lineNumber
      description
      debitAmount
      creditAmount
      netAmount
      customerId
      locationId
      glAccount {
        id
        accountCode
        name
      }
    }
  }
}
    `, {"hash":"sha256:fda3eefb446f90d8e933f63b8db868bf7045f0a841ecafdd013ff3240df17e1f"}) as unknown as TypedDocumentString<JournalEntriesBySourceQuery, JournalEntriesBySourceQueryVariables>;
export const JournalReversalTableDocument = new TypedDocumentString(`
    query JournalReversalTable($input: DataTableConnectionInput!) {
  journalReversals(input: $input) {
    edges {
      node {
        ...JournalReversalTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment JournalReversalTableRowFields on JournalReversal {
  id
  businessUnitId
  organizationId
  originalJournalEntryId
  reversalJournalEntryId
  postedBatchId
  status
  requestedAccountingDate
  resolvedFiscalYearId
  resolvedFiscalPeriodId
  reasonCode
  reasonText
  requestedById
  approvedById
  approvedAt
  rejectedById
  rejectedAt
  rejectionReason
  cancelledById
  cancelledAt
  cancelReason
  postedById
  postedAt
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:24e728ee31f68aae6a69d417eef59bab9bd326df315025eb96e50213d737aa49"}) as unknown as TypedDocumentString<JournalReversalTableQuery, JournalReversalTableQueryVariables>;
export const LocationCategoryTableDocument = new TypedDocumentString(`
    query LocationCategoryTable($input: DataTableConnectionInput!) {
  locationCategories(input: $input) {
    edges {
      node {
        ...LocationCategoryTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment LocationCategoryTableRowFields on LocationCategory {
  id
  businessUnitId
  organizationId
  name
  description
  type
  facilityType
  color
  hasSecureParking
  requiresAppointment
  allowsOvernight
  hasRestroom
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:2195db009823cc4e4abe821398a2ead3cc9de4499d7abf02057459e0a9feb0fb"}) as unknown as TypedDocumentString<LocationCategoryTableQuery, LocationCategoryTableQueryVariables>;
export const LocationTableDocument = new TypedDocumentString(`
    query LocationTable($input: DataTableConnectionInput!) {
  locations(input: $input) {
    edges {
      node {
        ...LocationTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment LocationTableRowFields on Location {
  id
  businessUnitId
  organizationId
  locationCategoryId
  stateId
  status
  code
  name
  description
  addressLine1
  addressLine2
  city
  postalCode
  version
  createdAt
  updatedAt
  state {
    id
    name
    abbreviation
  }
  locationCategory {
    id
    name
    color
  }
}`, {"hash":"sha256:277bd5cfec39789703fce55c06d35fddaea3f6a6e19dd6d4d20dd940a1a9f1bc"}) as unknown as TypedDocumentString<LocationTableQuery, LocationTableQueryVariables>;
export const ManualJournalTableDocument = new TypedDocumentString(`
    query ManualJournalTable($input: DataTableConnectionInput!) {
  manualJournals(input: $input) {
    edges {
      node {
        ...ManualJournalTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment ManualJournalTableRowFields on ManualJournal {
  id
  businessUnitId
  organizationId
  requestNumber
  status
  description
  reason
  accountingDate
  requestedFiscalYearId
  requestedFiscalPeriodId
  currencyCode
  totalDebit
  totalCredit
  approvedAt
  approvedById
  rejectedAt
  rejectedById
  rejectionReason
  cancelledAt
  cancelledById
  cancelReason
  postedBatchId
  createdById
  updatedById
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:68c27faeae7ef0ec21693a745b9c58fb933c9e3ab8a395de79705953a4200bbd"}) as unknown as TypedDocumentString<ManualJournalTableQuery, ManualJournalTableQueryVariables>;
export const NotificationListDocument = new TypedDocumentString(`
    query NotificationList($input: DataTableConnectionInput!, $filter: NotificationFilterInput) {
  notifications(input: $input, filter: $filter) {
    edges {
      node {
        ...NotificationFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment NotificationFields on Notification {
  id
  organizationId
  businessUnitId
  targetUserId
  eventType
  priority
  channel
  title
  message
  data
  relatedEntities
  source
  readAt
  dismissedAt
  createdAt
}`, {"hash":"sha256:f025dd4c0d391b965d6c43d4a747b7366afc5b88a7365cad8010cf64ae5a8283"}) as unknown as TypedDocumentString<NotificationListQuery, NotificationListQueryVariables>;
export const NotificationUnreadCountDocument = new TypedDocumentString(`
    query NotificationUnreadCount {
  notificationUnreadCount
}
    `, {"hash":"sha256:ed4fa686e9641b77e14b47a1c93d30e473f479f8cb960bc62def2000568e84a7"}) as unknown as TypedDocumentString<NotificationUnreadCountQuery, NotificationUnreadCountQueryVariables>;
export const MarkNotificationsReadDocument = new TypedDocumentString(`
    mutation MarkNotificationsRead($ids: [ID!]!) {
  markNotificationsRead(ids: $ids)
}
    `, {"hash":"sha256:1a766cf4ea3e134b35e5fa8ec5b904da700ab13ff3d61af7158809b586e6fe94"}) as unknown as TypedDocumentString<MarkNotificationsReadMutation, MarkNotificationsReadMutationVariables>;
export const MarkNotificationsUnreadDocument = new TypedDocumentString(`
    mutation MarkNotificationsUnread($ids: [ID!]!) {
  markNotificationsUnread(ids: $ids)
}
    `, {"hash":"sha256:4623f51d1c45a298af41a975f66d19897b98c9c1527f03162eefac1aef651ca2"}) as unknown as TypedDocumentString<MarkNotificationsUnreadMutation, MarkNotificationsUnreadMutationVariables>;
export const MarkAllNotificationsReadDocument = new TypedDocumentString(`
    mutation MarkAllNotificationsRead {
  markAllNotificationsRead
}
    `, {"hash":"sha256:e919497b911d73638f8329785ecb0b4b48a247bb6d037bf89b2a498c5bca336d"}) as unknown as TypedDocumentString<MarkAllNotificationsReadMutation, MarkAllNotificationsReadMutationVariables>;
export const DismissNotificationsDocument = new TypedDocumentString(`
    mutation DismissNotifications($ids: [ID!]!) {
  dismissNotifications(ids: $ids)
}
    `, {"hash":"sha256:762abd6aba103c349367b7834a0e909dd2e06b9d5c1a33f71a4467431db83d50"}) as unknown as TypedDocumentString<DismissNotificationsMutation, DismissNotificationsMutationVariables>;
export const RestoreNotificationsDocument = new TypedDocumentString(`
    mutation RestoreNotifications($ids: [ID!]!) {
  restoreNotifications(ids: $ids)
}
    `, {"hash":"sha256:e97ca2a47ac7291064a1651afaf8807310b842b010cc23d5b383cab58018d6e1"}) as unknown as TypedDocumentString<RestoreNotificationsMutation, RestoreNotificationsMutationVariables>;
export const OrderDetailDocument = new TypedDocumentString(`
    query OrderDetail($id: ID!) {
  order(id: $id) {
    id
    orderNumber
    status
    customerId
    ownerId
    poNumber
    bol
    currencyCode
    quotedAmount
    baseAmount
    totalAmount
    version
    createdAt
    updatedAt
    customer {
      id
      name
      code
    }
    legs {
      id
      proNumber
      status
      bol
      freightChargeAmount
      totalChargeAmount
    }
    charges {
      id
      description
      amount
      invoiceId
      version
      createdAt
    }
  }
}
    `, {"hash":"sha256:7f3565d2e4b7025b6b94522b2f084b22e66738f934977fcf89f7921873655f7b"}) as unknown as TypedDocumentString<OrderDetailQuery, OrderDetailQueryVariables>;
export const AttachOrderShipmentsDocument = new TypedDocumentString(`
    mutation AttachOrderShipments($orderId: ID!, $shipmentIds: [ID!]!) {
  attachOrderShipments(orderId: $orderId, shipmentIds: $shipmentIds) {
    ...OrderMutationResult
  }
}
    fragment OrderMutationResult on Order {
  id
  orderNumber
  status
  totalAmount
  version
}`, {"hash":"sha256:c5dd0f391421cd1c7def4a849abaf9630283cc0b9b5c4ad83164f677b79273a9"}) as unknown as TypedDocumentString<AttachOrderShipmentsMutation, AttachOrderShipmentsMutationVariables>;
export const DetachOrderShipmentDocument = new TypedDocumentString(`
    mutation DetachOrderShipment($orderId: ID!, $shipmentId: ID!) {
  detachOrderShipment(orderId: $orderId, shipmentId: $shipmentId) {
    ...OrderMutationResult
  }
}
    fragment OrderMutationResult on Order {
  id
  orderNumber
  status
  totalAmount
  version
}`, {"hash":"sha256:5a7b3fa35274ee455c2c6c8eb92842cbf663284cf5639cbc0c4dea9d7350984c"}) as unknown as TypedDocumentString<DetachOrderShipmentMutation, DetachOrderShipmentMutationVariables>;
export const CreateInvoiceFromOrderDocument = new TypedDocumentString(`
    mutation CreateInvoiceFromOrder($orderId: ID!) {
  createInvoiceFromOrder(orderId: $orderId) {
    id
    number
  }
}
    `, {"hash":"sha256:cae43848db3ff746b04aca0c2e169bedeb64675591f61d673d246905ba337a3b"}) as unknown as TypedDocumentString<CreateInvoiceFromOrderMutation, CreateInvoiceFromOrderMutationVariables>;
export const CreateOrderDocument = new TypedDocumentString(`
    mutation CreateOrder($input: OrderInput!) {
  createOrder(input: $input) {
    id
    orderNumber
    status
    version
  }
}
    `, {"hash":"sha256:7fb5e40596d163d5d39851904b98476de863ef34c103de223cdcdef3ee097041"}) as unknown as TypedDocumentString<CreateOrderMutation, CreateOrderMutationVariables>;
export const UpdateOrderDocument = new TypedDocumentString(`
    mutation UpdateOrder($id: ID!, $input: OrderInput!) {
  updateOrder(id: $id, input: $input) {
    id
    orderNumber
    status
    version
  }
}
    `, {"hash":"sha256:96308fccacc82642fbb51c915e2fa69d53fa1d7a4a5eb982d352dc4c0cdeb409"}) as unknown as TypedDocumentString<UpdateOrderMutation, UpdateOrderMutationVariables>;
export const AddOrderChargeDocument = new TypedDocumentString(`
    mutation AddOrderCharge($orderId: ID!, $description: String!, $amount: String!) {
  addOrderCharge(orderId: $orderId, description: $description, amount: $amount) {
    ...OrderMutationResult
  }
}
    fragment OrderMutationResult on Order {
  id
  orderNumber
  status
  totalAmount
  version
}`, {"hash":"sha256:2bf0e46b3ec50bc6c069b77db37155f0b4723c576a4ab0ae238e3d41eb2fa31d"}) as unknown as TypedDocumentString<AddOrderChargeMutation, AddOrderChargeMutationVariables>;
export const UpdateOrderChargeDocument = new TypedDocumentString(`
    mutation UpdateOrderCharge($input: UpdateOrderChargeInput!) {
  updateOrderCharge(input: $input) {
    ...OrderMutationResult
  }
}
    fragment OrderMutationResult on Order {
  id
  orderNumber
  status
  totalAmount
  version
}`, {"hash":"sha256:a92d821b04622cdf8267c6351a990835ddc95b0b98660680cab62b31dd72e495"}) as unknown as TypedDocumentString<UpdateOrderChargeMutation, UpdateOrderChargeMutationVariables>;
export const RemoveOrderChargeDocument = new TypedDocumentString(`
    mutation RemoveOrderCharge($input: RemoveOrderChargeInput!) {
  removeOrderCharge(input: $input) {
    ...OrderMutationResult
  }
}
    fragment OrderMutationResult on Order {
  id
  orderNumber
  status
  totalAmount
  version
}`, {"hash":"sha256:a75bec74d0ee9039320bcfcc64440bf120258d7bed35a4fce23b2e0aeb9f4779"}) as unknown as TypedDocumentString<RemoveOrderChargeMutation, RemoveOrderChargeMutationVariables>;
export const CloseOrderDocument = new TypedDocumentString(`
    mutation CloseOrder($id: ID!) {
  closeOrder(id: $id) {
    ...OrderMutationResult
  }
}
    fragment OrderMutationResult on Order {
  id
  orderNumber
  status
  totalAmount
  version
}`, {"hash":"sha256:29e28b70b87c2b56b742887aa61c4e902824411344b2406cb079fb5b235cb013"}) as unknown as TypedDocumentString<CloseOrderMutation, CloseOrderMutationVariables>;
export const CancelOrderDocument = new TypedDocumentString(`
    mutation CancelOrder($id: ID!, $cancelReason: String!) {
  cancelOrder(id: $id, cancelReason: $cancelReason) {
    ...OrderMutationResult
  }
}
    fragment OrderMutationResult on Order {
  id
  orderNumber
  status
  totalAmount
  version
}`, {"hash":"sha256:c2d8a6c844a02e81f0c6a42222ab9146f32e20dc17c14aa067b711e85838ea18"}) as unknown as TypedDocumentString<CancelOrderMutation, CancelOrderMutationVariables>;
export const OrderTableDocument = new TypedDocumentString(`
    query OrderTable($input: DataTableConnectionInput!) {
  orders(input: $input) {
    edges {
      node {
        ...OrderTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment OrderTableRowFields on Order {
  id
  ownerId
  businessUnitId
  organizationId
  customerId
  status
  orderNumber
  poNumber
  bol
  currencyCode
  quotedAmount
  baseAmount
  totalAmount
  version
  createdAt
  updatedAt
  customer {
    id
    name
    code
  }
}`, {"hash":"sha256:9e24ddaf07fd5362ac2d88844a75f0cc07107bfcdb5e00cd1ebc3c30c218aa0f"}) as unknown as TypedDocumentString<OrderTableQuery, OrderTableQueryVariables>;
export const OrganizationSettingsDocument = new TypedDocumentString(`
    query OrganizationSettings($id: ID!, $includeState: Boolean = true, $includeBu: Boolean = false) {
  organization(id: $id, includeState: $includeState, includeBu: $includeBu) {
    ...OrganizationSettingsFields
  }
}
    fragment OrganizationSettingsStateFields on UsState {
  id
  name
  abbreviation
}
fragment OrganizationSettingsFields on Organization {
  id
  version
  createdAt
  updatedAt
  bucketName
  businessUnitId
  loginSlug
  name
  scacCode
  dotNumber
  logoUrl
  addressLine1
  addressLine2
  city
  stateId
  postalCode
  timezone
  taxId
  state {
    ...OrganizationSettingsStateFields
  }
}`, {"hash":"sha256:f0607e7e7bb70dae8caafba84295b7a0c947b86d75e5502a1756f46bb3be4304"}) as unknown as TypedDocumentString<OrganizationSettingsQuery, OrganizationSettingsQueryVariables>;
export const UpdateOrganizationSettingsDocument = new TypedDocumentString(`
    mutation UpdateOrganizationSettings($id: ID!, $input: OrganizationInput!) {
  updateOrganization(id: $id, input: $input) {
    ...OrganizationSettingsFields
  }
}
    fragment OrganizationSettingsStateFields on UsState {
  id
  name
  abbreviation
}
fragment OrganizationSettingsFields on Organization {
  id
  version
  createdAt
  updatedAt
  bucketName
  businessUnitId
  loginSlug
  name
  scacCode
  dotNumber
  logoUrl
  addressLine1
  addressLine2
  city
  stateId
  postalCode
  timezone
  taxId
  state {
    ...OrganizationSettingsStateFields
  }
}`, {"hash":"sha256:9c93ea23726c32ce8a2683f6ff0e4f38d5a63a9ef0ddd9a86b9b98899aeaefe8"}) as unknown as TypedDocumentString<UpdateOrganizationSettingsMutation, UpdateOrganizationSettingsMutationVariables>;
export const RateTableTableDocument = new TypedDocumentString(`
    query RateTableTable($input: DataTableConnectionInput!) {
  rateTables(input: $input) {
    edges {
      node {
        ...RateTableTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment RateTableTableRowFields on RateTable {
  id
  businessUnitId
  organizationId
  name
  key
  description
  lookupType
  active
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:809e150e9d42fd0b2fecd3ced6e252820043552da234d6c6c72295e9303ffeee"}) as unknown as TypedDocumentString<RateTableTableQuery, RateTableTableQueryVariables>;
export const RecurringShipmentTableDocument = new TypedDocumentString(`
    query RecurringShipmentTable($input: DataTableConnectionInput!) {
  recurringShipments(input: $input) {
    edges {
      node {
        ...RecurringShipmentTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment RecurringShipmentTableRowFields on RecurringShipment {
  id
  businessUnitId
  organizationId
  sourceShipmentId
  customerId
  originLocationId
  destinationLocationId
  name
  description
  status
  cronExpression
  timezone
  startDate
  endDate
  maxOccurrences
  leadTimeDays
  skipWeekends
  exceptionPolicy
  blackoutDates
  autoGenerate
  nextOccurrenceAt
  lastOccurrenceAt
  lastRunAt
  generationCount
  consecutiveFailures
  version
  createdAt
  updatedAt
  customer {
    id
    name
    code
  }
  originLocation {
    id
    name
    code
  }
  destinationLocation {
    id
    name
    code
  }
}`, {"hash":"sha256:cc7af92cb220140002f4f6d8e4f32ee167ac757ee5427b9ff84403137e736eec"}) as unknown as TypedDocumentString<RecurringShipmentTableQuery, RecurringShipmentTableQueryVariables>;
export const CannedReportsDocument = new TypedDocumentString(`
    query CannedReports {
  cannedReports {
    key
    version
    name
    description
    category
    tags
    defaultFormat
    definition
  }
}
    `, {"hash":"sha256:597b62a2e0291d15c7fc013e195a15594efb4e35e2b82183a24c791037994b27"}) as unknown as TypedDocumentString<CannedReportsQuery, CannedReportsQueryVariables>;
export const ReportCatalogDocument = new TypedDocumentString(`
    query ReportCatalog {
  reportCatalog {
    version
    entities {
      key
      resource
      label
      pluralLabel
      description
      category
      ownScopeSupported
      fields {
        key
        label
        description
        type
        format
        nullable
        enumValues {
          value
          label
        }
        aggregations
        filterable
        groupable
        accessible
        sensitivity
      }
      edges {
        name
        label
        target
        cardinality
        traversable
      }
    }
  }
}
    `, {"hash":"sha256:79e369a4fec3bb0d7d5c6975d5782adb517ddeb55868c86d31e6f644f12d2d39"}) as unknown as TypedDocumentString<ReportCatalogQuery, ReportCatalogQueryVariables>;
export const ReportDefinitionsTableDocument = new TypedDocumentString(`
    query ReportDefinitionsTable($input: DataTableConnectionInput!) {
  reportDefinitions(input: $input) {
    edges {
      node {
        ...ReportDefinitionFields
      }
      cursor
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment ReportDefinitionFields on ReportDefinition {
  id
  name
  description
  category
  tags
  kind
  cannedKey
  cannedVersion
  ownerId
  visibility
  status
  diagnostics
  catalogVersion
  definition
  defaultFormat
  currentRevision
  lastRunAt
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:039c706284eb551f39048dcd134dcdc18d5c9cd8cef3875e0d17d3a0a6b35a10"}) as unknown as TypedDocumentString<ReportDefinitionsTableQuery, ReportDefinitionsTableQueryVariables>;
export const ReportDefinitionByIdDocument = new TypedDocumentString(`
    query ReportDefinitionById($id: ID!) {
  reportDefinition(id: $id) {
    ...ReportDefinitionFields
  }
}
    fragment ReportDefinitionFields on ReportDefinition {
  id
  name
  description
  category
  tags
  kind
  cannedKey
  cannedVersion
  ownerId
  visibility
  status
  diagnostics
  catalogVersion
  definition
  defaultFormat
  currentRevision
  lastRunAt
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:d55a36bcce0a8dba2f6772c0b29874ac0f7dfa18fb4f683fb8b9602fd744aea3"}) as unknown as TypedDocumentString<ReportDefinitionByIdQuery, ReportDefinitionByIdQueryVariables>;
export const ReportDefinitionRevisionsDocument = new TypedDocumentString(`
    query ReportDefinitionRevisions($definitionId: ID!, $limit: Int) {
  reportDefinitionRevisions(definitionId: $definitionId, limit: $limit) {
    id
    definitionId
    revisionNumber
    catalogVersion
    definition
    createdById
    createdAt
  }
}
    `, {"hash":"sha256:c2b09eb2de67685ef05b52a0b72e1dab1b2c0ab84c390073889a632d0192bb19"}) as unknown as TypedDocumentString<ReportDefinitionRevisionsQuery, ReportDefinitionRevisionsQueryVariables>;
export const CreateReportDefinitionDocument = new TypedDocumentString(`
    mutation CreateReportDefinition($input: SaveReportDefinitionInput!) {
  createReportDefinition(input: $input) {
    ...ReportDefinitionFields
  }
}
    fragment ReportDefinitionFields on ReportDefinition {
  id
  name
  description
  category
  tags
  kind
  cannedKey
  cannedVersion
  ownerId
  visibility
  status
  diagnostics
  catalogVersion
  definition
  defaultFormat
  currentRevision
  lastRunAt
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:02ff484673e0f31b72ae3222ce77fda8359c7deb67192b6642cd04a6a2113500"}) as unknown as TypedDocumentString<CreateReportDefinitionMutation, CreateReportDefinitionMutationVariables>;
export const UpdateReportDefinitionDocument = new TypedDocumentString(`
    mutation UpdateReportDefinition($input: UpdateReportDefinitionInput!) {
  updateReportDefinition(input: $input) {
    ...ReportDefinitionFields
  }
}
    fragment ReportDefinitionFields on ReportDefinition {
  id
  name
  description
  category
  tags
  kind
  cannedKey
  cannedVersion
  ownerId
  visibility
  status
  diagnostics
  catalogVersion
  definition
  defaultFormat
  currentRevision
  lastRunAt
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:8f03c81fa49d77ea4742568b2d61ed904b5c2ce14b46a73836390d3333c6b97b"}) as unknown as TypedDocumentString<UpdateReportDefinitionMutation, UpdateReportDefinitionMutationVariables>;
export const DeleteReportDefinitionDocument = new TypedDocumentString(`
    mutation DeleteReportDefinition($id: ID!) {
  deleteReportDefinition(id: $id)
}
    `, {"hash":"sha256:94b019b0a0a6bf268d050bd41d841b986997ed0566ca7673306374af6391871a"}) as unknown as TypedDocumentString<DeleteReportDefinitionMutation, DeleteReportDefinitionMutationVariables>;
export const ForkCannedReportDocument = new TypedDocumentString(`
    mutation ForkCannedReport($input: ForkCannedReportInput!) {
  forkCannedReport(input: $input) {
    ...ReportDefinitionFields
  }
}
    fragment ReportDefinitionFields on ReportDefinition {
  id
  name
  description
  category
  tags
  kind
  cannedKey
  cannedVersion
  ownerId
  visibility
  status
  diagnostics
  catalogVersion
  definition
  defaultFormat
  currentRevision
  lastRunAt
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:470c88501e4ff64893585861abe0b143a3ca126c3b2937fe59346dcc3f76864b"}) as unknown as TypedDocumentString<ForkCannedReportMutation, ForkCannedReportMutationVariables>;
export const ResetCannedForkDocument = new TypedDocumentString(`
    mutation ResetCannedFork($id: ID!) {
  resetCannedFork(id: $id) {
    ...ReportDefinitionFields
  }
}
    fragment ReportDefinitionFields on ReportDefinition {
  id
  name
  description
  category
  tags
  kind
  cannedKey
  cannedVersion
  ownerId
  visibility
  status
  diagnostics
  catalogVersion
  definition
  defaultFormat
  currentRevision
  lastRunAt
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:fdfc8cd8949b329a08fe31e350c065b5be2e732678ee7494e897c729e99ff96c"}) as unknown as TypedDocumentString<ResetCannedForkMutation, ResetCannedForkMutationVariables>;
export const PreviewReportDocument = new TypedDocumentString(`
    query PreviewReport($definition: ReportIRInput!, $params: JSON) {
  previewReport(definition: $definition, params: $params) {
    columns {
      id
      label
      type
      format
    }
    rows
    truncated
  }
}
    `, {"hash":"sha256:f1fe8109e84210a4d913215ba8b2bb2ad176112eef55cc94f38230d86c3c6582"}) as unknown as TypedDocumentString<PreviewReportQuery, PreviewReportQueryVariables>;
export const ReportRunsTableDocument = new TypedDocumentString(`
    query ReportRunsTable($input: DataTableConnectionInput!, $filter: ReportRunsFilterInput) {
  reportRuns(input: $input, filter: $filter) {
    edges {
      node {
        ...ReportRunFields
      }
      cursor
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment ReportRunFields on ReportRun {
  id
  definitionId
  revisionId
  cannedKey
  cannedVersion
  requestedById
  trigger
  params
  format
  status
  rowCount
  byteSize
  durationMs
  truncated
  error {
    code
    message
    detail
  }
  artifactExpiresAt
  cacheHit
  queuedAt
  startedAt
  completedAt
  version
  createdAt
}`, {"hash":"sha256:bcffde0e18264194687cba74abd8e8337428b2ae05d262ae1570c34e73205f69"}) as unknown as TypedDocumentString<ReportRunsTableQuery, ReportRunsTableQueryVariables>;
export const ReportRunByIdDocument = new TypedDocumentString(`
    query ReportRunById($id: ID!) {
  reportRun(id: $id) {
    ...ReportRunFields
  }
}
    fragment ReportRunFields on ReportRun {
  id
  definitionId
  revisionId
  cannedKey
  cannedVersion
  requestedById
  trigger
  params
  format
  status
  rowCount
  byteSize
  durationMs
  truncated
  error {
    code
    message
    detail
  }
  artifactExpiresAt
  cacheHit
  queuedAt
  startedAt
  completedAt
  version
  createdAt
}`, {"hash":"sha256:c99441d285d5a070ea5e519af3f18c58675a28f05533ae71510bf225280ee6a9"}) as unknown as TypedDocumentString<ReportRunByIdQuery, ReportRunByIdQueryVariables>;
export const RunReportDocument = new TypedDocumentString(`
    mutation RunReport($input: RunReportInput!) {
  runReport(input: $input) {
    ...ReportRunFields
  }
}
    fragment ReportRunFields on ReportRun {
  id
  definitionId
  revisionId
  cannedKey
  cannedVersion
  requestedById
  trigger
  params
  format
  status
  rowCount
  byteSize
  durationMs
  truncated
  error {
    code
    message
    detail
  }
  artifactExpiresAt
  cacheHit
  queuedAt
  startedAt
  completedAt
  version
  createdAt
}`, {"hash":"sha256:e4af42e2da79707541fc29f6455a42c80767116bdd1bee08dfc63146f32325fb"}) as unknown as TypedDocumentString<RunReportMutation, RunReportMutationVariables>;
export const CancelReportRunDocument = new TypedDocumentString(`
    mutation CancelReportRun($id: ID!) {
  cancelReportRun(id: $id) {
    ...ReportRunFields
  }
}
    fragment ReportRunFields on ReportRun {
  id
  definitionId
  revisionId
  cannedKey
  cannedVersion
  requestedById
  trigger
  params
  format
  status
  rowCount
  byteSize
  durationMs
  truncated
  error {
    code
    message
    detail
  }
  artifactExpiresAt
  cacheHit
  queuedAt
  startedAt
  completedAt
  version
  createdAt
}`, {"hash":"sha256:37a51fbc29f91bea2054db71bd1221a29bc4c38c1a8c0d7dde22c4f97fd7e9e4"}) as unknown as TypedDocumentString<CancelReportRunMutation, CancelReportRunMutationVariables>;
export const ReportSchedulesDocument = new TypedDocumentString(`
    query ReportSchedules($definitionId: ID) {
  reportSchedules(definitionId: $definitionId) {
    ...ReportScheduleFields
  }
}
    fragment ReportScheduleFields on ReportSchedule {
  id
  definitionId
  cronExpression
  timezone
  formats
  emailRecipients
  emailAttach
  notifyUserIds
  enabled
  runAsId
  lastRunId
  nextRunAt
  consecutiveFailures
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:fb0abce2c56d49308309eb048fcef0541109551d0f8cca54338715dd93a42d09"}) as unknown as TypedDocumentString<ReportSchedulesQuery, ReportSchedulesQueryVariables>;
export const CreateReportScheduleDocument = new TypedDocumentString(`
    mutation CreateReportSchedule($input: CreateReportScheduleInput!) {
  createReportSchedule(input: $input) {
    ...ReportScheduleFields
  }
}
    fragment ReportScheduleFields on ReportSchedule {
  id
  definitionId
  cronExpression
  timezone
  formats
  emailRecipients
  emailAttach
  notifyUserIds
  enabled
  runAsId
  lastRunId
  nextRunAt
  consecutiveFailures
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:ee897619e4e2b1caf4f1264ade16ce775982cf9c5cc88dc09ad205f30ccdd3b0"}) as unknown as TypedDocumentString<CreateReportScheduleMutation, CreateReportScheduleMutationVariables>;
export const UpdateReportScheduleDocument = new TypedDocumentString(`
    mutation UpdateReportSchedule($input: UpdateReportScheduleInput!) {
  updateReportSchedule(input: $input) {
    ...ReportScheduleFields
  }
}
    fragment ReportScheduleFields on ReportSchedule {
  id
  definitionId
  cronExpression
  timezone
  formats
  emailRecipients
  emailAttach
  notifyUserIds
  enabled
  runAsId
  lastRunId
  nextRunAt
  consecutiveFailures
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:a9575950d10c3f1a2351f3eaa0f4680bedbbbea15e4f29b1518f26542000f8e8"}) as unknown as TypedDocumentString<UpdateReportScheduleMutation, UpdateReportScheduleMutationVariables>;
export const DeleteReportScheduleDocument = new TypedDocumentString(`
    mutation DeleteReportSchedule($id: ID!) {
  deleteReportSchedule(id: $id)
}
    `, {"hash":"sha256:dfe8e966e00cda20ec071774d07a7c9d7f28d22648db9faa0ece2d7e233c2bf6"}) as unknown as TypedDocumentString<DeleteReportScheduleMutation, DeleteReportScheduleMutationVariables>;
export const RoleTableDocument = new TypedDocumentString(`
    query RoleTable($input: DataTableConnectionInput!) {
  roles(input: $input) {
    edges {
      node {
        ...RoleTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment RoleTableRowFields on Role {
  id
  businessUnitId
  organizationId
  name
  description
  coreResponsibility
  parentRoleIds
  maxSensitivity
  isSystem
  createdBy
  createdAt
  updatedAt
}`, {"hash":"sha256:2e3b4362769ce92ff4b9c8280388ed1d9d5a89d7fde4ced15da8a869256043e6"}) as unknown as TypedDocumentString<RoleTableQuery, RoleTableQueryVariables>;
export const ScimGroupRoleMappingsTableDocument = new TypedDocumentString(`
    query SCIMGroupRoleMappingsTable($input: DataTableConnectionInput!, $directoryId: ID!) {
  scimGroupRoleMappings(input: $input, directoryId: $directoryId) {
    edges {
      node {
        ...SCIMGroupRoleMappingTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment SCIMGroupRoleMappingTableRowFields on SCIMGroupRoleMapping {
  id
  directoryId
  externalGroupId
  displayName
  roleId
  version
  role {
    id
    name
  }
}`, {"hash":"sha256:6bf4bfdef8339181a130d96349331780bb6f19c451290ee16715438f8d96651f"}) as unknown as TypedDocumentString<ScimGroupRoleMappingsTableQuery, ScimGroupRoleMappingsTableQueryVariables>;
export const SelectOptionsDocument = new TypedDocumentString(`
    query SelectOptions($input: SelectOptionsInput!) {
  selectOptions(input: $input) {
    edges {
      node {
        id
        label
        description
        meta
      }
      cursor
    }
    pageInfo {
      hasNextPage
      endCursor
    }
    totalCount
  }
}
    `, {"hash":"sha256:61baa26c739e995aee3b16a4b9f4b584b628598c5e46d1f3886624091f1c12f2"}) as unknown as TypedDocumentString<SelectOptionsQuery, SelectOptionsQueryVariables>;
export const ServiceFailureReasonCodeTableDocument = new TypedDocumentString(`
    query ServiceFailureReasonCodeTable($input: DataTableConnectionInput!) {
  serviceFailureReasonCodes(input: $input) {
    edges {
      node {
        ...ServiceFailureReasonCodeTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment ServiceFailureReasonCodeTableRowFields on ServiceFailureReasonCode {
  id
  businessUnitId
  organizationId
  code
  label
  description
  category
  appliesTo
  defaultStatusCode
  defaultReasonCode
  defaultExceptionCode
  defaultNote
  active
  sortOrder
  externalMap
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:d52e5ebcf43703622505501e1bed670df8b7524dd05f1110417eed7d958323f9"}) as unknown as TypedDocumentString<ServiceFailureReasonCodeTableQuery, ServiceFailureReasonCodeTableQueryVariables>;
export const ServiceFailureTableDocument = new TypedDocumentString(`
    query ServiceFailureTable($input: DataTableConnectionInput!, $shipmentId: ID) {
  serviceFailures(input: $input, shipmentId: $shipmentId) {
    edges {
      node {
        ...ServiceFailureTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment ServiceFailureTableRowFields on ServiceFailure {
  id
  shipmentId
  number
  type
  source
  status
  stopType
  stopId
  scheduledCutoff
  actualArrival
  gracePeriodMinutes
  lateMinutes
  reasonCodeId
  notes
  detectedAt
  version
  shipment {
    id
    proNumber
    bol
  }
  stop {
    id
    type
    sequence
    locationId
    location {
      id
      name
      code
      city
      state {
        abbreviation
      }
    }
  }
  reasonCode {
    id
    code
    label
  }
}`, {"hash":"sha256:0a6e39abc0502b67c83033117053d0368665a3668d6f19166a8be8382bc1eb25"}) as unknown as TypedDocumentString<ServiceFailureTableQuery, ServiceFailureTableQueryVariables>;
export const ServiceTypeTableDocument = new TypedDocumentString(`
    query ServiceTypeTable($input: DataTableConnectionInput!) {
  serviceTypes(input: $input) {
    edges {
      node {
        ...ServiceTypeTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment ServiceTypeTableRowFields on ServiceType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  color
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:a4a14c4d0bc8254c150ca972341b60d1cddc3863437ed289e7c90dbacd9cb498"}) as unknown as TypedDocumentString<ServiceTypeTableQuery, ServiceTypeTableQueryVariables>;
export const ShipmentTypeTableDocument = new TypedDocumentString(`
    query ShipmentTypeTable($input: DataTableConnectionInput!) {
  shipmentTypes(input: $input) {
    edges {
      node {
        ...ShipmentTypeTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment ShipmentTypeTableRowFields on ShipmentType {
  id
  businessUnitId
  organizationId
  status
  code
  description
  color
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:425dfb86e8a1bb1b30377d884a6d93a1cf2cea2889aee419f9de30498c6e8522"}) as unknown as TypedDocumentString<ShipmentTypeTableQuery, ShipmentTypeTableQueryVariables>;
export const ShipmentCommandCenterTableDocument = new TypedDocumentString(`
    query ShipmentCommandCenterTable($input: ShipmentsInput!) {
  shipments(input: $input) {
    edges {
      node {
        ...ShipmentFields
      }
    }
    totalCount
    pageInfo {
      ...ShipmentPageInfoFields
    }
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}
fragment ShipmentPageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:4d28a793afb960f68dd010b19099e1f7bc0dfd278fb78ab01139d854554e379b"}) as unknown as TypedDocumentString<ShipmentCommandCenterTableQuery, ShipmentCommandCenterTableQueryVariables>;
export const ShipmentDetailDocument = new TypedDocumentString(`
    query ShipmentDetail($id: ID!, $expandShipmentDetails: Boolean = true) {
  shipment(id: $id, expandShipmentDetails: $expandShipmentDetails) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}`, {"hash":"sha256:df20d9c13e8b8d79f81f2fe031f751730fbcb1c787ba643b75c2f331104a5750"}) as unknown as TypedDocumentString<ShipmentDetailQuery, ShipmentDetailQueryVariables>;
export const ShipmentSavedViewCountsDocument = new TypedDocumentString(`
    query ShipmentSavedViewCounts($timezone: String!) {
  shipmentAnalytics(input: { include: "savedViewCounts", timezone: $timezone }) {
    page
    savedViewCounts {
      all
      transit
      atRisk
      unassigned
      deliveringToday
    }
  }
}
    `, {"hash":"sha256:cbed3f0cc310a0a4c3435b533a963c297ad2bad4a07174563944705242d2d168"}) as unknown as TypedDocumentString<ShipmentSavedViewCountsQuery, ShipmentSavedViewCountsQueryVariables>;
export const ShipmentPageAnalyticsDocument = new TypedDocumentString(`
    query ShipmentPageAnalytics($input: ShipmentAnalyticsInput!) {
  shipmentAnalytics(input: $input) {
    page
    savedViewCounts {
      all
      transit
      atRisk
      unassigned
      deliveringToday
    }
    activeShipments {
      count
      changeFromYesterday
      sparkline {
        hour
        value
      }
      breakdown {
        inTransit
        atRisk
        loading
        done
      }
    }
    onTimePercent {
      percent
      onTimeCount
      totalCount
      target
      deltaPp
      sevenDayPercent
    }
    profitability {
      avgCpm
      avgMarginPct
      hasMargin
      unprofitableCount
      shipmentCount
      totalMiles
    }
    revenueToday {
      total
      sparkline {
        hour
        value
      }
      deltaPct
      rpm
    }
    emptyMilePercent {
      percent
      emptyMiles
      totalMiles
      deltaPp
    }
    atRisk {
      count
      delta
      etaSlip
      weather
      reefer
    }
    unassigned {
      count
      delta
      revenueWaiting
    }
    readyToDispatch {
      count
      delta
      unassigned
      driverReady
    }
    detentionWatchlist {
      items {
        shipmentId
        customer
        dwellLabel
        tone
      }
    }
    customerMix {
      windowDays
      entries {
        customerId
        name
        revenue
        share
        loads
        trend
      }
    }
    tomorrowsPickups {
      date
      pickups {
        shipmentId
        proNumber
        pickupWindowStart
        customer
        origin
        destination
        driver
        status
      }
    }
    laneHeatmap {
      windowDays
      cells {
        origin
        destination
        count
      }
      total
    }
  }
}
    `, {"hash":"sha256:ad48e5077b2ccc6fd13488ff0477d404b19f9a4067a6d2dbc5451ec44869443e"}) as unknown as TypedDocumentString<ShipmentPageAnalyticsQuery, ShipmentPageAnalyticsQueryVariables>;
export const ShipmentTomorrowsPickupsDocument = new TypedDocumentString(`
    query ShipmentTomorrowsPickups($limit: Int, $offset: Int, $timezone: String) {
  shipmentAnalytics(
    input: {
      include: "tomorrowsPickups"
      limit: $limit
      offset: $offset
      timezone: $timezone
    }
  ) {
    page
    tomorrowsPickups {
      date
      pickups {
        shipmentId
        proNumber
        pickupWindowStart
        customer
        origin
        destination
        driver
        status
      }
    }
  }
}
    `, {"hash":"sha256:4efe02e85e165ab339b90c81ea8d05dad114942c74d9333034f58a4e6a609ee4"}) as unknown as TypedDocumentString<ShipmentTomorrowsPickupsQuery, ShipmentTomorrowsPickupsQueryVariables>;
export const UnassignedShipmentsDocument = new TypedDocumentString(`
    query UnassignedShipments($first: Int!, $after: String) {
  unassignedShipments(first: $first, after: $after) {
    edges {
      node {
        ...ShipmentFields
      }
    }
    totalCount
    pageInfo {
      ...ShipmentPageInfoFields
    }
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}
fragment ShipmentPageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:23ad07cfa42399cad844ccc2f57f993ffaa92d817a2b839116236fb8aeb1a388"}) as unknown as TypedDocumentString<UnassignedShipmentsQuery, UnassignedShipmentsQueryVariables>;
export const ExceptionShipmentsDocument = new TypedDocumentString(`
    query ExceptionShipments($input: ShipmentsInput!) {
  shipments(input: $input) {
    edges {
      node {
        ...ShipmentFields
      }
    }
    totalCount
    pageInfo {
      ...ShipmentPageInfoFields
    }
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}
fragment ShipmentPageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:dee39a3d7c719d04e89edbb797b19d1e25ab78b29f4f5c0be7126011eabb80b7"}) as unknown as TypedDocumentString<ExceptionShipmentsQuery, ExceptionShipmentsQueryVariables>;
export const MapShipmentsDocument = new TypedDocumentString(`
    query MapShipments($input: ShipmentsInput!) {
  shipments(input: $input) {
    edges {
      node {
        ...ShipmentFields
      }
    }
    totalCount
    pageInfo {
      ...ShipmentPageInfoFields
    }
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}
fragment ShipmentPageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:7db4c4d1b677282ede5ee206ac55d615348347ad352ee1f3c90a25d0bb9a36e5"}) as unknown as TypedDocumentString<MapShipmentsQuery, MapShipmentsQueryVariables>;
export const ShipmentCommentsDocument = new TypedDocumentString(`
    query ShipmentComments($shipmentId: ID!, $first: Int!, $after: String) {
  shipmentComments(shipmentId: $shipmentId, first: $first, after: $after) {
    edges {
      node {
        ...ShipmentCommentFields
      }
    }
    totalCount
    pageInfo {
      ...ShipmentPageInfoFields
    }
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentPageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment ShipmentCommentMentionFields on ShipmentCommentMention {
  id
  commentId
  mentionedUserId
  organizationId
  businessUnitId
  shipmentId
  createdAt
  mentionedUser {
    ...ShipmentUserFields
  }
}
fragment ShipmentCommentFields on ShipmentComment {
  id
  businessUnitId
  organizationId
  shipmentId
  userId
  comment
  type
  visibility
  priority
  source
  metadata
  editedAt
  version
  createdAt
  updatedAt
  mentionedUserIds
  user {
    ...ShipmentUserFields
  }
  mentionedUsers {
    ...ShipmentCommentMentionFields
  }
}`, {"hash":"sha256:cbec14ccc64595c5a8c99e7ff5824e326ff6933a0cf76f4277a1fc9abe0b4a23"}) as unknown as TypedDocumentString<ShipmentCommentsQuery, ShipmentCommentsQueryVariables>;
export const ShipmentCommentCountDocument = new TypedDocumentString(`
    query ShipmentCommentCount($shipmentId: ID!) {
  shipmentCommentCount(shipmentId: $shipmentId) {
    count
  }
}
    `, {"hash":"sha256:1f62df3579f042a9c8914aa2b124bb976b08c30fdb27dc1fa25926487e7d877e"}) as unknown as TypedDocumentString<ShipmentCommentCountQuery, ShipmentCommentCountQueryVariables>;
export const ShipmentEventsDocument = new TypedDocumentString(`
    query ShipmentEvents($input: ShipmentEventsInput!) {
  shipmentEvents(input: $input) {
    ...ShipmentEventFields
  }
}
    fragment ShipmentEventFields on ShipmentEvent {
  id
  organizationId
  businessUnitId
  shipmentId
  moveId
  stopId
  assignmentId
  commentId
  holdId
  type
  severity
  actorType
  actorId
  actorLabel
  summary
  proNumber
  previousStatus
  newStatus
  reason
  previousOwnerId
  newOwnerId
  primaryWorkerId
  secondaryWorkerId
  tractorId
  trailerId
  driverName
  holdType
  holdSeverity
  holdSource
  commentBody
  commentType
  commentVisibility
  commentPriority
  mentionedUserIds
  metadata
  occurredAt
  correlationId
  actor {
    id
    name
    emailAddress
    profilePicUrl
    thumbnailUrl
  }
  shipment {
    id
    proNumber
  }
}`, {"hash":"sha256:4bcc9366701fa873e374aa35b7744db7d18a26afa558617c8c443f780d1dabb5"}) as unknown as TypedDocumentString<ShipmentEventsQuery, ShipmentEventsQueryVariables>;
export const ShipmentBillingReadinessDocument = new TypedDocumentString(`
    query ShipmentBillingReadiness($shipmentId: ID!) {
  shipmentBillingReadiness(shipmentId: $shipmentId) {
    shipmentId
    shipmentStatus
    policy {
      shipmentBillingRequirementEnforcement
      rateValidationEnforcement
      billingExceptionDisposition
      notifyOnBillingExceptions
      readyToBillAssignmentMode
      billingQueueTransferMode
    }
    requirements {
      documentTypeId
      documentTypeCode
      documentTypeName
      satisfied
      documentCount
      documentIds
    }
    missingRequirements {
      documentTypeId
      documentTypeCode
      documentTypeName
      satisfied
      documentCount
      documentIds
    }
    validationFailures {
      field
      code
      message
    }
    warnings {
      code
      message
      context {
        documentTypeId
        documentTypeCode
        documentTypeName
        documentCount
        requirementCount
        missingRequirementCount
        serviceFailureIds
        unresolvedCount
      }
    }
    serviceFailureContext {
      hasUnresolved
      unresolvedCount
      serviceFailureIds
    }
    canMarkReadyToInvoice
    shouldAutoMarkReadyToInvoice
    shouldAutoTransferToBilling
  }
}
    `, {"hash":"sha256:e75cb6d00ed67d58a2fe75606c9449dd1e55a2f61b902db0aedf9941ee01a383"}) as unknown as TypedDocumentString<ShipmentBillingReadinessQuery, ShipmentBillingReadinessQueryVariables>;
export const ShipmentUiPolicyDocument = new TypedDocumentString(`
    query ShipmentUIPolicy {
  shipmentUIPolicy {
    allowMoveRemovals
    checkForDuplicateBols
    checkHazmatSegregation
    maxShipmentWeightLimit
  }
}
    `, {"hash":"sha256:fb8f36ec1209a13f87cfda6fda2017e70e17c899f6b2f5b22fc8b4fe8bdbace7"}) as unknown as TypedDocumentString<ShipmentUiPolicyQuery, ShipmentUiPolicyQueryVariables>;
export const ShipmentPreviousRatesDocument = new TypedDocumentString(`
    query ShipmentPreviousRates($input: ShipmentPreviousRatesInput!) {
  shipmentPreviousRates(input: $input) {
    items {
      shipmentId
      proNumber
      customerId
      serviceTypeId
      shipmentTypeId
      formulaTemplateId
      freightChargeAmount
      otherChargeAmount
      totalChargeAmount
      ratingUnit
      pieces
      weight
      createdAt
    }
    total
  }
}
    `, {"hash":"sha256:fb9ce636f0cfa91106dfcc559e31eb59e1e6cfa4d229668dbc2208e7a3730f9b"}) as unknown as TypedDocumentString<ShipmentPreviousRatesQuery, ShipmentPreviousRatesQueryVariables>;
export const CreateShipmentDocument = new TypedDocumentString(`
    mutation CreateShipment($input: ShipmentInput!) {
  createShipment(input: $input) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}`, {"hash":"sha256:326291ba8669217819cfd1c4af15079ce55b7da617c47bc50de2a9c369f71385"}) as unknown as TypedDocumentString<CreateShipmentMutation, CreateShipmentMutationVariables>;
export const UpdateShipmentDocument = new TypedDocumentString(`
    mutation UpdateShipment($id: ID!, $input: ShipmentInput!) {
  updateShipment(id: $id, input: $input) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}`, {"hash":"sha256:18c59e6b19b7aac3db3baf9965db1d32bf5f0b18de0f44697db50572377188d3"}) as unknown as TypedDocumentString<UpdateShipmentMutation, UpdateShipmentMutationVariables>;
export const CancelShipmentDocument = new TypedDocumentString(`
    mutation CancelShipment($id: ID!, $input: ShipmentCancelInput) {
  cancelShipment(id: $id, input: $input) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}`, {"hash":"sha256:c4aab7651a8aada657e62050b5050a42b0147ea3f5a0857ccf4630718d549e89"}) as unknown as TypedDocumentString<CancelShipmentMutation, CancelShipmentMutationVariables>;
export const UncancelShipmentDocument = new TypedDocumentString(`
    mutation UncancelShipment($id: ID!) {
  uncancelShipment(id: $id) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}`, {"hash":"sha256:6344073680c40eb23265bad1341c0ea6dfafd5663ec4449e2309c5fe14b6d754"}) as unknown as TypedDocumentString<UncancelShipmentMutation, UncancelShipmentMutationVariables>;
export const DuplicateShipmentDocument = new TypedDocumentString(`
    mutation DuplicateShipment($input: ShipmentDuplicateInput!) {
  duplicateShipment(input: $input) {
    workflowId
    runId
    taskQueue
    status
    submittedAt
  }
}
    `, {"hash":"sha256:0dcc6ec862a4ef66a9e7137e45548bb355204f1bc766d172a035b5e537298ecf"}) as unknown as TypedDocumentString<DuplicateShipmentMutation, DuplicateShipmentMutationVariables>;
export const TransferShipmentOwnershipDocument = new TypedDocumentString(`
    mutation TransferShipmentOwnership($id: ID!, $input: ShipmentTransferOwnershipInput!) {
  transferShipmentOwnership(id: $id, input: $input) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
  status
  locationCategoryId
  stateId
  addressLine1
  addressLine2
  city
  postalCode
  longitude
  latitude
}
fragment ShipmentWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment ShipmentTractorFields on Tractor {
  id
  code
}
fragment ShipmentTrailerFields on Trailer {
  id
  code
  equipmentTypeId
}
fragment ShipmentAssignmentFields on ShipmentAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  primaryWorkerId
  tractorId
  trailerId
  secondaryWorkerId
  status
  archivedAt
  version
  createdAt
  updatedAt
  tractor {
    ...ShipmentTractorFields
  }
  trailer {
    ...ShipmentTrailerFields
  }
  primaryWorker {
    ...ShipmentWorkerFields
  }
  secondaryWorker {
    ...ShipmentWorkerFields
  }
}
fragment ShipmentStopFields on ShipmentStop {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  locationId
  status
  type
  scheduleType
  sequence
  pieces
  weight
  scheduledWindowStart
  scheduledWindowEnd
  actualArrival
  actualDeparture
  countLateOverride
  countDetentionOverride
  addressLine
  version
  createdAt
  updatedAt
  location {
    ...ShipmentLocationFields
  }
}
fragment ShipmentMoveFields on ShipmentMove {
  id
  businessUnitId
  organizationId
  shipmentId
  status
  loaded
  sequence
  distance
  distanceSource
  distanceProvider
  distanceCalculatedAt
  distanceRouteSignature
  distanceDataVersion
  distanceRoutingType
  distanceUnits
  distanceMetadata
  version
  createdAt
  updatedAt
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
}
fragment ShipmentAdditionalChargeFields on ShipmentAdditionalCharge {
  id
  businessUnitId
  organizationId
  shipmentId
  accessorialChargeId
  isSystemGenerated
  method
  amount
  unit
  fuelSurchargeProgramId
  fuelSurchargeDetail
  version
  createdAt
  updatedAt
  accessorialCharge {
    id
    businessUnitId
    organizationId
    code
    description
    status
    method
    rateUnit
    amount
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentCommodityFields on ShipmentCommodity {
  id
  businessUnitId
  organizationId
  shipmentId
  commodityId
  pieces
  weight
  version
  createdAt
  updatedAt
  commodity {
    id
    businessUnitId
    organizationId
    hazardousMaterialId
    status
    name
    description
    minTemperature
    maxTemperature
    weightPerUnit
    linearFeetPerUnit
    maxQuantityPerShipment
    freightClass
    loadingInstructions
    stackable
    fragile
    version
    createdAt
    updatedAt
  }
}
fragment ShipmentRatingDetailFields on ShipmentRatingDetail {
  formulaTemplateId
  formulaTemplateName
  expression
  resolvedVariables
  result
  ratedAt
}
fragment ShipmentFields on Shipment {
  id
  businessUnitId
  organizationId
  sourceDocumentId
  serviceTypeId
  shipmentTypeId
  customerId
  tractorTypeId
  trailerTypeId
  ownerId
  enteredById
  canceledById
  formulaTemplateId
  consolidationGroupId
  orderId
  orderNumber
  orderStatus
  status
  tenderStatus
  entryMethod
  proNumber
  bol
  cancelReason
  otherChargeAmount
  freightChargeAmount
  baseRate
  totalChargeAmount
  pieces
  weight
  temperatureMin
  temperatureMax
  actualDeliveryDate
  actualShipDate
  canceledAt
  billingTransferStatus
  transferredToBillingAt
  markedReadyToBillAt
  billedAt
  ratingUnit
  fuelSurchargeLocked
  profitabilityEstimate {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    costPerMile
    estimatedCost
    profit
    marginPercent
    breakEvenRpm
    targetMarginPercent
    missingDistance
  }
  ratingDetail {
    ...ShipmentRatingDetailFields
  }
  version
  createdAt
  updatedAt
  moves {
    ...ShipmentMoveFields
  }
  additionalCharges {
    ...ShipmentAdditionalChargeFields
  }
  commodities {
    ...ShipmentCommodityFields
  }
  customer {
    id
    businessUnitId
    organizationId
    stateId
    status
    code
    name
    addressLine1
    addressLine2
    city
    postalCode
    isGeocoded
    longitude
    latitude
    placeId
    externalId
    allowConsolidation
    exclusiveConsolidation
    consolidationPriority
    version
    createdAt
    updatedAt
  }
  owner {
    ...ShipmentUserFields
  }
  formulaTemplate {
    id
    organizationId
    businessUnitId
    name
    description
    type
    expression
    status
    schemaId
    variableDefinitions {
      name
      type
      description
      required
      defaultValue
      source
    }
    metadata
    version
    sourceTemplateId
    sourceVersionNumber
    currentVersionNumber
    createdAt
    updatedAt
  }
}`, {"hash":"sha256:ee216ef9ed7f98ccdceb2c1293cebe603782238fe351089d291d1ebeed6f1deb"}) as unknown as TypedDocumentString<TransferShipmentOwnershipMutation, TransferShipmentOwnershipMutationVariables>;
export const TransferShipmentToBillingDocument = new TypedDocumentString(`
    mutation TransferShipmentToBilling($input: ShipmentTransferToBillingInput!) {
  transferShipmentToBilling(input: $input) {
    id
    organizationId
    businessUnitId
    shipmentId
    assignedBillerId
    number
    status
    billType
    exceptionReasonCode
    reviewNotes
    exceptionNotes
    reviewStartedAt
    reviewCompletedAt
    canceledById
    canceledAt
    cancelReason
    isAdjustmentOrigin
    sourceInvoiceId
    sourceInvoiceAdjustmentId
    sourceCreditMemoInvoiceId
    correctionGroupId
    rebillStrategy
    requiresReplacementReview
    rerateVariancePercent
    adjustmentContext
    version
    createdAt
    updatedAt
  }
}
    `, {"hash":"sha256:7849b77f08e7c2e7cb6af2c2abbc53185d0811092df155557be1c6803b335473"}) as unknown as TypedDocumentString<TransferShipmentToBillingMutation, TransferShipmentToBillingMutationVariables>;
export const BulkTransferShipmentsToBillingDocument = new TypedDocumentString(`
    mutation BulkTransferShipmentsToBilling($input: ShipmentBulkTransferToBillingInput!) {
  bulkTransferShipmentsToBilling(input: $input) {
    results {
      shipmentId
      success
      error
    }
    totalCount
    successCount
    errorCount
  }
}
    `, {"hash":"sha256:46beae0d55af6f6abff7b4c2090ce9ae974f805e505c3ff2c60ababb2f33e0a2"}) as unknown as TypedDocumentString<BulkTransferShipmentsToBillingMutation, BulkTransferShipmentsToBillingMutationVariables>;
export const CalculateShipmentTotalsDocument = new TypedDocumentString(`
    mutation CalculateShipmentTotals($input: ShipmentInput!) {
  calculateShipmentTotals(input: $input) {
    freightChargeAmount
    otherChargeAmount
    totalChargeAmount
    fuelSurcharge {
      accessorialChargeId
      isSystemGenerated
      method
      amount
      unit
      fuelSurchargeProgramId
      fuelSurchargeDetail
    }
  }
}
    `, {"hash":"sha256:675789448d139ef11053baf07c910d999193810b6acd81ae401229c1e1753a75"}) as unknown as TypedDocumentString<CalculateShipmentTotalsMutation, CalculateShipmentTotalsMutationVariables>;
export const CalculateShipmentDistanceDocument = new TypedDocumentString(`
    mutation CalculateShipmentDistance($input: ShipmentInput!) {
  calculateShipmentDistance(input: $input) {
    shipmentId
    totalDistance
    moves {
      moveId
      moveIndex
      distance
      source
      provider
      routingType
      dataVersion
      distanceUnits
      distanceProfileId
      distanceProfileName
      warnings
      calculatedAt
    }
  }
}
    `, {"hash":"sha256:5c8612acf5d98e8e255b7ec31d1fb37d4723fe9cfed4f6e2c2c4203ba5092b5a"}) as unknown as TypedDocumentString<CalculateShipmentDistanceMutation, CalculateShipmentDistanceMutationVariables>;
export const RecalculateShipmentDistanceDocument = new TypedDocumentString(`
    mutation RecalculateShipmentDistance($shipmentId: ID!) {
  recalculateShipmentDistance(shipmentId: $shipmentId) {
    shipmentId
    totalDistance
    moves {
      moveId
      moveIndex
      distance
      source
      provider
      routingType
      dataVersion
      distanceUnits
      distanceProfileId
      distanceProfileName
      warnings
      calculatedAt
    }
  }
}
    `, {"hash":"sha256:c22b19ad3ce0ea5856e7d3b13cbf94d2b5a34aadeaf0f90cf09c5ba8f6f87c51"}) as unknown as TypedDocumentString<RecalculateShipmentDistanceMutation, RecalculateShipmentDistanceMutationVariables>;
export const CheckShipmentDuplicateBolDocument = new TypedDocumentString(`
    mutation CheckShipmentDuplicateBol($input: ShipmentDuplicateBolInput!) {
  checkShipmentDuplicateBol(input: $input) {
    valid
  }
}
    `, {"hash":"sha256:245fce8ae3f1f985031b2343ce03fa257082b6453a6b738c209e49581315c33c"}) as unknown as TypedDocumentString<CheckShipmentDuplicateBolMutation, CheckShipmentDuplicateBolMutationVariables>;
export const CheckShipmentHazmatSegregationDocument = new TypedDocumentString(`
    mutation CheckShipmentHazmatSegregation($input: ShipmentHazmatInput!) {
  checkShipmentHazmatSegregation(input: $input) {
    valid
  }
}
    `, {"hash":"sha256:94a8f251053368e89199e02110d476108016a60da4e182320c793065a99b1b7a"}) as unknown as TypedDocumentString<CheckShipmentHazmatSegregationMutation, CheckShipmentHazmatSegregationMutationVariables>;
export const CalculateShipmentLoadingOptimizationDocument = new TypedDocumentString(`
    mutation CalculateShipmentLoadingOptimization($input: ShipmentLoadingOptimizationInput!) {
  calculateShipmentLoadingOptimization(input: $input) {
    trailerLengthFeet
    totalLinearFeet
    totalWeight
    maxWeight
    linearFeetUtil
    weightUtil
    utilizationScore
    utilizationGrade
    placements {
      commodityId
      commodityName
      positionFeet
      lengthFeet
      weight
      pieces
      stackable
      fragile
      isHazmat
      hazmatClass
      minTemp
      maxTemp
      loadingInstructions
      estimatedLength
      stopNumber
    }
    hazmatZones {
      commodityAId
      commodityBId
      commodityAName
      commodityBName
      ruleName
      segregationType
      requiredDistanceFeet
      actualDistanceFeet
      satisfied
    }
    warnings {
      type
      message
      severity
      commodityIds
    }
    axleWeights {
      axle
      weight
      limit
      percentage
      compliant
    }
    recommendations {
      type
      priority
      title
      description
      impact
      commodityIds
    }
    stopDividers {
      positionFeet
      stopNumber
      label
    }
    aiAnalysis
  }
}
    `, {"hash":"sha256:23ff92adf45fa82bf4f021539ac314f2922bd091b303c771fab8c553dbfe68dc"}) as unknown as TypedDocumentString<CalculateShipmentLoadingOptimizationMutation, CalculateShipmentLoadingOptimizationMutationVariables>;
export const CreateShipmentCommentDocument = new TypedDocumentString(`
    mutation CreateShipmentComment($shipmentId: ID!, $input: ShipmentCommentInput!) {
  createShipmentComment(shipmentId: $shipmentId, input: $input) {
    ...ShipmentCommentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentCommentMentionFields on ShipmentCommentMention {
  id
  commentId
  mentionedUserId
  organizationId
  businessUnitId
  shipmentId
  createdAt
  mentionedUser {
    ...ShipmentUserFields
  }
}
fragment ShipmentCommentFields on ShipmentComment {
  id
  businessUnitId
  organizationId
  shipmentId
  userId
  comment
  type
  visibility
  priority
  source
  metadata
  editedAt
  version
  createdAt
  updatedAt
  mentionedUserIds
  user {
    ...ShipmentUserFields
  }
  mentionedUsers {
    ...ShipmentCommentMentionFields
  }
}`, {"hash":"sha256:ba68a1443da7ccf0d3e78c144632560501ed854de5f5caf63349a8748218cf6b"}) as unknown as TypedDocumentString<CreateShipmentCommentMutation, CreateShipmentCommentMutationVariables>;
export const UpdateShipmentCommentDocument = new TypedDocumentString(`
    mutation UpdateShipmentComment($shipmentId: ID!, $commentId: ID!, $input: ShipmentCommentUpdateInput!) {
  updateShipmentComment(
    shipmentId: $shipmentId
    commentId: $commentId
    input: $input
  ) {
    ...ShipmentCommentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  username
  emailAddress
  timezone
  status
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentCommentMentionFields on ShipmentCommentMention {
  id
  commentId
  mentionedUserId
  organizationId
  businessUnitId
  shipmentId
  createdAt
  mentionedUser {
    ...ShipmentUserFields
  }
}
fragment ShipmentCommentFields on ShipmentComment {
  id
  businessUnitId
  organizationId
  shipmentId
  userId
  comment
  type
  visibility
  priority
  source
  metadata
  editedAt
  version
  createdAt
  updatedAt
  mentionedUserIds
  user {
    ...ShipmentUserFields
  }
  mentionedUsers {
    ...ShipmentCommentMentionFields
  }
}`, {"hash":"sha256:968a90a90453a3933f1a6a452652e577dfe90c1974ccd83311e102b50c4e0906"}) as unknown as TypedDocumentString<UpdateShipmentCommentMutation, UpdateShipmentCommentMutationVariables>;
export const DeleteShipmentCommentDocument = new TypedDocumentString(`
    mutation DeleteShipmentComment($shipmentId: ID!, $commentId: ID!) {
  deleteShipmentComment(shipmentId: $shipmentId, commentId: $commentId)
}
    `, {"hash":"sha256:a20dcdea6225911dd4742c1e415a5f1e2b04d0111fbaf5ecbda1e8136b3dfa14"}) as unknown as TypedDocumentString<DeleteShipmentCommentMutation, DeleteShipmentCommentMutationVariables>;
export const ShipmentProfitabilityDocument = new TypedDocumentString(`
    query ShipmentProfitability($shipmentId: ID!) {
  shipmentProfitability(shipmentId: $shipmentId) {
    shipmentId
    loadedMiles
    deadheadMiles
    totalMiles
    revenue
    estimatedCost
    profit
    marginPercent
    revenuePerLoadedMile
    breakEvenRpm
    missingDistance
    breakdown {
      category
      name
      costBehavior
      ratePerMile
      amount
      effectiveSource
    }
    profile {
      totalCpm
      variableCpm
      fixedCpm
      targetMarginPercent
      includeDeadheadMiles
      asOfDate
      fuel {
        pricePerGallon
        priceDate
        fuelIndexId
        milesPerGallon
        source
      }
      glWindow {
        fromDate
        toDate
        fleetMiles
        hasPostings
      }
    }
  }
}
    `, {"hash":"sha256:ba934decc6721bef0b703376a291e500f48d887692508b928c2286896b52d8a9"}) as unknown as TypedDocumentString<ShipmentProfitabilityQuery, ShipmentProfitabilityQueryVariables>;
export const UpdateSidebarPreferencesDocument = new TypedDocumentString(`
    mutation UpdateSidebarPreferences($input: SidebarPreferencesInput!) {
  updateSidebarPreferences(input: $input) {
    schemaVersion
    version
    sections {
      key
      hidden
    }
    attentionMetrics
    quickActionIds
    activity {
      pageSize
      defaultOpen
    }
  }
}
    `, {"hash":"sha256:977fedf72d0dd1e084eb48b093203298a4b2d9513240e0149488448e349d8128"}) as unknown as TypedDocumentString<UpdateSidebarPreferencesMutation, UpdateSidebarPreferencesMutationVariables>;
export const SidebarPreferencesDocument = new TypedDocumentString(`
    query SidebarPreferences {
  sidebarPreferences {
    schemaVersion
    version
    sections {
      key
      hidden
    }
    attentionMetrics
    quickActionIds
    activity {
      pageSize
      defaultOpen
    }
  }
}
    `, {"hash":"sha256:a136ac10eb71000bcfef94663b0a9df6cba4161eb0ec120f890e3e3a06a23bdc"}) as unknown as TypedDocumentString<SidebarPreferencesQuery, SidebarPreferencesQueryVariables>;
export const SidebarCustomizationOptionsDocument = new TypedDocumentString(`
    query SidebarCustomizationOptions {
  sidebarCustomizationOptions {
    sections {
      key
      label
      hideable
    }
    attentionMetrics {
      key
      label
    }
    quickActions {
      id
      label
    }
    maxQuickActions
    activityPageSizes
  }
}
    `, {"hash":"sha256:79b6c8e2b9458d391abe3a98706dc989e567930eaf28453bd9635cf1766b276d"}) as unknown as TypedDocumentString<SidebarCustomizationOptionsQuery, SidebarCustomizationOptionsQueryVariables>;
export const StoredMileageTableDocument = new TypedDocumentString(`
    query StoredMileageTable($input: DataTableConnectionInput!) {
  storedMileages(input: $input) {
    edges {
      node {
        ...StoredMileageTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment StoredMileageStopKeyFields on StopKey {
  method
  key
  city
  state
  postalCode
  placeId
  coordinates
}
fragment StoredMileageTableRowFields on StoredMileage {
  id
  businessUnitId
  organizationId
  status
  originKey {
    ...StoredMileageStopKeyFields
  }
  destinationKey {
    ...StoredMileageStopKeyFields
  }
  intermediateKeys {
    ...StoredMileageStopKeyFields
  }
  routeSignature
  routeHash
  distance
  distanceUnits
  provider
  source
  routingType
  method
  distanceProfileId
  distanceProfileName
  hitCount
  lastCalculatedAt
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:3a50b15af9e7b4e8c9b869e8638aae181cd9169fdc472212ce4fec2d87d8c1d1"}) as unknown as TypedDocumentString<StoredMileageTableQuery, StoredMileageTableQueryVariables>;
export const TcaSubscriptionTableDocument = new TypedDocumentString(`
    query TCASubscriptionTable($input: DataTableConnectionInput!) {
  tcaSubscriptions(input: $input) {
    edges {
      node {
        ...TCASubscriptionTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment TCASubscriptionTableRowFields on TCASubscription {
  id
  organizationId
  businessUnitId
  userId
  name
  tableName
  recordId
  eventTypes
  conditions
  conditionMatch
  watchedColumns
  customTitle
  customMessage
  topic
  priority
  status
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:3b200328291e0761d53c2426b15e1ab7cfd05ae2b55be15e33b603847b38174c"}) as unknown as TypedDocumentString<TcaSubscriptionTableQuery, TcaSubscriptionTableQueryVariables>;
export const TableConfigurationTableDocument = new TypedDocumentString(`
    query TableConfigurationTable($input: DataTableConnectionInput!, $resource: String, $visibility: ConfigurationVisibility) {
  tableConfigurations(input: $input, resource: $resource, visibility: $visibility) {
    edges {
      node {
        ...TableConfigurationFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment TableConfigurationFields on TableConfiguration {
  id
  organizationId
  businessUnitId
  userId
  name
  description
  resource
  tableConfig
  visibility
  isDefault
  isOrgDefault
  version
  createdAt
  updatedAt
  user {
    id
    name
    profilePicUrl
  }
}`, {"hash":"sha256:552aa20a98ddbbc40843b9a6743488d395872e70590701a81aa5b3480cf1d7df"}) as unknown as TypedDocumentString<TableConfigurationTableQuery, TableConfigurationTableQueryVariables>;
export const DefaultTableConfigurationDocument = new TypedDocumentString(`
    query DefaultTableConfiguration($resource: String!) {
  defaultTableConfiguration(resource: $resource) {
    ...TableConfigurationFields
  }
}
    fragment TableConfigurationFields on TableConfiguration {
  id
  organizationId
  businessUnitId
  userId
  name
  description
  resource
  tableConfig
  visibility
  isDefault
  isOrgDefault
  version
  createdAt
  updatedAt
  user {
    id
    name
    profilePicUrl
  }
}`, {"hash":"sha256:44172e730a5efee17642504f34cd054b9dacd048e99f174d4678866c287d99a5"}) as unknown as TypedDocumentString<DefaultTableConfigurationQuery, DefaultTableConfigurationQueryVariables>;
export const TableConfigurationDetailDocument = new TypedDocumentString(`
    query TableConfigurationDetail($id: ID!) {
  tableConfiguration(id: $id) {
    ...TableConfigurationFields
  }
}
    fragment TableConfigurationFields on TableConfiguration {
  id
  organizationId
  businessUnitId
  userId
  name
  description
  resource
  tableConfig
  visibility
  isDefault
  isOrgDefault
  version
  createdAt
  updatedAt
  user {
    id
    name
    profilePicUrl
  }
}`, {"hash":"sha256:d3517c4942a70bd816f40494468ad1aef6af9f5b261fe2da3a1f6d7308f61b0e"}) as unknown as TypedDocumentString<TableConfigurationDetailQuery, TableConfigurationDetailQueryVariables>;
export const CreateTableConfigurationDocument = new TypedDocumentString(`
    mutation CreateTableConfiguration($input: TableConfigurationInput!) {
  createTableConfiguration(input: $input) {
    ...TableConfigurationFields
  }
}
    fragment TableConfigurationFields on TableConfiguration {
  id
  organizationId
  businessUnitId
  userId
  name
  description
  resource
  tableConfig
  visibility
  isDefault
  isOrgDefault
  version
  createdAt
  updatedAt
  user {
    id
    name
    profilePicUrl
  }
}`, {"hash":"sha256:9da2c2f361a87c5bef4f0cb0beae047cf548a74b5c389efdf63ccd4adb0e8625"}) as unknown as TypedDocumentString<CreateTableConfigurationMutation, CreateTableConfigurationMutationVariables>;
export const UpdateTableConfigurationDocument = new TypedDocumentString(`
    mutation UpdateTableConfiguration($id: ID!, $input: TableConfigurationInput!) {
  updateTableConfiguration(id: $id, input: $input) {
    ...TableConfigurationFields
  }
}
    fragment TableConfigurationFields on TableConfiguration {
  id
  organizationId
  businessUnitId
  userId
  name
  description
  resource
  tableConfig
  visibility
  isDefault
  isOrgDefault
  version
  createdAt
  updatedAt
  user {
    id
    name
    profilePicUrl
  }
}`, {"hash":"sha256:4c1284c544aa1e09687d7ae906b44f39a035adef42ec417d124be8e5fc9946eb"}) as unknown as TypedDocumentString<UpdateTableConfigurationMutation, UpdateTableConfigurationMutationVariables>;
export const DeleteTableConfigurationDocument = new TypedDocumentString(`
    mutation DeleteTableConfiguration($id: ID!) {
  deleteTableConfiguration(id: $id)
}
    `, {"hash":"sha256:5c2f8a0d5ce9cc3f5ae7702ee9785c067097d3a04de36ae8b17d43afeb5e4951"}) as unknown as TypedDocumentString<DeleteTableConfigurationMutation, DeleteTableConfigurationMutationVariables>;
export const SetDefaultTableConfigurationDocument = new TypedDocumentString(`
    mutation SetDefaultTableConfiguration($id: ID!) {
  setDefaultTableConfiguration(id: $id) {
    ...TableConfigurationFields
  }
}
    fragment TableConfigurationFields on TableConfiguration {
  id
  organizationId
  businessUnitId
  userId
  name
  description
  resource
  tableConfig
  visibility
  isDefault
  isOrgDefault
  version
  createdAt
  updatedAt
  user {
    id
    name
    profilePicUrl
  }
}`, {"hash":"sha256:d186244dea4960f7116982c1281aa7b3daec229ba06143417bbd3064d746340b"}) as unknown as TypedDocumentString<SetDefaultTableConfigurationMutation, SetDefaultTableConfigurationMutationVariables>;
export const SetOrgDefaultTableConfigurationDocument = new TypedDocumentString(`
    mutation SetOrgDefaultTableConfiguration($id: ID!, $enabled: Boolean!) {
  setOrgDefaultTableConfiguration(id: $id, enabled: $enabled) {
    ...TableConfigurationFields
  }
}
    fragment TableConfigurationFields on TableConfiguration {
  id
  organizationId
  businessUnitId
  userId
  name
  description
  resource
  tableConfig
  visibility
  isDefault
  isOrgDefault
  version
  createdAt
  updatedAt
  user {
    id
    name
    profilePicUrl
  }
}`, {"hash":"sha256:739270b19dc50a3d42f57caf1791f4dc556c4b51547b84eef44000bceb25aadc"}) as unknown as TypedDocumentString<SetOrgDefaultTableConfigurationMutation, SetOrgDefaultTableConfigurationMutationVariables>;
export const UserTableDocument = new TypedDocumentString(`
    query UserTable($input: DataTableConnectionInput!) {
  users(input: $input) {
    edges {
      node {
        ...UserTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...DataTablePageInfoFields
    }
  }
}
    fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
fragment UserTableRowFields on User {
  id
  businessUnitId
  currentOrganizationId
  status
  name
  username
  emailAddress
  profilePicUrl
  thumbnailUrl
  timezone
  isLocked
  mustChangePassword
  version
  lastLoginAt
  createdAt
  updatedAt
}`, {"hash":"sha256:49712b2a1329e674ea6b11f1294601eb50deb21d53e79540876b0455d897db5d"}) as unknown as TypedDocumentString<UserTableQuery, UserTableQueryVariables>;
export const WorkerTableDocument = new TypedDocumentString(`
    query WorkerTable($input: DataTableConnectionInput!) {
  workers(input: $input) {
    edges {
      node {
        ...WorkerTableRowFields
      }
    }
    totalCount
    pageInfo {
      ...WorkerDataTablePageInfoFields
    }
  }
}
    fragment WorkerFleetCodeFields on FleetCode {
  id
  code
  color
}
fragment WorkerUsStateFields on UsState {
  id
  name
  abbreviation
}
fragment WorkerProfileTableFields on WorkerProfile {
  id
  workerId
  businessUnitId
  organizationId
  licenseStateId
  dob
  licenseNumber
  cdlClass
  cdlRestrictions
  endorsement
  hazmatExpiry
  licenseExpiry
  medicalCardExpiry
  medicalExaminerName
  medicalExaminerNpi
  twicCardNumber
  twicExpiry
  hireDate
  terminationDate
  physicalDueDate
  mvrDueDate
  complianceStatus
  isQualified
  disqualificationReason
  lastComplianceCheck
  lastMvrCheck
  lastDrugTest
  eldExempt
  shortHaulExempt
  version
  createdAt
  updatedAt
  licenseState {
    ...WorkerUsStateFields
  }
}
fragment WorkerTableRowFields on Worker {
  id
  businessUnitId
  organizationId
  stateId
  fleetCodeId
  managerId
  status
  type
  driverType
  profilePicUrl
  firstName
  lastName
  wholeName
  addressLine1
  addressLine2
  city
  postalCode
  email
  phoneNumber
  emergencyContactName
  emergencyContactPhone
  externalId
  assignmentBlocked
  gender
  canBeAssigned
  availableForDispatch
  version
  createdAt
  updatedAt
  customFields
  fleetCode {
    ...WorkerFleetCodeFields
  }
  state {
    ...WorkerUsStateFields
  }
  profile {
    ...WorkerProfileTableFields
  }
}
fragment WorkerDataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:214ae22d2686f4434a662bf4f790087d6d97d6d05302366ae7b48e470a014f18"}) as unknown as TypedDocumentString<WorkerTableQuery, WorkerTableQueryVariables>;
export const WorkerPtoTableDocument = new TypedDocumentString(`
    query WorkerPtoTable($input: WorkerPTOEntriesInput!) {
  workerPTOEntries(input: $input) {
    edges {
      node {
        ...WorkerPtoRowFields
      }
    }
    totalCount
    pageInfo {
      ...WorkerDataTablePageInfoFields
    }
  }
}
    fragment WorkerPtoWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment WorkerPtoRowFields on WorkerPTO {
  id
  workerId
  organizationId
  businessUnitId
  approverId
  rejectorId
  status
  type
  startDate
  endDate
  reason
  version
  createdAt
  updatedAt
  worker {
    ...WorkerPtoWorkerFields
  }
}
fragment WorkerDataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:897e7d8ddc98ce81d6a4959b8989e4a754f6a5cfaa4b8d0c265e9658673e4ec5"}) as unknown as TypedDocumentString<WorkerPtoTableQuery, WorkerPtoTableQueryVariables>;
export const UpcomingWorkerPtoDocument = new TypedDocumentString(`
    query UpcomingWorkerPto($input: UpcomingWorkerPTOInput!) {
  upcomingWorkerPTO(input: $input) {
    edges {
      node {
        ...WorkerPtoRowFields
      }
    }
    totalCount
    pageInfo {
      ...WorkerDataTablePageInfoFields
    }
  }
}
    fragment WorkerPtoWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment WorkerPtoRowFields on WorkerPTO {
  id
  workerId
  organizationId
  businessUnitId
  approverId
  rejectorId
  status
  type
  startDate
  endDate
  reason
  version
  createdAt
  updatedAt
  worker {
    ...WorkerPtoWorkerFields
  }
}
fragment WorkerDataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:06f4895f934fa8314b8c729fed0f509fa3f4a6a302a5c87cfb3b30cc4b5361e2"}) as unknown as TypedDocumentString<UpcomingWorkerPtoQuery, UpcomingWorkerPtoQueryVariables>;
export const WorkerPtoChartDataDocument = new TypedDocumentString(`
    query WorkerPtoChartData($input: WorkerPTOChartInput!) {
  workerPTOChartData(input: $input) {
    date
    vacation
    sick
    holiday
    bereavement
    maternity
    paternity
    personal
    workers
  }
}
    `, {"hash":"sha256:bde52602279b3f61aafc2fced2549700ae1249c36b37078532b4d1ead6266aad"}) as unknown as TypedDocumentString<WorkerPtoChartDataQuery, WorkerPtoChartDataQueryVariables>;
export const PatchWorkerDocument = new TypedDocumentString(`
    mutation PatchWorker($id: ID!, $input: WorkerPatchInput!) {
  patchWorker(id: $id, input: $input) {
    ...WorkerTableRowFields
  }
}
    fragment WorkerFleetCodeFields on FleetCode {
  id
  code
  color
}
fragment WorkerUsStateFields on UsState {
  id
  name
  abbreviation
}
fragment WorkerProfileTableFields on WorkerProfile {
  id
  workerId
  businessUnitId
  organizationId
  licenseStateId
  dob
  licenseNumber
  cdlClass
  cdlRestrictions
  endorsement
  hazmatExpiry
  licenseExpiry
  medicalCardExpiry
  medicalExaminerName
  medicalExaminerNpi
  twicCardNumber
  twicExpiry
  hireDate
  terminationDate
  physicalDueDate
  mvrDueDate
  complianceStatus
  isQualified
  disqualificationReason
  lastComplianceCheck
  lastMvrCheck
  lastDrugTest
  eldExempt
  shortHaulExempt
  version
  createdAt
  updatedAt
  licenseState {
    ...WorkerUsStateFields
  }
}
fragment WorkerTableRowFields on Worker {
  id
  businessUnitId
  organizationId
  stateId
  fleetCodeId
  managerId
  status
  type
  driverType
  profilePicUrl
  firstName
  lastName
  wholeName
  addressLine1
  addressLine2
  city
  postalCode
  email
  phoneNumber
  emergencyContactName
  emergencyContactPhone
  externalId
  assignmentBlocked
  gender
  canBeAssigned
  availableForDispatch
  version
  createdAt
  updatedAt
  customFields
  fleetCode {
    ...WorkerFleetCodeFields
  }
  state {
    ...WorkerUsStateFields
  }
  profile {
    ...WorkerProfileTableFields
  }
}`, {"hash":"sha256:7f614eb64dd42991c84620d0626f29319de8c30660b54dc87adb74e2e9b9b816"}) as unknown as TypedDocumentString<PatchWorkerMutation, PatchWorkerMutationVariables>;
export const ApproveWorkerPtoDocument = new TypedDocumentString(`
    mutation ApproveWorkerPto($id: ID!) {
  approveWorkerPTO(id: $id) {
    ...WorkerPtoRowFields
  }
}
    fragment WorkerPtoWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment WorkerPtoRowFields on WorkerPTO {
  id
  workerId
  organizationId
  businessUnitId
  approverId
  rejectorId
  status
  type
  startDate
  endDate
  reason
  version
  createdAt
  updatedAt
  worker {
    ...WorkerPtoWorkerFields
  }
}`, {"hash":"sha256:21010958257a62a735f4044929c3fc103181b56a46d4304fd953257172d88a71"}) as unknown as TypedDocumentString<ApproveWorkerPtoMutation, ApproveWorkerPtoMutationVariables>;
export const RejectWorkerPtoDocument = new TypedDocumentString(`
    mutation RejectWorkerPto($id: ID!, $reason: String!) {
  rejectWorkerPTO(id: $id, reason: $reason) {
    ...WorkerPtoRowFields
  }
}
    fragment WorkerPtoWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment WorkerPtoRowFields on WorkerPTO {
  id
  workerId
  organizationId
  businessUnitId
  approverId
  rejectorId
  status
  type
  startDate
  endDate
  reason
  version
  createdAt
  updatedAt
  worker {
    ...WorkerPtoWorkerFields
  }
}`, {"hash":"sha256:a7b30cefd403ecca3a57ce14133d160ac2e0867c727fa3039f55c433859ba230"}) as unknown as TypedDocumentString<RejectWorkerPtoMutation, RejectWorkerPtoMutationVariables>;