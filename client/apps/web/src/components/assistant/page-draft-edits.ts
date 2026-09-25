import { createOnceClaimer } from "@/lib/claim-once";
import type { AssistantArtifactEvent } from "@/types/assistant";
import type { PageDraftEdit, PageDraftSurface } from "@/types/page-draft";
import { useEffect } from "react";

/*
 * A page assistant does not save anything. Its draft tools hand back a change
 * — accept this field, match that stop, here is a formula — and the page
 * applies it to what it holds, the way "Take me there" moves the app. The
 * change arrives as a draft_edit artifact on the turn's stream.
 *
 * It is applied once, live, and never again: a replayed turn would otherwise
 * undo whatever the person changed since, and history never applies at all.
 */

/** Marks a draft change as applied; false when this tab already applied it. */
export const claimDraftEdit = createOnceClaimer("trenova-assistant-applied-draft-edits", 200);

/**
 * The changes to apply from what a turn has announced, in the order the
 * assistant made them. Only this page's changes are claimed, so a change for
 * another page is left for it.
 */
export function nextDraftEdits(
  artifacts: readonly AssistantArtifactEvent[],
  surface: PageDraftSurface,
  claim: (id: string) => boolean,
): PageDraftEdit[] {
  const edits: PageDraftEdit[] = [];
  for (const artifact of artifacts) {
    const draft = artifact.draft;
    if (artifact.kind !== "draft_edit" || !draft || draft.surface !== surface) {
      continue;
    }
    if (claim(artifact.id)) {
      edits.push(draft);
    }
  }

  return edits;
}

/** Applies each change a live turn hands the page, once. */
export function useApplyDraftEdits(
  artifacts: readonly AssistantArtifactEvent[],
  surface: PageDraftSurface | undefined,
  onEdit: ((edit: PageDraftEdit) => void) | undefined,
) {
  useEffect(() => {
    if (surface === undefined || onEdit === undefined) {
      return;
    }
    for (const edit of nextDraftEdits(artifacts, surface, claimDraftEdit)) {
      onEdit(edit);
    }
  }, [artifacts, onEdit, surface]);
}
