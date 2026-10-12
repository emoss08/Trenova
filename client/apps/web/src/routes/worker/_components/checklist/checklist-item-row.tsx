import { useT } from "@trenova/shared/i18n/use-t";
import { translate } from "@trenova/shared/i18n/runtime";
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
import {
  CheckIcon,
  RefreshCcw01Icon,
  SkipForwardIcon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
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

const STATUS_BADGE: Record<ChecklistItemStatus, "success" | "warning" | "neutral"> = {
  Pending: "neutral",
  Done: "success",
  Skipped: "warning",
  NotApplicable: "neutral",
};

function settledLine(item: WorkerChecklistItemRow): string | null {
  if (item.status === "Pending") return null;
  if (item.status === "Done" && item.autoCompleted) {
    if (item.evidenceCredential) {
      const credential = item.evidenceCredential.number ?? item.credentialType?.name ?? "";
      return item.evidenceCredential.expiresAt
        ? translate(
            "Satisfied by credential {0} · expires {1}",
            credential,
            formatUnixDateMedium(item.evidenceCredential.expiresAt),
          )
        : translate("Satisfied by credential {0}", credential).trim();
    }
    if (item.evidenceDocument) {
      return translate("Satisfied by document {0}", item.evidenceDocument.originalName);
    }
    return translate("Satisfied automatically");
  }
  const who = item.completedBy?.name;
  const when = item.completedAt ? formatUnixDateMedium(item.completedAt) : null;
  if (item.status === "Done") return completedLine(who, when);
  if (item.status === "Skipped") return skippedLine(who, when);
  return notApplicableLine(who, when);
}

function completedLine(who: string | undefined, when: string | null): string {
  if (who && when) return translate("Completed by {0} on {1}", who, when);
  if (who) return translate("Completed by {0}", who);
  if (when) return translate("Completed by someone on {0}", when);
  return translate("Completed by someone");
}

function skippedLine(who: string | undefined, when: string | null): string {
  if (who && when) return translate("Skipped by {0} on {1}", who, when);
  if (who) return translate("Skipped by {0}", who);
  if (when) return translate("Skipped by someone on {0}", when);
  return translate("Skipped by someone");
}

function notApplicableLine(who: string | undefined, when: string | null): string {
  if (who && when) return translate("Marked not applicable by {0} on {1}", who, when);
  if (who) return translate("Marked not applicable by {0}", who);
  if (when) return translate("Marked not applicable by someone on {0}", when);
  return translate("Marked not applicable by someone");
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
  const t = useT();

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
          label: t("Complete {0}", item.label),
          icon: CheckIcon,
          disabled: busy,
          onSelect: () => onComplete(item),
        });
      }
      actions.push({
        id: "skip",
        label: t("Skip {0}", item.label),
        icon: SkipForwardIcon,
        disabled: busy,
        onSelect: () => onSkip(item),
      });
      actions.push({
        id: "not-applicable",
        label: t("Mark {0} not applicable", item.label),
        icon: SlashCircle01Icon,
        disabled: busy,
        onSelect: () => onNotApplicable(item),
      });
    } else {
      actions.push({
        id: "reopen",
        label: t("Reopen {0}", item.label),
        icon: RefreshCcw01Icon,
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
          <SlashCircle01Icon className="size-3.5" />
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
            {t(item.label)}
          </span>
          {!item.required ? (
            <span className="text-xs text-muted-foreground">{t("Optional")}</span>
          ) : null}
          {auto ? (
            <Badge
              variant="neutral"
              appearance="outline"
              title={t("Completes itself from evidence")}
            >
              {t("Auto")}
            </Badge>
          ) : null}
          {status !== "Pending" ? (
            <Badge variant={STATUS_BADGE[status]}>{CHECKLIST_ITEM_STATUS_LABELS[status]}</Badge>
          ) : null}
          {pending && item.overdue ? <Badge variant="danger">{t("Overdue")}</Badge> : null}
        </div>
        {item.description ? (
          <p className="text-muted-foreground text-xs">{t(item.description)}</p>
        ) : null}
        <p className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-xs">
          <span>{CHECKLIST_ITEM_KIND_LABELS[kind]}</span>
          {pending && item.dueAt ? (
            <span>
              · {item.overdue ? t("Due since") : t("Due")} {formatUnixDateMedium(item.dueAt)}
            </span>
          ) : null}
          {line ? <span>· {line}</span> : null}
        </p>
        {item.note ? <p className="text-muted-foreground text-xs">“{item.note}”</p> : null}
      </div>

      <RowActionsMenu label={t("Actions for {0}", item.label)} actions={actions} />
    </li>
  );
}
