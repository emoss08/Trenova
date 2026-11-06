import { fetchAllRows } from "@/lib/data-table-export";
import { shipmentTimelineGraphQLConfig } from "@/lib/graphql/shipment";
import { ACTIVE_STAGES, type ActiveStage } from "@/lib/shipment-board/stage";
import {
  buildTimelineLoads,
  groupLoadsByStage,
  isoDay,
  overlapsRange,
  type TimelineRowMode,
} from "@/lib/shipment-board/timeline";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTableQueryOptions } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useEffect, useState } from "react";
import { SHIPMENT_LIST_KEY } from "../../shipment-queries";
import {
  ShipmentTimelineBodySkeleton,
  ShipmentTimelineSkeleton,
  type TimelineSkeletonShape,
} from "./shipment-timeline-skeleton";
import { createTimelineScrollMemory } from "@/components/timeline/timeline-canvas";
import { TimelineCard } from "./timeline/timeline-card";
import { TimelineCoverageLanes } from "./timeline/timeline-coverage-lanes";
import { TimelineHoverCard } from "./timeline/timeline-hover-card";
import { TimelineShipmentSheet } from "./timeline/timeline-shipment-rows";
import { TimelineToolbar } from "./timeline/timeline-toolbar";
import { useTimelineRange } from "./timeline/use-timeline-range";

/** The most loads one range draws; past it the timeline says so and asks for narrower filters. */
export const TIMELINE_ROW_LIMIT = 500;
const DAY = 86_400;
const NOON = 12 * 3_600;

type ShipmentTimelineProps = {
  queryOptions: Omit<DataTableQueryOptions, "cursor">;
};

/**
 * The board as a dispatch timeline: every load active in the visible range,
 * by shipment or by the driver or carrier hauling it, with its stop windows,
 * what has been driven and where its ETA lands. The range, its first day and
 * the rows live in the URL; the server returns only the loads the range shows,
 * narrowed by the board's own search, filters and quick filters.
 *
 * Nothing here reads the clock: the canvas's now line, the driven part of a
 * load under way and a driver's free time each read it themselves, so a minute
 * passing redraws those and not the sheet.
 */
