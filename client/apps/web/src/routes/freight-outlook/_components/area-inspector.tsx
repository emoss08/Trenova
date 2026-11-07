import { RECORD_LINKS } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import { filteredListPath } from "@/lib/filtered-list-path";
import type { ShipmentStatus } from "@trenova/shared/types/shipment";
import type {
  FreightAreaDetail,
  FreightPositioning,
  FreightReach,
  FreightSlot,
} from "@/lib/graphql/freight-outlook";
import { freightOutlook } from "@/lib/queries/freight-outlook";
import { useQuery } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { ChevronLeftIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatShare } from "@/lib/dispatch-forecast";
import { formatNumber } from "@trenova/shared/i18n/format";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatCentsWhole } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy, type ReactNode } from "react";
import { Link } from "react-router";
import { AreaInspectorSkeleton } from "./area-inspector-skeleton";
import { CollapseBriefButton } from "./brief-rail";
import { paceReadings, startsDay } from "./outlook-model";
import { Strong, dayKey, dayLabel, hourLabel, slotLabel, slotRange } from "./outlook-parts";
import { ReachMapSkeleton } from "./reach-map-skeleton";
import { useOutlookActions } from "./use-outlook-actions";

/** The shipment statuses still on their way somewhere, for the loads into an area. */
const OPEN_SHIPMENT_STATUSES: readonly ShipmentStatus[] = [
  "New",
  "PartiallyAssigned",
  "Assigned",
  "InTransit",
  "Delayed",
  "PartiallyCompleted",
];

const ReachMap = lazy(() => import("./reach-map"));

const KIND_LABELS = defineLabels({
  Send: "Send",
  Book: "Book",
  Keep: "Keep",
  Call: "Call",
});

const KIND_VARIANT = {
  Send: "brand",
  Book: "accent-teal",
  Keep: "accent-violet",
  Call: "warning",
} as const;

const REACH_LABELS = defineLabels({
  Here: "Here",
  OnTime: "In time",
  NotWorth: "Not worth the miles",
  TooLate: "Arrives too late",
  TooFar: "Too far",
});

export type OpenArea = { cell: number; slotStart: number };

function Block({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section aria-label={title} className="flex flex-col gap-2">
      <h3 className="flex items-center gap-2.5 text-xs font-normal text-muted-foreground after:h-px after:flex-1 after:bg-border-subtle">
        {title}
      </h3>
      {children}
    </section>
  );
}

const HOURS_WIDTH = 372;
const HOURS_HEIGHT = 128;
const HOURS_LEFT = 22;

function HoursChart({
  slots,
  starts,
}: {
  slots: readonly FreightSlot[];
  starts: readonly number[];
}) {
  const t = useT();
  const step = (HOURS_WIDTH - HOURS_LEFT - 4) / Math.max(slots.length, 1);
  const top = Math.max(4, ...slots.map((slot) => Math.max(slot.likelyHigh, slot.drivers))) + 1;
  const y = (value: number) => 8 + (1 - value / top) * (HOURS_HEIGHT - 28);
  const ticks = Array.from({ length: Math.floor(top / 2) + 1 }, (_, index) => index * 2);

  return (
    <>
      <svg
        viewBox={`0 0 ${HOURS_WIDTH} ${HOURS_HEIGHT}`}
        className="block h-auto w-full"
        role="img"
        aria-label={t("Loads likely, booked and drivers expected in each window")}
      >
        {ticks.map((tick) => (
          <g key={tick}>
            <line
              x1={HOURS_LEFT}
              x2={HOURS_WIDTH - 4}
              y1={y(tick)}
              y2={y(tick)}
              className="stroke-border-subtle"
            />
            <text
              x={HOURS_LEFT - 5}
              y={y(tick) + 3}
              textAnchor="end"
              className="fill-foreground-subtle text-2xs"
            >
              {formatNumber(tick)}
            </text>
          </g>
        ))}
        {slots.map((slot, index) => {
          const cx = HOURS_LEFT + step * (index + 0.5);
          const width = step * 0.5;
          const dayStart = startsDay(starts, index, dayKey);
          return (
            <g key={slot.start}>
              {slot.short >= 1.5 ? (
                <rect
                  x={HOURS_LEFT + step * index + 1}
                  y={8}
                  width={step - 2}
                  height={HOURS_HEIGHT - 28}
                  rx={3}
                  className="fill-warning/15"
                />
              ) : null}
              <rect
                x={cx - width / 2}
                y={y(slot.likelyHigh)}
                width={width}
                height={Math.max(2, y(slot.likelyLow) - y(slot.likelyHigh))}
                rx={3}
                className="fill-warning/25"
              />
              {slot.booked > 0 ? (
                <rect
                  x={cx - width / 2}
                  y={y(slot.booked)}
                  width={width}
                  height={y(0) - y(slot.booked)}
                  rx={2}
                  className="fill-warning-foreground/85"
                />
              ) : null}
              <line
                x1={cx - step * 0.4}
                x2={cx + step * 0.4}
                y1={y(slot.drivers)}
                y2={y(slot.drivers)}
                strokeWidth={2}
                className="stroke-foreground"
              />
              <text
                x={cx}
                y={HOURS_HEIGHT - 5}
                textAnchor="middle"
                className={cn(
                  "text-2xs",
                  dayStart ? "fill-foreground font-medium" : "fill-foreground-subtle",
                )}
              >
                {dayStart ? dayLabel(slot.start) : index % 2 === 0 ? hourLabel(slot.start) : ""}
              </text>
            </g>
          );
        })}
      </svg>
      <div className="flex flex-wrap gap-x-3.5 gap-y-1 text-2xs text-muted-foreground">
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden className="size-2.5 rounded-xs bg-warning-foreground/85" />
          {t("Booked")}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden className="size-2.5 rounded-xs bg-warning/25" />
          {t("Likely range")}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden className="h-0.5 w-4 rounded-full bg-foreground" />
          {t("Drivers expected")}
        </span>
      </div>
    </>
  );
}

