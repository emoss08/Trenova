import { describe, expect, it } from "vitest";
import type {
  ColumnDef,
  DataTableFilterField,
  FilterItem,
  SingleFilterItem,
} from "@trenova/shared/types/data-table";
import {
  convertFilterItemsToFieldFilters,
  convertFilterItemsToFilterGroups,
  generateFilterId,
  generateGroupId,
  buildFilterItemsFromUrlState,
  columnFacetField,
  getDefaultOperatorForVariant,
  getFilterableFields,
  getOperatorLabel,
  isChangedSince,
  getOperatorsForVariant,
  initializeFilterItemsFromFilterGroups,
  isValidFilterValue,
  operatorRequiresValue,
  sanitizeFilterValue,
  updateSortField,
} from "../data-table";

function makeFilter(overrides: Partial<SingleFilterItem> = {}): SingleFilterItem {
  return {
    type: "filter",
    id: "test-1",
    connector: "and",
    field: "name",
    apiField: "name",
    label: "Name",
    operator: "contains",
    value: "foo",
    filterType: "text",
    ...overrides,
  };
}

describe("getOperatorsForVariant", () => {
  it("returns text operators for text variant", () => {
    const ops = getOperatorsForVariant("text");
    expect(ops).toContain("contains");
    expect(ops).toContain("startswith");
    expect(ops).toContain("endswith");
  });

  it("returns number operators for number variant", () => {
    const ops = getOperatorsForVariant("number");
    expect(ops).toContain("gt");
    expect(ops).toContain("lte");
    expect(ops).not.toContain("contains");
  });

  it("returns date operators for date variant", () => {
    const ops = getOperatorsForVariant("date");
    expect(ops).toContain("today");
    expect(ops).toContain("daterange");
    expect(ops).toContain("lastndays");
  });

  it("returns select operators for select variant", () => {
    const ops = getOperatorsForVariant("select");
    expect(ops).toContain("in");
    expect(ops).toContain("notin");
  });

  it("returns boolean operators for boolean variant", () => {
    expect(getOperatorsForVariant("boolean")).toEqual(["eq"]);
  });

  it("falls back to text for unknown variant", () => {
    const ops = getOperatorsForVariant("unknown" as never);
    expect(ops).toEqual(getOperatorsForVariant("text"));
  });
});

describe("getOperatorLabel", () => {
  it("returns the label for a known operator", () => {
    expect(getOperatorLabel("contains")).toBe("contains");
    expect(getOperatorLabel("eq")).toBe("equals");
    expect(getOperatorLabel("isnull")).toBe("is empty");
  });

  it("falls back to raw operator string for unknown", () => {
    expect(getOperatorLabel("bogus" as never)).toBe("bogus");
  });
});

describe("getDefaultOperatorForVariant", () => {
  it("returns contains for text", () => {
    expect(getDefaultOperatorForVariant("text")).toBe("contains");
  });

  it("returns eq for number, date, select, boolean", () => {
    expect(getDefaultOperatorForVariant("number")).toBe("eq");
    expect(getDefaultOperatorForVariant("date")).toBe("eq");
    expect(getDefaultOperatorForVariant("select")).toBe("eq");
    expect(getDefaultOperatorForVariant("boolean")).toBe("eq");
  });

  it("returns eq for unknown variant", () => {
    expect(getDefaultOperatorForVariant("unknown" as never)).toBe("eq");
  });
});

describe("operatorRequiresValue", () => {
  it("returns true for value-requiring operators", () => {
    expect(operatorRequiresValue("eq")).toBe(true);
    expect(operatorRequiresValue("contains")).toBe(true);
    expect(operatorRequiresValue("gt")).toBe(true);
    expect(operatorRequiresValue("in")).toBe(true);
  });

  it("returns false for no-value operators", () => {
    expect(operatorRequiresValue("isnull")).toBe(false);
    expect(operatorRequiresValue("isnotnull")).toBe(false);
    expect(operatorRequiresValue("today")).toBe(false);
    expect(operatorRequiresValue("yesterday")).toBe(false);
    expect(operatorRequiresValue("tomorrow")).toBe(false);
  });
});

describe("isValidFilterValue", () => {
  it("returns false for null, undefined, empty string", () => {
    expect(isValidFilterValue("eq", null)).toBe(false);
    expect(isValidFilterValue("eq", undefined)).toBe(false);
    expect(isValidFilterValue("eq", "")).toBe(false);
  });

  it("returns false for empty array", () => {
    expect(isValidFilterValue("in", [])).toBe(false);
  });

  it("returns true for valid values", () => {
    expect(isValidFilterValue("eq", "hello")).toBe(true);
    expect(isValidFilterValue("eq", 0)).toBe(true);
    expect(isValidFilterValue("in", ["a", "b"])).toBe(true);
  });

  it("always returns true for no-value operators", () => {
    expect(isValidFilterValue("isnull", null)).toBe(true);
    expect(isValidFilterValue("isnotnull", undefined)).toBe(true);
    expect(isValidFilterValue("today", "")).toBe(true);
  });
});

