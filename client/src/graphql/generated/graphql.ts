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

export type AssignmentStatus =
  | 'Canceled'
  | 'Completed'
  | 'InProgress'
  | 'New';

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

export type DataTableConnectionInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
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

export type EmailProfileStatus =
  | 'Active'
  | 'Inactive';

export type EmailProvider =
  | 'Postmark'
  | 'Resend';

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

export type FiscalYearStatus =
  | 'Closed'
  | 'Draft'
  | 'Open'
  | 'PermanentlyClosed';

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

export type HoldSeverity =
  | 'Advisory'
  | 'Blocking'
  | 'Informational';

export type HoldType =
  | 'ComplianceHold'
  | 'CustomerHold'
  | 'FinanceHold'
  | 'OperationalHold';

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

export type NotificationPriority =
  | 'critical'
  | 'high'
  | 'low'
  | 'medium';

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

export type RateTableLookupType =
  | 'Exact'
  | 'Range';

export type RateUnit =
  | 'Day'
  | 'Hour'
  | 'Mile'
  | 'Stop';

export type RemoveOrderChargeInput = {
  chargeId: string | number;
  orderId: string | number;
};

export type SegregationType =
  | 'Barrier'
  | 'Distance'
  | 'Prohibited'
  | 'Separated';

export type SelectOptionResource =
  | 'EDI_TRANSFER'
  | 'EQUIPMENT_MANUFACTURER'
  | 'EQUIPMENT_TYPE'
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

