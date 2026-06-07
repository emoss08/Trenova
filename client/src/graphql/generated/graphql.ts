/* eslint-disable */
/** Internal type. DO NOT USE DIRECTLY. */
type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
/** Internal type. DO NOT USE DIRECTLY. */
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
import type { DocumentTypeDecoration } from '@graphql-typed-document-node/core';
export type AssignmentStatus =
  | 'Canceled'
  | 'Completed'
  | 'InProgress'
  | 'New';

export type BillType =
  | 'CreditMemo'
  | 'DebitMemo'
  | 'Invoice';

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

export type DataTableConnectionInput = {
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | null | undefined;
  filterGroups?: Array<FilterGroupInput> | null | undefined;
  first?: number | null | undefined;
  query?: string | null | undefined;
  sort?: Array<SortFieldInput> | null | undefined;
};

export type DriverType =
  | 'Local'
  | 'OTR'
  | 'Regional'
  | 'Team';

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

export type FieldFilterInput = {
  field: string;
  operator: string;
  value?: unknown;
};

export type FilterGroupInput = {
  filters: Array<FieldFilterInput>;
};

export type MoveStatus =
  | 'Assigned'
  | 'Canceled'
  | 'Completed'
  | 'InTransit'
  | 'New';

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

export type SelectOptionResource =
  | 'EQUIPMENT_MANUFACTURER'
  | 'EQUIPMENT_TYPE'
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

export type WorkerGender =
  | 'Female'
  | 'Male';

export type WorkerPatchInput = {
  driverType?: DriverType | null | undefined;
  status?: EntityStatus | null | undefined;
  type?: WorkerType | null | undefined;
};

export type WorkerType =
  | 'Contractor'
  | 'Employee';

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
  first: number;
  after?: string | null | undefined;
  query?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | FieldFilterInput | null | undefined;
  filterGroups?: Array<FilterGroupInput> | FilterGroupInput | null | undefined;
  sort?: Array<SortFieldInput> | SortFieldInput | null | undefined;
  includeEquipmentDetails?: boolean | null | undefined;
  includeFleetDetails?: boolean | null | undefined;
  includeWorkerDetails?: boolean | null | undefined;
}>;


