import { InfoPopover } from "@/components/info-popover";
import type { WorkerTrainingSummary } from "@/lib/graphql/worker-training";
import { ListChecksIcon, PlusIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { trainingProgress } from "@trenova/shared/lib/training";
import { cn } from "@trenova/shared/lib/utils";

type TrainingOverviewProps = {
  summary: WorkerTrainingSummary;
  canAssign: boolean;
  gapCount: number;
  assigningRequired: boolean;
  onAssign: () => void;
  onAssignRequired: () => void;
};

/**
 * The training standing in one row: how much of the required matrix is current, the counts
 * behind it, the qualification badge, and the two ways to assign.
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
  const progress = trainingProgress(summary.items);
  const dueSoon = summary.dueCount + summary.expiringCount;
  const count = (value: number, text: string, tone: string) => (
    <span className={cn(value > 0 && tone)}>{text}</span>
  );

  return (
    <section
      data-testid="training-overview"
      aria-label={t("Training standing")}
      className="flex flex-wrap items-center gap-x-5 gap-y-2 rounded-lg border bg-card px-4 py-3"
    >
      <div className="flex items-baseline gap-1.5">
        <span className="text-2xl leading-none font-semibold tracking-tight tabular-nums">
          {t("{0} of {1}", progress.satisfied, progress.required)}
        </span>
        <span className="text-sm text-muted-foreground">{t("required current")}</span>
        <InfoPopover title={t("Training standing")}>
          <p>
            {t(
              "The required courses come from the driver type's course matrix; a required course that was never assigned shows as not assigned. Current and expiring soon both count as current.",
            )}
          </p>
          <p>
            {t(
              "The worker reads as not qualified while any required course is missing, failed, expired or overdue.",
            )}
          </p>
        </InfoPopover>
      </div>
      <p className="text-xs text-muted-foreground tabular-nums">
        {count(dueSoon, t("{0} due soon", dueSoon), "text-warning-foreground")}
        {" · "}
        {count(
          summary.overdueCount,
          t("{0} overdue", summary.overdueCount),
          "text-danger-foreground",
        )}
        {" · "}
        {count(
          summary.expiredCount,
          t("{0} expired", summary.expiredCount),
          "text-danger-foreground",
        )}
        {summary.missingCount > 0 ? (
          <>
            {" · "}
            <span className="text-danger-foreground">
              {t("{0} not assigned", summary.missingCount)}
            </span>
          </>
        ) : null}
      </p>
      <Badge variant={summary.compliant ? "success" : "danger"} dot>
        {summary.compliant ? t("Qualified") : t("Not qualified")}
      </Badge>
      {canAssign ? (
        <div className="ml-auto flex items-center gap-2">
          {gapCount > 0 ? (
            <Button
              size="sm"
              variant="secondary"
              className="animate-expand-in motion-reduce:animate-none"
              isLoading={assigningRequired}
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
    </section>
  );
}
