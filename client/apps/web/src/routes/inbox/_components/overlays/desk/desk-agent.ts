import type { InboundClassification } from "@/lib/graphql/inbox";
import type { AgentChoice } from "@/lib/graphql/agent-definition";

/** What the Desk hand-off reads of an agent to choose one. */
export type AgentLike = Pick<AgentChoice, "id" | "name" | "description" | "template" | "toolNames">;

/** The kind of work a message needs, which decides the kind of agent it goes to first. */
export type AgentRole = "dispatch" | "billing" | "service";

export function agentRoleFor(classification: InboundClassification | null): AgentRole {
  switch (classification) {
    case "Tender":
    case "RateConfirmation":
      return "dispatch";
    case "Invoice":
    case "DetentionDispute":
    case "FactorRemittance":
    case "ProofOfDelivery":
      return "billing";
    default:
      return "service";
  }
}

/**
 * Word stems that say what an agent is for. An organization names its own
 * agents, so the hand-off reads their names, descriptions, templates and
 * tools rather than any id: "Carrier desk" is dispatch-like and "AR team" is
 * billing-like without either being configured as such.
 */
const ROLE_STEMS: Record<AgentRole, readonly string[]> = {
  dispatch: ["dispatch", "load", "tender", "carrier", "freight", "shipment", "book", "capacity"],
  billing: ["bill", "invoic", "detention", "pay", "settle", "account", "receivable", "collect"],
  service: ["customer", "service", "support", "status", "eta", "repl", "track", "update"],
};

/** Where a stem is found, and how much it says about the agent. */
const WEIGHTS = { name: 4, template: 2, description: 2, tools: 1 } as const;

function words(text: string): string[] {
  return text
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .toLowerCase()
    .split(/[^a-z]+/)
    .filter((word) => word !== "");
}

function hits(text: string, stems: readonly string[]): number {
  return words(text).filter((word) => stems.some((stem) => word.startsWith(stem))).length;
}

function score(agent: AgentLike, stems: readonly string[]): number {
  return (
    hits(agent.name, stems) * WEIGHTS.name +
    hits(agent.template ?? "", stems) * WEIGHTS.template +
    hits(agent.description ?? "", stems) * WEIGHTS.description +
    hits((agent.toolNames ?? []).join(" "), stems) * WEIGHTS.tools
  );
}

/**
 * The agent a message is handed to first: the one whose name and work best
 * fit what the message needs, the one asked last when two fit as well, and
 * the one asked last (or the first listed) when none fits at all.
 */
export function pickAgentFor<T extends AgentLike>(
  agents: readonly T[],
  classification: InboundClassification | null,
  lastAgentId: string | null = null,
): T | null {
  if (agents.length === 0) {
    return null;
  }
  const stems = ROLE_STEMS[agentRoleFor(classification)];
  let best: T | null = null;
  let bestScore = 0;
  for (const agent of agents) {
    const value = score(agent, stems);
    if (value > bestScore || (value === bestScore && value > 0 && agent.id === lastAgentId)) {
      best = agent;
      bestScore = value;
    }
  }
  if (best !== null) {
    return best;
  }

  return agents.find((agent) => agent.id === lastAgentId) ?? agents[0];
}
