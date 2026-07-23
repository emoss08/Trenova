import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { ShipmentEvent, ShipmentEventSeverity } from "@/types/shipment-event";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { useEffect, useMemo, useRef } from "react";
import { renderEvent } from "./event-renderer";
import { useShipmentEventsInfinite } from "./use-shipment-events";

const SEVERITY_DOT: Record<ShipmentEventSeverity, string> = {
  danger: "bg-destructive",
  success: "bg-success",
  brand: "bg-brand",
  info: "bg-info",
  muted: "bg-muted-foreground/40",
};

type Props = {
  shipmentId?: string;
  pageSize?: number;
  emptyLabel?: string;
  enabled?: boolean;
};

export function ActivityFeed({
  shipmentId,
  pageSize,
  emptyLabel = "No activity yet",
  enabled = true,
}: Props) {
  const query = useShipmentEventsInfinite({ shipmentId, pageSize, enabled });
  const events = useMemo(
    () => query.data?.pages.flatMap((page) => page) ?? [],
    [query.data?.pages],
  );

  const sentinelRef = useRef<HTMLDivElement>(null);
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

  return (
    <section className="cc-module-card flex min-h-[260px] flex-col">
      <header className="flex items-center justify-between border-b border-border px-3 py-2">
        <div className="flex items-center gap-2">
          <h3 className="cc-label text-foreground">Activity stream</h3>
          <span aria-hidden className="size-1.5 rounded-full bg-success" />
          <span className="font-mono text-[10px] text-muted-foreground">live</span>
        </div>
      </header>
      <ScrollArea className="min-h-0 flex-1" viewportClassName="max-h-[260px]">
        <FeedBody
          isLoading={query.isLoading}
          isError={query.isError}
          events={events}
          emptyLabel={emptyLabel}
        />
        {hasNextPage && (
          <div ref={sentinelRef} className="h-4" aria-hidden />
        )}
        {isFetchingNextPage && (
          <p className="px-3 py-1 text-center font-mono text-[10px] text-muted-foreground">
            Loading more…
          </p>
        )}
      </ScrollArea>
    </section>
  );
}

type FeedBodyProps = {
  isLoading: boolean;
  isError: boolean;
  events: ShipmentEvent[];
  emptyLabel: string;
};

function FeedBody({ isLoading, isError, events, emptyLabel }: FeedBodyProps) {
  if (isLoading) {
    return (
      <div className="flex flex-col gap-1 px-3 py-2">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-11/12" />
        <Skeleton className="h-4 w-9/12" />
      </div>
    );
  }

  if (isError) {
    return (
      <p className="px-3 py-2 text-[11px] text-destructive">
        Failed to load activity. Try refreshing.
      </p>
    );
  }

  if (events.length === 0) {
    return <p className="px-3 py-2 text-[11px] text-muted-foreground">{emptyLabel}</p>;
  }

  return (
    <ul className="flex flex-col">
      {events.map((event) => (
        <ActivityFeedItem key={event.id} event={event} />
      ))}
    </ul>
  );
}

function ActivityFeedItem({ event }: { event: ShipmentEvent }) {
  const rendered = renderEvent(event);
  return (
    <li className="group flex items-start gap-2 px-3 py-1.5 hover:bg-muted/40">
      <span className={cn("mt-1.5 size-1.5 shrink-0 rounded-full", SEVERITY_DOT[event.severity])} />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-start justify-between gap-2">
          <p className="min-w-0 flex-1 text-[11.5px] leading-snug text-foreground">
            {rendered.headline}
          </p>
          <time
            className="shrink-0 font-mono text-[10.5px] text-muted-foreground tabular-nums"
            dateTime={new Date(event.occurredAt * 1000).toISOString()}
          >
            {formatRelative(event.occurredAt)}
          </time>
        </div>
        {rendered.detail && (
          <p className="line-clamp-2 text-[11px] text-muted-foreground">{rendered.detail}</p>
        )}
        <p className="font-mono text-[10px] text-muted-foreground">{rendered.actorHandle}</p>
      </div>
    </li>
  );
}

function formatRelative(occurredAt: number): string {
  const date = fromUnixTime(occurredAt);
  return formatDistanceToNowStrict(date, { addSuffix: false });
}
