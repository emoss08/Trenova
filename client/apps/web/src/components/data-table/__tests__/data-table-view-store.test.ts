import { tableLayoutSchema, type ActiveTableView } from "@/types/table-configuration";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createDataTableViewStore } from "../data-table-view-store";

// A saved layout as the server returns it (myTableLayout), read through its schema.
const saved = tableLayoutSchema.parse({
  columnVisibility: { status: false },
  columnOrder: ["number", "status"],
  columnSizing: { number: 120 },
  columnPinning: { left: ["number"], right: [] },
  formatRules: [],
  density: "compact",
  activeViewId: "tcf_01",
  pinnedRowIds: ["shp_1", "shp_2"],
  pinnedRowsCollapsed: true,
  lastSeenAt: 1_779_811_265,
  markChangesSinceLastVisit: true,
  hideTotals: true,
  virtualized: true,
});

const view: ActiveTableView = {
  id: "tcf_01",
  name: "Late loads",
  config: {
    fieldFilters: [],
    filterGroups: [],
    joinOperator: "and",
    sort: [],
    pageSize: 25,
    columnVisibility: {},
    columnOrder: [],
    columnSizing: {},
    columnPinning: { left: [], right: [] },
    density: "comfortable",
    formatRules: [],
  },
};

afterEach(() => {
  vi.useRealTimers();
});

describe("a table's view store", () => {
  it("opens with the table's own defaults and no rows drawn yet", () => {
    const state = createDataTableViewStore("compact").getState();

    expect(state).toMatchObject({
      ready: false,
      density: "compact",
      formatRules: [],
      rowPinning: { top: [], bottom: [] },
      pinnedRowsCollapsed: false,
      hideTotals: false,
      virtualized: false,
      markChangesSinceLastVisit: false,
      changesSince: 0,
      activeView: null,
      selectingAll: false,
      bulkEditSelection: null,
      bulkEditJobId: null,
    });
    expect(state.seenRowIds.size).toBe(0);
  });

  it("lays a saved arrangement over the defaults in one change", () => {
    const store = createDataTableViewStore("comfortable");
    const listener = vi.fn();
    store.subscribe(listener);

    store.getState().actions.applySavedLayout(saved);

    expect(listener).toHaveBeenCalledTimes(1);
    expect(store.getState()).toMatchObject({
      density: "compact",
      rowPinning: { top: ["shp_1", "shp_2"], bottom: [] },
      pinnedRowsCollapsed: true,
      changesSince: 1_779_811_265,
      markChangesSinceLastVisit: true,
      hideTotals: true,
      virtualized: true,
    });
  });

  it("keeps the density it has when the saved layout names none", () => {
    const store = createDataTableViewStore("compact");

    store.getState().actions.applySavedLayout({ ...saved, density: null });

    expect(store.getState().density).toBe("compact");
  });

  it("goes back to the table's defaults, leaving the saved view, on a reset", () => {
    const store = createDataTableViewStore("comfortable");
    store.getState().actions.applySavedLayout(saved);
    store.getState().actions.setActiveView(view);

    store.getState().actions.resetLayout();

    expect(store.getState()).toMatchObject({
      density: "comfortable",
      rowPinning: { top: [], bottom: [] },
      pinnedRowsCollapsed: false,
      hideTotals: false,
      virtualized: false,
      markChangesSinceLastVisit: false,
      activeView: null,
    });
  });

  it("pins and unpins a row, and pins no more than the limit", () => {
    const store = createDataTableViewStore("comfortable");
    const { togglePinnedRow } = store.getState().actions;

    togglePinnedRow("shp_1", 2);
    togglePinnedRow("shp_2", 2);
    togglePinnedRow("shp_3", 2);
    expect(store.getState().rowPinning.top).toEqual(["shp_1", "shp_2"]);

    togglePinnedRow("shp_1", 2);
    expect(store.getState().rowPinning.top).toEqual(["shp_2"]);
  });

  it("follows the table's own pinning updates", () => {
    const store = createDataTableViewStore("comfortable");

    store.getState().actions.setRowPinning((current) => ({ ...current, top: ["shp_9"] }));

    expect(store.getState().rowPinning).toEqual({ top: ["shp_9"], bottom: [] });
  });

  it("marks an opened row seen once, and starts afresh when all are marked seen", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-15T12:00:00Z"));
    const store = createDataTableViewStore("comfortable");

    store.getState().actions.markSeen("shp_1");
    const seen = store.getState().seenRowIds;
    store.getState().actions.markSeen("shp_1");
    expect(store.getState().seenRowIds).toBe(seen);
    expect([...seen]).toEqual(["shp_1"]);

    store.getState().actions.markAllSeen();
    expect(store.getState().seenRowIds.size).toBe(0);
    expect(store.getState().changesSince).toBe(Date.parse("2026-11-15T12:00:00Z") / 1000);
  });

  it("leaves the saved view only when it is the one that went away", () => {
    const store = createDataTableViewStore("comfortable");
    store.getState().actions.setActiveView(view);

    store.getState().actions.forgetActiveView("tcf_other");
    expect(store.getState().activeView).toBe(view);

    store.getState().actions.forgetActiveView("tcf_01");
    expect(store.getState().activeView).toBeNull();
  });

  it("opens the bulk editor on a selection and tracks the edit it starts", () => {
    const store = createDataTableViewStore("comfortable");

    store.getState().actions.openBulkEdit({ ids: ["shp_1"] });
    expect(store.getState().bulkEditSelection).toEqual({ ids: ["shp_1"] });

    store.getState().actions.trackBulkEditJob("bej_1");
    store.getState().actions.closeBulkEdit();
    expect(store.getState()).toMatchObject({ bulkEditSelection: null, bulkEditJobId: "bej_1" });

    store.getState().actions.dismissBulkEditJob();
    expect(store.getState().bulkEditJobId).toBeNull();
  });

  it("gives each table a store of its own", () => {
    const first = createDataTableViewStore("comfortable");
    const second = createDataTableViewStore("comfortable");

    first.getState().actions.setDensity("compact");

    expect(second.getState().density).toBe("comfortable");
  });
});
