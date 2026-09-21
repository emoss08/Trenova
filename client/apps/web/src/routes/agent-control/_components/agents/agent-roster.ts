import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { TriggerMode } from "@/types/assistant";

export type AgentShelf = { trigger: TriggerMode; agents: AgentDefinitionRow[] };

/** The order the shelves read in: what people talk to first, then what runs on its own. */
export const TRIGGER_ORDER: readonly TriggerMode[] = ["Chat", "Scheduled", "Event", "Continuous"];

/**
 * Agents whose name, description or template contains the query, case
 * ignored. An empty query is every agent.
 */
export function filterAgents(
  agents: readonly AgentDefinitionRow[],
  query: string,
): AgentDefinitionRow[] {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return [...agents];
  }

  return agents.filter((agent) =>
    `${agent.name} ${agent.description} ${agent.template ?? ""}`.toLowerCase().includes(needle),
  );
}

/**
 * Shelves agents by what starts them, in a fixed order, leaving out empty
 * shelves. Within a shelf the enabled ones come first and then by name, so
 * what is live reads before what is parked.
 */
export function groupAgentsByTrigger(agents: readonly AgentDefinitionRow[]): AgentShelf[] {
  const byTrigger = new Map<TriggerMode, AgentDefinitionRow[]>();
  for (const agent of agents) {
    const shelf = byTrigger.get(agent.triggerMode);
    if (shelf) {
      shelf.push(agent);
    } else {
      byTrigger.set(agent.triggerMode, [agent]);
    }
  }

  return TRIGGER_ORDER.flatMap((trigger) => {
    const shelf = byTrigger.get(trigger);
    if (!shelf) {
      return [];
    }
    shelf.sort((a, b) => {
      if (a.enabled !== b.enabled) {
        return a.enabled ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });

    return [{ trigger, agents: shelf }];
  });
}
