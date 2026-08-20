import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { apiService } from "@/services/api";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { ShipmentComment } from "@/types/shipment-comment";
import { useInfiniteQuery } from "@tanstack/react-query";
import { PlusIcon } from "lucide-react";
import { parseAsString, useQueryState, useQueryStates } from "nuqs";
import { useEffect, useMemo, useRef } from "react";

const PAGE_SIZE = 10;

function formatUserHandle(comment: ShipmentComment): string {
  const emailHandle = comment.user?.emailAddress?.split("@")[0]?.trim();
  if (emailHandle) return `@${emailHandle.toLowerCase()}`;

  const nameParts = comment.user?.name?.trim().toLowerCase().split(/\s+/).filter(Boolean) ?? [];
  if (nameParts.length >= 2) {
    const firstInitial = nameParts[0]?.[0] ?? "";
    const lastName = nameParts[nameParts.length - 1]?.replace(/[^a-z0-9._-]/g, "") ?? "";
    if (firstInitial && lastName) return `@${firstInitial}.${lastName}`;
  }

  const fallbackName = nameParts[0]?.replace(/[^a-z0-9._-]/g, "");
  if (fallbackName) return `@${fallbackName}`;

  return `@${comment.source.toLowerCase()}`;
}

function formatCompactRelativeTime(timestamp: number): string {
  const seconds = Math.max(0, Math.floor((Date.now() - timestamp * 1000) / 1000));
  if (seconds < 60) return "now";

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;

  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d`;

  const weeks = Math.floor(days / 7);
  if (weeks < 5) return `${weeks}w`;

  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo`;

  return `${Math.floor(days / 365)}y`;
}

function CommentRow({ comment }: { comment: ShipmentComment }) {
  return (
    <li className="min-w-0">
      <p className="hover:bg-muted/40 flex min-w-0 items-baseline gap-1.5 rounded px-1.5 py-1 text-[11px] leading-tight transition-colors">
        <span className="font-table text-foreground shrink-0 font-semibold">
          {formatUserHandle(comment)}
        </span>
        <time
          dateTime={new Date(comment.createdAt * 1000).toISOString()}
          className="font-table text-muted-foreground shrink-0 text-[10px] tabular-nums"
        >
          {formatCompactRelativeTime(comment.createdAt)}
        </time>
        <span className="text-muted-foreground shrink-0">·</span>
        <span className="text-muted-foreground min-w-0 truncate">{comment.comment}</span>
      </p>
    </li>
  );
}

function LoadingRows() {
  return (
    <div className="flex flex-col gap-1 px-1.5 py-1">
      {Array.from({ length: 3 }).map((_, index) => (
        <div key={index} className="flex items-center gap-1.5 py-1">
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-6" />
          <Skeleton className="h-3 min-w-0 flex-1" />
        </div>
      ))}
    </div>
  );
}

function EmptyState({ onAddComment }: { onAddComment: () => void }) {
  return (
    <div className="text-muted-foreground flex flex-col items-start px-1.5 py-1 text-xs leading-snug">
      <p>
        No comments yet. Use <span className="font-table text-foreground">@mentions</span> to ping a
        teammate.
      </p>
      <button
        type="button"
        className="text-brand flex h-3 cursor-pointer flex-row px-0 py-0 text-xs hover:underline"
        onClick={onAddComment}
      >
        <PlusIcon className="size-3" />
        Add comment
      </button>
    </div>
  );
}

export function CommentBlock({ shipmentId }: { shipmentId: Shipment["id"] }) {
  const [, setSearchParams] = useQueryStates(panelSearchParamsParser);
  const [, setActiveTab] = useQueryState("tab", parseAsString.withDefault("details"));
  const hasShipmentId = Boolean(shipmentId);

  const query = useInfiniteQuery({
    queryKey: ["shipment-comments", shipmentId, "compact"],
    queryFn: async ({ pageParam }) => {
      if (!shipmentId) {
        throw new Error("Shipment ID is required");
      }

      return await apiService.shipmentCommentService.list(shipmentId, {
        limit: PAGE_SIZE,
        after: pageParam,
      });
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo?.hasNextPage ? lastPage.pageInfo.endCursor : undefined,
    enabled: hasShipmentId,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });

  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;

  const allComments = useMemo(
    () => query.data?.pages.flatMap((page) => page.results) ?? [],
    [query.data?.pages],
  );

  const total = query.data?.pages[0]?.count ?? allComments.length;
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const observerTarget = useRef<HTMLLIElement>(null);

  const handleAddComment = () => {
    if (!shipmentId) return;

    void setSearchParams({ panelType: "edit", panelEntityId: shipmentId });
    void setActiveTab("comments");
  };

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

  return (
    <div className="min-w-0">
      <div className="mb-1 flex items-center justify-between gap-2">
        <h5 className="cc-label">Comments</h5>
        <span className="font-table text-muted-foreground text-[10px] tabular-nums">{total}</span>
      </div>
      <ScrollArea
        ref={scrollAreaRef}
        className="border-border/70 bg-muted/20 h-24 rounded-md border"
        viewportClassName="px-1 py-1"
      >
        {query.isLoading ? (
          <LoadingRows />
        ) : query.isError ? (
          <p className="text-muted-foreground px-1.5 py-1 text-xs">Comments unavailable.</p>
        ) : allComments.length === 0 ? (
          <EmptyState onAddComment={handleAddComment} />
        ) : (
          <ol className="flex min-w-0 flex-col">
            {allComments.map((comment) => (
              <CommentRow key={comment.id} comment={comment} />
            ))}
            {isFetchingNextPage && (
              <li className="text-muted-foreground px-1.5 py-1 text-[10px]">
                Loading older comments...
              </li>
            )}
            <li ref={observerTarget} className="h-px" />
          </ol>
        )}
      </ScrollArea>
    </div>
  );
}
