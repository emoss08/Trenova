import { PREVIEW_STALE_TIME } from "@/components/assistant/proposal-preview/use-proposal-preview";
import { agentPreview } from "@/lib/queries/agent-preview";
import { useQueries } from "@tanstack/react-query";
import { useState } from "react";
import { MAX_BATCH, type BatchPreviewEntry } from "./batch-review-model";

/** What each waiting change would do, as far as it has been read. */
export type DecisionPreviews = {
  entry: (id: string) => BatchPreviewEntry | undefined;
  /** The change's preview is still being read for the first time. */
  reading: (id: string) => boolean;
  /** The read the change's preview was part of failed. */
  failed: (id: string) => boolean;
  /** Reads the change's preview again. */
  refetch: (id: string) => void;
};

function chunked(ids: readonly string[]): string[][] {
  const out: string[][] = [];
  for (let start = 0; start < ids.length; start += MAX_BATCH) {
    out.push(ids.slice(start, start + MAX_BATCH));
  }

  return out;
}

/**
 * Reads the previews of the changes the queue holds, a batch at a time as
 * they arrive. A change stays in the batch it was first read with, so the
 * queue shrinking or growing never changes what a batch already read asks
 * for, and only new arrivals are read. A batch none of whose changes still
 * wait is let go.
 */
export function useDecisionPreviews(ids: readonly string[]): DecisionPreviews {
  const [batches, setBatches] = useState<string[][]>([]);
  const waiting = new Set(ids);
  const known = new Set(batches.flat());
  const fresh = ids.filter((id) => !known.has(id));
  const live = batches.filter((batch) => batch.some((id) => waiting.has(id)));
  if (fresh.length > 0 || live.length !== batches.length) {
    setBatches([...live, ...chunked(fresh)]);
  }

  const results = useQueries({
    queries: batches.map((batch) => ({
      ...agentPreview.batch(batch),
      staleTime: PREVIEW_STALE_TIME,
    })),
  });

  const entries = new Map<string, BatchPreviewEntry>();
  const batchOf = new Map<string, number>();
  batches.forEach((batch, index) => {
    for (const id of batch) {
      batchOf.set(id, index);
    }
    for (const entry of results[index]?.data ?? []) {
      entries.set(entry.proposalId, entry);
    }
  });
  const result = (id: string) => {
    const index = batchOf.get(id);
    return index === undefined ? undefined : results[index];
  };

  return {
    entry: (id) => entries.get(id),
    reading: (id) => result(id)?.isPending ?? true,
    failed: (id) => result(id)?.isError ?? false,
    refetch: (id) => void result(id)?.refetch(),
  };
}
