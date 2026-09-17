import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { canAcknowledgeEvent, canResolveEvent } from "@/lib/carrier-intelligence";
import {
  acknowledgeCarrierIntelEvents,
  type CarrierIntelEvent,
} from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelSeverity } from "@trenova/graphql/generated/graphql";
import { Button } from "@trenova/shared/components/ui/button";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, CheckCheckIcon, CheckIcon, InboxIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { EventStatusBadge } from "./event-status-badge";
import { ResolveEventDialog } from "./resolve-event-dialog";
import { SeverityBadge } from "./severity-badge";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

export type EventTimelineProps = {
  events: readonly CarrierIntelEvent[];
  canUpdate: boolean;
  onChanged?: () => void;
  showSubject?: boolean;
  emptyMessage?: string;
  className?: string;
};

const DOT_CLASSES: Record<CarrierIntelSeverity, string> = {
  Critical: "bg-red-600",
  High: "bg-orange-500",
  Medium: "bg-yellow-500",
  Low: "bg-blue-500",
  Info: "bg-muted-foreground",
};

type EventRowProps = {
  event: CarrierIntelEvent;
  now: number;
  canUpdate: boolean;
  showSubject: boolean;
  acknowledging: boolean;
  onAcknowledge: (event: CarrierIntelEvent) => void;
  onResolve: (event: CarrierIntelEvent) => void;
};

function EventRow({
  event,
  now,
  canUpdate,
  showSubject,
  acknowledging,
  onAcknowledge,
  onResolve,
}: EventRowProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const closed = event.status === "Resolved" || event.status === "Dismissed";

  return (
    <li className="group relative flex gap-3 pb-4 pl-1 last:pb-0" data-event-status={event.status}>
      <span
        aria-hidden
        className="bg-border absolute top-4 bottom-0 left-[8px] w-px group-last:hidden"
      />
      <span
        aria-hidden
        className={cn(
          "ring-background relative mt-1.5 size-2.5 shrink-0 rounded-full ring-4",
          closed ? "bg-muted-foreground/40" : DOT_CLASSES[event.severity],
        )}
      />
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-1.5">
          <SeverityBadge severity={event.severity} />
          <EventStatusBadge status={event.status} />
          <span className="text-muted-foreground text-xs">
            {labels.section[event.category]} · {labels.eventSource[event.source]}
          </span>
          <span
            className="text-muted-foreground ml-auto text-xs"
            title={formatUnixDateTimeMedium(event.detectedAt)}
          >
            {formatRelativeTime(event.detectedAt - now)}
          </span>
        </div>
        <p className={cn("text-sm", closed && "text-muted-foreground")}>
          {showSubject && event.subjectName ? (
            <span className="font-medium">{event.subjectName}: </span>
          ) : null}
          {event.summary}
        </p>
        {event.priorValue !== null || event.currentValue !== null ? (
          <p className="flex flex-wrap items-center gap-1.5 text-xs">
            {event.fieldPath ? (
              <span className="text-muted-foreground font-mono">{event.fieldPath}</span>
            ) : null}
            <span className="bg-muted rounded px-1.5 py-0.5 line-through decoration-1">
              {event.priorValue ?? t("empty")}
            </span>
            <ArrowRightIcon className="text-muted-foreground size-3" aria-hidden />
            <span className="bg-muted rounded px-1.5 py-0.5 font-medium">
              {event.currentValue ?? t("empty")}
            </span>
          </p>
        ) : null}
        {event.vendorChangedAt ? (
          <p className="text-muted-foreground text-xs">
            {t("Changed at the source {0}", formatUnixDateTimeMedium(event.vendorChangedAt))}
          </p>
        ) : null}
        {event.status === "Resolved" && event.resolution ? (
          <p className="text-muted-foreground text-xs">
            {event.resolvedAt
              ? t(
                  "Resolved as {0} on {1}",
                  labels.resolution[event.resolution],
                  formatUnixDateTimeMedium(event.resolvedAt),
                )
              : t("Resolved as {0}", labels.resolution[event.resolution])}
            {event.resolutionNote ? ` · ${event.resolutionNote}` : null}
          </p>
        ) : null}
        {event.status === "Acknowledged" && event.acknowledgedAt ? (
          <p className="text-muted-foreground text-xs">
            {t("Acknowledged {0}", formatUnixDateTimeMedium(event.acknowledgedAt))}
          </p>
        ) : null}
        {canUpdate && (canAcknowledgeEvent(event.status) || canResolveEvent(event.status)) ? (
          <div className="flex items-center gap-1.5 pt-0.5">
            {canAcknowledgeEvent(event.status) ? (
              <Button
                type="button"
                size="xs"
                variant="outline"
                isLoading={acknowledging}
                onClick={() => onAcknowledge(event)}
              >
                <CheckIcon />
                {t("Acknowledge")}
              </Button>
            ) : null}
            {canResolveEvent(event.status) ? (
              <Button type="button" size="xs" variant="outline" onClick={() => onResolve(event)}>
                <CheckCheckIcon />
                {t("Resolve")}
              </Button>
            ) : null}
          </div>
        ) : null}
      </div>
    </li>
  );
}

export function EventTimeline({
  events,
  canUpdate,
  onChanged,
  showSubject = false,
  emptyMessage,
  className,
}: EventTimelineProps) {
  const t = useT();
  const now = useNowSeconds();
  const [resolving, setResolving] = useState<CarrierIntelEvent | null>(null);
  const [acknowledgingId, setAcknowledgingId] = useState<string | null>(null);

  const acknowledge = useApiMutation<number, string[]>({
    resourceName: "Carrier intelligence event",
    mutationFn: (ids) => acknowledgeCarrierIntelEvents(ids),
    onSuccess: (count) => {
      toast.success(count > 0 ? t("Event acknowledged") : t("The event was already acknowledged"));
      onChanged?.();
    },
    onSettled: () => setAcknowledgingId(null),
  });

  if (events.length === 0) {
    return (
      <div
        className={cn(
          "text-muted-foreground flex items-center gap-2 rounded-lg border border-dashed px-3 py-4 text-sm",
          className,
        )}
      >
        <InboxIcon className="size-4" aria-hidden />
        {emptyMessage ?? t("No change events recorded.")}
      </div>
    );
  }

  return (
    <>
      <ol className={cn("flex flex-col", className)}>
        {events.map((event) => (
          <EventRow
            key={event.id}
            event={event}
            now={now}
            canUpdate={canUpdate}
            showSubject={showSubject}
            acknowledging={acknowledgingId === event.id}
            onAcknowledge={(target) => {
              setAcknowledgingId(target.id);
              acknowledge.mutate([target.id]);
            }}
            onResolve={setResolving}
          />
        ))}
      </ol>
      <ResolveEventDialog
        event={resolving}
        open={resolving !== null}
        onOpenChange={(open) => {
          if (!open) {
            setResolving(null);
          }
        }}
        onResolved={() => onChanged?.()}
      />
    </>
  );
}
