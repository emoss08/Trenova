import { describeApiError } from "@/lib/api-error-message";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRowTotal } from "@trenova/shared/lib/row-total";
import type { ReactNode } from "react";
import type { PagedRows } from "./paged-rows";
import { CardMore, CardSectionError, CardSectionWait } from "./record-card";

/**
 * A list a card reads a page at a time: rows standing in while the first page
 * is read, what it says when there is nothing or it could not be read, the
 * rows themselves, and the next page on request with how many are shown. A
 * page that fails after the first says so under the rows it already has.
 */
export function CardPagedList<T>({
  list,
  emptyText,
  errorText,
  nextErrorText,
  loadingLabel,
  waitRows = 3,
  children,
}: {
  list: PagedRows<T>;
  emptyText: string;
  /** What the section says when the first page could not be read. */
  errorText: string;
  /** What it says when a later page could not be read. */
  nextErrorText: string;
  /** What the next-page button says while it reads. */
  loadingLabel: string;
  waitRows?: number;
  children: (rows: readonly T[]) => ReactNode;
}) {
  const t = useT();
  const { rows, error, pending } = list;
  if (error && !rows) {
    return <CardSectionError>{describeApiError(error, errorText)}</CardSectionError>;
  }
  if (pending || !rows) {
    return <CardSectionWait rows={waitRows} />;
  }
  if (rows.length === 0) {
    return <p className="dk-rec-note">{emptyText}</p>;
  }
  return (
    <>
      {children(rows)}
      {list.hasMore && (
        <CardMore
          shown={
            list.total === null
              ? t("{0} shown", rows.length)
              : list.totalExact
                ? t("{0} of {1} shown", rows.length, list.total)
                : t("{0} of about {1} shown", rows.length, formatRowTotal(list.total, false))
          }
          onMore={list.loadMore}
          loading={list.loadingMore}
          label={t("Show more")}
          loadingLabel={loadingLabel}
        />
      )}
      {error !== null && error !== undefined && (
        <CardSectionError>{describeApiError(error, nextErrorText)}</CardSectionError>
      )}
    </>
  );
}
