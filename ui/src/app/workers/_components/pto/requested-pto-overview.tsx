/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge, BadgeProps } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import LetterGlitch from "@/components/ui/letter-glitch";
import { VirtualCompatibleScrollArea } from "@/components/ui/virtual-scroll-area";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { formatRange, inclusiveDays } from "@/lib/date";
import { queries } from "@/lib/queries";
import { PTOFilterSchema, WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { APIError } from "@/types/errors";
import { PTOStatus, PTOType } from "@/types/worker";
import { useInfiniteQuery, useMutation } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { CalendarRange, EllipsisIcon, Loader2 } from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { PTOFilterPopover } from "./pto-filter-popover";
import { PTORejectionDialog } from "./pto-rejection-dialog";
import { usePTOFilters } from "./use-pto-filters";

export default function RequestedPTOOverview() {
  // Use the hook to get default values only, but maintain local state
  const { defaultValues } = usePTOFilters();

  // Local filter state for this component only
  const [filters, setFilters] = useState({
    startDate: defaultValues.startDate,
    endDate: defaultValues.endDate,
    type: undefined as PTOType | undefined,
    workerId: undefined as string | undefined,
  });

  const handleFilterSubmit = useCallback((data: PTOFilterSchema) => {
    setFilters({
      startDate: data.startDate,
      endDate: data.endDate,
      type: data.type as PTOType | undefined,
      workerId: data.workerId,
    });
  }, []);

  const resetFilters = useCallback(() => {
    setFilters({
      startDate: defaultValues.startDate,
      endDate: defaultValues.endDate,
      type: undefined,
      workerId: undefined,
    });
  }, [defaultValues]);

  const query = useInfiniteQuery({
    queryKey: [
      ...queries.worker.listUpcomingPTO._def,
      {
        type: filters.type,
        status: PTOStatus.Requested,
        startDate: filters.startDate,
        endDate: filters.endDate,
        workerId: filters.workerId,
      },
    ],
    queryFn: async ({ pageParam }) => {
      return await api.worker.listUpcomingPTO({
        filter: { limit: 20, offset: pageParam },
        type: filters.type,
        status: PTOStatus.Requested,
        startDate: filters.startDate,
        endDate: filters.endDate,
        workerId: filters.workerId,
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

  const virtualizer = useVirtualizer({
    count: allPTOItems.length,
    getScrollElement: () => scrollAreaRef.current,
    estimateSize: () => 75,
    overscan: 5,
  });

  const virtualItems = virtualizer.getVirtualItems();

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
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

  return (
    <div className="flex flex-col gap-1 flex-1">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium font-table">Requested PTO</h3>
        <div className="flex items-center gap-2">
          <PTOFilterPopover
            defaultValues={defaultValues}
            onSubmit={handleFilterSubmit}
            onReset={resetFilters}
          />
        </div>
      </div>
      {!query.isLoading && allPTOItems.length === 0 && (
        <div className="flex flex-col items-center size-full justify-center overflow-hidden border border-border rounded-md">
          <div className="relative size-full">
            <LetterGlitch
              glitchColors={["#9c9c9c", "#696969", "#424242"]}
              glitchSpeed={50}
              centerVignette={true}
              outerVignette={true}
              smooth={true}
              className="size-full"
              canvasClassName="size-full"
            />
            <div className="absolute inset-0 flex flex-col gap-1 items-center justify-center pointer-events-none">
              <p className="text-sm/none px-1 py-0.5 text-center font-medium uppercase select-none font-table dark:text-neutral-900 bg-amber-300 text-amber-950 dark:bg-amber-400">
                No data available
              </p>
              <p className="text-sm/none px-1 py-0.5 text-center font-medium uppercase select-none font-table dark:text-neutral-900 bg-neutral-900 text-white dark:bg-neutral-500">
                Try adjusting your filters or search query
              </p>
            </div>
          </div>
        </div>
      )}
      {!query.isLoading && allPTOItems.length > 0 && (
        <VirtualCompatibleScrollArea
          viewportRef={scrollAreaRef}
          className="border border-border rounded-md flex-1"
          viewportClassName="p-3"
        >
          {query.isLoading && (
            <div className="flex items-center justify-center h-[250px]">
              <Loader2 className="size-5 animate-spin text-muted-foreground" />
            </div>
          )}

          {allPTOItems.length > 0 && (
            <div
              style={{
                height: `${virtualizer.getTotalSize()}px`,
                width: "100%",
                position: "relative",
              }}
            >
              {virtualItems.map((virtualItem) => {
                const workerPTO = allPTOItems[virtualItem.index];
                return (
                  <div
                    key={virtualItem.key}
                    style={{
                      position: "absolute",
                      top: 0,
                      left: 0,
                      width: "100%",
                      height: `${virtualItem.size}px`,
                      transform: `translateY(${virtualItem.start}px)`,
                    }}
                  >
                    <div className="pb-1">
                      <UpcomingPTOCard workerPTO={workerPTO} />
                    </div>
                  </div>
                );
              })}

              {query.isFetchingNextPage && (
                <div
                  style={{
                    position: "absolute",
                    top: `${virtualizer.getTotalSize()}px`,
                    left: 0,
                    width: "100%",
                  }}
                  className="flex items-center justify-center py-4"
                >
                  <Loader2 className="size-4 animate-spin text-muted-foreground" />
                  <span className="ml-2 text-sm text-muted-foreground">
                    Loading more...
                  </span>
                </div>
              )}

              <div
                ref={observerTarget}
                style={{
                  position: "absolute",
                  top: `${virtualizer.getTotalSize()}px`,
                  left: 0,
                  width: "100%",
                  height: "1px",
                }}
              />
            </div>
          )}
        </VirtualCompatibleScrollArea>
      )}
    </div>
  );
}

const initials = (first?: string, last?: string) =>
  `${(first?.[0] ?? "").toUpperCase()}${(last?.[0] ?? "").toUpperCase()}`.trim() ||
  "•";

function usePTOTypeMeta(type: WorkerPTOSchema["type"]) {
  return useMemo(() => {
    switch (type) {
      case "Vacation":
        return {
          label: "Vacation",
          badgeVariant: "purple",
          accentClass: "from-purple-600 to-purple-600/5",
        };
      case "Sick":
        return {
          label: "Sick",
          badgeVariant: "red",
          accentClass: "from-red-600 to-red-600/5",
        };
      case "Holiday":
        return {
          label: "Holiday",
          badgeVariant: "info",
          accentClass: "from-blue-600 to-blue-600/5",
        };
      case "Bereavement":
        return {
          label: "Bereavement",
          badgeVariant: "active",
          accentClass: "from-green-600 to-green-600/5",
        };
      case "Maternity":
        return {
          label: "Maternity",
          badgeVariant: "pink",
          accentClass: "from-pink-600 to-pink-600/5",
        };
      case "Paternity":
        return {
          label: "Paternity",
          badgeVariant: "teal",
          accentClass: "from-teal-600 to-teal-600/5",
        };
      default:
        return {
          label: String(type),
          accentClass: "from-muted-foreground/30 to-transparent",
        };
    }
  }, [type]);
}

function usePTOStatusMeta(status: WorkerPTOSchema["status"]) {
  return useMemo(() => {
    switch (status) {
      case "Approved":
        return { label: "Approved", dot: "bg-emerald-500/90" };
      case "Rejected":
        return { label: "Rejected", dot: "bg-rose-500/90" };
      case "Cancelled":
        return { label: "Cancelled", dot: "bg-zinc-500/70" };
      default:
        return { label: "Requested", dot: "bg-muted-foreground/60" };
    }
  }, [status]);
}

export const UpcomingPTOCard = memo(function UpcomingPTOCard({
  workerPTO,
}: {
  workerPTO: WorkerPTOSchema;
}) {
  const { worker, startDate, endDate, type, status } = workerPTO;
  const [rejectPTODialogOpen, setRejectPTODialogOpen] = useState(false);

  const days = inclusiveDays(startDate, endDate);
  const range = formatRange(startDate, endDate);
  const { label, accentClass, badgeVariant } = usePTOTypeMeta(type);
  const statusMeta = usePTOStatusMeta(status);

  const { mutateAsync: approvePTO } = useMutation({
    mutationFn: () => api.worker.approvePTO(workerPTO.id),
    onSuccess: () => {
      toast.success("PTO approved");
      broadcastQueryInvalidation({
        queryKey: [...queries.worker.listUpcomingPTO._def] as string[],
        options: {
          correlationId: `approve-pto-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        toast.error("Failed to approve PTO", {
          description: error.message,
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
  });

  return (
    <>
      <div
        className="group relative overflow-hidden rounded-xl border border-border p-3 transition-colors"
        role="article"
        aria-label={`${worker?.firstName} ${worker?.lastName} • ${label} • ${range} • ${statusMeta.label}`}
      >
        <div
          className={cn(
            "pointer-events-none absolute inset-y-0 left-0 w-[3px] bg-gradient-to-b",
            accentClass,
          )}
          aria-hidden
        />
        <div className="flex items-center gap-3">
          <Avatar className="h-9 w-9 ring-1 ring-border">
            <AvatarImage
              src={worker?.profilePictureUrl ?? undefined}
              alt={`${worker?.firstName ?? ""} ${worker?.lastName ?? ""}`}
            />
            <AvatarFallback className="text-xs">
              {initials(worker?.firstName, worker?.lastName)}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 min-w-0">
                <span className="truncate font-medium">
                  {worker?.firstName} {worker?.lastName}
                </span>
                <Badge
                  withDot={false}
                  variant={badgeVariant as BadgeProps["variant"]}
                  className="shrink-0 gap-1 px-2 py-0.5 text-[11px] leading-4"
                >
                  {label}
                </Badge>
              </div>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button size="sm" variant="ghostInvert" className="size-6">
                    <EllipsisIcon />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent side="bottom" align="end">
                  <DropdownMenuLabel>Actions</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    title="Approve"
                    description="Approve this PTO request"
                    onClick={() => {
                      approvePTO();
                    }}
                    color="success"
                  />
                  <DropdownMenuItem
                    title="Reject"
                    description="Reject this PTO request"
                    onClick={() => {
                      setRejectPTODialogOpen(true);
                    }}
                    color="danger"
                  />
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
            <div className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground shrink-0">
              <CalendarRange className="size-3.5" aria-hidden />
              <span className="tabular-nums">{range}</span>
              <span aria-hidden>•</span>
              <span className="tabular-nums">{days}d</span>
            </div>
          </div>
        </div>
      </div>
      {rejectPTODialogOpen && (
        <PTORejectionDialog
          open={rejectPTODialogOpen}
          onOpenChange={setRejectPTODialogOpen}
          ptoId={workerPTO.id ?? ""}
        />
      )}
    </>
  );
});
