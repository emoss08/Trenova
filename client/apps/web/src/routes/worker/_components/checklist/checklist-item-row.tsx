import type { WorkerChecklistItemRow } from "@/lib/graphql/worker-checklist";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  CHECKLIST_ITEM_KIND_LABELS,
  CHECKLIST_ITEM_STATUS_LABELS,
  type ChecklistItemKind,
  type ChecklistItemStatus,
} from "@trenova/shared/types/worker-checklist";
import { CheckIcon, CircleSlashIcon, RotateCcwIcon, SkipForwardIcon } from "lucide-react";
import { CHECKLIST_ITEM_KIND_ICONS, isAutoSatisfied } from "./checklist-meta";

export type ChecklistItemPermissions = { canUpdate: boolean };

type ChecklistItemRowProps = {
  item: WorkerChecklistItemRow;
  checklistOpen: boolean;
  permissions: ChecklistItemPermissions;
  busy: boolean;
  onComplete: (item: WorkerChecklistItemRow) => void;
  onSkip: (item: WorkerChecklistItemRow) => void;
  onNotApplicable: (item: WorkerChecklistItemRow) => void;
  onReopen: (item: WorkerChecklistItemRow) => void;
};

const STATUS_BADGE: Record<ChecklistItemStatus, "active" | "warning" | "outline" | "secondary"> = {
  Pending: "outline",
  Done: "active",
  Skipped: "warning",
  NotApplicable: "secondary",
};

function settledLine(item: WorkerChecklistItemRow): string | null {
  if (item.status === "Pending") return null;
  if (item.status === "Done" && item.autoCompleted) {
    if (item.evidenceCredential) {
      const expires = item.evidenceCredential.expiresAt
        ? ` · expires ${formatUnixDateMedium(item.evidenceCredential.expiresAt)}`
        : "";
      return `Satisfied by credential ${item.evidenceCredential.number ?? item.credentialType?.name ?? ""}${expires}`.trim();
    }
    if (item.evidenceDocument) {
      return `Satisfied by document ${item.evidenceDocument.originalName}`;
    }
    return "Satisfied automatically";
  }
  const who = item.completedBy?.name ?? "someone";
  const when = item.completedAt ? ` on ${formatUnixDateMedium(item.completedAt)}` : "";
  const verb =
    item.status === "Done"
      ? "Completed by"
      : item.status === "Skipped"
        ? "Skipped by"
        : "Marked not applicable by";
  return `${verb} ${who}${when}`;
}

export function ChecklistItemRow({
  item,
  checklistOpen,
  permissions,
  busy,
  onComplete,
  onSkip,
  onNotApplicable,
  onReopen,
}: ChecklistItemRowProps) {
  const status = item.status as ChecklistItemStatus;
  const kind = item.kind as ChecklistItemKind;
  const Icon = CHECKLIST_ITEM_KIND_ICONS[kind];
  const auto = isAutoSatisfied(kind);
  const pending = status === "Pending";
  const showActions = checklistOpen && permissions.canUpdate;
  const line = settledLine(item);

  return (
    <li
      data-testid={`checklist-item-${item.id}`}
      data-status={status}
      data-overdue={item.overdue ? "true" : "false"}
      className={cn(
        "group flex items-start gap-3 px-3 py-2.5",
        !pending && "opacity-80",
        item.overdue && "bg-red-500/5",
      )}
    >
      <span
        className={cn(
          "mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full border",
          status === "Done" &&
            "border-green-500/40 bg-green-500/15 text-green-700 dark:text-green-400",
          status === "Skipped" &&
            "border-amber-500/40 bg-amber-500/15 text-amber-700 dark:text-amber-400",
          status === "NotApplicable" && "border-border bg-muted text-muted-foreground",
          pending && "border-border text-muted-foreground",
        )}
        aria-hidden
      >
        {status === "Done" ? (
          <CheckIcon className="size-3.5" />
        ) : status === "Skipped" ? (
          <SkipForwardIcon className="size-3.5" />
        ) : status === "NotApplicable" ? (
          <CircleSlashIcon className="size-3.5" />
        ) : (
          <Icon className="size-3.5" />
        )}
      </span>

      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex flex-wrap items-center gap-1.5">
          <span
            className={cn(
              "text-sm font-medium",
              status !== "Pending" && status !== "Done" && "line-through",
            )}
          >
            {item.label}
          </span>
          {!item.required ? (
            <span className="text-muted-foreground text-[10px] uppercase">Optional</span>
          ) : null}
          {auto ? (
            <Badge
              variant="info"
              className="px-1.5 py-0 text-[10px]"
              title="Completes itself from evidence"
            >
              Auto
            </Badge>
          ) : null}
          {status !== "Pending" ? (
            <Badge variant={STATUS_BADGE[status]} className="px-1.5 py-0 text-[10px]">
              {CHECKLIST_ITEM_STATUS_LABELS[status]}
            </Badge>
          ) : null}
        </div>
        {item.description ? (
          <p className="text-muted-foreground text-xs">{item.description}</p>
        ) : null}
        <p className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-[11px]">
          <span>{CHECKLIST_ITEM_KIND_LABELS[kind]}</span>
          {pending && item.dueAt ? (
            <span className={cn(item.overdue && "font-medium text-red-600 dark:text-red-400")}>
              · {item.overdue ? "Overdue since" : "Due"} {formatUnixDateMedium(item.dueAt)}
            </span>
          ) : null}
          {line ? <span>· {line}</span> : null}
        </p>
        {item.note ? <p className="text-xs italic">{item.note}</p> : null}
      </div>

      {showActions ? (
        <div className="flex shrink-0 items-center gap-0.5 opacity-70 transition-opacity group-hover:opacity-100">
          {pending && !auto ? (
            <Button
              size="sm"
              variant="outline"
              className="h-7"
              aria-label={`Complete ${item.label}`}
              disabled={busy}
              onClick={() => onComplete(item)}
            >
              <CheckIcon className="size-3.5" />
              Done
            </Button>
          ) : null}
          {pending ? (
            <>
              <Button
                size="icon"
                variant="ghost"
                className="size-7"
                aria-label={`Skip ${item.label}`}
                title="Skip"
                disabled={busy}
                onClick={() => onSkip(item)}
              >
                <SkipForwardIcon className="size-3.5" />
              </Button>
              <Button
                size="icon"
                variant="ghost"
                className="size-7"
                aria-label={`Mark ${item.label} not applicable`}
                title="Not applicable"
                disabled={busy}
                onClick={() => onNotApplicable(item)}
              >
                <CircleSlashIcon className="size-3.5" />
              </Button>
            </>
          ) : (
            <Button
              size="icon"
              variant="ghost"
              className="size-7"
              aria-label={`Reopen ${item.label}`}
              title="Reopen"
              disabled={busy}
              onClick={() => onReopen(item)}
            >
              <RotateCcwIcon className="size-3.5" />
            </Button>
          )}
        </div>
      ) : null}
    </li>
  );
}
