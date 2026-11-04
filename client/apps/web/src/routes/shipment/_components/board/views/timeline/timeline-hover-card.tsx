import { PointerHoverCard } from "@/components/pointer-hover-card";
import type { TimelineLoad } from "@/lib/shipment-board/timeline";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import type { ReactNode } from "react";
import { formatTimelineTime } from "./timeline-format";
import type { TimelineScale } from "./timeline-scale";

const TIMELINE_CARD_WIDTH = 288;

/**
 * The timeline's one hover card: the shared pointer card, showing a load's stops, how late it
 * runs and who has it. Bars reach it through `usePointerHover<TimelineLoad>()`.
 */
export function TimelineHoverCard({
  scale,
  children,
}: {
  scale: TimelineScale;
  children: ReactNode;
}) {
  return (
    <PointerHoverCard<TimelineLoad>
      itemId={(load) => load.id}
      width={TIMELINE_CARD_WIDTH}
      render={(load) => <BarDetail load={load} scale={scale} />}
    >
      {children}
    </PointerHoverCard>
  );
}

/** Every stop with its time, how late the load runs, and who has it. */
function BarDetail({ load, scale }: { load: TimelineLoad; scale: TimelineScale }) {
  const t = useT();
  const coverage = load.coverage;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex min-w-0 items-center gap-2">
        <span className="font-mono font-semibold tabular-nums">
          {load.shipment.proNumber ?? "—"}
        </span>
        <span className="text-muted-foreground truncate">{load.shipment.customer?.name}</span>
      </div>
      <ol className="flex flex-col gap-1">
        {load.stops.map((stop, index) => {
          let when: string;
          if (stop.done) {
            when = stop.departedAt
              ? t("Left {0}", formatTimelineTime(stop.departedAt, scale))
              : t("Arrived {0}", formatTimelineTime(stop.arrivedAt ?? stop.windowStart, scale));
          } else if (load.eta && load.eta.stopIndex === index) {
            when = t("ETA {0}", formatTimelineTime(load.eta.at, scale));
          } else {
            when = formatTimelineTime(stop.windowStart, scale);
          }
          return (
            <li key={stop.id} className="grid grid-cols-[1rem_minmax(0,1fr)_auto] gap-2">
              <span className="text-muted-foreground font-mono tabular-nums">{index + 1}</span>
              <span className="truncate">{stop.place || "—"}</span>
              <span className="text-muted-foreground font-mono tabular-nums">{when}</span>
            </li>
          );
        })}
      </ol>
      {load.eta && load.eta.lateMinutes > 0 ? (
        <span className="text-danger font-medium">
          {t("{0} past the appointment window", formatMinutesSpan(load.eta.lateMinutes))}
        </span>
      ) : null}
      <span className="text-muted-foreground border-subtle border-t pt-1.5">
        {coverage.kind === "driver" || coverage.kind === "carrier"
          ? [coverage.name, coverage.detail].filter(Boolean).join(" · ")
          : coverage.kind === "tendered"
            ? t("Tendered, waiting on the carrier")
            : t("Not covered")}
      </span>
    </div>
  );
}
