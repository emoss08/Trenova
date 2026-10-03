import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantHandoff } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  HANDOFF_MENU_CLOSED,
  carriedChips,
  handoffMenuReducer,
  handoffTargets,
} from "../handoff-state";

function agent(id: string, delegates: string[] = []): AgentChoice {
  return {
    id,
    name: id,
    description: "",
    template: null,
    icon: "",
    accent: "",
    toolNames: [],
    systemKey: "",
    starters: [],
    delegates: delegates.map((delegate) => ({
      id: delegate,
      name: delegate,
      icon: "",
      accent: "",
      template: null,
    })),
  };
}

describe("handoffTargets", () => {
  it("leaves out the conversation's own agent and puts its delegates first", () => {
    const billing = agent("billing", ["cash", "dispatch"]);
    const agents = [agent("general"), billing, agent("dispatch"), agent("cash")];

    expect(handoffTargets(agents, billing).map((target) => target.id)).toEqual([
      "cash",
      "dispatch",
      "general",
    ]);
  });

  it("offers everyone when the conversation's agent is unknown", () => {
    expect(handoffTargets([agent("a"), agent("b")], null).map((target) => target.id)).toEqual([
      "a",
      "b",
    ]);
  });
});

describe("handoffMenuReducer", () => {
  it("opens, closes and toggles", () => {
    const open = handoffMenuReducer(HANDOFF_MENU_CLOSED, { type: "toggle" });
    expect(open.open).toBe(true);
    expect(handoffMenuReducer(open, { type: "toggle" }).open).toBe(false);
    expect(handoffMenuReducer(open, { type: "close" }).open).toBe(false);
  });

  it("closes on a pick and ignores a second pick until the first settles", () => {
    const open = handoffMenuReducer(HANDOFF_MENU_CLOSED, { type: "toggle" });
    const picked = handoffMenuReducer(open, { type: "pick", agentId: "cash" });
    expect(picked).toEqual({ open: false, pendingAgentId: "cash" });

    expect(handoffMenuReducer(picked, { type: "pick", agentId: "dispatch" })).toBe(picked);
    expect(handoffMenuReducer(picked, { type: "toggle" })).toBe(picked);

    const settled = handoffMenuReducer(picked, { type: "settled" });
    expect(settled.pendingAgentId).toBeNull();
    expect(handoffMenuReducer(settled, { type: "toggle" }).open).toBe(true);
  });
});

describe("carriedChips", () => {
  const base: AssistantHandoff = {
    fromThreadId: "t1",
    toThreadId: "t2",
    fromAgentId: "a",
    fromAgentName: "A",
    toAgentId: "b",
    toAgentName: "B",
    summary: "The person is chasing two invoices.",
    facts: ["Invoice date is Oct 3", "Net 30"],
    artifacts: [{ id: "art1", sourceId: "art0", title: "Billing queue table", kind: "table_view" }],
    at: 0,
  };

  it("names the summary, the facts and each artifact", () => {
    expect(carriedChips(base)).toEqual([
      { kind: "summary" },
      { kind: "facts", count: 2 },
      { kind: "artifact", title: "Billing queue table" },
    ]);
  });

  it("leaves out what was not carried", () => {
    expect(carriedChips({ ...base, summary: " ", facts: [], artifacts: [] })).toEqual([]);
  });
});
