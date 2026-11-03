import { recordPath } from "@/config/record-links";
import {
  actualsLabel,
  etaReading,
  journeyProgress,
  journeyStops,
  laneOf,
  punctualityLabel,
  shipmentStage,
  stopTypeLabel,
  totalMiles,
  windowLabel,
  type JourneyStop,
} from "@/routes/desk/_components/artifacts/cards/shipment/shipment-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort, formatUnixTime } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { Shipment, ShipmentMove } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { Truck01Icon } from "@trenova/shared/components/icons";
import type { PaletteRecord } from "../../palette-model";
import { StatusDot } from "../../palette-parts";
import { shipmentStatus } from "../../palette-status";
import {
  PreviewError,
  PreviewFields,
  PreviewHead,
  PreviewSection,
  PreviewSkeleton,
  PreviewSource,
  StatePill,
} from "./preview-frame";
import { shipmentEventsQuery, shipmentPreviewQuery } from "./preview-queries";
import type { RecordPreviewProps } from "./record-preview-props";

function stopMoment(stop: JourneyStop | undefined): number | null {
  if (!stop) {
    return null;
  }
  return stop.departedAt ?? stop.arrivedAt ?? stop.windowStart;
}

function firstPickup(stops: readonly JourneyStop[]): JourneyStop | undefined {
  return stops.find((stop) => stop.phase !== "canceled" && stop.type.includes("Pickup"));
}

function lastDelivery(stops: readonly JourneyStop[]): JourneyStop | undefined {
  return [...stops]
    .reverse()
    .find((stop) => stop.phase !== "canceled" && stop.type.includes("Delivery"));
}

/** The move the shipment is on now: the first one still running, or the last one run. */
function currentMove(moves: readonly ShipmentMove[]): ShipmentMove | undefined {
  const ordered = [...moves].sort((a, b) => a.sequence - b.sequence);
  return (
    ordered.find((move) => move.status !== "Completed" && move.status !== "Canceled") ??
    [...ordered].reverse().find((move) => move.status !== "Canceled")
  );
}

/** How many stops the preview lists before it says how many more there are. */
const STOPS_SHOWN = 5;

const PUNCTUALITY_TEXT: Record<"early" | "onTime" | "late", string> = {
  early: "text-foreground-subtle",
  onTime: "text-foreground-subtle",
  late: "text-warning-subtle-foreground",
};

/**
 * Every stop in the order it is run, as the Desk's shipment card lists them
 * down its rail: what kind of stop and where, its window or when the truck
 * came and went, and whether it was on time. A run stop is filled, the one
 * the truck is at ringed, a canceled one struck through.
 */
export function StopList({ stops }: { stops: readonly JourneyStop[] }) {
  const t = useT();
  const shown = stops.slice(0, STOPS_SHOWN);
  const more = stops.length - shown.length;
  return (
    <ol className="flex flex-col">
      {shown.map((stop, index) => {
        const name = stop.place.name || stop.place.label;
        const where = stop.place.name && stop.place.label !== name ? stop.place.label : "";
        const when = actualsLabel(stop, t) || windowLabel(stop, t);
        const punctuality = stop.punctuality ? punctualityLabel(stop.punctuality, t) : "";
        return (
          <li key={stop.id} className="relative flex gap-3 pb-2.5 last:pb-0">
            {index < shown.length - 1 && (
              <span aria-hidden className="bg-border absolute top-3 bottom-0 left-[3px] w-px" />
            )}
            <span
              aria-hidden
              className={cn(
                "mt-1.5 size-[7px] shrink-0 rounded-full",
                stop.phase === "done"
                  ? "bg-foreground"
                  : stop.phase === "current"
                    ? "border-foreground bg-card border-2"
                    : "border-border-strong bg-card border",
              )}
            />
            <span className="flex min-w-0 flex-1 flex-col">
              <span
                className={cn(
                  "flex min-w-0 items-baseline gap-2 text-base",
                  stop.phase === "canceled" && "text-foreground-subtle line-through",
                )}
              >
                <span className="text-foreground-subtle shrink-0 text-sm">
                  {stopTypeLabel(stop.type, t)}
                </span>
                <span className="text-foreground truncate">{name || "—"}</span>
              </span>
              {(where || when) && (
                <span className="text-foreground-subtle truncate text-sm">
                  {[where, when].filter(Boolean).join(" · ")}
                </span>
              )}
            </span>
            {punctuality && stop.punctuality && (
              <span className={cn("shrink-0 text-sm", PUNCTUALITY_TEXT[stop.punctuality.state])}>
                {punctuality}
              </span>
            )}
          </li>
        );
      })}
      {more > 0 && (
        <li className="text-foreground-subtle pt-1 pl-5 text-sm">
          {t("{0, plural, one {# more stop} other {# more stops}}", more)}
        </li>
      )}
    </ol>
  );
}

