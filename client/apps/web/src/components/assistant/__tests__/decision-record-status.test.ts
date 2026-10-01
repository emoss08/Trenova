import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { recordStatus, type RecordStatusInput } from "../decision-record-status";

const t = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function status(overrides: Partial<RecordStatusInput>): string {
  return recordStatus(
    { state: "done", time: "8:52 PM", decidedByMe: true, own: false, ...overrides },
    t,
  );
}

describe("recordStatus", () => {
  it("points a waiting decision at the approval box", () => {
    expect(status({ state: "awaiting", time: "" })).toBe("Waiting — decide below");
  });

  it("says who approved it and when, and keeps approved and done apart", () => {
    expect(status({ state: "done" })).toBe("Approved by you 8:52 PM");
    expect(status({ state: "running" })).toBe("Approved by you 8:52 PM · running");
    expect(status({ state: "failed" })).toBe("Approved by you 8:52 PM · did not run");
    expect(status({ state: "simulated" })).toBe("Approved by you 8:52 PM · simulated");
  });

  it("does not claim the reader decided what someone else did", () => {
    expect(status({ decidedByMe: false })).toBe("Approved 8:52 PM");
    expect(status({ state: "declined", decidedByMe: false })).toBe("Rejected 8:52 PM");
    expect(status({ state: "declined" })).toBe("Rejected by you 8:52 PM");
    expect(status({ state: "declined", time: "" })).toBe("Rejected by you");
  });

  it("never calls a write that ran on its own approved", () => {
    expect(status({ own: true })).toBe("Done on its own");
    expect(status({ state: "running", own: true })).toBe("Running on its own");
    expect(status({ state: "failed", own: true })).toBe("Ran on its own · did not go through");
  });

  it("names what nobody decided", () => {
    expect(status({ state: "closed" })).toBe("Expired without a decision");
    expect(status({ state: "held" })).toBe("On hold");
  });
});
