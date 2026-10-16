import type { Journey } from "@/lib/shipment-tracking";
import { dayAndTime } from "@/lib/shipment-tracking";
import { CheckIcon, Truck01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { stopTypeLabel } from "@trenova/shared/lib/stop-type";
import { cn } from "@trenova/shared/lib/utils";
import type { PublicShipmentStop, PublicShipmentView } from "@trenova/shared/types/shipment-tracking";

function stopTime(t: ReturnType<typeof useT>, stop: PublicShipmentStop, now: number) {
  if (stop.departedAt != null) {
    return t("Left {0}", dayAndTime(t, stop.departedAt, now, { inline: true }));
  }
  if (stop.arrivedAt != null) {
    return t("Arrived {0}", dayAndTime(t, stop.arrivedAt, now, { inline: true }));
  }
  if (stop.estimatedArrival != null) {
    return t("Expected {0}", dayAndTime(t, stop.estimatedArrival, now, { inline: true }));
  }
  if (stop.windowStart != null) {
    return t("Due {0}", dayAndTime(t, stop.windowStart, now, { inline: true }));
  }
  return null;
}

/**
 * The route drawn as one line: stops spaced evenly along it, the part driven
 * filled in ink, and the truck where it is now. The line draws itself on load,
 * then the truck lands on it.
 */
export function RouteRibbon({
  view,
  journey,
  now,
}: {
  view: PublicShipmentView;
  journey: Journey;
  now: number;
}) {
  const t = useT();
  const stops = view.stops;
  const last = stops.length - 1;
  if (last < 1) {
    return null;
  }
  const at = (index: number) => `${(index / last) * 100}%`;
  const showTruck = journey.started && !journey.finished && view.status !== "Canceled";
  const whereNow = view.lastLocation
    ? t(
        "Near {0}, {1} · {2}",
        view.lastLocation.city,
        view.lastLocation.stateCode,
        formatSecondsAgo(now - view.lastLocation.reportedAt),
      )
    : journey.atStop
      ? t("At {0}", stops[journey.activeIndex]?.city ?? "")
      : t("On the road to {0}", stops[journey.activeIndex]?.city ?? "");

  return (
    <figure className="relative" aria-label={t("Route")}>
      <div className="relative mx-3 h-24">
        <div className="absolute inset-x-0 top-12 border-t border-dashed border-border" />
        <div
          className="st-fill absolute top-[47px] left-0 h-0.5 rounded-full bg-ink"
          style={{ width: `${journey.progress * 100}%` }}
        />
        {stops.map((stop, index) => {
          const done = stop.departedAt != null || (index === last && stop.arrivedAt != null);
          const current = journey.atStop && journey.activeIndex === index && !done;
          return (
            <span
              key={stop.sequence}
              className={cn(
                "absolute top-12 flex size-4 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full",
                done
                  ? "bg-ink text-background"
                  : current
                    ? "bg-background ring-2 ring-ink"
                    : "bg-background ring-1 ring-border",
              )}
              style={{ left: at(index) }}
            >
              {done && <CheckIcon className="size-2.5" aria-hidden />}
            </span>
          );
        })}
        {showTruck && (
          <div
            className="st-puck absolute top-12 flex -translate-x-1/2 -translate-y-1/2 flex-col items-center"
            style={{ left: `${journey.progress * 100}%` }}
          >
            <span className="absolute bottom-full mb-3 rounded-badge bg-background px-2.5 py-1 text-xs font-medium whitespace-nowrap ring-1 ring-foreground/10">
              {whereNow}
            </span>
            <span className="relative flex size-9 items-center justify-center">
              <span className="st-halo absolute inset-0 animate-live-ping rounded-full" />
              <span className="relative flex size-9 items-center justify-center rounded-full bg-ink text-background ring-4 ring-canvas">
                <Truck01Icon className="size-4" aria-hidden />
              </span>
            </span>
          </div>
        )}
      </div>
      <ol className="relative mx-3 mt-1 h-12 sm:h-16">
        {stops.map((stop, index) => {
          const time = stopTime(t, stop, now);
          const align =
            index === 0
              ? "left-0 items-start text-left"
              : index === last
                ? "right-0 items-end text-right"
                : "-translate-x-1/2 items-center text-center";
          return (
            <li
              key={stop.sequence}
              className={cn(
                "absolute top-0 flex max-w-48 flex-col gap-0.5",
                align,
                index !== 0 && index !== last && "hidden sm:flex",
              )}
              style={index === 0 || index === last ? undefined : { left: at(index) }}
            >
              <span className="hidden text-xs text-muted-foreground sm:inline">
                {stopTypeLabel(stop.type)}
              </span>
              <span className="text-sm font-semibold">
                {[stop.city, stop.stateCode].filter(Boolean).join(", ")}
              </span>
              {time && (
                <span className="hidden text-xs text-muted-foreground tabular-nums sm:inline">
                  {time}
                </span>
              )}
            </li>
          );
        })}
      </ol>
    </figure>
  );
}
