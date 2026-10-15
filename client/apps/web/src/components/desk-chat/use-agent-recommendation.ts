import type { AgentRecommendation } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { settleWithin } from "@trenova/shared/lib/async";
import { useCallback } from "react";
import {
  RECOMMENDATION_SETTLE_MS,
  RECOMMENDATION_STALE_MS,
  RECOMMENDATION_WAIT_MS,
  recommendationQueryText,
} from "./agent-recommendation-state";

export type AgentRecommendationState = {
  /** The match for the words as they last settled, or null while there is none. */
  recommendation: AgentRecommendation | null;
  /**
   * The match for exactly these words, for a question being sent: read from
   * the cache when typing settled on them, otherwise asked for and waited on
   * for a moment at most. Null when there is none in time.
   */
  resolve: (text: string) => Promise<AgentRecommendation | null>;
};

/**
 * Which of the person's agents holds the tools a question needs, worked out
 * on the server as they type. It asks once typing pauses, keeps the last
 * answer on screen while the next is fetched so the suggestion does not
 * flicker, and never retries: a suggestion that fails is simply not made.
 */
export function useAgentRecommendation({
  draft,
  agentId,
  enabled,
}: {
  draft: string;
  /** The agent the question would go to now; it is kept when it fits as well as any. */
  agentId: string | null;
  enabled: boolean;
}): AgentRecommendationState {
  const queryClient = useQueryClient();
  const settled = useDebounce(draft, RECOMMENDATION_SETTLE_MS);
  const text = enabled ? recommendationQueryText(settled) : null;
  const query = useQuery({
    ...assistant.agentRecommendation(text ?? "", agentId),
    enabled: text !== null,
    staleTime: RECOMMENDATION_STALE_MS,
    placeholderData: keepPreviousData,
    retry: false,
  });

  const resolve = useCallback(
    (content: string): Promise<AgentRecommendation | null> => {
      const wanted = enabled ? recommendationQueryText(content) : null;
      if (wanted === null) {
        return Promise.resolve(null);
      }

      return settleWithin(
        queryClient.fetchQuery({
          ...assistant.agentRecommendation(wanted, agentId),
          staleTime: RECOMMENDATION_STALE_MS,
          retry: false,
        }),
        RECOMMENDATION_WAIT_MS,
        null,
      );
    },
    [agentId, enabled, queryClient],
  );

  return {
    recommendation: text === null ? null : (query.data ?? null),
    resolve,
  };
}
