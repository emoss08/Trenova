import type { AssistantArtifact } from "@/types/assistant";

/**
 * One thing the conversation made, with every version of it: a table read
 * again in a later turn is the next version of the table read before, not a
 * second table. The latest is what the workspace opens.
 */
export type ArtifactLineage = {
  /** The first artifact's id, which names the lineage. */
  id: string;
  versions: AssistantArtifact[];
  latest: AssistantArtifact;
};

/** Groups artifacts by lineage, pinned then newest first, each with its versions oldest first. */
export function groupLineages(artifacts: readonly AssistantArtifact[]): ArtifactLineage[] {
  const byRoot = new Map<string, AssistantArtifact[]>();
  for (const artifact of artifacts) {
    const root = artifact.lineageId || artifact.id;
    const group = byRoot.get(root);
    if (group) {
      group.push(artifact);
    } else {
      byRoot.set(root, [artifact]);
    }
  }

  return [...byRoot.entries()]
    .map(([id, versions]) => {
      const ordered = [...versions].sort(
        (a, b) => (a.lineageSeq ?? 1) - (b.lineageSeq ?? 1) || a.createdAt - b.createdAt,
      );
      return { id, versions: ordered, latest: ordered[ordered.length - 1] };
    })
    .sort((a, b) => {
      const pinnedA = a.versions.some((version) => version.pinned);
      const pinnedB = b.versions.some((version) => version.pinned);
      if (pinnedA !== pinnedB) {
        return pinnedA ? -1 : 1;
      }
      return b.latest.createdAt - a.latest.createdAt || b.id.localeCompare(a.id);
    });
}

/** The lineage an artifact belongs to, by any of its versions' ids. */
export function lineageContaining(
  lineages: readonly ArtifactLineage[],
  artifactId: string | null | undefined,
): ArtifactLineage | null {
  if (!artifactId) {
    return null;
  }
  return (
    lineages.find(
      (lineage) =>
        lineage.id === artifactId || lineage.versions.some((version) => version.id === artifactId),
    ) ?? null
  );
}
