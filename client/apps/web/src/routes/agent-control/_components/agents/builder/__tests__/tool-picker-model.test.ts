import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  checkState,
  pickerGroups,
  pickerRows,
  setTools,
  type PickerFilter,
} from "../tool-picker-model";

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
    extension: "",
    grantedToEveryAgent: false,
    ...overrides,
  };
}

const catalog = [
  tool({ name: "get_shipment" }),
  tool({ name: "list_shipments" }),
  tool({ name: "cancel_shipment", kind: "action", description: "Cancels a shipment" }),
  tool({ name: "list_workers", resource: "worker", description: "Lists drivers" }),
  tool({ name: "update_worker", kind: "action", resource: "worker" }),
  tool({ name: "approve_worker_pto", kind: "action", resource: "worker_pto" }),
];

const byName = (name: string) => catalog.find((entry) => entry.name === name)!;
const ALL: PickerFilter = { query: "", group: "all", kind: "all", chosenOnly: false };
const names = (tools: readonly ToolCatalogEntry[]) => tools.map((entry) => entry.name);

describe("pickerGroups", () => {
  it("keeps every group and tool with no filter", () => {
    const groups = pickerGroups(catalog, [], ALL);
    expect(groups.map((group) => group.resource)).toEqual(["shipment", "worker", "worker_pto"]);
    expect(groups.flatMap((group) => names(group.tools))).toHaveLength(catalog.length);
  });

  it("narrows to reads or changes and drops the groups it empties", () => {
    const reads = pickerGroups(catalog, [], { ...ALL, kind: "read" });
    expect(reads.map((group) => group.resource)).toEqual(["shipment", "worker"]);
    expect(reads.flatMap((group) => names(group.tools))).toEqual([
      "get_shipment",
      "list_shipments",
      "list_workers",
    ]);

    const changes = pickerGroups(catalog, [], { ...ALL, kind: "act" });
    expect(changes.flatMap((group) => names(group.tools))).toEqual([
      "cancel_shipment",
      "update_worker",
      "approve_worker_pto",
    ]);
  });

  it("shows only chosen tools when asked, across groups", () => {
    const groups = pickerGroups(catalog, ["list_workers", "cancel_shipment"], {
      ...ALL,
      chosenOnly: true,
    });
    expect(groups.map((group) => group.resource)).toEqual(["shipment", "worker"]);
    expect(groups.flatMap((group) => names(group.tools))).toEqual([
      "cancel_shipment",
      "list_workers",
    ]);
  });

  it("narrows to one group and combines with the search", () => {
    const groups = pickerGroups(catalog, [], { ...ALL, group: "worker", query: "update" });
    expect(groups.map((group) => group.resource)).toEqual(["worker"]);
    expect(groups.flatMap((group) => names(group.tools))).toEqual(["update_worker"]);
  });

  it("keeps the rail's chosen count over the whole group however it is narrowed", () => {
    const groups = pickerGroups(catalog, ["get_shipment", "cancel_shipment"], {
      ...ALL,
      kind: "read",
    });
    expect(groups.find((group) => group.resource === "shipment")?.chosen).toBe(2);
  });
});

describe("checkState", () => {
  it("is none, some or all of the names", () => {
    const chosen = new Set(["a", "b"]);
    expect(checkState(["c", "d"], chosen)).toBe("none");
    expect(checkState(["a", "c"], chosen)).toBe("some");
    expect(checkState(["a", "b"], chosen)).toBe("all");
    expect(checkState([], chosen)).toBe("none");
  });
});

describe("pickerRows", () => {
  it("flattens each group into its header and then its tools", () => {
    const groups = pickerGroups(catalog, ["list_workers"], ALL);
    const rows = pickerRows(groups, new Set(["list_workers"]));
    expect(rows.map((row) => (row.kind === "group" ? `#${row.resource}` : row.tool.name))).toEqual([
      "#shipment",
      "get_shipment",
      "list_shipments",
      "cancel_shipment",
      "#worker",
      "list_workers",
      "update_worker",
      "#worker_pto",
      "approve_worker_pto",
    ]);
    const headers = rows.filter((row) => row.kind === "group");
    expect(headers.map((row) => row.state)).toEqual(["none", "some", "none"]);
    expect(new Set(rows.map((row) => row.key)).size).toBe(rows.length);
  });

  it("states a header over the shown tools only", () => {
    const chosen = new Set(["list_workers"]);
    const groups = pickerGroups(catalog, [...chosen], { ...ALL, kind: "read" });
    const worker = pickerRows(groups, chosen).find(
      (row) => row.kind === "group" && row.resource === "worker",
    );
    expect(worker?.kind === "group" && worker.state).toBe("all");
  });
});

describe("setTools", () => {
  const tierFor = (entry: ToolCatalogEntry): AutonomyTier =>
    entry.name === "cancel_shipment" ? "Propose" : "ActWithApproval";

  it("adds the missing tools in order and gives only the new changes a tier", () => {
    const next = setTools(
      ["update_worker"],
      { update_worker: "AutoExecute" },
      [byName("get_shipment"), byName("cancel_shipment"), byName("update_worker")],
      true,
      tierFor,
    );
    expect(next.selected).toEqual(["update_worker", "get_shipment", "cancel_shipment"]);
    expect(next.tiers).toEqual({ update_worker: "AutoExecute", cancel_shipment: "Propose" });
  });

  it("removes the given tools and their tiers, leaving every other choice alone", () => {
    const next = setTools(
      ["get_shipment", "cancel_shipment", "update_worker", "approve_worker_pto"],
      { cancel_shipment: "Propose", update_worker: "AutoExecute", approve_worker_pto: "Propose" },
      [byName("cancel_shipment"), byName("update_worker"), byName("list_workers")],
      false,
      tierFor,
    );
    expect(next.selected).toEqual(["get_shipment", "approve_worker_pto"]);
    expect(next.tiers).toEqual({ approve_worker_pto: "Propose" });
  });

  it("clearing what a filter shows leaves the hidden chosen tools untouched", () => {
    const selected = ["get_shipment", "cancel_shipment", "list_workers"];
    const shown = pickerGroups(catalog, selected, { ...ALL, kind: "read" }).flatMap(
      (group) => group.tools,
    );
    const next = setTools(selected, { cancel_shipment: "Propose" }, shown, false, tierFor);
    expect(next.selected).toEqual(["cancel_shipment"]);
    expect(next.tiers).toEqual({ cancel_shipment: "Propose" });
  });

  it("does not repeat a tool named twice", () => {
    const next = setTools([], {}, [byName("get_shipment"), byName("get_shipment")], true, tierFor);
    expect(next.selected).toEqual(["get_shipment"]);
  });
});