export type TractorTableQuery = { tractors: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TractorTableRowFieldsFragment': TractorTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

export type TrailerTableQueryVariables = Exact<{
  first: number;
  after?: string | null | undefined;
  query?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | FieldFilterInput | null | undefined;
  filterGroups?: Array<FilterGroupInput> | FilterGroupInput | null | undefined;
  sort?: Array<SortFieldInput> | SortFieldInput | null | undefined;
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

export type SelectOptionsQueryVariables = Exact<{
  input: SelectOptionsInput;
}>;


export type SelectOptionsQuery = { selectOptions: { totalCount: number | null, edges: Array<{ cursor: string, node: { id: string, label: string, description: string | null, meta: unknown } }>, pageInfo: { hasNextPage: boolean, endCursor: string | null } } };

export type ShipmentUserFieldsFragment = { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } & { ' $fragmentName'?: 'ShipmentUserFieldsFragment' };

export type ShipmentLocationFieldsFragment = { id: string, name: string, code: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, longitude: number | null, latitude: number | null } & { ' $fragmentName'?: 'ShipmentLocationFieldsFragment' };

export type ShipmentWorkerFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string, profilePicUrl: string } & { ' $fragmentName'?: 'ShipmentWorkerFieldsFragment' };

export type ShipmentTractorFieldsFragment = { id: string, code: string } & { ' $fragmentName'?: 'ShipmentTractorFieldsFragment' };

export type ShipmentTrailerFieldsFragment = { id: string, code: string } & { ' $fragmentName'?: 'ShipmentTrailerFieldsFragment' };

export type ShipmentAssignmentFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentMoveId: string | null, primaryWorkerId: string | null, tractorId: string | null, trailerId: string | null, secondaryWorkerId: string | null, status: AssignmentStatus, archivedAt: number | null, version: number, createdAt: number, updatedAt: number, tractor: { ' $fragmentRefs'?: { 'ShipmentTractorFieldsFragment': ShipmentTractorFieldsFragment } } | null, trailer: { ' $fragmentRefs'?: { 'ShipmentTrailerFieldsFragment': ShipmentTrailerFieldsFragment } } | null, primaryWorker: { ' $fragmentRefs'?: { 'ShipmentWorkerFieldsFragment': ShipmentWorkerFieldsFragment } } | null, secondaryWorker: { ' $fragmentRefs'?: { 'ShipmentWorkerFieldsFragment': ShipmentWorkerFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentAssignmentFieldsFragment' };

export type ShipmentStopFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentMoveId: string | null, locationId: string, status: StopStatus, type: StopType, scheduleType: StopScheduleType, sequence: number, pieces: number | null, weight: number | null, scheduledWindowStart: number, scheduledWindowEnd: number | null, actualArrival: number | null, actualDeparture: number | null, countLateOverride: boolean | null, countDetentionOverride: boolean | null, addressLine: string, version: number, createdAt: number, updatedAt: number, location: { ' $fragmentRefs'?: { 'ShipmentLocationFieldsFragment': ShipmentLocationFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentStopFieldsFragment' };

export type ShipmentMoveFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string | null, status: MoveStatus, loaded: boolean, sequence: number, distance: number | null, distanceSource: string | null, distanceProvider: string | null, distanceCalculatedAt: number | null, distanceRouteSignature: string | null, distanceDataVersion: string | null, distanceRoutingType: string | null, distanceUnits: string | null, distanceMetadata: unknown, version: number, createdAt: number, updatedAt: number, stops: Array<{ ' $fragmentRefs'?: { 'ShipmentStopFieldsFragment': ShipmentStopFieldsFragment } }>, assignment: { ' $fragmentRefs'?: { 'ShipmentAssignmentFieldsFragment': ShipmentAssignmentFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentMoveFieldsFragment' };

export type ShipmentAdditionalChargeFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string, accessorialChargeId: string, isSystemGenerated: boolean, method: string, amount: string, unit: number, version: number, createdAt: number, updatedAt: number, accessorialCharge: { id: string, businessUnitId: string, organizationId: string, code: string, description: string, status: EntityStatus, method: string, rateUnit: string, amount: string, version: number, createdAt: number, updatedAt: number } | null } & { ' $fragmentName'?: 'ShipmentAdditionalChargeFieldsFragment' };

export type ShipmentCommodityFieldsFragment = { id: string | null, businessUnitId: string, organizationId: string, shipmentId: string, commodityId: string, pieces: number, weight: number, version: number, createdAt: number, updatedAt: number, commodity: { id: string, businessUnitId: string, organizationId: string, hazardousMaterialId: string | null, status: EntityStatus, name: string, description: string, minTemperature: number | null, maxTemperature: number | null, weightPerUnit: number | null, linearFeetPerUnit: number | null, maxQuantityPerShipment: number | null, freightClass: string, loadingInstructions: string, stackable: boolean, fragile: boolean, version: number, createdAt: number, updatedAt: number } | null } & { ' $fragmentName'?: 'ShipmentCommodityFieldsFragment' };

export type ShipmentRatingDetailFieldsFragment = { formulaTemplateId: string, formulaTemplateName: string, expression: string, resolvedVariables: unknown, result: number, ratedAt: number } & { ' $fragmentName'?: 'ShipmentRatingDetailFieldsFragment' };

export type ShipmentFieldsFragment = { id: string, businessUnitId: string, organizationId: string, sourceDocumentId: string | null, serviceTypeId: string, shipmentTypeId: string, customerId: string, tractorTypeId: string | null, trailerTypeId: string | null, ownerId: string | null, enteredById: string | null, canceledById: string | null, formulaTemplateId: string, consolidationGroupId: string | null, status: ShipmentStatus, tenderStatus: ShipmentTenderStatus | null, entryMethod: ShipmentEntryMethod | null, proNumber: string, bol: string | null, cancelReason: string, otherChargeAmount: string, freightChargeAmount: string, baseRate: string, totalChargeAmount: string, pieces: number | null, weight: number | null, temperatureMin: number | null, temperatureMax: number | null, actualDeliveryDate: number | null, actualShipDate: number | null, canceledAt: number | null, billingTransferStatus: string | null, transferredToBillingAt: number | null, markedReadyToBillAt: number | null, billedAt: number | null, ratingUnit: number, version: number, createdAt: number, updatedAt: number, ratingDetail: { ' $fragmentRefs'?: { 'ShipmentRatingDetailFieldsFragment': ShipmentRatingDetailFieldsFragment } } | null, moves: Array<{ ' $fragmentRefs'?: { 'ShipmentMoveFieldsFragment': ShipmentMoveFieldsFragment } }>, additionalCharges: Array<{ ' $fragmentRefs'?: { 'ShipmentAdditionalChargeFieldsFragment': ShipmentAdditionalChargeFieldsFragment } }>, commodities: Array<{ ' $fragmentRefs'?: { 'ShipmentCommodityFieldsFragment': ShipmentCommodityFieldsFragment } }>, customer: { id: string, businessUnitId: string, organizationId: string, stateId: string, status: EntityStatus, code: string, name: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, isGeocoded: boolean, longitude: number | null, latitude: number | null, placeId: string, externalId: string, allowConsolidation: boolean, exclusiveConsolidation: boolean, consolidationPriority: number, version: number, createdAt: number, updatedAt: number } | null, owner: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, formulaTemplate: { id: string, organizationId: string, businessUnitId: string, name: string, description: string, type: string, expression: string, status: string, schemaId: string, metadata: unknown, version: number, sourceTemplateId: string | null, sourceVersionNumber: number | null, currentVersionNumber: number, createdAt: number, updatedAt: number, variableDefinitions: Array<{ name: string, type: string, description: string, required: boolean, defaultValue: unknown, source: string | null }> } | null } & { ' $fragmentName'?: 'ShipmentFieldsFragment' };

export type ShipmentPageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'ShipmentPageInfoFieldsFragment' };

export type ShipmentCommentMentionFieldsFragment = { id: string, commentId: string, mentionedUserId: string, organizationId: string | null, businessUnitId: string | null, shipmentId: string | null, createdAt: number, mentionedUser: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null } & { ' $fragmentName'?: 'ShipmentCommentMentionFieldsFragment' };

export type ShipmentCommentFieldsFragment = { id: string, businessUnitId: string | null, organizationId: string | null, shipmentId: string, userId: string | null, comment: string, type: ShipmentCommentType, visibility: ShipmentCommentVisibility, priority: ShipmentCommentPriority, source: ShipmentCommentSource, metadata: unknown, editedAt: number | null, version: number, createdAt: number, updatedAt: number, mentionedUserIds: Array<string>, user: { ' $fragmentRefs'?: { 'ShipmentUserFieldsFragment': ShipmentUserFieldsFragment } } | null, mentionedUsers: Array<{ ' $fragmentRefs'?: { 'ShipmentCommentMentionFieldsFragment': ShipmentCommentMentionFieldsFragment } }> | null } & { ' $fragmentName'?: 'ShipmentCommentFieldsFragment' };

export type ShipmentEventFieldsFragment = { id: string, organizationId: string, businessUnitId: string, shipmentId: string, moveId: string | null, stopId: string | null, assignmentId: string | null, commentId: string | null, holdId: string | null, type: ShipmentEventType, severity: ShipmentEventSeverity, actorType: ShipmentEventActorType, actorId: string | null, actorLabel: string, summary: string, proNumber: string | null, previousStatus: string | null, newStatus: string | null, reason: string | null, previousOwnerId: string | null, newOwnerId: string | null, primaryWorkerId: string | null, secondaryWorkerId: string | null, tractorId: string | null, trailerId: string | null, driverName: string | null, holdType: string | null, holdSeverity: string | null, holdSource: string | null, commentBody: string | null, commentType: string | null, commentVisibility: string | null, commentPriority: string | null, mentionedUserIds: Array<string>, metadata: unknown, occurredAt: number, correlationId: string | null, actor: { id: string, name: string, emailAddress: string, profilePicUrl: string, thumbnailUrl: string } | null, shipment: { id: string | null, proNumber: string | null } | null } & { ' $fragmentName'?: 'ShipmentEventFieldsFragment' };

export type ShipmentCommandCenterTableQueryVariables = Exact<{
  first: number;
  after?: string | null | undefined;
  query?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | FieldFilterInput | null | undefined;
  filterGroups?: Array<FilterGroupInput> | FilterGroupInput | null | undefined;
  sort?: Array<SortFieldInput> | SortFieldInput | null | undefined;
  expandShipmentDetails?: boolean | null | undefined;
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
  startDate?: number | null | undefined;
  endDate?: number | null | undefined;
  limit?: number | null | undefined;
  offset?: number | null | undefined;
  timezone?: string | null | undefined;
  windowDays?: number | null | undefined;
  include?: string | null | undefined;
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
  first: number;
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | FieldFilterInput | null | undefined;
}>;


export type ExceptionShipmentsQuery = { shipments: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'ShipmentFieldsFragment': ShipmentFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'ShipmentPageInfoFieldsFragment': ShipmentPageInfoFieldsFragment } } } };

export type MapShipmentsQueryVariables = Exact<{
  first: number;
  after?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | FieldFilterInput | null | undefined;
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
  shipmentId?: string | number | null | undefined;
  types?: Array<ShipmentEventType> | ShipmentEventType | null | undefined;
  limit?: number | null | undefined;
  before?: number | null | undefined;
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


export type TransferShipmentToBillingMutation = { transferShipmentToBilling: { id: string, organizationId: string, businessUnitId: string, shipmentId: string, assignedBillerId: string | null, number: string, status: BillingQueueStatus, billType: BillType, exceptionReasonCode: BillingQueueExceptionReasonCode | null, reviewNotes: string, exceptionNotes: string, reviewStartedAt: number | null, reviewCompletedAt: number | null, canceledById: string | null, canceledAt: number | null, cancelReason: string, isAdjustmentOrigin: boolean, sourceInvoiceId: string | null, sourceInvoiceAdjustmentId: string | null, sourceCreditMemoInvoiceId: string | null, correctionGroupId: string | null, rebillStrategy: string | null, requiresReplacementReview: boolean, rerateVariancePercent: string, adjustmentContext: unknown, version: number, createdAt: number, updatedAt: number } };

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

export type WorkerFleetCodeFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'WorkerFleetCodeFieldsFragment' };

export type WorkerUsStateFieldsFragment = { id: string, name: string, abbreviation: string } & { ' $fragmentName'?: 'WorkerUsStateFieldsFragment' };

export type WorkerProfileTableFieldsFragment = { id: string, workerId: string, businessUnitId: string, organizationId: string, licenseStateId: string | null, dob: number, licenseNumber: string, cdlClass: CdlClass, cdlRestrictions: string, endorsement: EndorsementType, hazmatExpiry: number | null, licenseExpiry: number, medicalCardExpiry: number | null, medicalExaminerName: string, medicalExaminerNpi: string, twicCardNumber: string, twicExpiry: number | null, hireDate: number, terminationDate: number | null, physicalDueDate: number | null, mvrDueDate: number | null, complianceStatus: ComplianceStatus, isQualified: boolean, disqualificationReason: string, lastComplianceCheck: number, lastMvrCheck: number, lastDrugTest: number, eldExempt: boolean, shortHaulExempt: boolean, version: number, createdAt: number, updatedAt: number, licenseState: { ' $fragmentRefs'?: { 'WorkerUsStateFieldsFragment': WorkerUsStateFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerProfileTableFieldsFragment' };

export type WorkerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, stateId: string, fleetCodeId: string | null, managerId: string | null, status: EntityStatus, type: WorkerType, driverType: DriverType, profilePicUrl: string, firstName: string, lastName: string, wholeName: string, addressLine1: string, addressLine2: string, city: string, postalCode: string, email: string, phoneNumber: string, emergencyContactName: string, emergencyContactPhone: string, externalId: string, assignmentBlocked: string, gender: WorkerGender, canBeAssigned: boolean, availableForDispatch: boolean, version: number, createdAt: number, updatedAt: number, customFields: unknown, fleetCode: { ' $fragmentRefs'?: { 'WorkerFleetCodeFieldsFragment': WorkerFleetCodeFieldsFragment } } | null, state: { ' $fragmentRefs'?: { 'WorkerUsStateFieldsFragment': WorkerUsStateFieldsFragment } } | null, profile: { ' $fragmentRefs'?: { 'WorkerProfileTableFieldsFragment': WorkerProfileTableFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerTableRowFieldsFragment' };

export type WorkerPtoWorkerFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string, profilePicUrl: string } & { ' $fragmentName'?: 'WorkerPtoWorkerFieldsFragment' };

export type WorkerPtoRowFieldsFragment = { id: string, workerId: string, organizationId: string, businessUnitId: string, approverId: string | null, rejectorId: string | null, status: PtoStatus, type: PtoType, startDate: number, endDate: number, reason: string, version: number, createdAt: number, updatedAt: number, worker: { ' $fragmentRefs'?: { 'WorkerPtoWorkerFieldsFragment': WorkerPtoWorkerFieldsFragment } } | null } & { ' $fragmentName'?: 'WorkerPtoRowFieldsFragment' };

export type WorkerDataTablePageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'WorkerDataTablePageInfoFieldsFragment' };

export type WorkerTableQueryVariables = Exact<{
  first: number;
  after?: string | null | undefined;
  query?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | FieldFilterInput | null | undefined;
  filterGroups?: Array<FilterGroupInput> | FilterGroupInput | null | undefined;
  sort?: Array<SortFieldInput> | SortFieldInput | null | undefined;
}>;


export type WorkerTableQuery = { workers: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerTableRowFieldsFragment': WorkerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'WorkerDataTablePageInfoFieldsFragment': WorkerDataTablePageInfoFieldsFragment } } } };

export type WorkerPtoTableQueryVariables = Exact<{
  first: number;
  after?: string | null | undefined;
  query?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | FieldFilterInput | null | undefined;
  filterGroups?: Array<FilterGroupInput> | FilterGroupInput | null | undefined;
  sort?: Array<SortFieldInput> | SortFieldInput | null | undefined;
  includeWorker?: boolean | null | undefined;
}>;


export type WorkerPtoTableQuery = { workerPTOEntries: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'WorkerDataTablePageInfoFieldsFragment': WorkerDataTablePageInfoFieldsFragment } } } };

export type UpcomingWorkerPtoQueryVariables = Exact<{
  first: number;
  after?: string | null | undefined;
  status?: PtoStatus | null | undefined;
  type?: PtoType | null | undefined;
  startDate?: number | null | undefined;
  endDate?: number | null | undefined;
  workerId?: string | number | null | undefined;
  fleetCodeId?: string | number | null | undefined;
  timezone?: string | null | undefined;
}>;


export type UpcomingWorkerPtoQuery = { upcomingWorkerPTO: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'WorkerPtoRowFieldsFragment': WorkerPtoRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'WorkerDataTablePageInfoFieldsFragment': WorkerDataTablePageInfoFieldsFragment } } } };

