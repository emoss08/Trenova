import type { WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
import { ChevronRightIcon } from "@trenova/shared/components/icons";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { trainingRecordStatusLabel } from "@trenova/shared/lib/training";
import { cn } from "@trenova/shared/lib/utils";
import type { WorkerTrainingStatus } from "@trenova/shared/types/worker-training";
import { useId, useState } from "react";

/**
 * Closed records, folded under the table. They stay because an auditor asks what was
 * completed and when; a row opens the record in the pane.
 */
export function TrainingHistory({
  records,
  onRead,
}: {
  records: WorkerTrainingRecordRow[];
  onRead: (record: WorkerTrainingRecordRow) => void;
}) {
  const t = useT();
  const regionId = useId();
  const [open, setOpen] = useState(false);
  if (records.length === 0) return null;

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
        <span className="font-normal tabular-nums">{records.length}</span>
        <span className="font-normal text-foreground-subtle">{t("earlier records")}</span>
      </button>
      {open ? (
        <ScrollArea
          className="max-h-72 animate-expand-in motion-reduce:animate-none"
          maskVariant="card"
        >
          <ul id={regionId} className="flex flex-col gap-px px-1.5 pb-1.5">
            {records.map((record) => (
              <li key={record.id}>
                <button
                  type="button"
                  onClick={() => onRead(record)}
                  className="grid w-full grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)] items-baseline gap-3 rounded-md ui-focus-ring px-2 py-1.5 text-left text-xs hover:bg-surface-hover"
                >
                  <span className="truncate text-sm font-medium">
                    {record.course?.name ?? t("Course")}
                  </span>
                  <span className="truncate font-mono text-muted-foreground tabular-nums">
                    {record.completedAt
                      ? record.score
                        ? t(
                            "{0} · {1}%",
                            formatUnixDateMedium(record.completedAt),
                            Number(Number(record.score).toFixed(0)),
                          )
                        : formatUnixDateMedium(record.completedAt)
                      : trainingRecordStatusLabel(record.status as WorkerTrainingStatus)}
                  </span>
                  <span className="truncate text-muted-foreground">
                    {record.expiresAt
                      ? t("Valid to {0}", formatUnixDateMedium(record.expiresAt))
                      : record.recordedBy?.name
                        ? t("by {0}", record.recordedBy.name)
                        : trainingRecordStatusLabel(record.status as WorkerTrainingStatus)}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </ScrollArea>
      ) : null}
    </div>
  );
}
