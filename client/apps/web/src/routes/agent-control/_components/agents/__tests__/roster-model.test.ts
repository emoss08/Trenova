import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { ToolCatalogEntry } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  agentMode,
  filterRoster,
  leadingStreak,
  modeFlags,
  rosterCounts,
  rowFact,
  tierSplit,
} from "../roster-model";

function agent(overrides: Partial<AgentDefinitionRow>): AgentDefinitionRow {
  return {
    id: "agdef_1",
    name: "Dispatch desk",
    description: "Looks up shipments and drivers",
    triggerMode: "Chat",
    enabled: true,
    shadowMode: false,
    simulationMode: false,
    pendingProposals: 0,
    eventKinds: [],
    accessMode: "Everyone",
    accessRoles: [],
    nextRunAt: null,
    toolNames: [],
    toolTiers: {},
    autonomyCeiling: "AutoExecute",
    ...overrides,
  } as AgentDefinitionRow;
}

function tool(name: string, kind: "query" | "action", core = false): ToolCatalogEntry {
  return {
    name,
    description: "",
    kind,
    resource: "shipment",
    operation: kind === "query" ? "read" : "update",
    defaultAutonomyTier: "",
    reversible: false,
    core,
    prerequisites: [],
    extension: "",
    grantedToEveryAgent: false,
  } as ToolCatalogEntry;
}

describe("rosterCounts and filterRoster", () => {
  const agents = [
    agent({ id: "a", name: "Billing exceptions", pendingProposals: 3 }),
    agent({ id: "b", name: "Coverage", shadowMode: true, pendingProposals: 2 }),
    agent({ id: "c", name: "Digest", enabled: false, pendingProposals: 5 }),
    agent({ id: "d", name: "Help", description: "Explains how to do things" }),
  ];

  it("counts a parked agent's waiting proposals as off, not waiting", () => {
    expect(rosterCounts(agents)).toEqual({ all: 4, waiting: 2, shadow: 1, off: 1 });
  });

  it("narrows by filter, then by name or description ignoring case", () => {
    expect(filterRoster(agents, "waiting", "").map((a) => a.id)).toEqual(["a", "b"]);
    expect(filterRoster(agents, "off", "").map((a) => a.id)).toEqual(["c"]);
    expect(filterRoster(agents, "all", "EXPLAINS").map((a) => a.id)).toEqual(["d"]);
    expect(filterRoster(agents, "shadow", "billing")).toEqual([]);
  });

  it("is everything for the all filter and a blank search", () => {
    expect(filterRoster(agents, "all", "   ")).toHaveLength(4);
    expect(rosterCounts([])).toEqual({ all: 0, waiting: 0, shadow: 0, off: 0 });
  });
});

describe("tierSplit", () => {
  const catalog = [
    tool("search_shipments", "query"),
    tool("get_shipment", "query"),
    tool("recall_memory", "query", true),
    tool("assign_driver", "action"),
    tool("tender_to_carrier", "action"),
    tool("release_hold", "action"),
  ];

  it("splits reads, proposals and acts, capping each change at the ceiling", () => {
    const split = tierSplit(
      agent({
        toolNames: [
          "search_shipments",
          "get_shipment",
          "recall_memory",
          "assign_driver",
          "tender_to_carrier",
          "release_hold",
          "retired_tool",
        ],
        toolTiers: {
          assign_driver: "AutoExecute",
          tender_to_carrier: "Propose",
          release_hold: "ActWithApproval",
        },
        autonomyCeiling: "ActWithApproval",
      }),
      catalog,
    );
    expect(split).toEqual([2, 1, 2]);
  });

  it("puts every change at the propose tier under a propose ceiling", () => {
    expect(
      tierSplit(
        agent({
          toolNames: ["assign_driver", "release_hold"],
          toolTiers: { assign_driver: "AutoExecute" },
          autonomyCeiling: "Propose",
        }),
        catalog,
      ),
    ).toEqual([0, 2, 0]);
  });
});

describe("rowFact", () => {
  const label = (kind: string) => `label:${kind}`;

  it("says when a scheduled or continuous agent runs next, and nothing before it is planned", () => {
    expect(rowFact(agent({ triggerMode: "Scheduled", nextRunAt: 1_800_000_000 }), label)).toEqual({
      kind: "schedule",
      nextRunAt: 1_800_000_000,
    });
    expect(rowFact(agent({ triggerMode: "Continuous", nextRunAt: 5 }), label)).toEqual({
      kind: "schedule",
      nextRunAt: 5,
    });
    expect(rowFact(agent({ triggerMode: "Scheduled", nextRunAt: null }), label)).toBeNull();
  });

  it("names the first event and how many more wake it", () => {
    expect(
      rowFact(agent({ triggerMode: "Event", eventKinds: ["BillingHoldPlaced", "RateMismatch", "X"] }), label),
    ).toEqual({ kind: "event", label: "label:BillingHoldPlaced", more: 2 });
    expect(rowFact(agent({ triggerMode: "Event", eventKinds: [] }), label)).toBeNull();
  });

  it("names the roles of a chat agent limited to roles, and nothing for one open to everyone", () => {
    expect(
      rowFact(
        agent({
          accessMode: "Roles",
          accessRoles: [
            { id: "r1", name: "Dispatch lead" },
            { id: "r2", name: "Owner" },
          ] as AgentDefinitionRow["accessRoles"],
        }),
        label,
      ),
    ).toEqual({ kind: "roles", names: ["Dispatch lead", "Owner"] });
    expect(rowFact(agent({ accessMode: "Roles", accessRoles: [] }), label)).toBeNull();
    expect(rowFact(agent({}), label)).toBeNull();
  });
});

describe("agentMode", () => {
  it("reads simulation over shadow, and round-trips through its flags", () => {
    expect(agentMode(agent({ shadowMode: true, simulationMode: true }))).toBe("sim");
    expect(agentMode(agent({ shadowMode: true }))).toBe("shadow");
    expect(agentMode(agent({}))).toBe("live");
    for (const mode of ["live", "shadow", "sim"] as const) {
      expect(agentMode(modeFlags(mode))).toBe(mode);
    }
  });
});

describe("leadingStreak", () => {
  it("takes the longest streak among tools still held, the first by name on a tie", () => {
    expect(
      leadingStreak(
        [
          { toolName: "release_hold", streak: 7 },
          { toolName: "assign_driver", streak: 7 },
          { toolName: "dropped_tool", streak: 40 },
          { toolName: "tender", streak: 2 },
        ],
        ["release_hold", "assign_driver", "tender"],
      ),
    ).toEqual({ toolName: "assign_driver", streak: 7 });
  });

  it("is nothing before any held tool has a clean approval", () => {
    expect(leadingStreak([{ toolName: "assign_driver", streak: 0 }], ["assign_driver"])).toBeNull();
    expect(leadingStreak([], ["assign_driver"])).toBeNull();
  });
});
