import type { TrainingExportHistoryEntry } from "@/lib/graphql/agent-control";

export type TrainingExportTotals = {
  exports: number;
  corrections: number;
  lastExportedAt: number | null;
};

/** How much of this organization's data the training exports have taken in all. */
export function trainingExportTotals(
  entries: readonly TrainingExportHistoryEntry[],
): TrainingExportTotals {
  let corrections = 0;
  let lastExportedAt: number | null = null;
  for (const entry of entries) {
    corrections += entry.examples;
    if (lastExportedAt === null || entry.exportedAt > lastExportedAt) {
      lastExportedAt = entry.exportedAt;
    }
  }

  return { exports: entries.length, corrections, lastExportedAt };
}
