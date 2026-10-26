import type { KeelDefinition } from "@/lib/report-definition";
import type { ReportDashboardFilter, ReportDashboardTile } from "@/types/report";
import { describe, expect, it } from "vitest";
import { applyTileFilters, type TileFilterState } from "../dashboard-state";

/**
 * A tile on screen and its sheet in the dashboard export fold the dashboard's
 * filters into the tile's report with the same function: keel's applyFilters on
 * the client, ir.ApplyFilters on the server. These cases are the contract that
 * function documents, so a tile and its export never read different rows.
 */

const tile: ReportDashboardTile = { id: "tile_1", kind: "table", x: 0, y: 0, w: 6, h: 4 };

function shipmentReport(overrides: Partial<KeelDefinition> = {}): KeelDefinition {
  return {
    version: 2,
    entity: "shipment",
    columns: [{ id: "pro", dimension: { field: "proNumber" } }],
    ...overrides,
  };
}

const statusFilter: ReportDashboardFilter = {
  id: "f_status",
  entity: "shipment",
  ref: { field: "status" },
  operator: "in",
};

function state(overrides: Partial<TileFilterState>): TileFilterState {
  return {
    filters: [],
    filterValues: {},
    tileFilters: [],
    tileFilterValues: {},
    params: {},
    ...overrides,
  };
}

describe("applyTileFilters", () => {
  it("replaces the value of the report's own condition on the same field, not adding one", () => {
    const report = shipmentReport({
      filters: {
        and: [
          { field: "status", op: "eq", value: "New" },
          { field: "customer.name", op: "eq", value: "Acme" },
        ],
      },
    });

    const { definition } = applyTileFilters(
      report,
      tile,
      state({ filters: [statusFilter], filterValues: { f_status: ["Completed"] } }),
    );

    expect(definition.filters).toEqual({
      and: [
        { field: "status", op: "in", value: ["Completed"] },
        { field: "customer.name", op: "eq", value: "Acme" },
      ],
    });
    expect(report.filters).toEqual({
      and: [
        { field: "status", op: "eq", value: "New" },
        { field: "customer.name", op: "eq", value: "Acme" },
      ],
    });
  });

  it("ANDs a new condition on top of the report's filters, keeping their grouping", () => {
    const own = { or: [{ field: "customer.name", op: "eq" as const, value: "Acme" }] };

    const { definition } = applyTileFilters(
      shipmentReport({ filters: own }),
      tile,
      state({ filters: [statusFilter], filterValues: { f_status: ["Completed"] } }),
    );

    expect(definition.filters).toEqual({
      and: [{ field: "status", op: "in", value: ["Completed"] }, own],
    });
  });

  it("lets the tile's own filter win over the dashboard's on the same field", () => {
    const tileStatus: ReportDashboardFilter = { ...statusFilter, id: "t_status", operator: "eq" };

    const { definition } = applyTileFilters(
      shipmentReport(),
      tile,
      state({
        filters: [statusFilter],
        filterValues: { f_status: ["New"] },
        tileFilters: [tileStatus],
        tileFilterValues: { t_status: "Completed" },
      }),
    );

    expect(definition.filters).toEqual({
      and: [{ field: "status", op: "eq", value: "Completed" }],
    });
  });

  it("leaves a report on another entity, and a filter with no value, alone", () => {
    const report = shipmentReport();
    const invoiceFilter: ReportDashboardFilter = { ...statusFilter, entity: "invoice" };

    const { definition } = applyTileFilters(
      report,
      tile,
      state({
        filters: [invoiceFilter, { ...statusFilter, id: "f_empty" }],
        filterValues: { f_status: ["Paid"], f_empty: [] },
      }),
    );

    expect(definition).toBe(report);
  });

  it("feeds each report parameter from its binding, else the dashboard value of its name", () => {
    const { params } = applyTileFilters(
      shipmentReport({
        parameters: [
          { name: "windowDays", type: "int" },
          { name: "region", type: "string" },
          { name: "unset", type: "string" },
        ],
      }),
      { ...tile, paramBindings: { region: "dashboardRegion" } },
      state({ params: { windowDays: 30, dashboardRegion: "West", unrelated: 1 } }),
    );

    expect(params).toEqual({ windowDays: 30, region: "West" });
  });

  it("passes no parameters to a report that declares none", () => {
    const { params } = applyTileFilters(
      shipmentReport(),
      tile,
      state({ params: { windowDays: 30 } }),
    );

    expect(params).toBeUndefined();
  });
});
