import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { describe, expect, it } from "vitest";
import { bulkEligibility } from "../billing-queue-bulk";

const item = (id: string, status: BillingQueueItem["status"]) =>
  ({ id, status }) as BillingQueueItem;

const ids = (items: BillingQueueItem[]) => items.map((i) => i.id);

describe("bulkEligibility", () => {
  const selection = [
    item("ready", "ReadyForReview"),
    item("review", "InReview"),
    item("held", "OnHold"),
    item("exception", "Exception"),
    item("approved", "Approved"),
    item("posted", "Posted"),
    item("canceled", "Canceled"),
  ];
  const result = bulkEligibility(selection);

  it("posts only the approved items", () => {
    expect(ids(result.postable)).toEqual(["approved"]);
  });

  it("approves and holds the undecided items that are not already held", () => {
    expect(ids(result.approvable)).toEqual(["ready", "review", "exception"]);
    expect(ids(result.holdable)).toEqual(["ready", "review", "exception"]);
  });

  it("lets every undecided item be assigned or canceled, held ones included", () => {
    expect(ids(result.open)).toEqual(["ready", "review", "held", "exception"]);
  });

  it("offers nothing for a selection of finished items", () => {
    expect(bulkEligibility([item("p", "Posted"), item("c", "Canceled")])).toEqual({
      open: [],
      approvable: [],
      holdable: [],
      postable: [],
    });
  });
});