const RECENT_WIDTH = 372;
const RECENT_HEIGHT = 92;
const RECENT_TOP = 14;
const RECENT_BOTTOM = 18;

function RecentChart({ detail }: { detail: FreightAreaDetail }) {
  const t = useT();
  const weeks = detail.recent;
  const top = Math.max(detail.recentHigh, ...weeks, 1);
  const plot = RECENT_HEIGHT - RECENT_TOP - RECENT_BOTTOM;
  const y = (value: number) => RECENT_TOP + (1 - value / top) * plot;
  const step = RECENT_WIDTH / Math.max(weeks.length, 1);
  const bar = Math.min(22, step * 0.55);

  return (
    <svg
      viewBox={`0 0 ${RECENT_WIDTH} ${RECENT_HEIGHT}`}
      className="block h-auto w-full"
      role="img"
      aria-label={t("Loads in this window in each of the last weeks, against its usual range")}
    >
      <rect
        x={0}
        y={y(detail.recentHigh)}
        width={RECENT_WIDTH}
        height={Math.max(2, y(detail.recentLow) - y(detail.recentHigh))}
        rx={3}
        className="fill-brand-subtle"
      />
      <line x1={0} x2={RECENT_WIDTH} y1={y(0)} y2={y(0)} className="stroke-border" />
      {weeks.map((loads, index) => {
        const x = step * (index + 0.5);
        const ago = weeks.length - index;
        const outside = loads < detail.recentLow || loads > detail.recentHigh;
        return (
          <g key={ago}>
            {loads > 0 ? (
              <rect
                x={x - bar / 2}
                y={y(loads)}
                width={bar}
                height={y(0) - y(loads)}
                rx={2}
                className={outside ? "fill-warning" : "fill-foreground/70"}
              />
            ) : null}
            <text
              x={x}
              y={y(loads) - 4}
              textAnchor="middle"
              className="fill-foreground text-2xs font-medium tabular-nums"
            >
              {formatNumber(loads)}
            </text>
            {index === 0 || ago === 1 ? (
              <text
                x={x}
                y={RECENT_HEIGHT - 3}
                textAnchor="middle"
                className="fill-foreground-subtle text-2xs"
              >
                {ago === 1 ? t("Last week") : t("{0} weeks ago", ago)}
              </text>
            ) : null}
          </g>
        );
      })}
    </svg>
  );
}

function ReachList({ reach }: { reach: readonly FreightReach[] }) {
  const t = useT();
  return (
    <ul className="divide-y divide-border-subtle">
      {reach.map((row) => (
        <li
          key={row.workerId}
          className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 py-2"
        >
          <span className="min-w-0">
            <span className="block truncate text-sm font-medium">{row.workerName}</span>
            <span className="block truncate text-xs text-foreground-subtle">
              {row.reach === "Here"
                ? t("Here, free {0}", slotLabel(row.freeAt))
                : t(
                    "{0}, {1, plural, one {# mile} other {# miles}}, free {2} · {3}",
                    row.label || t("Their drop"),
                    Math.round(row.miles),
                    slotLabel(row.freeAt),
                    REACH_LABELS[row.reach],
                  )}
            </span>
          </span>
          <span
            className={cn(
              "font-mono text-sm tabular-nums",
              row.netCents > 0 && row.reach !== "TooLate" && row.reach !== "TooFar"
                ? "text-success-foreground"
                : "text-foreground-subtle",
            )}
          >
            {row.netCents > 0
              ? t("+{0}", formatCentsWhole(row.netCents))
              : formatCentsWhole(row.netCents)}
          </span>
        </li>
      ))}
    </ul>
  );
}

