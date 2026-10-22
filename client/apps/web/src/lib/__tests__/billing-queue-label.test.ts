import { describe, expect, it } from "vitest";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { billingQueueItemLabel } from "@/lib/billing-queue-label";

function item(fields: Partial<BillingQueueItem>): BillingQueueItem {
  return {
    id: "bqi_01K99ZRDA85S8ZXERVM2F2W3B8",
    number: "",
    shipmentId: null,
    ...fields,
  } as BillingQueueItem;
}

describe("billingQueueItemLabel", () => {
  it("names a memo item, which has no shipment, by its billing number", () => {
    expect(billingQueueItemLabel(item({ number: "DM2610000041", billType: "DebitMemo" }))).toBe(
      "DM2610000041",
    );
  });

  it("falls back to the item's own id when an item has neither number nor shipment", () => {
    expect(billingQueueItemLabel(item({}))).toBe("bqi_01K99ZRD");
  });

  it("falls back to the shipment id when a shipment item has no number or PRO", () => {
    expect(billingQueueItemLabel(item({ shipmentId: "shp_01K99ZRDA85S8ZXERVM6V0HYYN" }))).toBe(
      "shp_01K99ZRD",
    );
  });
});
