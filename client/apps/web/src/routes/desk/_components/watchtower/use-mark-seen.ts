import { patchAttentionItem } from "@/lib/desk-attention-cache";
import { markWatchtowerItemSeen } from "@/lib/graphql/watchtower";
import { patchQueueItem } from "@/lib/watchtower-cache";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

/**
 * Marks the item in front seen for this reader. The dot goes at once, in
 * every cached list that holds the item; a mark that does not reach the
 * server puts it back.
 *
 * With `defer`, the mark waits until the reader moves off the item. A list of
 * only the unopened needs that: marked on opening, the item would leave at the
 * next read while it is being read, the one after it would come to the front
 * and be marked in turn, and the list would empty itself.
 */
export function useMarkSeen(current: { id: string; seen: boolean } | null, defer = false): void {
  const queryClient = useQueryClient();
  const currentId = current?.id ?? null;
  const currentSeen = current?.seen ?? true;
  const awaiting = useRef<string | null>(null);

  useEffect(() => {
    const mark = (id: string) => {
      patchQueueItem(queryClient, id, (item) => ({ ...item, seen: true }));
      patchAttentionItem(queryClient, id, (item) => ({ ...item, seen: true }));
      markWatchtowerItemSeen(id).catch(() => {
        patchQueueItem(queryClient, id, (item) => ({ ...item, seen: false }));
        patchAttentionItem(queryClient, id, (item) => ({ ...item, seen: false }));
      });
    };
    const left = awaiting.current;
    if (left !== null && left !== currentId) {
      awaiting.current = null;
      mark(left);
    }
    if (currentId === null || currentSeen) {
      return;
    }
    if (defer) {
      awaiting.current = currentId;
      return;
    }
    awaiting.current = null;
    mark(currentId);
  }, [currentId, currentSeen, defer, queryClient]);
}
