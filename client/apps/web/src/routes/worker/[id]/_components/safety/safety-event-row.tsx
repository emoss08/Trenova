import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import {
  Car01Icon,
  CheckCircleIcon,
  ChevronRightIcon,
  ClipboardCheckIcon,
  Edit02Icon,
  EyeIcon,
  GavelIcon,
  type IconComponent,
  ReceiptIcon,
  RefreshCcw01Icon,
  SearchLgIcon,
  Trash01Icon,
  ZapIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { csaBasicLabel } from "@trenova/shared/lib/csa";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { describeSafetyEvent } from "@trenova/shared/lib/safety";
import { cn } from "@trenova/shared/lib/utils";
import {
  SAFETY_EVENT_STATUS_LABELS,
  type InspectionResult,
  type SafetyEventKind,
  type SafetyEventStatus,
  type SafetySeverity,
} from "@trenova/shared/types/worker-safety";
import { useId, useState } from "react";
import { decimalAmount, eventStatusTone } from "./safety-model";

export type SafetyEventPermissions = {
  canUpdate: boolean;
  canClose: boolean;
  canDelete: boolean;
  canDiscipline: boolean;
};

export type SafetyEventHandlers = {
  onOpen: (event: WorkerSafetyEventRow) => void;
  onEdit: (event: WorkerSafetyEventRow) => void;
  onClose: (event: WorkerSafetyEventRow) => void;
  onReview: (event: WorkerSafetyEventRow) => void;
  onReopen: (event: WorkerSafetyEventRow) => void;
  onDelete: (event: WorkerSafetyEventRow) => void;
  onDiscipline: (event: WorkerSafetyEventRow) => void;
};

const KIND_ICONS: Record<string, IconComponent> = {
  Accident: Car01Icon,
  Incident: ZapIcon,
  NearMiss: EyeIcon,
  Citation: ReceiptIcon,
  Inspection: ClipboardCheckIcon,
};

export function safetyEventHeadline(event: WorkerSafetyEventRow): string {
  return describeSafetyEvent({
    kind: event.kind as SafetyEventKind,
    severity: event.severity as SafetySeverity,
    preventable: event.preventable,
    inspectionResult: (event.inspectionResult ?? null) as InspectionResult | null,
    inspectionLevel: event.inspectionLevel ?? null,
  });
}

export function formatDecimalMoney(amount: number): string {
  return formatNumber(amount, { style: "currency", currency: "USD" });
}

/** The actions an event's state and the reader's grants allow, in the menu's order. */
export function safetyEventActions(
  event: WorkerSafetyEventRow,
  permissions: SafetyEventPermissions,
  handlers: Omit<SafetyEventHandlers, "onOpen">,
  busy: boolean,
  t: (key: string, ...args: unknown[]) => string,
): RowAction[] {
  const status = event.status as SafetyEventStatus;
  const closed = status === "Closed";
  const actions: RowAction[] = [];
  if (!closed && permissions.canClose) {
    actions.push({
      id: "close",
      label: t("Close event"),
      icon: CheckCircleIcon,
      disabled: busy,
      onSelect: () => handlers.onClose(event),
    });
  }
  if (status === "Open" && permissions.canClose) {
    actions.push({
      id: "review",
      label: t("Mark under review"),
      icon: SearchLgIcon,
      disabled: busy,
      onSelect: () => handlers.onReview(event),
    });
  }
  if (closed && permissions.canClose) {
    actions.push({
      id: "reopen",
      label: t("Reopen event"),
      icon: RefreshCcw01Icon,
      disabled: busy,
      onSelect: () => handlers.onReopen(event),
    });
  }
  if (permissions.canDiscipline) {
    actions.push({
      id: "discipline",
      label: t("Issue action for this"),
      icon: GavelIcon,
      disabled: busy,
      onSelect: () => handlers.onDiscipline(event),
    });
  }
  if (permissions.canUpdate) {
    actions.push({
      id: "edit",
      label: t("Edit"),
      icon: Edit02Icon,
      disabled: busy,
      onSelect: () => handlers.onEdit(event),
    });
  }
  if (!closed && permissions.canDelete) {
    actions.push({
      id: "delete",
      label: t("Delete"),
      icon: Trash01Icon,
      disabled: busy,
      destructive: true,
      onSelect: () => handlers.onDelete(event),
    });
  }
  return actions;
}

/**
 * One event: its kind, headline and state, where and when, what happened, what it cost, and how
 * it was resolved. An inspection or citation unfolds in place to the violations it cited; the row
 * itself opens the event beside the list.
 */
export function SafetyEventRow({
  event,
  permissions,
  busy,
  handlers,
}: {
  event: WorkerSafetyEventRow;
  permissions: SafetyEventPermissions;
  busy: boolean;
  handlers: SafetyEventHandlers;
}) {
  const t = useT();
  const violationsId = useId();
  const [expanded, setExpanded] = useState(false);
  const status = event.status as SafetyEventStatus;
  const closed = status === "Closed";
  const Icon = KIND_ICONS[event.kind] ?? ZapIcon;
  const headline = safetyEventHeadline(event);
  const fine = decimalAmount(event.fineAmount);
  const cost = decimalAmount(event.costAmount);
  const violations = event.violations ?? [];
  const rolledOff = event.points > 0 && event.activePoints === 0;
  const meta = [
    formatUnixDateMedium(event.occurredAt),
    event.location,
    event.referenceNumber,
    event.recordedBy?.name ? t("recorded by {0}", event.recordedBy.name) : null,
  ].filter((part): part is string => Boolean(part));

  return (
    <div
      data-glide-target
      data-testid={`safety-event-${event.id}`}
      role="button"
      tabIndex={0}
      aria-label={t("{0}. {1}.", headline, t(SAFETY_EVENT_STATUS_LABELS[status] ?? status))}
      onClick={() => handlers.onOpen(event)}
      onKeyDown={(keyEvent) => {
        if (keyEvent.target !== keyEvent.currentTarget) return;
        if (keyEvent.key === "Enter" || keyEvent.key === " ") {
          keyEvent.preventDefault();
          handlers.onOpen(event);
        }
      }}
      className="grid cursor-pointer grid-cols-[1.75rem_minmax(0,1fr)_auto] items-start gap-3 rounded-md ui-focus-ring px-2 py-2.5 outline-none"
    >
      <span
        aria-hidden
        className="flex size-7 items-center justify-center rounded-md border border-border-subtle bg-sunken text-muted-foreground"
      >
        <Icon className="size-3.5" />
      </span>
      <div className="min-w-0">
        <div className="flex min-h-5 flex-wrap items-center gap-x-2 gap-y-1">
          <span className="font-medium">{headline}</span>
          <Badge variant={eventStatusTone(status)} dot={!closed}>
            {t(SAFETY_EVENT_STATUS_LABELS[status] ?? status)}
          </Badge>
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">{meta.join(" · ")}</p>
        <p className="mt-1 max-w-[72ch] text-pretty">{event.description}</p>
        {fine !== null || cost !== null ? (
          <p className="mt-1 text-xs text-muted-foreground tabular-nums">
            {[
              fine !== null ? t("Fine {0}", formatDecimalMoney(fine)) : null,
              cost !== null ? t("Cost {0}", formatDecimalMoney(cost)) : null,
            ]
              .filter(Boolean)
              .join(" · ")}
          </p>
        ) : null}
        {violations.length > 0 ? (
          <>
            <button
              type="button"
              aria-expanded={expanded}
              aria-controls={violationsId}
              onClick={(clickEvent) => {
                clickEvent.stopPropagation();
                setExpanded((open) => !open);
              }}
              onKeyDown={(keyEvent) => keyEvent.stopPropagation()}
              className="mt-1.5 inline-flex items-center gap-1 rounded-sm ui-focus-ring pr-1 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              <ChevronRightIcon
                className={cn(
                  "size-3 transition-transform duration-200 ease-settle",
                  expanded && "rotate-90",
                )}
              />
              {t("Violations cited")}
              <span className="ml-0.5 font-normal text-foreground-subtle tabular-nums">
                {formatNumber(violations.length)}
              </span>
            </button>
            {expanded ? (
              <div id={violationsId} className="motion-safe:animate-expand-in">
                <ViolationRows violations={violations} />
              </div>
            ) : null}
          </>
        ) : null}
        {closed && event.resolution ? (
          <p className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
            <CheckCircleIcon className="size-3 shrink-0 text-success-foreground" />
            {t("Resolved: {0}", event.resolution)}
          </p>
        ) : null}
      </div>
      <div
        className="flex min-h-5 items-center gap-1"
        onClick={(clickEvent) => clickEvent.stopPropagation()}
        onKeyDown={(keyEvent) => keyEvent.stopPropagation()}
      >
        <span
          className={cn(
            "flex flex-col items-end font-mono text-xs whitespace-nowrap tabular-nums",
            closed && "text-muted-foreground",
          )}
        >
          {t("{0, plural, one {# pt} other {# pts}}", event.activePoints)}
          {rolledOff ? (
            <span className="font-sans text-2xs text-foreground-subtle">{t("rolled off")}</span>
          ) : null}
        </span>
        <RowActionsMenu
          label={t("Actions for {0}", headline)}
          actions={safetyEventActions(event, permissions, handlers, busy, t)}
        />
      </div>
    </div>
  );
}

type ViolationLine = NonNullable<WorkerSafetyEventRow["violations"]>[number];

/** The violations an event cited: BASIC, code, what was cited and its weight. */
export function ViolationRows({ violations }: { violations: readonly ViolationLine[] }) {
  const t = useT();
  return (
    <div className="mt-1.5 divide-y divide-border-subtle rounded-md border border-border-subtle bg-sunken">
      {violations.map((violation) => (
        <div
          key={violation.id}
          className="grid grid-cols-[auto_auto_minmax(0,1fr)_auto] items-center gap-2.5 px-2.5 py-1.5 text-xs"
        >
          <Badge variant="neutral">{csaBasicLabel(violation.basic)}</Badge>
          <span className="font-mono">{violation.code ?? ""}</span>
          <span className="min-w-0">
            {violation.description}
            {violation.outOfService ? (
              <Badge variant="danger" className="ml-1.5">
                {t("Out of service")}
              </Badge>
            ) : null}
          </span>
          <span className="whitespace-nowrap text-muted-foreground tabular-nums">
            {t("weight {0}", violation.severityWeight)}
          </span>
        </div>
      ))}
    </div>
  );
}
