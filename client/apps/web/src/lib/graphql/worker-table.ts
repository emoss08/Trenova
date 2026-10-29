import { WorkerPtoTableDocument, WorkerTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

/**
 * The roster's read, narrowed to the workers with no valid credential of the
 * type a code names (none on file, archived or expired) when one is chosen.
 */
export function workerTableGraphQLConfig(withoutCredential: string | null) {
  return defineDataTableGraphQLConfig({
    document: WorkerTableDocument,
    operationName: "WorkerTable",
    connectionKey: "workers",
    extraVariables: withoutCredential ? { withoutCredential } : undefined,
  });
}

export const workerTableGraphQLConfigs = {
  worker: workerTableGraphQLConfig(null),
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
