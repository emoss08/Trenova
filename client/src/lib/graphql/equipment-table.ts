import {
  DATA_TABLE_CONNECTION_ARGUMENTS,
  DATA_TABLE_CONNECTION_VARIABLES,
  DATA_TABLE_PAGE_INFO_FRAGMENT,
  defineDataTableGraphQLConfig,
} from "@/lib/graphql/data-table";
import type { Tractor } from "@/types/tractor";
import type { Trailer } from "@/types/trailer";

const EQUIPMENT_TYPE_TABLE_FRAGMENT = `
  fragment EquipmentTypeTableFields on EquipmentType {
    id
    code
    color
  }
`;

const EQUIPMENT_MANUFACTURER_TABLE_FRAGMENT = `
  fragment EquipmentManufacturerTableFields on EquipmentManufacturer {
    id
    name
  }
`;

const FLEET_CODE_TABLE_FRAGMENT = `
  fragment FleetCodeTableFields on FleetCode {
    id
    code
    color
  }
`;

const US_STATE_TABLE_FRAGMENT = `
  fragment UsStateTableFields on UsState {
    id
    name
    abbreviation
  }
`;

const WORKER_TABLE_REFERENCE_FRAGMENT = `
  fragment WorkerTableReferenceFields on Worker {
    id
    firstName
    lastName
    wholeName
  }
`;

const TRACTOR_TABLE_ROW_FRAGMENT = `
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
`;

const TRAILER_TABLE_ROW_FRAGMENT = `
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
`;

export const TRACTOR_TABLE_GRAPHQL_DOCUMENT = `
  query TractorTable(
    ${DATA_TABLE_CONNECTION_VARIABLES}
    $includeEquipmentDetails: Boolean = true
    $includeFleetDetails: Boolean = true
    $includeWorkerDetails: Boolean = true
  ) {
    tractors(
      ${DATA_TABLE_CONNECTION_ARGUMENTS}
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

  ${TRACTOR_TABLE_ROW_FRAGMENT}
  ${EQUIPMENT_TYPE_TABLE_FRAGMENT}
  ${EQUIPMENT_MANUFACTURER_TABLE_FRAGMENT}
  ${FLEET_CODE_TABLE_FRAGMENT}
  ${US_STATE_TABLE_FRAGMENT}
  ${WORKER_TABLE_REFERENCE_FRAGMENT}
  ${DATA_TABLE_PAGE_INFO_FRAGMENT}
`;

export const TRAILER_TABLE_GRAPHQL_DOCUMENT = `
  query TrailerTable(
    ${DATA_TABLE_CONNECTION_VARIABLES}
    $includeEquipmentDetails: Boolean = true
    $includeFleetDetails: Boolean = true
  ) {
    trailers(
      ${DATA_TABLE_CONNECTION_ARGUMENTS}
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

  ${TRAILER_TABLE_ROW_FRAGMENT}
  ${EQUIPMENT_TYPE_TABLE_FRAGMENT}
  ${EQUIPMENT_MANUFACTURER_TABLE_FRAGMENT}
  ${FLEET_CODE_TABLE_FRAGMENT}
  ${US_STATE_TABLE_FRAGMENT}
  ${DATA_TABLE_PAGE_INFO_FRAGMENT}
`;

export const equipmentTableGraphQLConfigs = {
  tractor: defineDataTableGraphQLConfig<Tractor>({
    document: TRACTOR_TABLE_GRAPHQL_DOCUMENT,
    operationName: "TractorTable",
    connectionKey: "tractors",
    variables: {
      includeEquipmentDetails: true,
      includeFleetDetails: true,
      includeWorkerDetails: true,
    },
  }),
  trailer: defineDataTableGraphQLConfig<Trailer>({
    document: TRAILER_TABLE_GRAPHQL_DOCUMENT,
    operationName: "TrailerTable",
    connectionKey: "trailers",
    variables: {
      includeEquipmentDetails: true,
      includeFleetDetails: true,
    },
  }),
} as const;
