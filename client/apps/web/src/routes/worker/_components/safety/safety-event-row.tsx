import { useT } from "@trenova/shared/i18n/use-t";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { describeSafetyEvent } from "@trenova/shared/lib/safety";
import { cn } from "@trenova/shared/lib/utils";
import {
  SAFETY_EVENT_STATUS_LABELS,
  type InspectionResult,
  type SafetyEventKind,
  type SafetyEventStatus,
  type SafetySeverity,
} from "@trenova/shared/types/worker-safety";
import {
  CarFrontIcon,
  CheckCircle2Icon,
  ClipboardCheckIcon,
  EyeIcon,
  FileCheckIcon,
  GavelIcon,
  PencilIcon,
  RotateCcwIcon,
  ScrollTextIcon,
  SearchIcon,
  Trash2Icon,
  ZapIcon,
  type LucideIcon,
} from "lucide-react";
import { ViolationList } from "./violation-list";

export type SafetyEventPermissions = {
  canUpdate: boolean;
  canClose: boolean;
  canDelete: boolean;
  canDiscipline: boolean;
};

type SafetyEventRowProps = {
  event: WorkerSafetyEventRow;
  permissions: SafetyEventPermissions;
  busy: boolean;
  last?: boolean;
  onEdit: (event: WorkerSafetyEventRow) => void;
  onClose: (event: WorkerSafetyEventRow) => void;
  onReview: (event: WorkerSafetyEventRow) => void;
  onReopen: (event: WorkerSafetyEventRow) => void;
  onDelete: (event: WorkerSafetyEventRow) => void;
  onDiscipline: (event: WorkerSafetyEventRow) => void;
};

const STATUS_VARIANT: Record<SafetyEventStatus, "active" | "warning" | "inactive" | "outline"> = {
  Open: "inactive",
  UnderReview: "warning",
  Closed: "outline",
};

const KIND_ICONS: Record<string, LucideIcon> = {
  Accident: CarFrontIcon,
  Incident: ZapIcon,
  NearMiss: EyeIcon,
  Citation: ScrollTextIcon,
  Inspection: ClipboardCheckIcon,
};

/**
 * One event on the timeline. The rail marker carries the kind, the badge
 * carries the status, and everything the office can do lives behind one menu.
 */
