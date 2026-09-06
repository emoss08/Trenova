import { afterEach, describe, expect, it, vi } from "vitest";
import { notifyBulkOutcome } from "../bulk-outcome";

const { success, warning, error } = vi.hoisted(() => ({
  success: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
}));

vi.mock("sonner", () => ({
  toast: { success, warning, error },
}));

afterEach(() => {
  vi.clearAllMocks();
});

describe("notifyBulkOutcome", () => {
  it("reports a clean success with singular/plural nouns and skipped count", () => {
    notifyBulkOutcome(
      { succeeded: ["a"], failed: [] },
      { entity: "PTO request", verbPast: "Approved", skipped: 2 },
    );
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
      { entity: "PTO request", verbPast: "Rejected" },
    );
    expect(warning).toHaveBeenCalledExactlyOnceWith("Rejected 2 PTO requests; 4 failed", {
      description: "one; two; three",
    });
  });

  it("errors when nothing succeeded", () => {
    notifyBulkOutcome(
      { succeeded: [], failed: [{ id: "c", error: "PTO is rejected and cannot be approved" }] },
      { entity: "PTO request", verbPast: "Approved" },
    );
    expect(error).toHaveBeenCalledExactlyOnceWith("All 1 selected PTO request failed", {
      description: "PTO is rejected and cannot be approved",
    });
  });
});
