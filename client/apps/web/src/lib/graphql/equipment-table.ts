import {
  EquipmentTypeTableDocument,
  TractorTableDocument,
  TrailerTableDocument,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const equipmentTableGraphQLConfigs = {
  equipmentType: defineDataTableGraphQLConfig({
    document: EquipmentTypeTableDocument,
    operationName: "EquipmentTypeTable",
    connectionKey: "equipmentTypes",
  }),
  tractor: defineDataTableGraphQLConfig({
    document: TractorTableDocument,
    operationName: "TractorTable",
    connectionKey: "tractors",
    extraVariables: {
      includeEquipmentDetails: true,
      includeFleetDetails: true,
      includeWorkerDetails: true,
    },
  }),
  trailer: defineDataTableGraphQLConfig({
    document: TrailerTableDocument,
    operationName: "TrailerTable",
    connectionKey: "trailers",
    extraVariables: {
      includeEquipmentDetails: true,
      includeFleetDetails: true,
    },
  }),
} as const;

export type TractorRow = DataTableConfigRow<typeof equipmentTableGraphQLConfigs.tractor>;
export type TrailerRow = DataTableConfigRow<typeof equipmentTableGraphQLConfigs.trailer>;
