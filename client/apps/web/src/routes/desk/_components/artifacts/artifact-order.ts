import { ARTIFACT_KINDS } from "@/components/assistant/voice/artifact-chrome";
import type { AssistantArtifact } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * The order a conversation's artifacts are listed in: what the person pinned
 * first, and the newest first within each group. The server hands them back
 * oldest first, which is the order they were made in and the opposite of the
 * order anybody looks for one. Two made in the same second keep the order
 * they arrived in.
 */
export function orderArtifacts(artifacts: readonly AssistantArtifact[]): AssistantArtifact[] {
  return artifacts
    .map((artifact, index) => ({ artifact, index }))
    .sort((a, b) => {
      if (a.artifact.pinned !== b.artifact.pinned) {
        return a.artifact.pinned ? -1 : 1;
      }
      if (a.artifact.createdAt !== b.artifact.createdAt) {
        return b.artifact.createdAt - a.artifact.createdAt;
      }

      return a.index - b.index;
    })
    .map((entry) => entry.artifact);
}

const identity: TranslateFn = (message) => message ?? "";

/**
 * The artifacts a few typed letters find: by title, or by what kind of thing
 * they are, so "table" lists every table. The order is kept; a search narrows
 * the list, it never reshuffles it.
 */
export function filterArtifacts(
  artifacts: readonly AssistantArtifact[],
  query: string,
  t: TranslateFn = identity,
): AssistantArtifact[] {
  const needle = query.trim().toLocaleLowerCase();
  if (needle === "") {
    return [...artifacts];
  }

  return artifacts.filter((artifact) => {
    const kind = ARTIFACT_KINDS[artifact.kind];
    const haystack = [artifact.title, t(kind.label), kind.source ? t(kind.source) : ""];

    return haystack.some((text) => text.toLocaleLowerCase().includes(needle));
  });
}