function Positionings({ rows }: { rows: readonly FreightPositioning[] }) {
  const t = useT();
  const { callOff } = useOutlookActions();
  const { allowed: canCallOff } = usePermission(Resource.DriverPositioning, Operation.Cancel);
  return (
    <ul className="divide-y divide-border-subtle border-t border-border-subtle">
      {rows.map((row) => (
        <li key={row.id} className="flex items-center gap-3 py-2">
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm font-medium">{row.workerName}</span>
            <span className="block truncate text-xs text-foreground-subtle">
              {t("On the way to {0}, there by {1}", row.toLabel, slotLabel(row.arriveAt))}
            </span>
          </span>
          <Badge variant="success">{t("On the way")}</Badge>
          {canCallOff ? (
            <Button
              type="button"
              size="xs"
              variant="ghost"
              isLoading={callOff.isPending && callOff.variables === row.id}
              onClick={() => callOff.mutateAsync(row.id)}
            >
              {t("Call off")}
            </Button>
          ) : null}
        </li>
      ))}
    </ul>
  );
}

function AreaBody({ detail }: { detail: FreightAreaDetail }) {
  const t = useT();
  const rt = useRichT();
  const { area } = detail;
  const starts = area.slots.map((slot) => slot.start);
  const index = Math.max(
    0,
    area.slots.findIndex((slot) => slot.start === detail.slotStart),
  );
  const slot = area.slots[index];
  const pace = paceReadings(
    detail.pace,
    area.slots.map((_, i) => i),
  );
  const strong = (chunks: ReactNode) => <Strong>{chunks}</Strong>;
  const chips = detail.moves.filter((move) => move.kind !== "Call");

  return (
    <div className="flex flex-col gap-3.5">
      <div className="flex flex-col gap-1">
        <p className="text-xs text-foreground-subtle">
          {t("Area within about 30 mi of {0}", area.label)}
        </p>
        <h2 className="text-lg leading-tight font-semibold tracking-tight">{area.label}</h2>
        <p className="mt-0.5 text-xs leading-normal text-muted-foreground">
          {slot && slot.short >= 0.5
            ? rt(
                "<b>{0, plural, one {# truck} other {# trucks}} short {1}</b>. {2}–{3} loads likely, {4} booked, {5, plural, one {# driver} other {# drivers}} expected. One more driver ending here is worth <b>{6}</b>.",
                { b: strong },
                Math.round(slot.short),
                slotRange(slot.start),
                slot.likelyLow,
                slot.likelyHigh,
                slot.booked,
                slot.drivers,
                formatCentsWhole(area.valueCents),
              )
            : slot && slot.spare >= 0.5
              ? rt(
                  "<b>{0, plural, one {# truck} other {# trucks}} to spare {1}</b>. {2}–{3} loads likely against {4, plural, one {# driver} other {# drivers}}. A good place to send drivers from.",
                  { b: strong },
                  Math.round(slot.spare),
                  slotRange(slot.start),
                  slot.likelyLow,
                  slot.likelyHigh,
                  slot.drivers,
                )
              : t("Drivers and freight roughly match over the next 48 hours.")}
        </p>
      </div>

      {chips.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {chips.map((move) => {
            const incoming = move.toCell === area.cell;
            return (
              <Badge
                key={`${move.kind}:${move.workerId}:${move.toCell}:${move.slotStart}`}
                variant={KIND_VARIANT[move.kind]}
              >
                <span className="font-medium">{KIND_LABELS[move.kind]}</span>
                <span className="font-normal">
                  {incoming ? move.workerName : t("{0} → {1}", move.workerName, move.toLabel)}
                </span>
              </Badge>
            );
          })}
        </div>
      ) : null}

      <Block title={t("Hour by hour")}>
        <HoursChart slots={area.slots} starts={starts} />
      </Block>

      {area.customers.length > 0 ? (
        <Block title={t("Who books here")}>
          <ul className="divide-y divide-border-subtle">
            {area.customers.map((customer) => {
              const reading = pace.find((p) => p.pace.customerId === customer.customerId);
              return (
                <li
                  key={customer.customerId}
                  className="grid grid-cols-[minmax(0,1fr)_auto_auto] items-baseline gap-2.5 py-1.75 text-sm first:pt-0"
                >
                  <span className="truncate font-medium">{customer.name}</span>
                  <span className="text-xs text-foreground-subtle">
                    {t("{0} of loads", formatShare(customer.share))}
                  </span>
                  <span
                    className={cn(
                      "text-xs",
                      reading?.standing === "Behind"
                        ? "text-warning-foreground"
                        : "text-foreground-subtle",
                    )}
                  >
                    {reading
                      ? reading.standing === "Behind"
                        ? t("{0} of {1} in, behind", reading.tendered, Math.round(reading.usual))
                        : t("{0} of {1} in", reading.tendered, Math.round(reading.usual))
                      : t("On pace")}
                  </span>
                </li>
              );
            })}
          </ul>
        </Block>
      ) : null}

      {detail.positionings.length > 0 ? (
        <Block title={t("On the way")}>
          <Positionings rows={detail.positionings} />
        </Block>
      ) : null}

      {detail.reach.length > 0 ? (
        <Block title={t("Who can get there by {0}", slotLabel(detail.slotStart))}>
          <LazyComponent fallback={<ReachMapSkeleton />}>
            <ReachMap
              label={area.label}
              latitude={area.latitude}
              longitude={area.longitude}
              bounds={detail.bounds}
              reach={detail.reach}
            />
          </LazyComponent>
          <ReachList reach={detail.reach} />
        </Block>
      ) : null}

      {detail.recent.length > 0 && detail.recent.every((loads) => loads === 0) ? (
        <Block title={t("Track record here")}>
          <p className="text-xs text-foreground-subtle">
            {t(
              "No loads started here in {0} in any of the last {1} weeks, so there is no record to show yet.",
              slotRange(detail.slotStart),
              detail.recent.length,
            )}
          </p>
        </Block>
      ) : detail.recent.length > 0 ? (
        <Block title={t("Track record here")}>
          <RecentChart detail={detail} />
          <p className="text-xs text-foreground-subtle">
            {t(
              "{0}, the last {1} weeks, against its usual {2} to {3}.",
              slotRange(detail.slotStart),
              detail.recent.length,
              detail.recentLow,
              detail.recentHigh,
            )}
          </p>
        </Block>
      ) : null}

      <div className="flex flex-wrap gap-1.5 pt-1">
        <Button
          size="sm"
          variant="outline"
          nativeButton={false}
          render={<Link to="/dispatch/console" />}
        >
          {t("Open in console")}
        </Button>
        {detail.deliveryLocationIds.length > 0 ? (
          <Button
            size="sm"
            variant="ghost"
            nativeButton={false}
            render={
              <Link
                to={filteredListPath(RECORD_LINKS.shipment.path, [
                  {
                    field: "consigneeStop.locationId",
                    operator: "in",
                    value: detail.deliveryLocationIds,
                  },
                  { field: "status", operator: "in", value: OPEN_SHIPMENT_STATUSES },
                ])}
              />
            }
          >
            {t("Loads into {0}", area.label)}
          </Button>
        ) : (
          <p className="basis-full text-xs text-foreground-subtle">
            {t(
              "None of your locations are in {0} yet, so there are no loads into it to list.",
              area.label,
            )}
          </p>
        )}
      </div>
    </div>
  );
}

