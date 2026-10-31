import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  RouteIcon,
  Truck01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatUnixDateTimeShort,
  formatUnixMonthDay,
  formatUnixTime,
} from "@trenova/shared/lib/date";
import { STOP_TYPE_LABELS } from "@trenova/shared/lib/stop-type";
import { cn } from "@trenova/shared/lib/utils";
import type { Stop, StopActualAction, StopEta } from "@trenova/shared/types/shipment";
import { Fragment } from "react";
import { StopEtaLine } from "./stop-eta";

/** Where a stop stands, which its marker draws. */
type StopPhase = "done" | "here" | "next" | "canceled" | "invalid";

/** How the rail between two stops is drawn: travelled, travelling now, or still ahead. */
type LegPhase = "travelled" | "enroute" | "ahead";

function stopPhase(stop: Stop, invalid: boolean): StopPhase {
  if (invalid) return "invalid";
  if (stop.status === "Canceled") return "canceled";
  if (stop.actualDeparture || stop.status === "Completed") return "done";
  if (stop.actualArrival || stop.status === "InTransit") return "here";
  return "next";
}

function legPhase(from: StopPhase, to: StopPhase): LegPhase {
  if (from !== "done") return "ahead";
  return to === "done" || to === "here" ? "travelled" : "enroute";
}

/** The check call a stop is waiting on, when the person may record one now. */
export function stopCheckCallAction(stop: Stop, enabled: boolean): StopActualAction | null {
  if (!enabled || !stop.id || stop.status === "Canceled") return null;
  if (!stop.actualArrival) return "Arrive";
  if (!stop.actualDeparture) return "Depart";
  return null;
}

/** The stop in a few words, for the check call dialog: its type and where it is. */
export function stopDescription(stop: Stop, t: TranslateFn): string {
  const label = STOP_TYPE_LABELS[stop.type];
  const place = stop.location?.name || stop.addressLine;
  return place ? t("{0} · {1}", label, place) : label;
}

/**
 * A stop's place: the location the shipment read carries, or, for a stop whose location was
 * changed since, that location read on its own (once, and kept for five minutes).
 */
export function useStopPlace(stop: Stop | null) {
  const locationId = stop?.locationId ?? "";
  const embedded = stop?.location && stop.location.id === stop.locationId ? stop.location : null;
  const { data: fetched } = useQuery({
    queryKey: ["location", "selectOption", locationId],
    queryFn: () => apiService.locationService.getOption(locationId),
    enabled: !!locationId && !embedded,
    staleTime: 5 * 60 * 1000,
  });
  const location = embedded ?? fetched ?? null;
  const city = location
    ? [location.city, location.state?.abbreviation].filter(Boolean).join(", ")
    : "";
  return {
    name: location?.name || location?.addressLine1 || stop?.addressLine || "",
    city,
  };
}

function StopMarker({ phase }: { phase: StopPhase }) {
  return (
    <span
      aria-hidden
      className={cn(
        "relative z-10 flex size-4 items-center justify-center rounded-full transition-colors duration-300",
        phase === "done" && "bg-success text-foreground-on-solid",
        phase === "here" && "bg-card border-info border-2",
        phase === "next" && "bg-card border-border-strong border-2",
        phase === "canceled" && "bg-muted text-foreground-subtle",
        phase === "invalid" && "bg-danger-subtle border-danger text-danger border-2",
      )}
    >
      {phase === "done" && <CheckIcon className="animate-confirm size-2.5" strokeWidth={3} />}
      {phase === "here" && <span className="bg-info animate-confirm size-1.5 rounded-full" />}
      {phase === "canceled" && <XCloseIcon className="size-2.5" />}
    </span>
  );
}

/** A stretch of the rail: solid where it has been travelled, dashed where it is still ahead. */
function Rail({ phase, className }: { phase: LegPhase; className?: string }) {
  return (
    <span
      // A new key when a stretch is travelled, so it draws itself in from the stop it left.
      key={phase}
      aria-hidden
      className={cn(
        "absolute left-1/2 w-0 -translate-x-1/2 border-l-2",
        phase === "travelled"
          ? "border-success animate-draw origin-top"
          : "border-border-strong border-dashed",
        className,
      )}
    />
  );
}

