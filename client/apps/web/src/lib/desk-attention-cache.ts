import type { DeskAttention, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { deskAttention } from "@/lib/queries/desk-attention";
import type { QueryClient } from "@tanstack/react-query";

/**
 * Takes an item out of every cached Needs you list at once, dropping a row
 * when its last item goes, so the row leaves with the card rather than after
 * the next read.
 */
export function removeAttentionItem(queryClient: QueryClient, id: string): void {
  queryClient.setQueriesData<DeskAttention>({ queryKey: deskAttention._def }, (data) => {
    if (!data) {
      return data;
    }
    return {
      ...data,
      groups: data.groups
        .map((group) => ({ ...group, items: group.items.filter((item) => item.id !== id) }))
        .filter((group) => group.items.length > 0),
    };
  });
}

/** Rewrites one item wherever a cached Needs you list holds it. */
export function patchAttentionItem(
  queryClient: QueryClient,
  id: string,
  patch: (item: DeskAttentionItem) => DeskAttentionItem,
): void {
  queryClient.setQueriesData<DeskAttention>({ queryKey: deskAttention._def }, (data) => {
    if (!data) {
      return data;
    }
    return {
      ...data,
      groups: data.groups.map((group) => ({
        ...group,
        unseen: group.items.some((item) => !(item.id === id ? patch(item) : item).seen),
        items: group.items.map((item) => (item.id === id ? patch(item) : item)),
      })),
    };
  });
}
