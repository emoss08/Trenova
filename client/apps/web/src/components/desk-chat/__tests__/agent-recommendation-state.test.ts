import type { AgentChoice, AgentRecommendation } from "@/lib/graphql/agent-definition";
import { describe, expect, it } from "vitest";
import {
  RECOMMENDATION_MIN_LENGTH,
  agentToAsk,
  offersRecommendation,
  recommendationQueryText,
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

function recommending(agent: AgentChoice, current = false): AgentRecommendation {
  return { agent, current, tools: ["list_invoices"] };
}

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

describe("offersRecommendation", () => {
  it("offers an agent other than the one the question would go to", () => {
    expect(offersRecommendation(recommending(billing), dispatch.id)).toBe(true);
  });

  it("offers nothing when the agent chosen is already the best fit", () => {
    expect(offersRecommendation(recommending(dispatch, true), dispatch.id)).toBe(false);
    expect(offersRecommendation(recommending(billing), billing.id)).toBe(false);
  });

  it("does not offer again an agent the person turned down, but offers another", () => {
    expect(offersRecommendation(recommending(billing), dispatch.id, [billing.id])).toBe(false);
    expect(offersRecommendation(recommending(billing), dispatch.id, ["agdef_other"])).toBe(true);
  });

  it("offers nothing when there is no match", () => {
    expect(offersRecommendation(null, dispatch.id)).toBe(false);
  });
});

describe("agentToAsk", () => {
  it("asks the agent with the tools unless the person picked one", () => {
    expect(
      agentToAsk({
        agent: dispatch,
        recommendation: recommending(billing),
        pinned: false,
        keptOver: [],
      }),
    ).toBe(billing);
  });

  it("asks the agent picked from the picker, whatever fits better", () => {
    expect(
      agentToAsk({
        agent: dispatch,
        recommendation: recommending(billing),
        pinned: true,
        keptOver: [],
      }),
    ).toBe(dispatch);
  });

  it("asks the agent already chosen over one the person turned down", () => {
    expect(
      agentToAsk({
        agent: dispatch,
        recommendation: recommending(billing),
        pinned: false,
        keptOver: [billing.id],
      }),
    ).toBe(dispatch);
  });

  it("asks the agent already chosen when nothing fits better", () => {
    expect(agentToAsk({ agent: dispatch, recommendation: null, pinned: false, keptOver: [] })).toBe(
      dispatch,
    );
    expect(
      agentToAsk({
        agent: dispatch,
        recommendation: recommending(dispatch, true),
        pinned: false,
        keptOver: [],
      }),
    ).toBe(dispatch);
  });
});
