import type { WorkerChecklistItemRow, WorkerChecklistRow } from "@/lib/graphql/worker-checklist";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { RingGauge, type RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
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

const KIND_BADGE: Record<ChecklistKind, "active" | "orange" | "secondary"> = {
  Onboarding: "active",
  Offboarding: "orange",
  Custom: "secondary",
};

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
  const groups = useMemo(() => groupByOwner(checklist.items), [checklist.items]);
  const { progress } = checklist;
  const open = checklist.status === "Open";
  const tone: RingGaugeTone = !open
    ? "muted"
    : progress.overdue > 0
      ? "critical"
      : progress.complete
        ? "success"
        : "brand";

  return (
    <section
      data-testid={`checklist-${checklist.id}`}
      className={cn(
        "bg-card border-border flex flex-col overflow-hidden rounded-xl border",
        !open && "opacity-80",
      )}
    >
      <header className="flex flex-wrap items-center gap-4 p-4">
        <RingGauge value={progress.percent / 100} size={64} strokeWidth={6} tone={tone}>
          <span className="text-xs font-semibold tabular-nums">{progress.percent}%</span>
        </RingGauge>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-semibold">{checklist.name}</h3>
            <Badge variant={KIND_BADGE[checklist.kind as ChecklistKind] ?? "secondary"}>
              {CHECKLIST_KIND_LABELS[checklist.kind as ChecklistKind] ?? checklist.kind}
            </Badge>
            {checklist.status === "Completed" ? (
              <Badge variant="active">Completed</Badge>
            ) : checklist.status === "Cancelled" ? (
              <Badge variant="inactive">Cancelled</Badge>
            ) : null}
          </div>
          <p className="text-muted-foreground flex flex-wrap gap-x-3 text-xs">
            <span className="font-medium tabular-nums">
              {progress.requiredDone}/{progress.requiredTotal} required
            </span>
            <span>
              {progress.settled} of {progress.total} settled
            </span>
            {progress.overdue > 0 && open ? (
              <span className="font-medium text-red-600 dark:text-red-400">
                {progress.overdue} overdue
              </span>
            ) : null}
          </p>
          <p className="text-muted-foreground text-[11px]">
            Started {formatUnixDateMedium(checklist.startedAt)}
            {checklist.startedBy?.name ? ` by ${checklist.startedBy.name}` : ""}
            {checklist.dueAt && open ? ` · due ${formatUnixDateMedium(checklist.dueAt)}` : ""}
            {checklist.completedAt
              ? ` · completed ${formatUnixDateMedium(checklist.completedAt)}`
              : ""}
            {checklist.cancelledAt
              ? ` · cancelled ${formatUnixDateMedium(checklist.cancelledAt)}${checklist.cancelReason ? ` — ${checklist.cancelReason}` : ""}`
              : ""}
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
            Cancel
          </Button>
        ) : null}
      </header>

      <div className="divide-border border-border divide-y border-t">
        {groups.map((group) => (
          <div key={group.owner} data-testid={`checklist-owner-${group.owner}`}>
            <p className="bg-muted/40 text-muted-foreground px-3 py-1 text-[10px] font-semibold tracking-wide uppercase">
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
