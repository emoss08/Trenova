import type { AgentMatch } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { settleWithin } from "@trenova/shared/lib/async";
import {
  RECOMMENDATION_SETTLE_MS,
  RECOMMENDATION_STALE_MS,
  RECOMMENDATION_WAIT_MS,
  recommendationQueryText,
} from "./agent-recommendation-state";

export type AgentMatchState = {
  /** The match for the words as they last settled, or null while there is none. */
  match: AgentMatch | null;
  /**
   * The match for exactly these words, for a question being sent: read from
   * the cache when typing settled on them, otherwise asked for and waited on
   * for a moment at most. Null when there is none in time.
   */
  resolve: (text: string) => Promise<AgentMatch | null>;
};

/**
 * How well the person's agents fit a question, worked out on the server as
 * they type. It asks once typing pauses, keeps the last answer on screen while
 * the next is fetched so the suggestion does not flicker, and never retries:
 * a match that fails is simply not shown. With the box empty there is no
 * match, so a question just sent leaves nothing behind it.
 */
export function useAgentMatch({
  draft,
  agentId,
  contextKinds,
  enabled,
}: {
  draft: string;
  /** The agent the question would go to now; it is kept when it fits as well as any. */
  agentId: string | null;
  /** The kinds of record the question is about; tools for them rank higher. */
  contextKinds: readonly string[];
  enabled: boolean;
}): AgentMatchState {
  const queryClient = useQueryClient();
  const settled = useDebounce(draft, RECOMMENDATION_SETTLE_MS);
  const text = enabled ? recommendationQueryText(settled) : null;
  const query = useQuery({
    ...assistant.agentMatch(text ?? "", agentId, contextKinds),
    enabled: text !== null,
    staleTime: RECOMMENDATION_STALE_MS,
    placeholderData: keepPreviousData,
    retry: false,
  });

  const resolve = (content: string): Promise<AgentMatch | null> => {
    const wanted = enabled ? recommendationQueryText(content) : null;
    if (wanted === null) {
      return Promise.resolve(null);
    }

    return settleWithin(
      queryClient.fetchQuery({
        ...assistant.agentMatch(wanted, agentId, contextKinds),
        staleTime: RECOMMENDATION_STALE_MS,
        retry: false,
      }),
      RECOMMENDATION_WAIT_MS,
      null,
    );
  };

  return {
    match: text === null || draft.trim() === "" ? null : (query.data ?? null),
    resolve,
  };
}
