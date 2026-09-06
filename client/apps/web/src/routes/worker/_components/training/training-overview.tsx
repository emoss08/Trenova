import type { WorkerTrainingSummary } from "@/lib/graphql/worker-training";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { RingGauge, type RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
import { trainingProgress } from "@trenova/shared/lib/training";
import { cn } from "@trenova/shared/lib/utils";
import { ListChecksIcon, PlusIcon } from "lucide-react";
import { useMemo } from "react";

type TrainingOverviewProps = {
  summary: WorkerTrainingSummary;
  canAssign: boolean;
  gapCount: number;
  assigningRequired: boolean;
  onAssign: () => void;
  onAssignRequired: () => void;
};

export function TrainingOverview({
  summary,
  canAssign,
  gapCount,
  assigningRequired,
  onAssign,
  onAssignRequired,
}: TrainingOverviewProps) {
  const progress = useMemo(() => trainingProgress(summary.items), [summary.items]);
  const tone: RingGaugeTone = !summary.compliant
    ? "critical"
    : summary.dueCount + summary.expiringCount > 0
      ? "warning"
      : "success";

  return (
    <div
      data-testid="training-overview"
      className="bg-card border-border flex flex-wrap items-center gap-4 rounded-xl border p-4"
    >
      <RingGauge
        value={progress.ratio}
        size={72}
        strokeWidth={7}
        tone={tone}
        aria-label="Required training"
      >
        <span className="text-sm font-semibold tabular-nums">
          {progress.satisfied}/{progress.required}
        </span>
      </RingGauge>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold">Training matrix</h3>
          <Badge variant={summary.compliant ? "active" : "inactive"}>
            {summary.compliant ? "Qualified" : "Not qualified"}
          </Badge>
        </div>
        <p className="text-muted-foreground text-xs">
          {summary.requiredCount === 0
            ? "No courses are required for this worker's driver type."
            : `${progress.satisfied} of ${summary.requiredCount} required courses are current.`}
        </p>
        <div className="mt-1 flex flex-wrap gap-2">
          <StatTile label="Due" value={summary.dueCount} tone="info" />
          <StatTile label="Overdue" value={summary.overdueCount} tone="critical" />
          <StatTile label="Expiring" value={summary.expiringCount} tone="warning" />
          <StatTile label="Expired" value={summary.expiredCount} tone="critical" />
          <StatTile label="Not assigned" value={summary.missingCount} tone="muted" />
        </div>
      </div>

      {canAssign ? (
        <div className="flex flex-col gap-2 sm:flex-row">
          {gapCount > 0 ? (
            <Button
              size="sm"
              variant="outline"
              isLoading={assigningRequired}
              loadingText="Assigning..."
              onClick={onAssignRequired}
            >
              <ListChecksIcon className="size-3.5" />
              Assign required
            </Button>
          ) : null}
          <Button size="sm" onClick={onAssign}>
            <PlusIcon className="size-3.5" />
            Assign course
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function StatTile({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "info" | "warning" | "critical" | "muted";
}) {
  const active = value > 0;
  return (
    <div
      className={cn(
        "flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs",
        !active && "text-muted-foreground border-dashed",
        active &&
          tone === "info" &&
          "border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-400",
        active &&
          tone === "warning" &&
          "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-400",
        active &&
          tone === "critical" &&
          "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-400",
        active && tone === "muted" && "border-border bg-muted/50",
      )}
    >
      <span className="font-semibold tabular-nums">{value}</span>
      <span>{label}</span>
    </div>
  );
}
