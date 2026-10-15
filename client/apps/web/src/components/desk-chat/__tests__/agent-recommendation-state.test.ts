import type { AgentChoice, AgentMatch } from "@/lib/graphql/agent-definition";
import { describe, expect, it } from "vitest";
import {
  RECOMMENDATION_MIN_LENGTH,
  agentToAsk,
  matchContextKinds,
  noAgentFits,
  offersAgent,
  recommendationQueryText,
  routesOnSend,
  showsSuggestion,
  suggestionMode,
} from "../agent-recommendation-state";

function agentNamed(id: string, name: string): AgentChoice {
  return {
    id,
    name,
    description: "",
    template: null,
    icon: "",
    accent: "",
    toolNames: [],
    systemKey: "",
    starters: [],
  } as unknown as AgentChoice;
}

const dispatch = agentNamed("agdef_dispatch", "Dispatch");
const billing = agentNamed("agdef_billing", "Billing assistant");

function recommending(agent: AgentChoice, current = false): AgentMatch {
  return {
    outcome: current ? "CurrentAgent" : "BetterAgent",
    agent,
    tools: [{ name: "list_invoices", summary: "List invoices by status." }],
  };
}

const nobody: AgentMatch = {
  outcome: "NoAgent",
  agent: null,
  tools: [{ name: "list_invoices", summary: "List invoices by status." }],
};

describe("recommendationQueryText", () => {
  it("matches the question as typed, without the space around it", () => {
    expect(recommendationQueryText("  which invoices are past due?  ")).toBe(
      "which invoices are past due?",
    );
  });

  it("matches nothing shorter than a task can be named in", () => {
    const short = "x".repeat(RECOMMENDATION_MIN_LENGTH - 1);

    expect(recommendationQueryText(short)).toBeNull();
    expect(recommendationQueryText(`  ${short}  `)).toBeNull();
    expect(recommendationQueryText("x".repeat(RECOMMENDATION_MIN_LENGTH))).toBe(
      "x".repeat(RECOMMENDATION_MIN_LENGTH),
    );
  });

  it("leaves a slash command to the composer", () => {
    expect(recommendationQueryText("/explain this page")).toBeNull();
  });

  it("matches on the first 4000 characters of a long question, the most the server takes", () => {
    const text = recommendationQueryText("a".repeat(5000));

    expect(text).toHaveLength(4000);
  });
});

describe("offersAgent", () => {
  it("offers an agent other than the one the question would go to", () => {
    expect(offersAgent(recommending(billing), dispatch.id)).toBe(true);
  });

  it("offers nothing when the agent chosen is already the best fit", () => {
    expect(offersAgent(recommending(dispatch, true), dispatch.id)).toBe(false);
    expect(offersAgent(recommending(billing), billing.id)).toBe(false);
  });

  it("does not offer again an agent the person turned down, but offers another", () => {
    expect(offersAgent(recommending(billing), dispatch.id, [billing.id])).toBe(false);
    expect(offersAgent(recommending(billing), dispatch.id, ["agdef_other"])).toBe(true);
  });

  it("offers nothing when no agent fits or nothing matched", () => {
    expect(offersAgent(null, dispatch.id)).toBe(false);
    expect(offersAgent(nobody, dispatch.id)).toBe(false);
    expect(offersAgent({ outcome: "NoMatch", agent: null, tools: [] }, dispatch.id)).toBe(false);
  });
});

describe("noAgentFits", () => {
  it("is true only when tools match and none of the person's agents holds them", () => {
    expect(noAgentFits(nobody)).toBe(true);
    expect(noAgentFits(recommending(billing))).toBe(false);
    expect(noAgentFits(null)).toBe(false);
  });
});

describe("suggestionMode", () => {
  it("moves questions without asking whenever auto switch is on", () => {
    expect(suggestionMode({ autoSwitchAgent: "on", suggestAgent: "off" })).toBe("auto");
    expect(suggestionMode({ autoSwitchAgent: "on", suggestAgent: "suggest" })).toBe("auto");
  });

  it("follows the suggestion setting while auto switch is off", () => {
    expect(suggestionMode({ autoSwitchAgent: "off", suggestAgent: "switch" })).toBe("switch");
    expect(suggestionMode({ autoSwitchAgent: "off", suggestAgent: "suggest" })).toBe("suggest");
    expect(suggestionMode({ autoSwitchAgent: "off", suggestAgent: "off" })).toBe("off");
  });

  it("draws the note only where the person is asked, and routes only where they let it", () => {
    expect(
      ["auto", "switch", "suggest", "off"].map((mode) => showsSuggestion(mode as never)),
    ).toEqual([false, true, true, false]);
    expect(["auto", "switch", "suggest", "off"].map((mode) => routesOnSend(mode as never))).toEqual(
      [true, true, false, false],
    );
  });
});

describe("matchContextKinds", () => {
  it("names each kind of record mentioned, then the page's record and list, once each", () => {
    expect(
      matchContextKinds([{ type: "shipment" }, { type: "customer" }, { type: "shipment" }], {
        entityType: "invoice",
        view: { resource: "shipment" },
      }),
    ).toEqual(["shipment", "customer", "invoice"]);
  });

  it("names nothing without mentions or a page", () => {
    expect(matchContextKinds([], null)).toEqual([]);
    expect(matchContextKinds([], { entityType: "", view: null })).toEqual([]);
  });

  it("names at most as many kinds as the server takes", () => {
    const mentions = Array.from({ length: 15 }, (_, index) => ({ type: `kind${index}` }));

    expect(matchContextKinds(mentions, null)).toHaveLength(11);
  });
});

describe("agentToAsk", () => {
  it("asks the agent with the tools unless the person picked one", () => {
    expect(
      agentToAsk({
        agent: dispatch,
        match: recommending(billing),
        pinned: false,
        keptOver: [],
        mode: "switch",
      }),
    ).toBe(billing);
  });

  it("moves the question without asking when the person lets it", () => {
    expect(
      agentToAsk({
        agent: dispatch,
        match: recommending(billing),
        pinned: false,
        keptOver: [],
        mode: "auto",
      }),
    ).toBe(billing);
  });

  it("asks the agent chosen when the person only wants to be told", () => {
    for (const mode of ["suggest", "off"] as const) {
      expect(
        agentToAsk({
          agent: dispatch,
          match: recommending(billing),
          pinned: false,
          keptOver: [],
          mode,
        }),
      ).toBe(dispatch);
    }
  });

  it("asks the agent picked, whatever fits better", () => {
    expect(
      agentToAsk({
        agent: dispatch,
        match: recommending(billing),
        pinned: true,
        keptOver: [],
        mode: "auto",
      }),
    ).toBe(dispatch);
  });

  it("asks the agent already chosen over one the person turned down", () => {
    expect(
      agentToAsk({
        agent: dispatch,
        match: recommending(billing),
        pinned: false,
        keptOver: [billing.id],
        mode: "switch",
      }),
    ).toBe(dispatch);
  });

  it("asks the agent already chosen when nothing fits better", () => {
    for (const match of [null, recommending(dispatch, true), nobody]) {
      expect(
        agentToAsk({ agent: dispatch, match, pinned: false, keptOver: [], mode: "switch" }),
      ).toBe(dispatch);
    }
  });
});