describe("sanitizeFilterValue", () => {
  it("trims strings", () => {
    expect(sanitizeFilterValue("  hello  ")).toBe("hello");
    expect(sanitizeFilterValue("  ")).toBe("");
  });

  it("filters nulls from arrays", () => {
    expect(sanitizeFilterValue(["a", null, "b", undefined])).toEqual(["a", "b"]);
  });

  it("passes other types through unchanged", () => {
    expect(sanitizeFilterValue(42)).toBe(42);
    expect(sanitizeFilterValue(true)).toBe(true);
    expect(sanitizeFilterValue(null)).toBe(null);
  });
});

describe("updateSortField", () => {
  it("adds a new sort field", () => {
    const result = updateSortField([], "name", "asc");
    expect(result).toEqual([{ field: "name", direction: "asc" }]);
  });

  it("updates direction of existing sort field", () => {
    const current = [{ field: "name", direction: "asc" as const }];
    const result = updateSortField(current, "name", "desc");
    expect(result).toEqual([{ field: "name", direction: "desc" }]);
  });

  it("removes sort field when direction is null", () => {
    const current = [
      { field: "name", direction: "asc" as const },
      { field: "age", direction: "desc" as const },
    ];
    const result = updateSortField(current, "name", null);
    expect(result).toEqual([{ field: "age", direction: "desc" }]);
  });

  it("preserves other sort fields when updating", () => {
    const current = [
      { field: "name", direction: "asc" as const },
      { field: "age", direction: "desc" as const },
    ];
    const result = updateSortField(current, "name", "desc");
    expect(result).toEqual([
      { field: "name", direction: "desc" },
      { field: "age", direction: "desc" },
    ]);
  });
});

describe("convertFilterItemsToFilterGroups", () => {
  it("returns empty array for empty input", () => {
    expect(convertFilterItemsToFilterGroups([])).toEqual([]);
  });

  it("converts a single filter to a single group", () => {
    const items: FilterItem[] = [makeFilter({ value: "test" })];
    const groups = convertFilterItemsToFilterGroups(items);
    expect(groups).toHaveLength(1);
    expect(groups[0].filters).toHaveLength(1);
    expect(groups[0].filters[0]).toEqual({
      field: "name",
      operator: "contains",
      value: "test",
    });
  });

  it("puts AND-connected filters in separate groups", () => {
    const items: FilterItem[] = [
      makeFilter({ id: "1", value: "a" }),
      makeFilter({ id: "2", connector: "and", field: "age", apiField: "age", value: "25" }),
    ];
    const groups = convertFilterItemsToFilterGroups(items);
    expect(groups).toHaveLength(2);
    expect(groups[0].filters[0].field).toBe("name");
    expect(groups[1].filters[0].field).toBe("age");
  });

  it("puts OR-connected filters in the same group", () => {
    const items: FilterItem[] = [
      makeFilter({ id: "1", value: "a" }),
      makeFilter({ id: "2", connector: "or", field: "age", apiField: "age", value: "25" }),
    ];
    const groups = convertFilterItemsToFilterGroups(items);
    expect(groups).toHaveLength(1);
    expect(groups[0].filters).toHaveLength(2);
  });

  it("handles mixed AND/OR connectors", () => {
    const items: FilterItem[] = [
      makeFilter({ id: "1", value: "a" }),
      makeFilter({ id: "2", connector: "or", value: "b" }),
      makeFilter({ id: "3", connector: "and", field: "age", apiField: "age", value: "25" }),
    ];
    const groups = convertFilterItemsToFilterGroups(items);
    expect(groups).toHaveLength(2);
    expect(groups[0].filters).toHaveLength(2);
    expect(groups[1].filters).toHaveLength(1);
  });

  it("handles nested group items", () => {
    const items: FilterItem[] = [
      {
        type: "group",
        id: "g1",
        connector: "and",
        items: [
          makeFilter({ id: "1", value: "a" }),
          makeFilter({ id: "2", connector: "or", value: "b" }),
        ],
      },
    ];
    const groups = convertFilterItemsToFilterGroups(items);
    expect(groups).toHaveLength(1);
    expect(groups[0].filters).toHaveLength(2);
  });

  it("skips filters with invalid values", () => {
    const items: FilterItem[] = [
      makeFilter({ id: "1", value: "" }),
      makeFilter({ id: "2", value: "valid" }),
    ];
    const groups = convertFilterItemsToFilterGroups(items);
    expect(groups).toHaveLength(1);
    expect(groups[0].filters[0].value).toBe("valid");
  });
});

