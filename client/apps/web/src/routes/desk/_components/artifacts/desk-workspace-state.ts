import type { ArtifactLineage } from "./desk-lineage";

/** How many cards the stack holds, the open one included. */
export const STACK_SIZE = 8;
/** How many of the newest lineages always have a card. */
export const STACK_NEWEST = 3;
/** How many recently opened lineages the session remembers for the stack. */
export const RECENT_SIZE = 6;

/**
 * What ⌘J does from where the pane is: with every artifact already showing it
 * closes the pane, and from anywhere else it opens the pane on every artifact.
 */
export function commandJ(state: { open: boolean; browsing: boolean }): {
  open: boolean;
  browsing: boolean;
} {
  if (state.open && state.browsing) {
    return { open: false, browsing: false };
  }
  return { open: true, browsing: true };
}

/** The session's recently opened lineages, newest first, without repeats. */
export function pushRecent(recent: readonly string[], id: string): string[] {
  return [id, ...recent.filter((entry) => entry !== id)].slice(0, RECENT_SIZE);
}

/**
 * The cards the stack shows: the open one, the three newest, the pinned ones
 * and the ones opened lately, each once, at most eight. The rest are a ⌘J away.
 */
export function stackLineages(
  lineages: readonly ArtifactLineage[],
  active: ArtifactLineage,
  recent: readonly string[],
): ArtifactLineage[] {
  const byId = new Map(lineages.map((lineage) => [lineage.id, lineage]));
  byId.set(active.id, active);
  const isPinned = (lineage: ArtifactLineage) => lineage.versions.some((version) => version.pinned);
  const newest = [...lineages]
    .sort((a, b) => b.latest.createdAt - a.latest.createdAt || b.id.localeCompare(a.id))
    .slice(0, STACK_NEWEST)
    .map((lineage) => lineage.id);
  const pinned = lineages.filter(isPinned).map((lineage) => lineage.id);

  const ids = [...new Set([active.id, ...newest, ...pinned, ...recent])];
  return ids
    .map((id) => byId.get(id))
    .filter((lineage): lineage is ArtifactLineage => lineage !== undefined)
    .slice(0, STACK_SIZE);
}

/**
 * Where an artifact came from when it was made on an earlier day: the day and
 * the question of its turn. The day is "yesterday" or a short date. Null for
 * one made today, which needs no note.
 */
export function olderNote(
  createdAt: number,
  turn: string,
  now: Date,
): { day: string; turn: string } | null {
  const made = new Date(createdAt * 1000);
  if (made.toDateString() === now.toDateString()) {
    return null;
  }
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1);
  const day =
    made.toDateString() === yesterday.toDateString()
      ? "yesterday"
      : made.toLocaleDateString([], { weekday: "short", month: "short", day: "numeric" });
  return { day, turn };
}

/** Steps through every lineage, wrapping at either end. */
export function stepLineage(
  lineages: readonly ArtifactLineage[],
  activeId: string,
  step: number,
): string | null {
  if (lineages.length === 0) {
    return null;
  }
  const at = Math.max(
    0,
    lineages.findIndex((lineage) => lineage.id === activeId),
  );
  return lineages[(at + step + lineages.length) % lineages.length].id;
}