/** When the stop is due: its date, then the time (or the window's start and end). */
function StopTime({ stop }: { stop: Stop }) {
  const t = useT();
  const start = stop.scheduledWindowStart;
  if (!start || start <= 0) {
    return (
      <span className="text-foreground-subtle text-xs whitespace-nowrap">{t("Unscheduled")}</span>
    );
  }
  const end = stop.scheduledWindowEnd;
  return (
    <span className="flex flex-col whitespace-nowrap tabular-nums">
      <span className="text-xs font-medium">{formatUnixMonthDay(start)}</span>
      <span className="text-foreground-muted text-2xs">{formatUnixTime(start)}</span>
      {end && end > start ? (
        <span className="text-foreground-subtle text-2xs">{t("to {0}", formatUnixTime(end))}</span>
      ) : null}
    </span>
  );
}

const ROW_GRID = "grid grid-cols-[4.75rem_1rem_minmax(0,1fr)_auto] gap-x-3";

function StopRow({
  stop,
  order,
  phase,
  railBelow,
  errorMessages,
  eta,
  checkCallAction,
  onCheckCall,
  onEdit,
}: {
  stop: Stop;
  /** Its place in the move; each stop settles in a beat after the one before it. */
  order: number;
  phase: StopPhase;
  /** The rail leaving this stop for the next, or null for the last stop. */
  railBelow: LegPhase | null;
  errorMessages: string[];
  eta?: StopEta;
  checkCallAction: StopActualAction | null;
  onCheckCall: () => void;
  onEdit: () => void;
}) {
  const t = useT();
  const place = useStopPlace(stop);
  const typeLabel = STOP_TYPE_LABELS[stop.type];
  const canceled = phase === "canceled";
  const progress = stop.actualDeparture
    ? t("Departed {0}", formatUnixDateTimeShort(stop.actualDeparture))
    : stop.actualArrival
      ? t("Arrived {0}", formatUnixDateTimeShort(stop.actualArrival))
      : null;

  return (
    <li
      className={cn(ROW_GRID, "animate-rise")}
      style={{ animationDelay: `${Math.min(order, 5) * 60}ms` }}
      data-stop-phase={phase}
    >
      <div className="pt-px text-right">
        <StopTime stop={stop} />
      </div>
      <div className="relative flex justify-center pt-0.5">
        {railBelow && <Rail phase={railBelow} className="top-5 -bottom-px" />}
        <StopMarker key={phase} phase={phase} />
      </div>
      <div className="flex min-w-0 flex-col gap-0.5 pb-2">
        <button
          type="button"
          onClick={onEdit}
          className={cn(
            "ui-focus-ring min-w-0 self-start truncate rounded-sm text-left text-sm font-medium hover:underline",
            canceled && "text-foreground-muted line-through",
            !place.name && "text-foreground-muted font-normal",
          )}
          title={place.name || undefined}
        >
          {place.name || t("Choose a location")}
        </button>
        <span className="text-foreground-muted truncate text-xs">
          {place.city ? t("{0} · {1}", typeLabel, place.city) : typeLabel}
        </span>
        {errorMessages.length > 0 ? (
          <Tooltip>
            <TooltipTrigger
              render={
                <button
                  type="button"
                  onClick={onEdit}
                  className="text-danger ui-focus-ring flex items-center gap-1 self-start rounded-sm text-xs"
                />
              }
            >
              <AlertTriangleIcon className="size-3" />
              {t(
                "{0, plural, one {# problem to fix} other {# problems to fix}}",
                errorMessages.length,
              )}
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-xs">
              <ul className="space-y-1 text-xs">
                {errorMessages.map((message) => (
                  <li key={message}>{message}</li>
                ))}
              </ul>
            </TooltipContent>
          </Tooltip>
        ) : canceled ? null : progress ? (
          <span className="text-foreground-muted text-2xs tabular-nums">{progress}</span>
        ) : (
          <StopEtaLine eta={eta} />
        )}
      </div>
      <div className="pt-0.5">
        {checkCallAction && !canceled && errorMessages.length === 0 ? (
          <Button type="button" size="xxs" variant="outline" onClick={onCheckCall}>
            {checkCallAction === "Arrive" ? t("Arrive") : t("Depart")}
          </Button>
        ) : null}
      </div>
    </li>
  );
}

