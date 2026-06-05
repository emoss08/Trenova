import {
  WorkerPtoTableDocument,
  WorkerTableDocument,
  type WorkerPtoTableQueryVariables,
  type WorkerTableQueryVariables,
} from "@/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { Worker, WorkerPTO } from "@/types/worker";

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
    variables: {
      includeWorker: true,
    },
  }),
} as const;
