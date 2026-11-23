import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel } from "@/components/section-panel";
import type { WorkerChecklistItemRow, WorkerChecklistRow } from "@/lib/graphql/worker-checklist";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  CHECKLIST_KIND_LABELS,
  CHECKLIST_OWNER_LABELS,
  type ChecklistKind,
} from "@trenova/shared/types/worker-checklist";
import { useEffect, useRef } from "react";
import { ChecklistItemRow, type ChecklistItemPermissions } from "./checklist-item-row";
import { groupByOwner } from "./checklist-meta";

type ChecklistCardProps = {
  checklist: WorkerChecklistRow;
  permissions: ChecklistItemPermissions & { canCancel: boolean };
  busyItemId?: string;
  /** A link sent the reader here (`checklist` in the address). */
  highlighted?: boolean;
  onComplete: (item: WorkerChecklistItemRow) => void;
  onSkip: (item: WorkerChecklistItemRow) => void;
  onNotApplicable: (item: WorkerChecklistItemRow) => void;
  onReopen: (item: WorkerChecklistItemRow) => void;
  onCancel: (checklist: WorkerChecklistRow) => void;
};

function startedLine(checklist: WorkerChecklistRow): string {
  const on = formatUnixDateMedium(checklist.startedAt);
  const by = checklist.startedBy?.name;
  const due = checklist.dueAt ? formatUnixDateMedium(checklist.dueAt) : null;
  if (by && due) return translate("Started {0} by {1} · due {2}", on, by, due);
  if (by) return translate("Started {0} by {1}", on, by);
  if (due) return translate("Started {0} · due {1}", on, due);
  return translate("Started {0}", on);
}

/** One segment per required item: settled in ink, overdue in danger, the rest unfilled. */
export function progressSegments(
  progress: WorkerChecklistRow["progress"],
): ("on" | "late" | "off")[] {
  const total = Math.max(0, progress.requiredTotal);
  return Array.from({ length: total }, (_, index) =>
    index < progress.requiredDone
      ? "on"
      : index < progress.requiredDone + progress.overdue
        ? "late"
        : "off",
  );
}

/**
 * The open checklist: how far along it is as a figure and a bar, the counts behind them, and
 * the items grouped by whoever owns them.
 */
export function ChecklistCard({
  checklist,
  permissions,
  busyItemId,
  highlighted = false,
  onComplete,
  onSkip,
  onNotApplicable,
  onReopen,
  onCancel,
}: ChecklistCardProps) {
  const t = useT();
  const ref = useRef<HTMLDivElement>(null);
  const groups = groupByOwner(checklist.items);
  const { progress } = checklist;
  const open = checklist.status === "Open";
  const segments = progressSegments(progress);

  useEffect(() => {
    if (highlighted) ref.current?.scrollIntoView({ block: "nearest" });
  }, [highlighted]);

  return (
    <div
      ref={ref}
      data-testid={`checklist-${checklist.id}`}
      className={cn("rounded-lg", highlighted && "animate-attention-a")}
    >
      <SectionPanel
        title={checklist.name}
        hint={
          <Badge variant="neutral">
            {CHECKLIST_KIND_LABELS[checklist.kind as ChecklistKind] ?? t("Checklist")}
          </Badge>
        }
        action={
          open && permissions.canCancel ? (
            <Button size="sm" variant="outline" onClick={() => onCancel(checklist)}>
              {t("Cancel checklist")}
            </Button>
          ) : null
        }
      >
        <div className="grid grid-cols-[auto_minmax(0,1fr)] items-end gap-x-4 gap-y-1.5 border-b border-border-subtle px-4 pt-3 pb-3.5">
          <p className="col-span-full text-xs text-muted-foreground tabular-nums">
            {startedLine(checklist)}
          </p>
          <span className="text-3xl leading-none font-semibold tracking-tight tabular-nums">
            {t("{0}%", progress.percent)}
          </span>
          <div className="flex min-w-0 flex-col gap-1.5 pb-0.5">
            <div
              role="img"
              aria-label={t(
                "{0} of {1} required settled",
                progress.requiredDone,
                progress.requiredTotal,
              )}
              className="grid h-2 gap-0.5"
              style={{
                gridTemplateColumns: `repeat(${Math.max(1, segments.length)}, minmax(0, 1fr))`,
              }}
            >
              {segments.map((segment, index) => (
                <span
                  key={index}
                  className={cn(
                    "rounded-xs transition-colors",
                    segment === "on" && "bg-foreground",
                    segment === "late" && "bg-danger",
                    segment === "off" && "bg-surface-active",
                  )}
                />
              ))}
            </div>
            <p className="text-xs text-muted-foreground tabular-nums">
              {t("{0} of {1} required settled", progress.requiredDone, progress.requiredTotal)}
              {progress.overdue > 0 && open ? (
                <>
                  {" · "}
                  <span className="font-medium text-danger-foreground">
                    {t("{0} overdue", progress.overdue)}
                  </span>
                </>
              ) : null}
            </p>
          </div>
        </div>
        <HoverGlideGroup className="px-1.5 pb-1.5">
          {groups.map((group) => {
            const settled = group.items.filter((item) => item.status !== "Pending").length;
            return (
              <div key={group.owner} data-testid={`checklist-owner-${group.owner}`}>
                <p className="px-2 pt-3 pb-1 text-xs font-medium text-foreground-subtle">
                  {CHECKLIST_OWNER_LABELS[group.owner] ?? t("Other")}{" "}
                  <span className="font-normal text-foreground-subtle/80 tabular-nums">
                    {t("{0} of {1}", settled, group.items.length)}
                  </span>
                </p>
                <ul className="flex flex-col gap-px">
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
            );
          })}
        </HoverGlideGroup>
      </SectionPanel>
    </div>
  );
}
