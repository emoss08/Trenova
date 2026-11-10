import type { BriefingSectionKey } from "@trenova/graphql/generated/graphql";
import { formatList } from "@trenova/shared/i18n/format";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { PartOfDay } from "@trenova/shared/lib/date";

/** Each briefing section as the page titles it, in the language on screen. */
const BRIEFING_SECTION_LABELS: Record<BriefingSectionKey, string> = defineLabels({
  Attention: "Needs attention",
  Today: "Today",
  Coverage: "The next day",
  Exceptions: "Critical",
  Decisions: "Waiting on you",
  Compliance: "Credentials",
  Billing: "Billing queue",
  Cash: "Cash received yesterday",
});

/** The greeting for the time of day, by first name when there is one. */
export function greeting(t: TranslateFn, dayPart: PartOfDay, firstName: string): string {
  switch (dayPart) {
    case "morning":
      return firstName ? t("Good morning, {0}", firstName) : t("Good morning");
    case "afternoon":
      return firstName ? t("Good afternoon, {0}", firstName) : t("Good afternoon");
    default:
      return firstName ? t("Good evening, {0}", firstName) : t("Good evening");
  }
}

/**
 * What a briefing cut to the reader's access says about the cut: the sections
 * it leaves out, and those it shows only the part of they may open.
 */
export function withheldNote(
  t: TranslateFn,
  withheld: readonly { key: BriefingSectionKey; partial: boolean }[],
): string {
  const names = (partial: boolean) =>
    formatList(
      withheld
        .filter((section) => section.partial === partial)
        .map((section) => BRIEFING_SECTION_LABELS[section.key]),
    );
  const left = withheld.some((section) => !section.partial);
  const partial = withheld.some((section) => section.partial);
  if (left && partial) {
    return t(
      "This morning's briefing leaves out {0} and shows only the part of {1} you can open.",
      names(false),
      names(true),
    );
  }
  if (partial) {
    return t("This morning's briefing shows only the part of {0} you can open.", names(true));
  }
  return t("This morning's briefing leaves out {0}, whose records you can't open.", names(false));
}
