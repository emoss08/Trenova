import type {
  DeskIftaReturnCard,
  DeskIftaReturnLine,
  DeskIftaReturnProblem,
} from "@/lib/graphql/ifta-return-desk-card";
import {
  IFTA_GALLONS_SCALE,
  IFTA_MILES_DISPLAY_SCALE,
  formatIftaMeasure,
  isNegativeDecimal,
  periodInclusiveEnd,
} from "@/lib/ifta-return";
import type { IftaProblemCode, IftaReturnStatus } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  IFTA_FUEL_TYPE_LABELS,
  IFTA_RETURN_STATUS_LABELS,
} from "@trenova/shared/types/fuel-ifta-enums";
import type { CardCheckState, CardStat, CardTone } from "../../record-card/record-card";
import { cardDate, money } from "../../record-card/record-card-values";

/** How many lines the card draws before it counts the rest. */
export const IFTA_LINES_SHOWN = 8;

const DAY_SECONDS = 86_400;

/** The server's own names for each problem a return can carry (ifta.ProblemCode.Label). */
const PROBLEM_LABELS: Record<IftaProblemCode, string> = defineLabels({
  MissingRate: "Missing tax rate",
  NoFuelForType: "No fuel purchased for fuel type",
  UnattributedMiles: "Miles not attributed to a jurisdiction",
  NoTractorMiles: "Miles without a tractor",
  MileageMismatch: "Jurisdiction miles do not match move distance",
  NonMemberActivity: "Activity in a non-member jurisdiction",
  NonQualifiedActivity: "Activity on non-qualified tractors",
});

/** Where the return stands, in the IFTA page's words and colours. */
export function returnStanding(status: IftaReturnStatus): { label: string; tone: CardTone } {
  switch (status) {
    case "Draft":
      return { label: IFTA_RETURN_STATUS_LABELS.Draft, tone: "draft" };
    case "Finalized":
      return { label: IFTA_RETURN_STATUS_LABELS.Finalized, tone: "progress" };
    case "Filed":
      return { label: IFTA_RETURN_STATUS_LABELS.Filed, tone: "success" };
  }
}

/** The return's name: its quarter, and which amendment when it is one. */
export function returnTitle(card: DeskIftaReturnCard, t: TranslateFn): string {
  return card.amendmentNumber > 0
    ? t("{0}, amendment {1}", card.period.label, card.amendmentNumber)
    : card.period.label;
}

/** What the quarter comes to: the amount, and whether it is owed or a credit. */
export function netFigure(
  card: DeskIftaReturnCard,
  t: TranslateFn,
): { amount: string; words: string } {
  const credit = isNegativeDecimal(card.netDue);
  const magnitude = credit ? card.netDue.replace("-", "") : card.netDue;
  return {
    amount: money(magnitude, card.currencyCode),
    words: credit ? t("credit") : t("owed"),
  };
}

/** The facts under the status: when it is due or was filed, and when the quarter ends. */
export function returnFacts(card: DeskIftaReturnCard, now: number, t: TranslateFn): string[] {
  const lastDay = periodInclusiveEnd(card.period.end);
  return [
    card.status === "Filed" && card.filedAt
      ? t("Filed {0}", cardDate(card.filedAt))
      : t("Due {0}", cardDate(card.period.dueDate)),
    now < card.period.end ? t("Quarter ends {0}", cardDate(lastDay)) : "",
    card.filingReference?.trim() ? t("Reference {0}", card.filingReference.trim()) : "",
  ];
}

export type ReturnCheck = {
  key: string;
  state: CardCheckState;
  title: string;
  detail: string;
};

/**
 * The return's problems as checks, one per kind of problem: a missing rate
 * blocks finalizing it, every other problem asks for a look. A return with
 * none says so.
 */
