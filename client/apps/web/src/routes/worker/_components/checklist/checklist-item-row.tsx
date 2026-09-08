import type { WorkerChecklistItemRow } from "@/lib/graphql/worker-checklist";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { Badge } from "@trenova/shared/components/ui/badge";
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

/**
 * One item in the file. A settled row goes quiet rather than green; the badge
 * says how it was settled, and the marker circle only ever carries the kind
 * icon or a tick.
 */
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
  const line = settledLine(item);

  const actions: RowAction[] = [];
  if (checklistOpen && permissions.canUpdate) {
    if (pending) {
      if (!auto) {
        actions.push({
          id: "complete",
          label: `Complete ${item.label}`,
          icon: CheckIcon,
          disabled: busy,
          onSelect: () => onComplete(item),
        });
      }
      actions.push({
        id: "skip",
        label: `Skip ${item.label}`,
        icon: SkipForwardIcon,
        disabled: busy,
        onSelect: () => onSkip(item),
      });
      actions.push({
        id: "not-applicable",
        label: `Mark ${item.label} not applicable`,
        icon: CircleSlashIcon,
        disabled: busy,
        onSelect: () => onNotApplicable(item),
      });
    } else {
      actions.push({
        id: "reopen",
        label: `Reopen ${item.label}`,
        icon: RotateCcwIcon,
        disabled: busy,
        onSelect: () => onReopen(item),
      });
    }
  }

  return (
    <li
      data-testid={`checklist-item-${item.id}`}
      data-status={status}
      data-overdue={item.overdue ? "true" : "false"}
      className={cn(
        "group hover:bg-muted/30 flex items-start gap-3 px-3 py-2.5 transition-colors",
        !pending && "opacity-75",
      )}
    >
      <span
        className={cn(
          "mt-0.5 inline-flex size-6 shrink-0 items-center justify-center rounded-md",
          status === "Done" ? "bg-primary text-primary-foreground" : "bg-accent",
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
            <span className="text-2xs text-muted-foreground uppercase">Optional</span>
          ) : null}
          {auto ? (
            <Badge variant="outline" title="Completes itself from evidence">
              Auto
            </Badge>
          ) : null}
          {status !== "Pending" ? (
            <Badge variant={STATUS_BADGE[status]}>{CHECKLIST_ITEM_STATUS_LABELS[status]}</Badge>
          ) : null}
          {pending && item.overdue ? <Badge variant="inactive">Overdue</Badge> : null}
        </div>
        {item.description ? (
          <p className="text-muted-foreground text-xs">{item.description}</p>
        ) : null}
        <p className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-[11px]">
          <span>{CHECKLIST_ITEM_KIND_LABELS[kind]}</span>
          {pending && item.dueAt ? (
            <span>
              · {item.overdue ? "Due since" : "Due"} {formatUnixDateMedium(item.dueAt)}
            </span>
          ) : null}
          {line ? <span>· {line}</span> : null}
        </p>
        {item.note ? <p className="text-muted-foreground text-xs">“{item.note}”</p> : null}
      </div>

      <RowActionsMenu label={`Actions for ${item.label}`} actions={actions} />
    </li>
  );
}
