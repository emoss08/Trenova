import { ScrollArea } from "@/components/ui/scroll-area";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { PTOFilter, PTOType } from "@/types/worker";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { PTOFilterPopover } from "../pto-filter-popover";
import { HeaderContent } from "../pto-header-components";
import { usePTOFilters } from "../use-pto-filters";
import { ptoSearchParamsParser } from "../use-pto-state";
import {
  RequestedPTOEmptyState,
  RequestedPTOErrorState,
  RequestedPTOOverviewSkeleton,
} from "./requested-pto-state";
import { UpcomingPTOCard } from "./upcoming-pto-card";

export function RequestedPTOOverview() {
  const [searchParams] = useQueryStates(ptoSearchParamsParser);

  const query = useInfiniteQuery({
    queryKey: [
      ...queries.worker.listUpcomingPTO._def,
      {
        type: searchParams?.requestPTOFilters?.type,
        status: "Requested",
        startDate: searchParams?.requestPTOFilters?.startDate,
        endDate: searchParams?.requestPTOFilters?.endDate,
        workerId: searchParams?.requestPTOFilters?.workerId,
      },
    ],
    queryFn: async ({ pageParam }) => {
      return await apiService.workerService.listUpcomingPTO({
        filter: { limit: 20, offset: pageParam },
        type: searchParams?.requestPTOFilters?.type as PTOType | undefined,
        status: "Requested",
        startDate: searchParams?.requestPTOFilters?.startDate,
        endDate: searchParams?.requestPTOFilters?.endDate,
        workerId: searchParams?.requestPTOFilters?.workerId,
      });
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.next || lastPage.results.length === 20) {
        return lastPageParam + 20;
      }
      return undefined;
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });

  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;

  const allPTOItems = useMemo(
    () => query.data?.pages.flatMap((page) => page.results) ?? [],
    [query.data?.pages],
  );

  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const observerTarget = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const renderContent = () => {
    if (query.isError) {
      return <RequestedPTOErrorState />;
    }

    if (query.isLoading) {
      return <RequestedPTOOverviewSkeleton />;
    }

    if (allPTOItems.length === 0) {
      return <RequestedPTOEmptyState />;
    }

    return (
      <ScrollArea
        ref={scrollAreaRef}
        className="flex-1 rounded-md border border-border"
        viewportClassName="px-1.5 py-1"
      >
        <div
          style={{
            width: "100%",
            position: "relative",
          }}
        >
          {allPTOItems.map((workerPTO) => (
            <UpcomingPTOCard key={workerPTO.id} workerPTO={workerPTO} />
          ))}
          {query.isFetchingNextPage && (
            <div
              style={{
                position: "absolute",
                left: 0,
                width: "100%",
              }}
              className="flex items-center justify-center py-4"
            >
              <TextShimmer className="font-mono text-sm" duration={1}>
                Loading more...
              </TextShimmer>
            </div>
          )}
          <div
            ref={observerTarget}
            style={{
              position: "absolute",
              left: 0,
              width: "100%",
              height: "1px",
            }}
          />
        </div>
      </ScrollArea>
    );
  };

  return (
    <RequestedPTOOverviewOuter>
      <RequestedPTOHeader />
      {renderContent()}
    </RequestedPTOOverviewOuter>
  );
}

function RequestedPTOOverviewOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-1 flex-col gap-1">{children}</div>;
}

function RequestedPTOHeader() {
  const [, setSearchParams] = useQueryStates(ptoSearchParamsParser);
  const { defaultValues } = usePTOFilters();

  const handleFilterSubmit = useCallback(
    (data: PTOFilter) => {
      void setSearchParams({
        requestPTOFilters: {
          startDate: data.startDate,
          endDate: data.endDate,
          type: data.type as PTOType | undefined,
          workerId: data.workerId,
        },
      });
    },
    [setSearchParams],
  );

  const resetFilters = useCallback(() => {
    void setSearchParams({
      requestPTOFilters: {
        startDate: defaultValues.startDate,
        endDate: defaultValues.endDate,
        type: undefined,
        workerId: undefined,
      },
    });
  }, [defaultValues, setSearchParams]);

  return (
    <HeaderContent title="Requested PTO">
      <PTOFilterPopover
        defaultValues={defaultValues}
        onSubmit={handleFilterSubmit}
        onReset={resetFilters}
      />
    </HeaderContent>
  );
}
