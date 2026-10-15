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
    expect(offersRecommendation(recommending(billing), dispatch.id, null)).toBe(true);
  });

  it("offers nothing when the agent chosen is already the best fit", () => {
    expect(offersRecommendation(recommending(dispatch, true), dispatch.id, null)).toBe(false);
    expect(offersRecommendation(recommending(billing), billing.id, null)).toBe(false);
  });

  it("offers nothing when there is no match", () => {
    expect(offersRecommendation(null, dispatch.id, null)).toBe(false);
  });

  it("does not offer again an agent the person kept theirs over", () => {
    expect(offersRecommendation(recommending(billing), dispatch.id, billing.id)).toBe(false);
  });
});

describe("agentToAsk", () => {
  it("asks the agent with the tools until the person has chosen", () => {
    expect(
      agentToAsk({ agent: dispatch, recommendation: recommending(billing), chosen: false }),
    ).toBe(billing);
  });

  it("asks the agent the person chose, whatever fits better", () => {
    expect(
      agentToAsk({ agent: dispatch, recommendation: recommending(billing), chosen: true }),
    ).toBe(dispatch);
  });

  it("asks the agent already chosen when nothing fits better", () => {
    expect(agentToAsk({ agent: dispatch, recommendation: null, chosen: false })).toBe(dispatch);
    expect(
      agentToAsk({ agent: dispatch, recommendation: recommending(dispatch, true), chosen: false }),
    ).toBe(dispatch);
  });
});