export type ShipmentAdditionalChargeInput = {
  accessorialChargeId: string | number;
  amount?: string | null | undefined;
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
  after?: string | null | undefined;
  expandShipmentDetails?: boolean | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
  status?: string | null | undefined;
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

export type UpdateOrderChargeInput = {
  amount: string;
  chargeId: string | number;
  description: string;
  orderId: string | number;
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

export type ApiKeyTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, name: string, description: string, keyPrefix: string, status: string, expiresAt: number, lastUsedAt: number, permissionScope: string, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'ApiKeyTableRowFieldsFragment' };

export type ApiKeyTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type ApiKeyTableQuery = { apiKeys: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ApiKeyTableRowFieldsFragment': ApiKeyTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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

export type CustomFieldDefinitionTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, resourceType: string, name: string, label: string, description: string | null, fieldType: FieldType, isRequired: boolean, isActive: boolean, displayOrder: number, color: string | null, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CustomFieldDefinitionTableRowFieldsFragment' };

export type CustomFieldDefinitionTableQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type CustomFieldDefinitionTableQuery = { customFieldDefinitions: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'CustomFieldDefinitionTableRowFieldsFragment': CustomFieldDefinitionTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type CustomerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, stateId: string, status: EntityStatus, code: string, name: string, city: string | null, postalCode: string, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'CustomerTableRowFieldsFragment' };

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

export type FiscalYearTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, status: FiscalYearStatus, year: number, name: string, description: string, startDate: number, endDate: number, isCurrent: boolean, isCalendarYear: boolean, allowAdjustingEntries: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'FiscalYearTableRowFieldsFragment' };

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

export type NotificationFieldsFragment = { id: string, organizationId: string, businessUnitId: string | null, targetUserId: string | null, eventType: string, priority: NotificationPriority, channel: NotificationChannel, title: string, message: string, data: unknown, source: string, readAt: number | null, createdAt: number } & { ' $fragmentName'?: 'NotificationFieldsFragment' };

export type NotificationListQueryVariables = Exact<{
  input: DataTableConnectionInput;
}>;


export type NotificationListQuery = { notifications: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'NotificationFieldsFragment': NotificationFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type NotificationUnreadCountQueryVariables = Exact<{ [key: string]: never; }>;


export type NotificationUnreadCountQuery = { notificationUnreadCount: number };

export type MarkNotificationsReadMutationVariables = Exact<{
  ids: Array<string | number> | string | number;
}>;


export type MarkNotificationsReadMutation = { markNotificationsRead: boolean };

export type MarkAllNotificationsReadMutationVariables = Exact<{ [key: string]: never; }>;


export type MarkAllNotificationsReadMutation = { markAllNotificationsRead: boolean };

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

export type ShipmentTrailerFieldsFragment = { id: string, code: string } & { ' $fragmentName'?: 'ShipmentTrailerFieldsFragment' };

export type ShipmentAssignmentFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentMoveId: string | null, primaryWorkerId: string | null, tractorId: string | null, trailerId: string | null, secondaryWorkerId: string | null, status: AssignmentStatus, archivedAt: number | null, version: number, createdAt: number, updatedAt: number, tractor: { ' $fragmentRefs'?: { 'ShipmentTractorFieldsFragment': ShipmentTractorFieldsFragment } } | null, trailer: { ' $fragmentRefs'?: { 'ShipmentTrailerFieldsFragment': ShipmentTrailerFieldsFragment } } | null, primaryWorker: { ' $fragmentRefs'?: { 'ShipmentWorkerFieldsFragment': ShipmentWorkerFieldsFragment } } | null, secondaryWorker: { ' $fragmentRefs'?: { 'ShipmentWorkerFieldsFragment': ShipmentWorkerFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentAssignmentFieldsFragment' };

export type ShipmentStopFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentMoveId: string | null, locationId: string, status: StopStatus, type: StopType, scheduleType: StopScheduleType, sequence: number, pieces: number | null, weight: number | null, scheduledWindowStart: number, scheduledWindowEnd: number | null, actualArrival: number | null, actualDeparture: number | null, countLateOverride: boolean | null, countDetentionOverride: boolean | null, addressLine: string, version: number, createdAt: number, updatedAt: number, location: { ' $fragmentRefs'?: { 'ShipmentLocationFieldsFragment': ShipmentLocationFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentStopFieldsFragment' };

export type ShipmentMoveFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string | null, status: MoveStatus, loaded: boolean, sequence: number, distance: number | null, distanceSource: string | null, distanceProvider: string | null, distanceCalculatedAt: number | null, distanceRouteSignature: string | null, distanceDataVersion: string | null, distanceRoutingType: string | null, distanceUnits: string | null, distanceMetadata: unknown, version: number, createdAt: number, updatedAt: number, stops: Array<{ ' $fragmentRefs'?: { 'ShipmentStopFieldsFragment': ShipmentStopFieldsFragment } }>, assignment: { ' $fragmentRefs'?: { 'ShipmentAssignmentFieldsFragment': ShipmentAssignmentFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentMoveFieldsFragment' };

export type ShipmentAdditionalChargeFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string, accessorialChargeId: string, isSystemGenerated: boolean, method: string, amount: string, unit: number, version: number, createdAt: number, updatedAt: number, accessorialCharge: { id: string, businessUnitId: string, organizationId: string, code: string, description: string, status: EntityStatus, method: string, rateUnit: string, amount: string, version: number, createdAt: number, updatedAt: number } | null } & { ' $fragmentName'?: 'ShipmentAdditionalChargeFieldsFragment' };

export type ShipmentCommodityFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string, commodityId: string, pieces: number, weight: number, version: number, createdAt: number, updatedAt: number, commodity: { id: string, businessUnitId: string, organizationId: string, hazardousMaterialId: string | null, status: EntityStatus, name: string, description: string, minTemperature: number | null, maxTemperature: number | null, weightPerUnit: number | null, linearFeetPerUnit: number | null, maxQuantityPerShipment: number | null, freightClass: string, loadingInstructions: string, stackable: boolean, fragile: boolean, version: number, createdAt: number, updatedAt: number } | null } & { ' $fragmentName'?: 'ShipmentCommodityFieldsFragment' };

export type ShipmentRatingDetailFieldsFragment = { formulaTemplateId: string, formulaTemplateName: string, expression: string, resolvedVariables: unknown, result: number, ratedAt: number } & { ' $fragmentName'?: 'ShipmentRatingDetailFieldsFragment' };

export type ShipmentFieldsFragment = { id: string, businessUnitId: string, organizationId: string, sourceDocumentId: string | null, serviceTypeId: string, shipmentTypeId: string, customerId: string, tractorTypeId: string | null, trailerTypeId: string | null, ownerId: string | null, enteredById: string | null, canceledById: string | null, formulaTemplateId: string, consolidationGroupId: string | null, orderId: string | null, orderNumber: string | null, orderStatus: OrderStatus | null, status: ShipmentStatus, tenderStatus: ShipmentTenderStatus | null, entryMethod: ShipmentEntryMethod | null, proNumber: string, bol: string | null, cancelReason: string, otherChargeAmount: string, freightChargeAmount: string, baseRate: string, totalChargeAmount: string, pieces: number | null, weight: number | null, temperatureMin: number | null, temperatureMax: number | null, actualDeliveryDate: number | null, actualShipDate: number | null, canceledAt: number | null, billingTransferStatus: string | null, transferredToBillingAt: number | null, markedReadyToBillAt: number | null, billedAt: number | null, ratingUnit: number, version: number, createdAt: number, updatedAt: number, ratingDetail: { ' $fragmentRefs'?: { 'ShipmentRatingDetailFieldsFragment': ShipmentRatingDetailFieldsFragment } } | null, moves: Array<{ ' $fragmentRefs'?: { 'ShipmentMoveFieldsFragment': ShipmentMoveFieldsFragment } }>, additionalCharges: Array<{ ' $fragmentRefs'?: { 'ShipmentAdditionalChargeFieldsFragment': ShipmentAdditionalChargeFieldsFragment } }>, commodities: Array<{ ' $fragmentRefs'?: { 'ShipmentCommodityFieldsFragment': ShipmentCommodityFieldsFragment } }>, customer: { id: string, businessUnitId: string, organizationId: string, stateId: string, status: EntityStatus, code: string, name: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, isGeocoded: boolean, longitude: number | null, latitude: number | null, placeId: string, externalId: string, allowConsolidation: boolean, exclusiveConsolidation: boolean, consolidationPriority: number, version: number, createdAt: number, updatedAt: number } | null, owner: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, formulaTemplate: { id: string, organizationId: string, businessUnitId: string, name: string, description: string, type: string, expression: string, status: string, schemaId: string, metadata: unknown, version: number, sourceTemplateId: string | null, sourceVersionNumber: number | null, currentVersionNumber: number, createdAt: number, updatedAt: number, variableDefinitions: Array<{ name: string, type: string, description: string, required: boolean, defaultValue: unknown, source: string | null }> } | null } & { ' $fragmentName'?: 'ShipmentFieldsFragment' };

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


export type ShipmentPageAnalyticsQuery = { shipmentAnalytics: { page: string, savedViewCounts: { all: number | null, transit: number | null, atRisk: number | null, unassigned: number | null, deliveringToday: number | null } | null, activeShipments: { count: number, changeFromYesterday: number, sparkline: Array<{ hour: string, value: number }>, breakdown: { inTransit: number, atRisk: number, loading: number, done: number } } | null, onTimePercent: { percent: number, onTimeCount: number, totalCount: number, target: number | null, deltaPp: number, sevenDayPercent: number } | null, revenueToday: { total: number, deltaPct: number, rpm: number, sparkline: Array<{ hour: string, value: number }> } | null, emptyMilePercent: { percent: number, emptyMiles: number, totalMiles: number, deltaPp: number } | null, atRisk: { count: number, delta: number, etaSlip: number, weather: number, reefer: number } | null, unassigned: { count: number, delta: number, revenueWaiting: number } | null, readyToDispatch: { count: number, delta: number, unassigned: number, driverReady: number } | null, detentionWatchlist: { items: Array<{ shipmentId: string, customer: string, dwellLabel: string, tone: string }> } | null, customerMix: { windowDays: number, entries: Array<{ customerId: string, name: string, revenue: number, share: number, loads: number, trend: number }> } | null, tomorrowsPickups: { date: string, pickups: Array<{ shipmentId: string, proNumber: string, pickupWindowStart: number, customer: string, origin: string, destination: string, driver: string, status: string }> } | null, laneHeatmap: { windowDays: number, total: number, cells: Array<{ origin: string, destination: string, count: number }> } | null } };

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


export type CalculateShipmentTotalsMutation = { calculateShipmentTotals: { freightChargeAmount: string, otherChargeAmount: string, totalChargeAmount: string } };

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

export type TableConfigurationFieldsFragment = { id: string, organizationId: string, businessUnitId: string, userId: string, name: string, description: string, resource: string, tableConfig: unknown, visibility: ConfigurationVisibility, isDefault: boolean, version: number, createdAt: number, updatedAt: number } & { ' $fragmentName'?: 'TableConfigurationFieldsFragment' };

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
export const CustomerTableRowFieldsFragmentDoc = new TypedDocumentString(`
    fragment CustomerTableRowFields on Customer {
  id
  businessUnitId
  organizationId
  stateId
  status
  code
  name
  city
  postalCode
  version
  createdAt
  updatedAt
}
    `, {"fragmentName":"CustomerTableRowFields"}) as unknown as TypedDocumentString<CustomerTableRowFieldsFragment, unknown>;
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
}
    `, {"fragmentName":"FiscalYearTableRowFields"}) as unknown as TypedDocumentString<FiscalYearTableRowFieldsFragment, unknown>;
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
  source
  readAt
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
  version
  createdAt
  updatedAt
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
    fragment CustomerTableRowFields on Customer {
  id
  businessUnitId
  organizationId
  stateId
  status
  code
  name
  city
  postalCode
  version
  createdAt
  updatedAt
}
fragment DataTablePageInfoFields on PageInfo {
  hasNextPage
  endCursor
}`, {"hash":"sha256:5ac7ef474cb02cd3fca00e2b844961604af9f174a0cc8e792838273aa9df80ee"}) as unknown as TypedDocumentString<CustomerTableQuery, CustomerTableQueryVariables>;
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
}`, {"hash":"sha256:b201f867a66edbfce1f3b205c8cfe6b1a7882dc5896b6b3b2e149a9ed1d654b4"}) as unknown as TypedDocumentString<FiscalYearTableQuery, FiscalYearTableQueryVariables>;
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
    query NotificationList($input: DataTableConnectionInput!) {
  notifications(input: $input) {
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
  source
  readAt
  createdAt
}`, {"hash":"sha256:521222189fc8fc082707badddfab4a76f8bb053d4603d33f06ad0c47cc64d5c0"}) as unknown as TypedDocumentString<NotificationListQuery, NotificationListQueryVariables>;
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
export const MarkAllNotificationsReadDocument = new TypedDocumentString(`
    mutation MarkAllNotificationsRead {
  markAllNotificationsRead
}
    `, {"hash":"sha256:e919497b911d73638f8329785ecb0b4b48a247bb6d037bf89b2a498c5bca336d"}) as unknown as TypedDocumentString<MarkAllNotificationsReadMutation, MarkAllNotificationsReadMutationVariables>;
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
}`, {"hash":"sha256:f5cd9ca38789950ae0acecb4d1437f831a9ecc0ffbe1f0128329e8768fce0c5c"}) as unknown as TypedDocumentString<ShipmentCommandCenterTableQuery, ShipmentCommandCenterTableQueryVariables>;
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
}`, {"hash":"sha256:7140985f0760018f1faf7704de63168439a5c54d413497ca66def9eb10d863ae"}) as unknown as TypedDocumentString<ShipmentDetailQuery, ShipmentDetailQueryVariables>;
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
    `, {"hash":"sha256:2fdb047822c415d2359c27fae392ff5cd6db97f04c3a53aaf25d0af6c04125fd"}) as unknown as TypedDocumentString<ShipmentPageAnalyticsQuery, ShipmentPageAnalyticsQueryVariables>;
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
}`, {"hash":"sha256:259f6742486bf7a2f4cfd44a00c303d0ebcde9f379f8bc944e27331dedc78db0"}) as unknown as TypedDocumentString<UnassignedShipmentsQuery, UnassignedShipmentsQueryVariables>;
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
}`, {"hash":"sha256:de42ed156eb13c6b6cc8d1b3bc0ecf85dca6836909b7a91129fff1fe4747a010"}) as unknown as TypedDocumentString<ExceptionShipmentsQuery, ExceptionShipmentsQueryVariables>;
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
}`, {"hash":"sha256:2e34058deb77b3c563f1e11666c062c1c7e6e76b1d93bb238d368841d0f92ace"}) as unknown as TypedDocumentString<MapShipmentsQuery, MapShipmentsQueryVariables>;
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
}`, {"hash":"sha256:f93be74cadf7023b6128fefa39182823c0ca1fe0182fb5fa6ff1ada9fb6143d8"}) as unknown as TypedDocumentString<CreateShipmentMutation, CreateShipmentMutationVariables>;
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
}`, {"hash":"sha256:7fe07e15add8878c4054607895a727b9d1a203744fa1aad8130a3143e33014bf"}) as unknown as TypedDocumentString<UpdateShipmentMutation, UpdateShipmentMutationVariables>;
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
}`, {"hash":"sha256:5a11fe2606f52b3ce83d31f1fee592ff6de431bd22a97ab827abaa7787acef11"}) as unknown as TypedDocumentString<CancelShipmentMutation, CancelShipmentMutationVariables>;
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
}`, {"hash":"sha256:9da085bfa759ea5472ecaf9920de9b68f66c5ce60de7bfbe64372d5b0f3d89fd"}) as unknown as TypedDocumentString<UncancelShipmentMutation, UncancelShipmentMutationVariables>;
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
}`, {"hash":"sha256:96d83971d324ca9c7703920a5ec1c799ae2ea7c82397a5146ffdefc1cbd14b81"}) as unknown as TypedDocumentString<TransferShipmentOwnershipMutation, TransferShipmentOwnershipMutationVariables>;
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
  }
}
    `, {"hash":"sha256:43a6fc69562eda15d32d926f91ac7b9256b587b90c05d7dc4d5c964c0bc000b1"}) as unknown as TypedDocumentString<CalculateShipmentTotalsMutation, CalculateShipmentTotalsMutationVariables>;
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
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:021fa0f568747a3357e4b59ba635e5c80bdc5dc0ce28118e04e13d31c02931d4"}) as unknown as TypedDocumentString<TableConfigurationTableQuery, TableConfigurationTableQueryVariables>;
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
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:f5455dd64d6ae87c336154b63560ddd2f1a985d4eadfa843bcd18851517be0e5"}) as unknown as TypedDocumentString<DefaultTableConfigurationQuery, DefaultTableConfigurationQueryVariables>;
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
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:8b96b72329790f8c9ac9ad3392ff7e757e40951be3454e0a3bbf9635105fa359"}) as unknown as TypedDocumentString<TableConfigurationDetailQuery, TableConfigurationDetailQueryVariables>;
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
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:086f96a9688283d2b9077f23d851257bca7c1bb7718c7dee67e567d38c56aa74"}) as unknown as TypedDocumentString<CreateTableConfigurationMutation, CreateTableConfigurationMutationVariables>;
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
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:d031325d006875d779bce3b042ab8ae49adc6b7a35e7a31ee0208c2bcb829a8b"}) as unknown as TypedDocumentString<UpdateTableConfigurationMutation, UpdateTableConfigurationMutationVariables>;
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
  version
  createdAt
  updatedAt
}`, {"hash":"sha256:66c7e28c908e27654cc05dea05d78d7eab9c14ded06a12228376670fc3f80d8b"}) as unknown as TypedDocumentString<SetDefaultTableConfigurationMutation, SetDefaultTableConfigurationMutationVariables>;
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