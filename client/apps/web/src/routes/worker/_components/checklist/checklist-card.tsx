import { useT } from "@trenova/shared/i18n/use-t";
import type { WorkerChecklistItemRow, WorkerChecklistRow } from "@/lib/graphql/worker-checklist";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Progress } from "@trenova/shared/components/ui/progress";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  CHECKLIST_KIND_LABELS,
  CHECKLIST_OWNER_LABELS,
  type ChecklistKind,
} from "@trenova/shared/types/worker-checklist";
import { XIcon } from "lucide-react";
import { useMemo } from "react";
import { ChecklistItemRow, type ChecklistItemPermissions } from "./checklist-item-row";
import { groupByOwner } from "./checklist-meta";

type ChecklistCardProps = {
  checklist: WorkerChecklistRow;
  permissions: ChecklistItemPermissions & { canCancel: boolean };
  busyItemId?: string;
  onComplete: (item: WorkerChecklistItemRow) => void;
  onSkip: (item: WorkerChecklistItemRow) => void;
  onNotApplicable: (item: WorkerChecklistItemRow) => void;
  onReopen: (item: WorkerChecklistItemRow) => void;
  onCancel: (checklist: WorkerChecklistRow) => void;
};

/**
 * One checklist: a figure for how far along it is, the counts behind the
 * figure, and the items grouped by whoever owns them. The status badge is the
 * only colour; an overdue count is a fact in the counts line, not an alarm.
 */
export function ChecklistCard({
  checklist,
  permissions,
  busyItemId,
  onComplete,
  onSkip,
  onNotApplicable,
  onReopen,
  onCancel,
}: ChecklistCardProps) {
  const t = useT();

  const groups = useMemo(() => groupByOwner(checklist.items), [checklist.items]);
  const { progress } = checklist;
  const open = checklist.status === "Open";

  return (
    <section
      data-testid={`checklist-${checklist.id}`}
      className={cn("flex flex-col overflow-hidden rounded-lg border", !open && "opacity-80")}
    >
      <header className="flex flex-col gap-3 p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-sm font-semibold">{checklist.name}</h3>
              <Badge variant="outline">
                {CHECKLIST_KIND_LABELS[checklist.kind as ChecklistKind] ?? checklist.kind}
              </Badge>
              {checklist.status === "Completed" ? (
                <Badge variant="active">{t("Completed")}</Badge>
              ) : checklist.status === "Cancelled" ? (
                <Badge variant="inactive">{t("Cancelled")}</Badge>
              ) : null}
            </div>
            <p className="text-muted-foreground mt-0.5 text-xs">
              {t("Started {0} {1} {2} {3} {4}", formatUnixDateMedium(checklist.startedAt), checklist.startedBy?.name ? t("by {0}", checklist.startedBy.name) : "", checklist.dueAt && open ? t("· due {0}", formatUnixDateMedium(checklist.dueAt)) : "", checklist.completedAt
                ? t("· completed {0}", formatUnixDateMedium(checklist.completedAt))
                : "", checklist.cancelledAt
                ? t("· cancelled {0}{1}", formatUnixDateMedium(checklist.cancelledAt), checklist.cancelReason ? ` — ${checklist.cancelReason}` : "")
                : "")}
            </p>
          </div>
          {open && permissions.canCancel ? (
            <Button
              size="sm"
              variant="ghost"
              className="text-muted-foreground hover:text-destructive"
              aria-label={`Cancel ${checklist.name}`}
              onClick={() => onCancel(checklist)}
            >
              <XIcon className="size-3.5" />
              {t("Cancel")}
            </Button>
          ) : null}
        </div>

        <div className="flex items-center gap-4">
          <span className="text-2xl leading-none font-semibold tracking-tight tabular-nums">
            {progress.percent}%
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Progress value={progress.percent} className="h-1.5" />
            <p className="text-muted-foreground flex flex-wrap gap-x-3 text-xs tabular-nums">
              <span className="text-foreground font-medium">
                {t("{0}/{1} required", progress.requiredDone, progress.requiredTotal)}
              </span>
              <span>
                {t("{0} of {1} settled", progress.settled, progress.total)}
              </span>
              {progress.overdue > 0 && open ? <span>{t("{0} overdue", progress.overdue)}</span> : null}
            </p>
          </div>
        </div>

        {open && checklist.kind === "Onboarding" ? (
          <Alert>
            <AlertDescription>
              {t("When the last required item is settled this checklist closes on its own and the worker is marked qualified.")}
            </AlertDescription>
          </Alert>
        ) : null}
      </header>

      <div className="divide-border border-border divide-y border-t">
        {groups.map((group) => (
          <div key={group.owner} data-testid={`checklist-owner-${group.owner}`}>
            <p className="text-2xs bg-muted/40 text-muted-foreground px-3 py-1 font-medium uppercase">
              {CHECKLIST_OWNER_LABELS[group.owner] ?? group.owner}
            </p>
            <ul className="divide-border divide-y">
              {group.items.map((item) => (
                <ChecklistItemRow
                  key={item.id}
                  item={item}
                  checklistOpen={open}
                  permissions={permissions}
                  busy={busyItemId === item.id}
                  onComplete={onComplete}
                  onSkip={onSkip}
                  onNotApplicable={onNotApplicable}
                  onReopen={onReopen}
                />
              ))}
            </ul>
          </div>
        ))}
      </div>
    </section>
  );
}
