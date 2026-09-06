import type { WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
import { TrainingHealthBadge } from "@trenova/shared/components/training-health-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  WORKER_TRAINING_STATUS_LABELS,
  type WorkerTrainingHealth,
  type WorkerTrainingStatus,
} from "@trenova/shared/types/worker-training";
import { ChevronDownIcon } from "lucide-react";
import { useState } from "react";

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
        <ChevronDownIcon className={`size-3.5 transition-transform ${open ? "rotate-180" : ""}`} />
        History ({records.length})
      </Button>
      {open ? (
        <ul className="divide-border border-border divide-y rounded-lg border">
          {records.map((record) => (
            <li key={record.id} className="flex flex-wrap items-center gap-3 px-3 py-2 text-xs">
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{record.course?.name ?? "Course"}</p>
                <p className="text-muted-foreground">
                  {WORKER_TRAINING_STATUS_LABELS[record.status as WorkerTrainingStatus]}
                  {record.completedAt ? ` ${formatUnixDate(record.completedAt)}` : ""}
                  {record.expiresAt ? ` · valid until ${formatUnixDate(record.expiresAt)}` : ""}
                  {record.waivedReason ? ` · ${record.waivedReason}` : ""}
                  {record.recordedBy?.name ? ` · by ${record.recordedBy.name}` : ""}
                </p>
              </div>
              {record.score ? (
                <span className="tabular-nums">{Number(record.score).toFixed(2)}%</span>
              ) : null}
              <TrainingHealthBadge health={record.health as WorkerTrainingHealth} />
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
