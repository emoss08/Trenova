import type { EmploymentEffect } from "@/lib/graphql/worker-employment";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  CalendarRangeIcon,
  ClipboardCheckIcon,
  GraduationHat01Icon,
  type IconComponent,
  Lock01Icon,
  SlashCircle01Icon,
  SunsetIcon,
  Truck01Icon,
  Wallet02Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";

/** Time off types as the effect lines name them. */
export const EFFECT_PTO_TYPE_LABELS: Record<
  NonNullable<EmploymentEffect["ptoType"]>,
  string
> = defineLabels({
  Vacation: "Vacation",
  Sick: "Sick",
  Holiday: "Holiday",
  Bereavement: "Bereavement",
  Maternity: "Maternity",
  Paternity: "Paternity",
  Personal: "Personal",
});

/** Whether the line describes what saving will do, or what the save just did. */
export type EffectTense = "will" | "did";

export type EffectLine = {
  icon: IconComponent;
  text: string;
  /** A muted second line: the record's name, the time off type. */
  sub: string | null;
  /** Something the worker loses: drawn in the danger tone. */
  danger: boolean;
  /** Only after a save: the effect was attempted and did not happen. */
  failed: boolean;
};

function days(effect: EmploymentEffect): number {
  const value = Number(effect.days ?? 0);
  return Number.isFinite(value) ? value : 0;
}

function ptoType(effect: EmploymentEffect): string | null {
  return effect.ptoType ? EFFECT_PTO_TYPE_LABELS[effect.ptoType] : null;
}

/**
 * One effect of an employment event in words. The server sends the effect's kind and figures
 * and never any English, so every sentence is written here, with its plural, in both tenses.
 */
export function describeEmploymentEffect(
  effect: EmploymentEffect,
  t: TranslateFn,
  tense: EffectTense = "will",
): EffectLine {
  const will = tense === "will";
  const line = (
    icon: IconComponent,
    text: string,
    sub: string | null = null,
    danger = false,
  ): EffectLine => ({ icon, text, sub, danger, failed: false });

  switch (effect.kind) {
    case "DispatchRemoved":
      return line(
        SlashCircle01Icon,
        will
          ? t("Takes the worker off the dispatch board until it is lifted.")
          : t("Took the worker off the dispatch board."),
        null,
        true,
      );
    case "DispatchRestored":
      return line(
        Truck01Icon,
        will
          ? t("Puts the worker back on the dispatch board.")
          : t("Put the worker back on the dispatch board."),
      );
    case "PtoPolicyEnded":
      return line(
        SunsetIcon,
        will ? t("Ends the time off policy.") : t("Ended the time off policy."),
        effect.name,
        true,
      );
    case "PtoPaidOut":
      return line(
        Wallet02Icon,
        will
          ? t("Pays out {0, plural, one {# day} other {# days}} of time off.", days(effect))
          : t("Paid out {0, plural, one {# day} other {# days}} of time off.", days(effect)),
        ptoType(effect),
        true,
      );
    case "PtoForfeited":
      return line(
        XCloseIcon,
        will
          ? t("Forfeits {0, plural, one {# day} other {# days}} of time off.", days(effect))
          : t("Forfeited {0, plural, one {# day} other {# days}} of time off.", days(effect)),
        ptoType(effect),
        true,
      );
    case "TimeOffCancelled": {
      const from = formatUnixDateMedium(effect.startDate);
      const to = formatUnixDateMedium(effect.endDate);
      const range = effect.startDate && effect.endDate;
      const text = range
        ? will
          ? t("Cancels the time off request from {0} to {1}.", from, to)
          : t("Cancelled the time off request from {0} to {1}.", from, to)
        : will
          ? t("Cancels an upcoming time off request.")
          : t("Cancelled an upcoming time off request.");
      const type = ptoType(effect);
      const length =
        effect.days != null ? t("{0, plural, one {# day} other {# days}}", days(effect)) : null;
      return line(
        CalendarRangeIcon,
        text,
        type && length ? t("{0} · {1}", type, length) : (type ?? length),
        true,
      );
    }
    case "PayAssignmentEnded":
      return line(
        Wallet02Icon,
        will ? t("Ends the pay assignment.") : t("Ended the pay assignment."),
        effect.name,
        true,
      );
    case "PortalAccessRevoked":
      if (effect.failed) {
        return {
          icon: Lock01Icon,
          text: t("Dash access could not be shut off. Revoke it by hand."),
          sub: null,
          danger: true,
          failed: true,
        };
      }
      return line(
        Lock01Icon,
        will ? t("Revokes Dash access.") : t("Revoked Dash access."),
        null,
        true,
      );
    case "ChecklistStarted":
      return line(
        ClipboardCheckIcon,
        effect.name
          ? will
            ? t("Starts the {0} checklist.", effect.name)
            : t("Started the {0} checklist.", effect.name)
          : will
            ? t("Starts a checklist.")
            : t("Started a checklist."),
      );
    case "ChecklistClosed": {
      const count = effect.count ?? 1;
      if (count === 1 && effect.name) {
        return line(
          ClipboardCheckIcon,
          will
            ? t("Closes the {0} checklist, which this makes moot.", effect.name)
            : t("Closed the {0} checklist, which this made moot.", effect.name),
        );
      }
      return line(
        ClipboardCheckIcon,
        will
          ? t(
              "Closes {0, plural, one {# open checklist} other {# open checklists}} this makes moot.",
              count,
            )
          : t(
              "Closed {0, plural, one {# open checklist} other {# open checklists}} this made moot.",
              count,
            ),
      );
    }
    case "TrainingAssigned": {
      const count = effect.count ?? 0;
      return line(
        GraduationHat01Icon,
        will
          ? t("Assigns {0, plural, one {# required course} other {# required courses}}.", count)
          : t("Assigned {0, plural, one {# required course} other {# required courses}}.", count),
      );
    }
    case "DefaultPtoPolicyApplied":
      return line(
        SunsetIcon,
        will
          ? t("Applies the default time off policy.")
          : t("Applied the default time off policy."),
        effect.name,
      );
  }
}

/** The effects a save attempted and could not carry out; the person is told each one. */
export function failedEffects(effects: readonly EmploymentEffect[]): EmploymentEffect[] {
  return effects.filter((effect) => effect.failed);
}
