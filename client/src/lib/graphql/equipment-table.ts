import {
  TractorTableDocument,
  TrailerTableDocument,
  type TractorTableQueryVariables,
  type TrailerTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { Tractor } from "@/types/tractor";
import type { Trailer } from "@/types/trailer";

export const equipmentTableGraphQLConfigs = {
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
