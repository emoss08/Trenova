import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { describe, expect, it } from "vitest";
import { filterAgents, groupAgentsByTrigger } from "../agent-roster";

function agent(overrides: Partial<AgentDefinitionRow>): AgentDefinitionRow {
  return {
    id: "agdef_1",
    name: "Dispatch desk",
    description: "",
    template: "DispatchAssistant",
    triggerMode: "Chat",
    enabled: true,
    ...overrides,
  } as AgentDefinitionRow;
}

describe("groupAgentsByTrigger", () => {
  it("shelves agents by what starts them, chat first, live before parked", () => {
    const shelves = groupAgentsByTrigger([
      agent({ id: "a", name: "Zed digest", triggerMode: "Scheduled", enabled: false }),
      agent({ id: "b", name: "Billing desk", triggerMode: "Chat" }),
      agent({ id: "c", name: "Alpha digest", triggerMode: "Scheduled" }),
      agent({ id: "d", name: "Dispatch desk", triggerMode: "Chat" }),
      agent({ id: "e", name: "Help", triggerMode: "Chat", enabled: false }),
    ]);

    expect(shelves.map((shelf) => shelf.trigger)).toEqual(["Chat", "Scheduled"]);
    expect(shelves[0].agents.map((a) => a.name)).toEqual(["Billing desk", "Dispatch desk", "Help"]);
    expect(shelves[1].agents.map((a) => a.name)).toEqual(["Alpha digest", "Zed digest"]);
  });

  it("is empty for no agents", () => {
    expect(groupAgentsByTrigger([])).toEqual([]);
  });
});

describe("filterAgents", () => {
  it("matches name, description or template, ignoring case", () => {
    const agents = [
      agent({ id: "a", name: "Dispatch desk" }),
      agent({
        id: "b",
        name: "Help",
        description: "Explains how to do things",
        template: "GeneralAssistant",
      }),
      agent({ id: "c", name: "Compliance", template: "ComplianceAssistant" }),
    ];

    expect(filterAgents(agents, "DISPATCH").map((a) => a.id)).toEqual(["a"]);
    expect(filterAgents(agents, "explains").map((a) => a.id)).toEqual(["b"]);
    expect(filterAgents(agents, "complianceassistant").map((a) => a.id)).toEqual(["c"]);
    expect(filterAgents(agents, "  ")).toHaveLength(3);
  });
});
