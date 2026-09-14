import { describe, expect, it, vi } from "vitest";
import type {
  BulkBillingTransferResponse,
  BulkBillingTransferResult,
} from "@/lib/graphql/billing-transfer";
import { runBulkBillingTransfer } from "../run-bulk-billing-transfer";

function transferred(shipmentId: string): BulkBillingTransferResult {
  return {
    shipmentId,
    proNumber: `PRO-${shipmentId}`,
    success: true,
    markedReadyToInvoice: false,
    failureCode: null,
    error: null,
    billingQueueItem: {
      id: `bqi-${shipmentId}`,
      number: `INV-${shipmentId}`,
      status: "ReadyForReview",
    },
    missingRequirements: [],
    validationFailures: [],
  };
}

function blocked(shipmentId: string): BulkBillingTransferResult {
  return {
    shipmentId,
    proNumber: `PRO-${shipmentId}`,
    success: false,
    markedReadyToInvoice: false,
    failureCode: "RequirementsUnmet",
    error: "Shipment billing requirements must be resolved before transfer to billing",
    billingQueueItem: null,
    missingRequirements: [
      { documentTypeId: "dt_pod", documentTypeCode: "POD", documentTypeName: "Proof of Delivery" },
    ],
    validationFailures: [],
  };
}

function respond(results: BulkBillingTransferResult[]): BulkBillingTransferResponse {
  const successCount = results.filter((r) => r.success).length;
  return {
    results,
    totalCount: results.length,
    successCount,
    errorCount: results.length - successCount,
  };
}

const ids = (count: number) => Array.from({ length: count }, (_, i) => `shp_${i + 1}`);

describe("runBulkBillingTransfer", () => {
  it("sends the shipments in sequential batches and keeps every outcome in order", async () => {
    const calls: string[][] = [];
    const transfer = vi.fn(async (batch: string[]) => {
      calls.push(batch);
      return respond(batch.map((id) => (id === "shp_4" ? blocked(id) : transferred(id))));
    });
    const onProgress = vi.fn();

    const run = await runBulkBillingTransfer({
      shipmentIds: ids(5),
      chunkSize: 2,
      transfer,
      shouldStop: () => false,
      onProgress,
    });

    expect(calls).toEqual([["shp_1", "shp_2"], ["shp_3", "shp_4"], ["shp_5"]]);
    expect(run.results.map((r) => r.shipmentId)).toEqual(ids(5));
    expect(run.results.find((r) => r.shipmentId === "shp_4")?.failureCode).toBe(
      "RequirementsUnmet",
    );
    expect(run.notProcessedIds).toEqual([]);
    expect(run.stopped).toBe(false);
    expect(run.error).toBeNull();
    expect(onProgress).toHaveBeenCalledTimes(3);
    expect(onProgress).toHaveBeenLastCalledWith({ processedCount: 5, results: run.results });
  });

  it("never sends more than the server accepts in one request", async () => {
    const batchSizes: number[] = [];

    await runBulkBillingTransfer({
      shipmentIds: ids(230),
      transfer: async (batch) => {
        batchSizes.push(batch.length);
        return respond(batch.map(transferred));
      },
      shouldStop: () => false,
      onProgress: () => undefined,
    });

    expect(Math.max(...batchSizes)).toBeLessThanOrEqual(100);
    expect(batchSizes.reduce((sum, size) => sum + size, 0)).toBe(230);
  });

  it("sends a shipment selected twice only once", async () => {
    const transfer = vi.fn(async (batch: string[]) => respond(batch.map(transferred)));

    const run = await runBulkBillingTransfer({
      shipmentIds: ["shp_1", "shp_2", "shp_1"],
      chunkSize: 10,
      transfer,
      shouldStop: () => false,
      onProgress: () => undefined,
    });

    expect(transfer).toHaveBeenCalledWith(["shp_1", "shp_2"]);
    expect(run.results).toHaveLength(2);
  });

  it("finishes the batch in flight when stopped and leaves the rest unprocessed", async () => {
    let stopRequested = false;
    const transfer = vi.fn(async (batch: string[]) => {
      stopRequested = true;
      return respond(batch.map(transferred));
    });

    const run = await runBulkBillingTransfer({
      shipmentIds: ids(5),
      chunkSize: 2,
      transfer,
      shouldStop: () => stopRequested,
      onProgress: () => undefined,
    });

    expect(transfer).toHaveBeenCalledTimes(1);
    expect(run.results.map((r) => r.shipmentId)).toEqual(["shp_1", "shp_2"]);
    expect(run.notProcessedIds).toEqual(["shp_3", "shp_4", "shp_5"]);
    expect(run.stopped).toBe(true);
    expect(run.error).toBeNull();
  });

  it("does not send anything when stopped before the first batch", async () => {
    const transfer = vi.fn();

    const run = await runBulkBillingTransfer({
      shipmentIds: ids(3),
      transfer,
      shouldStop: () => true,
      onProgress: () => undefined,
    });

    expect(transfer).not.toHaveBeenCalled();
    expect(run.notProcessedIds).toEqual(ids(3));
    expect(run.stopped).toBe(true);
  });

  it("ends the run on a failed request, keeping what already transferred", async () => {
    const failure = new Error("Network request failed");
    const transfer = vi
      .fn<(batch: string[]) => Promise<BulkBillingTransferResponse>>()
      .mockImplementationOnce(async (batch) => respond(batch.map(transferred)))
      .mockRejectedValueOnce(failure);

    const run = await runBulkBillingTransfer({
      shipmentIds: ids(5),
      chunkSize: 2,
      transfer,
      shouldStop: () => false,
      onProgress: () => undefined,
    });

    expect(transfer).toHaveBeenCalledTimes(2);
    expect(run.results.map((r) => r.shipmentId)).toEqual(["shp_1", "shp_2"]);
    expect(run.notProcessedIds).toEqual(["shp_3", "shp_4", "shp_5"]);
    expect(run.error).toBe(failure);
    expect(run.stopped).toBe(false);
  });

  it("treats a shipment the server did not report on as unprocessed", async () => {
    const run = await runBulkBillingTransfer({
      shipmentIds: ids(3),
      chunkSize: 3,
      transfer: async () =>
        respond([transferred("shp_1"), transferred("shp_99"), blocked("shp_3")]),
      shouldStop: () => false,
      onProgress: () => undefined,
    });

    expect(run.results.map((r) => r.shipmentId)).toEqual(["shp_1", "shp_3"]);
    expect(run.notProcessedIds).toEqual(["shp_2"]);
  });

  it("returns an empty run for no shipments", async () => {
    const transfer = vi.fn();

    const run = await runBulkBillingTransfer({
      shipmentIds: [],
      transfer,
      shouldStop: () => false,
      onProgress: () => undefined,
    });

    expect(transfer).not.toHaveBeenCalled();
    expect(run).toEqual({ results: [], notProcessedIds: [], stopped: false, error: null });
  });
});
