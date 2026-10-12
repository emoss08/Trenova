import { useTrackingEventLabels } from "@/hooks/use-tracking-event-labels";
import { queries } from "@/lib/queries";
import {
  groupTrackingEventsByStop,
  trackingOutcomePhase,
  type StopReportGroup,
  type TrackedStop,
  type TrackingEventRow,
} from "@/lib/tracking-events";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { FormSection } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { stopTypeLabel } from "@trenova/shared/lib/stop-type";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";

type StopLabel = { stop: TrackedStop; label: string };

export function StopReports() {
  const t = useT();
  const { control } = useFormContext<Shipment>();
  const shipmentId = useWatch({ control, name: "id" });
  const moves = useWatch({ control, name: "moves" });

  const { data, isPending, isError, error } = useQuery({
    ...queries.shipment.trackingEvents(shipmentId ?? ""),
    enabled: !!shipmentId,
  });

  const stops = useMemo(() => {
    const labels = new Map<string, StopLabel>();
    (moves ?? []).forEach((move, moveIndex) => {
      (move.stops ?? []).forEach((stop: Stop, stopIndex) => {
        if (!stop.id) {
          return;
        }
        labels.set(stop.id, {
          stop: {
            id: stop.id,
            moveSequence: move.sequence ?? moveIndex,
            sequence: stop.sequence ?? stopIndex,
          },
          label: stopReportLabel(stop, stopIndex),
        });
      });
    });
    return labels;
  }, [moves]);

  const groups = useMemo(
    () =>
      groupTrackingEventsByStop(
        data ?? [],
        [...stops.values()].map((entry) => entry.stop),
      ),
    [data, stops],
  );

  if (!shipmentId) {
    return null;
  }

  return (
    <FormSection
      title={t("Stop reports")}
      description={t(
        "Every arrival and departure reported for this shipment, and whether it set the stop's times",
      )}
    >
      {isPending ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>
            {error.message || t("Stop reports could not be loaded.")}
          </AlertDescription>
        </Alert>
      ) : groups.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          {t("No arrivals or departures have been reported yet.")}
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {groups.map((group) => (
            <StopReportGroupView
              key={group.stopId}
              group={group}
              label={stops.get(group.stopId)?.label ?? t("A stop no longer on this shipment")}
            />
          ))}
        </div>
      )}
    </FormSection>
  );
}

function StopReportGroupView({ group, label }: { group: StopReportGroup; label: string }) {
  return (
    <section className="rounded-md border border-border">
      <h4 className="border-b border-border px-3 py-2 text-sm font-medium">{label}</h4>
      <ul className="divide-y divide-border">
        {group.events.map((event) => (
          <StopReportRow key={event.id} event={event} />
        ))}
      </ul>
    </section>
  );
}

function StopReportRow({ event }: { event: TrackingEventRow }) {
  const t = useT();
  const labels = useTrackingEventLabels();

  return (
    <li className="flex flex-col gap-1 px-3 py-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
        <span className="font-medium">{labels.kind(event.kind)}</span>
        <span>{formatUnixDateTime(event.eventAt)}</span>
        <span className="text-muted-foreground">
          {event.sourceStatus
            ? t("{0} ({1})", labels.source(event.source), event.sourceStatus)
            : labels.source(event.source)}
        </span>
        <Badge variant={phaseTone(trackingOutcomePhase(event.outcome))} className="ml-auto">
          {labels.outcome(event.outcome)}
        </Badge>
      </div>
      {event.outcomeReason ? (
        <p className="text-xs text-muted-foreground">{event.outcomeReason}</p>
      ) : null}
    </li>
  );
}

function stopReportLabel(stop: Stop, index: number): string {
  const place = stop.location?.name || stop.addressLine;
  const position = `${index + 1}. ${stopTypeLabel(stop.type)}`;
  return place ? `${position} · ${place}` : position;
}
