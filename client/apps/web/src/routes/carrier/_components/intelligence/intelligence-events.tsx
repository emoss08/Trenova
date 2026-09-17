import { useT } from "@trenova/shared/i18n/use-t";
import { EventTimeline } from "@/components/carrier-intelligence/event-timeline";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import {
  CARRIER_INTEL_EVENTS_KEY,
  fetchCarrierIntelEvents,
} from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelEventStatus } from "@trenova/graphql/generated/graphql";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { RefreshCwIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";

export type IntelligenceEventsProps = {
  carrierId: string;
  canUpdate: boolean;
  onChanged: () => void;
};

type EventScope = "open" | "all";

const PAGE_SIZE = 25;
const OPEN_STATUSES: CarrierIntelEventStatus[] = ["Open", "Acknowledged"];

export function IntelligenceEvents({ carrierId, canUpdate, onChanged }: IntelligenceEventsProps) {
  const t = useT();
  const [scope, setScope] = useState<EventScope>("open");

  const eventsQuery = useInfiniteQuery({
    queryKey: [CARRIER_INTEL_EVENTS_KEY, carrierId, scope],
    queryFn: ({ pageParam, signal }) =>
      fetchCarrierIntelEvents(
        {
          carrierId,
          statuses: scope === "open" ? OPEN_STATUSES : undefined,
          first: PAGE_SIZE,
          after: pageParam,
        },
        { signal },
      ),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNextPage ? lastPage.endCursor : undefined),
  });

  const events = useMemo(
    () => eventsQuery.data?.pages.flatMap((page) => page.events) ?? [],
    [eventsQuery.data],
  );
  const totalCount = eventsQuery.data?.pages[0]?.totalCount ?? null;

  const { refetch } = eventsQuery;
  const handleChanged = useCallback(() => {
    void refetch();
    onChanged();
  }, [onChanged, refetch]);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <SegmentedControl<EventScope>
          items={[
            { value: "open", label: t("Needs attention") },
            { value: "all", label: t("All changes") },
          ]}
          value={scope}
          onValueChange={setScope}
          aria-label={t("Change filter")}
        />
        <div className="flex items-center gap-2">
          {totalCount !== null ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("{0, plural, one {# change} other {# changes}}", totalCount)}
            </span>
          ) : null}
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            aria-label={t("Refresh events")}
            onClick={() => void refetch()}
            isLoading={eventsQuery.isRefetching}
          >
            <RefreshCwIcon />
          </Button>
        </div>
      </div>
      {eventsQuery.isPending ? (
        <div className="divide-border flex flex-col divide-y" aria-busy>
          {[0, 1, 2].map((index) => (
            <div key={index} className="flex items-start gap-3 py-2.5">
              <Skeleton className="mt-1.5 size-2 rounded-full" />
              <div className="flex flex-1 flex-col gap-1.5">
                <Skeleton className="h-3.5 w-1/2" />
                <Skeleton className="h-3 w-1/3" />
              </div>
            </div>
          ))}
        </div>
      ) : eventsQuery.isError ? (
        <IntelInlineError
          error={eventsQuery.error}
          title={t("Changes could not be loaded")}
          onRetry={() => void refetch()}
        />
      ) : (
        <>
          <EventTimeline
            events={events}
            canUpdate={canUpdate}
            onChanged={handleChanged}
            emptyMessage={
              scope === "open"
                ? t("Nothing needs attention. Changes that do will appear here.")
                : t("No changes have been recorded for this carrier.")
            }
          />
          {eventsQuery.hasNextPage ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="self-center"
              isLoading={eventsQuery.isFetchingNextPage}
              onClick={() => void eventsQuery.fetchNextPage()}
            >
              {t("Load more")}
            </Button>
          ) : null}
        </>
      )}
    </div>
  );
}
