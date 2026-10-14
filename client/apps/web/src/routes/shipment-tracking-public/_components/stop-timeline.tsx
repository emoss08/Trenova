import type { Journey } from "@/lib/shipment-tracking";
import { dayAndTime, windowRange } from "@/lib/shipment-tracking";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { stopTypeLabel } from "@trenova/shared/lib/stop-type";
import { cn } from "@trenova/shared/lib/utils";
import type { PublicShipmentStop, PublicShipmentView } from "@trenova/shared/types/shipment-tracking";

type StopState = "done" | "here" | "next" | "later";

function stateOf(stop: PublicShipmentStop, index: number, journey: Journey): StopState {
  if (stop.departedAt != null || (journey.finished && stop.arrivedAt != null)) {
    return "done";
  }
  if (stop.arrivedAt != null) {
    return "here";
  }
  return index === journey.activeIndex && journey.started ? "next" : "later";
}

function Node({ state }: { state: StopState }) {
  if (state === "done") {
    return (
      <span className="flex size-5 items-center justify-center rounded-full bg-ink text-background">
        <CheckIcon className="size-3" aria-hidden />
      </span>
    );
  }
  if (state === "here" || state === "next") {
    return (
      <span className="relative flex size-5 items-center justify-center rounded-full bg-background ring-2 ring-ink">
        <span className="st-dot size-1.5 rounded-full" />
      </span>
    );
  }
  return <span className="size-5 rounded-full bg-background ring-1 ring-border" />;
}

function Times({ stop, state, now }: { stop: PublicShipmentStop; state: StopState; now: number }) {
  const t = useT();
  const window = windowRange(t, stop);
  const lines: { text: string; strong?: boolean }[] = [];
  if (stop.arrivedAt != null) {
    lines.push({ text: t("Arrived {0}", dayAndTime(t, stop.arrivedAt, now, { inline: true })), strong: state === "here" });
  }
  if (stop.departedAt != null) {
    lines.push({ text: t("Left {0}", dayAndTime(t, stop.departedAt, now, { inline: true })) });
  }
  if (stop.arrivedAt == null && stop.estimatedArrival != null) {
    lines.push({ text: t("Expected {0}", dayAndTime(t, stop.estimatedArrival, now, { inline: true })), strong: true });
  }

  return (
    <div className="flex flex-col items-end gap-0.5 text-right tabular-nums">
      {lines.map((line) => (
        <span
          key={line.text}
          className={cn("text-xs", line.strong ? "font-medium text-foreground" : "text-muted-foreground")}
        >
          {line.text}
        </span>
      ))}
      {window && stop.windowStart != null && (
        <span className="text-xs text-muted-foreground">
          {lines.length === 0
            ? t("Appointment {0}, {1}", formatUnixMonthDay(stop.windowStart), window)
            : t("Appt. {0}", window)}
        </span>
      )}
      {stop.arrivedAt == null && (stop.running === "Late" || stop.running === "AtRisk") && (
        <Badge variant="warning" className="mt-1">
          {stop.running === "Late" ? t("Running late") : t("At risk")}
        </Badge>
      )}
    </div>
  );
}

/** Every stop in order, each with what has happened there and what is expected. */
export function StopTimeline({
  view,
  journey,
  now,
}: {
  view: PublicShipmentView;
  journey: Journey;
  now: number;
}) {
  const t = useT();

  return (
    <ol className="flex flex-col">
      {view.stops.map((stop, index) => {
        const state = stateOf(stop, index, journey);
        const isLast = index === view.stops.length - 1;
        return (
          <li key={stop.sequence} className="relative grid grid-cols-[20px_1fr_auto] gap-x-4 pb-7 last:pb-0">
            {!isLast && (
              <span
                className={cn(
                  "absolute top-6 bottom-1 left-[9.5px] w-px",
                  state === "done" ? "bg-ink" : "border-l border-dashed border-border",
                )}
                aria-hidden
              />
            )}
            <Node state={state} />
            <div className="flex min-w-0 flex-col gap-0.5">
              <span className="text-xs text-muted-foreground">
                {t("Stop {0} · {1}", stop.sequence, stopTypeLabel(stop.type))}
              </span>
              <span className="truncate text-sm font-semibold">
                {stop.locationName ?? [stop.city, stop.stateCode].filter(Boolean).join(", ")}
              </span>
              {(stop.locationName || stop.addressLine) && (
                <span className="truncate text-xs text-muted-foreground">
                  {[stop.addressLine, [stop.city, stop.stateCode].filter(Boolean).join(", ")]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
              )}
            </div>
            <Times stop={stop} state={state} now={now} />
          </li>
        );
      })}
    </ol>
  );
}
