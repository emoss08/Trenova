import {
  EquipmentTypeTableDocument,
  TractorTableDocument,
  TrailerTableDocument,
  type EquipmentTypeTableQueryVariables,
  type TractorTableQueryVariables,
  type TrailerTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { EquipmentType } from "@/types/equipment-type";
import type { Tractor } from "@/types/tractor";
import type { Trailer } from "@/types/trailer";

export const equipmentTableGraphQLConfigs = {
  equipmentType: defineDataTableGraphQLConfig<EquipmentType, EquipmentTypeTableQueryVariables>({
    document: EquipmentTypeTableDocument,
    operationName: "EquipmentTypeTable",
    connectionKey: "equipmentTypes",
    buildVariables: ({ pageSize, options }) => ({
      input: {
        first: pageSize,
        after: options?.cursor || undefined,
        query: options?.query || undefined,
        fieldFilters: options?.fieldFilters ?? [],
        filterGroups: options?.filterGroups ?? [],
        sort: options?.sort ?? [],
      },
    }),
  }),
  tractor: defineDataTableGraphQLConfig<Tractor, TractorTableQueryVariables>({
    document: TractorTableDocument,
    operationName: "TractorTable",
    connectionKey: "tractors",
    variables: {
      includeEquipmentDetails: true,
      includeFleetDetails: true,
      includeWorkerDetails: true,
    },
  }),
  trailer: defineDataTableGraphQLConfig<Trailer, TrailerTableQueryVariables>({
    document: TrailerTableDocument,
    operationName: "TrailerTable",
    connectionKey: "trailers",
    variables: {
      includeEquipmentDetails: true,
      includeFleetDetails: true,
    },
  }),
} as const;
