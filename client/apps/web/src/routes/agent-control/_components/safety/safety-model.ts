import type {
  AgentAutonomyAnswer,
  AgentEgressClass,
  AgentExternalRead,
  AgentSafetyHeader,
  AgentToolKind,
  AgentToolPolicy,
} from "@/lib/graphql/agent-safety";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { resourceLabel } from "../agents/tool-catalog";

/** Every class in the order the organization is left: nowhere first, money last. */
export const EGRESS_ORDER: readonly AgentEgressClass[] = [
  "None",
  "Personal",
  "Internal",
  "CustomerVisible",
  "DriverVisible",
  "ExternalRecipient",
  "Money",
];

/** The widest audience among a tool's classes; a tool that names none reaches nobody. */
export function widestEgress(egress: readonly AgentEgressClass[]): AgentEgressClass {
  return egress.reduce<AgentEgressClass>(
    (widest, next) => (EGRESS_ORDER.indexOf(next) > EGRESS_ORDER.indexOf(widest) ? next : widest),
    "None",
  );
}

/** The hue and chroma each audience's chip is drawn in, from the accent hue tokens. */
export const EGRESS_HUE: Record<AgentEgressClass, { hue: string; chroma: number }> = {
  None: { hue: "var(--hue-slate)", chroma: 0.02 },
  Personal: { hue: "var(--hue-teal)", chroma: 0.09 },
  Internal: { hue: "var(--hue-indigo)", chroma: 0.11 },
  CustomerVisible: { hue: "var(--hue-sky)", chroma: 0.11 },
  DriverVisible: { hue: "var(--hue-emerald)", chroma: 0.11 },
  ExternalRecipient: { hue: "var(--hue-violet)", chroma: 0.13 },
  Money: { hue: "var(--hue-amber)", chroma: 0.13 },
};

export function egressLabel(t: TranslateFn, egress: AgentEgressClass): string {
  switch (egress) {
    case "None":
      return t("Reads only");
    case "Personal":
      return t("Own records");
    case "Internal":
      return t("Internal");
    case "CustomerVisible":
      return t("Customer");
    case "DriverVisible":
      return t("Driver");
    case "ExternalRecipient":
      return t("Outside recipient");
    case "Money":
      return t("Money");
  }
}

/** The chip each answer is drawn as. */
export const ANSWER_CLASS: Record<AgentAutonomyAnswer, "r" | "c" | "w" | "p" | "s"> = {
  RUNS_ON_ITS_OWN: "r",
  CONDITIONAL: "c",
  NEEDS_APPROVAL: "w",
  PROPOSE_ONLY: "p",
  SIMULATED: "s",
};

/** Every answer from most done without a person to least. */
export const ANSWER_ORDER: readonly AgentAutonomyAnswer[] = [
  "RUNS_ON_ITS_OWN",
  "CONDITIONAL",
  "NEEDS_APPROVAL",
  "PROPOSE_ONLY",
  "SIMULATED",
];

export function answerLabel(t: TranslateFn, answer: AgentAutonomyAnswer): string {
  switch (answer) {
    case "RUNS_ON_ITS_OWN":
      return t("Runs on its own");
    case "CONDITIONAL":
      return t("Depends on the call");
    case "NEEDS_APPROVAL":
      return t("Needs approval");
    case "PROPOSE_ONLY":
      return t("Proposes only");
    case "SIMULATED":
      return t("Simulated");
  }
}

/** Every tier from the least an agent may do on its own to the most. */
export const TIER_ORDER: readonly AgentToolPolicy["maxTier"][] = [
  "Propose",
  "ActWithApproval",
  "AutoExecute",
];

export function tierLabel(t: TranslateFn, tier: AgentToolPolicy["maxTier"]): string {
  switch (tier) {
    case "AutoExecute":
      return t("Automatic");
    case "ActWithApproval":
      return t("Ask first");
    case "Propose":
      return t("Propose");
  }
}

/** Every reason the server names for holding a call back, in the order it checks them. */
export const HELD_BY_KEYS = [
  "agent_ceiling",
  "tool_max",
  "egress_class",
  "condition",
  "tainted",
  "tool_tier",
  "personal_exemption",
  "shadow_mode",
  "simulation_mode",
] as const;

/** Every reason a held tool goes no further, before or after outside text, each once. */
export function heldByOf(tool: {
  clean: { heldBy: readonly string[] };
  tainted: { heldBy: readonly string[] };
}): string[] {
  return [...new Set([...tool.clean.heldBy, ...tool.tainted.heldBy])];
}

/** The reasons the server names for holding a call back, in words. */
export function heldByLabel(t: TranslateFn, key: string): string {
  switch (key) {
    case "agent_ceiling":
      return t("Agent ceiling");
    case "tool_max":
      return t("Tool maximum");
    case "egress_class":
      return t("Who sees it");
    case "condition":
      return t("Record condition");
    case "tainted":
      return t("Read outside text");
    case "tool_tier":
      return t("Tool tier on the agent");
    case "personal_exemption":
      return t("Own records, person present");
    case "shadow_mode":
      return t("Shadow mode");
    case "simulation_mode":
      return t("Simulation");
    default:
      return key;
  }
}

export function needsLabel(t: TranslateFn, policy: Pick<AgentToolPolicy, "needs">): string {
  if (!policy.needs) {
    return t("Nothing: own records only");
  }
  return t("{0} · {1}", resourceLabel(policy.needs.resource), policy.needs.operation);
}

