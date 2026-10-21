"use no memo";
import { useDeferredMeasure } from "@/hooks/use-deferred-measure";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { cn } from "@trenova/shared/lib/utils";
import type { ShipmentEvent, ShipmentEventSeverity } from "@/types/shipment-event";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useMemo } from "react";
import { ActivityRowsSkeleton, ActivityTimelineSkeleton } from "../side-panel-skeleton";
import { renderEvent } from "./event-renderer";
import { useShipmentEventsInfinite } from "./use-shipment-events";

/** A one-line event: its padding and a line of small, snug text. */
const ROW_ESTIMATE = 29;
/** The events drawn waiting at the end of the list while older ones remain. */
const PENDING_ROWS = 3;
/** The list's padding above the first event and below the last. */
const LIST_PADDING = 12;
const OVERSCAN = 8;
/** Rows from the end of what is read at which the next page is asked for. */
const PREFETCH_ROWS = 5;
/** A first window tall enough for the panel before its viewport is measured. */
const INITIAL_HEIGHT = 720;

const SEVERITY_NODE: Record<ShipmentEventSeverity, string> = {
  danger: "bg-danger",
  success: "bg-success",
  brand: "bg-brand",
  info: "bg-brand",
  muted: "bg-muted-foreground/50",
};

function nodeClass(event: ShipmentEvent) {
  if (event.actorType === "system" && event.severity !== "danger") return "bg-accent-teal";
  return SEVERITY_NODE[event.severity];
}

/**
 * The organization's shipment events as a timeline, newest first: arrivals in
 * brand, delays in danger, what automation did in teal. Only the events near
 * the viewport are mounted. While older events remain, the list ends in rows
 * drawn as events waiting to load, and the next page is read as the reader
 * nears them, so a page arriving pushes them down rather than appearing.
 */
export function ActivityTimeline({ scrollElement }: { scrollElement: HTMLElement | null }) {
  const t = useT();
  const query = useShipmentEventsInfinite();
  const events = useMemo(
    () => query.data?.pages.flatMap((page) => page) ?? [],
    [query.data?.pages],
  );
  const { hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage } = query;
  const pending = hasNextPage === true;

  // oxlint-disable-next-line react/incompatible-library -- TanStack Virtual is incompatible with the React Compiler; file is already "use no memo", which oxlint 1.89 ignores
  const virtualizer = useVirtualizer({
    count: events.length + (pending ? 1 : 0),
    getScrollElement: () => scrollElement,
    estimateSize: (index) => (index < events.length ? ROW_ESTIMATE : ROW_ESTIMATE * PENDING_ROWS),
    getItemKey: (index) => (index < events.length ? events[index].id : "pending"),
    overscan: OVERSCAN,
    paddingStart: LIST_PADDING,
    paddingEnd: LIST_PADDING,
    initialRect: { width: 0, height: INITIAL_HEIGHT },
  });
  const measureRow = useDeferredMeasure(virtualizer);
  const items = virtualizer.getVirtualItems();
  const lastInView = items.at(-1)?.index ?? -1;

  useEffect(() => {
    if (!pending || isFetchingNextPage || isFetchNextPageError || events.length === 0) return;
    if (lastInView >= events.length - 1 - PREFETCH_ROWS) {
      void fetchNextPage();
    }
  }, [lastInView, events.length, pending, isFetchingNextPage, isFetchNextPageError, fetchNextPage]);

  if (query.isLoading) {
    return <ActivityTimelineSkeleton />;
  }
  if (query.isError && events.length === 0) {
    return <p className="text-danger p-4 text-sm">{t("Activity could not be loaded.")}</p>;
  }
  if (events.length === 0) {
    return <p className="text-muted-foreground p-4 text-sm">{t("No activity yet")}</p>;
  }

  const now = Date.now() / 1000;
  return (
    <ol
      className="relative px-4"
      style={{ height: virtualizer.getTotalSize() }}
      aria-label={t("Shipment activity")}
    >
      <span aria-hidden className="bg-border absolute top-5 bottom-5 left-[19px] w-px" />
      {items.map((item) => {
        if (item.index >= events.length) {
          return (
            <li
              key={item.key}
              data-index={item.index}
              ref={measureRow}
              className="absolute inset-x-4 top-0"
              style={{ transform: `translateY(${item.start}px)` }}
            >
              {isFetchNextPageError ? (
                <div className="flex items-center gap-3 py-1.5 pl-5">
                  <p className="text-muted-foreground min-w-0 flex-1 text-xs">
                    {t("Older activity could not be loaded.")}
                  </p>
                  <Button
                    type="button"
                    variant="outline"
                    size="xs"
                    onClick={() => void fetchNextPage()}
                  >
                    {t("Try again")}
                  </Button>
                </div>
              ) : (
                <div aria-busy aria-label={t("Loading older activity")}>
                  <div className="contents" aria-hidden>
                    <ActivityRowsSkeleton count={PENDING_ROWS} />
                  </div>
                </div>
              )}
            </li>
          );
        }
        const event = events[item.index];
        const rendered = renderEvent(event);
        return (
          <li
            key={item.key}
            data-index={item.index}
            ref={measureRow}
            className="absolute inset-x-4 top-0 flex items-start gap-3 py-1.5"
            style={{ transform: `translateY(${item.start}px)` }}
          >
            <span
              aria-hidden
              className={cn(
                "ring-raised relative mt-1.5 size-[7px] shrink-0 rounded-full ring-2",
                nodeClass(event),
              )}
            />
            <p className="min-w-0 flex-1 text-sm leading-snug">{rendered.headline}</p>
            <time
              className="text-muted-foreground shrink-0 font-mono text-xs tabular-nums"
              dateTime={new Date(event.occurredAt * 1000).toISOString()}
            >
              {formatRelativeTime(event.occurredAt - now)}
            </time>
          </li>
        );
      })}
    </ol>
  );
}
