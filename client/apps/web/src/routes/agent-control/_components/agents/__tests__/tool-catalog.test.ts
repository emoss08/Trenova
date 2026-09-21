import type { ToolCatalogEntry } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  effectiveTier,
  groupToolsByResource,
  resourceLabel,
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
