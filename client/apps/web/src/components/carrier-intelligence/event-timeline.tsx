import { useApiMutation } from "@/hooks/use-api-mutation";
import { canAcknowledgeEvent, canResolveEvent } from "@/lib/carrier-intelligence";
import {
  acknowledgeCarrierIntelEvents,
  type CarrierIntelEvent,
} from "@/lib/graphql/carrier-intelligence";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";
import { RelativeTime } from "./relative-time";
import { ResolveEventDialog } from "./resolve-event-dialog";
import { StatusDot, severityTone } from "./status-dot";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";
import { useEventPresenter } from "./use-event-presenter";

export type EventTimelineProps = {
  events: readonly CarrierIntelEvent[];
  canUpdate: boolean;
  onChanged?: () => void;
  emptyMessage?: ReactNode;
  className?: string;
};

type EventRowProps = {
  event: CarrierIntelEvent;
  title: string;
  canUpdate: boolean;
  acknowledging: boolean;
  onAcknowledge: (event: CarrierIntelEvent) => void;
  onResolve: (event: CarrierIntelEvent) => void;
};

function EventRow({
  event,
  title,
  canUpdate,
  acknowledging,
  onAcknowledge,
  onResolve,
}: EventRowProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const settled = event.status === "Resolved" || event.status === "Dismissed";
  const showAcknowledge = canUpdate && canAcknowledgeEvent(event.status);
  const showResolve = canUpdate && canResolveEvent(event.status);

  const resolver = event.resolvedBy?.name;
  const acknowledger = event.acknowledgedBy?.name;

  const outcome =
    event.status === "Resolved" && event.resolution
      ? [
          event.resolvedAt
            ? resolver
              ? t(
                  "Resolved as {0} by {1} on {2}",
                  labels.resolution[event.resolution],
                  resolver,
                  formatUnixDateTimeMedium(event.resolvedAt),
                )
              : t(
                  "Resolved as {0} on {1}",
                  labels.resolution[event.resolution],
                  formatUnixDateTimeMedium(event.resolvedAt),
                )
            : t("Resolved as {0}", labels.resolution[event.resolution]),
          event.resolutionNote,
        ]
          .filter(Boolean)
          .join(" · ")
      : event.status === "Acknowledged" && event.acknowledgedAt
        ? acknowledger
          ? t(
              "Acknowledged by {0} on {1}",
              acknowledger,
              formatUnixDateTimeMedium(event.acknowledgedAt),
            )
          : t("Acknowledged {0}", formatUnixDateTimeMedium(event.acknowledgedAt))
        : null;

  return (
    <li
      className="group flex min-h-14 items-start gap-3 py-2.5"
      data-event-status={event.status}
      data-event-id={event.id}
    >
      <span className="flex h-5 shrink-0 items-center">
        <StatusDot tone={settled ? "neutral" : severityTone(event.severity)} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span
          className={cn(
            "text-sm",
            settled ? "text-muted-foreground" : "text-foreground font-medium",
          )}
        >
          {title}
        </span>
        <span className="text-muted-foreground flex min-w-0 flex-wrap items-center gap-x-1.5 text-xs">
          <span>{labels.severity[event.severity]}</span>
          <span aria-hidden>·</span>
          <span>{labels.section[event.category]}</span>
          <span aria-hidden>·</span>
          <span>{labels.eventStatus[event.status]}</span>
          <span aria-hidden>·</span>
          <RelativeTime timestamp={event.detectedAt} />
        </span>
        {outcome ? <span className="text-muted-foreground text-xs">{outcome}</span> : null}
      </div>
      {showAcknowledge || showResolve ? (
        <div className="flex shrink-0 items-center gap-1">
          {showAcknowledge ? (
            <Button
              type="button"
              size="xs"
              variant="ghost"
              isLoading={acknowledging}
              onClick={() => onAcknowledge(event)}
            >
              {t("Acknowledge")}
            </Button>
          ) : null}
          {showResolve ? (
            <Button type="button" size="xs" variant="ghost" onClick={() => onResolve(event)}>
              {t("Resolve")}
            </Button>
          ) : null}
        </div>
      ) : null}
    </li>
  );
}

export function EventTimeline({
  events,
  canUpdate,
  onChanged,
  emptyMessage,
  className,
}: EventTimelineProps) {
  const t = useT();
  const present = useEventPresenter();
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
      <p className={cn("text-muted-foreground py-6 text-center text-xs", className)}>
        {emptyMessage ?? t("No changes recorded.")}
      </p>
    );
  }

  return (
    <>
      <ol className={cn("divide-border divide-y", className)} aria-label={t("Carrier changes")}>
        {events.map((event) => (
          <EventRow
            key={event.id}
            event={event}
            title={present(event).title}
            canUpdate={canUpdate}
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
        title={resolving ? present(resolving).title : null}
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
