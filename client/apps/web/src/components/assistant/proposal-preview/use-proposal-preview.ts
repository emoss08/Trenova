import type { PreviewScope } from "@/lib/graphql/agent-preview";
import { queries } from "@/lib/queries";
import { keepPreviousData, useQuery, type QueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { approvalGate, isPreviewConflict, type ApprovalGate } from "./preview-gate";

/**
 * How long a preview is trusted before a remount reads it again. Short: it is
 * the world as it is now, and the digest it carries has to match the server's
 * at the moment of approval. Realtime and every decision invalidate it sooner.
 */
export const PREVIEW_STALE_TIME = 15_000;

export function useProposalPreview({
  scope,
  id,
  modifications = null,
  enabled = true,
  keepPrevious = false,
}: {
  scope: PreviewScope;
  id: string;
  modifications?: Record<string, unknown> | null;
  enabled?: boolean;
  /**
   * Keeps the last preview on screen while the next draft's is read, so an
   * editor does not flash empty between keystrokes. The caller must not
   * approve against a placeholder: `isPlaceholderData` says when it is one.
   */
  keepPrevious?: boolean;
}) {
  return useQuery({
    ...queries.agentPreview.proposal(scope, id, modifications),
    enabled: enabled && id !== "",
    staleTime: PREVIEW_STALE_TIME,
    placeholderData: keepPrevious ? keepPreviousData : undefined,
  });
}

export function usePlanPreview({
  scope,
  id,
  enabled = true,
}: {
  scope: PreviewScope;
  id: string;
  enabled?: boolean;
}) {
  return useQuery({
    ...queries.agentPreview.plan(scope, id),
    enabled: enabled && id !== "",
    staleTime: PREVIEW_STALE_TIME,
  });
}

/** Warms the cache for a row the person is about to read. */
export function prefetchProposalPreview(client: QueryClient, scope: PreviewScope, id: string) {
  return client.prefetchQuery({
    ...queries.agentPreview.proposal(scope, id),
    staleTime: PREVIEW_STALE_TIME,
  });
}

export function prefetchPlanPreview(client: QueryClient, scope: PreviewScope, id: string) {
  return client.prefetchQuery({
    ...queries.agentPreview.plan(scope, id),
    staleTime: PREVIEW_STALE_TIME,
  });
}

type ApprovableQuery = {
  data: { digest: string; stale: boolean } | undefined;
  isPending: boolean;
  isError: boolean;
  refetch: () => Promise<unknown>;
};

/**
 * The approval side of a preview: the gate on the Approve button and the
 * answer to a refused approval.
 *
 * A digest that no longer matches means what the person approved is not what
 * would run. The server records nothing; this reads the preview again and
 * raises `changed`, so the surface can say the change looks different now and
 * leave Approve on for a second, informed look.
 */
export function useApprovalGate(query: ApprovableQuery): {
  gate: ApprovalGate;
  changed: boolean;
  /** Clears the notice when the person acts again. */
  acknowledge: () => void;
  /** True when the error was a digest mismatch and the preview is being read again. */
  handleDecisionError: (error: unknown) => boolean;
} {
  const [changed, setChanged] = useState(false);
  const { refetch } = query;

  const handleDecisionError = useCallback(
    (error: unknown) => {
      if (!isPreviewConflict(error)) {
        return false;
      }
      setChanged(true);
      void refetch();
      return true;
    },
    [refetch],
  );
  const acknowledge = useCallback(() => setChanged(false), []);

  return { gate: approvalGate(query), changed, acknowledge, handleDecisionError };
}