export default function ShipmentTimeline({ queryOptions }: ShipmentTimelineProps) {
  const t = useT();
  const { timezone, range, mode, qf, today, days, scale, setUrl, sheetRef } = useTimelineRange();
  // A selection belongs to the row mode it was made in: switching between
  // shipments and drivers & carriers closes the card, and switching back
  // brings it back.
  const [selection, setSelection] = useState<{ id: string; mode: TimelineRowMode } | null>(null);
  const selectedId = selection?.mode === mode ? selection.id : null;
  const [hiddenStages, setHiddenStages] = useState<ReadonlySet<ActiveStage>>(new Set());
  // Outlives either sheet, so switching rows keeps the hours in view.
  const [scrollMemory] = useState(createTimelineScrollMemory);

  const graphql = shipmentTimelineGraphQLConfig({
    quickFilters: qf,
    timezone,
    windowStart: scale.start,
    windowEnd: scale.end,
  });
  const { data, isPlaceholderData, isError, refetch } = useQuery({
    queryKey: [SHIPMENT_LIST_KEY, "timeline", graphql, queryOptions],
    queryFn: ({ signal }) =>
      fetchAllRows<Shipment>({
        graphql,
        options: queryOptions,
        maxRows: TIMELINE_ROW_LIMIT,
        signal,
      }),
    placeholderData: keepPreviousData,
    staleTime: 15_000,
  });

  useEffect(() => {
    if (!selectedId) return;
    const close = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || event.defaultPrevented) return;
      // A dialog over the timeline (assigning a load) takes its own Escape.
      if (event.target instanceof Element && event.target.closest('[role="dialog"]')) return;
      setSelection(null);
    };
    document.addEventListener("keydown", close);
    return () => document.removeEventListener("keydown", close);
  }, [selectedId]);

  if (!data) {
    return isError ? (
      <div className="flex h-full min-h-105 flex-col items-center justify-center gap-2 p-8 text-center">
        <p className="text-sm font-medium">{t("The timeline could not be loaded.")}</p>
        <Button variant="outline" size="sm" onClick={() => void refetch()}>
          {t("Try again")}
        </Button>
      </div>
    ) : (
      <ShipmentTimelineSkeleton />
    );
  }

  // While another range loads, the rows on hand belong to the old one: show the
  // sheet loading rather than the old rows filtered down to nothing.
  const loading = isPlaceholderData;
  const inRange = loading
    ? []
    : buildTimelineLoads(data).filter((load) => overlapsRange(load, scale.start, scale.end));
  const counts = Object.fromEntries(
    ACTIVE_STAGES.map((stage) => [stage, inRange.filter((load) => load.stage === stage).length]),
  ) as Record<ActiveStage, number>;
  const loads = inRange.filter((load) => !hiddenStages.has(load.stage));
  const selected = selectedId ? (inRange.find((load) => load.id === selectedId) ?? null) : null;
  const toggleSelected = (id: string) =>
    setSelection((current) => (current?.id === id && current.mode === mode ? null : { id, mode }));
  const moveBy = (direction: -1 | 1) =>
    void setUrl({
      tlday: isoDay(direction < 0 ? scale.start - days * DAY + NOON : scale.end + NOON, timezone),
    });

  return (
    <div ref={sheetRef} className="@container/timeline flex h-full min-h-105 flex-col">
      <TimelineToolbar
        scale={scale}
        range={range}
        mode={mode}
        onModeChange={(next) => void setUrl({ tlrows: next })}
        onRangeChange={(next) => void setUrl({ tl: next })}
        onEarlier={() => moveBy(-1)}
        onLater={() => moveBy(1)}
        onNow={() => void setUrl({ tlday: null })}
        counts={counts}
        hiddenStages={hiddenStages}
        onToggleStage={(stage) =>
          setHiddenStages((current) => {
            const next = new Set(current);
            if (next.has(stage)) next.delete(stage);
            else next.add(stage);
            return next;
          })
        }
      />
      {!loading && data.length >= TIMELINE_ROW_LIMIT ? (
        <p className="bg-warning-subtle text-warning-subtle-foreground border-warning-border shrink-0 border-b px-3 py-1.5 text-xs">
          {t(
            "Showing the first {0} loads in this range. Narrow the board's filters to see the rest.",
            TIMELINE_ROW_LIMIT,
          )}
        </p>
      ) : null}
      {loading ? (
        <div className="flex min-h-0 flex-1" aria-busy="true" aria-label={t("Loading the range")}>
          <ShipmentTimelineBodySkeleton
            scale={scale}
            today={today}
            mode={mode}
            shape={skeletonShape(data, mode)}
          />
        </div>
      ) : (
        <div className="relative min-h-0 flex-1">
          <TimelineHoverCard scale={scale}>
            {mode === "shipments" ? (
              <TimelineShipmentSheet
                loads={loads}
                scale={scale}
                today={today}
                selectedId={selectedId}
                onSelect={toggleSelected}
                scrollMemory={scrollMemory}
              />
            ) : (
              <TimelineCoverageLanes
                loads={loads}
                scale={scale}
                today={today}
                selectedId={selectedId}
                onSelect={toggleSelected}
                scrollMemory={scrollMemory}
              />
            )}
          </TimelineHoverCard>
          {selected ? (
            <TimelineCard load={selected} scale={scale} onClose={() => setSelection(null)} />
          ) : null}
        </div>
      )}
    </div>
  );
}

/**
 * The shape of the sheet being replaced: its stage groups' sizes by shipment,
 * or the pool, driver and carrier lanes by coverage. The new range's skeleton
 * takes it, so stepping a day does not reflow the sheet into a generic one.
 */
function skeletonShape(rows: readonly Shipment[], mode: TimelineRowMode): TimelineSkeletonShape {
  const loads = buildTimelineLoads(rows);
  if (mode === "shipments") return groupLoadsByStage(loads).map((group) => group.loads.length);
  const drivers = new Set<string>();
  const carriers = new Set<string>();
  for (const load of loads) {
    if (load.coverage.kind === "driver") drivers.add(load.coverage.id);
    if (load.coverage.kind === "carrier") carriers.add(load.coverage.id);
  }
  return [1, drivers.size, carriers.size];
}
