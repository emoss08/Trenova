import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import type { WorkerTrainingSummary } from "@/lib/graphql/worker-training";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { trainingProgress } from "@trenova/shared/lib/training";
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

/**
 * The roll-up. The figure that matters is how much of the required matrix is
 * current; the rest are counts to scan, so they are plain numbers and the
 * qualification badge is the only colour.
 */
export function TrainingOverview({
  summary,
  canAssign,
  gapCount,
  assigningRequired,
  onAssign,
  onAssignRequired,
}: TrainingOverviewProps) {
  const t = useT();

  const progress = useMemo(() => trainingProgress(summary.items), [summary.items]);

  return (
    <div data-testid="training-overview" className="flex flex-col gap-4 rounded-lg border p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="text-sm font-semibold">{t("Training matrix")}</h3>
          <Badge variant={summary.compliant ? "active" : "inactive"}>
            {summary.compliant ? t("Qualified") : t("Not qualified")}
          </Badge>
          <InfoPopover title={t("Training matrix")}>
            <p>
              {t(
                "The required courses come from the driver type's course matrix; a required course that was never assigned shows as Missing. One record speaks for each course: an open assignment first, otherwise the strongest closed record, with a completion or waiver outranking a lapsed one and a lapsed one outranking a failure.",
              )}
            </p>
            <p>
              {t(
                "Current and Expiring soon both count as current. The worker reads as not qualified while any required course is missing, failed, expired or overdue.",
              )}
            </p>
          </InfoPopover>
        </div>
        {canAssign ? (
          <div className="flex items-center gap-2">
            {gapCount > 0 ? (
              <Button
                size="sm"
                variant="outline"
                isLoading={assigningRequired}
                loadingText={t("Assigning...")}
                onClick={onAssignRequired}
              >
                <ListChecksIcon className="size-3.5" />
                {t("Assign required")}
              </Button>
            ) : null}
            <Button size="sm" onClick={onAssign}>
              <PlusIcon className="size-3.5" />
              {t("Assign course")}
            </Button>
          </div>
        ) : null}
      </div>

      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-baseline gap-1">
            <span className="text-3xl leading-none font-semibold tracking-tight tabular-nums">
              {progress.satisfied}/{progress.required}
            </span>
            <span className="text-muted-foreground text-xs">{t("required current")}</span>
          </div>
          <p className="text-muted-foreground mt-1 text-xs">
            {summary.requiredCount === 0
              ? t("No courses are required for this worker's driver type.")
              : t(
                  "{0} of {1} required courses are current.",
                  progress.satisfied,
                  summary.requiredCount,
                )}
          </p>
        </div>
        <dl className="grid grid-cols-5 gap-x-5 text-xs">
          <Count label={t("Due")} value={summary.dueCount} />
          <Count label={t("Overdue")} value={summary.overdueCount} />
          <Count label={t("Expiring")} value={summary.expiringCount} />
          <Count label={t("Expired")} value={summary.expiredCount} />
          <Count label={t("Not assigned")} value={summary.missingCount} />
        </dl>
      </div>
    </div>
  );
}

function Count({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex flex-col">
      <dt className="text-2xs text-muted-foreground whitespace-nowrap uppercase">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}
