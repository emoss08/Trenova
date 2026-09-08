import type { WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
import { TrainingHealthBadge } from "@trenova/shared/components/training-health-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  WORKER_TRAINING_STATUS_LABELS,
  type WorkerTrainingHealth,
  type WorkerTrainingStatus,
} from "@trenova/shared/types/worker-training";
import { ChevronDownIcon } from "lucide-react";
import { useState } from "react";

/**
 * Closed records, folded away. They stay because an auditor asks what was
 * completed and when, not because anybody reads them day to day.
 */
export function TrainingHistory({ records }: { records: WorkerTrainingRecordRow[] }) {
  const [open, setOpen] = useState(false);
  if (records.length === 0) return null;

  return (
    <section className="flex flex-col gap-2">
      <Button
        variant="ghost"
        size="sm"
        className="text-muted-foreground w-fit px-1"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <ChevronDownIcon className={cn("size-3.5 transition-transform", open && "rotate-180")} />
        History ({records.length})
      </Button>
      {open ? (
        <ul className="divide-border divide-y rounded-lg border">
          {records.map((record) => (
            <li
              key={record.id}
              className="grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-x-4 px-3 py-2.5 text-xs"
            >
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{record.course?.name ?? "Course"}</p>
                <p className="text-muted-foreground truncate">
                  {[
                    `${WORKER_TRAINING_STATUS_LABELS[record.status as WorkerTrainingStatus]}${
                      record.completedAt ? ` ${formatUnixDate(record.completedAt)}` : ""
                    }`,
                    record.expiresAt ? `valid until ${formatUnixDate(record.expiresAt)}` : null,
                    record.waivedReason,
                    record.recordedBy?.name ? `by ${record.recordedBy.name}` : null,
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </p>
              </div>
              <span className="text-muted-foreground tabular-nums">
                {record.score ? `${Number(record.score).toFixed(2)}%` : ""}
              </span>
              <TrainingHealthBadge health={record.health as WorkerTrainingHealth} />
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
