import {
  EquipmentTypeTableDocument,
  TractorTableDocument,
  TrailerTableDocument,
  type EquipmentTypeTableQueryVariables,
  type TractorTableQueryVariables,
  type TrailerTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { EquipmentType } from "@/types/equipment-type";
import type { Tractor } from "@/types/tractor";
import type { Trailer } from "@/types/trailer";

export const equipmentTableGraphQLConfigs = {
  equipmentType: defineDataTableGraphQLConfig<EquipmentType, EquipmentTypeTableQueryVariables>({
    document: EquipmentTypeTableDocument,
    operationName: "EquipmentTypeTable",
    connectionKey: "equipmentTypes",
  }),
  tractor: defineDataTableGraphQLConfig<Tractor, TractorTableQueryVariables>({
    document: TractorTableDocument,
    operationName: "TractorTable",
    connectionKey: "tractors",
    extraVariables: {
      includeEquipmentDetails: true,
      includeFleetDetails: true,
      includeWorkerDetails: true,
    },
  }),
  trailer: defineDataTableGraphQLConfig<Trailer, TrailerTableQueryVariables>({
    document: TrailerTableDocument,
    operationName: "TrailerTable",
    connectionKey: "trailers",
    extraVariables: {
      includeEquipmentDetails: true,
      includeFleetDetails: true,
    },
  }),
} as const;