/**
 * The stretch between two stops: the rail carries on, a truck rides it while the load is on its
 * way, and a quiet label names the leg's distance when the move is a single leg (its distance is
 * the move's).
 */
function LegRow({ phase, distance }: { phase: LegPhase; distance: string | null }) {
  const t = useT();
  const enroute = phase === "enroute";
  if (!enroute && !distance) {
    return (
      <li aria-hidden className={cn(ROW_GRID, "h-2")}>
        <span />
        <span className="relative">
          <Rail phase={phase} className="top-0 -bottom-1" />
        </span>
      </li>
    );
  }
  return (
    <li className={cn(ROW_GRID, "py-1")} data-leg-phase={phase}>
      <span />
      <span className="relative flex items-center justify-center">
        <Rail phase={enroute ? "travelled" : phase} className="-top-1 bottom-1/2" />
        <Rail phase={phase === "travelled" ? "travelled" : "ahead"} className="top-1/2 -bottom-2" />
        {enroute && (
          <span className="bg-info text-foreground-on-solid ring-card animate-arrive relative z-10 grid size-5 shrink-0 place-items-center rounded-full ring-2">
            <Truck01Icon aria-hidden className="size-3" />
          </span>
        )}
      </span>
      <span
        className={cn(
          "text-2xs flex items-center gap-1 tabular-nums",
          enroute ? "text-info font-medium" : "text-foreground-subtle",
        )}
      >
        {!enroute && <RouteIcon aria-hidden className="size-3" />}
        <span key={phase} className="animate-rise">
          {enroute ? (distance ? t("On the way · {0}", distance) : t("On the way")) : distance}
        </span>
      </span>
    </li>
  );
}

/**
 * A move's stops as an itinerary: each stop's date and time beside a marker on a rail (done,
 * where the truck is, still to come, canceled, or with something to fix), its place, type and
 * city, and, while it is open, its projected arrival and the check call it waits on. Between
 * stops the rail fills in as they are completed, with a truck on it while the load is on its way.
 * A stop's name opens the move for editing.
 */
export function MoveStops({
  stops,
  stopErrors,
  etas,
  distance,
  canRecordActuals,
  onCheckCall,
  onEdit,
}: {
  stops: readonly Stop[];
  /** Each stop's error messages, by its place in the move. */
  stopErrors: readonly string[][];
  etas: ReadonlyMap<string, StopEta>;
  /** The move's distance, shown on the leg of a two-stop move. */
  distance: string | null;
  /** Whether check calls may be recorded on this move now. */
  canRecordActuals: boolean;
  onCheckCall: (stop: Stop, action: StopActualAction) => void;
  onEdit: () => void;
}) {
  const phases = stops.map((stop, index) => stopPhase(stop, (stopErrors[index] ?? []).length > 0));
  const singleLeg = stops.length === 2;

  return (
    <ScrollArea maskVariant="card" className="max-h-80">
      <ol className="px-4 pt-1 pb-3">
        {stops.map((stop, index) => {
          // A check call is offered on a stop only once every earlier stop has been left.
          const earlierLeft = stops
            .slice(0, index)
            .every((prior) => prior.status === "Canceled" || !!prior.actualDeparture);
          const action = stopCheckCallAction(stop, canRecordActuals && earlierLeft);
          const next = index < stops.length - 1 ? phases[index + 1] : null;
          const leg = next ? legPhase(phases[index], next) : null;
          return (
            // oxlint-disable-next-line react/no-array-index-key -- a new stop has no id until it is saved, and its place in the move is what identifies it
            <Fragment key={stop.id ?? `new-${index}`}>
              <StopRow
                stop={stop}
                order={index}
                phase={phases[index]}
                railBelow={leg}
                errorMessages={stopErrors[index] ?? []}
                eta={stop.id ? etas.get(stop.id) : undefined}
                checkCallAction={action}
                onCheckCall={() => action && onCheckCall(stop, action)}
                onEdit={onEdit}
              />
              {leg && <LegRow phase={leg} distance={singleLeg ? distance : null} />}
            </Fragment>
          );
        })}
      </ol>
    </ScrollArea>
  );
}
