// tariff-coverage.ts says, in the reader's language, what a coverage region of a Tariff file is
// and that no test reaches it, for the editor's gutter, the tests panel and readiness; and picks
// the regions no test reached out of a run's coverage.

import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TariffCoverage, TariffRegion, TariffRegionKind } from "@trenova/shared/types/tariff";

/** A region's kind as a short noun: "if branch", "match arm". */
export const TARIFF_REGION_LABELS: Record<TariffRegionKind, string> = defineLabels({
  if: "If branch",
  else: "Else branch",
  arm: "Match arm",
  loop: "Loop body",
  option: "Option",
  charge: "Charge",
  refusal: "Refusal",
  function: "Function body",
});

/** The gutter's tooltip: that no test reaches the region, and what kind of region it is. */
export const TARIFF_UNREACHED_LABELS: Record<TariffRegionKind, string> = defineLabels({
  if: "No test reaches this branch of the if",
  else: "No test reaches this else branch",
  arm: "No test reaches this match arm",
  loop: "No test runs this loop's body",
  option: "No test reaches this option",
  charge: "No test bills this charge",
  refusal: "No test reaches this refusal",
  function: "No test calls this function",
});

/**
 * The regions no test or property case ran, outermost first where they nest: an unreached `if`
 * branch is listed once, not again for each charge inside it.
 */
export function unreachedRegions(coverage: TariffCoverage | null | undefined): TariffRegion[] {
  const unreached = (coverage?.regions ?? []).filter((region) => region.hits === 0);
  return unreached.filter(
    (region) =>
      !unreached.some(
        (other) =>
          other !== region &&
          other.range.from <= region.range.from &&
          region.range.to <= other.range.to &&
          (other.range.from < region.range.from || region.range.to < other.range.to),
      ),
  );
}
