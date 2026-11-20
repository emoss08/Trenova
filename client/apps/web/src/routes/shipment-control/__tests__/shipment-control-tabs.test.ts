import { describe, expect, it } from "vitest";
import { shipmentControlSchema } from "@/types/shipment-control";
import { shipmentControlTabs } from "../shipment-control-tabs";

const RECORD_KEYS = new Set([
  "id",
  "version",
  "createdAt",
  "updatedAt",
  "organizationId",
  "businessUnitId",
]);

describe("shipment control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = shipmentControlTabs({ detention: true }).flatMap((tab) => tab.fields);
    const editable = Object.keys(shipmentControlSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("keeps the entry checks and move removals on the entry tab", () => {
    const entry = shipmentControlTabs({ detention: true }).find((tab) => tab.value === "entry");

    expect(entry?.fields).toEqual([
      "maxShipmentWeightLimit",
      "checkForDuplicateBols",
      "checkHazmatSegregation",
      "allowMoveRemovals",
    ]);
  });

  it("offers every tab, in order, when the plan includes detention", () => {
    expect(shipmentControlTabs({ detention: true }).map((tab) => tab.value)).toEqual([
      "entry",
      "automation",
      "detention",
    ]);
  });

  it("leaves out detention when the plan does not include it", () => {
    expect(shipmentControlTabs({ detention: false }).map((tab) => tab.value)).toEqual([
      "entry",
      "automation",
    ]);
  });
});
