import { describe, expect, it } from "vitest";
import { scheduleState } from "../schedule-state";

const now = 1_760_000_000;

describe("scheduleState", () => {
  it("is pending while an approved snapshot's start is still ahead", () => {
    expect(scheduleState({ status: "Active", effectiveFrom: now + 3_600 }, now)).toBe("pending");
  });

  it("has started once the start has come, or when it billed from its approval", () => {
    expect(scheduleState({ status: "Active", effectiveFrom: now }, now)).toBe("started");
    expect(scheduleState({ status: "Active", effectiveFrom: now - 86_400 }, now)).toBe("started");
    expect(scheduleState({ status: "Active", effectiveFrom: null }, now)).toBe("started");
    expect(scheduleState({ status: "Active", effectiveFrom: undefined }, now)).toBe("started");
  });

  it("has no window for a snapshot that was never approved or was retired", () => {
    expect(scheduleState({ status: "Draft", effectiveFrom: now + 3_600 }, now)).toBe("unapproved");
    expect(scheduleState({ status: "InReview", effectiveFrom: null }, now)).toBe("unapproved");
    expect(scheduleState({ status: "Inactive", effectiveFrom: now + 3_600 }, now)).toBe(
      "unapproved",
    );
  });
});
