import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { agentRecency, type AgentRecency } from "@/lib/recent-agents";
import { useAssistantStore } from "@/stores/assistant-store";
import { useMemo, useState } from "react";
import { useAgentChoices, type AgentChoices } from "./use-agent-choices";

/** How many of the person's own agents lead the lists. */
export const RECENT_AGENT_LIMIT = 5;

type ThreadActivity = Parameters<typeof agentRecency>[0][number];

export type AskableAgent = {
  /** Who a question goes to now: the one chosen, else the one last asked, else the first. */
  agent: AgentChoice | null;
  choose: (agent: AgentChoice) => void;
  recency: AgentRecency;
  /** The unfiltered first page, shared with the picker and the directory through the cache. */
  choices: AgentChoices;
  /** Settled, and there is nobody to ask. */
  noneAvailable: boolean;
};

/**
 * The agent a surface is about to ask, before anyone has chosen one.
 *
 * It starts on the agent the person last asked, because most questions go
 * to the same one, and falls back to the first agent they can ask. Choosing
 * another — from the picker or anywhere else on the surface — holds until
 * the surface goes away.
 */
export function useAskableAgent({ threads }: { threads: readonly ThreadActivity[] }): AskableAgent {
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const recency = useMemo(
    () => agentRecency(threads, { limit: RECENT_AGENT_LIMIT, preferId: lastAgentId }),
    [lastAgentId, threads],
  );
  const choices = useAgentChoices({
    search: "",
    origin: "all",
    recentIds: recency.ids,
  });
  const [chosen, setChosen] = useState<AgentChoice | null>(null);

  return {
    agent: chosen ?? choices.recent[0] ?? choices.items[0] ?? null,
    choose: setChosen,
    recency,
    choices,
    noneAvailable:
      !choices.isLoading &&
      !choices.isError &&
      choices.recent.length === 0 &&
      choices.items.length === 0,
  };
}
