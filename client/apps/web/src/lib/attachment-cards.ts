import type { AssistantArtifact } from "@/types/assistant";

type CardSource = Pick<AssistantArtifact, "id" | "kind" | "payload" | "lineageSeq" | "updatedAt">;

/**
 * The newest extraction card read off each document a conversation holds, by
 * document id, so the file a person attached can open what was read from it.
 * A later version of a card (the model's reading after the rule-based one)
 * takes the place of the earlier.
 */
export function extractionCardsByDocument(
  artifacts: readonly CardSource[],
): ReadonlyMap<string, string> {
  const newest = new Map<string, CardSource>();
  for (const artifact of artifacts) {
    if (artifact.kind !== "extraction") {
      continue;
    }
    const documentId = artifact.payload.documentId;
    if (typeof documentId !== "string" || documentId === "") {
      continue;
    }
    const kept = newest.get(documentId);
    if (
      !kept ||
      artifact.lineageSeq > kept.lineageSeq ||
      (artifact.lineageSeq === kept.lineageSeq && artifact.updatedAt > kept.updatedAt)
    ) {
      newest.set(documentId, artifact);
    }
  }

  const cards = new Map<string, string>();
  for (const [documentId, artifact] of newest) {
    cards.set(documentId, artifact.id);
  }
  return cards;
}
