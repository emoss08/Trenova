import { resourceLabel } from "@/lib/resource-label";
import type { ToolCatalogEntry } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  effectiveTier,
  groupToolsByResource,
  impliedReads,
  splitCoreTools,
  summarizeSelection,
  toggleTool,
} from "../tool-catalog";

function tool(overrides: Partial<ToolCatalogEntry>): ToolCatalogEntry {
  return {
    name: "shipment_get",
    description: "Reads one shipment",
    parameters: null,
    kind: "query",
    resource: "shipment",
    operation: "read",
    defaultAutonomyTier: "",
    reversible: false,
    core: false,
    prerequisites: [],
    extension: "",
    grantedToEveryAgent: false,
    ...overrides,
  };
}

const catalog = [
  tool({ name: "shipment_get" }),
  tool({ name: "shipment_cancel", kind: "action", description: "Cancels a shipment" }),
  tool({ name: "worker_list", resource: "worker", description: "Lists drivers" }),
  tool({ name: "worker_approve_pto", kind: "action", resource: "worker_pto" }),
];

describe("groupToolsByResource", () => {
  it("groups alphabetically by label with reads before changes", () => {
    const groups = groupToolsByResource(catalog, ["shipment_cancel", "shipment_get"]);

    expect(groups.map((g) => g.label)).toEqual(["Shipment", "Worker", "Worker time off"]);
    expect(groups[0].tools.map((t) => t.name)).toEqual(["shipment_get", "shipment_cancel"]);
    expect(groups[0].chosen).toBe(2);
    expect(groups[1].chosen).toBe(0);
  });

  // The rail keeps saying what the agent holds while the list is narrowed,
  // so a search never makes a group look emptier than it is.
  it("narrows the tools by a query but counts the chosen over the whole group", () => {
    const groups = groupToolsByResource(catalog, ["shipment_cancel", "shipment_get"], "drivers");

    expect(groups.map((g) => g.label)).toEqual(["Worker"]);
    expect(
      groupToolsByResource(catalog, ["shipment_cancel", "shipment_get"], "cancel")[0],
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
      ["shipment_get", "shipment_cancel", "worker_approve_pto", "retired_tool"],
      catalog,
      { shipment_cancel: "AutoExecute" },
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
    tool({ name: "agent_list_memories", resource: "agent_memory", core: true }),
    tool({ name: "agent_create_memory", kind: "action", resource: "agent_memory", core: true }),
  ];

  it("splits the always-on tools from the ones an agent chooses", () => {
    const { core, selectable } = splitCoreTools(withCore);

    expect(core.map((t) => t.name)).toEqual(["agent_list_memories", "agent_create_memory"]);
    expect(selectable.map((t) => t.name)).toEqual(catalog.map((t) => t.name));
  });

  it("does not count a core tool an older agent still lists", () => {
    const summary = summarizeSelection(
      ["shipment_get", "agent_list_memories", "agent_create_memory"],
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
    tool({ name: "report_list", resource: "report" }),
    tool({ name: "report_delete", kind: "action", resource: "report" }),
    tool({
      name: "report_create_dashboard",
      kind: "action",
      resource: "report_dashboard",
      prerequisites: ["report_list", "report_delete", "missing_tool"],
    }),
    tool({
      name: "report_add_dashboard_tile",
      kind: "action",
      resource: "report_dashboard",
      prerequisites: ["report_list"],
    }),
  ];

  it("names each read a chosen tool depends on, with every tool that needs it", () => {
    const implied = impliedReads(["report_create_dashboard", "report_add_dashboard_tile"], withDependencies);

    expect(implied.map((entry) => entry.tool.name)).toEqual(["report_list"]);
    expect(implied[0].neededBy.map((entry) => entry.name)).toEqual([
      "report_create_dashboard",
      "report_add_dashboard_tile",
    ]);
  });

  it("does not repeat a read that was chosen outright", () => {
    expect(impliedReads(["report_create_dashboard", "report_list"], withDependencies)).toEqual([]);
  });
});

describe("extension tools", () => {
  const webSearch = tool({
    name: "web_search",
    resource: "web_research",
    extension: "Exa",
    grantedToEveryAgent: true,
  });
  const webRead = tool({ name: "web_read", resource: "web_research", extension: "Exa" });

  // An extension turned on for every agent gives its tools to all of them, so
  // a checkbox for one would change nothing.
  it("shows a tool given to every agent as always on", () => {
    const { core, selectable } = splitCoreTools([tool({}), webSearch, webRead]);

    expect(core.map((entry) => entry.name)).toEqual(["web_search"]);
    expect(selectable.map((entry) => entry.name)).toEqual(["shipment_get", "web_read"]);
  });

  it("does not count a tool every agent already holds as chosen", () => {
    const summary = summarizeSelection(
      ["shipment_get", "web_search", "web_read"],
      [tool({}), webSearch, webRead],
      {},
      "Propose",
    );

    expect(summary).toMatchObject({ reads: 2, changes: 0, unknown: [] });
  });
});
