import { choiceLabel, serviceFailureStatusChoices, serviceFailureTypeChoices } from "@/lib/choices";
import type {
  FailureScope,
  RecordFailure,
  RecordFailuresPage,
} from "@/lib/graphql/record-failures";
import { recordFailuresQuery } from "@/lib/queries/record-failures";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useInfiniteQuery } from "@tanstack/react-query";
import { usePagedRows, type PagedRows } from "./paged-rows";
import { cardDate, cityLine, waitText } from "./record-card-values";

/**
 * A service failure as one row reads it: its kind, where it happened or on
 * which load (whichever the list is not already about), how late, and when.
 */
export function failureLine(failure: RecordFailure, scope: FailureScope["kind"], t: TranslateFn) {
  const place = failure.stop?.location ?? null;
  return [
    choiceLabel(serviceFailureTypeChoices, failure.type),
    scope === "location"
      ? (failure.shipment?.proNumber ?? "")
      : place?.name.trim() || (place ? cityLine(place) : ""),
    failure.lateMinutes > 0 ? t("{0} late", waitText(failure.lateMinutes)) : "",
    cardDate(failure.actualArrival),
  ]
    .filter(Boolean)
    .join(" · ");
}

/** How a failure's status reads beside its row: open asks for a look, resolved is settled. */
export function failureStatus(status: RecordFailure["status"]): {
  label: string;
  state: "warn" | "ok" | null;
} {
  return {
    label: choiceLabel(serviceFailureStatusChoices, status),
    state: status === "Open" ? "warn" : status === "Resolved" ? "ok" : null,
  };
}

/**
 * A load's failures without the one the card is about: its row left out and
 * the count it belongs to taken down by one.
 */
export function withoutFailure(
  list: PagedRows<RecordFailure>,
  failureId: string,
): PagedRows<RecordFailure> {
  return {
    ...list,
    rows: list.rows?.filter((row) => row.id !== failureId),
    total: list.total === null ? null : Math.max(0, list.total - 1),
  };
}

/** A record's service failures, newest first, a page at a time. */
export function useRecordFailures({
  scope,
  pageSize,
  enabled,
}: {
  scope: FailureScope;
  pageSize: number;
  enabled: boolean;
}): PagedRows<RecordFailure> {
  return usePagedRows<RecordFailure, RecordFailuresPage>(
    useInfiniteQuery({
      ...recordFailuresQuery(scope, pageSize),
      enabled,
      retry: false,
    }),
  );
}
