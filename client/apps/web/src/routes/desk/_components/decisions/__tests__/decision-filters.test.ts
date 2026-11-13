import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  NO_DECISION_FACETS,
  NO_DECISION_FILTERS,
  changeLabel,
  decisionAddressOf,
  decisionFilterCount,
  decisionFilterTokens,
  decisionFiltersFromAddress,
  isDecisionNarrowed,
  pendingFilter,
  type DecisionFacets,
} from "../decision-filters";

const t = ((text: string) => text) as unknown as TranslateFn;

const FACETS: DecisionFacets = {
  agents: [
    { id: "agd_billing", name: "Billing clerk", count: 5 },
    { id: "agd_dispatch", name: "Dispatch planner", count: 2 },
  ],
  tools: [
    { tool: "reassign_move", count: 4 },
    { tool: "plan", count: 3 },
  ],
};

describe("the filters the Decisions lens keeps in its address", () => {
  it("reads one agent and one kind of change, and nothing when the address names neither", () => {
    expect(
      decisionFiltersFromAddress({ q: " acme ", agent: " agd_billing ", change: "plan" }),
    ).toEqual({
      search: "acme",
      agentId: "agd_billing",
      tool: "plan",
    });
    expect(decisionFiltersFromAddress({ q: "", agent: "", change: "" })).toEqual(
      NO_DECISION_FILTERS,
    );
  });

  it("leaves a filter that is off out of the address and out of what the queue is asked", () => {
    expect(decisionAddressOf({ search: "", agentId: "", tool: "plan" })).toEqual({
      q: null,
      agent: null,
      change: "plan",
    });
    expect(pendingFilter({ search: "", agentId: "", tool: "plan" })).toEqual({
      agentDefinitionId: undefined,
      toolName: "plan",
      search: undefined,
    });
    expect(pendingFilter({ search: "acme", agentId: "agd_billing", tool: "" })).toEqual({
      agentDefinitionId: "agd_billing",
      toolName: undefined,
      search: "acme",
    });
  });

  it("counts the filters that are on", () => {
    expect(decisionFilterCount(NO_DECISION_FILTERS)).toBe(0);
    expect(decisionFilterCount({ search: "", agentId: "agd_billing", tool: "" })).toBe(1);
    expect(decisionFilterCount({ search: "", agentId: "agd_billing", tool: "plan" })).toBe(2);
  });

  it("does not count the search as a filter, and knows the queue is narrowed by either", () => {
    expect(decisionFilterCount({ search: "acme", agentId: "", tool: "" })).toBe(0);
    expect(isDecisionNarrowed({ search: "acme", agentId: "", tool: "" })).toBe(true);
    expect(isDecisionNarrowed({ search: "", agentId: "", tool: "plan" })).toBe(true);
    expect(isDecisionNarrowed(NO_DECISION_FILTERS)).toBe(false);
  });

  it("names a plan as Plans and a write by what it does", () => {
    expect(changeLabel(t, "plan")).toBe("Plans");
    expect(changeLabel(t, "reassign_move")).not.toContain("_");
  });

  it("draws a token for the agent by its name and for the change in words", () => {
    expect(
      decisionFilterTokens(t, { search: "", agentId: "agd_dispatch", tool: "plan" }, FACETS).map(
        (token) => [token.type, token.label],
      ),
    ).toEqual([
      ["agent", "Dispatch planner"],
      ["change", "Plans"],
    ]);
  });

  it("never shows an agent's id when the agent has nothing waiting any more", () => {
    expect(
      decisionFilterTokens(t, { search: "", agentId: "agd_gone", tool: "" }, NO_DECISION_FACETS),
    ).toEqual([{ key: "agent", type: "agent", label: "One agent" }]);
  });
});
