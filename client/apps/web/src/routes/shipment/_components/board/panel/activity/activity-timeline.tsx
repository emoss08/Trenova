import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { cn } from "@trenova/shared/lib/utils";
import type { ShipmentEvent, ShipmentEventSeverity } from "@/types/shipment-event";
import { useEffect, useMemo, useRef } from "react";
import { renderEvent } from "./event-renderer";
import { useShipmentEventsInfinite } from "./use-shipment-events";

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
 * brand, delays in danger, what automation did in teal. Older events load as
 * the list scrolls.
 */
export function ActivityTimeline() {
  const t = useT();
  const query = useShipmentEventsInfinite();
  const events = useMemo(() => query.data?.pages.flatMap((page) => page) ?? [], [query.data?.pages]);
  const sentinelRef = useRef<HTMLLIElement>(null);
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;

  useEffect(() => {
    const target = sentinelRef.current;
    if (!target) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  if (query.isLoading) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-10/12" />
        <Skeleton className="h-4 w-9/12" />
      </div>
    );
  }
  if (query.isError) {
    return <p className="text-danger p-4 text-sm">{t("Activity could not be loaded.")}</p>;
  }
  if (events.length === 0) {
    return <p className="text-muted-foreground p-4 text-sm">{t("No activity yet")}</p>;
  }

  const now = Date.now() / 1000;
  return (
    <ol className="relative flex flex-col px-4 py-3" aria-label={t("Shipment activity")}>
      <span aria-hidden className="bg-border absolute top-5 bottom-5 left-[19px] w-px" />
      {events.map((event) => {
        const rendered = renderEvent(event);
        return (
          <li key={event.id} className="relative flex items-start gap-3 py-1.5">
            <span aria-hidden className={cn("ring-raised relative mt-1.5 size-[7px] shrink-0 rounded-full ring-2", nodeClass(event))} />
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
      {hasNextPage ? <li ref={sentinelRef} className="h-4" aria-hidden /> : null}
    </ol>
  );
}
