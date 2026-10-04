import { LOOKUP_ARTIFACT_KINDS, type AssistantArtifact } from "@/types/assistant";
import { deskArtKind } from "./desk-art-kinds";

/** No lookup is waiting on a reply. */
export const NO_PENDING_LOOKUPS: ReadonlySet<string> = new Set();

/**
 * The lookups a running turn has saved so far, added to the ones already
 * waiting. A lookup is kept only if the finished reply points to it, so until
 * then it is not the workspace's to show. The same set comes back when the
 * turn brought nothing new, so a reader keyed on it does not redraw.
 */
export function pendingLookupIds(
  artifacts: readonly { id: string; kind: string }[],
  current: ReadonlySet<string>,
): ReadonlySet<string> {
  const fresh = artifacts.filter(
    (artifact) => LOOKUP_ARTIFACT_KINDS.has(artifact.kind) && !current.has(artifact.id),
  );
  if (fresh.length === 0) {
    return current;
  }
  return new Set([...current, ...fresh.map((artifact) => artifact.id)]);
}

export type ArtifactCounts = {
  all: number;
  pinned: number;
  families: Record<string, number>;
};

/**
 * A page of artifacts without the lookups still waiting on the reply, and
 * its counts less the lineages that leaves empty. A lookup that is a later
 * version of something an earlier turn made hides only that version: the
 * lineage was there before the turn and stays counted.
 */
export function withoutPendingLookups<T extends AssistantArtifact>(
  results: readonly T[],
  counts: ArtifactCounts | undefined,
  pending: ReadonlySet<string>,
): { results: readonly T[]; counts: ArtifactCounts | undefined; hidden: number } {
  if (pending.size === 0) {
    return { results, counts, hidden: 0 };
  }
  const isPending = (artifact: T) =>
    pending.has(artifact.id) && LOOKUP_ARTIFACT_KINDS.has(artifact.kind);
  const kept = results.filter((artifact) => !isPending(artifact));
  if (kept.length === results.length) {
    return { results, counts, hidden: 0 };
  }

  const keptRoots = new Set(kept.map((artifact) => artifact.lineageId || artifact.id));
  const gone = new Map<string, T>();
  for (const artifact of results) {
    const root = artifact.lineageId || artifact.id;
    if (!keptRoots.has(root) && isPending(artifact)) {
      gone.set(root, artifact);
    }
  }

  if (!counts) {
    return { results: kept, counts, hidden: gone.size };
  }
  const families = { ...counts.families };
  let pinned = counts.pinned;
  for (const artifact of gone.values()) {
    const family = deskArtKind(artifact);
    if (families[family] !== undefined) {
      families[family] = Math.max(0, families[family] - 1);
    }
    if (artifact.pinned) {
      pinned -= 1;
    }
  }

  return {
    results: kept,
    counts: {
      all: Math.max(0, counts.all - gone.size),
      pinned: Math.max(0, pinned),
      families,
    },
    hidden: gone.size,
  };
}