export type WorkerPtoChartDataQueryVariables = Exact<{
  startDateFrom: number;
  startDateTo: number;
  type?: PtoType | null | undefined;
  workerId?: string | number | null | undefined;
  timezone?: string | null | undefined;
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
  emailAddress
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
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
  emailAddress
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
  emailAddress
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
export const TractorTableDocument = new TypedDocumentString(`
    query TractorTable($first: Int!, $after: String, $query: String, $fieldFilters: [FieldFilterInput!], $filterGroups: [FilterGroupInput!], $sort: [SortFieldInput!], $includeEquipmentDetails: Boolean = true, $includeFleetDetails: Boolean = true, $includeWorkerDetails: Boolean = true) {
  tractors(
    first: $first
    after: $after
    query: $query
    fieldFilters: $fieldFilters
    filterGroups: $filterGroups
    sort: $sort
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
}`, {"hash":"sha256:48e33e2d022f9b3fd926131b7eb932b0fb4d388627326e375d96d23a10648baf"}) as unknown as TypedDocumentString<TractorTableQuery, TractorTableQueryVariables>;
export const TrailerTableDocument = new TypedDocumentString(`
    query TrailerTable($first: Int!, $after: String, $query: String, $fieldFilters: [FieldFilterInput!], $filterGroups: [FilterGroupInput!], $sort: [SortFieldInput!], $includeEquipmentDetails: Boolean = true, $includeFleetDetails: Boolean = true) {
  trailers(
    first: $first
    after: $after
    query: $query
    fieldFilters: $fieldFilters
    filterGroups: $filterGroups
    sort: $sort
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
}`, {"hash":"sha256:9df2ee9fc4bfd5411a8dca95d3f610a7040e7de8656657a6ccd049afa65b4238"}) as unknown as TypedDocumentString<TrailerTableQuery, TrailerTableQueryVariables>;
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
export const ShipmentCommandCenterTableDocument = new TypedDocumentString(`
    query ShipmentCommandCenterTable($first: Int!, $after: String, $query: String, $fieldFilters: [FieldFilterInput!], $filterGroups: [FilterGroupInput!], $sort: [SortFieldInput!], $expandShipmentDetails: Boolean = true) {
  shipments(
    first: $first
    after: $after
    query: $query
    fieldFilters: $fieldFilters
    filterGroups: $filterGroups
    sort: $sort
    expandShipmentDetails: $expandShipmentDetails
  ) {
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
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:d0265141369328ad1c053b96d81f39e5ebc60acf60b7c5a768f7370378d3daa1"}) as unknown as TypedDocumentString<ShipmentCommandCenterTableQuery, ShipmentCommandCenterTableQueryVariables>;
export const ShipmentDetailDocument = new TypedDocumentString(`
    query ShipmentDetail($id: ID!, $expandShipmentDetails: Boolean = true) {
  shipment(id: $id, expandShipmentDetails: $expandShipmentDetails) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:967bfafc6c98ab8ce91b7a396c99b153495d95aae31655e8e495fbfcafeccb2b"}) as unknown as TypedDocumentString<ShipmentDetailQuery, ShipmentDetailQueryVariables>;
export const ShipmentSavedViewCountsDocument = new TypedDocumentString(`
    query ShipmentSavedViewCounts($timezone: String!) {
  shipmentAnalytics(include: "savedViewCounts", timezone: $timezone) {
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
    `, {"hash":"sha256:cc24216c8e71cfff81c03c0d8f5dee2b235d2ea7aa1af26d45bb932fc80211bb"}) as unknown as TypedDocumentString<ShipmentSavedViewCountsQuery, ShipmentSavedViewCountsQueryVariables>;
export const ShipmentPageAnalyticsDocument = new TypedDocumentString(`
    query ShipmentPageAnalytics($startDate: Int, $endDate: Int, $limit: Int, $offset: Int, $timezone: String, $windowDays: Int, $include: String) {
  shipmentAnalytics(
    startDate: $startDate
    endDate: $endDate
    limit: $limit
    offset: $offset
    timezone: $timezone
    windowDays: $windowDays
    include: $include
  ) {
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
    `, {"hash":"sha256:9fff5e0a3c9a2cddee9b3ac55cbd809c2be6b60e466b78b7adced0ea3a54e438"}) as unknown as TypedDocumentString<ShipmentPageAnalyticsQuery, ShipmentPageAnalyticsQueryVariables>;
export const ShipmentTomorrowsPickupsDocument = new TypedDocumentString(`
    query ShipmentTomorrowsPickups($limit: Int, $offset: Int, $timezone: String) {
  shipmentAnalytics(
    include: "tomorrowsPickups"
    limit: $limit
    offset: $offset
    timezone: $timezone
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
    `, {"hash":"sha256:4fde33920fe475d227451a6b115f2ea2f290ace0cc78634485da30b88bc74a87"}) as unknown as TypedDocumentString<ShipmentTomorrowsPickupsQuery, ShipmentTomorrowsPickupsQueryVariables>;
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
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:784d5b6dedb41311934ed104212acfa1044f6df45c670ab9504e5bc2e411c868"}) as unknown as TypedDocumentString<UnassignedShipmentsQuery, UnassignedShipmentsQueryVariables>;
export const ExceptionShipmentsDocument = new TypedDocumentString(`
    query ExceptionShipments($first: Int!, $after: String, $fieldFilters: [FieldFilterInput!]) {
  shipments(
    first: $first
    after: $after
    fieldFilters: $fieldFilters
    expandShipmentDetails: true
  ) {
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
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:d32d5850adf3e1f22111924782501028c87dad764459d7b2730aee98ff9b6480"}) as unknown as TypedDocumentString<ExceptionShipmentsQuery, ExceptionShipmentsQueryVariables>;
export const MapShipmentsDocument = new TypedDocumentString(`
    query MapShipments($first: Int!, $after: String, $fieldFilters: [FieldFilterInput!]) {
  shipments(
    first: $first
    after: $after
    fieldFilters: $fieldFilters
    expandShipmentDetails: true
  ) {
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
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:dc641d1d07391e1a3d2f6a31f6840c4fb26bd3f2fc6e3b094a2a3f31668df78a"}) as unknown as TypedDocumentString<MapShipmentsQuery, MapShipmentsQueryVariables>;
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
  emailAddress
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
}`, {"hash":"sha256:e37d1d8ecef666ea4927be5e4f5db33df6d1bef011ce747c8ba4a3231374ad65"}) as unknown as TypedDocumentString<ShipmentCommentsQuery, ShipmentCommentsQueryVariables>;
export const ShipmentCommentCountDocument = new TypedDocumentString(`
    query ShipmentCommentCount($shipmentId: ID!) {
  shipmentCommentCount(shipmentId: $shipmentId) {
    count
  }
}
    `, {"hash":"sha256:1f62df3579f042a9c8914aa2b124bb976b08c30fdb27dc1fa25926487e7d877e"}) as unknown as TypedDocumentString<ShipmentCommentCountQuery, ShipmentCommentCountQueryVariables>;
export const ShipmentEventsDocument = new TypedDocumentString(`
    query ShipmentEvents($shipmentId: ID, $types: [ShipmentEventType!], $limit: Int, $before: Int) {
  shipmentEvents(
    shipmentId: $shipmentId
    types: $types
    limit: $limit
    before: $before
  ) {
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
}`, {"hash":"sha256:550f52772125723808100c16a4ea99a13764c84ce076c293063ea3b26a3cdce2"}) as unknown as TypedDocumentString<ShipmentEventsQuery, ShipmentEventsQueryVariables>;
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
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:763b8342509da702c233892455443986299faa3a317aaa35a72f3e027e500415"}) as unknown as TypedDocumentString<CreateShipmentMutation, CreateShipmentMutationVariables>;
export const UpdateShipmentDocument = new TypedDocumentString(`
    mutation UpdateShipment($id: ID!, $input: ShipmentInput!) {
  updateShipment(id: $id, input: $input) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:c60996aced20e0b644a9c5bd979a1fc2001129b0cd3ad020ee2b10ce9af8829d"}) as unknown as TypedDocumentString<UpdateShipmentMutation, UpdateShipmentMutationVariables>;
export const CancelShipmentDocument = new TypedDocumentString(`
    mutation CancelShipment($id: ID!, $input: ShipmentCancelInput) {
  cancelShipment(id: $id, input: $input) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:740eb9a91490a75551c0f650bf6dfe31954516e463ce936af89553dbe059c83d"}) as unknown as TypedDocumentString<CancelShipmentMutation, CancelShipmentMutationVariables>;
export const UncancelShipmentDocument = new TypedDocumentString(`
    mutation UncancelShipment($id: ID!) {
  uncancelShipment(id: $id) {
    ...ShipmentFields
  }
}
    fragment ShipmentUserFields on User {
  id
  name
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:062057ae26309d7368a77f7f18f8de9fdd81441d2a0cbdc1caa47eaf8bb31604"}) as unknown as TypedDocumentString<UncancelShipmentMutation, UncancelShipmentMutationVariables>;
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
  emailAddress
  profilePicUrl
  thumbnailUrl
}
fragment ShipmentLocationFields on Location {
  id
  name
  code
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
}`, {"hash":"sha256:f1e9c5337e92f6819e31bc44c26c1b5c56b2c84381d41bf4a95f15a2cc546b61"}) as unknown as TypedDocumentString<TransferShipmentOwnershipMutation, TransferShipmentOwnershipMutationVariables>;
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
  emailAddress
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
}`, {"hash":"sha256:7355e95d725b5b0ee440bd7be434347cba34bdb2c65215838e2488848d696b77"}) as unknown as TypedDocumentString<CreateShipmentCommentMutation, CreateShipmentCommentMutationVariables>;
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
  emailAddress
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
}`, {"hash":"sha256:b02c8ec5923e4a1c2387bfc719947c5cb6f8e37c3fc4244b7464176b591ab295"}) as unknown as TypedDocumentString<UpdateShipmentCommentMutation, UpdateShipmentCommentMutationVariables>;
export const DeleteShipmentCommentDocument = new TypedDocumentString(`
    mutation DeleteShipmentComment($shipmentId: ID!, $commentId: ID!) {
  deleteShipmentComment(shipmentId: $shipmentId, commentId: $commentId)
}
    `, {"hash":"sha256:a20dcdea6225911dd4742c1e415a5f1e2b04d0111fbaf5ecbda1e8136b3dfa14"}) as unknown as TypedDocumentString<DeleteShipmentCommentMutation, DeleteShipmentCommentMutationVariables>;
export const WorkerTableDocument = new TypedDocumentString(`
    query WorkerTable($first: Int!, $after: String, $query: String, $fieldFilters: [FieldFilterInput!], $filterGroups: [FilterGroupInput!], $sort: [SortFieldInput!]) {
  workers(
    first: $first
    after: $after
    query: $query
    fieldFilters: $fieldFilters
    filterGroups: $filterGroups
    sort: $sort
  ) {
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
}`, {"hash":"sha256:04a18f51566b8eb924e96a832da2f36b971c41f41526d79634a73582499b3430"}) as unknown as TypedDocumentString<WorkerTableQuery, WorkerTableQueryVariables>;
export const WorkerPtoTableDocument = new TypedDocumentString(`
    query WorkerPtoTable($first: Int!, $after: String, $query: String, $fieldFilters: [FieldFilterInput!], $filterGroups: [FilterGroupInput!], $sort: [SortFieldInput!], $includeWorker: Boolean = true) {
  workerPTOEntries(
    first: $first
    after: $after
    query: $query
    fieldFilters: $fieldFilters
    filterGroups: $filterGroups
    sort: $sort
    includeWorker: $includeWorker
  ) {
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
}`, {"hash":"sha256:881fc2e33707f67149bfe0e449e4f465fb7ac35e9f3d78dfbdc13da242cf8dcf"}) as unknown as TypedDocumentString<WorkerPtoTableQuery, WorkerPtoTableQueryVariables>;
export const UpcomingWorkerPtoDocument = new TypedDocumentString(`
    query UpcomingWorkerPto($first: Int!, $after: String, $status: PTOStatus, $type: PTOType, $startDate: Int, $endDate: Int, $workerId: ID, $fleetCodeId: ID, $timezone: String) {
  upcomingWorkerPTO(
    first: $first
    after: $after
    status: $status
    type: $type
    startDate: $startDate
    endDate: $endDate
    workerId: $workerId
    fleetCodeId: $fleetCodeId
    timezone: $timezone
  ) {
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
}`, {"hash":"sha256:a7e36af4743fb4d0fac8940110a84285c629f224b50673ef477f2eb6caa52ff5"}) as unknown as TypedDocumentString<UpcomingWorkerPtoQuery, UpcomingWorkerPtoQueryVariables>;
export const WorkerPtoChartDataDocument = new TypedDocumentString(`
    query WorkerPtoChartData($startDateFrom: Int!, $startDateTo: Int!, $type: PTOType, $workerId: ID, $timezone: String) {
  workerPTOChartData(
    startDateFrom: $startDateFrom
    startDateTo: $startDateTo
    type: $type
    workerId: $workerId
    timezone: $timezone
  ) {
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
    `, {"hash":"sha256:433856e1bdc1cb2e480292aca0bb42a84b8e70ca2c4faec8eded4063354f114a"}) as unknown as TypedDocumentString<WorkerPtoChartDataQuery, WorkerPtoChartDataQueryVariables>;
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