import { describe, expect, it } from "vitest";
import {
  externalReadAllowed,
  movedBy,
  reasonMissing,
  ruleHasTier,
  tierAllowed,
  toToolRuleForm,
  toToolRuleInput,
} from "../tool-rule-form";
import { policy } from "./fixtures";

describe("tool rule form", () => {
  it("starts from the rule as it applies now, at the rule's version, with no reason", () => {
    expect(
      toToolRuleForm(
        policy({ maxTier: "ActWithApproval", readsExternal: "Marked", ruleVersion: 4 }),
      ),
    ).toEqual({ maxTier: "ActWithApproval", readsExternal: "Marked", reason: "", version: 4 });
  });

  it("sends the reason trimmed, and none at all when it is blank", () => {
    const values = { maxTier: "Propose", readsExternal: "Always", version: 1 } as const;

    expect(toToolRuleInput({ ...values, reason: "  Two wrong releases  " })).toEqual({
      maxTier: "Propose",
      readsExternal: "Always",
      reason: "Two wrong releases",
    });
    expect(toToolRuleInput({ ...values, reason: "   " }).reason).toBeNull();
  });

  it("offers only tiers at or below what the tool declares", () => {
    expect(tierAllowed("Propose", "ActWithApproval")).toBe(true);
    expect(tierAllowed("ActWithApproval", "ActWithApproval")).toBe(true);
    expect(tierAllowed("AutoExecute", "ActWithApproval")).toBe(false);
  });

  it("offers only treating as much or more as outside text", () => {
    expect(externalReadAllowed("Never", "Marked")).toBe(false);
    expect(externalReadAllowed("Marked", "Marked")).toBe(true);
    expect(externalReadAllowed("Always", "Marked")).toBe(true);
  });

  it("asks for a reason only when the most freedom moves", () => {
    const loaded = {
      maxTier: "AutoExecute",
      readsExternal: "Never",
      reason: "",
      version: 0,
    } as const;

    expect(reasonMissing({ ...loaded, maxTier: "Propose" }, loaded)).toBe(true);
    expect(reasonMissing({ ...loaded, maxTier: "Propose", reason: " why " }, loaded)).toBe(false);
    expect(reasonMissing({ ...loaded, readsExternal: "Always" }, loaded)).toBe(false);
  });

  it("gives a tier only to tools that change records", () => {
    expect(ruleHasTier(policy({ effect: "Change" }))).toBe(true);
    expect(ruleHasTier(policy({ effect: "Lookup", kind: "Query" }))).toBe(false);
  });

  it("names only the holders whose answer moves", () => {
    const moved = {
      agentId: "a",
      agentName: "A",
      before: "RUNS_ON_ITS_OWN",
      after: "NEEDS_APPROVAL",
    } as const;
    const same = {
      agentId: "b",
      agentName: "B",
      before: "PROPOSE_ONLY",
      after: "PROPOSE_ONLY",
    } as const;

    expect(movedBy([moved, same])).toEqual([moved]);
  });
});
