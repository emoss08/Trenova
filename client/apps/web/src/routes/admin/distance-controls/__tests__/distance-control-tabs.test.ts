import { describe, expect, it } from "vitest";
import { distanceControlSchema } from "@/types/distance-control";
import { distanceControlTabs } from "../distance-control-tabs";

const RECORD_KEYS = new Set([
  "id",
  "version",
  "createdAt",
  "updatedAt",
  "organizationId",
  "businessUnitId",
]);

describe("distance control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = distanceControlTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(distanceControlSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("has one tab, mileage, for every organization", () => {
    expect(distanceControlTabs().map((tab) => tab.value)).toEqual(["mileage"]);
  });
});
