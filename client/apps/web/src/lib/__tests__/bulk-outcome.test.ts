import { afterEach, describe, expect, it, vi } from "vitest";
import { translate } from "@trenova/shared/i18n/runtime";
import { notifyBulkOutcome, settleAll, type BulkOutcomeMessages } from "../bulk-outcome";

const { success, warning, error } = vi.hoisted(() => ({
  success: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success, warning, error },
}));

function ptoMessages(verb: "Approved" | "Rejected", skipped?: number): BulkOutcomeMessages {
  return {
    succeeded: (count) =>
      verb === "Approved"
        ? translate("Approved {0, plural, one {# PTO request} other {# PTO requests}}", count)
        : translate("Rejected {0, plural, one {# PTO request} other {# PTO requests}}", count),
    partial: (succeeded, failed) =>
      verb === "Approved"
        ? translate(
            "Approved {0, plural, one {# PTO request} other {# PTO requests}}; {1} failed",
            succeeded,
            failed,
          )
        : translate(
            "Rejected {0, plural, one {# PTO request} other {# PTO requests}}; {1} failed",
            succeeded,
            failed,
          ),
    allFailed: (failed) =>
      translate(
        "All {0, plural, one {# selected PTO request} other {# selected PTO requests}} failed",
        failed,
      ),
    skipped,
  };
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("notifyBulkOutcome", () => {
  it("reports a clean success with singular/plural nouns and skipped count", () => {
    notifyBulkOutcome({ succeeded: ["a"], failed: [] }, ptoMessages("Approved", 2));
    expect(success).toHaveBeenCalledExactlyOnceWith(
      "Approved 1 PTO request (2 ineligible skipped)",
    );
    expect(warning).not.toHaveBeenCalled();
    expect(error).not.toHaveBeenCalled();
  });

  it("warns on partial failure and lists at most three errors", () => {
    notifyBulkOutcome(
      {
        succeeded: ["a", "b"],
        failed: [
          { id: "c", error: "one" },
          { id: "d", error: "two" },
          { id: "e", error: "three" },
          { id: "f", error: "four" },
        ],
      },
      ptoMessages("Rejected"),
    );
    expect(warning).toHaveBeenCalledExactlyOnceWith("Rejected 2 PTO requests; 4 failed", {
      description: "one; two; three",
    });
  });

  it("errors when nothing succeeded", () => {
    notifyBulkOutcome(
      { succeeded: [], failed: [{ id: "c", error: "PTO is rejected and cannot be approved" }] },
      ptoMessages("Approved"),
    );
    expect(error).toHaveBeenCalledExactlyOnceWith("All 1 selected PTO request failed", {
      description: "PTO is rejected and cannot be approved",
    });
  });
});

describe("settleAll", () => {
  it("records every success and failure without stopping at the first", async () => {
    const rows = [1, 2, 3, 4, 5].map((n) => ({ id: `row-${n}`, n }));
    const outcome = await settleAll(rows, async (row) => {
      if (row.n % 2 === 0) throw new Error(`even ${row.n}`);
    });

    expect(outcome).toEqual({
      succeeded: ["row-1", "row-3", "row-5"],
      failed: [
        { id: "row-2", error: "even 2" },
        { id: "row-4", error: "even 4" },
      ],
    });
  });

  it("names a failed row by its label when one is given", async () => {
    const outcome = await settleAll(
      [{ id: "a", number: "INV-12" }],
      async () => {
        throw new Error("period closed");
      },
      { label: (row) => row.number },
    );

    expect(outcome.failed).toEqual([{ id: "a", error: "INV-12: period closed" }]);
  });

  it("never has more requests in flight than its concurrency", async () => {
    let inFlight = 0;
    let peak = 0;
    const rows = Array.from({ length: 12 }, (_, i) => ({ id: `${i}` }));
    const outcome = await settleAll(
      rows,
      async () => {
        inFlight += 1;
        peak = Math.max(peak, inFlight);
        await new Promise((resolve) => setTimeout(resolve, 1));
        inFlight -= 1;
      },
      { concurrency: 3 },
    );

    expect(peak).toBe(3);
    expect(outcome.succeeded).toHaveLength(12);
  });

  it("does nothing for no rows", async () => {
    expect(await settleAll([], async () => undefined)).toEqual({ succeeded: [], failed: [] });
  });
});
