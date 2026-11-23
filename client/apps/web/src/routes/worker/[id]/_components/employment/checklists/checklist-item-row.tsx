import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { WorkerChecklistItemRow } from "@/lib/graphql/worker-checklist";
import {
  CheckCircleIcon,
  RefreshCcw01Icon,
  SkipForwardIcon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  CHECKLIST_ITEM_STATUS_LABELS,
  type ChecklistItemKind,
  type ChecklistItemStatus,
} from "@trenova/shared/types/worker-checklist";
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

/** How an item settled, or what it is waiting on, in one line. */
export function itemLine(item: WorkerChecklistItemRow): string {
  if (item.status === "Pending") {
    if (item.dueAt) {
      return item.overdue
        ? translate("Was due {0}", formatUnixDateMedium(item.dueAt))
        : translate("Due {0}", formatUnixDateMedium(item.dueAt));
    }
    return isAutoSatisfied(item.kind)
      ? translate("Settles itself from the record")
      : translate("Open");
  }
  if (item.status === "Done" && item.autoCompleted) {
    if (item.evidenceCredential) {
      const credential = item.evidenceCredential.number ?? item.credentialType?.name ?? null;
      if (credential && item.evidenceCredential.expiresAt) {
        return translate(
          "Satisfied by credential {0} · expires {1}",
          credential,
          formatUnixDateMedium(item.evidenceCredential.expiresAt),
        );
      }
      return credential
        ? translate("Satisfied by credential {0}", credential)
        : translate("Satisfied by a credential");
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
 * One item: its kind's mark, its label with Auto and Optional, how it settled, its state and
 * a menu. A settled row goes quiet; Overdue is the only alarm.
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
  const Icon = CHECKLIST_ITEM_KIND_ICONS[item.kind as ChecklistItemKind];
  const auto = isAutoSatisfied(item.kind);
  const pending = status === "Pending";
  const overdue = pending && item.overdue;

  const actions: RowAction[] = [];
  if (checklistOpen && permissions.canUpdate) {
    if (pending) {
      if (!auto) {
        actions.push({
          id: "complete",
          label: t("Complete"),
          icon: CheckCircleIcon,
          disabled: busy,
          onSelect: () => onComplete(item),
        });
      }
      actions.push({
        id: "skip",
        label: t("Skip"),
        icon: SkipForwardIcon,
        disabled: busy,
        onSelect: () => onSkip(item),
      });
      actions.push({
        id: "not-applicable",
        label: t("Mark not applicable"),
        icon: SlashCircle01Icon,
        disabled: busy,
        onSelect: () => onNotApplicable(item),
      });
    } else {
      actions.push({
        id: "reopen",
        label: t("Reopen"),
        icon: RefreshCcw01Icon,
        disabled: busy,
        onSelect: () => onReopen(item),
      });
    }
  }

  return (
    <div
      data-glide-target
      data-testid={`checklist-item-${item.id}`}
      data-status={status}
      data-overdue={overdue ? "true" : "false"}
      className="grid grid-cols-[1rem_minmax(0,1fr)_auto_1.75rem] items-start gap-3 rounded-md px-2 py-2"
    >
      {Icon ? <Icon aria-hidden className="mt-0.5 size-3.5 text-muted-foreground" /> : <span />}
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-1.5">
          <span className={cn("text-sm font-medium", !pending && "text-muted-foreground")}>
            {item.label}
          </span>
          {auto ? (
            <Badge variant="neutral" title={t("Completes itself from evidence")}>
              {t("Auto")}
            </Badge>
          ) : null}
          {!item.required ? (
            <span className="text-xs text-foreground-subtle">{t("Optional")}</span>
          ) : null}
        </div>
        <p
          className={cn(
            "mt-0.5 text-xs",
            overdue ? "font-medium text-danger-foreground" : "text-muted-foreground",
          )}
        >
          {itemLine(item)}
        </p>
        {item.note ? (
          <p className="mt-0.5 text-xs text-muted-foreground">{t("Note: {0}", item.note)}</p>
        ) : null}
      </div>
      {overdue ? (
        <Badge variant="danger" dot>
          {t("Overdue")}
        </Badge>
      ) : !pending ? (
        <Badge variant={STATUS_BADGE[status]} dot>
          {CHECKLIST_ITEM_STATUS_LABELS[status]}
        </Badge>
      ) : (
        <span />
      )}
      {busy ? (
        <span className="flex size-7 items-center justify-center">
          <Spinner className="size-3.5" />
        </span>
      ) : (
        <RowActionsMenu label={t("Actions for {0}", item.label)} actions={actions} />
      )}
    </div>
  );
}