export function readsOutsideLabel(
  t: TranslateFn,
  policy: Pick<AgentToolPolicy, "readsExternal" | "source" | "carriesTaint">,
): string | null {
  const source = policy.source ? sourceLabel(t, policy.source) : "";
  switch (policy.readsExternal) {
    case "Always":
      return source === "" ? t("Always") : t("Always, from {0}", source);
    case "Marked":
      return source === "" ? t("When marked") : t("When marked, from {0}", source);
    case "Never":
      return policy.carriesTaint ? t("Carries outside text forward") : null;
  }
}

function sourceLabel(t: TranslateFn, source: NonNullable<AgentToolPolicy["source"]>): string {
  switch (source) {
    case "InboundMessage":
      return t("inbound messages");
    case "Document":
      return t("documents");
    case "EDI":
      return t("EDI");
    case "BankReceipt":
      return t("bank receipts");
    case "Weather":
      return t("weather alerts");
    case "Attachment":
      return t("attachments");
    case "Memory":
      return t("memory");
    case "RunRecord":
      return t("run records");
    case "Web":
      return t("the web");
    case "RecordNote":
      return t("record notes");
  }
}

/** Every way a tool's result can carry outside text, from none to always. */
export const EXTERNAL_READ_ORDER: readonly AgentExternalRead[] = ["Never", "Marked", "Always"];

export function externalReadLabel(t: TranslateFn, read: AgentExternalRead): string {
  switch (read) {
    case "Never":
      return t("Never");
    case "Marked":
      return t("When marked");
    case "Always":
      return t("Always");
  }
}

/** Every kind in the order a person meets them: reads, then changes, then the runtime's own. */
export const KIND_ORDER: readonly AgentToolKind[] = ["Query", "Action", "Runtime"];

export function kindLabel(t: TranslateFn, kind: AgentToolKind): string {
  switch (kind) {
    case "Query":
      return t("Reads");
    case "Action":
      return t("Changes");
    case "Runtime":
      return t("Runtime");
  }
}

/** The resources the server names, alphabetical by what a person reads, not by key. */
export function sortResources(resources: readonly string[]): string[] {
  return [...resources].sort((a, b) => resourceLabel(a).localeCompare(resourceLabel(b)));
}

type FilterChoice = { value: string; label: string };

/** The options a table offers for filtering a column, in the column's own order. */
export function egressChoices(t: TranslateFn): FilterChoice[] {
  return EGRESS_ORDER.map((egress) => ({ value: egress, label: egressLabel(t, egress) }));
}

export function tierChoices(t: TranslateFn): FilterChoice[] {
  return TIER_ORDER.map((tier) => ({ value: tier, label: tierLabel(t, tier) }));
}

export function kindChoices(t: TranslateFn): FilterChoice[] {
  return KIND_ORDER.map((kind) => ({ value: kind, label: kindLabel(t, kind) }));
}

export function externalReadChoices(t: TranslateFn): FilterChoice[] {
  return EXTERNAL_READ_ORDER.map((read) => ({ value: read, label: externalReadLabel(t, read) }));
}

export function answerChoices(t: TranslateFn): FilterChoice[] {
  return ANSWER_ORDER.map((answer) => ({ value: answer, label: answerLabel(t, answer) }));
}

export function heldByChoices(t: TranslateFn): FilterChoice[] {
  return HELD_BY_KEYS.map((key) => ({ value: key, label: heldByLabel(t, key) }));
}

export function resourceChoices(resources: readonly string[]): FilterChoice[] {
  return sortResources(resources).map((resource) => ({
    value: resource,
    label: resourceLabel(resource),
  }));
}

export function reachLabel(t: TranslateFn, safety: AgentSafetyHeader): string {
  if (safety.reach.accessMode === "Everyone") {
    return t("Everyone who can use the assistant");
  }
  if (safety.reach.roles.length === 0) {
    return t("Restricted to roles");
  }
  return t("Roles: {0}", safety.reach.roles.map((role) => role.name).join(", "));
}

/** What Nova says about safety, and who it would have someone review. */
export type SafetyFacts = {
  runs: number;
  runningTools: readonly string[];
  leave: number;
  open: readonly string[];
};

export function safetyFacts(summary: {
  runWithoutPerson: number;
  unattendedTools: readonly string[];
  leaveOrganization: number;
  openSensitiveAgentIds: readonly string[];
}): SafetyFacts {
  return {
    runs: summary.runWithoutPerson,
    runningTools: summary.unattendedTools,
    leave: summary.leaveOrganization,
    open: summary.openSensitiveAgentIds,
  };
}

/** How many open agents Nova's review compares at once. */
export const REVIEWED_AT_ONCE = 3;

/** The segments of the map of who sees the work: every audience but reads, widest last. */
export function egressSegments(
  counts: readonly { egress: AgentEgressClass; count: number }[],
): { egress: AgentEgressClass; count: number; grow: number }[] {
  const shown = counts.filter((entry) => entry.egress !== "None" && entry.count > 0);
  const total = shown.reduce((sum, entry) => sum + entry.count, 0);

  return EGRESS_ORDER.flatMap((egress) => {
    const entry = shown.find((candidate) => candidate.egress === egress);
    return entry ? [{ ...entry, grow: Math.max(entry.count, total * 0.09) }] : [];
  });
}