function RouteEnd({
  city,
  at,
  align,
  reached,
}: {
  city: string;
  at: number | null;
  align: "start" | "end";
  reached: boolean;
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1", align === "end" && "items-end text-right")}>
      <span
        aria-hidden
        className={cn(
          "size-2 rounded-full border-2",
          reached ? "border-foreground bg-foreground" : "border-foreground bg-transparent",
        )}
      />
      <span className="text-foreground truncate text-base font-semibold">{city || "—"}</span>
      <span className="text-foreground-subtle font-mono text-xs whitespace-nowrap">
        {at ? formatUnixDateTimeShort(at) : "—"}
      </span>
    </div>
  );
}

/**
 * Where the load is between its ends, as the Desk's record card draws it:
 * the driven part of the line solid, the rest dashed, the truck at the
 * point it has reached, and the miles and the ETA reading under it.
 */
function RouteCard({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const stops = journeyStops(shipment.moves);
  const lane = laneOf(stops);
  const progress = journeyProgress(shipment.moves, stops);
  const miles = totalMiles(shipment.moves);
  const reading = etaReading(shipment.eta, t);
  const pickup = firstPickup(stops);
  const delivery = lastDelivery(stops);
  const moving = progress > 0 && progress < 1;
  const percent = Math.round(progress * 100);
  const lateness = reading && reading.state !== "unknown" ? reading : null;

  return (
    <div className="bg-sunken rounded-surface flex flex-col gap-3 px-4 pt-3.5 pb-3">
      <div className="grid grid-cols-[minmax(0,1fr)_minmax(4rem,1.2fr)_minmax(0,1fr)] items-start gap-3">
        <RouteEnd
          city={lane.from?.label ?? ""}
          at={stopMoment(pickup)}
          align="start"
          reached={pickup?.phase === "done" || pickup?.phase === "current"}
        />
        <div className="relative mt-[1.6rem] h-0.5" aria-hidden>
          <div className="border-border-strong absolute inset-x-0 top-0 border-t-2 border-dashed" />
          <div
            className="bg-foreground cp-grow-x absolute top-0 left-0 h-0.5 rounded-full"
            style={{ width: `${percent}%` }}
          />
          {moving && (
            <span
              className="bg-card text-foreground ring-border cp-grow absolute top-0.25 flex size-6 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full ring-1"
              style={{ left: `${percent}%` }}
            >
              <Truck01Icon className="size-3" />
            </span>
          )}
        </div>
        <RouteEnd
          city={lane.to?.label ?? ""}
          at={stopMoment(delivery)}
          align="end"
          reached={delivery?.phase === "done"}
        />
      </div>
      {(miles !== null || lateness) && (
        <div className="border-border-subtle flex items-center justify-between gap-3 border-t pt-2.5 text-sm">
          <span className="text-foreground-subtle tabular-nums">
            {miles !== null &&
              (moving
                ? t(
                    "{0} mi · {1} to go",
                    Math.round(miles).toLocaleString(),
                    Math.round(miles * (1 - progress)).toLocaleString(),
                  )
                : t("{0} mi", Math.round(miles).toLocaleString()))}
          </span>
          {lateness && (
            <span className="text-foreground flex items-center gap-1.5">
              {(lateness.state === "warn" || lateness.state === "bad") && (
                <StatusDot state={lateness.state} className="size-1.5" />
              )}
              {lateness.label}
              {lateness.at !== null && (
                <span className="text-foreground-subtle">
                  · {t("ETA {0}", formatUnixTime(lateness.at))}
                </span>
              )}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function ShipmentBody({
  shipment,
  record,
  onNavigate,
}: {
  shipment: Shipment;
  record: PaletteRecord;
  onNavigate: (href: string) => void;
}) {
  const t = useT();
  const { data: events } = useQuery(shipmentEventsQuery(record.id));
  const status = shipmentStatus(shipment.status, t);
  const stage = shipmentStage(shipment);
  const reading = etaReading(shipment.eta, t);
  const move = currentMove(shipment.moves);
  const worker = move?.assignment?.primaryWorker;
  const workerName =
    worker?.wholeName || [worker?.firstName, worker?.lastName].filter(Boolean).join(" ");
  const total = Number(shipment.totalChargeAmount);
  const latest = events?.[0]?.occurredAt ?? shipment.updatedAt;
  const stops = journeyStops(shipment.moves);

  return (
    <>
      <PreviewSource
        kind="shipment"
        facts={[
          stage === "Moving" || stage === "Late" ? t("Live") : null,
          latest ? t("Updated {0}", formatUnixTime(latest)) : null,
        ]}
      />
      <PreviewHead
        title={shipment.proNumber || record.title}
        mono
        subtitle={[shipment.customer?.name, shipment.bol ? t("BOL {0}", shipment.bol) : null]
          .filter(Boolean)
          .join(" · ")}
        status={status ? <StatePill label={status.label} state={status.state} /> : undefined}
      />
      <RouteCard shipment={shipment} />
      {reading && reading.detail !== "" && (
        <p className="text-foreground-muted text-base first-letter:uppercase">{reading.detail}</p>
      )}
      {stops.length > 0 && (
        <PreviewSection
          label={t("Stops")}
          aside={t("{0, plural, one {# stop} other {# stops}}", stops.length)}
        >
          <StopList stops={stops} />
        </PreviewSection>
      )}
      <PreviewFields
        rows={[
          {
            label: t("Driver"),
            value:
              worker?.id && workerName ? (
                <button
                  type="button"
                  tabIndex={-1}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => onNavigate(recordPath("worker", worker.id!))}
                  className="decoration-border-strong hover:decoration-foreground underline underline-offset-2"
                >
                  {workerName}
                </button>
              ) : (
                t("Unassigned")
              ),
            state: worker?.id ? undefined : "warn",
          },
          Number.isFinite(total) && total > 0
            ? {
                label: t("Total"),
                value: <span className="font-mono">{formatCurrency(total)}</span>,
              }
            : null,
          {
            label: t("Tractor"),
            value: <span className="font-mono">{move?.assignment?.tractor?.code || "—"}</span>,
          },
          {
            label: t("Trailer"),
            value: <span className="font-mono">{move?.assignment?.trailer?.code || "—"}</span>,
          },
        ]}
      />
      {events && events.length > 0 && (
        <PreviewSection label={t("Latest")}>
          <ol className="flex flex-col">
            {events.map((event, index) => (
              <li key={event.id} className="relative flex gap-3 pb-2.5 last:pb-0">
                {index < events.length - 1 && (
                  <span aria-hidden className="bg-border absolute top-3 bottom-0 left-[3px] w-px" />
                )}
                <span
                  aria-hidden
                  className={cn(
                    "mt-1.5 size-[7px] shrink-0 rounded-full",
                    index === 0 ? "bg-foreground" : "border-border-strong bg-card border",
                  )}
                />
                <span className="text-foreground min-w-0 flex-1 text-base">{event.summary}</span>
                <span className="text-foreground-subtle shrink-0 font-mono text-xs">
                  {formatUnixDateTimeShort(event.occurredAt)}
                </span>
              </li>
            ))}
          </ol>
        </PreviewSection>
      )}
    </>
  );
}

export function ShipmentPreview({ record, onNavigate }: RecordPreviewProps) {
  const { data, isLoading, isError } = useQuery(shipmentPreviewQuery(record.id));

  if (isLoading) {
    return <PreviewSkeleton />;
  }
  if (isError || !data) {
    return <PreviewError />;
  }
  return <ShipmentBody shipment={data} record={record} onNavigate={onNavigate} />;
}
