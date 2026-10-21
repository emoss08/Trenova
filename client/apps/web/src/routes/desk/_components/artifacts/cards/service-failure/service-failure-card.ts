import {
  choiceLabel,
  serviceFailureReasonCategoryChoices,
  serviceFailureStatusChoices,
  stopTypeChoices,
} from "@/lib/choices";
import type { DeskServiceFailure } from "@/lib/graphql/service-failure-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { PathStep } from "../../record-card/card-path";
import type { CardTone } from "../../record-card/record-card";
import {
  cardDate,
  cardDay,
  cardTime,
  cityLine,
  waitText,
} from "../../record-card/record-card-values";

type FailureStatus = DeskServiceFailure["status"];

/** Where the failure stands, as the pill at the card's head says it. */
export function failureStanding(status: FailureStatus): { label: string; tone: CardTone } {
  const label = choiceLabel(serviceFailureStatusChoices, status);
  switch (status) {
    case "Open":
      return { label, tone: "warn" };
    case "Reviewed":
      return { label, tone: "progress" };
    case "Resolved":
      return { label, tone: "success" };
    case "Voided":
      return { label, tone: "muted" };
  }
}

/** The stop the failure happened at, by name, else by its town, else said to be unknown. */
export function failurePlace(failure: Pick<DeskServiceFailure, "stop">, t: TranslateFn): string {
  const location = failure.stop?.location ?? null;
  if (!location) return t("Unknown stop");
  return location.name.trim() || cityLine(location) || t("Unknown stop");
}

/**
 * Which stop on the load it was: its kind and its place in the run, "Delivery,
 * stop 3", from the stop the failure names or, when that is gone, the kind
 * the failure recorded.
 */
export function stopLine(
  failure: Pick<DeskServiceFailure, "stop" | "stopType">,
  t: TranslateFn,
): string {
  const kind = choiceLabel(stopTypeChoices, failure.stop?.type ?? failure.stopType);
  return failure.stop ? t("{0}, stop {1}", kind, failure.stop.sequence) : kind;
}

/** A moment on the failure's clock: the cutoff, the end of its grace, or the arrival. */
export type ClockEvent = {
  kind: "event";
  key: "due" | "grace" | "arrived";
  label: string;
  /** The time of day, with the day beside it when it is not the day the clock last named. */
  time: string;
  day: string | null;
  state: "warn" | "bad" | null;
};

/** The stretch between two moments, named by the window it falls in. */
export type ClockGap = {
  kind: "gap";
  key: string;
  window: "early" | "grace" | "late";
  text: string;
  /** How much of the clock's whole span the stretch takes, 0 to 1. */
  share: number;
};

const MINUTE = 60;

/**
 * The failure's clock in order: when the stop was due, when its
 * grace ran out and when the truck arrived, with each stretch between them
 * named by the window it falls in (before the cutoff, inside the grace, or
 * late) and its share of the whole. Grace of none leaves the cutoff alone;
 * an arrival inside the grace or before the cutoff lands where it fell.
 */
export function missClock(
  failure: Pick<DeskServiceFailure, "scheduledCutoff" | "actualArrival" | "gracePeriodMinutes">,
  t: TranslateFn,
): (ClockEvent | ClockGap)[] {
  const cutoff = failure.scheduledCutoff;
  const graceEnd = cutoff + Math.max(0, failure.gracePeriodMinutes) * MINUTE;
  const arrival = failure.actualArrival;
  const late = arrival > graceEnd;
  const moments: { key: ClockEvent["key"]; at: number; label: string }[] = [
    { key: "due", at: cutoff, label: t("Due by") },
    ...(graceEnd > cutoff ? [{ key: "grace" as const, at: graceEnd, label: t("Grace ends") }] : []),
    { key: "arrived", at: arrival, label: t("Arrived") },
  ];
  moments.sort((a, b) => a.at - b.at);
  const first = moments[0]?.at ?? cutoff;
  const span = Math.max(MINUTE, (moments.at(-1)?.at ?? cutoff) - first);

  const clock: (ClockEvent | ClockGap)[] = [];
  let lastDay = "";
  moments.forEach((moment, index) => {
    const previous = index > 0 ? moments[index - 1] : undefined;
    if (previous && moment.at > previous.at) {
      const seconds = moment.at - previous.at;
      const middle = previous.at + seconds / 2;
      const window = middle < cutoff ? "early" : middle < graceEnd ? "grace" : "late";
      const length = waitText(Math.ceil(seconds / MINUTE));
      clock.push({
        kind: "gap",
        key: `${previous.key}-${moment.key}`,
        window,
        text:
          window === "early"
            ? t("{0} before the cutoff", length)
            : window === "grace"
              ? t("{0} of grace", length)
              : t("Late by {0}", length),
        share: seconds / span,
      });
    }
    const day = cardDay(moment.at);
    clock.push({
      kind: "event",
      key: moment.key,
      label: moment.label,
      time: cardTime(moment.at),
      day: day === lastDay ? null : day,
      state:
        moment.key === "arrived"
          ? late
            ? "bad"
            : null
          : moment.key === "grace" && late
            ? "warn"
            : null,
    });
    lastDay = day;
  });
  return clock;
}

