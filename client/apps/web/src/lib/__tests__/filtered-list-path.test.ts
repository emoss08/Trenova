import { describe, expect, it } from "vitest";
import { filteredListPath } from "../filtered-list-path";

describe("filteredListPath", () => {
  it("carries the filters in the fieldFilters form the table reads back", () => {
    const path = filteredListPath("/shipment-management/shipments", [
      { field: "consigneeStop.locationId", operator: "in", value: ["loc_a", "loc_b"] },
    ]);

    const url = new URL(path, "https://app.test");
    expect(url.pathname).toBe("/shipment-management/shipments");
    expect(JSON.parse(url.searchParams.get("fieldFilters") ?? "")).toEqual([
      { field: "consigneeStop.locationId", operator: "in", value: ["loc_a", "loc_b"] },
    ]);
  });

  it("leaves a list with no filters as a plain path", () => {
    expect(filteredListPath("/shipment-management/shipments", [])).toBe(
      "/shipment-management/shipments",
    );
  });
});
