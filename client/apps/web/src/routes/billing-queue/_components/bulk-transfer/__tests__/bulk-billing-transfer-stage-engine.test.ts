import type { BillingTransferRun } from "@/lib/graphql/billing-transfer";
import { describe, expect, it, vi } from "vitest";
import {
  heldTotal,
  pickStageGroup,
  stageCountsOf,
  TransferStageEngine,
  stageDeficit,
  type StageCounts,
} from "../bulk-billing-transfer-stage-engine";
import { sheetMarksFor } from "../bulk-billing-transfer-stage-sheet";

const STEPS = ["status", "documents", "rate", "billed"] as const;

function counts(overrides: Partial<StageCounts> = {}): StageCounts {
  return { transferred: 0, held: {}, lastHeld: null, ...overrides };
}

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

describe("stageCountsOf", () => {
  it("reads the transferred count and each reason's count off the run", () => {
    expect(
      stageCountsOf(
        run({
          transferredCount: 7,
          failureCounts: [
            { code: "RequirementsUnmet", count: 2, retryable: true },
            { code: "AlreadyTransferred", count: 1, retryable: false },
          ],
        }),
      ),
    ).toEqual({
      transferred: 7,
      held: { RequirementsUnmet: 2, AlreadyTransferred: 1 },
      lastHeld: "RequirementsUnmet",
    });
  });

  it("starts with nothing held back when no reason has a count", () => {
    expect(stageCountsOf(run({ transferredCount: 3 }))).toEqual({
      transferred: 3,
      held: {},
      lastHeld: null,
    });
  });
});

describe("heldTotal", () => {
  it("adds every reason's count", () => {
    expect(heldTotal(counts({ held: { RateValidation: 2, NotFound: 3 } }))).toBe(5);
  });
});

describe("stageDeficit", () => {
  // Only what the run has answered since the stage last sent a sheet is left to draw;
  // what is already on the stacks or in the air is never drawn twice.
  it("is what the run has answered beyond what the stage has sent", () => {
    expect(
      stageDeficit(
        counts({ transferred: 9, held: { RequirementsUnmet: 3, RateValidation: 1 } }),
        counts({ transferred: 4, held: { RequirementsUnmet: 3 } }),
      ),
    ).toEqual([
      { code: null, count: 5 },
      { code: "RateValidation", count: 1 },
    ]);
  });

  it("is empty once the stage has sent everything the run answered", () => {
    const answered = counts({ transferred: 4, held: { NotFound: 1 } });
    expect(stageDeficit(answered, answered)).toEqual([]);
  });
});

describe("pickStageGroup", () => {
  const waiting = [
    { code: null, count: 6 },
    { code: "RateValidation" as const, count: 2 },
  ];

  it("picks an outcome in proportion to what is waiting", () => {
    expect(pickStageGroup(waiting, 0)?.code).toBeNull();
    expect(pickStageGroup(waiting, 0.74)?.code).toBeNull();
    expect(pickStageGroup(waiting, 0.76)?.code).toBe("RateValidation");
    expect(pickStageGroup(waiting, 1)?.code).toBe("RateValidation");
  });

  // Everything waiting with its outcome, so the desk never queues sheets the run answered
  // long ago and the stage stays within one sheet of the run.
  it("carries everything waiting with the outcome it picks", () => {
    expect(pickStageGroup(waiting, 0)).toEqual({ code: null, count: 6 });
    expect(pickStageGroup(waiting, 0.9)).toEqual({ code: "RateValidation", count: 2 });
  });

  it("carries one shipment when one is waiting", () => {
    expect(pickStageGroup([{ code: "NotFound", count: 1 }], 0.5)).toEqual({
      code: "NotFound",
      count: 1,
    });
  });

  it("sends nothing when nothing is waiting", () => {
    expect(pickStageGroup([], 0.5)).toBeNull();
  });
});

describe("TransferStageEngine", () => {
  it("lands a finished run at once with the run's own counts, then hands over", () => {
    vi.useFakeTimers();
    try {
      const onLanded = vi.fn();
      const onSettled = vi.fn();
      const engine = new TransferStageEngine(
        { anchors: () => null, onSteps: vi.fn(), onLanded, onSettled },
        counts(),
      );
      const finished = counts({ transferred: 17, held: { AlreadyTransferred: 4 } });

      engine.update(finished, true);

      expect(onLanded).toHaveBeenLastCalledWith(finished);
      expect(onSettled).not.toHaveBeenCalled();
      vi.advanceTimersByTime(900);
      expect(onSettled).toHaveBeenCalledOnce();
      engine.dispose();
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("sheetMarksFor", () => {
  it("passes every section of a bill that transferred", () => {
    expect(sheetMarksFor(null, STEPS)).toEqual({
      status: "pass",
      documents: "pass",
      rate: "pass",
      billed: "pass",
    });
  });

  it("passes the sections before the failing one and leaves the rest unread", () => {
    expect(sheetMarksFor("rate", STEPS)).toEqual({
      status: "pass",
      documents: "pass",
      rate: "fail",
    });
  });
});
