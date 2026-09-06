import type { WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
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
  CheckCircle2Icon,
  FileCheckIcon,
  GavelIcon,
  PencilIcon,
  RotateCcwIcon,
  SearchIcon,
  Trash2Icon,
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

export function SafetyEventRow({
  event,
  permissions,
  busy,
  onEdit,
  onClose,
  onReview,
  onReopen,
  onDelete,
  onDiscipline,
}: SafetyEventRowProps) {
  const status = event.status as SafetyEventStatus;
  const isClosed = status === "Closed";
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

  return (
    <div
      data-testid={`safety-event-${event.id}`}
      className={cn(
        "bg-card flex flex-col gap-2 rounded-xl border p-4",
        !isClosed && "border-amber-500/40",
      )}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm font-semibold">{headline}</p>
          <p className="text-muted-foreground text-[11px]">
            {formatUnixDate(event.occurredAt)}
            {event.location ? ` · ${event.location}` : ""}
            {event.referenceNumber ? ` · ${event.referenceNumber}` : ""}
            {event.recordedBy?.name ? ` · recorded by ${event.recordedBy.name}` : ""}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          {event.activePoints > 0 ? (
            <Badge variant="outline" className="px-1.5 py-0 text-[10px] tabular-nums">
              {event.activePoints} pt{event.activePoints === 1 ? "" : "s"}
            </Badge>
          ) : null}
          <Badge variant={STATUS_VARIANT[status] ?? "outline"}>
            {SAFETY_EVENT_STATUS_LABELS[status] ?? status}
          </Badge>
        </div>
      </div>

      <p className="text-sm">{event.description}</p>

      <div className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-[11px]">
        {event.fineAmount ? <span>Fine ${event.fineAmount}</span> : null}
        {event.costAmount ? <span>Cost ${event.costAmount}</span> : null}
        {event.pointsExpireAt && event.activePoints > 0 ? (
          <span>Points roll off {formatUnixDate(event.pointsExpireAt)}</span>
        ) : null}
        {event.document ? (
          <span className="flex items-center gap-1">
            <FileCheckIcon className="size-3" />
            Document on file
          </span>
        ) : null}
      </div>

      {isClosed && event.resolution ? (
        <p className="bg-muted/40 rounded-md px-2.5 py-1.5 text-xs">
          <span className="font-medium">Resolved: </span>
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

      <div className="flex flex-wrap items-center gap-2">
        {!isClosed && permissions.canClose ? (
          <Button size="sm" disabled={busy} onClick={() => onClose(event)}>
            <CheckCircle2Icon className="size-3.5" />
            Close event
          </Button>
        ) : null}
        {status === "Open" && permissions.canClose ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onReview(event)}>
            <SearchIcon className="size-3.5" />
            Mark under review
          </Button>
        ) : null}
        {isClosed && permissions.canClose ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onReopen(event)}>
            <RotateCcwIcon className="size-3.5" />
            Reopen
          </Button>
        ) : null}
        {permissions.canDiscipline ? (
          <Button
            size="sm"
            variant="outline"
            disabled={busy}
            aria-label={`Issue action for ${headline}`}
            onClick={() => onDiscipline(event)}
          >
            <GavelIcon className="size-3.5" />
            Issue action
          </Button>
        ) : null}
        {permissions.canUpdate ? (
          <Button
            size="sm"
            variant="ghost"
            className="text-muted-foreground"
            disabled={busy}
            aria-label={`Edit ${headline}`}
            onClick={() => onEdit(event)}
          >
            <PencilIcon className="size-3.5" />
            Edit
          </Button>
        ) : null}
        {!isClosed && permissions.canDelete ? (
          <Button
            size="sm"
            variant="ghost"
            className="text-destructive ml-auto"
            disabled={busy}
            aria-label={`Delete ${headline}`}
            onClick={() => onDelete(event)}
          >
            <Trash2Icon className="size-3.5" />
          </Button>
        ) : null}
      </div>
    </div>
  );
}
