import type { WorkerRecordQuery } from "@trenova/graphql/generated/graphql";
import { WorkerRecordDocument } from "@trenova/graphql/generated/documents/worker/record";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type WorkerRecord = NonNullable<WorkerRecordQuery["worker"]>;

export async function fetchWorkerRecord(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerRecord | null> {
  const data = await requestGraphQL({
    document: WorkerRecordDocument,
    operationName: "WorkerRecord",
    variables: { id },
    signal: options?.signal,
  });
  return data.worker ?? null;
}
