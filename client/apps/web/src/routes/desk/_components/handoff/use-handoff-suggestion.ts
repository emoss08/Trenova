import type { ThreadHistory } from "@/components/assistant/thread-history";
import { threadHistoryQueryOptions } from "@/components/assistant/use-thread-history";
import { useDeskStore } from "@/stores/desk-store";
import { hashKey, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useMemo, useSyncExternalStore } from "react";
import { shallow } from "zustand/shallow";
import { NO_HANDOFF_SUGGESTION, handoffSuggestion } from "./handoff-state";

const NO_PAGES: ThreadHistory["pages"] = [];

/**
 * The suggestion as one value read from the two places it comes from: the
 * thread's saved history in the query cache, and the reply being written,
 * which the conversation leaves in the Desk store. Either changing wakes the
 * reader, but the value only changes when the agents it names do, so a reply
 * being saved, which moves the same answer from one place to the other, does
 * not.
 */
function suggestionSource(queryClient: QueryClient, threadId: string) {
  const { queryKey } = threadHistoryQueryOptions(threadId);
  const queryHash = hashKey(queryKey);
  let current: readonly string[] = NO_HANDOFF_SUGGESTION;

  return {
    subscribe(notify: () => void): () => void {
      const stopHistory = queryClient.getQueryCache().subscribe((event) => {
        if (event.query.queryHash === queryHash) {
          notify();
        }
      });
      const stopLive = useDeskStore.subscribe((state, previous) => {
        if (state.liveHandoff[threadId] !== previous.liveHandoff[threadId]) {
          notify();
        }
      });

      return () => {
        stopHistory();
        stopLive();
      };
    },
    read(): readonly string[] {
      const history = queryClient.getQueryData<ThreadHistory>(queryKey);
      const next = handoffSuggestion(
        history?.pages ?? NO_PAGES,
        useDeskStore.getState().liveHandoff[threadId] ?? null,
      );
      if (!shallow(next, current)) {
        current = next;
      }

      return current;
    },
  };
}

/**
 * The agents the conversation's agent last said hold what it could not do,
 * newest first from the reply being written and then the saved history. It
 * only watches: nothing is fetched, and the caller renders again only when the
 * agents named change.
 */
export function useHandoffSuggestion(threadId: string): readonly string[] {
  const queryClient = useQueryClient();
  const source = useMemo(() => suggestionSource(queryClient, threadId), [queryClient, threadId]);

  return useSyncExternalStore(source.subscribe, source.read, source.read);
}
