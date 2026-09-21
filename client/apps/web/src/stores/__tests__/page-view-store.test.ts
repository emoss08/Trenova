import { describe, expect, it } from "vitest";
import { composePageView, type RegisteredKpi, type RegisteredTableView } from "../page-view-store";

const table: RegisteredTableView = {
  resource: "shipment",
  query: "acme",
  fieldFilters: [{ field: "status", operator: "in", value: ["InTransit", "Delayed"] }],
  filterGroups: [{ filters: [{ field: "createdAt", operator: "lastndays", value: 7 }] }],
  sort: [{ field: "createdAt", direction: "desc" }],
  selectedIds: ["shp_1", "shp_2"],
  selectionCount: 2,
  visibleColumns: ["proNumber", "status"],
  rowCount: 42,
};

/**
 * The view is what the server's PageView accepts (domain/agent/pagecontext.go):
 * a known resource, bounded lists, and figures as the person read them. The
 * client composes it from what the table and the strip registered, and never
 * sends more than the server would take.
 */
describe("composePageView", () => {
  it("composes the table's state and the figures into one view", () => {
    const kpis: RegisteredKpi[] = [
      { id: "a", label: "In transit", value: "12", sub: "of 42" },
      { id: "b", label: "Late", value: "3" },
    ];

    expect(composePageView(table, kpis)).toEqual({
      resource: "shipment",
      query: "acme",
      fieldFilters: table.fieldFilters,
      filterGroups: table.filterGroups,
      sort: table.sort,
      selection: { count: 2, ids: ["shp_1", "shp_2"] },
      kpis: [
        { label: "In transit", value: "12", sub: "of 42" },
        { label: "Late", value: "3", sub: "" },
      ],
      visibleColumns: ["proNumber", "status"],
      rowCount: 42,
    });
  });

  it("is nothing without a table, and figures alone are not a view", () => {
    expect(composePageView(null, [{ id: "a", label: "Late", value: "3" }])).toBeNull();
  });

  it("drops a table whose resource the permission registry does not know", () => {
    expect(composePageView({ ...table, resource: "Shipment" }, [])).toMatchObject({
      resource: "shipment",
    });
    expect(composePageView({ ...table, resource: "command_center" }, [])).toBeNull();
  });

  it("keeps every list within the server's bounds", () => {
    const many = (n: number) =>
      Array.from({ length: n }, (_, i) => ({ field: `f${i}`, operator: "eq", value: i }));
    const view = composePageView(
      {
        ...table,
        fieldFilters: many(25),
        filterGroups: Array.from({ length: 7 }, () => ({ filters: many(3) })),
        sort: Array.from({ length: 8 }, (_, i) => ({ field: `s${i}`, direction: "asc" })),
        selectedIds: Array.from({ length: 30 }, (_, i) => `shp_${i}`),
        selectionCount: 30,
        visibleColumns: Array.from({ length: 50 }, (_, i) => `c${i}`),
      },
      Array.from({ length: 15 }, (_, i) => ({ id: `k${i}`, label: `Figure ${i}`, value: `${i}` })),
    );

    expect(view?.fieldFilters).toHaveLength(20);
    expect(view?.filterGroups).toHaveLength(5);
    expect(view?.sort).toHaveLength(5);
    expect(view?.selection).toEqual({
      count: 30,
      ids: Array.from({ length: 25 }, (_, i) => `shp_${i}`),
    });
    expect(view?.kpis).toHaveLength(12);
    expect(view?.visibleColumns).toHaveLength(40);
  });

  it("leaves out an empty selection and an unknown row count", () => {
    const view = composePageView(
      { ...table, selectedIds: [], selectionCount: 0, rowCount: null },
      [],
    );

    expect(view?.selection).toBeUndefined();
    expect(view?.rowCount).toBeUndefined();
    expect(view?.kpis).toBeUndefined();
  });

  it("skips a figure whose label or value could not be read as text", () => {
    const view = composePageView(table, [
      { id: "a", label: "", value: "3" },
      { id: "b", label: "Late", value: "3" },
    ]);

    expect(view?.kpis).toEqual([{ label: "Late", value: "3", sub: "" }]);
  });
});
