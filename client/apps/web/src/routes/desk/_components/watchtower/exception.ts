import type {
  WatchtowerFilter,
  WatchtowerItem,
  WatchtowerItemEvent,
  WatchtowerItemStatus,
} from "@/lib/graphql/watchtower";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCompactAge, formatUnixDateTime } from "@trenova/shared/lib/date";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";

/**
 * Where an exception is in its owner's hands. Open asks for someone; a
 * snoozed one is set aside to come back; resolved is done whoever closed it.
 */
export function exceptionStatusAttrs(t: TranslateFn): Record<WatchtowerItemStatus, BadgeAttrProps> {
  return {
    Open: {
      phase: "attention",
      text: t("Open"),
      description: t("Nobody has said they are on it yet"),
    },
    Acknowledged: {
      phase: "awaiting",
      text: t("Acknowledged"),
      description: t("Someone has seen it and will get to it"),
    },
    InProgress: {
      phase: "active",
      text: t("In progress"),
      description: t("Someone is working on it now"),
    },
    Snoozed: {
      phase: "queued",
      text: t("Snoozed"),
      description: t("Set aside until a time; it comes back on its own"),
    },
    Resolved: {
      phase: "complete",
      text: t("Resolved"),
      description: t("Closed with a reason"),
    },
  };
}

/** The four ways a person looks at the queue: everything, theirs, nobody's, late. */
export type ExceptionView = "all" | "mine" | "unassigned" | "overdue";

export const EXCEPTION_VIEWS: readonly ExceptionView[] = ["all", "mine", "unassigned", "overdue"];

/**
 * What each view asks the server for. Only "everything" keeps the feed's
 * newest-first order; the other three are work lists, read soonest due first.
 */
export function exceptionViewFilter(view: ExceptionView): WatchtowerFilter {
  switch (view) {
    case "mine":
      return { assignedToMe: true, orderByDue: true };
    case "unassigned":
      return { unassigned: true, orderByDue: true };
    case "overdue":
      return { overdueOnly: true, orderByDue: true };
    default:
      return {};
  }
}

export type DueState =
  | { kind: "none" }
  | { kind: "snoozed"; until: number }
  | { kind: "overdue"; by: number }
  | { kind: "due"; in: number };

/**
 * How late an exception is, from the reader's clock. A snoozed or resolved
 * exception is never overdue, whatever its due time says.
 */
export function exceptionDueState(
  item: Pick<WatchtowerItem, "status" | "dueAt" | "snoozedUntil" | "resolvedAt">,
  now: number,
): DueState {
  if (item.resolvedAt != null || item.status === "Resolved") {
    return { kind: "none" };
  }
  if (item.status === "Snoozed" && item.snoozedUntil != null) {
    return { kind: "snoozed", until: item.snoozedUntil };
  }
  if (item.dueAt == null) {
    return { kind: "none" };
  }
  if (item.dueAt < now) {
    return { kind: "overdue", by: now - item.dueAt };
  }

  return { kind: "due", in: item.dueAt - now };
}

export function dueLabel(t: TranslateFn, state: DueState): string | null {
  switch (state.kind) {
    case "overdue":
      return t("Overdue {0}", formatCompactAge(state.by));
    case "due":
      return t("Due in {0}", formatCompactAge(state.in));
    case "snoozed":
      return t("Snoozed until {0}", formatUnixDateTime(state.until));
    default:
      return null;
  }
}

/** Who did it, in the history's words: a person by name, else the system. */
export function eventActor(t: TranslateFn, event: WatchtowerItemEvent): string {
  return event.actor?.name ?? t("Trenova");
}

/**
 * One history line as a sentence. Values the server stores as instants or
 * ids are turned back into words here; a code is shown by its label.
 */
export function eventSentence(
  t: TranslateFn,
  event: WatchtowerItemEvent,
  codeLabels: ReadonlyMap<string, string>,
): string {
  const instant = (value: string) => {
    const seconds = Number(value);
    return value !== "" && Number.isFinite(seconds) ? formatUnixDateTime(seconds) : "";
  };

  switch (event.kind) {
    case "Created":
      return t("Raised as {0}", event.toValue);
    case "Assigned":
      return t("Assigned");
    case "Unassigned":
      return t("Left with nobody");
    case "Acknowledged":
      return t("Acknowledged");
    case "Started":
      return t("Started working on it");
    case "Snoozed":
      return t("Snoozed until {0}", instant(event.toValue));
    case "Woke":
      return t("Came back from snooze");
    case "DueChanged":
      return event.toValue === ""
        ? t("Cleared the due time")
        : t("Due {0}", instant(event.toValue));
    case "Commented":
      return t("Commented");
    case "Escalated":
      return t("Escalated from {0} to {1}", event.fromValue, event.toValue);
    case "Resolved":
      return t("Resolved as {0}", codeLabels.get(event.toValue) ?? event.toValue);
    case "Reopened":
      return t("Reopened");
    case "SourceResolved":
      return t("The record cleared");
    case "SourceReopened":
      return t("The problem came back");
    default:
      return event.kind;
  }
}
