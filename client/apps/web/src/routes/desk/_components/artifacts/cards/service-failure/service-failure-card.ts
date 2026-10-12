import {
  choiceLabel,
  serviceFailureReasonCategoryChoices,
  serviceFailureSourceChoices,
  serviceFailureStatusChoices,
  serviceFailureTypeChoices,
  stopTypeChoices,
} from "@/lib/choices";
import type { DeskServiceFailure } from "@/lib/graphql/service-failure-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { PathStep } from "../../record-card/card-path";
import type { CardFact, CardStat, CardTone } from "../../record-card/record-card";
import {
  cardDate,
  cardMoment,
  cardMomentAfter,
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

/**
 * The failure's clock: when the stop was due, the grace it was given, when the
 * truck arrived (the time alone on the day it was due), and by how much it
 * missed. The arrival and the miss read bad.
 */
export function failureTimes(failure: DeskServiceFailure, t: TranslateFn): CardStat[] {
  return [
    { key: "due", label: t("Due by"), value: cardMoment(failure.scheduledCutoff) },
    { key: "grace", label: t("Grace"), value: waitText(failure.gracePeriodMinutes) },
    {
      key: "arrived",
      label: t("Arrived"),
      value: cardMomentAfter(failure.scheduledCutoff, failure.actualArrival),
      state: "bad",
    },
    { key: "late", label: t("Late by"), value: waitText(failure.lateMinutes), state: "bad" },
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

/** Why the failure happened, as its reason code names it, or null when none was given. */
export function failureCause(
  failure: Pick<DeskServiceFailure, "reasonCode">,
): { name: string; code: string; category: string; description: string } | null {
  const reason = failure.reasonCode ?? null;
  if (!reason) return null;
  return {
    name: reason.label.trim() || reason.code,
    code: reason.code,
    category: choiceLabel(serviceFailureReasonCategoryChoices, reason.category),
    description: reason.description.trim(),
  };
}

/** Whether the failure still asks for someone: open or reviewed, and not settled. */
export function isUnsettled(status: FailureStatus): boolean {
  return status === "Open" || status === "Reviewed";
}

/** An X12 code the failure's 214 sends in place of the one its type gives, or that it is derived. */
function x12Code(code: string, t: TranslateFn): string {
  return code.trim() || t("From the failure type");
}

/** The facts the card lists about the failure, in the order the office reads them. */
export function failureFacts(failure: DeskServiceFailure, t: TranslateFn): CardFact[] {
  return [
    { key: "number", label: t("Number"), value: failure.number },
    { key: "type", label: t("Type"), value: choiceLabel(serviceFailureTypeChoices, failure.type) },
    {
      key: "source",
      label: t("Source"),
      value: choiceLabel(serviceFailureSourceChoices, failure.source),
    },
    { key: "stop", label: t("Stop"), value: stopLine(failure, t) },
    {
      key: "status",
      label: t("214 status code"),
      value: x12Code(failure.x12StatusCodeOverride, t),
    },
    {
      key: "reason",
      label: t("214 reason code"),
      value: x12Code(failure.x12ReasonCodeOverride, t),
    },
    {
      key: "exception",
      label: t("214 exception code"),
      value: failure.x12ExceptionCode.trim() || t("None"),
    },
    { key: "updated", label: t("Last changed"), value: cardMoment(failure.updatedAt) },
  ];
}
