import { queries } from "@/lib/queries";
import type { AssistantArtifact } from "@/types/assistant";
import type { QueryClient } from "@tanstack/react-query";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { create } from "zustand";
import { entityCardFrom } from "./artifact-payloads";

/**
 * Which queue item each billing item artifact is showing. The artifact is one
 * card that shows whichever item was picked — from a queue table's row, or by
 * stepping up and down the queue — so the choice lives beside the artifact
 * rather than in its payload, and survives moving between artifacts.
 */
interface DeskBillingState {
  itemByLineage: Record<string, string>;
  select: (lineageId: string, itemId: string) => void;
}

export const useDeskBillingStore = create<DeskBillingState>()((set) => ({
  itemByLineage: {},
  select: (lineageId, itemId) =>
    set((state) =>
      state.itemByLineage[lineageId] === itemId
        ? state
        : { itemByLineage: { ...state.itemByLineage, [lineageId]: itemId } },
    ),
}));

/** The lineage the billing item artifact grows from: a looked-up queue item's card. */
export function isBillingCard(artifact: AssistantArtifact): boolean {
  return (
    artifact.kind === "entity_card" && entityCardFrom(artifact).entity === "billing_queue_item"
  );
}

/** The conversation's newest billing item artifact, if it has one, by lineage. */
export function billingCardLineage(queryClient: QueryClient, threadId: string): string | null {
  const list = queryClient.getQueryData<{ results: AssistantArtifact[] }>(
    queries.assistant.artifacts(threadId).queryKey,
  );
  const cards = (list?.results ?? []).filter(isBillingCard);
  if (cards.length === 0) return null;
  const newest = cards.reduce((a, b) => (b.createdAt > a.createdAt ? b : a));

  return newest.lineageId || newest.id;
}

/**
 * The slug a billing item artifact's link carries: the item it is showing,
 * so a copied link follows the selection rather than the item it was opened on.
 */
export function useBillingItemSlug(
  artifact: AssistantArtifact,
  queryClient: QueryClient,
): string | null {
  const lineageId = artifact.lineageId || artifact.id;
  const selected = useDeskBillingStore((state) => state.itemByLineage[lineageId]);
  if (!isBillingCard(artifact)) return null;
  const itemId = selected ?? entityCardFrom(artifact).recordId;
  const item = queryClient.getQueryData<BillingQueueItem>(
    queries.billingQueue.get(itemId).queryKey,
  );

  return (item?.number || itemId).toLowerCase();
}
