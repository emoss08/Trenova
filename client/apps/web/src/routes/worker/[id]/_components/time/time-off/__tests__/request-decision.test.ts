import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { describe, expect, it } from "vitest";
import { requestDecision } from "../request-decision";

const t = ((text: string) => text) as TranslateFn;

function entry(overrides: Partial<WorkerPTO>): WorkerPTO {
  return {
    status: "Requested",
    autoApproved: false,
    approver: null,
    rejector: null,
    cancelledBy: null,
    rejectionReason: null,
    cancellationReason: null,
    ...overrides,
  } as WorkerPTO;
}

describe("requestDecision", () => {
  it("is waiting only while the request is requested", () => {
    expect(requestDecision(t, entry({ status: "Requested" }))).toEqual({ kind: "waiting" });
  });

  it("names the approver of an approved request and nothing else", () => {
    expect(
      requestDecision(
        t,
        entry({
          status: "Approved",
          approver: { id: "usr_1", name: "Dana Whitfield" },
          rejectionReason: "stale",
        }),
      ),
    ).toEqual({ kind: "decided", actor: "Dana Whitfield", note: null });
  });

  it("says an approval with no approver was automatic, or leaves a dash when it was not", () => {
    expect(requestDecision(t, entry({ status: "Approved", autoApproved: true }))).toEqual({
      kind: "decided",
      actor: "Approved automatically",
      note: null,
    });
    expect(requestDecision(t, entry({ status: "Approved" }))).toEqual({
      kind: "decided",
      actor: null,
      note: null,
    });
  });

  it("names who rejected a request and why", () => {
    expect(
      requestDecision(
        t,
        entry({
          status: "Rejected",
          rejector: { id: "usr_2", name: "Marcus Bell" },
          rejectionReason: "Peak week, short on drivers",
        }),
      ),
    ).toEqual({ kind: "decided", actor: "Marcus Bell", note: "Peak week, short on drivers" });
  });

  it("names who cancelled a request, with the reason only when one was given", () => {
    expect(
      requestDecision(
        t,
        entry({
          status: "Cancelled",
          cancelledBy: { id: "usr_3", name: "Maria Alvarez" },
          cancellationReason: "  ",
        }),
      ),
    ).toEqual({ kind: "decided", actor: "Maria Alvarez", note: null });
    expect(
      requestDecision(
        t,
        entry({
          status: "Cancelled",
          cancelledBy: { id: "usr_3", name: "Maria Alvarez" },
          cancellationReason: "Plans changed",
        }),
      ),
    ).toEqual({ kind: "decided", actor: "Maria Alvarez", note: "Plans changed" });
  });
});
