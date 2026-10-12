import { recordPath } from "@/config/record-links";
import type { FailureScope, RecordFailure } from "@/lib/graphql/record-failures";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CardPagedList } from "./card-paged-list";
import type { PagedRows } from "./paged-rows";
import { CardRecordRow } from "./record-card";
import { failureLine, failureStatus } from "./record-failures";

/**
 * A record's service failures, newest first, a page at a time: each by its
 * number, what it was and how late, with its status, opened in the card's
 * place when the card can.
 */
export function CardFailures({
  failures,
  scope,
  onOpen,
  emptyText,
}: {
  failures: PagedRows<RecordFailure>;
  scope: FailureScope["kind"];
  onOpen: ((failureId: string) => void) | undefined;
  emptyText: string;
}) {
  const t = useT();
  return (
    <CardPagedList
      list={failures}
      waitRows={2}
      emptyText={emptyText}
      errorText={t("The service failures could not be read.")}
      nextErrorText={t("The next service failures could not be read.")}
      loadingLabel={t("Reading more service failures…")}
    >
      {(rows) => (
        <div className="dk-rec-list">
          {rows.map((failure) => {
            const status = failureStatus(failure.status);
            return (
              <CardRecordRow
                key={failure.id}
                to={recordPath("service_failure", failure.id)}
                onOpen={onOpen && (() => onOpen(failure.id))}
              >
                <span className="dk-rec-main">
                  <b>{failure.number}</b>
                  <span className="dk-rec-sub">{failureLine(failure, scope, t)}</span>
                </span>
                <span className={cn("dk-rec-fst", status.state && `dk-${status.state}`)}>
                  {status.label}
                </span>
              </CardRecordRow>
            );
          })}
        </div>
      )}
    </CardPagedList>
  );
}
