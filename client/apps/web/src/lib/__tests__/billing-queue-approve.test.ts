import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { ShipmentBillingReadiness } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import { approveBlocker } from "../billing-queue-approve";

function item(overrides: Partial<BillingQueueItem> = {}): BillingQueueItem {
  return {
    id: "bqi_1",
    status: "InReview",
    detentionHolds: [],
    ratingReviewHold: null,
    review: { checks: [], issues: [], needsCount: 0, ready: true, blocker: "", documents: [] },
    ...overrides,
  } as unknown as BillingQueueItem;
}

function readiness(overrides: Partial<ShipmentBillingReadiness>): ShipmentBillingReadiness {
  return {
    canMarkReadyToInvoice: true,
    missingRequirements: [],
    validationFailures: [],
    ...overrides,
  } as unknown as ShipmentBillingReadiness;
}

describe("approveBlocker", () => {
  it("lets an item with a clear review and met requirements be approved", () => {
    expect(approveBlocker(item(), readiness({}))).toBeNull();
    expect(approveBlocker(item(), undefined)).toBeNull();
  });

  it("counts every review item that needs the biller", () => {
    const flagged = item({
      review: { ...item().review!, needsCount: 3, ready: false, blocker: "issue" },
    });
    expect(approveBlocker(flagged, readiness({}))).toEqual({ kind: "review", count: 3 });
  });

  it("names what waits on someone else ahead of the review items it also raises", () => {
    const held = item({
      detentionHolds: [{}, {}] as never,
      review: { ...item().review!, needsCount: 2, ready: false, blocker: "issue" },
    });
    expect(approveBlocker(held, readiness({}))).toEqual({ kind: "detention", count: 2 });
    expect(approveBlocker(item({ ratingReviewHold: {} as never }), readiness({}))).toEqual({
      kind: "rating",
    });
  });

  it("names missing paperwork, then the shipment's own reason", () => {
    expect(
      approveBlocker(
        item(),
        readiness({ canMarkReadyToInvoice: false, missingRequirements: [{}] as never }),
      ),
    ).toEqual({ kind: "documents", count: 1 });
    expect(
      approveBlocker(
        item(),
        readiness({
          canMarkReadyToInvoice: false,
          validationFailures: [{ message: "A reason is required" }] as never,
        }),
      ),
    ).toEqual({ kind: "requirement", message: "A reason is required" });
  });
});