describe("convertFilterItemsToFieldFilters", () => {
  it("converts flat filters to FieldFilter array", () => {
    const items: FilterItem[] = [
      makeFilter({ value: "test" }),
      makeFilter({ id: "2", field: "age", apiField: "age", operator: "eq", value: 5 }),
    ];
    const result = convertFilterItemsToFieldFilters(items);
    expect(result).toEqual([
      { field: "name", operator: "contains", value: "test" },
      { field: "age", operator: "eq", value: 5 },
    ]);
  });

  it("returns null if groups are present", () => {
    const items: FilterItem[] = [
      {
        type: "group",
        id: "g1",
        connector: "and",
        items: [makeFilter()],
      },
    ];
    expect(convertFilterItemsToFieldFilters(items)).toBeNull();
  });
});

describe("round-trip: filterGroups ↔ filterItems", () => {
  it("preserves filter semantics through round-trip", () => {
    const items: FilterItem[] = [
      makeFilter({ id: "1", value: "hello", operator: "eq" }),
      makeFilter({
        id: "2",
        connector: "and",
        field: "status",
        apiField: "status",
        operator: "in",
        value: ["active", "pending"],
        filterType: "select",
      }),
    ];

    const groups = convertFilterItemsToFilterGroups(items);
    const restored = initializeFilterItemsFromFilterGroups(groups, []);

    expect(restored).toHaveLength(2);
    expect(restored[0]).toMatchObject({
      type: "filter",
      apiField: "name",
      operator: "eq",
      value: "hello",
    });
    expect(restored[1]).toMatchObject({
      type: "filter",
      apiField: "status",
      operator: "in",
      value: ["active", "pending"],
    });
  });
});

describe("generateFilterId", () => {
  it("returns a string matching the filter-* pattern", () => {
    expect(generateFilterId()).toMatch(/^filter-\d+-[a-z0-9]+$/);
  });

  it("generates unique ids", () => {
    const ids = new Set(Array.from({ length: 50 }, () => generateFilterId()));
    expect(ids.size).toBe(50);
  });
});

describe("generateGroupId", () => {
  it("returns a string matching the group-* pattern", () => {
    expect(generateGroupId()).toMatch(/^group-\d+-[a-z0-9]+$/);
  });
});

type ShipmentRow = { id: string; customer: { name: string }; status: string; notes: string };

const shipmentColumns: ColumnDef<ShipmentRow>[] = [
  {
    id: "customer",
    accessorKey: "customer",
    meta: {
      apiField: "customer.name",
      label: "Customer name",
      filterable: true,
      filterType: "text",
      defaultFilterOperator: "contains",
      extraFilters: [
        { apiField: "customerId", label: "Customer", filterType: "record", filterRecord: "CUSTOMER" },
      ],
    },
  },
  {
    id: "status",
    accessorKey: "status",
    meta: { apiField: "status", label: "Status", filterable: true, filterType: "select" },
  },
  { id: "notes", accessorKey: "notes", meta: { apiField: "notes", label: "Notes" } },
];

const tableFields: DataTableFilterField[] = [
  { apiField: "ownerId", label: "Owner", filterType: "record", filterRecord: "USER" },
  { apiField: "customerId", label: "Customer again", filterType: "record", filterRecord: "CUSTOMER" },
];

describe("getFilterableFields", () => {
  it("offers each column, then the column's extra filters, then the table's own fields", () => {
    const fields = getFilterableFields(shipmentColumns, tableFields);

    expect(fields.map((field) => field.apiField)).toEqual([
      "customer.name",
      "customerId",
      "status",
      "ownerId",
    ]);
  });

  it("leaves out a column that is not filterable", () => {
    const fields = getFilterableFields(shipmentColumns);

    expect(fields.some((field) => field.apiField === "notes")).toBe(false);
  });

  it("offers a field named twice once, as the first that names it", () => {
    const customer = getFilterableFields(shipmentColumns, tableFields).filter(
      (field) => field.apiField === "customerId",
    );

    expect(customer).toHaveLength(1);
    expect(customer[0]).toMatchObject({ label: "Customer", id: "customer:customerId" });
  });

  it("gives a record field its record kind and opens it on 'is any of'", () => {
    const owner = getFilterableFields(shipmentColumns, tableFields).find(
      (field) => field.apiField === "ownerId",
    );

    expect(owner).toMatchObject({
      id: "field:ownerId",
      filterType: "record",
      filterRecord: "USER",
      defaultOperator: "in",
    });
  });

  it("keeps a column's own default operator", () => {
    const name = getFilterableFields(shipmentColumns).find(
      (field) => field.apiField === "customer.name",
    );

    expect(name?.defaultOperator).toBe("contains");
  });
});