/** One code the failure's 214 carries: the one set on it, else where it comes from. */
export type EdiCode = {
  key: "status" | "reason" | "exception";
  label: string;
  code: string | null;
  note: string;
};

/**
 * The codes the failure's X12 214 sends: its status and reason codes, each the
 * one set on the failure or taken from its type, and an exception code when
 * one was given.
 */
export function ediCodes(
  failure: Pick<
    DeskServiceFailure,
    "x12StatusCodeOverride" | "x12ReasonCodeOverride" | "x12ExceptionCode"
  >,
  t: TranslateFn,
): EdiCode[] {
  const override = (key: "status" | "reason", label: string, value: string): EdiCode => {
    const code = value.trim();
    return {
      key,
      label,
      code: code || null,
      note: code ? t("Set on this failure") : t("From the failure type"),
    };
  };
  const exception = failure.x12ExceptionCode.trim();
  return [
    override("status", t("Status code"), failure.x12StatusCodeOverride),
    override("reason", t("Reason code"), failure.x12ReasonCodeOverride),
    {
      key: "exception",
      label: t("Exception code"),
      code: exception || null,
      note: exception ? t("Set on this failure") : t("None"),
    },
  ];
}

/** How the failure came to be recorded: noticed by the system, logged by hand, or received. */
function recordedLabel(source: DeskServiceFailure["source"], t: TranslateFn): string {
  switch (source) {
    case "Detected":
      return t("Detected");
    case "Manual":
      return t("Logged by hand");
    case "EDI":
      return t("Received over EDI");
    case "Integration":
      return t("Received from an integration");
  }
}

/** When a step was taken and by whom, as the line under its name says it. */
function takenLine(at: number | null | undefined, by: { name: string } | null | undefined) {
  return [cardDate(at ?? null), by?.name.trim() ?? ""].filter(Boolean).join(" · ");
}

/**
 * The failure's way from being recorded to settled, down a rail: recorded,
 * reviewed, then resolved or voided. A step taken says when and by whom; the
 * one the failure waits at says what it waits on; a void is a close, with why.
 */
export function reviewSteps(failure: DeskServiceFailure, t: TranslateFn): PathStep[] {
  const status = failure.status;
  const reviewed = failure.reviewedAt ?? null;
  const steps: PathStep[] = [
    {
      key: "recorded",
      label: recordedLabel(failure.source, t),
      line: takenLine(failure.detectedAt, failure.createdBy),
      state: "done",
    },
    {
      key: "reviewed",
      label: t("Reviewed"),
      line:
        reviewed !== null
          ? takenLine(reviewed, failure.reviewedBy)
          : status === "Open"
            ? t("Waiting on a reviewer")
            : t("Not reviewed"),
      state: reviewed !== null ? "done" : status === "Open" ? "now" : "todo",
    },
  ];
  if (status === "Voided") {
    steps.push({
      key: "voided",
      label: t("Voided"),
      line: takenLine(failure.voidedAt, failure.voidedBy),
      state: "now",
      tone: "closed",
      note: failure.voidReason,
    });
    return steps;
  }
  const resolved = failure.resolvedAt ?? null;
  steps.push({
    key: "resolved",
    label: t("Resolved"),
    line:
      resolved !== null
        ? takenLine(resolved, failure.resolvedBy)
        : status === "Reviewed"
          ? t("Waiting to be resolved")
          : t("Not yet"),
    state: resolved !== null ? "done" : status === "Reviewed" ? "now" : "todo",
  });
  return steps;
}

/**
 * Why the failure happened, as its reason code names it, or null when none was given. The
 * code's description is outside its reference view; without reason code read it is left out
 * and the cause stands on the code's name.
 */
export function failureCause(
  failure: Pick<DeskServiceFailure, "reasonCode">,
): { name: string; code: string; category: string; description: string } | null {
  const reason = failure.reasonCode ?? null;
  if (!reason) return null;
  return {
    name: reason.label.trim() || reason.code,
    code: reason.code,
    category: choiceLabel(serviceFailureReasonCategoryChoices, reason.category),
    description: reason.description?.trim() ?? "",
  };
}

/** Whether the failure still asks for someone: open or reviewed, and not settled. */
export function isUnsettled(status: FailureStatus): boolean {
  return status === "Open" || status === "Reviewed";
}
