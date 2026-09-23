import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { useRealtimeStore } from "@/stores/realtime-store";
import type { AssistantLiveTurnList } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback } from "react";
import { liveReplyCount, liveThreadIds } from "./active-turns";

/**
 * How often the list is read while the realtime connection is down. With the
 * connection up, a reply starting or closing arrives as an event and nothing
 * is polled; without it, this is the only way the markers ever clear.
 */
export const ACTIVE_TURNS_FALLBACK_POLL_MS = 30_000;

type Select<T> = (list: AssistantLiveTurnList) => T;

/**
 * The replies this person's conversations are still writing, wherever they
 * were asked. One query behind every marker, so the launcher, the lists and
 * the Desk never disagree about what is under way.
 */
function useActiveTurnsQuery<T>(select: Select<T>) {
  const { allowed } = usePermission(Resource.Assistant, Operation.Read);
  const connected = useRealtimeStore((state) => state.connectionState === "connected");

  return useQuery({
    ...queries.assistant.activeTurns(),
    enabled: allowed,
    select,
    refetchInterval: connected ? false : ACTIVE_TURNS_FALLBACK_POLL_MS,
    refetchIntervalInBackground: false,
  });
}

/** The conversations with a reply being written; empty until known. */
export function useLiveThreadIds(): ReadonlySet<string> {
  const { data } = useActiveTurnsQuery(liveThreadIds);

  return data ?? liveThreadIds(undefined);
}

/** How many replies are being written across every conversation. */
export function useLiveReplyCount(): number {
  const { data } = useActiveTurnsQuery(liveReplyCount);

  return data ?? 0;
}

/**
 * Marks the list stale after this tab starts, stops or finishes a reply, so
 * the markers move with the person's own action rather than a round-trip
 * later. The realtime event does the same for every other tab and device.
 */
export function useInvalidateActiveTurns(): () => Promise<void> {
  const queryClient = useQueryClient();

  return useCallback(
    () => queryClient.invalidateQueries({ queryKey: queries.assistant.activeTurns().queryKey }),
    [queryClient],
  );
}
