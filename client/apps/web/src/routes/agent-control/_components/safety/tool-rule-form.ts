import type {
  AgentExternalRead,
  AgentToolPolicy,
  AgentToolRuleImpact,
  AgentToolRuleInput,
} from "@/lib/graphql/agent-safety";
import { z } from "zod";
import { EXTERNAL_READ_ORDER, TIER_ORDER } from "./safety-model";

type Tier = AgentToolPolicy["maxTier"];

/** The longest reason the server keeps with a rule change. */
export const MAX_TOOL_RULE_REASON = 500;

export const toolRuleFormSchema = z.object({
  maxTier: z.enum(["Propose", "ActWithApproval", "AutoExecute"]),
  readsExternal: z.enum(["Never", "Marked", "Always"]),
  reason: z.string().max(MAX_TOOL_RULE_REASON),
  version: z.number().int().min(0),
});

export type ToolRuleFormValues = z.infer<typeof toolRuleFormSchema>;

/** The tool's rule as it applies now, ready to edit; the reason is always written fresh. */
export function toToolRuleForm(policy: AgentToolPolicy): ToolRuleFormValues {
  return {
    maxTier: policy.maxTier,
    readsExternal: policy.readsExternal,
    reason: "",
    version: policy.ruleVersion,
  };
}

/**
 * What the editor sends. Choosing what the tool declares returns it to the declared
 * rule, which the server stores as no override at all.
 */
export function toToolRuleInput(values: ToolRuleFormValues): AgentToolRuleInput {
  const reason = values.reason.trim();

  return {
    maxTier: values.maxTier,
    readsExternal: values.readsExternal,
    reason: reason === "" ? null : reason,
  };
}

/** Whether the tool changes records, so its most freedom is a choice at all. */
export function ruleHasTier(policy: Pick<AgentToolPolicy, "effect">): boolean {
  return policy.effect === "Change";
}

/** A rule here can only hold a tool lower than what it declares. */
export function tierAllowed(tier: Tier, declared: Tier): boolean {
  return TIER_ORDER.indexOf(tier) <= TIER_ORDER.indexOf(declared);
}

/** A rule here can only treat more of what a tool returns as outside text. */
export function externalReadAllowed(read: AgentExternalRead, declared: AgentExternalRead): boolean {
  return EXTERNAL_READ_ORDER.indexOf(read) >= EXTERNAL_READ_ORDER.indexOf(declared);
}

/** The most freedom changed without a reason, which the audit trail needs. */
export function reasonMissing(values: ToolRuleFormValues, loaded: ToolRuleFormValues): boolean {
  return values.maxTier !== loaded.maxTier && values.reason.trim() === "";
}

/** The holders whose answer the change moves. */
export function movedBy(impacts: readonly AgentToolRuleImpact[]): AgentToolRuleImpact[] {
  return impacts.filter((impact) => impact.before !== impact.after);
}
