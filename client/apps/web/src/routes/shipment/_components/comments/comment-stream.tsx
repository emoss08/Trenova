import { Button } from "@trenova/shared/components/ui/button";
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerViewport,
  useMessageScroller,
  useMessageScrollerScrollable,
} from "@trenova/shared/components/ui/message-scroller";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { fromUnixTime, isSameDay } from "date-fns";
import { MessageSquareIcon, RotateCcwIcon, SearchXIcon } from "lucide-react";
import { m } from "motion/react";
import { useEffect, useMemo, useRef, type ReactNode } from "react";
import { useNewCommentsPill } from "@/hooks/shipment-comments/use-new-comments-pill";
import type { LocalShipmentComment } from "@/lib/shipment-comment-cache";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { NewCommentsPill } from "./new-comments-pill";

export interface CommentJumpRequest {
  commentId: string;
  nonce: number;
}

export function CommentStream({
  shipmentId,
  comments,
  isLoading,
  isError,
  onRetry,
  isFiltered,
  hasNextPage,
  isFetchingNextPage,
  fetchNextPage,
  jumpRequest,
  highlightedCommentId,
  onHighlight,
  onClearFilters,
  renderComment,
}: {
  shipmentId: string;
  comments: LocalShipmentComment[];
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
  isFiltered: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => Promise<unknown>;
  jumpRequest: CommentJumpRequest | null;
  highlightedCommentId: string | null;
  onHighlight: (commentId: string | null) => void;
  onClearFilters: () => void;
  renderComment: (comment: LocalShipmentComment, highlighted: boolean) => ReactNode;
}) {
  const { scrollToEnd, scrollToMessage } = useMessageScroller();
  const scrollable = useMessageScrollerScrollable();
  const isAtEnd = !scrollable.end;
  const { unseenCount, clearUnseen } = useNewCommentsPill(shipmentId, isAtEnd);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const jumpAttemptsRef = useRef(0);

  const latestPendingId = useMemo(() => {
    for (let index = comments.length - 1; index >= 0; index--) {
      if (comments[index].pending) return comments[index].id;
    }
    return null;
  }, [comments]);

  useEffect(() => {
    if (latestPendingId) {
      scrollToEnd({ behavior: "smooth" });
    }
  }, [latestPendingId, scrollToEnd]);

  useEffect(() => {
    const target = sentinelRef.current;
    if (!target || isLoading) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    observer.observe(target);
    return () => observer.unobserve(target);
  }, [isLoading, hasNextPage, isFetchingNextPage, fetchNextPage]);

  useEffect(() => {
    if (!jumpRequest) return;

    const found = comments.some((comment) => comment.id === jumpRequest.commentId);
    if (found) {
      jumpAttemptsRef.current = 0;
      scrollToMessage(jumpRequest.commentId, { align: "center", behavior: "smooth" });
      onHighlight(jumpRequest.commentId);
      const timeout = window.setTimeout(() => onHighlight(null), 1500);
      return () => window.clearTimeout(timeout);
    }

    if (hasNextPage && jumpAttemptsRef.current < 5 && !isFetchingNextPage) {
      jumpAttemptsRef.current += 1;
      void fetchNextPage();
    }
  }, [
    jumpRequest,
    comments,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    scrollToMessage,
    onHighlight,
  ]);

  if (isLoading) return <LoadingSkeleton />;

  if (isError) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
        <MessageSquareIcon className="size-8 opacity-40" />
        <p className="text-sm font-medium">Comments could not be loaded</p>
        <Button type="button" variant="outline" size="xs" onClick={onRetry}>
          <RotateCcwIcon className="mr-1 size-3" />
          Try again
        </Button>
      </div>
    );
  }

  if (comments.length === 0) {
    return isFiltered ? (
      <div className="flex h-full flex-col items-center justify-center py-12 text-muted-foreground">
        <SearchXIcon className="mb-3 size-8 opacity-40" />
        <p className="text-sm font-medium">No comments match your filters</p>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="mt-2 text-xs"
          onClick={onClearFilters}
        >
          Clear filters
        </Button>
      </div>
    ) : (
      <div className="flex h-full flex-col items-center justify-center py-12 text-muted-foreground">
        <MessageSquareIcon className="mb-3 size-8 opacity-40" />
        <p className="text-sm font-medium">No comments yet</p>
        <p className="mt-1 max-w-[260px] text-center text-xs">
          Add operational notes, tag team members, and coordinate on this shipment.
        </p>
      </div>
    );
  }

  return (
    <MessageScroller className="h-full">
      <MessageScrollerViewport preserveScrollOnPrepend aria-label="Shipment comments">
        <MessageScrollerContent className="gap-0 px-2 py-2">
          <div ref={sentinelRef} className="h-px shrink-0" />
          {isFetchingNextPage && (
            <div className="flex items-center justify-center py-2">
              <span className="text-2xs text-muted-foreground">Loading older comments…</span>
            </div>
          )}
          {comments.map((comment, index) => {
            const previous = index > 0 ? comments[index - 1] : null;
            const showDivider =
              !previous ||
              !isSameDay(fromUnixTime(previous.createdAt), fromUnixTime(comment.createdAt));

            return (
              <MessageScrollerItem key={comment.id} messageId={comment.id}>
                {showDivider && <DayDivider timestamp={comment.createdAt} />}
                <m.div
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.2, ease: "easeOut" }}
                >
                  {renderComment(comment, highlightedCommentId === comment.id)}
                </m.div>
              </MessageScrollerItem>
            );
          })}
        </MessageScrollerContent>
      </MessageScrollerViewport>
      {unseenCount === 0 && <MessageScrollerButton />}
      <NewCommentsPill
        count={unseenCount}
        onClick={() => {
          clearUnseen();
          scrollToEnd({ behavior: "smooth" });
        }}
      />
    </MessageScroller>
  );
}

function DayDivider({ timestamp }: { timestamp: number }) {
  return (
    <div className="flex items-center gap-3 px-2 py-2">
      <span className="h-px flex-1 bg-border" />
      <span className="shrink-0 text-2xs font-medium text-muted-foreground">
        {formatUnixMonthDay(timestamp)}
      </span>
      <span className="h-px flex-1 bg-border" />
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="flex flex-col gap-3 px-4 py-3">
      {Array.from({ length: 3 }).map((_, index) => (
        <div key={index} className="flex items-start gap-3">
          <Skeleton className="size-9 rounded-full" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-3 w-full" />
            <Skeleton className="h-3 w-2/3" />
          </div>
        </div>
      ))}
    </div>
  );
}