export function returnChecks(card: DeskIftaReturnCard, now: number, t: TranslateFn): ReturnCheck[] {
  const groups = new Map<IftaProblemCode, DeskIftaReturnProblem[]>();
  for (const problem of card.problems) {
    groups.set(problem.code, [...(groups.get(problem.code) ?? []), problem]);
  }
  const checks: ReturnCheck[] = [...groups].map(([code, problems]) => ({
    key: code,
    state: code === "MissingRate" ? "bad" : "warn",
    title: PROBLEM_LABELS[code],
    detail:
      problems.length > 1
        ? t("{0}, and {1} more like it", problems[0].message, problems.length - 1)
        : problems[0].message,
  }));
  checks.sort((a, b) => Number(b.state === "bad") - Number(a.state === "bad"));
  if (card.status !== "Filed" && now > card.period.dueDate + DAY_SECONDS) {
    checks.push({
      key: "late",
      state: "warn",
      title: t("Past the filing date"),
      detail: t(
        "The return is not filed; the quarter's filing date was {0}.",
        cardDate(card.period.dueDate),
      ),
    });
  }
  if (checks.length === 0) {
    checks.push({
      key: "clear",
      state: "ok",
      title: t("Nothing on the return needs a look"),
      detail: t("Every line has a rate and every move's miles are on the return."),
    });
  }
  return checks;
}

/** The quarter's figures side by side. */
export function returnStats(card: DeskIftaReturnCard, t: TranslateFn): CardStat[] {
  return [
    {
      key: "miles",
      label: t("Miles"),
      value: formatIftaMeasure(card.totalMiles, IFTA_MILES_DISPLAY_SCALE),
    },
    {
      key: "paid",
      label: t("Tax-paid gallons"),
      value: formatIftaMeasure(card.totalTaxPaidGallons, IFTA_GALLONS_SCALE),
    },
    {
      key: "net",
      label: t("Net taxable gallons"),
      value: formatIftaMeasure(card.netTaxableGallons, IFTA_GALLONS_SCALE),
    },
    { key: "tax", label: t("Tax due"), value: money(card.taxDue, card.currencyCode) },
  ];
}

export type LineRow = {
  id: string;
  name: string;
  sub: string;
  amount: string;
  state: CardCheckState;
};

function weight(line: DeskIftaReturnLine): number {
  return line.rateMissing ? Number.POSITIVE_INFINITY : Math.abs(Number(line.lineTotal));
}

/** One line as the card writes it: the jurisdiction, its miles and fuel, and what it owes. */
export function lineRow(line: DeskIftaReturnLine, currency: string, t: TranslateFn): LineRow {
  const sub = t(
    "{0}, {1} mi, {2} gal bought",
    IFTA_FUEL_TYPE_LABELS[line.fuelType],
    formatIftaMeasure(line.totalMiles, IFTA_MILES_DISPLAY_SCALE),
    formatIftaMeasure(line.taxPaidGallons, IFTA_GALLONS_SCALE),
  );
  if (line.rateMissing) {
    return {
      id: line.id,
      name: line.jurisdiction.name,
      sub,
      amount: t("Rate missing"),
      state: "bad",
    };
  }
  const credit = isNegativeDecimal(line.lineTotal);
  const magnitude = money(credit ? line.lineTotal.replace("-", "") : line.lineTotal, currency);
  return {
    id: line.id,
    name: line.jurisdiction.name,
    sub,
    amount: credit ? t("{0} credit", magnitude) : t("{0} owed", magnitude),
    state: credit ? "ok" : "info",
  };
}

/** The lines that weigh most on the return, every unpriced one first, and how many are left. */
export function weightiestLines(
  card: DeskIftaReturnCard,
  t: TranslateFn,
): { rows: LineRow[]; rest: number } {
  const sorted = [...card.lines].sort((a, b) => weight(b) - weight(a));
  const missing = sorted.filter((line) => line.rateMissing).length;
  const shown = sorted.slice(0, Math.max(IFTA_LINES_SHOWN, missing));
  return {
    rows: shown.map((line) => lineRow(line, card.currencyCode, t)),
    rest: card.lines.length - shown.length,
  };
}
