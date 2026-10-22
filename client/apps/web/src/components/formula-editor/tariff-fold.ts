// tariff-fold.ts folds what `tariff.check`'s outline marks as foldable: blocks, match arms,
// records, lists, argument lists and doc comments.

import { foldService } from "@codemirror/language";
import type { Extension } from "@codemirror/state";
import { checkService, type TariffLanguageService } from "./tariff-service";

/** The range to fold for the line starting at `lineStart`, or null. */
export function foldAt(
  folding: readonly { range: { from: number; to: number } }[],
  lineStart: number,
  lineEnd: number,
): { from: number; to: number } | null {
  let best: { from: number; to: number } | null = null;
  for (const fold of folding) {
    const { from, to } = fold.range;
    if (from < lineStart || from > lineEnd || to <= lineEnd) continue;
    if (!best || to - from > best.to - best.from) best = { from: from + 1, to: to - 1 };
  }
  return best && best.to > best.from ? best : null;
}

export function tariffFolding(service: TariffLanguageService): Extension {
  return foldService.of((state, lineStart, lineEnd) => {
    const outline = checkService(service, state.doc.toString()).outline;
    return outline ? foldAt(outline.folding, lineStart, lineEnd) : null;
  });
}