describe("buildFilterItemsFromUrlState", () => {
  it("restores a record filter from the address with its record kind and label", () => {
    const fields = getFilterableFields(shipmentColumns, tableFields);
    const [item] = buildFilterItemsFromUrlState(
      {
        fieldFilters: [{ field: "ownerId", operator: "in", value: ["usr_1", "usr_2"] }],
        filterGroups: [],
      },
      fields,
    ) as SingleFilterItem[];

    expect(item).toMatchObject({
      field: "field:ownerId",
      apiField: "ownerId",
      label: "Owner",
      filterType: "record",
      filterRecord: "USER",
      operator: "in",
      value: ["usr_1", "usr_2"],
    });
  });

  it("restores a field it no longer offers as plain text rather than dropping it", () => {
    const [item] = buildFilterItemsFromUrlState(
      { fieldFilters: [{ field: "retiredId", operator: "eq", value: "x" }], filterGroups: [] },
      getFilterableFields(shipmentColumns),
    ) as SingleFilterItem[];

    expect(item).toMatchObject({ apiField: "retiredId", filterType: "text", value: "x" });
  });

  it("restores a group of several filters joined by 'or' after the first", () => {
    const [group] = buildFilterItemsFromUrlState(
      {
        fieldFilters: [],
        filterGroups: [
          {
            filters: [
              { field: "customerId", operator: "eq", value: "cus_1" },
              { field: "status", operator: "eq", value: "New" },
            ],
          },
        ],
      },
      getFilterableFields(shipmentColumns),
    );

    expect(group).toMatchObject({
      type: "group",
      items: [
        { apiField: "customerId", connector: "and", filterRecord: "CUSTOMER" },
        { apiField: "status", connector: "or" },
      ],
    });
  });
});

describe("isChangedSince", () => {
  it("marks a row updated after the moment", () => {
    expect(isChangedSince({ id: "a", updatedAt: 200 }, 100)).toBe(true);
  });

  it("does not mark a row updated at or before the moment", () => {
    expect(isChangedSince({ id: "a", updatedAt: 100 }, 100)).toBe(false);
    expect(isChangedSince({ id: "a", updatedAt: 50 }, 100)).toBe(false);
  });

  it("marks nothing on a first visit, when there is no moment yet", () => {
    expect(isChangedSince({ id: "a", updatedAt: 200 }, 0)).toBe(false);
  });

  it("never marks a row its query did not give an updatedAt", () => {
    expect(isChangedSince({ id: "a" }, 100)).toBe(false);
    expect(isChangedSince({ id: "a", updatedAt: "200" }, 100)).toBe(false);
    expect(isChangedSince(null, 100)).toBe(false);
  });
});

describe("columnFacetField", () => {
  const column = (meta: Record<string, unknown>) =>
    ({ id: "col", columnDef: { meta } }) as unknown as Parameters<typeof columnFacetField>[0];
  const fields = getFilterableFields(shipmentColumns, tableFields);

  it("counts a select column by its own field when the table can count it", () => {
    const field = columnFacetField(
      column({ apiField: "status", filterType: "select", label: "Status" }),
      fields,
      new Set(["status"]),
    );

    expect(field).toMatchObject({ apiField: "status", filterType: "select" });
  });

  it("counts a text column by the field it names, as that field is filtered", () => {
    const field = columnFacetField(
      column({ apiField: "customer.name", filterType: "text", facetField: "customerId" }),
      fields,
      new Set(["customerId"]),
    );

    expect(field).toMatchObject({ apiField: "customerId", filterType: "record" });
  });

  it("offers nothing for a field the server cannot count", () => {
    expect(
      columnFacetField(column({ apiField: "status", filterType: "select" }), fields, new Set()),
    ).toBeNull();
  });

  it("offers nothing for a plain text column or one turned off", () => {
    const facetable = new Set(["notes", "status"]);
    expect(
      columnFacetField(column({ apiField: "notes", filterType: "text" }), fields, facetable),
    ).toBeNull();
    expect(
      columnFacetField(
        column({ apiField: "status", filterType: "select", facetField: false }),
        fields,
        facetable,
      ),
    ).toBeNull();
  });
});
