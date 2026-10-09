import {
  initialTurnState,
  reduceTurn,
  turnHandOffAgents,
  type TurnHandoff,
} from "@/components/assistant/turn-stream";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import {
  parseAssistantStreamEvent,
  type AssistantHandoff,
  type AssistantStreamEvent,
} from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  HANDOFF_MENU_CLOSED,
  NO_HANDOFF_SUGGESTION,
  carriedChips,
  handoffMenuReducer,
  handoffSuggestion,
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

  it("puts the agents that hold what the conversation needs first, then delegates, then the rest", () => {
    const shipments = agent("shipments", ["dispatch", "cash"]);
    const agents = [
      agent("general"),
      shipments,
      agent("dispatch"),
      agent("cash"),
      agent("billing"),
      agent("exceptions"),
    ];

    expect(
      handoffTargets(agents, shipments, ["billing", "cash"]).map((target) => target.id),
    ).toEqual(["billing", "cash", "dispatch", "general", "exceptions"]);
  });

  it("ignores a suggestion naming the conversation's own agent or an agent the person cannot use", () => {
    const billing = agent("billing", ["cash"]);
    const agents = [agent("general"), billing, agent("cash")];

    expect(
      handoffTargets(agents, billing, ["billing", "retired"]).map((target) => target.id),
    ).toEqual(["cash", "general"]);
  });
});

describe("handoffSuggestion", () => {
  type Message = Parameters<typeof handoffSuggestion>[0][number]["results"][number];

  function user(): Message {
    return { role: "User", kind: "Message", handOffAgents: null };
  }

  function reply(): Message {
    return { role: "Assistant", kind: "Message", handOffAgents: null };
  }

  function found(ids: string[] | null, kind: Message["kind"] = "Message"): Message {
    return { role: "Tool", kind, handOffAgents: ids };
  }

  it("reads the newest find_tools result that named agents since the person last wrote", () => {
    const pages = [
      { results: [user(), found(["cash"]), found(["billing", "cash"]), reply()] },
    ];

    expect(handoffSuggestion(pages)).toEqual(["billing", "cash"]);
  });

  it("drops the suggestion once the person writes again", () => {
    const pages = [{ results: [user(), found(["billing"]), reply(), user()] }];

    expect(handoffSuggestion(pages)).toBe(NO_HANDOFF_SUGGESTION);
  });

  it("drops the suggestion when the person steered the reply after it", () => {
    const steer: Message = { role: "User", kind: "Steer", handOffAgents: null };
    const pages = [{ results: [user(), found(["billing"]), steer, reply()] }];

    expect(handoffSuggestion(pages)).toBe(NO_HANDOFF_SUGGESTION);
  });

  it("looks into older pages when the newest holds nothing", () => {
    const pages = [
      { results: [reply(), found(null), reply()] },
      { results: [user(), found(["billing"])] },
    ];

    expect(handoffSuggestion(pages)).toEqual(["billing"]);
  });

  it("passes over a step another agent took on a task it was handed", () => {
    const pages = [{ results: [user(), found(["shipments"], "Delegated"), reply()] }];

    expect(handoffSuggestion(pages)).toBe(NO_HANDOFF_SUGGESTION);
  });

  it("suggests nobody in an empty thread or one with no named agents", () => {
    expect(handoffSuggestion([])).toBe(NO_HANDOFF_SUGGESTION);
    expect(handoffSuggestion([{ results: [user(), found([]), reply()] }])).toBe(
      NO_HANDOFF_SUGGESTION,
    );
  });

  describe("with a reply being written", () => {
    const saved = [{ results: [user(), found(["billing"]), reply()] }];

    function wire(event: string, data: unknown): AssistantStreamEvent {
      const parsed = parseAssistantStreamEvent(event, JSON.stringify(data));
      if (parsed === null) {
        throw new Error(`the client does not know ${event}`);
      }
      return parsed;
    }

    /** The live turn as the Desk reports it, from the events the server sends. */
    function liveTurn(events: AssistantStreamEvent[], followUp = false): TurnHandoff {
      const turn = events.reduce(
        reduceTurn,
        initialTurnState(followUp ? "" : "Mark it ready to invoice", null, { followUp }),
      );
      return { asked: !turn.followUp, agentIds: turnHandOffAgents(turn.segments) };
    }

    const namedCash = [
      wire("accepted", { content: "Mark it ready to invoice" }),
      wire("tool_started", { callId: "c1", name: "find_tools", arguments: {} }),
      wire("tool_finished", {
        callId: "c1",
        name: "find_tools",
        content: "{}",
        handOffAgents: ["cash", "billing"],
      }),
      wire("delta", { text: "The cash assistant can do that." }),
    ];

    it("offers what the reply's own find_tools named before anything is saved", () => {
      const live = liveTurn(namedCash);

      expect(handoffSuggestion([], live)).toEqual(["cash", "billing"]);
      expect(handoffSuggestion(saved, live)).toBe(live.agentIds);
    });

    it("drops the saved suggestion while a reply to the person's new words names nobody", () => {
      const live = liveTurn([
        wire("accepted", { content: "Where is load 1042?" }),
        wire("delta", { text: "Checking." }),
      ]);

      expect(handoffSuggestion(saved, live)).toBe(NO_HANDOFF_SUGGESTION);
    });

    it("leaves the saved suggestion standing under a follow-up that names nobody", () => {
      const live = liveTurn([wire("delta", { text: "The decision went through." })], true);

      expect(handoffSuggestion(saved, live)).toEqual(["billing"]);
    });

    it("reads the saved history when no reply is being written", () => {
      expect(handoffSuggestion(saved, null)).toEqual(["billing"]);
    });

    it("passes over a find_tools answer another agent gave on a task it was handed", () => {
      const live = liveTurn([
        wire("accepted", { content: "Build it" }),
        wire("tool_started", {
          callId: "call_d",
          name: "delegate_task",
          arguments: { agentId: "agdef_other" },
        }),
        wire("tool_finished", {
          callId: "c9",
          name: "find_tools",
          content: "{}",
          handOffAgents: ["cash"],
          agentId: "agdef_other",
          delegateCallId: "call_d",
        }),
      ]);

      expect(handoffSuggestion(saved, live)).toBe(NO_HANDOFF_SUGGESTION);
    });
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
