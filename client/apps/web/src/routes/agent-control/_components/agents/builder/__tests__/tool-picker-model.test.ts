import type { AgentScope } from "@/lib/agent-scope";
import type { ToolCatalogEntry } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  checkState,
  pickerGroups,
  pickerRows,
  setHeld,
  type PickerFilter,
} from "../tool-picker-model";

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
  tool({ name: "shipment_list" }),
  tool({ name: "shipment_cancel", kind: "action", description: "Cancels a shipment" }),
  tool({ name: "worker_list", resource: "worker", description: "Lists drivers" }),
  tool({ name: "update_worker", kind: "action", resource: "worker" }),
  tool({ name: "worker_approve_pto", kind: "action", resource: "worker_pto" }),
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
      "shipment_get",
      "shipment_list",
      "worker_list",
    ]);

    const changes = pickerGroups(catalog, [], { ...ALL, kind: "act" });
    expect(changes.flatMap((group) => names(group.tools))).toEqual([
      "shipment_cancel",
      "update_worker",
      "worker_approve_pto",
    ]);
  });

  it("shows only chosen tools when asked, across groups", () => {
    const groups = pickerGroups(catalog, ["worker_list", "shipment_cancel"], {
      ...ALL,
      chosenOnly: true,
    });
    expect(groups.map((group) => group.resource)).toEqual(["shipment", "worker"]);
    expect(groups.flatMap((group) => names(group.tools))).toEqual([
      "shipment_cancel",
      "worker_list",
    ]);
  });

  it("narrows to one group and combines with the search", () => {
    const groups = pickerGroups(catalog, [], { ...ALL, group: "worker", query: "update" });
    expect(groups.map((group) => group.resource)).toEqual(["worker"]);
    expect(groups.flatMap((group) => names(group.tools))).toEqual(["update_worker"]);
  });

  it("keeps the rail's chosen count over the whole group however it is narrowed", () => {
    const groups = pickerGroups(catalog, ["shipment_get", "shipment_cancel"], {
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
    const groups = pickerGroups(catalog, ["worker_list"], ALL);
    const rows = pickerRows(groups, new Set(["worker_list"]));
    expect(rows.map((row) => (row.kind === "group" ? `#${row.resource}` : row.tool.name))).toEqual([
      "#shipment",
      "shipment_get",
      "shipment_list",
      "shipment_cancel",
      "#worker",
      "worker_list",
      "update_worker",
      "#worker_pto",
      "worker_approve_pto",
    ]);
    const headers = rows.filter((row) => row.kind === "group");
    expect(headers.map((row) => row.state)).toEqual(["none", "some", "none"]);
    expect(new Set(rows.map((row) => row.key)).size).toBe(rows.length);
  });

  it("states a header over the shown tools only", () => {
    const chosen = new Set(["worker_list"]);
    const groups = pickerGroups(catalog, [...chosen], { ...ALL, kind: "read" });
    const worker = pickerRows(groups, chosen).find(
      (row) => row.kind === "group" && row.resource === "worker",
    );
    expect(worker?.kind === "group" && worker.state).toBe("all");
  });
});

describe("setHeld", () => {
  const scope = (over: Partial<AgentScope> = {}): AgentScope => ({
    areas: [],
    capabilities: [],
    denyAreas: [],
    denyCapabilities: [],
    ...over,
  });

  it("names the tools a narrowed agent is given, once each, in order", () => {
    const next = setHeld(
      scope({ capabilities: ["update_worker"] }),
      [
        byName("shipment_get"),
        byName("shipment_cancel"),
        byName("update_worker"),
        byName("shipment_get"),
      ],
      true,
    );
    expect(next.capabilities).toEqual(["update_worker", "shipment_get", "shipment_cancel"]);
    expect(next.denyCapabilities).toEqual([]);
  });

  it("names nothing an area already gives, and lifts a switch-off instead", () => {
    const next = setHeld(
      scope({ areas: ["shipment"], denyCapabilities: ["shipment_cancel"] }),
      [byName("shipment_get"), byName("shipment_cancel")],
      true,
    );
    expect(next).toEqual(scope({ areas: ["shipment"] }));
  });

  it("drops a named tool and switches off one an area or everything still gives", () => {
    expect(
      setHeld(
        scope({ areas: ["shipment"], capabilities: ["update_worker"] }),
        [byName("update_worker"), byName("shipment_cancel")],
        false,
      ),
    ).toEqual(scope({ areas: ["shipment"], denyCapabilities: ["shipment_cancel"] }));
    expect(setHeld(scope(), [byName("shipment_cancel")], false)).toEqual(
      scope({ denyCapabilities: ["shipment_cancel"] }),
    );
  });

  it("treats a scope being narrowed that names nothing as holding nothing", () => {
    expect(setHeld(scope(), [byName("shipment_get")], true, true)).toEqual(
      scope({ capabilities: ["shipment_get"] }),
    );
    expect(
      setHeld(scope({ capabilities: ["shipment_get"] }), [byName("shipment_get")], false, true),
    ).toEqual(scope());
  });

  it("clearing what a filter shows leaves the hidden tools untouched", () => {
    const held = ["shipment_get", "shipment_cancel", "worker_list"];
    const shown = pickerGroups(catalog, held, { ...ALL, kind: "read" }).flatMap(
      (group) => group.tools,
    );
    expect(setHeld(scope({ capabilities: held }), shown, false)).toEqual(
      scope({ capabilities: ["shipment_cancel"] }),
    );
  });
});