/**
 * One area in the inspector, the way the console opens a move: the board stays in sight and
 * the brief comes back with Back or Escape.
 */
export function AreaInspector({
  open,
  onBack,
  onCollapse,
}: {
  open: OpenArea;
  onBack: () => void;
  onCollapse: () => void;
}) {
  const t = useT();
  const query = useQuery(freightOutlook.area(open.cell, open.slotStart));

  return (
    <div className="flex min-h-0 flex-1 animate-in flex-col duration-200 ease-settle fade-in-0 slide-in-from-right-3 motion-reduce:animate-none">
      <div className="flex h-11 shrink-0 items-center gap-2 border-b border-border-subtle px-3.5">
        <Button type="button" size="sm" variant="quiet" className="-ml-1.5 pl-1" onClick={onBack}>
          <ChevronLeftIcon className="size-3.5" />
          {t("Brief")}
        </Button>
        <span className="ml-auto">
          <CollapseBriefButton onCollapse={onCollapse} />
        </span>
        <Button
          type="button"
          size="icon-xs"
          variant="ghost"
          aria-label={t("Close")}
          onClick={onBack}
        >
          <XCloseIcon className="size-3" />
        </Button>
      </div>
      <ScrollArea
        maskVariant="card"
        maskHeight={18}
        className="min-h-0 flex-1"
        viewportClassName="overscroll-contain"
      >
        <div className="p-3.5">
          {query.data ? (
            <AreaBody detail={query.data} />
          ) : query.isError ? (
            <ErrorState
              error={query.error}
              layout="sheet"
              title={t("The area didn't load")}
              description={t("This isn't an empty area: try again in a moment.")}
              onRetry={() => void query.refetch()}
            />
          ) : (
            <AreaInspectorSkeleton />
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
