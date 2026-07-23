import {
  WorkerPtoTableDocument,
  WorkerTableDocument,
  type WorkerPtoTableQueryVariables,
  type WorkerTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { Worker, WorkerPTO } from "@trenova/shared/types/worker";

export const workerTableGraphQLConfigs = {
  worker: defineDataTableGraphQLConfig<Worker, WorkerTableQueryVariables>({
    document: WorkerTableDocument,
    operationName: "WorkerTable",
    connectionKey: "workers",
  }),
  pto: defineDataTableGraphQLConfig<WorkerPTO, WorkerPtoTableQueryVariables>({
    document: WorkerPtoTableDocument,
    operationName: "WorkerPtoTable",
    connectionKey: "workerPTOEntries",
    inputExtraVariables: {
      includeWorker: true,
    },
  }),
} as const;
