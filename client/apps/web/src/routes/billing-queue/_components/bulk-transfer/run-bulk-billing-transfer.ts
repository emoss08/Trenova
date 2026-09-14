import type {
  BulkBillingTransferResponse,
  BulkBillingTransferResult,
} from "@/lib/graphql/billing-transfer";
import { chunk } from "@trenova/shared/lib/utils";

/**
 * Shipments per request. The server accepts at most 100, and each shipment runs
 * its own readiness check, so a batch of 25 finishes well inside the request
 * timeout while still reporting progress often enough to watch.
 */
export const BULK_BILLING_TRANSFER_BATCH_SIZE = 25;

/** The most shipments the server resolves for a single "transfer all" run. */
export const MAX_BILLING_TRANSFER_CANDIDATE_IDS = 5000;

export type BulkBillingTransferRun = {
  results: BulkBillingTransferResult[];
  notProcessedIds: string[];
  stopped: boolean;
  error: unknown;
};

export type BulkBillingTransferProgress = {
  processedCount: number;
  results: BulkBillingTransferResult[];
};

type RunBulkBillingTransferParams = {
  shipmentIds: readonly string[];
  chunkSize?: number;
  transfer: (shipmentIds: string[]) => Promise<BulkBillingTransferResponse>;
  shouldStop: () => boolean;
  onProgress: (progress: BulkBillingTransferProgress) => void;
};

/**
 * Transfers shipments one batch at a time.
 *
 * A stop takes effect between batches, never inside one: a batch the server has
 * started is already moving shipments into the queue, so abandoning its response
 * would only hide outcomes that happened anyway. A failed request ends the run
 * for the same reason — whatever caused it will most likely fail the next batch
 * too, and the shipments it carried are reported as unprocessed rather than
 * guessed at.
 */
export async function runBulkBillingTransfer({
  shipmentIds,
  chunkSize = BULK_BILLING_TRANSFER_BATCH_SIZE,
  transfer,
  shouldStop,
  onProgress,
}: RunBulkBillingTransferParams): Promise<BulkBillingTransferRun> {
  const batches = chunk([...new Set(shipmentIds)], chunkSize);
  const results: BulkBillingTransferResult[] = [];
  const notProcessedIds: string[] = [];
  let processedCount = 0;

  for (let index = 0; index < batches.length; index++) {
    const batch = batches[index];

    if (shouldStop()) {
      notProcessedIds.push(...batches.slice(index).flat());
      return { results, notProcessedIds, stopped: true, error: null };
    }

    let response: BulkBillingTransferResponse;
    try {
      response = await transfer(batch);
    } catch (error) {
      notProcessedIds.push(...batches.slice(index).flat());
      return { results, notProcessedIds, stopped: false, error };
    }

    const reported = new Map(response.results.map((result) => [result.shipmentId, result]));
    for (const shipmentId of batch) {
      const result = reported.get(shipmentId);
      if (result) {
        results.push(result);
      } else {
        notProcessedIds.push(shipmentId);
      }
    }

    processedCount += batch.length;
    onProgress({ processedCount, results: [...results] });
  }

  return { results, notProcessedIds, stopped: false, error: null };
}
