import type {
  AgentAutonomyAnswer,
  AgentEgressClass,
  AgentSafety,
  AgentToolAutonomy,
  AgentToolPolicy,
} from "@/lib/graphql/agent-safety";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";
import type { BadgeAccent, BadgeAppearance } from "@trenova/shared/types/badge";
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

/**
 * Who sees a tool's work is a category, not a severity: a note inside the
 * organization is not "less wrong" than a message to a customer, it is a
 * different audience. Each class keeps its own accent.
 */
export const EGRESS_ACCENT: Record<AgentEgressClass, BadgeAccent> = {
  None: "accent-slate",
  Personal: "accent-teal",
  Internal: "accent-indigo",
  CustomerVisible: "accent-sky",
  DriverVisible: "accent-emerald",
  ExternalRecipient: "accent-violet",
  Money: "accent-amber",
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

/**
 * How much a person stands between the agent and the change. Running on its
 * own is under way without anyone, approval waits on a person, a proposal has
 * not started, and a simulated write never happens at all.
 */
export const ANSWER_BADGE: Record<
  AgentAutonomyAnswer,
  { phase: StatusPhase; appearance: BadgeAppearance }
> = {
  RUNS_ON_ITS_OWN: { phase: "active", appearance: "solid" },
  CONDITIONAL: { phase: "active", appearance: "subtle" },
  NEEDS_APPROVAL: { phase: "awaiting", appearance: "subtle" },
  PROPOSE_ONLY: { phase: "draft", appearance: "subtle" },
  SIMULATED: { phase: "closed", appearance: "outline" },
};

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

export function needsLabel(t: TranslateFn, policy: AgentToolPolicy): string {
  if (!policy.needs) {
    return t("Nothing: own records only");
  }
  return t("{0} · {1}", resourceLabel(policy.needs.resource), policy.needs.operation);
}

export function readsOutsideLabel(t: TranslateFn, policy: AgentToolPolicy): string | null {
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
  }
}

export type PolicyFilter = {
  egress: AgentEgressClass | "all";
  resource: string | "all";
};

export const ALL_POLICIES: PolicyFilter = { egress: "all", resource: "all" };

/** A tool that needs no grant is filed under "general", as the agent form files it. */
export function policyResource(policy: AgentToolPolicy): string {
  return policy.needs?.resource ?? "general";
}

export function filterPolicies(
  policies: readonly AgentToolPolicy[],
  filter: PolicyFilter,
): AgentToolPolicy[] {
  return policies.filter(
    (policy) =>
      (filter.egress === "all" || policy.egress.includes(filter.egress)) &&
      (filter.resource === "all" || policyResource(policy) === filter.resource),
  );
}

export function resourceOptions(policies: readonly AgentToolPolicy[]): string[] {
  return [...new Set(policies.map(policyResource))].sort((a, b) =>
    resourceLabel(a).localeCompare(resourceLabel(b)),
  );
}

export function answerWithoutAPerson(autonomy: AgentToolAutonomy): boolean {
  return autonomy.answer === "RUNS_ON_ITS_OWN" || autonomy.answer === "CONDITIONAL";
}

export type SafetyFigures = {
  /** Tools that change something and, on at least one agent, can do so without a person. */
  runWithoutPerson: number;
  /** Tools some of whose work leaves the organization. */
  leaveOrganization: number;
  /** Agents open to everyone that hold a tool reaching sensitive data or leaving. */
  openWithSensitive: number;
};

export function safetyFigures(
  policies: readonly AgentToolPolicy[],
  agents: readonly AgentSafety[],
): SafetyFigures {
  const changes = new Set(
    policies.filter((policy) => policy.effect === "Change").map((policy) => policy.name),
  );
  const unattended = new Set<string>();
  for (const safety of agents) {
    for (const tool of safety.tools) {
      if (changes.has(tool.policyName) && answerWithoutAPerson(tool.clean)) {
        unattended.add(tool.policyName);
      }
    }
  }

  return {
    runWithoutPerson: unattended.size,
    leaveOrganization: policies.filter((policy) => policy.leavesOrganization).length,
    openWithSensitive: agents.filter((safety) =>
      safety.reach.warnings.some((warning) => warning.kind === "OpenWithSensitiveTools"),
    ).length,
  };
}

/** The agents shown in the matrix: the ones picked, or the first agent until someone picks. */
export function selectedAgents(
  agents: readonly AgentSafety[],
  picked: readonly string[],
): AgentSafety[] {
  if (picked.length === 0) {
    return agents.slice(0, 1);
  }
  const wanted = new Set(picked);
  return agents.filter((safety) => wanted.has(safety.agentId));
}

/** Tools that act without a person first, so the rows worth reading lead. */
export function sortToolsByExposure<T extends { clean: AgentToolAutonomy; policyName: string }>(
  tools: readonly T[],
): T[] {
  const rank: Record<AgentAutonomyAnswer, number> = {
    RUNS_ON_ITS_OWN: 0,
    CONDITIONAL: 1,
    NEEDS_APPROVAL: 2,
    PROPOSE_ONLY: 3,
    SIMULATED: 4,
  };
  return [...tools].sort(
    (a, b) =>
      rank[a.clean.answer] - rank[b.clean.answer] || a.policyName.localeCompare(b.policyName),
  );
}

export function reachLabel(t: TranslateFn, safety: AgentSafety): string {
  if (safety.reach.accessMode === "Everyone") {
    return t("Everyone who can use the assistant");
  }
  if (safety.reach.roles.length === 0) {
    return t("Restricted to roles");
  }
  return t("Roles: {0}", safety.reach.roles.map((role) => role.name).join(", "));
}