export function SafetyEventRow({
  event,
  permissions,
  busy,
  last = false,
  onEdit,
  onClose,
  onReview,
  onReopen,
  onDelete,
  onDiscipline,
}: SafetyEventRowProps) {
  const t = useT();

  const status = event.status as SafetyEventStatus;
  const isClosed = status === "Closed";
  const Icon = KIND_ICONS[event.kind] ?? ZapIcon;
  // A crash has one BASIC and never needs choosing; an inspection or a citation
  // is where the codes actually live, so only those two carry the list.
  const suggestedBasic =
    event.kind === "Citation"
      ? "UnsafeDriving"
      : event.inspectionResult === "Pass"
        ? null
        : "VehicleMaintenance";
  const headline = describeSafetyEvent({
    kind: event.kind as SafetyEventKind,
    severity: event.severity as SafetySeverity,
    preventable: event.preventable,
    inspectionResult: (event.inspectionResult ?? null) as InspectionResult | null,
    inspectionLevel: event.inspectionLevel ?? null,
  });

  const actions: RowAction[] = [];
  if (!isClosed && permissions.canClose) {
    actions.push({
      id: "close",
      label: t("Close event"),
      icon: CheckCircle2Icon,
      disabled: busy,
      onSelect: () => onClose(event),
    });
  }
  if (status === "Open" && permissions.canClose) {
    actions.push({
      id: "review",
      label: t("Mark under review"),
      icon: SearchIcon,
      disabled: busy,
      onSelect: () => onReview(event),
    });
  }
  if (isClosed && permissions.canClose) {
    actions.push({
      id: "reopen",
      label: t("Reopen"),
      icon: RotateCcwIcon,
      disabled: busy,
      onSelect: () => onReopen(event),
    });
  }
  if (permissions.canDiscipline) {
    actions.push({
      id: "discipline",
      label: `Issue action for ${headline}`,
      icon: GavelIcon,
      disabled: busy,
      onSelect: () => onDiscipline(event),
    });
  }
  if (permissions.canUpdate) {
    actions.push({
      id: "edit",
      label: `Edit ${headline}`,
      icon: PencilIcon,
      disabled: busy,
      onSelect: () => onEdit(event),
    });
  }
  if (!isClosed && permissions.canDelete) {
    actions.push({
      id: "delete",
      label: `Delete ${headline}`,
      icon: Trash2Icon,
      disabled: busy,
      destructive: true,
      onSelect: () => onDelete(event),
    });
  }

  return (
    <li data-testid={`safety-event-${event.id}`} className="relative flex gap-3">
      <div className="flex flex-col items-center">
        <span
          className={cn(
            "inline-flex size-7 shrink-0 items-center justify-center rounded-full border",
            isClosed ? "bg-background text-muted-foreground" : "bg-accent text-foreground",
          )}
          aria-hidden
        >
          <Icon className="size-3.5" />
        </span>
        {!last ? <span className="bg-border my-1 w-px flex-1" aria-hidden /> : null}
      </div>

      <div className={cn("flex min-w-0 flex-1 flex-col gap-2 pb-5", last && "pb-0")}>
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <p className={cn("text-sm font-medium", isClosed && "text-muted-foreground")}>
                {headline}
              </p>
              <Badge variant={STATUS_VARIANT[status] ?? "outline"}>
                {SAFETY_EVENT_STATUS_LABELS[status] ?? status}
              </Badge>
              {event.activePoints > 0 ? (
                <Badge variant="outline" className="tabular-nums">
                  {t("{0, plural, one {# pt} other {# pts}}", event.activePoints)}
                </Badge>
              ) : null}
            </div>
            <p className="text-muted-foreground text-xs">
              {formatUnixDate(event.occurredAt)}
              {event.location ? ` · ${event.location}` : ""}
              {event.referenceNumber ? ` · ${event.referenceNumber}` : ""}
              {event.recordedBy?.name ? ` ${t("· recorded by {0}", event.recordedBy.name)}` : ""}
            </p>
          </div>
          <RowActionsMenu label={`Actions for ${headline}`} actions={actions} />
        </div>

        <p className="text-sm">{t(event.description)}</p>

        {event.fineAmount || event.costAmount || event.pointsExpireAt || event.document ? (
          <p className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-xs">
            {event.fineAmount ? <span>{t("Fine ${0}", event.fineAmount)}</span> : null}
            {event.costAmount ? <span>{t("Cost ${0}", event.costAmount)}</span> : null}
            {event.pointsExpireAt && event.activePoints > 0 ? (
              <span>{t("Points roll off {0}", formatUnixDate(event.pointsExpireAt))}</span>
            ) : null}
            {event.document ? (
              <span className="flex items-center gap-1">
                <FileCheckIcon className="size-3" />
                {t("Document on file")}
              </span>
            ) : null}
          </p>
        ) : null}

        {isClosed && event.resolution ? (
          <p className="bg-muted/30 rounded-lg border px-3 py-2 text-xs">
            <span className="font-medium">{t("Resolved:")} </span>
            {event.resolution}
            {event.closedBy?.name ? (
              <span className="text-muted-foreground">
                {" "}
                — {event.closedBy.name}
                {event.closedAt ? `, ${formatUnixDate(event.closedAt)}` : ""}
              </span>
            ) : null}
          </p>
        ) : null}

        {event.kind === "Inspection" || event.kind === "Citation" ? (
          <ViolationList safetyEventId={event.id} suggestedBasic={suggestedBasic} />
        ) : null}
      </div>
    </li>
  );
}
