/* eslint-disable */
/** Internal type. DO NOT USE DIRECTLY. */
type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
/** Internal type. DO NOT USE DIRECTLY. */
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
import type { DocumentTypeDecoration } from '@graphql-typed-document-node/core';
export type EquipmentStatus =
  | 'AtMaintenance'
  | 'Available'
  | 'OutOfService'
  | 'Sold';

export type FieldFilterInput = {
  field: string;
  operator: string;
  value?: unknown;
};

export type FilterGroupInput = {
  filters: Array<FieldFilterInput>;
};

export type SortFieldInput = {
  direction: string;
  field: string;
};

export type EquipmentTypeTableFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'EquipmentTypeTableFieldsFragment' };

export type EquipmentManufacturerTableFieldsFragment = { id: string, name: string } & { ' $fragmentName'?: 'EquipmentManufacturerTableFieldsFragment' };

export type FleetCodeTableFieldsFragment = { id: string, code: string, color: string } & { ' $fragmentName'?: 'FleetCodeTableFieldsFragment' };

export type UsStateTableFieldsFragment = { id: string, name: string, abbreviation: string } & { ' $fragmentName'?: 'UsStateTableFieldsFragment' };

export type WorkerTableReferenceFieldsFragment = { id: string, firstName: string, lastName: string, wholeName: string } & { ' $fragmentName'?: 'WorkerTableReferenceFieldsFragment' };

export type DataTablePageInfoFieldsFragment = { hasNextPage: boolean, endCursor: string | null } & { ' $fragmentName'?: 'DataTablePageInfoFieldsFragment' };

export type TractorTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, primaryWorkerId: string, equipmentTypeId: string, equipmentManufacturerId: string, stateId: string | null, fleetCodeId: string | null, secondaryWorkerId: string | null, status: EquipmentStatus, code: string, model: string, make: string, year: number | null, licensePlateNumber: string, registrationNumber: string, registrationExpiry: number | null, vin: string, lastKnownLocationId: string | null, lastKnownLocationName: string, version: number, createdAt: number, updatedAt: number, customFields: unknown, equipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeTableFieldsFragment': EquipmentTypeTableFieldsFragment } } | null, equipmentManufacturer: { ' $fragmentRefs'?: { 'EquipmentManufacturerTableFieldsFragment': EquipmentManufacturerTableFieldsFragment } } | null, fleetCode: { ' $fragmentRefs'?: { 'FleetCodeTableFieldsFragment': FleetCodeTableFieldsFragment } } | null, state: { ' $fragmentRefs'?: { 'UsStateTableFieldsFragment': UsStateTableFieldsFragment } } | null, primaryWorker: { ' $fragmentRefs'?: { 'WorkerTableReferenceFieldsFragment': WorkerTableReferenceFieldsFragment } } | null, secondaryWorker: { ' $fragmentRefs'?: { 'WorkerTableReferenceFieldsFragment': WorkerTableReferenceFieldsFragment } } | null } & { ' $fragmentName'?: 'TractorTableRowFieldsFragment' };

export type TrailerTableRowFieldsFragment = { id: string, businessUnitId: string, organizationId: string, equipmentTypeId: string, equipmentManufacturerId: string, registrationStateId: string | null, fleetCodeId: string | null, status: EquipmentStatus, code: string, model: string, make: string, year: number | null, licensePlateNumber: string, vin: string, registrationNumber: string, maxLoadWeight: number | null, lastInspectionDate: number | null, registrationExpiry: number | null, lastKnownLocationId: string | null, lastKnownLocationName: string, version: number, createdAt: number, updatedAt: number, customFields: unknown, equipmentType: { ' $fragmentRefs'?: { 'EquipmentTypeTableFieldsFragment': EquipmentTypeTableFieldsFragment } } | null, equipmentManufacturer: { ' $fragmentRefs'?: { 'EquipmentManufacturerTableFieldsFragment': EquipmentManufacturerTableFieldsFragment } } | null, fleetCode: { ' $fragmentRefs'?: { 'FleetCodeTableFieldsFragment': FleetCodeTableFieldsFragment } } | null, registrationState: { ' $fragmentRefs'?: { 'UsStateTableFieldsFragment': UsStateTableFieldsFragment } } | null } & { ' $fragmentName'?: 'TrailerTableRowFieldsFragment' };

export type TractorTableQueryVariables = Exact<{
  first: number;
  offset?: number | null | undefined;
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
  offset?: number | null | undefined;
  after?: string | null | undefined;
  query?: string | null | undefined;
  fieldFilters?: Array<FieldFilterInput> | FieldFilterInput | null | undefined;
  filterGroups?: Array<FilterGroupInput> | FilterGroupInput | null | undefined;
  sort?: Array<SortFieldInput> | SortFieldInput | null | undefined;
  includeEquipmentDetails?: boolean | null | undefined;
  includeFleetDetails?: boolean | null | undefined;
}>;


export type TrailerTableQuery = { trailers: { totalCount: number | null, edges: Array<{ node: { ' $fragmentRefs'?: { 'TrailerTableRowFieldsFragment': TrailerTableRowFieldsFragment } } }>, pageInfo: { ' $fragmentRefs'?: { 'DataTablePageInfoFieldsFragment': DataTablePageInfoFieldsFragment } } } };

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
export const TractorTableDocument = new TypedDocumentString(`
    query TractorTable($first: Int!, $offset: Int, $after: String, $query: String, $fieldFilters: [FieldFilterInput!], $filterGroups: [FilterGroupInput!], $sort: [SortFieldInput!], $includeEquipmentDetails: Boolean = true, $includeFleetDetails: Boolean = true, $includeWorkerDetails: Boolean = true) {
  tractors(
    first: $first
    offset: $offset
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
}`, {"hash":"sha256:5fac6658d775d4262c0850bdb2938194466f9631cb91203f6f60f73c7e3a3376"}) as unknown as TypedDocumentString<TractorTableQuery, TractorTableQueryVariables>;
export const TrailerTableDocument = new TypedDocumentString(`
    query TrailerTable($first: Int!, $offset: Int, $after: String, $query: String, $fieldFilters: [FieldFilterInput!], $filterGroups: [FilterGroupInput!], $sort: [SortFieldInput!], $includeEquipmentDetails: Boolean = true, $includeFleetDetails: Boolean = true) {
  trailers(
    first: $first
    offset: $offset
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
}`, {"hash":"sha256:7d14e89dd8650263df64b20cc350016579d0015061ecd9c1517bcf7089569334"}) as unknown as TypedDocumentString<TrailerTableQuery, TrailerTableQueryVariables>;