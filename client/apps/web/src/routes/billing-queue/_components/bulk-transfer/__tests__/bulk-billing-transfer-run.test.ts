import { describe, expect, it } from "vitest";
import type { BillingTransferRun } from "@/lib/graphql/billing-transfer";
import {
  canRetryRun,
  isRunStopping,
  isRunTerminal,
  runElapsedSeconds,
  runOutcome,
  runSecondsLeft,
  shouldPollRun,
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
    failureCounts: [],
    failureMessage: null,
    cancelRequestedAt: null,
    queuedAt: 1_788_000_000,
    startedAt: null,
    completedAt: null,
    ...overrides,
  };
}

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

describe("runOutcome", () => {
  it("names a run that moved every shipment", () => {
    expect(runOutcome(run({ status: "Completed", transferredCount: 10 }))).toBe("transferred");
  });

  it("names a run that moved some and held back the rest", () => {
    expect(
      runOutcome(run({ status: "Completed", transferredCount: 7, notTransferredCount: 3 })),
    ).toBe("partial");
  });

  it("names a run that moved none", () => {
    expect(runOutcome(run({ status: "Completed", notTransferredCount: 10 }))).toBe("blocked");
  });

  it("names a run that matched nothing", () => {
    expect(runOutcome(run({ status: "Completed", totalCount: 0 }))).toBe("empty");
  });

  // How a run ended outranks what it moved before it ended: a stopped or failed run
  // is reported as stopped even when every shipment it reached transferred.
  it("names how an interrupted run ended before what it moved", () => {
    expect(runOutcome(run({ status: "Canceled", transferredCount: 10 }))).toBe("canceled");
    expect(runOutcome(run({ status: "Failed", transferredCount: 10 }))).toBe("failed");
  });
});

describe("runElapsedSeconds", () => {
  it("counts nothing before the run starts", () => {
    expect(runElapsedSeconds(run({ startedAt: null }), 1_788_000_500)).toBeNull();
  });

  it("counts up to now while the run is going", () => {
    expect(runElapsedSeconds(run({ startedAt: 1_788_000_000 }), 1_788_000_045)).toBe(45);
  });

  it("stops the clock when the run completes", () => {
    expect(
      runElapsedSeconds(
        run({ status: "Completed", startedAt: 1_788_000_000, completedAt: 1_788_000_120 }),
        1_788_009_999,
      ),
    ).toBe(120);
  });
});

describe("runSecondsLeft", () => {
  // 75 shipments left at 2s each is 150s, which reads as three minutes rather than a
  // figure that ticks down with the clock and jumps back up at the next poll.
  it("projects the rest of the run from its pace so far, to the minute", () => {
    expect(
      runSecondsLeft(
        run({ startedAt: 1_788_000_000, totalCount: 100, processedCount: 25 }),
        1_788_000_050,
      ),
    ).toBe(180);
  });

  it("rounds a last minute to ten seconds", () => {
    expect(
      runSecondsLeft(
        run({ startedAt: 1_788_000_000, totalCount: 100, processedCount: 80 }),
        1_788_000_100,
      ),
    ).toBe(30);
  });

  // A handful of shipments, or a handful of seconds, is no pace to project from:
  // the estimate would swing from seconds to hours between polls.
  it("promises nothing before the pace settles", () => {
    expect(
      runSecondsLeft(
        run({ startedAt: 1_788_000_000, totalCount: 100, processedCount: 9 }),
        1_788_000_050,
      ),
    ).toBeNull();
    expect(
      runSecondsLeft(
        run({ startedAt: 1_788_000_000, totalCount: 100, processedCount: 40 }),
        1_788_000_004,
      ),
    ).toBeNull();
  });

  it("promises nothing once the run has ended or before it knows its size", () => {
    expect(
      runSecondsLeft(
        run({ status: "Canceled", startedAt: 1_788_000_000, processedCount: 40 }),
        1_788_000_050,
      ),
    ).toBeNull();
    expect(
      runSecondsLeft(run({ startedAt: 1_788_000_000, totalCount: 0 }), 1_788_000_050),
    ).toBeNull();
  });
});
