import { describe, expect, it } from "vitest";
import { nextItemAfterReassignment } from "@/lib/billing-queue-charges";

describe("nextItemAfterReassignment", () => {
  it("lands on another active item for the same shipment", () => {
    expect(
      nextItemAfterReassignment({ id: "bqi_1", shipmentId: "shp_1" }, {
        items: [
          { id: "bqi_9", shipmentId: "shp_2" },
          { id: "bqi_2", shipmentId: "shp_1" },
        ],
        canceledItemIds: ["bqi_1"],
      } as never),
    ).toBe("bqi_2");
  });

  it("never treats two shipment-less memo items as the same shipment", () => {
    expect(
      nextItemAfterReassignment({ id: "bqi_memo_1", shipmentId: null }, {
        items: [{ id: "bqi_memo_2", shipmentId: null }],
        canceledItemIds: ["bqi_memo_1"],
      } as never),
    ).toBeNull();
  });

  it("stays put when the item on screen was not canceled", () => {
    expect(
      nextItemAfterReassignment({ id: "bqi_1", shipmentId: "shp_1" }, {
        items: [{ id: "bqi_2", shipmentId: "shp_1" }],
        canceledItemIds: [],
      } as never),
    ).toBeUndefined();
  });
});
