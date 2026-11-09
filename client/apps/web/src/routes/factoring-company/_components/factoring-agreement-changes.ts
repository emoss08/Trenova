import { FACTORING_RECOURSE_LABELS, formatBasisPoints } from "@/lib/factoring";
import type { FactoringAgreement } from "@/lib/graphql/factoring";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatMinor } from "@trenova/shared/lib/money";

export function sortedTiers(agreement: FactoringAgreement) {
  return [...agreement.feeTiers].sort((a, b) => a.daysFrom - b.daysFrom);
}

type FeeTier = FactoringAgreement["feeTiers"][number];

/** One fee tier as the compact line the terms list: "0–30 2.5%", "75+ 6% + $25.00". */
export function feeTierText(t: TranslateFn, tier: FeeTier): string {
  const rate = formatBasisPoints(tier.rateBp);
  if (tier.daysTo == null) {
    return tier.flatMinor > 0
      ? t("{0}+ {1} + {2}", tier.daysFrom, rate, formatMinor(tier.flatMinor))
      : t("{0}+ {1}", tier.daysFrom, rate);
  }
  return tier.flatMinor > 0
    ? t("{0}–{1} {2} + {3}", tier.daysFrom, tier.daysTo, rate, formatMinor(tier.flatMinor))
    : t("{0}–{1} {2}", tier.daysFrom, tier.daysTo, rate);
}

/** What one agreement changes from another, as short chips: "advance 90%→92%". */
export function agreementDifferences(
  t: TranslateFn,
  draft: FactoringAgreement,
  active: FactoringAgreement,
) {
  const out: string[] = [];
  if (active.advanceRateBp !== draft.advanceRateBp) {
    out.push(
      t(
        "advance {0}→{1}",
        formatBasisPoints(active.advanceRateBp),
        formatBasisPoints(draft.advanceRateBp),
      ),
    );
  }
  if (active.reserveRateBp !== draft.reserveRateBp) {
    out.push(
      t(
        "reserve {0}→{1}",
        formatBasisPoints(active.reserveRateBp),
        formatBasisPoints(draft.reserveRateBp),
      ),
    );
  }
  if (active.minimumFeeMinor !== draft.minimumFeeMinor) {
    out.push(
      t("minimum {0}→{1}", formatMinor(active.minimumFeeMinor), formatMinor(draft.minimumFeeMinor)),
    );
  }
  if (active.fundingDays !== draft.fundingDays) {
    out.push(t("funding {0}→{1} days", active.fundingDays, draft.fundingDays));
  }
  if (active.chargebackWindowDays !== draft.chargebackWindowDays) {
    out.push(t("window {0}→{1} days", active.chargebackWindowDays, draft.chargebackWindowDays));
  }
  if (active.recourse !== draft.recourse) {
    out.push(FACTORING_RECOURSE_LABELS[draft.recourse]);
  }
  const before = sortedTiers(active);
  const after = sortedTiers(draft);
  if (
    before.length !== after.length ||
    before.some(
      (tier, index) =>
        tier.daysFrom !== after[index]?.daysFrom ||
        tier.daysTo !== after[index]?.daysTo ||
        tier.rateBp !== after[index]?.rateBp ||
        tier.flatMinor !== after[index]?.flatMinor,
    )
  ) {
    out.push(t("fee tiers"));
  }
  return out;
}
