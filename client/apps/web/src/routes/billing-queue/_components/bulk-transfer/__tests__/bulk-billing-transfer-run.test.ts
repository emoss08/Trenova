import { describe, expect, it } from "vitest";
import type { BillingTransferRun } from "@/lib/graphql/billing-transfer";
import {
  canRetryRun,
  isRunStopping,
  isRunTerminal,
  runProgress,
  shouldPollRun,
  summarizeRun,
} from "../bulk-billing-transfer-run";

function run(overrides: Partial<BillingTransferRun> = {}): BillingTransferRun {
  return {
    id: "btr_1",
    status: "Running",
    scope: "Selected",
    billType: "Invoice",
    searchQuery: null,
    shipmentStatus: null,
    sourceRunId: null,
    totalCount: 10,
    processedCount: 0,
    transferredCount: 0,
    notTransferredCount: 0,
    skippedCount: 0,
    markedReadyToInvoiceCount: 0,
    retryableCount: 0,
    unmatchedCount: 0,
    failureMessage: null,
    cancelRequestedAt: null,
    queuedAt: 1_788_000_000,
    startedAt: null,
    completedAt: null,
    ...overrides,
  };
}

describe("runProgress", () => {
  // A run that has not been told how much there is to do reports nothing. An
  // empty bar says "no progress", which is the opposite of "still counting".
  it("answers nothing while the total is unknown", () => {
    expect(runProgress(run({ totalCount: 0, processedCount: 0 }))).toBeNull();
    expect(runProgress(undefined)).toBeNull();
    expect(runProgress(null)).toBeNull();
  });

  it("distinguishes an unknown total from a genuine zero", () => {
    expect(runProgress(run({ totalCount: 10, processedCount: 0 }))).toBe(0);
  });

  it("reports the fraction processed", () => {
    expect(runProgress(run({ totalCount: 8, processedCount: 2 }))).toBe(0.25);
  });

  // Counters are recomputed from rows, so a retried batch cannot push them past
  // the total — but a bar that renders over 100% would be worse than clamping.
  it("never exceeds one", () => {
    expect(runProgress(run({ totalCount: 4, processedCount: 9 }))).toBe(1);
  });
});

describe("shouldPollRun", () => {
  it("keeps asking while the run is working", () => {
    expect(shouldPollRun(run({ status: "Queued" }))).toBe(true);
    expect(shouldPollRun(run({ status: "Running" }))).toBe(true);
  });

  // Polling a finished run forever is how a background job quietly becomes a
  // load on the database.
  it.each(["Completed", "Canceled", "Failed"] as const)("stops once %s", (status) => {
    expect(shouldPollRun(run({ status }))).toBe(false);
    expect(isRunTerminal(run({ status }))).toBe(true);
  });

  it("does not poll when there is no run", () => {
    expect(shouldPollRun(null)).toBe(false);
    expect(isRunTerminal(null)).toBe(false);
  });
});

describe("isRunStopping", () => {
  // The batch in flight always finishes, so there is a stretch where a stopped
  // run is still moving shipments. Saying so beats a button that looks broken.
  it("is true after a stop is asked for and before the run ends", () => {
    expect(isRunStopping(run({ cancelRequestedAt: 1_788_000_020 }))).toBe(true);
  });

  it("is false once the run has actually stopped", () => {
    expect(isRunStopping(run({ status: "Canceled", cancelRequestedAt: 1_788_000_020 }))).toBe(
      false,
    );
  });

  it("is false when nobody asked", () => {
    expect(isRunStopping(run())).toBe(false);
  });
});

describe("canRetryRun", () => {
  it("offers a retry for failures a second attempt could change", () => {
    expect(canRetryRun(run({ status: "Completed", retryableCount: 3 }))).toBe(true);
  });

  // A run that was stopped left shipments untouched; those are always worth
  // another attempt even though none of them failed.
  it("offers a retry for shipments the run never reached", () => {
    expect(canRetryRun(run({ status: "Canceled", retryableCount: 0, skippedCount: 4 }))).toBe(true);
  });

  it("offers nothing when every failure is final", () => {
    expect(canRetryRun(run({ status: "Completed", notTransferredCount: 2 }))).toBe(false);
  });

  it("offers nothing while the run is still going", () => {
    expect(canRetryRun(run({ status: "Running", retryableCount: 3 }))).toBe(false);
  });
});

describe("summarizeRun", () => {
  it("reads the counters the server settled on rather than recomputing them", () => {
    expect(
      summarizeRun(
        run({
          totalCount: 10,
          transferredCount: 6,
          notTransferredCount: 3,
          skippedCount: 1,
          markedReadyToInvoiceCount: 2,
          retryableCount: 3,
          unmatchedCount: 25,
        }),
      ),
    ).toEqual({
      transferred: 6,
      notTransferred: 3,
      notProcessed: 1,
      markedReadyToInvoice: 2,
      retryable: 3,
      unmatched: 25,
      total: 10,
    });
  });
});
