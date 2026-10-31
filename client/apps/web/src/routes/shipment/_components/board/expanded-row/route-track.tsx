import { STAGE_META } from "@/lib/shipment-board/stage";
import { getDestinationStop, getOriginStop, getRouteProgress } from "@/lib/shipment-utils";
import { Truck01Icon } from "@trenova/shared/components/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useIsTruncated } from "@trenova/shared/hooks/use-is-truncated";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone, formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";
import { useEtaDelta } from "../cells/eta-cell";

const time = (value: number | null | undefined) =>
  value ? formatToUserTimezone(value, { showTimeZone: false, showSeconds: false }) : "—";

/** Half the truck marker's width, so its centre never comes closer than this to either end. */
const TRUCK_HALF_WIDTH = "0.625rem";

/** Where the truck marker's centre sits: at the progress, but never over either end's dot. */
export function truckLeft(percent: number): string {
  return `clamp(${TRUCK_HALF_WIDTH}, ${percent}%, calc(100% - ${TRUCK_HALF_WIDTH}))`;
}

/** The stop's appointment: its window's start, then its end's time of day when it has one. */
export function windowText(stop: Stop | null, t: TranslateFn): string {
  if (!stop?.scheduledWindowStart) return "";
  const start = time(stop.scheduledWindowStart);
  const end = stop.scheduledWindowEnd;
  return end && end > stop.scheduledWindowStart
    ? t("{0} – {1}", start, formatUnixTime(end))
    : start;
}

function Endpoint({
  label,
  stop,
  status,
  statusClassName,
  align,
}: {
  label: string;
  stop: Stop | null;
  status: string;
  statusClassName?: string;
  align: "start" | "end";
}) {
  const t = useT();
  return (
    <div className={cn("flex min-w-0 flex-col gap-0.5", align === "end" && "items-end text-right")}>
      <span className="text-muted-foreground text-xs">{label}</span>
      <span className="truncate text-sm font-medium">{stop?.location?.name ?? "—"}</span>
      <span className="text-muted-foreground truncate text-xs">
        {[stop?.location?.city, windowText(stop, t)].filter(Boolean).join(" · ")}
      </span>
      <span className={cn("text-xs font-medium", statusClassName)}>{status}</span>
    </div>
  );
}

function TruncatedStatus({ text, className }: { text: string; className?: string }) {
  const [ref, truncated] = useIsTruncated<HTMLSpanElement>();

  return (
    <Tooltip disabled={!truncated}>
      <TooltipTrigger
        render={
          <span ref={ref} className={cn("max-w-full truncate text-sm font-medium", className)}>
            {text}
          </span>
        }
      />
      <TooltipContent>{text}</TooltipContent>
    </Tooltip>
  );
}

/** Where the load is between its pickup and its delivery, and how that compares with the plan. */
export function RouteTrack({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const origin = getOriginStop(shipment);
  const destination = getDestinationStop(shipment);
  const now = useNow();
  const progress = getRouteProgress(shipment, now);
  const stage = shipment.stage ?? "NeedsCoverage";
  const meta = STAGE_META[stage];
  const moving = stage === "Moving" || stage === "Late";
  const late = stage === "Late";
  const delivered = stage === "Delivered";
  const delta = useEtaDelta(shipment);

  const pickupStatus = origin?.actualDeparture
    ? t("Departed {0}", time(origin.actualDeparture))
    : t("Scheduled");
  const deliveryStatus = delivered
    ? t("Delivered {0}", time(destination?.actualArrival))
    : late && delta
      ? t("Will miss · {0}", delta.text)
      : shipment.eta?.estimatedArrival
        ? t("ETA {0}", time(shipment.eta.estimatedArrival))
        : t("Appointment {0}", time(destination?.scheduledWindowStart));

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2" aria-hidden>
        <span
          className={cn(
            "size-2.5 shrink-0 rounded-full border-2",
            progress.percent > 0
              ? cn("border-transparent", meta.swatchClassName)
              : "border-border-strong",
          )}
        />
        <span className="relative h-0.5 flex-1 border-t border-dashed border-border-strong">
          <span
            className={cn("absolute -top-px left-0 h-0.5 rounded-full", meta.swatchClassName)}
            style={{ width: `${progress.percent}%` }}
          />
          {late ? (
            // The stretch the load will not cover in time: from the truck to the delivery, never
            // past it.
            <span
              data-testid="route-late-hatch"
              className="absolute top-1/2 h-1.5 -translate-y-1/2 rounded-sm bg-[repeating-linear-gradient(135deg,var(--danger)_0_3px,transparent_3px_6px)] opacity-60"
              style={{ left: `${progress.percent}%`, right: 0 }}
            />
          ) : null}
          {moving ? (
            <span
              data-testid="route-truck"
              className="absolute top-1/2 grid size-5 -translate-x-1/2 -translate-y-1/2 place-items-center"
              style={{ left: truckLeft(progress.percent) }}
            >
              <span
                className={cn(
                  "animate-live-ping absolute inset-0 rounded-full opacity-40",
                  meta.swatchClassName,
                )}
              />
              <span
                className={cn(
                  "text-foreground-on-solid relative grid size-5 place-items-center rounded-full",
                  meta.swatchClassName,
                )}
              >
                <Truck01Icon className="size-3" />
              </span>
            </span>
          ) : null}
        </span>
        <span
          className={cn(
            "size-2.5 shrink-0 rounded-full border-2",
            delivered ? cn("border-transparent", meta.swatchClassName) : "border-border-strong",
          )}
        />
      </div>
      <div className="grid grid-cols-[1fr_auto_1fr] items-start gap-4">
        <Endpoint
          label={t("Pickup")}
          stop={origin}
          status={pickupStatus}
          statusClassName={origin?.actualDeparture ? "text-success" : "text-muted-foreground"}
          align="start"
        />
        <div className="flex max-w-sm min-w-0 flex-col items-center gap-0.5 pt-4 text-center">
          {moving ? (
            <>
              <TruncatedStatus
                text={late ? (shipment.eta?.reason ?? t("Running late")) : t("On schedule")}
                className={cn(late && "text-danger")}
              />
              <span className="text-muted-foreground font-mono text-xs tabular-nums">
                {t(
                  "{0} mi done · {1} to go",
                  progress.milesDone.toLocaleString(),
                  progress.milesLeft.toLocaleString(),
                )}
              </span>
            </>
          ) : (
            <span className="text-muted-foreground font-mono text-xs tabular-nums">
              {delivered
                ? t("{0} mi · delivered", progress.totalMiles.toLocaleString())
                : t("{0} mi · not started", progress.totalMiles.toLocaleString())}
            </span>
          )}
        </div>
        <Endpoint
          label={t("Delivery")}
          stop={destination}
          status={deliveryStatus}
          statusClassName={
            late ? "text-danger" : delivered ? "text-success" : "text-muted-foreground"
          }
          align="end"
        />
      </div>
    </div>
  );
}
