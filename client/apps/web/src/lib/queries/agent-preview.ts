import {
  fetchPlanPreview,
  fetchProposalPreview,
  type PreviewScope,
} from "@/lib/graphql/agent-preview";
import { stableStringify } from "@/lib/stable-stringify";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/**
 * Previews are keyed by who asks, which record, and — for a proposal — the
 * values an approver has in mind, written in a stable order so the same draft
 * typed twice reads the same entry. The unmodified preview keys on an empty
 * string, so every surface showing a proposal as proposed shares one entry and
 * one digest.
 */
export function modificationsKey(modifications: Record<string, unknown> | null): string {
  return modifications === null || Object.keys(modifications).length === 0
    ? ""
    : stableStringify(modifications);
}

export const agentPreview = createQueryKeys("agentPreview", {
  proposal: (
    scope: PreviewScope,
    id: string,
    modifications: Record<string, unknown> | null = null,
  ) => {
    const draft = modificationsKey(modifications);

    // The request is built from the key itself, so what is fetched is
    // exactly what is cached: the draft in its stable form.
    return {
      queryKey: [scope, id, draft],
      queryFn: ({ signal }: { signal?: AbortSignal }) =>
        fetchProposalPreview(
          {
            scope,
            id,
            modifications: draft === "" ? null : (JSON.parse(draft) as Record<string, unknown>),
          },
          { signal },
        ),
    };
  },
  plan: (scope: PreviewScope, id: string) => ({
    queryKey: [scope, id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchPlanPreview({ scope, id }, { signal }),
  }),
});
