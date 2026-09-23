import type { ToolCatalogEntry } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  effectiveTier,
  groupToolsByResource,
  resourceLabel,
  impliedReads,
  splitCoreTools,
  summarizeSelection,
  toggleTool,
} from "../tool-catalog";

function tool(overrides: Partial<ToolCatalogEntry>): ToolCatalogEntry {
  return {
    name: "get_shipment",
    description: "Reads one shipment",
    parameters: null,
    kind: "query",
    resource: "shipment",
    operation: "read",
    defaultAutonomyTier: "",
    reversible: false,
    core: false,
    prerequisites: [],
    ...overrides,
  };
}

const catalog = [
  tool({ name: "get_shipment" }),
  tool({ name: "cancel_shipment", kind: "action", description: "Cancels a shipment" }),
  tool({ name: "list_workers", resource: "worker", description: "Lists drivers" }),
  tool({ name: "approve_worker_pto", kind: "action", resource: "worker_pto" }),
];

describe("groupToolsByResource", () => {
  it("groups alphabetically by label with reads before changes", () => {
    const groups = groupToolsByResource(catalog, ["cancel_shipment", "get_shipment"]);

    expect(groups.map((g) => g.label)).toEqual(["Shipment", "Worker", "Worker time off"]);
    expect(groups[0].tools.map((t) => t.name)).toEqual(["get_shipment", "cancel_shipment"]);
    expect(groups[0].chosen).toBe(2);
    expect(groups[1].chosen).toBe(0);
  });

  // The rail keeps saying what the agent holds while the list is narrowed,
  // so a search never makes a group look emptier than it is.
  it("narrows the tools by a query but counts the chosen over the whole group", () => {
    const groups = groupToolsByResource(catalog, ["cancel_shipment", "get_shipment"], "drivers");

    expect(groups.map((g) => g.label)).toEqual(["Worker"]);
    expect(
      groupToolsByResource(catalog, ["cancel_shipment", "get_shipment"], "cancel")[0],
    ).toMatchObject({
      label: "Shipment",
      chosen: 2,
    });
  });

  it("reads an abbreviation the way a person says it", () => {
    expect(resourceLabel("worker_pto")).toBe("Worker time off");
    expect(resourceLabel("")).toBe("General");
  });
});

describe("summarizeSelection", () => {
  it("counts reads and changes, with each change at its effective tier", () => {
    const summary = summarizeSelection(
      ["get_shipment", "cancel_shipment", "approve_worker_pto", "retired_tool"],
      catalog,
      { cancel_shipment: "AutoExecute" },
      "ActWithApproval",
    );

    expect(summary.reads).toBe(1);
    expect(summary.changes).toBe(2);
    expect(summary.byTier).toEqual({ Propose: 0, ActWithApproval: 2, AutoExecute: 0 });
    expect(summary.unknown).toEqual(["retired_tool"]);
  });

  it("caps a tool's own tier at the ceiling", () => {
    expect(effectiveTier("x", { x: "AutoExecute" }, "Propose")).toBe("Propose");
    expect(effectiveTier("x", { x: "Propose" }, "AutoExecute")).toBe("Propose");
    expect(effectiveTier("x", {}, "ActWithApproval")).toBe("ActWithApproval");
  });
});

describe("toggleTool", () => {
  it("adds once and drops the tier with the tool", () => {
    const added = toggleTool(["a"], { a: "Propose" }, "b", true);
    expect(added.selected).toEqual(["a", "b"]);
    expect(toggleTool(added.selected, added.tiers, "b", true).selected).toEqual(["a", "b"]);

    const removed = toggleTool(["a", "b"], { a: "Propose", b: "AutoExecute" }, "b", false);
    expect(removed.selected).toEqual(["a"]);
    expect(removed.tiers).toEqual({ a: "Propose" });
  });
});

/**
 * Memory, escalation and review are held by every agent. The server marks
 * them core and strips them from a saved selection, so the picker must never
 * offer them as a choice or count them as one.
 */
describe("core tools", () => {
  const withCore = [
    ...catalog,
    tool({ name: "recall_memory", resource: "agent_memory", core: true }),
    tool({ name: "remember", kind: "action", resource: "agent_memory", core: true }),
  ];

  it("splits the always-on tools from the ones an agent chooses", () => {
    const { core, selectable } = splitCoreTools(withCore);

    expect(core.map((t) => t.name)).toEqual(["recall_memory", "remember"]);
    expect(selectable.map((t) => t.name)).toEqual(catalog.map((t) => t.name));
  });

  it("does not count a core tool an older agent still lists", () => {
    const summary = summarizeSelection(
      ["get_shipment", "recall_memory", "remember"],
      withCore,
      {},
      "Propose",
    );

    expect(summary).toMatchObject({ reads: 1, changes: 0, unknown: [] });
  });
});

/**
 * A tool whose arguments come from another holds that read with it: the
 * server grants it, so the form says so rather than leaving the agent looking
 * like it cannot find a report id.
 */
describe("impliedReads", () => {
  const withDependencies = [
    tool({ name: "list_reports", resource: "report" }),
    tool({ name: "delete_report", kind: "action", resource: "report" }),
    tool({
      name: "create_dashboard",
      kind: "action",
      resource: "report_dashboard",
      prerequisites: ["list_reports", "delete_report", "missing_tool"],
    }),
    tool({
      name: "add_dashboard_tile",
      kind: "action",
      resource: "report_dashboard",
      prerequisites: ["list_reports"],
    }),
  ];

  it("names each read a chosen tool depends on, with every tool that needs it", () => {
    const implied = impliedReads(["create_dashboard", "add_dashboard_tile"], withDependencies);

    expect(implied.map((entry) => entry.tool.name)).toEqual(["list_reports"]);
    expect(implied[0].neededBy.map((entry) => entry.name)).toEqual([
      "create_dashboard",
      "add_dashboard_tile",
    ]);
  });

  it("does not repeat a read that was chosen outright", () => {
    expect(impliedReads(["create_dashboard", "list_reports"], withDependencies)).toEqual([]);
  });
});
