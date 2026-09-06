import { WorkerPtoTableDocument, WorkerTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const workerTableGraphQLConfigs = {
  worker: defineDataTableGraphQLConfig({
    document: WorkerTableDocument,
    operationName: "WorkerTable",
    connectionKey: "workers",
  }),
  pto: defineDataTableGraphQLConfig({
    document: WorkerPtoTableDocument,
    operationName: "WorkerPtoTable",
    connectionKey: "workerPTOEntries",
    inputExtraVariables: {
      includeWorker: true,
    },
  }),
} as const;

export type WorkerRow = DataTableConfigRow<typeof workerTableGraphQLConfigs.worker>;
export type WorkerPTORow = DataTableConfigRow<typeof workerTableGraphQLConfigs.pto>;
