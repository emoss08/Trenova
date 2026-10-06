import { useT } from "@trenova/shared/i18n/use-t";
import { Truck01Icon } from "@trenova/shared/components/icons";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { STAGE_META } from "@/lib/shipment-board/stage";
import {
  getDestinationStop,
  getOriginStop,
  getRouteProgress,
} from "@/lib/shipment-utils";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";
import { useEtaDelta } from "../cells/eta-cell";

const time = (value: number | null | undefined) =>
  value ? formatToUserTimezone(value, { showTimeZone: false, showSeconds: false }) : "—";

function windowText(stop: Stop | null) {
  if (!stop?.scheduledWindowStart) return "";
  const start = time(stop.scheduledWindowStart);
  return stop.scheduledWindowEnd ? `${start}–${time(stop.scheduledWindowEnd).split(" ").at(-1)}` : start;
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
  return (
    <div className={cn("flex min-w-0 flex-col gap-0.5", align === "end" && "items-end text-right")}>
      <span className="text-muted-foreground text-xs">{label}</span>
      <span className="truncate text-sm font-medium">{stop?.location?.name ?? "—"}</span>
      <span className="text-muted-foreground truncate text-xs">
        {[stop?.location?.city, windowText(stop)].filter(Boolean).join(" · ")}
      </span>
      <span className={cn("text-xs font-medium", statusClassName)}>{status}</span>
    </div>
  );
}

/** Where the load is between its pickup and its delivery, and how that compares with the plan. */
export function RouteTrack({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const origin = getOriginStop(shipment);
  const destination = getDestinationStop(shipment);
  const progress = getRouteProgress(shipment, Date.now() / 1000);
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
            progress.percent > 0 ? cn("border-transparent", meta.swatchClassName) : "border-border-strong",
          )}
        />
        <span className="relative h-0.5 flex-1 border-t border-dashed border-border-strong">
          <span
            className={cn("absolute -top-px left-0 h-0.5 rounded-full", meta.swatchClassName)}
            style={{ width: `${progress.percent}%` }}
          />
          {late ? (
            <span
              className="absolute -top-[3px] h-1.5 w-12 rounded-sm bg-[repeating-linear-gradient(135deg,var(--danger)_0_3px,transparent_3px_6px)] opacity-60"
              style={{ left: `${progress.percent}%` }}
            />
          ) : null}
          {moving ? (
            <span
              className="absolute top-1/2 grid size-5 -translate-x-1/2 -translate-y-1/2 place-items-center"
              style={{ left: `${progress.percent}%` }}
            >
              <span className={cn("animate-live-ping absolute inset-0 rounded-full opacity-40", meta.swatchClassName)} />
              <span className={cn("text-foreground-on-solid relative grid size-5 place-items-center rounded-full", meta.swatchClassName)}>
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
        <div className="flex flex-col items-center gap-0.5 pt-4 text-center">
          {moving ? (
            <>
              <span className={cn("text-sm font-medium", late && "text-danger")}>
                {late ? (shipment.eta?.reason ?? t("Running late")) : t("On schedule")}
              </span>
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
          statusClassName={late ? "text-danger" : delivered ? "text-success" : "text-muted-foreground"}
          align="end"
        />
      </div>
    </div>
  );
}
