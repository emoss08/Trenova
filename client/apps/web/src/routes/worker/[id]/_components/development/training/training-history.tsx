import { VirtualRows } from "@/components/virtual-rows";
import { preloadOn } from "@/components/work-pane";
import type { WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ChevronRightIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { trainingRecordStatusLabel } from "@trenova/shared/lib/training";
import { cn } from "@trenova/shared/lib/utils";
import type { WorkerTrainingStatus } from "@trenova/shared/types/worker-training";
import { useId, useState } from "react";
import { ListMoreLine } from "../../list-more";
import type { WorkerList } from "../../use-worker-list";
import { CoursePane } from "../development-panes";
import { formatTrainingScore } from "./training-score";

/**
 * Closed records, folded under the table. They stay because an auditor asks what was
 * completed and when; a row opens the record in the pane. The records are read a page at a
 * time, and the fold reads the next page as its end nears.
 */
export function TrainingHistory({
  records,
  list,
  onRead,
}: {
  /** The loaded records that are history: closed and not standing in a course's slot. */
  records: readonly WorkerTrainingRecordRow[];
  list: WorkerList<WorkerTrainingRecordRow>;
  onRead: (record: WorkerTrainingRecordRow) => void;
}) {
  const t = useT();
  const regionId = useId();
  const [open, setOpen] = useState(false);
  if (records.length === 0 && !list.hasMore && !list.isError) return null;

  return (
    <div className="border-t">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={regionId}
        onClick={() => setOpen((value) => !value)}
        className="flex w-full items-center gap-1.5 ui-focus-ring px-3 py-2 text-left text-xs font-medium text-muted-foreground hover:text-foreground"
      >
        <ChevronRightIcon
          aria-hidden
          className={cn(
            "size-3.5 transition-transform duration-200 ease-settle",
            open && "rotate-90",
          )}
        />
        {t("History")}
        {list.hasMore || list.isError ? null : (
          <span className="font-normal tabular-nums">{records.length}</span>
        )}
        <span className="font-normal text-foreground-subtle">{t("earlier records")}</span>
      </button>
      {open ? (
        <div id={regionId} className="animate-expand-in motion-reduce:animate-none">
          {list.isError ? (
            <div className="px-3 pb-3">
              <ErrorState
                layout="compact"
                error={list.error}
                onRetry={list.refetch}
                title={t("The earlier records could not be loaded")}
              />
            </div>
          ) : (
            <VirtualRows
              aria-label={t("Earlier records")}
              rows={records.map((record) => ({
                key: record.id,
                render: () => <HistoryRow record={record} onRead={onRead} />,
              }))}
              mask="card"
              className="max-h-72"
              contentClassName="px-1.5 pb-1.5"
              hoverGlide
              estimateSize={30}
              onEndReached={list.hasMore && !list.loadMoreFailed ? list.loadMore : undefined}
              footer={<ListMoreLine more={list} />}
            />
          )}
        </div>
      ) : null}
    </div>
  );
}

function HistoryRow({
  record,
  onRead,
}: {
  record: WorkerTrainingRecordRow;
  onRead: (record: WorkerTrainingRecordRow) => void;
}) {
  const t = useT();
  const status = trainingRecordStatusLabel(record.status as WorkerTrainingStatus);
  return (
    <button
      type="button"
      data-glide-target
      {...preloadOn(CoursePane)}
      onClick={() => onRead(record)}
      className="grid w-full grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)] items-baseline gap-3 rounded-md ui-focus-ring px-2 py-1.5 text-left text-xs"
    >
      <span className="truncate text-sm font-medium">{record.course?.name ?? t("Course")}</span>
      <span className="truncate font-mono text-muted-foreground tabular-nums">
        {record.completedAt
          ? record.score
            ? t(
                "{0} · {1}%",
                formatUnixDateMedium(record.completedAt),
                formatTrainingScore(record.score),
              )
            : formatUnixDateMedium(record.completedAt)
          : status}
      </span>
      <span className="truncate text-muted-foreground">
        {record.expiresAt
          ? t("Valid to {0}", formatUnixDateMedium(record.expiresAt))
          : record.recordedBy?.name
            ? t("by {0}", record.recordedBy.name)
            : status}
      </span>
    </button>
  );
}
