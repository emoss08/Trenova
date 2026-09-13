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

export type AcknowledgeMyPolicyInput = {
  policyId: string | number;
  /** Typed full name. Required when the policy asks for a signature, and it has to be the name on the record. */
  signatureName?: string | null | undefined;
};

export type AddCarrierSettlementAdjustmentInput = {
  amountMinor: number;
  description: string;
  /** Optional GL account override — the adjustment posts there instead of the default expense account. */
  glAccountId?: string | number | null | undefined;
  settlementId: string | number;
};

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

export type AdjustWorkerPtoBalanceInput = {
  amountDays: string;
  effectiveAt?: number | null | undefined;
  note: string;
  ptoType: PtoType;
  workerId: string | number;
};

export type AgentAutonomyTier =
  | 'ActWithApproval'
  | 'AutoExecute'
  | 'Propose';

export type AgentControlInput = {
  billingAgentEnabled: boolean;
  decisionTimeoutSeconds: number;
  shadowMode: boolean;
};

export type AgentDecisionType =
  | 'Accepted'
  | 'Modified'
  | 'Rejected';

export type AgentExceptionCategory =
  | 'AccessorialDispute'
  | 'ConfidenceBelowThreshold'
  | 'CustomerInformationError'
  | 'DuplicateCharge'
  | 'IncorrectRates'
  | 'MissingBOL'
  | 'MissingDocumentation'
  | 'MissingReferenceNumber'
  | 'MissingRequiredDocument'
  | 'Other'
  | 'RateMissingBasis'
  | 'RateNotOnFile'
  | 'RateVarianceRequiresAction'
  | 'ServiceFailure'
  | 'UnableToDiagnose'
  | 'UnresolvedServiceFailures'
  | 'WeightDiscrepancy';

export type AgentExceptionResolveInput = {
  resolutionNotes?: string | null | undefined;
  resolutionState: AgentResolutionState;
};

export type AgentProposalDecisionInput = {
  decision: AgentDecisionType;
  modifications?: unknown;
  reasonCode: string;
};

export type AgentProposalStatus =
  | 'Accepted'
  | 'Expired'
  | 'Modified'
  | 'Pending'
  | 'Rejected'
  | 'Superseded';

export type AgentResolutionState =
  | 'Dismissed'
  | 'InReview'
  | 'Open'
  | 'Resolved';

export type AgentRunStatus =
  | 'AwaitingDecision'
  | 'Completed'
  | 'Diagnosing'
  | 'Failed'
  | 'GatheringContext'
  | 'Pending'
  | 'ShadowCompleted';

export type AgentSeverity =
  | 'Critical'
  | 'High'
  | 'Low'
  | 'Medium';

export type AgentSubjectType =
  | 'BillingQueueItem';

export type AgentType =
  | 'BillingException';

export type AmendWorkerEmploymentEventInput = {
  amendmentNote: string;
  documentId?: string | number | null | undefined;
  effectiveAt?: number | null | undefined;
  id: string | number;
  notes?: string | null | undefined;
  reason?: string | null | undefined;
  version?: number | null | undefined;
};

export type ApplyCustomerPaymentInput = {
  accountingDate: number;
  applications: Array<CustomerPaymentApplicationInput>;
  paymentId: string | number;
};

/**
 * What a delegation covers. A manager going away usually hands over everything;
 * one handing a specific queue to a specialist hands over one thing, and a
 * delegation that quietly covered more than it said would be worse than none.
 */
export type ApprovalScope =
  | 'All'
  | 'Expenses'
  | 'TimeOff';

export type ArchiveWorkerCredentialInput = {
  id: string | number;
  reason?: string | null | undefined;
  version?: number | null | undefined;
};

/**
 * Ties a discovered card to the equipment or driver that carries it. Passing null
 * for either clears it, and a card with neither can still not match a purchase.
 */
export type AssignFuelCardInput = {
  assignedTractorId?: string | number | null | undefined;
  assignedWorkerId?: string | number | null | undefined;
  id: string | number;
  version: number;
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

export type AssignShiftInput = {
  cycleOffsetWeeks?: number | null | undefined;
  effectiveFrom: number;
  notes?: string | null | undefined;
  shiftTemplateId: string | number;
  workerId: string | number;
};

export type AssignWorkerPtoPolicyInput = {
  effectiveFrom: number;
  note?: string | null | undefined;
  openingBalances?: Array<OpeningPtoBalanceInput> | null | undefined;
  ptoPolicyId: string | number;
  workerIds: Array<string | number>;
};

export type AssignWorkerTrainingInput = {
  courseId: string | number;
  /** Defaults to the course's due days after today. */
  dueAt?: number | null | undefined;
  notes?: string | null | undefined;
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

export type AttachWorkerCredentialDocumentInput = {
  documentId: string | number;
  id: string | number;
};

export type AttachWorkerTrainingDocumentInput = {
  documentId: string | number;
  id: string | number;
};

export type AuditCategory =
  | 'System'
  | 'User';

/**
 * What a driver would rather work on a weekday. It is a statement, never a
 * constraint: dispatch is free to override it, and the rota shows where it did so
 * the override is visible rather than silent.
 */
export type AvailabilityPreference =
  | 'Available'
  | 'Preferred'
  | 'Unavailable';

export type BackfillJurisdictionMilesInput = {
  /** Count the moves and miles that would be attributed without starting the workflow. */
  dryRun?: boolean | null | undefined;
  /**
   * Upper bound on moves re-routed in one run; defaults to 2000. Every move is a
   * billable distance request.
   */
  maxMoves?: number | null | undefined;
  periodEnd: number;
  periodStart: number;
};

export type BenefitEnrollmentStatus =
  | 'Active'
  | 'Ended'
  | 'Pending'
  | 'Waived';

export type BenefitPlanInput = {
  carrier?: string | null | undefined;
  code: string;
  description?: string | null | undefined;
  employeeCostMinor: number;
  employerCostMinor: number;
  name: string;
  payCodeId: string | number;
  planType: BenefitPlanType;
  planYear: number;
  policyNumber?: string | null | undefined;
  status?: EntityStatus | null | undefined;
  waitingPeriodDays?: number | null | undefined;
};

export type BenefitPlanType =
  | 'Dental'
  | 'Disability'
  | 'Life'
  | 'Medical'
  | 'Other'
  | 'Retirement'
  | 'Vision';

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

export type BulkAssignTrainingInput = {
  courseIds: Array<string | number>;
  /** Overrides each course's own due-days default for this rollout. */
  dueAt?: number | null | undefined;
  notes?: string | null | undefined;
  workerIds: Array<string | number>;
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

export type BulkWorkerPtoActionInput = {
  action: WorkerPtoBulkAction;
  ptoIds: Array<string | number>;
  reason?: string | null | undefined;
};

export type CdlClass =
  | 'A'
  | 'B'
  | 'C';

/**
 * One of the seven Behavior Analysis and Safety Improvement Categories the FMCSA
 * sorts roadside violations into. A safety director reads their fleet in these
 * terms, so the roll-up speaks them rather than inventing its own grouping.
 */
export type CsaBasic =
  | 'ControlledSubstances'
  | 'CrashIndicator'
  | 'DriverFitness'
  | 'HOSCompliance'
  | 'HazmatCompliance'
  | 'UnsafeDriving'
  | 'VehicleMaintenance';

export type CancelFuelCardInput = {
  id: string | number;
  /** At least ten characters. Cancelling is permanent. */
  reason: string;
  version: number;
};

export type CancelWorkerChecklistInput = {
  id: string | number;
  reason?: string | null | undefined;
  version?: number | null | undefined;
};

export type CancelWorkerTrainingInput = {
  id: string | number;
  reason?: string | null | undefined;
  version?: number | null | undefined;
};

export type CarrierAssignmentStatus =
  | 'Canceled'
  | 'Confirmed'
  | 'Pending';

export type CarrierComplianceStatus =
  | 'Disqualified'
  | 'Expired'
  | 'Pending'
  | 'Qualified';

export type CarrierCostEventStatus =
  | 'Attached'
  | 'Pending'
  | 'Settled'
  | 'Voided';

export type CarrierCostEventType =
  | 'Accessorial'
  | 'Adjustment'
  | 'FuelSurcharge'
  | 'LinehaulCost';

export type CarrierInsurancePolicyType =
  | 'AutoLiability'
  | 'CargoLiability'
  | 'GeneralLiability'
  | 'Umbrella'
  | 'WorkersComp';

export type CarrierInvoiceMatchActionInput = {
  matchId: string | number;
  note?: string | null | undefined;
};

export type CarrierInvoiceMatchStatus =
  | 'Matched'
  | 'Rejected'
  | 'Resolved'
  | 'Suggested'
  | 'Variance';

/**
 * How an invoice match came to exist: Manual for a dispatcher-created match, Auto
 * for one created by the inbound EDI 210 auto-match sweep.
 */
export type CarrierInvoiceMatchVia =
  | 'Auto'
  | 'Manual';

export type CarrierLedgerEntryType =
  | 'Adjustment'
  | 'Bill'
  | 'Payment';

export type CarrierPaymentMethod =
  | 'ACHManual'
  | 'Check';

export type CarrierRateMethod =
  | 'Flat'
  | 'PerMile';

export type CarrierSafetyRating =
  | 'Conditional'
  | 'NotRated'
  | 'Satisfactory'
  | 'Unsatisfactory';

export type CarrierSettlementActionInput = {
  reason?: string | null | undefined;
  settlementId: string | number;
};

export type CarrierSettlementBatchStatus =
  | 'Canceled'
  | 'Completed'
  | 'Open';

export type CarrierSettlementStatus =
  | 'Approved'
  | 'Draft'
  | 'Paid'
  | 'PendingApproval'
  | 'Posted'
  | 'Voided';

export type CarrierStatus =
  | 'Active'
  | 'DoNotUse'
  | 'Inactive';

export type CarrierTaxIdType =
  | 'EIN'
  | 'SSN';

export type CarrierType =
  | 'Broker'
  | 'Common'
  | 'Contract'
  | 'Exempt';

export type ClearinghouseQueryType =
  | 'AnnualLimited'
  | 'Full'
  | 'Limited'
  | 'PreEmploymentFull';

export type ClearinghouseResult =
  | 'ConsentDenied'
  | 'NoViolations'
  | 'Pending'
  | 'ViolationsFound';

export type ClockInput = {
  /** Any instant; defaults to now. A punch cannot be dated in the future. */
  at?: number | null | undefined;
  breakMinutes?: number | null | undefined;
  note?: string | null | undefined;
  payCodeId?: string | number | null | undefined;
  workerId: string | number;
};

export type CompleteClearinghouseQueryInput = {
  completedAt?: number | null | undefined;
  documentId?: string | number | null | undefined;
  notes?: string | null | undefined;
  queryId: string | number;
  reference?: string | null | undefined;
  result: ClearinghouseResult;
  violationCount?: number | null | undefined;
};

export type CompleteWorkerTrainingInput = {
  completedAt?: number | null | undefined;
  courseId?: string | number | null | undefined;
  documentId?: string | number | null | undefined;
  /**
   * The open record to close. Leave empty to file a completion straight against
   * workerId + courseId (a classroom session recorded after the fact).
   */
  id?: string | number | null | undefined;
  notes?: string | null | undefined;
  score?: string | null | undefined;
  version?: number | null | undefined;
  workerId?: string | number | null | undefined;
};

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

/**
 * Who the cover extends to. It is on the enrollment rather than the plan because
 * the same plan costs different amounts depending on how many people it covers.
 */
export type CoverageTier =
  | 'Employee'
  | 'EmployeeChildren'
  | 'EmployeeSpouse'
  | 'Family';

export type CreateCarrierInvoiceMatchInput = {
  /** Optional explicit assignment; otherwise resolved by pro number or shipment reference. */
  carrierAssignmentId?: string | number | null | undefined;
  /** Required for document AI sources; EDI sources take the carrier from the linked invoice. */
  carrierId?: string | number | null | undefined;
  documentAiExtractionId?: string | number | null | undefined;
  /** Exactly one source is required: an EDI carrier invoice or a document AI extraction. */
  ediCarrierInvoiceId?: string | number | null | undefined;
  /** Document AI sources only: the extracted invoice number. */
  invoiceNumber?: string | null | undefined;
  /** Document AI sources only: the extracted invoice total in minor units. */
  invoiceTotalMinor?: number | null | undefined;
  /** Document AI sources only: the pro number used to locate the assignment. */
  proNumber?: string | null | undefined;
  /** Document AI sources only: the shipment used to locate the assignment. */
  shipmentId?: string | number | null | undefined;
};

export type CreateFuelPurchaseImportInput = {
  defaultCurrency?: string | null | undefined;
  defaultFuelCardId?: string | number | null | undefined;
  defaultFuelType?: IftaFuelType | null | undefined;
  mapping?: unknown;
  provider: FuelCardProvider;
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

export type CreatePerformanceReviewInput = {
  periodEnd: number;
  periodStart: number;
  templateId: string | number;
  title?: string | null | undefined;
  workerId: string | number;
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
  alert?: ReportScheduleAlertInput | null | undefined;
  cronExpression: string;
  definitionId: string | number;
  emailAttach?: boolean | null | undefined;
  emailInline?: boolean | null | undefined;
  emailRecipients?: Array<string> | null | undefined;
  enabled: boolean;
  formats: Array<string>;
  notifyUserIds?: Array<string | number> | null | undefined;
  timezone?: string | null | undefined;
};

export type CreateReportViewInput = {
  definitionId: string | number;
  description?: string | null | undefined;
  format?: string | null | undefined;
  name: string;
  params?: unknown;
  pinned?: boolean | null | undefined;
  shared?: boolean | null | undefined;
};

export type CreateSettlementDisputeInput = {
  category: SettlementDisputeCategory;
  description: string;
  settlementId: string | number;
  settlementLineId?: string | number | null | undefined;
};

export type CreateWorkerPtoInput = {
  endDate: number;
  reason: string;
  startDate: number;
  type: PtoType;
  workerId: string | number;
};

/** How often a statement-billed customer is billed. */
export type CustomerBillingCycle =
  | 'BiWeekly'
  | 'Daily'
  | 'Immediate'
  | 'Monthly'
  | 'Quarterly'
  | 'SemiMonthly'
  | 'Weekly';

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

/** How many invoices a customer's freight turns into. */
export type CustomerInvoiceDelivery =
  /** One or more invoices per billing period, covering the period's shipments. */
  | 'Consolidated'
  /** One invoice per order, covering every billable leg. */
  | 'PerOrder'
  /** One invoice per shipment. */
  | 'PerShipment';

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

export type DotRandomPoolInput = {
  alcoholRatePercent: number;
  code: string;
  description?: string | null | undefined;
  drugRatePercent: number;
  includedDriverTypes?: Array<string> | null | undefined;
  isDefault?: boolean | null | undefined;
  name: string;
  period: RandomPeriod;
  status?: EntityStatus | null | undefined;
};

export type DotRandomPoolsInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: EntityStatus | null | undefined;
};

export type DotTestResult =
  | 'Adulterated'
  | 'Cancelled'
  | 'Invalid'
  | 'Negative'
  | 'NegativeDilute'
  | 'Pending'
  | 'Positive'
  | 'Refusal'
  | 'Substituted';

export type DotTestStatus =
  | 'AwaitingResult'
  | 'Cancelled'
  | 'Collected'
  | 'Completed'
  | 'Scheduled';

export type DotTestSubstance =
  | 'Alcohol'
  | 'Drug';

export type DotTestType =
  | 'FollowUp'
  | 'Other'
  | 'PostAccident'
  | 'PreEmployment'
  | 'Random'
  | 'ReasonableSuspicion'
  | 'ReturnToDuty';

/**
 * Where the return-to-duty process has got to. Follow-up testing happens after the
 * driver is back at work, so it is not a bar.
 */
export type DotViolationStatus =
  | 'FollowUp'
  | 'Open'
  | 'RTDPending'
  | 'Resolved'
  | 'SAPEvaluation';

export type DotViolationType =
  | 'ActualKnowledge'
  | 'AlcoholUse'
  | 'DrugUse'
  | 'Other'
  | 'PositiveTest'
  | 'TestRefusal';

export type DqfItemStatus =
  | 'Expired'
  | 'ExpiringSoon'
  | 'Missing'
  | 'NotApplicable'
  | 'Outstanding'
  | 'Satisfied';

/** Where a driver qualification requirement's evidence lives. */
export type DqfSection =
  | 'Credentials'
  | 'Documents'
  | 'DrugAlcohol'
  | 'SafetyHistory';

export type DataTableConnectionInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
};

export type DecideLeaveCaseInput = {
  approve: boolean;
  caseId: string | number;
  /**
   * Whether the leave counts against the FMLA entitlement. Ignored when the case
   * is denied: denied leave draws nothing down.
   */
  designate: boolean;
  notes?: string | null | undefined;
};

export type DecideProfileChangeInput = {
  approve: boolean;
  id: string | number;
  /** Required when turning a request down: a driver whose record did not change deserves to know why. */
  note?: string | null | undefined;
};

export type DelegateApprovalInput = {
  delegateId: string | number;
  /**
   * Whose approvals are being handed over. Left empty it is the signed-in user;
   * naming somebody else needs the manage grant.
   */
  delegatorId?: string | number | null | undefined;
  endsAt?: number | null | undefined;
  reason?: string | null | undefined;
  scope?: ApprovalScope | null | undefined;
  startsAt?: number | null | undefined;
};

export type DeleteTimeEntryInput = {
  id: string | number;
  reason: string;
};

export type DetachPayEventInput = {
  payEventId: string | number;
  settlementId: string | number;
};

export type DetentionBacktestInput = {
  assumeNoticeCompliance?: boolean | null | undefined;
  driverPayRate?: string | null | undefined;
  from: number;
  limit?: number | null | undefined;
  policy: DetentionPolicyInput;
  to: number;
};

export type DetentionCapKind =
  | 'LayoverBoundary'
  | 'MaxBillableMinutes'
  | 'MaxChargePerDay'
  | 'MaxChargePerShipment'
  | 'MaxChargePerStop'
  | 'None';

export type DetentionCapScope =
  | 'PerDay'
  | 'PerShipment'
  | 'PerStop';

export type DetentionClockStartBasis =
  | 'Appointment'
  | 'Arrival'
  | 'EarlierOfArrivalOrAppointment'
  | 'LaterOfArrivalOrAppointment';

export type DetentionDeskUrgency =
  | 'Accruing'
  | 'FreeTimeEnding'
  | 'Lost'
  | 'Normal'
  | 'NoticeDueSoon'
  | 'NoticeOverdue';

export type DetentionDisputeInput = {
  note: string;
  occurrenceId: string | number;
};

export type DetentionEvidenceKind =
  | 'Appointment'
  | 'Arrival'
  | 'Departure'
  | 'Dispute'
  | 'Document'
  | 'Notice'
  | 'Recalculated'
  | 'StatusChange'
  | 'Waiver';

export type DetentionEvidenceSource =
  | 'Document'
  | 'DriverApp'
  | 'EDI'
  | 'Geofence'
  | 'Manual'
  | 'System'
  | 'Telematics';

export type DetentionLateArrivalRule =
  | 'ClockFromAppointment'
  | 'Forfeit'
  | 'NoEffect'
  | 'ReduceFreeTime';

export type DetentionNoticeChannel =
  | 'EDI'
  | 'Email'
  | 'Manual'
  | 'Portal';

export type DetentionNoticeDeliveryStatus =
  | 'Bounced'
  | 'Delivered'
  | 'Failed'
  | 'Opened'
  | 'Queued'
  | 'Sent';

export type DetentionNoticeKind =
  | 'Final'
  | 'Started'
  | 'Update'
  | 'Warning';

export type DetentionNotificationRequirement =
  | 'Advisory'
  | 'None'
  | 'Required';

export type DetentionNotificationStatus =
  | 'Failed'
  | 'Late'
  | 'Missed'
  | 'NotRequired'
  | 'Pending'
  | 'Sent';

export type DetentionOccurrenceStatus =
  | 'Accruing'
  | 'Approved'
  | 'Billed'
  | 'Disputed'
  | 'NotBillable'
  | 'Pending'
  | 'Waived';

export type DetentionPolicyInput = {
  accessorialChargeId: string | number;
  appointmentStopsOnly?: boolean | null | undefined;
  attachNoticePdf?: boolean | null | undefined;
  autoApproveUnderAmount?: string | null | undefined;
  autoSendNotice?: boolean | null | undefined;
  billingFreeMinutes?: number | null | undefined;
  billingIncrementMinutes?: number | null | undefined;
  clockStartBasis?: DetentionClockStartBasis | null | undefined;
  code: string;
  comments?: string | null | undefined;
  commodityIds?: Array<string | number> | null | undefined;
  convertToLayoverAtMinutes?: number | null | undefined;
  currency?: string | null | undefined;
  customerId?: string | number | null | undefined;
  dayBoundaryMode?: DetentionCapScope | null | undefined;
  deliveryFreeMinutes?: number | null | undefined;
  description?: string | null | undefined;
  effectiveEndDate?: number | null | undefined;
  effectiveStartDate?: number | null | undefined;
  isOrgDefault?: boolean | null | undefined;
  lateArrivalGraceMinutes?: number | null | undefined;
  lateArrivalRule?: DetentionLateArrivalRule | null | undefined;
  layoverAccessorialChargeId?: string | number | null | undefined;
  locationId?: string | number | null | undefined;
  maxBillableMinutesPerStop?: number | null | undefined;
  maxChargePerDay?: string | null | undefined;
  maxChargePerShipment?: string | null | undefined;
  maxChargePerStop?: string | null | undefined;
  minimumBillableMinutes?: number | null | undefined;
  name: string;
  notificationDeadlineMinutes?: number | null | undefined;
  notificationLeadMinutes?: number | null | undefined;
  notificationRequirement?: DetentionNotificationRequirement | null | undefined;
  payFreeMinutes?: number | null | undefined;
  pickupFreeMinutes?: number | null | undefined;
  priority?: number | null | undefined;
  rateSource?: DetentionRateSource | null | undefined;
  requireApprovalOverAmount?: string | null | undefined;
  roundingMode?: DetentionRoundingMode | null | undefined;
  sendDepartureSummary?: boolean | null | undefined;
  serviceTypeIds?: Array<string | number> | null | undefined;
  shipmentTypeIds?: Array<string | number> | null | undefined;
  status?: DetentionPolicyStatus | null | undefined;
  stopTypes?: Array<StopType> | null | undefined;
  tiers?: Array<DetentionTierInput> | null | undefined;
  unnotifiedBehavior?: DetentionUnnotifiedBehavior | null | undefined;
  version?: number | null | undefined;
};

export type DetentionPolicyStatus =
  | 'Active'
  | 'Draft'
  | 'Inactive';

export type DetentionPreviewScenarioInput = {
  appointmentEnd?: number | null | undefined;
  appointmentStart?: number | null | undefined;
  arrivedAt: number;
  departedAt?: number | null | undefined;
  driverPayRate?: string | null | undefined;
  noticeSentAt?: number | null | undefined;
  scheduleType: StopScheduleType;
  stopType: StopType;
};

export type DetentionRateSource =
  | 'Accessorial'
  | 'Tiers';

export type DetentionRoundingMode =
  | 'Down'
  | 'Exact'
  | 'Nearest'
  | 'Up';

export type DetentionScoreBand =
  | 'Adequate'
  | 'AtRisk'
  | 'Strong'
  | 'Weak';

export type DetentionStatsInput = {
  from: number;
  limit?: number | null | undefined;
  to: number;
};

export type DetentionTierInput = {
  fromMinute: number;
  label?: string | null | undefined;
  rate: string;
  rateUnit?: DetentionTierRateUnit | null | undefined;
  sortOrder?: number | null | undefined;
  toMinute?: number | null | undefined;
};

export type DetentionTierRateUnit =
  | 'Day'
  | 'Flat'
  | 'Hour';

export type DetentionUnnotifiedBehavior =
  | 'Bill'
  | 'Flag'
  | 'Suppress';

export type DetentionWaiveInput = {
  note: string;
  occurrenceId: string | number;
  reason: DetentionWaiverReason;
};

export type DetentionWaiverReason =
  | 'CarrierFault'
  | 'CustomerGoodwill'
  | 'DataCorrection'
  | 'EquipmentIssue'
  | 'FacilityClosure'
  | 'ForceMajeure'
  | 'Other'
  | 'Weather';

export type DisciplinaryLevel =
  | 'Coaching'
  | 'FinalWarning'
  | 'Suspension'
  | 'Termination'
  | 'VerbalWarning'
  | 'WrittenWarning';

export type DisciplinaryStatus =
  | 'Active'
  | 'Expired'
  | 'Rescinded';

export type DispatchAssignMoveInput = {
  moveId: string | number;
  primaryWorkerId: string | number;
  /**
   * Replace an existing assignment rather than creating one. The console sets this when a
   * dispatcher drags a driver onto a move that is already covered.
   */
  reassign?: boolean | null | undefined;
  secondaryWorkerId?: string | number | null | undefined;
  tractorId: string | number;
  trailerId?: string | number | null | undefined;
};

export type DispatchAssignMoveToCarrierInput = {
  accessorials?: Array<DispatchCarrierAccessorialInput> | null | undefined;
  baseRate: string;
  carrierId: string | number;
  externalDriverName?: string | null | undefined;
  externalDriverPhone?: string | null | undefined;
  externalTractorNumber?: string | null | undefined;
  externalTrailerNumber?: string | null | undefined;
  fuelSurcharge?: string | null | undefined;
  moveId: string | number;
  /**
   * Proceed despite insurance policies that expire inside the warning window. Hard blockers
   * (inactive, unqualified, or expired coverage) can never be overridden.
   */
  overrideInsuranceWarning?: boolean | null | undefined;
  proNumber?: string | null | undefined;
  rateMethod: CarrierRateMethod;
  /**
   * Replace an existing carrier assignment rather than rejecting the request. The console
   * sets this when a dispatcher re-covers a move that already has carrier coverage.
   */
  replace?: boolean | null | undefined;
};

export type DispatchAssignmentPreviewInput = {
  moveId: string | number;
  tractorId?: string | number | null | undefined;
  trailerId?: string | number | null | undefined;
  workerId: string | number;
};

export type DispatchBoardInput = {
  customerIds?: Array<string | number> | null | undefined;
  fleetCodeIds?: Array<string | number> | null | undefined;
  includeCovered?: boolean | null | undefined;
  limit?: number | null | undefined;
  query?: string | null | undefined;
  serviceTypeIds?: Array<string | number> | null | undefined;
  windowEnd?: number | null | undefined;
  windowStart?: number | null | undefined;
  workerIds?: Array<string | number> | null | undefined;
};

export type DispatchCarrierAccessorialInput = {
  accessorialChargeId?: string | number | null | undefined;
  amount: string;
  description: string;
};

export type DispatchCarrierAssignmentPreviewInput = {
  carrierId: string | number;
};

export type DispatchDriverMovesInput = {
  limit?: number | null | undefined;
  windowEnd?: number | null | undefined;
  windowStart?: number | null | undefined;
  workerId: string | number;
};

export type DispatchMoveCandidatesInput = {
  fleetCodeIds?: Array<string | number> | null | undefined;
  includeBlocked?: boolean | null | undefined;
  limit?: number | null | undefined;
  moveId: string | number;
};

export type DispatchPlanInput = {
  /**
   * Commit the plan immediately where the organization's autonomy tier allows it. When
   * false the plan is only proposed, whatever the tier.
   */
  apply?: boolean | null | undefined;
  fleetCodeIds?: Array<string | number> | null | undefined;
  moveIds?: Array<string | number> | null | undefined;
  windowEnd?: number | null | undefined;
  windowStart?: number | null | undefined;
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

/**
 * How a driver hears about what they owe. Immediate is the default so no carrier
 * silently loses notices they already rely on.
 */
export type DriverDigestCadence =
  | 'Daily'
  | 'Immediate'
  | 'Weekly';

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

/**
 * The worker's testing record in one word. Unknown means nothing is on file, which
 * is not the same as clear.
 */
export type DrugAlcoholStatus =
  | 'Clear'
  | 'Pending'
  | 'Prohibited'
  | 'Unknown';

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

export type EmploymentVerificationMethod =
  | 'Email'
  | 'Fax'
  | 'Mail'
  | 'Other'
  | 'Phone'
  | 'Portal';

export type EmploymentVerificationStatus =
  | 'NoResponse'
  | 'NotApplicable'
  | 'Pending'
  | 'Received'
  | 'Requested';

export type EndBenefitEnrollmentInput = {
  effectiveTo?: number | null | undefined;
  id: string | number;
  notes?: string | null | undefined;
  version?: number | null | undefined;
};

export type EndWorkerPtoPolicyAssignmentInput = {
  assignmentId: string | number;
  effectiveTo?: number | null | undefined;
  version?: number | null | undefined;
};

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

export type EnrollBenefitInput = {
  benefitPlanId: string | number;
  coverageTier?: CoverageTier | null | undefined;
  effectiveFrom?: number | null | undefined;
  /** Overrides the tier-scaled price for a carrier that prices each tier separately. */
  employeeCostMinor?: number | null | undefined;
  notes?: string | null | undefined;
  /**
   * Records a decision not to take the cover. A waiver still produces an
   * enrollment, because "declined" and "nobody asked" are different facts and only
   * one of them is a problem at audit.
   */
  waive?: boolean | null | undefined;
  waivedReason?: string | null | undefined;
  workerId: string | number;
};

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
  /** Omit to leave unchanged; null is rejected. */
  class?: EquipmentClass | null | undefined;
  /** Omit to leave unchanged; null is rejected. */
  code?: string | null | undefined;
  /** Omit to leave unchanged; pass null to clear. */
  color?: string | null | undefined;
  /** Omit to leave unchanged; pass null to clear. */
  description?: string | null | undefined;
  /** Omit to leave unchanged; pass null to clear. */
  interiorLength?: number | null | undefined;
  /** Omit to leave unchanged; null is rejected. */
  status?: EntityStatus | null | undefined;
  /** Omit to leave unchanged; null is rejected. */
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

export type FleetSafetyInput = {
  /** Narrows every section to one terminal. */
  fleetCodeId?: string | number | null | undefined;
  /** How many drivers each of the best and worst lists carries. */
  rankLimit?: number | null | undefined;
  /**
   * The counting window in months, 1 to 36. The BASIC scores always use the
   * FMCSA's own twenty-four month look-back regardless, because a BASIC measured
   * over anything else is not a BASIC.
   */
  windowMonths?: number | null | undefined;
};

export type ForkCannedReportInput = {
  cannedKey: string;
  name?: string | null | undefined;
};

export type FormulaTemplateStatus =
  | 'Active'
  | 'Draft'
  | 'InReview'
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

export type FuelCardInput = {
  assignedTractorId?: string | number | null | undefined;
  assignedWorkerId?: string | number | null | undefined;
  expiresAt?: number | null | undefined;
  externalCardId?: string | null | undefined;
  label: string;
  lastFour: string;
  notes?: string | null | undefined;
  provider: FuelCardProvider;
  /** Active or Suspended. Cancelling goes through cancelFuelCard, which needs a reason. */
  status?: FuelCardStatus | null | undefined;
};

export type FuelCardProvider =
  | 'Comdata'
  | 'EFS'
  | 'Other'
  | 'WEX';

/** Active and Suspended move between each other; Cancelled is terminal. */
export type FuelCardStatus =
  | 'Active'
  | 'Cancelled'
  | 'Suspended';

export type FuelCardsInput = {
  after?: string | null | undefined;
  assignedTractorId?: string | number | null | undefined;
  assignedWorkerId?: string | number | null | undefined;
  /** Only cards a feed created rather than a person. */
  discoveredOnly?: boolean | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  provider?: FuelCardProvider | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: FuelCardStatus | null | undefined;
  /**
   * Only cards with neither a tractor nor a driver on them. A feed creates cards
   * in this state the first time it sees a transaction on one.
   */
  unassignedOnly?: boolean | null | undefined;
};

export type FuelImportFormat =
  /** Rows read from a provider's API rather than a file. */
  | 'API'
  | 'CSV'
  /**
   * A file whose fields are at fixed column positions, per the provider's record
   * layout. The layout is configured on the connection, because the networks
   * publish theirs under their own agreements and they differ by account.
   */
  | 'FixedWidth'
  | 'XLSX';

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

/** Whether a batch came from a person choosing a file or from a scheduled sync. */
export type FuelPurchaseImportOrigin =
  | 'Feed'
  | 'Upload';

/**
 * Only New rows commit. DuplicateInFile is the second occurrence of a reference
 * inside the statement; AlreadyImported matches a purchase already on file.
 */
export type FuelPurchaseImportRowStatus =
  | 'AlreadyImported'
  | 'Committed'
  | 'DuplicateInFile'
  | 'Error'
  | 'New'
  | 'Skipped';

export type FuelPurchaseImportRowsInput = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  statuses?: Array<FuelPurchaseImportRowStatus> | null | undefined;
};

/**
 * Pending until a statement is staged; Parsed once its rows have been read and
 * resolved; Committed when the rows became purchases. Failed and Discarded batches
 * can be staged again or left as a record of what was tried.
 */
export type FuelPurchaseImportStatus =
  | 'Committed'
  | 'Discarded'
  | 'Failed'
  | 'Parsed'
  | 'Pending';

export type FuelPurchaseImportsInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  /** Only imports still holding rows that could not be worked out. */
  heldRowsOnly?: boolean | null | undefined;
  /**
   * Feed lists the runs a scheduled sync opened; Upload lists the statements people
   * chose. Omit for both.
   */
  origin?: FuelPurchaseImportOrigin | null | undefined;
  provider?: FuelCardProvider | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  statuses?: Array<FuelPurchaseImportStatus> | null | undefined;
};

export type FuelPurchaseInput = {
  cardLastFour?: string | null | undefined;
  currencyCode?: string | null | undefined;
  fuelCardId?: string | number | null | undefined;
  fuelType: IftaFuelType;
  jurisdictionId: string | number;
  notes?: string | null | undefined;
  odometer?: number | null | undefined;
  purchasedAt: number;
  quantity: string;
  quantityUnit?: FuelQuantityUnit | null | undefined;
  taxPaid?: boolean | null | undefined;
  totalAmount: string;
  tractorId: string | number;
  transactionReference?: string | null | undefined;
  unitPrice?: string | null | undefined;
  vendor?: string | null | undefined;
  vendorCity?: string | null | undefined;
  workerId?: string | number | null | undefined;
};

export type FuelPurchaseSource =
  | 'CardImport'
  | 'Manual';

export type FuelPurchasesInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  /** Inclusive lower bound on purchasedAt. */
  from?: number | null | undefined;
  fuelCardId?: string | number | null | undefined;
  fuelTypes?: Array<IftaFuelType> | null | undefined;
  jurisdictionId?: string | number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  sources?: Array<FuelPurchaseSource> | null | undefined;
  taxPaid?: boolean | null | undefined;
  /** Exclusive upper bound on purchasedAt. */
  to?: number | null | undefined;
  tractorId?: string | number | null | undefined;
};

export type FuelQuantityUnit =
  | 'Gallon'
  | 'Litre';

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

export type GenerateCarrierSettlementBatchInput = {
  name?: string | null | undefined;
  notes?: string | null | undefined;
  periodEnd?: number | null | undefined;
  periodStart?: number | null | undefined;
};

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

export type GeneratePayrollExportInput = {
  note?: string | null | undefined;
  periodEnd: number;
  periodStart: number;
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

export type HomeLayoutInput = {
  /** When false the viewer follows their assigned preset and widgets is ignored. */
  customized: boolean;
  density: string;
  version: number;
  widgets: Array<HomeWidgetInput>;
};

/** Which tier of the resolution chain produced the home screen on display. */
export type HomeLayoutSource =
  | 'BUILT_IN'
  | 'ORG_DEFAULT'
  | 'ROLE_PRESET'
  | 'USER';

export type HomeWidgetConfigInput = {
  cannedKey?: string | null | undefined;
  chartId?: string | null | undefined;
  columnId?: string | null | undefined;
  dashboardId?: string | number | null | undefined;
  definitionId?: string | number | null | undefined;
  limit?: number | null | undefined;
  metric?: string | null | undefined;
  metrics?: Array<string> | null | undefined;
  text?: string | null | undefined;
  windowDays?: number | null | undefined;
};

export type HomeWidgetInput = {
  config?: HomeWidgetConfigInput | null | undefined;
  h: number;
  id: string;
  key: string;
  title?: string | null | undefined;
  w: number;
};

/**
 * Fuel product as classified for the International Fuel Tax Agreement. The first
 * fourteen are IFTA fuel types and enter the return; DEF, Reefer and Other are
 * tracked as spend only and never earn or owe tax.
 */
export type IftaFuelType =
  | 'A55'
  | 'Biodiesel'
  | 'CNG'
  | 'DEF'
  | 'Diesel'
  | 'E85'
  | 'Electricity'
  | 'Ethanol'
  | 'Gasohol'
  | 'Gasoline'
  | 'Hydrogen'
  | 'LNG'
  | 'M85'
  | 'Methanol'
  | 'Other'
  | 'Propane'
  | 'Reefer';

export type IftaJurisdictionStatus =
  | 'Active'
  | 'Inactive';

export type IftaMileageEntriesInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  /** Inclusive lower bound on traveledAt. */
  from?: number | null | undefined;
  jurisdictionId?: string | number | null | undefined;
  period?: IftaPeriodInput | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  sources?: Array<IftaMileageSource> | null | undefined;
  /** Exclusive upper bound on traveledAt. */
  to?: number | null | undefined;
  tractorId?: string | number | null | undefined;
};

export type IftaMileageEntryInput = {
  jurisdictionId: string | number;
  loaded?: boolean | null | undefined;
  miles: string;
  notes?: string | null | undefined;
  /**
   * Name the move when this entry corrects its routed miles; the entry then
   * replaces the move's jurisdiction rows on the return.
   */
  shipmentMoveId?: string | number | null | undefined;
  source?: IftaMileageSource | null | undefined;
  tractorId: string | number;
  traveledAt: number;
};

/**
 * Where a jurisdiction's miles came from. RouteCalculation rows are written by the
 * distance provider's state report; Manual entries are keyed by hand and, when they
 * name a shipment move, replace that move's routed rows so nothing is counted twice.
 */
export type IftaMileageSource =
  | 'Manual'
  | 'RouteCalculation'
  | 'Telematics';

export type IftaPeriodInput = {
  quarter: number;
  year: number;
};

/**
 * MissingRate blocks finalizing. Every other code is a warning the return carries
 * so the preparer can see what the figures leave out.
 */
export type IftaProblemCode =
  | 'MileageMismatch'
  | 'MissingRate'
  | 'NoFuelForType'
  | 'NoTractorMiles'
  | 'NonMemberActivity'
  | 'NonQualifiedActivity'
  | 'UnattributedMiles';

/**
 * Draft figures move with the data. Finalized locks them and can be reopened with
 * a reason. Filed is immutable; corrections open a new Draft through amendIftaReturn.
 */
export type IftaReturnStatus =
  | 'Draft'
  | 'Filed'
  | 'Finalized';

export type IftaReturnsInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  statuses?: Array<IftaReturnStatus> | null | undefined;
  year?: number | null | undefined;
};

export type IftaTaxRateInput = {
  fuelType: IftaFuelType;
  jurisdictionId: string | number;
  quarter: number;
  ratePerGallon: string;
  sourceNote?: string | null | undefined;
  sourceUrl?: string | null | undefined;
  surchargeRatePerGallon?: string | null | undefined;
  year: number;
};

export type IftaTaxRatesInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  fuelType?: IftaFuelType | null | undefined;
  jurisdictionId?: string | number | null | undefined;
  period?: IftaPeriodInput | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
};

export type InjuryCaseStatus =
  | 'Closed'
  | 'Open';

export type InjuryTreatment =
  | 'EmergencyRoom'
  | 'FirstAid'
  | 'Hospitalized'
  | 'MedicalTreatment'
  | 'None';

export type InspectionResult =
  | 'Fail'
  | 'OutOfService'
  | 'Pass';

export type InviteWorkerToPortalInput = {
  /** Overrides the email on the worker record when provided. */
  email?: string | null | undefined;
  workerId: string | number;
};

/** How verbose each section of a consolidated invoice is. */
export type InvoiceDetail =
  | 'Detailed'
  | 'Summary';

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

/** How the lines inside one invoice are organised. Never changes how many there are. */
export type InvoiceSectionKey =
  | 'Destination'
  | 'Origin'
  | 'PONumber'
  | 'Shipment';

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

/**
 * How many invoices a billing period yields. Customer means one; every other
 * member means one per distinct value of that key.
 */
export type InvoiceSplitKey =
  | 'Customer'
  | 'CustomerAndDestination'
  | 'CustomerAndOrder'
  | 'CustomerAndOrigin'
  | 'CustomerAndPONumber'
  | 'CustomerAndServiceType'
  | 'CustomerAndShipmentBOL';

export type InvoiceStatus =
  | 'Draft'
  | 'Posted';

export type IssueDisciplinaryActionInput = {
  details?: string | null | undefined;
  documentId?: string | number | null | undefined;
  expiresAt?: number | null | undefined;
  level: DisciplinaryLevel;
  occurredAt?: number | null | undefined;
  reason: string;
  /** Also record a Suspended / Terminated event on the timeline. */
  recordEmploymentEvent?: boolean | null | undefined;
  safetyEventId?: string | number | null | undefined;
  suspensionDays?: number | null | undefined;
  workerId: string | number;
};

export type IssuePayAdvanceInput = {
  amountMinor: number;
  currencyCode?: string | null | undefined;
  issuedDate: number;
  notes?: string | null | undefined;
  reference?: string | null | undefined;
  source: PayAdvanceSource;
  workerId: string | number;
};

/**
 * The part of the business a position sits in. Headcount is read by department
 * at least as often as by terminal, which is why it is an enum rather than free
 * text somebody spells three ways.
 */
export type JobDepartment =
  | 'Administration'
  | 'Billing'
  | 'Executive'
  | 'HumanResources'
  | 'Maintenance'
  | 'Operations'
  | 'Other'
  | 'Safety'
  | 'Sales';

export type JobPositionInput = {
  code: string;
  department: JobDepartment;
  description?: string | null | undefined;
  flsaExempt?: boolean | null | undefined;
  isDrivingPosition?: boolean | null | undefined;
  reportsToPositionId?: string | number | null | undefined;
  status?: EntityStatus | null | undefined;
  title: string;
};

export type JournalReversalStatus =
  | 'Approved'
  | 'Cancelled'
  | 'PendingApproval'
  | 'Posted'
  | 'Rejected'
  | 'Requested';

export type JurisdictionRuleStatus =
  | 'Active'
  | 'Draft'
  | 'Inactive';

export type JurisdictionVerificationState =
  | 'Disputed'
  | 'Unverified'
  | 'Verified';

export type LeaveCaseStatus =
  | 'Approved'
  | 'Closed'
  | 'Denied'
  | 'Pending';

export type LeaveCertificationStatus =
  | 'Insufficient'
  | 'NotRequired'
  | 'Overdue'
  | 'Received'
  | 'Requested'
  | 'Waived';

export type LeaveFrequency =
  | 'Continuous'
  | 'Intermittent'
  | 'ReducedSchedule';

/**
 * How the twelve-month period is measured. An employer picks one of the four in
 * 29 CFR 825.200(b) and must apply it to every employee alike.
 */
export type LeaveMeasurementMethod =
  | 'CalendarYear'
  | 'ForwardFromFirstUse'
  | 'HireAnniversary'
  | 'RollingBackward';

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

export type MarkCarrierSettlementPaidInput = {
  paymentMethod: string;
  paymentReference?: string | null | undefined;
  settlementId: string | number;
};

export type MarkDriverSettlementPaidInput = {
  paymentMethod: string;
  paymentReference?: string | null | undefined;
  settlementId: string | number;
};

export type MarkIftaReturnFiledInput = {
  /** Between the moment the return was finalized and now. */
  filedAt: number;
  filingReference?: string | null | undefined;
  id: string | number;
  version: number;
};

export type MatchRoutingGuideInput = {
  destinationCity?: string | null | undefined;
  destinationLocationId?: string | number | null | undefined;
  destinationState?: string | null | undefined;
  originCity?: string | null | undefined;
  originLocationId?: string | number | null | undefined;
  originState?: string | null | undefined;
};

export type MoveCoverageType =
  | 'carrier'
  | 'driver'
  | 'unassigned';

export type MoveStatus =
  | 'Assigned'
  | 'Canceled'
  | 'Completed'
  | 'InTransit'
  | 'New';

/**
 * The driver's own answer to a swap. Approving and rejecting are deliberately
 * absent: a swap is decided by the office.
 */
export type MyShiftSwapResponse =
  | 'Accepted'
  | 'Declined'
  | 'Withdrawn';

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

/**
 * The column of the OSHA 300 log a case lands in. The log records only the most
 * serious outcome, so a case that was restricted and then went days-away is a
 * days-away case.
 */
export type OshaCaseClassification =
  | 'DaysAway'
  | 'Death'
  | 'FirstAidOnly'
  | 'JobTransferOrRestriction'
  | 'NotRecordable'
  | 'OtherRecordable';

export type OshaIllnessType =
  | 'HearingLoss'
  | 'Injury'
  | 'OtherIllness'
  | 'Poisoning'
  | 'RespiratoryCondition'
  | 'SkinDisorder';

export type OshaSummaryStatus =
  | 'Certified'
  | 'Draft';

export type OpenEscrowAccountInput = {
  annualInterestRate?: string | null | undefined;
  currencyCode?: string | null | undefined;
  openedDate?: number | null | undefined;
  targetAmountMinor: number;
  workerId: string | number;
};

export type OpenLeaveCaseInput = {
  documentId?: string | number | null | undefined;
  eligibilityHoursWorked?: number | null | undefined;
  endsAt?: number | null | undefined;
  frequency?: LeaveFrequency | null | undefined;
  leaveType?: WorkerLeaveType | null | undefined;
  militaryCaregiver?: boolean | null | undefined;
  notes?: string | null | undefined;
  reason?: string | null | undefined;
  requestedAt?: number | null | undefined;
  startsAt: number;
  workerId: string | number;
};

export type OpeningPtoBalanceInput = {
  /** Opening balance, in days. */
  days: string;
  ptoType: PtoType;
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

export type OrgHolidayInput = {
  description?: string | null | undefined;
  holidayDate: number;
  kind: OrgHolidayKind;
  name: string;
  recursAnnually: boolean;
  version?: number | null | undefined;
};

export type OrgHolidayKind =
  | 'Blackout'
  | 'Holiday';

export type OrganizationInput = {
  addressLine1: string;
  addressLine2?: string | null | undefined;
  assetOperationsEnabled?: boolean | null | undefined;
  brokerageEnabled?: boolean | null | undefined;
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

export type PtoAccrualMethod =
  | 'FixedAnnualGrant'
  | 'Monthly'
  | 'None'
  | 'PerPayPeriod';

export type PtoAccrualTierInput = {
  accrualAmountDays: string;
  maxBalanceDays?: string | null | undefined;
  minMonths: number;
};

export type PtoAvailabilityInput = {
  endDate: number;
  excludePtoId?: string | number | null | undefined;
  ptoType: PtoType;
  startDate: number;
  workerId: string | number;
};

export type PtoLedgerActorType =
  | 'System'
  | 'User';

export type PtoLedgerEntryType =
  | 'Accrual'
  | 'Adjustment'
  | 'Carryover'
  | 'Expiry'
  | 'OpeningBalance'
  | 'Reversal'
  | 'Usage';

export type PtoPoliciesInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: PtoPolicyStatus | null | undefined;
};

export type PtoPolicyInput = {
  allowNegative: boolean;
  code: string;
  countWeekends: boolean;
  description?: string | null | undefined;
  enforceBalance: boolean;
  isDefault: boolean;
  name: string;
  negativeFloorDays?: string | null | undefined;
  requiresApproval: boolean;
  rules: Array<PtoPolicyRuleInput>;
  status: PtoPolicyStatus;
  version?: number | null | undefined;
  waitingPeriodDays: number;
  yearBasis: PtoYearBasis;
};

export type PtoPolicyRuleInput = {
  accrualAmountDays: string;
  accrualMethod: PtoAccrualMethod;
  carryoverCapDays?: string | null | undefined;
  carryoverExpiryDays?: number | null | undefined;
  maxBalanceDays?: string | null | undefined;
  onTermination?: PtoTerminationAction | null | undefined;
  ptoType: PtoType;
  tiers?: Array<PtoAccrualTierInput> | null | undefined;
};

export type PtoPolicyStatus =
  | 'Active'
  | 'Draft'
  | 'Inactive';

export type PtoStatus =
  | 'Approved'
  | 'Cancelled'
  | 'Rejected'
  | 'Requested';

export type PtoTerminationAction =
  | 'Forfeit'
  | 'PayOut';

export type PtoType =
  | 'Bereavement'
  | 'Holiday'
  | 'Maternity'
  | 'Paternity'
  | 'Personal'
  | 'Sick'
  | 'Vacation';

export type PtoYearBasis =
  | 'CalendarYear'
  | 'HireAnniversary';

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

export type PayrollExportStatus =
  | 'Draft'
  | 'Generated'
  | 'Voided';

export type PerformanceReviewStatus =
  | 'Acknowledged'
  | 'Closed'
  | 'Draft'
  | 'Submitted';

export type PerformanceReviewStatusInput = {
  id: string | number;
  version?: number | null | undefined;
};

export type PerformanceReviewTemplateInput = {
  cadenceMonths?: number | null | undefined;
  code: string;
  description?: string | null | undefined;
  isDefault: boolean;
  items: Array<ReviewItemInput>;
  name: string;
  status: EntityStatus;
  version?: number | null | undefined;
};

export type PerformanceReviewTemplatesInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: EntityStatus | null | undefined;
};

export type PeriodType =
  | 'Adjusting'
  | 'Month'
  | 'Quarter'
  | 'Week';

/**
 * Who a policy binds. A handbook for employees is not a contract term for an
 * owner-operator, and asking a contractor to sign one blurs a line the carrier's
 * lawyer would rather keep sharp.
 */
export type PolicyAudience =
  | 'All'
  | 'Contractors'
  | 'Employees';

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

/** Which roster a position holder comes from. */
export type PositionHolderKind =
  | 'User'
  | 'Worker';

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

export type ProfileChangeFilterInput = {
  limit?: number | null | undefined;
  statuses?: Array<ProfileChangeStatus> | null | undefined;
  /** Narrows the queue to the people a manager answers for. */
  teamOnly?: boolean | null | undefined;
  workerId?: string | number | null | undefined;
};

export type ProfileChangeStatus =
  | 'Approved'
  | 'Pending'
  | 'Rejected'
  | 'Withdrawn';

export type ProposeMyShiftSwapInput = {
  /** The day offered back, when the swap is a trade rather than a hand-off. */
  counterpartyShiftDate?: number | null | undefined;
  counterpartyWorkerId?: string | number | null | undefined;
  reason?: string | null | undefined;
  /** The day being given up. */
  shiftDate: number;
};

export type ProposeShiftSwapInput = {
  /** The day offered back, when the swap is a trade rather than a hand-off. */
  counterpartyShiftDate?: number | null | undefined;
  counterpartyWorkerId?: string | number | null | undefined;
  reason?: string | null | undefined;
  requestingWorkerId: string | number;
  /** The day being given up. */
  shiftDate: number;
};

export type RandomDrawStatus =
  | 'Cancelled'
  | 'Draft'
  | 'Final';

export type RandomEntryStatus =
  | 'Completed'
  | 'Excused'
  | 'Missed'
  | 'Notified'
  | 'Selected';

export type RandomPeriod =
  | 'Annual'
  | 'Monthly'
  | 'Quarterly'
  | 'SemiAnnual';

export type RateAgreementPartyType =
  | 'Carrier'
  | 'Customer';

export type RateAgreementStatus =
  | 'Active'
  | 'Archived'
  | 'Draft'
  | 'Expired'
  | 'InReview'
  | 'Suspended';

export type RateAgreementType =
  | 'Contract'
  | 'Dedicated'
  | 'Project'
  | 'Spot'
  | 'Tariff';

export type RateQuoteOutcome =
  | 'Error'
  | 'FormulaFallback'
  | 'ManualOverride'
  | 'NoRateFound'
  | 'Rated';

export type RateQuotePurpose =
  | 'Quote'
  | 'Rating'
  | 'Shopping'
  | 'Simulation'
  | 'WhatIf';

/** How a formula's computed charge is reduced to its billable precision. */
export type RateRoundingMode =
  | 'Down'
  | 'HalfEven'
  | 'HalfUp'
  | 'None'
  | 'Up';

export type RateUnit =
  | 'Day'
  | 'Hour'
  | 'Mile'
  | 'Stop';

export type RecognitionKind =
  | 'CustomerPraise'
  | 'Other'
  | 'Performance'
  | 'SafetyMilestone'
  | 'TeamPlayer'
  | 'Tenure';

export type RecordClearinghouseQueryInput = {
  completedAt?: number | null | undefined;
  consentExpiresAt?: number | null | undefined;
  consentObtainedAt?: number | null | undefined;
  documentId?: string | number | null | undefined;
  notes?: string | null | undefined;
  queryType: ClearinghouseQueryType;
  reference?: string | null | undefined;
  requestedAt?: number | null | undefined;
  result?: ClearinghouseResult | null | undefined;
  violationCount?: number | null | undefined;
  workerId: string | number;
};

export type RecordDotTestInput = {
  alcoholConcentration?: string | null | undefined;
  collectedAt?: number | null | undefined;
  collectionSite?: string | null | undefined;
  collectorName?: string | null | undefined;
  documentId?: string | number | null | undefined;
  drawEntryId?: string | number | null | undefined;
  isDot?: boolean | null | undefined;
  labName?: string | null | undefined;
  mroName?: string | null | undefined;
  mroVerifiedAt?: number | null | undefined;
  notes?: string | null | undefined;
  reason?: string | null | undefined;
  result?: DotTestResult | null | undefined;
  resultAt?: number | null | undefined;
  safetyEventId?: string | number | null | undefined;
  scheduledAt?: number | null | undefined;
  specimenId?: string | null | undefined;
  status?: DotTestStatus | null | undefined;
  substance: DotTestSubstance;
  testType: DotTestType;
  workerId: string | number;
};

export type RecordDotTestResultInput = {
  /**
   * Required for an alcohol test. The concentration decides the result, so a
   * reading at or above 0.04 is filed as positive whatever the caller sent.
   */
  alcoholConcentration?: string | null | undefined;
  documentId?: string | number | null | undefined;
  labName?: string | null | undefined;
  mroName?: string | null | undefined;
  mroVerifiedAt?: number | null | undefined;
  notes?: string | null | undefined;
  result: DotTestResult;
  resultAt?: number | null | undefined;
  testId: string | number;
};

export type RecordDotViolationInput = {
  documentId?: string | number | null | undefined;
  notes?: string | null | undefined;
  occurredAt: number;
  reportedToClearinghouseAt?: number | null | undefined;
  sapName?: string | null | undefined;
  sapReferredAt?: number | null | undefined;
  sourceTestId?: string | number | null | undefined;
  violationType: DotViolationType;
  workerId: string | number;
};

export type RecordEmploymentVerificationInput = {
  contactEmail?: string | null | undefined;
  contactName?: string | null | undefined;
  contactPhone?: string | null | undefined;
  employedFrom?: number | null | undefined;
  employedTo?: number | null | undefined;
  employerDotNumber?: string | null | undefined;
  employerMcNumber?: string | null | undefined;
  employerName: string;
  method?: EmploymentVerificationMethod | null | undefined;
  notes?: string | null | undefined;
  requestedAt?: number | null | undefined;
  status?: EmploymentVerificationStatus | null | undefined;
  wasDotRegulated?: boolean | null | undefined;
  workerId: string | number;
};

export type RecordLeaveCertificationInput = {
  caseId: string | number;
  documentId?: string | number | null | undefined;
  receivedAt?: number | null | undefined;
  recertificationDueAt?: number | null | undefined;
  status: LeaveCertificationStatus;
};

export type RecordLeaveDayInput = {
  caseId: string | number;
  hours: string;
  notes?: string | null | undefined;
  ptoId?: string | number | null | undefined;
  /** The day the leave was taken. */
  usedOn: number;
};

export type RecordMyStopActionInput = {
  action: PortalStopAction;
  moveId: string | number;
  stopId: string | number;
};

export type RecordSafetyViolationInput = {
  /**
   * Left empty, the BASIC falls back to what the event itself implies, so a clerk
   * keying an inspection does not have to classify every line to record one.
   */
  basic?: CsaBasic | null | undefined;
  code?: string | null | undefined;
  description: string;
  outOfService?: boolean | null | undefined;
  safetyEventId: string | number;
  severityWeight?: number | null | undefined;
};

export type RecordTimeEntryInput = {
  breakMinutes?: number | null | undefined;
  clockedInAt: number;
  clockedOutAt: number;
  /** Set to correct an existing entry; omit to add one. */
  id?: string | number | null | undefined;
  note?: string | null | undefined;
  payCodeId?: string | number | null | undefined;
  /** Required. A wage record changed with no reason recorded is not one anybody can defend. */
  reason: string;
  workerId: string | number;
};

export type RecordWorkerEmploymentEventInput = {
  documentId?: string | number | null | undefined;
  /** Promoted: the new driver type and/or worker type. */
  driverType?: DriverType | null | undefined;
  effectiveAt: number;
  /** Transferred: the destination fleet. Pass null to leave the worker unassigned. */
  fleetCodeId?: string | number | null | undefined;
  kind: WorkerEmploymentEventKind;
  /** LeaveStarted: what kind of leave this is. */
  leaveType?: WorkerLeaveType | null | undefined;
  managerId?: string | number | null | undefined;
  notes?: string | null | undefined;
  /** RateChanged: the new rate as entered, with an optional unit label. */
  rate?: string | null | undefined;
  rateUnit?: string | null | undefined;
  reason?: string | null | undefined;
  workerId: string | number;
  workerType?: WorkerType | null | undefined;
};

export type RecordWorkerInjuryInput = {
  bodyPart?: string | null | undefined;
  claimCarrier?: string | null | undefined;
  claimFiledAt?: number | null | undefined;
  claimNumber?: string | null | undefined;
  claimStatus?: WorkersCompClaimStatus | null | undefined;
  classification?: OshaCaseClassification | null | undefined;
  daysAway?: number | null | undefined;
  daysRestricted?: number | null | undefined;
  description: string;
  documentId?: string | number | null | undefined;
  harmfulAgent?: string | null | undefined;
  illnessType?: OshaIllnessType | null | undefined;
  location?: string | null | undefined;
  notes?: string | null | undefined;
  occurredAt: number;
  privacyCase?: boolean | null | undefined;
  reportedAt?: number | null | undefined;
  returnedToWorkAt?: number | null | undefined;
  safetyEventId?: string | number | null | undefined;
  treatment?: InjuryTreatment | null | undefined;
  workerId: string | number;
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

export type RemoveCarrierSettlementAdjustmentInput = {
  lineId: string | number;
  settlementId: string | number;
};

export type RemoveOrderChargeInput = {
  chargeId: string | number;
  orderId: string | number;
};

export type RemoveSettlementAdjustmentInput = {
  lineId: string | number;
  settlementId: string | number;
};

export type ReportBandInput = {
  edges?: Array<number> | null | undefined;
  width?: number | null | undefined;
};

export type ReportChartGoalInput = {
  columnId?: string | null | undefined;
  label?: string | null | undefined;
  value?: number | null | undefined;
};

export type ReportChartInput = {
  compareId?: string | null | undefined;
  curved?: boolean | null | undefined;
  goal?: ReportChartGoalInput | null | undefined;
  hideLegend?: boolean | null | undefined;
  id: string;
  labelColumnId?: string | null | undefined;
  latColumnId?: string | null | undefined;
  limit?: number | null | undefined;
  lngColumnId?: string | null | undefined;
  seriesIds?: Array<string> | null | undefined;
  showValues?: boolean | null | undefined;
  stacked?: boolean | null | undefined;
  title?: string | null | undefined;
  type: string;
  xColumnId?: string | null | undefined;
};

export type ReportColumnInput = {
  agg?: string | null | undefined;
  band?: ReportBandInput | null | undefined;
  bucket?: string | null | undefined;
  computed?: ReportComputedInput | null | undefined;
  display?: ReportDisplayInput | null | undefined;
  filter?: ReportFilterGroupInput | null | undefined;
  id: string;
  kind: string;
  label?: string | null | undefined;
  ref?: ReportFieldRefInput | null | undefined;
  transform?: ReportTransformInput | null | undefined;
};

export type ReportComputedInput = {
  format?: string | null | undefined;
  leftId?: string | null | undefined;
  leftValue?: number | null | undefined;
  op: string;
  rightId?: string | null | undefined;
  rightValue?: number | null | undefined;
};

export type ReportDashboardFilterInput = {
  default?: unknown;
  entity: string;
  id: string;
  label?: string | null | undefined;
  operator: string;
  ref: ReportFieldRefInput;
};

export type ReportDashboardLayoutInput = {
  filters?: Array<ReportDashboardFilterInput> | null | undefined;
  parameters?: Array<ReportParameterDefInput> | null | undefined;
  tiles: Array<ReportDashboardTileInput>;
};

export type ReportDashboardTileInput = {
  cannedKey?: string | null | undefined;
  chartId?: string | null | undefined;
  columnId?: string | null | undefined;
  definitionId?: string | number | null | undefined;
  h: number;
  id: string;
  kind: string;
  limit?: number | null | undefined;
  paramBindings?: unknown;
  text?: string | null | undefined;
  title?: string | null | undefined;
  w: number;
  x: number;
  y: number;
};

export type ReportDisplayInput = {
  boolStyle?: string | null | undefined;
  currency?: string | null | undefined;
  dateStyle?: string | null | undefined;
  decimals?: number | null | undefined;
  durationStyle?: string | null | undefined;
  durationUnit?: string | null | undefined;
  grouping?: boolean | null | undefined;
  negative?: string | null | undefined;
  notation?: string | null | undefined;
  nullText?: string | null | undefined;
  prefix?: string | null | undefined;
  rules?: Array<ReportDisplayRuleInput> | null | undefined;
  style?: string | null | undefined;
  suffix?: string | null | undefined;
};

export type ReportDisplayRuleInput = {
  op: string;
  tone: string;
  upper?: number | null | undefined;
  value: number;
};

export type ReportDrillInput = {
  columnId?: string | null | undefined;
  definition: ReportIrInput;
  dimensions: Array<ReportDrillValueInput>;
  limit?: number | null | undefined;
  params?: unknown;
};

export type ReportDrillValueInput = {
  columnId: string;
  value?: unknown;
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
  transform?: ReportTransformInput | null | undefined;
  value?: unknown;
};

export type ReportIrInput = {
  charts?: Array<ReportChartInput> | null | undefined;
  columns: Array<ReportColumnInput>;
  entity: string;
  filters?: ReportFilterGroupInput | null | undefined;
  having?: ReportFilterGroupInput | null | undefined;
  limit?: number | null | undefined;
  parameters?: Array<ReportParameterDefInput> | null | undefined;
  pivot?: ReportPivotInput | null | undefined;
  sort?: Array<ReportSortInput> | null | undefined;
  totals?: boolean | null | undefined;
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
  labels?: Array<string> | null | undefined;
  measureIds: Array<string>;
  ref: ReportFieldRefInput;
  values: Array<string>;
};

export type ReportRunsFilterInput = {
  definitionId?: string | number | null | undefined;
  mineOnly?: boolean | null | undefined;
  statuses?: Array<string> | null | undefined;
};

export type ReportScheduleAlertInput = {
  columnId?: string | null | undefined;
  operator: string;
  suppressWhileFiring?: boolean | null | undefined;
  threshold: number;
  value?: number | null | undefined;
};

export type ReportSortInput = {
  columnId: string;
  direction: string;
};

export type ReportTransformInput = {
  factor?: number | null | undefined;
  op: string;
  precision?: number | null | undefined;
};

export type RequestMyPtoInput = {
  endDate: number;
  reason: string;
  startDate: number;
  type: PortalPtoType;
};

export type RescindDisciplinaryActionInput = {
  id: string | number;
  reason: string;
  version?: number | null | undefined;
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

export type RespondToMyShiftSwapInput = {
  id: string | number;
  note?: string | null | undefined;
  response: MyShiftSwapResponse;
};

export type ReturnToDutyStatus =
  | 'Complete'
  | 'FollowUpTesting'
  | 'NotRequired'
  | 'RTDTestRequired'
  | 'SAPEvaluation';

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

export type ReviewGoalInput = {
  dueAt?: number | null | undefined;
  id?: string | null | undefined;
  status?: ReviewGoalStatus | null | undefined;
  title: string;
};

export type ReviewGoalStatus =
  | 'Done'
  | 'Dropped'
  | 'Open';

export type ReviewItemInput = {
  description?: string | null | undefined;
  key: string;
  label: string;
  weight: number;
};

export type ReviewRatingInput = {
  comment?: string | null | undefined;
  key: string;
  score?: number | null | undefined;
};

/**
 * What one worker is doing on one day. Anything above Scheduled overrides the
 * pattern, and the strongest override wins, so a driver on approved leave never
 * shows as rostered.
 */
export type RotaDayState =
  | 'Assigned'
  | 'Leave'
  | 'Off'
  | 'Scheduled'
  | 'TimeOff'
  | 'Unavailable';

export type RotaFilterInput = {
  /** Any instant in the week wanted; it resolves back to the Sunday that starts it. */
  at?: number | null | undefined;
  fleetCodeId?: string | number | null | undefined;
  limit?: number | null | undefined;
  /** Narrows the board to the people a manager answers for. */
  teamOnly?: boolean | null | undefined;
  weeks?: number | null | undefined;
  workerIds?: Array<string | number> | null | undefined;
};

export type RunDotRandomDrawInput = {
  /**
   * The moment the round is drawn for; leave empty for now. Naming it lets an
   * office catch up a period they missed.
   */
  at?: number | null | undefined;
  notes?: string | null | undefined;
  /** Leave empty to draw from the organisation's default pool. */
  poolId?: string | number | null | undefined;
};

export type RunPtoAccrualInput = {
  asOf?: number | null | undefined;
  rebuild?: boolean | null | undefined;
  workerId?: string | number | null | undefined;
};

export type RunReportInput = {
  cannedKey?: string | null | undefined;
  definitionId?: string | number | null | undefined;
  format: string;
  params?: unknown;
  viewId?: string | number | null | undefined;
};

export type SafetyEventKind =
  | 'Accident'
  | 'Citation'
  | 'Incident'
  | 'Inspection'
  | 'NearMiss';

export type SafetyEventStatus =
  | 'Closed'
  | 'Open'
  | 'UnderReview';

export type SafetyEventStatusInput = {
  id: string | number;
  resolution?: string | null | undefined;
  version?: number | null | undefined;
};

export type SafetyRating =
  | 'AtRisk'
  | 'Excellent'
  | 'Good'
  | 'Watch';

export type SafetySeverity =
  | 'Critical'
  | 'Major'
  | 'Minor'
  | 'Moderate';

export type SaveHomeLayoutPresetInput = {
  coreResponsibility?: string | null | undefined;
  description?: string | null | undefined;
  isOrgDefault: boolean;
  locked: boolean;
  name: string;
  priority: number;
  roleIds?: Array<string | number> | null | undefined;
  widgets: Array<HomeWidgetInput>;
};

export type SaveOshaSummaryInput = {
  averageEmployees?: number | null | undefined;
  executiveName?: string | null | undefined;
  executivePhone?: string | null | undefined;
  executiveTitle?: string | null | undefined;
  naicsCode?: string | null | undefined;
  notes?: string | null | undefined;
  postedFrom?: number | null | undefined;
  postedThrough?: number | null | undefined;
  submissionReference?: string | null | undefined;
  submittedAt?: number | null | undefined;
  totalHoursWorked?: number | null | undefined;
  year: number;
};

export type SaveReportDashboardInput = {
  category?: string | null | undefined;
  description?: string | null | undefined;
  layout: ReportDashboardLayoutInput;
  name: string;
  tags?: Array<string> | null | undefined;
  visibility?: string | null | undefined;
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

export type SaveTelematicsFormMappingInput = {
  description?: string | null | undefined;
  enabled: boolean;
  id?: string | number | null | undefined;
  items: Array<TelematicsFormMappingItemInput>;
  name: string;
  provider?: string | null | undefined;
  templateId: string;
  templateName?: string | null | undefined;
  version?: number | null | undefined;
};

export type SegregationType =
  | 'Barrier'
  | 'Distance'
  | 'Prohibited'
  | 'Separated';

export type SelectOptionResource =
  | 'ACCESSORIAL_CHARGE'
  | 'ACCOUNT_TYPE'
  | 'BENEFIT_PLAN'
  | 'CARRIER'
  | 'COMMODITY'
  | 'CUSTOMER'
  | 'DETENTION_POLICY'
  | 'DISTANCE_PROFILE'
  | 'DOCUMENT_TYPE'
  | 'EDI_COMMUNICATION_PROFILE'
  | 'EDI_CONNECTION'
  | 'EDI_DOCUMENT_TYPE'
  | 'EDI_MAPPING_PROFILE'
  | 'EDI_PARTNER'
  | 'EDI_PARTNER_DOCUMENT_PROFILE'
  | 'EDI_TEMPLATE'
  | 'EDI_TRANSACTION_SET'
  | 'EDI_TRANSFER'
  | 'EMAIL_PROFILE'
  | 'EQUIPMENT_MANUFACTURER'
  | 'EQUIPMENT_TYPE'
  | 'FISCAL_PERIOD'
  | 'FISCAL_YEAR'
  | 'FLEET_CODE'
  | 'FORMULA_TEMPLATE'
  | 'FUEL_CARD'
  | 'FUEL_INDEX'
  | 'FUEL_SURCHARGE_PROGRAM'
  | 'GL_ACCOUNT'
  | 'HAZARDOUS_MATERIAL'
  | 'IFTA_FUEL_TYPE'
  | 'IFTA_JURISDICTION'
  | 'JOB_POSITION'
  | 'LOCATION'
  | 'LOCATION_CATEGORY'
  | 'ORDER'
  | 'ORGANIZATION'
  | 'PAY_CODE'
  | 'PAY_PROFILE'
  | 'PERFORMANCE_REVIEW_TEMPLATE'
  | 'PTO_POLICY'
  | 'RATE_AGREEMENT'
  | 'RATE_MATRIX'
  | 'RATE_ZONE'
  | 'ROLE'
  | 'SERVICE_FAILURE_REASON_CODE'
  | 'SERVICE_TYPE'
  | 'SHIFT_TEMPLATE'
  | 'SHIPMENT'
  | 'SHIPMENT_TYPE'
  | 'TRACTOR'
  | 'TRAILER'
  | 'TRAINING_COURSE'
  | 'USER'
  | 'US_STATE'
  | 'WORKER'
  | 'WORKER_CREDENTIAL_TYPE'
  | 'WORKER_POLICY';

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

export type SetAvailabilityPreferenceInput = {
  dayOfWeek: number;
  note?: string | null | undefined;
  preference: AvailabilityPreference;
  workerId: string | number;
};

export type SetMyAvailabilityInput = {
  dayOfWeek: number;
  note?: string | null | undefined;
  preference: AvailabilityPreference;
};

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

/**
 * How far a swap has got. Two acceptances are needed — the colleague's and a
 * manager's — because a swap the office never saw is a shift nobody is covering.
 */
export type ShiftSwapStatus =
  | 'Accepted'
  | 'Approved'
  | 'Declined'
  | 'Proposed'
  | 'Rejected'
  | 'Withdrawn';

export type ShiftTemplateInput = {
  code: string;
  color?: string | null | undefined;
  cycleWeeks: number;
  daysOfWeek: string;
  description?: string | null | undefined;
  durationMinutes: number;
  name: string;
  startMinute: number;
  status?: EntityStatus | null | undefined;
};

export type ShipmentAdditionalChargeInput = {
  accessorialChargeId: string | number;
  amount?: string | null | undefined;
  detentionOccurrenceId?: string | number | null | undefined;
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
  attachmentDocumentIds?: Array<string | number> | null | undefined;
  /** Structured comment body (comment body v1). When present, the plain-text comment is derived server-side. */
  body?: unknown;
  /** Client-generated reference echoed on the created entity for optimistic-update reconciliation. */
  clientRef?: string | null | undefined;
  comment: string;
  mentionedUserIds?: Array<string | number> | null | undefined;
  parentCommentId?: string | number | null | undefined;
  priority?: ShipmentCommentPriority | null | undefined;
  requiresAcknowledgment?: boolean | null | undefined;
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
  attachmentDocumentIds?: Array<string | number> | null | undefined;
  /** Structured comment body (comment body v1). When present, the plain-text comment is derived server-side. */
  body?: unknown;
  comment: string;
  id: string | number;
  mentionedUserIds?: Array<string | number> | null | undefined;
  priority?: ShipmentCommentPriority | null | undefined;
  requiresAcknowledgment?: boolean | null | undefined;
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

export type ShipmentCommentsFilterInput = {
  authorIds?: Array<string | number> | null | undefined;
  mentionsUserId?: string | number | null | undefined;
  pinnedOnly?: boolean | null | undefined;
  priorities?: Array<ShipmentCommentPriority> | null | undefined;
  search?: string | null | undefined;
  types?: Array<ShipmentCommentType> | null | undefined;
  unresolvedOnly?: boolean | null | undefined;
};

export type ShipmentCommodityInput = {
  commodityId: string | number;
  heightFeet?: number | null | undefined;
  id?: string | number | null | undefined;
  lengthFeet?: number | null | undefined;
  pieces?: number | null | undefined;
  shipmentId?: string | number | null | undefined;
  version?: number | null | undefined;
  weight?: number | null | undefined;
  widthFeet?: number | null | undefined;
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
  | 'CarrierAssigned'
  | 'CarrierUnassigned'
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
  | 'RoutingGuideExhausted'
  | 'ShipmentCanceled'
  | 'ShipmentCreated'
  | 'ShipmentUncanceled'
  | 'ShipmentUpdated'
  | 'StatusChanged'
  | 'StopCompleted'
  | 'TenderAccepted'
  | 'TenderDeclined'
  | 'TenderDeliveryFailed'
  | 'TenderEntrySkipped'
  | 'TenderEntryWarned'
  | 'TenderExpired'
  | 'TenderLateResponse'
  | 'TenderNeedsReview'
  | 'TenderOffered'
  | 'TenderWithdrawn';

export type ShipmentEventsInput = {
  /** Only events recorded before this instant; pass the oldest occurredAt seen to page backwards. */
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
  /**
   * Why this shipment is billed at something other than its contract rate. It is
   * the only rating field a caller may write: everything else the rater owns is
   * an ordinary field, and everything else the system owns is restored on save.
   */
  rateOverrideReason?: string | null | undefined;
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

export type StageFuelPurchaseImportInput = {
  /**
   * The uploaded statement. The document must have been uploaded against this
   * batch (resource type fuel_purchase_import) or staging is refused.
   */
  documentId: string | number;
  id: string | number;
  mapping?: unknown;
};

export type StartWorkerChecklistInput = {
  startedAt?: number | null | undefined;
  templateId: string | number;
  workerId: string | number;
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

export type TelematicsFormMappingItemInput = {
  sourceFieldLabel: string;
  targetCustomFieldKey?: string | null | undefined;
  targetField?: string | null | undefined;
  targetKind: string;
};

export type TenderChannel =
  | 'EDI'
  | 'Email';

export type TenderMode =
  | 'SpotBroadcast'
  | 'SpotSequential'
  | 'Waterfall';

export type TenderOfferStatus =
  | 'Accepted'
  | 'Declined'
  | 'DeliveryFailed'
  | 'Expired'
  | 'Pending'
  | 'Sent'
  | 'Skipped'
  | 'Superseded'
  | 'Withdrawn';

export type TenderResponseSource =
  | 'EDI'
  | 'Email'
  | 'Manual';

export type TenderStatus =
  | 'Accepted'
  | 'Active'
  | 'Canceled'
  | 'Exhausted'
  | 'NeedsReview';

/**
 * Where a punch came from. It is kept because a manual entry and a clock punch
 * are different kinds of evidence at a wage-and-hour audit, and a screen that
 * cannot tell them apart cannot say which is which.
 */
export type TimeEntrySource =
  | 'Clock'
  | 'Import'
  | 'Manual'
  | 'Portal';

export type TimesheetFilterInput = {
  from?: number | null | undefined;
  limit?: number | null | undefined;
  statuses?: Array<TimesheetStatus> | null | undefined;
  /** Narrows the queue to the people a manager answers for. */
  teamOnly?: boolean | null | undefined;
  to?: number | null | undefined;
  unexportedOnly?: boolean | null | undefined;
  workerId?: string | number | null | undefined;
};

/** How far a week has got. */
export type TimesheetStatus =
  | 'Approved'
  | 'Locked'
  | 'Open'
  | 'Rejected'
  | 'Submitted';

export type TrainingCategory =
  | 'Compliance'
  | 'Equipment'
  | 'HazardousMaterials'
  | 'Orientation'
  | 'Other'
  | 'Safety';

export type TrainingCourseInput = {
  category: TrainingCategory;
  code: string;
  contentUrl?: string | null | undefined;
  delivery: TrainingDelivery;
  description?: string | null | undefined;
  dueDaysAfterAssignment: number;
  durationMinutes: number;
  isRequired: boolean;
  name: string;
  passingScore?: string | null | undefined;
  renewalWindowDays: number;
  requiredForDriverTypes?: Array<DriverType> | null | undefined;
  requiresAcknowledgement: boolean;
  sortOrder?: number | null | undefined;
  status: EntityStatus;
  validityMonths?: number | null | undefined;
  version?: number | null | undefined;
};

export type TrainingCoursesInput = {
  after?: string | null | undefined;
  category?: TrainingCategory | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: EntityStatus | null | undefined;
};

/**
 * How a course is taken. Online and Document courses can be finished from the
 * driver portal by acknowledging them; Classroom and OnTheJob need the office to
 * record the result.
 */
export type TrainingDelivery =
  | 'Classroom'
  | 'Document'
  | 'OnTheJob'
  | 'Online';

export type TransitionShiftSwapInput = {
  id: string | number;
  note?: string | null | undefined;
  status: ShiftSwapStatus;
};

export type TransitionTimesheetInput = {
  id: string | number;
  note?: string | null | undefined;
  status: TimesheetStatus;
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

export type UpdateBenefitPlanInput = {
  carrier?: string | null | undefined;
  code: string;
  description?: string | null | undefined;
  employeeCostMinor: number;
  employerCostMinor: number;
  id: string | number;
  name: string;
  payCodeId: string | number;
  planType: BenefitPlanType;
  planYear: number;
  policyNumber?: string | null | undefined;
  status?: EntityStatus | null | undefined;
  version?: number | null | undefined;
  waitingPeriodDays?: number | null | undefined;
};

export type UpdateCarrierSettlementControlInput = {
  autoAcceptWithinTolerance: boolean;
  autoGenerateBatches: boolean;
  autoMatchInboundInvoices: boolean;
  autoPostOnApprove: boolean;
  defaultApAccountId?: string | number | null | undefined;
  defaultPurchasedTransportationAccountId?: string | number | null | undefined;
  payDelayDays: number;
  payPeriodFrequency: PayPeriodFrequency;
  payTrigger: SettlementPayTrigger;
  periodEndDayOfWeek: number;
  varianceToleranceMinor: number;
  version: number;
};

export type UpdateDotRandomDrawEntryInput = {
  entryId: string | number;
  excuseReason?: string | null | undefined;
  status: RandomEntryStatus;
};

export type UpdateDotViolationInput = {
  documentId?: string | number | null | undefined;
  followUpEndsAt?: number | null | undefined;
  followUpTestCount?: number | null | undefined;
  notes?: string | null | undefined;
  reportedToClearinghouseAt?: number | null | undefined;
  sapEvaluationCompletedAt?: number | null | undefined;
  sapName?: string | null | undefined;
  sapReferredAt?: number | null | undefined;
  violationId: string | number;
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
  driverDigestCadence?: DriverDigestCadence | null | undefined;
  driverDigestWeekday?: number | null | undefined;
  enableDetentionAlerts: boolean;
  requireContactChangeApproval: boolean;
  requireExpenseReceipt: boolean;
  requireLoadAcknowledgment: boolean;
  sendCredentialReminders: boolean;
  showLoadPay: boolean;
  showPayEstimates: boolean;
  version: number;
};

export type UpdateEmploymentVerificationInput = {
  accidentCount?: number | null | undefined;
  contactEmail?: string | null | undefined;
  contactName?: string | null | undefined;
  contactPhone?: string | null | undefined;
  documentId?: string | number | null | undefined;
  drugAlcoholResponseReceivedAt?: number | null | undefined;
  employedFrom?: number | null | undefined;
  employedTo?: number | null | undefined;
  employerDotNumber?: string | null | undefined;
  employerMcNumber?: string | null | undefined;
  employerName?: string | null | undefined;
  findings?: string | null | undefined;
  hadAccidents?: boolean | null | undefined;
  hadDrugAlcoholViolations?: boolean | null | undefined;
  method?: EmploymentVerificationMethod | null | undefined;
  notes?: string | null | undefined;
  requestedAt?: number | null | undefined;
  responseReceivedAt?: number | null | undefined;
  status?: EmploymentVerificationStatus | null | undefined;
  verificationId: string | number;
  wasDotRegulated?: boolean | null | undefined;
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

export type UpdateHomeLayoutPresetInput = {
  coreResponsibility?: string | null | undefined;
  description?: string | null | undefined;
  id: string | number;
  isOrgDefault: boolean;
  locked: boolean;
  name: string;
  priority: number;
  roleIds?: Array<string | number> | null | undefined;
  version: number;
  widgets: Array<HomeWidgetInput>;
};

export type UpdateJobPositionInput = {
  code: string;
  department: JobDepartment;
  description?: string | null | undefined;
  flsaExempt?: boolean | null | undefined;
  id: string | number;
  isDrivingPosition?: boolean | null | undefined;
  reportsToPositionId?: string | number | null | undefined;
  status?: EntityStatus | null | undefined;
  title: string;
  version?: number | null | undefined;
};

export type UpdateLeaveCaseInput = {
  caseId: string | number;
  documentId?: string | number | null | undefined;
  eligibilityHoursWorked?: number | null | undefined;
  endsAt?: number | null | undefined;
  frequency?: LeaveFrequency | null | undefined;
  leaveType?: WorkerLeaveType | null | undefined;
  militaryCaregiver?: boolean | null | undefined;
  notes?: string | null | undefined;
  reason?: string | null | undefined;
  startsAt?: number | null | undefined;
};

export type UpdateLeaveControlInput = {
  certificationDueDays?: number | null | undefined;
  eligibilityHours?: number | null | undefined;
  eligibilityMonths?: number | null | undefined;
  entitlementWeeks?: string | null | undefined;
  measurementMethod?: LeaveMeasurementMethod | null | undefined;
  militaryCaregiverWeeks?: string | null | undefined;
  workweekHours?: string | null | undefined;
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

export type UpdatePerformanceReviewInput = {
  goals?: Array<ReviewGoalInput> | null | undefined;
  id: string | number;
  improvements?: string | null | undefined;
  periodEnd?: number | null | undefined;
  periodStart?: number | null | undefined;
  ratings: Array<ReviewRatingInput>;
  strengths?: string | null | undefined;
  summary?: string | null | undefined;
  title?: string | null | undefined;
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

export type UpdateReportDashboardInput = {
  category?: string | null | undefined;
  description?: string | null | undefined;
  id: string | number;
  layout: ReportDashboardLayoutInput;
  name: string;
  tags?: Array<string> | null | undefined;
  version: number;
  visibility?: string | null | undefined;
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
  alert?: ReportScheduleAlertInput | null | undefined;
  cronExpression: string;
  definitionId: string | number;
  emailAttach?: boolean | null | undefined;
  emailInline?: boolean | null | undefined;
  emailRecipients?: Array<string> | null | undefined;
  enabled: boolean;
  formats: Array<string>;
  id: string | number;
  notifyUserIds?: Array<string | number> | null | undefined;
  timezone?: string | null | undefined;
  version: number;
};

export type UpdateReportViewInput = {
  description?: string | null | undefined;
  format?: string | null | undefined;
  id: string | number;
  name: string;
  params?: unknown;
  pinned?: boolean | null | undefined;
  shared?: boolean | null | undefined;
  version: number;
};

export type UpdateSafetyViolationInput = {
  basic?: CsaBasic | null | undefined;
  code?: string | null | undefined;
  description: string;
  id: string | number;
  outOfService?: boolean | null | undefined;
  severityWeight?: number | null | undefined;
  version?: number | null | undefined;
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

export type UpdateWorkerCredentialInput = {
  documentId?: string | number | null | undefined;
  expiresAt?: number | null | undefined;
  id: string | number;
  issuedAt?: number | null | undefined;
  issuingAuthority?: string | null | undefined;
  notes?: string | null | undefined;
  number?: string | null | undefined;
  version: number;
};

export type UpdateWorkerInjuryInput = {
  bodyPart?: string | null | undefined;
  claimCarrier?: string | null | undefined;
  claimClosedAt?: number | null | undefined;
  claimFiledAt?: number | null | undefined;
  claimNumber?: string | null | undefined;
  claimStatus?: WorkersCompClaimStatus | null | undefined;
  classification?: OshaCaseClassification | null | undefined;
  daysAway?: number | null | undefined;
  daysRestricted?: number | null | undefined;
  description?: string | null | undefined;
  documentId?: string | number | null | undefined;
  harmfulAgent?: string | null | undefined;
  illnessType?: OshaIllnessType | null | undefined;
  injuryId: string | number;
  location?: string | null | undefined;
  notes?: string | null | undefined;
  occurredAt?: number | null | undefined;
  privacyCase?: boolean | null | undefined;
  reportedAt?: number | null | undefined;
  returnedToWorkAt?: number | null | undefined;
  safetyEventId?: string | number | null | undefined;
  status?: InjuryCaseStatus | null | undefined;
  treatment?: InjuryTreatment | null | undefined;
};

export type UpdateWorkerPtoInput = {
  endDate: number;
  id: string | number;
  reason: string;
  startDate: number;
  type: PtoType;
  version: number;
};

export type UpdateWorkerSafetyEventInput = {
  costAmount?: string | null | undefined;
  description: string;
  documentId?: string | number | null | undefined;
  fineAmount?: string | null | undefined;
  id: string | number;
  inspectionLevel?: number | null | undefined;
  inspectionResult?: InspectionResult | null | undefined;
  kind: SafetyEventKind;
  location?: string | null | undefined;
  occurredAt: number;
  outOfService?: boolean | null | undefined;
  points: number;
  pointsExpireAt?: number | null | undefined;
  preventable?: boolean | null | undefined;
  referenceNumber?: string | null | undefined;
  resolution?: string | null | undefined;
  severity: SafetySeverity;
  shipmentId?: string | number | null | undefined;
  version: number;
};

export type VoidPayrollExportInput = {
  id: string | number;
  reason: string;
};

export type WaiveWorkerTrainingInput = {
  id: string | number;
  reason: string;
  version?: number | null | undefined;
};

export type WorkerChecklistItemActionInput = {
  evidenceDocumentId?: string | number | null | undefined;
  id: string | number;
  note?: string | null | undefined;
  version?: number | null | undefined;
};

export type WorkerChecklistItemKind =
  | 'Credential'
  | 'Document'
  | 'Equipment'
  | 'PortalAccess'
  | 'Task';

export type WorkerChecklistItemStatus =
  | 'Done'
  | 'NotApplicable'
  | 'Pending'
  | 'Skipped';

export type WorkerChecklistKind =
  | 'Custom'
  | 'Offboarding'
  | 'Onboarding';

export type WorkerChecklistOwner =
  | 'Dispatch'
  | 'Fleet'
  | 'HR'
  | 'IT'
  | 'Payroll'
  | 'Safety';

export type WorkerChecklistStatus =
  | 'Cancelled'
  | 'Completed'
  | 'Open';

export type WorkerChecklistTemplateInput = {
  code: string;
  description?: string | null | undefined;
  isDefault: boolean;
  items: Array<WorkerChecklistTemplateItemInput>;
  kind: WorkerChecklistKind;
  name: string;
  status: EntityStatus;
  trigger: WorkerChecklistTrigger;
  version?: number | null | undefined;
};

export type WorkerChecklistTemplateItemInput = {
  credentialTypeId?: string | number | null | undefined;
  description?: string | null | undefined;
  documentTypeId?: string | number | null | undefined;
  dueOffsetDays: number;
  kind: WorkerChecklistItemKind;
  label: string;
  owner: WorkerChecklistOwner;
  required: boolean;
};

export type WorkerChecklistTemplatesInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  kind?: WorkerChecklistKind | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: EntityStatus | null | undefined;
  trigger?: WorkerChecklistTrigger | null | undefined;
};

export type WorkerChecklistTrigger =
  | 'Hired'
  | 'Manual'
  | 'Rehired'
  | 'Terminated';

export type WorkerConcernSeverity =
  | 'Critical'
  | 'Info'
  | 'Warning';

export type WorkerCredentialCategory =
  | 'Background'
  | 'Certification'
  | 'Endorsement'
  | 'License'
  | 'Medical'
  | 'Other'
  | 'Security';

/**
 * Evaluated state of one credential slot. Missing only appears on a summary item
 * for a required type the worker does not hold.
 */
export type WorkerCredentialHealth =
  | 'Expired'
  | 'ExpiringSoon'
  | 'Missing'
  | 'Valid';

export type WorkerCredentialInput = {
  credentialTypeId: string | number;
  documentId?: string | number | null | undefined;
  expiresAt?: number | null | undefined;
  issuedAt?: number | null | undefined;
  issuingAuthority?: string | null | undefined;
  notes?: string | null | undefined;
  number?: string | null | undefined;
  /**
   * Archive the worker's current active credential of this type so the new one
   * takes its slot. Without it a second active credential of a type is refused.
   */
  renew?: boolean | null | undefined;
  workerId: string | number;
};

export type WorkerCredentialStatus =
  | 'Active'
  | 'Archived';

export type WorkerCredentialTypeInput = {
  category: WorkerCredentialCategory;
  code: string;
  description?: string | null | undefined;
  isRequired: boolean;
  name: string;
  renewalWindowDays: number;
  requiredForDriverTypes?: Array<DriverType> | null | undefined;
  requiresDocument: boolean;
  requiresNumber: boolean;
  sortOrder?: number | null | undefined;
  status: EntityStatus;
  validityMonths?: number | null | undefined;
  version?: number | null | undefined;
};

export type WorkerCredentialTypesInput = {
  after?: string | null | undefined;
  category?: WorkerCredentialCategory | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: EntityStatus | null | undefined;
};

export type WorkerEmploymentEventKind =
  | 'Hired'
  | 'LeaveEnded'
  | 'LeaveStarted'
  | 'ProbationEnded'
  | 'Promoted'
  | 'RateChanged'
  | 'Rehired'
  | 'Reinstated'
  | 'Suspended'
  | 'Terminated'
  | 'Transferred';

export type WorkerGender =
  | 'Female'
  | 'Male';

export type WorkerLeaveType =
  | 'FMLA'
  | 'Medical'
  | 'Military'
  | 'Other'
  | 'Parental'
  | 'Personal';

export type WorkerPtoBulkAction =
  | 'Approve'
  | 'Cancel'
  | 'Reject';

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

export type WorkerPtoLedgerInput = {
  after?: string | null | undefined;
  effectiveFrom?: number | null | undefined;
  effectiveTo?: number | null | undefined;
  entryType?: PtoLedgerEntryType | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  includeWorker?: boolean | null | undefined;
  ptoType?: PtoType | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  workerId?: string | number | null | undefined;
};

export type WorkerPatchInput = {
  /** Omit to leave unchanged; null is rejected. */
  driverType?: DriverType | null | undefined;
  /** Omit to leave unchanged; null is rejected. */
  status?: EntityStatus | null | undefined;
  /** Omit to leave unchanged; null is rejected. */
  type?: WorkerType | null | undefined;
};

export type WorkerPolicyInput = {
  appliesTo: PolicyAudience;
  body?: string | null | undefined;
  code: string;
  documentId?: string | number | null | undefined;
  effectiveFrom?: number | null | undefined;
  requiresSignature: boolean;
  status?: EntityStatus | null | undefined;
  summary?: string | null | undefined;
  title: string;
  versionLabel: string;
};

export type WorkerRecognitionInput = {
  kind: RecognitionKind;
  message?: string | null | undefined;
  occurredAt?: number | null | undefined;
  title: string;
  visibleToWorker?: boolean | null | undefined;
  workerId: string | number;
};

export type WorkerSafetyEventInput = {
  costAmount?: string | null | undefined;
  description: string;
  documentId?: string | number | null | undefined;
  fineAmount?: string | null | undefined;
  inspectionLevel?: number | null | undefined;
  inspectionResult?: InspectionResult | null | undefined;
  kind: SafetyEventKind;
  location?: string | null | undefined;
  occurredAt: number;
  outOfService?: boolean | null | undefined;
  /** Leave null to take the default for the kind and severity. */
  points?: number | null | undefined;
  pointsExpireAt?: number | null | undefined;
  preventable?: boolean | null | undefined;
  referenceNumber?: string | null | undefined;
  severity: SafetySeverity;
  shipmentId?: string | number | null | undefined;
  workerId: string | number;
};

/**
 * The one-word answer to whether a worker is in good standing right now.
 * Blocked means they cannot be put on a load today; AtRisk means something on the
 * record has already lapsed; Watch means something is about to.
 */
export type WorkerStanding =
  | 'AtRisk'
  | 'Blocked'
  | 'Good'
  | 'Watch';

/**
 * Evaluated state of one course slot. Missing only appears on a summary item for
 * a required course the worker has no usable record for.
 */
export type WorkerTrainingHealth =
  | 'Current'
  | 'DueSoon'
  | 'Expired'
  | 'ExpiringSoon'
  | 'Failed'
  | 'Missing'
  | 'Overdue'
  | 'Scheduled';

export type WorkerTrainingStatus =
  | 'Assigned'
  | 'Cancelled'
  | 'Completed'
  | 'Expired'
  | 'Failed'
  | 'InProgress'
  | 'Waived';

export type WorkerType =
  | 'Contractor'
  | 'Employee';

export type WorkersCompClaimStatus =
  | 'Accepted'
  | 'Closed'
  | 'Denied'
  | 'Filed'
  | 'NotFiled';

export type WriteOffPayAdvanceInput = {
  advanceId: string | number;
  reason: string;
};

export type AccessorialChargeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, description: string, method: AccessorialMethod, rateUnit: RateUnit | null, amount: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'AccessorialChargeTableRowFieldsFragment' };

export type AccessorialChargeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type AccessorialChargeTableQuery = { accessorialCharges: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AccessorialChargeTableRowFieldsFragment': AccessorialChargeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type AccountTypeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, name: string, description: string | null, category: AccountCategory, color: string | null, isSystem: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'AccountTypeTableRowFieldsFragment' };

export type AccountTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type AccountTypeTableQuery = { accountTypes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AccountTypeTableRowFieldsFragment': AccountTypeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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

export type AgentControlFieldsFragment = { id: string, organizationId: string, businessUnitId: string, shadowMode: boolean, billingAgentEnabled: boolean, decisionTimeoutSeconds: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'AgentControlFieldsFragment' };

export type AgentControlSettingsQueryVariables = Exact<{ [key: string]: never; }>;


export type AgentControlSettingsQuery = { agentControl: { ' $fragmentRefs'?: { 'AgentControlFieldsFragment': AgentControlFieldsFragment } } };

export type UpdateAgentControlMutationVariables = Exact<{
  input: AgentControlInput;
}>;


export type UpdateAgentControlMutation = { updateAgentControl: { ' $fragmentRefs'?: { 'AgentControlFieldsFragment': AgentControlFieldsFragment } } };

export type AgentExceptionTableRowFieldsFragment = { id: string, organizationId: string, businessUnitId: string, runId: string, category: AgentExceptionCategory, severity: AgentSeverity, subjectType: AgentSubjectType, subjectId: string, attemptSummary: string, blastRadius: number, resolutionState: AgentResolutionState, resolutionNotes: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'AgentExceptionTableRowFieldsFragment' };

export type AgentExceptionDetailFieldsFragment = (
  { evidence: Array<{ ' $fragmentRefs'?: { 'AgentEvidenceRefFieldsFragment': AgentEvidenceRefFieldsFragment } }> }
  & { ' $fragmentRefs'?: { 'AgentExceptionTableRowFieldsFragment': AgentExceptionTableRowFieldsFragment } }
) & { ' $fragmentName'?: 'AgentExceptionDetailFieldsFragment' };

export type AgentExceptionTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type AgentExceptionTableQuery = { agentExceptions: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AgentExceptionTableRowFieldsFragment': AgentExceptionTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type AgentExceptionDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type AgentExceptionDetailQuery = { agentException: { ' $fragmentRefs'?: { 'AgentExceptionDetailFieldsFragment': AgentExceptionDetailFieldsFragment } } | null };

export type ResolveAgentExceptionMutationVariables = Exact<{
  id: string | number;
  input: AgentExceptionResolveInput;
}>;


export type ResolveAgentExceptionMutation = { resolveAgentException: { id: string, resolutionState: AgentResolutionState, resolutionNotes: string, version: number, updatedAt: number } };

export type AgentEvidenceRefFieldsFragment = { type: string, id: string, note: string } & { ' $fragmentName'?: 'AgentEvidenceRefFieldsFragment' };

export type AgentProposalTableRowFieldsFragment = { id: string, organizationId: string, businessUnitId: string, runId: string, toolName: string, toolParams: unknown, confidence: number, rationale: string, autonomyTier: AgentAutonomyTier, status: AgentProposalStatus, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'AgentProposalTableRowFieldsFragment' };

export type AgentProposalDetailFieldsFragment = (
  { evidence: Array<{ ' $fragmentRefs'?: { 'AgentEvidenceRefFieldsFragment': AgentEvidenceRefFieldsFragment } }> }
  & { ' $fragmentRefs'?: { 'AgentProposalTableRowFieldsFragment': AgentProposalTableRowFieldsFragment } }
) & { ' $fragmentName'?: 'AgentProposalDetailFieldsFragment' };

export type AgentProposalTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type AgentProposalTableQuery = { agentProposals: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AgentProposalTableRowFieldsFragment': AgentProposalTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type AgentProposalDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type AgentProposalDetailQuery = { agentProposal: { ' $fragmentRefs'?: { 'AgentProposalDetailFieldsFragment': AgentProposalDetailFieldsFragment } } | null };

export type DecideAgentProposalMutationVariables = Exact<{
  id: string | number;
  input: AgentProposalDecisionInput;
}>;


export type DecideAgentProposalMutation = { decideAgentProposal: { id: string, proposalId: string | null, decision: AgentDecisionType, reasonCode: string, decidedByUserId: string, version: number, createdAt: number } };

export type AgentRunTableRowFieldsFragment = { id: string, organizationId: string, businessUnitId: string, agentType: AgentType, subjectType: AgentSubjectType, subjectId: string, status: AgentRunStatus, workflowId: string, modelIdentifier: string, promptVersion: string, startedAt: number | null, completedAt: number | null, errorMessage: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'AgentRunTableRowFieldsFragment' };

export type AgentRunTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type AgentRunTableQuery = { agentRuns: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AgentRunTableRowFieldsFragment': AgentRunTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type AgentRunDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type AgentRunDetailQuery = { agentRun: (
    { inputContextHash: string }
    & { ' $fragmentRefs'?: { 'AgentRunTableRowFieldsFragment': AgentRunTableRowFieldsFragment } }
  ) | null };

export type ApiKeyTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, keyPrefix: string, status: string, expiresAt: number, lastUsedAt: number, permissionScope: string, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ApiKeyTableRowFieldsFragment' };

export type ApiKeyTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ApiKeyTableQuery = { apiKeys: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ApiKeyTableRowFieldsFragment': ApiKeyTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type AuditLogTableQuery = { auditEntries: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'AuditLogTableRowFieldsFragment': AuditLogTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type BenefitPlansQueryVariables = Exact<{
  activeOnly?: boolean | null | undefined;
  planYear?: number | null | undefined;
}>;


export type BenefitPlansQuery = { benefitPlans: Array<{ id: string, status: EntityStatus, code: string, name: string, description: string | null, planType: BenefitPlanType, carrier: string | null, policyNumber: string | null, payCodeId: string, planYear: number, employeeCostMinor: number, employerCostMinor: number, currencyCode: string, waitingPeriodDays: number, version: number, payCode: { id: string, code: string, description: string } | null }> };

export type WorkerBenefitEnrollmentsQueryVariables = Exact<{
  workerId: string | number;
  openOnly?: boolean | null | undefined;
}>;


export type WorkerBenefitEnrollmentsQuery = { workerBenefitEnrollments: Array<{ id: string, workerId: string, benefitPlanId: string, status: BenefitEnrollmentStatus, coverageTier: CoverageTier, effectiveFrom: number, effectiveTo: number | null, employeeCostMinor: number, employerCostMinor: number, recurringDeductionId: string | null, waivedReason: string | null, notes: string | null, version: number, benefitPlan: { id: string, code: string, name: string, planType: BenefitPlanType, carrier: string | null } | null }> };

export type BenefitEnrollmentsQueryVariables = Exact<{
  planId?: string | number | null | undefined;
  statuses?: Array<BenefitEnrollmentStatus> | BenefitEnrollmentStatus | null | undefined;
  openOnly?: boolean | null | undefined;
  limit?: number | null | undefined;
}>;


export type BenefitEnrollmentsQuery = { benefitEnrollments: Array<{ id: string, workerId: string, benefitPlanId: string, status: BenefitEnrollmentStatus, coverageTier: CoverageTier, effectiveFrom: number, effectiveTo: number | null, employeeCostMinor: number, employerCostMinor: number, waivedReason: string | null, notes: string | null, version: number, benefitPlan: { id: string, code: string, name: string, planType: BenefitPlanType, planYear: number, currencyCode: string } | null, worker: { id: string, firstName: string, lastName: string, profilePicUrl: string, fleetCode: { id: string, code: string, color: string } | null } | null }> };

export type BenefitCostsQueryVariables = Exact<{
  planYear?: number | null | undefined;
}>;


export type BenefitCostsQuery = { benefitCosts: Array<{ planId: string, planName: string, planType: string, enrolled: number, waived: number, employeeCostMinor: number, employerCostMinor: number }> };

export type WorkerTotalCompensationQueryVariables = Exact<{
  workerId: string | number;
  planYear?: number | null | undefined;
}>;


export type WorkerTotalCompensationQuery = { workerTotalCompensation: { asOf: number, planYear: number, grossPayMinor: number, employerBenefitMinor: number, employeeBenefitMinor: number, accruedTimeOffDays: string, totalCompensationMinor: number, enrollments: Array<{ id: string, status: BenefitEnrollmentStatus, coverageTier: CoverageTier, employeeCostMinor: number, employerCostMinor: number, benefitPlan: { id: string, name: string, planType: BenefitPlanType } | null }> } };

export type MyTotalCompensationQueryVariables = Exact<{
  planYear?: number | null | undefined;
}>;


export type MyTotalCompensationQuery = { myTotalCompensation: { asOf: number, planYear: number, grossPayMinor: number, employerBenefitMinor: number, employeeBenefitMinor: number, accruedTimeOffDays: string, totalCompensationMinor: number, enrollments: Array<{ id: string, status: BenefitEnrollmentStatus, coverageTier: CoverageTier, employeeCostMinor: number, employerCostMinor: number, benefitPlan: { id: string, name: string, planType: BenefitPlanType } | null }> } };

export type CreateBenefitPlanMutationVariables = Exact<{
  input: BenefitPlanInput;
}>;


export type CreateBenefitPlanMutation = { createBenefitPlan: { id: string, name: string, version: number } };

export type UpdateBenefitPlanMutationVariables = Exact<{
  input: UpdateBenefitPlanInput;
}>;


export type UpdateBenefitPlanMutation = { updateBenefitPlan: { id: string, name: string, version: number } };

export type EnrollBenefitMutationVariables = Exact<{
  input: EnrollBenefitInput;
}>;


export type EnrollBenefitMutation = { enrollBenefit: { id: string, status: BenefitEnrollmentStatus, employeeCostMinor: number, version: number } };

export type EndBenefitEnrollmentMutationVariables = Exact<{
  input: EndBenefitEnrollmentInput;
}>;


export type EndBenefitEnrollmentMutation = { endBenefitEnrollment: { id: string, status: BenefitEnrollmentStatus, effectiveTo: number | null, version: number } };

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

export type CarrierSettlementTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type CarrierSettlementTableQuery = { carrierSettlements: { totalCount?: number | null, edges: Array<{ node: { id: string, carrierId: string, batchId: string | null, settlementNumber: string, status: CarrierSettlementStatus, periodStart: number, periodEnd: number, payDate: number, grossCostMinor: number, adjustmentsMinor: number, netPayableMinor: number, shipmentCount: number, currencyCode: string, paymentMethod: string, paymentReference: string, version: number, createdAt: number, updatedAt: number, carrier: { id: string, code: string, name: string, scac: string | null } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CarrierSettlementDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type CarrierSettlementDetailQuery = { carrierSettlement: { id: string, carrierId: string, batchId: string | null, settlementNumber: string, status: CarrierSettlementStatus, periodStart: number, periodEnd: number, payDate: number, grossCostMinor: number, adjustmentsMinor: number, netPayableMinor: number, shipmentCount: number, currencyCode: string, notes: string, submittedById: string | null, submittedAt: number | null, approvedById: string | null, approvedAt: number | null, postedById: string | null, postedAt: number | null, postedJournalBatchId: string | null, paidAt: number | null, paidById: string | null, paymentMethod: string, paymentReference: string, paidJournalBatchId: string | null, voidedById: string | null, voidedAt: number | null, voidReason: string, voidJournalBatchId: string | null, version: number, createdAt: number, updatedAt: number, carrier: { id: string, code: string, name: string, scac: string | null, paymentMethod: CarrierPaymentMethod, paymentTermDays: number, remitToName: string | null, remitAddressLine1: string | null, remitAddressLine2: string | null, remitCity: string | null, remitPostalCode: string | null, remitState: { id: string, abbreviation: string } | null } | null, lines: Array<{ id: string, lineNumber: number, eventType: CarrierCostEventType, description: string, amountMinor: number, costEventId: string | null, glAccountId: string | null, shipmentId: string | null, moveId: string | null, proNumber: string }> | null } | null };

export type CarrierSettlementBatchTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type CarrierSettlementBatchTableQuery = { carrierSettlementBatches: { totalCount?: number | null, edges: Array<{ node: { id: string, status: CarrierSettlementBatchStatus, name: string, periodStart: number, periodEnd: number, payDate: number, settlementCount: number, totalGrossMinor: number, totalNetMinor: number, currencyCode: string, notes: string, generatedById: string | null, generatedAt: number | null, completedAt: number | null, canceledAt: number | null, version: number, createdAt: number, updatedAt: number } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CarrierSettlementBatchDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type CarrierSettlementBatchDetailQuery = { carrierSettlementBatch: { id: string, status: CarrierSettlementBatchStatus, name: string, periodStart: number, periodEnd: number, payDate: number, settlementCount: number, totalGrossMinor: number, totalNetMinor: number, currencyCode: string, notes: string, version: number, settlements: Array<{ id: string, settlementNumber: string, status: CarrierSettlementStatus, grossCostMinor: number, adjustmentsMinor: number, netPayableMinor: number, currencyCode: string, carrier: { id: string, code: string, name: string, scac: string | null } | null }> | null } | null };

export type CarrierCostEventTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type CarrierCostEventTableQuery = { carrierCostEvents: { totalCount?: number | null, edges: Array<{ node: { id: string, carrierId: string, carrierAssignmentId: string | null, shipmentId: string | null, moveId: string | null, settlementId: string | null, eventType: CarrierCostEventType, status: CarrierCostEventStatus, eventDate: number, amountMinor: number, currencyCode: string, description: string, proNumber: string, assignmentVersion: number, voidedAt: number | null, voidReason: string, version: number, createdAt: number, updatedAt: number, carrier: { id: string, code: string, name: string, scac: string | null } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CarrierSettlementControlQueryVariables = Exact<{ [key: string]: never; }>;


export type CarrierSettlementControlQuery = { carrierSettlementControl: { id: string, organizationId: string, businessUnitId: string, payTrigger: SettlementPayTrigger, payPeriodFrequency: PayPeriodFrequency, periodEndDayOfWeek: number, payDelayDays: number, autoGenerateBatches: boolean, autoPostOnApprove: boolean, varianceToleranceMinor: number, autoMatchInboundInvoices: boolean, autoAcceptWithinTolerance: boolean, defaultApAccountId: string | null, defaultPurchasedTransportationAccountId: string | null, version: number } };

export type CurrentCarrierSettlementPeriodQueryVariables = Exact<{ [key: string]: never; }>;


export type CurrentCarrierSettlementPeriodQuery = { currentCarrierSettlementPeriod: { periodStart: number, periodEnd: number, payDate: number } };

export type CarrierSettlementWorkspaceSummaryQueryVariables = Exact<{
  periodStart?: number | null | undefined;
  periodEnd?: number | null | undefined;
}>;


export type CarrierSettlementWorkspaceSummaryQuery = { carrierSettlementWorkspaceSummary: { periodStart: number, periodEnd: number, payDate: number, draftCount: number, pendingApprovalCount: number, approvedCount: number, postedCount: number, paidCount: number, totalNetMinor: number, totalGrossMinor: number, pendingEventCount: number, pendingAmountMinor: number, pendingCarrierCount: number, openBatchId: string | null } };

export type CarrierLedgerEntriesQueryVariables = Exact<{
  carrierId: string | number;
  limit?: number | null | undefined;
}>;


export type CarrierLedgerEntriesQuery = { carrierLedgerEntries: Array<{ id: string, carrierId: string, entryType: CarrierLedgerEntryType, sourceObjectType: string, sourceObjectId: string, sourceEventType: string, relatedSettlementId: string | null, journalBatchId: string | null, documentNumber: string, transactionDate: number, lineNumber: number, amountMinor: number, createdAt: number }> };

export type CarrierInvoiceMatchesQueryVariables = Exact<{
  status?: CarrierInvoiceMatchStatus | null | undefined;
  carrierId?: string | number | null | undefined;
  limit?: number | null | undefined;
  offset?: number | null | undefined;
}>;


export type CarrierInvoiceMatchesQuery = { carrierInvoiceMatches: { totalCount: number, items: Array<{ id: string, ediCarrierInvoiceId: string | null, documentAiExtractionId: string | null, carrierId: string, carrierAssignmentId: string, carrierSettlementId: string | null, adjustmentCostEventId: string | null, status: CarrierInvoiceMatchStatus, matchedVia: CarrierInvoiceMatchVia, invoiceNumber: string, invoiceTotalMinor: number, expectedTotalMinor: number, varianceMinor: number, currencyCode: string, resolutionNote: string, resolvedById: string | null, resolvedAt: number | null, version: number, createdAt: number, updatedAt: number, carrier: { id: string, code: string, name: string, scac: string | null } | null, carrierAssignment: { id: string, shipmentMoveId: string, status: CarrierAssignmentStatus, rateMethod: CarrierRateMethod, baseRate: string, baseAmount: string, fuelSurcharge: string, accessorialTotal: string, totalCost: string, currencyCode: string, proNumber: string | null, accessorials: Array<{ id: string, description: string, amount: string }> | null } | null }> } };

export type EdiCarrierInvoicesQueryVariables = Exact<{
  reconciliationStatus?: string | null | undefined;
  limit?: number | null | undefined;
  offset?: number | null | undefined;
}>;


export type EdiCarrierInvoicesQuery = { ediCarrierInvoices: { totalCount: number, items: Array<{ id: string, carrierId: string | null, shipmentId: string | null, invoiceNumber: string, invoiceDate: number | null, deliveryDate: number | null, shipmentReference: string, bol: string, proNumber: string, billToName: string, currencyCode: string, totalAmount: string | null, expectedAmount: string | null, varianceAmount: string | null, reconciliationStatus: string, reconciliationNotes: string, version: number, createdAt: number, updatedAt: number }> } };

export type SuggestCarrierForEdiInvoiceQueryVariables = Exact<{
  invoiceId: string | number;
}>;


export type SuggestCarrierForEdiInvoiceQuery = { suggestCarrierForEdiInvoice: { id: string, code: string, name: string, scac: string | null, dotNumber: string | null } | null };

export type ExportCarrierSettlementBatchCsvQueryVariables = Exact<{
  batchId: string | number;
}>;


export type ExportCarrierSettlementBatchCsvQuery = { exportCarrierSettlementBatchCsv: string };

export type GenerateCarrierSettlementBatchMutationVariables = Exact<{
  input: GenerateCarrierSettlementBatchInput;
}>;


export type GenerateCarrierSettlementBatchMutation = { generateCarrierSettlementBatch: { id: string, name: string, settlementCount: number, totalGrossMinor: number, totalNetMinor: number } };

export type SubmitCarrierSettlementMutationVariables = Exact<{
  input: CarrierSettlementActionInput;
}>;


export type SubmitCarrierSettlementMutation = { submitCarrierSettlement: { id: string, status: CarrierSettlementStatus, version: number } };

export type ApproveCarrierSettlementMutationVariables = Exact<{
  input: CarrierSettlementActionInput;
}>;


export type ApproveCarrierSettlementMutation = { approveCarrierSettlement: { id: string, status: CarrierSettlementStatus, version: number } };

export type RejectCarrierSettlementMutationVariables = Exact<{
  input: CarrierSettlementActionInput;
}>;


export type RejectCarrierSettlementMutation = { rejectCarrierSettlement: { id: string, status: CarrierSettlementStatus, version: number } };

export type PostCarrierSettlementMutationVariables = Exact<{
  input: CarrierSettlementActionInput;
}>;


export type PostCarrierSettlementMutation = { postCarrierSettlement: { id: string, status: CarrierSettlementStatus, version: number } };

export type MarkCarrierSettlementPaidMutationVariables = Exact<{
  input: MarkCarrierSettlementPaidInput;
}>;


export type MarkCarrierSettlementPaidMutation = { markCarrierSettlementPaid: { id: string, status: CarrierSettlementStatus, version: number } };

export type VoidCarrierSettlementMutationVariables = Exact<{
  input: CarrierSettlementActionInput;
}>;


export type VoidCarrierSettlementMutation = { voidCarrierSettlement: { id: string, status: CarrierSettlementStatus, version: number } };

export type RecalculateCarrierSettlementMutationVariables = Exact<{
  input: CarrierSettlementActionInput;
}>;


export type RecalculateCarrierSettlementMutation = { recalculateCarrierSettlement: { id: string, version: number } };

export type AddCarrierSettlementAdjustmentMutationVariables = Exact<{
  input: AddCarrierSettlementAdjustmentInput;
}>;


export type AddCarrierSettlementAdjustmentMutation = { addCarrierSettlementAdjustment: { id: string, version: number } };

export type RemoveCarrierSettlementAdjustmentMutationVariables = Exact<{
  input: RemoveCarrierSettlementAdjustmentInput;
}>;


export type RemoveCarrierSettlementAdjustmentMutation = { removeCarrierSettlementAdjustment: { id: string, version: number } };

export type UpdateCarrierSettlementControlMutationVariables = Exact<{
  input: UpdateCarrierSettlementControlInput;
}>;


export type UpdateCarrierSettlementControlMutation = { updateCarrierSettlementControl: { id: string, version: number } };

export type LinkEdiCarrierInvoiceToCarrierMutationVariables = Exact<{
  invoiceId: string | number;
  carrierId: string | number;
}>;


export type LinkEdiCarrierInvoiceToCarrierMutation = { linkEdiCarrierInvoiceToCarrier: { id: string, carrierId: string | null, reconciliationStatus: string, version: number } };

export type CreateCarrierInvoiceMatchMutationVariables = Exact<{
  input: CreateCarrierInvoiceMatchInput;
}>;


export type CreateCarrierInvoiceMatchMutation = { createCarrierInvoiceMatch: { id: string, status: CarrierInvoiceMatchStatus, invoiceTotalMinor: number, expectedTotalMinor: number, varianceMinor: number, version: number } };

export type AcceptCarrierInvoiceMatchMutationVariables = Exact<{
  input: CarrierInvoiceMatchActionInput;
}>;


export type AcceptCarrierInvoiceMatchMutation = { acceptCarrierInvoiceMatch: { id: string, status: CarrierInvoiceMatchStatus, version: number } };

export type AcceptCarrierInvoiceMatchWithVarianceMutationVariables = Exact<{
  input: CarrierInvoiceMatchActionInput;
}>;


export type AcceptCarrierInvoiceMatchWithVarianceMutation = { acceptCarrierInvoiceMatchWithVariance: { id: string, status: CarrierInvoiceMatchStatus, adjustmentCostEventId: string | null, version: number } };

export type RejectCarrierInvoiceMatchMutationVariables = Exact<{
  input: CarrierInvoiceMatchActionInput;
}>;


export type RejectCarrierInvoiceMatchMutation = { rejectCarrierInvoiceMatch: { id: string, status: CarrierInvoiceMatchStatus, version: number } };

export type CarrierContactFieldsFragment = { id: string, businessUnitId: string, organizationId: string, carrierId: string, name: string, title: string | null, email: string | null, phone: string | null, isPrimary: boolean, receivesRateConfirmations: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CarrierContactFieldsFragment' };

export type CarrierInsurancePolicyFieldsFragment = { id: string, businessUnitId: string, organizationId: string, carrierId: string, policyType: CarrierInsurancePolicyType, policyNumber: string, providerName: string, coverageAmount: string, effectiveDate: number, expirationDate: number, isVerified: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CarrierInsurancePolicyFieldsFragment' };

export type CarrierTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, stateId: string | null, remitStateId: string | null, status: CarrierStatus, code: string, name: string, dbaName: string | null, carrierType: CarrierType, dotNumber: string | null, mcNumber: string | null, scac: string | null, complianceStatus: CarrierComplianceStatus, safetyRating: CarrierSafetyRating, qualifiedAt: number | null, disqualifiedReason: string | null, taxId: string | null, taxIdType: CarrierTaxIdType | null, w9OnFile: boolean, is1099Eligible: boolean, paymentMethod: CarrierPaymentMethod, paymentTermDays: number, remitToName: string | null, remitAddressLine1: string | null, remitAddressLine2: string | null, remitCity: string | null, remitPostalCode: string | null, addressLine1: string | null, addressLine2: string | null, city: string | null, postalCode: string | null, phone: string | null, email: string | null, externalId: string | null, notes: string | null, version: number, createdAt: number, updatedAt: number, contacts: Array<{ ' $fragmentRefs'?: { 'CarrierContactFieldsFragment': CarrierContactFieldsFragment } }> | null, insurancePolicies: Array<{ ' $fragmentRefs'?: { 'CarrierInsurancePolicyFieldsFragment': CarrierInsurancePolicyFieldsFragment } }> | null } & { ' $fragmentName'?: 'CarrierTableRowFieldsFragment' };

export type CarrierTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type CarrierTableQuery = { carriers: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'CarrierTableRowFieldsFragment': CarrierTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CommodityTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, hazardousMaterialId: string | null, status: EntityStatus, name: string, description: string, minTemperature: number | null, maxTemperature: number | null, weightPerUnit: number | null, linearFeetPerUnit: number | null, maxQuantityPerShipment: number | null, freightClass: FreightClass | null, loadingInstructions: string | null, stackable: boolean, fragile: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CommodityTableRowFieldsFragment' };

export type CommodityTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type CommodityTableQuery = { commodities: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'CommodityTableRowFieldsFragment': CommodityTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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

export type CustomFieldDefinitionTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, resourceType: string, name: string, label: string, description: string | null, fieldType: FieldType, isRequired: boolean, isActive: boolean, displayOrder: number, color: string | null, defaultValue: unknown, version: number, createdAt: number, updatedAt: number, options: Array<{ value: string, label: string, color: string, description: string }>, validationRules: { minLength: number | null, maxLength: number | null, min: number | null, max: number | null, pattern: string | null } | null, uiAttributes: { placeholder: string, helpText: string, width: string } | null } & { ' $fragmentName'?: 'CustomFieldDefinitionTableRowFieldsFragment' };

export type CustomFieldDefinitionTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type CustomFieldDefinitionTableQuery = { customFieldDefinitions: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'CustomFieldDefinitionTableRowFieldsFragment': CustomFieldDefinitionTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CustomerPaymentTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type CustomerPaymentTableQuery = { customerPayments: { totalCount?: number | null, edges: Array<{ node: { id: string, organizationId: string, businessUnitId: string, customerId: string, paymentDate: number, accountingDate: number, amountMinor: number, appliedAmountMinor: number, unappliedAmountMinor: number, status: CustomerPaymentStatus, paymentMethod: CustomerPaymentMethod, referenceNumber: string, memo: string, currencyCode: string, postedBatchId: string | null, reversalBatchId: string | null, reversedById: string | null, reversedAt: number | null, reversalReason: string, createdById: string, updatedById: string | null, version: number, createdAt: number, updatedAt: number, customer: { id: string, code: string, name: string } | null, applications: Array<{ id: string, customerPaymentId: string, invoiceId: string, appliedAmountMinor: number, shortPayAmountMinor: number, lineNumber: number, createdAt: number, updatedAt: number }> | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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

export type CustomerBillingProfileFieldsFragment = { id: string, businessUnitId: string, organizationId: string, customerId: string, invoiceDelivery: CustomerInvoiceDelivery, billingCycle: CustomerBillingCycle, billingCycleAnchorDay: number, billingCycleTimezone: string, lastBilledPeriodEnd: number | null, paymentTerm: CustomerPaymentTerm, hasBillingControlOverrides: boolean, creditLimit: string | null, creditBalance: string, creditStatus: CustomerCreditStatus, enforceCreditLimit: boolean, autoCreditHold: boolean, creditHoldReason: string, autoSendInvoiceOnGeneration: boolean, splitBy: InvoiceSplitKey, sectionBy: InvoiceSectionKey, invoiceDetail: InvoiceDetail, minConsolidatedAmount: string | null, maxShipmentsPerInvoice: number, invoiceNumberFormat: CustomerInvoiceNumberFormat, customerInvoicePrefix: string, invoiceCopies: number, revenueAccountId: string | null, arAccountId: string | null, applyLateCharges: boolean, lateChargeRate: string | null, gracePeriodDays: number, taxExempt: boolean, taxExemptNumber: string, enforceCustomerBillingReq: boolean, validateCustomerRates: boolean, autoTransfer: boolean, autoMarkReadyToBill: boolean, autoApprove: boolean, autoBill: boolean, countLateOnlyOnAppointmentStops: boolean, autoApplyAccessorials: boolean, billingCurrency: string, requirePONumber: boolean, requireBOLNumber: boolean, requireDeliveryNumber: boolean, invoiceAdjustmentSupportingDocumentPolicy: CustomerInvoiceAdjustmentSupportingDocumentPolicy, defaultBillerId: string | null, billingNotes: string, fuelSurchargeMode: CustomerFuelSurchargeMode, fuelSurchargeProgramId: string | null, version: number, createdAt: number, updatedAt: number, documentTypes: Array<{ id: string, code: string, name: string, color: string, documentClassification: DocumentClassification, documentCategory: DocumentCategory }> | null } & { ' $fragmentName'?: 'CustomerBillingProfileFieldsFragment' };

export type CustomerEmailProfileFieldsFragment = { id: string, businessUnitId: string, organizationId: string, customerId: string, subject: string, comment: string, fromEmail: string, toRecipients: string, ccRecipients: string, bccRecipients: string, attachmentName: string, readReceipt: boolean, includeShipmentDetail: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CustomerEmailProfileFieldsFragment' };

export type CustomerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, stateId: string, status: EntityStatus, code: string, name: string, addressLine1: string | null, addressLine2: string | null, city: string | null, postalCode: string, isGeocoded: boolean, longitude: number | null, latitude: number | null, placeId: string | null, externalId: string | null, allowConsolidation: boolean, exclusiveConsolidation: boolean, consolidationPriority: number, version: number, createdAt: number, updatedAt: number, billingProfile: { ' $fragmentRefs'?: { 'CustomerBillingProfileFieldsFragment': CustomerBillingProfileFieldsFragment } } | null, emailProfile: { ' $fragmentRefs'?: { 'CustomerEmailProfileFieldsFragment': CustomerEmailProfileFieldsFragment } } | null } & { ' $fragmentName'?: 'CustomerTableRowFieldsFragment' };

export type CustomerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type CustomerTableQuery = { customers: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'CustomerTableRowFieldsFragment': CustomerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DetentionFacilityStatsQueryVariables = Exact<{
  input: DetentionStatsInput;
}>;


export type DetentionFacilityStatsQuery = { detentionFacilityStats: Array<{ locationId: string, locationName: string, stopCount: number, breachCount: number, avgDwellMinutes: number, medianDwellMinutes: number, p90DwellMinutes: number, billedAmount: string, driverPayAmount: string, netMargin: string, waivedAmount: string, disputeCount: number, suppressedCount: number }> };

export type DetentionCustomerStatsQueryVariables = Exact<{
  input: DetentionStatsInput;
}>;


export type DetentionCustomerStatsQuery = { detentionCustomerStats: Array<{ customerId: string, customerName: string, stopCount: number, breachCount: number, billedAmount: string, driverPayAmount: string, netMargin: string, waivedAmount: string, disputeCount: number, suppressedCount: number }> };

export type DetentionWaiverStatsQueryVariables = Exact<{
  input: DetentionStatsInput;
}>;


export type DetentionWaiverStatsQuery = { detentionWaiverStats: Array<{ reason: string, waiverCount: number, approverCount: number, waivedAmount: string }> };

export type DetentionPolicyPreviewQueryVariables = Exact<{
  policy: DetentionPolicyInput;
  scenario: DetentionPreviewScenarioInput;
}>;


export type DetentionPolicyPreviewQuery = { detentionPolicyPreview: { policySnapshot: unknown, rawDwellMinutes: number, freeMinutesGranted: number, billableMinutes: number, roundedMinutes: number, billableAmount: string, grossAmount: string, driverPayAmount: string, netMargin: string, arrivedLate: boolean, capApplied: DetentionCapKind, status: DetentionOccurrenceStatus, notificationStatus: DetentionNotificationStatus, suppressedByGate: boolean, calculationTrace: unknown, receipt: string } };

export type DetentionBacktestMutationVariables = Exact<{
  input: DetentionBacktestInput;
}>;


export type DetentionBacktestMutation = { detentionBacktest: { stopsEvaluated: number, stopsMatched: number, stopsBillable: number, stopsForfeited: number, stopsSuppressed: number, proposedRevenue: string, baselineRevenue: string, revenueDelta: string, proposedDriverPay: string, proposedNetMargin: string, negativeMarginStops: number, from: number, to: number, truncated: boolean, byCustomer: Array<{ key: string, label: string, stopCount: number, billableCount: number, proposedAmount: string, baselineAmount: string, delta: string, driverPayAmount: string, netMargin: string }>, byFacility: Array<{ key: string, label: string, stopCount: number, billableCount: number, proposedAmount: string, baselineAmount: string, delta: string, driverPayAmount: string, netMargin: string }> } };

export type DetentionOccurrenceFieldsFragment = { id: string, businessUnitId: string, organizationId: string, shipmentId: string, shipmentMoveId: string, stopId: string, customerId: string, locationId: string, detentionPolicyId: string | null, policySnapshot: unknown, calculationTrace: unknown, stopType: string, scheduleType: string, appointmentStart: number | null, appointmentEnd: number | null, arrivedAt: number | null, departedAt: number | null, clockStartAt: number, clockStopAt: number | null, freeTimeExpiresAt: number, noticeDueAt: number | null, noticeDeadlineAt: number | null, isOpen: boolean, arrivedLate: boolean, lateByMinutes: number, freeMinutesGranted: number, rawDwellMinutes: number, billableMinutes: number, roundedMinutes: number, billableUnits: string, grossAmount: string, billableAmount: string, driverPayMinutes: number, driverPayAmount: string, netMargin: string, capApplied: DetentionCapKind, convertedToLayover: boolean, currency: string, status: DetentionOccurrenceStatus, notificationStatus: DetentionNotificationStatus, noticeSentAt: number | null, suppressedByGate: boolean, requiresApproval: boolean, waiverReason: DetentionWaiverReason | null, waiverNote: string, waivedAt: number | null, waivedAmount: string, disputeNote: string, disputedAt: number | null, collectabilityScore: number, evidenceHead: string, additionalChargeId: string | null, locationName: string, customerName: string, shipmentProNumber: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'DetentionOccurrenceFieldsFragment' };

export type DetentionEvidenceFieldsFragment = { id: string, detentionOccurrenceId: string, sequence: number, kind: DetentionEvidenceKind, source: DetentionEvidenceSource, summary: string, observedAt: number, recordedAt: number, recordedById: string | null, documentId: string | null, payload: unknown, prevHash: string, hash: string, createdAt: number } & { ' $fragmentName'?: 'DetentionEvidenceFieldsFragment' };

export type DetentionNoticeFieldsFragment = { id: string, detentionOccurrenceId: string, threadKey: string, kind: DetentionNoticeKind, channel: DetentionNoticeChannel, deliveryStatus: DetentionNoticeDeliveryStatus, recipients: Array<string> | null, subject: string, body: string, scheduledFor: number, sentAt: number | null, deliveredAt: number | null, openedAt: number | null, failedAt: number | null, failureReason: string, sentById: string | null, wasAutomatic: boolean, satisfiesRequirement: boolean, quotedFreeMinutes: number, quotedRate: string | null, quotedAmount: string | null, createdAt: number } & { ' $fragmentName'?: 'DetentionNoticeFieldsFragment' };

export type DetentionCollectabilityFieldsFragment = { score: number, band: DetentionScoreBand, chainValid: boolean, summary: string, factors: Array<{ key: string, label: string, earned: number, possible: number, detail: string, remedy: string }> } & { ' $fragmentName'?: 'DetentionCollectabilityFieldsFragment' };

export type DetentionDeskQueryVariables = Exact<{ [key: string]: never; }>;


export type DetentionDeskQuery = { detentionDesk: Array<{ minutesUntilFreeEnds: number, minutesUntilNoticeDue: number | null, noticeWindowOpen: boolean, amountAtRisk: string, urgency: DetentionDeskUrgency, occurrence: { ' $fragmentRefs'?: { 'DetentionOccurrenceFieldsFragment': DetentionOccurrenceFieldsFragment } } }> };

export type DetentionOccurrenceDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type DetentionOccurrenceDetailQuery = { detentionOccurrence: { receipt: string, occurrence: { ' $fragmentRefs'?: { 'DetentionOccurrenceFieldsFragment': DetentionOccurrenceFieldsFragment } }, evidence: Array<{ ' $fragmentRefs'?: { 'DetentionEvidenceFieldsFragment': DetentionEvidenceFieldsFragment } }> | null, notices: Array<{ ' $fragmentRefs'?: { 'DetentionNoticeFieldsFragment': DetentionNoticeFieldsFragment } }> | null, collectability: { ' $fragmentRefs'?: { 'DetentionCollectabilityFieldsFragment': DetentionCollectabilityFieldsFragment } } } };

export type ShipmentDetentionQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type ShipmentDetentionQuery = { shipmentDetention: Array<{ ' $fragmentRefs'?: { 'DetentionOccurrenceFieldsFragment': DetentionOccurrenceFieldsFragment } }> };

export type DetentionDisputePacketQueryVariables = Exact<{
  occurrenceId: string | number;
}>;


export type DetentionDisputePacketQuery = { detentionDisputePacket: { policySnapshot: unknown, receipt: string, chainVerified: boolean, generatedAt: number, occurrence: { ' $fragmentRefs'?: { 'DetentionOccurrenceFieldsFragment': DetentionOccurrenceFieldsFragment } }, evidence: Array<{ ' $fragmentRefs'?: { 'DetentionEvidenceFieldsFragment': DetentionEvidenceFieldsFragment } }> | null, notices: Array<{ ' $fragmentRefs'?: { 'DetentionNoticeFieldsFragment': DetentionNoticeFieldsFragment } }> | null, collectability: { ' $fragmentRefs'?: { 'DetentionCollectabilityFieldsFragment': DetentionCollectabilityFieldsFragment } } } };

export type WaiveDetentionOccurrenceMutationVariables = Exact<{
  input: DetentionWaiveInput;
}>;


export type WaiveDetentionOccurrenceMutation = { waiveDetentionOccurrence: { ' $fragmentRefs'?: { 'DetentionOccurrenceFieldsFragment': DetentionOccurrenceFieldsFragment } } };

export type ApproveDetentionOccurrenceMutationVariables = Exact<{
  occurrenceId: string | number;
}>;


export type ApproveDetentionOccurrenceMutation = { approveDetentionOccurrence: { ' $fragmentRefs'?: { 'DetentionOccurrenceFieldsFragment': DetentionOccurrenceFieldsFragment } } };

export type DisputeDetentionOccurrenceMutationVariables = Exact<{
  input: DetentionDisputeInput;
}>;


export type DisputeDetentionOccurrenceMutation = { disputeDetentionOccurrence: { ' $fragmentRefs'?: { 'DetentionOccurrenceFieldsFragment': DetentionOccurrenceFieldsFragment } } };

export type SendDetentionNoticeMutationVariables = Exact<{
  occurrenceId: string | number;
}>;


export type SendDetentionNoticeMutation = { sendDetentionNotice: { ' $fragmentRefs'?: { 'DetentionOccurrenceFieldsFragment': DetentionOccurrenceFieldsFragment } } };

export type DetentionPolicyTierFieldsFragment = { id: string | null, fromMinute: number, toMinute: number | null, rate: string, rateUnit: DetentionTierRateUnit, label: string, sortOrder: number } & { ' $fragmentName'?: 'DetentionPolicyTierFieldsFragment' };

export type DetentionPolicyRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, code: string, description: string, status: DetentionPolicyStatus, isOrgDefault: boolean, priority: number, specificityScore: number, customerId: string | null, locationId: string | null, shipmentTypeIds: Array<string>, serviceTypeIds: Array<string>, commodityIds: Array<string>, stopTypes: Array<StopType>, appointmentStopsOnly: boolean, effectiveStartDate: number | null, effectiveEndDate: number | null, clockStartBasis: DetentionClockStartBasis, lateArrivalRule: DetentionLateArrivalRule, lateArrivalGraceMinutes: number, billingFreeMinutes: number, pickupFreeMinutes: number | null, deliveryFreeMinutes: number | null, payFreeMinutes: number | null, minimumBillableMinutes: number, billingIncrementMinutes: number, roundingMode: DetentionRoundingMode, rateSource: DetentionRateSource, accessorialChargeId: string, maxBillableMinutesPerStop: number | null, maxChargePerStop: string | null, maxChargePerDay: string | null, maxChargePerShipment: string | null, dayBoundaryMode: DetentionCapScope, convertToLayoverAtMinutes: number | null, layoverAccessorialChargeId: string | null, notificationRequirement: DetentionNotificationRequirement, notificationLeadMinutes: number, notificationDeadlineMinutes: number, unnotifiedBehavior: DetentionUnnotifiedBehavior, autoSendNotice: boolean, attachNoticePdf: boolean, sendDepartureSummary: boolean, requireApprovalOverAmount: string | null, autoApproveUnderAmount: string | null, currency: string, comments: string, version: number, createdAt: number, updatedAt: number, tiers: Array<{ ' $fragmentRefs'?: { 'DetentionPolicyTierFieldsFragment': DetentionPolicyTierFieldsFragment } }> } & { ' $fragmentName'?: 'DetentionPolicyRowFieldsFragment' };

export type DetentionPolicyTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type DetentionPolicyTableQuery = { detentionPolicies: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DetentionPolicyRowFieldsFragment': DetentionPolicyRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DetentionPolicyQueryVariables = Exact<{
  id: string | number;
}>;


export type DetentionPolicyQuery = { detentionPolicy: { ' $fragmentRefs'?: { 'DetentionPolicyRowFieldsFragment': DetentionPolicyRowFieldsFragment } } | null };

export type CreateDetentionPolicyMutationVariables = Exact<{
  input: DetentionPolicyInput;
}>;


export type CreateDetentionPolicyMutation = { createDetentionPolicy: { ' $fragmentRefs'?: { 'DetentionPolicyRowFieldsFragment': DetentionPolicyRowFieldsFragment } } };

export type UpdateDetentionPolicyMutationVariables = Exact<{
  id: string | number;
  input: DetentionPolicyInput;
}>;


export type UpdateDetentionPolicyMutation = { updateDetentionPolicy: { ' $fragmentRefs'?: { 'DetentionPolicyRowFieldsFragment': DetentionPolicyRowFieldsFragment } } };

export type DeleteDetentionPolicyMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteDetentionPolicyMutation = { deleteDetentionPolicy: boolean };

export type DispatchBoardQueryVariables = Exact<{
  input: DispatchBoardInput;
}>;


export type DispatchBoardQuery = { dispatchBoard: { windowStart: number, windowEnd: number, generatedAt: number, summary: { uncoveredMoves: number, coveredMoves: number, lateMoves: number, atRiskMoves: number, unseatedDrivers: number, availableDrivers: number, assignedToday: number, averageDeadheadMiles: number, utilizationPercent: number }, moves: Array<{ moveId: string, shipmentId: string, proNumber: string, bol: string, moveStatus: string, shipmentStatus: string, sequence: number, moveCount: number, loaded: boolean, distance: number | null, revenue: number | null, customerId: string | null, customerName: string, serviceTypeId: string | null, serviceTypeCode: string, requiredTractorTypeId: string | null, requiredTrailerTypeId: string | null, temperatureMin: number | null, temperatureMax: number | null, hasHazmat: boolean, hasActiveHold: boolean, urgency: string, minutesToPickup: number, isCovered: boolean, originStopId: string | null, originLocationId: string | null, originName: string, originCity: string, originState: string, originLatitude: number | null, originLongitude: number | null, originWindowStart: number, originWindowEnd: number | null, originActualArrival: number | null, destinationStopId: string | null, destinationLocationId: string | null, destinationName: string, destinationCity: string, destinationState: string, destinationLatitude: number | null, destinationLongitude: number | null, destinationWindowStart: number, destinationWindowEnd: number | null, assignmentId: string | null, assignedWorkerId: string | null, assignedWorkerName: string, assignedTractorId: string | null, assignedTractorCode: string, assignedTrailerId: string | null, assignedTrailerCode: string, assignmentAckStatus: string, previousMoveTrailerId: string | null, coverageType: string, carrierAssignmentId: string | null, assignedCarrierId: string | null, assignedCarrierName: string, carrierTotalCost: number | null, liveTender: { id: string, status: TenderStatus, mode: TenderMode, currentRank: number, offerCount: number, currentCarrierName: string, currentOfferExpiresAt: number | null } | null }>, drivers: Array<{ workerId: string, firstName: string, lastName: string, workerType: string, driverType: string, fleetCodeId: string | null, fleetCodeName: string, city: string, stateAbbreviation: string, postalCode: string, profilePicUrl: string, assignmentBlocked: string, availableForDispatch: boolean, tractorId: string | null, tractorCode: string, tractorTypeId: string | null, tractorAvailable: boolean, openAssignments: number, availability: string, dutyStatus: string, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, breakRemainingMs: number, hosRecordedAt: number, hosIsStale: boolean, latitude: number | null, longitude: number | null, formattedLocation: string, positionRecordedAt: number, projectedTimeAvailable: number, committedMiles: number, committedRevenue: number, commitments: Array<{ moveId: string, shipmentId: string, proNumber: string, moveStatus: string, windowStart: number, windowEnd: number, destinationCity: string, destinationState: string, destinationLatitude: number | null, destinationLongitude: number | null, trailerId: string | null }>, timeOff: Array<{ startDate: number, endDate: number, type: string }>, findings: Array<{ code: string, severity: string, field: string, message: string, regulation: string | null }> }> } };

export type DispatchMoveCandidatesQueryVariables = Exact<{
  input: DispatchMoveCandidatesInput;
}>;


export type DispatchMoveCandidatesQuery = { dispatchMoveCandidates: Array<{ workerId: string, workerName: string, tractorId: string | null, trailerId: string | null, moveId: string, score: number, verdict: string, blocked: boolean, deadheadMiles: number | null, estimatedDriveMs: number, projectedArrival: number, minutesOfSlack: number, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, projectedTimeAvailable: number, hosStrategy: string, hosRestStartDeadline: number, hosProjectedDriveMs: number, hosProjectedShiftMs: number, hosProjectedCycleMs: number, findings: Array<{ code: string, severity: string, field: string, message: string, regulation: string | null }>, factors: Array<{ key: string, label: string, raw: number, weight: number, contribution: number, detail: string }> }> };

export type DispatchDriverMovesQueryVariables = Exact<{
  input: DispatchDriverMovesInput;
}>;


export type DispatchDriverMovesQuery = { dispatchDriverMoves: Array<{ move: { moveId: string, shipmentId: string, proNumber: string, bol: string, moveStatus: string, shipmentStatus: string, sequence: number, moveCount: number, loaded: boolean, distance: number | null, revenue: number | null, customerId: string | null, customerName: string, serviceTypeId: string | null, serviceTypeCode: string, requiredTractorTypeId: string | null, requiredTrailerTypeId: string | null, temperatureMin: number | null, temperatureMax: number | null, hasHazmat: boolean, hasActiveHold: boolean, urgency: string, minutesToPickup: number, isCovered: boolean, originStopId: string | null, originLocationId: string | null, originName: string, originCity: string, originState: string, originLatitude: number | null, originLongitude: number | null, originWindowStart: number, originWindowEnd: number | null, originActualArrival: number | null, destinationStopId: string | null, destinationLocationId: string | null, destinationName: string, destinationCity: string, destinationState: string, destinationLatitude: number | null, destinationLongitude: number | null, destinationWindowStart: number, destinationWindowEnd: number | null, assignmentId: string | null, assignedWorkerId: string | null, assignedWorkerName: string, assignedTractorId: string | null, assignedTractorCode: string, assignedTrailerId: string | null, assignedTrailerCode: string, assignmentAckStatus: string, previousMoveTrailerId: string | null, coverageType: string, carrierAssignmentId: string | null, assignedCarrierId: string | null, assignedCarrierName: string, carrierTotalCost: number | null }, score: { workerId: string, workerName: string, tractorId: string | null, trailerId: string | null, moveId: string, score: number, verdict: string, blocked: boolean, deadheadMiles: number | null, estimatedDriveMs: number, projectedArrival: number, minutesOfSlack: number, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, projectedTimeAvailable: number, hosStrategy: string, hosRestStartDeadline: number, hosProjectedDriveMs: number, hosProjectedShiftMs: number, hosProjectedCycleMs: number, findings: Array<{ code: string, severity: string, field: string, message: string, regulation: string | null }>, factors: Array<{ key: string, label: string, raw: number, weight: number, contribution: number, detail: string }> } }> };

export type DispatchAssignmentPreviewQueryVariables = Exact<{
  input: DispatchAssignmentPreviewInput;
}>;


export type DispatchAssignmentPreviewQuery = { dispatchAssignmentPreview: { moveId: string, workerId: string, tractorId: string | null, trailerId: string | null, blocked: boolean, requiresOverride: boolean, score: { workerId: string, workerName: string, tractorId: string | null, trailerId: string | null, moveId: string, score: number, verdict: string, blocked: boolean, deadheadMiles: number | null, estimatedDriveMs: number, projectedArrival: number, minutesOfSlack: number, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, projectedTimeAvailable: number, hosStrategy: string, hosRestStartDeadline: number, hosProjectedDriveMs: number, hosProjectedShiftMs: number, hosProjectedCycleMs: number, findings: Array<{ code: string, severity: string, field: string, message: string, regulation: string | null }>, factors: Array<{ key: string, label: string, raw: number, weight: number, contribution: number, detail: string }> } } };

export type DispatchAssignMovesMutationVariables = Exact<{
  input: Array<DispatchAssignMoveInput> | DispatchAssignMoveInput;
}>;


export type DispatchAssignMovesMutation = { dispatchAssignMoves: { succeeded: number, failed: number, results: Array<{ moveId: string, success: boolean, assignmentId: string | null, error: string | null, findings: Array<{ code: string, severity: string, field: string, message: string, regulation: string | null }> }> } };

export type DispatchUnassignMovesMutationVariables = Exact<{
  moveIds: Array<string | number> | string | number;
}>;


export type DispatchUnassignMovesMutation = { dispatchUnassignMoves: { succeeded: number, failed: number, results: Array<{ moveId: string, success: boolean, assignmentId: string | null, error: string | null, findings: Array<{ code: string, severity: string, field: string, message: string, regulation: string | null }> }> } };

export type DispatchCarrierAssignmentPreviewQueryVariables = Exact<{
  input: DispatchCarrierAssignmentPreviewInput;
}>;


export type DispatchCarrierAssignmentPreviewQuery = { dispatchCarrierAssignmentPreview: { blockers: Array<string>, warnings: Array<string> } };

export type DispatchAssignMoveToCarrierMutationVariables = Exact<{
  input: DispatchAssignMoveToCarrierInput;
}>;


export type DispatchAssignMoveToCarrierMutation = { dispatchAssignMoveToCarrier: { id: string, shipmentMoveId: string, carrierId: string, status: CarrierAssignmentStatus, rateMethod: CarrierRateMethod, baseRate: string, baseAmount: string, fuelSurcharge: string, accessorialTotal: string, totalCost: string, currencyCode: string, proNumber: string | null, externalDriverName: string | null, externalDriverPhone: string | null, externalTractorNumber: string | null, externalTrailerNumber: string | null, confirmedAt: number | null, canceledAt: number | null, cancellationReason: string | null, carrier: { id: string, name: string, scac: string | null } | null, accessorials: Array<{ id: string, accessorialChargeId: string | null, description: string, amount: string }> | null } };

export type DispatchCancelCarrierAssignmentMutationVariables = Exact<{
  moveId: string | number;
  reason: string;
}>;


export type DispatchCancelCarrierAssignmentMutation = { dispatchCancelCarrierAssignment: boolean };

export type DispatchPlanAutoAssignMutationVariables = Exact<{
  input: DispatchPlanInput;
}>;


export type DispatchPlanAutoAssignMutation = { dispatchPlanAutoAssign: { runId: string | null, shadowMode: boolean, autonomyTier: string, totalScore: number, generatedAt: number, assignments: Array<{ moveId: string, proNumber: string, workerId: string, workerName: string, tractorId: string | null, trailerId: string | null, confidence: number, rationale: string, autoExecutable: boolean, proposalId: string | null, score: { workerId: string, workerName: string, tractorId: string | null, trailerId: string | null, moveId: string, score: number, verdict: string, blocked: boolean, deadheadMiles: number | null, estimatedDriveMs: number, projectedArrival: number, minutesOfSlack: number, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, projectedTimeAvailable: number, hosStrategy: string, hosRestStartDeadline: number, hosProjectedDriveMs: number, hosProjectedShiftMs: number, hosProjectedCycleMs: number, findings: Array<{ code: string, severity: string, field: string, message: string, regulation: string | null }>, factors: Array<{ key: string, label: string, raw: number, weight: number, contribution: number, detail: string }> } }>, uncovered: Array<{ moveId: string, proNumber: string, reason: string, bestBlockedFindings: Array<{ code: string, severity: string, field: string, message: string, regulation: string | null }> }> } };

export type DistanceOverrideLocationFieldsFragment = { id: string, name: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, state: { id: string, abbreviation: string } | null } & { ' $fragmentName'?: 'DistanceOverrideLocationFieldsFragment' };

export type DistanceOverrideTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, originLocationId: string, destinationLocationId: string, customerId: string | null, distance: number, version: number, createdAt: number, updatedAt: number, originLocation: { ' $fragmentRefs'?: { 'DistanceOverrideLocationFieldsFragment': DistanceOverrideLocationFieldsFragment } } | null, destinationLocation: { ' $fragmentRefs'?: { 'DistanceOverrideLocationFieldsFragment': DistanceOverrideLocationFieldsFragment } } | null, customer: { id: string, name: string } | null, intermediateStops: Array<{ locationId: string, stopOrder: number }> | null } & { ' $fragmentName'?: 'DistanceOverrideTableRowFieldsFragment' };

export type DistanceOverrideTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type DistanceOverrideTableQuery = { distanceOverrides: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DistanceOverrideTableRowFieldsFragment': DistanceOverrideTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DistanceProfileTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, status: string, isDefault: boolean, provider: string, dataVersion: string, region: string, routingType: string, distanceUnits: string, locationGranularity: string, profileName: string, highwayOnly: boolean, tollRoads: boolean, bordersOpen: boolean, includeTollData: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'DistanceProfileTableRowFieldsFragment' };

export type DistanceProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type DistanceProfileTableQuery = { distanceProfiles: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DistanceProfileTableRowFieldsFragment': DistanceProfileTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DocumentPacketRuleTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, resourceType: string, documentTypeId: string, required: boolean, allowMultiple: boolean, displayOrder: number, expirationRequired: boolean, expirationWarningDays: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'DocumentPacketRuleTableRowFieldsFragment' };

export type DocumentPacketRuleTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type DocumentPacketRuleTableQuery = { documentPacketRules: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DocumentPacketRuleTableRowFieldsFragment': DocumentPacketRuleTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DocumentTypeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string, color: string, documentClassification: DocumentClassification, documentCategory: DocumentCategory, isSystem: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'DocumentTypeTableRowFieldsFragment' };

export type DocumentTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type DocumentTypeTableQuery = { documentTypes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'DocumentTypeTableRowFieldsFragment': DocumentTypeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type SettlementDisputeTableQuery = { settlementDisputes: { totalCount?: number | null, edges: Array<{ node: { id: string, settlementId: string, settlementLineId: string | null, workerId: string, status: SettlementDisputeStatus, category: SettlementDisputeCategory, description: string, resolutionNote: string, resolvedAt: number | null, createdAt: number, updatedAt: number, version: number, worker: { id: string, firstName: string, lastName: string } | null, settlement: { id: string, settlementNumber: string, netPayMinor: number, currencyCode: string, status: DriverSettlementStatus } | null, resolvedBy: { id: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type DriverExpenseTableQuery = { driverExpenses: { totalCount?: number | null, edges: Array<{ node: { id: string, workerId: string, shipmentId: string | null, status: DriverExpenseStatus, amountMinor: number, currencyCode: string, description: string, incurredDate: number, receiptDocumentId: string | null, reviewNote: string, reviewedAt: number | null, settlementLineId: string | null, createdAt: number, version: number, worker: { id: string, firstName: string, lastName: string } | null, payCode: { id: string, code: string, description: string } | null, reviewedBy: { id: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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


export type DashControlQuery = { dashControl: { id: string, requireLoadAcknowledgment: boolean, allowLoadRefusals: boolean, allowStopActions: boolean, allowLoadDocumentUpload: boolean, allowLoadComments: boolean, showLoadPay: boolean, showPayEstimates: boolean, allowExpenseSubmission: boolean, requireExpenseReceipt: boolean, allowSettlementDisputes: boolean, allowProfileDocumentUpload: boolean, allowContactInfoEdit: boolean, allowPtoRequests: boolean, sendCredentialReminders: boolean, requireContactChangeApproval: boolean, driverDigestCadence: DriverDigestCadence, driverDigestWeekday: number, enableDetentionAlerts: boolean, detentionAlertThresholdMinutes: number, version: number } };

export type UpdateDashControlMutationVariables = Exact<{
  input: UpdateDashControlInput;
}>;


export type UpdateDashControlMutation = { updateDashControl: { id: string, requireLoadAcknowledgment: boolean, allowLoadRefusals: boolean, allowStopActions: boolean, allowLoadDocumentUpload: boolean, allowLoadComments: boolean, showLoadPay: boolean, showPayEstimates: boolean, allowExpenseSubmission: boolean, requireExpenseReceipt: boolean, allowSettlementDisputes: boolean, allowProfileDocumentUpload: boolean, allowContactInfoEdit: boolean, allowPtoRequests: boolean, sendCredentialReminders: boolean, requireContactChangeApproval: boolean, driverDigestCadence: DriverDigestCadence, driverDigestWeekday: number, enableDetentionAlerts: boolean, detentionAlertThresholdMinutes: number, version: number } };

export type MyPortalFeaturesQueryVariables = Exact<{ [key: string]: never; }>;


export type MyPortalFeaturesQuery = { myPortalFeatures: { requireLoadAcknowledgment: boolean, allowLoadRefusals: boolean, allowStopActions: boolean, allowLoadDocumentUpload: boolean, allowLoadComments: boolean, showLoadPay: boolean, showPayEstimates: boolean, allowExpenseSubmission: boolean, requireExpenseReceipt: boolean, allowSettlementDisputes: boolean, allowProfileDocumentUpload: boolean, allowContactInfoEdit: boolean, allowPtoRequests: boolean, ptoBalances: boolean, leaveBalance: boolean, schedule: boolean, shiftSwaps: boolean, requireContactChangeApproval: boolean, policiesOutstanding: number } };

export type MyPoliciesQueryVariables = Exact<{ [key: string]: never; }>;


export type MyPoliciesQuery = { myPolicies: Array<{ id: string, code: string, title: string, summary: string | null, body: string | null, hasDocument: boolean, versionLabel: string, requiresSignature: boolean, effectiveFrom: number, acknowledgedAt: number | null, signatureName: string | null }> };

export type MyPolicyDocumentUrlQueryVariables = Exact<{
  policyId: string | number;
}>;


export type MyPolicyDocumentUrlQuery = { myPolicyDocumentUrl: string };

export type MyProfileChangeRequestsQueryVariables = Exact<{ [key: string]: never; }>;


export type MyProfileChangeRequestsQuery = { myProfileChangeRequests: Array<{ id: string, status: ProfileChangeStatus, note: string | null, submittedAt: number, decidedAt: number | null, decisionNote: string | null, changes: Array<{ field: string, label: string, from: string, to: string }> }> };

export type AcknowledgeMyPolicyMutationVariables = Exact<{
  input: AcknowledgeMyPolicyInput;
}>;


export type AcknowledgeMyPolicyMutation = { acknowledgeMyPolicy: { id: string, policyId: string, versionLabel: string, acknowledgedAt: number, signatureName: string | null } };

export type WithdrawMyProfileChangeMutationVariables = Exact<{
  id: string | number;
}>;


export type WithdrawMyProfileChangeMutation = { withdrawMyProfileChange: { id: string, status: ProfileChangeStatus } };

export type MyScheduleQueryVariables = Exact<{
  at?: number | null | undefined;
  weeks?: number | null | undefined;
}>;


export type MyScheduleQuery = { mySchedule: { weekStart: number, weekEnd: number, shiftName: string | null, shiftCode: string | null, shiftColor: string | null, scheduledDays: number, days: Array<{ date: number, state: RotaDayState, scheduled: boolean, startMinute: number, durationMinutes: number, preference: AvailabilityPreference | null, assignmentCount: number }> } };

export type MyAvailabilityQueryVariables = Exact<{ [key: string]: never; }>;


export type MyAvailabilityQuery = { myAvailability: Array<{ id: string, dayOfWeek: number, preference: AvailabilityPreference, note: string | null }> };

export type MyShiftSwapsQueryVariables = Exact<{ [key: string]: never; }>;


export type MyShiftSwapsQuery = { myShiftSwaps: Array<{ id: string, status: ShiftSwapStatus, outgoing: boolean, counterpartyName: string | null, shiftDate: number, counterpartyShiftDate: number | null, reason: string | null, responseNote: string | null, respondedAt: number | null, decidedAt: number | null }> };

export type SetMyAvailabilityMutationVariables = Exact<{
  input: SetMyAvailabilityInput;
}>;


export type SetMyAvailabilityMutation = { setMyAvailability: { id: string, dayOfWeek: number, preference: AvailabilityPreference, note: string | null } };

export type ProposeMyShiftSwapMutationVariables = Exact<{
  input: ProposeMyShiftSwapInput;
}>;


export type ProposeMyShiftSwapMutation = { proposeMyShiftSwap: { id: string, status: ShiftSwapStatus, shiftDate: number } };

export type RespondToMyShiftSwapMutationVariables = Exact<{
  input: RespondToMyShiftSwapInput;
}>;


export type RespondToMyShiftSwapMutation = { respondToMyShiftSwap: { id: string, status: ShiftSwapStatus, respondedAt: number | null } };

export type MyHosStateQueryVariables = Exact<{ [key: string]: never; }>;


export type MyHosStateQuery = { myHosState: { workerId: string, workerName: string, provider: string, dutyStatus: string | null, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, cycleTomorrowMs: number, breakRemainingMs: number, cycleStartedAt: number | null, shiftDrivingViolationMs: number, cycleViolationMs: number, currentVehicleId: string | null, rulesetCycle: string | null, rulesetShift: string | null, driveLimitMs: number, shiftLimitMs: number, cycleLimitMs: number, breakLimitMs: number, recordedAt: number } | null };

export type MyHosDailyLogsQueryVariables = Exact<{
  startDate: string;
  endDate: string;
}>;


export type MyHosDailyLogsQuery = { myHosDailyLogs: Array<{ startAt: number, endAt: number, driveDistanceMeters: number, driveDurationMs: number, onDutyDurationMs: number, offDutyDurationMs: number, sleeperBerthDurationMs: number, isCertified: boolean, certifiedAt: number | null, shippingDocs: string | null, vehicleNames: Array<string> | null }> };

export type MyHosViolationsQueryVariables = Exact<{
  since?: number | null | undefined;
}>;


export type MyHosViolationsQuery = { myHosViolations: Array<{ workerId: string, violationType: string, description: string | null, durationMs: number, violationStartAt: number, detectedAt: number }> };

export type MyCredentialsQueryVariables = Exact<{ [key: string]: never; }>;


export type MyCredentialsQuery = { myCredentials: Array<{ id: string | null, credentialTypeId: string, name: string, category: WorkerCredentialCategory, health: WorkerCredentialHealth, daysUntilExpiry: number | null, expiresAt: number | null, numberMasked: string, required: boolean, verified: boolean, requiresDocument: boolean, documentId: string | null }> };

export type MyPtoBalancesQueryVariables = Exact<{ [key: string]: never; }>;


export type MyPtoBalancesQuery = { myPtoBalances: Array<{ ptoType: PtoType, enforced: boolean, balanceDays: string, pendingDays: string, availableDays: string, accruedYtdDays: string, usedYtdDays: string, nextAccrual: { entryType: PtoLedgerEntryType, periodKey: string, effectiveAt: number, nominalDays: string, deferred: boolean } | null }> };

export type PortalTrainingFieldsFragment = { id: string | null, courseId: string, name: string, description: string | null, category: TrainingCategory, delivery: TrainingDelivery, contentUrl: string | null, durationMinutes: number, status: WorkerTrainingStatus | null, health: WorkerTrainingHealth, required: boolean, requiresAcknowledgement: boolean, scored: boolean, dueAt: number | null, daysUntilDue: number | null, startedAt: number | null, completedAt: number | null, expiresAt: number | null, daysUntilExpiry: number | null, acknowledgedAt: number | null, score: string | null } & { ' $fragmentName'?: 'PortalTrainingFieldsFragment' };

export type MyTrainingQueryVariables = Exact<{ [key: string]: never; }>;


export type MyTrainingQuery = { myTraining: Array<{ ' $fragmentRefs'?: { 'PortalTrainingFieldsFragment': PortalTrainingFieldsFragment } }> };

export type StartMyTrainingMutationVariables = Exact<{
  id: string | number;
}>;


export type StartMyTrainingMutation = { startMyTraining: { ' $fragmentRefs'?: { 'PortalTrainingFieldsFragment': PortalTrainingFieldsFragment } } };

export type AcknowledgeMyTrainingMutationVariables = Exact<{
  id: string | number;
}>;


export type AcknowledgeMyTrainingMutation = { acknowledgeMyTraining: { ' $fragmentRefs'?: { 'PortalTrainingFieldsFragment': PortalTrainingFieldsFragment } } };

export type MySafetyScorecardQueryVariables = Exact<{ [key: string]: never; }>;


export type MySafetyScorecardQuery = { mySafetyScorecard: { workerId: string, asOf: number, score: number, rating: SafetyRating, activePoints: number, pointsWatchThreshold: number, pointsAtRiskThreshold: number, accidents: number, preventableAccidents: number, incidents: number, nearMisses: number, citations: number, inspections: number, inspectionsPassed: number, inspectionsFailed: number, outOfServiceOrders: number, cleanInspectionRate: number | null, openEvents: number, activeDiscipline: number, highestDiscipline: DisciplinaryLevel | null, daysSinceLastEvent: number | null, lastEventAt: number | null, recognitions: number } };

export type MyRecognitionsQueryVariables = Exact<{ [key: string]: never; }>;


export type MyRecognitionsQuery = { myRecognitions: Array<{ id: string, kind: RecognitionKind, title: string, message: string | null, occurredAt: number, awardedBy: { id: string, name: string } | null }> };

export type MyDisciplinaryActionsQueryVariables = Exact<{ [key: string]: never; }>;


export type MyDisciplinaryActionsQuery = { myDisciplinaryActions: Array<{ id: string, level: DisciplinaryLevel, status: DisciplinaryStatus, reason: string, details: string | null, issuedAt: number, expiresAt: number | null, suspensionDays: number | null, acknowledgedAt: number | null, workerComment: string | null, active: boolean, issuedBy: { id: string, name: string } | null }> };

export type AcknowledgeMyDisciplinaryActionMutationVariables = Exact<{
  id: string | number;
  comment?: string | null | undefined;
}>;


export type AcknowledgeMyDisciplinaryActionMutation = { acknowledgeMyDisciplinaryAction: { id: string, level: DisciplinaryLevel, status: DisciplinaryStatus, acknowledgedAt: number | null, workerComment: string | null, active: boolean } };

export type MyReviewsQueryVariables = Exact<{ [key: string]: never; }>;


export type MyReviewsQuery = { myReviews: Array<{ id: string, title: string, status: PerformanceReviewStatus, periodStart: number, periodEnd: number, overallScore: string | null, summary: string | null, strengths: string | null, improvements: string | null, submittedAt: number | null, acknowledgedAt: number | null, workerComment: string | null, closedAt: number | null, ratings: Array<{ key: string, label: string, weight: number, score: number | null, comment: string | null }>, goals: Array<{ id: string, title: string, dueAt: number | null, status: ReviewGoalStatus }>, reviewer: { id: string, name: string } | null }> };

export type AcknowledgeMyReviewMutationVariables = Exact<{
  id: string | number;
  comment?: string | null | undefined;
}>;


export type AcknowledgeMyReviewMutation = { acknowledgeMyReview: { id: string, status: PerformanceReviewStatus, acknowledgedAt: number | null, workerComment: string | null } };

export type MyLeaveQueryVariables = Exact<{ [key: string]: never; }>;


export type MyLeaveQuery = { myLeave: { entitlement: { method: LeaveMeasurementMethod, totalHours: string, usedHours: string, remainingHours: string, totalWeeks: string, usedWeeks: string, remainingWeeks: string, exhausted: boolean, eligibleOnTenure: boolean, monthsEmployed: number, openCaseCount: number, militaryCaregiver: boolean, window: { from: number, through: number } }, cases: Array<{ id: string, leaveType: WorkerLeaveType, status: LeaveCaseStatus, frequency: LeaveFrequency, fmlaDesignated: boolean, reason: string | null, startsAt: number, endsAt: number | null, decidedAt: number | null, certificationStatus: LeaveCertificationStatus, certificationDueAt: number | null, certificationLate: boolean, hoursUsed: string, hoursCharged: string }> } };

export type PayProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type PayProfileTableQuery = { payProfiles: { totalCount?: number | null, edges: Array<{ node: { id: string, organizationId: string, businessUnitId: string, status: EntityStatus, name: string, description: string, classification: PayeeClassification, currencyCode: string, guaranteedPeriodMinimumMinor: number, perDiemRatePerMile: string, perDiemDailyCapMinor: number, version: number, createdAt: number, updatedAt: number, activeAssignmentCount: number, components: Array<{ id: string, kind: PayComponentKind, method: PayCalcMethod, description: string, rate: string, revenueBasis: PayRevenueBasis | null, freeTimeMinutes: number, minAmountMinor: number | null, maxAmountMinor: number | null, sequence: number, isActive: boolean, bands: Array<{ minMiles: number, maxMiles: number, rate: string }> | null }> | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type RecurringDeductionTableQuery = { recurringDeductions: { totalCount?: number | null, edges: Array<{ node: { id: string, workerId: string, payCodeId: string, escrowAccountId: string | null, status: RecurringDeductionStatus, frequency: RecurringDeductionFrequency, description: string, amountMinor: number, totalCapMinor: number | null, deductedToDateMinor: number, startDate: number, endDate: number | null, currencyCode: string, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null, payCode: { id: string, code: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RecurringEarningTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RecurringEarningTableQuery = { recurringEarnings: { totalCount?: number | null, edges: Array<{ node: { id: string, workerId: string, payCodeId: string, status: RecurringEarningStatus, frequency: RecurringEarningFrequency, description: string, amountMinor: number, totalCapMinor: number | null, paidToDateMinor: number, startDate: number, endDate: number | null, currencyCode: string, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null, payCode: { id: string, code: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type PayCodeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type PayCodeTableQuery = { payCodes: { totalCount?: number | null, edges: Array<{ node: { id: string, status: EntityStatus, direction: PayCodeDirection, code: string, name: string, description: string, taxable: boolean, countsTowardGuarantee: boolean, glAccountId: string | null, defaultAmountMinor: number | null, isSystem: boolean, version: number, createdAt: number, updatedAt: number, glAccount: { id: string, accountCode: string, name: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type PayAdvanceTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type PayAdvanceTableQuery = { payAdvances: { totalCount?: number | null, edges: Array<{ node: { id: string, workerId: string, status: PayAdvanceStatus, source: PayAdvanceSource, reference: string, issuedDate: number, amountMinor: number, recoveredMinor: number, writtenOffMinor: number, outstandingMinor: number, writeOffReason: string, notes: string, currencyCode: string, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EscrowAccountTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EscrowAccountTableQuery = { escrowAccounts: { totalCount?: number | null, edges: Array<{ node: { id: string, workerId: string, status: EscrowAccountStatus, targetAmountMinor: number, balanceMinor: number, annualInterestRate: string, lastInterestAccrualDate: number | null, openedDate: number, closedDate: number | null, currencyCode: string, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EscrowAccountDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type EscrowAccountDetailQuery = { escrowAccount: { id: string, workerId: string, status: EscrowAccountStatus, targetAmountMinor: number, balanceMinor: number, annualInterestRate: string, lastInterestAccrualDate: number | null, openedDate: number, closedDate: number | null, currencyCode: string, version: number, worker: { id: string, firstName: string, lastName: string } | null, transactions: Array<{ id: string, type: EscrowTransactionType, amountMinor: number, balanceAfterMinor: number, occurredDate: number, description: string, settlementId: string | null, createdAt: number }> | null } | null };

export type DriverSettlementTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type DriverSettlementTableQuery = { driverSettlements: { totalCount?: number | null, edges: Array<{ node: { id: string, workerId: string, batchId: string | null, settlementNumber: string, status: DriverSettlementStatus, classification: PayeeClassification, payProfileName: string, periodStart: number, periodEnd: number, payDate: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, carryForwardInMinor: number, carryForwardOutMinor: number, netPayMinor: number, totalMiles: string, shipmentCount: number, currencyCode: string, hasExceptions: boolean, version: number, createdAt: number, updatedAt: number, worker: { id: string, firstName: string, lastName: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DriverSettlementDetailQueryVariables = Exact<{
  id: string | number;
}>;


export type DriverSettlementDetailQuery = { driverSettlement: { id: string, workerId: string, batchId: string | null, payProfileId: string | null, settlementNumber: string, status: DriverSettlementStatus, classification: PayeeClassification, payProfileName: string, periodStart: number, periodEnd: number, payDate: number, grossEarningsMinor: number, reimbursementsMinor: number, deductionsMinor: number, carryForwardInMinor: number, carryForwardOutMinor: number, netPayMinor: number, totalMiles: string, shipmentCount: number, currencyCode: string, hasExceptions: boolean, notes: string, submittedById: string | null, submittedAt: number | null, approvedById: string | null, approvedAt: number | null, postedById: string | null, postedAt: number | null, paidAt: number | null, paymentMethod: string, paymentReference: string, voidedById: string | null, voidedAt: number | null, voidReason: string, version: number, createdAt: number, updatedAt: number, exceptions: Array<{ code: string, severity: string, message: string }> | null, worker: { id: string, firstName: string, lastName: string } | null, lines: Array<{ id: string, lineNumber: number, category: SettlementLineCategory, componentKind: PayComponentKind | null, method: PayCalcMethod | null, description: string, quantity: string, rate: string, amountMinor: number, shipmentId: string | null, moveId: string | null, payEventId: string | null, recurringDeductionId: string | null, advanceId: string | null, escrowAccountId: string | null, proNumber: string }> | null } | null };

export type SettlementBatchTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type SettlementBatchTableQuery = { settlementBatches: { totalCount?: number | null, edges: Array<{ node: { id: string, status: SettlementBatchStatus, name: string, periodStart: number, periodEnd: number, payDate: number, settlementCount: number, exceptionCount: number, totalGrossMinor: number, totalNetMinor: number, currencyCode: string, notes: string, generatedById: string | null, generatedAt: number | null, completedAt: number | null, canceledAt: number | null, version: number, createdAt: number, updatedAt: number } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type DriverPayEventTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type DriverPayEventTableQuery = { driverPayEvents: { totalCount?: number | null, edges: Array<{ node: { id: string, workerId: string, shipmentId: string, moveId: string | null, settlementId: string | null, status: DriverPayEventStatus, eventDate: number, grossAmountMinor: number, totalMiles: string, currencyCode: string, proNumber: string, onHold: boolean, holdReason: string, voidedAt: number | null, voidReason: string, version: number, createdAt: number, updatedAt: number, components: Array<{ kind: PayComponentKind, method: PayCalcMethod, description: string, quantity: string, rate: string, amountMinor: number }> | null, worker: { id: string, firstName: string, lastName: string } | null } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type EdiTemplateListQuery = { ediTemplates: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiTemplateListFieldsFragment': EdiTemplateListFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type EdiPartnerTableQuery = { ediPartners: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiPartnerRowFieldsFragment': EdiPartnerRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiCommunicationProfileRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, ediPartnerId: string | null, ediConnectionId: string | null, method: EdiConnectionMethod, status: EntityStatus, name: string, description: string | null, config: unknown, version: number, createdAt: number, updatedAt: number, secretState: Array<{ key: string }> | null, partner: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'EdiCommunicationProfileRowFieldsFragment' };

export type EdiCommunicationProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EdiCommunicationProfileTableQuery = { ediCommunicationProfiles: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiCommunicationProfileRowFieldsFragment': EdiCommunicationProfileRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiTransferRowFieldsFragment = { id: string, sourceOrganizationId: string, sourceBusinessUnitId: string, targetOrganizationId: string, targetBusinessUnitId: string, sourcePartnerId: string, targetPartnerId: string, sourceShipmentId: string | null, targetShipmentId: string | null, inboundMessageId: string | null, status: EdiTransferStatus, tenderPayload: unknown, mappingSnapshot: unknown, rejectionReason: string | null, failureReason: string | null, submittedAt: number, processedAt: number | null, version: number, createdAt: number, updatedAt: number, sourcePartner: { id: string, code: string, name: string } | null, targetPartner: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'EdiTransferRowFieldsFragment' };

export type EdiTransferTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  direction: EdiTransferDirection;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EdiTransferTableQuery = { ediTransfers: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiTransferRowFieldsFragment': EdiTransferRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiMessageRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, ediPartnerId: string, documentTypeId: string, partnerDocumentProfileId: string | null, shipmentId: string | null, transferId: string | null, inboundFileId: string | null, direction: EdiDocumentDirection, transactionSet: string, x12Version: string, status: EdiMessageStatus, interchangeControlNumber: string, groupControlNumber: string, transactionControlNumber: string, segmentCount: number, deliveryStatus: EdiMessageDeliveryStatus | null, deliveryRemotePath: string | null, deliveryAttempts: number, deliveryLastAttemptAt: number | null, deliverySentAt: number | null, deliveryLastError: string | null, ackStatus: EdiMessageAckStatus | null, ackMessageId: string | null, ackReceivedAt: number | null, ackLastError: string | null, generatedAt: number, version: number, partner: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'EdiMessageRowFieldsFragment' };

export type EdiMessageTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EdiMessageTableQuery = { ediMessages: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiMessageRowFieldsFragment': EdiMessageRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiInboundFileRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, communicationProfileId: string, ediPartnerId: string | null, method: EdiConnectionMethod, remotePath: string, fileName: string, checksum: string, sizeBytes: number, interchangeControlNumber: string | null, isaSenderQualifier: string | null, isaSenderId: string | null, isaReceiverQualifier: string | null, isaReceiverId: string | null, status: EdiInboundFileStatus, failureReason: string | null, transactionCount: number, receivedAt: number, processedAt: number | null, version: number, partner: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'EdiInboundFileRowFieldsFragment' };

export type EdiInboundFileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EdiInboundFileTableQuery = { ediInboundFiles: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiInboundFileRowFieldsFragment': EdiInboundFileRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiMappingProfileRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, ediPartnerId: string, name: string, description: string | null, version: number, createdAt: number, updatedAt: number, partner: { id: string, code: string, name: string } | null, entries: Array<{ id: string, entityType: EdiMappingEntityType, sourceId: string, sourceLabel: string | null, targetId: string, targetLabel: string | null }> | null } & { ' $fragmentName'?: 'EdiMappingProfileRowFieldsFragment' };

export type EdiMappingProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EdiMappingProfileTableQuery = { ediMappingProfiles: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiMappingProfileRowFieldsFragment': EdiMappingProfileRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EdiTestCaseRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, partnerDocumentProfileId: string, name: string, description: string | null, expectedWarnings: number, expectedErrors: number, version: number, createdAt: number, updatedAt: number, documentProfile: { id: string, name: string, direction: EdiDocumentDirection, transactionSet: string, partner: { id: string, code: string, name: string } | null } | null } & { ' $fragmentName'?: 'EdiTestCaseRowFieldsFragment' };

export type EdiTestCaseTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  partnerDocumentProfileId?: string | number | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EdiTestCaseTableQuery = { ediTestCases: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EdiTestCaseRowFieldsFragment': EdiTestCaseRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type EmailProfileTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, senderName: string, senderEmail: string, replyToEmail: string, provider: EmailProvider, status: EmailProfileStatus, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'EmailProfileTableRowFieldsFragment' };

export type EmailProfileTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EmailProfileTableQuery = { emailProfiles: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EmailProfileTableRowFieldsFragment': EmailProfileTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EquipmentManufacturerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, name: string, description: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'EquipmentManufacturerTableRowFieldsFragment' };

export type EquipmentManufacturerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EquipmentManufacturerTableQuery = { equipmentManufacturers: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EquipmentManufacturerTableRowFieldsFragment': EquipmentManufacturerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EquipmentTypeTableFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'EquipmentTypeTableFieldsFragment' };

export type EquipmentTypeConfigurationRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, description: string, class: EquipmentClass, color: string, interiorLength: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'EquipmentTypeConfigurationRowFieldsFragment' };

export type EquipmentManufacturerTableFieldsFragment = { id: string, name: string } & { ' $fragmentName'?: 'EquipmentManufacturerTableFieldsFragment' };

export type FleetCodeTableFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'FleetCodeTableFieldsFragment' };

export type UsStateTableFieldsFragment = { id: string, name: string, abbreviation: string } & { ' $fragmentName'?: 'UsStateTableFieldsFragment' };

export type WorkerTableReferenceFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string } & { ' $fragmentName'?: 'WorkerTableReferenceFieldsFragment' };

export type DataTablePageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'DataTablePageInfoFieldsFragment' };

export type TractorTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, primaryWorkerId: string, equipmentTypeId: string, equipmentManufacturerId: string, stateId: string | null, fleetCodeId: string | null, secondaryWorkerId: string | null, status: EquipmentStatus, code: string, model: string, make: string, year: number | null, licensePlateNumber: string, registrationNumber: string, registrationExpiry: number | null, vin: string, externalId: string, lastKnownLocationId: string | null, lastKnownLocationName: string, fuelType: IftaFuelType, iftaQualified: boolean, version: number, createdAt: number, updatedAt: number, customFields: unknown, equipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeTableFieldsFragment': EquipmentTypeTableFieldsFragment } } | null, equipmentManufacturer: { ' $fragmentRefs'?: { 'EquipmentManufacturerTableFieldsFragment': EquipmentManufacturerTableFieldsFragment } } | null, fleetCode: { ' $fragmentRefs'?: { 'FleetCodeTableFieldsFragment': FleetCodeTableFieldsFragment } } | null, state: { ' $fragmentRefs'?: { 'UsStateTableFieldsFragment': UsStateTableFieldsFragment } } | null, primaryWorker: { ' $fragmentRefs'?: { 'WorkerTableReferenceFieldsFragment': WorkerTableReferenceFieldsFragment } } | null, secondaryWorker: { ' $fragmentRefs'?: { 'WorkerTableReferenceFieldsFragment': WorkerTableReferenceFieldsFragment } } | null } & { ' $fragmentName'?: 'TractorTableRowFieldsFragment' };

export type TrailerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, equipmentTypeId: string, equipmentManufacturerId: string, registrationStateId: string | null, fleetCodeId: string | null, status: EquipmentStatus, code: string, model: string, make: string, year: number | null, licensePlateNumber: string, vin: string, externalId: string, registrationNumber: string, maxLoadWeight: number | null, lastInspectionDate: number | null, registrationExpiry: number | null, lastKnownLocationId: string | null, lastKnownLocationName: string, version: number, createdAt: number, updatedAt: number, customFields: unknown, equipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeTableFieldsFragment': EquipmentTypeTableFieldsFragment } } | null, equipmentManufacturer: { ' $fragmentRefs'?: { 'EquipmentManufacturerTableFieldsFragment': EquipmentManufacturerTableFieldsFragment } } | null, fleetCode: { ' $fragmentRefs'?: { 'FleetCodeTableFieldsFragment': FleetCodeTableFieldsFragment } } | null, registrationState: { ' $fragmentRefs'?: { 'UsStateTableFieldsFragment': UsStateTableFieldsFragment } } | null } & { ' $fragmentName'?: 'TrailerTableRowFieldsFragment' };

export type TractorTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeEquipmentDetails?: boolean | null | undefined;
  includeFleetDetails?: boolean | null | undefined;
  includeWorkerDetails?: boolean | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type TractorTableQuery = { tractors: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TractorTableRowFieldsFragment': TractorTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type TrailerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeEquipmentDetails?: boolean | null | undefined;
  includeFleetDetails?: boolean | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type TrailerTableQuery = { trailers: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TrailerTableRowFieldsFragment': TrailerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type EquipmentTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  classes?: Array<EquipmentClass> | EquipmentClass | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type EquipmentTypeTableQuery = { equipmentTypes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'EquipmentTypeConfigurationRowFieldsFragment': EquipmentTypeConfigurationRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type FiscalYearTableQuery = { fiscalYears: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FiscalYearTableRowFieldsFragment': FiscalYearTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FleetCodeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, managerId: string, status: EntityStatus, code: string, description: string, revenueGoal: number | null, deadheadGoal: number | null, mileageGoal: number | null, color: string, version: number, createdAt: number, updatedAt: number, manager: { id: string, name: string } | null } & { ' $fragmentName'?: 'FleetCodeTableRowFieldsFragment' };

export type FleetCodeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type FleetCodeTableQuery = { fleetCodes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FleetCodeTableRowFieldsFragment': FleetCodeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FleetSafetyQueryVariables = Exact<{
  input?: FleetSafetyInput | null | undefined;
}>;


export type FleetSafetyQuery = { fleetSafety: { asOf: number, windowMonths: number, workers: number, averageScore: number, atRisk: number, watch: number, totalEvents: number, totalPoints: number, openEvents: number, outOfServiceOrders: number, basicsInferred: boolean, ratings: Array<{ rating: SafetyRating, workers: number }>, basics: Array<{ basic: CsaBasic, violations: number, events: number, weightedScore: number, outOfService: number, inferred: boolean }>, kinds: Array<{ kind: SafetyEventKind, events: number, points: number, preventable: number, outOfService: number, open: number }>, terminals: Array<{ fleetCodeId: string | null, code: string, description: string, color: string, workers: number, atRisk: number, watch: number, averageScore: number }>, trend: Array<{ periodStart: number, events: number, accidents: number, preventable: number, citations: number, inspections: number, outOfService: number, points: number }>, worst: Array<{ workerId: string, name: string, fleetCodeId: string | null, fleetCode: string, fleetColor: string, rating: SafetyRating, score: number, activePoints: number, events: number, lastEventAt: number | null }>, best: Array<{ workerId: string, name: string, fleetCodeId: string | null, fleetCode: string, fleetColor: string, rating: SafetyRating, score: number, activePoints: number, events: number, lastEventAt: number | null }> } };

export type WorkerSafetyViolationsQueryVariables = Exact<{
  safetyEventId?: string | number | null | undefined;
  workerId?: string | number | null | undefined;
}>;


export type WorkerSafetyViolationsQuery = { workerSafetyViolations: Array<{ id: string, safetyEventId: string, workerId: string, basic: CsaBasic, code: string | null, description: string, severityWeight: number, outOfService: boolean, version: number }> };

export type RecordSafetyViolationMutationVariables = Exact<{
  input: RecordSafetyViolationInput;
}>;


export type RecordSafetyViolationMutation = { recordSafetyViolation: { id: string, basic: CsaBasic, version: number } };

export type UpdateSafetyViolationMutationVariables = Exact<{
  input: UpdateSafetyViolationInput;
}>;


export type UpdateSafetyViolationMutation = { updateSafetyViolation: { id: string, basic: CsaBasic, version: number } };

export type DeleteSafetyViolationMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteSafetyViolationMutation = { deleteSafetyViolation: boolean };

export type FormulaTemplateTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, type: FormulaTemplateType, expression: string, status: FormulaTemplateStatus, schemaId: string, minCharge: string | null, maxCharge: string | null, roundingMode: RateRoundingMode, roundingPrecision: number, sourceTemplateId: string | null, sourceVersionNumber: number | null, version: number, currentVersionNumber: number, approvedAt: number | null, usageCount: number, scenarioCount: number, createdAt: number, updatedAt: number, variableDefinitions: Array<{ name: string, type: string, description: string, required: boolean, defaultValue: unknown, source: string | null }>, breakdownDefinitions: Array<{ name: string, label: string, expression: string }> } & { ' $fragmentName'?: 'FormulaTemplateTableRowFieldsFragment' };

export type FormulaTemplateTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type FormulaTemplateTableQuery = { formulaTemplates: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FormulaTemplateTableRowFieldsFragment': FormulaTemplateTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FuelCardFieldsFragment = { id: string, businessUnitId: string, organizationId: string, provider: FuelCardProvider, lastFour: string, label: string, externalCardId: string | null, assignedWorkerId: string | null, assignedTractorId: string | null, status: FuelCardStatus, expiresAt: number | null, cancelledAt: number | null, cancelReason: string | null, notes: string | null, discoveredAt: number | null, version: number, createdAt: number, updatedAt: number, assignedWorker: { id: string, wholeName: string, firstName: string, lastName: string } | null, assignedTractor: { id: string, code: string } | null } & { ' $fragmentName'?: 'FuelCardFieldsFragment' };

export type FuelCardTableQueryVariables = Exact<{
  input: FuelCardsInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type FuelCardTableQuery = { fuelCards: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FuelCardFieldsFragment': FuelCardFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type FuelCardQueryVariables = Exact<{
  id: string | number;
}>;


export type FuelCardQuery = { fuelCard: { ' $fragmentRefs'?: { 'FuelCardFieldsFragment': FuelCardFieldsFragment } } };

export type CreateFuelCardMutationVariables = Exact<{
  input: FuelCardInput;
}>;


export type CreateFuelCardMutation = { createFuelCard: { ' $fragmentRefs'?: { 'FuelCardFieldsFragment': FuelCardFieldsFragment } } };

export type UpdateFuelCardMutationVariables = Exact<{
  id: string | number;
  version: number;
  input: FuelCardInput;
}>;


export type UpdateFuelCardMutation = { updateFuelCard: { ' $fragmentRefs'?: { 'FuelCardFieldsFragment': FuelCardFieldsFragment } } };

export type CancelFuelCardMutationVariables = Exact<{
  input: CancelFuelCardInput;
}>;


export type CancelFuelCardMutation = { cancelFuelCard: { ' $fragmentRefs'?: { 'FuelCardFieldsFragment': FuelCardFieldsFragment } } };

export type AssignFuelCardMutationVariables = Exact<{
  input: AssignFuelCardInput;
}>;


export type AssignFuelCardMutation = { assignFuelCard: { ' $fragmentRefs'?: { 'FuelCardFieldsFragment': FuelCardFieldsFragment } } };

export type SyncFuelCardFeedMutationVariables = Exact<{
  provider: FuelCardProvider;
}>;


export type SyncFuelCardFeedMutation = { syncFuelCardFeed: { provider: FuelCardProvider, batchId: string | null, fetched: number, committed: number, queued: number, alreadyImported: number, cardsDiscovered: number } };

export type FuelPurchaseImportBatchFieldsFragment = { id: string, businessUnitId: string, organizationId: string, provider: FuelCardProvider, origin: FuelPurchaseImportOrigin, feedReference: string | null, documentId: string | null, fileName: string | null, sourceFormat: FuelImportFormat | null, status: FuelPurchaseImportStatus, defaultFuelType: IftaFuelType | null, defaultFuelCardId: string | null, defaultCurrency: string, mapping: unknown, unmappedHeaders: Array<string>, rowCount: number, errorCount: number, committedCount: number, error: string | null, uploadedById: string | null, stagedAt: number | null, committedAt: number | null, committedById: string | null, version: number, createdAt: number, updatedAt: number, summary: { rowCount: number, newCount: number, duplicateInFileCount: number, alreadyImportedCount: number, errorCount: number, totalGallons: string, totalAmount: string, byFuelType: unknown, byJurisdiction: unknown, earliestPurchasedAt: number | null, latestPurchasedAt: number | null } | null, document: { id: string, fileName: string, originalName: string, fileType: string, fileSize: number, createdAt: number } | null, defaultFuelCard: { id: string, provider: FuelCardProvider, lastFour: string, label: string } | null } & { ' $fragmentName'?: 'FuelPurchaseImportBatchFieldsFragment' };

export type FuelPurchaseImportRowFieldsFragment = { id: string, importBatchId: string, rowNumber: number, cells: Array<string>, transactionReference: string | null, status: FuelPurchaseImportRowStatus, error: string | null, resolvedTractorId: string | null, resolvedFuelCardId: string | null, resolvedJurisdictionId: string | null, resolutionNotes: Array<string>, fuelPurchaseId: string | null, createdAt: number, parsed: { purchasedAt: number | null, vendor: string | null, vendorCity: string | null, jurisdictionCode: string | null, fuelType: IftaFuelType | null, quantity: string | null, quantityUnit: FuelQuantityUnit | null, gallons: string | null, unitPrice: string | null, totalAmount: string | null, currencyCode: string | null, transactionReference: string | null, cardLastFour: string | null, tractorCode: string | null, odometer: number | null } | null, resolvedTractor: { id: string, code: string } | null } & { ' $fragmentName'?: 'FuelPurchaseImportRowFieldsFragment' };

export type FuelPurchaseImportQueryVariables = Exact<{
  id: string | number;
}>;


export type FuelPurchaseImportQuery = { fuelPurchaseImport: { ' $fragmentRefs'?: { 'FuelPurchaseImportBatchFieldsFragment': FuelPurchaseImportBatchFieldsFragment } } };

export type FuelPurchaseImportRowsQueryVariables = Exact<{
  id: string | number;
  input?: FuelPurchaseImportRowsInput | null | undefined;
}>;


export type FuelPurchaseImportRowsQuery = { fuelPurchaseImport: { id: string, rows: { totalCount: number | null, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'FuelPurchaseImportRowFieldsFragment': FuelPurchaseImportRowFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } } };

export type FuelPurchaseImportTemplateQueryVariables = Exact<{
  provider: FuelCardProvider;
}>;


export type FuelPurchaseImportTemplateQuery = { fuelPurchaseImportTemplate: { fileName: string, content: string } };

export type CreateFuelPurchaseImportMutationVariables = Exact<{
  input: CreateFuelPurchaseImportInput;
}>;


export type CreateFuelPurchaseImportMutation = { createFuelPurchaseImport: { ' $fragmentRefs'?: { 'FuelPurchaseImportBatchFieldsFragment': FuelPurchaseImportBatchFieldsFragment } } };

export type StageFuelPurchaseImportMutationVariables = Exact<{
  input: StageFuelPurchaseImportInput;
}>;


export type StageFuelPurchaseImportMutation = { stageFuelPurchaseImport: { ' $fragmentRefs'?: { 'FuelPurchaseImportBatchFieldsFragment': FuelPurchaseImportBatchFieldsFragment } } };

export type CommitFuelPurchaseImportMutationVariables = Exact<{
  id: string | number;
  version: number;
}>;


export type CommitFuelPurchaseImportMutation = { commitFuelPurchaseImport: { ' $fragmentRefs'?: { 'FuelPurchaseImportBatchFieldsFragment': FuelPurchaseImportBatchFieldsFragment } } };

export type DiscardFuelPurchaseImportMutationVariables = Exact<{
  id: string | number;
  version: number;
  reason?: string | null | undefined;
}>;


export type DiscardFuelPurchaseImportMutation = { discardFuelPurchaseImport: { ' $fragmentRefs'?: { 'FuelPurchaseImportBatchFieldsFragment': FuelPurchaseImportBatchFieldsFragment } } };

export type ResolveFuelPurchaseImportRowsMutationVariables = Exact<{
  id: string | number;
  version: number;
}>;


export type ResolveFuelPurchaseImportRowsMutation = { resolveFuelPurchaseImportRows: { reviewed: number, resolved: number, committed: number, queued: number, batch: { ' $fragmentRefs'?: { 'FuelPurchaseImportBatchFieldsFragment': FuelPurchaseImportBatchFieldsFragment } } } };

export type FuelPurchaseImportTableQueryVariables = Exact<{
  input: FuelPurchaseImportsInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type FuelPurchaseImportTableQuery = { fuelPurchaseImports: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FuelPurchaseImportBatchFieldsFragment': FuelPurchaseImportBatchFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type FuelPurchaseFieldsFragment = { id: string, businessUnitId: string, organizationId: string, tractorId: string, workerId: string | null, jurisdictionId: string, fuelCardId: string | null, cardLastFour: string | null, purchasedAt: number, vendor: string | null, vendorCity: string | null, fuelType: IftaFuelType, quantity: string, quantityUnit: FuelQuantityUnit, gallons: string, unitPrice: string | null, totalAmount: string, currencyCode: string, odometer: number | null, transactionReference: string | null, source: FuelPurchaseSource, importBatchId: string | null, taxPaid: boolean, notes: string | null, createdById: string | null, version: number, createdAt: number, updatedAt: number, tractor: { id: string, code: string } | null, worker: { id: string, wholeName: string, firstName: string, lastName: string } | null, jurisdiction: { id: string, countryCode: string, code: string, name: string }, fuelCard: { id: string, provider: FuelCardProvider, lastFour: string, label: string } | null } & { ' $fragmentName'?: 'FuelPurchaseFieldsFragment' };

export type FuelPurchaseTableQueryVariables = Exact<{
  input: FuelPurchasesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type FuelPurchaseTableQuery = { fuelPurchases: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FuelPurchaseFieldsFragment': FuelPurchaseFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type FuelPurchaseQueryVariables = Exact<{
  id: string | number;
}>;


export type FuelPurchaseQuery = { fuelPurchase: { ' $fragmentRefs'?: { 'FuelPurchaseFieldsFragment': FuelPurchaseFieldsFragment } } };

export type CreateFuelPurchaseMutationVariables = Exact<{
  input: FuelPurchaseInput;
}>;


export type CreateFuelPurchaseMutation = { createFuelPurchase: { ' $fragmentRefs'?: { 'FuelPurchaseFieldsFragment': FuelPurchaseFieldsFragment } } };

export type UpdateFuelPurchaseMutationVariables = Exact<{
  id: string | number;
  version: number;
  input: FuelPurchaseInput;
}>;


export type UpdateFuelPurchaseMutation = { updateFuelPurchase: { ' $fragmentRefs'?: { 'FuelPurchaseFieldsFragment': FuelPurchaseFieldsFragment } } };

export type DeleteFuelPurchaseMutationVariables = Exact<{
  id: string | number;
  version: number;
}>;


export type DeleteFuelPurchaseMutation = { deleteFuelPurchase: boolean };

export type FuelIndexFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, code: string, description: string, source: FuelIndexSource, fuelType: FuelType, region: string, eiaSeriesId: string, currency: string, isActive: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'FuelIndexFieldsFragment' };

export type FuelSurchargeProgramFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, code: string, description: string, status: FuelSurchargeProgramStatus, fuelIndexId: string, accessorialChargeId: string, method: FuelSurchargeProgramMethod, pegPrice: string | null, increment: string | null, incrementRate: string | null, milesPerGallon: string | null, percentBasis: FuelSurchargePercentBasis, stepRounding: FuelSurchargeStepRounding, rateRounding: FuelSurchargeRateRounding, ratePrecision: number, minAmount: string | null, maxAmount: string | null, dateBasis: FuelSurchargeDateBasis, priceEffectiveDay: number, missingPriceFallback: FuelSurchargeMissingPriceFallback, effectiveStartDate: number | null, effectiveEndDate: number | null, shipmentTypeIds: Array<string> | null, serviceTypeIds: Array<string> | null, tractorTypeIds: Array<string> | null, trailerTypeIds: Array<string> | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'FuelSurchargeProgramFieldsFragment' };

export type FuelIndexTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type FuelIndexTableQuery = { fuelIndexes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FuelIndexFieldsFragment': FuelIndexFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type FuelSurchargeProgramTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type FuelSurchargeProgramTableQuery = { fuelSurchargePrograms: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'FuelSurchargeProgramFieldsFragment': FuelSurchargeProgramFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type HazardousMaterialTableQuery = { hazardousMaterials: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'HazardousMaterialTableRowFieldsFragment': HazardousMaterialTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type HazmatSegregationRuleTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, name: string, description: string, exceptionNotes: string, referenceCode: string, regulationSource: string, distanceUnit: string, classA: HazardousClass, classB: HazardousClass, segregationType: SegregationType, hasExceptions: boolean, hazmatAId: string | null, hazmatBId: string | null, minimumDistance: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'HazmatSegregationRuleTableRowFieldsFragment' };

export type HazmatSegregationRuleTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type HazmatSegregationRuleTableQuery = { hazmatSegregationRules: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'HazmatSegregationRuleTableRowFieldsFragment': HazmatSegregationRuleTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type HoldReasonTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, type: HoldType, code: string, label: string, description: string, active: boolean, defaultSeverity: HoldSeverity, defaultBlocksDispatch: boolean, defaultBlocksDelivery: boolean, defaultBlocksBilling: boolean, defaultVisibleToCustomer: boolean, sortOrder: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'HoldReasonTableRowFieldsFragment' };

export type HoldReasonTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type HoldReasonTableQuery = { holdReasons: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'HoldReasonTableRowFieldsFragment': HoldReasonTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type HomeWidgetFieldsFragment = { id: string, key: string, title: string | null, w: number, h: number, config: { metric: string | null, metrics: Array<string> | null, definitionId: string | null, cannedKey: string | null, chartId: string | null, columnId: string | null, dashboardId: string | null, text: string | null, limit: number | null, windowDays: number | null } } & { ' $fragmentName'?: 'HomeWidgetFieldsFragment' };

export type HomeLayoutFieldsFragment = { schemaVersion: number, version: number, source: HomeLayoutSource, presetId: string | null, presetName: string | null, locked: boolean, canCustomize: boolean, density: string, widgets: Array<{ ' $fragmentRefs'?: { 'HomeWidgetFieldsFragment': HomeWidgetFieldsFragment } }> } & { ' $fragmentName'?: 'HomeLayoutFieldsFragment' };

export type HomeLayoutPresetFieldsFragment = { id: string, name: string, description: string | null, roleIds: Array<string>, coreResponsibility: string | null, isOrgDefault: boolean, locked: boolean, priority: number, assignedUserCount: number, version: number, createdAt: number, updatedAt: number, widgets: Array<{ ' $fragmentRefs'?: { 'HomeWidgetFieldsFragment': HomeWidgetFieldsFragment } }> } & { ' $fragmentName'?: 'HomeLayoutPresetFieldsFragment' };

export type UpdateHomeLayoutMutationVariables = Exact<{
  input: HomeLayoutInput;
}>;


export type UpdateHomeLayoutMutation = { updateHomeLayout: { ' $fragmentRefs'?: { 'HomeLayoutFieldsFragment': HomeLayoutFieldsFragment } } };

export type ResetHomeLayoutMutationVariables = Exact<{ [key: string]: never; }>;


export type ResetHomeLayoutMutation = { resetHomeLayout: { ' $fragmentRefs'?: { 'HomeLayoutFieldsFragment': HomeLayoutFieldsFragment } } };

export type CreateHomeLayoutPresetMutationVariables = Exact<{
  input: SaveHomeLayoutPresetInput;
}>;


export type CreateHomeLayoutPresetMutation = { createHomeLayoutPreset: { ' $fragmentRefs'?: { 'HomeLayoutPresetFieldsFragment': HomeLayoutPresetFieldsFragment } } };

export type UpdateHomeLayoutPresetMutationVariables = Exact<{
  input: UpdateHomeLayoutPresetInput;
}>;


export type UpdateHomeLayoutPresetMutation = { updateHomeLayoutPreset: { ' $fragmentRefs'?: { 'HomeLayoutPresetFieldsFragment': HomeLayoutPresetFieldsFragment } } };

export type DeleteHomeLayoutPresetMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteHomeLayoutPresetMutation = { deleteHomeLayoutPreset: boolean };

export type HomeLayoutQueryVariables = Exact<{ [key: string]: never; }>;


export type HomeLayoutQuery = { homeLayout: { ' $fragmentRefs'?: { 'HomeLayoutFieldsFragment': HomeLayoutFieldsFragment } } };

export type HomeWidgetCatalogQueryVariables = Exact<{ [key: string]: never; }>;


export type HomeWidgetCatalogQuery = { homeWidgetCatalog: { gridColumns: number, maxWidgets: number, densities: Array<string>, categories: Array<{ key: string, label: string, description: string }>, metrics: Array<{ key: string, label: string }>, widgets: Array<{ key: string, label: string, description: string, category: string, configKind: string, analyticsInclude: string | null, defaultW: number, defaultH: number, minW: number, minH: number, maxW: number, maxH: number }> } };

export type HomeLayoutPresetsQueryVariables = Exact<{ [key: string]: never; }>;


export type HomeLayoutPresetsQuery = { homeLayoutPresets: Array<{ ' $fragmentRefs'?: { 'HomeLayoutPresetFieldsFragment': HomeLayoutPresetFieldsFragment } }> };

export type HomeLayoutPresetQueryVariables = Exact<{
  id: string | number;
}>;


export type HomeLayoutPresetQuery = { homeLayoutPreset: { ' $fragmentRefs'?: { 'HomeLayoutPresetFieldsFragment': HomeLayoutPresetFieldsFragment } } };

export type HomeLayoutPreviewQueryVariables = Exact<{
  presetId?: string | number | null | undefined;
  roleId?: string | number | null | undefined;
}>;


export type HomeLayoutPreviewQuery = { homeLayoutPreview: { ' $fragmentRefs'?: { 'HomeLayoutFieldsFragment': HomeLayoutFieldsFragment } } };

export type IftaMileageEntryFieldsFragment = { id: string, businessUnitId: string, organizationId: string, tractorId: string, jurisdictionId: string, traveledAt: number, year: number, quarter: number, miles: string, loaded: boolean, source: IftaMileageSource, shipmentMoveId: string | null, notes: string | null, createdById: string | null, version: number, createdAt: number, updatedAt: number, tractor: { id: string, code: string } | null, jurisdiction: { id: string, countryCode: string, code: string, name: string } } & { ' $fragmentName'?: 'IftaMileageEntryFieldsFragment' };

export type IftaMileageEntryTableQueryVariables = Exact<{
  input: IftaMileageEntriesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type IftaMileageEntryTableQuery = { iftaMileageEntries: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'IftaMileageEntryFieldsFragment': IftaMileageEntryFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type IftaMileageEntryQueryVariables = Exact<{
  id: string | number;
}>;


export type IftaMileageEntryQuery = { iftaMileageEntry: { ' $fragmentRefs'?: { 'IftaMileageEntryFieldsFragment': IftaMileageEntryFieldsFragment } } };

export type CreateIftaMileageEntryMutationVariables = Exact<{
  input: IftaMileageEntryInput;
}>;


export type CreateIftaMileageEntryMutation = { createIftaMileageEntry: { ' $fragmentRefs'?: { 'IftaMileageEntryFieldsFragment': IftaMileageEntryFieldsFragment } } };

export type UpdateIftaMileageEntryMutationVariables = Exact<{
  id: string | number;
  version: number;
  input: IftaMileageEntryInput;
}>;


export type UpdateIftaMileageEntryMutation = { updateIftaMileageEntry: { ' $fragmentRefs'?: { 'IftaMileageEntryFieldsFragment': IftaMileageEntryFieldsFragment } } };

export type DeleteIftaMileageEntryMutationVariables = Exact<{
  id: string | number;
  version: number;
}>;


export type DeleteIftaMileageEntryMutation = { deleteIftaMileageEntry: boolean };

export type IftaJurisdictionFieldsFragment = { id: string, countryCode: string, code: string, name: string, usStateId: string | null, isIftaMember: boolean, hasSurcharge: boolean, sortOrder: number, status: IftaJurisdictionStatus } & { ' $fragmentName'?: 'IftaJurisdictionFieldsFragment' };

export type IftaJurisdictionsQueryVariables = Exact<{
  membersOnly?: boolean | null | undefined;
}>;


export type IftaJurisdictionsQuery = { iftaJurisdictions: Array<{ ' $fragmentRefs'?: { 'IftaJurisdictionFieldsFragment': IftaJurisdictionFieldsFragment } }> };

export type IftaPeriodFieldsFragment = { year: number, quarter: number, key: string, label: string, start: number, end: number, dueDate: number } & { ' $fragmentName'?: 'IftaPeriodFieldsFragment' };

export type IftaReturnLineFieldsFragment = { id: string, returnId: string, jurisdictionId: string, fuelType: IftaFuelType, isIftaMember: boolean, totalMiles: string, taxableMiles: string, routeMiles: string, manualMiles: string, loadedMiles: string, emptyMiles: string, taxPaidGallons: string, taxPaidGallonsRaw: string, purchaseCount: number, taxableGallons: string, netTaxableGallons: string, ratePerGallon: string | null, surchargeRatePerGallon: string | null, rateMissing: boolean, taxDue: string, surchargeDue: string, lineTotal: string, sortOrder: number, jurisdiction: { id: string, countryCode: string, code: string, name: string, isIftaMember: boolean, hasSurcharge: boolean } } & { ' $fragmentName'?: 'IftaReturnLineFieldsFragment' };

export type IftaReturnFieldsFragment = { id: string, businessUnitId: string, organizationId: string, year: number, quarter: number, amendmentNumber: number, amendsReturnId: string | null, status: IftaReturnStatus, timezone: string, periodStart: number, periodEnd: number, totalMiles: string, totalTaxableMiles: string, totalGallons: string, totalTaxPaidGallons: string, netTaxableGallons: string, taxDue: string, surchargeDue: string, netDue: string, currencyCode: string, unattributedMiles: string, unattributedMoveCount: number, noTractorMiles: string, noTractorMoveCount: number, computedAt: number | null, finalizedAt: number | null, finalizedById: string | null, filedAt: number | null, filedById: string | null, filingReference: string | null, reopenedAt: number | null, reopenedById: string | null, reopenReason: string | null, version: number, createdAt: number, updatedAt: number, period: { ' $fragmentRefs'?: { 'IftaPeriodFieldsFragment': IftaPeriodFieldsFragment } }, amendsReturn: { id: string, amendmentNumber: number, status: IftaReturnStatus } | null, fleetMpgByFuelType: Array<{ fuelType: IftaFuelType, mpg: string | null, totalMiles: string, totalGallons: string }>, problems: Array<{ code: IftaProblemCode, message: string, jurisdictionCode: string | null, fuelType: IftaFuelType | null, amount: string | null }>, finalizedBy: { id: string, name: string } | null, filedBy: { id: string, name: string } | null, lines: Array<{ ' $fragmentRefs'?: { 'IftaReturnLineFieldsFragment': IftaReturnLineFieldsFragment } }> } & { ' $fragmentName'?: 'IftaReturnFieldsFragment' };

export type IftaReturnSummaryFieldsFragment = { id: string, year: number, quarter: number, amendmentNumber: number, status: IftaReturnStatus, totalMiles: string, totalTaxPaidGallons: string, netDue: string, currencyCode: string, computedAt: number | null, finalizedAt: number | null, filedAt: number | null, filingReference: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'IftaReturnSummaryFieldsFragment' };

export type IftaReturnForPeriodQueryVariables = Exact<{
  input: IftaPeriodInput;
}>;


export type IftaReturnForPeriodQuery = { iftaReturnForPeriod: { ' $fragmentRefs'?: { 'IftaReturnFieldsFragment': IftaReturnFieldsFragment } } | null };

export type IftaReturnQueryVariables = Exact<{
  id: string | number;
}>;


export type IftaReturnQuery = { iftaReturn: { ' $fragmentRefs'?: { 'IftaReturnFieldsFragment': IftaReturnFieldsFragment } } };

export type IftaReturnTableQueryVariables = Exact<{
  input: IftaReturnsInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type IftaReturnTableQuery = { iftaReturns: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'IftaReturnSummaryFieldsFragment': IftaReturnSummaryFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type IftaCurrentPeriodQueryVariables = Exact<{ [key: string]: never; }>;


export type IftaCurrentPeriodQuery = { iftaCurrentPeriod: { ' $fragmentRefs'?: { 'IftaPeriodFieldsFragment': IftaPeriodFieldsFragment } } };

export type IftaPeriodQueryVariables = Exact<{
  year: number;
  quarter: number;
}>;


export type IftaPeriodQuery = { iftaPeriod: { ' $fragmentRefs'?: { 'IftaPeriodFieldsFragment': IftaPeriodFieldsFragment } } };

export type GenerateIftaReturnMutationVariables = Exact<{
  period: IftaPeriodInput;
}>;


export type GenerateIftaReturnMutation = { generateIftaReturn: { ' $fragmentRefs'?: { 'IftaReturnFieldsFragment': IftaReturnFieldsFragment } } };

export type RecomputeIftaReturnMutationVariables = Exact<{
  id: string | number;
  version: number;
}>;


export type RecomputeIftaReturnMutation = { recomputeIftaReturn: { ' $fragmentRefs'?: { 'IftaReturnFieldsFragment': IftaReturnFieldsFragment } } };

export type FinalizeIftaReturnMutationVariables = Exact<{
  id: string | number;
  version: number;
}>;


export type FinalizeIftaReturnMutation = { finalizeIftaReturn: { ' $fragmentRefs'?: { 'IftaReturnFieldsFragment': IftaReturnFieldsFragment } } };

export type ReopenIftaReturnMutationVariables = Exact<{
  id: string | number;
  version: number;
  reason: string;
}>;


export type ReopenIftaReturnMutation = { reopenIftaReturn: { ' $fragmentRefs'?: { 'IftaReturnFieldsFragment': IftaReturnFieldsFragment } } };

export type MarkIftaReturnFiledMutationVariables = Exact<{
  input: MarkIftaReturnFiledInput;
}>;


export type MarkIftaReturnFiledMutation = { markIftaReturnFiled: { ' $fragmentRefs'?: { 'IftaReturnFieldsFragment': IftaReturnFieldsFragment } } };

export type AmendIftaReturnMutationVariables = Exact<{
  id: string | number;
  reason: string;
}>;


export type AmendIftaReturnMutation = { amendIftaReturn: { ' $fragmentRefs'?: { 'IftaReturnFieldsFragment': IftaReturnFieldsFragment } } };

export type DeleteIftaReturnMutationVariables = Exact<{
  id: string | number;
  version: number;
}>;


export type DeleteIftaReturnMutation = { deleteIftaReturn: boolean };

export type BackfillJurisdictionMilesMutationVariables = Exact<{
  input: BackfillJurisdictionMilesInput;
}>;


export type BackfillJurisdictionMilesMutation = { backfillJurisdictionMiles: { started: boolean, dryRun: boolean, unattributedMoves: number, unattributedMiles: string, workflowId: string | null } };

export type IftaTaxRateFieldsFragment = { id: string, jurisdictionId: string, year: number, quarter: number, fuelType: IftaFuelType, ratePerGallon: string, surchargeRatePerGallon: string | null, sourceNote: string | null, sourceUrl: string | null, version: number, createdAt: number, updatedAt: number, jurisdiction: { id: string, countryCode: string, code: string, name: string, hasSurcharge: boolean, isIftaMember: boolean } } & { ' $fragmentName'?: 'IftaTaxRateFieldsFragment' };

export type IftaTaxRateTableQueryVariables = Exact<{
  input: IftaTaxRatesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type IftaTaxRateTableQuery = { iftaTaxRates: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'IftaTaxRateFieldsFragment': IftaTaxRateFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type UpsertIftaTaxRatesMutationVariables = Exact<{
  input: Array<IftaTaxRateInput> | IftaTaxRateInput;
}>;


export type UpsertIftaTaxRatesMutation = { upsertIftaTaxRates: Array<{ ' $fragmentRefs'?: { 'IftaTaxRateFieldsFragment': IftaTaxRateFieldsFragment } }> };

export type DeleteIftaTaxRateMutationVariables = Exact<{
  id: string | number;
  version: number;
}>;


export type DeleteIftaTaxRateMutation = { deleteIftaTaxRate: boolean };

export type InvoiceTableRowFieldsFragment = { id: string, billingQueueItemId: string, shipmentId: string | null, customerId: string, number: string, billType: BillType, status: InvoiceStatus, paymentTerm: InvoicePaymentTerm, currencyCode: string, invoiceDate: number, dueDate: number | null, billToName: string, subtotalAmount: string, otherAmount: string, totalAmount: string, appliedAmount: string, settlementStatus: InvoiceSettlementStatus, disputeStatus: InvoiceDisputeStatus, sendStatus: InvoiceSendStatus, isAdjustmentArtifact: boolean, version: number, createdAt: number, updatedAt: number, customer: { id: string, name: string, code: string } | null } & { ' $fragmentName'?: 'InvoiceTableRowFieldsFragment' };

export type InvoiceTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type InvoiceTableQuery = { invoices: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'InvoiceTableRowFieldsFragment': InvoiceTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type JournalReversalTableQuery = { journalReversals: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'JournalReversalTableRowFieldsFragment': JournalReversalTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type JurisdictionRuleOverrideTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, stateId: string, maxWidthFeet: number | null, maxHeightFeet: number | null, maxLengthFeet: number | null, maxWeightPounds: number | null, permitLeadTimeDays: number | null, daylightOnly: boolean | null, holidayRestricted: boolean | null, reason: string, version: number, createdAt: number, updatedAt: number, state: { id: string, name: string, abbreviation: string } | null } & { ' $fragmentName'?: 'JurisdictionRuleOverrideTableRowFieldsFragment' };

export type JurisdictionRuleOverrideTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type JurisdictionRuleOverrideTableQuery = { jurisdictionRuleOverrides: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'JurisdictionRuleOverrideTableRowFieldsFragment': JurisdictionRuleOverrideTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type JurisdictionRuleTableRowFieldsFragment = { id: string, stateId: string, status: JurisdictionRuleStatus, maxWidthFeet: number, maxHeightFeet: number, maxLengthFeet: number, maxWeightPounds: number, superloadWidthFeet: number | null, superloadWeightPounds: number | null, daylightOnly: boolean, rushHourRestricted: boolean, weekendRestricted: boolean, holidayRestricted: boolean, permitLeadTimeDays: number, permitValidityDays: number, permitBaseFee: string | null, permitPerMileFee: string | null, sourceNote: string | null, sourceUrl: string | null, verificationState: JurisdictionVerificationState, verifiedAt: number | null, effectiveStartDate: number | null, effectiveEndDate: number | null, version: number, createdAt: number, updatedAt: number, state: { id: string, name: string, abbreviation: string } | null } & { ' $fragmentName'?: 'JurisdictionRuleTableRowFieldsFragment' };

export type JurisdictionRuleTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type JurisdictionRuleTableQuery = { jurisdictionRules: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'JurisdictionRuleTableRowFieldsFragment': JurisdictionRuleTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type LocationCategoryTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, type: LocationCategoryType, facilityType: FacilityType | null, color: string, hasSecureParking: boolean, requiresAppointment: boolean, allowsOvernight: boolean, hasRestroom: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'LocationCategoryTableRowFieldsFragment' };

export type LocationCategoryTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type LocationCategoryTableQuery = { locationCategories: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'LocationCategoryTableRowFieldsFragment': LocationCategoryTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type LocationTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, locationCategoryId: string, stateId: string, status: EntityStatus, code: string, name: string, description: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, isGeocoded: boolean, latitude: number | null, longitude: number | null, placeId: string, version: number, createdAt: number, updatedAt: number, state: { id: string, name: string, abbreviation: string } | null, locationCategory: { id: string, name: string, color: string } | null } & { ' $fragmentName'?: 'LocationTableRowFieldsFragment' };

export type LocationTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type LocationTableQuery = { locations: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'LocationTableRowFieldsFragment': LocationTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ManualJournalTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, requestNumber: string, status: ManualJournalStatus, description: string, reason: string | null, accountingDate: number, requestedFiscalYearId: string, requestedFiscalPeriodId: string, currencyCode: string, totalDebit: number, totalCredit: number, approvedAt: number | null, approvedById: string | null, rejectedAt: number | null, rejectedById: string | null, rejectionReason: string | null, cancelledAt: number | null, cancelledById: string | null, cancelReason: string | null, postedBatchId: string | null, createdById: string | null, updatedById: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ManualJournalTableRowFieldsFragment' };

export type ManualJournalTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ManualJournalTableQuery = { manualJournals: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ManualJournalTableRowFieldsFragment': ManualJournalTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type NotificationFieldsFragment = { id: string, organizationId: string, businessUnitId: string | null, targetUserId: string | null, eventType: string, priority: NotificationPriority, channel: NotificationChannel, title: string, message: string, data: unknown, relatedEntities: unknown, source: string, readAt: number | null, dismissedAt: number | null, createdAt: number } & { ' $fragmentName'?: 'NotificationFieldsFragment' };

export type NotificationListQueryVariables = Exact<{
  input: DataTableConnectionInput;
  filter?: NotificationFilterInput | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type NotificationListQuery = { notifications: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'NotificationFieldsFragment': NotificationFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type NotificationUnreadCountQueryVariables = Exact<{ [key: string]: never; }>;


export type NotificationUnreadCountQuery = { notificationUnreadCount: number };

export type MyNotificationListQueryVariables = Exact<{
  input: DataTableConnectionInput;
  filter?: NotificationFilterInput | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type MyNotificationListQuery = { myNotifications: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'NotificationFieldsFragment': NotificationFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type MyNotificationUnreadCountQueryVariables = Exact<{ [key: string]: never; }>;


export type MyNotificationUnreadCountQuery = { myNotificationUnreadCount: number };

export type MarkAllMyNotificationsReadMutationVariables = Exact<{ [key: string]: never; }>;


export type MarkAllMyNotificationsReadMutation = { markAllMyNotificationsRead: boolean };

export type MarkMyNotificationsReadMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type MarkMyNotificationsReadMutation = { markMyNotificationsRead: boolean };

export type MarkMyNotificationsUnreadMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type MarkMyNotificationsUnreadMutation = { markMyNotificationsUnread: boolean };

export type DismissMyNotificationsMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type DismissMyNotificationsMutation = { dismissMyNotifications: boolean };

export type RestoreMyNotificationsMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type RestoreMyNotificationsMutation = { restoreMyNotifications: boolean };

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
  offCycleReason?: string | null | undefined;
}>;


export type CreateInvoiceFromOrderMutation = { createInvoiceFromOrder: { id: string, number: string } };

export type CreateInvoiceFromShipmentsMutationVariables = Exact<{
  shipmentIds: Array<string | number> | string | number;
  offCycleReason?: string | null | undefined;
}>;


export type CreateInvoiceFromShipmentsMutation = { createInvoiceFromShipments: { id: string, number: string } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type OrderTableQuery = { orders: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'OrderTableRowFieldsFragment': OrderTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type OrgHolidayFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, holidayDate: number, kind: OrgHolidayKind, recursAnnually: boolean, description: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'OrgHolidayFieldsFragment' };

export type OrgHolidaysQueryVariables = Exact<{
  year?: number | null | undefined;
}>;


export type OrgHolidaysQuery = { orgHolidays: Array<{ ' $fragmentRefs'?: { 'OrgHolidayFieldsFragment': OrgHolidayFieldsFragment } }> };

export type CreateOrgHolidayMutationVariables = Exact<{
  input: OrgHolidayInput;
}>;


export type CreateOrgHolidayMutation = { createOrgHoliday: { ' $fragmentRefs'?: { 'OrgHolidayFieldsFragment': OrgHolidayFieldsFragment } } };

export type UpdateOrgHolidayMutationVariables = Exact<{
  id: string | number;
  input: OrgHolidayInput;
}>;


export type UpdateOrgHolidayMutation = { updateOrgHoliday: { ' $fragmentRefs'?: { 'OrgHolidayFieldsFragment': OrgHolidayFieldsFragment } } };

export type DeleteOrgHolidayMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteOrgHolidayMutation = { deleteOrgHoliday: boolean };

export type JobPositionsQueryVariables = Exact<{
  activeOnly?: boolean | null | undefined;
  drivingOnly?: boolean | null | undefined;
}>;


export type JobPositionsQuery = { jobPositions: Array<{ id: string, status: EntityStatus, code: string, title: string, description: string | null, department: JobDepartment, flsaExempt: boolean, isDrivingPosition: boolean, reportsToPositionId: string | null, version: number, reportsTo: { id: string, code: string, title: string } | null }> };

export type HeadcountQueryVariables = Exact<{ [key: string]: never; }>;


export type HeadcountQuery = { headcount: { activeTotal: number, driverTotal: number, terminated: number, staffTotal: number, byFleet: Array<{ key: string, label: string, code: string, color: string, workers: number, drivers: number, terminated: number, staff: number }>, byPosition: Array<{ key: string, label: string, code: string, color: string, workers: number, drivers: number, terminated: number, staff: number }>, byDepartment: Array<{ key: string, label: string, code: string, color: string, workers: number, drivers: number, terminated: number, staff: number }> } };

export type JobPositionHoldersQueryVariables = Exact<{
  id: string | number;
}>;


export type JobPositionHoldersQuery = { jobPositionHolders: Array<{ kind: PositionHolderKind, id: string, name: string, status: string, detail: string }> };

export type MyTeamQueryVariables = Exact<{
  includeInactive?: boolean | null | undefined;
}>;


export type MyTeamQuery = { myTeam: Array<{ workerId: string, name: string, status: string, fleetCodeId: string | null, fleetCode: string, fleetColor: string, positionId: string | null, positionTitle: string, direct: boolean, managerId: string | null, complianceStatus: string, trainingHealth: string, safetyRating: string, hireDate: number, terminationDate: number | null }> };

export type ApprovalDelegationsQueryVariables = Exact<{
  delegatorId?: string | number | null | undefined;
  delegateId?: string | number | null | undefined;
  activeOnly?: boolean | null | undefined;
}>;


export type ApprovalDelegationsQuery = { approvalDelegations: Array<{ id: string, delegatorId: string, delegateId: string, scope: ApprovalScope, startsAt: number, endsAt: number | null, reason: string | null, revokedAt: number | null, version: number, delegator: { id: string, name: string } | null, delegate: { id: string, name: string } | null }> };

export type AssignWorkerPositionMutationVariables = Exact<{
  workerId: string | number;
  positionId?: string | number | null | undefined;
}>;


export type AssignWorkerPositionMutation = { assignWorkerPosition: boolean };

export type AssignUserPositionMutationVariables = Exact<{
  userId: string | number;
  positionId?: string | number | null | undefined;
}>;


export type AssignUserPositionMutation = { assignUserPosition: boolean };

export type CreateJobPositionMutationVariables = Exact<{
  input: JobPositionInput;
}>;


export type CreateJobPositionMutation = { createJobPosition: { id: string, title: string, version: number } };

export type UpdateJobPositionMutationVariables = Exact<{
  input: UpdateJobPositionInput;
}>;


export type UpdateJobPositionMutation = { updateJobPosition: { id: string, title: string, version: number } };

export type DelegateApprovalMutationVariables = Exact<{
  input: DelegateApprovalInput;
}>;


export type DelegateApprovalMutation = { delegateApproval: { id: string, scope: ApprovalScope, startsAt: number, endsAt: number | null, version: number } };

export type RevokeApprovalDelegationMutationVariables = Exact<{
  id: string | number;
}>;


export type RevokeApprovalDelegationMutation = { revokeApprovalDelegation: { id: string, revokedAt: number | null, version: number } };

export type OrganizationSettingsStateFieldsFragment = { id: string, name: string, abbreviation: string } & { ' $fragmentName'?: 'OrganizationSettingsStateFieldsFragment' };

export type OrganizationSettingsFieldsFragment = { id: string, version: number, createdAt: number, updatedAt: number, bucketName: string, businessUnitId: string, loginSlug: string, name: string, scacCode: string, dotNumber: string, logoUrl: string, addressLine1: string, addressLine2: string, city: string, stateId: string, postalCode: string, timezone: string, taxId: string, brokerageEnabled: boolean, assetOperationsEnabled: boolean, state: { ' $fragmentRefs'?: { 'OrganizationSettingsStateFieldsFragment': OrganizationSettingsStateFieldsFragment } } | null } & { ' $fragmentName'?: 'OrganizationSettingsFieldsFragment' };

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

export type PerformanceReviewTemplateFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string | null, status: EntityStatus, isDefault: boolean, cadenceMonths: number | null, openReviewCount: number, version: number, createdAt: number, updatedAt: number, items: Array<{ key: string, label: string, description: string | null, weight: number }> } & { ' $fragmentName'?: 'PerformanceReviewTemplateFieldsFragment' };

export type PerformanceReviewFieldsFragment = { id: string, businessUnitId: string, organizationId: string, workerId: string, templateId: string, reviewerId: string | null, status: PerformanceReviewStatus, title: string, periodStart: number, periodEnd: number, overallScore: string | null, summary: string | null, strengths: string | null, improvements: string | null, submittedAt: number | null, acknowledgedAt: number | null, workerComment: string | null, closedAt: number | null, closedById: string | null, nextReviewAt: number | null, version: number, createdAt: number, updatedAt: number, template: { id: string, code: string, name: string, cadenceMonths: number | null, items: Array<{ key: string, label: string, description: string | null, weight: number }> } | null, reviewer: { id: string, name: string } | null, ratings: Array<{ key: string, label: string, weight: number, score: number | null, comment: string | null }>, goals: Array<{ id: string, title: string, dueAt: number | null, status: ReviewGoalStatus }> } & { ' $fragmentName'?: 'PerformanceReviewFieldsFragment' };

export type PerformanceReviewTemplateTableQueryVariables = Exact<{
  input: PerformanceReviewTemplatesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type PerformanceReviewTemplateTableQuery = { performanceReviewTemplates: { totalCount?: number | null, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'PerformanceReviewTemplateFieldsFragment': PerformanceReviewTemplateFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type ActivePerformanceReviewTemplatesQueryVariables = Exact<{ [key: string]: never; }>;


export type ActivePerformanceReviewTemplatesQuery = { activePerformanceReviewTemplates: Array<{ ' $fragmentRefs'?: { 'PerformanceReviewTemplateFieldsFragment': PerformanceReviewTemplateFieldsFragment } }> };

export type WorkerPerformanceReviewsQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerPerformanceReviewsQuery = { workerPerformanceReviews: Array<{ ' $fragmentRefs'?: { 'PerformanceReviewFieldsFragment': PerformanceReviewFieldsFragment } }> };

export type CreatePerformanceReviewTemplateMutationVariables = Exact<{
  input: PerformanceReviewTemplateInput;
}>;


export type CreatePerformanceReviewTemplateMutation = { createPerformanceReviewTemplate: { ' $fragmentRefs'?: { 'PerformanceReviewTemplateFieldsFragment': PerformanceReviewTemplateFieldsFragment } } };

export type UpdatePerformanceReviewTemplateMutationVariables = Exact<{
  id: string | number;
  input: PerformanceReviewTemplateInput;
}>;


export type UpdatePerformanceReviewTemplateMutation = { updatePerformanceReviewTemplate: { ' $fragmentRefs'?: { 'PerformanceReviewTemplateFieldsFragment': PerformanceReviewTemplateFieldsFragment } } };

export type ArchivePerformanceReviewTemplateMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type ArchivePerformanceReviewTemplateMutation = { archivePerformanceReviewTemplate: { ' $fragmentRefs'?: { 'PerformanceReviewTemplateFieldsFragment': PerformanceReviewTemplateFieldsFragment } } };

export type RestorePerformanceReviewTemplateMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type RestorePerformanceReviewTemplateMutation = { restorePerformanceReviewTemplate: { ' $fragmentRefs'?: { 'PerformanceReviewTemplateFieldsFragment': PerformanceReviewTemplateFieldsFragment } } };

export type CreatePerformanceReviewMutationVariables = Exact<{
  input: CreatePerformanceReviewInput;
}>;


export type CreatePerformanceReviewMutation = { createPerformanceReview: { ' $fragmentRefs'?: { 'PerformanceReviewFieldsFragment': PerformanceReviewFieldsFragment } } };

export type UpdatePerformanceReviewMutationVariables = Exact<{
  input: UpdatePerformanceReviewInput;
}>;


export type UpdatePerformanceReviewMutation = { updatePerformanceReview: { ' $fragmentRefs'?: { 'PerformanceReviewFieldsFragment': PerformanceReviewFieldsFragment } } };

export type SubmitPerformanceReviewMutationVariables = Exact<{
  input: PerformanceReviewStatusInput;
}>;


export type SubmitPerformanceReviewMutation = { submitPerformanceReview: { ' $fragmentRefs'?: { 'PerformanceReviewFieldsFragment': PerformanceReviewFieldsFragment } } };

export type ReopenPerformanceReviewMutationVariables = Exact<{
  input: PerformanceReviewStatusInput;
}>;


export type ReopenPerformanceReviewMutation = { reopenPerformanceReview: { ' $fragmentRefs'?: { 'PerformanceReviewFieldsFragment': PerformanceReviewFieldsFragment } } };

export type ClosePerformanceReviewMutationVariables = Exact<{
  input: PerformanceReviewStatusInput;
}>;


export type ClosePerformanceReviewMutation = { closePerformanceReview: { ' $fragmentRefs'?: { 'PerformanceReviewFieldsFragment': PerformanceReviewFieldsFragment } } };

export type DeletePerformanceReviewMutationVariables = Exact<{
  id: string | number;
}>;


export type DeletePerformanceReviewMutation = { deletePerformanceReview: boolean };

export type PtoPolicyRuleFieldsFragment = { id: string, ptoPolicyId: string, ptoType: PtoType, accrualMethod: PtoAccrualMethod, accrualAmountDays: string, maxBalanceDays: string | null, carryoverCapDays: string | null, carryoverExpiryDays: number, onTermination: PtoTerminationAction, sortOrder: number, tiers: Array<{ minMonths: number, accrualAmountDays: string, maxBalanceDays: string | null }> } & { ' $fragmentName'?: 'PtoPolicyRuleFieldsFragment' };

export type PtoPolicyFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, code: string, description: string | null, status: PtoPolicyStatus, isDefault: boolean, yearBasis: PtoYearBasis, countWeekends: boolean, waitingPeriodDays: number, requiresApproval: boolean, enforceBalance: boolean, allowNegative: boolean, negativeFloorDays: string, openAssignmentCount: number, version: number, createdAt: number, updatedAt: number, rules: Array<{ id: string, ptoPolicyId: string, ptoType: PtoType, accrualMethod: PtoAccrualMethod, accrualAmountDays: string, maxBalanceDays: string | null, carryoverCapDays: string | null, carryoverExpiryDays: number, onTermination: PtoTerminationAction, sortOrder: number, tiers: Array<{ minMonths: number, accrualAmountDays: string, maxBalanceDays: string | null }> }> } & { ' $fragmentName'?: 'PtoPolicyFieldsFragment' };

export type PtoPolicyAssignmentFieldsFragment = { id: string, workerId: string, ptoPolicyId: string, effectiveFrom: number, effectiveTo: number | null, assignedById: string | null, note: string | null, version: number, createdAt: number, updatedAt: number, ptoPolicy: { id: string, name: string, code: string, status: PtoPolicyStatus, countWeekends: boolean, requiresApproval: boolean, enforceBalance: boolean } | null } & { ' $fragmentName'?: 'PtoPolicyAssignmentFieldsFragment' };

export type PlannedPtoAccrualFieldsFragment = { entryType: PtoLedgerEntryType, periodKey: string, effectiveAt: number, nominalDays: string, deferred: boolean } & { ' $fragmentName'?: 'PlannedPtoAccrualFieldsFragment' };

export type WorkerPtoBalanceFieldsFragment = { ptoType: PtoType, tracked: boolean, enforced: boolean, balanceDays: string, pendingDays: string, availableDays: string, accruedYtdDays: string, usedYtdDays: string, carriedDays: string, maxBalanceDays: string | null, nextAccrual: { entryType: PtoLedgerEntryType, periodKey: string, effectiveAt: number, nominalDays: string, deferred: boolean } | null } & { ' $fragmentName'?: 'WorkerPtoBalanceFieldsFragment' };

export type WorkerPtoLedgerEntryFieldsFragment = { id: string, workerId: string, ptoType: PtoType, entryType: PtoLedgerEntryType, amountDays: string, balanceAfterDays: string, sequence: number, effectiveAt: number, periodKey: string | null, sourcePtoId: string | null, assignmentId: string | null, ptoPolicyId: string | null, actorType: PtoLedgerActorType, createdById: string | null, note: string | null, createdAt: number } & { ' $fragmentName'?: 'WorkerPtoLedgerEntryFieldsFragment' };

export type PtoPolicyTableQueryVariables = Exact<{
  input: PtoPoliciesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type PtoPolicyTableQuery = { ptoPolicies: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'PtoPolicyFieldsFragment': PtoPolicyFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type PtoPolicyQueryVariables = Exact<{
  id: string | number;
}>;


export type PtoPolicyQuery = { ptoPolicy: { ' $fragmentRefs'?: { 'PtoPolicyFieldsFragment': PtoPolicyFieldsFragment } } };

export type WorkerPtoPolicyAssignmentsQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerPtoPolicyAssignmentsQuery = { workerPtoPolicyAssignments: Array<{ ' $fragmentRefs'?: { 'PtoPolicyAssignmentFieldsFragment': PtoPolicyAssignmentFieldsFragment } }> };

export type WorkerPtoBalancesQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerPtoBalancesQuery = { workerPtoBalances: Array<{ ' $fragmentRefs'?: { 'WorkerPtoBalanceFieldsFragment': WorkerPtoBalanceFieldsFragment } }> };

export type WorkerPtoLedgerQueryVariables = Exact<{
  input: WorkerPtoLedgerInput;
}>;


export type WorkerPtoLedgerQuery = { workerPtoLedger: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerPtoLedgerEntryFieldsFragment': WorkerPtoLedgerEntryFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type WorkerPtoAvailabilityQueryVariables = Exact<{
  input: PtoAvailabilityInput;
}>;


export type WorkerPtoAvailabilityQuery = { workerPtoAvailability: { tracked: boolean, enforced: boolean, allowed: boolean, days: string, balanceDays: string, pendingDays: string, projectedAccrualDays: string, projectedAvailableDays: string, floorDays: string, message: string | null } };

export type PreviewWorkerPtoAccrualQueryVariables = Exact<{
  workerId: string | number;
  asOf?: number | null | undefined;
}>;


export type PreviewWorkerPtoAccrualQuery = { previewWorkerPtoAccrual: Array<{ ' $fragmentRefs'?: { 'PlannedPtoAccrualFieldsFragment': PlannedPtoAccrualFieldsFragment } }> };

export type PtoBalanceSummaryQueryVariables = Exact<{ [key: string]: never; }>;


export type PtoBalanceSummaryQuery = { ptoBalanceSummary: { workersTracked: number, workersUnassigned: number, totalBalanceDays: string, totalPendingDays: string, accruedYtdDays: string, usedYtdDays: string } };

export type PtoLiabilityReportQueryVariables = Exact<{
  asOf?: number | null | undefined;
}>;


export type PtoLiabilityReportQuery = { ptoLiabilityReport: { asOf: number, workersTracked: number, totalBalanceDays: string, liabilityDays: string, forfeitableDays: string, rows: Array<{ workerId: string, ptoType: PtoType, balanceDays: string, accruedYtdDays: string, usedYtdDays: string, onTermination: PtoTerminationAction, liabilityDays: string, worker: { id: string, wholeName: string, firstName: string, lastName: string, profilePicUrl: string, status: EntityStatus } | null }> } };

export type CreatePtoPolicyMutationVariables = Exact<{
  input: PtoPolicyInput;
}>;


export type CreatePtoPolicyMutation = { createPtoPolicy: { ' $fragmentRefs'?: { 'PtoPolicyFieldsFragment': PtoPolicyFieldsFragment } } };

export type UpdatePtoPolicyMutationVariables = Exact<{
  id: string | number;
  input: PtoPolicyInput;
}>;


export type UpdatePtoPolicyMutation = { updatePtoPolicy: { ' $fragmentRefs'?: { 'PtoPolicyFieldsFragment': PtoPolicyFieldsFragment } } };

export type ArchivePtoPolicyMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type ArchivePtoPolicyMutation = { archivePtoPolicy: { ' $fragmentRefs'?: { 'PtoPolicyFieldsFragment': PtoPolicyFieldsFragment } } };

export type RestorePtoPolicyMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type RestorePtoPolicyMutation = { restorePtoPolicy: { ' $fragmentRefs'?: { 'PtoPolicyFieldsFragment': PtoPolicyFieldsFragment } } };

export type AssignWorkerPtoPolicyMutationVariables = Exact<{
  input: AssignWorkerPtoPolicyInput;
}>;


export type AssignWorkerPtoPolicyMutation = { assignWorkerPtoPolicy: { assignments: Array<{ ' $fragmentRefs'?: { 'PtoPolicyAssignmentFieldsFragment': PtoPolicyAssignmentFieldsFragment } }>, failures: Array<{ workerId: string, error: string }> } };

export type EndWorkerPtoPolicyAssignmentMutationVariables = Exact<{
  input: EndWorkerPtoPolicyAssignmentInput;
}>;


export type EndWorkerPtoPolicyAssignmentMutation = { endWorkerPtoPolicyAssignment: { ' $fragmentRefs'?: { 'PtoPolicyAssignmentFieldsFragment': PtoPolicyAssignmentFieldsFragment } } };

export type AdjustWorkerPtoBalanceMutationVariables = Exact<{
  input: AdjustWorkerPtoBalanceInput;
}>;


export type AdjustWorkerPtoBalanceMutation = { adjustWorkerPtoBalance: { ' $fragmentRefs'?: { 'WorkerPtoLedgerEntryFieldsFragment': WorkerPtoLedgerEntryFieldsFragment } } };

export type RunPtoAccrualMutationVariables = Exact<{
  input: RunPtoAccrualInput;
}>;


export type RunPtoAccrualMutation = { runPtoAccrual: { queued: boolean, workflowId: string | null, workersProcessed: number, workersSkipped: number, entriesPosted: number, entriesCapped: number, entriesSkipped: number } };

export type RateAgreementRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, partyType: RateAgreementPartyType, customerId: string | null, carrierId: string | null, code: string, name: string, description: string, agreementType: RateAgreementType, status: RateAgreementStatus, contractRef: string, priority: number, effectiveFrom: number, effectiveTo: number | null, autoRenew: boolean, renewalNoticeDays: number, currency: string, defaultMinCharge: string | null, defaultMaxCharge: string | null, marginFloorPercent: string | null, maxPayPercentOfSell: string | null, submittedById: string | null, submittedAt: number | null, approvedById: string | null, approvedAt: number | null, reviewComment: string, currentVersionNumber: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'RateAgreementRowFieldsFragment' };

export type RateZoneRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string, status: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'RateZoneRowFieldsFragment' };

export type RateMatrixRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string, status: string, formulaTemplateId: string, formulaTemplateName: string, currency: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'RateMatrixRowFieldsFragment' };

export type RateQuoteRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, shipmentId: string | null, partyType: RateAgreementPartyType, partyId: string, purpose: RateQuotePurpose, outcome: RateQuoteOutcome, rateAgreementId: string | null, rateAgreementRuleId: string | null, formulaTemplateId: string | null, specificityScore: number, currency: string, billingCurrency: string, linehaulAmount: string, totalAmount: string, billingAmount: string, foregoneAmount: string | null, overrideReason: string, asOf: number, ratedAt: number, ratedById: string | null, engineVersion: string, createdAt: number } & { ' $fragmentName'?: 'RateQuoteRowFieldsFragment' };

export type RateAgreementTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RateAgreementTableQuery = { rateAgreements: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RateAgreementRowFieldsFragment': RateAgreementRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RateZoneTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RateZoneTableQuery = { rateZones: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RateZoneRowFieldsFragment': RateZoneRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RateMatrixTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RateMatrixTableQuery = { rateMatrices: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RateMatrixRowFieldsFragment': RateMatrixRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RateQuoteTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RateQuoteTableQuery = { rateQuotes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RateQuoteRowFieldsFragment': RateQuoteRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RecurringShipmentTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, sourceShipmentId: string, customerId: string | null, originLocationId: string | null, destinationLocationId: string | null, name: string, description: string, status: RecurringShipmentStatus, cronExpression: string, timezone: string, startDate: number | null, endDate: number | null, maxOccurrences: number | null, leadTimeDays: number, skipWeekends: boolean, exceptionPolicy: RecurringShipmentExceptionPolicy, blackoutDates: Array<string> | null, autoGenerate: boolean, nextOccurrenceAt: number | null, lastOccurrenceAt: number | null, lastRunAt: number | null, generationCount: number, consecutiveFailures: number, version: number, createdAt: number, updatedAt: number, customer: { id: string, name: string, code: string } | null, originLocation: { id: string, name: string, code: string } | null, destinationLocation: { id: string, name: string, code: string } | null } & { ' $fragmentName'?: 'RecurringShipmentTableRowFieldsFragment' };

export type RecurringShipmentTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RecurringShipmentTableQuery = { recurringShipments: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RecurringShipmentTableRowFieldsFragment': RecurringShipmentTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CannedReportsQueryVariables = Exact<{ [key: string]: never; }>;


export type CannedReportsQuery = { cannedReports: Array<{ key: string, version: string, name: string, description: string, category: string, tags: Array<string>, defaultFormat: string, definition: unknown }> };

export type ReportCatalogQueryVariables = Exact<{ [key: string]: never; }>;


export type ReportCatalogQuery = { reportCatalog: { version: string, entities: Array<{ key: string, resource: string, label: string, pluralLabel: string, description: string | null, category: string, ownScopeSupported: boolean, fields: Array<{ key: string, label: string, description: string | null, type: string, format: string | null, nullable: boolean, aggregations: Array<string>, filterable: boolean, groupable: boolean, accessible: boolean, sensitivity: string, enumValues: Array<{ value: string, label: string }> }>, edges: Array<{ name: string, label: string, target: string, cardinality: string, traversable: boolean }> }> } };

export type ReportDashboardFieldsFragment = { id: string, name: string, description: string, category: string, tags: Array<string>, ownerId: string, visibility: string, layout: unknown, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ReportDashboardFieldsFragment' };

export type ReportDashboardsQueryVariables = Exact<{ [key: string]: never; }>;


export type ReportDashboardsQuery = { reportDashboards: Array<{ ' $fragmentRefs'?: { 'ReportDashboardFieldsFragment': ReportDashboardFieldsFragment } }> };

export type ReportDashboardByIdQueryVariables = Exact<{
  id: string | number;
}>;


export type ReportDashboardByIdQuery = { reportDashboard: { ' $fragmentRefs'?: { 'ReportDashboardFieldsFragment': ReportDashboardFieldsFragment } } };

export type CreateReportDashboardMutationVariables = Exact<{
  input: SaveReportDashboardInput;
}>;


export type CreateReportDashboardMutation = { createReportDashboard: { ' $fragmentRefs'?: { 'ReportDashboardFieldsFragment': ReportDashboardFieldsFragment } } };

export type UpdateReportDashboardMutationVariables = Exact<{
  input: UpdateReportDashboardInput;
}>;


export type UpdateReportDashboardMutation = { updateReportDashboard: { ' $fragmentRefs'?: { 'ReportDashboardFieldsFragment': ReportDashboardFieldsFragment } } };

export type DeleteReportDashboardMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteReportDashboardMutation = { deleteReportDashboard: boolean };

export type ReportDefinitionFieldsFragment = { id: string, name: string, description: string, category: string, tags: Array<string>, kind: string, cannedKey: string | null, cannedVersion: string | null, ownerId: string, visibility: string, status: string, diagnostics: Array<string>, catalogVersion: string, definition: unknown, defaultFormat: string, currentRevision: number, lastRunAt: number | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ReportDefinitionFieldsFragment' };

export type ReportDefinitionsTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ReportDefinitionsTableQuery = { reportDefinitions: { totalCount?: number, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'ReportDefinitionFieldsFragment': ReportDefinitionFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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

export type ReportDefinitionOptionFieldsFragment = { id: string, name: string, description: string, category: string, kind: string, status: string, visibility: string, lastRunAt: number | null, updatedAt: number } & { ' $fragmentName'?: 'ReportDefinitionOptionFieldsFragment' };

export type ReportDefinitionOptionsQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type ReportDefinitionOptionsQuery = { reportDefinitions: { edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'ReportDefinitionOptionFieldsFragment': ReportDefinitionOptionFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ReportPreviewFieldsFragment = { rows: unknown, totals: unknown, truncated: boolean, columns: Array<{ id: string, label: string, type: string, format: string | null, display: { style: string, decimals: number, grouping: boolean, currency: string, negative: string, notation: string, prefix: string, suffix: string, dateStyle: string, boolStyle: string, durationUnit: string, durationStyle: string, nullText: string, rules: Array<{ op: string, value: number, upper: number, tone: string }>, band: { width: number, edges: Array<number> } | null } }> } & { ' $fragmentName'?: 'ReportPreviewFieldsFragment' };

export type PreviewReportQueryVariables = Exact<{
  definition: ReportIrInput;
  params?: unknown;
  supersede?: boolean | null | undefined;
}>;


export type PreviewReportQuery = { previewReport: { ' $fragmentRefs'?: { 'ReportPreviewFieldsFragment': ReportPreviewFieldsFragment } } };

export type DrillThroughReportQueryVariables = Exact<{
  input: ReportDrillInput;
}>;


export type DrillThroughReportQuery = { drillThroughReport: { ' $fragmentRefs'?: { 'ReportPreviewFieldsFragment': ReportPreviewFieldsFragment } } };

export type ReportRunFieldsFragment = { id: string, definitionId: string | null, revisionId: string | null, cannedKey: string | null, cannedVersion: string | null, requestedById: string, trigger: string, params: unknown, format: string, status: string, rowCount: number, byteSize: number, durationMs: number, truncated: boolean, artifactExpiresAt: number | null, cacheHit: boolean, queuedAt: number | null, startedAt: number | null, completedAt: number | null, version: number, createdAt: number, error: { code: string, message: string, detail: string | null } | null } & { ' $fragmentName'?: 'ReportRunFieldsFragment' };

export type ReportRunsTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  filter?: ReportRunsFilterInput | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ReportRunsTableQuery = { reportRuns: { totalCount?: number, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'ReportRunFieldsFragment': ReportRunFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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

export type ReportScheduleFieldsFragment = { id: string, definitionId: string, cronExpression: string, timezone: string, formats: Array<string>, emailRecipients: Array<string>, emailAttach: boolean, emailInline: boolean, notifyUserIds: Array<string>, alertFiring: boolean, enabled: boolean, runAsId: string, lastRunId: string | null, nextRunAt: number | null, consecutiveFailures: number, version: number, createdAt: number, updatedAt: number, alert: { operator: string, threshold: number, columnId: string | null, value: number | null, suppressWhileFiring: boolean } | null } & { ' $fragmentName'?: 'ReportScheduleFieldsFragment' };

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

export type ReportViewFieldsFragment = { id: string, definitionId: string, ownerId: string, name: string, description: string | null, params: unknown, shared: boolean, pinned: boolean, format: string | null, lastRunAt: number | null, runCount: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ReportViewFieldsFragment' };

export type ReportViewsQueryVariables = Exact<{
  definitionId?: string | number | null | undefined;
}>;


export type ReportViewsQuery = { reportViews: Array<{ ' $fragmentRefs'?: { 'ReportViewFieldsFragment': ReportViewFieldsFragment } }> };

export type CreateReportViewMutationVariables = Exact<{
  input: CreateReportViewInput;
}>;


export type CreateReportViewMutation = { createReportView: { ' $fragmentRefs'?: { 'ReportViewFieldsFragment': ReportViewFieldsFragment } } };

export type UpdateReportViewMutationVariables = Exact<{
  input: UpdateReportViewInput;
}>;


export type UpdateReportViewMutation = { updateReportView: { ' $fragmentRefs'?: { 'ReportViewFieldsFragment': ReportViewFieldsFragment } } };

export type DeleteReportViewMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteReportViewMutation = { deleteReportView: boolean };

export type RoleTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, coreResponsibility: string | null, parentRoleIds: Array<string> | null, maxSensitivity: string, isSystem: boolean, createdBy: string, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'RoleTableRowFieldsFragment' };

export type RoleTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RoleTableQuery = { roles: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RoleTableRowFieldsFragment': RoleTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RoutingGuideEntryFieldsFragment = { id: string, routingGuideId: string, carrierId: string, rank: number, rateMethod: CarrierRateMethod, rate: string, useContractRate: boolean, offerTtlSeconds: number, channel: TenderChannel, version: number, createdAt: number, updatedAt: number, carrier: { id: string, name: string, scac: string | null } | null } & { ' $fragmentName'?: 'RoutingGuideEntryFieldsFragment' };

export type RoutingGuideRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, status: EntityStatus, originLocationId: string | null, destinationLocationId: string | null, originCity: string, originState: string, destinationCity: string, destinationState: string, specificity: number, version: number, createdAt: number, updatedAt: number, entries: Array<{ ' $fragmentRefs'?: { 'RoutingGuideEntryFieldsFragment': RoutingGuideEntryFieldsFragment } }> | null } & { ' $fragmentName'?: 'RoutingGuideRowFieldsFragment' };

export type RoutingGuideTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RoutingGuideTableQuery = { routingGuides: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'RoutingGuideRowFieldsFragment': RoutingGuideRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type RoutingGuideOptionsQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type RoutingGuideOptionsQuery = { routingGuides: { totalCount?: number | null, edges: Array<{ node: { id: string, name: string, status: EntityStatus, originLocationId: string | null, destinationLocationId: string | null, originCity: string, originState: string, destinationCity: string, destinationState: string, specificity: number, entries: Array<{ id: string, carrierId: string, rank: number, rateMethod: CarrierRateMethod, rate: string, offerTtlSeconds: number, channel: TenderChannel, carrier: { id: string, name: string, scac: string | null } | null }> | null } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type MatchRoutingGuideQueryVariables = Exact<{
  input: MatchRoutingGuideInput;
}>;


export type MatchRoutingGuideQuery = { matchRoutingGuide: { id: string, name: string, description: string, status: EntityStatus, originLocationId: string | null, destinationLocationId: string | null, originCity: string, originState: string, destinationCity: string, destinationState: string, specificity: number, entries: Array<{ id: string, carrierId: string, rank: number, rateMethod: CarrierRateMethod, rate: string, offerTtlSeconds: number, channel: TenderChannel, carrier: { id: string, name: string, scac: string | null } | null }> | null } | null };

export type ShiftTemplatesQueryVariables = Exact<{
  activeOnly?: boolean | null | undefined;
  limit?: number | null | undefined;
}>;


export type ShiftTemplatesQuery = { shiftTemplates: Array<{ id: string, status: EntityStatus, code: string, name: string, description: string | null, color: string | null, daysOfWeek: string, startMinute: number, durationMinutes: number, cycleWeeks: number, version: number, activeAssignmentCount: number }> };

export type RotaQueryVariables = Exact<{
  filter?: RotaFilterInput | null | undefined;
}>;


export type RotaQuery = { rota: { weekStart: number, weekEnd: number, weeks: number, scheduledDays: number, conflicts: number, rows: Array<{ workerId: string, name: string, fleetCode: string | null, fleetColor: string | null, shiftCode: string | null, shiftName: string | null, shiftColor: string | null, scheduledDays: number, conflicts: number, days: Array<{ date: number, state: RotaDayState, scheduled: boolean, startMinute: number, durationMinutes: number, preference: AvailabilityPreference | null, assignmentCount: number, isConflict: boolean }> }> } };

export type WorkerShiftAssignmentsQueryVariables = Exact<{
  workerId: string | number;
  activeOnly?: boolean | null | undefined;
}>;


export type WorkerShiftAssignmentsQuery = { workerShiftAssignments: Array<{ id: string, workerId: string, shiftTemplateId: string, effectiveFrom: number, effectiveTo: number | null, cycleOffsetWeeks: number, notes: string | null, version: number, shiftTemplate: { id: string, code: string, name: string, color: string | null, daysOfWeek: string, startMinute: number, durationMinutes: number, cycleWeeks: number } | null }> };

export type WorkerAvailabilityPreferencesQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerAvailabilityPreferencesQuery = { workerAvailabilityPreferences: Array<{ id: string, workerId: string, dayOfWeek: number, preference: AvailabilityPreference, note: string | null, version: number }> };

export type ShiftSwapRequestsQueryVariables = Exact<{
  workerId?: string | number | null | undefined;
  openOnly?: boolean | null | undefined;
  since?: number | null | undefined;
  limit?: number | null | undefined;
}>;


export type ShiftSwapRequestsQuery = { shiftSwapRequests: Array<{ id: string, requestingWorkerId: string, counterpartyWorkerId: string | null, status: ShiftSwapStatus, shiftDate: number, counterpartyShiftDate: number | null, reason: string | null, responseNote: string | null, respondedAt: number | null, decidedAt: number | null, version: number, requestingWorker: { id: string, firstName: string, lastName: string } | null, counterpartyWorker: { id: string, firstName: string, lastName: string } | null }> };

export type CreateShiftTemplateMutationVariables = Exact<{
  input: ShiftTemplateInput;
}>;


export type CreateShiftTemplateMutation = { createShiftTemplate: { id: string, code: string, name: string } };

export type UpdateShiftTemplateMutationVariables = Exact<{
  id: string | number;
  input: ShiftTemplateInput;
}>;


export type UpdateShiftTemplateMutation = { updateShiftTemplate: { id: string, code: string, name: string, status: EntityStatus } };

export type AssignWorkerShiftMutationVariables = Exact<{
  input: AssignShiftInput;
}>;


export type AssignWorkerShiftMutation = { assignWorkerShift: { id: string, workerId: string, shiftTemplateId: string, effectiveFrom: number } };

export type EndWorkerShiftAssignmentMutationVariables = Exact<{
  id: string | number;
  effectiveTo: number;
}>;


export type EndWorkerShiftAssignmentMutation = { endWorkerShiftAssignment: { id: string, effectiveTo: number | null } };

export type SetWorkerAvailabilityPreferenceMutationVariables = Exact<{
  input: SetAvailabilityPreferenceInput;
}>;


export type SetWorkerAvailabilityPreferenceMutation = { setWorkerAvailabilityPreference: { id: string, workerId: string, dayOfWeek: number, preference: AvailabilityPreference, note: string | null } };

export type ProposeShiftSwapMutationVariables = Exact<{
  input: ProposeShiftSwapInput;
}>;


export type ProposeShiftSwapMutation = { proposeShiftSwap: { id: string, status: ShiftSwapStatus, shiftDate: number } };

export type TransitionShiftSwapMutationVariables = Exact<{
  input: TransitionShiftSwapInput;
}>;


export type TransitionShiftSwapMutation = { transitionShiftSwap: { id: string, status: ShiftSwapStatus, respondedAt: number | null, decidedAt: number | null } };

export type ScimGroupRoleMappingTableRowFieldsFragment = { id: string, directoryId: string, externalGroupId: string, displayName: string, roleId: string, version: number, role: { id: string, name: string } | null } & { ' $fragmentName'?: 'ScimGroupRoleMappingTableRowFieldsFragment' };

export type ScimGroupRoleMappingsTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  directoryId: string | number;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ScimGroupRoleMappingsTableQuery = { scimGroupRoleMappings: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ScimGroupRoleMappingTableRowFieldsFragment': ScimGroupRoleMappingTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type SelectOptionsQueryVariables = Exact<{
  input: SelectOptionsInput;
}>;


export type SelectOptionsQuery = { selectOptions: { totalCount: number | null, edges: Array<{ cursor: string, node: { id: string, label: string, description: string | null, meta: unknown } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type WorkerPoliciesQueryVariables = Exact<{
  activeOnly?: boolean | null | undefined;
  limit?: number | null | undefined;
}>;


export type WorkerPoliciesQuery = { workerPolicies: Array<{ id: string, status: EntityStatus, code: string, title: string, summary: string | null, body: string | null, documentId: string | null, versionLabel: string, requiresSignature: boolean, appliesTo: PolicyAudience, effectiveFrom: number, version: number }> };

export type WorkerPolicyComplianceQueryVariables = Exact<{
  id: string | number;
}>;


export type WorkerPolicyComplianceQuery = { workerPolicyCompliance: { signed: number, outstanding: number, policy: { id: string, title: string, versionLabel: string }, rows: Array<{ workerId: string, workerName: string, workerType: string, acknowledgedAt: number | null, signatureName: string | null }> } };

export type WorkerPolicyAcknowledgementsQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerPolicyAcknowledgementsQuery = { workerPolicyAcknowledgements: Array<{ id: string, policyId: string, versionLabel: string, acknowledgedAt: number, signatureName: string | null, documentChecksum: string | null, policy: { id: string, title: string, versionLabel: string } | null }> };

export type ProfileChangeRequestsQueryVariables = Exact<{
  filter?: ProfileChangeFilterInput | null | undefined;
}>;


export type ProfileChangeRequestsQuery = { profileChangeRequests: Array<{ id: string, workerId: string, status: ProfileChangeStatus, note: string | null, submittedAt: number, decidedAt: number | null, decisionNote: string | null, version: number, changes: Array<{ field: string, label: string, from: string, to: string }>, worker: { id: string, firstName: string, lastName: string } | null }> };

export type CreateWorkerPolicyMutationVariables = Exact<{
  input: WorkerPolicyInput;
}>;


export type CreateWorkerPolicyMutation = { createWorkerPolicy: { id: string, code: string, title: string, versionLabel: string } };

export type UpdateWorkerPolicyMutationVariables = Exact<{
  id: string | number;
  input: WorkerPolicyInput;
}>;


export type UpdateWorkerPolicyMutation = { updateWorkerPolicy: { id: string, code: string, title: string, versionLabel: string, status: EntityStatus } };

export type DecideProfileChangeMutationVariables = Exact<{
  input: DecideProfileChangeInput;
}>;


export type DecideProfileChangeMutation = { decideProfileChange: { id: string, status: ProfileChangeStatus, decidedAt: number | null } };

export type ServiceFailureReasonCodeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, label: string, description: string, category: ServiceFailureReasonCategory, appliesTo: ServiceFailureReasonCodeAppliesTo, defaultStatusCode: string, defaultReasonCode: string, defaultExceptionCode: string, defaultNote: string, active: boolean, sortOrder: number, externalMap: unknown, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ServiceFailureReasonCodeTableRowFieldsFragment' };

export type ServiceFailureReasonCodeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ServiceFailureReasonCodeTableQuery = { serviceFailureReasonCodes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ServiceFailureReasonCodeTableRowFieldsFragment': ServiceFailureReasonCodeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ServiceFailureTableRowFieldsFragment = { id: string, shipmentId: string, shipmentMoveId: string, number: string, type: ServiceFailureType, source: ServiceFailureSource, status: ServiceFailureStatus, stopType: StopType, stopId: string, scheduledCutoff: number, actualArrival: number, gracePeriodMinutes: number, lateMinutes: number, reasonCodeId: string | null, notes: string, internalNotes: string, x12StatusCodeOverride: string, x12ReasonCodeOverride: string, x12ExceptionCode: string, detectedAt: number, version: number, shipment: { id: string, proNumber: string, bol: string | null } | null, stop: { id: string, type: StopType, sequence: number, locationId: string, location: { id: string, name: string, code: string, city: string, state: { abbreviation: string } | null } | null } | null, reasonCode: { id: string, code: string, label: string } | null } & { ' $fragmentName'?: 'ServiceFailureTableRowFieldsFragment' };

export type ServiceFailureTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  shipmentId?: string | number | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ServiceFailureTableQuery = { serviceFailures: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ServiceFailureTableRowFieldsFragment': ServiceFailureTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ServiceTypeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, description: string | null, color: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ServiceTypeTableRowFieldsFragment' };

export type ServiceTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ServiceTypeTableQuery = { serviceTypes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ServiceTypeTableRowFieldsFragment': ServiceTypeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ShipmentTypeTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: EntityStatus, code: string, description: string | null, color: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ShipmentTypeTableRowFieldsFragment' };

export type ShipmentTypeTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ShipmentTypeTableQuery = { shipmentTypes: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentTypeTableRowFieldsFragment': ShipmentTypeTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type ShipmentUserFieldsFragment = { id: string, name: string, username: string, emailAddress: string, timezone: string, status: EntityStatus, profilePicUrl: string, thumbnailUrl: string } & { ' $fragmentName'?: 'ShipmentUserFieldsFragment' };

export type ShipmentLocationFieldsFragment = { id: string, name: string, code: string, status: EntityStatus, locationCategoryId: string, stateId: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, longitude: number | null, latitude: number | null } & { ' $fragmentName'?: 'ShipmentLocationFieldsFragment' };

export type ShipmentWorkerFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string, profilePicUrl: string } & { ' $fragmentName'?: 'ShipmentWorkerFieldsFragment' };

export type ShipmentTractorFieldsFragment = { id: string, code: string } & { ' $fragmentName'?: 'ShipmentTractorFieldsFragment' };

export type ShipmentTrailerFieldsFragment = { id: string, code: string, equipmentTypeId: string } & { ' $fragmentName'?: 'ShipmentTrailerFieldsFragment' };

export type ShipmentAssignmentFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentMoveId: string | null, primaryWorkerId: string | null, tractorId: string | null, trailerId: string | null, secondaryWorkerId: string | null, status: AssignmentStatus, archivedAt: number | null, version: number, createdAt: number, updatedAt: number, tractor: { ' $fragmentRefs'?: { 'ShipmentTractorFieldsFragment': ShipmentTractorFieldsFragment } } | null, trailer: { ' $fragmentRefs'?: { 'ShipmentTrailerFieldsFragment': ShipmentTrailerFieldsFragment } } | null, primaryWorker: { ' $fragmentRefs'?: { 'ShipmentWorkerFieldsFragment': ShipmentWorkerFieldsFragment } } | null, secondaryWorker: { ' $fragmentRefs'?: { 'ShipmentWorkerFieldsFragment': ShipmentWorkerFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentAssignmentFieldsFragment' };

export type ShipmentStopFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentMoveId: string | null, locationId: string, status: StopStatus, type: StopType, scheduleType: StopScheduleType, sequence: number, pieces: number | null, weight: number | null, scheduledWindowStart: number, scheduledWindowEnd: number | null, actualArrival: number | null, actualDeparture: number | null, countLateOverride: boolean | null, countDetentionOverride: boolean | null, addressLine: string, version: number, createdAt: number, updatedAt: number, location: { ' $fragmentRefs'?: { 'ShipmentLocationFieldsFragment': ShipmentLocationFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentStopFieldsFragment' };

export type ShipmentMoveFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string | null, status: MoveStatus, loaded: boolean, sequence: number, distance: number | null, distanceSource: string | null, distanceProvider: string | null, distanceCalculatedAt: number | null, distanceRouteSignature: string | null, distanceDataVersion: string | null, distanceRoutingType: string | null, distanceUnits: string | null, distanceMetadata: unknown, version: number, createdAt: number, updatedAt: number, coverageType: MoveCoverageType, stops: Array<{ ' $fragmentRefs'?: { 'ShipmentStopFieldsFragment': ShipmentStopFieldsFragment } }>, assignment: { ' $fragmentRefs'?: { 'ShipmentAssignmentFieldsFragment': ShipmentAssignmentFieldsFragment } } | null, carrierAssignment: { ' $fragmentRefs'?: { 'ShipmentCarrierAssignmentFieldsFragment': ShipmentCarrierAssignmentFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentMoveFieldsFragment' };

export type ShipmentCarrierAssignmentFieldsFragment = { id: string, businessUnitId: string, organizationId: string, shipmentMoveId: string, carrierId: string, status: CarrierAssignmentStatus, rateMethod: CarrierRateMethod, baseRate: string, baseAmount: string, fuelSurcharge: string, accessorialTotal: string, totalCost: string, currencyCode: string, proNumber: string | null, externalDriverName: string | null, externalDriverPhone: string | null, externalTractorNumber: string | null, externalTrailerNumber: string | null, confirmedAt: number | null, canceledAt: number | null, cancellationReason: string | null, version: number, createdAt: number, updatedAt: number, carrier: { id: string, code: string, name: string, scac: string | null } | null, accessorials: Array<{ id: string, carrierAssignmentId: string, accessorialChargeId: string | null, description: string, amount: string, version: number }> | null } & { ' $fragmentName'?: 'ShipmentCarrierAssignmentFieldsFragment' };

export type ShipmentAdditionalChargeFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string, accessorialChargeId: string, isSystemGenerated: boolean, method: string, amount: string, unit: number, fuelSurchargeProgramId: string | null, fuelSurchargeDetail: unknown, detentionOccurrenceId: string | null, version: number, createdAt: number, updatedAt: number, accessorialCharge: { id: string, businessUnitId: string, organizationId: string, code: string, description: string, status: EntityStatus, method: string, rateUnit: string, amount: string, version: number, createdAt: number, updatedAt: number } | null } & { ' $fragmentName'?: 'ShipmentAdditionalChargeFieldsFragment' };

export type ShipmentCommodityFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string, commodityId: string, pieces: number, weight: number, lengthFeet: number | null, widthFeet: number | null, heightFeet: number | null, version: number, createdAt: number, updatedAt: number, commodity: { id: string, businessUnitId: string, organizationId: string, hazardousMaterialId: string | null, status: EntityStatus, name: string, description: string, minTemperature: number | null, maxTemperature: number | null, weightPerUnit: number | null, linearFeetPerUnit: number | null, maxQuantityPerShipment: number | null, freightClass: string, loadingInstructions: string, stackable: boolean, fragile: boolean, version: number, createdAt: number, updatedAt: number } | null } & { ' $fragmentName'?: 'ShipmentCommodityFieldsFragment' };

export type ShipmentRatingDetailFieldsFragment = { formulaTemplateId: string, formulaTemplateName: string, expression: string, resolvedVariables: unknown, result: number, ratedAt: number, versionNumber: number, rateQuoteId: string, agreementId: string, agreementName: string, ruleId: string, ruleLabel: string, source: string, explanation: string, breakdown: Array<{ name: string, label: string, amount: number, error: string }>, guardrail: { applied: boolean, bound: string, rawResult: number, minCharge: number | null, maxCharge: number | null } | null } & { ' $fragmentName'?: 'ShipmentRatingDetailFieldsFragment' };

export type ShipmentFieldsFragment = { id: string, businessUnitId: string, organizationId: string, sourceDocumentId: string | null, serviceTypeId: string, shipmentTypeId: string, customerId: string, tractorTypeId: string | null, trailerTypeId: string | null, ownerId: string | null, enteredById: string | null, canceledById: string | null, formulaTemplateId: string, consolidationGroupId: string | null, orderId: string | null, orderNumber: string | null, orderStatus: OrderStatus | null, status: ShipmentStatus, tenderStatus: ShipmentTenderStatus | null, entryMethod: ShipmentEntryMethod | null, proNumber: string, bol: string | null, cancelReason: string, otherChargeAmount: string, freightChargeAmount: string, baseRate: string, totalChargeAmount: string, pieces: number | null, weight: number | null, temperatureMin: number | null, temperatureMax: number | null, actualDeliveryDate: number | null, actualShipDate: number | null, canceledAt: number | null, billingTransferStatus: string | null, transferredToBillingAt: number | null, markedReadyToBillAt: number | null, billedAt: number | null, ratingUnit: number, fuelSurchargeLocked: boolean, autoRated: boolean, autoRatedAt: number | null, rateAgreementId: string | null, rateAgreementRuleId: string | null, rateQuoteId: string | null, rateOverrideAmount: string | null, rateOverrideReason: string | null, rateOverrideAt: number | null, rateLocked: boolean, version: number, createdAt: number, updatedAt: number, profitabilityEstimate: { shipmentId: string, loadedMiles: number, deadheadMiles: number, totalMiles: number, costPerMile: string, estimatedCost: string, profit: string, marginPercent: string | null, breakEvenRpm: string | null, targetMarginPercent: string | null, missingDistance: boolean } | null, ratingDetail: { ' $fragmentRefs'?: { 'ShipmentRatingDetailFieldsFragment': ShipmentRatingDetailFieldsFragment } } | null, moves: Array<{ ' $fragmentRefs'?: { 'ShipmentMoveFieldsFragment': ShipmentMoveFieldsFragment } }>, additionalCharges: Array<{ ' $fragmentRefs'?: { 'ShipmentAdditionalChargeFieldsFragment': ShipmentAdditionalChargeFieldsFragment } }>, commodities: Array<{ ' $fragmentRefs'?: { 'ShipmentCommodityFieldsFragment': ShipmentCommodityFieldsFragment } }>, customer: { id: string, businessUnitId: string, organizationId: string, stateId: string, status: EntityStatus, code: string, name: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, isGeocoded: boolean, longitude: number | null, latitude: number | null, placeId: string, externalId: string, allowConsolidation: boolean, exclusiveConsolidation: boolean, consolidationPriority: number, version: number, createdAt: number, updatedAt: number, ediPartner: { id: string, name: string, code: string } | null } | null, owner: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, formulaTemplate: { id: string, organizationId: string, businessUnitId: string, name: string, description: string, type: string, expression: string, status: string, schemaId: string, metadata: unknown, version: number, sourceTemplateId: string | null, sourceVersionNumber: number | null, currentVersionNumber: number, createdAt: number, updatedAt: number, variableDefinitions: Array<{ name: string, type: string, description: string, required: boolean, defaultValue: unknown, source: string | null }> } | null } & { ' $fragmentName'?: 'ShipmentFieldsFragment' };

export type ShipmentPageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'ShipmentPageInfoFieldsFragment' };

export type ShipmentCommentMentionFieldsFragment = { id: string, commentId: string, mentionedUserId: string, organizationId: string | null, businessUnitId: string | null, shipmentId: string | null, createdAt: number, mentionedUser: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentCommentMentionFieldsFragment' };

export type ShipmentCommentAttachmentFieldsFragment = { documentId: string, fileName: string, originalName: string | null, fileSize: number, mimeType: string, previewUrl: string | null, downloadUrl: string, createdAt: number } & { ' $fragmentName'?: 'ShipmentCommentAttachmentFieldsFragment' };

export type ShipmentCommentAcknowledgmentFieldsFragment = { id: string, commentId: string, userId: string, acknowledgedAt: number, user: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentCommentAcknowledgmentFieldsFragment' };

export type ShipmentCommentFieldsFragment = { id: string, businessUnitId: string | null, organizationId: string | null, shipmentId: string, userId: string | null, parentCommentId: string | null, replyCount: number, comment: string, body: unknown, type: ShipmentCommentType, visibility: ShipmentCommentVisibility, priority: ShipmentCommentPriority, source: ShipmentCommentSource, metadata: unknown, editedAt: number | null, pinnedAt: number | null, pinnedById: string | null, resolvedAt: number | null, resolvedById: string | null, requiresAcknowledgment: boolean, deletedAt: number | null, version: number, createdAt: number, updatedAt: number, mentionedUserIds: Array<string>, user: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, pinnedBy: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, resolvedBy: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, mentionedUsers: Array<{ ' $fragmentRefs'?: { 'ShipmentCommentMentionFieldsFragment': ShipmentCommentMentionFieldsFragment } }> | null, acknowledgments: Array<{ ' $fragmentRefs'?: { 'ShipmentCommentAcknowledgmentFieldsFragment': ShipmentCommentAcknowledgmentFieldsFragment } }> | null, attachments: Array<{ ' $fragmentRefs'?: { 'ShipmentCommentAttachmentFieldsFragment': ShipmentCommentAttachmentFieldsFragment } }> | null } & { ' $fragmentName'?: 'ShipmentCommentFieldsFragment' };

type ShipmentEventFields_ShipmentAssignmentEvent_Fragment = { __typename: 'ShipmentAssignmentEvent', moveId: string | null, assignmentId: string | null, primaryWorkerId: string | null, secondaryWorkerId: string | null, tractorId: string | null, trailerId: string | null, driverName: string | null, id: string, organizationId: string, businessUnitId: string, shipmentId: string, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFields_ShipmentAssignmentEvent_Fragment' };

type ShipmentEventFields_ShipmentCarrierEvent_Fragment = { __typename: 'ShipmentCarrierEvent', moveId: string | null, carrierId: string | null, carrierName: string | null, totalCost: string | null, reason: string | null, proNumber: string | null, id: string, organizationId: string, businessUnitId: string, shipmentId: string, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFields_ShipmentCarrierEvent_Fragment' };

type ShipmentEventFields_ShipmentCommentEvent_Fragment = { __typename: 'ShipmentCommentEvent', commentId: string | null, commentBody: string | null, commentType: ShipmentCommentType | null, commentVisibility: ShipmentCommentVisibility | null, commentPriority: ShipmentCommentPriority | null, mentionedUserIds: Array<string>, id: string, organizationId: string, businessUnitId: string, shipmentId: string, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFields_ShipmentCommentEvent_Fragment' };

type ShipmentEventFields_ShipmentHoldEvent_Fragment = { __typename: 'ShipmentHoldEvent', holdId: string | null, holdType: HoldType | null, holdSeverity: HoldSeverity | null, holdSource: string | null, id: string, organizationId: string, businessUnitId: string, shipmentId: string, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFields_ShipmentHoldEvent_Fragment' };

type ShipmentEventFields_ShipmentLifecycleEvent_Fragment = { __typename: 'ShipmentLifecycleEvent', proNumber: string | null, previousStatus: string | null, newStatus: string | null, reason: string | null, id: string, organizationId: string, businessUnitId: string, shipmentId: string, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFields_ShipmentLifecycleEvent_Fragment' };

type ShipmentEventFields_ShipmentMoveEvent_Fragment = { __typename: 'ShipmentMoveEvent', moveId: string | null, stopId: string | null, previousStatus: string | null, newStatus: string | null, id: string, organizationId: string, businessUnitId: string, shipmentId: string, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFields_ShipmentMoveEvent_Fragment' };

type ShipmentEventFields_ShipmentOwnershipEvent_Fragment = { __typename: 'ShipmentOwnershipEvent', proNumber: string | null, previousOwnerId: string | null, newOwnerId: string | null, id: string, organizationId: string, businessUnitId: string, shipmentId: string, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFields_ShipmentOwnershipEvent_Fragment' };

type ShipmentEventFields_ShipmentTenderEvent_Fragment = { __typename: 'ShipmentTenderEvent', tenderId: string | null, offerId: string | null, moveId: string | null, carrierName: string | null, rank: number | null, channel: string | null, source: string | null, reason: string | null, action: string | null, mode: string | null, error: string | null, reasons: Array<string>, warnings: Array<string>, id: string, organizationId: string, businessUnitId: string, shipmentId: string, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFields_ShipmentTenderEvent_Fragment' };

export type ShipmentEventFieldsFragment =
  | ShipmentEventFields_ShipmentAssignmentEvent_Fragment
  | ShipmentEventFields_ShipmentCarrierEvent_Fragment
  | ShipmentEventFields_ShipmentCommentEvent_Fragment
  | ShipmentEventFields_ShipmentHoldEvent_Fragment
  | ShipmentEventFields_ShipmentLifecycleEvent_Fragment
  | ShipmentEventFields_ShipmentMoveEvent_Fragment
  | ShipmentEventFields_ShipmentOwnershipEvent_Fragment
  | ShipmentEventFields_ShipmentTenderEvent_Fragment
;

export type ShipmentCommandCenterTableQueryVariables = Exact<{
  input: ShipmentsInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type ShipmentCommandCenterTableQuery = { shipments: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

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
  filter?: ShipmentCommentsFilterInput | null | undefined;
}>;


export type ShipmentCommentsQuery = { shipmentComments: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

export type ShipmentCommentRepliesQueryVariables = Exact<{
  shipmentId: string | number;
  commentId: string | number;
  first: number;
  after?: string | null | undefined;
}>;


export type ShipmentCommentRepliesQuery = { shipmentCommentReplies: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

export type ShipmentCommentCountQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type ShipmentCommentCountQuery = { shipmentCommentCount: { count: number } };

export type ShipmentEventsQueryVariables = Exact<{
  input: ShipmentEventsInput;
}>;


export type ShipmentEventsQuery = { shipmentEvents: Array<
    | { ' $fragmentRefs'?: { 'ShipmentEventFields_ShipmentAssignmentEvent_Fragment': ShipmentEventFields_ShipmentAssignmentEvent_Fragment } }
    | { ' $fragmentRefs'?: { 'ShipmentEventFields_ShipmentCarrierEvent_Fragment': ShipmentEventFields_ShipmentCarrierEvent_Fragment } }
    | { ' $fragmentRefs'?: { 'ShipmentEventFields_ShipmentCommentEvent_Fragment': ShipmentEventFields_ShipmentCommentEvent_Fragment } }
    | { ' $fragmentRefs'?: { 'ShipmentEventFields_ShipmentHoldEvent_Fragment': ShipmentEventFields_ShipmentHoldEvent_Fragment } }
    | { ' $fragmentRefs'?: { 'ShipmentEventFields_ShipmentLifecycleEvent_Fragment': ShipmentEventFields_ShipmentLifecycleEvent_Fragment } }
    | { ' $fragmentRefs'?: { 'ShipmentEventFields_ShipmentMoveEvent_Fragment': ShipmentEventFields_ShipmentMoveEvent_Fragment } }
    | { ' $fragmentRefs'?: { 'ShipmentEventFields_ShipmentOwnershipEvent_Fragment': ShipmentEventFields_ShipmentOwnershipEvent_Fragment } }
    | { ' $fragmentRefs'?: { 'ShipmentEventFields_ShipmentTenderEvent_Fragment': ShipmentEventFields_ShipmentTenderEvent_Fragment } }
  > };

export type ShipmentBillingReadinessQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type ShipmentBillingReadinessQuery = { shipmentBillingReadiness: { shipmentId: string, shipmentStatus: ShipmentStatus, canMarkReadyToInvoice: boolean, shouldAutoMarkReadyToInvoice: boolean, shouldAutoTransferToBilling: boolean, policy: { shipmentBillingRequirementEnforcement: string, rateValidationEnforcement: string, billingExceptionDisposition: string, notifyOnBillingExceptions: boolean, readyToBillAssignmentMode: string, billingQueueTransferMode: string }, requirements: Array<{ documentTypeId: string, documentTypeCode: string, documentTypeName: string, satisfied: boolean, documentCount: number, documentIds: Array<string> }>, missingRequirements: Array<{ documentTypeId: string, documentTypeCode: string, documentTypeName: string, satisfied: boolean, documentCount: number, documentIds: Array<string> }>, validationFailures: Array<{ field: string, code: string, message: string }>, warnings: Array<{ code: string, message: string, context: { documentTypeId: string | null, documentTypeCode: string | null, documentTypeName: string | null, documentCount: number | null, requirementCount: number | null, missingRequirementCount: number | null, serviceFailureIds: Array<string> | null, unresolvedCount: number | null } | null }>, serviceFailureContext: { hasUnresolved: boolean, unresolvedCount: number, serviceFailureIds: Array<string> } } };

export type ShipmentUiPolicyQueryVariables = Exact<{ [key: string]: never; }>;


export type ShipmentUiPolicyQuery = { shipmentUIPolicy: { allowMoveRemovals: boolean, checkForDuplicateBols: boolean, checkHazmatSegregation: boolean, maxShipmentWeightLimit: number, profile: unknown } };

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

export type ShipmentContractRateFieldsFragment = { applied: boolean, outcome: string, agreementId: string | null, agreementName: string, ruleId: string | null, ruleLabel: string, formulaTemplateId: string | null, formulaTemplateName: string, baseRate: string | null, linehaulAmount: string, otherChargeAmount: string, totalChargeAmount: string, previousLinehaulAmount: string, explanation: string, accessorials: Array<{ accessorialChargeId: string, description: string, method: AccessorialMethod, amount: string, unit: number }> } & { ' $fragmentName'?: 'ShipmentContractRateFieldsFragment' };

export type PreviewShipmentContractRateMutationVariables = Exact<{
  input: ShipmentInput;
}>;


export type PreviewShipmentContractRateMutation = { previewShipmentContractRate: { ' $fragmentRefs'?: { 'ShipmentContractRateFieldsFragment': ShipmentContractRateFieldsFragment } } };

export type AutoRateShipmentMutationVariables = Exact<{
  id: string | number;
}>;


export type AutoRateShipmentMutation = { autoRateShipment: { shipment: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } }, contractRate: { ' $fragmentRefs'?: { 'ShipmentContractRateFieldsFragment': ShipmentContractRateFieldsFragment } } } };

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

export type PinShipmentCommentMutationVariables = Exact<{
  shipmentId: string | number;
  commentId: string | number;
}>;


export type PinShipmentCommentMutation = { pinShipmentComment: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } };

export type UnpinShipmentCommentMutationVariables = Exact<{
  shipmentId: string | number;
  commentId: string | number;
}>;


export type UnpinShipmentCommentMutation = { unpinShipmentComment: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } };

export type ResolveShipmentCommentMutationVariables = Exact<{
  shipmentId: string | number;
  commentId: string | number;
}>;


export type ResolveShipmentCommentMutation = { resolveShipmentComment: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } };

export type UnresolveShipmentCommentMutationVariables = Exact<{
  shipmentId: string | number;
  commentId: string | number;
}>;


export type UnresolveShipmentCommentMutation = { unresolveShipmentComment: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } };

export type AcknowledgeShipmentCommentMutationVariables = Exact<{
  shipmentId: string | number;
  commentId: string | number;
}>;


export type AcknowledgeShipmentCommentMutation = { acknowledgeShipmentComment: { ' $fragmentRefs'?: { 'ShipmentCommentFieldsFragment': ShipmentCommentFieldsFragment } } };

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
  includeTotalCount?: boolean | null | undefined;
}>;


export type StoredMileageTableQuery = { storedMileages: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'StoredMileageTableRowFieldsFragment': StoredMileageTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type TcaSubscriptionTableRowFieldsFragment = { id: string, organizationId: string, businessUnitId: string, userId: string, name: string, tableName: string, recordId: string | null, eventTypes: Array<string>, conditions: Array<unknown>, conditionMatch: string, watchedColumns: Array<string>, customTitle: string, customMessage: string, topic: string, priority: string, status: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'TcaSubscriptionTableRowFieldsFragment' };

export type TcaSubscriptionTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type TcaSubscriptionTableQuery = { tcaSubscriptions: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TcaSubscriptionTableRowFieldsFragment': TcaSubscriptionTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type TableConfigurationFieldsFragment = { id: string, organizationId: string, businessUnitId: string, userId: string, name: string, description: string, resource: string, tableConfig: unknown, visibility: ConfigurationVisibility, isDefault: boolean, isOrgDefault: boolean, version: number, createdAt: number, updatedAt: number, user: { id: string, name: string, profilePicUrl: string } | null } & { ' $fragmentName'?: 'TableConfigurationFieldsFragment' };

export type TableConfigurationTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  resource?: string | null | undefined;
  visibility?: ConfigurationVisibility | null | undefined;
  includeTotalCount?: boolean | null | undefined;
}>;


export type TableConfigurationTableQuery = { tableConfigurations: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TableConfigurationFieldsFragment': TableConfigurationFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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

export type VehiclePositionsQueryVariables = Exact<{
  maxAgeSeconds?: number | null | undefined;
}>;


export type VehiclePositionsQuery = { vehiclePositions: Array<{ tractorId: string, tractorCode: string, provider: string, providerVehicleId: string, latitude: number, longitude: number, headingDegrees: number, speedMph: number, engineState: string | null, fuelPercent: number | null, odometerMeters: number | null, formattedLocation: string | null, recordedAt: number, receivedAt: number, primaryWorkerId: string | null, primaryWorkerName: string | null }> };

export type WorkerHosStatesQueryVariables = Exact<{
  workerIds?: Array<string | number> | string | number | null | undefined;
  limit?: number | null | undefined;
}>;


export type WorkerHosStatesQuery = { workerHosStates: Array<{ workerId: string, workerName: string, provider: string, providerDriverId: string, dutyStatus: string | null, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, cycleTomorrowMs: number, breakRemainingMs: number, cycleStartedAt: number | null, shiftDrivingViolationMs: number, cycleViolationMs: number, currentVehicleId: string | null, currentTractorId: string | null, rulesetCycle: string | null, rulesetShift: string | null, rulesetJurisdiction: string | null, driveLimitMs: number, shiftLimitMs: number, cycleLimitMs: number, breakLimitMs: number, recordedAt: number }> };

export type WorkerHosStateQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerHosStateQuery = { workerHosState: { workerId: string, workerName: string, provider: string, providerDriverId: string, dutyStatus: string | null, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, cycleTomorrowMs: number, breakRemainingMs: number, cycleStartedAt: number | null, shiftDrivingViolationMs: number, cycleViolationMs: number, currentVehicleId: string | null, currentTractorId: string | null, rulesetCycle: string | null, rulesetShift: string | null, rulesetJurisdiction: string | null, driveLimitMs: number, shiftLimitMs: number, cycleLimitMs: number, breakLimitMs: number, recordedAt: number } | null };

export type WorkerHosViolationsQueryVariables = Exact<{
  workerId?: string | number | null | undefined;
  since?: number | null | undefined;
  limit?: number | null | undefined;
}>;


export type WorkerHosViolationsQuery = { workerHosViolations: Array<{ workerId: string, violationType: string, description: string | null, durationMs: number, violationStartAt: number, dayStartAt: number | null, dayEndAt: number | null, detectedAt: number }> };

export type TelematicsStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type TelematicsStatusQuery = { telematicsStatus: { provider: string, enabled: boolean, configured: boolean, webhookConfigured: boolean, lastPolledAt: number | null, lastSuccessAt: number | null, failureCount: number, lastError: string | null, mappedTractors: number, totalTractors: number, mappedWorkers: number } };

export type WorkerHosLogsQueryVariables = Exact<{
  workerId: string | number;
  startTime: number;
  endTime: number;
}>;


export type WorkerHosLogsQuery = { workerHosLogs: Array<{ hosStatusType: string, logStartAt: number, logEndAt: number | null, remark: string | null, vehicleId: string | null, vehicleName: string | null, latitude: number | null, longitude: number | null, codrivers: Array<string> | null }> };

export type WorkerHosDailyLogsQueryVariables = Exact<{
  workerId: string | number;
  startDate: string;
  endDate: string;
}>;


export type WorkerHosDailyLogsQuery = { workerHosDailyLogs: Array<{ startAt: number, endAt: number, driveDistanceMeters: number, activeDurationMs: number, driveDurationMs: number, onDutyDurationMs: number, offDutyDurationMs: number, sleeperBerthDurationMs: number, personalConveyanceDurationMs: number, yardMoveDurationMs: number, isCertified: boolean, certifiedAt: number | null, shippingDocs: string | null, vehicleNames: Array<string> | null }> };

export type ShipmentDriverFeasibilityQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type ShipmentDriverFeasibilityQuery = { shipmentDriverFeasibility: Array<{ workerId: string, workerName: string, dutyStatus: string | null, driveRemainingMs: number, shiftRemainingMs: number, cycleRemainingMs: number, deadheadMiles: number | null, estimatedDriveMs: number, verdict: string, reasons: Array<string>, tractorId: string | null, tractorCode: string | null, recordedAt: number }> };

export type VehicleInspectionsQueryVariables = Exact<{
  tractorId?: string | number | null | undefined;
  workerId?: string | number | null | undefined;
  since?: number | null | undefined;
  limit?: number | null | undefined;
}>;


export type VehicleInspectionsQuery = { vehicleInspections: Array<{ id: string, provider: string, inspectionType: string, safetyStatus: string, tractorId: string | null, workerId: string | null, workerName: string | null, startedAt: number, endedAt: number, odometerMeters: number | null, location: string | null, signed: boolean, defectCount: number, unresolvedDefectCount: number, defects: unknown }> };

export type WorkerFormSubmissionsQueryVariables = Exact<{
  workerId: string | number;
  startTime: number;
  endTime: number;
}>;


export type WorkerFormSubmissionsQuery = { workerFormSubmissions: Array<{ id: string, templateId: string, templateName: string, submittedAt: number, fields: Array<{ label: string, value: string }> }> };

export type HosCertificationSummaryQueryVariables = Exact<{
  startDate: string;
  endDate: string;
}>;


export type HosCertificationSummaryQuery = { hosCertificationSummary: Array<{ workerId: string, workerName: string, uncertifiedDays: number, totalDays: number }> };

export type ShipmentFormSubmissionsQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type ShipmentFormSubmissionsQuery = { shipmentFormSubmissions: Array<{ id: string, provider: string, templateId: string, templateName: string, workerId: string | null, workerName: string, shipmentId: string | null, stopId: string | null, submittedAt: number, applied: boolean, appliedFields: number, fields: Array<{ label: string, type: string, value: string }> }> };

export type TelematicsFormMappingsQueryVariables = Exact<{ [key: string]: never; }>;


export type TelematicsFormMappingsQuery = { telematicsFormMappings: Array<{ id: string, provider: string, templateId: string, templateName: string, name: string, description: string, enabled: boolean, version: number, items: Array<{ id: string, sourceFieldLabel: string, targetKind: string, targetField: string, targetCustomFieldKey: string }> }> };

export type SaveTelematicsFormMappingMutationVariables = Exact<{
  input: SaveTelematicsFormMappingInput;
}>;


export type SaveTelematicsFormMappingMutation = { saveTelematicsFormMapping: { id: string, name: string } };

export type DeleteTelematicsFormMappingMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteTelematicsFormMappingMutation = { deleteTelematicsFormMapping: boolean };

export type TendersByShipmentQueryVariables = Exact<{
  shipmentId: string | number;
}>;


export type TendersByShipmentQuery = { tendersByShipment: Array<{ id: string, shipmentId: string, shipmentMoveId: string, routingGuideId: string | null, mode: TenderMode, status: TenderStatus, currentRank: number, cancellationReason: string, acceptedOfferId: string | null, acceptedAt: number | null, exhaustedAt: number | null, canceledAt: number | null, createdAt: number, updatedAt: number, routingGuide: { id: string, name: string, specificity: number } | null, offers: Array<{ id: string, tenderId: string, carrierId: string, rank: number, rateMethod: CarrierRateMethod, rate: string, offerTtlSeconds: number, channel: TenderChannel, status: TenderOfferStatus, recipientEmail: string, sentAt: number | null, expiresAt: number | null, respondedAt: number | null, responseSource: TenderResponseSource | null, declineReason: string, deliveryError: string, createdAt: number, updatedAt: number, carrier: { id: string, name: string, scac: string | null } | null }> | null }> };

export type LiveTenderByMoveQueryVariables = Exact<{
  moveId: string | number;
}>;


export type LiveTenderByMoveQuery = { liveTenderByMove: { id: string, shipmentId: string, shipmentMoveId: string, routingGuideId: string | null, mode: TenderMode, status: TenderStatus, currentRank: number, cancellationReason: string, acceptedOfferId: string | null, acceptedAt: number | null, exhaustedAt: number | null, canceledAt: number | null, createdAt: number, updatedAt: number, routingGuide: { id: string, name: string, specificity: number } | null, offers: Array<{ id: string, tenderId: string, carrierId: string, rank: number, rateMethod: CarrierRateMethod, rate: string, offerTtlSeconds: number, channel: TenderChannel, status: TenderOfferStatus, recipientEmail: string, sentAt: number | null, expiresAt: number | null, respondedAt: number | null, responseSource: TenderResponseSource | null, declineReason: string, deliveryError: string, createdAt: number, updatedAt: number, carrier: { id: string, name: string, scac: string | null } | null }> | null } | null };

export type TimesheetsQueryVariables = Exact<{
  filter?: TimesheetFilterInput | null | undefined;
}>;


export type TimesheetsQuery = { timesheets: Array<{ id: string, workerId: string, status: TimesheetStatus, periodStart: number, periodEnd: number, regularMinutes: number, overtimeMinutes: number, paidLeaveMinutes: number, totalMinutes: number, entryCount: number, overtimeThresholdMinutes: number, submittedAt: number | null, approvedAt: number | null, decisionNote: string | null, payrollExportId: string | null, version: number, worker: { id: string, firstName: string, lastName: string, type: WorkerType } | null }> };

export type TimesheetQueryVariables = Exact<{
  id: string | number;
}>;


export type TimesheetQuery = { timesheet: { id: string, workerId: string, status: TimesheetStatus, periodStart: number, periodEnd: number, regularMinutes: number, overtimeMinutes: number, paidLeaveMinutes: number, totalMinutes: number, entryCount: number, overtimeThresholdMinutes: number, submittedAt: number | null, approvedAt: number | null, decisionNote: string | null, payrollExportId: string | null, version: number, worker: { id: string, firstName: string, lastName: string } | null, entries: Array<{ id: string, workerId: string, source: TimeEntrySource, clockedInAt: number, clockedOutAt: number | null, breakMinutes: number, paidMinutes: number, note: string | null, editReason: string | null, version: number }> | null } };

export type OpenTimeClockEntryQueryVariables = Exact<{
  workerId: string | number;
}>;


export type OpenTimeClockEntryQuery = { openTimeClockEntry: { id: string, workerId: string, clockedInAt: number, source: TimeEntrySource, note: string | null } | null };

export type OpenTimeClockEntriesQueryVariables = Exact<{
  teamOnly?: boolean | null | undefined;
  limit?: number | null | undefined;
}>;


export type OpenTimeClockEntriesQuery = { openTimeClockEntries: Array<{ id: string, workerId: string, source: TimeEntrySource, clockedInAt: number, breakMinutes: number, note: string | null, worker: { id: string, firstName: string, lastName: string, profilePicUrl: string, fleetCode: { id: string, code: string, color: string } | null } | null }> };

export type TimeClockEntriesQueryVariables = Exact<{
  workerId: string | number;
  from?: number | null | undefined;
  to?: number | null | undefined;
  limit?: number | null | undefined;
}>;


export type TimeClockEntriesQuery = { timeClockEntries: Array<{ id: string, workerId: string, timesheetId: string | null, source: TimeEntrySource, clockedInAt: number, clockedOutAt: number | null, breakMinutes: number, paidMinutes: number, note: string | null, editReason: string | null, version: number }> };

export type PayrollExportsQueryVariables = Exact<{
  limit?: number | null | undefined;
}>;


export type PayrollExportsQuery = { payrollExports: Array<{ id: string, status: PayrollExportStatus, periodStart: number, periodEnd: number, timesheetCount: number, regularMinutes: number, overtimeMinutes: number, paidLeaveMinutes: number, generatedAt: number | null, voidedAt: number | null, voidReason: string | null, note: string | null, version: number }> };

export type PayrollExportRowsQueryVariables = Exact<{
  id: string | number;
}>;


export type PayrollExportRowsQuery = { payrollExportRows: Array<{ workerId: string, workerName: string, periodStart: number, periodEnd: number, regularMinutes: number, overtimeMinutes: number, paidLeaveMinutes: number }> };

export type ClockInMutationVariables = Exact<{
  input: ClockInput;
}>;


export type ClockInMutation = { clockIn: { id: string, clockedInAt: number } };

export type ClockOutMutationVariables = Exact<{
  input: ClockInput;
}>;


export type ClockOutMutation = { clockOut: { id: string, clockedOutAt: number | null, paidMinutes: number } };

export type RecordTimeEntryMutationVariables = Exact<{
  input: RecordTimeEntryInput;
}>;


export type RecordTimeEntryMutation = { recordTimeEntry: { id: string, clockedInAt: number, clockedOutAt: number | null, paidMinutes: number } };

export type DeleteTimeEntryMutationVariables = Exact<{
  input: DeleteTimeEntryInput;
}>;


export type DeleteTimeEntryMutation = { deleteTimeEntry: boolean };

export type TransitionTimesheetMutationVariables = Exact<{
  input: TransitionTimesheetInput;
}>;


export type TransitionTimesheetMutation = { transitionTimesheet: { id: string, status: TimesheetStatus, submittedAt: number | null, approvedAt: number | null } };

export type GeneratePayrollExportMutationVariables = Exact<{
  input: GeneratePayrollExportInput;
}>;


export type GeneratePayrollExportMutation = { generatePayrollExport: { id: string, status: PayrollExportStatus, timesheetCount: number, regularMinutes: number, overtimeMinutes: number } };

export type VoidPayrollExportMutationVariables = Exact<{
  input: VoidPayrollExportInput;
}>;


export type VoidPayrollExportMutation = { voidPayrollExport: { id: string, status: PayrollExportStatus, voidedAt: number | null } };

export type UserTableRowFieldsFragment = { id: string, businessUnitId: string, currentOrganizationId: string, status: EntityStatus, name: string, username: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string, timezone: string, isLocked: boolean, mustChangePassword: boolean, version: number, lastLoginAt: number | null, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'UserTableRowFieldsFragment' };

export type UserTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type UserTableQuery = { users: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'UserTableRowFieldsFragment': UserTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type WorkerChecklistTemplateItemFieldsFragment = { id: string, templateId: string, label: string, description: string | null, kind: WorkerChecklistItemKind, required: boolean, dueOffsetDays: number, owner: WorkerChecklistOwner, credentialTypeId: string | null, documentTypeId: string | null, documentTypeName: string | null, sortOrder: number, credentialType: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'WorkerChecklistTemplateItemFieldsFragment' };

export type WorkerChecklistTemplateFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string | null, kind: WorkerChecklistKind, trigger: WorkerChecklistTrigger, status: EntityStatus, isDefault: boolean, openChecklistCount: number, version: number, createdAt: number, updatedAt: number, items: Array<{ id: string, templateId: string, label: string, description: string | null, kind: WorkerChecklistItemKind, required: boolean, dueOffsetDays: number, owner: WorkerChecklistOwner, credentialTypeId: string | null, documentTypeId: string | null, documentTypeName: string | null, sortOrder: number, credentialType: { id: string, code: string, name: string } | null }> } & { ' $fragmentName'?: 'WorkerChecklistTemplateFieldsFragment' };

export type WorkerChecklistItemFieldsFragment = { id: string, checklistId: string, label: string, description: string | null, kind: WorkerChecklistItemKind, required: boolean, owner: WorkerChecklistOwner, dueAt: number | null, overdue: boolean, credentialTypeId: string | null, documentTypeId: string | null, status: WorkerChecklistItemStatus, completedById: string | null, completedAt: number | null, autoCompleted: boolean, note: string | null, evidenceDocumentId: string | null, evidenceCredentialId: string | null, sortOrder: number, version: number, completedBy: { id: string, name: string } | null, evidenceDocument: { id: string, originalName: string, fileType: string } | null, evidenceCredential: { id: string, number: string | null, expiresAt: number | null } | null, credentialType: { id: string, code: string, name: string } | null } & { ' $fragmentName'?: 'WorkerChecklistItemFieldsFragment' };

export type WorkerChecklistFieldsFragment = { id: string, workerId: string, templateId: string | null, name: string, kind: WorkerChecklistKind, status: WorkerChecklistStatus, startedAt: number, dueAt: number | null, completedAt: number | null, cancelledAt: number | null, cancelReason: string | null, sourceEventId: string | null, startedById: string | null, version: number, createdAt: number, updatedAt: number, startedBy: { id: string, name: string } | null, progress: { total: number, settled: number, requiredTotal: number, requiredDone: number, overdue: number, percent: number, complete: boolean }, items: Array<{ id: string, checklistId: string, label: string, description: string | null, kind: WorkerChecklistItemKind, required: boolean, owner: WorkerChecklistOwner, dueAt: number | null, overdue: boolean, credentialTypeId: string | null, documentTypeId: string | null, status: WorkerChecklistItemStatus, completedById: string | null, completedAt: number | null, autoCompleted: boolean, note: string | null, evidenceDocumentId: string | null, evidenceCredentialId: string | null, sortOrder: number, version: number, completedBy: { id: string, name: string } | null, evidenceDocument: { id: string, originalName: string, fileType: string } | null, evidenceCredential: { id: string, number: string | null, expiresAt: number | null } | null, credentialType: { id: string, code: string, name: string } | null }> } & { ' $fragmentName'?: 'WorkerChecklistFieldsFragment' };

export type WorkerChecklistTemplateTableQueryVariables = Exact<{
  input: WorkerChecklistTemplatesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type WorkerChecklistTemplateTableQuery = { workerChecklistTemplates: { totalCount?: number | null, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'WorkerChecklistTemplateFieldsFragment': WorkerChecklistTemplateFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type ActiveWorkerChecklistTemplatesQueryVariables = Exact<{ [key: string]: never; }>;


export type ActiveWorkerChecklistTemplatesQuery = { activeWorkerChecklistTemplates: Array<{ ' $fragmentRefs'?: { 'WorkerChecklistTemplateFieldsFragment': WorkerChecklistTemplateFieldsFragment } }> };

export type WorkerChecklistsQueryVariables = Exact<{
  workerId: string | number;
  includeClosed?: boolean | null | undefined;
}>;


export type WorkerChecklistsQuery = { workerChecklists: Array<{ ' $fragmentRefs'?: { 'WorkerChecklistFieldsFragment': WorkerChecklistFieldsFragment } }> };

export type CreateWorkerChecklistTemplateMutationVariables = Exact<{
  input: WorkerChecklistTemplateInput;
}>;


export type CreateWorkerChecklistTemplateMutation = { createWorkerChecklistTemplate: { ' $fragmentRefs'?: { 'WorkerChecklistTemplateFieldsFragment': WorkerChecklistTemplateFieldsFragment } } };

export type UpdateWorkerChecklistTemplateMutationVariables = Exact<{
  id: string | number;
  input: WorkerChecklistTemplateInput;
}>;


export type UpdateWorkerChecklistTemplateMutation = { updateWorkerChecklistTemplate: { ' $fragmentRefs'?: { 'WorkerChecklistTemplateFieldsFragment': WorkerChecklistTemplateFieldsFragment } } };

export type ArchiveWorkerChecklistTemplateMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type ArchiveWorkerChecklistTemplateMutation = { archiveWorkerChecklistTemplate: { ' $fragmentRefs'?: { 'WorkerChecklistTemplateFieldsFragment': WorkerChecklistTemplateFieldsFragment } } };

export type RestoreWorkerChecklistTemplateMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type RestoreWorkerChecklistTemplateMutation = { restoreWorkerChecklistTemplate: { ' $fragmentRefs'?: { 'WorkerChecklistTemplateFieldsFragment': WorkerChecklistTemplateFieldsFragment } } };

export type StartWorkerChecklistMutationVariables = Exact<{
  input: StartWorkerChecklistInput;
}>;


export type StartWorkerChecklistMutation = { startWorkerChecklist: { ' $fragmentRefs'?: { 'WorkerChecklistFieldsFragment': WorkerChecklistFieldsFragment } } };

export type CompleteWorkerChecklistItemMutationVariables = Exact<{
  input: WorkerChecklistItemActionInput;
}>;


export type CompleteWorkerChecklistItemMutation = { completeWorkerChecklistItem: { ' $fragmentRefs'?: { 'WorkerChecklistFieldsFragment': WorkerChecklistFieldsFragment } } };

export type SkipWorkerChecklistItemMutationVariables = Exact<{
  input: WorkerChecklistItemActionInput;
}>;


export type SkipWorkerChecklistItemMutation = { skipWorkerChecklistItem: { ' $fragmentRefs'?: { 'WorkerChecklistFieldsFragment': WorkerChecklistFieldsFragment } } };

export type MarkWorkerChecklistItemNotApplicableMutationVariables = Exact<{
  input: WorkerChecklistItemActionInput;
}>;


export type MarkWorkerChecklistItemNotApplicableMutation = { markWorkerChecklistItemNotApplicable: { ' $fragmentRefs'?: { 'WorkerChecklistFieldsFragment': WorkerChecklistFieldsFragment } } };

export type ReopenWorkerChecklistItemMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type ReopenWorkerChecklistItemMutation = { reopenWorkerChecklistItem: { ' $fragmentRefs'?: { 'WorkerChecklistFieldsFragment': WorkerChecklistFieldsFragment } } };

export type CancelWorkerChecklistMutationVariables = Exact<{
  input: CancelWorkerChecklistInput;
}>;


export type CancelWorkerChecklistMutation = { cancelWorkerChecklist: { ' $fragmentRefs'?: { 'WorkerChecklistFieldsFragment': WorkerChecklistFieldsFragment } } };

export type WorkerCredentialTypeFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string | null, category: WorkerCredentialCategory, status: EntityStatus, isRequired: boolean, requiredForDriverTypes: Array<DriverType>, renewalWindowDays: number, validityMonths: number | null, requiresNumber: boolean, requiresDocument: boolean, profileField: string | null, isSystem: boolean, sortOrder: number, activeCredentialCount: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'WorkerCredentialTypeFieldsFragment' };

export type WorkerCredentialFieldsFragment = { id: string, workerId: string, credentialTypeId: string, status: WorkerCredentialStatus, number: string | null, issuingAuthority: string | null, issuedAt: number | null, expiresAt: number | null, documentId: string | null, notes: string | null, verifiedById: string | null, verifiedAt: number | null, archivedById: string | null, archivedAt: number | null, archiveReason: string | null, health: WorkerCredentialHealth, daysUntilExpiry: number | null, version: number, createdAt: number, updatedAt: number, credentialType: { id: string, code: string, name: string, category: WorkerCredentialCategory, isRequired: boolean, renewalWindowDays: number, validityMonths: number | null, requiresNumber: boolean, requiresDocument: boolean, profileField: string | null, isSystem: boolean, sortOrder: number } | null, document: { id: string, fileName: string, originalName: string, fileType: string, fileSize: number, createdAt: number } | null, verifiedBy: { id: string, name: string } | null } & { ' $fragmentName'?: 'WorkerCredentialFieldsFragment' };

export type WorkerCredentialTypeTableQueryVariables = Exact<{
  input: WorkerCredentialTypesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type WorkerCredentialTypeTableQuery = { workerCredentialTypes: { totalCount?: number | null, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'WorkerCredentialTypeFieldsFragment': WorkerCredentialTypeFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type WorkerCredentialsQueryVariables = Exact<{
  workerId: string | number;
  includeArchived?: boolean | null | undefined;
}>;


export type WorkerCredentialsQuery = { workerCredentials: Array<{ ' $fragmentRefs'?: { 'WorkerCredentialFieldsFragment': WorkerCredentialFieldsFragment } }> };

export type WorkerCredentialSummaryQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerCredentialSummaryQuery = { workerCredentialSummary: { workerId: string, complianceStatus: ComplianceStatus, requiredCount: number, validCount: number, expiringCount: number, expiredCount: number, missingCount: number, items: Array<{ health: WorkerCredentialHealth, daysUntilExpiry: number | null, required: boolean, credentialType: { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string | null, category: WorkerCredentialCategory, status: EntityStatus, isRequired: boolean, requiredForDriverTypes: Array<DriverType>, renewalWindowDays: number, validityMonths: number | null, requiresNumber: boolean, requiresDocument: boolean, profileField: string | null, isSystem: boolean, sortOrder: number, activeCredentialCount: number, version: number, createdAt: number, updatedAt: number }, credential: { id: string, workerId: string, credentialTypeId: string, status: WorkerCredentialStatus, number: string | null, issuingAuthority: string | null, issuedAt: number | null, expiresAt: number | null, documentId: string | null, notes: string | null, verifiedById: string | null, verifiedAt: number | null, archivedById: string | null, archivedAt: number | null, archiveReason: string | null, health: WorkerCredentialHealth, daysUntilExpiry: number | null, version: number, createdAt: number, updatedAt: number, credentialType: { id: string, code: string, name: string, category: WorkerCredentialCategory, isRequired: boolean, renewalWindowDays: number, validityMonths: number | null, requiresNumber: boolean, requiresDocument: boolean, profileField: string | null, isSystem: boolean, sortOrder: number } | null, document: { id: string, fileName: string, originalName: string, fileType: string, fileSize: number, createdAt: number } | null, verifiedBy: { id: string, name: string } | null } | null }> } };

export type CredentialExpiryForecastQueryVariables = Exact<{
  days?: number | null | undefined;
  limit?: number | null | undefined;
}>;


export type CredentialExpiryForecastQuery = { credentialExpiryForecast: { horizonDays: number, expiringCount: number, expiredCount: number, items: Array<{ health: WorkerCredentialHealth, daysUntilExpiry: number, credential: { id: string, workerId: string, expiresAt: number | null, number: string | null, credentialType: { id: string, code: string, name: string, category: WorkerCredentialCategory, isRequired: boolean } | null, worker: { id: string, firstName: string, lastName: string, fleetCodeId: string | null } | null } }> } };

export type CreateWorkerCredentialTypeMutationVariables = Exact<{
  input: WorkerCredentialTypeInput;
}>;


export type CreateWorkerCredentialTypeMutation = { createWorkerCredentialType: { ' $fragmentRefs'?: { 'WorkerCredentialTypeFieldsFragment': WorkerCredentialTypeFieldsFragment } } };

export type UpdateWorkerCredentialTypeMutationVariables = Exact<{
  id: string | number;
  input: WorkerCredentialTypeInput;
}>;


export type UpdateWorkerCredentialTypeMutation = { updateWorkerCredentialType: { ' $fragmentRefs'?: { 'WorkerCredentialTypeFieldsFragment': WorkerCredentialTypeFieldsFragment } } };

export type ArchiveWorkerCredentialTypeMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type ArchiveWorkerCredentialTypeMutation = { archiveWorkerCredentialType: { ' $fragmentRefs'?: { 'WorkerCredentialTypeFieldsFragment': WorkerCredentialTypeFieldsFragment } } };

export type RestoreWorkerCredentialTypeMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type RestoreWorkerCredentialTypeMutation = { restoreWorkerCredentialType: { ' $fragmentRefs'?: { 'WorkerCredentialTypeFieldsFragment': WorkerCredentialTypeFieldsFragment } } };

export type CreateWorkerCredentialMutationVariables = Exact<{
  input: WorkerCredentialInput;
}>;


export type CreateWorkerCredentialMutation = { createWorkerCredential: { ' $fragmentRefs'?: { 'WorkerCredentialFieldsFragment': WorkerCredentialFieldsFragment } } };

export type UpdateWorkerCredentialMutationVariables = Exact<{
  input: UpdateWorkerCredentialInput;
}>;


export type UpdateWorkerCredentialMutation = { updateWorkerCredential: { ' $fragmentRefs'?: { 'WorkerCredentialFieldsFragment': WorkerCredentialFieldsFragment } } };

export type VerifyWorkerCredentialMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type VerifyWorkerCredentialMutation = { verifyWorkerCredential: { ' $fragmentRefs'?: { 'WorkerCredentialFieldsFragment': WorkerCredentialFieldsFragment } } };

export type ArchiveWorkerCredentialMutationVariables = Exact<{
  input: ArchiveWorkerCredentialInput;
}>;


export type ArchiveWorkerCredentialMutation = { archiveWorkerCredential: { ' $fragmentRefs'?: { 'WorkerCredentialFieldsFragment': WorkerCredentialFieldsFragment } } };

export type AttachWorkerCredentialDocumentMutationVariables = Exact<{
  input: AttachWorkerCredentialDocumentInput;
}>;


export type AttachWorkerCredentialDocumentMutation = { attachWorkerCredentialDocument: { ' $fragmentRefs'?: { 'WorkerCredentialFieldsFragment': WorkerCredentialFieldsFragment } } };

export type DriverQualificationFileQueryVariables = Exact<{
  workerId: string | number;
}>;


export type DriverQualificationFileQuery = { driverQualificationFile: { workerId: string, complete: boolean, missingRequired: number, expiringSoon: number, expired: number, outstanding: number, hireDate: number, terminationDate: number | null, safetyHistoryDueAt: number, safetyHistoryLate: boolean, retentionExpiresAt: number | null, purgeEligible: boolean, items: Array<{ section: DqfSection, code: string, name: string, status: DqfItemStatus, detail: string, regulation: string | null, expiresAt: number | null }>, verifications: Array<{ id: string, workerId: string, employerName: string, employerDotNumber: string | null, employerMcNumber: string | null, contactName: string | null, contactPhone: string | null, contactEmail: string | null, employedFrom: number | null, employedTo: number | null, wasDotRegulated: boolean, status: EmploymentVerificationStatus, method: EmploymentVerificationMethod, requestedAt: number | null, responseReceivedAt: number | null, lastFollowUpAt: number | null, followUpCount: number, drugAlcoholResponseReceivedAt: number | null, hadAccidents: boolean, accidentCount: number, hadDrugAlcoholViolations: boolean, findings: string | null, notes: string | null, documentId: string | null, version: number }> } };

export type OutstandingEmploymentVerificationsQueryVariables = Exact<{ [key: string]: never; }>;


export type OutstandingEmploymentVerificationsQuery = { outstandingEmploymentVerifications: Array<{ id: string, workerId: string, employerName: string, status: EmploymentVerificationStatus, method: EmploymentVerificationMethod, requestedAt: number | null, lastFollowUpAt: number | null, followUpCount: number, version: number }> };

export type DqfRetentionCandidatesQueryVariables = Exact<{ [key: string]: never; }>;


export type DqfRetentionCandidatesQuery = { dqfRetentionCandidates: Array<{ workerId: string, firstName: string, lastName: string, terminationDate: number }> };

export type RecordEmploymentVerificationMutationVariables = Exact<{
  input: RecordEmploymentVerificationInput;
}>;


export type RecordEmploymentVerificationMutation = { recordEmploymentVerification: { id: string, employerName: string, status: EmploymentVerificationStatus, version: number } };

export type UpdateEmploymentVerificationMutationVariables = Exact<{
  input: UpdateEmploymentVerificationInput;
}>;


export type UpdateEmploymentVerificationMutation = { updateEmploymentVerification: { id: string, employerName: string, status: EmploymentVerificationStatus, version: number } };

export type MarkEmploymentVerificationRequestedMutationVariables = Exact<{
  id: string | number;
}>;


export type MarkEmploymentVerificationRequestedMutation = { markEmploymentVerificationRequested: { id: string, status: EmploymentVerificationStatus, requestedAt: number | null, version: number } };

export type RecordEmploymentVerificationFollowUpMutationVariables = Exact<{
  id: string | number;
}>;


export type RecordEmploymentVerificationFollowUpMutation = { recordEmploymentVerificationFollowUp: { id: string, followUpCount: number, lastFollowUpAt: number | null, version: number } };

export type DeleteEmploymentVerificationMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteEmploymentVerificationMutation = { deleteEmploymentVerification: boolean };

export type WorkerDotTestFieldsFragment = { id: string, workerId: string, testType: DotTestType, substance: DotTestSubstance, status: DotTestStatus, result: DotTestResult, isDot: boolean, reason: string | null, scheduledAt: number | null, collectedAt: number | null, resultAt: number | null, collectionSite: string | null, collectorName: string | null, specimenId: string | null, labName: string | null, mroName: string | null, mroVerifiedAt: number | null, alcoholConcentration: string | null, safetyEventId: string | null, drawEntryId: string | null, documentId: string | null, notes: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'WorkerDotTestFieldsFragment' };

export type WorkerDotViolationFieldsFragment = { id: string, workerId: string, violationType: DotViolationType, status: DotViolationStatus, occurredAt: number, sourceTestId: string | null, reportedToClearinghouseAt: number | null, sapName: string | null, sapReferredAt: number | null, sapEvaluationCompletedAt: number | null, rtdTestId: string | null, rtdCompletedAt: number | null, followUpTestCount: number, followUpTestsCompleted: number, followUpEndsAt: number | null, resolvedAt: number | null, documentId: string | null, notes: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'WorkerDotViolationFieldsFragment' };

export type ClearinghouseQueryFieldsFragment = { id: string, workerId: string, queryType: ClearinghouseQueryType, result: ClearinghouseResult, consentObtainedAt: number | null, consentExpiresAt: number | null, requestedAt: number, completedAt: number | null, violationCount: number, reference: string | null, documentId: string | null, notes: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ClearinghouseQueryFieldsFragment' };

export type DotRandomPoolFieldsFragment = { id: string, code: string, name: string, description: string | null, status: EntityStatus, period: RandomPeriod, drugRatePercent: number, alcoholRatePercent: number, includedDriverTypes: Array<string>, isDefault: boolean, meetsDotMinimums: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'DotRandomPoolFieldsFragment' };

export type DotRandomDrawEntryFieldsFragment = { id: string, drawId: string, workerId: string, substance: DotTestSubstance, rank: number, status: RandomEntryStatus, notifiedAt: number | null, completedAt: number | null, testId: string | null, excuseReason: string | null, version: number, worker: { id: string, firstName: string, lastName: string } | null } & { ' $fragmentName'?: 'DotRandomDrawEntryFieldsFragment' };

export type DotRandomDrawFieldsFragment = { id: string, poolId: string, periodKey: string, periodStart: number, periodEnd: number, status: RandomDrawStatus, poolSize: number, drugTarget: number, alcoholTarget: number, drugSelected: number, alcoholSelected: number, seed: string, method: string, notes: string | null, drawnAt: number, finalizedAt: number | null, version: number } & { ' $fragmentName'?: 'DotRandomDrawFieldsFragment' };

export type WorkerDrugAlcoholFileQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerDrugAlcoholFileQuery = { workerDrugAlcoholStanding: { status: DrugAlcoholStatus, returnToDuty: ReturnToDutyStatus, lastClearinghouseQueryAt: number | null, nextClearinghouseQueryDue: number | null, hasPreEmploymentTest: boolean, hasPreEmploymentQuery: boolean, openTestCount: number }, workerDotTests: Array<{ ' $fragmentRefs'?: { 'WorkerDotTestFieldsFragment': WorkerDotTestFieldsFragment } }>, workerDotViolations: Array<{ ' $fragmentRefs'?: { 'WorkerDotViolationFieldsFragment': WorkerDotViolationFieldsFragment } }>, workerClearinghouseQueries: Array<{ ' $fragmentRefs'?: { 'ClearinghouseQueryFieldsFragment': ClearinghouseQueryFieldsFragment } }>, workerRandomSelections: Array<{ ' $fragmentRefs'?: { 'DotRandomDrawEntryFieldsFragment': DotRandomDrawEntryFieldsFragment } }> };

export type DotRandomPoolsQueryVariables = Exact<{
  input: DotRandomPoolsInput;
}>;


export type DotRandomPoolsQuery = { dotRandomPools: { totalCount: number | null, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'DotRandomPoolFieldsFragment': DotRandomPoolFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type DotRandomDrawsQueryVariables = Exact<{
  poolId?: string | number | null | undefined;
}>;


export type DotRandomDrawsQuery = { dotRandomDraws: Array<{ id: string, poolId: string, periodKey: string, periodStart: number, periodEnd: number, status: RandomDrawStatus, poolSize: number, drugTarget: number, alcoholTarget: number, drugSelected: number, alcoholSelected: number, seed: string, method: string, notes: string | null, drawnAt: number, finalizedAt: number | null, version: number, pool: { id: string, code: string, name: string, period: RandomPeriod } | null }> };

export type DotRandomDrawQueryVariables = Exact<{
  id: string | number;
}>;


export type DotRandomDrawQuery = { dotRandomDraw: { id: string, poolId: string, periodKey: string, periodStart: number, periodEnd: number, status: RandomDrawStatus, poolSize: number, drugTarget: number, alcoholTarget: number, drugSelected: number, alcoholSelected: number, seed: string, method: string, notes: string | null, drawnAt: number, finalizedAt: number | null, version: number, pool: { id: string, code: string, name: string, period: RandomPeriod, drugRatePercent: number, alcoholRatePercent: number } | null, entries: Array<{ id: string, drawId: string, workerId: string, substance: DotTestSubstance, rank: number, status: RandomEntryStatus, notifiedAt: number | null, completedAt: number | null, testId: string | null, excuseReason: string | null, version: number, worker: { id: string, firstName: string, lastName: string } | null }> } };

export type RecordDotTestMutationVariables = Exact<{
  input: RecordDotTestInput;
}>;


export type RecordDotTestMutation = { recordDotTest: { ' $fragmentRefs'?: { 'WorkerDotTestFieldsFragment': WorkerDotTestFieldsFragment } } };

export type RecordDotTestResultMutationVariables = Exact<{
  input: RecordDotTestResultInput;
}>;


export type RecordDotTestResultMutation = { recordDotTestResult: { ' $fragmentRefs'?: { 'WorkerDotTestFieldsFragment': WorkerDotTestFieldsFragment } } };

export type CancelDotTestMutationVariables = Exact<{
  id: string | number;
  reason: string;
}>;


export type CancelDotTestMutation = { cancelDotTest: { ' $fragmentRefs'?: { 'WorkerDotTestFieldsFragment': WorkerDotTestFieldsFragment } } };

export type RecordDotViolationMutationVariables = Exact<{
  input: RecordDotViolationInput;
}>;


export type RecordDotViolationMutation = { recordDotViolation: { ' $fragmentRefs'?: { 'WorkerDotViolationFieldsFragment': WorkerDotViolationFieldsFragment } } };

export type UpdateDotViolationMutationVariables = Exact<{
  input: UpdateDotViolationInput;
}>;


export type UpdateDotViolationMutation = { updateDotViolation: { ' $fragmentRefs'?: { 'WorkerDotViolationFieldsFragment': WorkerDotViolationFieldsFragment } } };

export type RecordClearinghouseQueryMutationVariables = Exact<{
  input: RecordClearinghouseQueryInput;
}>;


export type RecordClearinghouseQueryMutation = { recordClearinghouseQuery: { ' $fragmentRefs'?: { 'ClearinghouseQueryFieldsFragment': ClearinghouseQueryFieldsFragment } } };

export type CompleteClearinghouseQueryMutationVariables = Exact<{
  input: CompleteClearinghouseQueryInput;
}>;


export type CompleteClearinghouseQueryMutation = { completeClearinghouseQuery: { ' $fragmentRefs'?: { 'ClearinghouseQueryFieldsFragment': ClearinghouseQueryFieldsFragment } } };

export type CreateDotRandomPoolMutationVariables = Exact<{
  input: DotRandomPoolInput;
}>;


export type CreateDotRandomPoolMutation = { createDotRandomPool: { ' $fragmentRefs'?: { 'DotRandomPoolFieldsFragment': DotRandomPoolFieldsFragment } } };

export type UpdateDotRandomPoolMutationVariables = Exact<{
  id: string | number;
  version: number;
  input: DotRandomPoolInput;
}>;


export type UpdateDotRandomPoolMutation = { updateDotRandomPool: { ' $fragmentRefs'?: { 'DotRandomPoolFieldsFragment': DotRandomPoolFieldsFragment } } };

export type RunDotRandomDrawMutationVariables = Exact<{
  input: RunDotRandomDrawInput;
}>;


export type RunDotRandomDrawMutation = { runDotRandomDraw: { id: string, poolId: string, periodKey: string, periodStart: number, periodEnd: number, status: RandomDrawStatus, poolSize: number, drugTarget: number, alcoholTarget: number, drugSelected: number, alcoholSelected: number, seed: string, method: string, notes: string | null, drawnAt: number, finalizedAt: number | null, version: number, entries: Array<{ id: string, drawId: string, workerId: string, substance: DotTestSubstance, rank: number, status: RandomEntryStatus, notifiedAt: number | null, completedAt: number | null, testId: string | null, excuseReason: string | null, version: number, worker: { id: string, firstName: string, lastName: string } | null }> } };

export type FinalizeDotRandomDrawMutationVariables = Exact<{
  id: string | number;
}>;


export type FinalizeDotRandomDrawMutation = { finalizeDotRandomDraw: { ' $fragmentRefs'?: { 'DotRandomDrawFieldsFragment': DotRandomDrawFieldsFragment } } };

export type CancelDotRandomDrawMutationVariables = Exact<{
  id: string | number;
  reason: string;
}>;


export type CancelDotRandomDrawMutation = { cancelDotRandomDraw: { ' $fragmentRefs'?: { 'DotRandomDrawFieldsFragment': DotRandomDrawFieldsFragment } } };

export type UpdateDotRandomDrawEntryMutationVariables = Exact<{
  input: UpdateDotRandomDrawEntryInput;
}>;


export type UpdateDotRandomDrawEntryMutation = { updateDotRandomDrawEntry: { ' $fragmentRefs'?: { 'DotRandomDrawEntryFieldsFragment': DotRandomDrawEntryFieldsFragment } } };

export type WorkerEmploymentEventFieldsFragment = { id: string, workerId: string, kind: WorkerEmploymentEventKind, effectiveAt: number, reason: string | null, notes: string | null, documentId: string | null, recordedById: string | null, amendedById: string | null, amendedAt: number | null, amendmentNote: string | null, version: number, createdAt: number, updatedAt: number, fromValues: Array<{ key: string, value: string }>, toValues: Array<{ key: string, value: string }>, document: { id: string, fileName: string, originalName: string, fileType: string, fileSize: number, createdAt: number } | null, recordedBy: { id: string, name: string } | null, amendedBy: { id: string, name: string } | null } & { ' $fragmentName'?: 'WorkerEmploymentEventFieldsFragment' };

export type WorkerEmploymentEventsQueryVariables = Exact<{
  workerId: string | number;
  kinds?: Array<WorkerEmploymentEventKind> | WorkerEmploymentEventKind | null | undefined;
}>;


export type WorkerEmploymentEventsQuery = { workerEmploymentEvents: Array<{ ' $fragmentRefs'?: { 'WorkerEmploymentEventFieldsFragment': WorkerEmploymentEventFieldsFragment } }> };

export type RecordWorkerEmploymentEventMutationVariables = Exact<{
  input: RecordWorkerEmploymentEventInput;
}>;


export type RecordWorkerEmploymentEventMutation = { recordWorkerEmploymentEvent: { event: { ' $fragmentRefs'?: { 'WorkerEmploymentEventFieldsFragment': WorkerEmploymentEventFieldsFragment } }, cascade: { ptoAssignmentEnded: boolean, payAssignmentEnded: boolean, upcomingPtoCancelled: number, defaultPolicyApplied: boolean, checklistStarted: boolean, checklistsClosed: number, ptoPaidOutDays: string, ptoForfeitedDays: string, trainingAssigned: number, portalAccessRevoked: boolean, portalRevocationError: string } } };

export type AmendWorkerEmploymentEventMutationVariables = Exact<{
  input: AmendWorkerEmploymentEventInput;
}>;


export type AmendWorkerEmploymentEventMutation = { amendWorkerEmploymentEvent: { ' $fragmentRefs'?: { 'WorkerEmploymentEventFieldsFragment': WorkerEmploymentEventFieldsFragment } } };

export type WorkerInjuriesQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerInjuriesQuery = { workerInjuries: Array<{ id: string, workerId: string, caseNumber: number, caseYear: number, classification: OshaCaseClassification, illnessType: OshaIllnessType, treatment: InjuryTreatment, status: InjuryCaseStatus, recordable: boolean, occurredAt: number, reportedAt: number | null, returnedToWorkAt: number | null, location: string | null, description: string, bodyPart: string | null, harmfulAgent: string | null, daysAway: number, daysRestricted: number, privacyCase: boolean, logName: string, claimStatus: WorkersCompClaimStatus, claimNumber: string | null, claimCarrier: string | null, claimFiledAt: number | null, claimClosedAt: number | null, safetyEventId: string | null, documentId: string | null, notes: string | null, version: number }> };

export type OshaLogQueryVariables = Exact<{
  year?: number | null | undefined;
}>;


export type OshaLogQuery = { oshaLog: { year: number, postFrom: number, postThrough: number, totalRecordableIncidentRate: number | null, daysAwayRestrictedRate: number | null, totals: { deaths: number, daysAwayCases: number, jobTransferCases: number, otherRecordableCases: number, totalRecordableCases: number, totalDaysAway: number, totalDaysRestricted: number, injuryCount: number, skinDisorderCount: number, respiratoryCount: number, poisoningCount: number, hearingLossCount: number, otherIllnessCount: number, openCases: number }, summary: { id: string, year: number, status: OshaSummaryStatus, naicsCode: string | null, averageEmployees: number, totalHoursWorked: number, executiveName: string | null, executiveTitle: string | null, executivePhone: string | null, certifiedAt: number | null, postedFrom: number | null, postedThrough: number | null, submittedAt: number | null, submissionReference: string | null, notes: string | null, version: number } | null, cases: Array<{ id: string, workerId: string, caseNumber: number, caseYear: number, classification: OshaCaseClassification, illnessType: OshaIllnessType, treatment: InjuryTreatment, status: InjuryCaseStatus, recordable: boolean, occurredAt: number, reportedAt: number | null, returnedToWorkAt: number | null, logName: string, location: string | null, description: string, bodyPart: string | null, harmfulAgent: string | null, daysAway: number, daysRestricted: number, privacyCase: boolean, claimStatus: WorkersCompClaimStatus, claimNumber: string | null, claimCarrier: string | null, claimFiledAt: number | null, claimClosedAt: number | null, safetyEventId: string | null, documentId: string | null, notes: string | null, version: number, worker: { id: string, firstName: string, lastName: string, profilePicUrl: string } | null }> } };

export type OshaSummariesQueryVariables = Exact<{ [key: string]: never; }>;


export type OshaSummariesQuery = { oshaSummaries: Array<{ id: string, year: number, status: OshaSummaryStatus, certifiedAt: number | null, submittedAt: number | null, version: number }> };

export type RecordWorkerInjuryMutationVariables = Exact<{
  input: RecordWorkerInjuryInput;
}>;


export type RecordWorkerInjuryMutation = { recordWorkerInjury: { id: string, caseNumber: number, caseYear: number, classification: OshaCaseClassification, recordable: boolean, version: number } };

export type UpdateWorkerInjuryMutationVariables = Exact<{
  input: UpdateWorkerInjuryInput;
}>;


export type UpdateWorkerInjuryMutation = { updateWorkerInjury: { id: string, caseNumber: number, caseYear: number, classification: OshaCaseClassification, status: InjuryCaseStatus, recordable: boolean, version: number } };

export type DeleteWorkerInjuryMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteWorkerInjuryMutation = { deleteWorkerInjury: boolean };

export type SaveOshaSummaryMutationVariables = Exact<{
  input: SaveOshaSummaryInput;
}>;


export type SaveOshaSummaryMutation = { saveOshaSummary: { id: string, year: number, status: OshaSummaryStatus, version: number } };

export type CertifyOshaSummaryMutationVariables = Exact<{
  year: number;
}>;


export type CertifyOshaSummaryMutation = { certifyOshaSummary: { id: string, year: number, status: OshaSummaryStatus, certifiedAt: number | null, version: number } };

export type UncertifyOshaSummaryMutationVariables = Exact<{
  year: number;
}>;


export type UncertifyOshaSummaryMutation = { uncertifyOshaSummary: { id: string, year: number, status: OshaSummaryStatus, version: number } };

export type WorkerLeaveFileQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerLeaveFileQuery = { workerLeaveEntitlement: { method: LeaveMeasurementMethod, totalHours: string, usedHours: string, remainingHours: string, totalWeeks: string, usedWeeks: string, remainingWeeks: string, exhausted: boolean, eligibleOnTenure: boolean, monthsEmployed: number, openCaseCount: number, militaryCaregiver: boolean, window: { from: number, through: number } }, workerLeaveCases: Array<{ id: string, workerId: string, leaveType: WorkerLeaveType, status: LeaveCaseStatus, frequency: LeaveFrequency, reason: string | null, fmlaDesignated: boolean, militaryCaregiver: boolean, requestedAt: number, startsAt: number, endsAt: number | null, decidedAt: number | null, closedAt: number | null, certificationStatus: LeaveCertificationStatus, certificationRequestedAt: number | null, certificationDueAt: number | null, certificationReceivedAt: number | null, recertificationDueAt: number | null, certificationLate: boolean, eligibilityHoursWorked: number | null, notes: string | null, version: number, entries: Array<{ id: string, leaveCaseId: string, usedOn: number, hours: string, countsAgainstEntitlement: boolean, ptoId: string | null, notes: string | null, version: number }> }> };

export type LeaveControlQueryVariables = Exact<{ [key: string]: never; }>;


export type LeaveControlQuery = { leaveControl: { id: string, measurementMethod: LeaveMeasurementMethod, entitlementWeeks: string, militaryCaregiverWeeks: string, workweekHours: string, eligibilityMonths: number, eligibilityHours: number, certificationDueDays: number, version: number } };

export type OpenLeaveCaseMutationVariables = Exact<{
  input: OpenLeaveCaseInput;
}>;


export type OpenLeaveCaseMutation = { openLeaveCase: { id: string, status: LeaveCaseStatus, version: number } };

export type UpdateLeaveCaseMutationVariables = Exact<{
  input: UpdateLeaveCaseInput;
}>;


export type UpdateLeaveCaseMutation = { updateLeaveCase: { id: string, status: LeaveCaseStatus, version: number } };

export type DecideLeaveCaseMutationVariables = Exact<{
  input: DecideLeaveCaseInput;
}>;


export type DecideLeaveCaseMutation = { decideLeaveCase: { id: string, status: LeaveCaseStatus, fmlaDesignated: boolean, version: number } };

export type CloseLeaveCaseMutationVariables = Exact<{
  id: string | number;
}>;


export type CloseLeaveCaseMutation = { closeLeaveCase: { id: string, status: LeaveCaseStatus, version: number } };

export type RequestLeaveCertificationMutationVariables = Exact<{
  caseId: string | number;
  dueAt?: number | null | undefined;
}>;


export type RequestLeaveCertificationMutation = { requestLeaveCertification: { id: string, certificationStatus: LeaveCertificationStatus, certificationDueAt: number | null, version: number } };

export type RecordLeaveCertificationMutationVariables = Exact<{
  input: RecordLeaveCertificationInput;
}>;


export type RecordLeaveCertificationMutation = { recordLeaveCertification: { id: string, certificationStatus: LeaveCertificationStatus, version: number } };

export type RecordLeaveDayMutationVariables = Exact<{
  input: RecordLeaveDayInput;
}>;


export type RecordLeaveDayMutation = { recordLeaveDay: { id: string, usedOn: number, hours: string, countsAgainstEntitlement: boolean, version: number } };

export type DeleteLeaveDayMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteLeaveDayMutation = { deleteLeaveDay: boolean };

export type UpdateLeaveControlMutationVariables = Exact<{
  input: UpdateLeaveControlInput;
}>;


export type UpdateLeaveControlMutation = { updateLeaveControl: { id: string, measurementMethod: LeaveMeasurementMethod, version: number } };

export type WorkerOverviewQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerOverviewQuery = { workerOverview: { asOf: number, standing: WorkerStanding, nextReviewAt: number | null, concerns: Array<{ severity: WorkerConcernSeverity, code: string, headline: string, detail: string, tab: string }>, worker: { id: string, firstName: string, lastName: string, status: EntityStatus, canBeAssigned: boolean, type: WorkerType, driverType: DriverType, fleetCode: { id: string, code: string, color: string } | null, profile: { hireDate: number, terminationDate: number | null, complianceStatus: ComplianceStatus, isQualified: boolean } | null }, credentials: { complianceStatus: ComplianceStatus, requiredCount: number, validCount: number, expiringCount: number, expiredCount: number, missingCount: number } | null, training: { compliant: boolean, requiredCount: number, currentCount: number, dueCount: number, overdueCount: number, expiringCount: number, expiredCount: number, missingCount: number } | null, safety: { score: number, rating: SafetyRating, activePoints: number, pointsWatchThreshold: number, pointsAtRiskThreshold: number, accidents: number, preventableAccidents: number, citations: number, outOfServiceOrders: number, openEvents: number, activeDiscipline: number, highestDiscipline: DisciplinaryLevel | null, recognitions: number, daysSinceLastEvent: number | null } | null, checklist: { id: string, name: string, kind: WorkerChecklistKind, status: WorkerChecklistStatus, dueAt: number | null, progress: { total: number, settled: number, requiredTotal: number, requiredDone: number, overdue: number, percent: number, complete: boolean } } | null, pto: Array<{ ptoType: PtoType, tracked: boolean, balanceDays: string, pendingDays: string, availableDays: string }> | null, openReview: { id: string, title: string, status: PerformanceReviewStatus, periodEnd: number, overallScore: string | null } | null, lastReview: { id: string, title: string, status: PerformanceReviewStatus, periodEnd: number, overallScore: string | null } | null } };

export type WorkerRosterAttentionQueryVariables = Exact<{ [key: string]: never; }>;


export type WorkerRosterAttentionQuery = { workerRosterAttention: { activeWorkers: number, nonCompliant: number, trainingOverdue: number, atRisk: number, expiringSoon: number, reviewsAwaitingSignOff: number, ptoLiabilityDays: string, leaveCertificationsOutstanding: number } };

export type WorkerSafetyEventFieldsFragment = { id: string, businessUnitId: string, organizationId: string, workerId: string, kind: SafetyEventKind, severity: SafetySeverity, status: SafetyEventStatus, occurredAt: number, location: string | null, description: string, preventable: boolean, points: number, pointsExpireAt: number | null, activePoints: number, referenceNumber: string | null, shipmentId: string | null, inspectionLevel: number | null, inspectionResult: InspectionResult | null, outOfService: boolean, fineAmount: string | null, costAmount: string | null, documentId: string | null, recordedById: string | null, closedById: string | null, closedAt: number | null, resolution: string | null, version: number, createdAt: number, updatedAt: number, document: { id: string, fileName: string, originalName: string, fileType: string, fileSize: number, createdAt: number } | null, recordedBy: { id: string, name: string } | null, closedBy: { id: string, name: string } | null } & { ' $fragmentName'?: 'WorkerSafetyEventFieldsFragment' };

export type WorkerDisciplinaryActionFieldsFragment = { id: string, businessUnitId: string, organizationId: string, workerId: string, level: DisciplinaryLevel, status: DisciplinaryStatus, reason: string, details: string | null, occurredAt: number | null, issuedAt: number, expiresAt: number | null, suspensionDays: number | null, safetyEventId: string | null, documentId: string | null, issuedById: string | null, acknowledgedAt: number | null, workerComment: string | null, rescindedAt: number | null, rescindedById: string | null, rescindReason: string | null, active: boolean, version: number, createdAt: number, updatedAt: number, safetyEvent: { id: string, kind: SafetyEventKind, severity: SafetySeverity, occurredAt: number, description: string } | null, issuedBy: { id: string, name: string } | null } & { ' $fragmentName'?: 'WorkerDisciplinaryActionFieldsFragment' };

export type WorkerRecognitionFieldsFragment = { id: string, businessUnitId: string, organizationId: string, workerId: string, kind: RecognitionKind, title: string, message: string | null, occurredAt: number, awardedById: string | null, visibleToWorker: boolean, version: number, createdAt: number, updatedAt: number, awardedBy: { id: string, name: string } | null } & { ' $fragmentName'?: 'WorkerRecognitionFieldsFragment' };

export type SafetyScorecardFieldsFragment = { workerId: string, asOf: number, score: number, rating: SafetyRating, activePoints: number, pointsWatchThreshold: number, pointsAtRiskThreshold: number, accidents: number, preventableAccidents: number, incidents: number, nearMisses: number, citations: number, inspections: number, inspectionsPassed: number, inspectionsFailed: number, outOfServiceOrders: number, cleanInspectionRate: number | null, openEvents: number, activeDiscipline: number, highestDiscipline: DisciplinaryLevel | null, daysSinceLastEvent: number | null, lastEventAt: number | null, recognitions: number } & { ' $fragmentName'?: 'SafetyScorecardFieldsFragment' };

export type WorkerSafetyEventsQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerSafetyEventsQuery = { workerSafetyEvents: Array<{ ' $fragmentRefs'?: { 'WorkerSafetyEventFieldsFragment': WorkerSafetyEventFieldsFragment } }> };

export type WorkerSafetyScorecardQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerSafetyScorecardQuery = { workerSafetyScorecard: { ' $fragmentRefs'?: { 'SafetyScorecardFieldsFragment': SafetyScorecardFieldsFragment } } };

export type WorkerDisciplinaryActionsQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerDisciplinaryActionsQuery = { workerDisciplinaryActions: Array<{ ' $fragmentRefs'?: { 'WorkerDisciplinaryActionFieldsFragment': WorkerDisciplinaryActionFieldsFragment } }> };

export type WorkerDisciplinaryLadderQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerDisciplinaryLadderQuery = { workerDisciplinaryLadder: { highestLevel: DisciplinaryLevel | null, suggestedLevel: DisciplinaryLevel, atFinalStep: boolean, activeActions: Array<{ id: string, level: DisciplinaryLevel, status: DisciplinaryStatus, reason: string, issuedAt: number, expiresAt: number | null, active: boolean }> } };

export type WorkerRecognitionsQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerRecognitionsQuery = { workerRecognitions: Array<{ ' $fragmentRefs'?: { 'WorkerRecognitionFieldsFragment': WorkerRecognitionFieldsFragment } }> };

export type DefaultSafetyPointsQueryVariables = Exact<{
  kind: SafetyEventKind;
  severity: SafetySeverity;
  preventable?: boolean | null | undefined;
  inspectionResult?: InspectionResult | null | undefined;
}>;


export type DefaultSafetyPointsQuery = { defaultSafetyPoints: number };

export type CreateWorkerSafetyEventMutationVariables = Exact<{
  input: WorkerSafetyEventInput;
}>;


export type CreateWorkerSafetyEventMutation = { createWorkerSafetyEvent: { ' $fragmentRefs'?: { 'WorkerSafetyEventFieldsFragment': WorkerSafetyEventFieldsFragment } } };

export type UpdateWorkerSafetyEventMutationVariables = Exact<{
  input: UpdateWorkerSafetyEventInput;
}>;


export type UpdateWorkerSafetyEventMutation = { updateWorkerSafetyEvent: { ' $fragmentRefs'?: { 'WorkerSafetyEventFieldsFragment': WorkerSafetyEventFieldsFragment } } };

export type CloseWorkerSafetyEventMutationVariables = Exact<{
  input: SafetyEventStatusInput;
}>;


export type CloseWorkerSafetyEventMutation = { closeWorkerSafetyEvent: { ' $fragmentRefs'?: { 'WorkerSafetyEventFieldsFragment': WorkerSafetyEventFieldsFragment } } };

export type ReviewWorkerSafetyEventMutationVariables = Exact<{
  input: SafetyEventStatusInput;
}>;


export type ReviewWorkerSafetyEventMutation = { reviewWorkerSafetyEvent: { ' $fragmentRefs'?: { 'WorkerSafetyEventFieldsFragment': WorkerSafetyEventFieldsFragment } } };

export type ReopenWorkerSafetyEventMutationVariables = Exact<{
  input: SafetyEventStatusInput;
}>;


export type ReopenWorkerSafetyEventMutation = { reopenWorkerSafetyEvent: { ' $fragmentRefs'?: { 'WorkerSafetyEventFieldsFragment': WorkerSafetyEventFieldsFragment } } };

export type DeleteWorkerSafetyEventMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteWorkerSafetyEventMutation = { deleteWorkerSafetyEvent: boolean };

export type IssueDisciplinaryActionMutationVariables = Exact<{
  input: IssueDisciplinaryActionInput;
}>;


export type IssueDisciplinaryActionMutation = { issueDisciplinaryAction: { action: { ' $fragmentRefs'?: { 'WorkerDisciplinaryActionFieldsFragment': WorkerDisciplinaryActionFieldsFragment } }, employmentEvent: { id: string, kind: WorkerEmploymentEventKind, effectiveAt: number } | null } };

export type RescindDisciplinaryActionMutationVariables = Exact<{
  input: RescindDisciplinaryActionInput;
}>;


export type RescindDisciplinaryActionMutation = { rescindDisciplinaryAction: { ' $fragmentRefs'?: { 'WorkerDisciplinaryActionFieldsFragment': WorkerDisciplinaryActionFieldsFragment } } };

export type GiveWorkerRecognitionMutationVariables = Exact<{
  input: WorkerRecognitionInput;
}>;


export type GiveWorkerRecognitionMutation = { giveWorkerRecognition: { ' $fragmentRefs'?: { 'WorkerRecognitionFieldsFragment': WorkerRecognitionFieldsFragment } } };

export type DeleteWorkerRecognitionMutationVariables = Exact<{
  id: string | number;
}>;


export type DeleteWorkerRecognitionMutation = { deleteWorkerRecognition: boolean };

export type TrainingCourseFieldsFragment = { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string | null, category: TrainingCategory, status: EntityStatus, delivery: TrainingDelivery, contentUrl: string | null, durationMinutes: number, passingScore: string | null, validityMonths: number | null, renewalWindowDays: number, isRequired: boolean, requiredForDriverTypes: Array<DriverType>, dueDaysAfterAssignment: number, requiresAcknowledgement: boolean, sortOrder: number, openRecordCount: number, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'TrainingCourseFieldsFragment' };

export type WorkerTrainingRecordFieldsFragment = { id: string, businessUnitId: string, organizationId: string, workerId: string, courseId: string, status: WorkerTrainingStatus, assignedAt: number, dueAt: number | null, startedAt: number | null, completedAt: number | null, expiresAt: number | null, score: string | null, passed: boolean | null, acknowledgedAt: number | null, documentId: string | null, assignedById: string | null, recordedById: string | null, notes: string | null, waivedReason: string | null, health: WorkerTrainingHealth, daysUntilDue: number | null, daysUntilExpiry: number | null, version: number, createdAt: number, updatedAt: number, course: { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string | null, category: TrainingCategory, status: EntityStatus, delivery: TrainingDelivery, contentUrl: string | null, durationMinutes: number, passingScore: string | null, validityMonths: number | null, renewalWindowDays: number, isRequired: boolean, requiredForDriverTypes: Array<DriverType>, dueDaysAfterAssignment: number, requiresAcknowledgement: boolean, sortOrder: number, openRecordCount: number, version: number, createdAt: number, updatedAt: number } | null, document: { id: string, fileName: string, originalName: string, fileType: string, fileSize: number, createdAt: number } | null, assignedBy: { id: string, name: string } | null, recordedBy: { id: string, name: string } | null } & { ' $fragmentName'?: 'WorkerTrainingRecordFieldsFragment' };

export type TrainingCourseTableQueryVariables = Exact<{
  input: TrainingCoursesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type TrainingCourseTableQuery = { trainingCourses: { totalCount?: number | null, edges: Array<{ cursor: string, node: { ' $fragmentRefs'?: { 'TrainingCourseFieldsFragment': TrainingCourseFieldsFragment } } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type ActiveTrainingCoursesQueryVariables = Exact<{ [key: string]: never; }>;


export type ActiveTrainingCoursesQuery = { activeTrainingCourses: Array<{ ' $fragmentRefs'?: { 'TrainingCourseFieldsFragment': TrainingCourseFieldsFragment } }> };

export type WorkerTrainingRecordsQueryVariables = Exact<{
  workerId: string | number;
  includeClosed?: boolean | null | undefined;
}>;


export type WorkerTrainingRecordsQuery = { workerTrainingRecords: Array<{ ' $fragmentRefs'?: { 'WorkerTrainingRecordFieldsFragment': WorkerTrainingRecordFieldsFragment } }> };

export type WorkerTrainingSummaryQueryVariables = Exact<{
  workerId: string | number;
}>;


export type WorkerTrainingSummaryQuery = { workerTrainingSummary: { workerId: string, compliant: boolean, requiredCount: number, currentCount: number, dueCount: number, overdueCount: number, expiringCount: number, expiredCount: number, missingCount: number, items: Array<{ health: WorkerTrainingHealth, daysUntilDue: number | null, daysUntilExpiry: number | null, required: boolean, course: { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string | null, category: TrainingCategory, status: EntityStatus, delivery: TrainingDelivery, contentUrl: string | null, durationMinutes: number, passingScore: string | null, validityMonths: number | null, renewalWindowDays: number, isRequired: boolean, requiredForDriverTypes: Array<DriverType>, dueDaysAfterAssignment: number, requiresAcknowledgement: boolean, sortOrder: number, openRecordCount: number, version: number, createdAt: number, updatedAt: number }, record: { id: string, businessUnitId: string, organizationId: string, workerId: string, courseId: string, status: WorkerTrainingStatus, assignedAt: number, dueAt: number | null, startedAt: number | null, completedAt: number | null, expiresAt: number | null, score: string | null, passed: boolean | null, acknowledgedAt: number | null, documentId: string | null, assignedById: string | null, recordedById: string | null, notes: string | null, waivedReason: string | null, health: WorkerTrainingHealth, daysUntilDue: number | null, daysUntilExpiry: number | null, version: number, createdAt: number, updatedAt: number, course: { id: string, businessUnitId: string, organizationId: string, code: string, name: string, description: string | null, category: TrainingCategory, status: EntityStatus, delivery: TrainingDelivery, contentUrl: string | null, durationMinutes: number, passingScore: string | null, validityMonths: number | null, renewalWindowDays: number, isRequired: boolean, requiredForDriverTypes: Array<DriverType>, dueDaysAfterAssignment: number, requiresAcknowledgement: boolean, sortOrder: number, openRecordCount: number, version: number, createdAt: number, updatedAt: number } | null, document: { id: string, fileName: string, originalName: string, fileType: string, fileSize: number, createdAt: number } | null, assignedBy: { id: string, name: string } | null, recordedBy: { id: string, name: string } | null } | null }> } };

export type CreateTrainingCourseMutationVariables = Exact<{
  input: TrainingCourseInput;
}>;


export type CreateTrainingCourseMutation = { createTrainingCourse: { ' $fragmentRefs'?: { 'TrainingCourseFieldsFragment': TrainingCourseFieldsFragment } } };

export type UpdateTrainingCourseMutationVariables = Exact<{
  id: string | number;
  input: TrainingCourseInput;
}>;


export type UpdateTrainingCourseMutation = { updateTrainingCourse: { ' $fragmentRefs'?: { 'TrainingCourseFieldsFragment': TrainingCourseFieldsFragment } } };

export type ArchiveTrainingCourseMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type ArchiveTrainingCourseMutation = { archiveTrainingCourse: { ' $fragmentRefs'?: { 'TrainingCourseFieldsFragment': TrainingCourseFieldsFragment } } };

export type RestoreTrainingCourseMutationVariables = Exact<{
  id: string | number;
  version?: number | null | undefined;
}>;


export type RestoreTrainingCourseMutation = { restoreTrainingCourse: { ' $fragmentRefs'?: { 'TrainingCourseFieldsFragment': TrainingCourseFieldsFragment } } };

export type AssignWorkerTrainingMutationVariables = Exact<{
  input: AssignWorkerTrainingInput;
}>;


export type AssignWorkerTrainingMutation = { assignWorkerTraining: { ' $fragmentRefs'?: { 'WorkerTrainingRecordFieldsFragment': WorkerTrainingRecordFieldsFragment } } };

export type BulkAssignTrainingMutationVariables = Exact<{
  input: BulkAssignTrainingInput;
}>;


export type BulkAssignTrainingMutation = { bulkAssignTraining: { assignedCount: number, skippedCount: number, failedCount: number, outcomes: Array<{ workerId: string, courseId: string, recordId: string | null, skipped: boolean, error: string }> } };

export type AssignRequiredWorkerTrainingMutationVariables = Exact<{
  workerId: string | number;
}>;


export type AssignRequiredWorkerTrainingMutation = { assignRequiredWorkerTraining: Array<{ ' $fragmentRefs'?: { 'WorkerTrainingRecordFieldsFragment': WorkerTrainingRecordFieldsFragment } }> };

export type CompleteWorkerTrainingMutationVariables = Exact<{
  input: CompleteWorkerTrainingInput;
}>;


export type CompleteWorkerTrainingMutation = { completeWorkerTraining: { ' $fragmentRefs'?: { 'WorkerTrainingRecordFieldsFragment': WorkerTrainingRecordFieldsFragment } } };

export type WaiveWorkerTrainingMutationVariables = Exact<{
  input: WaiveWorkerTrainingInput;
}>;


export type WaiveWorkerTrainingMutation = { waiveWorkerTraining: { ' $fragmentRefs'?: { 'WorkerTrainingRecordFieldsFragment': WorkerTrainingRecordFieldsFragment } } };

export type CancelWorkerTrainingMutationVariables = Exact<{
  input: CancelWorkerTrainingInput;
}>;


export type CancelWorkerTrainingMutation = { cancelWorkerTraining: { ' $fragmentRefs'?: { 'WorkerTrainingRecordFieldsFragment': WorkerTrainingRecordFieldsFragment } } };

export type AttachWorkerTrainingDocumentMutationVariables = Exact<{
  input: AttachWorkerTrainingDocumentInput;
}>;


export type AttachWorkerTrainingDocumentMutation = { attachWorkerTrainingDocument: { ' $fragmentRefs'?: { 'WorkerTrainingRecordFieldsFragment': WorkerTrainingRecordFieldsFragment } } };

export type WorkerFleetCodeFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'WorkerFleetCodeFieldsFragment' };

export type WorkerUsStateFieldsFragment = { id: string, name: string, abbreviation: string } & { ' $fragmentName'?: 'WorkerUsStateFieldsFragment' };

export type WorkerProfileTableFieldsFragment = { id: string, workerId: string, businessUnitId: string, organizationId: string, licenseStateId: string | null, dob: number, licenseNumber: string, cdlClass: CdlClass, cdlRestrictions: string, endorsement: EndorsementType, hazmatExpiry: number | null, licenseExpiry: number, medicalCardExpiry: number | null, medicalExaminerName: string, medicalExaminerNpi: string, twicCardNumber: string, twicExpiry: number | null, hireDate: number, terminationDate: number | null, physicalDueDate: number | null, mvrDueDate: number | null, complianceStatus: ComplianceStatus, trainingHealth: WorkerTrainingHealth, safetyRating: SafetyRating, safetyScore: number, nextCredentialExpiry: number | null, nextTrainingDue: number | null, drugAlcoholStatus: DrugAlcoholStatus, isQualified: boolean, disqualificationReason: string, lastComplianceCheck: number, lastMvrCheck: number, lastDrugTest: number, eldExempt: boolean, shortHaulExempt: boolean, version: number, createdAt: number, updatedAt: number, licenseState: { ' $fragmentRefs'?: { 'WorkerUsStateFieldsFragment': WorkerUsStateFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerProfileTableFieldsFragment' };

export type WorkerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, stateId: string, fleetCodeId: string | null, managerId: string | null, status: EntityStatus, type: WorkerType, driverType: DriverType, leaveType: WorkerLeaveType | null, profilePicUrl: string, firstName: string, lastName: string, wholeName: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, email: string, phoneNumber: string, emergencyContactName: string, emergencyContactPhone: string, externalId: string, assignmentBlocked: string, gender: WorkerGender, canBeAssigned: boolean, availableForDispatch: boolean, version: number, createdAt: number, updatedAt: number, customFields: unknown, fleetCode: { ' $fragmentRefs'?: { 'WorkerFleetCodeFieldsFragment': WorkerFleetCodeFieldsFragment } } | null, state: { ' $fragmentRefs'?: { 'WorkerUsStateFieldsFragment': WorkerUsStateFieldsFragment } } | null, profile: { ' $fragmentRefs'?: { 'WorkerProfileTableFieldsFragment': WorkerProfileTableFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerTableRowFieldsFragment' };

export type WorkerPtoWorkerFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string, profilePicUrl: string } & { ' $fragmentName'?: 'WorkerPtoWorkerFieldsFragment' };

export type WorkerPtoActorFieldsFragment = { id: string, name: string } & { ' $fragmentName'?: 'WorkerPtoActorFieldsFragment' };

export type WorkerPtoRowFieldsFragment = { id: string, workerId: string, organizationId: string, businessUnitId: string, approverId: string | null, rejectorId: string | null, cancelledById: string | null, status: PtoStatus, type: PtoType, startDate: number, endDate: number, reason: string, rejectionReason: string | null, cancellationReason: string | null, days: string, balanceAfterDays: string | null, autoApproved: boolean, version: number, createdAt: number, updatedAt: number, worker: { ' $fragmentRefs'?: { 'WorkerPtoWorkerFieldsFragment': WorkerPtoWorkerFieldsFragment } } | null, approver: { ' $fragmentRefs'?: { 'WorkerPtoActorFieldsFragment': WorkerPtoActorFieldsFragment } } | null, rejector: { ' $fragmentRefs'?: { 'WorkerPtoActorFieldsFragment': WorkerPtoActorFieldsFragment } } | null, cancelledBy: { ' $fragmentRefs'?: { 'WorkerPtoActorFieldsFragment': WorkerPtoActorFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerPtoRowFieldsFragment' };

export type WorkerDataTablePageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'WorkerDataTablePageInfoFieldsFragment' };

export type WorkerTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type WorkerTableQuery = { workers: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerTableRowFieldsFragment': WorkerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'WorkerDataTablePageInfoFieldsFragment': WorkerDataTablePageInfoFieldsFragment } } } };

export type WorkerPtoTableQueryVariables = Exact<{
  input: WorkerPtoEntriesInput;
  includeTotalCount?: boolean | null | undefined;
}>;


export type WorkerPtoTableQuery = { workerPTOEntries: { totalCount?: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'WorkerDataTablePageInfoFieldsFragment': WorkerDataTablePageInfoFieldsFragment } } } };

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

export type CreateWorkerPtoMutationVariables = Exact<{
  input: CreateWorkerPtoInput;
}>;


export type CreateWorkerPtoMutation = { createWorkerPTO: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } };

export type UpdateWorkerPtoMutationVariables = Exact<{
  input: UpdateWorkerPtoInput;
}>;


export type UpdateWorkerPtoMutation = { updateWorkerPTO: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } };

export type ApproveWorkerPtoMutationVariables = Exact<{
  id: string | number;
}>;


export type ApproveWorkerPtoMutation = { approveWorkerPTO: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } };

export type RejectWorkerPtoMutationVariables = Exact<{
  id: string | number;
  reason: string;
}>;


export type RejectWorkerPtoMutation = { rejectWorkerPTO: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } };

export type CancelWorkerPtoMutationVariables = Exact<{
  id: string | number;
  reason?: string | null | undefined;
}>;


export type CancelWorkerPtoMutation = { cancelWorkerPTO: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } };

export type BulkWorkerPtoActionMutationVariables = Exact<{
  input: BulkWorkerPtoActionInput;
}>;


export type BulkWorkerPtoActionMutation = { bulkWorkerPTOAction: { successCount: number, failureCount: number, results: Array<{ ptoId: string, success: boolean, error: string }> } };

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
export const AgentControlFieldsFragmentDoc = new TypedDocumentString(`
    fragment AgentControlFields on AgentControl {
  id
  organizationId
  businessUnitId
  shadowMode
  billingAgentEnabled
  decisionTimeoutSeconds
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"AgentControlFields"}) as unknown as TypedDocumentString<AgentControlFieldsFragment, unknown>;
export const AgentExceptionTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment AgentExceptionTableRowFields on AgentException {
  id
  organizationId
  businessUnitId
  runId
  category
  severity
  subjectType
  subjectId
  attemptSummary
  blastRadius
  resolutionState
  resolutionNotes
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"AgentExceptionTableRowFields"}) as unknown as TypedDocumentString<AgentExceptionTableRowFieldsFragment, unknown>;
export const AgentEvidenceRefFieldsFragmentDoc = new TypedDocumentString(`
    fragment AgentEvidenceRefFields on AgentEvidenceRef {
  type
  id
  note
}
    `, {"fragmentName":"AgentEvidenceRefFields"}) as unknown as TypedDocumentString<AgentEvidenceRefFieldsFragment, unknown>;
export const AgentExceptionDetailFieldsFragmentDoc = new TypedDocumentString(`
    fragment AgentExceptionDetailFields on AgentException {
  ...AgentExceptionTableRowFields
  evidence {
    ...AgentEvidenceRefFields
  }
}
    fragment AgentExceptionTableRowFields on AgentException {
  id
  organizationId
  businessUnitId
  runId
  category
  severity
  subjectType
  subjectId
  attemptSummary
  blastRadius
  resolutionState
  resolutionNotes
  version
  createdAt
  updatedAt
}
fragment AgentEvidenceRefFields on AgentEvidenceRef {
  type
  id
  note
}`, {"fragmentName":"AgentExceptionDetailFields"}) as unknown as TypedDocumentString<AgentExceptionDetailFieldsFragment, unknown>;
export const AgentProposalTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment AgentProposalTableRowFields on AgentProposal {
  id
  organizationId
  businessUnitId
  runId
  toolName
  toolParams
  confidence
  rationale
  autonomyTier
  status
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"AgentProposalTableRowFields"}) as unknown as TypedDocumentString<AgentProposalTableRowFieldsFragment, unknown>;
export const AgentProposalDetailFieldsFragmentDoc = new TypedDocumentString(`
    fragment AgentProposalDetailFields on AgentProposal {
  ...AgentProposalTableRowFields
  evidence {
    ...AgentEvidenceRefFields
  }
}
    fragment AgentEvidenceRefFields on AgentEvidenceRef {
  type
  id
  note
}
fragment AgentProposalTableRowFields on AgentProposal {
  id
  organizationId
  businessUnitId
  runId
  toolName
  toolParams
  confidence
  rationale
  autonomyTier
  status
  version
  createdAt
  updatedAt
}`, {"fragmentName":"AgentProposalDetailFields"}) as unknown as TypedDocumentString<AgentProposalDetailFieldsFragment, unknown>;
export const AgentRunTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment AgentRunTableRowFields on AgentRun {
  id
  organizationId
  businessUnitId
  agentType
  subjectType
  subjectId
  status
  workflowId
  modelIdentifier
  promptVersion
  startedAt
  completedAt
  errorMessage
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"AgentRunTableRowFields"}) as unknown as TypedDocumentString<AgentRunTableRowFieldsFragment, unknown>;
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
export const CarrierContactFieldsFragmentDoc = new TypedDocumentString(`
    fragment CarrierContactFields on CarrierContact {
  id
  businessUnitId
  organizationId
  carrierId
  name
  title
  email
  phone
  isPrimary
  receivesRateConfirmations
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"CarrierContactFields"}) as unknown as TypedDocumentString<CarrierContactFieldsFragment, unknown>;
export const CarrierInsurancePolicyFieldsFragmentDoc = new TypedDocumentString(`
    fragment CarrierInsurancePolicyFields on CarrierInsurancePolicy {
  id
  businessUnitId
  organizationId
  carrierId
  policyType
  policyNumber
  providerName
  coverageAmount
  effectiveDate
  expirationDate
  isVerified
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"CarrierInsurancePolicyFields"}) as unknown as TypedDocumentString<CarrierInsurancePolicyFieldsFragment, unknown>;
export const CarrierTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment CarrierTableRowFields on Carrier {
  id
  businessUnitId
  organizationId
  stateId
  remitStateId
  status
  code
  name
  dbaName
  carrierType
  dotNumber
  mcNumber
  scac
  complianceStatus
  safetyRating
  qualifiedAt
  disqualifiedReason
  taxId
  taxIdType
  w9OnFile
  is1099Eligible
  paymentMethod
  paymentTermDays
  remitToName
  remitAddressLine1
  remitAddressLine2
  remitCity
  remitPostalCode
  addressLine1
  addressLine2
  city
  postalCode
  phone
  email
  externalId
  notes
  version
  createdAt
  updatedAt
  contacts {
    ...CarrierContactFields
  }
  insurancePolicies {
    ...CarrierInsurancePolicyFields
  }
}
    fragment CarrierContactFields on CarrierContact {
  id
  businessUnitId
  organizationId
  carrierId
  name
  title
  email
  phone
  isPrimary
  receivesRateConfirmations
  version
  createdAt
  updatedAt
}
fragment CarrierInsurancePolicyFields on CarrierInsurancePolicy {
  id
  businessUnitId
  organizationId
  carrierId
  policyType
  policyNumber
  providerName
  coverageAmount
  effectiveDate
  expirationDate
  isVerified
  version
  createdAt
  updatedAt
}`, {"fragmentName":"CarrierTableRowFields"}) as unknown as TypedDocumentString<CarrierTableRowFieldsFragment, unknown>;
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
  options {
    value
    label
    color
    description
  }
  validationRules {
    minLength
    maxLength
    min
    max
    pattern
  }
  defaultValue
  uiAttributes {
    placeholder
    helpText
    width
  }
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
  invoiceDelivery
  billingCycle
  billingCycleAnchorDay
  billingCycleTimezone
  lastBilledPeriodEnd
  paymentTerm
  hasBillingControlOverrides
  creditLimit
  creditBalance
  creditStatus
  enforceCreditLimit
  autoCreditHold
  creditHoldReason
  autoSendInvoiceOnGeneration
  splitBy
  sectionBy
  invoiceDetail
  minConsolidatedAmount
  maxShipmentsPerInvoice
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
  autoApprove
  autoBill
  countLateOnlyOnAppointmentStops
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
  invoiceDelivery
  billingCycle
  billingCycleAnchorDay
  billingCycleTimezone
  lastBilledPeriodEnd
  paymentTerm
  hasBillingControlOverrides
  creditLimit
  creditBalance
  creditStatus
  enforceCreditLimit
  autoCreditHold
  creditHoldReason
  autoSendInvoiceOnGeneration
  splitBy
  sectionBy
  invoiceDetail
  minConsolidatedAmount
  maxShipmentsPerInvoice
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
  autoApprove
  autoBill
  countLateOnlyOnAppointmentStops
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
export const DetentionOccurrenceFieldsFragmentDoc = new TypedDocumentString(`
    fragment DetentionOccurrenceFields on DetentionOccurrence {
  id
  businessUnitId
  organizationId
  shipmentId
  shipmentMoveId
  stopId
  customerId
  locationId
  detentionPolicyId
  policySnapshot
  calculationTrace
  stopType
  scheduleType
  appointmentStart
  appointmentEnd
  arrivedAt
  departedAt
  clockStartAt
  clockStopAt
  freeTimeExpiresAt
  noticeDueAt
  noticeDeadlineAt
  isOpen
  arrivedLate
  lateByMinutes
  freeMinutesGranted
  rawDwellMinutes
  billableMinutes
  roundedMinutes
  billableUnits
  grossAmount
  billableAmount
  driverPayMinutes
  driverPayAmount
  netMargin
  capApplied
  convertedToLayover
  currency
  status
  notificationStatus
  noticeSentAt
  suppressedByGate
  requiresApproval
  waiverReason
  waiverNote
  waivedAt
  waivedAmount
  disputeNote
  disputedAt
  collectabilityScore
  evidenceHead
  additionalChargeId
  locationName
  customerName
  shipmentProNumber
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"DetentionOccurrenceFields"}) as unknown as TypedDocumentString<DetentionOccurrenceFieldsFragment, unknown>;
export const DetentionEvidenceFieldsFragmentDoc = new TypedDocumentString(`
    fragment DetentionEvidenceFields on DetentionEvidence {
  id
  detentionOccurrenceId
  sequence
  kind
  source
  summary
  observedAt
  recordedAt
  recordedById
  documentId
  payload
  prevHash
  hash
  createdAt
}
    `, {"fragmentName":"DetentionEvidenceFields"}) as unknown as TypedDocumentString<DetentionEvidenceFieldsFragment, unknown>;
export const DetentionNoticeFieldsFragmentDoc = new TypedDocumentString(`
    fragment DetentionNoticeFields on DetentionNotice {
  id
  detentionOccurrenceId
  threadKey
  kind
  channel
  deliveryStatus
  recipients
  subject
  body
  scheduledFor
  sentAt
  deliveredAt
  openedAt
  failedAt
  failureReason
  sentById
  wasAutomatic
  satisfiesRequirement
  quotedFreeMinutes
  quotedRate
  quotedAmount
  createdAt
}
    `, {"fragmentName":"DetentionNoticeFields"}) as unknown as TypedDocumentString<DetentionNoticeFieldsFragment, unknown>;
export const DetentionCollectabilityFieldsFragmentDoc = new TypedDocumentString(`
    fragment DetentionCollectabilityFields on DetentionCollectability {
  score
  band
  chainValid
  summary
  factors {
    key
    label
    earned
    possible
    detail
    remedy
  }
}
    `, {"fragmentName":"DetentionCollectabilityFields"}) as unknown as TypedDocumentString<DetentionCollectabilityFieldsFragment, unknown>;
export const DetentionPolicyTierFieldsFragmentDoc = new TypedDocumentString(`
    fragment DetentionPolicyTierFields on DetentionPolicyTier {
  id
  fromMinute
  toMinute
  rate
  rateUnit
  label
  sortOrder
}
    `, {"fragmentName":"DetentionPolicyTierFields"}) as unknown as TypedDocumentString<DetentionPolicyTierFieldsFragment, unknown>;
export const DetentionPolicyRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment DetentionPolicyRowFields on DetentionPolicy {
  id
  businessUnitId
  organizationId
  name
  code
  description
  status
  isOrgDefault
  priority
  specificityScore
  customerId
  locationId
  shipmentTypeIds
  serviceTypeIds
  commodityIds
  stopTypes
  appointmentStopsOnly
  effectiveStartDate
  effectiveEndDate
  clockStartBasis
  lateArrivalRule
  lateArrivalGraceMinutes
  billingFreeMinutes
  pickupFreeMinutes
  deliveryFreeMinutes
  payFreeMinutes
  minimumBillableMinutes
  billingIncrementMinutes
  roundingMode
  rateSource
  accessorialChargeId
  tiers {
    ...DetentionPolicyTierFields
  }
  maxBillableMinutesPerStop
  maxChargePerStop
  maxChargePerDay
  maxChargePerShipment
  dayBoundaryMode
  convertToLayoverAtMinutes
  layoverAccessorialChargeId
  notificationRequirement
  notificationLeadMinutes
  notificationDeadlineMinutes
  unnotifiedBehavior
  autoSendNotice
  attachNoticePdf
  sendDepartureSummary
  requireApprovalOverAmount
  autoApproveUnderAmount
  currency
  comments
  version
  createdAt
  updatedAt
}
    fragment DetentionPolicyTierFields on DetentionPolicyTier {
  id
  fromMinute
  toMinute
  rate
  rateUnit
  label
  sortOrder
}`, {"fragmentName":"DetentionPolicyRowFields"}) as unknown as TypedDocumentString<DetentionPolicyRowFieldsFragment, unknown>;
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
  highwayOnly
  tollRoads
  bordersOpen
  includeTollData
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
export const PortalTrainingFieldsFragmentDoc = new TypedDocumentString(`
    fragment PortalTrainingFields on PortalTraining {
  id
  courseId
  name
  description
  category
  delivery
  contentUrl
  durationMinutes
  status
  health
  required
  requiresAcknowledgement
  scored
  dueAt
  daysUntilDue
  startedAt
  completedAt
  expiresAt
  daysUntilExpiry
  acknowledgedAt
  score
}
    `, {"fragmentName":"PortalTrainingFields"}) as unknown as TypedDocumentString<PortalTrainingFieldsFragment, unknown>;
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
  config
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
  externalId
  lastKnownLocationId
  lastKnownLocationName
  fuelType
  iftaQualified
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
  externalId
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
  variableDefinitions {
    name
    type
    description
    required
    defaultValue
    source
  }
  breakdownDefinitions {
    name
    label
    expression
  }
  minCharge
  maxCharge
  roundingMode
  roundingPrecision
  sourceTemplateId
  sourceVersionNumber
  version
  currentVersionNumber
  approvedAt
  usageCount
  scenarioCount
  createdAt
  updatedAt
}
    `, {"fragmentName":"FormulaTemplateTableRowFields"}) as unknown as TypedDocumentString<FormulaTemplateTableRowFieldsFragment, unknown>;
export const FuelCardFieldsFragmentDoc = new TypedDocumentString(`
    fragment FuelCardFields on FuelCard {
  id
  businessUnitId
  organizationId
  provider
  lastFour
  label
  externalCardId
  assignedWorkerId
  assignedTractorId
  status
  expiresAt
  cancelledAt
  cancelReason
  notes
  discoveredAt
  version
  createdAt
  updatedAt
  assignedWorker {
    id
    wholeName
    firstName
    lastName
  }
  assignedTractor {
    id
    code
  }
}
    `, {"fragmentName":"FuelCardFields"}) as unknown as TypedDocumentString<FuelCardFieldsFragment, unknown>;
export const FuelPurchaseImportBatchFieldsFragmentDoc = new TypedDocumentString(`
    fragment FuelPurchaseImportBatchFields on FuelPurchaseImportBatch {
  id
  businessUnitId
  organizationId
  provider
  origin
  feedReference
  documentId
  fileName
  sourceFormat
  status
  defaultFuelType
  defaultFuelCardId
  defaultCurrency
  mapping
  unmappedHeaders
  summary {
    rowCount
    newCount
    duplicateInFileCount
    alreadyImportedCount
    errorCount
    totalGallons
    totalAmount
    byFuelType
    byJurisdiction
    earliestPurchasedAt
    latestPurchasedAt
  }
  rowCount
  errorCount
  committedCount
  error
  uploadedById
  stagedAt
  committedAt
  committedById
  version
  createdAt
  updatedAt
  document {
    id
    fileName
    originalName
    fileType
    fileSize
    createdAt
  }
  defaultFuelCard {
    id
    provider
    lastFour
    label
  }
}
    `, {"fragmentName":"FuelPurchaseImportBatchFields"}) as unknown as TypedDocumentString<FuelPurchaseImportBatchFieldsFragment, unknown>;
export const FuelPurchaseImportRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment FuelPurchaseImportRowFields on FuelPurchaseImportRow {
  id
  importBatchId
  rowNumber
  cells
  parsed {
    purchasedAt
    vendor
    vendorCity
    jurisdictionCode
    fuelType
    quantity
    quantityUnit
    gallons
    unitPrice
    totalAmount
    currencyCode
    transactionReference
    cardLastFour
    tractorCode
    odometer
  }
  transactionReference
  status
  error
  resolvedTractorId
  resolvedFuelCardId
  resolvedJurisdictionId
  resolutionNotes
  fuelPurchaseId
  createdAt
  resolvedTractor {
    id
    code
  }
}
    `, {"fragmentName":"FuelPurchaseImportRowFields"}) as unknown as TypedDocumentString<FuelPurchaseImportRowFieldsFragment, unknown>;
export const FuelPurchaseFieldsFragmentDoc = new TypedDocumentString(`
    fragment FuelPurchaseFields on FuelPurchase {
  id
  businessUnitId
  organizationId
  tractorId
  workerId
  jurisdictionId
  fuelCardId
  cardLastFour
  purchasedAt
  vendor
  vendorCity
  fuelType
  quantity
  quantityUnit
  gallons
  unitPrice
  totalAmount
  currencyCode
  odometer
  transactionReference
  source
  importBatchId
  taxPaid
  notes
  createdById
  version
  createdAt
  updatedAt
  tractor {
    id
    code
  }
  worker {
    id
    wholeName
    firstName
    lastName
  }
  jurisdiction {
    id
    countryCode
    code
    name
  }
  fuelCard {
    id
    provider
    lastFour
    label
  }
}
    `, {"fragmentName":"FuelPurchaseFields"}) as unknown as TypedDocumentString<FuelPurchaseFieldsFragment, unknown>;
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
export const HomeWidgetFieldsFragmentDoc = new TypedDocumentString(`
    fragment HomeWidgetFields on HomeWidget {
  id
  key
  title
  w
  h
  config {
    metric
    metrics
    definitionId
    cannedKey
    chartId
    columnId
    dashboardId
    text
    limit
    windowDays
  }
}
    `, {"fragmentName":"HomeWidgetFields"}) as unknown as TypedDocumentString<HomeWidgetFieldsFragment, unknown>;
export const HomeLayoutFieldsFragmentDoc = new TypedDocumentString(`
    fragment HomeLayoutFields on HomeLayout {
  schemaVersion
  version
  source
  presetId
  presetName
  locked
  canCustomize
  density
  widgets {
    ...HomeWidgetFields
  }
}
    fragment HomeWidgetFields on HomeWidget {
  id
  key
  title
  w
  h
  config {
    metric
    metrics
    definitionId
    cannedKey
    chartId
    columnId
    dashboardId
    text
    limit
    windowDays
  }
}`, {"fragmentName":"HomeLayoutFields"}) as unknown as TypedDocumentString<HomeLayoutFieldsFragment, unknown>;
export const HomeLayoutPresetFieldsFragmentDoc = new TypedDocumentString(`
    fragment HomeLayoutPresetFields on HomeLayoutPreset {
  id
  name
  description
  roleIds
  coreResponsibility
  isOrgDefault
  locked
  priority
  assignedUserCount
  version
  createdAt
  updatedAt
  widgets {
    ...HomeWidgetFields
  }
}
    fragment HomeWidgetFields on HomeWidget {
  id
  key
  title
  w
  h
  config {
    metric
    metrics
    definitionId
    cannedKey
    chartId
    columnId
    dashboardId
    text
    limit
    windowDays
  }
}`, {"fragmentName":"HomeLayoutPresetFields"}) as unknown as TypedDocumentString<HomeLayoutPresetFieldsFragment, unknown>;
export const IftaMileageEntryFieldsFragmentDoc = new TypedDocumentString(`
    fragment IftaMileageEntryFields on IFTAJurisdictionMileageEntry {
  id
  businessUnitId
  organizationId
  tractorId
  jurisdictionId
  traveledAt
  year
  quarter
  miles
  loaded
  source
  shipmentMoveId
  notes
  createdById
  version
  createdAt
  updatedAt
  tractor {
    id
    code
  }
  jurisdiction {
    id
    countryCode
    code
    name
  }
}
    `, {"fragmentName":"IftaMileageEntryFields"}) as unknown as TypedDocumentString<IftaMileageEntryFieldsFragment, unknown>;
export const IftaJurisdictionFieldsFragmentDoc = new TypedDocumentString(`
    fragment IftaJurisdictionFields on IFTAJurisdiction {
  id
  countryCode
  code
  name
  usStateId
  isIftaMember
  hasSurcharge
  sortOrder
  status
}
    `, {"fragmentName":"IftaJurisdictionFields"}) as unknown as TypedDocumentString<IftaJurisdictionFieldsFragment, unknown>;
export const IftaPeriodFieldsFragmentDoc = new TypedDocumentString(`
    fragment IftaPeriodFields on IFTAPeriod {
  year
  quarter
  key
  label
  start
  end
  dueDate
}
    `, {"fragmentName":"IftaPeriodFields"}) as unknown as TypedDocumentString<IftaPeriodFieldsFragment, unknown>;
export const IftaReturnLineFieldsFragmentDoc = new TypedDocumentString(`
    fragment IftaReturnLineFields on IFTAReturnLine {
  id
  returnId
  jurisdictionId
  fuelType
  isIftaMember
  totalMiles
  taxableMiles
  routeMiles
  manualMiles
  loadedMiles
  emptyMiles
  taxPaidGallons
  taxPaidGallonsRaw
  purchaseCount
  taxableGallons
  netTaxableGallons
  ratePerGallon
  surchargeRatePerGallon
  rateMissing
  taxDue
  surchargeDue
  lineTotal
  sortOrder
  jurisdiction {
    id
    countryCode
    code
    name
    isIftaMember
    hasSurcharge
  }
}
    `, {"fragmentName":"IftaReturnLineFields"}) as unknown as TypedDocumentString<IftaReturnLineFieldsFragment, unknown>;
export const IftaReturnFieldsFragmentDoc = new TypedDocumentString(`
    fragment IftaReturnFields on IFTAReturn {
  id
  businessUnitId
  organizationId
  year
  quarter
  period {
    ...IftaPeriodFields
  }
  amendmentNumber
  amendsReturnId
  amendsReturn {
    id
    amendmentNumber
    status
  }
  status
  timezone
  periodStart
  periodEnd
  totalMiles
  totalTaxableMiles
  totalGallons
  totalTaxPaidGallons
  netTaxableGallons
  taxDue
  surchargeDue
  netDue
  currencyCode
  fleetMpgByFuelType {
    fuelType
    mpg
    totalMiles
    totalGallons
  }
  unattributedMiles
  unattributedMoveCount
  noTractorMiles
  noTractorMoveCount
  problems {
    code
    message
    jurisdictionCode
    fuelType
    amount
  }
  computedAt
  finalizedAt
  finalizedById
  finalizedBy {
    id
    name
  }
  filedAt
  filedById
  filedBy {
    id
    name
  }
  filingReference
  reopenedAt
  reopenedById
  reopenReason
  lines {
    ...IftaReturnLineFields
  }
  version
  createdAt
  updatedAt
}
    fragment IftaPeriodFields on IFTAPeriod {
  year
  quarter
  key
  label
  start
  end
  dueDate
}
fragment IftaReturnLineFields on IFTAReturnLine {
  id
  returnId
  jurisdictionId
  fuelType
  isIftaMember
  totalMiles
  taxableMiles
  routeMiles
  manualMiles
  loadedMiles
  emptyMiles
  taxPaidGallons
  taxPaidGallonsRaw
  purchaseCount
  taxableGallons
  netTaxableGallons
  ratePerGallon
  surchargeRatePerGallon
  rateMissing
  taxDue
  surchargeDue
  lineTotal
  sortOrder
  jurisdiction {
    id
    countryCode
    code
    name
    isIftaMember
    hasSurcharge
  }
}`, {"fragmentName":"IftaReturnFields"}) as unknown as TypedDocumentString<IftaReturnFieldsFragment, unknown>;
export const IftaReturnSummaryFieldsFragmentDoc = new TypedDocumentString(`
    fragment IftaReturnSummaryFields on IFTAReturn {
  id
  year
  quarter
  amendmentNumber
  status
  totalMiles
  totalTaxPaidGallons
  netDue
  currencyCode
  computedAt
  finalizedAt
  filedAt
  filingReference
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"IftaReturnSummaryFields"}) as unknown as TypedDocumentString<IftaReturnSummaryFieldsFragment, unknown>;
export const IftaTaxRateFieldsFragmentDoc = new TypedDocumentString(`
    fragment IftaTaxRateFields on IFTATaxRate {
  id
  jurisdictionId
  year
  quarter
  fuelType
  ratePerGallon
  surchargeRatePerGallon
  sourceNote
  sourceUrl
  version
  createdAt
  updatedAt
  jurisdiction {
    id
    countryCode
    code
    name
    hasSurcharge
    isIftaMember
  }
}
    `, {"fragmentName":"IftaTaxRateFields"}) as unknown as TypedDocumentString<IftaTaxRateFieldsFragment, unknown>;
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
export const JurisdictionRuleOverrideTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment JurisdictionRuleOverrideTableRowFields on JurisdictionRuleOverride {
  id
  businessUnitId
  organizationId
  stateId
  maxWidthFeet
  maxHeightFeet
  maxLengthFeet
  maxWeightPounds
  permitLeadTimeDays
  daylightOnly
  holidayRestricted
  reason
  version
  createdAt
  updatedAt
  state {
    id
    name
    abbreviation
  }
}
    `, {"fragmentName":"JurisdictionRuleOverrideTableRowFields"}) as unknown as TypedDocumentString<JurisdictionRuleOverrideTableRowFieldsFragment, unknown>;
export const JurisdictionRuleTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment JurisdictionRuleTableRowFields on JurisdictionRule {
  id
  stateId
  status
  maxWidthFeet
  maxHeightFeet
  maxLengthFeet
  maxWeightPounds
  superloadWidthFeet
  superloadWeightPounds
  daylightOnly
  rushHourRestricted
  weekendRestricted
  holidayRestricted
  permitLeadTimeDays
  permitValidityDays
  permitBaseFee
  permitPerMileFee
  sourceNote
  sourceUrl
  verificationState
  verifiedAt
  effectiveStartDate
  effectiveEndDate
  version
  createdAt
  updatedAt
  state {
    id
    name
    abbreviation
  }
}
    `, {"fragmentName":"JurisdictionRuleTableRowFields"}) as unknown as TypedDocumentString<JurisdictionRuleTableRowFieldsFragment, unknown>;
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
  isGeocoded
  latitude
  longitude
  placeId
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
export const OrgHolidayFieldsFragmentDoc = new TypedDocumentString(`
    fragment OrgHolidayFields on OrgHoliday {
  id
  businessUnitId
  organizationId
  name
  holidayDate
  kind
  recursAnnually
  description
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"OrgHolidayFields"}) as unknown as TypedDocumentString<OrgHolidayFieldsFragment, unknown>;
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
  brokerageEnabled
  assetOperationsEnabled
  state {
    ...OrganizationSettingsStateFields
  }
}
    fragment OrganizationSettingsStateFields on UsState {
  id
  name
  abbreviation
}`, {"fragmentName":"OrganizationSettingsFields"}) as unknown as TypedDocumentString<OrganizationSettingsFieldsFragment, unknown>;
export const PerformanceReviewTemplateFieldsFragmentDoc = new TypedDocumentString(`
    fragment PerformanceReviewTemplateFields on PerformanceReviewTemplate {
  id
  businessUnitId
  organizationId
  code
  name
  description
  status
  isDefault
  cadenceMonths
  items {
    key
    label
    description
    weight
  }
  openReviewCount
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"PerformanceReviewTemplateFields"}) as unknown as TypedDocumentString<PerformanceReviewTemplateFieldsFragment, unknown>;
export const PerformanceReviewFieldsFragmentDoc = new TypedDocumentString(`
    fragment PerformanceReviewFields on PerformanceReview {
  id
  businessUnitId
  organizationId
  workerId
  templateId
  template {
    id
    code
    name
    cadenceMonths
    items {
      key
      label
      description
      weight
    }
  }
  reviewerId
  reviewer {
    id
    name
  }
  status
  title
  periodStart
  periodEnd
  ratings {
    key
    label
    weight
    score
    comment
  }
  overallScore
  summary
  strengths
  improvements
  goals {
    id
    title
    dueAt
    status
  }
  submittedAt
  acknowledgedAt
  workerComment
  closedAt
  closedById
  nextReviewAt
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"PerformanceReviewFields"}) as unknown as TypedDocumentString<PerformanceReviewFieldsFragment, unknown>;
export const PtoPolicyRuleFieldsFragmentDoc = new TypedDocumentString(`
    fragment PtoPolicyRuleFields on PTOPolicyRule {
  id
  ptoPolicyId
  ptoType
  accrualMethod
  accrualAmountDays
  maxBalanceDays
  carryoverCapDays
  carryoverExpiryDays
  tiers {
    minMonths
    accrualAmountDays
    maxBalanceDays
  }
  onTermination
  sortOrder
}
    `, {"fragmentName":"PtoPolicyRuleFields"}) as unknown as TypedDocumentString<PtoPolicyRuleFieldsFragment, unknown>;
export const PtoPolicyFieldsFragmentDoc = new TypedDocumentString(`
    fragment PtoPolicyFields on PTOPolicy {
  id
  businessUnitId
  organizationId
  name
  code
  description
  status
  isDefault
  yearBasis
  countWeekends
  waitingPeriodDays
  requiresApproval
  enforceBalance
  allowNegative
  negativeFloorDays
  openAssignmentCount
  version
  createdAt
  updatedAt
  rules {
    id
    ptoPolicyId
    ptoType
    accrualMethod
    accrualAmountDays
    maxBalanceDays
    carryoverCapDays
    carryoverExpiryDays
    tiers {
      minMonths
      accrualAmountDays
      maxBalanceDays
    }
    onTermination
    sortOrder
  }
}
    `, {"fragmentName":"PtoPolicyFields"}) as unknown as TypedDocumentString<PtoPolicyFieldsFragment, unknown>;
export const PtoPolicyAssignmentFieldsFragmentDoc = new TypedDocumentString(`
    fragment PtoPolicyAssignmentFields on WorkerPTOPolicyAssignment {
  id
  workerId
  ptoPolicyId
  effectiveFrom
  effectiveTo
  assignedById
  note
  version
  createdAt
  updatedAt
  ptoPolicy {
    id
    name
    code
    status
    countWeekends
    requiresApproval
    enforceBalance
  }
}
    `, {"fragmentName":"PtoPolicyAssignmentFields"}) as unknown as TypedDocumentString<PtoPolicyAssignmentFieldsFragment, unknown>;
export const PlannedPtoAccrualFieldsFragmentDoc = new TypedDocumentString(`
    fragment PlannedPtoAccrualFields on PlannedPTOAccrual {
  entryType
  periodKey
  effectiveAt
  nominalDays
  deferred
}
    `, {"fragmentName":"PlannedPtoAccrualFields"}) as unknown as TypedDocumentString<PlannedPtoAccrualFieldsFragment, unknown>;
export const WorkerPtoBalanceFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerPtoBalanceFields on WorkerPTOBalance {
  ptoType
  tracked
  enforced
  balanceDays
  pendingDays
  availableDays
  accruedYtdDays
  usedYtdDays
  carriedDays
  maxBalanceDays
  nextAccrual {
    entryType
    periodKey
    effectiveAt
    nominalDays
    deferred
  }
}
    `, {"fragmentName":"WorkerPtoBalanceFields"}) as unknown as TypedDocumentString<WorkerPtoBalanceFieldsFragment, unknown>;
export const WorkerPtoLedgerEntryFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerPtoLedgerEntryFields on WorkerPTOLedgerEntry {
  id
  workerId
  ptoType
  entryType
  amountDays
  balanceAfterDays
  sequence
  effectiveAt
  periodKey
  sourcePtoId
  assignmentId
  ptoPolicyId
  actorType
  createdById
  note
  createdAt
}
    `, {"fragmentName":"WorkerPtoLedgerEntryFields"}) as unknown as TypedDocumentString<WorkerPtoLedgerEntryFieldsFragment, unknown>;
export const RateAgreementRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment RateAgreementRowFields on RateAgreement {
  id
  businessUnitId
  organizationId
  partyType
  customerId
  carrierId
  code
  name
  description
  agreementType
  status
  contractRef
  priority
  effectiveFrom
  effectiveTo
  autoRenew
  renewalNoticeDays
  currency
  defaultMinCharge
  defaultMaxCharge
  marginFloorPercent
  maxPayPercentOfSell
  submittedById
  submittedAt
  approvedById
  approvedAt
  reviewComment
  currentVersionNumber
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"RateAgreementRowFields"}) as unknown as TypedDocumentString<RateAgreementRowFieldsFragment, unknown>;
export const RateZoneRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment RateZoneRowFields on RateZone {
  id
  businessUnitId
  organizationId
  code
  name
  description
  status
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"RateZoneRowFields"}) as unknown as TypedDocumentString<RateZoneRowFieldsFragment, unknown>;
export const RateMatrixRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment RateMatrixRowFields on RateMatrix {
  id
  businessUnitId
  organizationId
  code
  name
  description
  status
  formulaTemplateId
  formulaTemplateName
  currency
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"RateMatrixRowFields"}) as unknown as TypedDocumentString<RateMatrixRowFieldsFragment, unknown>;
export const RateQuoteRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment RateQuoteRowFields on RateQuote {
  id
  businessUnitId
  organizationId
  shipmentId
  partyType
  partyId
  purpose
  outcome
  rateAgreementId
  rateAgreementRuleId
  formulaTemplateId
  specificityScore
  currency
  billingCurrency
  linehaulAmount
  totalAmount
  billingAmount
  foregoneAmount
  overrideReason
  asOf
  ratedAt
  ratedById
  engineVersion
  createdAt
}
    `, {"fragmentName":"RateQuoteRowFields"}) as unknown as TypedDocumentString<RateQuoteRowFieldsFragment, unknown>;
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
export const ReportDashboardFieldsFragmentDoc = new TypedDocumentString(`
    fragment ReportDashboardFields on ReportDashboard {
  id
  name
  description
  category
  tags
  ownerId
  visibility
  layout
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ReportDashboardFields"}) as unknown as TypedDocumentString<ReportDashboardFieldsFragment, unknown>;
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
export const ReportDefinitionOptionFieldsFragmentDoc = new TypedDocumentString(`
    fragment ReportDefinitionOptionFields on ReportDefinition {
  id
  name
  description
  category
  kind
  status
  visibility
  lastRunAt
  updatedAt
}
    `, {"fragmentName":"ReportDefinitionOptionFields"}) as unknown as TypedDocumentString<ReportDefinitionOptionFieldsFragment, unknown>;
export const ReportPreviewFieldsFragmentDoc = new TypedDocumentString(`
    fragment ReportPreviewFields on ReportPreview {
  columns {
    id
    label
    type
    format
    display {
      style
      decimals
      grouping
      currency
      negative
      notation
      prefix
      suffix
      dateStyle
      boolStyle
      durationUnit
      durationStyle
      nullText
      rules {
        op
        value
        upper
        tone
      }
      band {
        width
        edges
      }
    }
  }
  rows
  totals
  truncated
}
    `, {"fragmentName":"ReportPreviewFields"}) as unknown as TypedDocumentString<ReportPreviewFieldsFragment, unknown>;
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
  emailInline
  notifyUserIds
  alert {
    operator
    threshold
    columnId
    value
    suppressWhileFiring
  }
  alertFiring
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
export const ReportViewFieldsFragmentDoc = new TypedDocumentString(`
    fragment ReportViewFields on ReportView {
  id
  definitionId
  ownerId
  name
  description
  params
  shared
  pinned
  format
  lastRunAt
  runCount
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ReportViewFields"}) as unknown as TypedDocumentString<ReportViewFieldsFragment, unknown>;
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
export const RoutingGuideEntryFieldsFragmentDoc = new TypedDocumentString(`
    fragment RoutingGuideEntryFields on RoutingGuideEntry {
  id
  routingGuideId
  carrierId
  rank
  rateMethod
  rate
  useContractRate
  offerTtlSeconds
  channel
  version
  createdAt
  updatedAt
  carrier {
    id
    name
    scac
  }
}
    `, {"fragmentName":"RoutingGuideEntryFields"}) as unknown as TypedDocumentString<RoutingGuideEntryFieldsFragment, unknown>;
export const RoutingGuideRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment RoutingGuideRowFields on RoutingGuide {
  id
  businessUnitId
  organizationId
  name
  description
  status
  originLocationId
  destinationLocationId
  originCity
  originState
  destinationCity
  destinationState
  specificity
  version
  createdAt
  updatedAt
  entries {
    ...RoutingGuideEntryFields
  }
}
    fragment RoutingGuideEntryFields on RoutingGuideEntry {
  id
  routingGuideId
  carrierId
  rank
  rateMethod
  rate
  useContractRate
  offerTtlSeconds
  channel
  version
  createdAt
  updatedAt
  carrier {
    id
    name
    scac
  }
}`, {"fragmentName":"RoutingGuideRowFields"}) as unknown as TypedDocumentString<RoutingGuideRowFieldsFragment, unknown>;
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
  shipmentMoveId
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
  internalNotes
  x12StatusCodeOverride
  x12ReasonCodeOverride
  x12ExceptionCode
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
  versionNumber
  breakdown {
    name
    label
    amount
    error
  }
  guardrail {
    applied
    bound
    rawResult
    minCharge
    maxCharge
  }
  rateQuoteId
  agreementId
  agreementName
  ruleId
  ruleLabel
  source
  explanation
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
export const ShipmentCarrierAssignmentFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentCarrierAssignmentFields on CarrierAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  carrierId
  status
  rateMethod
  baseRate
  baseAmount
  fuelSurcharge
  accessorialTotal
  totalCost
  currencyCode
  proNumber
  externalDriverName
  externalDriverPhone
  externalTractorNumber
  externalTrailerNumber
  confirmedAt
  canceledAt
  cancellationReason
  version
  createdAt
  updatedAt
  carrier {
    id
    code
    name
    scac
  }
  accessorials {
    id
    carrierAssignmentId
    accessorialChargeId
    description
    amount
    version
  }
}
    `, {"fragmentName":"ShipmentCarrierAssignmentFields"}) as unknown as TypedDocumentString<ShipmentCarrierAssignmentFieldsFragment, unknown>;
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
  coverageType
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
  carrierAssignment {
    ...ShipmentCarrierAssignmentFields
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
}
fragment ShipmentCarrierAssignmentFields on CarrierAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  carrierId
  status
  rateMethod
  baseRate
  baseAmount
  fuelSurcharge
  accessorialTotal
  totalCost
  currencyCode
  proNumber
  externalDriverName
  externalDriverPhone
  externalTractorNumber
  externalTrailerNumber
  confirmedAt
  canceledAt
  cancellationReason
  version
  createdAt
  updatedAt
  carrier {
    id
    code
    name
    scac
  }
  accessorials {
    id
    carrierAssignmentId
    accessorialChargeId
    description
    amount
    version
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
  detentionOccurrenceId
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
  lengthFeet
  widthFeet
  heightFeet
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
  autoRated
  autoRatedAt
  rateAgreementId
  rateAgreementRuleId
  rateQuoteId
  rateOverrideAmount
  rateOverrideReason
  rateOverrideAt
  rateLocked
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
    ediPartner {
      id
      name
      code
    }
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
  coverageType
  stops {
    ...ShipmentStopFields
  }
  assignment {
    ...ShipmentAssignmentFields
  }
  carrierAssignment {
    ...ShipmentCarrierAssignmentFields
  }
}
fragment ShipmentCarrierAssignmentFields on CarrierAssignment {
  id
  businessUnitId
  organizationId
  shipmentMoveId
  carrierId
  status
  rateMethod
  baseRate
  baseAmount
  fuelSurcharge
  accessorialTotal
  totalCost
  currencyCode
  proNumber
  externalDriverName
  externalDriverPhone
  externalTractorNumber
  externalTrailerNumber
  confirmedAt
  canceledAt
  cancellationReason
  version
  createdAt
  updatedAt
  carrier {
    id
    code
    name
    scac
  }
  accessorials {
    id
    carrierAssignmentId
    accessorialChargeId
    description
    amount
    version
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
  detentionOccurrenceId
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
  lengthFeet
  widthFeet
  heightFeet
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
  versionNumber
  breakdown {
    name
    label
    amount
    error
  }
  guardrail {
    applied
    bound
    rawResult
    minCharge
    maxCharge
  }
  rateQuoteId
  agreementId
  agreementName
  ruleId
  ruleLabel
  source
  explanation
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
export const ShipmentCommentAcknowledgmentFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentCommentAcknowledgmentFields on ShipmentCommentAcknowledgment {
  id
  commentId
  userId
  acknowledgedAt
  user {
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
}`, {"fragmentName":"ShipmentCommentAcknowledgmentFields"}) as unknown as TypedDocumentString<ShipmentCommentAcknowledgmentFieldsFragment, unknown>;
export const ShipmentCommentAttachmentFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentCommentAttachmentFields on ShipmentCommentAttachment {
  documentId
  fileName
  originalName
  fileSize
  mimeType
  previewUrl
  downloadUrl
  createdAt
}
    `, {"fragmentName":"ShipmentCommentAttachmentFields"}) as unknown as TypedDocumentString<ShipmentCommentAttachmentFieldsFragment, unknown>;
export const ShipmentCommentFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentCommentFields on ShipmentComment {
  id
  businessUnitId
  organizationId
  shipmentId
  userId
  parentCommentId
  replyCount
  comment
  body
  type
  visibility
  priority
  source
  metadata
  editedAt
  pinnedAt
  pinnedById
  resolvedAt
  resolvedById
  requiresAcknowledgment
  deletedAt
  version
  createdAt
  updatedAt
  mentionedUserIds
  user {
    ...ShipmentUserFields
  }
  pinnedBy {
    ...ShipmentUserFields
  }
  resolvedBy {
    ...ShipmentUserFields
  }
  mentionedUsers {
    ...ShipmentCommentMentionFields
  }
  acknowledgments {
    ...ShipmentCommentAcknowledgmentFields
  }
  attachments {
    ...ShipmentCommentAttachmentFields
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
fragment ShipmentCommentAttachmentFields on ShipmentCommentAttachment {
  documentId
  fileName
  originalName
  fileSize
  mimeType
  previewUrl
  downloadUrl
  createdAt
}
fragment ShipmentCommentAcknowledgmentFields on ShipmentCommentAcknowledgment {
  id
  commentId
  userId
  acknowledgedAt
  user {
    ...ShipmentUserFields
  }
}`, {"fragmentName":"ShipmentCommentFields"}) as unknown as TypedDocumentString<ShipmentCommentFieldsFragment, unknown>;
export const ShipmentEventFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentEventFields on ShipmentEvent {
  __typename
  id
  organizationId
  businessUnitId
  shipmentId
  type
  severity
  actorType
  actorId
  actorLabel
  summary
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
  ... on ShipmentLifecycleEvent {
    proNumber
    previousStatus
    newStatus
    reason
  }
  ... on ShipmentOwnershipEvent {
    proNumber
    previousOwnerId
    newOwnerId
  }
  ... on ShipmentMoveEvent {
    moveId
    stopId
    previousStatus
    newStatus
  }
  ... on ShipmentAssignmentEvent {
    moveId
    assignmentId
    primaryWorkerId
    secondaryWorkerId
    tractorId
    trailerId
    driverName
  }
  ... on ShipmentCarrierEvent {
    moveId
    carrierId
    carrierName
    totalCost
    reason
    proNumber
  }
  ... on ShipmentTenderEvent {
    tenderId
    offerId
    moveId
    carrierName
    rank
    channel
    source
    reason
    action
    mode
    error
    reasons
    warnings
  }
  ... on ShipmentHoldEvent {
    holdId
    holdType
    holdSeverity
    holdSource
  }
  ... on ShipmentCommentEvent {
    commentId
    commentBody
    commentType
    commentVisibility
    commentPriority
    mentionedUserIds
  }
}
    `, {"fragmentName":"ShipmentEventFields"}) as unknown as TypedDocumentString<ShipmentEventFieldsFragment, unknown>;
export const ShipmentContractRateFieldsFragmentDoc = new TypedDocumentString(`
    fragment ShipmentContractRateFields on ShipmentContractRate {
  applied
  outcome
  agreementId
  agreementName
  ruleId
  ruleLabel
  formulaTemplateId
  formulaTemplateName
  baseRate
  linehaulAmount
  otherChargeAmount
  totalChargeAmount
  previousLinehaulAmount
  explanation
  accessorials {
    accessorialChargeId
    description
    method
    amount
    unit
  }
}
    `, {"fragmentName":"ShipmentContractRateFields"}) as unknown as TypedDocumentString<ShipmentContractRateFieldsFragment, unknown>;
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
export const WorkerChecklistTemplateItemFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerChecklistTemplateItemFields on WorkerChecklistTemplateItem {
  id
  templateId
  label
  description
  kind
  required
  dueOffsetDays
  owner
  credentialTypeId
  documentTypeId
  documentTypeName
  sortOrder
  credentialType {
    id
    code
    name
  }
}
    `, {"fragmentName":"WorkerChecklistTemplateItemFields"}) as unknown as TypedDocumentString<WorkerChecklistTemplateItemFieldsFragment, unknown>;
export const WorkerChecklistTemplateFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerChecklistTemplateFields on WorkerChecklistTemplate {
  id
  businessUnitId
  organizationId
  code
  name
  description
  kind
  trigger
  status
  isDefault
  openChecklistCount
  version
  createdAt
  updatedAt
  items {
    id
    templateId
    label
    description
    kind
    required
    dueOffsetDays
    owner
    credentialTypeId
    documentTypeId
    documentTypeName
    sortOrder
    credentialType {
      id
      code
      name
    }
  }
}
    `, {"fragmentName":"WorkerChecklistTemplateFields"}) as unknown as TypedDocumentString<WorkerChecklistTemplateFieldsFragment, unknown>;
export const WorkerChecklistItemFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerChecklistItemFields on WorkerChecklistItem {
  id
  checklistId
  label
  description
  kind
  required
  owner
  dueAt
  overdue
  credentialTypeId
  documentTypeId
  status
  completedById
  completedAt
  autoCompleted
  note
  evidenceDocumentId
  evidenceCredentialId
  sortOrder
  version
  completedBy {
    id
    name
  }
  evidenceDocument {
    id
    originalName
    fileType
  }
  evidenceCredential {
    id
    number
    expiresAt
  }
  credentialType {
    id
    code
    name
  }
}
    `, {"fragmentName":"WorkerChecklistItemFields"}) as unknown as TypedDocumentString<WorkerChecklistItemFieldsFragment, unknown>;
export const WorkerChecklistFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerChecklistFields on WorkerChecklist {
  id
  workerId
  templateId
  name
  kind
  status
  startedAt
  dueAt
  completedAt
  cancelledAt
  cancelReason
  sourceEventId
  startedById
  version
  createdAt
  updatedAt
  startedBy {
    id
    name
  }
  progress {
    total
    settled
    requiredTotal
    requiredDone
    overdue
    percent
    complete
  }
  items {
    id
    checklistId
    label
    description
    kind
    required
    owner
    dueAt
    overdue
    credentialTypeId
    documentTypeId
    status
    completedById
    completedAt
    autoCompleted
    note
    evidenceDocumentId
    evidenceCredentialId
    sortOrder
    version
    completedBy {
      id
      name
    }
    evidenceDocument {
      id
      originalName
      fileType
    }
    evidenceCredential {
      id
      number
      expiresAt
    }
    credentialType {
      id
      code
      name
    }
  }
}
    `, {"fragmentName":"WorkerChecklistFields"}) as unknown as TypedDocumentString<WorkerChecklistFieldsFragment, unknown>;
export const WorkerCredentialTypeFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerCredentialTypeFields on WorkerCredentialType {
  id
  businessUnitId
  organizationId
  code
  name
  description
  category
  status
  isRequired
  requiredForDriverTypes
  renewalWindowDays
  validityMonths
  requiresNumber
  requiresDocument
  profileField
  isSystem
  sortOrder
  activeCredentialCount
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"WorkerCredentialTypeFields"}) as unknown as TypedDocumentString<WorkerCredentialTypeFieldsFragment, unknown>;
export const WorkerCredentialFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerCredentialFields on WorkerCredential {
  id
  workerId
  credentialTypeId
  status
  number
  issuingAuthority
  issuedAt
  expiresAt
  documentId
  notes
  verifiedById
  verifiedAt
  archivedById
  archivedAt
  archiveReason
  health
  daysUntilExpiry
  version
  createdAt
  updatedAt
  credentialType {
    id
    code
    name
    category
    isRequired
    renewalWindowDays
    validityMonths
    requiresNumber
    requiresDocument
    profileField
    isSystem
    sortOrder
  }
  document {
    id
    fileName
    originalName
    fileType
    fileSize
    createdAt
  }
  verifiedBy {
    id
    name
  }
}
    `, {"fragmentName":"WorkerCredentialFields"}) as unknown as TypedDocumentString<WorkerCredentialFieldsFragment, unknown>;
export const WorkerDotTestFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerDotTestFields on WorkerDOTTest {
  id
  workerId
  testType
  substance
  status
  result
  isDot
  reason
  scheduledAt
  collectedAt
  resultAt
  collectionSite
  collectorName
  specimenId
  labName
  mroName
  mroVerifiedAt
  alcoholConcentration
  safetyEventId
  drawEntryId
  documentId
  notes
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"WorkerDotTestFields"}) as unknown as TypedDocumentString<WorkerDotTestFieldsFragment, unknown>;
export const WorkerDotViolationFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerDotViolationFields on WorkerDOTViolation {
  id
  workerId
  violationType
  status
  occurredAt
  sourceTestId
  reportedToClearinghouseAt
  sapName
  sapReferredAt
  sapEvaluationCompletedAt
  rtdTestId
  rtdCompletedAt
  followUpTestCount
  followUpTestsCompleted
  followUpEndsAt
  resolvedAt
  documentId
  notes
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"WorkerDotViolationFields"}) as unknown as TypedDocumentString<WorkerDotViolationFieldsFragment, unknown>;
export const ClearinghouseQueryFieldsFragmentDoc = new TypedDocumentString(`
    fragment ClearinghouseQueryFields on WorkerClearinghouseQuery {
  id
  workerId
  queryType
  result
  consentObtainedAt
  consentExpiresAt
  requestedAt
  completedAt
  violationCount
  reference
  documentId
  notes
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"ClearinghouseQueryFields"}) as unknown as TypedDocumentString<ClearinghouseQueryFieldsFragment, unknown>;
export const DotRandomPoolFieldsFragmentDoc = new TypedDocumentString(`
    fragment DotRandomPoolFields on DOTRandomPool {
  id
  code
  name
  description
  status
  period
  drugRatePercent
  alcoholRatePercent
  includedDriverTypes
  isDefault
  meetsDotMinimums
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"DotRandomPoolFields"}) as unknown as TypedDocumentString<DotRandomPoolFieldsFragment, unknown>;
export const DotRandomDrawEntryFieldsFragmentDoc = new TypedDocumentString(`
    fragment DotRandomDrawEntryFields on DOTRandomDrawEntry {
  id
  drawId
  workerId
  substance
  rank
  status
  notifiedAt
  completedAt
  testId
  excuseReason
  version
  worker {
    id
    firstName
    lastName
  }
}
    `, {"fragmentName":"DotRandomDrawEntryFields"}) as unknown as TypedDocumentString<DotRandomDrawEntryFieldsFragment, unknown>;
export const DotRandomDrawFieldsFragmentDoc = new TypedDocumentString(`
    fragment DotRandomDrawFields on DOTRandomDraw {
  id
  poolId
  periodKey
  periodStart
  periodEnd
  status
  poolSize
  drugTarget
  alcoholTarget
  drugSelected
  alcoholSelected
  seed
  method
  notes
  drawnAt
  finalizedAt
  version
}
    `, {"fragmentName":"DotRandomDrawFields"}) as unknown as TypedDocumentString<DotRandomDrawFieldsFragment, unknown>;
export const WorkerEmploymentEventFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerEmploymentEventFields on WorkerEmploymentEvent {
  id
  workerId
  kind
  effectiveAt
  reason
  notes
  fromValues {
    key
    value
  }
  toValues {
    key
    value
  }
  documentId
  document {
    id
    fileName
    originalName
    fileType
    fileSize
    createdAt
  }
  recordedById
  recordedBy {
    id
    name
  }
  amendedById
  amendedBy {
    id
    name
  }
  amendedAt
  amendmentNote
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"WorkerEmploymentEventFields"}) as unknown as TypedDocumentString<WorkerEmploymentEventFieldsFragment, unknown>;
export const WorkerSafetyEventFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerSafetyEventFields on WorkerSafetyEvent {
  id
  businessUnitId
  organizationId
  workerId
  kind
  severity
  status
  occurredAt
  location
  description
  preventable
  points
  pointsExpireAt
  activePoints
  referenceNumber
  shipmentId
  inspectionLevel
  inspectionResult
  outOfService
  fineAmount
  costAmount
  documentId
  document {
    id
    fileName
    originalName
    fileType
    fileSize
    createdAt
  }
  recordedById
  recordedBy {
    id
    name
  }
  closedById
  closedBy {
    id
    name
  }
  closedAt
  resolution
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"WorkerSafetyEventFields"}) as unknown as TypedDocumentString<WorkerSafetyEventFieldsFragment, unknown>;
export const WorkerDisciplinaryActionFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerDisciplinaryActionFields on WorkerDisciplinaryAction {
  id
  businessUnitId
  organizationId
  workerId
  level
  status
  reason
  details
  occurredAt
  issuedAt
  expiresAt
  suspensionDays
  safetyEventId
  safetyEvent {
    id
    kind
    severity
    occurredAt
    description
  }
  documentId
  issuedById
  issuedBy {
    id
    name
  }
  acknowledgedAt
  workerComment
  rescindedAt
  rescindedById
  rescindReason
  active
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"WorkerDisciplinaryActionFields"}) as unknown as TypedDocumentString<WorkerDisciplinaryActionFieldsFragment, unknown>;
export const WorkerRecognitionFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerRecognitionFields on WorkerRecognition {
  id
  businessUnitId
  organizationId
  workerId
  kind
  title
  message
  occurredAt
  awardedById
  awardedBy {
    id
    name
  }
  visibleToWorker
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"WorkerRecognitionFields"}) as unknown as TypedDocumentString<WorkerRecognitionFieldsFragment, unknown>;
export const SafetyScorecardFieldsFragmentDoc = new TypedDocumentString(`
    fragment SafetyScorecardFields on SafetyScorecard {
  workerId
  asOf
  score
  rating
  activePoints
  pointsWatchThreshold
  pointsAtRiskThreshold
  accidents
  preventableAccidents
  incidents
  nearMisses
  citations
  inspections
  inspectionsPassed
  inspectionsFailed
  outOfServiceOrders
  cleanInspectionRate
  openEvents
  activeDiscipline
  highestDiscipline
  daysSinceLastEvent
  lastEventAt
  recognitions
}
    `, {"fragmentName":"SafetyScorecardFields"}) as unknown as TypedDocumentString<SafetyScorecardFieldsFragment, unknown>;
export const TrainingCourseFieldsFragmentDoc = new TypedDocumentString(`
    fragment TrainingCourseFields on TrainingCourse {
  id
  businessUnitId
  organizationId
  code
  name
  description
  category
  status
  delivery
  contentUrl
  durationMinutes
  passingScore
  validityMonths
  renewalWindowDays
  isRequired
  requiredForDriverTypes
  dueDaysAfterAssignment
  requiresAcknowledgement
  sortOrder
  openRecordCount
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"TrainingCourseFields"}) as unknown as TypedDocumentString<TrainingCourseFieldsFragment, unknown>;
export const WorkerTrainingRecordFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerTrainingRecordFields on WorkerTrainingRecord {
  id
  businessUnitId
  organizationId
  workerId
  courseId
  course {
    id
    businessUnitId
    organizationId
    code
    name
    description
    category
    status
    delivery
    contentUrl
    durationMinutes
    passingScore
    validityMonths
    renewalWindowDays
    isRequired
    requiredForDriverTypes
    dueDaysAfterAssignment
    requiresAcknowledgement
    sortOrder
    openRecordCount
    version
    createdAt
    updatedAt
  }
  status
  assignedAt
  dueAt
  startedAt
  completedAt
  expiresAt
  score
  passed
  acknowledgedAt
  documentId
  document {
    id
    fileName
    originalName
    fileType
    fileSize
    createdAt
  }
  assignedById
  assignedBy {
    id
    name
  }
  recordedById
  recordedBy {
    id
    name
  }
  notes
  waivedReason
  health
  daysUntilDue
  daysUntilExpiry
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"WorkerTrainingRecordFields"}) as unknown as TypedDocumentString<WorkerTrainingRecordFieldsFragment, unknown>;
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
  trainingHealth
  safetyRating
  safetyScore
  nextCredentialExpiry
  nextTrainingDue
  drugAlcoholStatus
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
  leaveType
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
  trainingHealth
  safetyRating
  safetyScore
  nextCredentialExpiry
  nextTrainingDue
  drugAlcoholStatus
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
export const WorkerPtoActorFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerPtoActorFields on User {
  id
  name
}
    `, {"fragmentName":"WorkerPtoActorFields"}) as unknown as TypedDocumentString<WorkerPtoActorFieldsFragment, unknown>;
export const WorkerPtoRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerPtoRowFields on WorkerPTO {
  id
  workerId
  organizationId
  businessUnitId
  approverId
  rejectorId
  cancelledById
  status
  type
  startDate
  endDate
  reason
  rejectionReason
  cancellationReason
  days
  balanceAfterDays
  autoApproved
  version
  createdAt
  updatedAt
  worker {
    ...WorkerPtoWorkerFields
  }
  approver {
    ...WorkerPtoActorFields
  }
  rejector {
    ...WorkerPtoActorFields
  }
  cancelledBy {
    ...WorkerPtoActorFields
  }
}
    fragment WorkerPtoWorkerFields on Worker {
  id
  firstName
  lastName
  wholeName
  profilePicUrl
}
fragment WorkerPtoActorFields on User {
  id
  name
}`, {"fragmentName":"WorkerPtoRowFields"}) as unknown as TypedDocumentString<WorkerPtoRowFieldsFragment, unknown>;
export const WorkerDataTablePageInfoFieldsFragmentDoc = new TypedDocumentString(`
    fragment WorkerDataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}
    `, {"fragmentName":"WorkerDataTablePageInfoFields"}) as unknown as TypedDocumentString<WorkerDataTablePageInfoFieldsFragment, unknown>;
export const AccessorialChargeTableDocument = {"__meta__":{"kind":"query","name":"AccessorialChargeTable","hash":"sha256:428bf0351875289ecd242e7153b17c69f386b3edb5106b4bc369b85341769d91"}} as unknown as TypedDocumentString<AccessorialChargeTableQuery, AccessorialChargeTableQueryVariables>;
export const AccountTypeTableDocument = {"__meta__":{"kind":"query","name":"AccountTypeTable","hash":"sha256:bd52997a38905cd2b8343527e55ae1488b2e2f87bc1e50192a1909481e075956"}} as unknown as TypedDocumentString<AccountTypeTableQuery, AccountTypeTableQueryVariables>;
export const ArAgingSummaryDocument = {"__meta__":{"kind":"query","name":"ArAgingSummary","hash":"sha256:6e0cbca355dfb7e59c403efe34be542aee4f26ff779d480e08e077574626daf9"}} as unknown as TypedDocumentString<ArAgingSummaryQuery, ArAgingSummaryQueryVariables>;
export const ArOpenItemsDocument = {"__meta__":{"kind":"query","name":"ArOpenItems","hash":"sha256:7d695f3c3b6eacc1f00c10b1905a5d1d33969269106ffe2ef5c30e81146fc7d3"}} as unknown as TypedDocumentString<ArOpenItemsQuery, ArOpenItemsQueryVariables>;
export const ArCustomerLedgerDocument = {"__meta__":{"kind":"query","name":"ArCustomerLedger","hash":"sha256:d9554d04015d108ca655b857f923eae260ae36f143a509025f79717a4fdf65c4"}} as unknown as TypedDocumentString<ArCustomerLedgerQuery, ArCustomerLedgerQueryVariables>;
export const ArCustomerStatementDocument = {"__meta__":{"kind":"query","name":"ArCustomerStatement","hash":"sha256:716ca016b568a28b37004384395e0071a31d5fb8334f818d3a1a1aa5a403c21b"}} as unknown as TypedDocumentString<ArCustomerStatementQuery, ArCustomerStatementQueryVariables>;
export const ArDashboardKpisDocument = {"__meta__":{"kind":"query","name":"ArDashboardKpis","hash":"sha256:a05fd02645fddb941d688972550b27be3ce6665510aecb4170c44dd11c491452"}} as unknown as TypedDocumentString<ArDashboardKpisQuery, ArDashboardKpisQueryVariables>;
export const ArDsoTrendDocument = {"__meta__":{"kind":"query","name":"ArDsoTrend","hash":"sha256:031255e1f9c64413b438a2b3825d9dd8c2101fb1705a6c0e6e66175243334571"}} as unknown as TypedDocumentString<ArDsoTrendQuery, ArDsoTrendQueryVariables>;
export const ArAgingTrendDocument = {"__meta__":{"kind":"query","name":"ArAgingTrend","hash":"sha256:ef0336444abe3d8b2ab3101645e9f24367fbb81bda8e4c66f902a2fbfedda191"}} as unknown as TypedDocumentString<ArAgingTrendQuery, ArAgingTrendQueryVariables>;
export const ArCashFlowForecastDocument = {"__meta__":{"kind":"query","name":"ArCashFlowForecast","hash":"sha256:a5a6d234e6dcbd2be9e4244023746ee4c4644b2f93108d908e5f457f74031246"}} as unknown as TypedDocumentString<ArCashFlowForecastQuery, ArCashFlowForecastQueryVariables>;
export const ArCollectionPerformanceDocument = {"__meta__":{"kind":"query","name":"ArCollectionPerformance","hash":"sha256:d3e0b01af564df96113a703515f4c80fdefdbd7b2e70914bd8b7567153fc7e66"}} as unknown as TypedDocumentString<ArCollectionPerformanceQuery, ArCollectionPerformanceQueryVariables>;
export const ArTopOverdueCustomersDocument = {"__meta__":{"kind":"query","name":"ArTopOverdueCustomers","hash":"sha256:dddd0e21b1ac01157ee2f9d7b763f1f83ec482078ae54b30d159aa1e3641e408"}} as unknown as TypedDocumentString<ArTopOverdueCustomersQuery, ArTopOverdueCustomersQueryVariables>;
export const ArCollectionsWorklistDocument = {"__meta__":{"kind":"query","name":"ArCollectionsWorklist","hash":"sha256:a72cdc4147d001e5acaecc0659d81becc1a7fb642c63c8e3f79240651ffadb40"}} as unknown as TypedDocumentString<ArCollectionsWorklistQuery, ArCollectionsWorklistQueryVariables>;
export const ArPaymentStatsDocument = {"__meta__":{"kind":"query","name":"ArPaymentStats","hash":"sha256:a4fe33f6233932aadde3e5ec2e4dc656c78b2e73188bab35638f674c3045bbeb"}} as unknown as TypedDocumentString<ArPaymentStatsQuery, ArPaymentStatsQueryVariables>;
export const ArCustomerProfileDocument = {"__meta__":{"kind":"query","name":"ArCustomerProfile","hash":"sha256:b82086fc8a84f2dcc1c322b26634a1465bf4d240b6a5ff5f9bfd36300fbe7b37"}} as unknown as TypedDocumentString<ArCustomerProfileQuery, ArCustomerProfileQueryVariables>;
export const AgentControlSettingsDocument = {"__meta__":{"kind":"query","name":"AgentControlSettings","hash":"sha256:a0288f356c0efe08e5b5e03734dc189a623608a752569930855e8cbf64a92ec8"}} as unknown as TypedDocumentString<AgentControlSettingsQuery, AgentControlSettingsQueryVariables>;
export const UpdateAgentControlDocument = {"__meta__":{"kind":"mutation","name":"UpdateAgentControl","hash":"sha256:58093530f37a6c071acec43d9b2b1887de5e9e0dee5f701e95ae8772ff374946"}} as unknown as TypedDocumentString<UpdateAgentControlMutation, UpdateAgentControlMutationVariables>;
export const AgentExceptionTableDocument = {"__meta__":{"kind":"query","name":"AgentExceptionTable","hash":"sha256:25ab7e258b1999dd80da81ecf0ad0c5b956991f6fc73cf33a2cd45def0a97b41"}} as unknown as TypedDocumentString<AgentExceptionTableQuery, AgentExceptionTableQueryVariables>;
export const AgentExceptionDetailDocument = {"__meta__":{"kind":"query","name":"AgentExceptionDetail","hash":"sha256:a5f862a28f545ff7151df8c5e238d4c4ea80f137f9c237f2de408fa670227069"}} as unknown as TypedDocumentString<AgentExceptionDetailQuery, AgentExceptionDetailQueryVariables>;
export const ResolveAgentExceptionDocument = {"__meta__":{"kind":"mutation","name":"ResolveAgentException","hash":"sha256:7560a022b9583caf64b19551a5703e3d4717a7ee8297e5359121c469f4357010"}} as unknown as TypedDocumentString<ResolveAgentExceptionMutation, ResolveAgentExceptionMutationVariables>;
export const AgentProposalTableDocument = {"__meta__":{"kind":"query","name":"AgentProposalTable","hash":"sha256:c9ac832fcd7d24834e9d2a255e3ce6871dc33a5f265d399c94cd7dc0104b11f6"}} as unknown as TypedDocumentString<AgentProposalTableQuery, AgentProposalTableQueryVariables>;
export const AgentProposalDetailDocument = {"__meta__":{"kind":"query","name":"AgentProposalDetail","hash":"sha256:04b976a90448c11f4bcd6c48fa4cbb4395bbcd84206f529aa13c6fb8d02408ca"}} as unknown as TypedDocumentString<AgentProposalDetailQuery, AgentProposalDetailQueryVariables>;
export const DecideAgentProposalDocument = {"__meta__":{"kind":"mutation","name":"DecideAgentProposal","hash":"sha256:ba06fd0f5bb9168980d5d967514bf0bcbd80382200e836955aa5704c4c9f1836"}} as unknown as TypedDocumentString<DecideAgentProposalMutation, DecideAgentProposalMutationVariables>;
export const AgentRunTableDocument = {"__meta__":{"kind":"query","name":"AgentRunTable","hash":"sha256:fa9353177a2669c44a06fceca571cd7f13eedc630e6d6c0e519af077639e1479"}} as unknown as TypedDocumentString<AgentRunTableQuery, AgentRunTableQueryVariables>;
export const AgentRunDetailDocument = {"__meta__":{"kind":"query","name":"AgentRunDetail","hash":"sha256:1c4280969a19e8c0d0948fd97bb58d9d25538909c8f9a0224646c8813dfcdfa3"}} as unknown as TypedDocumentString<AgentRunDetailQuery, AgentRunDetailQueryVariables>;
export const ApiKeyTableDocument = {"__meta__":{"kind":"query","name":"ApiKeyTable","hash":"sha256:aeacf34d9ae14863db97c29a2ea928d83c46bba47f49ecd6a05ccdf7d4a33951"}} as unknown as TypedDocumentString<ApiKeyTableQuery, ApiKeyTableQueryVariables>;
export const AttentionSummaryDocument = {"__meta__":{"kind":"query","name":"AttentionSummary","hash":"sha256:f60e116af6655582ac87c443bb9a03a7990af7fdd7671aa37452ea7bc48074b9"}} as unknown as TypedDocumentString<AttentionSummaryQuery, AttentionSummaryQueryVariables>;
export const RecentActivityDocument = {"__meta__":{"kind":"query","name":"RecentActivity","hash":"sha256:3fe2bf53bf715a8f3c32bf9ed4a7f61e822b4d79f54f0564b6b647cf54b13407"}} as unknown as TypedDocumentString<RecentActivityQuery, RecentActivityQueryVariables>;
export const AuditLogTableDocument = {"__meta__":{"kind":"query","name":"AuditLogTable","hash":"sha256:25bcb0e024c30d7999751de7b49eaab0773e668bf4ae7c845b79d54c634acd55"}} as unknown as TypedDocumentString<AuditLogTableQuery, AuditLogTableQueryVariables>;
export const BenefitPlansDocument = {"__meta__":{"kind":"query","name":"BenefitPlans","hash":"sha256:2567ccf7c3576f87385f4b57db408918a0a5b33b31eb1018cd5405f4e4e76c2a"}} as unknown as TypedDocumentString<BenefitPlansQuery, BenefitPlansQueryVariables>;
export const WorkerBenefitEnrollmentsDocument = {"__meta__":{"kind":"query","name":"WorkerBenefitEnrollments","hash":"sha256:59102c2ac200a90ce1c1994ab3745c81b199b3dea70ba5dfd55b0c57e530e8f4"}} as unknown as TypedDocumentString<WorkerBenefitEnrollmentsQuery, WorkerBenefitEnrollmentsQueryVariables>;
export const BenefitEnrollmentsDocument = {"__meta__":{"kind":"query","name":"BenefitEnrollments","hash":"sha256:76e1d395564d801bd963846ad035ca701d7b748b6f51c2205081f2f469937707"}} as unknown as TypedDocumentString<BenefitEnrollmentsQuery, BenefitEnrollmentsQueryVariables>;
export const BenefitCostsDocument = {"__meta__":{"kind":"query","name":"BenefitCosts","hash":"sha256:3d4ae7ffd3d6ed029c7c0f5fc2b9c949ee8e309121001a0afab537fe505920ed"}} as unknown as TypedDocumentString<BenefitCostsQuery, BenefitCostsQueryVariables>;
export const WorkerTotalCompensationDocument = {"__meta__":{"kind":"query","name":"WorkerTotalCompensation","hash":"sha256:3c03c3ffb6dfcefc0f694cfa7adc42d81766d9d8426078de4586ce1401b4d82e"}} as unknown as TypedDocumentString<WorkerTotalCompensationQuery, WorkerTotalCompensationQueryVariables>;
export const MyTotalCompensationDocument = {"__meta__":{"kind":"query","name":"MyTotalCompensation","hash":"sha256:b0992f04068d95517dd04872f28d0f213e2a05d2539eb01f840b57e36661e18e"}} as unknown as TypedDocumentString<MyTotalCompensationQuery, MyTotalCompensationQueryVariables>;
export const CreateBenefitPlanDocument = {"__meta__":{"kind":"mutation","name":"CreateBenefitPlan","hash":"sha256:b8f3e8e6c1ae9f27fd8e950238683eaea204e1a98e3e78274cb0e4c4c33129f3"}} as unknown as TypedDocumentString<CreateBenefitPlanMutation, CreateBenefitPlanMutationVariables>;
export const UpdateBenefitPlanDocument = {"__meta__":{"kind":"mutation","name":"UpdateBenefitPlan","hash":"sha256:e24f090f1778c9c2ef705205de8f8567ffae532ac19857056675976989290e3c"}} as unknown as TypedDocumentString<UpdateBenefitPlanMutation, UpdateBenefitPlanMutationVariables>;
export const EnrollBenefitDocument = {"__meta__":{"kind":"mutation","name":"EnrollBenefit","hash":"sha256:d2af259b015e6e47f89598a6cc26f033ef6e80609f3f4c8d702fda84bc2df1d8"}} as unknown as TypedDocumentString<EnrollBenefitMutation, EnrollBenefitMutationVariables>;
export const EndBenefitEnrollmentDocument = {"__meta__":{"kind":"mutation","name":"EndBenefitEnrollment","hash":"sha256:269f0db73fa33163afee2142e5f751214078a9ae4171658a2610e7bd0bdb7a7c"}} as unknown as TypedDocumentString<EndBenefitEnrollmentMutation, EndBenefitEnrollmentMutationVariables>;
export const UpdateBillingQueueStatusDocument = {"__meta__":{"kind":"mutation","name":"UpdateBillingQueueStatus","hash":"sha256:941445cb1c9ee5677f5b115525133c96fcdac96cbfa2591e8f587620cba61272"}} as unknown as TypedDocumentString<UpdateBillingQueueStatusMutation, UpdateBillingQueueStatusMutationVariables>;
export const AssignBillingQueueBillerDocument = {"__meta__":{"kind":"mutation","name":"AssignBillingQueueBiller","hash":"sha256:9850f8da7824976fc1edee6d78fb22738914fc8ffefb4d3588e54e6c6f66bd9f"}} as unknown as TypedDocumentString<AssignBillingQueueBillerMutation, AssignBillingQueueBillerMutationVariables>;
export const CarrierSettlementTableDocument = {"__meta__":{"kind":"query","name":"CarrierSettlementTable","hash":"sha256:eadf21c42815fc3a90fbd8614c13a9e4960c83698f4cc6a50395a0232c35a5d1"}} as unknown as TypedDocumentString<CarrierSettlementTableQuery, CarrierSettlementTableQueryVariables>;
export const CarrierSettlementDetailDocument = {"__meta__":{"kind":"query","name":"CarrierSettlementDetail","hash":"sha256:cc5b9ce4ed7968de7ec3e10c1c9d40d598baaca2516559ddccbf9ac24d9f8b90"}} as unknown as TypedDocumentString<CarrierSettlementDetailQuery, CarrierSettlementDetailQueryVariables>;
export const CarrierSettlementBatchTableDocument = {"__meta__":{"kind":"query","name":"CarrierSettlementBatchTable","hash":"sha256:073382253e84d646f0909341ebf8fa4421b0830503743352686b6e3e2faee5bc"}} as unknown as TypedDocumentString<CarrierSettlementBatchTableQuery, CarrierSettlementBatchTableQueryVariables>;
export const CarrierSettlementBatchDetailDocument = {"__meta__":{"kind":"query","name":"CarrierSettlementBatchDetail","hash":"sha256:53418bc43e1ee89c98fab0ea098fa24ddc5bbf657ce45db363ecbfaa732027e3"}} as unknown as TypedDocumentString<CarrierSettlementBatchDetailQuery, CarrierSettlementBatchDetailQueryVariables>;
export const CarrierCostEventTableDocument = {"__meta__":{"kind":"query","name":"CarrierCostEventTable","hash":"sha256:ffd7c0de29c14f3794189521a424fa9089bb2f909037045050944c849466a344"}} as unknown as TypedDocumentString<CarrierCostEventTableQuery, CarrierCostEventTableQueryVariables>;
export const CarrierSettlementControlDocument = {"__meta__":{"kind":"query","name":"CarrierSettlementControl","hash":"sha256:163a29bd917af8c9a666cbe4fb7b3719add8334156a43b084a0cda9ae286e360"}} as unknown as TypedDocumentString<CarrierSettlementControlQuery, CarrierSettlementControlQueryVariables>;
export const CurrentCarrierSettlementPeriodDocument = {"__meta__":{"kind":"query","name":"CurrentCarrierSettlementPeriod","hash":"sha256:189c9afc2c1a0801d13970c56e448bb681693ed1ce58627a9bf2b0d9f73c3b30"}} as unknown as TypedDocumentString<CurrentCarrierSettlementPeriodQuery, CurrentCarrierSettlementPeriodQueryVariables>;
export const CarrierSettlementWorkspaceSummaryDocument = {"__meta__":{"kind":"query","name":"CarrierSettlementWorkspaceSummary","hash":"sha256:15c008d3edf287354dea1cad9aa58b18d7a4d32e3fb5847253a9566c5b5e8027"}} as unknown as TypedDocumentString<CarrierSettlementWorkspaceSummaryQuery, CarrierSettlementWorkspaceSummaryQueryVariables>;
export const CarrierLedgerEntriesDocument = {"__meta__":{"kind":"query","name":"CarrierLedgerEntries","hash":"sha256:49627fbc33be15958d7c8171a3fbb43dac4071a3cde7bf612425a460768179b2"}} as unknown as TypedDocumentString<CarrierLedgerEntriesQuery, CarrierLedgerEntriesQueryVariables>;
export const CarrierInvoiceMatchesDocument = {"__meta__":{"kind":"query","name":"CarrierInvoiceMatches","hash":"sha256:342bebfd33438b8caa7ec38fa67c34c7bef06005a70f0da7fcfd47edbb73f190"}} as unknown as TypedDocumentString<CarrierInvoiceMatchesQuery, CarrierInvoiceMatchesQueryVariables>;
export const EdiCarrierInvoicesDocument = {"__meta__":{"kind":"query","name":"EdiCarrierInvoices","hash":"sha256:b44bbc0b1942001aace1ccb57d8ac692a610dab639177488d6232457787622ce"}} as unknown as TypedDocumentString<EdiCarrierInvoicesQuery, EdiCarrierInvoicesQueryVariables>;
export const SuggestCarrierForEdiInvoiceDocument = {"__meta__":{"kind":"query","name":"SuggestCarrierForEdiInvoice","hash":"sha256:b5aa64a822ffc92d294a71816236a2a3be0bcb0db952715b5d46d530e648c382"}} as unknown as TypedDocumentString<SuggestCarrierForEdiInvoiceQuery, SuggestCarrierForEdiInvoiceQueryVariables>;
export const ExportCarrierSettlementBatchCsvDocument = {"__meta__":{"kind":"query","name":"ExportCarrierSettlementBatchCsv","hash":"sha256:add91144201a316a09dd8e91a464313bed62d216d536227cb36f80898d00533b"}} as unknown as TypedDocumentString<ExportCarrierSettlementBatchCsvQuery, ExportCarrierSettlementBatchCsvQueryVariables>;
export const GenerateCarrierSettlementBatchDocument = {"__meta__":{"kind":"mutation","name":"GenerateCarrierSettlementBatch","hash":"sha256:5d4b52fe4c6841d6cf84a1fca15826838cc8790d5f7d01cbaef718546d527a93"}} as unknown as TypedDocumentString<GenerateCarrierSettlementBatchMutation, GenerateCarrierSettlementBatchMutationVariables>;
export const SubmitCarrierSettlementDocument = {"__meta__":{"kind":"mutation","name":"SubmitCarrierSettlement","hash":"sha256:5f603312efac3036a8f1dd5f427616a81a9cf7c7cf2ac5110b7fd85e5dad0347"}} as unknown as TypedDocumentString<SubmitCarrierSettlementMutation, SubmitCarrierSettlementMutationVariables>;
export const ApproveCarrierSettlementDocument = {"__meta__":{"kind":"mutation","name":"ApproveCarrierSettlement","hash":"sha256:ad9b585d98587acc4f26b0a456f6e7cbe326d30103de05891ac19ec0a5341fc7"}} as unknown as TypedDocumentString<ApproveCarrierSettlementMutation, ApproveCarrierSettlementMutationVariables>;
export const RejectCarrierSettlementDocument = {"__meta__":{"kind":"mutation","name":"RejectCarrierSettlement","hash":"sha256:bd2180a2b51de2f424838c4fcc66cac3ccedea1ea8d621c1adca0a5d53884d8a"}} as unknown as TypedDocumentString<RejectCarrierSettlementMutation, RejectCarrierSettlementMutationVariables>;
export const PostCarrierSettlementDocument = {"__meta__":{"kind":"mutation","name":"PostCarrierSettlement","hash":"sha256:f2d7453179bf7a7d2cd2921956c2ffe96ed97f6b7bd46bad5ebe8060d17fc694"}} as unknown as TypedDocumentString<PostCarrierSettlementMutation, PostCarrierSettlementMutationVariables>;
export const MarkCarrierSettlementPaidDocument = {"__meta__":{"kind":"mutation","name":"MarkCarrierSettlementPaid","hash":"sha256:4f6a8d1f6671d63951f4e036fe9cd48fbd8daebb5ee0d181581456fa35a77ef6"}} as unknown as TypedDocumentString<MarkCarrierSettlementPaidMutation, MarkCarrierSettlementPaidMutationVariables>;
export const VoidCarrierSettlementDocument = {"__meta__":{"kind":"mutation","name":"VoidCarrierSettlement","hash":"sha256:ce8b05d78d10fc5db7cbca3f04e6fa1ca48c7c00db88aed242b2ba027f73a5dc"}} as unknown as TypedDocumentString<VoidCarrierSettlementMutation, VoidCarrierSettlementMutationVariables>;
export const RecalculateCarrierSettlementDocument = {"__meta__":{"kind":"mutation","name":"RecalculateCarrierSettlement","hash":"sha256:878471da098a46cb2287d43f18ee444fafd902b86dec4543712fbe00afb952db"}} as unknown as TypedDocumentString<RecalculateCarrierSettlementMutation, RecalculateCarrierSettlementMutationVariables>;
export const AddCarrierSettlementAdjustmentDocument = {"__meta__":{"kind":"mutation","name":"AddCarrierSettlementAdjustment","hash":"sha256:9c2ecc166a009e4f0e95e171200ec99918e5f9a587eb5f79ca65c094776bbb49"}} as unknown as TypedDocumentString<AddCarrierSettlementAdjustmentMutation, AddCarrierSettlementAdjustmentMutationVariables>;
export const RemoveCarrierSettlementAdjustmentDocument = {"__meta__":{"kind":"mutation","name":"RemoveCarrierSettlementAdjustment","hash":"sha256:f1824d88917e5f75145c6705c91310427bd15c0dce5852153e16d5c74679829a"}} as unknown as TypedDocumentString<RemoveCarrierSettlementAdjustmentMutation, RemoveCarrierSettlementAdjustmentMutationVariables>;
export const UpdateCarrierSettlementControlDocument = {"__meta__":{"kind":"mutation","name":"UpdateCarrierSettlementControl","hash":"sha256:9eec23817d83cb811a311687e73e4dc975104a4c45000d129aa4f75849dcb86a"}} as unknown as TypedDocumentString<UpdateCarrierSettlementControlMutation, UpdateCarrierSettlementControlMutationVariables>;
export const LinkEdiCarrierInvoiceToCarrierDocument = {"__meta__":{"kind":"mutation","name":"LinkEdiCarrierInvoiceToCarrier","hash":"sha256:16d8dd8e30b134141007f62ea5284a5cb79916d189e7b0c77823bab024b3a249"}} as unknown as TypedDocumentString<LinkEdiCarrierInvoiceToCarrierMutation, LinkEdiCarrierInvoiceToCarrierMutationVariables>;
export const CreateCarrierInvoiceMatchDocument = {"__meta__":{"kind":"mutation","name":"CreateCarrierInvoiceMatch","hash":"sha256:17d57367630a91c6f3c359aef767e708b7d8977134f4e9387954d105e56a43b3"}} as unknown as TypedDocumentString<CreateCarrierInvoiceMatchMutation, CreateCarrierInvoiceMatchMutationVariables>;
export const AcceptCarrierInvoiceMatchDocument = {"__meta__":{"kind":"mutation","name":"AcceptCarrierInvoiceMatch","hash":"sha256:a2d49f45f44e726172c1f4e35fa621fa4c98082208b56b31a29ac4beb9f28a76"}} as unknown as TypedDocumentString<AcceptCarrierInvoiceMatchMutation, AcceptCarrierInvoiceMatchMutationVariables>;
export const AcceptCarrierInvoiceMatchWithVarianceDocument = {"__meta__":{"kind":"mutation","name":"AcceptCarrierInvoiceMatchWithVariance","hash":"sha256:060f67125969b0df8bd23b2d406141579141403904677fcf383e944aae0c9349"}} as unknown as TypedDocumentString<AcceptCarrierInvoiceMatchWithVarianceMutation, AcceptCarrierInvoiceMatchWithVarianceMutationVariables>;
export const RejectCarrierInvoiceMatchDocument = {"__meta__":{"kind":"mutation","name":"RejectCarrierInvoiceMatch","hash":"sha256:9afe5a46af162af1d29113765c8b1107deee2246b7294f54301c3b2884169ea1"}} as unknown as TypedDocumentString<RejectCarrierInvoiceMatchMutation, RejectCarrierInvoiceMatchMutationVariables>;
export const CarrierTableDocument = {"__meta__":{"kind":"query","name":"CarrierTable","hash":"sha256:ac478c9f580a938e561cbf61ba1416f2413126aa194b2de52cb8b6836e859a23"}} as unknown as TypedDocumentString<CarrierTableQuery, CarrierTableQueryVariables>;
export const CommodityTableDocument = {"__meta__":{"kind":"query","name":"CommodityTable","hash":"sha256:02239b2db6c74085ddc074c2c20f287e0c6b1e42c2eeb5efdc2a731076b2a042"}} as unknown as TypedDocumentString<CommodityTableQuery, CommodityTableQueryVariables>;
export const CostingControlPageDocument = {"__meta__":{"kind":"query","name":"CostingControlPage","hash":"sha256:a85cccb870b7669eca888497e484d84fee24b403957e9d3bbf4ff03b337551ea"}} as unknown as TypedDocumentString<CostingControlPageQuery, CostingControlPageQueryVariables>;
export const ResolvedCostProfilePageDocument = {"__meta__":{"kind":"query","name":"ResolvedCostProfilePage","hash":"sha256:0b2352614b5935706f571748ef919218386d7f44cee104d0a0382467511caf67"}} as unknown as TypedDocumentString<ResolvedCostProfilePageQuery, ResolvedCostProfilePageQueryVariables>;
export const UpdateCostingControlDocument = {"__meta__":{"kind":"mutation","name":"UpdateCostingControl","hash":"sha256:c1105bb2e20563d6d25437d41bf5dc0dbe04cd9acf23164e4782478cbd49fa0d"}} as unknown as TypedDocumentString<UpdateCostingControlMutation, UpdateCostingControlMutationVariables>;
export const UpdateCostCategoryDocument = {"__meta__":{"kind":"mutation","name":"UpdateCostCategory","hash":"sha256:2c74749981a6ed8680896dcf651f47a87e3aa698c2d3bf3c5c9b717e359ca828"}} as unknown as TypedDocumentString<UpdateCostCategoryMutation, UpdateCostCategoryMutationVariables>;
export const CustomFieldDefinitionTableDocument = {"__meta__":{"kind":"query","name":"CustomFieldDefinitionTable","hash":"sha256:8879bd728ec6963d8c91ff6a845341b39caa7a34b6a0e2422922a1c1cdfb52db"}} as unknown as TypedDocumentString<CustomFieldDefinitionTableQuery, CustomFieldDefinitionTableQueryVariables>;
export const CustomerPaymentTableDocument = {"__meta__":{"kind":"query","name":"CustomerPaymentTable","hash":"sha256:9c4c5cd618cfa71a14c83740af38de060712349638f6dc73e636996c9ea8602e"}} as unknown as TypedDocumentString<CustomerPaymentTableQuery, CustomerPaymentTableQueryVariables>;
export const CustomerPaymentDetailDocument = {"__meta__":{"kind":"query","name":"CustomerPaymentDetail","hash":"sha256:63ba0ffb6ef2656f7dc2721a0f6fc0da8dc0d184ee08e3871f26e1048b6a9c3f"}} as unknown as TypedDocumentString<CustomerPaymentDetailQuery, CustomerPaymentDetailQueryVariables>;
export const PostAndApplyCustomerPaymentDocument = {"__meta__":{"kind":"mutation","name":"PostAndApplyCustomerPayment","hash":"sha256:8509b32952e2ba614257d3189c57cbd58da45afbb0dafb31f17958e117f62ea6"}} as unknown as TypedDocumentString<PostAndApplyCustomerPaymentMutation, PostAndApplyCustomerPaymentMutationVariables>;
export const ApplyUnappliedCustomerPaymentDocument = {"__meta__":{"kind":"mutation","name":"ApplyUnappliedCustomerPayment","hash":"sha256:1c0798232c1c035894870e85c421a9f0f214ff7407eb9f35155cfd9b87a9b4e0"}} as unknown as TypedDocumentString<ApplyUnappliedCustomerPaymentMutation, ApplyUnappliedCustomerPaymentMutationVariables>;
export const ReverseCustomerPaymentDocument = {"__meta__":{"kind":"mutation","name":"ReverseCustomerPayment","hash":"sha256:fe84be95798f92734cec53df3348b8593909fafde378deb329d583affe25145f"}} as unknown as TypedDocumentString<ReverseCustomerPaymentMutation, ReverseCustomerPaymentMutationVariables>;
export const CustomerTableDocument = {"__meta__":{"kind":"query","name":"CustomerTable","hash":"sha256:0b8a7471c7b9d00fdcba83362242cfa0a99843540f13ccd742487c4acd845ec0"}} as unknown as TypedDocumentString<CustomerTableQuery, CustomerTableQueryVariables>;
export const DetentionFacilityStatsDocument = {"__meta__":{"kind":"query","name":"DetentionFacilityStats","hash":"sha256:c7c1f1b8d0b5c5fa3dced842b3436b910f8525f3f3dc5992cd753d0e247a40c2"}} as unknown as TypedDocumentString<DetentionFacilityStatsQuery, DetentionFacilityStatsQueryVariables>;
export const DetentionCustomerStatsDocument = {"__meta__":{"kind":"query","name":"DetentionCustomerStats","hash":"sha256:188bf76366f9651b4c37387fa0792d2ba42578a237866a2bbfa3a1d0d904c056"}} as unknown as TypedDocumentString<DetentionCustomerStatsQuery, DetentionCustomerStatsQueryVariables>;
export const DetentionWaiverStatsDocument = {"__meta__":{"kind":"query","name":"DetentionWaiverStats","hash":"sha256:23077469702670463ce465420c6147c9583c2fbb143df6d60a5c0ac2276d0da1"}} as unknown as TypedDocumentString<DetentionWaiverStatsQuery, DetentionWaiverStatsQueryVariables>;
export const DetentionPolicyPreviewDocument = {"__meta__":{"kind":"query","name":"DetentionPolicyPreview","hash":"sha256:f328a9b7f74696985a4257d65ce18dbd22f852376305567d6eaf1cf2ff6eb490"}} as unknown as TypedDocumentString<DetentionPolicyPreviewQuery, DetentionPolicyPreviewQueryVariables>;
export const DetentionBacktestDocument = {"__meta__":{"kind":"mutation","name":"DetentionBacktest","hash":"sha256:3fb6d3d1bc6ca9a6ba843f0529dea282ba78b256406c80d9a47d886ef7014aef"}} as unknown as TypedDocumentString<DetentionBacktestMutation, DetentionBacktestMutationVariables>;
export const DetentionDeskDocument = {"__meta__":{"kind":"query","name":"DetentionDesk","hash":"sha256:1c057e39e0bca0b66bc728666181aadc64acf47ab7c444fab7af0d586fe4798a"}} as unknown as TypedDocumentString<DetentionDeskQuery, DetentionDeskQueryVariables>;
export const DetentionOccurrenceDetailDocument = {"__meta__":{"kind":"query","name":"DetentionOccurrenceDetail","hash":"sha256:70b004f2c36c5ec853d192f295f27795a2d43a3b8487ac7d57cc1a9d9f0043c5"}} as unknown as TypedDocumentString<DetentionOccurrenceDetailQuery, DetentionOccurrenceDetailQueryVariables>;
export const ShipmentDetentionDocument = {"__meta__":{"kind":"query","name":"ShipmentDetention","hash":"sha256:700e9ac1cfa6d524e49b8709c7cfd3569762f1179b0071e3f4a5da81147a4e99"}} as unknown as TypedDocumentString<ShipmentDetentionQuery, ShipmentDetentionQueryVariables>;
export const DetentionDisputePacketDocument = {"__meta__":{"kind":"query","name":"DetentionDisputePacket","hash":"sha256:0dadc48c79e19edf845ee5b5d468fae9a80a12ccbe5e84016fc736075299b293"}} as unknown as TypedDocumentString<DetentionDisputePacketQuery, DetentionDisputePacketQueryVariables>;
export const WaiveDetentionOccurrenceDocument = {"__meta__":{"kind":"mutation","name":"WaiveDetentionOccurrence","hash":"sha256:d665c2bf1d09b15cc03c4ebeaeafb7783fab89d40503826a7c2ab8da9d050f71"}} as unknown as TypedDocumentString<WaiveDetentionOccurrenceMutation, WaiveDetentionOccurrenceMutationVariables>;
export const ApproveDetentionOccurrenceDocument = {"__meta__":{"kind":"mutation","name":"ApproveDetentionOccurrence","hash":"sha256:a64a4971ebe414d368676e932585afc0795028454f7617f0efcf75036fe89f00"}} as unknown as TypedDocumentString<ApproveDetentionOccurrenceMutation, ApproveDetentionOccurrenceMutationVariables>;
export const DisputeDetentionOccurrenceDocument = {"__meta__":{"kind":"mutation","name":"DisputeDetentionOccurrence","hash":"sha256:3276979f1b20a5fc9d0004594e930277fbf789cb7505a4dda9561d58d93e2619"}} as unknown as TypedDocumentString<DisputeDetentionOccurrenceMutation, DisputeDetentionOccurrenceMutationVariables>;
export const SendDetentionNoticeDocument = {"__meta__":{"kind":"mutation","name":"SendDetentionNotice","hash":"sha256:71ee21b771997298c7f03866527d9b7bd1a50d61093a2fb35c53618ec7d6d68d"}} as unknown as TypedDocumentString<SendDetentionNoticeMutation, SendDetentionNoticeMutationVariables>;
export const DetentionPolicyTableDocument = {"__meta__":{"kind":"query","name":"DetentionPolicyTable","hash":"sha256:f73af8919386e166d164dd3f5a6f2ccfc7037989d7a0cc53b3bcf5268898ca64"}} as unknown as TypedDocumentString<DetentionPolicyTableQuery, DetentionPolicyTableQueryVariables>;
export const DetentionPolicyDocument = {"__meta__":{"kind":"query","name":"DetentionPolicy","hash":"sha256:74adcb143195478e6f8c1ee0a8fa6f1df478a06423b437885f6da18e69581560"}} as unknown as TypedDocumentString<DetentionPolicyQuery, DetentionPolicyQueryVariables>;
export const CreateDetentionPolicyDocument = {"__meta__":{"kind":"mutation","name":"CreateDetentionPolicy","hash":"sha256:2d5cbd988b8197f0ef9057001a7eab554153e7fe0c54148326540bf3ae4ba416"}} as unknown as TypedDocumentString<CreateDetentionPolicyMutation, CreateDetentionPolicyMutationVariables>;
export const UpdateDetentionPolicyDocument = {"__meta__":{"kind":"mutation","name":"UpdateDetentionPolicy","hash":"sha256:ec5c989dff8b8c35e44583c673bab420544afcae506928ee1115b69b2a4a2032"}} as unknown as TypedDocumentString<UpdateDetentionPolicyMutation, UpdateDetentionPolicyMutationVariables>;
export const DeleteDetentionPolicyDocument = {"__meta__":{"kind":"mutation","name":"DeleteDetentionPolicy","hash":"sha256:b3080ac3dc795c66808fe6644957188d98fc8bd5457f85ddc9cc6328d05c4f94"}} as unknown as TypedDocumentString<DeleteDetentionPolicyMutation, DeleteDetentionPolicyMutationVariables>;
export const DispatchBoardDocument = {"__meta__":{"kind":"query","name":"DispatchBoard","hash":"sha256:50b6c0ad2fc90911a6afdd7c2b6407b8cc0adbe576a47aa4f1ff556862346023"}} as unknown as TypedDocumentString<DispatchBoardQuery, DispatchBoardQueryVariables>;
export const DispatchMoveCandidatesDocument = {"__meta__":{"kind":"query","name":"DispatchMoveCandidates","hash":"sha256:d4079e5cb8e89f1fc42cc1139ce50534affbc526e20802fb0795b1188cef69f3"}} as unknown as TypedDocumentString<DispatchMoveCandidatesQuery, DispatchMoveCandidatesQueryVariables>;
export const DispatchDriverMovesDocument = {"__meta__":{"kind":"query","name":"DispatchDriverMoves","hash":"sha256:b65d9432b9b4aca443f35d7269f7df80a7a38c24b7ed8f5196b79f58506da189"}} as unknown as TypedDocumentString<DispatchDriverMovesQuery, DispatchDriverMovesQueryVariables>;
export const DispatchAssignmentPreviewDocument = {"__meta__":{"kind":"query","name":"DispatchAssignmentPreview","hash":"sha256:0ef23657908b270b5e8b0364d038b26a3d1b953f2d37de39dffc2696e4cd1361"}} as unknown as TypedDocumentString<DispatchAssignmentPreviewQuery, DispatchAssignmentPreviewQueryVariables>;
export const DispatchAssignMovesDocument = {"__meta__":{"kind":"mutation","name":"DispatchAssignMoves","hash":"sha256:42f79a523ad939739ca9bdf31d4accd65bb426270dc846d2d28c60123919ab4f"}} as unknown as TypedDocumentString<DispatchAssignMovesMutation, DispatchAssignMovesMutationVariables>;
export const DispatchUnassignMovesDocument = {"__meta__":{"kind":"mutation","name":"DispatchUnassignMoves","hash":"sha256:efe06714ee634572b3eb36d985bd6cd8fdf400b9e481e3107da70d9b105dcc22"}} as unknown as TypedDocumentString<DispatchUnassignMovesMutation, DispatchUnassignMovesMutationVariables>;
export const DispatchCarrierAssignmentPreviewDocument = {"__meta__":{"kind":"query","name":"DispatchCarrierAssignmentPreview","hash":"sha256:ad34207e4c1e5cb8720cd75fce163dcebd3ea29c87474c4d3ae3e0d06308ab56"}} as unknown as TypedDocumentString<DispatchCarrierAssignmentPreviewQuery, DispatchCarrierAssignmentPreviewQueryVariables>;
export const DispatchAssignMoveToCarrierDocument = {"__meta__":{"kind":"mutation","name":"DispatchAssignMoveToCarrier","hash":"sha256:47fbec559e3836ac2dd0e05df0e991e451bba80b3bfcf67cfb81eacd83671f8f"}} as unknown as TypedDocumentString<DispatchAssignMoveToCarrierMutation, DispatchAssignMoveToCarrierMutationVariables>;
export const DispatchCancelCarrierAssignmentDocument = {"__meta__":{"kind":"mutation","name":"DispatchCancelCarrierAssignment","hash":"sha256:d4293d76cf90c0647e0c816ecf938b39da7165270b1ca6a05b9e7b4cd49e49ad"}} as unknown as TypedDocumentString<DispatchCancelCarrierAssignmentMutation, DispatchCancelCarrierAssignmentMutationVariables>;
export const DispatchPlanAutoAssignDocument = {"__meta__":{"kind":"mutation","name":"DispatchPlanAutoAssign","hash":"sha256:cf7f6fb258631ab19ef6b06110fede4dbb790462eca8c29f3842f316227ff2e4"}} as unknown as TypedDocumentString<DispatchPlanAutoAssignMutation, DispatchPlanAutoAssignMutationVariables>;
export const DistanceOverrideTableDocument = {"__meta__":{"kind":"query","name":"DistanceOverrideTable","hash":"sha256:0ab0bbef6f74978c45d192e5ed2e6f7b07941548322077947145a3361de0f8f0"}} as unknown as TypedDocumentString<DistanceOverrideTableQuery, DistanceOverrideTableQueryVariables>;
export const DistanceProfileTableDocument = {"__meta__":{"kind":"query","name":"DistanceProfileTable","hash":"sha256:c662d2be53498bf094b24c0593f9815cc76f9bb76dfa5b0de3e750eaf3ca15f7"}} as unknown as TypedDocumentString<DistanceProfileTableQuery, DistanceProfileTableQueryVariables>;
export const DocumentPacketRuleTableDocument = {"__meta__":{"kind":"query","name":"DocumentPacketRuleTable","hash":"sha256:b48c9d5f6f128ccc24ef727cfc720c3f7fe470ac5bb808fabf9992aa1ff108d3"}} as unknown as TypedDocumentString<DocumentPacketRuleTableQuery, DocumentPacketRuleTableQueryVariables>;
export const DocumentTypeTableDocument = {"__meta__":{"kind":"query","name":"DocumentTypeTable","hash":"sha256:407941510e99bf8b3133b28178542bd908f312b90789fa0c2de2b976f57aa482"}} as unknown as TypedDocumentString<DocumentTypeTableQuery, DocumentTypeTableQueryVariables>;
export const WorkerPortalStatusDocument = {"__meta__":{"kind":"query","name":"WorkerPortalStatus","hash":"sha256:747e063d2865387a9bfee8d33322637bfb155421a1605b8b7bee154804b3c4a1"}} as unknown as TypedDocumentString<WorkerPortalStatusQuery, WorkerPortalStatusQueryVariables>;
export const InviteWorkerToPortalDocument = {"__meta__":{"kind":"mutation","name":"InviteWorkerToPortal","hash":"sha256:aca1f784efb7ea2322d18e6ef69d1981810ffac214ea568ae44b56a0cf721a48"}} as unknown as TypedDocumentString<InviteWorkerToPortalMutation, InviteWorkerToPortalMutationVariables>;
export const RevokeWorkerPortalAccessDocument = {"__meta__":{"kind":"mutation","name":"RevokeWorkerPortalAccess","hash":"sha256:f155b1032fe0e180152934a5fb457e75109136e2ffbde362d474828e42dabd7b"}} as unknown as TypedDocumentString<RevokeWorkerPortalAccessMutation, RevokeWorkerPortalAccessMutationVariables>;
export const SettlementDisputeTableDocument = {"__meta__":{"kind":"query","name":"SettlementDisputeTable","hash":"sha256:7e33296ee79a9433244415c5aa65630214f29447010c2fbd6865bc25ce287dc2"}} as unknown as TypedDocumentString<SettlementDisputeTableQuery, SettlementDisputeTableQueryVariables>;
export const SettlementDisputeDetailDocument = {"__meta__":{"kind":"query","name":"SettlementDisputeDetail","hash":"sha256:ccd8fbb15a3ec77478291e05d6dd931d3c71923ffea585cd51b02ecc3964cd26"}} as unknown as TypedDocumentString<SettlementDisputeDetailQuery, SettlementDisputeDetailQueryVariables>;
export const OpenSettlementDisputeCountDocument = {"__meta__":{"kind":"query","name":"OpenSettlementDisputeCount","hash":"sha256:d8f8fb203d4850d7671c64c9b442e243f87771ccdb245289725411c8fc1bbbbe"}} as unknown as TypedDocumentString<OpenSettlementDisputeCountQuery, OpenSettlementDisputeCountQueryVariables>;
export const StartSettlementDisputeReviewDocument = {"__meta__":{"kind":"mutation","name":"StartSettlementDisputeReview","hash":"sha256:d9763f9364a8c8ffb2df5481c6da57a9824953a6f9e7fcca26cde2d132c12028"}} as unknown as TypedDocumentString<StartSettlementDisputeReviewMutation, StartSettlementDisputeReviewMutationVariables>;
export const ResolveSettlementDisputeDocument = {"__meta__":{"kind":"mutation","name":"ResolveSettlementDispute","hash":"sha256:0a9ff3966e92976c95b3815f2820954520d283fd44748dceeb18629e05758401"}} as unknown as TypedDocumentString<ResolveSettlementDisputeMutation, ResolveSettlementDisputeMutationVariables>;
export const MyPortalProfileDocument = {"__meta__":{"kind":"query","name":"MyPortalProfile","hash":"sha256:d9f1923f100755639ac4dcb381b4d12d4ec966896dc62f6feeace051bbb313f8"}} as unknown as TypedDocumentString<MyPortalProfileQuery, MyPortalProfileQueryVariables>;
export const MyLoadsDocument = {"__meta__":{"kind":"query","name":"MyLoads","hash":"sha256:234ab5affc2ec541d377704704289944b19ac2f12553fdb404bce4323e11ba7c"}} as unknown as TypedDocumentString<MyLoadsQuery, MyLoadsQueryVariables>;
export const MyLoadCommentsDocument = {"__meta__":{"kind":"query","name":"MyLoadComments","hash":"sha256:bb8f38084baf00216f6da811c1d01e04b2dfdcf4ac579b48771e2c0103e3e490"}} as unknown as TypedDocumentString<MyLoadCommentsQuery, MyLoadCommentsQueryVariables>;
export const RecordMyStopActionDocument = {"__meta__":{"kind":"mutation","name":"RecordMyStopAction","hash":"sha256:2e9bf84e0ce7dbfd352cba8d60db65f4912a71052aff8f90c7ef530277a91e24"}} as unknown as TypedDocumentString<RecordMyStopActionMutation, RecordMyStopActionMutationVariables>;
export const CreateMyLoadCommentDocument = {"__meta__":{"kind":"mutation","name":"CreateMyLoadComment","hash":"sha256:d1ae76e14b0a4562299fb803f6445db0ba915c29e9de6be29855c42edf501d17"}} as unknown as TypedDocumentString<CreateMyLoadCommentMutation, CreateMyLoadCommentMutationVariables>;
export const MyPeriodSummaryDocument = {"__meta__":{"kind":"query","name":"MyPeriodSummary","hash":"sha256:300c67ba34b8708f1e1352f6bf455729f24ad3f18ba8f7a97d2886d2ba12e382"}} as unknown as TypedDocumentString<MyPeriodSummaryQuery, MyPeriodSummaryQueryVariables>;
export const MyRecentPayEventsDocument = {"__meta__":{"kind":"query","name":"MyRecentPayEvents","hash":"sha256:b21ae5d25eaa70e20606dd2bc6decf9b6be568727a1fd3069ffcd9e43a2ffc1b"}} as unknown as TypedDocumentString<MyRecentPayEventsQuery, MyRecentPayEventsQueryVariables>;
export const MySettlementsDocument = {"__meta__":{"kind":"query","name":"MySettlements","hash":"sha256:2ee91e8c25e4027169a740ef45aaf59ebab0fdba2f9fe2c1616d4a5f6a5bb491"}} as unknown as TypedDocumentString<MySettlementsQuery, MySettlementsQueryVariables>;
export const MySettlementDocument = {"__meta__":{"kind":"query","name":"MySettlement","hash":"sha256:13d06f21c46b4eeeb624270d875cdccc3574ce029799208b15d05c3cc38e499d"}} as unknown as TypedDocumentString<MySettlementQuery, MySettlementQueryVariables>;
export const MyEscrowDocument = {"__meta__":{"kind":"query","name":"MyEscrow","hash":"sha256:d6e92e5dd91225da94c876899ec76b6539d6b200adc66d8f5c9f16743631c4e7"}} as unknown as TypedDocumentString<MyEscrowQuery, MyEscrowQueryVariables>;
export const MyAdvancesDocument = {"__meta__":{"kind":"query","name":"MyAdvances","hash":"sha256:4ad05d0be74ab6dec588aa0b7db8ab77d9cd5d605096e13a97cb20bce852f47f"}} as unknown as TypedDocumentString<MyAdvancesQuery, MyAdvancesQueryVariables>;
export const MyDisputesDocument = {"__meta__":{"kind":"query","name":"MyDisputes","hash":"sha256:ce60a2cceb081d891881972f30a664ff30829c3bee0d0ea86b2d4bb9214d78f6"}} as unknown as TypedDocumentString<MyDisputesQuery, MyDisputesQueryVariables>;
export const CreateSettlementDisputeDocument = {"__meta__":{"kind":"mutation","name":"CreateSettlementDispute","hash":"sha256:369c2c8a4105f0c1c8e19d5e899d99997ba5449ea24316561c28e415cc008123"}} as unknown as TypedDocumentString<CreateSettlementDisputeMutation, CreateSettlementDisputeMutationVariables>;
export const WithdrawSettlementDisputeDocument = {"__meta__":{"kind":"mutation","name":"WithdrawSettlementDispute","hash":"sha256:8ac805417a3dde06fdd223772563f68688bdb73966c0424fcfc441df9db0c205"}} as unknown as TypedDocumentString<WithdrawSettlementDisputeMutation, WithdrawSettlementDisputeMutationVariables>;
export const MyComplianceProfileDocument = {"__meta__":{"kind":"query","name":"MyComplianceProfile","hash":"sha256:ac5af8e8d15be26168448355e9771b11417890d634039dbdf7f39317d739b532"}} as unknown as TypedDocumentString<MyComplianceProfileQuery, MyComplianceProfileQueryVariables>;
export const UpdateMyContactInfoDocument = {"__meta__":{"kind":"mutation","name":"UpdateMyContactInfo","hash":"sha256:e81e7bb10b0ba505ac01858582f241192fb8f9552a6ea9ef910afba79ff73f1d"}} as unknown as TypedDocumentString<UpdateMyContactInfoMutation, UpdateMyContactInfoMutationVariables>;
export const MyPtoDocument = {"__meta__":{"kind":"query","name":"MyPto","hash":"sha256:9fcfb3c6f94692c333a8f9b3f6074f033ace1ec340b3c7ffc601c00618a7e264"}} as unknown as TypedDocumentString<MyPtoQuery, MyPtoQueryVariables>;
export const RequestMyPtoDocument = {"__meta__":{"kind":"mutation","name":"RequestMyPto","hash":"sha256:b1ad80c4985bc9a0b66fe33cb36e37d957a107db4c25fe06a4154b0e76ed7be6"}} as unknown as TypedDocumentString<RequestMyPtoMutation, RequestMyPtoMutationVariables>;
export const CancelMyPtoDocument = {"__meta__":{"kind":"mutation","name":"CancelMyPto","hash":"sha256:f6bff19ede90794d5d921c0277bb7254497e80ce7fb32e10870af9dffc87759c"}} as unknown as TypedDocumentString<CancelMyPtoMutation, CancelMyPtoMutationVariables>;
export const MyExpensesDocument = {"__meta__":{"kind":"query","name":"MyExpenses","hash":"sha256:a37af7272e3a8791c6ea1257e9786bfdb352b5c0c58e29ff8cb01fb72de1145f"}} as unknown as TypedDocumentString<MyExpensesQuery, MyExpensesQueryVariables>;
export const SubmitMyExpenseDocument = {"__meta__":{"kind":"mutation","name":"SubmitMyExpense","hash":"sha256:a7598d8fcd7abe6245cd2dfa7858051fc11048da1632d7f04fbf0914e4dc5ebc"}} as unknown as TypedDocumentString<SubmitMyExpenseMutation, SubmitMyExpenseMutationVariables>;
export const CancelMyExpenseDocument = {"__meta__":{"kind":"mutation","name":"CancelMyExpense","hash":"sha256:ff0f019a13713002ba005dabcfbaaf348ab036dbb4461aa7f8ad6393d208cb3a"}} as unknown as TypedDocumentString<CancelMyExpenseMutation, CancelMyExpenseMutationVariables>;
export const RespondToMyAssignmentDocument = {"__meta__":{"kind":"mutation","name":"RespondToMyAssignment","hash":"sha256:211393bb6c83113d7199803c0c269021ebfac1ecf2a65eb46143051c7e3fb3f2"}} as unknown as TypedDocumentString<RespondToMyAssignmentMutation, RespondToMyAssignmentMutationVariables>;
export const MyLoadPayEstimateDocument = {"__meta__":{"kind":"query","name":"MyLoadPayEstimate","hash":"sha256:153135d927a73aaa32b52850ee67d3f02242447d08c58e4f0058d052f85f32b2"}} as unknown as TypedDocumentString<MyLoadPayEstimateQuery, MyLoadPayEstimateQueryVariables>;
export const MyYtdPayDocument = {"__meta__":{"kind":"query","name":"MyYtdPay","hash":"sha256:3020bc2943b245e09336415b3901f2c59161751172c3a12e1b721972bb8ed133"}} as unknown as TypedDocumentString<MyYtdPayQuery, MyYtdPayQueryVariables>;
export const DriverExpenseTableDocument = {"__meta__":{"kind":"query","name":"DriverExpenseTable","hash":"sha256:990a0defa6fbd08c471ed43984f9eb20bdc89a5da52087a4f9d86d24fe872c2f"}} as unknown as TypedDocumentString<DriverExpenseTableQuery, DriverExpenseTableQueryVariables>;
export const DriverExpenseDetailDocument = {"__meta__":{"kind":"query","name":"DriverExpenseDetail","hash":"sha256:b6dfe3c16c23e841da42029ab4276a328018d85777371bfc36808ae4868cee2d"}} as unknown as TypedDocumentString<DriverExpenseDetailQuery, DriverExpenseDetailQueryVariables>;
export const PendingDriverExpenseCountDocument = {"__meta__":{"kind":"query","name":"PendingDriverExpenseCount","hash":"sha256:ea576d13a22021a5d5e9559b60c5a2082502418662f1ab9e11591b1a0af0e4de"}} as unknown as TypedDocumentString<PendingDriverExpenseCountQuery, PendingDriverExpenseCountQueryVariables>;
export const ReviewDriverExpenseDocument = {"__meta__":{"kind":"mutation","name":"ReviewDriverExpense","hash":"sha256:a7977fe40219d006a216e85a7db70e407be08880a8b5b0fd602d53c4603f6bef"}} as unknown as TypedDocumentString<ReviewDriverExpenseMutation, ReviewDriverExpenseMutationVariables>;
export const DashControlDocument = {"__meta__":{"kind":"query","name":"DashControl","hash":"sha256:05c5ee2e31f35ce98607b9480dd62a50333920a92a38a97aea5ecc2412d7e4e0"}} as unknown as TypedDocumentString<DashControlQuery, DashControlQueryVariables>;
export const UpdateDashControlDocument = {"__meta__":{"kind":"mutation","name":"UpdateDashControl","hash":"sha256:9c74c4d12aab4434426e15100dac774adb7ff554ccf6c0d99d739c9e76f96a51"}} as unknown as TypedDocumentString<UpdateDashControlMutation, UpdateDashControlMutationVariables>;
export const MyPortalFeaturesDocument = {"__meta__":{"kind":"query","name":"MyPortalFeatures","hash":"sha256:459b63241e1be1af7c9fb7316cdb0073d1bcb5a676c993dfcc20406c4eddc057"}} as unknown as TypedDocumentString<MyPortalFeaturesQuery, MyPortalFeaturesQueryVariables>;
export const MyPoliciesDocument = {"__meta__":{"kind":"query","name":"MyPolicies","hash":"sha256:d16d53d3c136c220ac6edf9f3e475b4bcb811a4942f3b155687da99043e9218e"}} as unknown as TypedDocumentString<MyPoliciesQuery, MyPoliciesQueryVariables>;
export const MyPolicyDocumentUrlDocument = {"__meta__":{"kind":"query","name":"MyPolicyDocumentUrl","hash":"sha256:64bcddbe678b43839784053a010d836739dd62b0356739d2132e148c63dbb793"}} as unknown as TypedDocumentString<MyPolicyDocumentUrlQuery, MyPolicyDocumentUrlQueryVariables>;
export const MyProfileChangeRequestsDocument = {"__meta__":{"kind":"query","name":"MyProfileChangeRequests","hash":"sha256:4d027186cb5216adbe7014082bfb1290accbf0f78d108900e8bdb6d54ef59d22"}} as unknown as TypedDocumentString<MyProfileChangeRequestsQuery, MyProfileChangeRequestsQueryVariables>;
export const AcknowledgeMyPolicyDocument = {"__meta__":{"kind":"mutation","name":"AcknowledgeMyPolicy","hash":"sha256:3c67f194d4aec5ed17d704f56fc9a5bf8db6cd4a68cbb51c766b9067775bdf10"}} as unknown as TypedDocumentString<AcknowledgeMyPolicyMutation, AcknowledgeMyPolicyMutationVariables>;
export const WithdrawMyProfileChangeDocument = {"__meta__":{"kind":"mutation","name":"WithdrawMyProfileChange","hash":"sha256:480e8d96d236b74cfc80fd9f5054b496256ce962892c1e67374fc0b670bb284c"}} as unknown as TypedDocumentString<WithdrawMyProfileChangeMutation, WithdrawMyProfileChangeMutationVariables>;
export const MyScheduleDocument = {"__meta__":{"kind":"query","name":"MySchedule","hash":"sha256:bb3428f6652f623670ee1c99b22b5eebf6f057bf55c71c32efcda65dd1c92380"}} as unknown as TypedDocumentString<MyScheduleQuery, MyScheduleQueryVariables>;
export const MyAvailabilityDocument = {"__meta__":{"kind":"query","name":"MyAvailability","hash":"sha256:0919244209ecd24ab6b4ebe80efc336980fe0a4f458a91c4140ac488216e0e2a"}} as unknown as TypedDocumentString<MyAvailabilityQuery, MyAvailabilityQueryVariables>;
export const MyShiftSwapsDocument = {"__meta__":{"kind":"query","name":"MyShiftSwaps","hash":"sha256:cc36a25ddafd5f35665c5e7d1fc76f6608dc5d42ecda321c1aaab64e7e89f216"}} as unknown as TypedDocumentString<MyShiftSwapsQuery, MyShiftSwapsQueryVariables>;
export const SetMyAvailabilityDocument = {"__meta__":{"kind":"mutation","name":"SetMyAvailability","hash":"sha256:b9475cc94da421406926afe5ec773fa2a647956291cf4d1611b006adab1e46c9"}} as unknown as TypedDocumentString<SetMyAvailabilityMutation, SetMyAvailabilityMutationVariables>;
export const ProposeMyShiftSwapDocument = {"__meta__":{"kind":"mutation","name":"ProposeMyShiftSwap","hash":"sha256:501f9d3a5dd0787092cc2d3cbbe3e42c209488fbddc0913b3520e85851462b14"}} as unknown as TypedDocumentString<ProposeMyShiftSwapMutation, ProposeMyShiftSwapMutationVariables>;
export const RespondToMyShiftSwapDocument = {"__meta__":{"kind":"mutation","name":"RespondToMyShiftSwap","hash":"sha256:9df2fde95833532dd637a948768fc1e26c8e50f550a0b1cd911d0a53e7e7ed15"}} as unknown as TypedDocumentString<RespondToMyShiftSwapMutation, RespondToMyShiftSwapMutationVariables>;
export const MyHosStateDocument = {"__meta__":{"kind":"query","name":"MyHosState","hash":"sha256:c1834d0c849a7849f22da9cae7f0c1da9d78e6d9327007af8e1069447be19ba6"}} as unknown as TypedDocumentString<MyHosStateQuery, MyHosStateQueryVariables>;
export const MyHosDailyLogsDocument = {"__meta__":{"kind":"query","name":"MyHosDailyLogs","hash":"sha256:9f6e740bf1e3561177b847ebf0a2d747cdd8485cdd8e8a6aacfce1b68c44b17b"}} as unknown as TypedDocumentString<MyHosDailyLogsQuery, MyHosDailyLogsQueryVariables>;
export const MyHosViolationsDocument = {"__meta__":{"kind":"query","name":"MyHosViolations","hash":"sha256:1f76c5fd3230363ff10241c8842e42d7a30427892165ab63b3724d6fa67b14b2"}} as unknown as TypedDocumentString<MyHosViolationsQuery, MyHosViolationsQueryVariables>;
export const MyCredentialsDocument = {"__meta__":{"kind":"query","name":"MyCredentials","hash":"sha256:a5d7ec091451adb72991252cc222500b60e3d08ccdc4e5f6decc03e716d85f2e"}} as unknown as TypedDocumentString<MyCredentialsQuery, MyCredentialsQueryVariables>;
export const MyPtoBalancesDocument = {"__meta__":{"kind":"query","name":"MyPtoBalances","hash":"sha256:153a013227a3c9825dafdeccd31298a8b4f0b9c62e065d5e214658204263aa24"}} as unknown as TypedDocumentString<MyPtoBalancesQuery, MyPtoBalancesQueryVariables>;
export const MyTrainingDocument = {"__meta__":{"kind":"query","name":"MyTraining","hash":"sha256:140d174e53b4638e853c915a5f5d6b3b562b5e45e90e1d88c75ac7e4d9a63544"}} as unknown as TypedDocumentString<MyTrainingQuery, MyTrainingQueryVariables>;
export const StartMyTrainingDocument = {"__meta__":{"kind":"mutation","name":"StartMyTraining","hash":"sha256:24f55edf56fe01fb7e8692cf681deeec1d4cc230fc4586373b6b3a0a68b9072b"}} as unknown as TypedDocumentString<StartMyTrainingMutation, StartMyTrainingMutationVariables>;
export const AcknowledgeMyTrainingDocument = {"__meta__":{"kind":"mutation","name":"AcknowledgeMyTraining","hash":"sha256:005f6e1094d9df42a5531992a421972d170f2c1d59e5ed6f62afe18255d66b6a"}} as unknown as TypedDocumentString<AcknowledgeMyTrainingMutation, AcknowledgeMyTrainingMutationVariables>;
export const MySafetyScorecardDocument = {"__meta__":{"kind":"query","name":"MySafetyScorecard","hash":"sha256:05daef897c88c84fd13fcbd853889642e291f06a72d0358ab94a8077cf5952a6"}} as unknown as TypedDocumentString<MySafetyScorecardQuery, MySafetyScorecardQueryVariables>;
export const MyRecognitionsDocument = {"__meta__":{"kind":"query","name":"MyRecognitions","hash":"sha256:801e6a0ecdfe79258b919f109b95f292f0b69e3cad310d6a585f464794581343"}} as unknown as TypedDocumentString<MyRecognitionsQuery, MyRecognitionsQueryVariables>;
export const MyDisciplinaryActionsDocument = {"__meta__":{"kind":"query","name":"MyDisciplinaryActions","hash":"sha256:52fa7096cc3645e01a421b84db5495a636d50ecf3ad75fcf5a5a7b2407690a13"}} as unknown as TypedDocumentString<MyDisciplinaryActionsQuery, MyDisciplinaryActionsQueryVariables>;
export const AcknowledgeMyDisciplinaryActionDocument = {"__meta__":{"kind":"mutation","name":"AcknowledgeMyDisciplinaryAction","hash":"sha256:c1b89e62f958457c23b61dc85b70be69bbdfd6daee938340f28b393171b24ace"}} as unknown as TypedDocumentString<AcknowledgeMyDisciplinaryActionMutation, AcknowledgeMyDisciplinaryActionMutationVariables>;
export const MyReviewsDocument = {"__meta__":{"kind":"query","name":"MyReviews","hash":"sha256:a7bed103f228bd36310c406a82821fdd10ac7f6e8a48d42c0e305d5908ae209e"}} as unknown as TypedDocumentString<MyReviewsQuery, MyReviewsQueryVariables>;
export const AcknowledgeMyReviewDocument = {"__meta__":{"kind":"mutation","name":"AcknowledgeMyReview","hash":"sha256:c7c2b8a9b23822531e9f5787d630e6c9a01c1cb0455d2b4fd59e9dcd8bfeb9b5"}} as unknown as TypedDocumentString<AcknowledgeMyReviewMutation, AcknowledgeMyReviewMutationVariables>;
export const MyLeaveDocument = {"__meta__":{"kind":"query","name":"MyLeave","hash":"sha256:f5a39098e5d30f883abd0052accff180b38b2766aeaf78296caf500436e0cef6"}} as unknown as TypedDocumentString<MyLeaveQuery, MyLeaveQueryVariables>;
export const PayProfileTableDocument = {"__meta__":{"kind":"query","name":"PayProfileTable","hash":"sha256:3989e49328ac63b780194fec25e0906117a2e8e36b6a93cff9b3606f1e069c0a"}} as unknown as TypedDocumentString<PayProfileTableQuery, PayProfileTableQueryVariables>;
export const WorkerPayAssignmentsDocument = {"__meta__":{"kind":"query","name":"WorkerPayAssignments","hash":"sha256:1bf635cd5fa402d1768672762dcc0cca45e44bfd28abb08b7c7caa1a7efc6528"}} as unknown as TypedDocumentString<WorkerPayAssignmentsQuery, WorkerPayAssignmentsQueryVariables>;
export const EffectiveWorkerPayAssignmentDocument = {"__meta__":{"kind":"query","name":"EffectiveWorkerPayAssignment","hash":"sha256:2249cf1dac4bf029650c70b40a3b51522bdfb085d8d4b0783a8e58b649b5524b"}} as unknown as TypedDocumentString<EffectiveWorkerPayAssignmentQuery, EffectiveWorkerPayAssignmentQueryVariables>;
export const PayProfileAssignmentsDocument = {"__meta__":{"kind":"query","name":"PayProfileAssignments","hash":"sha256:a5cd25fafd7706fafd7a770aec517285d5ed7b8ab090cbd5756fb66f7e528221"}} as unknown as TypedDocumentString<PayProfileAssignmentsQuery, PayProfileAssignmentsQueryVariables>;
export const PayProfileDetailDocument = {"__meta__":{"kind":"query","name":"PayProfileDetail","hash":"sha256:7c7934aa1908ddd5d2d39e488200ef13338e7cdbd5eb59f8a8489cb7a835be3b"}} as unknown as TypedDocumentString<PayProfileDetailQuery, PayProfileDetailQueryVariables>;
export const RecurringDeductionTableDocument = {"__meta__":{"kind":"query","name":"RecurringDeductionTable","hash":"sha256:2091e3bab8818f8f4a18a10e312c215085875fc70cbcd99bb82c1c2ea4365f31"}} as unknown as TypedDocumentString<RecurringDeductionTableQuery, RecurringDeductionTableQueryVariables>;
export const RecurringEarningTableDocument = {"__meta__":{"kind":"query","name":"RecurringEarningTable","hash":"sha256:bb6735298137ec6251ef3203a3d02ef9e86f4ec0fe7e6cb6c21de0921df49652"}} as unknown as TypedDocumentString<RecurringEarningTableQuery, RecurringEarningTableQueryVariables>;
export const PayCodeTableDocument = {"__meta__":{"kind":"query","name":"PayCodeTable","hash":"sha256:1ca53d6794a3244983c50cc59845638848fab2830144fa5bf1bbd5bc316ae7f9"}} as unknown as TypedDocumentString<PayCodeTableQuery, PayCodeTableQueryVariables>;
export const PayAdvanceTableDocument = {"__meta__":{"kind":"query","name":"PayAdvanceTable","hash":"sha256:10c3df571b55032b71491b50a4a24ab051acf8b45a2915d43242a252092c642a"}} as unknown as TypedDocumentString<PayAdvanceTableQuery, PayAdvanceTableQueryVariables>;
export const EscrowAccountTableDocument = {"__meta__":{"kind":"query","name":"EscrowAccountTable","hash":"sha256:916f0bb7ac7aa5145b3d17132631582632435a983dc6071c949475b621237fdf"}} as unknown as TypedDocumentString<EscrowAccountTableQuery, EscrowAccountTableQueryVariables>;
export const EscrowAccountDetailDocument = {"__meta__":{"kind":"query","name":"EscrowAccountDetail","hash":"sha256:57e8a0b5d1c97b85fcb0dc2aa0b40ffaec3b9fbfaf20e3d00e4983912db939b9"}} as unknown as TypedDocumentString<EscrowAccountDetailQuery, EscrowAccountDetailQueryVariables>;
export const DriverSettlementTableDocument = {"__meta__":{"kind":"query","name":"DriverSettlementTable","hash":"sha256:c6c4db00b320c409c60a7180e9cf218de3977bd5c3124a806e71f26a35f5a477"}} as unknown as TypedDocumentString<DriverSettlementTableQuery, DriverSettlementTableQueryVariables>;
export const DriverSettlementDetailDocument = {"__meta__":{"kind":"query","name":"DriverSettlementDetail","hash":"sha256:0f96d6515e17c15b83b8b0e56cf436a5626dd09d6bbb87efbcfbdbddcf0bf68b"}} as unknown as TypedDocumentString<DriverSettlementDetailQuery, DriverSettlementDetailQueryVariables>;
export const SettlementBatchTableDocument = {"__meta__":{"kind":"query","name":"SettlementBatchTable","hash":"sha256:09560c29a693aea9a98d7c3f6dcfee012251d437b9c294c5a00ab886507980aa"}} as unknown as TypedDocumentString<SettlementBatchTableQuery, SettlementBatchTableQueryVariables>;
export const DriverPayEventTableDocument = {"__meta__":{"kind":"query","name":"DriverPayEventTable","hash":"sha256:86f914da516d0e76e82f1f4f8b76dba9ecd3618902c43e88b06315e3b3b8d41a"}} as unknown as TypedDocumentString<DriverPayEventTableQuery, DriverPayEventTableQueryVariables>;
export const WorkerEarningsSummaryDocument = {"__meta__":{"kind":"query","name":"WorkerEarningsSummary","hash":"sha256:604360bedc59b36d762fa6780dbfe073bfbfeb71015006a8a74717663ea17f9a"}} as unknown as TypedDocumentString<WorkerEarningsSummaryQuery, WorkerEarningsSummaryQueryVariables>;
export const WorkerYtdPaySummariesDocument = {"__meta__":{"kind":"query","name":"WorkerYtdPaySummaries","hash":"sha256:f7d96ae4a669343c9c3f27a40e57063c9565295d8c2fa4924e44a2643b5ae67b"}} as unknown as TypedDocumentString<WorkerYtdPaySummariesQuery, WorkerYtdPaySummariesQueryVariables>;
export const SettlementControlDocument = {"__meta__":{"kind":"query","name":"SettlementControl","hash":"sha256:000046c1cbdafc864b73f222fca8fad96ed5a4ff21e87d5807bae7f56f070673"}} as unknown as TypedDocumentString<SettlementControlQuery, SettlementControlQueryVariables>;
export const SettlementWorkspaceSummaryDocument = {"__meta__":{"kind":"query","name":"SettlementWorkspaceSummary","hash":"sha256:e98cd9c3d1b5be60e144a421e8fd46bbffe5d3b35d7562010a32d16b4392f17d"}} as unknown as TypedDocumentString<SettlementWorkspaceSummaryQuery, SettlementWorkspaceSummaryQueryVariables>;
export const UnsettledWorkerSummariesDocument = {"__meta__":{"kind":"query","name":"UnsettledWorkerSummaries","hash":"sha256:fa612e4d6775831d3817cd06ba109f7cfe50a6dcdcf6ed42fc76a7075c5e7925"}} as unknown as TypedDocumentString<UnsettledWorkerSummariesQuery, UnsettledWorkerSummariesQueryVariables>;
export const CurrentSettlementPeriodDocument = {"__meta__":{"kind":"query","name":"CurrentSettlementPeriod","hash":"sha256:18a696ad99213f20822751f7dd47c0e146724a9a15cd4acbf0725a1964074680"}} as unknown as TypedDocumentString<CurrentSettlementPeriodQuery, CurrentSettlementPeriodQueryVariables>;
export const PreviewDriverSettlementDocument = {"__meta__":{"kind":"query","name":"PreviewDriverSettlement","hash":"sha256:1f44e9579262d8defa899f36df03c1e833bf4b67a2d8b7ad5a5be848c51f7935"}} as unknown as TypedDocumentString<PreviewDriverSettlementQuery, PreviewDriverSettlementQueryVariables>;
export const ExportSettlementBatchCsvDocument = {"__meta__":{"kind":"query","name":"ExportSettlementBatchCsv","hash":"sha256:09b35d0d3a76868967b6a609edee59aab83a14a47829d873be063ad8bafd1dd4"}} as unknown as TypedDocumentString<ExportSettlementBatchCsvQuery, ExportSettlementBatchCsvQueryVariables>;
export const CreatePayProfileDocument = {"__meta__":{"kind":"mutation","name":"CreatePayProfile","hash":"sha256:057e95826edc7c7f2e812cc37894744703aee39c278c09bde038098c6dc7e632"}} as unknown as TypedDocumentString<CreatePayProfileMutation, CreatePayProfileMutationVariables>;
export const UpdatePayProfileDocument = {"__meta__":{"kind":"mutation","name":"UpdatePayProfile","hash":"sha256:2aa4747470ca1fb5b86dfdf513276a06a579f785fb0ffc605e38f60feca010c7"}} as unknown as TypedDocumentString<UpdatePayProfileMutation, UpdatePayProfileMutationVariables>;
export const AssignPayProfileToWorkerDocument = {"__meta__":{"kind":"mutation","name":"AssignPayProfileToWorker","hash":"sha256:81e72cc2a26f0710b7df1d899b7ff8e97f26b9ab1d5f37d8c99876073c9f5bf0"}} as unknown as TypedDocumentString<AssignPayProfileToWorkerMutation, AssignPayProfileToWorkerMutationVariables>;
export const EndWorkerPayAssignmentDocument = {"__meta__":{"kind":"mutation","name":"EndWorkerPayAssignment","hash":"sha256:971182611f2f9ad0a2c0ee7a12060d646bce07472d790f6d363790dedb7ae52a"}} as unknown as TypedDocumentString<EndWorkerPayAssignmentMutation, EndWorkerPayAssignmentMutationVariables>;
export const CreateRecurringDeductionDocument = {"__meta__":{"kind":"mutation","name":"CreateRecurringDeduction","hash":"sha256:27048f4cb91e9ceb4cc32391c58f7012d07253930cd32f1377de5a33a5eb6c67"}} as unknown as TypedDocumentString<CreateRecurringDeductionMutation, CreateRecurringDeductionMutationVariables>;
export const UpdateRecurringDeductionDocument = {"__meta__":{"kind":"mutation","name":"UpdateRecurringDeduction","hash":"sha256:60ff79afe87b8907fa14e6470065770f4022043a122db9866b9138d2b42ca15a"}} as unknown as TypedDocumentString<UpdateRecurringDeductionMutation, UpdateRecurringDeductionMutationVariables>;
export const CreatePayCodeDocument = {"__meta__":{"kind":"mutation","name":"CreatePayCode","hash":"sha256:aa0e0bad8977697bc927a0e8dc931a1e29d5699e278af3e61a2b70feb5ea08ee"}} as unknown as TypedDocumentString<CreatePayCodeMutation, CreatePayCodeMutationVariables>;
export const UpdatePayCodeDocument = {"__meta__":{"kind":"mutation","name":"UpdatePayCode","hash":"sha256:54054095fdcf12e64ef6dd8c31e280536a250d01bca95292636c708c73cbfd4e"}} as unknown as TypedDocumentString<UpdatePayCodeMutation, UpdatePayCodeMutationVariables>;
export const CreateRecurringEarningDocument = {"__meta__":{"kind":"mutation","name":"CreateRecurringEarning","hash":"sha256:f92e8c58da59d89b21355bb1378503a710fccf747afea2987da9b5c11ea5aaab"}} as unknown as TypedDocumentString<CreateRecurringEarningMutation, CreateRecurringEarningMutationVariables>;
export const UpdateRecurringEarningDocument = {"__meta__":{"kind":"mutation","name":"UpdateRecurringEarning","hash":"sha256:15d49ce8f2769edc8dc6c562aef844ce80a3c2fb015a337f1a57bf16b2377ffc"}} as unknown as TypedDocumentString<UpdateRecurringEarningMutation, UpdateRecurringEarningMutationVariables>;
export const IssuePayAdvanceDocument = {"__meta__":{"kind":"mutation","name":"IssuePayAdvance","hash":"sha256:a473c5b8196a7a4b792c124cd2f065f1361bf5327ff39363222ead87010fd6f5"}} as unknown as TypedDocumentString<IssuePayAdvanceMutation, IssuePayAdvanceMutationVariables>;
export const WriteOffPayAdvanceDocument = {"__meta__":{"kind":"mutation","name":"WriteOffPayAdvance","hash":"sha256:1e61ca8e3b303b5eeede39215a299509c6d83213342e1d13138c19d24c4c09f2"}} as unknown as TypedDocumentString<WriteOffPayAdvanceMutation, WriteOffPayAdvanceMutationVariables>;
export const OpenEscrowAccountDocument = {"__meta__":{"kind":"mutation","name":"OpenEscrowAccount","hash":"sha256:44f99ba015deaaba7ba8a36c235f1b9cd31960106e649665df8597d4104f2646"}} as unknown as TypedDocumentString<OpenEscrowAccountMutation, OpenEscrowAccountMutationVariables>;
export const UpdateEscrowAccountDocument = {"__meta__":{"kind":"mutation","name":"UpdateEscrowAccount","hash":"sha256:7c8e094b2dff1858e448e77b9315e02eabe34847e1c4bb36aaaa9da9afb8b791"}} as unknown as TypedDocumentString<UpdateEscrowAccountMutation, UpdateEscrowAccountMutationVariables>;
export const AdjustEscrowAccountDocument = {"__meta__":{"kind":"mutation","name":"AdjustEscrowAccount","hash":"sha256:67d1baa098e04646476ef4bec1bb7633ac6caac81c71f4b71b7b54b539c60a21"}} as unknown as TypedDocumentString<AdjustEscrowAccountMutation, AdjustEscrowAccountMutationVariables>;
export const CloseEscrowAccountDocument = {"__meta__":{"kind":"mutation","name":"CloseEscrowAccount","hash":"sha256:4058437f8024641e9e3da7a2d98b8655beb6dee382b4b9c3e77f255c6da27328"}} as unknown as TypedDocumentString<CloseEscrowAccountMutation, CloseEscrowAccountMutationVariables>;
export const GenerateSettlementBatchDocument = {"__meta__":{"kind":"mutation","name":"GenerateSettlementBatch","hash":"sha256:08e2f9acd12b2117789bfffc7bc36d074c3c14ae66935f1953cd371093439277"}} as unknown as TypedDocumentString<GenerateSettlementBatchMutation, GenerateSettlementBatchMutationVariables>;
export const GenerateDriverSettlementDocument = {"__meta__":{"kind":"mutation","name":"GenerateDriverSettlement","hash":"sha256:0d8cbbe97e9f00c3c4c3e36e9bcc4c027760a26fc34c241cc15df6742056a4a2"}} as unknown as TypedDocumentString<GenerateDriverSettlementMutation, GenerateDriverSettlementMutationVariables>;
export const SubmitDriverSettlementDocument = {"__meta__":{"kind":"mutation","name":"SubmitDriverSettlement","hash":"sha256:690603bb04ff7ceab1c1ac6949e3e5f932152daab6396a7f36fa8d72396e4aaf"}} as unknown as TypedDocumentString<SubmitDriverSettlementMutation, SubmitDriverSettlementMutationVariables>;
export const ApproveDriverSettlementDocument = {"__meta__":{"kind":"mutation","name":"ApproveDriverSettlement","hash":"sha256:8256dbbc6c4af57dfaafe586fbc3cbc789baaa24e23ff173c0251b0ad208d7c0"}} as unknown as TypedDocumentString<ApproveDriverSettlementMutation, ApproveDriverSettlementMutationVariables>;
export const RejectDriverSettlementDocument = {"__meta__":{"kind":"mutation","name":"RejectDriverSettlement","hash":"sha256:655f7d88cbe5f76b89bc3ec04a1dde34465fefbdf00304559620b0b20dcc64f2"}} as unknown as TypedDocumentString<RejectDriverSettlementMutation, RejectDriverSettlementMutationVariables>;
export const PostDriverSettlementDocument = {"__meta__":{"kind":"mutation","name":"PostDriverSettlement","hash":"sha256:528e783cf5406e6b3c351f88e4deb5941a3bfc5b90c5d547e7c8e07b28ba136c"}} as unknown as TypedDocumentString<PostDriverSettlementMutation, PostDriverSettlementMutationVariables>;
export const MarkDriverSettlementPaidDocument = {"__meta__":{"kind":"mutation","name":"MarkDriverSettlementPaid","hash":"sha256:52293ccd71fa3f9f752a2c6ee79ece56b5b121240379e0a777dbf2dd1b11788e"}} as unknown as TypedDocumentString<MarkDriverSettlementPaidMutation, MarkDriverSettlementPaidMutationVariables>;
export const VoidDriverSettlementDocument = {"__meta__":{"kind":"mutation","name":"VoidDriverSettlement","hash":"sha256:f5a663c3939fddfbe2d23987cad6a6ca7fd8a03df0e933270cfe021136c166e8"}} as unknown as TypedDocumentString<VoidDriverSettlementMutation, VoidDriverSettlementMutationVariables>;
export const RecalculateDriverSettlementDocument = {"__meta__":{"kind":"mutation","name":"RecalculateDriverSettlement","hash":"sha256:527b34093726f31a9b0c76a39a26b9776e811990ecdf563159d42623c9947350"}} as unknown as TypedDocumentString<RecalculateDriverSettlementMutation, RecalculateDriverSettlementMutationVariables>;
export const AddDriverSettlementAdjustmentDocument = {"__meta__":{"kind":"mutation","name":"AddDriverSettlementAdjustment","hash":"sha256:847f907ae868014335a3a237a6ce3f3b7a0a0b56eea44d3b8e278bdc7b3fe2cb"}} as unknown as TypedDocumentString<AddDriverSettlementAdjustmentMutation, AddDriverSettlementAdjustmentMutationVariables>;
export const RemoveDriverSettlementAdjustmentDocument = {"__meta__":{"kind":"mutation","name":"RemoveDriverSettlementAdjustment","hash":"sha256:fa1ff7af4e0d35dbb7c44913e146dbfa4c122f02807028dfb69ce7d2710bee46"}} as unknown as TypedDocumentString<RemoveDriverSettlementAdjustmentMutation, RemoveDriverSettlementAdjustmentMutationVariables>;
export const HoldDriverPayEventDocument = {"__meta__":{"kind":"mutation","name":"HoldDriverPayEvent","hash":"sha256:19b550da25ad6ee05d644fc426fb0cb4eed437c28373ed7a55847e5218753a97"}} as unknown as TypedDocumentString<HoldDriverPayEventMutation, HoldDriverPayEventMutationVariables>;
export const ReleaseDriverPayEventDocument = {"__meta__":{"kind":"mutation","name":"ReleaseDriverPayEvent","hash":"sha256:74e127862ab228018eb6817bcf2dfe47df3616ff045b776f9ceb7fb7768f8875"}} as unknown as TypedDocumentString<ReleaseDriverPayEventMutation, ReleaseDriverPayEventMutationVariables>;
export const AttachPayEventsToSettlementDocument = {"__meta__":{"kind":"mutation","name":"AttachPayEventsToSettlement","hash":"sha256:ce8dcbcff0fbfc094af72f65403e51dd3e05b3e66be8f6e093daf7020d5647e6"}} as unknown as TypedDocumentString<AttachPayEventsToSettlementMutation, AttachPayEventsToSettlementMutationVariables>;
export const DetachPayEventFromSettlementDocument = {"__meta__":{"kind":"mutation","name":"DetachPayEventFromSettlement","hash":"sha256:30116a97824390c55a39f8976f0816dda38f06d8e92b2d06a87d7fb8c697672c"}} as unknown as TypedDocumentString<DetachPayEventFromSettlementMutation, DetachPayEventFromSettlementMutationVariables>;
export const BulkDriverSettlementActionDocument = {"__meta__":{"kind":"mutation","name":"BulkDriverSettlementAction","hash":"sha256:400632fe9026dfd69abf6dfee45c166104d0d028b6c3f13ab32b90a73da8eda6"}} as unknown as TypedDocumentString<BulkDriverSettlementActionMutation, BulkDriverSettlementActionMutationVariables>;
export const UpdateSettlementControlDocument = {"__meta__":{"kind":"mutation","name":"UpdateSettlementControl","hash":"sha256:d1cd27e9fd5f72b0c09c7000850c3749c37cdb3b1e2eefa5ec9244a59b47d002"}} as unknown as TypedDocumentString<UpdateSettlementControlMutation, UpdateSettlementControlMutationVariables>;
export const SettlementBatchDetailDocument = {"__meta__":{"kind":"query","name":"SettlementBatchDetail","hash":"sha256:f417f184b9f58856ebe25866495f696c2fa0cc4b56bf14615f9c74ed3a5d3250"}} as unknown as TypedDocumentString<SettlementBatchDetailQuery, SettlementBatchDetailQueryVariables>;
export const UnsettledPayEventsDocument = {"__meta__":{"kind":"query","name":"UnsettledPayEvents","hash":"sha256:f9b3d5579995e1ea62171d1cd9c081bd5018e816c58f119b94589db91905ef98"}} as unknown as TypedDocumentString<UnsettledPayEventsQuery, UnsettledPayEventsQueryVariables>;
export const PayWorkerNowDocument = {"__meta__":{"kind":"mutation","name":"PayWorkerNow","hash":"sha256:db65206e7e8cc3bd6558487534d20c043d05e59b9bbef71d89b006fd38580973"}} as unknown as TypedDocumentString<PayWorkerNowMutation, PayWorkerNowMutationVariables>;
export const EdiPartnerScorecardsDocument = {"__meta__":{"kind":"query","name":"EdiPartnerScorecards","hash":"sha256:f96494e915c8a2ece90302cd0a0fc58743dacddaa6685b3cabdae9b07c4cbda4"}} as unknown as TypedDocumentString<EdiPartnerScorecardsQuery, EdiPartnerScorecardsQueryVariables>;
export const EdiVolumeSeriesDocument = {"__meta__":{"kind":"query","name":"EdiVolumeSeries","hash":"sha256:8021ec391401918feed133209023963adf7ea69667f15ceb36ea26dd898e5c74"}} as unknown as TypedDocumentString<EdiVolumeSeriesQuery, EdiVolumeSeriesQueryVariables>;
export const EdiTemplateListDocument = {"__meta__":{"kind":"query","name":"EdiTemplateList","hash":"sha256:a702460c84a4357ec9a9f0c021d1cd597b922e7a244abe96babeb696645b5b12"}} as unknown as TypedDocumentString<EdiTemplateListQuery, EdiTemplateListQueryVariables>;
export const EdiPartnerReadinessDocument = {"__meta__":{"kind":"query","name":"EdiPartnerReadiness","hash":"sha256:6a4869ef7f675e627e9080ece4fbf1ef01427e11cc848095267bbc312d95db31"}} as unknown as TypedDocumentString<EdiPartnerReadinessQuery, EdiPartnerReadinessQueryVariables>;
export const EdiSummaryDocument = {"__meta__":{"kind":"query","name":"EdiSummary","hash":"sha256:3ac851e5c896dbfb25ef2c5275379c815430951cf1f410cb48d6587ccbf2ce0a"}} as unknown as TypedDocumentString<EdiSummaryQuery, EdiSummaryQueryVariables>;
export const EdiPartnerTableDocument = {"__meta__":{"kind":"query","name":"EdiPartnerTable","hash":"sha256:8908cf2f54060bd19a629b5dc91bd2152fd6a41cf42c5a3b64709a6cae5e9890"}} as unknown as TypedDocumentString<EdiPartnerTableQuery, EdiPartnerTableQueryVariables>;
export const EdiCommunicationProfileTableDocument = {"__meta__":{"kind":"query","name":"EdiCommunicationProfileTable","hash":"sha256:5df5e2aef6aca48d461aff67416683ff6068406e5a5a8ad370e49159414b0d4c"}} as unknown as TypedDocumentString<EdiCommunicationProfileTableQuery, EdiCommunicationProfileTableQueryVariables>;
export const EdiTransferTableDocument = {"__meta__":{"kind":"query","name":"EdiTransferTable","hash":"sha256:319972bb69abd3402fe157f59247682e29d5a63122478ad98c738d3f551295fb"}} as unknown as TypedDocumentString<EdiTransferTableQuery, EdiTransferTableQueryVariables>;
export const EdiMessageTableDocument = {"__meta__":{"kind":"query","name":"EdiMessageTable","hash":"sha256:3febe3037092f7bc428a08f93920ee0f7aca26f1f25b092b5e29eea4b9ef780b"}} as unknown as TypedDocumentString<EdiMessageTableQuery, EdiMessageTableQueryVariables>;
export const EdiInboundFileTableDocument = {"__meta__":{"kind":"query","name":"EdiInboundFileTable","hash":"sha256:b40bc2760c60e9939959b3be5dcbb0001a555eb36d74faf67d68791d35e110fc"}} as unknown as TypedDocumentString<EdiInboundFileTableQuery, EdiInboundFileTableQueryVariables>;
export const EdiMappingProfileTableDocument = {"__meta__":{"kind":"query","name":"EdiMappingProfileTable","hash":"sha256:fdf308c922fc97a1874315f013d2aa3179116876be6307aa8f01672080addecb"}} as unknown as TypedDocumentString<EdiMappingProfileTableQuery, EdiMappingProfileTableQueryVariables>;
export const EdiTestCaseTableDocument = {"__meta__":{"kind":"query","name":"EdiTestCaseTable","hash":"sha256:d756c48e3126c301617365bb1e51392207be8ea5dfe5eb5edd98ee944fe9ae2e"}} as unknown as TypedDocumentString<EdiTestCaseTableQuery, EdiTestCaseTableQueryVariables>;
export const EmailProfileTableDocument = {"__meta__":{"kind":"query","name":"EmailProfileTable","hash":"sha256:6715e7bbdbc8507f91814cf918ce342f71e29f1ff52f82d38056f843a8ffa79f"}} as unknown as TypedDocumentString<EmailProfileTableQuery, EmailProfileTableQueryVariables>;
export const EquipmentManufacturerTableDocument = {"__meta__":{"kind":"query","name":"EquipmentManufacturerTable","hash":"sha256:1ad59b9754cf4b8c511c8cf3af0762a6da6adf252d5bb3fd05b50b349bcdaec7"}} as unknown as TypedDocumentString<EquipmentManufacturerTableQuery, EquipmentManufacturerTableQueryVariables>;
export const TractorTableDocument = {"__meta__":{"kind":"query","name":"TractorTable","hash":"sha256:19329f9543d8714fd002b65bb9f04fcacc89b404bbb21d62cd6f42b21b4553e5"}} as unknown as TypedDocumentString<TractorTableQuery, TractorTableQueryVariables>;
export const TrailerTableDocument = {"__meta__":{"kind":"query","name":"TrailerTable","hash":"sha256:f5ca58d5c5853b2e6ac5dab10cf884d1c5ccce29cda55b95957215797da21f42"}} as unknown as TypedDocumentString<TrailerTableQuery, TrailerTableQueryVariables>;
export const EquipmentTypeTableDocument = {"__meta__":{"kind":"query","name":"EquipmentTypeTable","hash":"sha256:434594d9f9c59a4377be5e555d4baa8f80e86e8a0eea9ec679c7cb7ba3b95f05"}} as unknown as TypedDocumentString<EquipmentTypeTableQuery, EquipmentTypeTableQueryVariables>;
export const EquipmentTypeDocument = {"__meta__":{"kind":"query","name":"EquipmentType","hash":"sha256:77492fa4f96133c985d9c81eff2aea6ca53852b002fa122718aeae37e4fd5b06"}} as unknown as TypedDocumentString<EquipmentTypeQuery, EquipmentTypeQueryVariables>;
export const CreateEquipmentTypeDocument = {"__meta__":{"kind":"mutation","name":"CreateEquipmentType","hash":"sha256:804bf1016e1197c35723926d418334090e7ae5a6d68fe1e8a31517d60b87796f"}} as unknown as TypedDocumentString<CreateEquipmentTypeMutation, CreateEquipmentTypeMutationVariables>;
export const UpdateEquipmentTypeDocument = {"__meta__":{"kind":"mutation","name":"UpdateEquipmentType","hash":"sha256:05012e23f44106760ea0b9c4574517ce5095c9de4eddae711e8582b983f1d849"}} as unknown as TypedDocumentString<UpdateEquipmentTypeMutation, UpdateEquipmentTypeMutationVariables>;
export const PatchEquipmentTypeDocument = {"__meta__":{"kind":"mutation","name":"PatchEquipmentType","hash":"sha256:c5c81ee0f81421708ff81f3e3e2c42c61d42b40f5a5cee5993a778df6b36ca1d"}} as unknown as TypedDocumentString<PatchEquipmentTypeMutation, PatchEquipmentTypeMutationVariables>;
export const BulkUpdateEquipmentTypeStatusDocument = {"__meta__":{"kind":"mutation","name":"BulkUpdateEquipmentTypeStatus","hash":"sha256:a7e10803dec124d60c6f74fbce991a84c22d9e6dd45781e8bdc63de2ac10b9b3"}} as unknown as TypedDocumentString<BulkUpdateEquipmentTypeStatusMutation, BulkUpdateEquipmentTypeStatusMutationVariables>;
export const FiscalYearTableDocument = {"__meta__":{"kind":"query","name":"FiscalYearTable","hash":"sha256:b71efb13dab593e5639accbcbd43154e83864413588b315ef028c1029087a5b7"}} as unknown as TypedDocumentString<FiscalYearTableQuery, FiscalYearTableQueryVariables>;
export const FleetCodeTableDocument = {"__meta__":{"kind":"query","name":"FleetCodeTable","hash":"sha256:aa2917e7de6d4a5981909b298d418eeaa9d673b8eef470e6ed424667aa344b43"}} as unknown as TypedDocumentString<FleetCodeTableQuery, FleetCodeTableQueryVariables>;
export const FleetSafetyDocument = {"__meta__":{"kind":"query","name":"FleetSafety","hash":"sha256:6e9a229e80256b02928afbbb05c93d141f6955ef63bf3b19a4bfb228954c1334"}} as unknown as TypedDocumentString<FleetSafetyQuery, FleetSafetyQueryVariables>;
export const WorkerSafetyViolationsDocument = {"__meta__":{"kind":"query","name":"WorkerSafetyViolations","hash":"sha256:1859f2efa257fe49b8d1e3cee59b70277336765f19fa7e9c0c5bfe25006aa484"}} as unknown as TypedDocumentString<WorkerSafetyViolationsQuery, WorkerSafetyViolationsQueryVariables>;
export const RecordSafetyViolationDocument = {"__meta__":{"kind":"mutation","name":"RecordSafetyViolation","hash":"sha256:d027b3e1623e9042d553fd585d136c190cce82ef45b0986f1e6d15fb33244849"}} as unknown as TypedDocumentString<RecordSafetyViolationMutation, RecordSafetyViolationMutationVariables>;
export const UpdateSafetyViolationDocument = {"__meta__":{"kind":"mutation","name":"UpdateSafetyViolation","hash":"sha256:cae84810be9706e503d19974a6e82eb84ad55f67779583dfcdce0eecb6087b19"}} as unknown as TypedDocumentString<UpdateSafetyViolationMutation, UpdateSafetyViolationMutationVariables>;
export const DeleteSafetyViolationDocument = {"__meta__":{"kind":"mutation","name":"DeleteSafetyViolation","hash":"sha256:6799fc0a158227f6ff06996ab7ce4f2f8fc5386a8c20515dd973a0dc3ac1ead0"}} as unknown as TypedDocumentString<DeleteSafetyViolationMutation, DeleteSafetyViolationMutationVariables>;
export const FormulaTemplateTableDocument = {"__meta__":{"kind":"query","name":"FormulaTemplateTable","hash":"sha256:9fe86d3e1a8ccdb28fd5932ef322980f9ce4605286640620ee0f98833cbda924"}} as unknown as TypedDocumentString<FormulaTemplateTableQuery, FormulaTemplateTableQueryVariables>;
export const FuelCardTableDocument = {"__meta__":{"kind":"query","name":"FuelCardTable","hash":"sha256:fe01c96e14012ea096b5fb1f250fc042b4171ee9ab1264f1bd7d84c8a8f9c52b"}} as unknown as TypedDocumentString<FuelCardTableQuery, FuelCardTableQueryVariables>;
export const FuelCardDocument = {"__meta__":{"kind":"query","name":"FuelCard","hash":"sha256:b6577686f4d8e00dd0fd4f4a9ba529b2b9874da0e842888cbe96ad80334c0dfb"}} as unknown as TypedDocumentString<FuelCardQuery, FuelCardQueryVariables>;
export const CreateFuelCardDocument = {"__meta__":{"kind":"mutation","name":"CreateFuelCard","hash":"sha256:2403b71f501d02a0850060a7cad83dbae2bb9b2a0fb75ee686a03eeff90cf2a1"}} as unknown as TypedDocumentString<CreateFuelCardMutation, CreateFuelCardMutationVariables>;
export const UpdateFuelCardDocument = {"__meta__":{"kind":"mutation","name":"UpdateFuelCard","hash":"sha256:7b988de6a70d1106cd0aa908f5d9c95e0984a9f8193b669e7d3028db10f18a46"}} as unknown as TypedDocumentString<UpdateFuelCardMutation, UpdateFuelCardMutationVariables>;
export const CancelFuelCardDocument = {"__meta__":{"kind":"mutation","name":"CancelFuelCard","hash":"sha256:57ea8939185a798d8b009ea00c709e301b26e00a3630c851edd91af69c3e0ce2"}} as unknown as TypedDocumentString<CancelFuelCardMutation, CancelFuelCardMutationVariables>;
export const AssignFuelCardDocument = {"__meta__":{"kind":"mutation","name":"AssignFuelCard","hash":"sha256:4294d48aa2abbc1b3d399f93288d3259fd57d03440f6baf1c6288c30ffa7df7f"}} as unknown as TypedDocumentString<AssignFuelCardMutation, AssignFuelCardMutationVariables>;
export const SyncFuelCardFeedDocument = {"__meta__":{"kind":"mutation","name":"SyncFuelCardFeed","hash":"sha256:4ae581acc39b91d46ed28d1f27902d16277724e2bb9b983d6750fa2fc19e1134"}} as unknown as TypedDocumentString<SyncFuelCardFeedMutation, SyncFuelCardFeedMutationVariables>;
export const FuelPurchaseImportDocument = {"__meta__":{"kind":"query","name":"FuelPurchaseImport","hash":"sha256:1bb31f5cb19365fea83264e7ee2b831e7a6901d16a56cc1986b4a95a82718193"}} as unknown as TypedDocumentString<FuelPurchaseImportQuery, FuelPurchaseImportQueryVariables>;
export const FuelPurchaseImportRowsDocument = {"__meta__":{"kind":"query","name":"FuelPurchaseImportRows","hash":"sha256:76069fcd08f48d67bd04dd94e0702df4d9971c03c2ff5959ba8d3706aa9d5384"}} as unknown as TypedDocumentString<FuelPurchaseImportRowsQuery, FuelPurchaseImportRowsQueryVariables>;
export const FuelPurchaseImportTemplateDocument = {"__meta__":{"kind":"query","name":"FuelPurchaseImportTemplate","hash":"sha256:31cc70334d73006c9b344e26682020d505d5893d2b8b8253be40add52cb77d4d"}} as unknown as TypedDocumentString<FuelPurchaseImportTemplateQuery, FuelPurchaseImportTemplateQueryVariables>;
export const CreateFuelPurchaseImportDocument = {"__meta__":{"kind":"mutation","name":"CreateFuelPurchaseImport","hash":"sha256:e95a1303958d8995c8a2ec927507eb9891ed516ce966a5fc6301fd0b402c60ef"}} as unknown as TypedDocumentString<CreateFuelPurchaseImportMutation, CreateFuelPurchaseImportMutationVariables>;
export const StageFuelPurchaseImportDocument = {"__meta__":{"kind":"mutation","name":"StageFuelPurchaseImport","hash":"sha256:140a72958a0096aed76adac20a3728560f4d956101610e7e66db13bae9e37ed3"}} as unknown as TypedDocumentString<StageFuelPurchaseImportMutation, StageFuelPurchaseImportMutationVariables>;
export const CommitFuelPurchaseImportDocument = {"__meta__":{"kind":"mutation","name":"CommitFuelPurchaseImport","hash":"sha256:9406a447878e33fc2130176e7200ea5953a6d29fa828067cd14cb1a7aba97a60"}} as unknown as TypedDocumentString<CommitFuelPurchaseImportMutation, CommitFuelPurchaseImportMutationVariables>;
export const DiscardFuelPurchaseImportDocument = {"__meta__":{"kind":"mutation","name":"DiscardFuelPurchaseImport","hash":"sha256:88d15ca5892546f827423faae288ce44e51cb6ad35dcc5f710eac111661e7ba7"}} as unknown as TypedDocumentString<DiscardFuelPurchaseImportMutation, DiscardFuelPurchaseImportMutationVariables>;
export const ResolveFuelPurchaseImportRowsDocument = {"__meta__":{"kind":"mutation","name":"ResolveFuelPurchaseImportRows","hash":"sha256:893b4bba0a2264b92217328e52b5c509bb6c7b6dfc917ef39f1641420a8dd042"}} as unknown as TypedDocumentString<ResolveFuelPurchaseImportRowsMutation, ResolveFuelPurchaseImportRowsMutationVariables>;
export const FuelPurchaseImportTableDocument = {"__meta__":{"kind":"query","name":"FuelPurchaseImportTable","hash":"sha256:0d95f1db2e963ce5bc359e00b949d9e9ba9fd34165ce64ee980ee8ae5c6a9ef2"}} as unknown as TypedDocumentString<FuelPurchaseImportTableQuery, FuelPurchaseImportTableQueryVariables>;
export const FuelPurchaseTableDocument = {"__meta__":{"kind":"query","name":"FuelPurchaseTable","hash":"sha256:1b517bbfca7c783424cb89b37ff0e1b284c061714a3775c215929551249ca481"}} as unknown as TypedDocumentString<FuelPurchaseTableQuery, FuelPurchaseTableQueryVariables>;
export const FuelPurchaseDocument = {"__meta__":{"kind":"query","name":"FuelPurchase","hash":"sha256:d42b03441fabf22f72b2a52b37d9e38d3667ece2aa8a8a2ca48b4eeac53665a8"}} as unknown as TypedDocumentString<FuelPurchaseQuery, FuelPurchaseQueryVariables>;
export const CreateFuelPurchaseDocument = {"__meta__":{"kind":"mutation","name":"CreateFuelPurchase","hash":"sha256:ef463f75895788d035e4a316c19e1a029c9cdac9c08aceb92d5d7adb9137fed7"}} as unknown as TypedDocumentString<CreateFuelPurchaseMutation, CreateFuelPurchaseMutationVariables>;
export const UpdateFuelPurchaseDocument = {"__meta__":{"kind":"mutation","name":"UpdateFuelPurchase","hash":"sha256:9c66705e46769dae40d259e6a9ec6b70d50dad0ec0c2d96b4fd4d095a0d11ce3"}} as unknown as TypedDocumentString<UpdateFuelPurchaseMutation, UpdateFuelPurchaseMutationVariables>;
export const DeleteFuelPurchaseDocument = {"__meta__":{"kind":"mutation","name":"DeleteFuelPurchase","hash":"sha256:92b48dc6aa4689d56ec42b7b5c698aa79081711032f51ce315819719f6534355"}} as unknown as TypedDocumentString<DeleteFuelPurchaseMutation, DeleteFuelPurchaseMutationVariables>;
export const FuelIndexTableDocument = {"__meta__":{"kind":"query","name":"FuelIndexTable","hash":"sha256:a0b86e8015ebbd3614c2170d990849389d7542c3697779df9db1a33d99c5dc4f"}} as unknown as TypedDocumentString<FuelIndexTableQuery, FuelIndexTableQueryVariables>;
export const FuelSurchargeProgramTableDocument = {"__meta__":{"kind":"query","name":"FuelSurchargeProgramTable","hash":"sha256:b5f34d25e9ba03e9f90cf0266c8c8d59bef2ac3dd81311e620a280bb7bc981ec"}} as unknown as TypedDocumentString<FuelSurchargeProgramTableQuery, FuelSurchargeProgramTableQueryVariables>;
export const FuelSurchargeProgramDetailDocument = {"__meta__":{"kind":"query","name":"FuelSurchargeProgramDetail","hash":"sha256:668cfbb25c0bc4ff5599aa92de019bbabb47ff8af16e9eb7f9a53e59698287a9"}} as unknown as TypedDocumentString<FuelSurchargeProgramDetailQuery, FuelSurchargeProgramDetailQueryVariables>;
export const FuelDashboardDocument = {"__meta__":{"kind":"query","name":"FuelDashboard","hash":"sha256:eb8d81ae4caebe6f5986cc6160e44c6c083ea25481fb4f66856d980d1b92ad43"}} as unknown as TypedDocumentString<FuelDashboardQuery, FuelDashboardQueryVariables>;
export const FuelIndexPriceHistoryDocument = {"__meta__":{"kind":"query","name":"FuelIndexPriceHistory","hash":"sha256:2642b855e49ae205529b62bbdaa39e0c30f1476c31773ad47b63b0995208912c"}} as unknown as TypedDocumentString<FuelIndexPriceHistoryQuery, FuelIndexPriceHistoryQueryVariables>;
export const FuelProgramCurrentRatesDocument = {"__meta__":{"kind":"query","name":"FuelProgramCurrentRates","hash":"sha256:b6ded035e6dea96b6e99b7d7515aa56dab0a13682606b38c7374f980be5a15f0"}} as unknown as TypedDocumentString<FuelProgramCurrentRatesQuery, FuelProgramCurrentRatesQueryVariables>;
export const GenerateFuelSurchargeTableDocument = {"__meta__":{"kind":"query","name":"GenerateFuelSurchargeTable","hash":"sha256:a9038137e3854fa509ebd9e1ffaead13acfb73244483ca6a157c1676bab27dc1"}} as unknown as TypedDocumentString<GenerateFuelSurchargeTableQuery, GenerateFuelSurchargeTableQueryVariables>;
export const EiaSeriesOptionsDocument = {"__meta__":{"kind":"query","name":"EIASeriesOptions","hash":"sha256:aaf292fcd2d43d06a7f9694efaa21c08892e926da551fec3ee3747349ca474ee"}} as unknown as TypedDocumentString<EiaSeriesOptionsQuery, EiaSeriesOptionsQueryVariables>;
export const CreateFuelIndexDocument = {"__meta__":{"kind":"mutation","name":"CreateFuelIndex","hash":"sha256:c319990c4b5f3d40c1860bbed53325ee8253ff65822862a15d29417e5eb7c576"}} as unknown as TypedDocumentString<CreateFuelIndexMutation, CreateFuelIndexMutationVariables>;
export const UpdateFuelIndexDocument = {"__meta__":{"kind":"mutation","name":"UpdateFuelIndex","hash":"sha256:4f399ea1e3726ba5b46b72cceedf871a79d6c5491ce72140b22f0e3ab382cf5e"}} as unknown as TypedDocumentString<UpdateFuelIndexMutation, UpdateFuelIndexMutationVariables>;
export const DeleteFuelIndexDocument = {"__meta__":{"kind":"mutation","name":"DeleteFuelIndex","hash":"sha256:2755ccaf08a669e4ef6d4431518c9b88f6050aac4722e8317ec51c0ce60a5b5b"}} as unknown as TypedDocumentString<DeleteFuelIndexMutation, DeleteFuelIndexMutationVariables>;
export const AddFuelIndexPriceDocument = {"__meta__":{"kind":"mutation","name":"AddFuelIndexPrice","hash":"sha256:df875f75a8e1490c5ede2d25401a7f4532db9104475770a27e64e660735d7f51"}} as unknown as TypedDocumentString<AddFuelIndexPriceMutation, AddFuelIndexPriceMutationVariables>;
export const UpdateFuelIndexPriceDocument = {"__meta__":{"kind":"mutation","name":"UpdateFuelIndexPrice","hash":"sha256:edc0d10d2ca1db5814601c0fbd342451e82e378fe16c5543c3b815a412a1e255"}} as unknown as TypedDocumentString<UpdateFuelIndexPriceMutation, UpdateFuelIndexPriceMutationVariables>;
export const DeleteFuelIndexPriceDocument = {"__meta__":{"kind":"mutation","name":"DeleteFuelIndexPrice","hash":"sha256:8903bc47ee21d85ead19724a808087144465a1468b0323a72386d1ee5b627b20"}} as unknown as TypedDocumentString<DeleteFuelIndexPriceMutation, DeleteFuelIndexPriceMutationVariables>;
export const CreateFuelSurchargeProgramDocument = {"__meta__":{"kind":"mutation","name":"CreateFuelSurchargeProgram","hash":"sha256:c38694d28544cc10ca5710f466a1932d5a860625706c97f7fee1ea1257db38b0"}} as unknown as TypedDocumentString<CreateFuelSurchargeProgramMutation, CreateFuelSurchargeProgramMutationVariables>;
export const UpdateFuelSurchargeProgramDocument = {"__meta__":{"kind":"mutation","name":"UpdateFuelSurchargeProgram","hash":"sha256:2d15f5ba8184f8be6fdf1628b9d674c4c98f117fa8c36e0cabd847068c7cd45c"}} as unknown as TypedDocumentString<UpdateFuelSurchargeProgramMutation, UpdateFuelSurchargeProgramMutationVariables>;
export const DeleteFuelSurchargeProgramDocument = {"__meta__":{"kind":"mutation","name":"DeleteFuelSurchargeProgram","hash":"sha256:29699115bac5fef098c018951b45e9a68e8f1ab0fd98c45f23d69cfc60885a4b"}} as unknown as TypedDocumentString<DeleteFuelSurchargeProgramMutation, DeleteFuelSurchargeProgramMutationVariables>;
export const HazardousMaterialTableDocument = {"__meta__":{"kind":"query","name":"HazardousMaterialTable","hash":"sha256:9be076a80d56dbcc7fa1ee6f0cdb7c67de04d63c2bbda28e685f21bd190afa5c"}} as unknown as TypedDocumentString<HazardousMaterialTableQuery, HazardousMaterialTableQueryVariables>;
export const HazmatSegregationRuleTableDocument = {"__meta__":{"kind":"query","name":"HazmatSegregationRuleTable","hash":"sha256:8a556213d0fc4aa8da4e9bcfbed7b07f3fd1cc94c9d2e2231e806516c2b6f1b9"}} as unknown as TypedDocumentString<HazmatSegregationRuleTableQuery, HazmatSegregationRuleTableQueryVariables>;
export const HoldReasonTableDocument = {"__meta__":{"kind":"query","name":"HoldReasonTable","hash":"sha256:b68375fa81667eca3b794d6327160d9216180ef9f0aec9bea33dd529ce119231"}} as unknown as TypedDocumentString<HoldReasonTableQuery, HoldReasonTableQueryVariables>;
export const UpdateHomeLayoutDocument = {"__meta__":{"kind":"mutation","name":"UpdateHomeLayout","hash":"sha256:d5c57c1e547d5c9b9a1b393c73fa835c9cab7da77e8bd9d4f070a55faa37d430"}} as unknown as TypedDocumentString<UpdateHomeLayoutMutation, UpdateHomeLayoutMutationVariables>;
export const ResetHomeLayoutDocument = {"__meta__":{"kind":"mutation","name":"ResetHomeLayout","hash":"sha256:77165d1c06d424dbbedcb1c0e1c9ef91913e538faac9d5a11d24a9d5668cf3ed"}} as unknown as TypedDocumentString<ResetHomeLayoutMutation, ResetHomeLayoutMutationVariables>;
export const CreateHomeLayoutPresetDocument = {"__meta__":{"kind":"mutation","name":"CreateHomeLayoutPreset","hash":"sha256:d75d1fcd085631b841bf464bf9004ea6eaa56436b8ddc03ddda5d7dd85d5e7b1"}} as unknown as TypedDocumentString<CreateHomeLayoutPresetMutation, CreateHomeLayoutPresetMutationVariables>;
export const UpdateHomeLayoutPresetDocument = {"__meta__":{"kind":"mutation","name":"UpdateHomeLayoutPreset","hash":"sha256:305586d1cca594b06d305d8f7f79580a7ec087a587014916f98e3c5f7239d2f9"}} as unknown as TypedDocumentString<UpdateHomeLayoutPresetMutation, UpdateHomeLayoutPresetMutationVariables>;
export const DeleteHomeLayoutPresetDocument = {"__meta__":{"kind":"mutation","name":"DeleteHomeLayoutPreset","hash":"sha256:9e1e4363c27b26dfa7a016bf82fa31f1b06ede585f4e79943094ddd3610b973d"}} as unknown as TypedDocumentString<DeleteHomeLayoutPresetMutation, DeleteHomeLayoutPresetMutationVariables>;
export const HomeLayoutDocument = {"__meta__":{"kind":"query","name":"HomeLayout","hash":"sha256:54555d91b387c7e74269ce62063e35106cd2b8291936c35a3d0642268410856f"}} as unknown as TypedDocumentString<HomeLayoutQuery, HomeLayoutQueryVariables>;
export const HomeWidgetCatalogDocument = {"__meta__":{"kind":"query","name":"HomeWidgetCatalog","hash":"sha256:5e16e534f1d96c3cb75fff356ae936c19faf5985d49593b3d59491d02b777ba7"}} as unknown as TypedDocumentString<HomeWidgetCatalogQuery, HomeWidgetCatalogQueryVariables>;
export const HomeLayoutPresetsDocument = {"__meta__":{"kind":"query","name":"HomeLayoutPresets","hash":"sha256:4e593b5900c91c160d041d71d66f1246ed5f1962ead7fe0e6f4f217c00427400"}} as unknown as TypedDocumentString<HomeLayoutPresetsQuery, HomeLayoutPresetsQueryVariables>;
export const HomeLayoutPresetDocument = {"__meta__":{"kind":"query","name":"HomeLayoutPreset","hash":"sha256:6ab7e3b94462884c03c1835ebff74e5d06d7f94b1ae69512da5254714985fe63"}} as unknown as TypedDocumentString<HomeLayoutPresetQuery, HomeLayoutPresetQueryVariables>;
export const HomeLayoutPreviewDocument = {"__meta__":{"kind":"query","name":"HomeLayoutPreview","hash":"sha256:1cf255a093a992d50b4a00e4b558a23190b43df72bb4ff298821a5b9180946df"}} as unknown as TypedDocumentString<HomeLayoutPreviewQuery, HomeLayoutPreviewQueryVariables>;
export const IftaMileageEntryTableDocument = {"__meta__":{"kind":"query","name":"IftaMileageEntryTable","hash":"sha256:66058c6e3361c2a41ad513f392898d2614472a9a0c4155073ccb377c4b2668c5"}} as unknown as TypedDocumentString<IftaMileageEntryTableQuery, IftaMileageEntryTableQueryVariables>;
export const IftaMileageEntryDocument = {"__meta__":{"kind":"query","name":"IftaMileageEntry","hash":"sha256:ae12f876fe7b51bb56cec67fb0992841b7d64b926e75a539ccdfdef1e594c4c7"}} as unknown as TypedDocumentString<IftaMileageEntryQuery, IftaMileageEntryQueryVariables>;
export const CreateIftaMileageEntryDocument = {"__meta__":{"kind":"mutation","name":"CreateIftaMileageEntry","hash":"sha256:78ff7b8f4d3dfa83100e6d8bd140384bc601c69d7628004b9fc8bc90f20c1660"}} as unknown as TypedDocumentString<CreateIftaMileageEntryMutation, CreateIftaMileageEntryMutationVariables>;
export const UpdateIftaMileageEntryDocument = {"__meta__":{"kind":"mutation","name":"UpdateIftaMileageEntry","hash":"sha256:db3417161c4d8d67d296d7d97e8ee8e50641e3e4399c022a38960338f4c41b8a"}} as unknown as TypedDocumentString<UpdateIftaMileageEntryMutation, UpdateIftaMileageEntryMutationVariables>;
export const DeleteIftaMileageEntryDocument = {"__meta__":{"kind":"mutation","name":"DeleteIftaMileageEntry","hash":"sha256:a50349f07575b1af39b4754a08c6edb7785fc426ee1cf766eb11b225c68e4213"}} as unknown as TypedDocumentString<DeleteIftaMileageEntryMutation, DeleteIftaMileageEntryMutationVariables>;
export const IftaJurisdictionsDocument = {"__meta__":{"kind":"query","name":"IftaJurisdictions","hash":"sha256:36e4d33c7597c2be3de87f817943565acf932a13cf8b63e8b31a81ab198d622d"}} as unknown as TypedDocumentString<IftaJurisdictionsQuery, IftaJurisdictionsQueryVariables>;
export const IftaReturnForPeriodDocument = {"__meta__":{"kind":"query","name":"IftaReturnForPeriod","hash":"sha256:0eeba98dd4b248b1e1270a038d6ea8ebf0b6e9fc2c8eb0c383fd83f5467aa006"}} as unknown as TypedDocumentString<IftaReturnForPeriodQuery, IftaReturnForPeriodQueryVariables>;
export const IftaReturnDocument = {"__meta__":{"kind":"query","name":"IftaReturn","hash":"sha256:13ccd49d841eeb5e6362844e3d0e6c5d242d1a8ad2f3b62b57366810d3663ac9"}} as unknown as TypedDocumentString<IftaReturnQuery, IftaReturnQueryVariables>;
export const IftaReturnTableDocument = {"__meta__":{"kind":"query","name":"IftaReturnTable","hash":"sha256:8f9c89533e5ba9202016848c4002d479c483ecd941dd366ae23d9b3340514584"}} as unknown as TypedDocumentString<IftaReturnTableQuery, IftaReturnTableQueryVariables>;
export const IftaCurrentPeriodDocument = {"__meta__":{"kind":"query","name":"IftaCurrentPeriod","hash":"sha256:65c7343838c2470ef9556bd4dd4020b769865da0d7f214079ca7573a5c9b80f3"}} as unknown as TypedDocumentString<IftaCurrentPeriodQuery, IftaCurrentPeriodQueryVariables>;
export const IftaPeriodDocument = {"__meta__":{"kind":"query","name":"IftaPeriod","hash":"sha256:5233eec4be49a9b867ac2b5ba34c80d373cb002420627cd092080cec01da9057"}} as unknown as TypedDocumentString<IftaPeriodQuery, IftaPeriodQueryVariables>;
export const GenerateIftaReturnDocument = {"__meta__":{"kind":"mutation","name":"GenerateIftaReturn","hash":"sha256:18a9184c4e27e604547100180762480fe5b3ddd0a3441ce483c2a823c12466f2"}} as unknown as TypedDocumentString<GenerateIftaReturnMutation, GenerateIftaReturnMutationVariables>;
export const RecomputeIftaReturnDocument = {"__meta__":{"kind":"mutation","name":"RecomputeIftaReturn","hash":"sha256:5ad33bca01818ff7b9ea259638a7342923b673d0ab22ec9cecc849844edb189c"}} as unknown as TypedDocumentString<RecomputeIftaReturnMutation, RecomputeIftaReturnMutationVariables>;
export const FinalizeIftaReturnDocument = {"__meta__":{"kind":"mutation","name":"FinalizeIftaReturn","hash":"sha256:8d0095fa44207f65499ba0e281e7dfb4cc21f972cd17339b5369bdfe8634833d"}} as unknown as TypedDocumentString<FinalizeIftaReturnMutation, FinalizeIftaReturnMutationVariables>;
export const ReopenIftaReturnDocument = {"__meta__":{"kind":"mutation","name":"ReopenIftaReturn","hash":"sha256:dc138f18d68ef10805888857fb9537b772bfca78db588d2bbe00ccdb199dc7b8"}} as unknown as TypedDocumentString<ReopenIftaReturnMutation, ReopenIftaReturnMutationVariables>;
export const MarkIftaReturnFiledDocument = {"__meta__":{"kind":"mutation","name":"MarkIftaReturnFiled","hash":"sha256:873c922d30e34be17ce51a19c034596c9b4077216fa99b7ffbf11961b30e728f"}} as unknown as TypedDocumentString<MarkIftaReturnFiledMutation, MarkIftaReturnFiledMutationVariables>;
export const AmendIftaReturnDocument = {"__meta__":{"kind":"mutation","name":"AmendIftaReturn","hash":"sha256:12ba63b9267166c4c24236ed8793d38c7deb0dcf45f28a55544c007d275765b1"}} as unknown as TypedDocumentString<AmendIftaReturnMutation, AmendIftaReturnMutationVariables>;
export const DeleteIftaReturnDocument = {"__meta__":{"kind":"mutation","name":"DeleteIftaReturn","hash":"sha256:7f7310c3528509f8c259bed8da8483f566528b3753f7700a2622038436c48885"}} as unknown as TypedDocumentString<DeleteIftaReturnMutation, DeleteIftaReturnMutationVariables>;
export const BackfillJurisdictionMilesDocument = {"__meta__":{"kind":"mutation","name":"BackfillJurisdictionMiles","hash":"sha256:d0b29a09ad751cb2ea77772d7acc10cfc0ae1875f9b5cdfa70f4dd9991ca33e1"}} as unknown as TypedDocumentString<BackfillJurisdictionMilesMutation, BackfillJurisdictionMilesMutationVariables>;
export const IftaTaxRateTableDocument = {"__meta__":{"kind":"query","name":"IftaTaxRateTable","hash":"sha256:75d9876e1746a57e7069435584d0c2968945fa0273152ed0c5b0d4ac7c2ca8c3"}} as unknown as TypedDocumentString<IftaTaxRateTableQuery, IftaTaxRateTableQueryVariables>;
export const UpsertIftaTaxRatesDocument = {"__meta__":{"kind":"mutation","name":"UpsertIftaTaxRates","hash":"sha256:40687ccd074f12591f177689ac65f9ebc0c02b96ee78cbd3a20c61f549c6f28b"}} as unknown as TypedDocumentString<UpsertIftaTaxRatesMutation, UpsertIftaTaxRatesMutationVariables>;
export const DeleteIftaTaxRateDocument = {"__meta__":{"kind":"mutation","name":"DeleteIftaTaxRate","hash":"sha256:3ea06bea9c4fe480fb22642ac0608f800ac5ce37559751b23103bc25274b2755"}} as unknown as TypedDocumentString<DeleteIftaTaxRateMutation, DeleteIftaTaxRateMutationVariables>;
export const InvoiceTableDocument = {"__meta__":{"kind":"query","name":"InvoiceTable","hash":"sha256:63d68f9764990f48180ca1e6c938cec819f504ae13f959b9b1abd06dffdd4609"}} as unknown as TypedDocumentString<InvoiceTableQuery, InvoiceTableQueryVariables>;
export const JournalEntryDetailDocument = {"__meta__":{"kind":"query","name":"JournalEntryDetail","hash":"sha256:9115c76311ea912a3c9abf400bf6fc66c811ec2227b1500769f88a829782a646"}} as unknown as TypedDocumentString<JournalEntryDetailQuery, JournalEntryDetailQueryVariables>;
export const JournalSourceByObjectDocument = {"__meta__":{"kind":"query","name":"JournalSourceByObject","hash":"sha256:9fc6924e799999cbc0d1e413752b76e244eb6f3d8f0cef4eec55f1c5d2b785af"}} as unknown as TypedDocumentString<JournalSourceByObjectQuery, JournalSourceByObjectQueryVariables>;
export const JournalEntriesBySourceDocument = {"__meta__":{"kind":"query","name":"JournalEntriesBySource","hash":"sha256:fda3eefb446f90d8e933f63b8db868bf7045f0a841ecafdd013ff3240df17e1f"}} as unknown as TypedDocumentString<JournalEntriesBySourceQuery, JournalEntriesBySourceQueryVariables>;
export const JournalReversalTableDocument = {"__meta__":{"kind":"query","name":"JournalReversalTable","hash":"sha256:4cfe99ebcf84db5bd24a4b69d10a9faec9e060b9594c5efce8d8f4445a06afdf"}} as unknown as TypedDocumentString<JournalReversalTableQuery, JournalReversalTableQueryVariables>;
export const JurisdictionRuleOverrideTableDocument = {"__meta__":{"kind":"query","name":"JurisdictionRuleOverrideTable","hash":"sha256:308688c1ff8677796acd9575f32c4ad0d890bbee12abccbf63e5c3ec5804c11e"}} as unknown as TypedDocumentString<JurisdictionRuleOverrideTableQuery, JurisdictionRuleOverrideTableQueryVariables>;
export const JurisdictionRuleTableDocument = {"__meta__":{"kind":"query","name":"JurisdictionRuleTable","hash":"sha256:d4c8d8690aaf40f0f90a9fb47da5503d91df2371c1a129a4f12eae53e42f09e8"}} as unknown as TypedDocumentString<JurisdictionRuleTableQuery, JurisdictionRuleTableQueryVariables>;
export const LocationCategoryTableDocument = {"__meta__":{"kind":"query","name":"LocationCategoryTable","hash":"sha256:5ca16e1292c673eceb32a9d1189fe4af156f3ffa10b7088854e40761fbf5e1bf"}} as unknown as TypedDocumentString<LocationCategoryTableQuery, LocationCategoryTableQueryVariables>;
export const LocationTableDocument = {"__meta__":{"kind":"query","name":"LocationTable","hash":"sha256:db0953dee7cb9ece4aa05feb18d26f2986a6636cc3ad0b168e41c7ece0e7bd43"}} as unknown as TypedDocumentString<LocationTableQuery, LocationTableQueryVariables>;
export const ManualJournalTableDocument = {"__meta__":{"kind":"query","name":"ManualJournalTable","hash":"sha256:8cd46bc08994ea9aee268bf5b08a578d5a39f6d8628919ae255b58c350fabebf"}} as unknown as TypedDocumentString<ManualJournalTableQuery, ManualJournalTableQueryVariables>;
export const NotificationListDocument = {"__meta__":{"kind":"query","name":"NotificationList","hash":"sha256:d495edf2331e2fb69b5bfd7c646882ba80f82d2b859e740d86606ece6b83aac5"}} as unknown as TypedDocumentString<NotificationListQuery, NotificationListQueryVariables>;
export const NotificationUnreadCountDocument = {"__meta__":{"kind":"query","name":"NotificationUnreadCount","hash":"sha256:ed4fa686e9641b77e14b47a1c93d30e473f479f8cb960bc62def2000568e84a7"}} as unknown as TypedDocumentString<NotificationUnreadCountQuery, NotificationUnreadCountQueryVariables>;
export const MyNotificationListDocument = {"__meta__":{"kind":"query","name":"MyNotificationList","hash":"sha256:f98a49121bd13b35e14f592c2847ecf4ff8310f03fd345d6bb6b65dfb8977679"}} as unknown as TypedDocumentString<MyNotificationListQuery, MyNotificationListQueryVariables>;
export const MyNotificationUnreadCountDocument = {"__meta__":{"kind":"query","name":"MyNotificationUnreadCount","hash":"sha256:ad2ceb6f0515af73bb851c124af38bc7fcef461bdcb62803ce866504fb1a6751"}} as unknown as TypedDocumentString<MyNotificationUnreadCountQuery, MyNotificationUnreadCountQueryVariables>;
export const MarkAllMyNotificationsReadDocument = {"__meta__":{"kind":"mutation","name":"MarkAllMyNotificationsRead","hash":"sha256:e9902bd6a7ebb19a5be05f820482fe885b4ed3baf8ec9c159b054cb4448459f7"}} as unknown as TypedDocumentString<MarkAllMyNotificationsReadMutation, MarkAllMyNotificationsReadMutationVariables>;
export const MarkMyNotificationsReadDocument = {"__meta__":{"kind":"mutation","name":"MarkMyNotificationsRead","hash":"sha256:729524debcb5ff3140bec03e8447c4bdd6929cbaae1c1c9940753533eb3da14b"}} as unknown as TypedDocumentString<MarkMyNotificationsReadMutation, MarkMyNotificationsReadMutationVariables>;
export const MarkMyNotificationsUnreadDocument = {"__meta__":{"kind":"mutation","name":"MarkMyNotificationsUnread","hash":"sha256:cf8016014675f422deed28de48142e48a8001d4eb8981be7cd93313d81d396d3"}} as unknown as TypedDocumentString<MarkMyNotificationsUnreadMutation, MarkMyNotificationsUnreadMutationVariables>;
export const DismissMyNotificationsDocument = {"__meta__":{"kind":"mutation","name":"DismissMyNotifications","hash":"sha256:6cae1945265d9c230b3cff31c4045b71cc7332491be831de11ed0093396935ee"}} as unknown as TypedDocumentString<DismissMyNotificationsMutation, DismissMyNotificationsMutationVariables>;
export const RestoreMyNotificationsDocument = {"__meta__":{"kind":"mutation","name":"RestoreMyNotifications","hash":"sha256:371399164c8a2f505fa278c9d27180d252d712e6c89b0d5b498e0c9f7429b6c9"}} as unknown as TypedDocumentString<RestoreMyNotificationsMutation, RestoreMyNotificationsMutationVariables>;
export const MarkNotificationsReadDocument = {"__meta__":{"kind":"mutation","name":"MarkNotificationsRead","hash":"sha256:1a766cf4ea3e134b35e5fa8ec5b904da700ab13ff3d61af7158809b586e6fe94"}} as unknown as TypedDocumentString<MarkNotificationsReadMutation, MarkNotificationsReadMutationVariables>;
export const MarkNotificationsUnreadDocument = {"__meta__":{"kind":"mutation","name":"MarkNotificationsUnread","hash":"sha256:4623f51d1c45a298af41a975f66d19897b98c9c1527f03162eefac1aef651ca2"}} as unknown as TypedDocumentString<MarkNotificationsUnreadMutation, MarkNotificationsUnreadMutationVariables>;
export const MarkAllNotificationsReadDocument = {"__meta__":{"kind":"mutation","name":"MarkAllNotificationsRead","hash":"sha256:e919497b911d73638f8329785ecb0b4b48a247bb6d037bf89b2a498c5bca336d"}} as unknown as TypedDocumentString<MarkAllNotificationsReadMutation, MarkAllNotificationsReadMutationVariables>;
export const DismissNotificationsDocument = {"__meta__":{"kind":"mutation","name":"DismissNotifications","hash":"sha256:762abd6aba103c349367b7834a0e909dd2e06b9d5c1a33f71a4467431db83d50"}} as unknown as TypedDocumentString<DismissNotificationsMutation, DismissNotificationsMutationVariables>;
export const RestoreNotificationsDocument = {"__meta__":{"kind":"mutation","name":"RestoreNotifications","hash":"sha256:e97ca2a47ac7291064a1651afaf8807310b842b010cc23d5b383cab58018d6e1"}} as unknown as TypedDocumentString<RestoreNotificationsMutation, RestoreNotificationsMutationVariables>;
export const OrderDetailDocument = {"__meta__":{"kind":"query","name":"OrderDetail","hash":"sha256:7f3565d2e4b7025b6b94522b2f084b22e66738f934977fcf89f7921873655f7b"}} as unknown as TypedDocumentString<OrderDetailQuery, OrderDetailQueryVariables>;
export const AttachOrderShipmentsDocument = {"__meta__":{"kind":"mutation","name":"AttachOrderShipments","hash":"sha256:c5dd0f391421cd1c7def4a849abaf9630283cc0b9b5c4ad83164f677b79273a9"}} as unknown as TypedDocumentString<AttachOrderShipmentsMutation, AttachOrderShipmentsMutationVariables>;
export const DetachOrderShipmentDocument = {"__meta__":{"kind":"mutation","name":"DetachOrderShipment","hash":"sha256:5a7b3fa35274ee455c2c6c8eb92842cbf663284cf5639cbc0c4dea9d7350984c"}} as unknown as TypedDocumentString<DetachOrderShipmentMutation, DetachOrderShipmentMutationVariables>;
export const CreateInvoiceFromOrderDocument = {"__meta__":{"kind":"mutation","name":"CreateInvoiceFromOrder","hash":"sha256:775335abc4ecdb1bd747990a951043d00c3822e84b8937e46c301ad1d79bb30f"}} as unknown as TypedDocumentString<CreateInvoiceFromOrderMutation, CreateInvoiceFromOrderMutationVariables>;
export const CreateInvoiceFromShipmentsDocument = {"__meta__":{"kind":"mutation","name":"CreateInvoiceFromShipments","hash":"sha256:d4c1361e483de462a2dbadc15deb482f9aab682208e5b391e6582fcb701aed2f"}} as unknown as TypedDocumentString<CreateInvoiceFromShipmentsMutation, CreateInvoiceFromShipmentsMutationVariables>;
export const CreateOrderDocument = {"__meta__":{"kind":"mutation","name":"CreateOrder","hash":"sha256:7fb5e40596d163d5d39851904b98476de863ef34c103de223cdcdef3ee097041"}} as unknown as TypedDocumentString<CreateOrderMutation, CreateOrderMutationVariables>;
export const UpdateOrderDocument = {"__meta__":{"kind":"mutation","name":"UpdateOrder","hash":"sha256:96308fccacc82642fbb51c915e2fa69d53fa1d7a4a5eb982d352dc4c0cdeb409"}} as unknown as TypedDocumentString<UpdateOrderMutation, UpdateOrderMutationVariables>;
export const AddOrderChargeDocument = {"__meta__":{"kind":"mutation","name":"AddOrderCharge","hash":"sha256:b1a4b7643a5a90dc1d54ec27959c4dbc92d94fa7c45274d378e16fcabceabf36"}} as unknown as TypedDocumentString<AddOrderChargeMutation, AddOrderChargeMutationVariables>;
export const UpdateOrderChargeDocument = {"__meta__":{"kind":"mutation","name":"UpdateOrderCharge","hash":"sha256:a92d821b04622cdf8267c6351a990835ddc95b0b98660680cab62b31dd72e495"}} as unknown as TypedDocumentString<UpdateOrderChargeMutation, UpdateOrderChargeMutationVariables>;
export const RemoveOrderChargeDocument = {"__meta__":{"kind":"mutation","name":"RemoveOrderCharge","hash":"sha256:a75bec74d0ee9039320bcfcc64440bf120258d7bed35a4fce23b2e0aeb9f4779"}} as unknown as TypedDocumentString<RemoveOrderChargeMutation, RemoveOrderChargeMutationVariables>;
export const CloseOrderDocument = {"__meta__":{"kind":"mutation","name":"CloseOrder","hash":"sha256:29e28b70b87c2b56b742887aa61c4e902824411344b2406cb079fb5b235cb013"}} as unknown as TypedDocumentString<CloseOrderMutation, CloseOrderMutationVariables>;
export const CancelOrderDocument = {"__meta__":{"kind":"mutation","name":"CancelOrder","hash":"sha256:c2d8a6c844a02e81f0c6a42222ab9146f32e20dc17c14aa067b711e85838ea18"}} as unknown as TypedDocumentString<CancelOrderMutation, CancelOrderMutationVariables>;
export const OrderTableDocument = {"__meta__":{"kind":"query","name":"OrderTable","hash":"sha256:da5fb471366bc747111feec70ba55fc514d137062a3b58f107ab9ea8ec017c1a"}} as unknown as TypedDocumentString<OrderTableQuery, OrderTableQueryVariables>;
export const OrgHolidaysDocument = {"__meta__":{"kind":"query","name":"OrgHolidays","hash":"sha256:85f3e7cef0f9aba50b97cef2fa4d323c77d0ac315e6c68a70bb92ae0667f4583"}} as unknown as TypedDocumentString<OrgHolidaysQuery, OrgHolidaysQueryVariables>;
export const CreateOrgHolidayDocument = {"__meta__":{"kind":"mutation","name":"CreateOrgHoliday","hash":"sha256:bbfda70d2124fa386a6edac1bdb870addc8c4c28d01f6faf77a8dbd3302dc839"}} as unknown as TypedDocumentString<CreateOrgHolidayMutation, CreateOrgHolidayMutationVariables>;
export const UpdateOrgHolidayDocument = {"__meta__":{"kind":"mutation","name":"UpdateOrgHoliday","hash":"sha256:66bf22b62518d8ed2465729f42928f3500a72588e2c43587571d24e08ad21cd9"}} as unknown as TypedDocumentString<UpdateOrgHolidayMutation, UpdateOrgHolidayMutationVariables>;
export const DeleteOrgHolidayDocument = {"__meta__":{"kind":"mutation","name":"DeleteOrgHoliday","hash":"sha256:77755adf35c13425eb649644016f482b87682416ff74ad6ba740b9e8f4d366c6"}} as unknown as TypedDocumentString<DeleteOrgHolidayMutation, DeleteOrgHolidayMutationVariables>;
export const JobPositionsDocument = {"__meta__":{"kind":"query","name":"JobPositions","hash":"sha256:297d7dc041761e5f52aa7576b606ee11dce20d05a8e70eb8e9040485c0b347cc"}} as unknown as TypedDocumentString<JobPositionsQuery, JobPositionsQueryVariables>;
export const HeadcountDocument = {"__meta__":{"kind":"query","name":"Headcount","hash":"sha256:a465fd40460ccbc7d4a412a1e0d3f78101d54f436d31b3e6b512a52abaf478e8"}} as unknown as TypedDocumentString<HeadcountQuery, HeadcountQueryVariables>;
export const JobPositionHoldersDocument = {"__meta__":{"kind":"query","name":"JobPositionHolders","hash":"sha256:52015b6b422ba0f5f168cd2a5deb4df591bb631ab002d3cf65c7f4c737f2361c"}} as unknown as TypedDocumentString<JobPositionHoldersQuery, JobPositionHoldersQueryVariables>;
export const MyTeamDocument = {"__meta__":{"kind":"query","name":"MyTeam","hash":"sha256:98848b2d79664809fa2139d3037d6cacb86c588cf11ee5ad303fd919f6e47c9a"}} as unknown as TypedDocumentString<MyTeamQuery, MyTeamQueryVariables>;
export const ApprovalDelegationsDocument = {"__meta__":{"kind":"query","name":"ApprovalDelegations","hash":"sha256:904abb7b074d030255e95e0961b43262bc441bb4025ac4acaf8dfd7c696f5ed4"}} as unknown as TypedDocumentString<ApprovalDelegationsQuery, ApprovalDelegationsQueryVariables>;
export const AssignWorkerPositionDocument = {"__meta__":{"kind":"mutation","name":"AssignWorkerPosition","hash":"sha256:522e27140eb4575c291273126ab4bc35cea904416f0f592f66a09e56d51f3694"}} as unknown as TypedDocumentString<AssignWorkerPositionMutation, AssignWorkerPositionMutationVariables>;
export const AssignUserPositionDocument = {"__meta__":{"kind":"mutation","name":"AssignUserPosition","hash":"sha256:f5275f25d13601ad7e7222ccacde02928b055315eb169b02de357eb43cdbc965"}} as unknown as TypedDocumentString<AssignUserPositionMutation, AssignUserPositionMutationVariables>;
export const CreateJobPositionDocument = {"__meta__":{"kind":"mutation","name":"CreateJobPosition","hash":"sha256:ea76cbe061c46ef447cb9a471bca62fae67aed5262380d631370a035b6d49ffa"}} as unknown as TypedDocumentString<CreateJobPositionMutation, CreateJobPositionMutationVariables>;
export const UpdateJobPositionDocument = {"__meta__":{"kind":"mutation","name":"UpdateJobPosition","hash":"sha256:72d0dd1e3604bbd3563c7ce4d292b7291f4b46d4b06b324b4268671d1e3e7dda"}} as unknown as TypedDocumentString<UpdateJobPositionMutation, UpdateJobPositionMutationVariables>;
export const DelegateApprovalDocument = {"__meta__":{"kind":"mutation","name":"DelegateApproval","hash":"sha256:93460a1c55714740ee8c7f8b74b8479a9c2341f6d6b489187a7721b71ae30e56"}} as unknown as TypedDocumentString<DelegateApprovalMutation, DelegateApprovalMutationVariables>;
export const RevokeApprovalDelegationDocument = {"__meta__":{"kind":"mutation","name":"RevokeApprovalDelegation","hash":"sha256:64aabe091bc3aa34ea63d08b5db39b589b72f35a8d174af29a317121005e4bb5"}} as unknown as TypedDocumentString<RevokeApprovalDelegationMutation, RevokeApprovalDelegationMutationVariables>;
export const OrganizationSettingsDocument = {"__meta__":{"kind":"query","name":"OrganizationSettings","hash":"sha256:f59e843fddaab74c31d9f3aa0a606e4b81cc2d14a917c3d66acea23683149f6e"}} as unknown as TypedDocumentString<OrganizationSettingsQuery, OrganizationSettingsQueryVariables>;
export const UpdateOrganizationSettingsDocument = {"__meta__":{"kind":"mutation","name":"UpdateOrganizationSettings","hash":"sha256:2570700a28884d232092d9f71c54fedff5dd1a7323568e57a8c4b571869f400e"}} as unknown as TypedDocumentString<UpdateOrganizationSettingsMutation, UpdateOrganizationSettingsMutationVariables>;
export const PerformanceReviewTemplateTableDocument = {"__meta__":{"kind":"query","name":"PerformanceReviewTemplateTable","hash":"sha256:bb9322d30da08406462040a79f5c2fc9cbf85637bb8ea6558434f5db5997ef7f"}} as unknown as TypedDocumentString<PerformanceReviewTemplateTableQuery, PerformanceReviewTemplateTableQueryVariables>;
export const ActivePerformanceReviewTemplatesDocument = {"__meta__":{"kind":"query","name":"ActivePerformanceReviewTemplates","hash":"sha256:11a93324897a2f419cf086637b8dea480b191919c61569940368ef4ae1398bfb"}} as unknown as TypedDocumentString<ActivePerformanceReviewTemplatesQuery, ActivePerformanceReviewTemplatesQueryVariables>;
export const WorkerPerformanceReviewsDocument = {"__meta__":{"kind":"query","name":"WorkerPerformanceReviews","hash":"sha256:deb16e65c7cea1ea36de973cfdc16e6022d40e4ec4cadf2b308b721fde15e0a4"}} as unknown as TypedDocumentString<WorkerPerformanceReviewsQuery, WorkerPerformanceReviewsQueryVariables>;
export const CreatePerformanceReviewTemplateDocument = {"__meta__":{"kind":"mutation","name":"CreatePerformanceReviewTemplate","hash":"sha256:cfdb98685da0e678aeb30e13b7d3797a24c89fe3327cab810c280038863e99c4"}} as unknown as TypedDocumentString<CreatePerformanceReviewTemplateMutation, CreatePerformanceReviewTemplateMutationVariables>;
export const UpdatePerformanceReviewTemplateDocument = {"__meta__":{"kind":"mutation","name":"UpdatePerformanceReviewTemplate","hash":"sha256:ebf24392f5495b4d25cc8d3d0875a512a2f07c37b9add8c8cbd5b51f1145c2d5"}} as unknown as TypedDocumentString<UpdatePerformanceReviewTemplateMutation, UpdatePerformanceReviewTemplateMutationVariables>;
export const ArchivePerformanceReviewTemplateDocument = {"__meta__":{"kind":"mutation","name":"ArchivePerformanceReviewTemplate","hash":"sha256:ed2fcbfba7246830a03a670ea1a706a71aae164322659bf7cddff67c344fe530"}} as unknown as TypedDocumentString<ArchivePerformanceReviewTemplateMutation, ArchivePerformanceReviewTemplateMutationVariables>;
export const RestorePerformanceReviewTemplateDocument = {"__meta__":{"kind":"mutation","name":"RestorePerformanceReviewTemplate","hash":"sha256:f551a416c8b75806b5e979dfae9ee0f73fc1072ae6aa2e03ac3c297871787f5f"}} as unknown as TypedDocumentString<RestorePerformanceReviewTemplateMutation, RestorePerformanceReviewTemplateMutationVariables>;
export const CreatePerformanceReviewDocument = {"__meta__":{"kind":"mutation","name":"CreatePerformanceReview","hash":"sha256:dc6cd880dcf90dd76fad702f603721bf62a36d7d0bb414ecde69e4dab6050e5e"}} as unknown as TypedDocumentString<CreatePerformanceReviewMutation, CreatePerformanceReviewMutationVariables>;
export const UpdatePerformanceReviewDocument = {"__meta__":{"kind":"mutation","name":"UpdatePerformanceReview","hash":"sha256:78f11d4291124375b9fb8f9ec0307218e84f0e1b9ea2bfa7102a575c69eacb19"}} as unknown as TypedDocumentString<UpdatePerformanceReviewMutation, UpdatePerformanceReviewMutationVariables>;
export const SubmitPerformanceReviewDocument = {"__meta__":{"kind":"mutation","name":"SubmitPerformanceReview","hash":"sha256:8a798f9d9fb5a1661ed4585e955066579f83940beb9e5245b4538515ff886d67"}} as unknown as TypedDocumentString<SubmitPerformanceReviewMutation, SubmitPerformanceReviewMutationVariables>;
export const ReopenPerformanceReviewDocument = {"__meta__":{"kind":"mutation","name":"ReopenPerformanceReview","hash":"sha256:aad588a781f9bd0a69b22cea66a0155dbae60270d618869d67ee1160751c0a12"}} as unknown as TypedDocumentString<ReopenPerformanceReviewMutation, ReopenPerformanceReviewMutationVariables>;
export const ClosePerformanceReviewDocument = {"__meta__":{"kind":"mutation","name":"ClosePerformanceReview","hash":"sha256:85bc357bda914d9e5554d8b2f464e05e72f419881a6513fc9d4eadc214e70b6b"}} as unknown as TypedDocumentString<ClosePerformanceReviewMutation, ClosePerformanceReviewMutationVariables>;
export const DeletePerformanceReviewDocument = {"__meta__":{"kind":"mutation","name":"DeletePerformanceReview","hash":"sha256:7bc8f4751dceb92e8db26ad9b0ca9b5ec34ae318396ec826f0953738c34909ce"}} as unknown as TypedDocumentString<DeletePerformanceReviewMutation, DeletePerformanceReviewMutationVariables>;
export const PtoPolicyTableDocument = {"__meta__":{"kind":"query","name":"PtoPolicyTable","hash":"sha256:0efb44e4f416a85656cdf8267b6ac87b1371a0912a554333e44452f89c420430"}} as unknown as TypedDocumentString<PtoPolicyTableQuery, PtoPolicyTableQueryVariables>;
export const PtoPolicyDocument = {"__meta__":{"kind":"query","name":"PtoPolicy","hash":"sha256:335b38a9d27e09e894c4334bf5cdf48e75049cb04f51d1d9ed140124267d1bd0"}} as unknown as TypedDocumentString<PtoPolicyQuery, PtoPolicyQueryVariables>;
export const WorkerPtoPolicyAssignmentsDocument = {"__meta__":{"kind":"query","name":"WorkerPtoPolicyAssignments","hash":"sha256:fcf14dc0a3ec782be8a5aa87232e8badced8cf0bd3c89fdc0af59bdb29215883"}} as unknown as TypedDocumentString<WorkerPtoPolicyAssignmentsQuery, WorkerPtoPolicyAssignmentsQueryVariables>;
export const WorkerPtoBalancesDocument = {"__meta__":{"kind":"query","name":"WorkerPtoBalances","hash":"sha256:82a747fab6af45b8de61146243d58bc4a7ed8e3369b53d76b2e9f5b3d27da8fd"}} as unknown as TypedDocumentString<WorkerPtoBalancesQuery, WorkerPtoBalancesQueryVariables>;
export const WorkerPtoLedgerDocument = {"__meta__":{"kind":"query","name":"WorkerPtoLedger","hash":"sha256:63f7efbaff3c4b175210cfd3d38f2482b60f5af74b5ad9f1b776f2d5bb59fbbf"}} as unknown as TypedDocumentString<WorkerPtoLedgerQuery, WorkerPtoLedgerQueryVariables>;
export const WorkerPtoAvailabilityDocument = {"__meta__":{"kind":"query","name":"WorkerPtoAvailability","hash":"sha256:34e4d71608e38d84d94ac1dd69ea8990b00343ffbf7ea51c831f2e99bf9fb2f8"}} as unknown as TypedDocumentString<WorkerPtoAvailabilityQuery, WorkerPtoAvailabilityQueryVariables>;
export const PreviewWorkerPtoAccrualDocument = {"__meta__":{"kind":"query","name":"PreviewWorkerPtoAccrual","hash":"sha256:8278eea311fe4bdb7bcc293badd91ec75ac9cc63b29f18bc1afc70e511656698"}} as unknown as TypedDocumentString<PreviewWorkerPtoAccrualQuery, PreviewWorkerPtoAccrualQueryVariables>;
export const PtoBalanceSummaryDocument = {"__meta__":{"kind":"query","name":"PtoBalanceSummary","hash":"sha256:f95c26c6131ca7452e1095119d2627e2cea50394bfda9ef8cd2bce37b9171679"}} as unknown as TypedDocumentString<PtoBalanceSummaryQuery, PtoBalanceSummaryQueryVariables>;
export const PtoLiabilityReportDocument = {"__meta__":{"kind":"query","name":"PtoLiabilityReport","hash":"sha256:e778679b21665a60eed7194357c4baeb78e7d11f90955069ff42b197f6022fef"}} as unknown as TypedDocumentString<PtoLiabilityReportQuery, PtoLiabilityReportQueryVariables>;
export const CreatePtoPolicyDocument = {"__meta__":{"kind":"mutation","name":"CreatePtoPolicy","hash":"sha256:dd29e011aef48d479b4af9aeafcf104646c97bf3c1cb57a58f5f9b73b59a39d2"}} as unknown as TypedDocumentString<CreatePtoPolicyMutation, CreatePtoPolicyMutationVariables>;
export const UpdatePtoPolicyDocument = {"__meta__":{"kind":"mutation","name":"UpdatePtoPolicy","hash":"sha256:f53fe1e8778cb2c2dfbb7e047214c07c304227ab1d7a7f3fbb38d8642b47700e"}} as unknown as TypedDocumentString<UpdatePtoPolicyMutation, UpdatePtoPolicyMutationVariables>;
export const ArchivePtoPolicyDocument = {"__meta__":{"kind":"mutation","name":"ArchivePtoPolicy","hash":"sha256:08abf3278ec9697a583a2d2e894ad8aa03f1f672f6a844f12dc156b6774cb210"}} as unknown as TypedDocumentString<ArchivePtoPolicyMutation, ArchivePtoPolicyMutationVariables>;
export const RestorePtoPolicyDocument = {"__meta__":{"kind":"mutation","name":"RestorePtoPolicy","hash":"sha256:79e6576be8891fe638cec25dcd1ac1f4aa468768d49988774a57838ebf8e2f4c"}} as unknown as TypedDocumentString<RestorePtoPolicyMutation, RestorePtoPolicyMutationVariables>;
export const AssignWorkerPtoPolicyDocument = {"__meta__":{"kind":"mutation","name":"AssignWorkerPtoPolicy","hash":"sha256:ede6a94633150ea6a67f014c7c2fd058e8281b6bff72db042e8e459332e8a982"}} as unknown as TypedDocumentString<AssignWorkerPtoPolicyMutation, AssignWorkerPtoPolicyMutationVariables>;
export const EndWorkerPtoPolicyAssignmentDocument = {"__meta__":{"kind":"mutation","name":"EndWorkerPtoPolicyAssignment","hash":"sha256:f9ff2611c47c157836dcb8f726f5998d9678fa3ebb70cda279ee5b0743b361f3"}} as unknown as TypedDocumentString<EndWorkerPtoPolicyAssignmentMutation, EndWorkerPtoPolicyAssignmentMutationVariables>;
export const AdjustWorkerPtoBalanceDocument = {"__meta__":{"kind":"mutation","name":"AdjustWorkerPtoBalance","hash":"sha256:a3fce577ecde28e859be1eaf870d6a466af8a4785cce1ba86e1b62254741724f"}} as unknown as TypedDocumentString<AdjustWorkerPtoBalanceMutation, AdjustWorkerPtoBalanceMutationVariables>;
export const RunPtoAccrualDocument = {"__meta__":{"kind":"mutation","name":"RunPtoAccrual","hash":"sha256:47af4f61dbe7bbf69b8be7097b975e2c2b82f8d425c01690fbd96cebaf8b72c7"}} as unknown as TypedDocumentString<RunPtoAccrualMutation, RunPtoAccrualMutationVariables>;
export const RateAgreementTableDocument = {"__meta__":{"kind":"query","name":"RateAgreementTable","hash":"sha256:19878581f83da9c8d035951edb5c3b9d7e8c65381a5928f513e0fde06e34fb40"}} as unknown as TypedDocumentString<RateAgreementTableQuery, RateAgreementTableQueryVariables>;
export const RateZoneTableDocument = {"__meta__":{"kind":"query","name":"RateZoneTable","hash":"sha256:5c85f76694ab5702f5ea01ffaae7a1b360b3f9f4845cb21d3230f95cf45bd3d8"}} as unknown as TypedDocumentString<RateZoneTableQuery, RateZoneTableQueryVariables>;
export const RateMatrixTableDocument = {"__meta__":{"kind":"query","name":"RateMatrixTable","hash":"sha256:65be740d59f9fe8439905de2e2b2a65d1a468902d99009a28fb0577333ddd82f"}} as unknown as TypedDocumentString<RateMatrixTableQuery, RateMatrixTableQueryVariables>;
export const RateQuoteTableDocument = {"__meta__":{"kind":"query","name":"RateQuoteTable","hash":"sha256:88e2aef95489941a0c4ba7d6658c28748e949efb2bc7e48800463e41a8406141"}} as unknown as TypedDocumentString<RateQuoteTableQuery, RateQuoteTableQueryVariables>;
export const RecurringShipmentTableDocument = {"__meta__":{"kind":"query","name":"RecurringShipmentTable","hash":"sha256:39b9629192b6601cb0ad2d676939a2f409a667373cf616f7bc85fb478d36c6d7"}} as unknown as TypedDocumentString<RecurringShipmentTableQuery, RecurringShipmentTableQueryVariables>;
export const CannedReportsDocument = {"__meta__":{"kind":"query","name":"CannedReports","hash":"sha256:597b62a2e0291d15c7fc013e195a15594efb4e35e2b82183a24c791037994b27"}} as unknown as TypedDocumentString<CannedReportsQuery, CannedReportsQueryVariables>;
export const ReportCatalogDocument = {"__meta__":{"kind":"query","name":"ReportCatalog","hash":"sha256:79e369a4fec3bb0d7d5c6975d5782adb517ddeb55868c86d31e6f644f12d2d39"}} as unknown as TypedDocumentString<ReportCatalogQuery, ReportCatalogQueryVariables>;
export const ReportDashboardsDocument = {"__meta__":{"kind":"query","name":"ReportDashboards","hash":"sha256:618bd91da756dd1bf620c19604d5bb26ba4ce73841606ed814c47eadefb76229"}} as unknown as TypedDocumentString<ReportDashboardsQuery, ReportDashboardsQueryVariables>;
export const ReportDashboardByIdDocument = {"__meta__":{"kind":"query","name":"ReportDashboardById","hash":"sha256:de2d6876d41ebd263d1936da997cb1ca41d0f92d79b248ad8ac08e86f28bd6df"}} as unknown as TypedDocumentString<ReportDashboardByIdQuery, ReportDashboardByIdQueryVariables>;
export const CreateReportDashboardDocument = {"__meta__":{"kind":"mutation","name":"CreateReportDashboard","hash":"sha256:63539c84ad92d771c2e2aee365288dffc7a677c5933bb8140e4557ff544cf7d6"}} as unknown as TypedDocumentString<CreateReportDashboardMutation, CreateReportDashboardMutationVariables>;
export const UpdateReportDashboardDocument = {"__meta__":{"kind":"mutation","name":"UpdateReportDashboard","hash":"sha256:db102e93d960dee77a8717c0905a53040d9796c3cc5608e76b769ca76543b918"}} as unknown as TypedDocumentString<UpdateReportDashboardMutation, UpdateReportDashboardMutationVariables>;
export const DeleteReportDashboardDocument = {"__meta__":{"kind":"mutation","name":"DeleteReportDashboard","hash":"sha256:67202bb86983a8504070713cc489306fa4a88eb9a2d7476926a7bf482ebd0395"}} as unknown as TypedDocumentString<DeleteReportDashboardMutation, DeleteReportDashboardMutationVariables>;
export const ReportDefinitionsTableDocument = {"__meta__":{"kind":"query","name":"ReportDefinitionsTable","hash":"sha256:dba10c95c235c804727c7e756e0673e7bfa9dcd3443ac1d846ab32ea07d96de7"}} as unknown as TypedDocumentString<ReportDefinitionsTableQuery, ReportDefinitionsTableQueryVariables>;
export const ReportDefinitionByIdDocument = {"__meta__":{"kind":"query","name":"ReportDefinitionById","hash":"sha256:d55a36bcce0a8dba2f6772c0b29874ac0f7dfa18fb4f683fb8b9602fd744aea3"}} as unknown as TypedDocumentString<ReportDefinitionByIdQuery, ReportDefinitionByIdQueryVariables>;
export const ReportDefinitionRevisionsDocument = {"__meta__":{"kind":"query","name":"ReportDefinitionRevisions","hash":"sha256:c2b09eb2de67685ef05b52a0b72e1dab1b2c0ab84c390073889a632d0192bb19"}} as unknown as TypedDocumentString<ReportDefinitionRevisionsQuery, ReportDefinitionRevisionsQueryVariables>;
export const CreateReportDefinitionDocument = {"__meta__":{"kind":"mutation","name":"CreateReportDefinition","hash":"sha256:02ff484673e0f31b72ae3222ce77fda8359c7deb67192b6642cd04a6a2113500"}} as unknown as TypedDocumentString<CreateReportDefinitionMutation, CreateReportDefinitionMutationVariables>;
export const UpdateReportDefinitionDocument = {"__meta__":{"kind":"mutation","name":"UpdateReportDefinition","hash":"sha256:8f03c81fa49d77ea4742568b2d61ed904b5c2ce14b46a73836390d3333c6b97b"}} as unknown as TypedDocumentString<UpdateReportDefinitionMutation, UpdateReportDefinitionMutationVariables>;
export const DeleteReportDefinitionDocument = {"__meta__":{"kind":"mutation","name":"DeleteReportDefinition","hash":"sha256:94b019b0a0a6bf268d050bd41d841b986997ed0566ca7673306374af6391871a"}} as unknown as TypedDocumentString<DeleteReportDefinitionMutation, DeleteReportDefinitionMutationVariables>;
export const ForkCannedReportDocument = {"__meta__":{"kind":"mutation","name":"ForkCannedReport","hash":"sha256:470c88501e4ff64893585861abe0b143a3ca126c3b2937fe59346dcc3f76864b"}} as unknown as TypedDocumentString<ForkCannedReportMutation, ForkCannedReportMutationVariables>;
export const ResetCannedForkDocument = {"__meta__":{"kind":"mutation","name":"ResetCannedFork","hash":"sha256:fdfc8cd8949b329a08fe31e350c065b5be2e732678ee7494e897c729e99ff96c"}} as unknown as TypedDocumentString<ResetCannedForkMutation, ResetCannedForkMutationVariables>;
export const ReportDefinitionOptionsDocument = {"__meta__":{"kind":"query","name":"ReportDefinitionOptions","hash":"sha256:d6c98fc9e87e488486890e898c0c758e16c3e2d046b597e93911513b64b2fd42"}} as unknown as TypedDocumentString<ReportDefinitionOptionsQuery, ReportDefinitionOptionsQueryVariables>;
export const PreviewReportDocument = {"__meta__":{"kind":"query","name":"PreviewReport","hash":"sha256:8093eaf35edcef1de2de2366a7c65d3b3b6c12b432e8ceaf62d76c6e247a6f7d"}} as unknown as TypedDocumentString<PreviewReportQuery, PreviewReportQueryVariables>;
export const DrillThroughReportDocument = {"__meta__":{"kind":"query","name":"DrillThroughReport","hash":"sha256:4dcc2a7446bfffed53694614a973a051d531d33ee0dc0696611fa2fdaf89351e"}} as unknown as TypedDocumentString<DrillThroughReportQuery, DrillThroughReportQueryVariables>;
export const ReportRunsTableDocument = {"__meta__":{"kind":"query","name":"ReportRunsTable","hash":"sha256:a0c575b11f65353b9eb77c3940c12fd0bc82feb31b6159173ca8036c97262a70"}} as unknown as TypedDocumentString<ReportRunsTableQuery, ReportRunsTableQueryVariables>;
export const ReportRunByIdDocument = {"__meta__":{"kind":"query","name":"ReportRunById","hash":"sha256:c99441d285d5a070ea5e519af3f18c58675a28f05533ae71510bf225280ee6a9"}} as unknown as TypedDocumentString<ReportRunByIdQuery, ReportRunByIdQueryVariables>;
export const RunReportDocument = {"__meta__":{"kind":"mutation","name":"RunReport","hash":"sha256:e4af42e2da79707541fc29f6455a42c80767116bdd1bee08dfc63146f32325fb"}} as unknown as TypedDocumentString<RunReportMutation, RunReportMutationVariables>;
export const CancelReportRunDocument = {"__meta__":{"kind":"mutation","name":"CancelReportRun","hash":"sha256:37a51fbc29f91bea2054db71bd1221a29bc4c38c1a8c0d7dde22c4f97fd7e9e4"}} as unknown as TypedDocumentString<CancelReportRunMutation, CancelReportRunMutationVariables>;
export const ReportSchedulesDocument = {"__meta__":{"kind":"query","name":"ReportSchedules","hash":"sha256:2a556320ec0993ae682b12e9e882914106b10f1efad0d86978363d5c11ebb0eb"}} as unknown as TypedDocumentString<ReportSchedulesQuery, ReportSchedulesQueryVariables>;
export const CreateReportScheduleDocument = {"__meta__":{"kind":"mutation","name":"CreateReportSchedule","hash":"sha256:907de6b8f0e39ca398a1e197391b3674219741259c01eb7ed29cc3be47bcc952"}} as unknown as TypedDocumentString<CreateReportScheduleMutation, CreateReportScheduleMutationVariables>;
export const UpdateReportScheduleDocument = {"__meta__":{"kind":"mutation","name":"UpdateReportSchedule","hash":"sha256:5bb42d52908ad8346bebbc9944d422dfb6f8b8604f7f7d80db6b3a2933f7369d"}} as unknown as TypedDocumentString<UpdateReportScheduleMutation, UpdateReportScheduleMutationVariables>;
export const DeleteReportScheduleDocument = {"__meta__":{"kind":"mutation","name":"DeleteReportSchedule","hash":"sha256:dfe8e966e00cda20ec071774d07a7c9d7f28d22648db9faa0ece2d7e233c2bf6"}} as unknown as TypedDocumentString<DeleteReportScheduleMutation, DeleteReportScheduleMutationVariables>;
export const ReportViewsDocument = {"__meta__":{"kind":"query","name":"ReportViews","hash":"sha256:600b2d0308c168f6ea3cc20979d60661695409e382b3563966bbbcdb9eba2c60"}} as unknown as TypedDocumentString<ReportViewsQuery, ReportViewsQueryVariables>;
export const CreateReportViewDocument = {"__meta__":{"kind":"mutation","name":"CreateReportView","hash":"sha256:98f7d9af3b3d8bd99781ee22198f1938a90f7739b986d704463e504d1b2c6b5b"}} as unknown as TypedDocumentString<CreateReportViewMutation, CreateReportViewMutationVariables>;
export const UpdateReportViewDocument = {"__meta__":{"kind":"mutation","name":"UpdateReportView","hash":"sha256:b2575a3fea359ee9b24c354d454aa68f5aa61298c45c180ccd79ba4dfcc7ae7d"}} as unknown as TypedDocumentString<UpdateReportViewMutation, UpdateReportViewMutationVariables>;
export const DeleteReportViewDocument = {"__meta__":{"kind":"mutation","name":"DeleteReportView","hash":"sha256:3e58e3f6975d4537a51320fcbef5cd996c743d72d0006a8510d9791788e86966"}} as unknown as TypedDocumentString<DeleteReportViewMutation, DeleteReportViewMutationVariables>;
export const RoleTableDocument = {"__meta__":{"kind":"query","name":"RoleTable","hash":"sha256:f9ba0ee6a3cf8ba0c1449a77fcc482b705168d4f0047d2051cd028dbe74816e2"}} as unknown as TypedDocumentString<RoleTableQuery, RoleTableQueryVariables>;
export const RoutingGuideTableDocument = {"__meta__":{"kind":"query","name":"RoutingGuideTable","hash":"sha256:00a94193da3a11ddba6612f27c1fe204bfbadb329afd05294e4c0e2060e9d419"}} as unknown as TypedDocumentString<RoutingGuideTableQuery, RoutingGuideTableQueryVariables>;
export const RoutingGuideOptionsDocument = {"__meta__":{"kind":"query","name":"RoutingGuideOptions","hash":"sha256:10f0354a3d4353f0b9f8e43ecd05d89f2b5cf88d51f178f8b69cccb7db3ce32b"}} as unknown as TypedDocumentString<RoutingGuideOptionsQuery, RoutingGuideOptionsQueryVariables>;
export const MatchRoutingGuideDocument = {"__meta__":{"kind":"query","name":"MatchRoutingGuide","hash":"sha256:486025d32ac9cfeaeeda8feccd9dbd3b7efb320829b23306b727962d78d0264c"}} as unknown as TypedDocumentString<MatchRoutingGuideQuery, MatchRoutingGuideQueryVariables>;
export const ShiftTemplatesDocument = {"__meta__":{"kind":"query","name":"ShiftTemplates","hash":"sha256:fed89d4da0d0b311f889914ec96d500d1cb129fe34e089923b4ecbcb676e4a11"}} as unknown as TypedDocumentString<ShiftTemplatesQuery, ShiftTemplatesQueryVariables>;
export const RotaDocument = {"__meta__":{"kind":"query","name":"Rota","hash":"sha256:b123f3dca52bb1332d111335bb733e1e5036fa5a906795567d099123afdf49b6"}} as unknown as TypedDocumentString<RotaQuery, RotaQueryVariables>;
export const WorkerShiftAssignmentsDocument = {"__meta__":{"kind":"query","name":"WorkerShiftAssignments","hash":"sha256:d6f768637bd0926e833b7acdee42eb50e649a1a06b026c4c2c8852a016484dad"}} as unknown as TypedDocumentString<WorkerShiftAssignmentsQuery, WorkerShiftAssignmentsQueryVariables>;
export const WorkerAvailabilityPreferencesDocument = {"__meta__":{"kind":"query","name":"WorkerAvailabilityPreferences","hash":"sha256:82ce6da812e6e6af74e5db8188ce3dbfc5048fd70c722a89a8277cb714d0e5a1"}} as unknown as TypedDocumentString<WorkerAvailabilityPreferencesQuery, WorkerAvailabilityPreferencesQueryVariables>;
export const ShiftSwapRequestsDocument = {"__meta__":{"kind":"query","name":"ShiftSwapRequests","hash":"sha256:ddc6208c93f72be57bee681f5003436a37fa8d7092ea6716e72cb13b012b3132"}} as unknown as TypedDocumentString<ShiftSwapRequestsQuery, ShiftSwapRequestsQueryVariables>;
export const CreateShiftTemplateDocument = {"__meta__":{"kind":"mutation","name":"CreateShiftTemplate","hash":"sha256:35d7e22f75ae74d31fbd07c5c6f3c8f710e53c49dc3a66e93d9b2f7a75ccce01"}} as unknown as TypedDocumentString<CreateShiftTemplateMutation, CreateShiftTemplateMutationVariables>;
export const UpdateShiftTemplateDocument = {"__meta__":{"kind":"mutation","name":"UpdateShiftTemplate","hash":"sha256:96ba5c0cc5c93e91255c224e0d8f89b118bb507ea37ebc2b2115158f0d962524"}} as unknown as TypedDocumentString<UpdateShiftTemplateMutation, UpdateShiftTemplateMutationVariables>;
export const AssignWorkerShiftDocument = {"__meta__":{"kind":"mutation","name":"AssignWorkerShift","hash":"sha256:d001b46f7e4b9ad9c0ae16b8855ec727f142614a56cc64d3a094d6a8cea1e79a"}} as unknown as TypedDocumentString<AssignWorkerShiftMutation, AssignWorkerShiftMutationVariables>;
export const EndWorkerShiftAssignmentDocument = {"__meta__":{"kind":"mutation","name":"EndWorkerShiftAssignment","hash":"sha256:26048860ae9025b2e2192c0fc9169a9209e0a7b261f5c36f4285a841a86a07de"}} as unknown as TypedDocumentString<EndWorkerShiftAssignmentMutation, EndWorkerShiftAssignmentMutationVariables>;
export const SetWorkerAvailabilityPreferenceDocument = {"__meta__":{"kind":"mutation","name":"SetWorkerAvailabilityPreference","hash":"sha256:e4565ae80871c382a4a0ac21d6ef64bb948cd994eb7bbf0aaa6ce95c5eb9f85e"}} as unknown as TypedDocumentString<SetWorkerAvailabilityPreferenceMutation, SetWorkerAvailabilityPreferenceMutationVariables>;
export const ProposeShiftSwapDocument = {"__meta__":{"kind":"mutation","name":"ProposeShiftSwap","hash":"sha256:70a24c343012df3b577a84797a9a8097db3088d305a382de626b7e395118df6b"}} as unknown as TypedDocumentString<ProposeShiftSwapMutation, ProposeShiftSwapMutationVariables>;
export const TransitionShiftSwapDocument = {"__meta__":{"kind":"mutation","name":"TransitionShiftSwap","hash":"sha256:7b3f42ea7e0888c605e6c1a307aa93d8b1be7e48aca356609f24369aa251e244"}} as unknown as TypedDocumentString<TransitionShiftSwapMutation, TransitionShiftSwapMutationVariables>;
export const ScimGroupRoleMappingsTableDocument = {"__meta__":{"kind":"query","name":"SCIMGroupRoleMappingsTable","hash":"sha256:f81b2b7365860ca8a6c0770e4d92c23f5c2da117745ee4a6a3e5b92bc14cadc8"}} as unknown as TypedDocumentString<ScimGroupRoleMappingsTableQuery, ScimGroupRoleMappingsTableQueryVariables>;
export const SelectOptionsDocument = {"__meta__":{"kind":"query","name":"SelectOptions","hash":"sha256:61baa26c739e995aee3b16a4b9f4b584b628598c5e46d1f3886624091f1c12f2"}} as unknown as TypedDocumentString<SelectOptionsQuery, SelectOptionsQueryVariables>;
export const WorkerPoliciesDocument = {"__meta__":{"kind":"query","name":"WorkerPolicies","hash":"sha256:fea267a8ea8d07e3f9d44d6c64435bced19dbbe0887f73179ad6668a771a5f51"}} as unknown as TypedDocumentString<WorkerPoliciesQuery, WorkerPoliciesQueryVariables>;
export const WorkerPolicyComplianceDocument = {"__meta__":{"kind":"query","name":"WorkerPolicyCompliance","hash":"sha256:51d5c5915e99f0448361f968f50ba2483c051c0a9f6e20224ec320fc507a9e42"}} as unknown as TypedDocumentString<WorkerPolicyComplianceQuery, WorkerPolicyComplianceQueryVariables>;
export const WorkerPolicyAcknowledgementsDocument = {"__meta__":{"kind":"query","name":"WorkerPolicyAcknowledgements","hash":"sha256:aacc234ed6fdd60ca8484418bbd16781388d795a314764674b0846de09c99952"}} as unknown as TypedDocumentString<WorkerPolicyAcknowledgementsQuery, WorkerPolicyAcknowledgementsQueryVariables>;
export const ProfileChangeRequestsDocument = {"__meta__":{"kind":"query","name":"ProfileChangeRequests","hash":"sha256:150517c49c5694caa8c604d52cb600d31778ff19e07b90cfa2804159af401700"}} as unknown as TypedDocumentString<ProfileChangeRequestsQuery, ProfileChangeRequestsQueryVariables>;
export const CreateWorkerPolicyDocument = {"__meta__":{"kind":"mutation","name":"CreateWorkerPolicy","hash":"sha256:d00ef7ae80bd93841b748a1bf0e1f02ed0c5693f66b4e88c8c58c15319c7c0b0"}} as unknown as TypedDocumentString<CreateWorkerPolicyMutation, CreateWorkerPolicyMutationVariables>;
export const UpdateWorkerPolicyDocument = {"__meta__":{"kind":"mutation","name":"UpdateWorkerPolicy","hash":"sha256:d3f5e0446b95e5f7fdc703eedca4eb8c8a794d16c3491b3163e8fae37605e7dc"}} as unknown as TypedDocumentString<UpdateWorkerPolicyMutation, UpdateWorkerPolicyMutationVariables>;
export const DecideProfileChangeDocument = {"__meta__":{"kind":"mutation","name":"DecideProfileChange","hash":"sha256:b9fbfae26045f2d782234256da0675b4132c549dd45627b12b64d0da7a15c767"}} as unknown as TypedDocumentString<DecideProfileChangeMutation, DecideProfileChangeMutationVariables>;
export const ServiceFailureReasonCodeTableDocument = {"__meta__":{"kind":"query","name":"ServiceFailureReasonCodeTable","hash":"sha256:041698d7f51840151afa5a8fb10ceb205540001dc66ed1969c2366060585b3aa"}} as unknown as TypedDocumentString<ServiceFailureReasonCodeTableQuery, ServiceFailureReasonCodeTableQueryVariables>;
export const ServiceFailureTableDocument = {"__meta__":{"kind":"query","name":"ServiceFailureTable","hash":"sha256:802d21c82ae8c40acf8a3e43537d88efc7781983e248ad672961a74b46c52bd0"}} as unknown as TypedDocumentString<ServiceFailureTableQuery, ServiceFailureTableQueryVariables>;
export const ServiceTypeTableDocument = {"__meta__":{"kind":"query","name":"ServiceTypeTable","hash":"sha256:ba2cc0fdc314c6c3e25d306f5ad63a4c09ac96b537a0bf60b874ea22eec70682"}} as unknown as TypedDocumentString<ServiceTypeTableQuery, ServiceTypeTableQueryVariables>;
export const ShipmentTypeTableDocument = {"__meta__":{"kind":"query","name":"ShipmentTypeTable","hash":"sha256:2be2cf7c6760639a92a3977f36a489f31e14b4c27edae1049a9589cce837a534"}} as unknown as TypedDocumentString<ShipmentTypeTableQuery, ShipmentTypeTableQueryVariables>;
export const ShipmentCommandCenterTableDocument = {"__meta__":{"kind":"query","name":"ShipmentCommandCenterTable","hash":"sha256:c4bdb53bfcb7e188a11e6c7bb9681a5403adae31292d554466ff3c087803f161"}} as unknown as TypedDocumentString<ShipmentCommandCenterTableQuery, ShipmentCommandCenterTableQueryVariables>;
export const ShipmentDetailDocument = {"__meta__":{"kind":"query","name":"ShipmentDetail","hash":"sha256:d8478c4de5838531f8e2ab5031ec571d956c6c852aee8f89329944503141e579"}} as unknown as TypedDocumentString<ShipmentDetailQuery, ShipmentDetailQueryVariables>;
export const ShipmentSavedViewCountsDocument = {"__meta__":{"kind":"query","name":"ShipmentSavedViewCounts","hash":"sha256:cbed3f0cc310a0a4c3435b533a963c297ad2bad4a07174563944705242d2d168"}} as unknown as TypedDocumentString<ShipmentSavedViewCountsQuery, ShipmentSavedViewCountsQueryVariables>;
export const ShipmentPageAnalyticsDocument = {"__meta__":{"kind":"query","name":"ShipmentPageAnalytics","hash":"sha256:ad48e5077b2ccc6fd13488ff0477d404b19f9a4067a6d2dbc5451ec44869443e"}} as unknown as TypedDocumentString<ShipmentPageAnalyticsQuery, ShipmentPageAnalyticsQueryVariables>;
export const ShipmentTomorrowsPickupsDocument = {"__meta__":{"kind":"query","name":"ShipmentTomorrowsPickups","hash":"sha256:4efe02e85e165ab339b90c81ea8d05dad114942c74d9333034f58a4e6a609ee4"}} as unknown as TypedDocumentString<ShipmentTomorrowsPickupsQuery, ShipmentTomorrowsPickupsQueryVariables>;
export const UnassignedShipmentsDocument = {"__meta__":{"kind":"query","name":"UnassignedShipments","hash":"sha256:7f3cc26e6ecb02400d7031ee412e04e9e1b19e2ba711b1260ddfd47e433431b4"}} as unknown as TypedDocumentString<UnassignedShipmentsQuery, UnassignedShipmentsQueryVariables>;
export const ExceptionShipmentsDocument = {"__meta__":{"kind":"query","name":"ExceptionShipments","hash":"sha256:dad4d2061dde853d418386fb69882d3177ca131fd7ff1bd7342609891c1a7d27"}} as unknown as TypedDocumentString<ExceptionShipmentsQuery, ExceptionShipmentsQueryVariables>;
export const MapShipmentsDocument = {"__meta__":{"kind":"query","name":"MapShipments","hash":"sha256:b0c4c6e063ee02ad3619c6311c21f51246aad66c199b5f7aea39e91b23ea46ff"}} as unknown as TypedDocumentString<MapShipmentsQuery, MapShipmentsQueryVariables>;
export const ShipmentCommentsDocument = {"__meta__":{"kind":"query","name":"ShipmentComments","hash":"sha256:e8ded6c042536cd06b3552020cc7245d5d5585ebae1cee2cdf5d4369b07b33a7"}} as unknown as TypedDocumentString<ShipmentCommentsQuery, ShipmentCommentsQueryVariables>;
export const ShipmentCommentRepliesDocument = {"__meta__":{"kind":"query","name":"ShipmentCommentReplies","hash":"sha256:c3d7d09953b6ceb3ea38fbc547d409fea336121d2b3bcacefce4b894f29bd3c6"}} as unknown as TypedDocumentString<ShipmentCommentRepliesQuery, ShipmentCommentRepliesQueryVariables>;
export const ShipmentCommentCountDocument = {"__meta__":{"kind":"query","name":"ShipmentCommentCount","hash":"sha256:1f62df3579f042a9c8914aa2b124bb976b08c30fdb27dc1fa25926487e7d877e"}} as unknown as TypedDocumentString<ShipmentCommentCountQuery, ShipmentCommentCountQueryVariables>;
export const ShipmentEventsDocument = {"__meta__":{"kind":"query","name":"ShipmentEvents","hash":"sha256:78c0855d984d37ab5e0f0b5cb23b5bdb5d86ba194c21d403a3339edf336821f5"}} as unknown as TypedDocumentString<ShipmentEventsQuery, ShipmentEventsQueryVariables>;
export const ShipmentBillingReadinessDocument = {"__meta__":{"kind":"query","name":"ShipmentBillingReadiness","hash":"sha256:e75cb6d00ed67d58a2fe75606c9449dd1e55a2f61b902db0aedf9941ee01a383"}} as unknown as TypedDocumentString<ShipmentBillingReadinessQuery, ShipmentBillingReadinessQueryVariables>;
export const ShipmentUiPolicyDocument = {"__meta__":{"kind":"query","name":"ShipmentUIPolicy","hash":"sha256:31816c9ef557fefb9f366f7c6de4a496827f84f1ce406e42f21e1444ddefb7a1"}} as unknown as TypedDocumentString<ShipmentUiPolicyQuery, ShipmentUiPolicyQueryVariables>;
export const ShipmentPreviousRatesDocument = {"__meta__":{"kind":"query","name":"ShipmentPreviousRates","hash":"sha256:fb9ce636f0cfa91106dfcc559e31eb59e1e6cfa4d229668dbc2208e7a3730f9b"}} as unknown as TypedDocumentString<ShipmentPreviousRatesQuery, ShipmentPreviousRatesQueryVariables>;
export const CreateShipmentDocument = {"__meta__":{"kind":"mutation","name":"CreateShipment","hash":"sha256:139044137e871fc8bbb8516bc7d081398965fa9ec043a85b1dc3d122d3065c3a"}} as unknown as TypedDocumentString<CreateShipmentMutation, CreateShipmentMutationVariables>;
export const UpdateShipmentDocument = {"__meta__":{"kind":"mutation","name":"UpdateShipment","hash":"sha256:019662c077905427d5c61a8e2356e29040259e294c809950f9bb7976de5c37da"}} as unknown as TypedDocumentString<UpdateShipmentMutation, UpdateShipmentMutationVariables>;
export const CancelShipmentDocument = {"__meta__":{"kind":"mutation","name":"CancelShipment","hash":"sha256:8caac86e360152b31fbf382515084d43043d3a6cfa4ae8714bd0e1f3e89029ff"}} as unknown as TypedDocumentString<CancelShipmentMutation, CancelShipmentMutationVariables>;
export const UncancelShipmentDocument = {"__meta__":{"kind":"mutation","name":"UncancelShipment","hash":"sha256:a1cfa490077521e9eed86fa9b287ca266730a37d6bfb21b2f130e92c1a36d964"}} as unknown as TypedDocumentString<UncancelShipmentMutation, UncancelShipmentMutationVariables>;
export const DuplicateShipmentDocument = {"__meta__":{"kind":"mutation","name":"DuplicateShipment","hash":"sha256:0dcc6ec862a4ef66a9e7137e45548bb355204f1bc766d172a035b5e537298ecf"}} as unknown as TypedDocumentString<DuplicateShipmentMutation, DuplicateShipmentMutationVariables>;
export const TransferShipmentOwnershipDocument = {"__meta__":{"kind":"mutation","name":"TransferShipmentOwnership","hash":"sha256:9a6103656bdc65c8fc5c09f445ae0b982d3bcb71ff9a0a5c08b4f4594b13d0a9"}} as unknown as TypedDocumentString<TransferShipmentOwnershipMutation, TransferShipmentOwnershipMutationVariables>;
export const TransferShipmentToBillingDocument = {"__meta__":{"kind":"mutation","name":"TransferShipmentToBilling","hash":"sha256:7849b77f08e7c2e7cb6af2c2abbc53185d0811092df155557be1c6803b335473"}} as unknown as TypedDocumentString<TransferShipmentToBillingMutation, TransferShipmentToBillingMutationVariables>;
export const BulkTransferShipmentsToBillingDocument = {"__meta__":{"kind":"mutation","name":"BulkTransferShipmentsToBilling","hash":"sha256:46beae0d55af6f6abff7b4c2090ce9ae974f805e505c3ff2c60ababb2f33e0a2"}} as unknown as TypedDocumentString<BulkTransferShipmentsToBillingMutation, BulkTransferShipmentsToBillingMutationVariables>;
export const CalculateShipmentTotalsDocument = {"__meta__":{"kind":"mutation","name":"CalculateShipmentTotals","hash":"sha256:675789448d139ef11053baf07c910d999193810b6acd81ae401229c1e1753a75"}} as unknown as TypedDocumentString<CalculateShipmentTotalsMutation, CalculateShipmentTotalsMutationVariables>;
export const PreviewShipmentContractRateDocument = {"__meta__":{"kind":"mutation","name":"PreviewShipmentContractRate","hash":"sha256:b3609be8eacbbcd92db634d5a0939e1cf3bd0d4c1635b56ac8a7e1082d7aaaff"}} as unknown as TypedDocumentString<PreviewShipmentContractRateMutation, PreviewShipmentContractRateMutationVariables>;
export const AutoRateShipmentDocument = {"__meta__":{"kind":"mutation","name":"AutoRateShipment","hash":"sha256:0afeac8a383afc1ec96502ad9878cd88eaaa4aeb3ba477c4d4d49537be41c6c0"}} as unknown as TypedDocumentString<AutoRateShipmentMutation, AutoRateShipmentMutationVariables>;
export const CalculateShipmentDistanceDocument = {"__meta__":{"kind":"mutation","name":"CalculateShipmentDistance","hash":"sha256:5c8612acf5d98e8e255b7ec31d1fb37d4723fe9cfed4f6e2c2c4203ba5092b5a"}} as unknown as TypedDocumentString<CalculateShipmentDistanceMutation, CalculateShipmentDistanceMutationVariables>;
export const RecalculateShipmentDistanceDocument = {"__meta__":{"kind":"mutation","name":"RecalculateShipmentDistance","hash":"sha256:c22b19ad3ce0ea5856e7d3b13cbf94d2b5a34aadeaf0f90cf09c5ba8f6f87c51"}} as unknown as TypedDocumentString<RecalculateShipmentDistanceMutation, RecalculateShipmentDistanceMutationVariables>;
export const CheckShipmentDuplicateBolDocument = {"__meta__":{"kind":"mutation","name":"CheckShipmentDuplicateBol","hash":"sha256:245fce8ae3f1f985031b2343ce03fa257082b6453a6b738c209e49581315c33c"}} as unknown as TypedDocumentString<CheckShipmentDuplicateBolMutation, CheckShipmentDuplicateBolMutationVariables>;
export const CheckShipmentHazmatSegregationDocument = {"__meta__":{"kind":"mutation","name":"CheckShipmentHazmatSegregation","hash":"sha256:94a8f251053368e89199e02110d476108016a60da4e182320c793065a99b1b7a"}} as unknown as TypedDocumentString<CheckShipmentHazmatSegregationMutation, CheckShipmentHazmatSegregationMutationVariables>;
export const CalculateShipmentLoadingOptimizationDocument = {"__meta__":{"kind":"mutation","name":"CalculateShipmentLoadingOptimization","hash":"sha256:23ff92adf45fa82bf4f021539ac314f2922bd091b303c771fab8c553dbfe68dc"}} as unknown as TypedDocumentString<CalculateShipmentLoadingOptimizationMutation, CalculateShipmentLoadingOptimizationMutationVariables>;
export const CreateShipmentCommentDocument = {"__meta__":{"kind":"mutation","name":"CreateShipmentComment","hash":"sha256:170709ad4ba931f717d70c9688af8f8b3849e6ead4f054f05126967f35335c41"}} as unknown as TypedDocumentString<CreateShipmentCommentMutation, CreateShipmentCommentMutationVariables>;
export const UpdateShipmentCommentDocument = {"__meta__":{"kind":"mutation","name":"UpdateShipmentComment","hash":"sha256:4a2ffc78c7fadd6a7382d1feb950e2396675b1e15759f429a52a25d374a94e60"}} as unknown as TypedDocumentString<UpdateShipmentCommentMutation, UpdateShipmentCommentMutationVariables>;
export const DeleteShipmentCommentDocument = {"__meta__":{"kind":"mutation","name":"DeleteShipmentComment","hash":"sha256:a20dcdea6225911dd4742c1e415a5f1e2b04d0111fbaf5ecbda1e8136b3dfa14"}} as unknown as TypedDocumentString<DeleteShipmentCommentMutation, DeleteShipmentCommentMutationVariables>;
export const PinShipmentCommentDocument = {"__meta__":{"kind":"mutation","name":"PinShipmentComment","hash":"sha256:d2b2515bb8f7ad0f27a906f1b8c247ab3e3940a09ea0cbcb373c89d874b6ce1a"}} as unknown as TypedDocumentString<PinShipmentCommentMutation, PinShipmentCommentMutationVariables>;
export const UnpinShipmentCommentDocument = {"__meta__":{"kind":"mutation","name":"UnpinShipmentComment","hash":"sha256:c61d890d2ec45e52d93b3c2d3e9fe855d5ff6132c8710fead259c1287fe38117"}} as unknown as TypedDocumentString<UnpinShipmentCommentMutation, UnpinShipmentCommentMutationVariables>;
export const ResolveShipmentCommentDocument = {"__meta__":{"kind":"mutation","name":"ResolveShipmentComment","hash":"sha256:267ac8318e557775cbb53e7534ca53ba43c55eea99288a29493b247eb767b259"}} as unknown as TypedDocumentString<ResolveShipmentCommentMutation, ResolveShipmentCommentMutationVariables>;
export const UnresolveShipmentCommentDocument = {"__meta__":{"kind":"mutation","name":"UnresolveShipmentComment","hash":"sha256:71471eb23afe1984ba946b9d6455a7f5a2d1ae63c87d3dfdd8b6091d000a8dac"}} as unknown as TypedDocumentString<UnresolveShipmentCommentMutation, UnresolveShipmentCommentMutationVariables>;
export const AcknowledgeShipmentCommentDocument = {"__meta__":{"kind":"mutation","name":"AcknowledgeShipmentComment","hash":"sha256:bf20a040f4fdcf8f104457c01182123d1f077daa36e7e4df5b3ff04f28f9081f"}} as unknown as TypedDocumentString<AcknowledgeShipmentCommentMutation, AcknowledgeShipmentCommentMutationVariables>;
export const ShipmentProfitabilityDocument = {"__meta__":{"kind":"query","name":"ShipmentProfitability","hash":"sha256:ba934decc6721bef0b703376a291e500f48d887692508b928c2286896b52d8a9"}} as unknown as TypedDocumentString<ShipmentProfitabilityQuery, ShipmentProfitabilityQueryVariables>;
export const UpdateSidebarPreferencesDocument = {"__meta__":{"kind":"mutation","name":"UpdateSidebarPreferences","hash":"sha256:977fedf72d0dd1e084eb48b093203298a4b2d9513240e0149488448e349d8128"}} as unknown as TypedDocumentString<UpdateSidebarPreferencesMutation, UpdateSidebarPreferencesMutationVariables>;
export const SidebarPreferencesDocument = {"__meta__":{"kind":"query","name":"SidebarPreferences","hash":"sha256:a136ac10eb71000bcfef94663b0a9df6cba4161eb0ec120f890e3e3a06a23bdc"}} as unknown as TypedDocumentString<SidebarPreferencesQuery, SidebarPreferencesQueryVariables>;
export const SidebarCustomizationOptionsDocument = {"__meta__":{"kind":"query","name":"SidebarCustomizationOptions","hash":"sha256:79b6c8e2b9458d391abe3a98706dc989e567930eaf28453bd9635cf1766b276d"}} as unknown as TypedDocumentString<SidebarCustomizationOptionsQuery, SidebarCustomizationOptionsQueryVariables>;
export const StoredMileageTableDocument = {"__meta__":{"kind":"query","name":"StoredMileageTable","hash":"sha256:a66d19788a7328a4776edf7ee5612af9fa8663d1019f8c20f700fbc321945678"}} as unknown as TypedDocumentString<StoredMileageTableQuery, StoredMileageTableQueryVariables>;
export const TcaSubscriptionTableDocument = {"__meta__":{"kind":"query","name":"TCASubscriptionTable","hash":"sha256:95cbea4e2c46a39616eafb7d6768643c6dea54ff2d0206fc3002d31745721746"}} as unknown as TypedDocumentString<TcaSubscriptionTableQuery, TcaSubscriptionTableQueryVariables>;
export const TableConfigurationTableDocument = {"__meta__":{"kind":"query","name":"TableConfigurationTable","hash":"sha256:6f402040f38a4d15dc9818af5c03f5245b13ffdbee1110e495e01afefd6c97c8"}} as unknown as TypedDocumentString<TableConfigurationTableQuery, TableConfigurationTableQueryVariables>;
export const DefaultTableConfigurationDocument = {"__meta__":{"kind":"query","name":"DefaultTableConfiguration","hash":"sha256:44172e730a5efee17642504f34cd054b9dacd048e99f174d4678866c287d99a5"}} as unknown as TypedDocumentString<DefaultTableConfigurationQuery, DefaultTableConfigurationQueryVariables>;
export const TableConfigurationDetailDocument = {"__meta__":{"kind":"query","name":"TableConfigurationDetail","hash":"sha256:d3517c4942a70bd816f40494468ad1aef6af9f5b261fe2da3a1f6d7308f61b0e"}} as unknown as TypedDocumentString<TableConfigurationDetailQuery, TableConfigurationDetailQueryVariables>;
export const CreateTableConfigurationDocument = {"__meta__":{"kind":"mutation","name":"CreateTableConfiguration","hash":"sha256:9da2c2f361a87c5bef4f0cb0beae047cf548a74b5c389efdf63ccd4adb0e8625"}} as unknown as TypedDocumentString<CreateTableConfigurationMutation, CreateTableConfigurationMutationVariables>;
export const UpdateTableConfigurationDocument = {"__meta__":{"kind":"mutation","name":"UpdateTableConfiguration","hash":"sha256:4c1284c544aa1e09687d7ae906b44f39a035adef42ec417d124be8e5fc9946eb"}} as unknown as TypedDocumentString<UpdateTableConfigurationMutation, UpdateTableConfigurationMutationVariables>;
export const DeleteTableConfigurationDocument = {"__meta__":{"kind":"mutation","name":"DeleteTableConfiguration","hash":"sha256:5c2f8a0d5ce9cc3f5ae7702ee9785c067097d3a04de36ae8b17d43afeb5e4951"}} as unknown as TypedDocumentString<DeleteTableConfigurationMutation, DeleteTableConfigurationMutationVariables>;
export const SetDefaultTableConfigurationDocument = {"__meta__":{"kind":"mutation","name":"SetDefaultTableConfiguration","hash":"sha256:d186244dea4960f7116982c1281aa7b3daec229ba06143417bbd3064d746340b"}} as unknown as TypedDocumentString<SetDefaultTableConfigurationMutation, SetDefaultTableConfigurationMutationVariables>;
export const SetOrgDefaultTableConfigurationDocument = {"__meta__":{"kind":"mutation","name":"SetOrgDefaultTableConfiguration","hash":"sha256:739270b19dc50a3d42f57caf1791f4dc556c4b51547b84eef44000bceb25aadc"}} as unknown as TypedDocumentString<SetOrgDefaultTableConfigurationMutation, SetOrgDefaultTableConfigurationMutationVariables>;
export const VehiclePositionsDocument = {"__meta__":{"kind":"query","name":"VehiclePositions","hash":"sha256:1ea010c608c8a6a56a22bdb956c0b3456a84a178dbd0b26b546394151f2643b7"}} as unknown as TypedDocumentString<VehiclePositionsQuery, VehiclePositionsQueryVariables>;
export const WorkerHosStatesDocument = {"__meta__":{"kind":"query","name":"WorkerHosStates","hash":"sha256:e993196b73825d21c3252bd8ab5924aa96894e77dfbff5a79ff2c27b82ce94c9"}} as unknown as TypedDocumentString<WorkerHosStatesQuery, WorkerHosStatesQueryVariables>;
export const WorkerHosStateDocument = {"__meta__":{"kind":"query","name":"WorkerHosState","hash":"sha256:cc49820a722034cd85add21c54f6a0f71735ab5ee21b8fda9bb4fa168950b241"}} as unknown as TypedDocumentString<WorkerHosStateQuery, WorkerHosStateQueryVariables>;
export const WorkerHosViolationsDocument = {"__meta__":{"kind":"query","name":"WorkerHosViolations","hash":"sha256:ff58ab4e981d04a859aa25372aecce12c32f9555c11a6b538bb91c9967b0832b"}} as unknown as TypedDocumentString<WorkerHosViolationsQuery, WorkerHosViolationsQueryVariables>;
export const TelematicsStatusDocument = {"__meta__":{"kind":"query","name":"TelematicsStatus","hash":"sha256:1d9495578b6b9b5df107368f4aa3a8dd865e9206d2783973300edad4ae0e41ea"}} as unknown as TypedDocumentString<TelematicsStatusQuery, TelematicsStatusQueryVariables>;
export const WorkerHosLogsDocument = {"__meta__":{"kind":"query","name":"WorkerHosLogs","hash":"sha256:1ee28f025f06aad3ef1b99f06e5075125388ba20443c21755dd66f603fc37402"}} as unknown as TypedDocumentString<WorkerHosLogsQuery, WorkerHosLogsQueryVariables>;
export const WorkerHosDailyLogsDocument = {"__meta__":{"kind":"query","name":"WorkerHosDailyLogs","hash":"sha256:afb2b821da02a2d1299a5244e45e4d69fa3fca98fc85b75a166d75f02f348620"}} as unknown as TypedDocumentString<WorkerHosDailyLogsQuery, WorkerHosDailyLogsQueryVariables>;
export const ShipmentDriverFeasibilityDocument = {"__meta__":{"kind":"query","name":"ShipmentDriverFeasibility","hash":"sha256:fdb4def530fa43a2a133da4ed4fc2520176f2f85cec98b87c6757cc39bd37ba8"}} as unknown as TypedDocumentString<ShipmentDriverFeasibilityQuery, ShipmentDriverFeasibilityQueryVariables>;
export const VehicleInspectionsDocument = {"__meta__":{"kind":"query","name":"VehicleInspections","hash":"sha256:ced1b500c8d2c6b9d7fb02e97ff75a39886d4c86224963eccecfa4c749a63bdf"}} as unknown as TypedDocumentString<VehicleInspectionsQuery, VehicleInspectionsQueryVariables>;
export const WorkerFormSubmissionsDocument = {"__meta__":{"kind":"query","name":"WorkerFormSubmissions","hash":"sha256:8f4bcad47d806d5e7737ee733281e3aaf8a1934abeaa3636b86aa0746fb154e8"}} as unknown as TypedDocumentString<WorkerFormSubmissionsQuery, WorkerFormSubmissionsQueryVariables>;
export const HosCertificationSummaryDocument = {"__meta__":{"kind":"query","name":"HosCertificationSummary","hash":"sha256:5ef397ad8ab2cda3c54bdfb4972f31a59a23254d83eaf9cf1bd373d3a6250ac9"}} as unknown as TypedDocumentString<HosCertificationSummaryQuery, HosCertificationSummaryQueryVariables>;
export const ShipmentFormSubmissionsDocument = {"__meta__":{"kind":"query","name":"ShipmentFormSubmissions","hash":"sha256:e12871be510b7dfe523e84064a1bc8275664174502694f53ce189baffb0544b6"}} as unknown as TypedDocumentString<ShipmentFormSubmissionsQuery, ShipmentFormSubmissionsQueryVariables>;
export const TelematicsFormMappingsDocument = {"__meta__":{"kind":"query","name":"TelematicsFormMappings","hash":"sha256:93efe56f6c97c1f71dfc6b2194bc7218d22a6cb0394af546447cf262a85546e6"}} as unknown as TypedDocumentString<TelematicsFormMappingsQuery, TelematicsFormMappingsQueryVariables>;
export const SaveTelematicsFormMappingDocument = {"__meta__":{"kind":"mutation","name":"SaveTelematicsFormMapping","hash":"sha256:4d1161b463a6581182544db2f8fff4d6006221c9955cbf2dcedac8f1b939aa76"}} as unknown as TypedDocumentString<SaveTelematicsFormMappingMutation, SaveTelematicsFormMappingMutationVariables>;
export const DeleteTelematicsFormMappingDocument = {"__meta__":{"kind":"mutation","name":"DeleteTelematicsFormMapping","hash":"sha256:40eca474cafce94d59c9ce91ea624bdb51a477fa552b3f43e0a773c64fd39baf"}} as unknown as TypedDocumentString<DeleteTelematicsFormMappingMutation, DeleteTelematicsFormMappingMutationVariables>;
export const TendersByShipmentDocument = {"__meta__":{"kind":"query","name":"TendersByShipment","hash":"sha256:f6ad3ca8a9af0ed699faccd13b6de03828ced3e5cd1dc6645b77ae90367b9d4c"}} as unknown as TypedDocumentString<TendersByShipmentQuery, TendersByShipmentQueryVariables>;
export const LiveTenderByMoveDocument = {"__meta__":{"kind":"query","name":"LiveTenderByMove","hash":"sha256:c78680524944d9d61eddd81bb36fc68cfbd4c54eb78ea54156184d287f76f01c"}} as unknown as TypedDocumentString<LiveTenderByMoveQuery, LiveTenderByMoveQueryVariables>;
export const TimesheetsDocument = {"__meta__":{"kind":"query","name":"Timesheets","hash":"sha256:1a3d62286efe8793991946101a825bae461f64101f9b2d0b37973d5e228493e0"}} as unknown as TypedDocumentString<TimesheetsQuery, TimesheetsQueryVariables>;
export const TimesheetDocument = {"__meta__":{"kind":"query","name":"Timesheet","hash":"sha256:12711c2b971babc2942d764b5117b49c3a678cc51c7e642fc8b9affe70bb320b"}} as unknown as TypedDocumentString<TimesheetQuery, TimesheetQueryVariables>;
export const OpenTimeClockEntryDocument = {"__meta__":{"kind":"query","name":"OpenTimeClockEntry","hash":"sha256:1272e13c5df6e922bb9654c8b1c49a45e7dc60e640b9d7246ab34285c1882fc4"}} as unknown as TypedDocumentString<OpenTimeClockEntryQuery, OpenTimeClockEntryQueryVariables>;
export const OpenTimeClockEntriesDocument = {"__meta__":{"kind":"query","name":"OpenTimeClockEntries","hash":"sha256:98cc8429a287bb408dc0b19de2fd3f19191c70cf94b5b28aa012618153446cca"}} as unknown as TypedDocumentString<OpenTimeClockEntriesQuery, OpenTimeClockEntriesQueryVariables>;
export const TimeClockEntriesDocument = {"__meta__":{"kind":"query","name":"TimeClockEntries","hash":"sha256:0f6d1c017299bf302802a64bf4f95a82176d6fe82dfb004c973abfd959402084"}} as unknown as TypedDocumentString<TimeClockEntriesQuery, TimeClockEntriesQueryVariables>;
export const PayrollExportsDocument = {"__meta__":{"kind":"query","name":"PayrollExports","hash":"sha256:b5858725b15914817f0c75820ddbaab0fb39595b887b6d4c3e927a0265dc46b4"}} as unknown as TypedDocumentString<PayrollExportsQuery, PayrollExportsQueryVariables>;
export const PayrollExportRowsDocument = {"__meta__":{"kind":"query","name":"PayrollExportRows","hash":"sha256:aff11886ac35bfb8c18224f9e7af7325515fb7c7b15cd7ea5802afa38de499e7"}} as unknown as TypedDocumentString<PayrollExportRowsQuery, PayrollExportRowsQueryVariables>;
export const ClockInDocument = {"__meta__":{"kind":"mutation","name":"ClockIn","hash":"sha256:4a314b75dc73b07fc6de052a29ec5d64ede456e48ffa7c8b535a5bb03f4b2fef"}} as unknown as TypedDocumentString<ClockInMutation, ClockInMutationVariables>;
export const ClockOutDocument = {"__meta__":{"kind":"mutation","name":"ClockOut","hash":"sha256:ec932579773088d7e682499acbc07bf157f2b3c0f6f711d97eb930278589fd46"}} as unknown as TypedDocumentString<ClockOutMutation, ClockOutMutationVariables>;
export const RecordTimeEntryDocument = {"__meta__":{"kind":"mutation","name":"RecordTimeEntry","hash":"sha256:eb687fef8a3d832a55fc277666814a488f95138985a6c3ec5c482434fc39b73a"}} as unknown as TypedDocumentString<RecordTimeEntryMutation, RecordTimeEntryMutationVariables>;
export const DeleteTimeEntryDocument = {"__meta__":{"kind":"mutation","name":"DeleteTimeEntry","hash":"sha256:589bdd27d5db234afedff7f63f016e22b9f847113562129fbbc91089e345fa32"}} as unknown as TypedDocumentString<DeleteTimeEntryMutation, DeleteTimeEntryMutationVariables>;
export const TransitionTimesheetDocument = {"__meta__":{"kind":"mutation","name":"TransitionTimesheet","hash":"sha256:dfc4ed27650848f5717c720bb246c172df69fe44f78a9cce0061aa8dcd68ea01"}} as unknown as TypedDocumentString<TransitionTimesheetMutation, TransitionTimesheetMutationVariables>;
export const GeneratePayrollExportDocument = {"__meta__":{"kind":"mutation","name":"GeneratePayrollExport","hash":"sha256:d2e6498db231b0929eaae44b9fec9404182112f932187530c9b5d214f16cb761"}} as unknown as TypedDocumentString<GeneratePayrollExportMutation, GeneratePayrollExportMutationVariables>;
export const VoidPayrollExportDocument = {"__meta__":{"kind":"mutation","name":"VoidPayrollExport","hash":"sha256:91d91e0679fb937df8c50be37f8d75aa727a2a797c7a9a886e9afe3e903b89c2"}} as unknown as TypedDocumentString<VoidPayrollExportMutation, VoidPayrollExportMutationVariables>;
export const UserTableDocument = {"__meta__":{"kind":"query","name":"UserTable","hash":"sha256:40300ce9b4742ab9f0008bfa9e6af539b18334e725df797823f41d75af2f3b48"}} as unknown as TypedDocumentString<UserTableQuery, UserTableQueryVariables>;
export const WorkerChecklistTemplateTableDocument = {"__meta__":{"kind":"query","name":"WorkerChecklistTemplateTable","hash":"sha256:f2eda82e9be4f74f7c9dc81c979e31c50ed9bc54a77cbc16e0faeecbc20376e2"}} as unknown as TypedDocumentString<WorkerChecklistTemplateTableQuery, WorkerChecklistTemplateTableQueryVariables>;
export const ActiveWorkerChecklistTemplatesDocument = {"__meta__":{"kind":"query","name":"ActiveWorkerChecklistTemplates","hash":"sha256:7ea423ffb6c8654acdf33a23df25d34a19e6af10a5300c0a6f3bf0f0f25a30ab"}} as unknown as TypedDocumentString<ActiveWorkerChecklistTemplatesQuery, ActiveWorkerChecklistTemplatesQueryVariables>;
export const WorkerChecklistsDocument = {"__meta__":{"kind":"query","name":"WorkerChecklists","hash":"sha256:04a8feaf12d8632f22968dc84a503969e99a8a2475d5dd7db3f25cf26416fcea"}} as unknown as TypedDocumentString<WorkerChecklistsQuery, WorkerChecklistsQueryVariables>;
export const CreateWorkerChecklistTemplateDocument = {"__meta__":{"kind":"mutation","name":"CreateWorkerChecklistTemplate","hash":"sha256:1b4ca303c12022854b04aece48dadf86c4847b20ba7bc8a0e689928a07eed4c4"}} as unknown as TypedDocumentString<CreateWorkerChecklistTemplateMutation, CreateWorkerChecklistTemplateMutationVariables>;
export const UpdateWorkerChecklistTemplateDocument = {"__meta__":{"kind":"mutation","name":"UpdateWorkerChecklistTemplate","hash":"sha256:e5c77e2496c98fb0e112ef2e3e9dd5c3a78565cd8bea2b90d24691166b404eed"}} as unknown as TypedDocumentString<UpdateWorkerChecklistTemplateMutation, UpdateWorkerChecklistTemplateMutationVariables>;
export const ArchiveWorkerChecklistTemplateDocument = {"__meta__":{"kind":"mutation","name":"ArchiveWorkerChecklistTemplate","hash":"sha256:000fd899b4ccdbbe75bca57ad1288f1fbeae4ade33ca4d3c65349c39858ed57c"}} as unknown as TypedDocumentString<ArchiveWorkerChecklistTemplateMutation, ArchiveWorkerChecklistTemplateMutationVariables>;
export const RestoreWorkerChecklistTemplateDocument = {"__meta__":{"kind":"mutation","name":"RestoreWorkerChecklistTemplate","hash":"sha256:60d70c8e9fca300cda22eb4bbdf001804827dae40da225a595f2a6f18a7c3f0f"}} as unknown as TypedDocumentString<RestoreWorkerChecklistTemplateMutation, RestoreWorkerChecklistTemplateMutationVariables>;
export const StartWorkerChecklistDocument = {"__meta__":{"kind":"mutation","name":"StartWorkerChecklist","hash":"sha256:0da3f21bbda8c367b66349f05c37fa773a1690610705d18d5821f9ba0d0561ea"}} as unknown as TypedDocumentString<StartWorkerChecklistMutation, StartWorkerChecklistMutationVariables>;
export const CompleteWorkerChecklistItemDocument = {"__meta__":{"kind":"mutation","name":"CompleteWorkerChecklistItem","hash":"sha256:b6fe07424ef9e2601a8df6fb37e7dff2aafbfea10c517b1f5b7575a757a9f76b"}} as unknown as TypedDocumentString<CompleteWorkerChecklistItemMutation, CompleteWorkerChecklistItemMutationVariables>;
export const SkipWorkerChecklistItemDocument = {"__meta__":{"kind":"mutation","name":"SkipWorkerChecklistItem","hash":"sha256:24743c9867706ad0ccef2021c57dd0170aa5093f27aded8784d288f6c188f1f6"}} as unknown as TypedDocumentString<SkipWorkerChecklistItemMutation, SkipWorkerChecklistItemMutationVariables>;
export const MarkWorkerChecklistItemNotApplicableDocument = {"__meta__":{"kind":"mutation","name":"MarkWorkerChecklistItemNotApplicable","hash":"sha256:881762fac545216632939cc7b39f83574634b658f1a4d650eace3af4f2c3cfb4"}} as unknown as TypedDocumentString<MarkWorkerChecklistItemNotApplicableMutation, MarkWorkerChecklistItemNotApplicableMutationVariables>;
export const ReopenWorkerChecklistItemDocument = {"__meta__":{"kind":"mutation","name":"ReopenWorkerChecklistItem","hash":"sha256:f4f4e9e31f071740b4809d176bb8998c3ffb80916dff15e0bb84da655c49aafc"}} as unknown as TypedDocumentString<ReopenWorkerChecklistItemMutation, ReopenWorkerChecklistItemMutationVariables>;
export const CancelWorkerChecklistDocument = {"__meta__":{"kind":"mutation","name":"CancelWorkerChecklist","hash":"sha256:addd413a576790a2a87288624d1cc084e037e2414db1d3d1aac7ef612f045276"}} as unknown as TypedDocumentString<CancelWorkerChecklistMutation, CancelWorkerChecklistMutationVariables>;
export const WorkerCredentialTypeTableDocument = {"__meta__":{"kind":"query","name":"WorkerCredentialTypeTable","hash":"sha256:a89342f74638193feedaa4f3874a5e6f97e1ffbc23d81e9ac2c90024cf21c60d"}} as unknown as TypedDocumentString<WorkerCredentialTypeTableQuery, WorkerCredentialTypeTableQueryVariables>;
export const WorkerCredentialsDocument = {"__meta__":{"kind":"query","name":"WorkerCredentials","hash":"sha256:0320ec07050c260a17fd2849704f8d82aa62d764a60c9b8df3456bbd1ef34d77"}} as unknown as TypedDocumentString<WorkerCredentialsQuery, WorkerCredentialsQueryVariables>;
export const WorkerCredentialSummaryDocument = {"__meta__":{"kind":"query","name":"WorkerCredentialSummary","hash":"sha256:2d9cb3e5cccc1c8c6f1c33e6844d9a40273d024c9516fa232a9150e68db8a5a8"}} as unknown as TypedDocumentString<WorkerCredentialSummaryQuery, WorkerCredentialSummaryQueryVariables>;
export const CredentialExpiryForecastDocument = {"__meta__":{"kind":"query","name":"CredentialExpiryForecast","hash":"sha256:fe715ac7fa2322f8e5ad39e40177591a80100a7d3b1e0703e3c6d7f3090e88a0"}} as unknown as TypedDocumentString<CredentialExpiryForecastQuery, CredentialExpiryForecastQueryVariables>;
export const CreateWorkerCredentialTypeDocument = {"__meta__":{"kind":"mutation","name":"CreateWorkerCredentialType","hash":"sha256:54d0bc77c8a7628f0506d55bd93a86475cc201e9c3e4e3fdf14602a9531fdca2"}} as unknown as TypedDocumentString<CreateWorkerCredentialTypeMutation, CreateWorkerCredentialTypeMutationVariables>;
export const UpdateWorkerCredentialTypeDocument = {"__meta__":{"kind":"mutation","name":"UpdateWorkerCredentialType","hash":"sha256:c5ef7f6e1509b3c318213e649a8a9853c1699797022b5d573f7b8140020868e0"}} as unknown as TypedDocumentString<UpdateWorkerCredentialTypeMutation, UpdateWorkerCredentialTypeMutationVariables>;
export const ArchiveWorkerCredentialTypeDocument = {"__meta__":{"kind":"mutation","name":"ArchiveWorkerCredentialType","hash":"sha256:f42cf88f7f524429829009827cbe4e3db787f33105205d045baa205360e2afa3"}} as unknown as TypedDocumentString<ArchiveWorkerCredentialTypeMutation, ArchiveWorkerCredentialTypeMutationVariables>;
export const RestoreWorkerCredentialTypeDocument = {"__meta__":{"kind":"mutation","name":"RestoreWorkerCredentialType","hash":"sha256:d772d4b964c5adfadbe1c8dee17e5650fc7053d910dbaff21b57d5a41c5d7800"}} as unknown as TypedDocumentString<RestoreWorkerCredentialTypeMutation, RestoreWorkerCredentialTypeMutationVariables>;
export const CreateWorkerCredentialDocument = {"__meta__":{"kind":"mutation","name":"CreateWorkerCredential","hash":"sha256:f232665aad628b3968797ceecb9a4c6a07c92d732dbd526f3dd0e73ce0bcc015"}} as unknown as TypedDocumentString<CreateWorkerCredentialMutation, CreateWorkerCredentialMutationVariables>;
export const UpdateWorkerCredentialDocument = {"__meta__":{"kind":"mutation","name":"UpdateWorkerCredential","hash":"sha256:c5951c750b019fb4b800b253fd747b10ea17c2e90f344e27065b19e95302c9a5"}} as unknown as TypedDocumentString<UpdateWorkerCredentialMutation, UpdateWorkerCredentialMutationVariables>;
export const VerifyWorkerCredentialDocument = {"__meta__":{"kind":"mutation","name":"VerifyWorkerCredential","hash":"sha256:0c01bb1bc452347831f5d0cad19075b7d183fafb2512c618ca4cb9341adce768"}} as unknown as TypedDocumentString<VerifyWorkerCredentialMutation, VerifyWorkerCredentialMutationVariables>;
export const ArchiveWorkerCredentialDocument = {"__meta__":{"kind":"mutation","name":"ArchiveWorkerCredential","hash":"sha256:f2fb9b04e7db4c0c4d0e5ad5fae584f6b22821a0cb94431a0471da4d200f4591"}} as unknown as TypedDocumentString<ArchiveWorkerCredentialMutation, ArchiveWorkerCredentialMutationVariables>;
export const AttachWorkerCredentialDocumentDocument = {"__meta__":{"kind":"mutation","name":"AttachWorkerCredentialDocument","hash":"sha256:c8fdce1ebd201267552bef9db629a8bac5798e0cf8dc4647efb93c4a0b4469e0"}} as unknown as TypedDocumentString<AttachWorkerCredentialDocumentMutation, AttachWorkerCredentialDocumentMutationVariables>;
export const DriverQualificationFileDocument = {"__meta__":{"kind":"query","name":"DriverQualificationFile","hash":"sha256:c327cd209c4906e271ec4c2f0d0467fcd9206ef1e6ba8f3c9390a1bb30ac9a01"}} as unknown as TypedDocumentString<DriverQualificationFileQuery, DriverQualificationFileQueryVariables>;
export const OutstandingEmploymentVerificationsDocument = {"__meta__":{"kind":"query","name":"OutstandingEmploymentVerifications","hash":"sha256:acfb692dd34077cb99269bfc9fb5f58d281c2c77d5be71c225d894ba2d49dec2"}} as unknown as TypedDocumentString<OutstandingEmploymentVerificationsQuery, OutstandingEmploymentVerificationsQueryVariables>;
export const DqfRetentionCandidatesDocument = {"__meta__":{"kind":"query","name":"DqfRetentionCandidates","hash":"sha256:3bbff8bf13ac7e808446b33d7476ca6da5ad3290af1d5bee5937188efd342d2c"}} as unknown as TypedDocumentString<DqfRetentionCandidatesQuery, DqfRetentionCandidatesQueryVariables>;
export const RecordEmploymentVerificationDocument = {"__meta__":{"kind":"mutation","name":"RecordEmploymentVerification","hash":"sha256:f92145b68e20d280b72c9000481c1b5c7ac652b0a1b9bc370c33faaa0f6ba5b2"}} as unknown as TypedDocumentString<RecordEmploymentVerificationMutation, RecordEmploymentVerificationMutationVariables>;
export const UpdateEmploymentVerificationDocument = {"__meta__":{"kind":"mutation","name":"UpdateEmploymentVerification","hash":"sha256:145bc4559dd5e8b8c3a869aa6057b02e051674b5782ce55040eddc67e6f08676"}} as unknown as TypedDocumentString<UpdateEmploymentVerificationMutation, UpdateEmploymentVerificationMutationVariables>;
export const MarkEmploymentVerificationRequestedDocument = {"__meta__":{"kind":"mutation","name":"MarkEmploymentVerificationRequested","hash":"sha256:16f41b9cc2d3ebaa1254a2097fecd51c38068ee66f1adbe498f2352c67201a76"}} as unknown as TypedDocumentString<MarkEmploymentVerificationRequestedMutation, MarkEmploymentVerificationRequestedMutationVariables>;
export const RecordEmploymentVerificationFollowUpDocument = {"__meta__":{"kind":"mutation","name":"RecordEmploymentVerificationFollowUp","hash":"sha256:b99a508cb77a8ca49489c362d29f6bd2f43f2482540d0579e3ef2ac6b565cf5c"}} as unknown as TypedDocumentString<RecordEmploymentVerificationFollowUpMutation, RecordEmploymentVerificationFollowUpMutationVariables>;
export const DeleteEmploymentVerificationDocument = {"__meta__":{"kind":"mutation","name":"DeleteEmploymentVerification","hash":"sha256:2dfa1f2485487765d5685e98d3310fc103a960deac46ed9f7e14f3844ddb3860"}} as unknown as TypedDocumentString<DeleteEmploymentVerificationMutation, DeleteEmploymentVerificationMutationVariables>;
export const WorkerDrugAlcoholFileDocument = {"__meta__":{"kind":"query","name":"WorkerDrugAlcoholFile","hash":"sha256:575ff9dd65911e7520d5a442e061bc04662319ce99c2d94f4edb5ce66db278e5"}} as unknown as TypedDocumentString<WorkerDrugAlcoholFileQuery, WorkerDrugAlcoholFileQueryVariables>;
export const DotRandomPoolsDocument = {"__meta__":{"kind":"query","name":"DotRandomPools","hash":"sha256:d9cce52798e8b3e78e73078994d87c7d71fcfabb190a3d2ae1ab7e1f8dd94e7e"}} as unknown as TypedDocumentString<DotRandomPoolsQuery, DotRandomPoolsQueryVariables>;
export const DotRandomDrawsDocument = {"__meta__":{"kind":"query","name":"DotRandomDraws","hash":"sha256:2b9cf8ff63760434e41734de883989c9ec2899d3622a3fe63e1a4cc8f6961db6"}} as unknown as TypedDocumentString<DotRandomDrawsQuery, DotRandomDrawsQueryVariables>;
export const DotRandomDrawDocument = {"__meta__":{"kind":"query","name":"DotRandomDraw","hash":"sha256:c20b5525dc92b2ed8d5bf6878bf74630bc7675c5af5eb482ab4236ad5877d2b8"}} as unknown as TypedDocumentString<DotRandomDrawQuery, DotRandomDrawQueryVariables>;
export const RecordDotTestDocument = {"__meta__":{"kind":"mutation","name":"RecordDotTest","hash":"sha256:b2908a883b8bec6e5eda71009eeec99b0eeb6023d4c7ea327e87016fd9dafcce"}} as unknown as TypedDocumentString<RecordDotTestMutation, RecordDotTestMutationVariables>;
export const RecordDotTestResultDocument = {"__meta__":{"kind":"mutation","name":"RecordDotTestResult","hash":"sha256:684ac19b7196bb5dcb098af6ceceb9dd060e69fb1752fc20c3dfdce10ef75a01"}} as unknown as TypedDocumentString<RecordDotTestResultMutation, RecordDotTestResultMutationVariables>;
export const CancelDotTestDocument = {"__meta__":{"kind":"mutation","name":"CancelDotTest","hash":"sha256:9ce0182954510643a78156c2ecb154b982a9b50ab4a8caa40a98baf298fca41f"}} as unknown as TypedDocumentString<CancelDotTestMutation, CancelDotTestMutationVariables>;
export const RecordDotViolationDocument = {"__meta__":{"kind":"mutation","name":"RecordDotViolation","hash":"sha256:8c95cc2d8ac8faca249b8dcc7bc2875d491597a955f3fbb5997d8df159f249b7"}} as unknown as TypedDocumentString<RecordDotViolationMutation, RecordDotViolationMutationVariables>;
export const UpdateDotViolationDocument = {"__meta__":{"kind":"mutation","name":"UpdateDotViolation","hash":"sha256:ec1b689b9e0c5d924d73dea13db85488b24001c01689d00ad61027e34702473c"}} as unknown as TypedDocumentString<UpdateDotViolationMutation, UpdateDotViolationMutationVariables>;
export const RecordClearinghouseQueryDocument = {"__meta__":{"kind":"mutation","name":"RecordClearinghouseQuery","hash":"sha256:f409ce268eed9a197ae3e5c92d6a93150573747214db06e3cdf09faa145de65f"}} as unknown as TypedDocumentString<RecordClearinghouseQueryMutation, RecordClearinghouseQueryMutationVariables>;
export const CompleteClearinghouseQueryDocument = {"__meta__":{"kind":"mutation","name":"CompleteClearinghouseQuery","hash":"sha256:4da13d3ff23d4807ed1af51624f533d78688ee58fdec660c9948063383305006"}} as unknown as TypedDocumentString<CompleteClearinghouseQueryMutation, CompleteClearinghouseQueryMutationVariables>;
export const CreateDotRandomPoolDocument = {"__meta__":{"kind":"mutation","name":"CreateDotRandomPool","hash":"sha256:914edd7583c56f609a47ad575220ef612bb948f8a2c632e315d1f70012d63fbd"}} as unknown as TypedDocumentString<CreateDotRandomPoolMutation, CreateDotRandomPoolMutationVariables>;
export const UpdateDotRandomPoolDocument = {"__meta__":{"kind":"mutation","name":"UpdateDotRandomPool","hash":"sha256:ebc12ee29004a2ea08c78d7ac7b347df0ad4f2bb03f69ce8559a4df8e4e06de8"}} as unknown as TypedDocumentString<UpdateDotRandomPoolMutation, UpdateDotRandomPoolMutationVariables>;
export const RunDotRandomDrawDocument = {"__meta__":{"kind":"mutation","name":"RunDotRandomDraw","hash":"sha256:e10793af5487cdd479e723b39754d4a63a01f0cd6180f123f67e4c4bf7422448"}} as unknown as TypedDocumentString<RunDotRandomDrawMutation, RunDotRandomDrawMutationVariables>;
export const FinalizeDotRandomDrawDocument = {"__meta__":{"kind":"mutation","name":"FinalizeDotRandomDraw","hash":"sha256:9a66f7bb547d421b128ab59c142332b95b297d08d0f34a077450ccbbb740b2ca"}} as unknown as TypedDocumentString<FinalizeDotRandomDrawMutation, FinalizeDotRandomDrawMutationVariables>;
export const CancelDotRandomDrawDocument = {"__meta__":{"kind":"mutation","name":"CancelDotRandomDraw","hash":"sha256:3873a5e97d88cd556b9383195485e502cd198d41b034e3945703f12e99531539"}} as unknown as TypedDocumentString<CancelDotRandomDrawMutation, CancelDotRandomDrawMutationVariables>;
export const UpdateDotRandomDrawEntryDocument = {"__meta__":{"kind":"mutation","name":"UpdateDotRandomDrawEntry","hash":"sha256:257f8030a8c6d044ebb9e30aafca59e3908310484dcfb2e09e7fe471a8da063b"}} as unknown as TypedDocumentString<UpdateDotRandomDrawEntryMutation, UpdateDotRandomDrawEntryMutationVariables>;
export const WorkerEmploymentEventsDocument = {"__meta__":{"kind":"query","name":"WorkerEmploymentEvents","hash":"sha256:d8403e85963f946f041a760e19e39811a4913e1fecd0a6a7629c92353724d41f"}} as unknown as TypedDocumentString<WorkerEmploymentEventsQuery, WorkerEmploymentEventsQueryVariables>;
export const RecordWorkerEmploymentEventDocument = {"__meta__":{"kind":"mutation","name":"RecordWorkerEmploymentEvent","hash":"sha256:a1934f179b72af6cceeff8c4541137566522d253469ab84624083777d163d8c1"}} as unknown as TypedDocumentString<RecordWorkerEmploymentEventMutation, RecordWorkerEmploymentEventMutationVariables>;
export const AmendWorkerEmploymentEventDocument = {"__meta__":{"kind":"mutation","name":"AmendWorkerEmploymentEvent","hash":"sha256:99e58a22719473b054e5a4591d2553caf4e1191e1b69bc77f6ea0683bd81edbf"}} as unknown as TypedDocumentString<AmendWorkerEmploymentEventMutation, AmendWorkerEmploymentEventMutationVariables>;
export const WorkerInjuriesDocument = {"__meta__":{"kind":"query","name":"WorkerInjuries","hash":"sha256:28de1e1adc4c00d369501b49ab3033fa8cb7e9511b87b423743f05b9b865b817"}} as unknown as TypedDocumentString<WorkerInjuriesQuery, WorkerInjuriesQueryVariables>;
export const OshaLogDocument = {"__meta__":{"kind":"query","name":"OshaLog","hash":"sha256:002d71fed8f001b6015722e5c3a3c34ac0766cb7d16e5e9bba09c053c4e95acb"}} as unknown as TypedDocumentString<OshaLogQuery, OshaLogQueryVariables>;
export const OshaSummariesDocument = {"__meta__":{"kind":"query","name":"OshaSummaries","hash":"sha256:21dde6660c05c8cb697843a8247948a9cf17f369acd8981b36f1c273ca32793e"}} as unknown as TypedDocumentString<OshaSummariesQuery, OshaSummariesQueryVariables>;
export const RecordWorkerInjuryDocument = {"__meta__":{"kind":"mutation","name":"RecordWorkerInjury","hash":"sha256:be687be20f7cc054ccf0b8427b5b8e7e66d48cc049c3b7dcc52eab7d58adc957"}} as unknown as TypedDocumentString<RecordWorkerInjuryMutation, RecordWorkerInjuryMutationVariables>;
export const UpdateWorkerInjuryDocument = {"__meta__":{"kind":"mutation","name":"UpdateWorkerInjury","hash":"sha256:957e2cc1c02b08496d7cdd56241d790f0f1398283aa53ced6c0167b5743cb744"}} as unknown as TypedDocumentString<UpdateWorkerInjuryMutation, UpdateWorkerInjuryMutationVariables>;
export const DeleteWorkerInjuryDocument = {"__meta__":{"kind":"mutation","name":"DeleteWorkerInjury","hash":"sha256:b080cba57322aaf61aa597b758c6f426370e78e0a4acc91e404531524d5f33e5"}} as unknown as TypedDocumentString<DeleteWorkerInjuryMutation, DeleteWorkerInjuryMutationVariables>;
export const SaveOshaSummaryDocument = {"__meta__":{"kind":"mutation","name":"SaveOshaSummary","hash":"sha256:d3d5a213419b0cf54a16e2b81eae35f7cf60f2adbfbc85eb358ad52e42d91b6b"}} as unknown as TypedDocumentString<SaveOshaSummaryMutation, SaveOshaSummaryMutationVariables>;
export const CertifyOshaSummaryDocument = {"__meta__":{"kind":"mutation","name":"CertifyOshaSummary","hash":"sha256:72ae75efe26bdcf125e68327ec4f64578881dd9c7e6802ec3d901991e8c4c012"}} as unknown as TypedDocumentString<CertifyOshaSummaryMutation, CertifyOshaSummaryMutationVariables>;
export const UncertifyOshaSummaryDocument = {"__meta__":{"kind":"mutation","name":"UncertifyOshaSummary","hash":"sha256:49e498d4ad617c2cc2b0afb6b73cb03872cd0bdfbe103b0b0b560734b6107972"}} as unknown as TypedDocumentString<UncertifyOshaSummaryMutation, UncertifyOshaSummaryMutationVariables>;
export const WorkerLeaveFileDocument = {"__meta__":{"kind":"query","name":"WorkerLeaveFile","hash":"sha256:485100de9cb769981a503c0e5b4b31b9d67dea301c04fd05f713ec10fb814673"}} as unknown as TypedDocumentString<WorkerLeaveFileQuery, WorkerLeaveFileQueryVariables>;
export const LeaveControlDocument = {"__meta__":{"kind":"query","name":"LeaveControl","hash":"sha256:5f97e698c7f6ab2fa43f1be1df7b9b7fff1b6c19b1a616fa1160eb666d0636fc"}} as unknown as TypedDocumentString<LeaveControlQuery, LeaveControlQueryVariables>;
export const OpenLeaveCaseDocument = {"__meta__":{"kind":"mutation","name":"OpenLeaveCase","hash":"sha256:2afe2bf68d5d9f94bbcf1cdfed3398af22f642ac5c55d5eedf23d0ca25ce2a42"}} as unknown as TypedDocumentString<OpenLeaveCaseMutation, OpenLeaveCaseMutationVariables>;
export const UpdateLeaveCaseDocument = {"__meta__":{"kind":"mutation","name":"UpdateLeaveCase","hash":"sha256:eadafa1198956887226ade7e9f1e8f589c3f62cb0f11423a5024b4b7571177ce"}} as unknown as TypedDocumentString<UpdateLeaveCaseMutation, UpdateLeaveCaseMutationVariables>;
export const DecideLeaveCaseDocument = {"__meta__":{"kind":"mutation","name":"DecideLeaveCase","hash":"sha256:b71b3a19666b864226da5e8a0e028a45e3941da9b70762260828669b3c202cb9"}} as unknown as TypedDocumentString<DecideLeaveCaseMutation, DecideLeaveCaseMutationVariables>;
export const CloseLeaveCaseDocument = {"__meta__":{"kind":"mutation","name":"CloseLeaveCase","hash":"sha256:74e52582d97104f1a623eb81a913ec50d74dde0ae19545be6f91c0a5c001dac0"}} as unknown as TypedDocumentString<CloseLeaveCaseMutation, CloseLeaveCaseMutationVariables>;
export const RequestLeaveCertificationDocument = {"__meta__":{"kind":"mutation","name":"RequestLeaveCertification","hash":"sha256:5cb6e293dfce383070ceb1edf8018318375b0aceeba72932571c417184a1e596"}} as unknown as TypedDocumentString<RequestLeaveCertificationMutation, RequestLeaveCertificationMutationVariables>;
export const RecordLeaveCertificationDocument = {"__meta__":{"kind":"mutation","name":"RecordLeaveCertification","hash":"sha256:ca0a71667fdc8ee5c3d449298941d08512e4c909f4f5cfc2e7b2675a1703fe05"}} as unknown as TypedDocumentString<RecordLeaveCertificationMutation, RecordLeaveCertificationMutationVariables>;
export const RecordLeaveDayDocument = {"__meta__":{"kind":"mutation","name":"RecordLeaveDay","hash":"sha256:50d5f1d160645eafecad5d3e5b30f83facf49fff25f4e9d7581599334a604b90"}} as unknown as TypedDocumentString<RecordLeaveDayMutation, RecordLeaveDayMutationVariables>;
export const DeleteLeaveDayDocument = {"__meta__":{"kind":"mutation","name":"DeleteLeaveDay","hash":"sha256:23c110a998d6cad5673bc5e358a902af975db969212e3fd4697e556dd4955184"}} as unknown as TypedDocumentString<DeleteLeaveDayMutation, DeleteLeaveDayMutationVariables>;
export const UpdateLeaveControlDocument = {"__meta__":{"kind":"mutation","name":"UpdateLeaveControl","hash":"sha256:4a3445d070a097d3716d461cce5e688eb21dba166476abde57633257dd03ab32"}} as unknown as TypedDocumentString<UpdateLeaveControlMutation, UpdateLeaveControlMutationVariables>;
export const WorkerOverviewDocument = {"__meta__":{"kind":"query","name":"WorkerOverview","hash":"sha256:35cc7a76b7b2dea3dd46b31cc5d03616056a10f6aa65bd567b27db3dd06fd146"}} as unknown as TypedDocumentString<WorkerOverviewQuery, WorkerOverviewQueryVariables>;
export const WorkerRosterAttentionDocument = {"__meta__":{"kind":"query","name":"WorkerRosterAttention","hash":"sha256:879fe8fa662872397b3a0f0754b44542bf340c53e31fed34221cd2c96b0bce7f"}} as unknown as TypedDocumentString<WorkerRosterAttentionQuery, WorkerRosterAttentionQueryVariables>;
export const WorkerSafetyEventsDocument = {"__meta__":{"kind":"query","name":"WorkerSafetyEvents","hash":"sha256:b52fc45abc77427feb06d193f67f30a82d82f717906b89f69fcf1474b15816e6"}} as unknown as TypedDocumentString<WorkerSafetyEventsQuery, WorkerSafetyEventsQueryVariables>;
export const WorkerSafetyScorecardDocument = {"__meta__":{"kind":"query","name":"WorkerSafetyScorecard","hash":"sha256:f06e4d47f8b4b06a9d12388dac48c7a270e5355b19325a667e5cbf6d85e1b6b6"}} as unknown as TypedDocumentString<WorkerSafetyScorecardQuery, WorkerSafetyScorecardQueryVariables>;
export const WorkerDisciplinaryActionsDocument = {"__meta__":{"kind":"query","name":"WorkerDisciplinaryActions","hash":"sha256:40b26c6b00d04cdddc9f97a895fe1342f22e545e43c48db6046361ad4d82971e"}} as unknown as TypedDocumentString<WorkerDisciplinaryActionsQuery, WorkerDisciplinaryActionsQueryVariables>;
export const WorkerDisciplinaryLadderDocument = {"__meta__":{"kind":"query","name":"WorkerDisciplinaryLadder","hash":"sha256:be5f4847eba8f9564cc4ce3c58c2042fc39dbe531ba8b2cbe55410685fb96e23"}} as unknown as TypedDocumentString<WorkerDisciplinaryLadderQuery, WorkerDisciplinaryLadderQueryVariables>;
export const WorkerRecognitionsDocument = {"__meta__":{"kind":"query","name":"WorkerRecognitions","hash":"sha256:452f5562faad8ba65aa8a89425b2f7bb24a47586a4540422b671e6707c516700"}} as unknown as TypedDocumentString<WorkerRecognitionsQuery, WorkerRecognitionsQueryVariables>;
export const DefaultSafetyPointsDocument = {"__meta__":{"kind":"query","name":"DefaultSafetyPoints","hash":"sha256:9d9719bc77a713b804412ac148244016ed93d3ca7ff3f5f0d6104e9c9363d188"}} as unknown as TypedDocumentString<DefaultSafetyPointsQuery, DefaultSafetyPointsQueryVariables>;
export const CreateWorkerSafetyEventDocument = {"__meta__":{"kind":"mutation","name":"CreateWorkerSafetyEvent","hash":"sha256:fb751baeffc28e7ec239700c1f4319c561edd4e76a7d1e6c325b67a2a2f3542b"}} as unknown as TypedDocumentString<CreateWorkerSafetyEventMutation, CreateWorkerSafetyEventMutationVariables>;
export const UpdateWorkerSafetyEventDocument = {"__meta__":{"kind":"mutation","name":"UpdateWorkerSafetyEvent","hash":"sha256:8f80eec69e783caad246fa1610b18ed83b59ff4661407292c9f9ebedb21ad928"}} as unknown as TypedDocumentString<UpdateWorkerSafetyEventMutation, UpdateWorkerSafetyEventMutationVariables>;
export const CloseWorkerSafetyEventDocument = {"__meta__":{"kind":"mutation","name":"CloseWorkerSafetyEvent","hash":"sha256:4c5c9cb5193084928041bee5cf5f38e2d85d54001e29c691069b2d7d5c30eb38"}} as unknown as TypedDocumentString<CloseWorkerSafetyEventMutation, CloseWorkerSafetyEventMutationVariables>;
export const ReviewWorkerSafetyEventDocument = {"__meta__":{"kind":"mutation","name":"ReviewWorkerSafetyEvent","hash":"sha256:7e92eeab4433a6e2b2d3e348bc7a5133c725c7cd9655fdeccefa2176157840d9"}} as unknown as TypedDocumentString<ReviewWorkerSafetyEventMutation, ReviewWorkerSafetyEventMutationVariables>;
export const ReopenWorkerSafetyEventDocument = {"__meta__":{"kind":"mutation","name":"ReopenWorkerSafetyEvent","hash":"sha256:ba9af3c08eaea14fbec22e142035d6af28a203c5ff5d803a1d55ae7af537babc"}} as unknown as TypedDocumentString<ReopenWorkerSafetyEventMutation, ReopenWorkerSafetyEventMutationVariables>;
export const DeleteWorkerSafetyEventDocument = {"__meta__":{"kind":"mutation","name":"DeleteWorkerSafetyEvent","hash":"sha256:3334719731522d2eb493aa8bea9d0a7952eb47c94f42389d5870d277d5194280"}} as unknown as TypedDocumentString<DeleteWorkerSafetyEventMutation, DeleteWorkerSafetyEventMutationVariables>;
export const IssueDisciplinaryActionDocument = {"__meta__":{"kind":"mutation","name":"IssueDisciplinaryAction","hash":"sha256:5be8f6ff970c73a08b17bbc21d1e35f66d037fcbfde479f8e53c3f885d502f0b"}} as unknown as TypedDocumentString<IssueDisciplinaryActionMutation, IssueDisciplinaryActionMutationVariables>;
export const RescindDisciplinaryActionDocument = {"__meta__":{"kind":"mutation","name":"RescindDisciplinaryAction","hash":"sha256:4ad5c615c6e3ed57b2c4cd3e13017a7b17d750efe5de99b54d2339761af808fc"}} as unknown as TypedDocumentString<RescindDisciplinaryActionMutation, RescindDisciplinaryActionMutationVariables>;
export const GiveWorkerRecognitionDocument = {"__meta__":{"kind":"mutation","name":"GiveWorkerRecognition","hash":"sha256:ec15ad3d7af4fcfad4fdfb3ed21971e56fd9a840ddc5775a7c72a3b9b2d966e3"}} as unknown as TypedDocumentString<GiveWorkerRecognitionMutation, GiveWorkerRecognitionMutationVariables>;
export const DeleteWorkerRecognitionDocument = {"__meta__":{"kind":"mutation","name":"DeleteWorkerRecognition","hash":"sha256:5006d4464e8e52acec5aff0094afbf9f9bc16565fb723eeea3e226f1137f7bf1"}} as unknown as TypedDocumentString<DeleteWorkerRecognitionMutation, DeleteWorkerRecognitionMutationVariables>;
export const TrainingCourseTableDocument = {"__meta__":{"kind":"query","name":"TrainingCourseTable","hash":"sha256:3950fe64badf5834025ac8f16b197ff3d30925f841658c84a7182d628bf5b23d"}} as unknown as TypedDocumentString<TrainingCourseTableQuery, TrainingCourseTableQueryVariables>;
export const ActiveTrainingCoursesDocument = {"__meta__":{"kind":"query","name":"ActiveTrainingCourses","hash":"sha256:3c5b9eb735372bab6586e7cf360b65df874096d4b6c1f3e2cc7fae7e2e831e8e"}} as unknown as TypedDocumentString<ActiveTrainingCoursesQuery, ActiveTrainingCoursesQueryVariables>;
export const WorkerTrainingRecordsDocument = {"__meta__":{"kind":"query","name":"WorkerTrainingRecords","hash":"sha256:6a6491d9e5c4b8ae0f1afa5440cc27d19819ded3caff4f7f1eaa0adf39d09e7d"}} as unknown as TypedDocumentString<WorkerTrainingRecordsQuery, WorkerTrainingRecordsQueryVariables>;
export const WorkerTrainingSummaryDocument = {"__meta__":{"kind":"query","name":"WorkerTrainingSummary","hash":"sha256:00d4ef6d2c6f557b54e7ef711f9e8c7fa4b4fd79edd71f740b08ff4b5f5d7384"}} as unknown as TypedDocumentString<WorkerTrainingSummaryQuery, WorkerTrainingSummaryQueryVariables>;
export const CreateTrainingCourseDocument = {"__meta__":{"kind":"mutation","name":"CreateTrainingCourse","hash":"sha256:5b2713ba86c3352a564491fa7e3947e8b588a932e02566e94b04d7c7e183e0bf"}} as unknown as TypedDocumentString<CreateTrainingCourseMutation, CreateTrainingCourseMutationVariables>;
export const UpdateTrainingCourseDocument = {"__meta__":{"kind":"mutation","name":"UpdateTrainingCourse","hash":"sha256:5290b6ae1bdab71f8bbfd8fe6d174ffde582dd169afaa2c6005507314d3e8084"}} as unknown as TypedDocumentString<UpdateTrainingCourseMutation, UpdateTrainingCourseMutationVariables>;
export const ArchiveTrainingCourseDocument = {"__meta__":{"kind":"mutation","name":"ArchiveTrainingCourse","hash":"sha256:aed6e41f1139cffe56d33f176576f8bc6f2e9919b4325c49e0fdb64d615ed5ba"}} as unknown as TypedDocumentString<ArchiveTrainingCourseMutation, ArchiveTrainingCourseMutationVariables>;
export const RestoreTrainingCourseDocument = {"__meta__":{"kind":"mutation","name":"RestoreTrainingCourse","hash":"sha256:90680b2feabb93cd253c4c7fb1f5dee9f484a831586aa99adabaed6935f23406"}} as unknown as TypedDocumentString<RestoreTrainingCourseMutation, RestoreTrainingCourseMutationVariables>;
export const AssignWorkerTrainingDocument = {"__meta__":{"kind":"mutation","name":"AssignWorkerTraining","hash":"sha256:1c75548ac9021115b51c8583950d23c9147a530611d7fe1ab42e3cf99dc61444"}} as unknown as TypedDocumentString<AssignWorkerTrainingMutation, AssignWorkerTrainingMutationVariables>;
export const BulkAssignTrainingDocument = {"__meta__":{"kind":"mutation","name":"BulkAssignTraining","hash":"sha256:52c6ea0a83af8cd6002c3cbabbba36d803bc7ca805fc041be617808a20052847"}} as unknown as TypedDocumentString<BulkAssignTrainingMutation, BulkAssignTrainingMutationVariables>;
export const AssignRequiredWorkerTrainingDocument = {"__meta__":{"kind":"mutation","name":"AssignRequiredWorkerTraining","hash":"sha256:06f801082bce378d54638ce65cf8f8af60e0bb49c75466f00dd61dc137c64f71"}} as unknown as TypedDocumentString<AssignRequiredWorkerTrainingMutation, AssignRequiredWorkerTrainingMutationVariables>;
export const CompleteWorkerTrainingDocument = {"__meta__":{"kind":"mutation","name":"CompleteWorkerTraining","hash":"sha256:9cb4aa45f2f42cca640e60119d7c8f8a5d8de158f74cf9bd0b4bf2e89b59899f"}} as unknown as TypedDocumentString<CompleteWorkerTrainingMutation, CompleteWorkerTrainingMutationVariables>;
export const WaiveWorkerTrainingDocument = {"__meta__":{"kind":"mutation","name":"WaiveWorkerTraining","hash":"sha256:350fe9c828db8280c73f467546951e0582dbc9aa61ef6d8057142e7183fecf65"}} as unknown as TypedDocumentString<WaiveWorkerTrainingMutation, WaiveWorkerTrainingMutationVariables>;
export const CancelWorkerTrainingDocument = {"__meta__":{"kind":"mutation","name":"CancelWorkerTraining","hash":"sha256:c8cef6816ba0c3c31b816e04eee2d39a6f6d1de48a792899bb04e7c176e0041c"}} as unknown as TypedDocumentString<CancelWorkerTrainingMutation, CancelWorkerTrainingMutationVariables>;
export const AttachWorkerTrainingDocumentDocument = {"__meta__":{"kind":"mutation","name":"AttachWorkerTrainingDocument","hash":"sha256:a75449ecb3e1254bd57fe9990dfd0621f0a95872c7533fe0544c8038d828a66b"}} as unknown as TypedDocumentString<AttachWorkerTrainingDocumentMutation, AttachWorkerTrainingDocumentMutationVariables>;
export const WorkerTableDocument = {"__meta__":{"kind":"query","name":"WorkerTable","hash":"sha256:e9ee7e749dc0eb15e4f0a379d789595467e46cbca4783137b1028ebc7d336b06"}} as unknown as TypedDocumentString<WorkerTableQuery, WorkerTableQueryVariables>;
export const WorkerPtoTableDocument = {"__meta__":{"kind":"query","name":"WorkerPtoTable","hash":"sha256:abf71b65ef60aa73f2ba26a10bd87f0746377d78969a808aec7b3bb07ff60f2f"}} as unknown as TypedDocumentString<WorkerPtoTableQuery, WorkerPtoTableQueryVariables>;
export const UpcomingWorkerPtoDocument = {"__meta__":{"kind":"query","name":"UpcomingWorkerPto","hash":"sha256:3a0547c905dcf4eccb27cd5143630e8104fea9b81e522543feebc26aa4fcf7e0"}} as unknown as TypedDocumentString<UpcomingWorkerPtoQuery, UpcomingWorkerPtoQueryVariables>;
export const WorkerPtoChartDataDocument = {"__meta__":{"kind":"query","name":"WorkerPtoChartData","hash":"sha256:bde52602279b3f61aafc2fced2549700ae1249c36b37078532b4d1ead6266aad"}} as unknown as TypedDocumentString<WorkerPtoChartDataQuery, WorkerPtoChartDataQueryVariables>;
export const PatchWorkerDocument = {"__meta__":{"kind":"mutation","name":"PatchWorker","hash":"sha256:8fd7071d40df508b3377f56fbad7d068151d9a8af449fd88b376acf2878f688b"}} as unknown as TypedDocumentString<PatchWorkerMutation, PatchWorkerMutationVariables>;
export const CreateWorkerPtoDocument = {"__meta__":{"kind":"mutation","name":"CreateWorkerPto","hash":"sha256:2cf4c70d0b8b4f82b6a5ccbbbb99e1bc05925498d9371bceca5a3509478a9aec"}} as unknown as TypedDocumentString<CreateWorkerPtoMutation, CreateWorkerPtoMutationVariables>;
export const UpdateWorkerPtoDocument = {"__meta__":{"kind":"mutation","name":"UpdateWorkerPto","hash":"sha256:f29f960dbc7833263f8cfac3f63dcba9c9ff587043929c85964cb9912f27af36"}} as unknown as TypedDocumentString<UpdateWorkerPtoMutation, UpdateWorkerPtoMutationVariables>;
export const ApproveWorkerPtoDocument = {"__meta__":{"kind":"mutation","name":"ApproveWorkerPto","hash":"sha256:984e5e9bd25d5f8869225e1f21b764d3166c7d215203ea0eb1de40891715ccfa"}} as unknown as TypedDocumentString<ApproveWorkerPtoMutation, ApproveWorkerPtoMutationVariables>;
export const RejectWorkerPtoDocument = {"__meta__":{"kind":"mutation","name":"RejectWorkerPto","hash":"sha256:17f8d639b8658d8cd0dc7db608ee8f03e87d5b1ef7dad84fc967a835a5df2faa"}} as unknown as TypedDocumentString<RejectWorkerPtoMutation, RejectWorkerPtoMutationVariables>;
export const CancelWorkerPtoDocument = {"__meta__":{"kind":"mutation","name":"CancelWorkerPto","hash":"sha256:6dcffdea8c84e24ac4a96933a1e8fe8a150f7ad7bd70f20c520dc094cfa84557"}} as unknown as TypedDocumentString<CancelWorkerPtoMutation, CancelWorkerPtoMutationVariables>;
export const BulkWorkerPtoActionDocument = {"__meta__":{"kind":"mutation","name":"BulkWorkerPtoAction","hash":"sha256:37dae0cea5b6c223053ca79228ca78aaebad1c1892e70ce648d64412bbc9526b"}} as unknown as TypedDocumentString<BulkWorkerPtoActionMutation, BulkWorkerPtoActionMutationVariables>;