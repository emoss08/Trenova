import { describe, expect, it } from "vitest";
import type { FilterItem } from "@trenova/shared/types/data-table";
import { activeFacetCount, facetSelection, toggleFacetValue } from "../data-table-facets";

const status = {
  key: "status",
  label: "Status",
  field: "status",
  values: [
    { value: "Delayed", label: "Delayed", count: 3 },
    { value: "InTransit", label: "In transit", count: 9 },
  ],
};

const other: FilterItem = {
  id: "f1",
  type: "filter",
  connector: "and",
  field: "proNumber",
  apiField: "proNumber",
  label: "PRO",
  operator: "contains",
  value: "S26",
  filterType: "text",
};

describe("facet filters", () => {
  it("adds a value as an `in` filter on the facet's field, leaving other filters alone", () => {
    const next = toggleFacetValue([other], status, "Delayed");
    expect(next[0]).toBe(other);
    expect(next[1]).toMatchObject({
      type: "filter",
      field: "status",
      apiField: "status",
      operator: "in",
      value: ["Delayed"],
      filterType: "select",
      filterOptions: [
        { value: "Delayed", label: "Delayed" },
        { value: "InTransit", label: "In transit" },
      ],
    });
    expect(facetSelection(next, "status")).toEqual(["Delayed"]);
  });

  it("adds and removes further values on the same filter, dropping it when empty", () => {
    const one = toggleFacetValue([], status, "Delayed");
    const two = toggleFacetValue(one, status, "InTransit");
    expect(two).toHaveLength(1);
    expect(facetSelection(two, "status")).toEqual(["Delayed", "InTransit"]);
    expect(activeFacetCount(two, [status])).toBe(2);
    const back = toggleFacetValue(toggleFacetValue(two, status, "Delayed"), status, "InTransit");
    expect(back).toEqual([]);
  });

  it("reads a selection that came from a saved view or the builder as `eq`", () => {
    const fromBuilder: FilterItem = {
      ...other,
      id: "f2",
      field: "status",
      apiField: "status",
      operator: "eq",
      value: "Delayed",
      filterType: "select",
    };
    expect(facetSelection([fromBuilder], "status")).toEqual(["Delayed"]);
  });
});
