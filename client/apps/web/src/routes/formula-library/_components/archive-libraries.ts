import { archiveRefusal } from "@/lib/archive-refusal";

/** A library the server would only archive once the person confirmed it, in its words. */
export type HeldArchive<T> = { item: T; message: string };

export type ArchiveOutcome<T> = {
  archived: number;
  /** Refused for a reason confirming cannot change: a permission, a lost connection. */
  failed: number;
  /** Refused only until confirmed; asked for again with `confirmInUse`. */
  held: HeldArchive<T>[];
  /** The server's words for the refusals that were about permission. */
  forbidden: string[];
};

/**
 * Archives each library on its own, in order. Without confirmation, one templates still
 * import is held with the server's message so the person can confirm it; once confirmed, every
 * refusal is a failure.
 */
export async function archiveLibraries<T>(
  items: readonly T[],
  confirming: boolean,
  archive: (item: T, confirmInUse: boolean) => Promise<unknown>,
): Promise<ArchiveOutcome<T>> {
  const outcome: ArchiveOutcome<T> = { archived: 0, failed: 0, held: [], forbidden: [] };
  for (const item of items) {
    try {
      await archive(item, confirming);
      outcome.archived += 1;
    } catch (error) {
      const refusal = archiveRefusal(error);
      if (refusal?.kind === "needsConfirmation" && !confirming) {
        outcome.held.push({ item, message: refusal.message });
        continue;
      }
      outcome.failed += 1;
      if (refusal?.kind === "forbidden") outcome.forbidden.push(refusal.message);
    }
  }
  return outcome;
}
